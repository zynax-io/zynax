// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"regexp"
	"strings"
)

// ansiEscapeRe matches ANSI/VT escape sequences: CSI (ESC[…), OSC (ESC]…BEL/ST),
// and simple ESC-prefixed forms. Stripping them stops attacker-influenced
// workflow output from driving the user's terminal (ADR-042 §6).
var ansiEscapeRe = regexp.MustCompile("\x1b(\\[[0-9;?]*[ -/]*[@-~]|\\][^\x07\x1b]*(\x07|\x1b\\\\)|[@-Z\\\\-_])")

// sanitizeForTTY strips ANSI escape sequences and C0/C1 control characters —
// keeping only the common whitespace \t, \n, \r — from untrusted output before
// it is printed to a terminal. Workflow outputs and completion text are
// attacker-influenced (ADR-042 §6).
func sanitizeForTTY(s string) string {
	s = ansiEscapeRe.ReplaceAllString(s, "")
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			return r
		case r < 0x20, r >= 0x7f && r <= 0x9f:
			return -1
		default:
			return r
		}
	}, s)
}
