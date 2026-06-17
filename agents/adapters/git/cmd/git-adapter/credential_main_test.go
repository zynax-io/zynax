// SPDX-License-Identifier: Apache-2.0

// Whitebox coverage for the G.7 (#1262) credential-source wiring in main: PAT vs
// GitHub App mode selection, App credential resolution, and the scope gate over a
// refreshable source. No real network — App minting is never triggered here.
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/credential"
)

func appKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}))
}

func TestResolveCredentialSource_PATMode(t *testing.T) {
	t.Setenv("CRED_PAT_ENV", "ghp_test_pat_value")
	cfg := &config.AdapterConfig{Git: config.GitConfig{Provider: "github", AuthEnv: "CRED_PAT_ENV"}}
	src, seed, err := resolveCredentialSource(cfg)
	if err != nil {
		t.Fatalf("PAT mode: %v", err)
	}
	if seed != "ghp_test_pat_value" {
		t.Fatalf("seed token mismatch: %q", seed)
	}
	tok, err := src.Token(context.Background())
	if err != nil || tok != "ghp_test_pat_value" {
		t.Fatalf("static source token = %q err=%v", tok, err)
	}
}

func TestResolveCredentialSource_PATMissingEnv(t *testing.T) {
	cfg := &config.AdapterConfig{Git: config.GitConfig{Provider: "github", AuthEnv: "CRED_PAT_UNSET_XYZ"}}
	if _, _, err := resolveCredentialSource(cfg); err == nil {
		t.Fatal("expected error when PAT env var is unset")
	}
}

func TestResolveCredentialSource_AppMode(t *testing.T) {
	t.Setenv("CRED_APP_ID", "12345")
	t.Setenv("CRED_INSTALL_ID", "67890")
	t.Setenv("CRED_APP_KEY", appKeyPEM(t))
	cfg := &config.AdapterConfig{Git: config.GitConfig{Provider: "github", App: &config.GitHubAppConfig{
		AppIDEnv: "CRED_APP_ID", InstallationIDEnv: "CRED_INSTALL_ID", PrivateKeyEnv: "CRED_APP_KEY",
	}}}
	src, seed, err := resolveCredentialSource(cfg)
	if err != nil {
		t.Fatalf("App mode: %v", err)
	}
	if seed != "" {
		t.Fatalf("App-mode seed token should be empty, got %q", seed)
	}
	if _, ok := src.(*credential.AppSource); !ok {
		t.Fatalf("expected *credential.AppSource, got %T", src)
	}
}

func TestResolveCredentialSource_AppMode_BadKey(t *testing.T) {
	t.Setenv("CRED_APP_ID", "1")
	t.Setenv("CRED_INSTALL_ID", "1")
	t.Setenv("CRED_APP_KEY", "not a pem key")
	cfg := &config.AdapterConfig{Git: config.GitConfig{Provider: "github", App: &config.GitHubAppConfig{
		AppIDEnv: "CRED_APP_ID", InstallationIDEnv: "CRED_INSTALL_ID", PrivateKeyEnv: "CRED_APP_KEY",
	}}}
	if _, _, err := resolveCredentialSource(cfg); err == nil {
		t.Fatal("expected error for an invalid App private key")
	}
}

func TestResolveCredentialSource_AppMode_MissingEnv(t *testing.T) {
	cfg := &config.AdapterConfig{Git: config.GitConfig{Provider: "github", App: &config.GitHubAppConfig{
		AppIDEnv: "CRED_APP_ID_UNSET", InstallationIDEnv: "CRED_INSTALL_ID_UNSET", PrivateKeyEnv: "CRED_APP_KEY_UNSET",
	}}}
	if _, _, err := resolveCredentialSource(cfg); err == nil {
		t.Fatal("expected error when App env vars are unset")
	}
}

func TestRunScopeGate_StaticSourcePasses(t *testing.T) {
	// newScopeProbe defaults to a passing fine-grained fake (see scope_main_test.go TestMain).
	if err := runScopeGate(context.Background(), credential.NewStaticSource("ghp_fine_grained")); err != nil {
		t.Fatalf("scope gate over a static source should pass: %v", err)
	}
}
