// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the engine-adapter service.
// It wires the Temporal worker (IRInterpreterWorkflow + activities) and the
// gRPC server (EngineAdapterService). All business logic lives in internal/.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/engine-adapter/internal/api"
	"github.com/zynax-io/zynax/services/engine-adapter/internal/domain"
	"github.com/zynax-io/zynax/services/engine-adapter/internal/infrastructure"
)

type config struct {
	GRPCPort            int
	MetricsPort         int
	LogLevel            string
	TemporalHostPort    string
	TemporalNamespace   string
	TemporalTaskQueue   string
	TaskBrokerAddr      string
	EventBusAddr        string
	ActiveEngine        string
	GRPCCallTimeoutS    int
	MaxActivityAttempts int32
	LivenessThresholdS  int
	TLSCert             string
	TLSKey              string
	TLSCA               string
}

func loadConfig() config {
	return config{
		GRPCPort:            getEnvInt("ZYNAX_ENGINE_ADAPTER_GRPC_PORT", 50055),
		MetricsPort:         getEnvInt("ZYNAX_ENGINE_ADAPTER_METRICS_PORT", 9095),
		LogLevel:            getEnv("ZYNAX_ENGINE_ADAPTER_LOG_LEVEL", "info"),
		TemporalHostPort:    getEnv("ZYNAX_ENGINE_ADAPTER_TEMPORAL_HOST_PORT", "localhost:7233"),
		TemporalNamespace:   getEnv("ZYNAX_ENGINE_ADAPTER_TEMPORAL_NAMESPACE", "default"),
		TemporalTaskQueue:   getEnv("ZYNAX_ENGINE_ADAPTER_TEMPORAL_TASK_QUEUE", "engine-adapter"),
		TaskBrokerAddr:      getEnv("ZYNAX_ENGINE_ADAPTER_TASK_BROKER_ADDR", "localhost:50053"),
		EventBusAddr:        getEnv("ZYNAX_ENGINE_ADAPTER_EVENTBUS_ADDR", "localhost:50056"),
		ActiveEngine:        getEnv("ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE", "temporal"),
		GRPCCallTimeoutS:    getEnvInt("ZYNAX_ENGINE_ADAPTER_GRPC_CALL_TIMEOUT_S", 30),
		MaxActivityAttempts: getEnvInt32("ZYNAX_ENGINE_MAX_ACTIVITY_ATTEMPTS", 3),
		LivenessThresholdS:  getEnvInt("ZYNAX_ENGINE_ADAPTER_LIVENESS_THRESHOLD_S", 60),
		TLSCert:             getEnv("ZYNAX_TLS_CERT", ""),
		TLSKey:              getEnv("ZYNAX_TLS_KEY", ""),
		TLSCA:               getEnv("ZYNAX_TLS_CA", ""),
	}
}

func main() {
	cfg := loadConfig()
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	})))

	if err := run(cfg); err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

// run contains the service lifecycle. Deferred cleanups execute before returning.
func run(cfg config) error {
	engine, cleanup, brokerConn, err := buildEngine(cfg)
	if err != nil {
		return fmt.Errorf("engine setup: %w", err)
	}
	defer cleanup()

	brokerReadyFn := func() bool {
		s := brokerConn.GetState()
		return s != connectivity.TransientFailure && s != connectivity.Shutdown
	}
	probes := api.NewProbes(int64(cfg.LivenessThresholdS), brokerReadyFn)

	grpcSrv, err := startGRPC(cfg, engine, probes)
	if err != nil {
		return fmt.Errorf("gRPC start: %w", err)
	}

	httpSrv := startHTTP(cfg, probes)

	// Mark started: engine built, Temporal worker running, gRPC server listening.
	probes.MarkStarted()

	slog.Info("engine-adapter started",
		"grpc_port", cfg.GRPCPort,
		"metrics_port", cfg.MetricsPort,
		"engine", cfg.ActiveEngine,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	<-ctx.Done()

	slog.Info("shutting down")
	grpcSrv.GracefulStop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if shutdownErr := httpSrv.Shutdown(shutdownCtx); shutdownErr != nil {
		slog.Error("http shutdown error", "err", shutdownErr)
	}
	slog.Info("shutdown complete")
	return nil
}

// buildEngine creates the WorkflowEngine and its underlying Temporal worker.
// The returned cleanup function stops the worker and closes connections.
// brokerConn is returned separately so main can use it for the readiness probe.
func buildEngine(cfg config) (domain.WorkflowEngine, func(), *grpc.ClientConn, error) {
	if cfg.ActiveEngine != "temporal" {
		return nil, func() {}, nil, fmt.Errorf("unsupported engine %q: only \"temporal\" is supported in M3", cfg.ActiveEngine)
	}

	infrastructure.DefaultActivityMaxAttempts = cfg.MaxActivityAttempts

	tc, err := client.Dial(client.Options{
		HostPort:  cfg.TemporalHostPort,
		Namespace: cfg.TemporalNamespace,
	})
	if err != nil {
		return nil, func() {}, nil, fmt.Errorf("temporal client: %w", err)
	}

	brokerConn, eventBusConn, err := dialGRPCClients(cfg)
	if err != nil {
		tc.Close()
		return nil, func() {}, nil, err
	}

	callTimeout := time.Duration(cfg.GRPCCallTimeoutS) * time.Second
	dispatcher := domain.NewCapabilityDispatcher(zynaxv1.NewTaskBrokerServiceClient(brokerConn), callTimeout)
	activityWorker := &infrastructure.ActivityWorker{
		EventBus: zynaxv1.NewEventBusServiceClient(eventBusConn),
	}

	w := worker.New(tc, cfg.TemporalTaskQueue, worker.Options{})
	w.RegisterWorkflow(infrastructure.IRInterpreterWorkflow)
	w.RegisterActivity(dispatcher.DispatchCapabilityActivity)
	w.RegisterActivity(activityWorker.PublishLifecycleEventActivity)

	if err := w.Start(); err != nil {
		tc.Close()
		_ = brokerConn.Close()
		_ = eventBusConn.Close()
		return nil, func() {}, nil, fmt.Errorf("temporal worker: %w", err)
	}

	cleanup := func() {
		w.Stop()
		tc.Close()
		_ = brokerConn.Close()
		_ = eventBusConn.Close()
	}

	return infrastructure.NewTemporalEngine(tc, cfg.TemporalTaskQueue, cfg.TemporalNamespace), cleanup, brokerConn, nil
}

// dialGRPCClients creates lazy gRPC connections to task-broker and event-bus.
// grpc.NewClient never blocks — connections are established on first use (lazy dial).
func dialGRPCClients(cfg config) (*grpc.ClientConn, *grpc.ClientConn, error) {
	creds, err := infrastructure.TLSCreds(cfg.TLSCert, cfg.TLSKey, cfg.TLSCA)
	if err != nil {
		return nil, nil, fmt.Errorf("tls credentials: %w", err)
	}
	brokerConn, err := grpc.NewClient(cfg.TaskBrokerAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, nil, fmt.Errorf("task-broker dial: %w", err)
	}
	// Dial EventBusService with lazy connection — a non-reachable event bus must
	// not prevent startup. grpc.NewClient defers connection until first RPC call.
	eventBusCreds, err := infrastructure.TLSCreds(cfg.TLSCert, cfg.TLSKey, cfg.TLSCA)
	if err != nil {
		_ = brokerConn.Close()
		return nil, nil, fmt.Errorf("event-bus tls credentials: %w", err)
	}
	eventBusConn, err := grpc.NewClient(cfg.EventBusAddr, grpc.WithTransportCredentials(eventBusCreds))
	if err != nil {
		_ = brokerConn.Close()
		return nil, nil, fmt.Errorf("event-bus dial: %w", err)
	}
	return brokerConn, eventBusConn, nil
}

func startGRPC(cfg config, engine domain.WorkflowEngine, probes *api.Probes) (*grpc.Server, error) {
	serverCreds, err := infrastructure.TLSCreds(cfg.TLSCert, cfg.TLSKey, cfg.TLSCA)
	if err != nil {
		return nil, fmt.Errorf("tls credentials: %w", err)
	}
	srv := grpc.NewServer(
		grpc.Creds(serverCreds),
		grpc.ChainUnaryInterceptor(makeRequestIDInterceptor(probes)),
	)
	reflection.Register(srv)

	healthSvc := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSvc)
	healthSvc.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	zynaxv1.RegisterEngineAdapterServiceServer(srv, api.NewHandler(engine))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		return nil, fmt.Errorf("listen :%d: %w", cfg.GRPCPort, err)
	}

	go func() {
		if serveErr := srv.Serve(lis); serveErr != nil {
			slog.Error("grpc serve error", "err", serveErr)
		}
	}()

	return srv, nil
}

func startHTTP(cfg config, probes *api.Probes) *http.Server {
	mux := http.NewServeMux()
	probes.Register(mux)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.MetricsPort),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server error", "err", err)
		}
	}()
	return srv
}

// makeRequestIDInterceptor returns a gRPC unary interceptor that propagates
// request-id metadata and calls probes.RecordWork() after each successful call.
func makeRequestIDInterceptor(probes *api.Probes) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("request-id"); len(vals) > 0 {
				slog.Info("grpc request", "method", info.FullMethod, "request_id", vals[0])
			}
		}
		resp, err := handler(ctx, req)
		if err == nil {
			probes.RecordWork()
		}
		return resp, err
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvInt32(key string, fallback int32) int32 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			return int32(n) //nolint:gosec // G115: ParseInt with bitSize=32 guarantees value fits in int32
		}
	}
	return fallback
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
