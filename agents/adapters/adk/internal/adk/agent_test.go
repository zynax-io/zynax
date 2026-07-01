// SPDX-License-Identifier: Apache-2.0

package adk

import (
	"context"
	"iter"
	"testing"

	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
)

// nopLLM is a minimal model.LLM stand-in for construction tests.
type nopLLM struct{}

func (nopLLM) Name() string { return "nop" }
func (nopLLM) GenerateContent(context.Context, *model.LLMRequest, bool) iter.Seq2[*model.LLMResponse, error] {
	return func(func(*model.LLMResponse, error) bool) {}
}

func TestNewAgent(t *testing.T) {
	ag, err := NewAgent(AgentSpec{Name: "triage", Description: "classify", Instruction: "be terse"}, nopLLM{})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	if ag == nil {
		t.Fatal("nil agent")
	}
}

func TestNewRunner(t *testing.T) {
	r, err := NewRunner("adk-adapter", AgentSpec{Name: "triage", Instruction: "be terse"}, nopLLM{}, session.InMemoryService())
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if r == nil {
		t.Fatal("nil runner")
	}
}
