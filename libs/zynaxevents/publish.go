// SPDX-License-Identifier: Apache-2.0

package zynaxevents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	nats "github.com/nats-io/nats.go"

	"github.com/zynax-io/zynax/libs/zynaxobs"
)

// cloudEventEnvelope is the JSON wire format for a CloudEvent published to JetStream.
// Field names follow the CloudEvents v1.0 JSON format specification. The byte
// shape (including field order) is pinned by testdata/golden/cloudevent_envelope.json.
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

// ensureStream creates the JetStream stream if it does not already exist.
// If the stream already exists with the same config, this is a no-op (idempotent).
func (c *Client) ensureStream(eventType string) error {
	name := StreamName(eventType)

	cfg := &nats.StreamConfig{
		Name:      name,
		Subjects:  streamSubjects(eventType),
		Retention: nats.LimitsPolicy,
		Storage:   nats.FileStorage,
		Replicas:  1,
	}

	_, err := c.js.AddStream(cfg)
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
func (c *Client) Publish(ctx context.Context, event CloudEvent) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context: %w", err)
	}

	if err := c.ensureStream(event.Type); err != nil {
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

	// Carry the W3C traceparent so a subscriber's delivery span stitches to the
	// publisher's trace across this async NATS hop. nats.Header is a
	// map[string][]string, accepted directly by the carrier. No-op when no
	// active span / OTLP endpoint is configured.
	zynaxobs.InjectMapHeader(ctx, msg.Header)

	pubAck, err := c.js.PublishMsg(msg)
	if err != nil {
		return "", fmt.Errorf("nats publish: publish msg: %w", err)
	}

	return fmt.Sprintf("%s:%d", pubAck.Stream, pubAck.Sequence), nil
}
