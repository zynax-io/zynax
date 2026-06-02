// SPDX-License-Identifier: Apache-2.0

// Package event_bus_bdd_test is a placeholder for service-level BDD steps.
//
// The event-bus service is not yet implemented — it is a contract-only stub.
// The feature scenarios exercise NATS JetStream publish/subscribe semantics
// that require a real broker. Implementation is M6 scope.
//
// BDD contract tests (proto-level stubs) are in protos/tests/event_bus_service/.
package event_bus_bdd_test

import "testing"

func TestFeatures(t *testing.T) {
	t.Skip("event-bus is not yet implemented — NATS JetStream backend is M6 scope")
}
