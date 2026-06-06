// SPDX-License-Identifier: Apache-2.0

package check_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax-ci/check"
)

func writeGoMod(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDeps_AllMatch(t *testing.T) {
	root := t.TempDir()
	mod := "module github.com/example/a\n\ngo 1.26.4\n\nrequire (\n\tgoogle.golang.org/grpc v1.81.1\n\tgopkg.in/yaml.v3 v3.0.1\n)\n"
	writeGoMod(t, filepath.Join(root, "a"), mod)
	writeGoMod(t, filepath.Join(root, "b"), mod)

	r, err := check.Deps(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Violations) != 0 {
		t.Errorf("expected no violations, got %d: %+v", len(r.Violations), r.Violations)
	}
	if len(r.GoModFiles) != 2 {
		t.Errorf("expected 2 go.mod files, got %d", len(r.GoModFiles))
	}
}

func TestDeps_GrpcMismatch(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, filepath.Join(root, "a"), "module github.com/example/a\n\ngo 1.26.4\n\nrequire google.golang.org/grpc v1.81.1\n")
	writeGoMod(t, filepath.Join(root, "b"), "module github.com/example/b\n\ngo 1.26.4\n\nrequire google.golang.org/grpc v1.80.0\n")

	r, err := check.Deps(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %+v", len(r.Violations), r.Violations)
	}
	if r.Violations[0].Module != "google.golang.org/grpc" {
		t.Errorf("expected grpc violation, got %s", r.Violations[0].Module)
	}
}

func TestDeps_GoDirectiveMismatch(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, filepath.Join(root, "a"), "module github.com/example/a\n\ngo 1.26.4\n")
	writeGoMod(t, filepath.Join(root, "b"), "module github.com/example/b\n\ngo 1.25.0\n")

	r, err := check.Deps(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, v := range r.Violations {
		if v.Module == "go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected go directive violation, none found")
	}
}

func TestDeps_AbsentModuleNotViolation(t *testing.T) {
	root := t.TempDir()
	// Only "a" has grpc; "b" doesn't — no violation (only one occurrence).
	writeGoMod(t, filepath.Join(root, "a"), "module github.com/example/a\n\ngo 1.26.4\n\nrequire google.golang.org/grpc v1.81.1\n")
	writeGoMod(t, filepath.Join(root, "b"), "module github.com/example/b\n\ngo 1.26.4\n")

	r, err := check.Deps(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range r.Violations {
		if v.Module == "google.golang.org/grpc" {
			t.Errorf("grpc should not be a violation when only one module declares it; got %+v", v)
		}
	}
}

func TestDeps_VendorSkipped(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, filepath.Join(root, "a"), "module github.com/example/a\n\ngo 1.26.4\n")
	writeGoMod(t, filepath.Join(root, "a", "vendor", "something"), "module vendor/something\n\ngo 1.20.0\n")

	r, err := check.Deps(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// vendor go.mod should be skipped — only "a/go.mod" found
	if len(r.GoModFiles) != 1 {
		t.Errorf("expected 1 go.mod file (vendor skipped), got %d: %v", len(r.GoModFiles), r.GoModFiles)
	}
	if len(r.Violations) != 0 {
		t.Errorf("expected no violations, got %d", len(r.Violations))
	}
}
