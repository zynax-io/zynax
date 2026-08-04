// SPDX-License-Identifier: Apache-2.0

package zynaxevents

import (
	"errors"
	"fmt"
	"time"

	nats "github.com/nats-io/nats.go"
)

// dlqDuplicateWindow is the JetStream deduplication window on a DLQ stream: two
// publishes carrying the same "Nats-Msg-Id" within it collapse to one message.
// It backs the forwarder's idempotency (see dlqMessageID) and is set to the
// JetStream default explicitly, so already-provisioned DLQ streams keep exactly
// the behaviour they have today.
const dlqDuplicateWindow = 2 * time.Minute

// dlqStreamName returns the JetStream stream name for the dead-letter queue
// associated with a source stream. The DLQ stream name is derived by prefixing
// the source stream name with "DLQ_". The corresponding NATS subject is
// "zynax.dlq.<original-subject-root>".
// Example: source stream "ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW" → "DLQ_ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW"
func dlqStreamName(sourceStreamName string) string {
	return dlqStreamPrefix + sourceStreamName
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
func (c *Client) ensureDLQStream(sourceStreamName, eventType string) error {
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
		Duplicates:   dlqDuplicateWindow,
	}

	_, err := c.js.AddStream(cfg)
	if err != nil {
		if errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
			return nil
		}
		return fmt.Errorf("jetstream add dlq stream %s: %w", dlqName, err)
	}
	return nil
}
