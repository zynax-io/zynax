// SPDX-License-Identifier: Apache-2.0

package imagesreport

import (
	"fmt"
	"strings"
)

// RetagInput holds the inputs to the release-tag computation.
type RetagInput struct {
	// Ref is the image repository without a tag, e.g.
	// ghcr.io/zynax-io/zynax/tools.
	Ref string
	// SHA is the commit being published (github.sha).
	SHA string
	// GitRef is the triggering git ref (github.ref), e.g. refs/heads/main or
	// refs/tags/v1.2.3.
	GitRef string
}

// refNamePrefix is the prefix stripped from a refs/tags/<name> git ref.
const refNamePrefix = "refs/tags/"

// FinalTags computes the fully-qualified tags to apply when promoting a
// multi-arch manifest, parity with the tools-image create-*-manifest blocks:
// always main-<sha> and latest; plus the version tag when GitRef is refs/tags/v*.
// The order matches the bash: main-<sha>, [version], latest.
func FinalTags(in RetagInput) ([]string, error) {
	if in.Ref == "" {
		return nil, fmt.Errorf("imagesreport: retag: ref is required")
	}
	if in.SHA == "" {
		return nil, fmt.Errorf("imagesreport: retag: sha is required")
	}
	tags := []string{in.Ref + ":main-" + in.SHA}
	if v := versionTag(in.GitRef); v != "" {
		tags = append(tags, in.Ref+":"+v)
	}
	tags = append(tags, in.Ref+":latest")
	return tags, nil
}

// versionTag returns the v* tag name when gitRef is a refs/tags/v* ref, else "".
// Parity with the bash `grep -q '^refs/tags/v'` test plus ${github.ref_name}.
func versionTag(gitRef string) string {
	if !strings.HasPrefix(gitRef, refNamePrefix) {
		return ""
	}
	name := strings.TrimPrefix(gitRef, refNamePrefix)
	if strings.HasPrefix(name, "v") {
		return name
	}
	return ""
}
