// SPDX-License-Identifier: Apache-2.0

// Package engine_adapter_bdd_test is a placeholder for service-level BDD steps.
//
// The engine-adapter feature scenarios require a running Temporal server to be
// meaningful (Submit, Signal, Cancel, Watch all delegate to Temporal). Unit
// tests for the Temporal integration are in services/engine-adapter/internal/;
// the contract tests (stub-level) are in protos/tests/engine_adapter_service/.
//
// Full service-level BDD for engine-adapter is deferred until a Temporal dev
// container is available in the CI service-test stack (M6/M7 scope).
package engine_adapter_bdd_test

import "testing"

func TestFeatures(t *testing.T) {
	t.Skip("engine-adapter BDD requires a running Temporal server — deferred to M6/M7 when a Temporal dev-container is wired into service-test CI")
}
