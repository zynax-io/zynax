// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the event-bus service.
// It connects to NATS JetStream (when ZYNAX_EVENTBUS_NATS_URL is set),
// wires the gRPC server, and registers EventBusService handlers.
// All business logic lives in internal/.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	nats "github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/zynax-io/zynax/libs/zynaxconfig"
	"github.com/zynax-io/zynax/libs/zynaxobs"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/event-bus/internal/api"
	"github.com/zynax-io/zynax/services/event-bus/internal/domain"
	"github.com/zynax-io/zynax/services/event-bus/internal/infrastructure"
)

type config struct {
	zynaxconfig.Base
	TLSCert              string `envconfig:"TLS_CERT"`
	TLSKey               string `envconfig:"TLS_KEY"`
	TLSCA                string `envconfig:"TLS_CA"`
	NATSUrl              string `envconfig:"NATS_URL"`
	StreamRetentionHours int    `envconfig:"STREAM_RETENTION_HOURS" default:"24"`
	DLQMaxRetries        int    `envconfig:"DLQ_MAX_RETRIES" default:"5"`
}

func main() {
	cfg := config{}
	cfg.GRPCPort = 50054 // ZYNAX_EVENTBUS_GRPC_PORT overrides
	if err := zynaxconfig.Load("EVENTBUS", &cfg); err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}
	zynaxconfig.SetDefaultLogger(cfg.LogLevel)
	if err := run(cfg); err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

func run(cfg config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	tracerShutdown, err := zynaxobs.InitTracer(ctx, "event-bus")
	if err != nil {
		return fmt.Errorf("event-bus: tracer init: %w", err)
	}
	defer func() { _ = tracerShutdown(context.Background()) }()

	metricsSrv := zynaxobs.StartMetricsServer(cfg.HealthPort)
	defer func() { _ = metricsSrv.Shutdown(context.Background()) }()

	creds, err := infrastructure.TLSCreds(cfg.TLSCert, cfg.TLSKey, cfg.TLSCA)
	if err != nil {
		return fmt.Errorf("event-bus: tls credentials: %w", err)
	}

	var bus domain.EventBus
	if cfg.NATSUrl != "" {
		nb, err := infrastructure.NewNATSEventBus(cfg.NATSUrl, natsTLSOptions(cfg)...)
		if err != nil {
			return fmt.Errorf("event-bus: nats: %w", err)
		}
		defer nb.Close()
		bus = nb
	}

	handler := api.NewHandler(bus)

	tracingUnary, tracingStream := zynaxobs.TracingServerInterceptors()
	srv := grpc.NewServer(
		grpc.Creds(creds),
		grpc.StatsHandler(zynaxobs.TracingStatsHandler()),
		grpc.ChainUnaryInterceptor(tracingUnary, zynaxobs.MetricsUnaryInterceptor("event-bus")),
		grpc.ChainStreamInterceptor(tracingStream),
	)
	reflection.Register(srv)
	zynaxv1.RegisterEventBusServiceServer(srv, handler)

	healthSvc := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSvc)
	setHealth(healthSvc, grpc_health_v1.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("event-bus: listen: %w", err)
	}

	go func() {
		slog.Info("event-bus started", "grpc_port", cfg.GRPCPort)
		if err := srv.Serve(lis); err != nil {
			slog.Error("grpc serve error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	// Drain: report NOT_SERVING so load balancers stop routing before the
	// graceful stop completes (canvas O-step 2, #656).
	setHealth(healthSvc, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	srv.GracefulStop()
	return nil
}

// setHealth sets both the overall "" key and the per-service named key to the
// given serving status (canvas O-step 2, #656).
func setHealth(h *health.Server, st grpc_health_v1.HealthCheckResponse_ServingStatus) {
	h.SetServingStatus("", st)
	h.SetServingStatus(zynaxv1.EventBusService_ServiceDesc.ServiceName, st)
}

// natsTLSOptions returns the client-certificate dial options when the facade
// has a TLS identity configured — it dials the TLS/verify_and_map broker with
// its own cert-manager identity (ADR-046), the same PEMs as gRPC mTLS.
func natsTLSOptions(cfg config) []nats.Option {
	if cfg.TLSCert == "" {
		return nil
	}
	return []nats.Option{
		nats.ClientCert(cfg.TLSCert, cfg.TLSKey),
		nats.RootCAs(cfg.TLSCA),
	}
}
