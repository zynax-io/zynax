// SPDX-License-Identifier: Apache-2.0

// Package domain contains the core port definitions and value types for the
// engine-adapter service. It has zero imports from api or infrastructure layers.
package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
)

// blockingBroker blocks DispatchTask until the context is cancelled, simulating
// a hanging task-broker to exercise the grpcCallTimeout deadline.
type blockingBroker struct{}

func (b *blockingBroker) DispatchTask(ctx context.Context, _ *zynaxv1.DispatchTaskRequest, _ ...grpc.CallOption) (*zynaxv1.DispatchTaskResponse, error) {
	<-ctx.Done()
	return nil, fmt.Errorf("blocking broker: %w", ctx.Err())
}

func (b *blockingBroker) GetTask(_ context.Context, _ *zynaxv1.GetTaskRequest, _ ...grpc.CallOption) (*zynaxv1.WorkflowTask, error) {
	return nil, nil
}
func (b *blockingBroker) AcknowledgeTask(_ context.Context, _ *zynaxv1.AcknowledgeTaskRequest, _ ...grpc.CallOption) (*zynaxv1.AcknowledgeTaskResponse, error) {
	return nil, nil
}
func (b *blockingBroker) CancelTask(_ context.Context, _ *zynaxv1.CancelTaskRequest, _ ...grpc.CallOption) (*zynaxv1.CancelTaskResponse, error) {
	return nil, nil
}
func (b *blockingBroker) ListTasks(_ context.Context, _ *zynaxv1.ListTasksRequest, _ ...grpc.CallOption) (*zynaxv1.ListTasksResponse, error) {
	return nil, nil
}

// stubBroker is a hand-written mock of TaskBrokerServiceClient.
// It is unexported; it covers only the two methods used by CapabilityDispatcher.
type stubBroker struct {
	dispatchResp *zynaxv1.DispatchTaskResponse
	dispatchErr  error

	// tasks is a sequence of responses returned by successive GetTask calls.
	tasks   []*zynaxv1.WorkflowTask
	taskErr error
	callIdx int
}

func (s *stubBroker) DispatchTask(_ context.Context, _ *zynaxv1.DispatchTaskRequest, _ ...grpc.CallOption) (*zynaxv1.DispatchTaskResponse, error) {
	return s.dispatchResp, s.dispatchErr
}

func (s *stubBroker) GetTask(_ context.Context, _ *zynaxv1.GetTaskRequest, _ ...grpc.CallOption) (*zynaxv1.WorkflowTask, error) {
	if s.taskErr != nil {
		return nil, s.taskErr
	}
	if s.callIdx >= len(s.tasks) {
		return s.tasks[len(s.tasks)-1], nil
	}
	t := s.tasks[s.callIdx]
	s.callIdx++
	return t, nil
}

// Remaining TaskBrokerServiceClient methods — not used; satisfy the interface.
func (s *stubBroker) AcknowledgeTask(_ context.Context, _ *zynaxv1.AcknowledgeTaskRequest, _ ...grpc.CallOption) (*zynaxv1.AcknowledgeTaskResponse, error) {
	return nil, nil
}
func (s *stubBroker) CancelTask(_ context.Context, _ *zynaxv1.CancelTaskRequest, _ ...grpc.CallOption) (*zynaxv1.CancelTaskResponse, error) {
	return nil, nil
}
func (s *stubBroker) ListTasks(_ context.Context, _ *zynaxv1.ListTasksRequest, _ ...grpc.CallOption) (*zynaxv1.ListTasksResponse, error) {
	return nil, nil
}

func newDispatcher(broker *stubBroker) *CapabilityDispatcher {
	d := NewCapabilityDispatcher(broker, 30*time.Second)
	d.pollInterval = time.Millisecond
	return d
}

func TestDispatchCapabilityActivity_SuccessWithEvent(t *testing.T) {
	broker := &stubBroker{
		dispatchResp: &zynaxv1.DispatchTaskResponse{TaskId: "t1"},
		tasks: []*zynaxv1.WorkflowTask{
			{Status: zynaxv1.TaskStatus_TASK_STATUS_DISPATCHED},
			{
				Status:        zynaxv1.TaskStatus_TASK_STATUS_COMPLETED,
				ResultPayload: []byte(`{"_event":"review.approved","data":"ok"}`),
			},
		},
	}
	d := newDispatcher(broker)

	result, err := d.DispatchCapabilityActivity(context.Background(), ActivityInput{
		CapabilityName: "summarize",
		WorkflowID:     "wf-1",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EventType != "review.approved" {
		t.Errorf("EventType = %q; want %q", result.EventType, "review.approved")
	}
}

func TestDispatchCapabilityActivity_SuccessNoEvent(t *testing.T) {
	broker := &stubBroker{
		dispatchResp: &zynaxv1.DispatchTaskResponse{TaskId: "t2"},
		tasks: []*zynaxv1.WorkflowTask{
			{
				Status:        zynaxv1.TaskStatus_TASK_STATUS_COMPLETED,
				ResultPayload: []byte(`{"data":"result"}`),
			},
		},
	}
	d := newDispatcher(broker)

	result, err := d.DispatchCapabilityActivity(context.Background(), ActivityInput{
		CapabilityName: "summarize",
		WorkflowID:     "wf-2",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EventType != "summarize.completed" {
		t.Errorf("EventType = %q; want %q", result.EventType, "summarize.completed")
	}
}

func TestDispatchCapabilityActivity_Failed(t *testing.T) {
	broker := &stubBroker{
		dispatchResp: &zynaxv1.DispatchTaskResponse{TaskId: "t3"},
		tasks: []*zynaxv1.WorkflowTask{
			{
				Status: zynaxv1.TaskStatus_TASK_STATUS_FAILED,
				Error:  &zynaxv1.TaskError{Message: "agent crashed"},
			},
		},
	}
	d := newDispatcher(broker)

	_, err := d.DispatchCapabilityActivity(context.Background(), ActivityInput{
		CapabilityName: "summarize",
		WorkflowID:     "wf-3",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "agent crashed") {
		t.Errorf("error %q does not mention agent error message", err.Error())
	}
}

func TestDispatchCapabilityActivity_DispatchError(t *testing.T) {
	broker := &stubBroker{
		dispatchErr: errors.New("broker unavailable"),
	}
	d := newDispatcher(broker)

	_, err := d.DispatchCapabilityActivity(context.Background(), ActivityInput{
		CapabilityName: "summarize",
		WorkflowID:     "wf-4",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "broker unavailable") {
		t.Errorf("error %q does not mention broker error", err.Error())
	}
}

func TestDispatchCapabilityActivity_ContextCancelled(t *testing.T) {
	broker := &stubBroker{
		dispatchResp: &zynaxv1.DispatchTaskResponse{TaskId: "t5"},
		tasks: []*zynaxv1.WorkflowTask{
			{Status: zynaxv1.TaskStatus_TASK_STATUS_PENDING},
		},
	}
	d := newDispatcher(broker)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := d.DispatchCapabilityActivity(ctx, ActivityInput{
		CapabilityName: "summarize",
		WorkflowID:     "wf-5",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled in chain, got: %v", err)
	}
}

func TestExtractResult_EmptyPayload(t *testing.T) {
	result, err := extractResult(nil, "classify")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EventType != "classify.completed" {
		t.Errorf("EventType = %q; want %q", result.EventType, "classify.completed")
	}
}

func TestExtractResult_EmptyEventString(t *testing.T) {
	result, err := extractResult([]byte(`{"_event":""}`), "classify")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EventType != "classify.completed" {
		t.Errorf("empty _event should fall back to default; got %q", result.EventType)
	}
}

func TestExtractResult_InvalidJSON(t *testing.T) {
	_, err := extractResult([]byte(`not-json`), "classify")
	if err == nil {
		t.Fatal("expected error for invalid JSON payload")
	}
}

func TestHandleStatus_Cancelled(t *testing.T) {
	task := &zynaxv1.WorkflowTask{Status: zynaxv1.TaskStatus_TASK_STATUS_CANCELLED}
	_, done, err := handleStatus(task, "cap")
	if !done {
		t.Error("CANCELLED should be terminal (done=true)")
	}
	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("expected cancelled error, got %v", err)
	}
}

func TestDispatch_BrokerTimeout(t *testing.T) {
	d := NewCapabilityDispatcher(&blockingBroker{}, 50*time.Millisecond)
	_, err := d.DispatchCapabilityActivity(context.Background(), ActivityInput{
		CapabilityName: "summarize",
		WorkflowID:     "wf-timeout",
	})
	if err == nil {
		t.Fatal("expected error when broker hangs past deadline")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded in chain, got: %v", err)
	}
}

func TestPoll_GetTaskError(t *testing.T) {
	broker := &stubBroker{
		dispatchResp: &zynaxv1.DispatchTaskResponse{TaskId: "t6"},
		taskErr:      errors.New("broker down"),
	}
	d := newDispatcher(broker)

	_, err := d.DispatchCapabilityActivity(context.Background(), ActivityInput{
		CapabilityName: "classify",
		WorkflowID:     "wf-6",
	})
	if err == nil {
		t.Fatal("expected error from GetTask failure")
	}
	if !strings.Contains(err.Error(), "broker down") {
		t.Errorf("error %q should mention broker down", err.Error())
	}
}
