// SPDX-License-Identifier: Apache-2.0

package images

import (
	"fmt"
	"regexp"
	"strings"
)

// DigestRe validates a full image digest string (sha256:<64 hex>).
var DigestRe = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

var (
	nameLineRe   = regexp.MustCompile(`^\s*-\s+name:\s+(\S+)\s*$`)
	digestLineRe = regexp.MustCompile(`^(\s*digest:\s*)sha256:[0-9a-f]{64}\s*$`)
)

// Upsert sets the digest for the entry named name in images.yaml text, doing a
// line-based edit so comments and formatting are preserved (a YAML round-trip
// would not guarantee that). It returns the new text and an action of
// "updated", "unchanged", or "added".
//
// If name has no entry yet, a new one is appended (ref must be non-empty) with
// an empty consumers list, so first-time promotions self-register.
func Upsert(text, name, ref, digest string) (newText, action string, err error) {
	var out strings.Builder
	current := ""
	found := false
	changed := false
	for _, line := range splitKeepEnds(text) {
		if m := nameLineRe.FindStringSubmatch(line); m != nil {
			current = m[1]
		}
		if current == name {
			found = true
			if dm := digestLineRe.FindStringSubmatch(line); dm != nil {
				newLine := dm[1] + digest + "\n"
				if newLine != line {
					changed = true
				}
				line = newLine
			}
		}
		out.WriteString(line)
	}

	if found {
		if changed {
			return out.String(), "updated", nil
		}
		return out.String(), "unchanged", nil
	}

	if ref == "" {
		return "", "", fmt.Errorf("no entry named %q and --ref not provided", name)
	}
	result := out.String()
	if len(result) > 0 && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	result += fmt.Sprintf("\n  - name: %s\n    ref: %s\n    digest: %s\n    consumers: []\n", name, ref, digest)
	return result, "added", nil
}

// splitKeepEnds splits text into lines, keeping each line's trailing newline,
// mirroring Python's str.splitlines(keepends=True) for "\n"-terminated text.
func splitKeepEnds(text string) []string {
	if text == "" {
		return nil
	}
	parts := strings.SplitAfter(text, "\n")
	// SplitAfter leaves a trailing "" when text ends in "\n"; drop it so we
	// don't emit a phantom empty line.
	if last := len(parts) - 1; last >= 0 && parts[last] == "" {
		parts = parts[:last]
	}
	return parts
}
