// SPDX-License-Identifier: Apache-2.0

package images

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckViolation records one consumer that does not contain the expected digest.
type CheckViolation struct {
	File           string
	Image          string
	ExpectedDigest string
}

// CheckReport is the result of a Check run.
type CheckReport struct {
	Violations []CheckViolation
}

// Check reads every consumer file listed in f and reports any that do not
// contain the canonical digest for their image.
func Check(f File, repoRoot string) (CheckReport, error) {
	var report CheckReport
	for _, entry := range f.Images {
		for _, rel := range entry.Consumers {
			full := filepath.Join(repoRoot, rel)
			data, err := os.ReadFile(full) //nolint:gosec
			if err != nil {
				return report, fmt.Errorf("images check: read %s: %w", rel, err)
			}
			if !strings.Contains(string(data), entry.Digest) {
				report.Violations = append(report.Violations, CheckViolation{
					File:           rel,
					Image:          entry.Name,
					ExpectedDigest: entry.Digest,
				})
			}
		}
	}
	return report, nil
}

// PrintCheckReport writes the report to w. Returns true when no violations found.
func PrintCheckReport(w *os.File, r CheckReport) bool {
	if len(r.Violations) == 0 {
		_, _ = fmt.Fprintln(w, "✅  All consumer files are aligned with images/images.yaml.")
		return true
	}
	_, _ = fmt.Fprintf(w, "❌  Image digest mismatch in %d consumer(s):\n\n", len(r.Violations))
	for _, v := range r.Violations {
		_, _ = fmt.Fprintf(w, "  %-52s  image: %s\n", v.File, v.Image)
		_, _ = fmt.Fprintf(w, "    expected: %s\n\n", v.ExpectedDigest)
	}
	_, _ = fmt.Fprintln(w, "Fix: run 'make sync-images' then commit.")
	return false
}
