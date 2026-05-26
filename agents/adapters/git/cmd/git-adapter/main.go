// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the git-adapter gRPC service.
// Config path from ADAPTER_CONFIG env var; registry endpoint from config.
// Full bootstrap wiring is added in O4 (#402).
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("git-adapter error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfgPath := os.Getenv("ADAPTER_CONFIG")
	if cfgPath == "" {
		return fmt.Errorf("ADAPTER_CONFIG env var is required")
	}
	_, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	// Full gRPC server bootstrap added in O4 (#402).
	return fmt.Errorf("git-adapter not yet fully bootstrapped — O4 (#402) pending")
}
