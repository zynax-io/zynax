// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const bumpOldDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const bumpNewDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

const bumpImagesYAML = `# SPDX-License-Identifier: Apache-2.0

images:

  - name: ci-runner
    ref: ghcr.io/zynax-io/zynax/ci-runner
    digest: ` + bumpOldDigest + `
    consumers:
      - ci.yml
`

func bumpRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "images", "images.yaml"), bumpImagesYAML)
	mustWrite(t, filepath.Join(root, "ci.yml"),
		"image: ghcr.io/zynax-io/zynax/ci-runner@"+bumpOldDigest+"\n")
	return root
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRunBumpRunner_Updates(t *testing.T) {
	root := bumpRepo(t)
	bumpRunnerRoot, bumpRunnerDryRun = root, false
	defer func() { bumpRunnerRoot, bumpRunnerDryRun = ".", false }()
	c, out := cmdWithIO(t, "")
	if err := runBumpRunner(c, []string{bumpNewDigest}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Updated ci-runner digest") {
		t.Fatalf("want update summary, got:\n%s", out.String())
	}
	data, _ := os.ReadFile(filepath.Join(root, "ci.yml")) //nolint:gosec
	if !strings.Contains(string(data), bumpNewDigest) {
		t.Errorf("ci.yml not re-stamped:\n%s", data)
	}
}

func TestRunBumpRunner_DryRun(t *testing.T) {
	root := bumpRepo(t)
	bumpRunnerRoot, bumpRunnerDryRun = root, true
	defer func() { bumpRunnerRoot, bumpRunnerDryRun = ".", false }()
	c, out := cmdWithIO(t, "")
	if err := runBumpRunner(c, []string{bumpNewDigest}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Would update") {
		t.Fatalf("want dry-run wording, got:\n%s", out.String())
	}
	data, _ := os.ReadFile(filepath.Join(root, "ci.yml")) //nolint:gosec
	if strings.Contains(string(data), bumpNewDigest) {
		t.Error("dry-run must not write ci.yml")
	}
}

func TestRunBumpRunner_NoOp(t *testing.T) {
	root := bumpRepo(t)
	bumpRunnerRoot, bumpRunnerDryRun = root, false
	defer func() { bumpRunnerRoot, bumpRunnerDryRun = ".", false }()
	c, out := cmdWithIO(t, "")
	if err := runBumpRunner(c, []string{bumpOldDigest}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "nothing to do") {
		t.Fatalf("want no-op summary, got:\n%s", out.String())
	}
}

func TestRunBumpRunner_InvalidDigest(t *testing.T) {
	root := bumpRepo(t)
	bumpRunnerRoot, bumpRunnerDryRun = root, false
	defer func() { bumpRunnerRoot, bumpRunnerDryRun = ".", false }()
	c, _ := cmdWithIO(t, "")
	if err := runBumpRunner(c, []string{"sha256:nothex"}); err == nil {
		t.Fatal("want error for invalid digest")
	}
}
