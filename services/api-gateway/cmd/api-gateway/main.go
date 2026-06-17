// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the api-gateway service.
// It wires gRPC clients to WorkflowCompilerService and EngineAdapterService,
// creates the domain ApplyService, and starts the HTTP server.
// All business logic lives in internal/.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/zynax-io/zynax/libs/zynaxobs"
	"github.com/zynax-io/zynax/services/api-gateway/internal/api"
	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
	"github.com/zynax-io/zynax/services/api-gateway/internal/infrastructure"
)

type config struct {
	HTTPPort           int    `envconfig:"HTTP_PORT" default:"8080"`
	CompilerAddr       string `envconfig:"COMPILER_ADDR" default:"localhost:50054"`
	EngineAddr         string `envconfig:"ENGINE_ADDR" default:"localhost:50055"`
	RegistryAddr       string `envconfig:"REGISTRY_ADDR" default:"localhost:50052"`
	EventBusAddr       string `envconfig:"EVENT_BUS_ADDR" default:"localhost:50056"`
	LogLevel           string `envconfig:"LOG_LEVEL" default:"info"`
	APIKey             string `envconfig:"API_KEY"`
	DevInsecure        bool   `envconfig:"DEV_INSECURE"`
	GRPCCallTimeoutS   int    `envconfig:"GRPC_CALL_TIMEOUT_S" default:"30"`
	LivenessThresholdS int    `envconfig:"LIVENESS_THRESHOLD_S" default:"60"`
	TLSCert            string `envconfig:"TLS_CERT"`
	TLSKey             string `envconfig:"TLS_KEY"`
	TLSCA              string `envconfig:"TLS_CA"`
}

// validateConfig rejects an empty API key unless ZYNAX_GW_DEV_INSECURE=1 is set.
// Keeps production deployments from silently accepting all requests on misconfiguration.
func validateConfig(cfg config) error {
	if cfg.APIKey == "" && !cfg.DevInsecure {
		return fmt.Errorf(
			"ZYNAX_GW_API_KEY is not set; refusing to start " +
				"(set ZYNAX_GW_DEV_INSECURE=1 to allow an empty key in development)",
		)
	}
	return nil
}

func main() {
	var cfg config
	if err := envconfig.Process("ZYNAX_GW", &cfg); err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	})))
	if err := validateConfig(cfg); err != nil {
		slog.Error("startup validation failed", "err", err)
		os.Exit(1)
	}
	if cfg.APIKey == "" {
		slog.Warn("ZYNAX_GW_API_KEY not set — auth disabled (dev-insecure mode)")
	}
	if err := run(cfg); err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

// run contains the service lifecycle. Deferred cleanups execute before returning.
func run(cfg config) error {
	callTimeout := time.Duration(cfg.GRPCCallTimeoutS) * time.Second
	clients, cleanup, err := infrastructure.NewGatewayClients(cfg.CompilerAddr, cfg.EngineAddr, cfg.RegistryAddr, cfg.EventBusAddr, callTimeout, cfg.TLSCert, cfg.TLSKey, cfg.TLSCA)
	if err != nil {
		return fmt.Errorf("gateway clients: %w", err)
	}
	defer cleanup()

	probes := api.NewProbes(int64(cfg.LivenessThresholdS), clients.ConnectionsReady)

	svc := domain.NewApplyService(clients, clients, clients, clients)
	handler := api.NewHandler(svc, cfg.APIKey)

	tracerShutdown, err := zynaxobs.InitTracer(context.Background(), "api-gateway")
	if err != nil {
		return fmt.Errorf("api-gateway: tracer init: %w", err)
	}
	defer func() { _ = tracerShutdown(context.Background()) }()

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	probes.Register(mux)
	zynaxobs.RegisterMetrics(mux)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           zynaxobs.HTTPMiddleware("api-gateway", zynaxobs.MetricsHTTPMiddleware("api-gateway", maxBodyMiddleware(api.RequestIDMiddleware(workRecordMiddleware(probes, mux))))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Mark the service started: config parsed, clients dialed, server starting.
	probes.MarkStarted()

	return serveUntilShutdown(srv, cfg.HTTPPort)
}

func serveUntilShutdown(srv *http.Server, port int) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		slog.Info("api-gateway started", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http serve error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		return fmt.Errorf("api-gateway: shutdown: %w", err)
	}
	return nil
}

// workRecordMiddleware calls probes.RecordWork() after any non-probe request
// that completes with a 2xx HTTP status code.
func workRecordMiddleware(probes *api.Probes, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/startupz", "/readyz", "/livez", "/healthz", "/metrics":
			next.ServeHTTP(w, r)
			return
		}
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rec, r)
		if rec.code >= 200 && rec.code < 300 {
			probes.RecordWork()
		}
	})
}

// statusRecorder captures the HTTP status code written by a handler.
type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

func maxBodyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB
		next.ServeHTTP(w, r)
	})
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
