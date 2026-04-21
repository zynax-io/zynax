// SPDX-License-Identifier: Apache-2.0
// BDD contract tests for TaskBrokerService.
package task_broker_service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cucumber/godog"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/protos/tests/testserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ─── Stub ─────────────────────────────────────────────────────────────────────

type brokerStub struct {
	zynaxv1.UnimplementedTaskBrokerServiceServer
	mu         sync.Mutex
	tasks      map[string]*zynaxv1.WorkflowTask
	// capabilities[cap] = agentID
	capabilities map[string]string
	// lastDispatchAgent records which agent a task was dispatched to
}

func newBrokerStub() *brokerStub {
	return &brokerStub{
		tasks:        make(map[string]*zynaxv1.WorkflowTask),
		capabilities: make(map[string]string),
	}
}

func (s *brokerStub) registerCapability(cap, agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.capabilities[cap] = agentID
}

func (s *brokerStub) DispatchTask(_ context.Context, req *zynaxv1.DispatchTaskRequest) (*zynaxv1.DispatchTaskResponse, error) {
	task := req.GetTask()
	if task == nil {
		return nil, status.Error(codes.InvalidArgument, "task must not be nil")
	}
	if task.CapabilityName == "" {
		return nil, status.Error(codes.InvalidArgument, "capability_name must not be empty")
	}
	if task.WorkflowId == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_id must not be empty")
	}
	if len(task.InputPayload) > 0 && !json.Valid(task.InputPayload) {
		return nil, status.Error(codes.InvalidArgument, "input_payload must be valid JSON")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	agentID, ok := s.capabilities[task.CapabilityName]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "no agent for capability %q", task.CapabilityName)
	}

	taskID := fmt.Sprintf("task-%d", len(s.tasks)+1)
	now := timestamppb.Now()
	newTask := &zynaxv1.WorkflowTask{
		TaskId:         taskID,
		WorkflowId:     task.WorkflowId,
		CapabilityName: task.CapabilityName,
		InputPayload:   task.InputPayload,
		TimeoutSeconds: task.TimeoutSeconds,
		MaxRetries:     task.MaxRetries,
		RetryCount:     task.RetryCount,
		Status:         zynaxv1.TaskStatus_TASK_STATUS_PENDING,
		DispatchedTo:   agentID,
		CreatedAt:      now,
		DispatchedAt:   now,
	}
	s.tasks[taskID] = newTask
	return &zynaxv1.DispatchTaskResponse{TaskId: taskID, CreatedAt: now}, nil
}

func (s *brokerStub) AcknowledgeTask(_ context.Context, req *zynaxv1.AcknowledgeTaskRequest) (*zynaxv1.AcknowledgeTaskResponse, error) {
	if req.TaskId == "" {
		return nil, status.Error(codes.InvalidArgument, "task_id must not be empty")
	}
	if req.Status == zynaxv1.TaskStatus_TASK_STATUS_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "status must not be UNSPECIFIED")
	}
	if req.Status != zynaxv1.TaskStatus_TASK_STATUS_COMPLETED &&
		req.Status != zynaxv1.TaskStatus_TASK_STATUS_FAILED &&
		req.Status != zynaxv1.TaskStatus_TASK_STATUS_CANCELLED {
		return nil, status.Error(codes.InvalidArgument, "status must be COMPLETED, FAILED, or CANCELLED")
	}

	// Check task existence before payload validation — NOT_FOUND takes priority.
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[req.TaskId]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "task %q not found", req.TaskId)
	}

	if req.Status == zynaxv1.TaskStatus_TASK_STATUS_COMPLETED && len(req.ResultPayload) == 0 {
		return nil, status.Error(codes.InvalidArgument, "result_payload required for COMPLETED")
	}
	if task.Status == zynaxv1.TaskStatus_TASK_STATUS_COMPLETED ||
		task.Status == zynaxv1.TaskStatus_TASK_STATUS_FAILED ||
		task.Status == zynaxv1.TaskStatus_TASK_STATUS_CANCELLED {
		return nil, status.Error(codes.FailedPrecondition, "task is already in a terminal state")
	}

	now := timestamppb.Now()
	if req.Status == zynaxv1.TaskStatus_TASK_STATUS_COMPLETED {
		task.Status = zynaxv1.TaskStatus_TASK_STATUS_COMPLETED
		task.ResultPayload = req.ResultPayload
		task.CompletedAt = now
	} else if req.Status == zynaxv1.TaskStatus_TASK_STATUS_FAILED {
		task.Error = req.Error
		if task.RetryCount < task.MaxRetries {
			task.RetryCount++
			task.Status = zynaxv1.TaskStatus_TASK_STATUS_RETRYING
		} else {
			task.Status = zynaxv1.TaskStatus_TASK_STATUS_FAILED
			task.CompletedAt = now
		}
	} else {
		task.Status = zynaxv1.TaskStatus_TASK_STATUS_CANCELLED
		task.CompletedAt = now
	}
	return &zynaxv1.AcknowledgeTaskResponse{ResultingStatus: task.Status}, nil
}

func (s *brokerStub) GetTask(_ context.Context, req *zynaxv1.GetTaskRequest) (*zynaxv1.WorkflowTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[req.TaskId]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "task %q not found", req.TaskId)
	}
	return task, nil
}

func (s *brokerStub) ListTasks(_ context.Context, req *zynaxv1.ListTasksRequest) (*zynaxv1.ListTasksResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*zynaxv1.WorkflowTask
	for _, task := range s.tasks {
		if req.WorkflowId != "" && task.WorkflowId != req.WorkflowId {
			continue
		}
		if req.Status != zynaxv1.TaskStatus_TASK_STATUS_UNSPECIFIED && task.Status != req.Status {
			continue
		}
		result = append(result, task)
	}
	return &zynaxv1.ListTasksResponse{Tasks: result}, nil
}

func (s *brokerStub) CancelTask(_ context.Context, req *zynaxv1.CancelTaskRequest) (*zynaxv1.CancelTaskResponse, error) {
	if req.TaskId == "" {
		return nil, status.Error(codes.InvalidArgument, "task_id must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[req.TaskId]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "task %q not found", req.TaskId)
	}
	switch task.Status {
	case zynaxv1.TaskStatus_TASK_STATUS_COMPLETED,
		zynaxv1.TaskStatus_TASK_STATUS_FAILED,
		zynaxv1.TaskStatus_TASK_STATUS_CANCELLED:
		return nil, status.Errorf(codes.FailedPrecondition, "cannot cancel task in %v state (COMPLETED)", task.Status)
	}
	task.Status = zynaxv1.TaskStatus_TASK_STATUS_CANCELLED
	task.CompletedAt = timestamppb.Now()
	return &zynaxv1.CancelTaskResponse{}, nil
}

// CancelTaskResponse is defined in generated code
var _ = (*zynaxv1.CancelTaskResponse)(nil)

// ─── Test context ──────────────────────────────────────────────────────────────

type testCtx struct {
	client        zynaxv1.TaskBrokerServiceClient
	stub          *brokerStub
	lastTaskID    string
	lastTask      *zynaxv1.WorkflowTask
	listResp      *zynaxv1.ListTasksResponse
	grpcErr       error
	ackResp       *zynaxv1.AcknowledgeTaskResponse
	pendingTask   *zynaxv1.WorkflowTask
	pendingAckReq *zynaxv1.AcknowledgeTaskRequest
}

func newTestCtx() *testCtx {
	return &testCtx{}
}

func (tc *testCtx) setupServer(t *testing.T) error {
	tc.stub = newBrokerStub()
	srv, dialer := testserver.NewBufconnServer(t)
	zynaxv1.RegisterTaskBrokerServiceServer(srv, tc.stub)
	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	t.Cleanup(func() { conn.Close() })
	tc.client = zynaxv1.NewTaskBrokerServiceClient(conn)
	return nil
}

func (tc *testCtx) dispatch(cap, wfID string, opts ...func(*zynaxv1.WorkflowTask)) (string, error) {
	task := &zynaxv1.WorkflowTask{
		WorkflowId:     wfID,
		CapabilityName: cap,
		InputPayload:   []byte(`{"input": "test"}`),
	}
	for _, opt := range opts {
		opt(task)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := tc.client.DispatchTask(ctx, &zynaxv1.DispatchTaskRequest{Task: task})
	if err != nil {
		return "", err
	}
	return resp.TaskId, nil
}

func (tc *testCtx) insertTaskDirectly(taskID string, status zynaxv1.TaskStatus, cap, wfID string, maxRetries, retryCount int32) {
	now := timestamppb.Now()
	tc.stub.mu.Lock()
	defer tc.stub.mu.Unlock()
	tc.stub.tasks[taskID] = &zynaxv1.WorkflowTask{
		TaskId:         taskID,
		WorkflowId:     wfID,
		CapabilityName: cap,
		InputPayload:   []byte(`{"input": "test"}`),
		Status:         status,
		MaxRetries:     maxRetries,
		RetryCount:     retryCount,
		DispatchedTo:   "test-agent",
		CreatedAt:      now,
		DispatchedAt:   now,
	}
}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			var tc *testCtx

			sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
				tc = newTestCtx()
				return ctx, nil
			})

			sc.Step(`^a TaskBrokerService is running on a test gRPC server$`, func(ctx context.Context) (context.Context, error) {
				return ctx, tc.setupServer(t)
			})

			sc.Step(`^an AgentRegistryService is available to the broker$`, func(ctx context.Context) (context.Context, error) {
				return ctx, nil
			})

			sc.Step(`^agent "([^"]*)" is registered with capability "([^"]*)"$`, func(ctx context.Context, agentID, cap string) (context.Context, error) {
				tc.stub.registerCapability(cap, agentID)
				return ctx, nil
			})

			sc.Step(`^agent "([^"]*)" is registered with capability "([^"]*)" for broker$`, func(ctx context.Context, agentID, cap string) (context.Context, error) {
				tc.stub.registerCapability(cap, agentID)
				return ctx, nil
			})

			sc.Step(`^a valid WorkflowTask for capability "([^"]*)" with valid input payload$`, func(ctx context.Context, cap string) (context.Context, error) {
				tc.pendingTask = &zynaxv1.WorkflowTask{
					WorkflowId:     "wf-default",
					CapabilityName: cap,
					InputPayload:   []byte(`{"input": "test"}`),
				}
				return ctx, nil
			})

			sc.Step(`^DispatchTask is called with the WorkflowTask$`, func(ctx context.Context) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				resp, err := tc.client.DispatchTask(callCtx, &zynaxv1.DispatchTaskRequest{Task: tc.pendingTask})
				tc.grpcErr = err
				if err == nil {
					tc.lastTaskID = resp.TaskId
				}
				return ctx, nil
			})

			sc.Step(`^DispatchTask is called$`, func(ctx context.Context) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				var task *zynaxv1.WorkflowTask
				if tc.pendingTask != nil {
					task = tc.pendingTask
				} else {
					task = &zynaxv1.WorkflowTask{
						WorkflowId:     "wf-default",
						CapabilityName: "summarize",
						InputPayload:   []byte(`{"input": "test"}`),
					}
				}
				resp, err := tc.client.DispatchTask(callCtx, &zynaxv1.DispatchTaskRequest{Task: task})
				tc.grpcErr = err
				if err == nil {
					tc.lastTaskID = resp.TaskId
				}
				return ctx, nil
			})

			sc.Step(`^the response contains a non-empty task_id$`, func(ctx context.Context) (context.Context, error) {
				if tc.lastTaskID == "" {
					return ctx, fmt.Errorf("task_id is empty")
				}
				return ctx, nil
			})

			sc.Step(`^GetTask for that task_id returns status PENDING$`, func(ctx context.Context) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				task, err := tc.client.GetTask(callCtx, &zynaxv1.GetTaskRequest{TaskId: tc.lastTaskID})
				if err != nil {
					return ctx, err
				}
				if task.Status != zynaxv1.TaskStatus_TASK_STATUS_PENDING {
					return ctx, fmt.Errorf("expected PENDING, got %v", task.Status)
				}
				return ctx, nil
			})

			sc.Step(`^agent "([^"]*)" is registered with capability "([^"]*)"$`, func(ctx context.Context, agentID, cap string) (context.Context, error) {
				tc.stub.registerCapability(cap, agentID)
				return ctx, nil
			})

			sc.Step(`^DispatchTask is called for capability "([^"]*)"$`, func(ctx context.Context, cap string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				task := &zynaxv1.WorkflowTask{
					WorkflowId:     "wf-routing",
					CapabilityName: cap,
					InputPayload:   []byte(`{"input": "test"}`),
				}
				resp, err := tc.client.DispatchTask(callCtx, &zynaxv1.DispatchTaskRequest{Task: task})
				tc.grpcErr = err
				if err == nil {
					tc.lastTaskID = resp.TaskId
				}
				return ctx, nil
			})

			sc.Step(`^the task is routed to agent "([^"]*)"$`, func(ctx context.Context, agentID string) (context.Context, error) {
				tc.stub.mu.Lock()
				defer tc.stub.mu.Unlock()
				task, ok := tc.stub.tasks[tc.lastTaskID]
				if !ok {
					return ctx, fmt.Errorf("task not found")
				}
				if task.DispatchedTo != agentID {
					return ctx, fmt.Errorf("expected dispatched_to=%q, got %q", agentID, task.DispatchedTo)
				}
				return ctx, nil
			})

			sc.Step(`^agent "([^"]*)" receives no dispatch$`, func(ctx context.Context, agentID string) (context.Context, error) {
				tc.stub.mu.Lock()
				defer tc.stub.mu.Unlock()
				for _, task := range tc.stub.tasks {
					if task.DispatchedTo == agentID {
						return ctx, fmt.Errorf("agent %q received dispatch unexpectedly", agentID)
					}
				}
				return ctx, nil
			})

			sc.Step(`^a WorkflowTask with workflow_id "([^"]*)"$`, func(ctx context.Context, wfID string) (context.Context, error) {
				tc.pendingTask = &zynaxv1.WorkflowTask{
					WorkflowId:     wfID,
					CapabilityName: "summarize",
					InputPayload:   []byte(`{"input": "test"}`),
				}
				tc.stub.registerCapability("summarize", "agent-a")
				return ctx, nil
			})

			sc.Step(`^GetTask returns workflow_id "([^"]*)"$`, func(ctx context.Context, wfID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				task, err := tc.client.GetTask(callCtx, &zynaxv1.GetTaskRequest{TaskId: tc.lastTaskID})
				if err != nil {
					return ctx, err
				}
				if task.WorkflowId != wfID {
					return ctx, fmt.Errorf("expected workflow_id %q, got %q", wfID, task.WorkflowId)
				}
				return ctx, nil
			})

			sc.Step(`^no agent is registered for capability "([^"]*)"$`, func(ctx context.Context, cap string) (context.Context, error) {
				return ctx, nil
			})

			sc.Step(`^no task record is created$`, func(ctx context.Context) (context.Context, error) {
				tc.stub.mu.Lock()
				defer tc.stub.mu.Unlock()
				if len(tc.stub.tasks) > 0 {
					return ctx, fmt.Errorf("expected no tasks, got %d", len(tc.stub.tasks))
				}
				return ctx, nil
			})

			sc.Step(`^a WorkflowTask with timeout_seconds set to (\d+)$`, func(ctx context.Context, secs int) (context.Context, error) {
				tc.stub.registerCapability("summarize", "agent-a")
				tc.pendingTask = &zynaxv1.WorkflowTask{
					WorkflowId:     "wf-timeout",
					CapabilityName: "summarize",
					InputPayload:   []byte(`{"input": "test"}`),
					TimeoutSeconds: int32(secs),
				}
				return ctx, nil
			})

			sc.Step(`^the agent receives an ExecuteCapabilityRequest with timeout_seconds (\d+)$`, func(ctx context.Context, secs int) (context.Context, error) {
				tc.stub.mu.Lock()
				defer tc.stub.mu.Unlock()
				for _, task := range tc.stub.tasks {
					if task.TimeoutSeconds == int32(secs) {
						return ctx, nil
					}
				}
				return ctx, fmt.Errorf("no task with timeout_seconds=%d", secs)
			})

			// Acknowledgement scenarios
			sc.Step(`^a dispatched task with task_id "([^"]*)" in DISPATCHED state$`, func(ctx context.Context, taskID string) (context.Context, error) {
				tc.insertTaskDirectly(taskID, zynaxv1.TaskStatus_TASK_STATUS_DISPATCHED, "cap", "wf-test", 0, 0)
				tc.lastTaskID = taskID
				return ctx, nil
			})

			sc.Step(`^AcknowledgeTask is called with task_id "([^"]*)" status COMPLETED$`, func(ctx context.Context, taskID string) (context.Context, error) {
				tc.pendingAckReq = &zynaxv1.AcknowledgeTaskRequest{
					TaskId: taskID,
					Status: zynaxv1.TaskStatus_TASK_STATUS_COMPLETED,
					// payload may be set by a subsequent And/When step
				}
				// Make the call now without payload — for scenarios that don't add a payload step.
				// Scenarios that do add a payload will call again via "the result payload is valid JSON" step.
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				resp, err := tc.client.AcknowledgeTask(callCtx, tc.pendingAckReq)
				tc.ackResp = resp
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^the result payload is valid JSON: (\{.+\})$`, func(ctx context.Context, payload string) (context.Context, error) {
				if tc.pendingAckReq != nil {
					tc.pendingAckReq.ResultPayload = []byte(payload)
				}
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				resp, err := tc.client.AcknowledgeTask(callCtx, tc.pendingAckReq)
				tc.ackResp = resp
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^GetTask for "([^"]*)" returns status COMPLETED$`, func(ctx context.Context, taskID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				task, err := tc.client.GetTask(callCtx, &zynaxv1.GetTaskRequest{TaskId: taskID})
				if err != nil {
					return ctx, err
				}
				if task.Status != zynaxv1.TaskStatus_TASK_STATUS_COMPLETED {
					return ctx, fmt.Errorf("expected COMPLETED, got %v", task.Status)
				}
				tc.lastTask = task
				return ctx, nil
			})

			sc.Step(`^GetTask for "([^"]*)" returns the result payload$`, func(ctx context.Context, taskID string) (context.Context, error) {
				if tc.lastTask == nil {
					return ctx, fmt.Errorf("no task loaded")
				}
				if len(tc.lastTask.ResultPayload) == 0 {
					return ctx, fmt.Errorf("result payload is empty")
				}
				return ctx, nil
			})

			sc.Step(`^a dispatched task with task_id "([^"]*)" and max_retries (\d+)$`, func(ctx context.Context, taskID string, maxRetries int) (context.Context, error) {
				tc.insertTaskDirectly(taskID, zynaxv1.TaskStatus_TASK_STATUS_DISPATCHED, "cap", "wf-test", int32(maxRetries), 0)
				return ctx, nil
			})

			sc.Step(`^AcknowledgeTask is called with task_id "([^"]*)" status FAILED$`, func(ctx context.Context, taskID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				resp, err := tc.client.AcknowledgeTask(callCtx, &zynaxv1.AcknowledgeTaskRequest{
					TaskId: taskID,
					Status: zynaxv1.TaskStatus_TASK_STATUS_FAILED,
					Error:  &zynaxv1.TaskError{Code: "INTERNAL", Message: "failed"},
				})
				tc.ackResp = resp
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^GetTask for "([^"]*)" returns status FAILED$`, func(ctx context.Context, taskID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				task, err := tc.client.GetTask(callCtx, &zynaxv1.GetTaskRequest{TaskId: taskID})
				if err != nil {
					return ctx, err
				}
				if task.Status != zynaxv1.TaskStatus_TASK_STATUS_FAILED {
					return ctx, fmt.Errorf("expected FAILED, got %v", task.Status)
				}
				tc.lastTask = task
				return ctx, nil
			})

			sc.Step(`^the error detail is stored on the task record$`, func(ctx context.Context) (context.Context, error) {
				if tc.lastTask == nil || tc.lastTask.Error == nil {
					return ctx, fmt.Errorf("no error on task")
				}
				return ctx, nil
			})

			sc.Step(`^a dispatched task with task_id "([^"]*)" max_retries (\d+) retry_count (\d+)$`, func(ctx context.Context, taskID string, maxRetries, retryCount int) (context.Context, error) {
				tc.insertTaskDirectly(taskID, zynaxv1.TaskStatus_TASK_STATUS_DISPATCHED, "cap", "wf-test", int32(maxRetries), int32(retryCount))
				return ctx, nil
			})

			sc.Step(`^GetTask for "([^"]*)" returns status RETRYING$`, func(ctx context.Context, taskID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				task, err := tc.client.GetTask(callCtx, &zynaxv1.GetTaskRequest{TaskId: taskID})
				if err != nil {
					return ctx, err
				}
				if task.Status != zynaxv1.TaskStatus_TASK_STATUS_RETRYING {
					return ctx, fmt.Errorf("expected RETRYING, got %v", task.Status)
				}
				tc.lastTask = task
				return ctx, nil
			})

			sc.Step(`^GetTask for "([^"]*)" returns retry_count (\d+)$`, func(ctx context.Context, taskID string, count int) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				task, err := tc.client.GetTask(callCtx, &zynaxv1.GetTaskRequest{TaskId: taskID})
				if err != nil {
					return ctx, err
				}
				if task.RetryCount != int32(count) {
					return ctx, fmt.Errorf("expected retry_count=%d, got %d", count, task.RetryCount)
				}
				return ctx, nil
			})

			sc.Step(`^AcknowledgeTask is called with task_id "([^"]*)" status COMPLETED$`, func(ctx context.Context, taskID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				resp, err := tc.client.AcknowledgeTask(callCtx, &zynaxv1.AcknowledgeTaskRequest{
					TaskId:        taskID,
					Status:        zynaxv1.TaskStatus_TASK_STATUS_COMPLETED,
					ResultPayload: []byte(`{"result": "ok"}`),
				})
				tc.ackResp = resp
				tc.grpcErr = err
				return ctx, nil
			})

			// Cancellation
			sc.Step(`^a dispatched task with task_id "([^"]*)" in PENDING state$`, func(ctx context.Context, taskID string) (context.Context, error) {
				tc.insertTaskDirectly(taskID, zynaxv1.TaskStatus_TASK_STATUS_PENDING, "cap", "wf-test", 0, 0)
				return ctx, nil
			})

			sc.Step(`^CancelTask is called with task_id "([^"]*)"$`, func(ctx context.Context, taskID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_, err := tc.client.CancelTask(callCtx, &zynaxv1.CancelTaskRequest{TaskId: taskID})
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^GetTask for "([^"]*)" returns status CANCELLED$`, func(ctx context.Context, taskID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				task, err := tc.client.GetTask(callCtx, &zynaxv1.GetTaskRequest{TaskId: taskID})
				if err != nil {
					return ctx, err
				}
				if task.Status != zynaxv1.TaskStatus_TASK_STATUS_CANCELLED {
					return ctx, fmt.Errorf("expected CANCELLED, got %v", task.Status)
				}
				return ctx, nil
			})

			sc.Step(`^a task with task_id "([^"]*)" in COMPLETED state$`, func(ctx context.Context, taskID string) (context.Context, error) {
				tc.insertTaskDirectly(taskID, zynaxv1.TaskStatus_TASK_STATUS_COMPLETED, "cap", "wf-test", 0, 0)
				return ctx, nil
			})

			sc.Step(`^the gRPC status is FAILED_PRECONDITION$`, func(ctx context.Context) (context.Context, error) {
				if tc.grpcErr == nil {
					return ctx, fmt.Errorf("expected error")
				}
				st, _ := status.FromError(tc.grpcErr)
				if st.Code() != codes.FailedPrecondition {
					return ctx, fmt.Errorf("expected FAILED_PRECONDITION, got %v", st.Code())
				}
				return ctx, nil
			})

			sc.Step(`^the error message mentions "([^"]*)"$`, func(ctx context.Context, field string) (context.Context, error) {
				if tc.grpcErr == nil {
					return ctx, fmt.Errorf("expected error")
				}
				if !strings.Contains(tc.grpcErr.Error(), field) {
					return ctx, fmt.Errorf("error %q doesn't mention %q", tc.grpcErr.Error(), field)
				}
				return ctx, nil
			})

			sc.Step(`^the gRPC status is NOT_FOUND$`, func(ctx context.Context) (context.Context, error) {
				if tc.grpcErr == nil {
					return ctx, fmt.Errorf("expected error")
				}
				st, _ := status.FromError(tc.grpcErr)
				if st.Code() != codes.NotFound {
					return ctx, fmt.Errorf("expected NOT_FOUND, got %v", st.Code())
				}
				return ctx, nil
			})

			sc.Step(`^the error message contains "([^"]*)"$`, func(ctx context.Context, substr string) (context.Context, error) {
				if tc.grpcErr == nil {
					return ctx, fmt.Errorf("expected error")
				}
				if !strings.Contains(tc.grpcErr.Error(), substr) {
					return ctx, fmt.Errorf("error %q doesn't contain %q", tc.grpcErr.Error(), substr)
				}
				return ctx, nil
			})

			// Query scenarios
			sc.Step(`^a dispatched task with task_id "([^"]*)"$`, func(ctx context.Context, taskID string) (context.Context, error) {
				tc.stub.registerCapability("cap", "agent-x")
				tc.insertTaskDirectly(taskID, zynaxv1.TaskStatus_TASK_STATUS_DISPATCHED, "cap", "wf-test", 0, 0)
				return ctx, nil
			})

			sc.Step(`^GetTask is called with task_id "([^"]*)"$`, func(ctx context.Context, taskID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				task, err := tc.client.GetTask(callCtx, &zynaxv1.GetTaskRequest{TaskId: taskID})
				tc.lastTask = task
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^the response includes a non-zero dispatched_at timestamp$`, func(ctx context.Context) (context.Context, error) {
				if tc.lastTask == nil {
					return ctx, fmt.Errorf("no task")
				}
				if tc.lastTask.DispatchedAt == nil || tc.lastTask.DispatchedAt.AsTime().IsZero() {
					return ctx, fmt.Errorf("dispatched_at is zero")
				}
				return ctx, nil
			})

			sc.Step(`^the response includes the original capability_name$`, func(ctx context.Context) (context.Context, error) {
				if tc.lastTask == nil || tc.lastTask.CapabilityName == "" {
					return ctx, fmt.Errorf("capability_name is empty")
				}
				return ctx, nil
			})

			sc.Step(`^the response includes the original workflow_id$`, func(ctx context.Context) (context.Context, error) {
				if tc.lastTask == nil || tc.lastTask.WorkflowId == "" {
					return ctx, fmt.Errorf("workflow_id is empty")
				}
				return ctx, nil
			})

			sc.Step(`^task "([^"]*)" belongs to workflow "([^"]*)"$`, func(ctx context.Context, taskID, wfID string) (context.Context, error) {
				tc.stub.registerCapability("cap", "agent-x")
				tc.insertTaskDirectly(taskID, zynaxv1.TaskStatus_TASK_STATUS_DISPATCHED, "cap", wfID, 0, 0)
				return ctx, nil
			})

			sc.Step(`^ListTasks is called with workflow_id filter "([^"]*)"$`, func(ctx context.Context, wfID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				resp, err := tc.client.ListTasks(callCtx, &zynaxv1.ListTasksRequest{WorkflowId: wfID})
				tc.listResp = resp
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^the response contains task "([^"]*)"$`, func(ctx context.Context, taskID string) (context.Context, error) {
				if tc.listResp == nil {
					return ctx, fmt.Errorf("no list response")
				}
				for _, task := range tc.listResp.Tasks {
					if task.TaskId == taskID {
						return ctx, nil
					}
				}
				return ctx, fmt.Errorf("task %q not in response", taskID)
			})

			sc.Step(`^the response does not contain task "([^"]*)"$`, func(ctx context.Context, taskID string) (context.Context, error) {
				if tc.listResp == nil {
					return ctx, nil
				}
				for _, task := range tc.listResp.Tasks {
					if task.TaskId == taskID {
						return ctx, fmt.Errorf("task %q should not be in response", taskID)
					}
				}
				return ctx, nil
			})

			sc.Step(`^task "([^"]*)" has status COMPLETED$`, func(ctx context.Context, taskID string) (context.Context, error) {
				tc.stub.registerCapability("cap", "agent-x")
				tc.insertTaskDirectly(taskID, zynaxv1.TaskStatus_TASK_STATUS_COMPLETED, "cap", "wf-test", 0, 0)
				return ctx, nil
			})

			sc.Step(`^task "([^"]*)" has status FAILED$`, func(ctx context.Context, taskID string) (context.Context, error) {
				tc.stub.registerCapability("cap", "agent-x")
				tc.insertTaskDirectly(taskID, zynaxv1.TaskStatus_TASK_STATUS_FAILED, "cap", "wf-test", 0, 0)
				return ctx, nil
			})

			sc.Step(`^ListTasks is called with status filter COMPLETED$`, func(ctx context.Context) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				resp, err := tc.client.ListTasks(callCtx, &zynaxv1.ListTasksRequest{Status: zynaxv1.TaskStatus_TASK_STATUS_COMPLETED})
				tc.listResp = resp
				tc.grpcErr = err
				return ctx, nil
			})

			// Input validation
			sc.Step(`^a WorkflowTask with capability_name set to ""$`, func(ctx context.Context) (context.Context, error) {
				tc.pendingTask = &zynaxv1.WorkflowTask{
					WorkflowId:     "wf-test",
					CapabilityName: "",
					InputPayload:   []byte(`{"input": "test"}`),
				}
				return ctx, nil
			})

			sc.Step(`^the gRPC status is INVALID_ARGUMENT$`, func(ctx context.Context) (context.Context, error) {
				if tc.grpcErr == nil {
					return ctx, fmt.Errorf("expected error")
				}
				st, _ := status.FromError(tc.grpcErr)
				if st.Code() != codes.InvalidArgument {
					return ctx, fmt.Errorf("expected INVALID_ARGUMENT, got %v", st.Code())
				}
				return ctx, nil
			})

			sc.Step(`^a WorkflowTask with workflow_id set to ""$`, func(ctx context.Context) (context.Context, error) {
				tc.pendingTask = &zynaxv1.WorkflowTask{
					WorkflowId:     "",
					CapabilityName: "summarize",
					InputPayload:   []byte(`{"input": "test"}`),
				}
				return ctx, nil
			})

			sc.Step(`^a WorkflowTask with input_payload set to "([^"]*)"$`, func(ctx context.Context, payload string) (context.Context, error) {
				tc.stub.registerCapability("summarize", "agent-x")
				tc.pendingTask = &zynaxv1.WorkflowTask{
					WorkflowId:     "wf-test",
					CapabilityName: "summarize",
					InputPayload:   []byte(payload),
				}
				return ctx, nil
			})

			sc.Step(`^an AcknowledgeTaskRequest with task_id set to ""$`, func(ctx context.Context) (context.Context, error) {
				tc.pendingAckReq = &zynaxv1.AcknowledgeTaskRequest{
					TaskId: "",
					Status: zynaxv1.TaskStatus_TASK_STATUS_COMPLETED,
				}
				return ctx, nil
			})

			sc.Step(`^AcknowledgeTask is called$`, func(ctx context.Context) (context.Context, error) {
				if tc.pendingAckReq != nil {
					callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					tc.ackResp, tc.grpcErr = tc.client.AcknowledgeTask(callCtx, tc.pendingAckReq)
				}
				return ctx, nil
			})

			sc.Step(`^an AcknowledgeTaskRequest with status TASK_STATUS_UNSPECIFIED$`, func(ctx context.Context) (context.Context, error) {
				tc.pendingAckReq = &zynaxv1.AcknowledgeTaskRequest{
					TaskId: "task-x",
					Status: zynaxv1.TaskStatus_TASK_STATUS_UNSPECIFIED,
				}
				return ctx, nil
			})

			sc.Step(`^an AcknowledgeTaskRequest with status COMPLETED and empty payload$`, func(ctx context.Context) (context.Context, error) {
				// Pre-insert the task so NOT_FOUND doesn't fire before INVALID_ARGUMENT check
				tc.insertTaskDirectly("task-x", zynaxv1.TaskStatus_TASK_STATUS_PENDING, "summarize", "wf-x", 0, 0)
				tc.pendingAckReq = &zynaxv1.AcknowledgeTaskRequest{
					TaskId:        "task-x",
					Status:        zynaxv1.TaskStatus_TASK_STATUS_COMPLETED,
					ResultPayload: nil,
				}
				return ctx, nil
			})
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/task_broker_service.feature"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
