// SPDX-License-Identifier: Apache-2.0

// Package imagesreport holds the deterministic, tested logic behind the
// `zynax-ci images meta|cleanup|retag` verbs. It is the Go replacement for the
// report-image-meta composite action, the pr-image-cleanup gh/jq blocks, and
// the tools-image retag/prune blocks (ADR-036, M7 EPIC S step S.5).
//
// The external primitives stay in the shell: the workflow pipes the GHCR index
// manifest JSON (curl), the packages-API versions list (gh api), and runs
// crane/imagetools for the actual copy/delete. This package only computes the
// decisions — annotation checks, version-id selection, and the release-tag
// list — so they get unit tests and govulncheck instead of living untested in
// YAML. Per-platform size budgeting stays in the workflow's curl loop.
package imagesreport

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ExpectedAttestations is the dual-arch SLSA provenance count (ADR-025).
const ExpectedAttestations = 2

// index is the parsed OCI image index (manifest list).
type index struct {
	Annotations map[string]string `json:"annotations"`
	Manifests   []struct {
		Platform struct {
			OS string `json:"os"`
		} `json:"platform"`
	} `json:"manifests"`
}

// Meta is the result of inspecting an index manifest.
type Meta struct {
	Description   string
	Title         string
	Attestations  int
	IndexResolved bool
}

// ParseMeta reads the index manifest JSON and returns the annotation report. An
// empty or whitespace-only body is non-fatal (IndexResolved stays false), parity
// with the bash retry-then-skip path when GHCR lags after a push.
func ParseMeta(indexJSON io.Reader) (Meta, error) {
	data, err := io.ReadAll(indexJSON)
	if err != nil {
		return Meta{}, fmt.Errorf("imagesreport: read index: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return Meta{}, nil
	}
	var idx index
	if err := json.Unmarshal(data, &idx); err != nil {
		return Meta{}, fmt.Errorf("imagesreport: parse index: %w", err)
	}
	m := Meta{
		IndexResolved: true,
		Description:   idx.Annotations["org.opencontainers.image.description"],
		Title:         idx.Annotations["org.opencontainers.image.title"],
		Attestations:  countAttestations(idx),
	}
	return m, nil
}

// countAttestations counts the unknown-OS (SLSA provenance) descriptors.
func countAttestations(idx index) int {
	n := 0
	for _, d := range idx.Manifests {
		if d.Platform.OS == "unknown" {
			n++
		}
	}
	return n
}
