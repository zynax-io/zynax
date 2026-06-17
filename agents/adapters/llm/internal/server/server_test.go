// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/llm/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/llm/internal/provider"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const chatSchema = `{"type":"object","required":["prompt"],"properties":{"prompt":{"type":"string"}},"additionalProperties":false}`

// stubProvider streams a single token then closes.
type stubProvider struct{}

func (stubProvider) Stream(_ context.Context, _ string) (<-chan provider.Chunk, error) {
	out := make(chan provider.Chunk, 1)
	out <- provider.Chunk{Text: "hi"}
	close(out)
	return out, nil
}

// fakeStream implements AgentService_ExecuteCapabilityServer capturing events.
type fakeStream struct {
	grpc.ServerStream
	ctx    context.Context
	events []*zynaxv1.TaskEvent
}

func (f *fakeStream) Send(ev *zynaxv1.TaskEvent) error {
	f.events = append(f.events, ev)
	return nil
}

func (f *fakeStream) Context() context.Context {
	if f.ctx == nil {
		return context.Background()
	}
	return f.ctx
}

func testServer(t *testing.T) *AgentServer {
	t.Helper()
	cfg := &config.AdapterConfig{
		Capabilities: []config.CapabilityConfig{{
			Name:             "chat_completion",
			Description:      "Stream a chat completion.",
			InputSchemaJSON:  chatSchema,
			OutputSchemaJSON: `{"type":"object","properties":{"completion":{"type":"string"}}}`,
		}},
	}
	s, err := NewAgentServer(cfg, stubProvider{})
	if err != nil {
		t.Fatalf("NewAgentServer: %v", err)
	}
	return s
}

func TestExecuteCapabilityHappyPath(t *testing.T) {
	s := testServer(t)
	st := &fakeStream{}
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t1",
		CapabilityName: "chat_completion",
		InputPayload:   []byte(`{"prompt":"hi"}`),
	}
	if err := s.ExecuteCapability(req, st); err != nil {
		t.Fatalf("ExecuteCapability: %v", err)
	}
	if len(st.events) < 2 {
		t.Fatalf("want PROGRESS+COMPLETED, got %d events", len(st.events))
	}
	if last := st.events[len(st.events)-1]; last.GetEventType() != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
		t.Errorf("final: want COMPLETED, got %v", last.GetEventType())
	}
}

func TestExecuteCapabilityValidation(t *testing.T) {
	s := testServer(t)
	tests := []struct {
		name string
		req  *zynaxv1.ExecuteCapabilityRequest
		code codes.Code
	}{
		{"empty task_id", &zynaxv1.ExecuteCapabilityRequest{CapabilityName: "chat_completion"}, codes.InvalidArgument},
		{"empty capability", &zynaxv1.ExecuteCapabilityRequest{TaskId: "t"}, codes.InvalidArgument},
		{"unknown capability", &zynaxv1.ExecuteCapabilityRequest{TaskId: "t", CapabilityName: "nope"}, codes.NotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &fakeStream{}
			err := s.ExecuteCapability(tt.req, st)
			if status.Code(err) != tt.code {
				t.Errorf("code = %v, want %v", status.Code(err), tt.code)
			}
			if len(st.events) != 0 {
				t.Errorf("no TaskEvent must be emitted, got %d", len(st.events))
			}
		})
	}
}

func TestGetCapabilitySchema(t *testing.T) {
	s := testServer(t)
	resp, err := s.GetCapabilitySchema(context.Background(),
		&zynaxv1.GetCapabilitySchemaRequest{CapabilityName: "chat_completion"})
	if err != nil {
		t.Fatalf("GetCapabilitySchema: %v", err)
	}
	if resp.GetCapabilityName() != "chat_completion" {
		t.Errorf("name = %q", resp.GetCapabilityName())
	}
	if resp.GetInputSchemaJson() == "" || resp.GetOutputSchemaJson() == "" {
		t.Errorf("schemas must be non-empty")
	}
	if resp.GetDescription() == "" {
		t.Errorf("description must be non-empty")
	}
}

func TestGetCapabilitySchemaErrors(t *testing.T) {
	s := testServer(t)
	tests := []struct {
		name string
		req  *zynaxv1.GetCapabilitySchemaRequest
		code codes.Code
	}{
		{"empty name", &zynaxv1.GetCapabilitySchemaRequest{}, codes.InvalidArgument},
		{"unknown", &zynaxv1.GetCapabilitySchemaRequest{CapabilityName: "nope"}, codes.NotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.GetCapabilitySchema(context.Background(), tt.req)
			if status.Code(err) != tt.code {
				t.Errorf("code = %v, want %v", status.Code(err), tt.code)
			}
		})
	}
}

func TestNewAgentServerBadSchema(t *testing.T) {
	cfg := &config.AdapterConfig{
		Capabilities: []config.CapabilityConfig{{Name: "x", InputSchemaJSON: `{bad`}},
	}
	if _, err := NewAgentServer(cfg, stubProvider{}); err == nil {
		t.Fatal("want error for malformed schema")
	}
}
