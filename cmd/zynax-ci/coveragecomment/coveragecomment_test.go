// SPDX-License-Identifier: Apache-2.0

package coveragecomment

import (
	"strings"
	"testing"
)

func TestGateIcon(t *testing.T) {
	tests := []struct {
		name, pct, gate, want string
	}{
		{"above", "91.2", "90", "✅"},
		{"equal", "90", "90", "✅"},
		{"equal_decimal", "90.0", "90", "✅"},
		{"below", "88.5", "90", "❌"},
		{"empty_pct_defaults_zero", "", "90", "❌"},
		{"empty_gate_defaults_zero", "0", "", "✅"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gateIcon(tt.pct, tt.gate); got != tt.want {
				t.Fatalf("gateIcon(%q,%q)=%q want %q", tt.pct, tt.gate, got, tt.want)
			}
		})
	}
}

func TestRenderEmpty(t *testing.T) {
	got, err := Render(strings.NewReader(""), Gates{}, Meta{SHA: "abcdef1234567", RunNumber: "42", RunURL: "u"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "_No coverage data collected") {
		t.Fatalf("expected no-data note, got:\n%s", got)
	}
	if !strings.Contains(got, "<sub>Run [#42](u) · `abcdef1`</sub>") {
		t.Fatalf("footer mismatch:\n%s", got)
	}
}

func TestRenderShortSHAAndMissingRunNumber(t *testing.T) {
	got, err := Render(strings.NewReader(""), Gates{}, Meta{SHA: "abc"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "<sub>Run [#?]() · `abc`</sub>") {
		t.Fatalf("short-sha footer mismatch:\n%s", got)
	}
}

func TestRenderDefaultGates(t *testing.T) {
	in := "service|api-gateway||91.2\n"
	got, err := Render(strings.NewReader(in), Gates{}, Meta{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "gate ≥ 90%") {
		t.Fatalf("expected default domain gate 90, got:\n%s", got)
	}
}

func TestRenderBlankLinesIgnored(t *testing.T) {
	in := "\nservice|api-gateway||91.2\n\n"
	got, err := Render(strings.NewReader(in), Gates{}, Meta{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "`services/api-gateway`") {
		t.Fatalf("expected service row, got:\n%s", got)
	}
}

// goldenInput / goldenOutput reproduce the exact bytes build-coverage-comment.sh
// emits for a representative multi-category profile (parity check).
const goldenInput = `service|api-gateway||91.2
service|task-broker||90.0
adapter|http||87.4
cli|zynax||78.5
cli|zynax-ci||80.0
python|sdk||92.1
`

const goldenOutput = `<!-- zynax-coverage-report -->
## Coverage Report

### Go services — ` + "`internal/domain`" + ` (gate ≥ 90%)

| Service | Coverage | Gate |
|---------|----------|------|
| ` + "`services/api-gateway`" + ` | **91.2%** | ✅ |
| ` + "`services/task-broker`" + ` | **90.0%** | ✅ |

### Go adapters (gate ≥ 85%)

| Adapter | Coverage | Gate |
|---------|----------|------|
| ` + "`agents/adapters/http`" + ` | **87.4%** | ✅ |

### CLI tools (gate ≥ 79% zynax / 80% zynax-ci)

| Tool | Coverage | Gate |
|------|----------|------|
| ` + "`cmd/zynax`" + ` | **78.5%** | ❌ |
| ` + "`cmd/zynax-ci`" + ` | **80.0%** | ✅ |

### Python (gate ≥ 90%)

| Module | Coverage | Gate |
|--------|----------|------|
| ` + "`agents/sdk`" + ` | **92.1%** | ✅ |

<sub>Run [#7](https://example/run/1) · ` + "`abc1234`" + `</sub>
`

func TestRenderGoldenParity(t *testing.T) {
	gates := Gates{Domain: "90", Adapter: "85", CLIZynax: "79", CLIZynaxCI: "80", Python: "90"}
	meta := Meta{RunNumber: "7", RunURL: "https://example/run/1", SHA: "abc1234ffff"}
	got, err := Render(strings.NewReader(goldenInput), gates, meta)
	if err != nil {
		t.Fatal(err)
	}
	if got != goldenOutput {
		t.Fatalf("golden mismatch.\n--- got ---\n%s\n--- want ---\n%s", got, goldenOutput)
	}
}
