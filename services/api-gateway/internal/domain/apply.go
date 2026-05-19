// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// manifestIDLen is the number of hex characters used as the workflow ID suffix.
// 16 hex chars = 8 bytes = 64 bits of the SHA-256 digest — sufficient for
// workflow-scoped uniqueness within a single Temporal namespace.
const manifestIDLen = 16

// status values from the proto WorkflowStatus enum (via String()).
const (
	statusRunning   = "WORKFLOW_STATUS_RUNNING"
	statusCompleted = "WORKFLOW_STATUS_COMPLETED"
)

// ManifestWorkflowID derives a deterministic workflow identifier from raw
// manifest YAML. The YAML is canonicalised before hashing so that semantically
// equivalent documents (differing only in whitespace or indentation) produce
// the same ID. Format: "wf-" + first manifestIDLen hex chars of SHA-256.
func ManifestWorkflowID(manifestYAML []byte) string {
	sum := sha256.Sum256(canonicaliseYAML(manifestYAML))
	return "wf-" + hex.EncodeToString(sum[:])[:manifestIDLen]
}

// canonicaliseYAML parses and re-marshals YAML to normalise trailing
// whitespace and indentation differences. Falls back to raw bytes on error.
func canonicaliseYAML(raw []byte) []byte {
	var v any
	if err := yaml.Unmarshal(raw, &v); err != nil {
		return raw
	}
	out, err := yaml.Marshal(v)
	if err != nil {
		return raw
	}
	return out
}

// rerunWorkflowID appends a Unix-second timestamp to baseID so that a re-run
// of a completed workflow gets a unique Temporal workflow ID.
func rerunWorkflowID(baseID string, t time.Time) string {
	return fmt.Sprintf("%s-%d", baseID, t.Unix())
}

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
	// Status is "new" when a fresh workflow was started, "existing" when a
	// running workflow with the same manifest hash was found. Empty for dry
	// runs and agent registrations.
	Status string
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
	return s.submit(ctx, req.ManifestYAML, compiled, req.EngineHint)
}

// submit checks for an existing workflow execution with the hash-derived ID
// and applies idempotency logic before starting a new run:
//   - Running  → return existing run_id with Status "existing"
//   - Completed → start new run with a timestamp-suffixed ID (re-run)
//   - Not found / failed / other terminal → start new run with hash ID
func (s *ApplyService) submit(ctx context.Context, manifestYAML []byte, compiled CompileResult, engineHint string) (ApplyResult, error) {
	workflowID := ManifestWorkflowID(manifestYAML)

	existing, err := s.engine.GetWorkflowStatus(ctx, workflowID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return ApplyResult{}, fmt.Errorf("api-gateway: check existing workflow: %w", err)
	}
	if err == nil {
		switch existing.Status {
		case statusRunning:
			return ApplyResult{RunID: existing.RunID, Warnings: compiled.Warnings, Status: "existing"}, nil
		case statusCompleted:
			workflowID = rerunWorkflowID(workflowID, time.Now())
		}
	}

	runID, err := s.engine.SubmitWorkflow(ctx, compiled.IRBytes, engineHint, workflowID)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("api-gateway: %w", err)
	}
	return ApplyResult{RunID: runID, Warnings: compiled.Warnings, Status: "new"}, nil
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

// CancelWorkflow requests cancellation of a running workflow.
func (s *ApplyService) CancelWorkflow(ctx context.Context, runID string) error {
	if err := s.engine.CancelWorkflow(ctx, runID); err != nil {
		return fmt.Errorf("api-gateway: %w", err)
	}
	return nil
}

// WatchWorkflowLogs streams lifecycle events for runID, calling send for each event.
// Returns when the stream closes, ctx is cancelled, or send returns an error.
func (s *ApplyService) WatchWorkflowLogs(ctx context.Context, runID string, send func(WatchEvent) error) error {
	if err := s.engine.WatchWorkflow(ctx, runID, send); err != nil {
		return fmt.Errorf("api-gateway: %w", err)
	}
	return nil
}
