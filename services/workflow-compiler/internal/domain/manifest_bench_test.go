// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	_ "embed"
	"testing"
)

// codeReviewYAML is the reference code-review workflow used as a realistic
// parse target for BenchmarkParseManifest. It is embedded (not read from the
// repo-root spec/ tree) so the domain benchmark stays self-contained and runs
// from the package directory under GOWORK=off, with no dependency on the
// working directory. Keep it in sync with spec/workflows/examples/code-review.yaml
// (EPIC R canvas step O1, #493).
//
//go:embed testdata/code-review.yaml
var codeReviewYAML []byte

// BenchmarkParseManifest measures the cost of parsing the realistic
// code-review workflow manifest through ParseManifest. Target: < 500 µs/op
// (canvas O1). The Go benchmark harness drives the parse repeatedly (b.N
// iterations), realising the canvas's "parse a real workflow 1000×" intent
// without hard-coding the loop count. A guard fails the benchmark if the
// fixture ever stops parsing cleanly, so the benchmark also doubles as a
// regression check on the embedded example.
func BenchmarkParseManifest(b *testing.B) {
	ctx := context.Background()
	if _, errs := ParseManifest(ctx, codeReviewYAML); len(errs) != 0 {
		b.Fatalf("fixture must parse cleanly, got %d errors: %v", len(errs), errs)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseManifest(ctx, codeReviewYAML)
	}
}
