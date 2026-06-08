// SPDX-License-Identifier: Apache-2.0

// Package domain defines the core types and interfaces for the event-bus service.
// This package has zero imports from api or infrastructure — it is the innermost layer.
package domain

import "time"

// CloudEvent is the domain representation of a CNCF CloudEvents v1.0 envelope.
// Field names mirror the CloudEvents specification attribute names exactly.
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
	// Time is when the event occurred.
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

// Topic is a dot-separated event type prefix used for routing.
// Example: "zynax.v1.engine-adapter.workflow"
type Topic string

// ConsumerGroup is a named durable subscriber group in JetStream.
// A ConsumerGroup receives exactly one copy of each matching event (at-least-once delivery).
type ConsumerGroup string

// SubscribeRequest carries the parameters for opening a subscription stream.
type SubscribeRequest struct {
	// SubscriberID is a unique identifier for this subscriber. Must be non-empty.
	SubscriberID string
	// TypePattern is a glob expression matching event types to receive.
	// "*" matches a single segment; "**" matches zero or more segments.
	TypePattern string
	// WorkflowID is an optional scope filter; empty means "all workflows".
	WorkflowID string
}
