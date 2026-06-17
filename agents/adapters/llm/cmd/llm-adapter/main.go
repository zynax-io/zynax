// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the llm-adapter gRPC service. This
// scaffold (M7.P.2) loads and validates configuration and resolves the API-key
// secret from the environment; provider routing, the AgentService server, and
// registry bootstrap are added in later EPIC P steps (P.3–P.5).
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/zynax-io/zynax/agents/adapters/llm/internal/config"
)

// configEnvVar names the env var holding the YAML config path (prefix ZYNAX_LLM_).
const configEnvVar = "ZYNAX_LLM_CONFIG"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("llm-adapter error", "err", err)
		os.Exit(1)
	}
}

// run loads config, resolves the credential, and logs readiness. The gRPC serve
// loop is wired in a later P step; keeping run() pure makes it test-friendly.
func run() error {
	cfgPath := os.Getenv(configEnvVar)
	if cfgPath == "" {
		return fmt.Errorf("%s env var is required", configEnvVar)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if _, err := cfg.ResolveSecret(); err != nil {
		return fmt.Errorf("resolve secret: %w", err)
	}
	// Fields are operator-controlled config (not request input); the API-key
	// Secret is never logged. //nolint:gosec — matches sibling git-adapter.
	slog.Info("llm-adapter config loaded", //nolint:gosec
		"agent_id", cfg.AgentID,
		"provider", cfg.Provider.Name,
		"endpoint", cfg.Endpoint,
		"capabilities", len(cfg.Capabilities),
	)
	return nil
}
