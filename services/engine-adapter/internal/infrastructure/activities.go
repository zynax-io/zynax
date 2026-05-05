// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides concrete implementations of domain ports backed by
// external services and SDKs. Only this package may import engine-specific dependencies
// such as the Temporal Go SDK (ADR-015).
package infrastructure

import (
	"context"
	"log/slog"
)

// PublishLifecycleEventActivity is registered with the Temporal worker and called
// by IRInterpreterWorkflow to emit workflow lifecycle events. Publication is
// best-effort: the workflow suppresses errors from this activity (see
// temporalEventPublisher.Publish). Full EventBus integration is deferred to M4.
func PublishLifecycleEventActivity(_ context.Context, eventType, workflowID, stateID string) error {
	slog.Debug("lifecycle event", "event_type", eventType, "workflow_id", workflowID, "state_id", stateID)
	return nil
}
