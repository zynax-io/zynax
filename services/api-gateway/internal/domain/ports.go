// SPDX-License-Identifier: Apache-2.0

// Package domain contains the api-gateway's core value objects and port
// interfaces. Nothing in this package may import infrastructure packages or
// gRPC SDK types — all I/O crosses the boundary via the interfaces below.
package domain

import "context"

// CompileError is a single diagnostic returned by the workflow compiler.
type CompileError struct {
	Code    string
	Message string
	Line    int32
}

// CompileResult carries the outcome of a WorkflowCompilerService call.
// IRBytes is an opaque serialised WorkflowIR proto — the domain treats it
// as an uninterpreted byte slice. Namespace mirrors the namespace embedded
// in the compiled WorkflowIR so that the engine port can propagate it
// without re-deserialising the IR bytes.
type CompileResult struct {
	IRBytes   []byte
	Namespace string
	Warnings  []string
	Errors    []CompileError
}

// WorkflowRunSummary is the domain view of a submitted workflow execution.
type WorkflowRunSummary struct {
	RunID        string
	WorkflowID   string
	Status       string
	CurrentState string
}

// WatchEvent is a single lifecycle event emitted by a streaming WatchWorkflow call.
type WatchEvent struct {
	RunID     string
	EventType string
	FromState string
	ToState   string
	Status    string
	Timestamp string // RFC3339; empty when the engine omits it
	Payload   string // JSON string or empty
}

// CompilerPort is the gateway's outbound dependency on WorkflowCompilerService.
type CompilerPort interface {
	CompileWorkflow(ctx context.Context, manifestYAML []byte, namespace string, dryRun bool) (CompileResult, error)
}

// EnginePort is the gateway's outbound dependency on EngineAdapterService.
type EnginePort interface {
	// SubmitWorkflow submits irBytes to the engine under the given workflowID
	// within namespace. namespace is propagated to SubmitWorkflowRequest.namespace
	// so the engine can enforce namespace-scoped capability routing at execution
	// time. The engine uses workflowID as the Temporal workflow identifier so that
	// subsequent DescribeWorkflowExecution lookups find the same execution.
	SubmitWorkflow(ctx context.Context, irBytes []byte, engineHint, workflowID, namespace string) (string, error)
	GetWorkflowStatus(ctx context.Context, runID string) (WorkflowRunSummary, error)
	CancelWorkflow(ctx context.Context, runID string) error
	// WatchWorkflow streams lifecycle events for runID, calling send for each.
	// Returns when the stream closes, ctx is cancelled, or send returns an error.
	WatchWorkflow(ctx context.Context, runID string, send func(WatchEvent) error) error
}

// AgentRegistration is the domain view of a successful RegisterAgent response.
type AgentRegistration struct {
	AgentID string
}

// RegistryPort is the gateway's outbound dependency on AgentRegistryService.
type RegistryPort interface {
	RegisterAgent(ctx context.Context, manifestYAML []byte, namespace string) (AgentRegistration, error)
}
