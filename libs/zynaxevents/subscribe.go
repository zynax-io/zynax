// SPDX-License-Identifier: Apache-2.0

package zynaxevents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	nats "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/zynax-io/zynax/libs/zynaxobs"
)

// tracerName is the OTel instrumentation scope for spans this package creates
// (the NATS consumer-side delivery span). Renamed with the ADR-046 move from
// the event-bus facade — the trace scope is process metadata, not wire bytes.
const tracerName = "github.com/zynax-io/zynax/libs/zynaxevents"

// openSubscription creates a durable JetStream subscription for the given stream.
// It first attempts to bind to an existing durable consumer, then falls back to
// creating a new one with DeliverLast, AckExplicit, MaxDeliver=5, retry backoff,
// and DLQ subject routing for exhausted deliveries.
func (c *Client) openSubscription(streamName, subject, durName, dlqDeliverSubj string) (*nats.Subscription, error) {
	sub, err := c.js.SubscribeSync(
		subject,
		nats.Durable(durName),
		nats.Bind(streamName, durName),
		nats.DeliverLast(),
		nats.AckExplicit(),
		nats.MaxDeliver(5),
		nats.BackOff(RetryBackoff),
		nats.DeliverSubject(dlqDeliverSubj),
	)
	if err == nil {
		return sub, nil
	}
	// Consumer not yet registered — create it without Bind.
	sub, err = c.js.SubscribeSync(
		subject,
		nats.Durable(durName),
		nats.DeliverLast(),
		nats.AckExplicit(),
		nats.MaxDeliver(5),
		nats.BackOff(RetryBackoff),
	)
	if err != nil {
		return nil, fmt.Errorf("open subscription %s: %w", durName, err)
	}
	return sub, nil
}

// dispatchMsg decodes a NATS message into a CloudEvent, applies the glob
// pattern and workflow_id filters, then sends to ch. Returns true if the
// goroutine should stop (context cancelled during send).
func dispatchMsg(ctx context.Context, msg *nats.Msg, req SubscribeRequest, ch chan<- CloudEvent) bool {
	// Stitch this delivery to the publisher's trace across the async NATS hop by
	// extracting the W3C traceparent carried in the message headers.
	// nats.Header is a map[string][]string, accepted directly by the carrier.
	// No-op when the message carries no traceparent.
	ctx = zynaxobs.ExtractMapHeader(ctx, msg.Header)
	ctx, span := otel.Tracer(tracerName).Start(
		ctx, "EventBus.deliver", trace.WithSpanKind(trace.SpanKindConsumer),
	)
	defer span.End()

	var env cloudEventEnvelope
	if err := json.Unmarshal(msg.Data, &env); err != nil {
		_ = msg.Nak()
		return false
	}

	event := CloudEvent{
		ID:              env.ID,
		Source:          env.Source,
		SpecVersion:     env.SpecVersion,
		Type:            env.Type,
		DataContentType: env.DataContentType,
		WorkflowID:      env.WorkflowID,
		RunID:           env.RunID,
		Namespace:       env.Namespace,
		CapabilityName:  env.CapabilityName,
		Data:            env.Data,
	}

	if !MatchesGlob(req.TypePattern, event.Type) {
		_ = msg.Ack()
		return false
	}
	if req.WorkflowID != "" && event.WorkflowID != req.WorkflowID {
		_ = msg.Ack()
		return false
	}

	select {
	case <-ctx.Done():
		_ = msg.Nak()
		return true
	case ch <- event:
		_ = msg.Ack()
	}

	// A workflow-scoped subscription is a per-run follower: once that run
	// reaches a terminal lifecycle event no further events are coming, so
	// deliver the terminal event (above) and then close the stream. Wildcard
	// subscriptions (no WorkflowID scope) span many runs and must not close on
	// one run's terminal event.
	if req.WorkflowID != "" && IsTerminalEventType(event.Type) {
		return true
	}
	return false
}

// Subscribe creates a durable JetStream push consumer for the subscriber and
// returns a channel that delivers matching CloudEvents until ctx is cancelled.
// Consumer config: DeliverLastPolicy, AckExplicit, MaxDeliver=5, retry backoff.
// Glob pattern matching and workflow_id filtering are applied in this layer.
// A DLQ stream ("zynax.dlq.<topic>") is created idempotently to capture events
// that exhaust all delivery retries.
func (c *Client) Subscribe(ctx context.Context, req SubscribeRequest) (<-chan CloudEvent, error) {
	if ctx.Err() != nil {
		return nil, fmt.Errorf("context: %w", ctx.Err())
	}

	streamSubject := StreamSubjectFromPattern(req.TypePattern)
	streamName := StreamName(streamSubject)

	if err := c.ensureStream(streamSubject); err != nil {
		return nil, fmt.Errorf("subscribe: ensure stream: %w", err)
	}

	// Ensure DLQ stream exists before wiring the consumer's DeliverSubject.
	if err := c.ensureDLQStream(streamName, streamSubject); err != nil {
		return nil, fmt.Errorf("subscribe: ensure dlq stream: %w", err)
	}

	// Build the DLQ deliver subject for exhausted messages.
	dlqSubj := dlqDeliverSubject(streamSubject)

	sub, err := c.openSubscription(
		streamName,
		SubjectFilter(streamSubject),
		DurableConsumerName(req.SubscriberID),
		dlqSubj,
	)
	if err != nil {
		return nil, fmt.Errorf("subscribe: jetstream subscribe: %w", err)
	}

	ch := make(chan CloudEvent, 64)

	go func() {
		defer close(ch)
		defer func() { _ = sub.Unsubscribe() }()

		for ctx.Err() == nil {
			msg, msgErr := sub.NextMsg(100 * time.Millisecond)
			if msgErr != nil {
				if errors.Is(msgErr, nats.ErrConnectionClosed) || errors.Is(msgErr, nats.ErrBadSubscription) {
					return
				}
				continue // ErrTimeout or transient — retry
			}
			if dispatchMsg(ctx, msg, req, ch) {
				return
			}
		}
	}()

	return ch, nil
}

// Unsubscribe deletes the durable JetStream consumer for subscriberID across
// all streams. It is a stateless operation: it iterates all known streams and
// removes the consumer from whichever stream owns it.
// Returns ErrSubscriberNotFound if no consumer was found on any stream.
func (c *Client) Unsubscribe(ctx context.Context, subscriberID string) error {
	if ctx.Err() != nil {
		return fmt.Errorf("context: %w", ctx.Err())
	}

	durName := DurableConsumerName(subscriberID)

	// Iterate all streams and attempt to delete the consumer.
	namesCh := c.js.StreamNames()
	found := false
	for name := range namesCh {
		if ctx.Err() != nil {
			return fmt.Errorf("context: %w", ctx.Err())
		}
		// Skip DLQ streams — consumers are managed by the DLQ machinery.
		if strings.HasPrefix(name, "DLQ_") {
			continue
		}
		err := c.js.DeleteConsumer(name, durName)
		if err == nil {
			found = true
			break
		}
		if errors.Is(err, nats.ErrConsumerNotFound) {
			continue
		}
		return fmt.Errorf("unsubscribe: delete consumer %s from stream %s: %w", durName, name, err)
	}

	if !found {
		return ErrSubscriberNotFound
	}
	return nil
}
