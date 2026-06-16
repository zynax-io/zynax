// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc/metadata"
)

// eventSink implements zynaxv1.AgentService_ExecuteCapabilityServer so the MCP
// shim can drive the adapter's existing streaming handlers and collapse the
// PROGRESS/COMPLETED/FAILED stream into a single MCP tool result. PROGRESS
// events are dropped (MCP tool calls are request/response, not streaming).
type eventSink struct {
	ctx  context.Context
	last *zynaxv1.TaskEvent
}

func newEventSink(ctx context.Context) *eventSink {
	return &eventSink{ctx: ctx}
}

func (s *eventSink) Send(e *zynaxv1.TaskEvent) error {
	// Keep only terminal events; PROGRESS is not surfaced to MCP callers.
	if e.GetEventType() == zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS {
		return nil
	}
	s.last = e
	return nil
}

func (s *eventSink) Context() context.Context     { return s.ctx }
func (s *eventSink) SetHeader(metadata.MD) error  { return nil }
func (s *eventSink) SendHeader(metadata.MD) error { return nil }
func (s *eventSink) SetTrailer(metadata.MD)       {}
func (s *eventSink) SendMsg(interface{}) error    { return nil }
func (s *eventSink) RecvMsg(interface{}) error    { return nil }

// result returns the tool output text and whether it represents an error. On a
// FAILED terminal event it returns the CapabilityError code+message; on
// COMPLETED it returns the JSON output payload verbatim (no Git logic here —
// the payload is whatever the adapter handler produced).
func (s *eventSink) result() (text string, isError bool) {
	if s.last == nil {
		return "no terminal event received from adapter", true
	}
	if s.last.GetEventType() == zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		if e := s.last.GetError(); e != nil {
			return e.GetCode() + ": " + e.GetMessage(), true
		}
		return "capability failed", true
	}
	return string(s.last.GetPayload()), false
}
