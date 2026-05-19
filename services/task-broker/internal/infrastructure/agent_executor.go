// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
)

// AgentExecutor implements domain.CapabilityExecutor by dialing agent gRPC endpoints.
type AgentExecutor struct{}

// NewAgentExecutor constructs an AgentExecutor.
func NewAgentExecutor() *AgentExecutor { return &AgentExecutor{} }

// Execute opens a connection to the agent, calls ExecuteCapability, and streams
// TaskEvents until a terminal COMPLETED or FAILED event is received.
func (e *AgentExecutor) Execute(ctx context.Context, agent domain.AgentInfo, task *domain.Task) ([]byte, *domain.TaskError, error) {
	conn, err := grpc.NewClient(agent.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("task-broker: dial agent %q: %w", agent.AgentID, err)
	}
	defer func() { _ = conn.Close() }()

	reqID, err := newRequestID()
	if err != nil {
		return nil, nil, fmt.Errorf("task-broker: generate request ID: %w", err)
	}

	client := zynaxv1.NewAgentServiceClient(conn)
	stream, err := client.ExecuteCapability(ctx, &zynaxv1.ExecuteCapabilityRequest{
		RequestId:      reqID,
		TaskId:         task.TaskID,
		WorkflowId:     task.WorkflowID,
		CapabilityName: task.CapabilityName,
		InputPayload:   task.InputPayload,
		TimeoutSeconds: task.TimeoutSeconds,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("task-broker: agent %q execute: %w", agent.AgentID, err)
	}

	for {
		ev, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, nil, fmt.Errorf("task-broker: agent %q stream: %w", agent.AgentID, err)
		}
		switch ev.GetEventType() {
		case zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED:
			return ev.GetPayload(), nil, nil
		case zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED:
			te := &domain.TaskError{}
			if ce := ev.GetError(); ce != nil {
				te.Code = ce.GetCode()
				te.Message = ce.GetMessage()
				te.Details = ce.GetDetails()
			}
			return nil, te, nil
		}
	}
	return nil, nil, nil
}

func newRequestID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("task-broker: rand.Read: %w", err)
	}
	return hex.EncodeToString(b), nil
}
