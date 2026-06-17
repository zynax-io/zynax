// SPDX-License-Identifier: Apache-2.0

package imagesreport

import (
	"strings"
	"testing"
)

const sampleIndex = `{
  "annotations": {
    "org.opencontainers.image.description": "Zynax tools image",
    "org.opencontainers.image.title": "tools"
  },
  "manifests": [
    {"platform": {"os": "linux"}},
    {"platform": {"os": "linux"}},
    {"platform": {"os": "unknown"}},
    {"platform": {"os": "unknown"}}
  ]
}`

func TestParseMeta_AnnotationsAndAttestations(t *testing.T) {
	m, err := ParseMeta(strings.NewReader(sampleIndex))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.IndexResolved || m.Description != "Zynax tools image" || m.Title != "tools" {
		t.Errorf("annotations not parsed: resolved=%v desc=%q title=%q", m.IndexResolved, m.Description, m.Title)
	}
	if m.Attestations != ExpectedAttestations {
		t.Errorf("attestations = %d, want %d", m.Attestations, ExpectedAttestations)
	}
}

func TestParseMeta_EmptyIndexNonFatal(t *testing.T) {
	m, err := ParseMeta(strings.NewReader("  \n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.IndexResolved || m.MissingDescription() {
		t.Error("empty index must be unresolved and not fail the description check")
	}
	if !strings.Contains(m.Summary("tools", "sha256:x", ""), "report skipped") {
		t.Error("unresolved index should render the skipped block")
	}
}

func TestParseMeta_Invalid(t *testing.T) {
	if _, err := ParseMeta(strings.NewReader("{not json}")); err == nil {
		t.Error("expected parse error for invalid index JSON")
	}
}

func TestMeta_MissingDescriptionFails(t *testing.T) {
	m, _ := ParseMeta(strings.NewReader(`{"annotations":{},"manifests":[]}`))
	if !m.MissingDescription() {
		t.Error("expected MissingDescription when description annotation absent")
	}
}

func TestMeta_SummaryParity(t *testing.T) {
	m, _ := ParseMeta(strings.NewReader(sampleIndex))
	s := m.Summary("tools", "sha256:idx", "extra line")
	for _, want := range []string{"## tools", "**Digest:** `sha256:idx`", "extra line"} {
		if !strings.Contains(s, want) {
			t.Errorf("summary missing %q:\n%s", want, s)
		}
	}
}
