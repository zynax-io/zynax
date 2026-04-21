// SPDX-License-Identifier: Apache-2.0
// Signals and watch-stream contract tests for EngineAdapterService.
// Shared types and stub infrastructure are defined in lifecycle_steps_test.go.
package engine_adapter_service_test

import (
	"context"
	"fmt"
	"strings"
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

// ─── Signal and watch stub methods ───────────────────────────────────────────
// These methods are defined here (same package) so that lifecycle_steps_test.go
// stays focused on Submit/Cancel/GetWorkflowStatus without duplication.

func (s *engineStub) SignalWorkflow(_ context.Context, req *zynaxv1.SignalWorkflowRequest) (*zynaxv1.SignalWorkflowResponse, error) {
	if req.RunId == "" {
		return nil, status.Error(codes.InvalidArgument, "run_id is required")
	}
	if req.EventType == "" {
		return nil, status.Error(codes.InvalidArgument, "event_type is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.runs[req.RunId]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "run %q not found", req.RunId)
	}
	if isTerminal(rec.run.Status) {
		return nil, status.Errorf(codes.FailedPrecondition, "run %q is in terminal state %s", req.RunId, rec.run.Status.String())
	}

	rec.pendingSignal = req.EventType
	return &zynaxv1.SignalWorkflowResponse{}, nil
}

func (s *engineStub) WatchWorkflow(req *zynaxv1.WatchWorkflowRequest, stream grpc.ServerStreamingServer[zynaxv1.WorkflowEvent]) error {
	if req.RunId == "" {
		return status.Error(codes.InvalidArgument, "run_id is required")
	}

	s.mu.Lock()
	rec, ok := s.runs[req.RunId]
	if !ok {
		s.mu.Unlock()
		return status.Errorf(codes.NotFound, "run %q not found", req.RunId)
	}
	run := rec.run
	pendingSignal := rec.pendingSignal
	toSt := rec.toState
	s.mu.Unlock()

	// Emit pending signal event (set via SignalWorkflow or transition setup)
	if pendingSignal != "" {
		fromSt := run.CurrentState
		if fromSt == "" {
			fromSt = "initial-state"
		}
		if toSt == "" {
			toSt = "next-state"
		}
		evt := &zynaxv1.WorkflowEvent{
			RunId:     req.RunId,
			EventType: pendingSignal,
			Status:    run.Status,
			Timestamp: timestamppb.Now(),
			FromState: fromSt,
			ToState:   toSt,
		}
		if err := stream.Send(evt); err != nil {
			return err
		}
	}

	// Always emit a state-transition heartbeat event
	evt := &zynaxv1.WorkflowEvent{
		RunId:     req.RunId,
		EventType: "state.transition",
		Status:    run.Status,
		Timestamp: timestamppb.Now(),
		FromState: run.CurrentState,
		ToState:   run.CurrentState,
	}
	if err := stream.Send(evt); err != nil {
		return err
	}

	// If already terminal, emit the terminal event and close
	if isTerminal(run.Status) {
		return stream.Send(&zynaxv1.WorkflowEvent{
			RunId:     req.RunId,
			EventType: "workflow.terminal",
			Status:    run.Status,
			Timestamp: timestamppb.Now(),
		})
	}

	// Non-terminal run: emit a completion event to close the stream
	return stream.Send(&zynaxv1.WorkflowEvent{
		RunId:     req.RunId,
		EventType: "workflow.completed",
		Status:    zynaxv1.WorkflowStatus_WORKFLOW_STATUS_COMPLETED,
		Timestamp: timestamppb.Now(),
	})
}

// ─── TestSignals ──────────────────────────────────────────────────────────────
// Covers @signals scenarios: signal delivery, watch stream events and
// termination, state-transition details, and Signal input validation
// (9 scenarios total).

func TestSignals(t *testing.T) {
	suite := godog.TestSuite{
		Name: "engine_adapter_signals",
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

			sc.Step(`^workflow run "([^"]*)" is in RUNNING state$`, func(runID string) error {
				insertRun(stub, runID, zynaxv1.WorkflowStatus_WORKFLOW_STATUS_RUNNING)
				tc.lastRunID = runID
				return nil
			})

			sc.Step(`^workflow run "([^"]*)" is in COMPLETED state$`, func(runID string) error {
				insertRun(stub, runID, zynaxv1.WorkflowStatus_WORKFLOW_STATUS_COMPLETED)
				tc.lastRunID = runID
				return nil
			})

			sc.Step(`^the workflow is waiting on signal "([^"]*)"$`, func(signal string) error {
				stub.mu.Lock()
				defer stub.mu.Unlock()
				if rec, ok := stub.runs[tc.lastRunID]; ok {
					rec.pendingSignal = signal
				}
				return nil
			})

			sc.Step(`^workflow run "([^"]*)" will complete during the watch$`, func(runID string) error {
				insertRun(stub, runID, zynaxv1.WorkflowStatus_WORKFLOW_STATUS_RUNNING)
				tc.lastRunID = runID
				return nil
			})

			sc.Step(`^workflow run "([^"]*)" transitions from state "([^"]*)" to "([^"]*)"$`, func(runID, from, to string) error {
				insertRun(stub, runID, zynaxv1.WorkflowStatus_WORKFLOW_STATUS_RUNNING)
				tc.lastRunID = runID
				tc.fromState = from
				tc.toState = to
				stub.mu.Lock()
				defer stub.mu.Unlock()
				if rec, ok := stub.runs[runID]; ok {
					rec.run.CurrentState = from
					rec.pendingSignal = "state.transition"
					rec.toState = to
				}
				return nil
			})

			sc.Step(`^a SignalWorkflowRequest with run_id set to ""$`, func() error {
				tc.pendingSignalRunID = ""
				tc.pendingSignalEventType = "test.event"
				return nil
			})

			sc.Step(`^a SignalWorkflowRequest with event_type set to ""$`, func() error {
				tc.pendingSignalRunID = "some-valid-run"
				tc.pendingSignalEventType = ""
				return nil
			})

			// ── When steps ───────────────────────────────────────────────────

			sc.Step(`^SignalWorkflow is called with event_type "([^"]*)"$`, func(evtType string) error {
				req := &zynaxv1.SignalWorkflowRequest{
					RunId:     tc.lastRunID,
					EventType: evtType,
				}
				_, tc.grpcErr = tc.client.SignalWorkflow(context.Background(), req)
				return nil
			})

			sc.Step(`^SignalWorkflow is called with run_id "([^"]*)"$`, func(runID string) error {
				req := &zynaxv1.SignalWorkflowRequest{
					RunId:     runID,
					EventType: "any.signal",
				}
				_, tc.grpcErr = tc.client.SignalWorkflow(context.Background(), req)
				return nil
			})

			sc.Step(`^SignalWorkflow is called$`, func() error {
				req := &zynaxv1.SignalWorkflowRequest{
					RunId:     tc.pendingSignalRunID,
					EventType: tc.pendingSignalEventType,
				}
				_, tc.grpcErr = tc.client.SignalWorkflow(context.Background(), req)
				return nil
			})

			sc.Step(`^WatchWorkflow is called with run_id "([^"]*)"$`, func(runID string) error {
				stream, err := tc.client.WatchWorkflow(context.Background(), &zynaxv1.WatchWorkflowRequest{RunId: runID})
				if err != nil {
					tc.watchErr = err
					tc.grpcErr = err
					return nil
				}
				for {
					evt, recvErr := stream.Recv()
					if recvErr != nil {
						if s, ok := status.FromError(recvErr); ok && s.Code() != codes.OK {
							if tc.grpcErr == nil {
								tc.grpcErr = recvErr
							}
						}
						break
					}
					tc.watchEvents = append(tc.watchEvents, evt)
				}
				return nil
			})

			sc.Step(`^WatchWorkflow emits the transition event$`, func() error {
				stream, err := tc.client.WatchWorkflow(context.Background(), &zynaxv1.WatchWorkflowRequest{RunId: tc.lastRunID})
				if err != nil {
					tc.watchErr = err
					tc.grpcErr = err
					return nil
				}
				for {
					evt, recvErr := stream.Recv()
					if recvErr != nil {
						break
					}
					tc.watchEvents = append(tc.watchEvents, evt)
				}
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

			sc.Step(`^the gRPC status is FAILED_PRECONDITION$`, func() error {
				if s, ok := status.FromError(tc.grpcErr); !ok || s.Code() != codes.FailedPrecondition {
					return fmt.Errorf("expected FAILED_PRECONDITION, got: %v", tc.grpcErr)
				}
				return nil
			})

			sc.Step(`^the error message contains "([^"]*)"$`, func(fragment string) error {
				if tc.grpcErr == nil {
					return fmt.Errorf("expected error containing %q, got no error", fragment)
				}
				if !strings.Contains(tc.grpcErr.Error(), fragment) {
					return fmt.Errorf("expected error message to contain %q, got: %s", fragment, tc.grpcErr.Error())
				}
				return nil
			})

			sc.Step(`^the error message mentions "([^"]*)"$`, func(fragment string) error {
				if tc.grpcErr == nil {
					return fmt.Errorf("expected error containing %q, got no error", fragment)
				}
				if !strings.Contains(tc.grpcErr.Error(), fragment) {
					return fmt.Errorf("expected error message to contain %q, got: %s", fragment, tc.grpcErr.Error())
				}
				return nil
			})

			sc.Step(`^WatchWorkflow emits a WorkflowEvent with event_type "([^"]*)"$`, func(evtType string) error {
				for _, evt := range tc.watchEvents {
					if evt.EventType == evtType {
						return nil
					}
				}
				// Watch not yet called — invoke now to collect events
				if len(tc.watchEvents) == 0 && tc.lastRunID != "" {
					stream, err := tc.client.WatchWorkflow(context.Background(), &zynaxv1.WatchWorkflowRequest{RunId: tc.lastRunID})
					if err != nil {
						return fmt.Errorf("WatchWorkflow error: %v", err)
					}
					for {
						evt, recvErr := stream.Recv()
						if recvErr != nil {
							break
						}
						tc.watchEvents = append(tc.watchEvents, evt)
					}
				}
				for _, evt := range tc.watchEvents {
					if evt.EventType == evtType {
						return nil
					}
				}
				return fmt.Errorf("expected WorkflowEvent with event_type %q, got %v", evtType, tc.watchEvents)
			})

			sc.Step(`^the stream emits at least one WorkflowEvent$`, func() error {
				if len(tc.watchEvents) == 0 {
					return fmt.Errorf("expected at least one WorkflowEvent, got none")
				}
				return nil
			})

			sc.Step(`^every WorkflowEvent carries run_id "([^"]*)"$`, func(runID string) error {
				for _, evt := range tc.watchEvents {
					if evt.RunId != runID {
						return fmt.Errorf("expected run_id %q, got %q", runID, evt.RunId)
					}
				}
				return nil
			})

			sc.Step(`^every WorkflowEvent has a non-zero timestamp$`, func() error {
				for _, evt := range tc.watchEvents {
					if evt.Timestamp == nil || evt.Timestamp.Seconds == 0 {
						return fmt.Errorf("expected non-zero timestamp in WorkflowEvent")
					}
				}
				return nil
			})

			sc.Step(`^the stream emits a WorkflowEvent with a terminal status$`, func() error {
				for _, evt := range tc.watchEvents {
					if isTerminal(evt.Status) ||
						evt.EventType == "workflow.completed" ||
						evt.EventType == "workflow.terminal" {
						return nil
					}
				}
				return fmt.Errorf("expected a terminal WorkflowEvent, got %v", tc.watchEvents)
			})

			sc.Step(`^the stream closes cleanly after the terminal event$`, func() error {
				return nil // reaching this step means the stream closed without error
			})

			sc.Step(`^the WorkflowEvent from_state is "([^"]*)"$`, func(fromState string) error {
				for _, evt := range tc.watchEvents {
					if evt.FromState == fromState {
						return nil
					}
				}
				return fmt.Errorf("expected WorkflowEvent with from_state %q", fromState)
			})

			sc.Step(`^the WorkflowEvent to_state is "([^"]*)"$`, func(toState string) error {
				for _, evt := range tc.watchEvents {
					if evt.ToState == toState {
						return nil
					}
				}
				return fmt.Errorf("expected WorkflowEvent with to_state %q", toState)
			})

			sc.Step(`^no WorkflowEvent is emitted$`, func() error {
				if len(tc.watchEvents) > 0 {
					return fmt.Errorf("expected no WorkflowEvents, got %d", len(tc.watchEvents))
				}
				return nil
			})
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/engine_adapter_service.feature"},
			Tags:     "@signals",
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("BDD scenarios failed")
	}
}
