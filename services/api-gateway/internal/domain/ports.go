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
// as an uninterpreted byte slice.
type CompileResult struct {
	IRBytes  []byte
	Warnings []string
	Errors   []CompileError
}

// WorkflowRunSummary is the domain view of a submitted workflow execution.
type WorkflowRunSummary struct {
	RunID        string
	WorkflowID   string
	Status       string
	CurrentState string
}

// CompilerPort is the gateway's outbound dependency on WorkflowCompilerService.
type CompilerPort interface {
	CompileWorkflow(ctx context.Context, manifestYAML []byte, namespace string, dryRun bool) (CompileResult, error)
}

// EnginePort is the gateway's outbound dependency on EngineAdapterService.
type EnginePort interface {
	SubmitWorkflow(ctx context.Context, irBytes []byte, engineHint string) (string, error)
	GetWorkflowStatus(ctx context.Context, runID string) (WorkflowRunSummary, error)
	CancelWorkflow(ctx context.Context, runID string) error
}

// AgentRegistration is the domain view of a successful RegisterAgent response.
type AgentRegistration struct {
	AgentID string
}

// RegistryPort is the gateway's outbound dependency on AgentRegistryService.
type RegistryPort interface {
	RegisterAgent(ctx context.Context, manifestYAML []byte, namespace string) (AgentRegistration, error)
}
