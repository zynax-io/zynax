// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax/validate"
)

// fullCanvas is a minimal but structurally complete REASONS Canvas.
const fullCanvas = `# REASONS Canvas — Test

**Status:** Aligned

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
content
`

func TestCanvas_ValidCanvas(t *testing.T) {
	f := writeTempMD(t, fullCanvas)
	errs, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("unexpected error: %s", e)
		}
	}
}

func TestCanvas_RealCanvas(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	canvas := filepath.Join(filepath.Dir(file), "../../../docs/spdd/314-yaml-system-cli/canvas.md")
	errs, err := validate.Canvas(canvas)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("real canvas failed: %s", e)
		}
	}
}

func TestCanvas_MissingStatus(t *testing.T) {
	content := strings.ReplaceAll(fullCanvas, "**Status:** Aligned\n", "")
	f := writeTempMD(t, content)
	errs, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainsMessage(t, errs, "**Status:**")
}

func TestCanvas_MissingRSection(t *testing.T) {
	content := strings.ReplaceAll(fullCanvas, "## R — Requirements\n", "")
	f := writeTempMD(t, content)
	errs, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainsMessage(t, errs, "R — Requirements")
}

func TestCanvas_MissingSecondS(t *testing.T) {
	// Remove the Safeguards S section (second ## S —).
	// The first ## S — (Structure) remains, so count == 1.
	content := strings.ReplaceAll(fullCanvas, "## S — Safeguards\n", "")
	f := writeTempMD(t, content)
	errs, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainsMessage(t, errs, "Safeguards")
}

func TestCanvas_BothSMissing(t *testing.T) {
	content := strings.ReplaceAll(fullCanvas, "## S — Structure\n", "")
	content = strings.ReplaceAll(content, "## S — Safeguards\n", "")
	f := writeTempMD(t, content)
	errs, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Both Structure and Safeguards should be reported.
	assertContainsMessage(t, errs, "S — Structure")
	assertContainsMessage(t, errs, "Safeguards")
}

func TestCanvas_MultipleMissing(t *testing.T) {
	content := strings.ReplaceAll(fullCanvas, "## N — Norms\n", "")
	content = strings.ReplaceAll(content, "## O — Operations\n", "")
	f := writeTempMD(t, content)
	errs, err := validate.Canvas(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContainsMessage(t, errs, "N — Norms")
	assertContainsMessage(t, errs, "O — Operations")
}

func TestCanvas_FileNotFound(t *testing.T) {
	_, err := validate.Canvas("/nonexistent/canvas.md")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func writeTempMD(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "canvas.md")
	if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
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
