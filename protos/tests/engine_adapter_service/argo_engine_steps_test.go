// SPDX-License-Identifier: Apache-2.0
// Package engine_adapter_service provides BDD contract tests for EngineAdapterService.
// This file wires the argo_engine.feature scenarios to the shared step definitions
// defined in lifecycle_steps_test.go. The feature file is committed first per ADR-016.
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
)

// TestArgoEngine runs all @argo-submit, @argo-query, and @argo-cancel scenarios
// from features/argo_engine.feature using the shared in-memory stub from
// lifecycle_steps_test.go.
//
//nolint:cyclop,funlen
func TestArgoEngine(t *testing.T) {
	suite := godog.TestSuite{
		Name: "engine_adapter_argo_engine",
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
			t.Cleanup(func() { _ = conn.Close() }) //nolint:errcheck

			tc := &engineCtx{
				client: zynaxv1.NewEngineAdapterServiceClient(conn),
				stub:   stub,
			}

			sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
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

			sc.Step(`^a compiled WorkflowIR$`, func() error {
				tc.lastWorkflowID = "wf-generic"
				return nil
			})

			sc.Step(`^the SubmitWorkflowRequest has engine_hint "([^"]*)"$`, func(hint string) error {
				tc.pendingEngineHint = hint
				return nil
			})

			sc.Step(`^a SubmitWorkflowRequest with no workflow_ir$`, func() error {
				tc.lastWorkflowID = ""
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

			sc.Step(`^workflow run "([^"]*)" is in COMPLETED state$`, func(runID string) error {
				insertRun(stub, runID, zynaxv1.WorkflowStatus_WORKFLOW_STATUS_COMPLETED)
				tc.lastRunID = runID
				return nil
			})

			// ── When steps ───────────────────────────────────────────────────

			sc.Step(`^SubmitWorkflow is called with the IR$`, func() error {
				var ir *zynaxv1.WorkflowIR
				if tc.lastWorkflowID != "" {
					ns := tc.pendingNS
					if ns == "" {
						ns = "default" //nolint:goconst
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
					return fmt.Errorf("expected OK, got error: %w", tc.grpcErr)
				}
				return nil
			})

			sc.Step(`^the gRPC status is INVALID_ARGUMENT$`, func() error {
				if s, ok := status.FromError(tc.grpcErr); !ok || s.Code() != codes.InvalidArgument {
					return fmt.Errorf("expected INVALID_ARGUMENT, got: %w", tc.grpcErr)
				}
				return nil
			})

			sc.Step(`^the gRPC status is NOT_FOUND$`, func() error {
				if s, ok := status.FromError(tc.grpcErr); !ok || s.Code() != codes.NotFound {
					return fmt.Errorf("expected NOT_FOUND, got: %w", tc.grpcErr)
				}
				return nil
			})

			sc.Step(`^the gRPC status is ALREADY_EXISTS$`, func() error {
				if s, ok := status.FromError(tc.grpcErr); !ok || s.Code() != codes.AlreadyExists {
					return fmt.Errorf("expected ALREADY_EXISTS, got: %w", tc.grpcErr)
				}
				return nil
			})

			sc.Step(`^the gRPC status is FAILED_PRECONDITION$`, func() error {
				if s, ok := status.FromError(tc.grpcErr); !ok || s.Code() != codes.FailedPrecondition {
					return fmt.Errorf("expected FAILED_PRECONDITION, got: %w", tc.grpcErr)
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
					return fmt.Errorf("GetWorkflowStatus error: %w", err)
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
					return fmt.Errorf("expected error containing %q, got: %s", fragment, msg)
				}
				return nil
			})

			sc.Step(`^GetWorkflowStatus for "([^"]*)" returns status CANCELLED$`, func(runID string) error {
				resp, err := tc.client.GetWorkflowStatus(context.Background(), &zynaxv1.GetWorkflowStatusRequest{RunId: runID})
				if err != nil {
					return fmt.Errorf("GetWorkflowStatus error: %w", err)
				}
				if resp.Status != zynaxv1.WorkflowStatus_WORKFLOW_STATUS_CANCELLED {
					return fmt.Errorf("expected CANCELLED, got %s", resp.Status)
				}
				return nil
			})

			sc.Step(`^the cancellation reason is stored on the run record$`, func() error {
				if tc.lastRunID == "" {
					return fmt.Errorf("no lastRunID")
				}
				resp, err := tc.client.GetWorkflowStatus(context.Background(), &zynaxv1.GetWorkflowStatusRequest{RunId: tc.lastRunID})
				if err != nil {
					return fmt.Errorf("GetWorkflowStatus error: %w", err)
				}
				if resp.CancellationReason == "" {
					return fmt.Errorf("expected non-empty cancellation_reason")
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

			sc.Step(`^the response includes a non-empty current_state$`, func() error {
				if tc.statusResp == nil {
					return fmt.Errorf("status response is nil")
				}
				if tc.statusResp.CurrentState == "" {
					return fmt.Errorf("current_state is empty")
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
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features/argo_engine.feature"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("BDD scenarios failed")
	}
}
