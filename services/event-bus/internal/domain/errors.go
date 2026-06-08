// SPDX-License-Identifier: Apache-2.0

package domain

import "errors"

var (
	// ErrTopicNotFound is returned when a requested topic stream does not exist.
	ErrTopicNotFound = errors.New("topic not found")
	// ErrSubscriberNotFound is returned when Unsubscribe targets an unknown subscriber ID.
	ErrSubscriberNotFound = errors.New("subscriber not found")
	// ErrDeadLetter is returned when an event exhausts all delivery retries and is
	// moved to the dead-letter queue.
	ErrDeadLetter = errors.New("event moved to dead letter queue")
)
