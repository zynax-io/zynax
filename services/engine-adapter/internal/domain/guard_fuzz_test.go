// SPDX-License-Identifier: Apache-2.0

package domain

// Fuzz target for the CEL guard evaluator (#1417 — 2026-06-18 architecture
// review T2.1/R3: the guard evaluator sits on untrusted manifest input with no
// fuzz coverage). Mirrors the FuzzParseManifest house style (workflow-compiler,
// #1210): CI runs the seed corpus only via `go test`; longer campaigns run
// locally via `make fuzz` (FUZZ_SERVICES includes engine-adapter).
//
// The fuzz body asserts only structural invariants — never a specific verdict,
// since fuzz inputs are unknown by definition:
//   - evalGuard never panics, whatever the expression or context.
//   - Fail-closed: an empty/whitespace expression is always false.
//   - Deterministic: the same (expr, ctx) evaluates to the same verdict twice
//     (the cel.Program cache must not change the answer).
//
// Note: evalGuard caches one cel.Program per unique expression (sync.Map), so
// a long campaign grows that cache monotonically — expected, bounded by the
// campaign length, and irrelevant for the seed-only CI run.

import (
	"io"
	"log/slog"
	"strings"
	"testing"
)

func FuzzEvalGuard(f *testing.F) {
	// evalGuard warns on every compile/eval failure; almost every fuzz input
	// fails to compile, so silence slog for the duration of the target.
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	f.Cleanup(func() { slog.SetDefault(prev) })

	// Real guard shapes from the interpreter tests and spec/workflows/examples/
	// (the template-syntax examples exercise the compile-rejection path), plus
	// hand-picked edge cases.
	for _, expr := range []string{
		`ctx.status == "approved"`,
		`ctx.status != "pending"`,
		`"approved" == "approved"`,
		`severity == 'low'`,
		`escalation_count < 2`,
		`{{ .context.iteration_count }} < 3`,
		"true",
		"false",
		"",
		"   ",
		"ctx",
		`ctx["status"] == "approved"`,
		`ctx.missing == "x"`,
		"1 + 1",
		"((((((((((true))))))))))",
		"\x00",
	} {
		f.Add(expr, "status", "approved")
	}

	f.Fuzz(func(t *testing.T, expr, key, val string) {
		ctx := map[string]string{"status": "approved"}
		if key != "" {
			ctx[key] = val
		}

		got := evalGuard(expr, ctx)
		if again := evalGuard(expr, ctx); again != got {
			t.Errorf("non-deterministic: evalGuard(%q) = %v, then %v", expr, got, again)
		}
		if strings.TrimSpace(expr) == "" && got {
			t.Errorf("fail-closed violated: blank expression %q evaluated to true", expr)
		}
	})
}
