// SPDX-License-Identifier: Apache-2.0

// Package releasehelpers holds the deterministic decision logic ported from the
// release.yml gh-api/jq + assembly run: blocks (ADR-036). It computes which
// promoted main-<sha> image a release tag promotes (the per-service matrix that
// feeds cosign/crane) and assembles the GitHub Release notes. The external
// primitives — git, gh, docker, cosign, crane, syft — stay in the workflow
// shell; this package only makes their decisions testable.
package releasehelpers

import (
	"fmt"
	"strings"
)

// shortSHALen is the length of the short commit SHA used in the main-<sha:0:8>
// fallback tag (parity with the bash ${sha:0:8}).
const shortSHALen = 8

// mainTagPrefix is the per-commit promoted-image tag prefix (main-<sha>).
const mainTagPrefix = "main-"

// SelectSourceTag picks the source image tag a release promotes, parity with the
// release.yml retag-version "Resolve version and source image" block: walk the
// first-parent commit SHAs newest-first and return the first main-<sha> (or its
// main-<sha[:8]> short form) that exists in the promoted tag set. An empty
// result (no match within the candidate window) is the caller's signal to
// exclude the service from the release — it is not an error.
func SelectSourceTag(candidateSHAs, existingTags []string) string {
	promoted := promotedSet(existingTags)
	if len(promoted) == 0 {
		return ""
	}
	for _, sha := range candidateSHAs {
		if sha == "" {
			continue
		}
		if full := mainTagPrefix + sha; promoted[full] {
			return full
		}
		if len(sha) >= shortSHALen {
			if short := mainTagPrefix + sha[:shortSHALen]; promoted[short] {
				return short
			}
		}
	}
	return ""
}

// promotedSet collects the main-<...> tags from a GHCR tag list into a lookup
// set (parity with the bash `grep '^main-'` filter over the existing tags).
func promotedSet(existingTags []string) map[string]bool {
	set := make(map[string]bool, len(existingTags))
	for _, t := range existingTags {
		t = strings.TrimSpace(t)
		if strings.HasPrefix(t, mainTagPrefix) {
			set[t] = true
		}
	}
	return set
}

// VersionRef is the fully-qualified promote source/target pair for one service.
type VersionRef struct {
	Service string // service image name, e.g. api-gateway
	Source  string // <prefix>/<service>:main-<sha>
	Target  string // <prefix>/<service>:<version>
}

// PromoteRef builds the source→target image references a service promote applies,
// given the image prefix (ghcr.io/zynax-io/zynax), the service name, the matched
// source tag (from SelectSourceTag), and the release version. It returns ok=false
// when srcTag is empty so the caller skips the service (parity with the bash
// `if [ -z "${SRC_TAG}" ]` early-exit guard).
func PromoteRef(prefix, service, srcTag, version string) (VersionRef, bool, error) {
	if prefix == "" {
		return VersionRef{}, false, fmt.Errorf("releasehelpers: matrix: prefix is required")
	}
	if service == "" {
		return VersionRef{}, false, fmt.Errorf("releasehelpers: matrix: service is required")
	}
	if version == "" {
		return VersionRef{}, false, fmt.Errorf("releasehelpers: matrix: version is required")
	}
	if srcTag == "" {
		return VersionRef{}, false, nil
	}
	base := prefix + "/" + service
	return VersionRef{
		Service: service,
		Source:  base + ":" + srcTag,
		Target:  base + ":" + version,
	}, true, nil
}
