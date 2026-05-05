// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides concrete implementations of domain ports backed by
// external services and SDKs. Only this package may import engine-specific dependencies
// such as the Temporal Go SDK (ADR-015).
package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/engine-adapter/internal/domain"
)

// stubTemporalClient implements temporalClient for tests.
type stubTemporalClient struct {
	executeRun  client.WorkflowRun
	executeErr  error
	signalErr   error
	cancelErr   error
	descResps   []*workflowservice.DescribeWorkflowExecutionResponse
	descErr     error
	descCallIdx int
}

func (s *stubTemporalClient) ExecuteWorkflow(
	_ context.Context,
	_ client.StartWorkflowOptions,
	_ interface{},
	_ ...interface{},
) (client.WorkflowRun, error) {
	return s.executeRun, s.executeErr
}

func (s *stubTemporalClient) SignalWorkflow(_ context.Context, _, _, _ string, _ interface{}) error {
	return s.signalErr
}

func (s *stubTemporalClient) CancelWorkflow(_ context.Context, _, _ string) error {
	return s.cancelErr
}

func (s *stubTemporalClient) DescribeWorkflowExecution(
	_ context.Context, _, _ string,
) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	if s.descErr != nil {
		return nil, s.descErr
	}
	if len(s.descResps) == 0 {
		return nil, errors.New("no describe responses configured")
	}
	idx := s.descCallIdx
	if idx >= len(s.descResps) {
		idx = len(s.descResps) - 1
	}
	s.descCallIdx++
	return s.descResps[idx], nil
}

// stubWorkflowRun implements client.WorkflowRun for tests.
type stubWorkflowRun struct{ id string }

func (r *stubWorkflowRun) GetID() string                              { return r.id }
func (r *stubWorkflowRun) GetRunID() string                           { return r.id + "-run" }
func (r *stubWorkflowRun) Get(_ context.Context, _ interface{}) error { return nil }
func (r *stubWorkflowRun) GetWithOptions(_ context.Context, _ interface{}, _ client.WorkflowRunGetOptions) error {
	return nil
}

func descResp(status enumspb.WorkflowExecutionStatus) *workflowservice.DescribeWorkflowExecutionResponse {
	return &workflowservice.DescribeWorkflowExecutionResponse{
		WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{
			Status: status,
		},
	}
}

func newTestEngine(c *stubTemporalClient) *TemporalEngine {
	e := newTemporalEngine(c, "default", "default")
	e.pollInterval = time.Millisecond
	return e
}

func TestTemporalEngine_Submit_Success(t *testing.T) {
	stub := &stubTemporalClient{executeRun: &stubWorkflowRun{id: "wf-1"}}
	engine := newTestEngine(stub)

	ir := &zynaxv1.WorkflowIR{WorkflowId: "wf-1"}
	run, err := engine.Submit(context.Background(), ir, map[string]string{"env": "test"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.RunID != "wf-1" {
		t.Errorf("RunID = %q; want %q", run.RunID, "wf-1")
	}
	if run.Status != domain.WorkflowStatusPending {
		t.Errorf("Status = %v; want Pending", run.Status)
	}
	if run.Labels["env"] != "test" {
		t.Errorf("Labels not forwarded: %v", run.Labels)
	}
}

func TestTemporalEngine_Submit_Error(t *testing.T) {
	stub := &stubTemporalClient{executeErr: errors.New("temporal unavailable")}
	_, err := newTestEngine(stub).Submit(context.Background(), &zynaxv1.WorkflowIR{WorkflowId: "wf-x"}, nil)
	if err == nil || !containsStr(err.Error(), "temporal unavailable") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestTemporalEngine_Signal_Success(t *testing.T) {
	stub := &stubTemporalClient{}
	err := newTestEngine(stub).Signal(context.Background(), "wf-1", "review.approved", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTemporalEngine_Signal_Error(t *testing.T) {
	stub := &stubTemporalClient{signalErr: errors.New("not found")}
	err := newTestEngine(stub).Signal(context.Background(), "wf-1", "review.approved", nil)
	if err == nil || !containsStr(err.Error(), "not found") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestTemporalEngine_Cancel_Success(t *testing.T) {
	stub := &stubTemporalClient{}
	err := newTestEngine(stub).Cancel(context.Background(), "wf-1", "user requested")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTemporalEngine_Cancel_Error(t *testing.T) {
	stub := &stubTemporalClient{cancelErr: errors.New("already terminal")}
	err := newTestEngine(stub).Cancel(context.Background(), "wf-1", "")
	if err == nil || !containsStr(err.Error(), "already terminal") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestTemporalEngine_GetStatus_Running(t *testing.T) {
	stub := &stubTemporalClient{
		descResps: []*workflowservice.DescribeWorkflowExecutionResponse{
			descResp(enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING),
		},
	}
	run, err := newTestEngine(stub).GetStatus(context.Background(), "wf-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.Status != domain.WorkflowStatusRunning {
		t.Errorf("Status = %v; want Running", run.Status)
	}
}

func TestTemporalEngine_GetStatus_AllMappings(t *testing.T) {
	cases := []struct {
		in   enumspb.WorkflowExecutionStatus
		want domain.WorkflowStatus
	}{
		{enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING, domain.WorkflowStatusRunning},
		{enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED, domain.WorkflowStatusCompleted},
		{enumspb.WORKFLOW_EXECUTION_STATUS_FAILED, domain.WorkflowStatusFailed},
		{enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT, domain.WorkflowStatusFailed},
		{enumspb.WORKFLOW_EXECUTION_STATUS_CANCELED, domain.WorkflowStatusCancelled},
		{enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED, domain.WorkflowStatusCancelled},
	}
	for _, tc := range cases {
		got := temporalStatusToDomain(tc.in)
		if got != tc.want {
			t.Errorf("temporalStatusToDomain(%v) = %v; want %v", tc.in, got, tc.want)
		}
	}
}

func TestTemporalEngine_Watch_TerminatesOnCompleted(t *testing.T) {
	stub := &stubTemporalClient{
		descResps: []*workflowservice.DescribeWorkflowExecutionResponse{
			descResp(enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING),
			descResp(enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED),
		},
	}
	var received []domain.WorkflowStatus
	err := newTestEngine(stub).Watch(context.Background(), "wf-1", func(ev *domain.WorkflowEvent) error {
		received = append(received, ev.Status)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(received) < 2 {
		t.Fatalf("expected ≥2 events, got %d", len(received))
	}
	if received[len(received)-1] != domain.WorkflowStatusCompleted {
		t.Errorf("last status = %v; want Completed", received[len(received)-1])
	}
}

func TestTemporalEngine_Watch_ContextCancelled(t *testing.T) {
	stub := &stubTemporalClient{
		descResps: []*workflowservice.DescribeWorkflowExecutionResponse{
			descResp(enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING),
		},
	}
	engine := newTestEngine(stub)
	engine.pollInterval = time.Hour // long interval so ctx cancellation fires first

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := engine.Watch(ctx, "wf-1", func(_ *domain.WorkflowEvent) error { return nil })
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled in chain, got: %v", err)
	}
}

// containsStr is a test helper; it lives here to avoid importing strings in non-test code.
func containsStr(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
