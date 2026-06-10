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
		nb, err := infrastructure.NewNATSEventBus(cfg.NATSUrl)
		if err != nil {
			return fmt.Errorf("event-bus: nats: %w", err)
		}
		defer nb.Close()
		bus = nb
	}

	handler := api.NewHandler(bus)

	srv := grpc.NewServer(
		grpc.Creds(creds),
		grpc.StatsHandler(zynaxobs.TracingStatsHandler()),
		grpc.ChainUnaryInterceptor(zynaxobs.MetricsUnaryInterceptor("event-bus")),
	)
	reflection.Register(srv)
	zynaxv1.RegisterEventBusServiceServer(srv, handler)

	healthSvc := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSvc)
	healthSvc.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

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
	srv.GracefulStop()
	return nil
}
