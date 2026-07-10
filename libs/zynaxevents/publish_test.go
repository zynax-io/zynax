// SPDX-License-Identifier: Apache-2.0

package zynaxevents

import (
	"context"
	"errors"
	"testing"
)

func TestPublishRejectsReservedDLQPrefixBeforeStream(t *testing.T) {
	client := &Client{}
	event := CloudEvent{
		ID:          "evt-dlq",
		Source:      "test",
		SpecVersion: "1.0",
		Type:        "zynax.dlq.zynax.v1.task-broker.task.dead",
	}

	gotID, err := client.Publish(context.Background(), event)
	if !errors.Is(err, ErrReservedPrefix) {
		t.Fatalf("Publish() error = %v, want ErrReservedPrefix", err)
	}
	if gotID != "" {
		t.Fatalf("Publish() id = %q, want empty id", gotID)
	}
}
