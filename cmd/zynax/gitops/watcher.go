// SPDX-License-Identifier: Apache-2.0

// Package gitops implements the zynax gitops watch sub-command.
// It watches a directory for YAML manifest changes and re-applies them
// to the api-gateway when their content changes.
package gitops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const stateFile = ".zynax-watch.state"

// ApplyFunc is called when a YAML file should be re-applied.
// It receives the file path and its content, and returns the run_id or agent_id.
type ApplyFunc func(ctx context.Context, path string, content []byte) (string, error)

// Watcher watches a directory tree for YAML changes and calls apply on each changed file.
type Watcher struct {
	dir   string
	apply ApplyFunc
	state map[string]string // path → sha256 hex
}

// New creates a Watcher for dir, using apply to submit changed manifests.
func New(dir string, apply ApplyFunc) *Watcher {
	return &Watcher{
		dir:   dir,
		apply: apply,
		state: make(map[string]string),
	}
}

// RunSync loads state and applies any changed YAML files once, then returns.
// It is used in unit tests and by Run before entering the fsnotify loop.
func (w *Watcher) RunSync(ctx context.Context) error {
	if err := w.loadState(); err != nil {
		return fmt.Errorf("gitops: load state: %w", err)
	}
	if err := w.syncAll(ctx); err != nil {
		return err
	}
	return w.saveState()
}

// Run starts the watch loop. It blocks until ctx is cancelled (Ctrl+C → exit 0).
func (w *Watcher) Run(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("gitops: create watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }()

	if err := w.addDirs(watcher); err != nil {
		return err
	}

	// Apply any changed files that accumulated since last run.
	if err := w.RunSync(ctx); err != nil {
		return err
	}

	slog.Info("gitops: watching", "dir", w.dir, "state_file", stateFile)

	for {
		select {
		case <-ctx.Done():
			_ = w.saveState()
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if !isYAML(event.Name) {
				continue
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
				w.applyFile(ctx, event.Name)
			}
			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				delete(w.state, event.Name)
				_ = w.saveState()
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			slog.Warn("gitops: watcher error", "err", err)
		}
	}
}

// applyFile hashes the file and calls apply only if the content changed.
func (w *Watcher) applyFile(ctx context.Context, path string) {
	content, err := os.ReadFile(path) //nolint:gosec // path is from fsnotify events on a trusted watch dir
	if err != nil {
		slog.Warn("gitops: read file", "path", path, "err", err)
		return
	}

	hash := hashContent(content)
	if w.state[path] == hash {
		slog.Debug("gitops: unchanged, skipping", "path", path)
		return
	}

	start := time.Now()
	id, err := w.apply(ctx, path, content)
	if err != nil {
		slog.Error("gitops: apply failed", "path", path, "err", err)
		return
	}

	w.state[path] = hash
	_ = w.saveState()
	slog.Info("gitops: applied", "path", path, "id", id, "elapsed_ms", time.Since(start).Milliseconds())
}

// syncAll applies all YAML files in the watched directory that have changed
// since the last run (compares against the persisted state file).
func (w *Watcher) syncAll(ctx context.Context) error {
	return filepath.WalkDir(w.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !isYAML(path) {
			return err
		}
		w.applyFile(ctx, path)
		return nil
	})
}

// addDirs registers the target directory and all subdirectories with the watcher.
func (w *Watcher) addDirs(fw *fsnotify.Watcher) error {
	return filepath.WalkDir(w.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if err := fw.Add(path); err != nil {
				return fmt.Errorf("gitops: watch dir %q: %w", path, err)
			}
		}
		return nil
	})
}

// ── State persistence ─────────────────────────────────────────────────────

func (w *Watcher) loadState() error {
	statePath := filepath.Join(w.dir, stateFile)
	f, err := os.Open(statePath) //nolint:gosec // statePath is always under the user-supplied watch dir
	if os.IsNotExist(err) {
		return nil // first run — no state yet
	}
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return json.NewDecoder(f).Decode(&w.state)
}

func (w *Watcher) saveState() error {
	statePath := filepath.Join(w.dir, stateFile)
	f, err := os.CreateTemp(filepath.Dir(statePath), ".zynax-watch-tmp-")
	if err != nil {
		return err
	}
	tmpName := f.Name()
	if err := json.NewEncoder(f).Encode(w.state); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpName)
		return err
	}
	_ = f.Close()
	return os.Rename(tmpName, statePath)
}

// ── Helpers ───────────────────────────────────────────────────────────────

func isYAML(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

// hashContent returns a stable hex-encoded SHA-256 of content.
func hashContent(content []byte) string {
	h := sha256.New()
	_, _ = h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}
