// SPDX-License-Identifier: Apache-2.0

// Package main (whitebox) tests the startup least-privilege scope gate wiring
// (G.5 / #1260). A package-wide TestMain installs a no-network fake probe so the
// pre-existing run()/serve() tests never reach the real GitHub API; individual
// tests below swap the probe to exercise enforce/warn/error branches.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/auth"
)

// fakeProbe returns a fixed header/error and never touches the network.
type fakeProbe struct {
	hdr http.Header
	err error
}

func (f fakeProbe) Probe(_ context.Context) (http.Header, error) { return f.hdr, f.err }

func scopeHdr(scopes string, set bool) http.Header {
	h := http.Header{}
	if set {
		h.Set("X-OAuth-Scopes", scopes)
	}
	return h
}

// TestMain defaults newScopeProbe to a passing fine-grained fake for the whole
// package, then restores it. This keeps the legacy run() tests offline.
func TestMain(m *testing.M) {
	orig := newScopeProbe
	newScopeProbe = func(string) (scopeValidator, error) {
		return fakeProbe{hdr: scopeHdr("", false)}, nil
	}
	code := m.Run()
	newScopeProbe = orig
	os.Exit(code)
}

func TestValidateTokenScope_EnforceRejects(t *testing.T) {
	p := fakeProbe{hdr: scopeHdr("repo", true)}
	if err := validateTokenScope(context.Background(), p, auth.ModeEnforce); err == nil {
		t.Fatal("enforce mode must reject an over-broad token")
	} else if !errors.Is(err, auth.ErrOverBroadScope) {
		t.Fatalf("expected ErrOverBroadScope, got %v", err)
	}
}

func TestValidateTokenScope_WarnAllows(t *testing.T) {
	p := fakeProbe{hdr: scopeHdr("repo", true)}
	if err := validateTokenScope(context.Background(), p, auth.ModeWarn); err != nil {
		t.Fatalf("warn mode must not fail startup: %v", err)
	}
}

func TestValidateTokenScope_FineGrainedPasses(t *testing.T) {
	p := fakeProbe{hdr: scopeHdr("", false)}
	if err := validateTokenScope(context.Background(), p, auth.ModeEnforce); err != nil {
		t.Fatalf("fine-grained token must pass: %v", err)
	}
}

func TestValidateTokenScope_NarrowClassicPasses(t *testing.T) {
	p := fakeProbe{hdr: scopeHdr("public_repo, read:user", true)}
	if err := validateTokenScope(context.Background(), p, auth.ModeEnforce); err != nil {
		t.Fatalf("narrow classic token must pass: %v", err)
	}
}

func TestValidateTokenScope_ProbeErrorIsNonFatal(t *testing.T) {
	// A probe transport failure must not block startup — it warns and proceeds.
	p := fakeProbe{err: errors.New("network down")}
	if err := validateTokenScope(context.Background(), p, auth.ModeEnforce); err != nil {
		t.Fatalf("probe transport error must be non-fatal: %v", err)
	}
}

// TestRun_OverBroadTokenFailsStartup wires the over-broad probe through run() to
// confirm the gate aborts before any registry dial.
func TestRun_OverBroadTokenFailsStartup(t *testing.T) {
	orig := newScopeProbe
	newScopeProbe = func(string) (scopeValidator, error) {
		return fakeProbe{hdr: scopeHdr("repo", true)}, nil
	}
	defer func() { newScopeProbe = orig }()

	dir := t.TempDir()
	cfgPath := dir + "/git-adapter.yaml"
	writeScopeCfg(t, cfgPath)
	t.Setenv("ADAPTER_CONFIG", cfgPath)
	t.Setenv("GIT_TOKEN_SCOPE_1260", "fake-token-value-1260")
	t.Setenv("GIT_ADAPTER_SCOPE_MODE", "enforce")

	if err := run(); err == nil {
		t.Fatal("expected run() to fail for an over-broad token in enforce mode")
	} else if !errors.Is(err, auth.ErrOverBroadScope) {
		t.Fatalf("expected ErrOverBroadScope, got %v", err)
	}
}

// TestNewScopeProbe_Default exercises the production factory (no network call is
// made — only client construction).
func TestNewScopeProbe_Default(t *testing.T) {
	orig := newScopeProbe
	newScopeProbe = orig // ensure default
	p, err := orig("some-token-value-1260")
	if err != nil {
		t.Fatalf("default probe factory: %v", err)
	}
	if p == nil {
		t.Fatal("default probe factory returned nil probe")
	}
}

func writeScopeCfg(t *testing.T, path string) {
	t.Helper()
	const body = "agent_id: git-test\nname: Git Test\n" +
		"endpoint: \"127.0.0.1:0\"\nregistry_endpoint: \"127.0.0.1:9090\"\n" +
		"git:\n  provider: github\n  auth_env: GIT_TOKEN_SCOPE_1260\n" +
		"capabilities:\n  - name: open_pr\n    owner: o\n    repo: r\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
