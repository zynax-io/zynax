// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides adapters for external dependencies used by
// the event-bus service: NATS JetStream client and TLS credential helper.
package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	nats "github.com/nats-io/nats.go"

	"github.com/zynax-io/zynax/services/event-bus/internal/domain"
)

// NATSEventBus implements domain.EventBus backed by NATS JetStream.
type NATSEventBus struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

// NewNATSEventBus connects to NATS at url, creates a JetStream context, and
// returns a ready-to-use NATSEventBus. Caller must call Close when done.
func NewNATSEventBus(url string) (*NATSEventBus, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats jetstream context: %w", err)
	}
	return &NATSEventBus{conn: nc, js: js}, nil
}

// Close drains and closes the underlying NATS connection.
func (b *NATSEventBus) Close() {
	_ = b.conn.Drain()
}

// cloudEventEnvelope is the JSON wire format for a CloudEvent published to JetStream.
// Field names follow the CloudEvents v1.0 JSON format specification.
type cloudEventEnvelope struct {
	SpecVersion     string `json:"specversion"`
	ID              string `json:"id"`
	Source          string `json:"source"`
	Type            string `json:"type"`
	DataContentType string `json:"datacontenttype,omitempty"`
	WorkflowID      string `json:"workflowid,omitempty"`
	RunID           string `json:"runid,omitempty"`
	Namespace       string `json:"namespace,omitempty"`
	CapabilityName  string `json:"capabilityname,omitempty"`
	Data            []byte `json:"data,omitempty"`
}

// StreamName derives a JetStream stream name from a dot-separated event type.
// The stream covers the full event type as subject filter using ">" wildcard.
// Examples:
//
//	"zynax.v1.engine-adapter.workflow.completed" → "ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW"
//	"zynax.v1.agent-registry.agent.registered"  → "ZYNAX_V1_AGENT_REGISTRY_AGENT"
//
// Dashes are replaced with underscores; dots become underscores; all uppercase.
// The last segment (the verb) is dropped so all events of the same entity
// share a single stream.
func StreamName(eventType string) string {
	parts := strings.Split(eventType, ".")
	// Use all but the last segment (verb) for the stream name.
	prefix := parts
	if len(parts) > 1 {
		prefix = parts[:len(parts)-1]
	}
	name := strings.Join(prefix, "_")
	name = strings.ReplaceAll(name, "-", "_")
	return strings.ToUpper(name)
}

// SubjectFilter returns the NATS subject filter for a stream created from eventType.
// All events whose type starts with the entity prefix are captured.
func SubjectFilter(eventType string) string {
	parts := strings.Split(eventType, ".")
	// Drop the last segment (verb) and replace with ">" wildcard.
	if len(parts) > 1 {
		prefix := strings.Join(parts[:len(parts)-1], ".")
		return prefix + ".>"
	}
	return eventType + ".>"
}

// ensureStream creates the JetStream stream if it does not already exist.
// If the stream already exists with the same config, this is a no-op (idempotent).
func (b *NATSEventBus) ensureStream(eventType string) error {
	name := StreamName(eventType)
	filter := SubjectFilter(eventType)

	cfg := &nats.StreamConfig{
		Name:      name,
		Subjects:  []string{filter},
		Retention: nats.LimitsPolicy,
		Storage:   nats.FileStorage,
		Replicas:  1,
	}

	_, err := b.js.AddStream(cfg)
	if err != nil {
		// nats.ErrStreamNameAlreadyInUse is returned when the stream already exists.
		if errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
			return nil
		}
		return fmt.Errorf("jetstream add stream %s: %w", name, err)
	}
	return nil
}

// Publish submits a CloudEvent to the JetStream stream for the event type.
// It idempotently ensures the stream exists before publishing.
// Returns a composite "STREAM:sequence" identifier as the event ID on success.
func (b *NATSEventBus) Publish(ctx context.Context, event domain.CloudEvent) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context: %w", err)
	}

	if err := b.ensureStream(event.Type); err != nil {
		return "", fmt.Errorf("nats publish: ensure stream: %w", err)
	}

	env := cloudEventEnvelope{
		SpecVersion:     event.SpecVersion,
		ID:              event.ID,
		Source:          event.Source,
		Type:            event.Type,
		DataContentType: event.DataContentType,
		WorkflowID:      event.WorkflowID,
		RunID:           event.RunID,
		Namespace:       event.Namespace,
		CapabilityName:  event.CapabilityName,
		Data:            event.Data,
	}

	payload, err := json.Marshal(env)
	if err != nil {
		return "", fmt.Errorf("nats publish: marshal event: %w", err)
	}

	msg := &nats.Msg{
		Subject: event.Type,
		Data:    payload,
		Header:  nats.Header{},
	}
	msg.Header.Set("Content-Type", "application/cloudevents+json")
	msg.Header.Set("ce-id", event.ID)
	msg.Header.Set("ce-type", event.Type)
	msg.Header.Set("ce-source", event.Source)

	pubAck, err := b.js.PublishMsg(msg)
	if err != nil {
		return "", fmt.Errorf("nats publish: publish msg: %w", err)
	}

	return fmt.Sprintf("%s:%d", pubAck.Stream, pubAck.Sequence), nil
}

// Subscribe is a stub — full implementation in O3 (#825).
func (b *NATSEventBus) Subscribe(_ context.Context, _ domain.SubscribeRequest) (<-chan domain.CloudEvent, error) {
	return nil, errors.New("not implemented")
}

// Unsubscribe is a stub — full implementation in O4 (#826).
func (b *NATSEventBus) Unsubscribe(_ context.Context, _ string) error {
	return errors.New("not implemented")
}
