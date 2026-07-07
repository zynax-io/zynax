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
	"strings"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/zynax-io/zynax/libs/zynaxobs"
	"github.com/zynax-io/zynax/services/api-gateway/internal/api"
	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
	"github.com/zynax-io/zynax/services/api-gateway/internal/infrastructure"
	"github.com/zynax-io/zynax/services/api-gateway/internal/infrastructure/crd"
)

type config struct {
	HTTPPort           int    `envconfig:"HTTP_PORT" default:"8080"`
	CompilerAddr       string `envconfig:"COMPILER_ADDR" default:"localhost:50054"`
	EngineAddr         string `envconfig:"ENGINE_ADDR" default:"localhost:50055"`
	RegistryAddr       string `envconfig:"REGISTRY_ADDR" default:"localhost:50052"`
	NATSURL            string `envconfig:"NATS_URL" default:"nats://localhost:4222"`
	LogLevel           string `envconfig:"LOG_LEVEL" default:"info"`
	GRPCCallTimeoutS   int    `envconfig:"GRPC_CALL_TIMEOUT_S" default:"30"`
	LivenessThresholdS int    `envconfig:"LIVENESS_THRESHOLD_S" default:"60"`
	TLSCert            string `envconfig:"TLS_CERT"`
	TLSKey             string `envconfig:"TLS_KEY"`
	TLSCA              string `envconfig:"TLS_CA"`
	// JetStream client identity (ADR-046 Decision #4), decoupled from the
	// service-wide TLS flag so the broker can be fail-closed mTLS while the
	// gRPC mesh profile stays unchanged. Falls back to TLS_* when unset.
	EventsTLSCert string `envconfig:"EVENTS_TLS_CERT"`
	EventsTLSKey  string `envconfig:"EVENTS_TLS_KEY"`
	EventsTLSCA   string `envconfig:"EVENTS_TLS_CA"`
	// Embedded Workflow CRD controller (ADR-043, M8.E). Off by default: when
	// disabled, api-gateway serves only the REST apply path and starts no
	// controller-runtime manager. When enabled, a namespaced, Lease-elected
	// manager reconciles Workflow CRs (zynax.io/v1alpha1) through the existing
	// compile->submit path. Requires the Workflow CRD + controller RBAC shipped
	// by the chart (#1610).
	CRDControllerEnabled bool   `envconfig:"CRD_CONTROLLER_ENABLED" default:"false"`
	WatchNamespace       string `envconfig:"WATCH_NAMESPACE"`
	ElectionNamespace    string `envconfig:"ELECTION_NAMESPACE"`
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
	// Bearer auth is enforced at the Gateway API edge (ADR-044/M8.F), not here;
	// the api-gateway is reachable only through that edge (NetworkPolicy lockdown).
	if err := run(cfg); err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

// run contains the service lifecycle. Deferred cleanups execute before returning.
func run(cfg config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	callTimeout := time.Duration(cfg.GRPCCallTimeoutS) * time.Second
	eventsTLSCert, eventsTLSKey, eventsTLSCA := cfg.EventsTLSCert, cfg.EventsTLSKey, cfg.EventsTLSCA
	if eventsTLSCert == "" {
		eventsTLSCert, eventsTLSKey, eventsTLSCA = cfg.TLSCert, cfg.TLSKey, cfg.TLSCA
	}
	clients, cleanup, err := infrastructure.NewGatewayClients(cfg.CompilerAddr, cfg.EngineAddr, cfg.RegistryAddr, cfg.NATSURL, callTimeout, cfg.TLSCert, cfg.TLSKey, cfg.TLSCA, eventsTLSCert, eventsTLSKey, eventsTLSCA)
	if err != nil {
		return fmt.Errorf("gateway clients: %w", err)
	}
	defer cleanup()

	probes := api.NewProbes(int64(cfg.LivenessThresholdS), clients.ConnectionsReady)

	svc := domain.NewApplyService(clients, clients, clients, clients)
	handler := api.NewHandler(svc)

	// Start the embedded Workflow CRD controller when enabled. It reconciles
	// Workflow CRs through the same ApplyService the REST path uses; a failure
	// here is fatal to startup, but the manager itself is fault-isolated (its
	// goroutine logs and exits without taking down the HTTP surface).
	if cfg.CRDControllerEnabled {
		if err := crd.StartController(ctx, svc, resolveWatchNamespace(cfg)); err != nil {
			return fmt.Errorf("api-gateway: workflow controller: %w", err)
		}
	}

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

	return serveUntilShutdown(ctx, srv, cfg.HTTPPort)
}

// resolveWatchNamespace picks the controller's namespace scope: explicit env,
// else the in-cluster ServiceAccount namespace, else the election namespace
// (local dev). NewManager rejects an empty result — a cluster-scope watch is
// never used (namespaced RBAC). Mirrors the agent-registry scheduler.
func resolveWatchNamespace(cfg config) string {
	if cfg.WatchNamespace != "" {
		return cfg.WatchNamespace
	}
	if b, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(b)); ns != "" {
			return ns
		}
	}
	return cfg.ElectionNamespace
}

func serveUntilShutdown(ctx context.Context, srv *http.Server, port int) error {
	go func() {
		slog.Info("api-gateway started", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http serve error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	// The drain context is deliberately NOT derived from ctx: ctx is already
	// cancelled here (that is what woke us), so an inherited context would make
	// Shutdown return immediately instead of draining for up to 10s.
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil { //nolint:contextcheck // graceful-drain context must outlive the cancelled parent
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

// Flush forwards to the underlying ResponseWriter's Flusher so streaming
// endpoints (GET /workflows/{id}/logs SSE) still work behind this wrapper.
// Without it, the wrapper hides the embedded Flusher and the logs handler's
// w.(http.Flusher) assertion fails with HTTP 500 "streaming not supported".
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap exposes the wrapped ResponseWriter to http.ResponseController so the
// logs handler can reset the write deadline for long-running streams.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
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
