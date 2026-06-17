// SPDX-License-Identifier: Apache-2.0

// Package coveragecomment renders the PR coverage markdown comment from a
// coverage results file. It is the tested Go replacement for
// .github/scripts/build-coverage-comment.sh (ADR-036, M7 EPIC S step S.2).
package coveragecomment

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Gates holds the per-category coverage thresholds (percentages). Zero values
// fall back to the documented defaults via gateOrDefault.
type Gates struct {
	Domain     string
	Adapter    string
	CLIZynax   string
	CLIZynaxCI string
	Python     string
}

// Meta holds the footer fields rendered at the bottom of the comment.
type Meta struct {
	RunNumber string
	RunURL    string
	SHA       string
}

// row is a single parsed coverage-results line: type|name|pkg|pct.
type row struct {
	kind string
	name string
	pct  string
}

// defaults mirror the bash fallbacks in build-coverage-comment.sh.
var defaults = map[string]string{
	"domain": "90", "adapter": "85", "zynax": "79", "zynax-ci": "80", "python": "90",
}

// Render parses the coverage-results stream and returns the markdown comment,
// byte-identical to build-coverage-comment.sh on the same input.
func Render(results io.Reader, gates Gates, meta Meta) (string, error) {
	rows, err := parse(results)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("<!-- zynax-coverage-report -->\n## Coverage Report\n\n")
	renderServices(&b, rows, gateOrDefault(gates.Domain, "domain"))
	renderAdapters(&b, rows, gateOrDefault(gates.Adapter, "adapter"))
	renderCLI(&b, rows, gateOrDefault(gates.CLIZynax, "zynax"), gateOrDefault(gates.CLIZynaxCI, "zynax-ci"))
	renderPython(&b, rows, gateOrDefault(gates.Python, "python"))
	if len(rows) == 0 {
		b.WriteString("_No coverage data collected — tests may have been skipped._\n\n")
	}
	fmt.Fprintf(&b, "<sub>Run [#%s](%s) · `%s`</sub>\n", orQuestion(meta.RunNumber), meta.RunURL, sha7(meta.SHA))
	return b.String(), nil
}

// parse reads pipe-delimited rows, ignoring blank lines (parity with grep).
func parse(r io.Reader) ([]row, error) {
	var rows []row
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.SplitN(line, "|", 4)
		if len(f) < 4 {
			continue
		}
		rows = append(rows, row{kind: f[0], name: f[1], pct: f[3]})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read coverage results: %w", err)
	}
	return rows, nil
}

// gateIcon returns ✅ when pct ≥ gate, else ❌ (parity with the awk helper).
func gateIcon(pct, gate string) string {
	p, g := toFloat(pct), toFloat(gate)
	if p >= g {
		return "✅"
	}
	return "❌"
}

func toFloat(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return f
}

func gateOrDefault(v, key string) string {
	if v == "" {
		return defaults[key]
	}
	return v
}

func orQuestion(s string) string {
	if s == "" {
		return "?"
	}
	return s
}

func sha7(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}
