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
	"time"

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

// streamSubjectDepth is the number of leading subject segments that identify
// the entity owning a JetStream stream, per the platform topic taxonomy
// "zynax.<version>.<service>.<entity>.<event_type>" (root AGENTS.md): the
// entity prefix is always the first four segments, while the event_type verb
// may itself contain dots (e.g. "state.entered").
//
// Deriving every stream at the same fixed depth makes subject filters either
// identical (same stream) or pairwise disjoint by construction. The previous
// "drop the last segment" rule derived overlapping filters for event types of
// different depth within the same prefix tree (e.g. "….workflow.state.entered"
// vs "….workflow.completed"), and NATS rejected the second stream with
// "subjects overlap with an existing stream" (err 10065), silently making
// whole event families undeliverable — see #1149.
const streamSubjectDepth = 4

// streamPrefix returns the leading subject segments (at most
// streamSubjectDepth) that identify the stream owning eventType. Event types
// shorter than the taxonomy depth are used verbatim.
func streamPrefix(eventType string) string {
	parts := strings.Split(eventType, ".")
	if len(parts) > streamSubjectDepth {
		return strings.Join(parts[:streamSubjectDepth], ".")
	}
	return eventType
}

// StreamName derives a JetStream stream name from a dot-separated event type.
// Examples:
//
//	"zynax.v1.engine-adapter.workflow.completed"     → "ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW"
//	"zynax.v1.engine-adapter.workflow.state.entered" → "ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW"
//	"zynax.v1.agent-registry.agent.registered"       → "ZYNAX_V1_AGENT_REGISTRY_AGENT"
//
// Dashes are replaced with underscores; dots become underscores; all uppercase.
// All events under the same entity prefix (first streamSubjectDepth segments)
// share a single stream regardless of how many segments the verb has.
func StreamName(eventType string) string {
	name := strings.ReplaceAll(streamPrefix(eventType), ".", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return strings.ToUpper(name)
}

// SubjectFilter returns the widest NATS subject filter for the stream that
// owns eventType: "<entity-prefix>.>" for taxonomy-depth event types, or the
// literal event type when it has fewer segments than the taxonomy depth
// (literal subjects can never overlap a fixed-depth wildcard filter).
func SubjectFilter(eventType string) string {
	prefix := streamPrefix(eventType)
	if prefix != eventType {
		return prefix + ".>"
	}
	return eventType
}

// streamSubjects returns the full subject set for the stream owning eventType.
// Taxonomy-depth streams carry both the literal entity prefix and its ".>"
// wildcard so a (degenerate) event type that IS the entity prefix still lands
// on the same stream instead of requiring an overlapping second stream.
func streamSubjects(eventType string) []string {
	prefix := streamPrefix(eventType)
	if prefix != eventType {
		return []string{prefix, prefix + ".>"}
	}
	return []string{eventType}
}

// ensureStream creates the JetStream stream if it does not already exist.
// If the stream already exists with the same config, this is a no-op (idempotent).
func (b *NATSEventBus) ensureStream(eventType string) error {
	name := StreamName(eventType)

	cfg := &nats.StreamConfig{
		Name:      name,
		Subjects:  streamSubjects(eventType),
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

// DurableConsumerName converts a subscriber_id into a valid JetStream durable
// consumer name. JetStream consumer names may not contain spaces, dots, or
// special characters; we replace every non-alphanumeric-or-dash character with
// an underscore and truncate at 200 bytes to stay under the NATS limit.
// Exported for testing.
func DurableConsumerName(subscriberID string) string {
	var b strings.Builder
	for _, r := range subscriberID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	name := b.String()
	if len(name) > 200 {
		name = name[:200]
	}
	return name
}

// MatchesGlob reports whether eventType matches a glob pattern where
// "*" matches exactly one dot-separated segment and "**" matches zero or more
// dot-separated segments.
func MatchesGlob(pattern, eventType string) bool {
	return matchGlobSegments(strings.Split(pattern, "."), strings.Split(eventType, "."))
}

func matchGlobSegments(pat, seg []string) bool {
	for len(pat) > 0 {
		p := pat[0]
		if p == "**" {
			// "**" at end matches everything remaining (zero or more segments).
			if len(pat) == 1 {
				return true
			}
			// Try matching the rest of the pattern against every suffix of seg (including empty).
			rest := pat[1:]
			for j := 0; j <= len(seg); j++ {
				if matchGlobSegments(rest, seg[j:]) {
					return true
				}
			}
			return false
		}
		if len(seg) == 0 {
			return false
		}
		if p != "*" && p != seg[0] {
			return false
		}
		pat = pat[1:]
		seg = seg[1:]
	}
	return len(seg) == 0
}

// StreamSubjectFromPattern extracts a concrete subject from a glob pattern so
// we can create/reuse the correct JetStream stream.
// Examples:
//
//	"zynax.v1.engine-adapter.workflow.*" → "zynax.v1.engine-adapter.workflow.x"
//	"zynax.v1.**"                         → "zynax.v1.x"
//	"zynax.v1.workflow.completed"         → "zynax.v1.workflow.completed"
//
// Exported for testing.
func StreamSubjectFromPattern(pattern string) string {
	parts := strings.Split(pattern, ".")
	concrete := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "*" || p == "**" {
			concrete = append(concrete, "x")
			break
		}
		concrete = append(concrete, p)
	}
	return strings.Join(concrete, ".")
}

// RetryBackoff is the ordered list of retry delays applied to NATS JetStream
// consumer redelivery attempts. Five entries align with MaxDeliver=5.
// After the fifth delivery attempt the message is forwarded to the DLQ subject.
// Exported for testing to verify the backoff policy.
var RetryBackoff = []time.Duration{
	1 * time.Second,
	5 * time.Second,
	30 * time.Second,
	2 * time.Minute,
	5 * time.Minute,
}

// dlqStreamName returns the JetStream stream name for the dead-letter queue
// associated with a source stream. The DLQ stream name is derived by prefixing
// the source stream name with "DLQ_". The corresponding NATS subject is
// "zynax.dlq.<original-subject-root>".
// Example: source stream "ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW" → "DLQ_ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW"
func dlqStreamName(sourceStreamName string) string {
	return "DLQ_" + sourceStreamName
}

// dlqDeliverSubject returns the concrete NATS subject that carries messages
// which exhausted their delivery retries for the stream owning eventType.
// It is an exact subject (no wildcard) so DLQ stream filters are pairwise
// disjoint by construction — the same #1149 overlap class applied to the
// "zynax.dlq." namespace. The "zynax.dlq." prefix is reserved: event types
// must not be published under it.
// Example: "zynax.v1.engine-adapter.workflow.completed" →
// "zynax.dlq.zynax.v1.engine-adapter.workflow.dead"
func dlqDeliverSubject(eventType string) string {
	return "zynax.dlq." + streamPrefix(eventType) + ".dead"
}

// ensureDLQStream creates the dead-letter queue JetStream stream for a source
// stream if it does not already exist. The DLQ stream uses WorkQueuePolicy
// (each message consumed once) and captures the exact "zynax.dlq.<prefix>.dead"
// deliver subject for the source stream.
func (b *NATSEventBus) ensureDLQStream(sourceStreamName, eventType string) error {
	dlqName := dlqStreamName(sourceStreamName)
	dlqSubj := dlqDeliverSubject(eventType)

	cfg := &nats.StreamConfig{
		Name:         dlqName,
		Subjects:     []string{dlqSubj},
		Retention:    nats.WorkQueuePolicy,
		Storage:      nats.FileStorage,
		Replicas:     1,
		MaxMsgs:      -1,
		MaxConsumers: 1,
	}

	_, err := b.js.AddStream(cfg)
	if err != nil {
		if errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
			return nil
		}
		return fmt.Errorf("jetstream add dlq stream %s: %w", dlqName, err)
	}
	return nil
}

// openSubscription creates a durable JetStream subscription for the given stream.
// It first attempts to bind to an existing durable consumer, then falls back to
// creating a new one with DeliverLast, AckExplicit, MaxDeliver=5, retry backoff,
// and DLQ subject routing for exhausted deliveries.
func (b *NATSEventBus) openSubscription(streamName, subject, durName, dlqDeliverSubj string) (*nats.Subscription, error) {
	sub, err := b.js.SubscribeSync(
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
	sub, err = b.js.SubscribeSync(
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

// dispatchMsg decodes a NATS message into a domain.CloudEvent, applies the
// glob pattern and workflow_id filters, then sends to ch. Returns true if the
// goroutine should stop (context cancelled during send).
func dispatchMsg(ctx context.Context, msg *nats.Msg, req domain.SubscribeRequest, ch chan<- domain.CloudEvent) bool {
	var env cloudEventEnvelope
	if err := json.Unmarshal(msg.Data, &env); err != nil {
		_ = msg.Nak()
		return false
	}

	event := domain.CloudEvent{
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
	if req.WorkflowID != "" && domain.IsTerminalEventType(event.Type) {
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
func (b *NATSEventBus) Subscribe(ctx context.Context, req domain.SubscribeRequest) (<-chan domain.CloudEvent, error) {
	if ctx.Err() != nil {
		return nil, fmt.Errorf("context: %w", ctx.Err())
	}

	streamSubject := StreamSubjectFromPattern(req.TypePattern)
	streamName := StreamName(streamSubject)

	if err := b.ensureStream(streamSubject); err != nil {
		return nil, fmt.Errorf("subscribe: ensure stream: %w", err)
	}

	// Ensure DLQ stream exists before wiring the consumer's DeliverSubject.
	if err := b.ensureDLQStream(streamName, streamSubject); err != nil {
		return nil, fmt.Errorf("subscribe: ensure dlq stream: %w", err)
	}

	// Build the DLQ deliver subject for exhausted messages.
	dlqSubj := dlqDeliverSubject(streamSubject)

	sub, err := b.openSubscription(
		streamName,
		SubjectFilter(streamSubject),
		DurableConsumerName(req.SubscriberID),
		dlqSubj,
	)
	if err != nil {
		return nil, fmt.Errorf("subscribe: jetstream subscribe: %w", err)
	}

	ch := make(chan domain.CloudEvent, 64)

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
// Returns domain.ErrSubscriberNotFound if no consumer was found on any stream.
func (b *NATSEventBus) Unsubscribe(ctx context.Context, subscriberID string) error {
	if ctx.Err() != nil {
		return fmt.Errorf("context: %w", ctx.Err())
	}

	durName := DurableConsumerName(subscriberID)

	// Iterate all streams and attempt to delete the consumer.
	namesCh := b.js.StreamNames()
	found := false
	for name := range namesCh {
		if ctx.Err() != nil {
			return fmt.Errorf("context: %w", ctx.Err())
		}
		// Skip DLQ streams — consumers are managed by the DLQ machinery.
		if strings.HasPrefix(name, "DLQ_") {
			continue
		}
		err := b.js.DeleteConsumer(name, durName)
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
		return domain.ErrSubscriberNotFound
	}
	return nil
}
