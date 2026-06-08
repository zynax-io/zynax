// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides adapters for external dependencies used by
// the event-bus service: NATS JetStream client and TLS credential helper.
package infrastructure

import (
	"context"
	"errors"
	"fmt"

	nats "github.com/nats-io/nats.go"

	"github.com/zynax-io/zynax/services/event-bus/internal/domain"
)

// NATSEventBus implements domain.EventBus backed by NATS JetStream.
// O1 scaffold: connect + ping only; Publish/Subscribe/Unsubscribe return
// errors.New("not implemented") — these are wired in O2–O4 (#824–#826).
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

// Publish is a stub — full implementation in O2 (#824).
func (b *NATSEventBus) Publish(_ context.Context, _ domain.CloudEvent) (string, error) {
	return "", errors.New("not implemented")
}

// Subscribe is a stub — full implementation in O3 (#825).
func (b *NATSEventBus) Subscribe(_ context.Context, _ domain.SubscribeRequest) (<-chan domain.CloudEvent, error) {
	return nil, errors.New("not implemented")
}

// Unsubscribe is a stub — full implementation in O4 (#826).
func (b *NATSEventBus) Unsubscribe(_ context.Context, _ string) error {
	return errors.New("not implemented")
}
