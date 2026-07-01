// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// recordingRunner is a clusterRunner that records the script path and env it was
// asked to run and returns a canned error — it never execs anything. Shared by
// the down and up command tests (same package).
type recordingRunner struct {
	called bool
	script string
	env    []string
	err    error
}

func (r *recordingRunner) run(_ context.Context, _, _ io.Writer, script string, env []string) error {
	r.called = true
	r.script = script
	r.env = env
	return r.err
}

// envHas reports whether env contains the exact "KEY=VALUE" entry.
func envHas(env []string, kv string) bool {
	for _, e := range env {
		if e == kv {
			return true
		}
	}
	return false
}

// fixedRoot returns a findRoot that always resolves to root (no filesystem walk).
func fixedRoot(root string) func(string) (string, error) {
	return func(string) (string, error) { return root, nil }
}

// ── reporoot.go ───────────────────────────────────────────────────────────────

func TestFindRepoRoot_FlagOverride(t *testing.T) {
	root := repoRoot(t) // the real checkout — contains scripts/e2e/cluster-up.sh
	got, err := findRepoRoot(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := filepath.Abs(root)
	if got != want {
		t.Errorf("findRepoRoot(%q) = %q, want %q", root, got, want)
	}
}

func TestFindRepoRoot_FlagWithoutSentinel(t *testing.T) {
	if _, err := findRepoRoot(t.TempDir()); err == nil {
		t.Error("expected error for a --repo-root dir missing the sentinel script")
	}
}

func TestFindRepoRoot_EnvVar(t *testing.T) {
	root := repoRoot(t)
	t.Setenv(repoRootEnv, root)
	got, err := findRepoRoot("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := filepath.Abs(root)
	if got != want {
		t.Errorf("findRepoRoot via %s = %q, want %q", repoRootEnv, got, want)
	}
}

func TestFindRepoRoot_EnvVarWithoutSentinel(t *testing.T) {
	t.Setenv(repoRootEnv, t.TempDir())
	if _, err := findRepoRoot(""); err == nil {
		t.Errorf("expected error when %s points at a tree without the sentinel", repoRootEnv)
	}
}

func TestFindRepoRoot_WalkUpFromSubdir(t *testing.T) {
	// Build an isolated fake checkout: <tmp>/scripts/e2e/cluster-up.sh + a nested subdir.
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "scripts", "e2e"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, clusterUpRel), []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(tmp, "a", "b")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv(repoRootEnv, "") // ensure the env branch does not short-circuit the walk
	t.Chdir(sub)

	got, err := findRepoRoot("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := filepath.Abs(tmp)
	gotAbs, _ := filepath.Abs(got)
	if gotAbs != want {
		t.Errorf("walk-up findRepoRoot = %q, want %q", gotAbs, want)
	}
}

func TestFindRepoRoot_OutsideCheckout(t *testing.T) {
	tmp := t.TempDir() // no sentinel anywhere above it
	t.Setenv(repoRootEnv, "")
	t.Chdir(tmp)
	_, err := findRepoRoot("")
	if !errors.Is(err, errNoRepoRoot) {
		t.Errorf("outside a checkout want errNoRepoRoot, got %v", err)
	}
}

// ── down.go ───────────────────────────────────────────────────────────────────

func TestRunDown_ForwardsClusterNameAndScript(t *testing.T) {
	downClusterName = "my-cluster"
	downRepoRoot = ""
	defer func() { downClusterName = "" }()

	rr := &recordingRunner{}
	deps := clusterDeps{run: rr.run, findRoot: fixedRoot("/repo")}
	if err := runDown(context.Background(), io.Discard, io.Discard, deps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rr.called {
		t.Fatal("runner was not invoked")
	}
	if rr.script != filepath.Join("/repo", clusterDownRel) {
		t.Errorf("script = %q, want %q", rr.script, filepath.Join("/repo", clusterDownRel))
	}
	if !envHas(rr.env, "CLUSTER_NAME=my-cluster") {
		t.Errorf("CLUSTER_NAME not forwarded; env tail=%v", rr.env[len(rr.env)-1:])
	}
}

func TestRunDown_NoClusterName_AppendsNothing(t *testing.T) {
	downClusterName = ""
	downRepoRoot = ""

	rr := &recordingRunner{}
	deps := clusterDeps{run: rr.run, findRoot: fixedRoot("/repo")}
	if err := runDown(context.Background(), io.Discard, io.Discard, deps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Robust against an ambient CLUSTER_NAME in the test env: assert runDown
	// appended zero entries rather than checking for absence of the key.
	if len(rr.env) != len(os.Environ()) {
		t.Errorf("runDown appended %d env entries with an empty --cluster-name, want 0",
			len(rr.env)-len(os.Environ()))
	}
}

func TestRunDown_RepoRootError_RunnerNotCalled(t *testing.T) {
	downRepoRoot = ""
	rr := &recordingRunner{}
	deps := clusterDeps{
		run:      rr.run,
		findRoot: func(string) (string, error) { return "", errNoRepoRoot },
	}
	err := runDown(context.Background(), io.Discard, io.Discard, deps)
	if !errors.Is(err, errNoRepoRoot) {
		t.Errorf("want errNoRepoRoot, got %v", err)
	}
	if rr.called {
		t.Error("runner must not run when repo-root resolution fails")
	}
}

func TestRunDown_RunnerErrorPropagates(t *testing.T) {
	downClusterName = ""
	downRepoRoot = ""
	rr := &recordingRunner{err: errors.New("boom")}
	deps := clusterDeps{run: rr.run, findRoot: fixedRoot("/repo")}
	if err := runDown(context.Background(), io.Discard, io.Discard, deps); err == nil {
		t.Fatal("expected the runner error to propagate")
	}
}

// ── streamRunner (production exec seam) ───────────────────────────────────────

func TestStreamRunner_StreamsStdoutAndPassesEnv(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "echo.sh")
	// Echoes a marker plus the value of an env var supplied via the overlay.
	body := "#!/bin/sh\necho \"marker:$ZYNAX_TEST_VAR\"\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil { //nolint:gosec // G306: test-only executable script in a temp dir
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	env := append(os.Environ(), "ZYNAX_TEST_VAR=hello")
	if err := streamRunner(context.Background(), &out, &errOut, script, env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "marker:hello") {
		t.Errorf("streamRunner did not stream stdout / pass env overlay; stdout=%q", out.String())
	}
}

func TestStreamRunner_NonZeroExitErrorsNamesScript(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fail.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 3\n"), 0o700); err != nil { //nolint:gosec // G306: test-only executable script in a temp dir
		t.Fatal(err)
	}
	err := streamRunner(context.Background(), io.Discard, io.Discard, script, os.Environ())
	if err == nil {
		t.Fatal("expected an error on non-zero script exit")
	}
	if !strings.Contains(err.Error(), "fail.sh") {
		t.Errorf("error should name the script (base), got %v", err)
	}
}

func TestDownCmd_Registered(t *testing.T) {
	c, _, err := rootCmd.Find([]string{"down"})
	if err != nil || c.Name() != "down" {
		t.Fatalf("down command not registered: cmd=%v err=%v", c, err)
	}
	if c.GroupID != beginnerGroupID {
		t.Errorf("down GroupID = %q, want %q", c.GroupID, beginnerGroupID)
	}
}
