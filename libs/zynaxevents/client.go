// SPDX-License-Identifier: Apache-2.0

package zynaxevents

import (
	"fmt"

	nats "github.com/nats-io/nats.go"
)

// Client is the shared JetStream events client. It carries the platform
// eventing conventions verbatim from the retired EventBusService facade.
type Client struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

// New connects to NATS at url, creates a JetStream context, and returns a
// ready-to-use Client. Callers must call Close when done. natsOpts are passed
// through to nats.Connect — the TLS/cert-manager identity options (ADR-046
// Decision #4) ride here without an API break.
func New(url string, natsOpts ...nats.Option) (*Client, error) {
	nc, err := nats.Connect(url, natsOpts...)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats jetstream context: %w", err)
	}
	return &Client{conn: nc, js: js}, nil
}

// Close drains and closes the underlying NATS connection.
func (c *Client) Close() {
	_ = c.conn.Drain()
}
