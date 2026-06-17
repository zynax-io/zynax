// SPDX-License-Identifier: Apache-2.0

package bumprunner_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax-ci/bumprunner"
)

const (
	oldDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	newDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

// imagesYAML mirrors the real file's comment-preceded, multi-entry shape so the
// scoped digest replacement is exercised against realistic content.
const imagesYAML = `# SPDX-License-Identifier: Apache-2.0
# Single source of truth.

images:

  - name: ci-runner
    ref: ghcr.io/zynax-io/zynax/ci-runner
    digest: ` + oldDigest + `
    consumers:
      - ci.yml
      - digest.txt

  - name: golang-alpine
    ref: golang
    tag: "1.26.4-alpine"
    digest: ` + golangDigest + `
    consumers:
      - other.yml
`

const golangDigest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
const ciWorkflow = "container:\n  image: ghcr.io/zynax-io/zynax/ci-runner@" + oldDigest + "\n"
const digestTxt = oldDigest + "\n"
const otherWorkflow = "FROM golang:1.26.4-alpine@" + golangDigest + "\n"

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func setupRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "images/images.yaml", imagesYAML)
	writeFile(t, root, "ci.yml", ciWorkflow)
	writeFile(t, root, "digest.txt", digestTxt)
	writeFile(t, root, "other.yml", otherWorkflow)
	return root
}

func read(t *testing.T, root, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, name)) //nolint:gosec
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestValidateDigest(t *testing.T) {
	cases := []struct {
		name   string
		digest string
		ok     bool
	}{
		{"valid", newDigest, true},
		{"missing prefix", strings.TrimPrefix(newDigest, "sha256:"), false},
		{"too short", "sha256:abc", false},
		{"uppercase hex", "sha256:" + strings.Repeat("A", 64), false},
		{"empty", "", false},
		{"trailing space", newDigest + " ", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := bumprunner.ValidateDigest(tc.digest)
			if tc.ok && err != nil {
				t.Fatalf("want valid, got error: %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatalf("want error for %q", tc.digest)
			}
		})
	}
}

func TestBump_UpdatesYAMLAndConsumers(t *testing.T) {
	root := setupRepo(t)
	res, err := bumprunner.Bump(root, newDigest, false)
	if err != nil {
		t.Fatalf("Bump: %v", err)
	}
	if !res.YAMLChanged {
		t.Error("expected YAMLChanged=true")
	}
	if res.Before != oldDigest || res.After != newDigest {
		t.Errorf("before/after mismatch: %s -> %s", res.Before, res.After)
	}
	if len(res.ChangedFiles) != 2 {
		t.Fatalf("expected 2 changed consumers, got %d: %v", len(res.ChangedFiles), res.ChangedFiles)
	}
	// SoT updated, scoped to ci-runner only.
	yaml := read(t, root, "images/images.yaml")
	if !strings.Contains(yaml, "digest: "+newDigest) {
		t.Errorf("images.yaml ci-runner digest not bumped:\n%s", yaml)
	}
	// golang-alpine digest must be untouched by the scoped replacement.
	if !strings.Contains(yaml, golangDigest) {
		t.Errorf("golang-alpine digest was corrupted:\n%s", yaml)
	}
	if !strings.Contains(read(t, root, "ci.yml"), newDigest) {
		t.Error("ci.yml not re-stamped")
	}
	if !strings.Contains(read(t, root, "digest.txt"), newDigest) {
		t.Error("digest.txt not re-stamped")
	}
	// Unrelated image's consumer must stay put.
	if strings.Contains(read(t, root, "other.yml"), newDigest) {
		t.Error("other.yml (golang-alpine consumer) was wrongly re-stamped")
	}
}

func TestBump_Idempotent(t *testing.T) {
	root := setupRepo(t)
	res, err := bumprunner.Bump(root, oldDigest, false)
	if err != nil {
		t.Fatalf("Bump: %v", err)
	}
	if res.YAMLChanged {
		t.Error("expected YAMLChanged=false for a no-op bump")
	}
	if len(res.ChangedFiles) != 0 {
		t.Errorf("expected 0 changed files, got %v", res.ChangedFiles)
	}
}

func TestBump_DryRun_NoWrites(t *testing.T) {
	root := setupRepo(t)
	res, err := bumprunner.Bump(root, newDigest, true)
	if err != nil {
		t.Fatalf("Bump: %v", err)
	}
	if !res.YAMLChanged || len(res.ChangedFiles) != 2 {
		t.Fatalf("dry-run should report the would-be changes, got changed=%v files=%v", res.YAMLChanged, res.ChangedFiles)
	}
	// Nothing on disk may change in dry-run.
	if strings.Contains(read(t, root, "images/images.yaml"), newDigest) {
		t.Error("dry-run modified images.yaml")
	}
	if strings.Contains(read(t, root, "ci.yml"), newDigest) {
		t.Error("dry-run modified ci.yml")
	}
}

func TestBump_InvalidDigest(t *testing.T) {
	root := setupRepo(t)
	if _, err := bumprunner.Bump(root, "sha256:nothex", false); err == nil {
		t.Fatal("want error for invalid digest")
	}
}

func TestBump_MissingImagesYAML(t *testing.T) {
	root := t.TempDir()
	if _, err := bumprunner.Bump(root, newDigest, false); err == nil {
		t.Fatal("want error when images.yaml is absent")
	}
}

func TestBump_RunnerEntryAbsent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "images/images.yaml", "images:\n  - name: other\n    ref: r\n    digest: "+oldDigest+"\n")
	if _, err := bumprunner.Bump(root, newDigest, false); err == nil {
		t.Fatal("want error when ci-runner entry is missing")
	}
}
