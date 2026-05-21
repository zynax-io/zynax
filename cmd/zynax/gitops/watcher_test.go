// SPDX-License-Identifier: Apache-2.0

package gitops_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zynax-io/zynax/cmd/zynax/gitops"
)

const testRunID = "wf-test"

// ── hashContent (via state round-trip) ────────────────────────────────────

func TestWatcher_SkipsUnchangedFiles(t *testing.T) {
	dir := t.TempDir()
	content := []byte("kind: Workflow\n")
	yamlPath := filepath.Join(dir, "wf.yaml")
	if err := os.WriteFile(yamlPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	calls := 0
	applyFn := func(_ context.Context, _ string, _ []byte) (string, error) {
		calls++
		return testRunID, nil
	}

	w := gitops.New(dir, applyFn)

	// First sync — file is new → should apply.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := runSync(ctx, w); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 apply call on first sync, got %d", calls)
	}

	// Second sync — file unchanged → should skip.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	w2 := gitops.New(dir, applyFn) // new watcher reads persisted state
	if err := runSync(ctx2, w2); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 total apply calls after second sync (no change), got %d", calls)
	}
}

func TestWatcher_ReappliesChangedFiles(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "wf.yaml")
	if err := os.WriteFile(yamlPath, []byte("kind: Workflow\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	calls := 0
	applyFn := func(_ context.Context, _ string, _ []byte) (string, error) {
		calls++
		return testRunID, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	w := gitops.New(dir, applyFn)
	if err := runSync(ctx, w); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 apply on first sync, got %d", calls)
	}

	// Modify the file.
	if err := os.WriteFile(yamlPath, []byte("kind: Workflow\nmetadata:\n  name: updated\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	w2 := gitops.New(dir, applyFn)
	if err := runSync(ctx2, w2); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 total apply calls after file change, got %d", calls)
	}
}

func TestWatcher_IgnoresNonYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# docs\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	calls := 0
	applyFn := func(_ context.Context, _ string, _ []byte) (string, error) {
		calls++
		return testRunID, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	w := gitops.New(dir, applyFn)
	if err := runSync(ctx, w); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("expected 0 apply calls for non-YAML files, got %d", calls)
	}
}

func TestWatcher_StatePersisted(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "wf.yaml")
	if err := os.WriteFile(yamlPath, []byte("kind: Workflow\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	applyFn := func(_ context.Context, _ string, _ []byte) (string, error) {
		return "wf-state-test", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	gitops.New(dir, applyFn)
	w := gitops.New(dir, applyFn)
	if err := runSync(ctx, w); err != nil {
		t.Fatal(err)
	}

	// State file should exist and contain the YAML path.
	stateData, err := os.ReadFile(filepath.Join(dir, ".zynax-watch.state")) //nolint:gosec // fixed path under t.TempDir()
	if err != nil {
		t.Fatalf("state file not created: %v", err)
	}
	var state map[string]string
	if err := json.Unmarshal(stateData, &state); err != nil {
		t.Fatalf("state file not valid JSON: %v", err)
	}
	if _, ok := state[yamlPath]; !ok {
		t.Errorf("expected %q in state, got keys: %v", yamlPath, state)
	}
}

// runSync cancels ctx immediately after the initial sync completes to avoid
// blocking on the fsnotify loop in unit tests.
func runSync(ctx context.Context, w *gitops.Watcher) error {
	syncCtx, syncCancel := context.WithCancel(ctx)
	defer syncCancel()
	return w.RunSync(syncCtx)
}

func TestWatcher_Run_ImmediateCancelCoversAddDirs(t *testing.T) {
	dir := t.TempDir()
	w := gitops.New(dir, func(_ context.Context, _ string, _ []byte) (string, error) {
		return "wf", nil
	})
	// Pre-cancel the context so Run() exits immediately after addDirs+RunSync.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := w.Run(ctx); err != nil {
		t.Errorf("unexpected error from Run with cancelled ctx: %v", err)
	}
}

func TestWatcher_ApplyFile_ApplyFuncError(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "wf.yaml")
	if err := os.WriteFile(yamlPath, []byte("kind: Workflow\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	applyFn := func(_ context.Context, _ string, _ []byte) (string, error) {
		return "", fmt.Errorf("apply error")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	w := gitops.New(dir, applyFn)
	// RunSync calls applyFile which calls applyFn; applyFn returns error.
	// The watcher logs the error but RunSync still returns nil (applyFile is fire-and-forget).
	if err := w.RunSync(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWatcher_SaveState_ReadonlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses file-permission checks; read-only dir test not meaningful")
	}
	dir := t.TempDir()
	// Make the dir read-only so CreateTemp inside saveState fails.
	if err := os.Chmod(dir, 0o555); err != nil { //nolint:gosec // 0o555 is intentional: read-only dir to test saveState failure
		t.Skip("cannot chmod dir (may be running as root)")
	}
	defer os.Chmod(dir, 0o755) //nolint:errcheck,gosec // restore test dir permissions on exit

	w := gitops.New(dir, func(_ context.Context, _ string, _ []byte) (string, error) {
		return "wf", nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// RunSync calls saveState; saveState fails on read-only dir.
	err := w.RunSync(ctx)
	if err == nil {
		t.Error("expected error from RunSync when state dir is read-only")
	}
}
