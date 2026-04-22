// Package config loads workflow-compiler configuration from environment variables.
package config

import "github.com/kelseyhightower/envconfig"

// Config holds all runtime configuration for the workflow-compiler service.
type Config struct {
	// GRPCPort is the port the gRPC server listens on.
	GRPCPort int `envconfig:"ZYNAX_WC_PORT" default:"50054"`
	// MetricsPort is the port for /healthz and /metrics.
	MetricsPort int `envconfig:"ZYNAX_WC_METRICS_PORT" default:"9094"`
	// LogLevel controls structured log verbosity (debug, info, warn, error).
	LogLevel string `envconfig:"ZYNAX_WC_LOG_LEVEL" default:"info"`
}

// Load reads Config from environment variables.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
