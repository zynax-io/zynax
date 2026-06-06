// SPDX-License-Identifier: Apache-2.0

// Package zynaxconfig provides shared configuration primitives for all Zynax platform services.
// Each service embeds Base and declares only its own extra fields, then calls Load.
package zynaxconfig

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/kelseyhightower/envconfig"
)

// Base holds fields present in every Zynax platform service.
// Services embed this struct; GRPCPort should be pre-initialised to the
// service-specific default before calling Load (no tag default — each service differs).
type Base struct {
	LogLevel   string `envconfig:"LOG_LEVEL"   default:"info"`
	GRPCPort   int    `envconfig:"GRPC_PORT"`
	HealthPort int    `envconfig:"HEALTH_PORT" default:"9090"`
}

// Load processes environment variables with prefix "ZYNAX_<service>" into dst.
// dst must be a pointer to a struct that embeds Base.
// Standard env-var grammar: ZYNAX_<SERVICE>_<FIELD>.
func Load[T any](service string, dst *T) error {
	if err := envconfig.Process("ZYNAX_"+service, dst); err != nil {
		return fmt.Errorf("zynaxconfig: load %s: %w", service, err)
	}
	return nil
}

// ParseLogLevel converts a level string to slog.Level.
// Unknown values fall back to slog.LevelInfo.
func ParseLogLevel(level string) slog.Level {
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

// SetDefaultLogger configures the global slog logger with JSON output at the given level.
func SetDefaultLogger(level string) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: ParseLogLevel(level),
	})))
}
