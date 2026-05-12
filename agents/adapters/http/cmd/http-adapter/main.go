// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the http-adapter gRPC service.
// Business logic lives in internal/; full bootstrap is wired in step #396.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
)

// compile-time check: agentServer satisfies AgentServiceServer.
// Replaced by internal/adapter.AgentServer in step #396.
var _ zynaxv1.AgentServiceServer = (*agentServer)(nil)

type agentServer struct {
	zynaxv1.UnimplementedAgentServiceServer
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("http-adapter error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	srv := grpc.NewServer()
	zynaxv1.RegisterAgentServiceServer(srv, &agentServer{})
	<-ctx.Done()
	srv.GracefulStop()
	slog.Info("http-adapter stopped")
	return nil
}
