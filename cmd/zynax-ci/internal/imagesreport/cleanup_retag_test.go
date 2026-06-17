// SPDX-License-Identifier: Apache-2.0

package imagesreport

import (
	"strings"
	"testing"
)

const versionsJSON = `[
  {"id": 101, "updated_at": "2026-06-01T00:00:00Z", "metadata": {"container": {"tags": ["pr-abc123", "main-abc123"]}}},
  {"id": 102, "updated_at": "2026-06-02T00:00:00Z", "metadata": {"container": {"tags": ["main-def456"]}}},
  {"id": 103, "updated_at": "2026-06-03T00:00:00Z", "metadata": {"container": {"tags": ["main-aaa789"]}}},
  {"id": 104, "updated_at": "2026-06-04T00:00:00Z", "metadata": {"container": {"tags": ["latest", "main"]}}},
  {"id": 105, "updated_at": "2026-06-05T00:00:00Z", "metadata": {"container": {"tags": ["main-bbb012", "v1.2.3"]}}},
  {"id": 106, "updated_at": "2026-06-06T00:00:00Z", "metadata": {"container": {"tags": ["pr-only-tag"]}}}
]`

const testRef = "ghcr.io/z/tools"

func TestSelectByTag(t *testing.T) {
	ids, err := SelectByTag(strings.NewReader(versionsJSON), "pr-abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != 101 {
		t.Errorf("SelectByTag = %v, want [101]", ids)
	}
}

func TestSelectByTag_NoMatch(t *testing.T) {
	ids, err := SelectByTag(strings.NewReader(versionsJSON), "pr-nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("SelectByTag no-match = %v, want empty", ids)
	}
}

func TestSelectPrunable_KeepsNewest(t *testing.T) {
	// Prunable per-commit builds: 101,102,103 (105 has v-tag → protected;
	// 104 has latest/main → protected; 106 has no main-<sha> → not a build).
	// Newest-first: 103, 102, 101. keep=1 → prune 102, 101.
	ids, err := SelectPrunable(strings.NewReader(versionsJSON), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 102 || ids[1] != 101 {
		t.Errorf("SelectPrunable keep=1 = %v, want [102 101]", ids)
	}
}

func TestSelectPrunable_KeepAll(t *testing.T) {
	ids, err := SelectPrunable(strings.NewReader(versionsJSON), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("keep>=candidates should prune nothing, got %v", ids)
	}
}

func TestSelectPrunable_NegativeKeepAndProtected(t *testing.T) {
	// keep<0 is treated as 0 → all three prunable builds (101,102,103) selected;
	// 104 (latest/main), 105 (v-tag), 106 (no main-<sha>) are excluded.
	ids, err := SelectPrunable(strings.NewReader(versionsJSON), -5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("negative keep treated as 0, want 3 prunable, got %v", ids)
	}
	for _, id := range ids {
		if id == 104 || id == 105 || id == 106 {
			t.Errorf("id %d must not be prunable (protected/non-build)", id)
		}
	}
}

func TestFormatIDs(t *testing.T) {
	got := FormatIDs([]int64{101, 102})
	if got != "101\n102\n" {
		t.Errorf("FormatIDs = %q, want \"101\\n102\\n\"", got)
	}
	if FormatIDs(nil) != "" {
		t.Error("FormatIDs(nil) should be empty")
	}
}

func TestDecodeVersions_Invalid(t *testing.T) {
	if _, err := SelectByTag(strings.NewReader("{not json}"), "x"); err == nil {
		t.Error("expected parse error for invalid JSON")
	}
}

func TestFinalTags_MainBranch(t *testing.T) {
	tags, err := FinalTags(RetagInput{Ref: testRef, SHA: "abc123", GitRef: "refs/heads/main"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"ghcr.io/z/tools:main-abc123", "ghcr.io/z/tools:latest"}
	if len(tags) != 2 || tags[0] != want[0] || tags[1] != want[1] {
		t.Errorf("FinalTags main = %v, want %v", tags, want)
	}
	// A non-v refs/tags ref must not add a version tag either.
	nv, _ := FinalTags(RetagInput{Ref: testRef, SHA: "abc", GitRef: "refs/tags/nightly"})
	if len(nv) != 2 {
		t.Errorf("non-v tag must not add a version tag, got %v", nv)
	}
}

func TestFinalTags_VersionTag(t *testing.T) {
	tags, err := FinalTags(RetagInput{Ref: testRef, SHA: "abc123", GitRef: "refs/tags/v1.2.3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"ghcr.io/z/tools:main-abc123", "ghcr.io/z/tools:v1.2.3", "ghcr.io/z/tools:latest"}
	if len(tags) != 3 || tags[1] != want[1] {
		t.Errorf("FinalTags version = %v, want %v", tags, want)
	}
}

func TestFinalTags_Validation(t *testing.T) {
	if _, err := FinalTags(RetagInput{SHA: "abc"}); err == nil {
		t.Error("expected error for missing ref")
	}
	if _, err := FinalTags(RetagInput{Ref: testRef}); err == nil {
		t.Error("expected error for missing sha")
	}
}
