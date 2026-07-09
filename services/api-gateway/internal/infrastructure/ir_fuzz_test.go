// SPDX-License-Identifier: Apache-2.0

package infrastructure

// Fuzz target for WorkflowIR proto unmarshalling (#1417 — 2026-06-18
// architecture review T2.1/R3). SubmitWorkflow (clients.go) unmarshals
// compiler-produced IR bytes that cross a service boundary, so arbitrary or
// corrupted input must never panic. Mirrors the FuzzParseManifest house style
// (workflow-compiler, #1210): CI runs the seed corpus only via `go test`;
// longer campaigns run locally via `make fuzz` (FUZZ_SERVICES includes
// api-gateway).
//
// Invariants asserted for every input:
//   - proto.Unmarshal never panics on arbitrary bytes.
//   - Deterministic: two unmarshals of the same bytes agree on error-ness and
//     produce proto.Equal messages.
//   - Round-trip: a successfully unmarshalled message re-marshals, and the
//     re-unmarshalled copy is proto.Equal to the original.

import (
	"testing"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/protobuf/proto"
)

func FuzzUnmarshalWorkflowIR(f *testing.F) {
	valid, err := proto.Marshal(&zynaxv1.WorkflowIR{WorkflowId: "wf-fuzz-seed"})
	if err != nil {
		f.Fatalf("marshalling the seed IR: %v", err)
	}
	f.Add(valid)
	f.Add([]byte{})
	f.Add([]byte{0xff, 0xff, 0xff})
	f.Add(valid[:len(valid)/2])                     // truncated message
	f.Add(append(append([]byte{}, valid...), 0x08)) // trailing truncated field

	f.Fuzz(func(t *testing.T, data []byte) {
		first := &zynaxv1.WorkflowIR{}
		errFirst := proto.Unmarshal(data, first)

		second := &zynaxv1.WorkflowIR{}
		errSecond := proto.Unmarshal(data, second)
		if (errFirst == nil) != (errSecond == nil) {
			t.Fatalf("non-deterministic unmarshal error: %v vs %v", errFirst, errSecond)
		}
		if errFirst != nil {
			return // rejected input — rejection is a valid outcome, panic is not
		}
		if !proto.Equal(first, second) {
			t.Fatal("non-deterministic unmarshal result for identical bytes")
		}

		remarshalled, err := proto.Marshal(first)
		if err != nil {
			t.Fatalf("re-marshal of a successfully unmarshalled IR failed: %v", err)
		}
		back := &zynaxv1.WorkflowIR{}
		if err := proto.Unmarshal(remarshalled, back); err != nil {
			t.Fatalf("round-trip unmarshal failed: %v", err)
		}
		if !proto.Equal(first, back) {
			t.Fatal("round-trip changed the message")
		}
	})
}
