// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// sectionCheck is a single structural requirement for a REASONS Canvas.
type sectionCheck struct {
	prefix   string // Markdown heading prefix to detect (e.g. "## R —")
	name     string // human-readable label for error messages
	minCount int    // minimum required occurrences
}

// canvasSections lists all 7 required REASONS sections plus the second S.
// The two "## S —" entries enforce both Structure (≥1) and Safeguards (≥2).
var canvasSections = []sectionCheck{
	{"## R —", "R — Requirements", 1},
	{"## E —", "E — Entities", 1},
	{"## A —", "A — Approach", 1},
	{"## S —", "S — Structure", 1},
	{"## O —", "O — Operations", 1},
	{"## N —", "N — Norms", 1},
	{"## S —", "S — Safeguards (second S —)", 2},
}

const statusMarker = "**Status:**"

// Canvas validates that a Markdown file contains all seven REASONS section
// headers and a **Status:** field. Reports each missing element by name.
//
// Returns (nil, nil) on success; ([errors…], nil) on structural failures;
// (nil, err) on I/O errors.
func Canvas(filePath string) ([]ValidationError, error) {
	counts, hasStatus, err := scanCanvasFile(filePath)
	if err != nil {
		return nil, err
	}
	return buildCanvasErrors(filePath, counts, hasStatus), nil
}

// scanCanvasFile reads the Markdown file and counts section header occurrences.
func scanCanvasFile(filePath string) (counts map[string]int, hasStatus bool, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, false, fmt.Errorf("validate: read %q: %w", filePath, err)
	}
	defer f.Close()

	counts = make(map[string]int)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, statusMarker) {
			hasStatus = true
		}
		trimmed := strings.TrimSpace(line)
		for _, sec := range canvasSections {
			if strings.HasPrefix(trimmed, sec.prefix) {
				counts[sec.prefix]++
				break
			}
		}
	}
	return counts, hasStatus, scanner.Err()
}

// buildCanvasErrors compares observed counts against requirements.
func buildCanvasErrors(filePath string, counts map[string]int, hasStatus bool) []ValidationError {
	var errs []ValidationError
	for _, sec := range canvasSections {
		if counts[sec.prefix] < sec.minCount {
			errs = append(errs, ValidationError{
				File:    filePath,
				Message: "missing section: " + sec.name,
			})
		}
	}
	if !hasStatus {
		errs = append(errs, ValidationError{
			File:    filePath,
			Message: "missing **Status:** field",
		})
	}
	return errs
}
