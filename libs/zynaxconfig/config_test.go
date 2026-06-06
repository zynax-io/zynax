// SPDX-License-Identifier: Apache-2.0

package zynaxconfig_test

import (
	"log/slog"
	"testing"

	"github.com/zynax-io/zynax/libs/zynaxconfig"
)

func TestLoad_defaults(t *testing.T) {
	type cfg struct {
		zynaxconfig.Base
		Extra string `envconfig:"EXTRA" default:"hello"`
	}

	var c cfg
	c.GRPCPort = 9999 // service-specific default; no env var set → stays 9999
	if err := zynaxconfig.Load("TESTDEFAULT", &c); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.GRPCPort != 9999 {
		t.Errorf("GRPCPort: got %d, want 9999", c.GRPCPort)
	}
	if c.HealthPort != 9090 {
		t.Errorf("HealthPort: got %d, want 9090", c.HealthPort)
	}
	if c.LogLevel != "info" {
		t.Errorf("LogLevel: got %q, want \"info\"", c.LogLevel)
	}
	if c.Extra != "hello" {
		t.Errorf("Extra: got %q, want \"hello\"", c.Extra)
	}
}

func TestLoad_envOverride(t *testing.T) {
	type cfg struct {
		zynaxconfig.Base
	}

	t.Setenv("ZYNAX_TESTENV_GRPC_PORT", "12345")
	t.Setenv("ZYNAX_TESTENV_LOG_LEVEL", "debug")
	t.Setenv("ZYNAX_TESTENV_HEALTH_PORT", "9191")

	var c cfg
	c.GRPCPort = 50099
	if err := zynaxconfig.Load("TESTENV", &c); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.GRPCPort != 12345 {
		t.Errorf("GRPCPort: got %d, want 12345", c.GRPCPort)
	}
	if c.LogLevel != "debug" {
		t.Errorf("LogLevel: got %q, want \"debug\"", c.LogLevel)
	}
	if c.HealthPort != 9191 {
		t.Errorf("HealthPort: got %d, want 9191", c.HealthPort)
	}
}

func TestParseLogLevel(t *testing.T) {
	cases := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"info", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"unknown", slog.LevelInfo},
	}
	for _, tc := range cases {
		got := zynaxconfig.ParseLogLevel(tc.in)
		if got != tc.want {
			t.Errorf("ParseLogLevel(%q): got %v, want %v", tc.in, got, tc.want)
		}
	}
}
