// SPDX-License-Identifier: Apache-2.0

package coveragecomment

import (
	"fmt"
	"strings"
)

// filter returns the rows of a given kind, preserving input order.
func filter(rows []row, kind string) []row {
	var out []row
	for _, r := range rows {
		if r.kind == kind {
			out = append(out, r)
		}
	}
	return out
}

func renderServices(b *strings.Builder, rows []row, gate string) {
	rs := filter(rows, "service")
	if len(rs) == 0 {
		return
	}
	fmt.Fprintf(b, "### Go services — `internal/domain` (gate ≥ %s%%)\n\n", gate)
	b.WriteString("| Service | Coverage | Gate |\n|---------|----------|------|\n")
	for _, r := range rs {
		fmt.Fprintf(b, "| `services/%s` | **%s%%** | %s |\n", r.name, r.pct, gateIcon(r.pct, gate))
	}
	b.WriteString("\n")
}

func renderAdapters(b *strings.Builder, rows []row, gate string) {
	rs := filter(rows, "adapter")
	if len(rs) == 0 {
		return
	}
	fmt.Fprintf(b, "### Go adapters (gate ≥ %s%%)\n\n", gate)
	b.WriteString("| Adapter | Coverage | Gate |\n|---------|----------|------|\n")
	for _, r := range rs {
		fmt.Fprintf(b, "| `agents/adapters/%s` | **%s%%** | %s |\n", r.name, r.pct, gateIcon(r.pct, gate))
	}
	b.WriteString("\n")
}

func renderCLI(b *strings.Builder, rows []row, zynaxGate, zynaxCIGate string) {
	rs := filter(rows, "cli")
	if len(rs) == 0 {
		return
	}
	fmt.Fprintf(b, "### CLI tools (gate ≥ %s%% zynax / %s%% zynax-ci)\n\n", zynaxGate, zynaxCIGate)
	b.WriteString("| Tool | Coverage | Gate |\n|------|----------|------|\n")
	for _, r := range rs {
		g := zynaxCIGate
		if r.name == "zynax" {
			g = zynaxGate
		}
		fmt.Fprintf(b, "| `cmd/%s` | **%s%%** | %s |\n", r.name, r.pct, gateIcon(r.pct, g))
	}
	b.WriteString("\n")
}

func renderPython(b *strings.Builder, rows []row, gate string) {
	rs := filter(rows, "python")
	if len(rs) == 0 {
		return
	}
	fmt.Fprintf(b, "### Python (gate ≥ %s%%)\n\n", gate)
	b.WriteString("| Module | Coverage | Gate |\n|--------|----------|------|\n")
	for _, r := range rs {
		fmt.Fprintf(b, "| `agents/%s` | **%s%%** | %s |\n", r.name, r.pct, gateIcon(r.pct, gate))
	}
	b.WriteString("\n")
}
