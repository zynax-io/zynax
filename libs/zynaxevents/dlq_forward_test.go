// SPDX-License-Identifier: Apache-2.0

package zynaxevents

// Unit gate for the max-deliveries → DLQ mover (#1653). The end-to-end proof
// (a real exhausted message landing on DLQ_<src>) lives in the integration BDD
// suite under tests/; these cases pin the pure decisions the mover makes before
// it touches the broker — advisory decoding, the skip predicates that keep it
// from publishing to a wrong or reserved subject, the deterministic
// deduplication key, and header handling.

import (
	"encoding/json"
	"testing"

	nats "github.com/nats-io/nats.go"
)

// realAdvisory is a VERBATIM nats-server 2.10 max-deliveries advisory captured
// off "$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>" during the #1653 runtime
// smoke. Note the schema type is "…v1.max_deliver" while the subject says
// MAX_DELIVERIES — the mover routes on the subject for exactly this reason.
const realAdvisory = `{"type":"io.nats.jetstream.advisory.v1.max_deliver",` +
	`"id":"U6MOzumNuu5Wz5ckhGihOV","timestamp":"2026-08-04T21:01:07.687256441Z",` +
	`"stream":"ZYNAX_V1_SMOKE_DLQ","consumer":"smoke-fail-1",` +
	`"stream_seq":1,"deliveries":5}`

func TestMaxDeliveriesAdvisory_Decode(t *testing.T) {
	var adv maxDeliveriesAdvisory
	if err := json.Unmarshal([]byte(realAdvisory), &adv); err != nil {
		t.Fatalf("unmarshal advisory: %v", err)
	}
	if adv.Type != "io.nats.jetstream.advisory.v1.max_deliver" {
		t.Errorf("Type = %q", adv.Type)
	}
	if adv.Stream != "ZYNAX_V1_SMOKE_DLQ" {
		t.Errorf("Stream = %q, want ZYNAX_V1_SMOKE_DLQ", adv.Stream)
	}
	if adv.Consumer != "smoke-fail-1" {
		t.Errorf("Consumer = %q, want smoke-fail-1", adv.Consumer)
	}
	if adv.StreamSeq != 1 {
		t.Errorf("StreamSeq = %d, want 1", adv.StreamSeq)
	}
	if adv.Deliveries != 5 {
		t.Errorf("Deliveries = %d, want 5", adv.Deliveries)
	}
}

func TestAdvisoryNotActionable(t *testing.T) {
	cases := []struct {
		name string
		adv  maxDeliveriesAdvisory
		want bool
	}{
		{"actionable", maxDeliveriesAdvisory{Stream: "ZYNAX_V1_A_B", StreamSeq: 7}, true},
		{"no stream", maxDeliveriesAdvisory{StreamSeq: 7}, false},
		{"no sequence", maxDeliveriesAdvisory{Stream: "ZYNAX_V1_A_B"}, false},
		{"dlq is a terminus, never a source", maxDeliveriesAdvisory{Stream: "DLQ_ZYNAX_V1_A_B", StreamSeq: 7}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reason, ok := advisoryNotActionable(tc.adv)
			if ok != tc.want {
				t.Fatalf("advisoryNotActionable(%+v) = %v (%q), want %v", tc.adv, ok, reason, tc.want)
			}
			if !ok && reason == "" {
				t.Error("a skipped advisory must carry a reason")
			}
		})
	}
}

func TestSourceForwardable(t *testing.T) {
	cases := []struct {
		name    string
		stream  string
		subject string
		want    bool
	}{
		{
			"taxonomy subject on its derived stream",
			"ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW", "zynax.v1.engine-adapter.workflow.completed", true,
		},
		{
			"deeper verb, same depth-4 stream (#1149)",
			"ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW", "zynax.v1.engine-adapter.workflow.state.entered", true,
		},
		{
			"already under the reserved dlq prefix",
			"DLQ_ZYNAX_V1_TASK_BROKER_TASK", "zynax.dlq.zynax.v1.task-broker.task.dead", false,
		},
		{
			"stream not derived from the subject taxonomy",
			"SOME_FOREIGN_STREAM", "orders.created", false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reason, ok := sourceForwardable(tc.stream, tc.subject)
			if ok != tc.want {
				t.Fatalf("sourceForwardable(%q, %q) = %v (%q), want %v", tc.stream, tc.subject, ok, reason, tc.want)
			}
			if !ok && reason == "" {
				t.Error("a skipped source must carry a reason")
			}
		})
	}
}

// TestDLQMessageID_MatchesDLQStream pins the deduplication key to the DLQ
// stream name plus the source sequence: the same message always yields the same
// id, so a replayed advisory or a second forwarder cannot duplicate a rescue.
func TestDLQMessageID_MatchesDLQStream(t *testing.T) {
	src := StreamName("zynax.v1.engine-adapter.workflow.completed")
	got := dlqMessageID(src, 42)
	want := "DLQ_ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW:42"
	if got != want {
		t.Errorf("dlqMessageID(%q, 42) = %q, want %q", src, got, want)
	}
	if again := dlqMessageID(src, 42); again != got {
		t.Errorf("dlqMessageID is not deterministic: %q then %q", got, again)
	}
	if other := dlqMessageID(src, 43); other == got {
		t.Error("different sequences must not collide on one id")
	}
}

func TestDLQHeader(t *testing.T) {
	src := nats.Header{}
	src.Set("Content-Type", "application/cloudevents+json")
	src.Set("ce-id", "evt-1")
	src.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	// A JetStream-reserved header set by the original publisher must not be
	// able to displace the forwarder's own deduplication key.
	src.Set("Nats-Msg-Id", "publisher-chosen-id")

	adv := maxDeliveriesAdvisory{
		Stream: "ZYNAX_V1_TASK_BROKER_TASK", Consumer: "broker-1", StreamSeq: 9, Deliveries: 5,
	}
	got := dlqHeader(src, adv, "zynax.v1.task-broker.task.dispatched")

	if v := got.Get("Nats-Msg-Id"); v != "" {
		t.Errorf("reserved Nats- header leaked into the DLQ copy: %q", v)
	}
	want := map[string]string{
		"Content-Type":          "application/cloudevents+json",
		"ce-id":                 "evt-1",
		"traceparent":           "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		HeaderDLQSourceStream:   "ZYNAX_V1_TASK_BROKER_TASK",
		HeaderDLQSourceSubject:  "zynax.v1.task-broker.task.dispatched",
		HeaderDLQSourceSequence: "9",
		HeaderDLQConsumer:       "broker-1",
		HeaderDLQDeliveries:     "5",
	}
	for k, v := range want {
		if got.Get(k) != v {
			t.Errorf("header %q = %q, want %q", k, got.Get(k), v)
		}
	}

	// The copy must be independent: mutating it cannot corrupt the source.
	got.Set("ce-id", "mutated")
	if src.Get("ce-id") != "evt-1" {
		t.Error("dlqHeader aliased the source header values")
	}
}

// TestDLQDeliverSubject_MatchesProvisionedStream is the "publish to the exact
// reserved subject" guard: the subject the forwarder publishes on must be the
// one ensureDLQStream provisions for the same source subject.
func TestDLQDeliverSubject_MatchesProvisionedStream(t *testing.T) {
	subjects := []string{
		"zynax.v1.engine-adapter.workflow.completed",
		"zynax.v1.engine-adapter.workflow.state.entered",
		"zynax.v1.task-broker.task.dispatched",
	}
	for _, s := range subjects {
		if got, want := dlqStreamName(StreamName(s)), "DLQ_"+StreamName(s); got != want {
			t.Errorf("dlqStreamName(%q) = %q, want %q", s, got, want)
		}
		if got := dlqDeliverSubject(s); got != "zynax.dlq."+streamPrefix(s)+".dead" {
			t.Errorf("dlqDeliverSubject(%q) = %q", s, got)
		}
	}
}
