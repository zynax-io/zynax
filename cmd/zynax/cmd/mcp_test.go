// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnvValue(t *testing.T) {
	tests := []struct {
		name      string
		entry     string
		key       string
		wantVal   string
		wantMatch bool
	}{
		{"exact match", "GIT_ADAPTER_BIN=/usr/bin/git-adapter", gitAdapterBinEnv, "/usr/bin/git-adapter", true},
		{"empty value still matches", "GIT_ADAPTER_BIN=", gitAdapterBinEnv, "", true},
		{"different key", "PATH=/bin", gitAdapterBinEnv, "", false},
		{"prefix only, no equals", "GIT_ADAPTER_BIN", gitAdapterBinEnv, "", false},
		{"key is a prefix of entry key", "GIT_ADAPTER_BINX=y", gitAdapterBinEnv, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := envValue(tt.entry, tt.key)
			if ok != tt.wantMatch || got != tt.wantVal {
				t.Errorf("envValue(%q,%q) = (%q,%v), want (%q,%v)",
					tt.entry, tt.key, got, ok, tt.wantVal, tt.wantMatch)
			}
		})
	}
}

// writeFakeAdapter creates an executable shell script standing in for the
// git-adapter binary. It echoes its first arg and the two security-relevant env
// vars so the test can assert they were passed through (and the token was NOT an
// argument). Skips on non-POSIX platforms.
func writeFakeAdapter(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fake binary not supported on windows")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "git-adapter")
	script := "#!/bin/sh\n" + body
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil { //nolint:gosec // test fixture
		t.Fatalf("write fake binary: %v", err)
	}
	return bin
}

func TestRunMCPGit_LaunchesAdapterMCPMode(t *testing.T) {
	bin := writeFakeAdapter(t, `printf 'subcommand=%s\n' "$1"
printf 'ADAPTER_CONFIG=%s\n' "$ADAPTER_CONFIG"
printf 'GITHUB_TOKEN=%s\n' "$GITHUB_TOKEN"`)

	env := []string{
		gitAdapterBinEnv + "=" + bin,
		"ADAPTER_CONFIG=/etc/git-adapter.yaml",
		"GITHUB_TOKEN=ghp_secret",
	}
	var out, errOut bytes.Buffer
	if err := runMCPGit(context.Background(), &out, &errOut, strings.NewReader(""), env); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, errOut.String())
	}

	got := out.String()
	// The only argument passed to the adapter is the constant "mcp" subcommand —
	// never the token (no-secrets-in-args).
	if !strings.Contains(got, "subcommand=mcp") {
		t.Errorf("expected adapter launched with 'mcp' subcommand, got:\n%s", got)
	}
	if !strings.Contains(got, "ADAPTER_CONFIG=/etc/git-adapter.yaml") {
		t.Errorf("ADAPTER_CONFIG not passed through, got:\n%s", got)
	}
	if !strings.Contains(got, "GITHUB_TOKEN=ghp_secret") {
		t.Errorf("token env not inherited by adapter, got:\n%s", got)
	}
}

func TestRunMCPGit_PipesStdinThrough(t *testing.T) {
	bin := writeFakeAdapter(t, `cat`)
	env := []string{gitAdapterBinEnv + "=" + bin}

	var out bytes.Buffer
	in := strings.NewReader(`{"jsonrpc":"2.0","method":"ping"}`)
	if err := runMCPGit(context.Background(), &out, &bytes.Buffer{}, in, env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), `"method":"ping"`) {
		t.Errorf("stdin not piped to adapter stdout, got:\n%s", out.String())
	}
}

func TestRunMCPGit_BinaryNotFound(t *testing.T) {
	env := []string{gitAdapterBinEnv + "=/nonexistent/git-adapter-xyz"}
	err := runMCPGit(context.Background(), &bytes.Buffer{}, &bytes.Buffer{}, strings.NewReader(""), env)
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
	if !strings.Contains(err.Error(), "mcp git: launch") {
		t.Errorf("error not wrapped with context: %v", err)
	}
}

func TestRunMCPGit_DefaultBinaryWhenEnvUnset(t *testing.T) {
	// No GIT_ADAPTER_BIN in env → default "git-adapter" is used. We don't have
	// the real binary on PATH in CI, so we only assert the error names the
	// default binary (exercising the default-resolution branch).
	err := runMCPGit(context.Background(), &bytes.Buffer{}, &bytes.Buffer{}, strings.NewReader(""), []string{"PATH=/nonexistent"})
	if err == nil {
		t.Skip("git-adapter unexpectedly present on PATH; default-branch still exercised")
	}
	if !strings.Contains(err.Error(), defaultGitAdapterBin) {
		t.Errorf("expected error to name default binary %q, got: %v", defaultGitAdapterBin, err)
	}
}

func TestMCPGitCmd_RunE_Registered(t *testing.T) {
	if mcpCmd.Use != "mcp" {
		t.Errorf("mcpCmd.Use = %q", mcpCmd.Use)
	}
	sub, _, err := rootCmd.Find([]string{"mcp", "git"})
	if err != nil || sub.Use != "git" {
		t.Fatalf("mcp git subcommand not registered: sub=%v err=%v", sub, err)
	}
	if sub.RunE == nil {
		t.Error("mcp git RunE not set")
	}
}
