// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax-ci/validate"
)

// fullCanvas is a minimal but structurally complete REASONS Canvas.
const fullCanvas = `# REASONS Canvas — Test

**Issue:** #1
**Author:** Test Author
**Date:** 2026-01-01
**Status:** Aligned

---

## R — Requirements
content

## E — Entities
content

## A — Approach
content

## S — Structure
content

## O — Operations
content

## N — Norms
content

## S — Safeguards

### Context Security
- [x] No Tier 2 content
`

func TestCanvas_ValidAligned(t *testing.T) {
	f := writeTempCanvas(t, fullCanvas)
	errs, warns, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
	if len(warns) != 0 {
		t.Errorf("expected no warnings, got: %v", warns)
	}
}

func TestCanvas_DraftStatus_IsWarning(t *testing.T) {
	content := strings.ReplaceAll(fullCanvas, "**Status:** Aligned", "**Status:** Draft")
	f := writeTempCanvas(t, content)
	errs, warns, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors for Draft status, got: %v", errs)
	}
	if len(warns) == 0 {
		t.Error("expected a warning for Draft status, got none")
	}
}

func TestCanvas_TerminalStatuses_AreValid(t *testing.T) {
	// Superseded/Rejected are terminal states for an abandoned or replaced canvas
	// (e.g. its ADR was Rejected) and must validate without errors or warnings.
	for _, status := range []string{"Superseded", "Rejected"} {
		t.Run(status, func(t *testing.T) {
			content := strings.ReplaceAll(fullCanvas, "**Status:** Aligned", "**Status:** "+status)
			f := writeTempCanvas(t, content)
			errs, warns, err := validate.Canvas(f)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(errs) != 0 {
				t.Errorf("expected no errors for %s status, got: %v", status, errs)
			}
			if len(warns) != 0 {
				t.Errorf("expected no warnings for %s status, got: %v", status, warns)
			}
		})
	}
}

func TestCanvas_InvalidStatus(t *testing.T) {
	content := strings.ReplaceAll(fullCanvas, "**Status:** Aligned", "**Status:** InProgress")
	f := writeTempCanvas(t, content)
	errs, _, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainsMessage(t, errs, "invalid Status")
}

func TestCanvas_MissingIssueField(t *testing.T) {
	content := strings.ReplaceAll(fullCanvas, "**Issue:** #1\n", "")
	f := writeTempCanvas(t, content)
	errs, _, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainsMessage(t, errs, "**Issue:**")
}

func TestCanvas_MissingAuthorField(t *testing.T) {
	content := strings.ReplaceAll(fullCanvas, "**Author:** Test Author\n", "")
	f := writeTempCanvas(t, content)
	errs, _, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainsMessage(t, errs, "**Author:**")
}

func TestCanvas_MissingRSection(t *testing.T) {
	content := strings.ReplaceAll(fullCanvas, "## R — Requirements\n", "")
	f := writeTempCanvas(t, content)
	errs, _, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainsMessage(t, errs, "R — Requirements")
}

func TestCanvas_MissingSecondS(t *testing.T) {
	content := strings.ReplaceAll(fullCanvas, "## S — Safeguards\n", "")
	f := writeTempCanvas(t, content)
	errs, _, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainsMessage(t, errs, "Safeguards")
}

func TestCanvas_BothSMissing(t *testing.T) {
	content := strings.ReplaceAll(fullCanvas, "## S — Structure\n", "")
	content = strings.ReplaceAll(content, "## S — Safeguards\n", "")
	f := writeTempCanvas(t, content)
	errs, _, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainsMessage(t, errs, "S — Structure")
	assertContainsMessage(t, errs, "Safeguards")
}

func TestCanvas_MissingContextSecurity(t *testing.T) {
	content := strings.ReplaceAll(fullCanvas, "Context Security", "No-Security-Marker")
	f := writeTempCanvas(t, content)
	errs, _, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainsMessage(t, errs, "Context Security")
}

func TestCanvas_PrivateFilePresent(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "canvas.md")
	if err := os.WriteFile(f, []byte(fullCanvas), 0o600); err != nil {
		t.Fatal(err)
	}
	private := filepath.Join(dir, "canvas.private.md")
	if err := os.WriteFile(private, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	errs, _, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainsMessage(t, errs, "canvas.private.md")
}

func TestCanvas_MultipleMissing(t *testing.T) {
	content := strings.ReplaceAll(fullCanvas, "## N — Norms\n", "")
	content = strings.ReplaceAll(content, "## O — Operations\n", "")
	f := writeTempCanvas(t, content)
	errs, _, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainsMessage(t, errs, "N — Norms")
	assertContainsMessage(t, errs, "O — Operations")
}

func TestValidationError_Error(t *testing.T) {
	e := validate.ValidationError{File: "canvas.md", Message: "missing R section"}
	if e.Error() != "canvas.md: missing R section" {
		t.Errorf("Error() = %q, want %q", e.Error(), "canvas.md: missing R section")
	}
}

func TestCanvas_FileNotFound(t *testing.T) {
	_, _, err := validate.Canvas("/nonexistent/canvas.md")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestCanvas_RealCanvas(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	canvas := filepath.Join(filepath.Dir(file), "../../../docs/spdd/314-yaml-system-cli/canvas.md")
	errs, _, err := validate.Canvas(canvas)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("real canvas failed: %s", e)
		}
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func writeTempCanvas(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "canvas.md")
	if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return f
}

func assertContainsMessage(t *testing.T, errs []validate.ValidationError, substr string) {
	t.Helper()
	for _, e := range errs {
		if strings.Contains(e.Message, substr) {
			return
		}
	}
	t.Errorf("expected an error containing %q; got: %v", substr, errs)
}
