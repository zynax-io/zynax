// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const metaIndexJSON = `{"annotations":{"org.opencontainers.image.description":"d","org.opencontainers.image.title":"tools"},"manifests":[{"digest":"sha256:a","platform":{"os":"linux","architecture":"amd64"}}]}`

const (
	testDigest = "sha256:idx"
	testLabel  = "tools"
)

func resetMetaFlags() {
	metaLabel, metaDigest, metaExtra, metaIndex, metaSummary = "", "", "", "", ""
}

func TestRunImagesMeta_OK(t *testing.T) {
	resetMetaFlags()
	defer resetMetaFlags()
	metaLabel, metaDigest, metaExtra = testLabel, testDigest, "synced to images.yaml"
	c, out := cmdWithIO(t, metaIndexJSON)
	if err := runImagesMeta(c, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "## tools") {
		t.Errorf("want summary header, got:\n%s", out.String())
	}
}

func TestRunImagesMeta_MissingDescriptionFails(t *testing.T) {
	resetMetaFlags()
	defer resetMetaFlags()
	metaLabel, metaDigest = testLabel, testDigest
	c, _ := cmdWithIO(t, `{"annotations":{},"manifests":[]}`)
	if err := runImagesMeta(c, nil); err == nil {
		t.Fatal("want error for missing description annotation")
	}
}

func TestRunImagesMeta_WritesSummaryFile(t *testing.T) {
	resetMetaFlags()
	defer resetMetaFlags()
	dir := t.TempDir()
	summary := filepath.Join(dir, "summary.md")
	metaLabel, metaDigest, metaSummary = testLabel, testDigest, summary
	c, _ := cmdWithIO(t, metaIndexJSON)
	if err := runImagesMeta(c, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(summary) //nolint:gosec
	if !strings.Contains(string(data), "## tools") {
		t.Errorf("summary file not written:\n%s", data)
	}
}

func TestRunImagesMeta_BadFile(t *testing.T) {
	resetMetaFlags()
	defer resetMetaFlags()
	metaIndex = filepath.Join(t.TempDir(), "missing.json")
	c, _ := cmdWithIO(t, "")
	if err := runImagesMeta(c, nil); err == nil {
		t.Error("want error opening missing index file")
	}
}

const cleanupVersionsJSON = `[{"id":101,"updated_at":"2026-06-01T00:00:00Z","metadata":{"container":{"tags":["pr-abc","main-abc"]}}},{"id":102,"updated_at":"2026-06-02T00:00:00Z","metadata":{"container":{"tags":["main-def"]}}}]`

func resetCleanupFlags() {
	cleanupTag, cleanupVerJSON, cleanupKeep, cleanupPrune = "", "", 0, false
}

func TestRunImagesCleanup_ByTag(t *testing.T) {
	resetCleanupFlags()
	defer resetCleanupFlags()
	cleanupTag = "pr-abc"
	c, out := cmdWithIO(t, cleanupVersionsJSON)
	if err := runImagesCleanup(c, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out.String()) != "101" {
		t.Errorf("want id 101, got %q", out.String())
	}
}

func TestRunImagesCleanup_Prune(t *testing.T) {
	resetCleanupFlags()
	defer resetCleanupFlags()
	cleanupPrune, cleanupKeep = true, 1
	c, out := cmdWithIO(t, cleanupVersionsJSON)
	if err := runImagesCleanup(c, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Newest-first: 102, 101; keep=1 → prune 101.
	if strings.TrimSpace(out.String()) != "101" {
		t.Errorf("want pruned id 101, got %q", out.String())
	}
}

func TestRunImagesCleanup_TagRequired(t *testing.T) {
	resetCleanupFlags()
	defer resetCleanupFlags()
	c, _ := cmdWithIO(t, cleanupVersionsJSON)
	if err := runImagesCleanup(c, nil); err == nil {
		t.Error("want error when neither --tag nor --prune set")
	}
}

func TestRunImagesRetag(t *testing.T) {
	retagRef, retagSHA, retagGitRef = "ghcr.io/z/tools", "abc123", "refs/tags/v1.2.3"
	defer func() { retagRef, retagSHA, retagGitRef = "", "", "" }()
	c, out := cmdWithIO(t, "")
	if err := runImagesRetag(c, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	for _, want := range []string{"ghcr.io/z/tools:main-abc123", "ghcr.io/z/tools:v1.2.3", "ghcr.io/z/tools:latest"} {
		if !strings.Contains(got, want) {
			t.Errorf("want tag %q, got:\n%s", want, got)
		}
	}
	retagRef = ""
	if err := runImagesRetag(c, nil); err == nil {
		t.Error("want error for missing ref")
	}
}
