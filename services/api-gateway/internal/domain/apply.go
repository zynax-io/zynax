// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
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
	Warnings []string
	Errors   []CompileError
	// Status is "new" when a fresh workflow was started, "existing" when a
	// running workflow with the same manifest hash was found. Empty for dry runs.
	Status string
}

// ApplyService orchestrates manifest apply operations.
type ApplyService struct {
	compiler CompilerPort
	engine   EnginePort
	eventbus EventBusPort // optional; nil falls back to engine-history-only logs
}

// NewApplyService constructs an ApplyService with the given ports. eventbus may
// be nil, in which case WatchWorkflowLogs streams engine history only (no
// capability events are merged).
func NewApplyService(compiler CompilerPort, engine EnginePort, eventbus EventBusPort) *ApplyService {
	return &ApplyService{compiler: compiler, engine: engine, eventbus: eventbus}
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
//
// The compiled namespace is forwarded to EnginePort.SubmitWorkflow so that
// the engine can enforce namespace-scoped capability routing.
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

	runID, err := s.engine.SubmitWorkflow(ctx, compiled.IRBytes, engineHint, workflowID, compiled.Namespace)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("api-gateway: %w", err)
	}
	return ApplyResult{RunID: runID, Warnings: compiled.Warnings, Status: "new"}, nil
}

// ApplyAgentDef is retired (ADR-039): the Agent custom resource is the single
// source of truth for agent identity, so the gateway no longer forwards
// AgentDef manifests as push registrations — the push client was deleted in
// M9.A step 1 (#1697). The route answers unconditionally with a migration
// pointer to the Agent CRD; the request body is never parsed or forwarded.
func (s *ApplyService) ApplyAgentDef(_ context.Context, _ ApplyRequest) (ApplyResult, error) {
	return ApplyResult{}, ErrAgentDefRetired
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

// PublishEvent injects a business/lifecycle event into the run identified by
// ev.RunID so an event-driven workflow can advance (e.g. review.approved →
// merge). It validates that the run id and event type are non-empty
// (ErrInvalidEvent otherwise) and returns the bus-assigned event id. Returns
// ErrEngineUnavailable when no event-bus port is configured.
func (s *ApplyService) PublishEvent(ctx context.Context, ev EventPublish) (string, error) {
	if strings.TrimSpace(ev.RunID) == "" {
		return "", fmt.Errorf("%w: run id is required", ErrInvalidEvent)
	}
	if strings.TrimSpace(ev.Type) == "" {
		return "", fmt.Errorf("%w: event type is required", ErrInvalidEvent)
	}
	if s.eventbus == nil {
		return "", fmt.Errorf("api-gateway: %w", ErrEngineUnavailable)
	}
	eventID, err := s.eventbus.PublishEvent(ctx, ev)
	if err != nil {
		return "", fmt.Errorf("api-gateway: %w", err)
	}
	return eventID, nil
}

// WatchWorkflowLogs streams a run's logs by merging two sources into a single
// chronological stream (EPIC L step 3 / #1182):
//
//   - the engine's state-transition history (EnginePort.WatchWorkflow), and
//   - workflow-scoped capability CloudEvents (EventBusPort.SubscribeWorkflowEvents).
//
// Both upstreams run concurrently; their events are serialised through a single
// mutex-guarded send so the HTTP handler never sees interleaved writes. The
// engine history stream is authoritative for stream lifetime: once it ends
// (terminal workflow state) the event subscription is cancelled and the call
// returns. Returns when the engine stream closes, ctx is cancelled, or send
// returns an error.
//
// When no event-bus port is configured (eventbus == nil) the method streams the
// engine history only, preserving the prior behaviour.
func (s *ApplyService) WatchWorkflowLogs(ctx context.Context, runID string, send func(WatchEvent) error) error {
	if s.eventbus == nil {
		if err := s.engine.WatchWorkflow(ctx, runID, send); err != nil {
			return fmt.Errorf("api-gateway: %w", err)
		}
		return nil
	}

	// Resolve the workflow_id used to scope the event subscription. A failure
	// here (e.g. the run is unknown) is surfaced via the engine watch below, so
	// fall back to engine-history-only rather than dropping the whole stream.
	workflowID := runID
	if run, err := s.engine.GetWorkflowStatus(ctx, runID); err == nil && run.WorkflowID != "" {
		workflowID = run.WorkflowID
	}

	subCtx, cancelSub := context.WithCancel(ctx)
	defer cancelSub()

	var mu sync.Mutex
	safeSend := func(ev WatchEvent) error {
		mu.Lock()
		defer mu.Unlock()
		return send(ev)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Best-effort: a capability-event subscription error must not abort the
		// authoritative engine history stream. Cancellation on engine-stream
		// completion surfaces here as a context error and is ignored.
		_ = s.eventbus.SubscribeWorkflowEvents(subCtx, workflowID, safeSend)
	}()

	engErr := s.engine.WatchWorkflow(ctx, runID, safeSend)
	cancelSub() // engine history ended (terminal) — stop the event subscription
	wg.Wait()
	if engErr != nil {
		return fmt.Errorf("api-gateway: %w", engErr)
	}
	return nil
}
