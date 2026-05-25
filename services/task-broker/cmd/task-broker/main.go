// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the task-broker service.
// It wires the in-memory repository, agent registry finder, agent executor,
// and gRPC server. All business logic lives in internal/.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/task-broker/internal/api"
	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
	"github.com/zynax-io/zynax/services/task-broker/internal/infrastructure"
)

type config struct {
	GRPCPort         int    `envconfig:"GRPC_PORT" default:"50053"`
	RegistryAddr     string `envconfig:"REGISTRY_ADDR" default:"localhost:50052"`
	LogLevel         string `envconfig:"LOG_LEVEL" default:"info"`
	GRPCCallTimeoutS int    `envconfig:"GRPC_CALL_TIMEOUT_S" default:"30"`
}

func main() {
	var cfg config
	if err := envconfig.Process("ZYNAX_BROKER", &cfg); err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	})))
	if err := run(cfg); err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

func run(cfg config) error {
	callTimeout := time.Duration(cfg.GRPCCallTimeoutS) * time.Second
	finder, finderCleanup, err := infrastructure.NewRegistryClient(cfg.RegistryAddr, callTimeout)
	if err != nil {
		return fmt.Errorf("task-broker: registry client: %w", err)
	}
	defer finderCleanup()

	repo := infrastructure.NewMemoryRepo()
	executor := infrastructure.NewAgentExecutor()
	svc := domain.NewTaskService(repo, finder, executor)

	srv := grpc.NewServer()
	reflection.Register(srv)
	zynaxv1.RegisterTaskBrokerServiceServer(srv, api.NewHandler(svc))

	healthSvc := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSvc)
	healthSvc.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("task-broker: listen: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		slog.Info("task-broker started", "grpc_port", cfg.GRPCPort)
		if err := srv.Serve(lis); err != nil {
			slog.Error("grpc serve error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	srv.GracefulStop()
	return nil
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
