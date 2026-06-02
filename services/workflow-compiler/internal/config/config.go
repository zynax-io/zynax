// Package config loads workflow-compiler configuration from environment variables.
package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all runtime configuration for the workflow-compiler service.
type Config struct {
	// GRPCPort is the port the gRPC server listens on.
	GRPCPort int `envconfig:"ZYNAX_WC_PORT" default:"50054"`
	// MetricsPort is the port for /healthz and /metrics.
	MetricsPort int `envconfig:"ZYNAX_WC_METRICS_PORT" default:"9094"`
	// LogLevel controls structured log verbosity (debug, info, warn, error).
	LogLevel string `envconfig:"ZYNAX_WC_LOG_LEVEL" default:"info"`
	// TLSCert is the path to the service TLS certificate PEM file.
	TLSCert string `envconfig:"ZYNAX_TLS_CERT"`
	// TLSKey is the path to the service TLS private key PEM file.
	TLSKey string `envconfig:"ZYNAX_TLS_KEY"`
	// TLSCA is the path to the CA certificate bundle PEM file for verifying peers.
	TLSCA string `envconfig:"ZYNAX_TLS_CA"`
}

// Load reads Config from environment variables.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return &cfg, nil
}
