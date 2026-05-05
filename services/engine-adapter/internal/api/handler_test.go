// SPDX-License-Identifier: Apache-2.0

// Package api_test verifies the EngineAdapterService gRPC handler via a real
// in-memory gRPC server (bufconn), exercising the full request/response path
// for each of the 5 BDD scenarios in engine_adapter.feature.
package api_test

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/engine-adapter/internal/api"
	"github.com/zynax-io/zynax/services/engine-adapter/internal/domain"
)

// ─── stub engine ─────────────────────────────────────────────────────────────

type stubEngine struct {
	submitRun *domain.WorkflowRun
	submitErr error
	signalErr error
	cancelErr error
	statusRun *domain.WorkflowRun
	statusErr error
	watchEvts []*domain.WorkflowEvent
	watchErr  error
}

func (s *stubEngine) Submit(_ context.Context, _ *zynaxv1.WorkflowIR, labels map[string]string) (*domain.WorkflowRun, error) {
	if s.submitErr != nil {
		return nil, s.submitErr
	}
	run := *s.submitRun
	run.Labels = labels
	return &run, nil
}

func (s *stubEngine) Signal(_ context.Context, _, _ string, _ []byte) error { return s.signalErr }
func (s *stubEngine) Cancel(_ context.Context, _, _ string) error           { return s.cancelErr }

func (s *stubEngine) GetStatus(_ context.Context, _ string) (*domain.WorkflowRun, error) {
	return s.statusRun, s.statusErr
}

func (s *stubEngine) Watch(_ context.Context, _ string, send func(*domain.WorkflowEvent) error) error {
	for _, ev := range s.watchEvts {
		if err := send(ev); err != nil {
			return err
		}
	}
	return s.watchErr
}

// ─── test server helper ───────────────────────────────────────────────────────

func dialTestServer(t *testing.T, engine domain.WorkflowEngine) zynaxv1.EngineAdapterServiceClient {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	zynaxv1.RegisterEngineAdapterServiceServer(srv, api.NewHandler(engine))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() {
		srv.GracefulStop()
		_ = lis.Close()
	})
	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return zynaxv1.NewEngineAdapterServiceClient(conn)
}

// ─── BDD Scenario 1: Submit IR creates a running execution ───────────────────

func TestHandler_SubmitWorkflow_Success(t *testing.T) {
	engine := &stubEngine{
		submitRun: &domain.WorkflowRun{
			RunID:     "run-1",
			Status:    domain.WorkflowStatusRunning,
			Namespace: "default",
		},
	}
	client := dialTestServer(t, engine)
	ir := &zynaxv1.WorkflowIR{WorkflowId: "wf-1", InitialState: "review"}
	resp, err := client.SubmitWorkflow(context.Background(), &zynaxv1.SubmitWorkflowRequest{WorkflowIr: ir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.RunId == "" {
		t.Error("expected non-empty RunId")
	}
}

func TestHandler_SubmitWorkflow_NilIR(t *testing.T) {
	client := dialTestServer(t, &stubEngine{})
	_, err := client.SubmitWorkflow(context.Background(), &zynaxv1.SubmitWorkflowRequest{})
	if code := status.Code(err); code != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", code)
	}
}

func TestHandler_SubmitWorkflow_EngineError(t *testing.T) {
	engine := &stubEngine{submitErr: domain.ErrEngineUnavailable}
	client := dialTestServer(t, engine)
	ir := &zynaxv1.WorkflowIR{WorkflowId: "wf-2"}
	_, err := client.SubmitWorkflow(context.Background(), &zynaxv1.SubmitWorkflowRequest{WorkflowIr: ir})
	if code := status.Code(err); code != codes.Unavailable {
		t.Errorf("expected Unavailable, got %v", code)
	}
}

// ─── BDD Scenario 2: Signal transitions workflow ──────────────────────────────

func TestHandler_SignalWorkflow_Success(t *testing.T) {
	client := dialTestServer(t, &stubEngine{})
	_, err := client.SignalWorkflow(context.Background(), &zynaxv1.SignalWorkflowRequest{
		RunId:     "run-1",
		EventType: "review.approved",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandler_SignalWorkflow_EmptyRunID(t *testing.T) {
	client := dialTestServer(t, &stubEngine{})
	_, err := client.SignalWorkflow(context.Background(), &zynaxv1.SignalWorkflowRequest{})
	if code := status.Code(err); code != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", code)
	}
}

func TestHandler_SignalWorkflow_NotFound(t *testing.T) {
	engine := &stubEngine{signalErr: domain.ErrExecutionNotFound}
	client := dialTestServer(t, engine)
	_, err := client.SignalWorkflow(context.Background(), &zynaxv1.SignalWorkflowRequest{RunId: "missing"})
	if code := status.Code(err); code != codes.NotFound {
		t.Errorf("expected NotFound, got %v", code)
	}
}

// ─── BDD Scenario 4: Cancel terminates a running execution ───────────────────

func TestHandler_CancelWorkflow_Success(t *testing.T) {
	client := dialTestServer(t, &stubEngine{})
	_, err := client.CancelWorkflow(context.Background(), &zynaxv1.CancelWorkflowRequest{
		RunId:  "run-1",
		Reason: "user requested",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandler_CancelWorkflow_EmptyRunID(t *testing.T) {
	client := dialTestServer(t, &stubEngine{})
	_, err := client.CancelWorkflow(context.Background(), &zynaxv1.CancelWorkflowRequest{})
	if code := status.Code(err); code != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", code)
	}
}

func TestHandler_CancelWorkflow_TerminalState(t *testing.T) {
	engine := &stubEngine{cancelErr: domain.ErrTerminalState}
	client := dialTestServer(t, engine)
	_, err := client.CancelWorkflow(context.Background(), &zynaxv1.CancelWorkflowRequest{RunId: "run-1"})
	if code := status.Code(err); code != codes.FailedPrecondition {
		t.Errorf("expected FailedPrecondition, got %v", code)
	}
}

// ─── GetWorkflowStatus (supports Scenarios 1 and 4) ──────────────────────────

func TestHandler_GetWorkflowStatus_Running(t *testing.T) {
	engine := &stubEngine{
		statusRun: &domain.WorkflowRun{
			RunID:  "run-1",
			Status: domain.WorkflowStatusRunning,
		},
	}
	client := dialTestServer(t, engine)
	resp, err := client.GetWorkflowStatus(context.Background(), &zynaxv1.GetWorkflowStatusRequest{RunId: "run-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != zynaxv1.WorkflowStatus_WORKFLOW_STATUS_RUNNING {
		t.Errorf("expected RUNNING, got %v", resp.Status)
	}
}

func TestHandler_GetWorkflowStatus_Cancelled(t *testing.T) {
	engine := &stubEngine{
		statusRun: &domain.WorkflowRun{
			RunID:  "run-1",
			Status: domain.WorkflowStatusCancelled,
		},
	}
	client := dialTestServer(t, engine)
	resp, err := client.GetWorkflowStatus(context.Background(), &zynaxv1.GetWorkflowStatusRequest{RunId: "run-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != zynaxv1.WorkflowStatus_WORKFLOW_STATUS_CANCELLED {
		t.Errorf("expected CANCELLED, got %v", resp.Status)
	}
}

func TestHandler_GetWorkflowStatus_EmptyRunID(t *testing.T) {
	client := dialTestServer(t, &stubEngine{})
	_, err := client.GetWorkflowStatus(context.Background(), &zynaxv1.GetWorkflowStatusRequest{})
	if code := status.Code(err); code != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", code)
	}
}

func TestHandler_GetWorkflowStatus_NotFound(t *testing.T) {
	engine := &stubEngine{statusErr: domain.ErrExecutionNotFound}
	client := dialTestServer(t, engine)
	_, err := client.GetWorkflowStatus(context.Background(), &zynaxv1.GetWorkflowStatusRequest{RunId: "missing"})
	if code := status.Code(err); code != codes.NotFound {
		t.Errorf("expected NotFound, got %v", code)
	}
}

// ─── WatchWorkflow ─────────────────────────────────────────────────────────────

func TestHandler_WatchWorkflow_StreamsEventsToTerminal(t *testing.T) {
	engine := &stubEngine{
		watchEvts: []*domain.WorkflowEvent{
			{RunID: "run-1", Status: domain.WorkflowStatusRunning, Timestamp: time.Now()},
			{RunID: "run-1", Status: domain.WorkflowStatusCompleted, Timestamp: time.Now()},
		},
	}
	client := dialTestServer(t, engine)
	stream, err := client.WatchWorkflow(context.Background(), &zynaxv1.WatchWorkflowRequest{RunId: "run-1"})
	if err != nil {
		t.Fatalf("watch setup error: %v", err)
	}
	var events []*zynaxv1.WorkflowEvent
	for {
		ev, recvErr := stream.Recv()
		if recvErr != nil {
			break
		}
		events = append(events, ev)
	}
	if len(events) < 2 {
		t.Fatalf("expected ≥2 events, got %d", len(events))
	}
	if events[len(events)-1].Status != zynaxv1.WorkflowStatus_WORKFLOW_STATUS_COMPLETED {
		t.Errorf("last status = %v; want COMPLETED", events[len(events)-1].Status)
	}
}

func TestHandler_WatchWorkflow_EmptyRunID(t *testing.T) {
	client := dialTestServer(t, &stubEngine{})
	stream, err := client.WatchWorkflow(context.Background(), &zynaxv1.WatchWorkflowRequest{})
	if err != nil {
		t.Fatalf("watch setup error: %v", err)
	}
	_, err = stream.Recv()
	if code := status.Code(err); code != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", code)
	}
}

// ─── BDD Scenario 5: Engine swappability ─────────────────────────────────────

func TestHandler_EngineSwappability(t *testing.T) {
	temporalEng := &stubEngine{submitRun: &domain.WorkflowRun{RunID: "run-temporal", Engine: "temporal"}}
	langgraphEng := &stubEngine{submitRun: &domain.WorkflowRun{RunID: "run-langgraph", Engine: "langgraph"}}

	ir := &zynaxv1.WorkflowIR{WorkflowId: "wf-swap"}
	req := &zynaxv1.SubmitWorkflowRequest{WorkflowIr: ir}

	clientA := dialTestServer(t, temporalEng)
	clientB := dialTestServer(t, langgraphEng)

	respA, err := clientA.SubmitWorkflow(context.Background(), req)
	if err != nil {
		t.Fatalf("temporal engine error: %v", err)
	}
	respB, err := clientB.SubmitWorkflow(context.Background(), req)
	if err != nil {
		t.Fatalf("langgraph engine error: %v", err)
	}
	// Both engines produce the same gRPC response structure.
	if respA.RunId == "" || respB.RunId == "" {
		t.Error("expected non-empty RunId from both engines")
	}
	if respA.SubmittedAt == nil || respB.SubmittedAt == nil {
		t.Error("expected SubmittedAt from both engines")
	}
}
