// SPDX-License-Identifier: Apache-2.0

package benchgate

import (
	"strings"
	"testing"
)

// noRegression is a benchstat table where every delta is within threshold or "~".
const noRegression = `goos: linux
goarch: amd64
                  │  baseline   │           new            │
                  │   sec/op    │   sec/op     vs base     │
Compile-8            1.20µ ± 2%   1.18µ ± 3%        ~ (p=0.31)
Interpret-8          3.00µ ± 1%   3.45µ ± 2%   +15.00% (p=0.00)
`

// regression has one row over the 20% threshold.
const regression = `                  │  baseline   │           new            │
                  │   sec/op    │   sec/op     vs base     │
Compile-8            1.20µ ± 2%   1.49µ ± 3%   +24.10% (p=0.00)
Interpret-8          3.00µ ± 1%   2.85µ ± 2%    -5.00% (p=0.00)
`

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name      string
		report    string
		threshold float64
		want      int     // expected regression count
		wantPct   float64 // expected first regression magnitude (0 if none)
	}{
		{"no_regression_default", noRegression, DefaultThresholdPct, 0, 0},
		{"regression_default", regression, DefaultThresholdPct, 1, 24.10},
		{"boundary_equal_not_flagged", "Foo-8  +20.00% (p=0.00)\n", 20, 0, 0},
		{"boundary_above_flagged", "Foo-8  +20.01% (p=0.00)\n", 20, 1, 20.01},
		{"negative_delta_ignored", "Foo-8  -99.00% (p=0.00)\n", 20, 0, 0},
		{"tilde_ignored", "Foo-8  ~ (p=0.31)\n", 20, 0, 0},
		{"lower_threshold_catches_15pct", noRegression, 10, 1, 15.0},
		{"empty_report", "", DefaultThresholdPct, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Evaluate(strings.NewReader(tt.report), tt.threshold)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(res.Regressions); got != tt.want {
				t.Fatalf("regressions=%d want %d (%+v)", got, tt.want, res.Regressions)
			}
			if tt.want > 0 && res.Regressions[0].Pct != tt.wantPct {
				t.Fatalf("first pct=%v want %v", res.Regressions[0].Pct, tt.wantPct)
			}
			if res.Regressed() != (tt.want > 0) {
				t.Fatalf("Regressed()=%v want %v", res.Regressed(), tt.want > 0)
			}
		})
	}
}

func TestSummary(t *testing.T) {
	pass, _ := Evaluate(strings.NewReader(noRegression), DefaultThresholdPct)
	if s := Summary(pass, false); !strings.Contains(s, "✅ No benchmark regressed beyond 20%.") {
		t.Fatalf("pass summary mismatch:\n%s", s)
	}

	fail, _ := Evaluate(strings.NewReader(regression), DefaultThresholdPct)
	warn := Summary(fail, false)
	if !strings.Contains(warn, "⚠️  REGRESSION:") || !strings.Contains(warn, "fail-open mode") {
		t.Fatalf("warn summary mismatch:\n%s", warn)
	}
	if strings.Contains(warn, "ENFORCED") {
		t.Fatalf("warn summary must not mention ENFORCED:\n%s", warn)
	}

	enforced := Summary(fail, true)
	if !strings.Contains(enforced, "❌ Benchmark regression gate ENFORCED — failing build.") {
		t.Fatalf("enforced summary mismatch:\n%s", enforced)
	}
	if strings.Contains(enforced, "fail-open mode") {
		t.Fatalf("enforced summary must not mention fail-open:\n%s", enforced)
	}
}
