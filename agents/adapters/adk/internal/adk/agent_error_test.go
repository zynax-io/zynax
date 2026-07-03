// SPDX-License-Identifier: Apache-2.0

// Error-branch test for the ADK construction seam. Note: llmagent.New and
// runner.New accept empty names and nil models in the current ADK release, so
// the only reachable constructor error is a missing session service.
package adk

import (
	"strings"
	"testing"
)

func TestNewRunner_NilSessionService(t *testing.T) {
	_, err := NewRunner("adk-adapter", AgentSpec{Name: "triage", Instruction: "x"}, nopLLM{}, nil)
	if err == nil {
		t.Fatal("expected an error for a nil session service")
	}
	if !strings.Contains(err.Error(), "triage") {
		t.Errorf("error %q should name the failing spec", err)
	}
}
