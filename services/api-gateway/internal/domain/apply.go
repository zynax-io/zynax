// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"errors"
	"fmt"
)

// Sentinel errors surfaced to the HTTP handler for status-code mapping.
var (
	ErrCompilationFailed  = errors.New("api-gateway: compilation failed")
	ErrEngineUnavailable  = errors.New("api-gateway: engine unavailable")
	ErrNotFound           = errors.New("api-gateway: not found")
	ErrAgentAlreadyExists = errors.New("api-gateway: agent already registered")
)

// ApplyRequest carries the parameters for a manifest apply operation.
type ApplyRequest struct {
	ManifestYAML []byte
	Namespace    string
	DryRun       bool
	EngineHint   string
}

// ApplyResult carries the outcome of an apply operation.
type ApplyResult struct {
	RunID    string
	AgentID  string
	Warnings []string
	Errors   []CompileError
}

// ApplyService orchestrates manifest apply operations.
type ApplyService struct {
	compiler CompilerPort
	engine   EnginePort
	registry RegistryPort
}

// NewApplyService constructs an ApplyService with the given ports.
func NewApplyService(compiler CompilerPort, engine EnginePort, registry RegistryPort) *ApplyService {
	return &ApplyService{compiler: compiler, engine: engine, registry: registry}
}

// ApplyWorkflow compiles a Workflow manifest and, unless dry_run, submits it
// to the engine adapter. Returns ErrCompilationFailed (with Errors populated)
// when the manifest has structural errors.
func (s *ApplyService) ApplyWorkflow(ctx context.Context, req ApplyRequest) (ApplyResult, error) {
	compiled, err := s.compiler.CompileWorkflow(ctx, req.ManifestYAML, req.Namespace, req.DryRun)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("api-gateway: %w", err)
	}
	if len(compiled.Errors) > 0 {
		return ApplyResult{Errors: compiled.Errors, Warnings: compiled.Warnings}, ErrCompilationFailed
	}
	if req.DryRun {
		return ApplyResult{Warnings: compiled.Warnings}, nil
	}
	return s.submit(ctx, compiled, req.EngineHint)
}

func (s *ApplyService) submit(ctx context.Context, compiled CompileResult, engineHint string) (ApplyResult, error) {
	runID, err := s.engine.SubmitWorkflow(ctx, compiled.IRBytes, engineHint)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("api-gateway: %w", err)
	}
	return ApplyResult{RunID: runID, Warnings: compiled.Warnings}, nil
}

// ApplyAgentDef registers an AgentDef manifest with the agent registry.
// Returns ErrAgentAlreadyExists when the registry reports ALREADY_EXISTS.
func (s *ApplyService) ApplyAgentDef(ctx context.Context, req ApplyRequest) (ApplyResult, error) {
	reg, err := s.registry.RegisterAgent(ctx, req.ManifestYAML, req.Namespace)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("api-gateway: %w", err)
	}
	return ApplyResult{AgentID: reg.AgentID}, nil
}

// GetWorkflowStatus returns the current status of a workflow run.
func (s *ApplyService) GetWorkflowStatus(ctx context.Context, runID string) (WorkflowRunSummary, error) {
	run, err := s.engine.GetWorkflowStatus(ctx, runID)
	if err != nil {
		return WorkflowRunSummary{}, fmt.Errorf("api-gateway: %w", err)
	}
	return run, nil
}
