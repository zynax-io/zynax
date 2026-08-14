// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resetRenderOpts restores the package-level flag state between cases.
func resetRenderOpts(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		renderOpts.matrix, renderOpts.out, renderOpts.headline = "", "-", false
		matrixOpts.root, matrixOpts.outcomes, matrixOpts.out = ".", "", "-"
	})
}

// emitMatrix runs the real emitter over the live repo with an EMPTY outcomes
// directory — the honest all-NOT_RUN document, which is exactly what a release
// publishes when no conformance run is available for the released revision.
func emitMatrix(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	outcomes := filepath.Join(dir, "outcomes")
	if err := os.Mkdir(outcomes, 0o750); err != nil {
		t.Fatal(err)
	}
	matrixOpts.root, matrixOpts.outcomes = repoRoot(t), outcomes
	matrixOpts.out = filepath.Join(dir, "zecs-matrix.json")
	if err := runConformanceMatrix(fakeCmd(t), nil); err != nil {
		t.Fatalf("emit the matrix: %v", err)
	}
	return matrixOpts.out
}

// The publication chain, end to end on the live suite definition: emit the
// document, then render THAT document. The rendering must be a view of the file,
// never a recomputation.
func TestRunConformanceRender_RendersAnEmittedDocument(t *testing.T) {
	resetRenderOpts(t)
	matrix := emitMatrix(t)

	renderOpts.matrix, renderOpts.out, renderOpts.headline = matrix, "-", false
	var out bytes.Buffer
	cmd := fakeCmd(t)
	cmd.SetOut(&out)
	if err := runConformanceRender(cmd, nil); err != nil {
		t.Fatalf("render: %v", err)
	}
	rendered := out.String()
	for _, want := range []string{
		"### Engine conformance — ZECS",
		"| Scenario |",
		"NOT_RUN",
		"Incomplete run — not a conformance result",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendering does not contain %q\n---\n%s", want, rendered)
		}
	}
	// No cell and no claim may read PASS: the only "PASS" the document carries is
	// the manifest's own caveat about reading PASS on two legs.
	if strings.Contains(rendered, "| PASS") || strings.Contains(rendered, "legs: PASS") {
		t.Errorf("a run with no observations must not render a PASS result:\n%s", rendered)
	}
}

// --headline is what the release-notes line is built from: exactly one line.
func TestRunConformanceRender_HeadlineIsOneLine(t *testing.T) {
	resetRenderOpts(t)
	renderOpts.matrix, renderOpts.out, renderOpts.headline = emitMatrix(t), "-", true
	var out bytes.Buffer
	cmd := fakeCmd(t)
	cmd.SetOut(&out)
	if err := runConformanceRender(cmd, nil); err != nil {
		t.Fatalf("render: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one line, got %d: %q", len(lines), out.String())
	}
	for _, want := range []string{"ZECS", "required legs:", "all legs:"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("headline %q does not carry %q", lines[0], want)
		}
	}
}

// --out writes the file a release attaches next to zecs-matrix.json.
func TestRunConformanceRender_WritesToFile(t *testing.T) {
	resetRenderOpts(t)
	dest := filepath.Join(t.TempDir(), "zecs-conformance.md")
	renderOpts.matrix, renderOpts.out, renderOpts.headline = emitMatrix(t), dest, false
	if err := runConformanceRender(fakeCmd(t), nil); err != nil {
		t.Fatalf("render: %v", err)
	}
	body, err := os.ReadFile(dest) //nolint:gosec // dest is t.TempDir()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "### Engine conformance") {
		t.Errorf("unexpected file contents:\n%s", body)
	}
}

// Reading stdin, and refusing to publish what this build cannot render.
func TestRunConformanceRender_StdinAndUnrenderableInput(t *testing.T) {
	resetRenderOpts(t)
	renderOpts.matrix, renderOpts.out, renderOpts.headline = "-", "-", true

	cmd := fakeCmd(t)
	cmd.SetIn(strings.NewReader(`{"schema_version":"1","short_name":"ZECS","version":"v0.8.0",` +
		`"result":"PASS","enforced_result":"PASS","complete":true,"legs":[]}`))
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runConformanceRender(cmd, nil); err != nil {
		t.Fatalf("render from stdin: %v", err)
	}
	if !strings.Contains(out.String(), "ZECS v0.8.0") {
		t.Errorf("unexpected headline: %q", out.String())
	}

	future := fakeCmd(t)
	future.SetIn(strings.NewReader(`{"schema_version":"99"}`))
	if err := runConformanceRender(future, nil); err == nil {
		t.Error("expected a document this build cannot render to be rejected")
	}
}

func TestRunConformanceRender_MissingInputAndUnwritableOutput(t *testing.T) {
	resetRenderOpts(t)
	renderOpts.matrix, renderOpts.out = filepath.Join(t.TempDir(), "absent.json"), "-"
	if err := runConformanceRender(fakeCmd(t), nil); err == nil {
		t.Error("expected an error for a matrix file that does not exist")
	}

	renderOpts.matrix = emitMatrix(t)
	renderOpts.out = filepath.Join(t.TempDir(), "no-such-dir", "out.md")
	if err := runConformanceRender(fakeCmd(t), nil); err == nil {
		t.Error("expected an error for an unwritable output path")
	}
}
