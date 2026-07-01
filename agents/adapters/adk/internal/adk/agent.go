// SPDX-License-Identifier: Apache-2.0

// Package adk builds Google ADK (Go) runtimes for the adapter. It is the thin
// seam between Zynax config and the ADK framework (ADR-038): one llmagent per
// capability, each wrapped in a Runner backed by an in-memory session service.
// The control plane never imports this package — only the adapter binary does.
package adk

import (
	"fmt"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
)

// AgentSpec is the minimal description needed to build one ADK-backed capability.
type AgentSpec struct {
	Name        string
	Description string
	Instruction string
}

// NewAgent constructs an ADK llmagent over the given model.LLM. Tools and
// sub-agents are intentionally empty in S3 (#1479) — the reasoning core is a
// single instruction-driven agent; tool wiring is future scope.
func NewAgent(spec AgentSpec, llm model.LLM) (agent.Agent, error) {
	ag, err := llmagent.New(llmagent.Config{
		Name:        spec.Name,
		Description: spec.Description,
		Model:       llm,
		Instruction: spec.Instruction,
	})
	if err != nil {
		return nil, fmt.Errorf("adk: build llmagent %q: %w", spec.Name, err)
	}
	return ag, nil
}

// NewRunner builds an agent per spec and wraps it in a Runner. appName scopes the
// session namespace; sessions are created lazily at Run time (AutoCreateSession),
// keyed by the sessionID the caller passes (the Zynax workflow_id). The session
// service is shared across capabilities so a workflow's turns accumulate.
func NewRunner(appName string, spec AgentSpec, llm model.LLM, sess session.Service) (*runner.Runner, error) {
	ag, err := NewAgent(spec, llm)
	if err != nil {
		return nil, err
	}
	r, err := runner.New(runner.Config{
		AppName:           appName,
		Agent:             ag,
		SessionService:    sess,
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, fmt.Errorf("adk: build runner %q: %w", spec.Name, err)
	}
	return r, nil
}
