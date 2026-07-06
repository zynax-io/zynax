// SPDX-License-Identifier: Apache-2.0

// Package zynaxevents is the shared NATS JetStream events client (ADR-046,
// M8.H). It carries the platform eventing conventions VERBATIM from the
// retired EventBusService facade (services/event-bus): depth-4 stream
// derivation with the #1149 disjoint-filter rule, DLQ_<src> /
// zynax.dlq.<prefix>.dead / WorkQueuePolicy dead-lettering, the
// MaxDeliver=5-aligned retry backoff, durable-name sanitizing, glob
// subscription matching, the CloudEvents v1.0 JSON envelope with ce-*
// headers, and zynaxobs trace inject/extract.
//
// The wire shape is pinned by the golden fixtures in testdata/golden/ — both
// this library and the (deprecated, M9-removed) facade must stay golden. The
// AsyncAPI spec (spec/asyncapi/zynax-events.yaml) remains the single contract
// of record; this library realises it, it does not define it.
package zynaxevents

import (
	"errors"
	"strings"
	"time"
)

// CloudEvent is the client representation of a CNCF CloudEvents v1.0 envelope.
// Field names mirror the CloudEvents specification attribute names exactly.
// (Moved verbatim from the event-bus domain, ADR-046.)
type CloudEvent struct {
	// ID uniquely identifies the event. Must be non-empty on publication.
	ID string
	// Source is a URI reference identifying the event producer.
	Source string
	// SpecVersion is always "1.0" per the CloudEvents specification.
	SpecVersion string
	// Type identifies the event topic (e.g. "zynax.v1.engine-adapter.workflow.completed").
	Type string
	// DataContentType indicates the media type of Data (e.g. "application/json").
	DataContentType string
	// Time is when the event occurred. It is NOT part of the wire envelope.
	Time time.Time
	// Data is the opaque event payload.
	Data []byte
	// WorkflowID is an optional Zynax-specific extension attribute scoping the event.
	WorkflowID string
	// RunID is an optional Zynax-specific run identifier extension attribute.
	RunID string
	// Namespace is an optional Zynax-specific tenant namespace extension attribute.
	Namespace string
	// CapabilityName is an optional Zynax-specific capability that produced the event.
	CapabilityName string
}

// SubscribeRequest carries the parameters for opening a subscription stream.
type SubscribeRequest struct {
	// SubscriberID is a unique identifier for this subscriber. Must be non-empty.
	// It becomes the durable JetStream consumer name after sanitizing.
	SubscriberID string
	// TypePattern is a glob expression matching event types to receive.
	// "*" matches a single segment; "**" matches zero or more segments.
	TypePattern string
	// WorkflowID is an optional scope filter; empty means "all workflows".
	// A workflow-scoped subscription closes after the run's terminal event.
	WorkflowID string
}

// ErrSubscriberNotFound is returned when Unsubscribe targets an unknown subscriber ID.
var ErrSubscriberNotFound = errors.New("subscriber not found")

// ErrReservedPrefix is returned when publishing under a reserved event-type namespace.
var ErrReservedPrefix = errors.New("reserved event type prefix")

const reservedDLQPrefix = "zynax.dlq."

// terminalWorkflowVerbs are the workflow lifecycle event-type verbs that mark a
// run as finished. They mirror the engine-adapter lifecycle events
// ("zynax.workflow.completed"/"failed", see engine-adapter interpreter) plus the
// other Temporal terminal outcomes a run may end in. A workflow-scoped
// subscriber receives the terminal event and then the stream closes.
var terminalWorkflowVerbs = []string{
	"workflow.completed",
	"workflow.failed",
	"workflow.terminated",
	"workflow.canceled",
	"workflow.timed_out",
}

// IsTerminalEventType reports whether eventType denotes a terminal workflow
// lifecycle event — i.e. a run reaching a finished state. Matching is by
// dot-segment suffix so it is independent of the taxonomy prefix
// ("zynax.v1.engine-adapter.workflow.completed" and a bare
// "zynax.workflow.completed" both match). It backs the "stream closes on
// terminal state" guarantee for workflow-scoped subscriptions.
func IsTerminalEventType(eventType string) bool {
	for _, verb := range terminalWorkflowVerbs {
		if eventType == verb || strings.HasSuffix(eventType, "."+verb) {
			return true
		}
	}
	return false
}
