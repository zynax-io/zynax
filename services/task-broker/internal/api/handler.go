// SPDX-License-Identifier: Apache-2.0

// Package api implements the TaskBrokerService gRPC server handler.
package api

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
)

// Handler implements TaskBrokerServiceServer, delegating all calls to a domain.TaskService.
type Handler struct {
	zynaxv1.UnimplementedTaskBrokerServiceServer
	svc *domain.TaskService
}

// NewHandler constructs a Handler wrapping the given TaskService.
func NewHandler(svc *domain.TaskService) *Handler { return &Handler{svc: svc} }

// DispatchTask validates the WorkflowTask, finds an eligible agent, and returns the broker-assigned task_id.
func (h *Handler) DispatchTask(ctx context.Context, req *zynaxv1.DispatchTaskRequest) (*zynaxv1.DispatchTaskResponse, error) {
	wt := req.GetTask()
	task := &domain.Task{
		WorkflowID:     wt.GetWorkflowId(),
		CapabilityName: wt.GetCapabilityName(),
		InputPayload:   wt.GetInputPayload(),
		TimeoutSeconds: wt.GetTimeoutSeconds(),
		MaxRetries:     wt.GetMaxRetries(),
	}
	taskID, createdAt, err := h.svc.DispatchTask(ctx, task)
	if err != nil {
		return nil, grpcErr(err)
	}
	return &zynaxv1.DispatchTaskResponse{
		TaskId:    taskID,
		CreatedAt: timestamppb.New(createdAt),
	}, nil
}

// AcknowledgeTask records the outcome of a capability execution and applies retry logic.
func (h *Handler) AcknowledgeTask(ctx context.Context, req *zynaxv1.AcknowledgeTaskRequest) (*zynaxv1.AcknowledgeTaskResponse, error) {
	var taskErr *domain.TaskError
	if e := req.GetError(); e != nil {
		taskErr = &domain.TaskError{
			Code:    e.GetCode(),
			Message: e.GetMessage(),
			Details: e.GetDetails(),
		}
	}
	reqStatus := domain.TaskStatus(req.GetStatus()) //nolint:gosec // G115: proto enum values fit in int32
	resulting, err := h.svc.AcknowledgeTask(ctx, req.GetTaskId(), reqStatus, req.GetResultPayload(), taskErr)
	if err != nil {
		return nil, grpcErr(err)
	}
	return &zynaxv1.AcknowledgeTaskResponse{
		ResultingStatus: zynaxv1.TaskStatus(resulting), //nolint:gosec // G115: domain enum mirrors proto enum
	}, nil
}

// GetTask returns the current state of a task by its broker-assigned id.
func (h *Handler) GetTask(ctx context.Context, req *zynaxv1.GetTaskRequest) (*zynaxv1.WorkflowTask, error) {
	task, err := h.svc.GetTask(ctx, req.GetTaskId())
	if err != nil {
		return nil, grpcErr(err)
	}
	return taskToProto(task), nil
}

// ListTasks returns a filtered, paginated page of tasks.
func (h *Handler) ListTasks(ctx context.Context, req *zynaxv1.ListTasksRequest) (*zynaxv1.ListTasksResponse, error) {
	filter := domain.ListFilter{
		WorkflowID: req.GetWorkflowId(),
		Status:     domain.TaskStatus(req.GetStatus()), //nolint:gosec // G115: proto enum mirrors domain enum
		AgentID:    req.GetAgentId(),
		PageToken:  req.GetPageToken(),
		PageSize:   req.GetPageSize(),
	}
	result, err := h.svc.ListTasks(ctx, filter)
	if err != nil {
		return nil, grpcErr(err)
	}
	tasks := make([]*zynaxv1.WorkflowTask, len(result.Tasks))
	for i, t := range result.Tasks {
		tasks[i] = taskToProto(t)
	}
	return &zynaxv1.ListTasksResponse{
		Tasks:         tasks,
		NextPageToken: result.NextPageToken,
	}, nil
}

// CancelTask transitions a non-terminal task to CANCELLED.
func (h *Handler) CancelTask(ctx context.Context, req *zynaxv1.CancelTaskRequest) (*zynaxv1.CancelTaskResponse, error) {
	cancelledAt, err := h.svc.CancelTask(ctx, req.GetTaskId())
	if err != nil {
		return nil, grpcErr(err)
	}
	return &zynaxv1.CancelTaskResponse{
		CancelledAt: timestamppb.New(cancelledAt),
	}, nil
}

func taskToProto(t *domain.Task) *zynaxv1.WorkflowTask {
	wt := &zynaxv1.WorkflowTask{
		TaskId:         t.TaskID,
		WorkflowId:     t.WorkflowID,
		CapabilityName: t.CapabilityName,
		InputPayload:   t.InputPayload,
		TimeoutSeconds: t.TimeoutSeconds,
		MaxRetries:     t.MaxRetries,
		RetryCount:     t.RetryCount,
		Status:         zynaxv1.TaskStatus(t.Status), //nolint:gosec // G115: domain enum mirrors proto enum
		DispatchedTo:   t.DispatchedTo,
		ResultPayload:  t.ResultPayload,
		CreatedAt:      tsOrNil(t.CreatedAt),
		DispatchedAt:   tsOrNil(t.DispatchedAt),
		CompletedAt:    tsOrNil(t.CompletedAt),
	}
	if t.Error != nil {
		wt.Error = &zynaxv1.TaskError{
			Code:    t.Error.Code,
			Message: t.Error.Message,
			Details: t.Error.Details,
		}
	}
	return wt
}

func tsOrNil(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func grpcErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrTaskNotFound) {
		return status.Errorf(codes.NotFound, "%v", err)
	}
	if errors.Is(err, domain.ErrNoEligibleAgent) {
		return status.Errorf(codes.NotFound, "%v", err)
	}
	if errors.Is(err, domain.ErrTaskTerminal) {
		return status.Errorf(codes.FailedPrecondition, "%v", err)
	}
	if errors.Is(err, domain.ErrInvalidArgument) {
		return status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if errors.Is(err, context.Canceled) {
		return status.Errorf(codes.Canceled, "%v", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Errorf(codes.DeadlineExceeded, "%v", err)
	}
	return status.Errorf(codes.Internal, "%v", err)
}
