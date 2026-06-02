// SPDX-License-Identifier: Apache-2.0

// Package memory_service_bdd_test is a placeholder for service-level BDD steps.
//
// The memory-service is not yet implemented — it is a contract-only stub.
// The feature scenarios exercise Redis KV + pgvector semantics (TTL expiry,
// vector similarity search) that require a real storage backend.
// Implementation is M6 scope.
//
// BDD contract tests (proto-level stubs) are in protos/tests/memory_service/.
package memory_service_bdd_test

import "testing"

func TestFeatures(t *testing.T) {
	t.Skip("memory-service is not yet implemented — Redis/pgvector backend is M6 scope")
}
