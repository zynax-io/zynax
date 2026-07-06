// Package config loads workflow-compiler configuration from environment variables.
package config

import (
	"fmt"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain"
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

	// ── Policy configuration (M6; quota removed by ADR-045 in M8) ─────────
	// The engine allow-list is read from env vars — the REST-path dual-guard
	// of ADR-045 §3. The ZYNAX_POLICY_MAX_CONCURRENT quota knob was removed:
	// the compiler quota check was never enforced in production (nil counter)
	// and quota is unenforced until the engine-adapter QuotaChecker is wired.

	// PolicyNamespace is the namespace these policy settings apply to.
	// Leave empty to disable policy enforcement for all namespaces.
	PolicyNamespace string `envconfig:"ZYNAX_POLICY_NAMESPACE"`

	// PolicyAllowedEngines is a comma-separated list of engine identifiers
	// that the namespace is allowed to use (e.g. "temporal,argo").
	// An empty value means "no restriction" (any engine is permitted).
	// Only evaluated when ZYNAX_POLICY_NAMESPACE is set.
	PolicyAllowedEngines string `envconfig:"ZYNAX_POLICY_ALLOWED_ENGINES"`
}

// Load reads Config from environment variables.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return &cfg, nil
}

// PolicyGates returns the routing-policy configs derived from the
// environment-variable-backed Config. Returns a nil slice when policy
// enforcement is disabled (ZYNAX_POLICY_NAMESPACE is unset).
//
// Only a single namespace policy is supported. Multi-namespace policy for the
// CR path is expressed as multiple ValidatingAdmissionPolicyBindings (ADR-045).
func (c *Config) PolicyGates() []domain.RoutingPolicyConfig {
	if c.PolicyNamespace == "" {
		return nil
	}

	var engines []string
	if c.PolicyAllowedEngines != "" {
		for _, e := range strings.Split(c.PolicyAllowedEngines, ",") {
			e = strings.TrimSpace(e)
			if e != "" {
				engines = append(engines, e)
			}
		}
	}

	return []domain.RoutingPolicyConfig{{
		Namespace:      c.PolicyNamespace,
		AllowedEngines: engines,
	}}
}
