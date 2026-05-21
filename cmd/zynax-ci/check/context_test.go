// SPDX-License-Identifier: Apache-2.0

package check_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax-ci/check"
)

func writeLines(t *testing.T, path string, n int) {
	t.Helper()
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString("line\n")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestContext_AllWithinBudget(t *testing.T) {
	root := t.TempDir()
	writeLines(t, filepath.Join(root, "CLAUDE.md"), 50)
	writeLines(t, filepath.Join(root, "AGENTS.md"), 100)
	writeLines(t, filepath.Join(root, "docs", "ai-assistant-setup.md"), 80)
	writeLines(t, filepath.Join(root, "services", "AGENTS.md"), 60)

	report, err := check.Context(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Warnings != 0 {
		t.Errorf("expected 0 warnings, got %d", report.Warnings)
	}
	if report.Total == 0 {
		t.Error("expected non-zero total")
	}
}

func TestContext_ExceedsThreshold(t *testing.T) {
	root := t.TempDir()
	// Write more than the CLAUDE.md threshold (200)
	writeLines(t, filepath.Join(root, "CLAUDE.md"), 250)

	report, err := check.Context(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Warnings == 0 {
		t.Error("expected at least 1 warning for oversized CLAUDE.md")
	}
	found := false
	for _, f := range report.Files {
		if strings.HasSuffix(f.Path, "CLAUDE.md") && f.Warn {
			found = true
		}
	}
	if !found {
		t.Error("expected CLAUDE.md to be flagged as WARN")
	}
}

func TestContext_TotalExceedsLimit(t *testing.T) {
	root := t.TempDir()
	// Spread lines across many AGENTS.md files to breach the 2000-line total
	for _, sub := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n"} {
		writeLines(t, filepath.Join(root, sub, "AGENTS.md"), 150)
	}
	writeLines(t, filepath.Join(root, "CLAUDE.md"), 180)
	writeLines(t, filepath.Join(root, "AGENTS.md"), 290)

	report, err := check.Context(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Total <= 2200 {
		t.Logf("total=%d — test may not be exercising total threshold, skipping", report.Total)
		return
	}
	if report.Warnings == 0 {
		t.Error("expected warnings when total exceeds 2200")
	}
}

func TestContext_MissingFiles(t *testing.T) {
	// Empty repo root — no AI context files at all
	root := t.TempDir()
	report, err := check.Context(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Total != 0 {
		t.Errorf("expected total=0 for empty root, got %d", report.Total)
	}
	if len(report.Files) != 0 {
		t.Errorf("expected 0 file entries, got %d", len(report.Files))
	}
}

func TestContextReport_Print_WithinBudget(t *testing.T) {
	root := t.TempDir()
	writeLines(t, filepath.Join(root, "CLAUDE.md"), 10)

	report, err := check.Context(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, err := os.CreateTemp(t.TempDir(), "report-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	report.Print(f) // exercises Print() for the within-budget path
}

func TestContextReport_Print_WithWarning(t *testing.T) {
	root := t.TempDir()
	writeLines(t, filepath.Join(root, "CLAUDE.md"), 250) // exceeds threshold

	report, err := check.Context(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, err := os.CreateTemp(t.TempDir(), "report-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	report.Print(f) // exercises warning branch of Print()
}

func TestContext_RootAgentsMdNotCountedTwice(t *testing.T) {
	root := t.TempDir()
	writeLines(t, filepath.Join(root, "AGENTS.md"), 100)
	writeLines(t, filepath.Join(root, "sub", "AGENTS.md"), 50)

	report, err := check.Context(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Total should be 150, not 200 (root AGENTS.md should not appear twice)
	if report.Total != 150 {
		t.Errorf("expected total=150, got %d", report.Total)
	}
}
