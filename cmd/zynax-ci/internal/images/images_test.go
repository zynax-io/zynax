// SPDX-License-Identifier: Apache-2.0

package images_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax-ci/internal/images"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func makeImagesYAML(entries ...string) string {
	body := "images:\n"
	for _, e := range entries {
		body += e
	}
	return body
}

const ciRunnerEntry = `
  - name: ci-runner
    ref: ghcr.io/example/ci-runner
    digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
    consumers:
      - workflow.yml
      - digest.txt
`

const goodWorkflow = `container:
  image: ghcr.io/example/ci-runner@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
`
const staleWorkflow = `container:
  image: ghcr.io/example/ci-runner@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
`
const goodDigestTxt = `sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
`
const staleDigestTxt = `sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
`

func TestLoad(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "images"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "images/images.yaml", makeImagesYAML(ciRunnerEntry))

	f, err := images.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(f.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(f.Images))
	}
	if f.Images[0].Name != "ci-runner" {
		t.Errorf("unexpected name: %s", f.Images[0].Name)
	}
}

func TestCheck_AllGood(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "images"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "images/images.yaml", makeImagesYAML(ciRunnerEntry))
	writeFile(t, root, "workflow.yml", goodWorkflow)
	writeFile(t, root, "digest.txt", goodDigestTxt)

	f, _ := images.Load(root)
	report, err := images.Check(f, root)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(report.Violations) != 0 {
		t.Errorf("expected no violations, got %d: %+v", len(report.Violations), report.Violations)
	}
}

func TestCheck_Violation(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "images"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "images/images.yaml", makeImagesYAML(ciRunnerEntry))
	writeFile(t, root, "workflow.yml", staleWorkflow)
	writeFile(t, root, "digest.txt", goodDigestTxt)
	writeFile(t, root, "digest.txt", goodDigestTxt)

	f, _ := images.Load(root)
	report, err := images.Check(f, root)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(report.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(report.Violations))
	}
	if report.Violations[0].File != "workflow.yml" {
		t.Errorf("unexpected file: %s", report.Violations[0].File)
	}
}

func TestSync_RefBased(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "images"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "images/images.yaml", makeImagesYAML(ciRunnerEntry))
	writeFile(t, root, "workflow.yml", staleWorkflow)
	writeFile(t, root, "digest.txt", goodDigestTxt)
	writeFile(t, root, "digest.txt", goodDigestTxt)

	f, _ := images.Load(root)
	results, err := images.Sync(f, root, false)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	var changed int
	for _, r := range results {
		if r.Changed {
			changed++
		}
	}
	if changed != 1 {
		t.Errorf("expected 1 file changed, got %d", changed)
	}
	data, _ := os.ReadFile(filepath.Join(root, "workflow.yml")) //nolint:gosec
	target := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if !strings.Contains(string(data), target) {
		t.Errorf("workflow.yml not updated: %s", data)
	}
}

func TestSync_FallbackRaw(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "images"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "images/images.yaml", makeImagesYAML(ciRunnerEntry))
	writeFile(t, root, "workflow.yml", goodWorkflow)
	writeFile(t, root, "digest.txt", staleDigestTxt)

	f, _ := images.Load(root)
	results, err := images.Sync(f, root, false)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	var changed int
	for _, r := range results {
		if r.Changed {
			changed++
		}
	}
	if changed != 1 {
		t.Errorf("expected 1 file changed, got %d", changed)
	}
	data, _ := os.ReadFile(filepath.Join(root, "digest.txt")) //nolint:gosec
	target := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if !strings.Contains(string(data), target) {
		t.Errorf("digest.txt not updated: %s", data)
	}
}

func TestSync_DryRun(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "images"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "images/images.yaml", makeImagesYAML(ciRunnerEntry))
	wfPath := writeFile(t, root, "workflow.yml", staleWorkflow)
	writeFile(t, root, "digest.txt", goodDigestTxt)

	f, _ := images.Load(root)
	results, err := images.Sync(f, root, true)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if !results[0].Changed {
		t.Error("expected Changed=true in dry-run mode")
	}
	data, _ := os.ReadFile(wfPath) //nolint:gosec
	if !strings.Contains(string(data), "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb") {
		t.Error("dry-run should not modify workflow.yml — original stale digest should remain")
	}
}

func TestSync_Idempotent(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "images"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "images/images.yaml", makeImagesYAML(ciRunnerEntry))
	writeFile(t, root, "workflow.yml", goodWorkflow)
	writeFile(t, root, "digest.txt", goodDigestTxt)

	f, _ := images.Load(root)
	results, err := images.Sync(f, root, false)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	for _, r := range results {
		if r.Changed {
			t.Errorf("idempotent run: %s was changed but should be no-op", r.File)
		}
	}
}

func TestPrintCheckReport_Ok(t *testing.T) {
	r := images.CheckReport{}
	ok := images.PrintCheckReport(os.Stdout, r)
	if !ok {
		t.Error("expected PrintCheckReport to return true for empty violations")
	}
}

func TestPrintCheckReport_Violations(t *testing.T) {
	r := images.CheckReport{
		Violations: []images.CheckViolation{
			{File: "workflow.yml", Image: "ci-runner", ExpectedDigest: "sha256:aaa..."},
		},
	}
	ok := images.PrintCheckReport(os.Stdout, r)
	if ok {
		t.Error("expected PrintCheckReport to return false when violations exist")
	}
}
