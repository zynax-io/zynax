// SPDX-License-Identifier: Apache-2.0

// Package validate provides validators for REASONS Canvases, YAML manifests,
// JSON Schemas, and capability definitions used by zynax-ci.
package validate

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidationError is a single hard failure from validation.
type ValidationError struct {
	File    string `json:"file"`
	Path    string `json:"path,omitempty"` // JSON Pointer to the failing element (optional)
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: at %s: %s", e.File, e.Path, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.File, e.Message)
}

// ValidationWarning is an advisory finding that does not fail validation.
type ValidationWarning struct {
	File    string `json:"file"`
	Message string `json:"message"`
}

// validStatuses is the closed set of canvas lifecycle statuses. Draft → Aligned →
// Implemented → Synced is the active path; Superseded and Rejected are terminal
// states for a canvas whose design was abandoned or replaced (e.g. its ADR was
// Rejected), which must remain valid so a closed/superseded canvas does not fail
// the repo-wide canvas validation.
var validStatuses = map[string]bool{
	"Draft": true, "Aligned": true, "Implemented": true, "Synced": true,
	"Superseded": true, "Rejected": true,
}

type sectionCheck struct {
	prefix   string
	name     string
	minCount int
}

// canvasSections mirrors the seven REASONS headings.
// The two "## S —" entries enforce both Structure (≥1) and Safeguards (≥2).
var canvasSections = []sectionCheck{
	{"## R —", "R — Requirements", 1},
	{"## E —", "E — Entities", 1},
	{"## A —", "A — Approach", 1},
	{"## S —", "S — Structure", 1},
	{"## O —", "O — Operations", 1},
	{"## N —", "N — Norms", 1},
	{"## S —", "Safeguards (second ## S — section)", 2},
}

var requiredFields = []string{"**Issue:**", "**Author:**", "**Date:**", "**Status:**"}

const securityMarker = "Context Security"

var statusRe = regexp.MustCompile(`\*\*Status:\*\*\s*(\w+)`)

// Canvas validates a single canvas.md file for full structural completeness.
//
// Checks (full parity with tools/validate_canvas.py):
//   - Seven REASONS sections present
//   - **Issue:**, **Author:**, **Date:**, **Status:** header fields present
//   - Status value is one of Draft, Aligned, Implemented, Synced, Superseded, Rejected
//   - Context Security checklist marker present
//   - canvas.private.md not committed alongside canvas.md
//
// Returns errors (hard failures), warnings (advisory), and any I/O error.
func Canvas(filePath string) ([]ValidationError, []ValidationWarning, error) {
	f, err := os.Open(filePath) //nolint:gosec // filePath is caller-supplied canvas path
	if err != nil {
		return nil, nil, fmt.Errorf("validate: read %q: %w", filePath, err)
	}
	defer func() { _ = f.Close() }()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("validate: scan %q: %w", filePath, err)
	}

	text := strings.Join(lines, "\n")
	var errs []ValidationError
	var warns []ValidationWarning

	checkHeaderFields(filePath, text, &errs)
	checkStatus(filePath, text, &errs, &warns)
	checkSections(filePath, lines, &errs)
	checkSecurityMarker(filePath, text, &errs)
	checkNoPrivateFile(filePath, &errs)

	return errs, warns, nil
}

func checkHeaderFields(filePath, text string, errs *[]ValidationError) {
	for _, field := range requiredFields {
		if !strings.Contains(text, field) {
			*errs = append(*errs, ValidationError{File: filePath, Message: "missing header field: " + field})
		}
	}
}

func checkStatus(filePath, text string, errs *[]ValidationError, warns *[]ValidationWarning) {
	m := statusRe.FindStringSubmatch(text)
	if m == nil {
		return
	}
	status := m[1]
	if !validStatuses[status] {
		*errs = append(*errs, ValidationError{
			File:    filePath,
			Message: fmt.Sprintf("invalid Status %q — must be one of: Aligned, Draft, Implemented, Synced, Superseded, Rejected", status),
		})
		return
	}
	if status == "Draft" {
		*warns = append(*warns, ValidationWarning{
			File:    filePath,
			Message: "Canvas is Draft — must reach Aligned before /spdd-generate runs",
		})
	}
}

func checkSections(filePath string, lines []string, errs *[]ValidationError) {
	counts := make(map[string]int)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, sec := range canvasSections {
			if strings.HasPrefix(trimmed, sec.prefix) {
				counts[sec.prefix]++
				break
			}
		}
	}
	for _, sec := range canvasSections {
		if counts[sec.prefix] < sec.minCount {
			*errs = append(*errs, ValidationError{
				File:    filePath,
				Message: "missing section: " + sec.name,
			})
		}
	}
}

func checkSecurityMarker(filePath, text string, errs *[]ValidationError) {
	if !strings.Contains(text, securityMarker) {
		*errs = append(*errs, ValidationError{
			File:    filePath,
			Message: "missing Context Security checklist in Safeguards section",
		})
	}
}

func checkNoPrivateFile(filePath string, errs *[]ValidationError) {
	privatePath := filepath.Join(filepath.Dir(filePath), "canvas.private.md")
	if _, err := os.Stat(privatePath); err == nil {
		*errs = append(*errs, ValidationError{
			File:    filePath,
			Message: fmt.Sprintf("canvas.private.md found at %s — must be gitignored, never committed", privatePath),
		})
	}
}
