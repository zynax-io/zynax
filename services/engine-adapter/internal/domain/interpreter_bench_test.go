// SPDX-License-Identifier: Apache-2.0

// Package domain contains the core port definitions and value types for the
// engine-adapter service. It has zero imports from api or infrastructure layers.
package domain

import (
	"context"
	"testing"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

// benchIR builds a deterministic 5-state, 10-action workflow that the
// interpreter drives end-to-end on every iteration. Each non-terminal state
// runs two synchronous actions and transitions unconditionally to the next
// state, so a single Run exercises action dispatch, template resolution,
// transition resolution, payload merge, and lifecycle event publication —
// the hot path the regression gate guards (EPIC R canvas step O1, #493).
//
// Shape: s0 → s1 → s2 → s3 → done. Four normal states × two actions = eight
// dispatched capabilities; the two extra actions counted toward the
// "10-action" target are the unconditional-transition matches the canvas
// scopes — the workflow remains representative without async or guarded edges
// that would add nondeterminism to the benchmark.
func benchIR() *zynaxv1.WorkflowIR {
	state := func(id, next string) *zynaxv1.StateIR {
		return normal(id,
			[]*zynaxv1.ActionIR{action(id + "-a"), action(id + "-b")},
			[]*zynaxv1.TransitionIR{transition(id+"-b.completed", next, nil)},
		)
	}
	return buildIR("bench-wf", "s0",
		state("s0", "s1"),
		state("s1", "s2"),
		state("s2", "s3"),
		state("s3", "s4"),
		state("s4", "done"),
		terminal("done"),
	)
}

// BenchmarkIRInterpreter measures the cost of driving a 5-state, 10-action
// workflow through the IR interpreter with stub executor/publisher ports.
// Target: < 1 ms/op (canvas O1). The stubs return immediately so the figure
// isolates interpreter overhead from any I/O.
func BenchmarkIRInterpreter(b *testing.B) {
	ir := benchIR()
	exec := &stubExecutor{}
	interp := &IRInterpreter{}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// A fresh publisher per iteration keeps its event slice from growing
		// unboundedly across iterations and skewing allocation accounting.
		if err := interp.Run(ctx, ir, exec, &stubPublisher{}); err != nil {
			b.Fatalf("Run: %v", err)
		}
	}
}
