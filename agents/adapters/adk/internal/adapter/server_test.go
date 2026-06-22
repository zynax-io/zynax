// SPDX-License-Identifier: Apache-2.0

package adapter

import (
	"context"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/adk/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeStream captures sent TaskEvents. Embedding the generated stream interface
// supplies the gRPC plumbing methods the skeleton never calls.
type fakeStream struct {
	zynaxv1.AgentService_ExecuteCapabilityServer
	events []*zynaxv1.TaskEvent
}

func (f *fakeStream) Send(e *zynaxv1.TaskEvent) error { f.events = append(f.events, e); return nil }
func (f *fakeStream) Context() context.Context        { return context.Background() }

func testServer() *AgentServer {
	return NewAgentServer(&config.AdapterConfig{
		Capabilities: []config.CapabilityConfig{
			{Name: "triage", Description: "classify", InputSchemaJSON: `{"type":"object"}`, OutputSchemaJSON: `{"type":"string"}`},
		},
	})
}

func TestExecuteCapability(t *testing.T) {
	cases := []struct {
		name      string
		req       *zynaxv1.ExecuteCapabilityRequest
		wantGRPC  codes.Code // OK ⇒ expect a terminal event instead of a gRPC error
		wantEvent string     // CapabilityError code on the terminal FAILED event
	}{
		{"empty task_id", &zynaxv1.ExecuteCapabilityRequest{CapabilityName: "triage"}, codes.InvalidArgument, ""},
		{"empty capability", &zynaxv1.ExecuteCapabilityRequest{TaskId: "t1"}, codes.InvalidArgument, ""},
		{"unknown capability", &zynaxv1.ExecuteCapabilityRequest{TaskId: "t1", CapabilityName: "missing"}, codes.OK, "INVALID_INPUT"},
		{"known not wired", &zynaxv1.ExecuteCapabilityRequest{TaskId: "t1", CapabilityName: "triage"}, codes.OK, "UNIMPLEMENTED"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stream := &fakeStream{}
			err := testServer().ExecuteCapability(tc.req, stream)
			if tc.wantGRPC != codes.OK {
				if status.Code(err) != tc.wantGRPC {
					t.Fatalf("code = %v, want %v", status.Code(err), tc.wantGRPC)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(stream.events) != 1 {
				t.Fatalf("expected exactly one terminal event, got %d", len(stream.events))
			}
			ev := stream.events[0]
			if ev.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
				t.Errorf("event_type = %v, want FAILED", ev.EventType)
			}
			if ev.Error.GetCode() != tc.wantEvent {
				t.Errorf("code = %q, want %q", ev.Error.GetCode(), tc.wantEvent)
			}
			if ev.TaskId != "t1" || ev.Timestamp == nil {
				t.Errorf("task_id/timestamp not populated: %+v", ev)
			}
		})
	}
}

func TestGetCapabilitySchema(t *testing.T) {
	resp, err := testServer().GetCapabilitySchema(context.Background(), &zynaxv1.GetCapabilitySchemaRequest{CapabilityName: "triage"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CapabilityName != "triage" || resp.InputSchemaJson != `{"type":"object"}` || resp.OutputSchemaJson != `{"type":"string"}` {
		t.Errorf("resp = %+v", resp)
	}

	if _, err := testServer().GetCapabilitySchema(context.Background(), &zynaxv1.GetCapabilitySchemaRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("empty name code = %v, want InvalidArgument", status.Code(err))
	}
	if _, err := testServer().GetCapabilitySchema(context.Background(), &zynaxv1.GetCapabilitySchemaRequest{CapabilityName: "missing"}); status.Code(err) != codes.NotFound {
		t.Errorf("unknown code = %v, want NotFound", status.Code(err))
	}
}
