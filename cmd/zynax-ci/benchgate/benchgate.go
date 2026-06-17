// SPDX-License-Identifier: Apache-2.0

// Package benchgate parses benchstat output and applies the benchmark
// regression threshold. It is the tested Go replacement for
// tools/ci/bench-regression.sh (ADR-036, M7 EPIC S step S.3).
//
// benchstat prints a "vs base" percentage like "+24.10%" on rows whose metric
// changed; "~" means no statistically significant change. A positive delta on a
// time/op (sec/op) row is a regression. The gate flags any positive delta whose
// magnitude exceeds the threshold percent.
package benchgate

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// DefaultThresholdPct mirrors THRESHOLD_PCT's default in bench-regression.sh.
const DefaultThresholdPct = 20.0

// pctRe matches a signed percentage token, e.g. +24.10% or -5.00% (parity with
// the `grep -oE '[+-][0-9]+(\.[0-9]+)?%'` extraction in the bash).
var pctRe = regexp.MustCompile(`[+-][0-9]+(\.[0-9]+)?%`)

// Regression is a single benchstat row flagged as a regression.
type Regression struct {
	Line string  // the full benchstat line that regressed
	Pct  float64 // the positive delta magnitude (percent)
}

// Result holds the outcome of evaluating a benchstat report.
type Result struct {
	Threshold   float64
	Regressions []Regression
}

// Regressed reports whether any row regressed beyond the threshold.
func (r Result) Regressed() bool { return len(r.Regressions) > 0 }

// Evaluate parses a benchstat report and returns the regressions that exceed
// thresholdPct. Decisions are identical to the bash: for each line, take the
// first signed percentage token; only positive deltas count; flag when the
// magnitude is strictly greater than the threshold.
func Evaluate(report io.Reader, thresholdPct float64) (Result, error) {
	res := Result{Threshold: thresholdPct}
	sc := bufio.NewScanner(report)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		pct, ok := positiveDelta(line)
		if !ok {
			continue
		}
		if pct > thresholdPct {
			res.Regressions = append(res.Regressions, Regression{Line: line, Pct: pct})
		}
	}
	if err := sc.Err(); err != nil {
		return Result{}, fmt.Errorf("read benchstat report: %w", err)
	}
	return res, nil
}

// positiveDelta returns the magnitude of the first signed percentage token on
// the line and whether it is a positive (slower = regression) delta.
func positiveDelta(line string) (float64, bool) {
	tok := pctRe.FindString(line)
	if tok == "" {
		return 0, false
	}
	num := strings.TrimSuffix(tok, "%")
	if !strings.HasPrefix(num, "+") {
		return 0, false // only positive deltas are regressions
	}
	mag, err := strconv.ParseFloat(strings.TrimPrefix(num, "+"), 64)
	if err != nil {
		return 0, false
	}
	return mag, true
}

// Summary renders the human-readable gate summary, parity with the bash echoes.
// enforce selects the fail-closed wording; the caller decides the exit code.
func Summary(res Result, enforce bool) string {
	var b strings.Builder
	if !res.Regressed() {
		fmt.Fprintf(&b, "✅ No benchmark regressed beyond %s%%.\n", trimPct(res.Threshold))
		return b.String()
	}
	for _, r := range res.Regressions {
		fmt.Fprintf(&b, "⚠️  REGRESSION: %s  (>%s%%)\n", r.Line, trimPct(res.Threshold))
	}
	if enforce {
		b.WriteString("❌ Benchmark regression gate ENFORCED — failing build.\n")
		return b.String()
	}
	b.WriteString("⚠️  Benchmark regression detected, but gate is in fail-open mode\n")
	b.WriteString("    (baseline not yet stabilised over 3 runs). Set BENCH_GATE_ENFORCE=true to block.\n")
	return b.String()
}

// trimPct renders a threshold without a trailing ".0" so "20" stays "20".
func trimPct(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
