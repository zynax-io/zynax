// SPDX-License-Identifier: Apache-2.0
// Package engine_adapter_service provides BDD contract tests for EngineAdapterService.
package engine_adapter_service_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/cucumber/godog"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/protos/tests/testserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ─── In-memory stub ──────────────────────────────────────────────────────────

type runRecord struct {
	run           *zynaxv1.WorkflowRun
	pendingSignal string // signal waiting for delivery via WatchWorkflow
	toState       string // target state for transition events
}

type engineStub struct {
	zynaxv1.UnimplementedEngineAdapterServiceServer
	mu   sync.Mutex
	runs map[string]*runRecord // keyed by run_id
}

func newEngineStub() *engineStub {
	return &engineStub{runs: make(map[string]*runRecord)}
}

func isTerminal(s zynaxv1.WorkflowStatus) bool {
	return s == zynaxv1.WorkflowStatus_WORKFLOW_STATUS_COMPLETED ||
		s == zynaxv1.WorkflowStatus_WORKFLOW_STATUS_FAILED ||
		s == zynaxv1.WorkflowStatus_WORKFLOW_STATUS_CANCELLED
}

func (s *engineStub) SubmitWorkflow(_ context.Context, req *zynaxv1.SubmitWorkflowRequest) (*zynaxv1.SubmitWorkflowResponse, error) {
	if req.WorkflowIr == nil {
		return nil, status.Error(codes.InvalidArgument, "workflow_ir is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	runID := fmt.Sprintf("run-%d", len(s.runs)+1)

	if existing, ok := s.runs[req.WorkflowIr.WorkflowId]; ok {
		if !isTerminal(existing.run.Status) {
			return nil, status.Errorf(codes.AlreadyExists, "run with id %q already exists and is active", req.WorkflowIr.WorkflowId)
		}
	}

	ns := req.WorkflowIr.Namespace
	if ns == "" {
		ns = req.Namespace
	}

	labels := req.Labels
	if labels == nil {
		labels = map[string]string{}
	}

	engine := req.EngineHint
	if engine == "" {
		engine = "default"
	}

	run := &zynaxv1.WorkflowRun{
		RunId:        runID,
		WorkflowId:   req.WorkflowIr.WorkflowId,
		Namespace:    ns,
		Status:       zynaxv1.WorkflowStatus_WORKFLOW_STATUS_RUNNING,
		CurrentState: "initial-state",
		Labels:       labels,
		Engine:       engine,
		SubmittedAt:  timestamppb.Now(),
		StartedAt:    timestamppb.Now(),
	}
	s.runs[runID] = &runRecord{run: run}

	return &zynaxv1.SubmitWorkflowResponse{
		RunId:       runID,
		SubmittedAt: run.SubmittedAt,
	}, nil
}

func (s *engineStub) CancelWorkflow(_ context.Context, req *zynaxv1.CancelWorkflowRequest) (*zynaxv1.CancelWorkflowResponse, error) {
	if req.RunId == "" {
		return nil, status.Error(codes.InvalidArgument, "run_id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.runs[req.RunId]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "run %q not found", req.RunId)
	}
	if isTerminal(rec.run.Status) {
		return nil, status.Errorf(codes.FailedPrecondition, "run %q is already in terminal state %s", req.RunId, rec.run.Status.String())
	}

	rec.run.Status = zynaxv1.WorkflowStatus_WORKFLOW_STATUS_CANCELLED
	rec.run.CancellationReason = req.Reason
	rec.run.FinishedAt = timestamppb.Now()

	return &zynaxv1.CancelWorkflowResponse{}, nil
}

func (s *engineStub) GetWorkflowStatus(_ context.Context, req *zynaxv1.GetWorkflowStatusRequest) (*zynaxv1.WorkflowRun, error) {
	if req.RunId == "" {
		return nil, status.Error(codes.InvalidArgument, "run_id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.runs[req.RunId]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "run %q not found", req.RunId)
	}
	return rec.run, nil
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

func insertRun(stub *engineStub, runID string, wfStatus zynaxv1.WorkflowStatus) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	run := &zynaxv1.WorkflowRun{
		RunId:        runID,
		WorkflowId:   "wf-" + runID,
		Namespace:    "default",
		Status:       wfStatus,
		CurrentState: "active-state",
		Engine:       "default",
		SubmittedAt:  timestamppb.Now(),
		StartedAt:    timestamppb.Now(),
	}
	if isTerminal(wfStatus) {
		run.FinishedAt = timestamppb.Now()
	}
	stub.runs[runID] = &runRecord{run: run}
}

// ─── Test context ─────────────────────────────────────────────────────────────

type engineCtx struct {
	client                 zynaxv1.EngineAdapterServiceClient
	stub                   *engineStub
	lastRunID              string
	lastWorkflowID         string
	pendingNS              string
	pendingLabels          map[string]string
	pendingEngineHint      string
	submitResp             *zynaxv1.SubmitWorkflowResponse
	statusResp             *zynaxv1.WorkflowRun
	watchEvents            []*zynaxv1.WorkflowEvent
	watchErr               error
	grpcErr                error
	fromState              string
	toState                string
	pendingSignalRunID     string
	pendingSignalEventType string
	pendingCancelRunID     string
}

type godogEKey struct{}

// ─── TestLifecycle ────────────────────────────────────────────────────────────
// Covers @lifecycle scenarios: submission, status query, cancellation, and
// input validation for Submit/Cancel/GetWorkflowStatus (15 scenarios total).

func TestLifecycle(t *testing.T) {
	suite := godog.TestSuite{
		Name: "engine_adapter_lifecycle",
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			srv, dialer := testserver.NewBufconnServer(t)
			stub := newEngineStub()
			zynaxv1.RegisterEngineAdapterServiceServer(srv, stub)

			conn, err := grpc.NewClient(
				"passthrough://bufnet",
				grpc.WithContextDialer(dialer),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				t.Fatalf("failed to dial: %v", err)
			}
			t.Cleanup(func() { conn.Close() })

			tc := &engineCtx{
				client: zynaxv1.NewEngineAdapterServiceClient(conn),
				stub:   stub,
			}

			sc.Before(func(ctx context.Context, scenario *godog.Scenario) (context.Context, error) {
				tc.lastRunID = ""
				tc.lastWorkflowID = ""
				tc.pendingNS = ""
				tc.pendingLabels = nil
				tc.pendingEngineHint = ""
				tc.submitResp = nil
				tc.statusResp = nil
				tc.watchEvents = nil
				tc.watchErr = nil
				tc.grpcErr = nil
				tc.fromState = ""
				tc.toState = ""
				tc.pendingSignalRunID = ""
				tc.pendingSignalEventType = ""
				tc.pendingCancelRunID = ""
				stub.mu.Lock()
				stub.runs = make(map[string]*runRecord)
				stub.mu.Unlock()
				return context.WithValue(ctx, godogEKey{}, t), nil
			})

			// ── Given steps ──────────────────────────────────────────────────

			sc.Step(`^an EngineAdapterService is running on a test gRPC server$`, func() error {
				return nil
			})

			sc.Step(`^a compiled WorkflowIR for workflow "([^"]*)"$`, func(name string) error {
				tc.lastWorkflowID = name
				return nil
			})

			sc.Step(`^a compiled WorkflowIR with namespace "([^"]*)"$`, func(ns string) error {
				tc.lastWorkflowID = "wf-ns-test"
				tc.pendingNS = ns
				return nil
			})

			sc.Step(`^a WorkflowIR and SubmitWorkflowRequest labels \{"([^"]*)": "([^"]*)"\}$`, func(key, value string) error {
				tc.lastWorkflowID = "wf-label-test"
				tc.pendingLabels = map[string]string{key: value}
				return nil
			})

			sc.Step(`^a compiled WorkflowIR$`, func() error {
				tc.lastWorkflowID = "wf-generic"
				return nil
			})

			sc.Step(`^the SubmitWorkflowRequest has engine_hint "([^"]*)"$`, func(hint string) error {
				tc.pendingEngineHint = hint
				return nil
			})

			sc.Step(`^a workflow run "([^"]*)" is already RUNNING$`, func(runID string) error {
				insertRun(stub, runID, zynaxv1.WorkflowStatus_WORKFLOW_STATUS_RUNNING)
				tc.lastRunID = runID
				return nil
			})

			sc.Step(`^workflow run "([^"]*)" is in RUNNING state$`, func(runID string) error {
				insertRun(stub, runID, zynaxv1.WorkflowStatus_WORKFLOW_STATUS_RUNNING)
				tc.lastRunID = runID
				return nil
			})

			sc.Step(`^workflow run "([^"]*)" is in PENDING state$`, func(runID string) error {
				insertRun(stub, runID, zynaxv1.WorkflowStatus_WORKFLOW_STATUS_PENDING)
				tc.lastRunID = runID
				return nil
			})

			sc.Step(`^workflow run "([^"]*)" is in COMPLETED state$`, func(runID string) error {
				insertRun(stub, runID, zynaxv1.WorkflowStatus_WORKFLOW_STATUS_COMPLETED)
				tc.lastRunID = runID
				return nil
			})

			sc.Step(`^workflow run "([^"]*)" has reached COMPLETED state$`, func(runID string) error {
				insertRun(stub, runID, zynaxv1.WorkflowStatus_WORKFLOW_STATUS_COMPLETED)
				tc.lastRunID = runID
				return nil
			})

			sc.Step(`^a SubmitWorkflowRequest with no workflow_ir$`, func() error {
				tc.lastWorkflowID = ""
				return nil
			})

			sc.Step(`^a CancelWorkflowRequest with run_id set to ""$`, func() error {
				tc.pendingCancelRunID = ""
				return nil
			})

			sc.Step(`^a GetWorkflowStatusRequest with run_id set to ""$`, func() error {
				return nil
			})

			// ── When steps ───────────────────────────────────────────────────

			sc.Step(`^SubmitWorkflow is called with the IR$`, func() error {
				var ir *zynaxv1.WorkflowIR
				if tc.lastWorkflowID != "" {
					ns := tc.pendingNS
					if ns == "" {
						ns = "default"
					}
					ir = &zynaxv1.WorkflowIR{
						WorkflowId: tc.lastWorkflowID,
						Name:       tc.lastWorkflowID,
						Namespace:  ns,
					}
				}
				req := &zynaxv1.SubmitWorkflowRequest{
					WorkflowIr: ir,
					Labels:     tc.pendingLabels,
					EngineHint: tc.pendingEngineHint,
				}
				tc.submitResp, tc.grpcErr = tc.client.SubmitWorkflow(context.Background(), req)
				if tc.grpcErr == nil {
					tc.lastRunID = tc.submitResp.RunId
				}
				return nil
			})

			sc.Step(`^SubmitWorkflow is called with the same run_id "([^"]*)"$`, func(runID string) error {
				stub.mu.Lock()
				stub.runs[runID] = &runRecord{run: &zynaxv1.WorkflowRun{
					RunId:  runID,
					Status: zynaxv1.WorkflowStatus_WORKFLOW_STATUS_RUNNING,
				}}
				stub.mu.Unlock()
				ir := &zynaxv1.WorkflowIR{WorkflowId: runID}
				_, tc.grpcErr = tc.client.SubmitWorkflow(context.Background(), &zynaxv1.SubmitWorkflowRequest{WorkflowIr: ir})
				return nil
			})

			sc.Step(`^SubmitWorkflow is called$`, func() error {
				var ir *zynaxv1.WorkflowIR
				if tc.lastWorkflowID != "" {
					ns := tc.pendingNS
					if ns == "" {
						ns = "default"
					}
					ir = &zynaxv1.WorkflowIR{
						WorkflowId: tc.lastWorkflowID,
						Name:       tc.lastWorkflowID,
						Namespace:  ns,
					}
				}
				req := &zynaxv1.SubmitWorkflowRequest{
					WorkflowIr: ir,
					Labels:     tc.pendingLabels,
					EngineHint: tc.pendingEngineHint,
				}
				tc.submitResp, tc.grpcErr = tc.client.SubmitWorkflow(context.Background(), req)
				if tc.grpcErr == nil && tc.submitResp != nil {
					tc.lastRunID = tc.submitResp.RunId
				}
				return nil
			})

			sc.Step(`^CancelWorkflow is called with run_id "([^"]*)" and reason "([^"]*)"$`, func(runID, reason string) error {
				tc.lastRunID = runID
				req := &zynaxv1.CancelWorkflowRequest{RunId: runID, Reason: reason}
				_, tc.grpcErr = tc.client.CancelWorkflow(context.Background(), req)
				return nil
			})

			sc.Step(`^CancelWorkflow is called with run_id "([^"]*)"$`, func(runID string) error {
				tc.lastRunID = runID
				req := &zynaxv1.CancelWorkflowRequest{RunId: runID}
				_, tc.grpcErr = tc.client.CancelWorkflow(context.Background(), req)
				return nil
			})

			sc.Step(`^CancelWorkflow is called$`, func() error {
				req := &zynaxv1.CancelWorkflowRequest{RunId: tc.pendingCancelRunID}
				_, tc.grpcErr = tc.client.CancelWorkflow(context.Background(), req)
				return nil
			})

			sc.Step(`^GetWorkflowStatus for that run_id returns status RUNNING$`, func() error {
				resp, err := tc.client.GetWorkflowStatus(context.Background(), &zynaxv1.GetWorkflowStatusRequest{RunId: tc.lastRunID})
				if err != nil {
					return fmt.Errorf("GetWorkflowStatus error: %v", err)
				}
				if resp.Status != zynaxv1.WorkflowStatus_WORKFLOW_STATUS_RUNNING {
					return fmt.Errorf("expected RUNNING, got %s", resp.Status)
				}
				return nil
			})

			sc.Step(`^GetWorkflowStatus returns namespace "([^"]*)"$`, func(ns string) error {
				if tc.submitResp == nil {
					return fmt.Errorf("no submit response available")
				}
				resp, err := tc.client.GetWorkflowStatus(context.Background(), &zynaxv1.GetWorkflowStatusRequest{RunId: tc.submitResp.RunId})
				if err != nil {
					return fmt.Errorf("GetWorkflowStatus error: %v", err)
				}
				if resp.Namespace != ns {
					return fmt.Errorf("expected namespace %q, got %q", ns, resp.Namespace)
				}
				return nil
			})

			sc.Step(`^GetWorkflowStatus returns label "([^"]*)" with value "([^"]*)"$`, func(key, value string) error {
				if tc.submitResp == nil {
					return fmt.Errorf("no submit response available")
				}
				resp, err := tc.client.GetWorkflowStatus(context.Background(), &zynaxv1.GetWorkflowStatusRequest{RunId: tc.submitResp.RunId})
				if err != nil {
					return fmt.Errorf("GetWorkflowStatus error: %v", err)
				}
				if resp.Labels[key] != value {
					return fmt.Errorf("expected label %q=%q, got %q", key, value, resp.Labels[key])
				}
				return nil
			})

			sc.Step(`^GetWorkflowStatus for "([^"]*)" returns status CANCELLED$`, func(runID string) error {
				resp, err := tc.client.GetWorkflowStatus(context.Background(), &zynaxv1.GetWorkflowStatusRequest{RunId: runID})
				if err != nil {
					return fmt.Errorf("GetWorkflowStatus error: %v", err)
				}
				if resp.Status != zynaxv1.WorkflowStatus_WORKFLOW_STATUS_CANCELLED {
					return fmt.Errorf("expected CANCELLED, got %s", resp.Status)
				}
				return nil
			})

			sc.Step(`^GetWorkflowStatus is called with run_id "([^"]*)"$`, func(runID string) error {
				tc.statusResp, tc.grpcErr = tc.client.GetWorkflowStatus(context.Background(), &zynaxv1.GetWorkflowStatusRequest{RunId: runID})
				return nil
			})

			sc.Step(`^GetWorkflowStatus is called$`, func() error {
				tc.statusResp, tc.grpcErr = tc.client.GetWorkflowStatus(context.Background(), &zynaxv1.GetWorkflowStatusRequest{RunId: ""})
				return nil
			})

			// ── Then steps ───────────────────────────────────────────────────

			sc.Step(`^the gRPC status is OK$`, func() error {
				if tc.grpcErr != nil {
					return fmt.Errorf("expected OK, got error: %v", tc.grpcErr)
				}
				return nil
			})

			sc.Step(`^the gRPC status is INVALID_ARGUMENT$`, func() error {
				if s, ok := status.FromError(tc.grpcErr); !ok || s.Code() != codes.InvalidArgument {
					return fmt.Errorf("expected INVALID_ARGUMENT, got: %v", tc.grpcErr)
				}
				return nil
			})

			sc.Step(`^the gRPC status is NOT_FOUND$`, func() error {
				if s, ok := status.FromError(tc.grpcErr); !ok || s.Code() != codes.NotFound {
					return fmt.Errorf("expected NOT_FOUND, got: %v", tc.grpcErr)
				}
				return nil
			})

			sc.Step(`^the gRPC status is ALREADY_EXISTS$`, func() error {
				if s, ok := status.FromError(tc.grpcErr); !ok || s.Code() != codes.AlreadyExists {
					return fmt.Errorf("expected ALREADY_EXISTS, got: %v", tc.grpcErr)
				}
				return nil
			})

			sc.Step(`^the gRPC status is FAILED_PRECONDITION$`, func() error {
				if s, ok := status.FromError(tc.grpcErr); !ok || s.Code() != codes.FailedPrecondition {
					return fmt.Errorf("expected FAILED_PRECONDITION, got: %v", tc.grpcErr)
				}
				return nil
			})

			sc.Step(`^the response contains a non-empty run_id$`, func() error {
				if tc.submitResp == nil {
					return fmt.Errorf("submit response is nil")
				}
				if tc.submitResp.RunId == "" {
					return fmt.Errorf("run_id is empty")
				}
				return nil
			})

			sc.Step(`^the workflow is executed by the "([^"]*)" engine$`, func(engine string) error {
				if tc.submitResp == nil {
					return fmt.Errorf("submit response is nil")
				}
				resp, err := tc.client.GetWorkflowStatus(context.Background(), &zynaxv1.GetWorkflowStatusRequest{RunId: tc.submitResp.RunId})
				if err != nil {
					return fmt.Errorf("GetWorkflowStatus error: %v", err)
				}
				if resp.Engine != engine {
					return fmt.Errorf("expected engine %q, got %q", engine, resp.Engine)
				}
				return nil
			})

			sc.Step(`^the error message contains "([^"]*)"$`, func(fragment string) error {
				if tc.grpcErr == nil {
					return fmt.Errorf("expected error containing %q, got no error", fragment)
				}
				msg := tc.grpcErr.Error()
				if !strings.Contains(msg, fragment) {
					return fmt.Errorf("expected error message to contain %q, got: %s", fragment, msg)
				}
				return nil
			})

			sc.Step(`^the error message mentions "([^"]*)"$`, func(fragment string) error {
				if tc.grpcErr == nil {
					return fmt.Errorf("expected error containing %q, got no error", fragment)
				}
				msg := tc.grpcErr.Error()
				if !strings.Contains(msg, fragment) {
					return fmt.Errorf("expected error message to contain %q, got: %s", fragment, msg)
				}
				return nil
			})

			sc.Step(`^the cancellation reason is stored on the run record$`, func() error {
				if tc.lastRunID == "" {
					return fmt.Errorf("no lastRunID")
				}
				resp, err := tc.client.GetWorkflowStatus(context.Background(), &zynaxv1.GetWorkflowStatusRequest{RunId: tc.lastRunID})
				if err != nil {
					return fmt.Errorf("GetWorkflowStatus error: %v", err)
				}
				if resp.CancellationReason == "" {
					return fmt.Errorf("expected non-empty cancellation_reason")
				}
				return nil
			})

			sc.Step(`^the response includes a non-empty current_state$`, func() error {
				if tc.statusResp == nil {
					return fmt.Errorf("status response is nil")
				}
				if tc.statusResp.CurrentState == "" {
					return fmt.Errorf("current_state is empty")
				}
				return nil
			})

			sc.Step(`^the response includes a non-zero started_at timestamp$`, func() error {
				if tc.statusResp == nil {
					return fmt.Errorf("status response is nil")
				}
				if tc.statusResp.StartedAt == nil || tc.statusResp.StartedAt.Seconds == 0 {
					return fmt.Errorf("started_at is zero")
				}
				return nil
			})

			sc.Step(`^the response includes the workflow_id from the original IR$`, func() error {
				if tc.statusResp == nil {
					return fmt.Errorf("status response is nil")
				}
				if tc.statusResp.WorkflowId == "" {
					return fmt.Errorf("workflow_id is empty")
				}
				return nil
			})

			sc.Step(`^the response status is RUNNING$`, func() error {
				if tc.statusResp == nil {
					return fmt.Errorf("status response is nil")
				}
				if tc.statusResp.Status != zynaxv1.WorkflowStatus_WORKFLOW_STATUS_RUNNING {
					return fmt.Errorf("expected RUNNING, got %s", tc.statusResp.Status)
				}
				return nil
			})

			sc.Step(`^the response includes a non-zero finished_at timestamp$`, func() error {
				if tc.statusResp == nil {
					return fmt.Errorf("status response is nil")
				}
				if tc.statusResp.FinishedAt == nil || tc.statusResp.FinishedAt.Seconds == 0 {
					return fmt.Errorf("finished_at is zero")
				}
				return nil
			})
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/engine_adapter_service.feature"},
			Tags:     "@lifecycle",
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("BDD scenarios failed")
	}
}
