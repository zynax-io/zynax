// SPDX-License-Identifier: Apache-2.0

package domain

import "context"

// EventBus is the primary domain port for async event routing.
// Implementations: NATSEventBus (production), in-memory fake (unit tests).
type EventBus interface {
	// Publish submits a CloudEvent for delivery to all matching subscribers.
	// Returns the service-assigned event ID on success.
	Publish(ctx context.Context, event CloudEvent) (eventID string, err error)

	// Subscribe opens a channel that delivers CloudEvents matching req.TypePattern
	// and optional req.WorkflowID scope. The channel is closed when Unsubscribe
	// is called for req.SubscriberID or when ctx is cancelled.
	Subscribe(ctx context.Context, req SubscribeRequest) (<-chan CloudEvent, error)

	// Unsubscribe closes the active subscription for subscriberID.
	// Returns ErrSubscriberNotFound if no active subscription exists.
	Unsubscribe(ctx context.Context, subscriberID string) error
}
