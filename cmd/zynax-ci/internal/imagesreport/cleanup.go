// SPDX-License-Identifier: Apache-2.0

package imagesreport

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
)

// version is one entry of the GitHub packages-API versions list.
type version struct {
	ID       int64 `json:"id"`
	Metadata struct {
		Container struct {
			Tags []string `json:"tags"`
		} `json:"container"`
	} `json:"metadata"`
	UpdatedAt string `json:"updated_at"`
}

// numericID guards the DELETE call (parity with the bash `^[0-9]+$` check). The
// API id is already an int64, so any decoded value is numeric; this stays as a
// defensive boundary mirroring the original.
func numericID(id int64) bool { return id >= 0 }

// mainShaRe matches the per-commit tag `main-<hex>` (parity with the prune
// jq `test("^main-[a-f0-9]")`).
var mainShaRe = regexp.MustCompile(`^main-[a-f0-9]`)

// versionTagRe matches a release tag `v<digit>...` (parity with `test("^v[0-9]")`).
var versionTagRe = regexp.MustCompile(`^v[0-9]`)

// decodeVersions parses a packages-API versions JSON array.
func decodeVersions(r io.Reader) ([]version, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("imagesreport: read versions: %w", err)
	}
	var vs []version
	if err := json.Unmarshal(data, &vs); err != nil {
		return nil, fmt.Errorf("imagesreport: parse versions: %w", err)
	}
	return vs, nil
}

// SelectByTag returns the version ids whose tag set contains exactTag — the
// pr-image-cleanup selection (parity with `select(.tags[]? == "pr-<sha>")`).
// A missing package or empty list yields no ids (a quiet no-op upstream).
func SelectByTag(versionsJSON io.Reader, exactTag string) ([]int64, error) {
	vs, err := decodeVersions(versionsJSON)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for _, v := range vs {
		if !numericID(v.ID) {
			continue
		}
		for _, t := range v.Metadata.Container.Tags {
			if t == exactTag {
				ids = append(ids, v.ID)
				break
			}
		}
	}
	return ids, nil
}

// SelectPrunable returns the version ids of stale per-commit builds to delete:
// versions whose tags include a main-<sha> tag and whose tags never include
// latest/main or any v<digit> release tag, sorted newest-first, keeping the
// first `keep` (parity with the tools-image prune jq pipeline).
func SelectPrunable(versionsJSON io.Reader, keep int) ([]int64, error) {
	vs, err := decodeVersions(versionsJSON)
	if err != nil {
		return nil, err
	}
	candidates := filterPrunable(vs)
	sortNewestFirst(candidates)
	if keep < 0 {
		keep = 0
	}
	if keep >= len(candidates) {
		return nil, nil
	}
	ids := make([]int64, 0, len(candidates)-keep)
	for _, v := range candidates[keep:] {
		ids = append(ids, v.ID)
	}
	return ids, nil
}

// filterPrunable keeps versions that are a per-commit build and carry no
// protected (latest/main/v*) tag.
func filterPrunable(vs []version) []version {
	var out []version
	for _, v := range vs {
		if isPerCommitBuild(v) && !hasProtectedTag(v) {
			out = append(out, v)
		}
	}
	return out
}

// isPerCommitBuild reports whether any tag matches main-<sha>.
func isPerCommitBuild(v version) bool {
	for _, t := range v.Metadata.Container.Tags {
		if mainShaRe.MatchString(t) {
			return true
		}
	}
	return false
}

// hasProtectedTag reports whether any tag is latest, main, or a v* release tag.
func hasProtectedTag(v version) bool {
	for _, t := range v.Metadata.Container.Tags {
		if t == "latest" || t == "main" || versionTagRe.MatchString(t) {
			return true
		}
	}
	return false
}

// sortNewestFirst orders versions by UpdatedAt descending (RFC3339 strings sort
// lexicographically, matching the jq `sort_by(.ts) | reverse`).
func sortNewestFirst(vs []version) {
	for i := 1; i < len(vs); i++ {
		for j := i; j > 0 && vs[j].UpdatedAt > vs[j-1].UpdatedAt; j-- {
			vs[j], vs[j-1] = vs[j-1], vs[j]
		}
	}
}

// FormatIDs renders ids one-per-line for the shell DELETE loop to consume.
func FormatIDs(ids []int64) string {
	var b []byte
	for _, id := range ids {
		b = append(b, []byte(strconv.FormatInt(id, 10))...)
		b = append(b, '\n')
	}
	return string(b)
}
