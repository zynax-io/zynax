// SPDX-License-Identifier: Apache-2.0

package imagesreport

import (
	"fmt"
	"strings"
)

// MissingDescription reports whether the FAIL condition fired: the index was
// resolved but carries no description annotation (parity with the bash hard
// exit 1). When the index never resolved the check is skipped (non-fatal).
func (m Meta) MissingDescription() bool {
	return m.IndexResolved && m.Description == ""
}

// Summary renders the GITHUB_STEP_SUMMARY markdown block for an image, parity
// with report-image-meta's header. When the index was not resolved it emits the
// skipped-report block instead (non-fatal GHCR read lag).
func (m Meta) Summary(label, digest, extra string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n", label)
	fmt.Fprintf(&b, "**Digest:** `%s`\n", digest)
	if !m.IndexResolved {
		b.WriteString("\n_Metadata report skipped: GHCR index manifest unavailable after retries (non-fatal)._\n")
		return b.String()
	}
	if extra != "" {
		fmt.Fprintf(&b, "\n%s\n", extra)
	}
	return b.String()
}
