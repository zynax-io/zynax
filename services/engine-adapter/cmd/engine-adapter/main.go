// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the engine-adapter service.
// It wires the Temporal worker (IRInterpreterWorkflow + activities) and the
// gRPC server (EngineAdapterService). All business logic lives in internal/.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
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
	"google.golang.org/grpc/credentials/insecure"
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
	ActiveEngine        string
	GRPCCallTimeoutS    int
	MaxActivityAttempts int
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
		ActiveEngine:        getEnv("ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE", "temporal"),
		GRPCCallTimeoutS:    getEnvInt("ZYNAX_ENGINE_ADAPTER_GRPC_CALL_TIMEOUT_S", 30),
		MaxActivityAttempts: getEnvInt("ZYNAX_ENGINE_MAX_ACTIVITY_ATTEMPTS", 3),
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
	engine, cleanup, err := buildEngine(cfg)
	if err != nil {
		return fmt.Errorf("engine setup: %w", err)
	}
	defer cleanup()

	grpcSrv, err := startGRPC(cfg, engine)
	if err != nil {
		return fmt.Errorf("gRPC start: %w", err)
	}

	httpSrv := startHTTP(cfg)

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
func buildEngine(cfg config) (domain.WorkflowEngine, func(), error) {
	if cfg.ActiveEngine != "temporal" {
		return nil, func() {}, fmt.Errorf("unsupported engine %q: only \"temporal\" is supported in M3", cfg.ActiveEngine)
	}

	attempts := cfg.MaxActivityAttempts
	if attempts > math.MaxInt32 {
		attempts = math.MaxInt32
	}
	infrastructure.DefaultActivityMaxAttempts = int32(attempts) //nolint:gosec // G115: bounded above by MaxInt32 check

	tc, err := client.Dial(client.Options{
		HostPort:  cfg.TemporalHostPort,
		Namespace: cfg.TemporalNamespace,
	})
	if err != nil {
		return nil, func() {}, fmt.Errorf("temporal client: %w", err)
	}

	brokerConn, err := grpc.NewClient(
		cfg.TaskBrokerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		tc.Close()
		return nil, func() {}, fmt.Errorf("task-broker dial: %w", err)
	}

	callTimeout := time.Duration(cfg.GRPCCallTimeoutS) * time.Second
	dispatcher := domain.NewCapabilityDispatcher(zynaxv1.NewTaskBrokerServiceClient(brokerConn), callTimeout)

	w := worker.New(tc, cfg.TemporalTaskQueue, worker.Options{})
	w.RegisterWorkflow(infrastructure.IRInterpreterWorkflow)
	w.RegisterActivity(dispatcher.DispatchCapabilityActivity)
	w.RegisterActivity(infrastructure.PublishLifecycleEventActivity)

	if err := w.Start(); err != nil {
		tc.Close()
		_ = brokerConn.Close()
		return nil, func() {}, fmt.Errorf("temporal worker: %w", err)
	}

	cleanup := func() {
		w.Stop()
		tc.Close()
		_ = brokerConn.Close()
	}

	return infrastructure.NewTemporalEngine(tc, cfg.TemporalTaskQueue, cfg.TemporalNamespace), cleanup, nil
}

func startGRPC(cfg config, engine domain.WorkflowEngine) (*grpc.Server, error) {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(requestIDServerInterceptor),
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

func startHTTP(cfg config) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/startupz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

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

func requestIDServerInterceptor(
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
	return handler(ctx, req)
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
