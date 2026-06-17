// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"embed"
	"io/fs"
	"testing"
)

// fuzzCorpus embeds the seed corpus for FuzzParseManifest. The files are copies
// of spec/workflows/examples/*.yaml — the canvas-mandated seed source (EPIC R
// canvas step R.2, #1210). They are embedded (not read from the repo-root spec/
// tree) so the domain fuzz target stays self-contained and runs from the package
// directory under GOWORK=off, with no dependency on the working directory —
// mirroring the BenchmarkParseManifest embed pattern. Keep them in sync with
// spec/workflows/examples/.
//
//go:embed testdata/fuzz_corpus/*.yaml
var fuzzCorpus embed.FS

// FuzzParseManifest fuzzes the YAML->IR compiler entry point ParseManifest with
// untrusted bytes. ParseManifest accepts arbitrary external manifest input, so a
// malformed manifest must never panic — it must return a structured ParseErrors
// instead (EPIC R canvas, review gap M2/R14; #1210).
//
// The seed corpus is the set of real example workflows under
// spec/workflows/examples/ (valid Workflow manifests plus non-Workflow kinds such
// as Policy/AgentDef that exercise the rejection path), augmented with hand-picked
// edge cases. CI runs this seed-only (no fuzz campaign) via `go test`; longer
// campaigns run locally via `make fuzz DURATION=60s`.
//
// The fuzz body asserts only structural invariants — never a specific output,
// since fuzz inputs are by definition unknown (canvas Safeguards):
//   - ParseManifest never panics.
//   - It returns exactly one of: (non-nil manifest, no errors) or
//     (nil manifest, at least one error). A non-nil manifest with errors, or a
//     nil manifest with no errors, is a contract violation.
func FuzzParseManifest(f *testing.F) {
	seedFromCorpus(f)

	// Hand-picked edge cases beyond the example corpus.
	for _, seed := range [][]byte{
		nil,
		[]byte(""),
		[]byte("\x00"),
		[]byte("kind: Workflow"),
		[]byte("not: yaml: : :"),
		[]byte("- - - - -"),
		minimalValid(),
	} {
		f.Add(seed)
	}

	ctx := context.Background()
	f.Fuzz(func(t *testing.T, data []byte) {
		manifest, errs := ParseManifest(ctx, data)
		switch {
		case manifest == nil && len(errs) == 0:
			t.Fatalf("ParseManifest returned nil manifest with no errors for input %q", data)
		case manifest != nil && len(errs) != 0:
			t.Fatalf("ParseManifest returned a manifest AND %d errors for input %q", len(errs), data)
		}
	})
}

// seedFromCorpus adds every embedded example YAML as a fuzz seed.
func seedFromCorpus(f *testing.F) {
	f.Helper()
	entries, err := fs.ReadDir(fuzzCorpus, "testdata/fuzz_corpus")
	if err != nil {
		f.Fatalf("read embedded fuzz corpus: %v", err)
	}
	if len(entries) == 0 {
		f.Fatal("fuzz corpus is empty: expected seeds from spec/workflows/examples")
	}
	for _, e := range entries {
		data, readErr := fuzzCorpus.ReadFile("testdata/fuzz_corpus/" + e.Name())
		if readErr != nil {
			f.Fatalf("read seed %s: %v", e.Name(), readErr)
		}
		f.Add(data)
	}
}
