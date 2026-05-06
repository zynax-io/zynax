// SPDX-License-Identifier: Apache-2.0

package gitops_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zynax-io/zynax/cmd/zynax/gitops"
)

// ── hashContent (via state round-trip) ────────────────────────────────────

func TestWatcher_SkipsUnchangedFiles(t *testing.T) {
	dir := t.TempDir()
	content := []byte("kind: Workflow\n")
	yamlPath := filepath.Join(dir, "wf.yaml")
	if err := os.WriteFile(yamlPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	calls := 0
	applyFn := func(_ context.Context, _ string, _ []byte) (string, error) {
		calls++
		return "wf-test", nil
	}

	w := gitops.New(dir, applyFn)

	// First sync — file is new → should apply.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := runSync(w, ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 apply call on first sync, got %d", calls)
	}

	// Second sync — file unchanged → should skip.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	w2 := gitops.New(dir, applyFn) // new watcher reads persisted state
	if err := runSync(w2, ctx2); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 total apply calls after second sync (no change), got %d", calls)
	}
}

func TestWatcher_ReappliesChangedFiles(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "wf.yaml")
	if err := os.WriteFile(yamlPath, []byte("kind: Workflow\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	calls := 0
	applyFn := func(_ context.Context, _ string, _ []byte) (string, error) {
		calls++
		return "wf-test", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	w := gitops.New(dir, applyFn)
	if err := runSync(w, ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 apply on first sync, got %d", calls)
	}

	// Modify the file.
	if err := os.WriteFile(yamlPath, []byte("kind: Workflow\nmetadata:\n  name: updated\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	w2 := gitops.New(dir, applyFn)
	if err := runSync(w2, ctx2); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 total apply calls after file change, got %d", calls)
	}
}

func TestWatcher_IgnoresNonYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	calls := 0
	applyFn := func(_ context.Context, _ string, _ []byte) (string, error) {
		calls++
		return "wf-test", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	w := gitops.New(dir, applyFn)
	if err := runSync(w, ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("expected 0 apply calls for non-YAML files, got %d", calls)
	}
}

func TestWatcher_StatePersisted(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "wf.yaml")
	if err := os.WriteFile(yamlPath, []byte("kind: Workflow\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	applyFn := func(_ context.Context, _ string, _ []byte) (string, error) {
		return "wf-state-test", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	gitops.New(dir, applyFn)
	w := gitops.New(dir, applyFn)
	if err := runSync(w, ctx); err != nil {
		t.Fatal(err)
	}

	// State file should exist and contain the YAML path.
	stateData, err := os.ReadFile(filepath.Join(dir, ".zynax-watch.state"))
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
func runSync(w *gitops.Watcher, ctx context.Context) error {
	syncCtx, syncCancel := context.WithCancel(ctx)
	defer syncCancel()
	return w.RunSync(syncCtx)
}
