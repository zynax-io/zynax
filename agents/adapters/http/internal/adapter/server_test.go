// SPDX-License-Identifier: Apache-2.0

package adapter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/http/internal/adapter"
	"github.com/zynax-io/zynax/agents/adapters/http/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc/metadata"
)

// fakeStream is a test double for AgentService_ExecuteCapabilityServer.
type fakeStream struct {
	ctx    context.Context
	events []*zynaxv1.TaskEvent
}

func (f *fakeStream) Send(e *zynaxv1.TaskEvent) error {
	f.events = append(f.events, e)
	return nil
}
func (f *fakeStream) Context() context.Context     { return f.ctx }
func (f *fakeStream) SetHeader(metadata.MD) error  { return nil }
func (f *fakeStream) SendHeader(metadata.MD) error { return nil }
func (f *fakeStream) SetTrailer(metadata.MD)       {}
func (f *fakeStream) SendMsg(any) error            { return nil }
func (f *fakeStream) RecvMsg(any) error            { return nil }

func (f *fakeStream) lastEvent() *zynaxv1.TaskEvent {
	return f.events[len(f.events)-1]
}

func newServer(t *testing.T, url string) *adapter.AgentServer {
	t.Helper()
	return adapter.NewAgentServer(&config.AdapterConfig{
		AgentID:          "test",
		Endpoint:         "0.0.0.0:8080",
		RegistryEndpoint: "registry:9090",
		Capabilities:     []config.RouteConfig{{Name: "call_api", Method: "POST", URL: url}},
	})
}

func TestExecuteCapability_2xx_CompletedWithPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"ok"}`))
	}))
	defer srv.Close()

	stream := &fakeStream{ctx: context.Background()}
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "task-1",
		CapabilityName: "call_api",
		InputPayload:   []byte(`{"key":"value"}`),
	}
	if err := newServer(t, srv.URL).ExecuteCapability(req, stream); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	last := stream.lastEvent()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
		t.Errorf("event_type = %v, want COMPLETED", last.EventType)
	}
	if string(last.Payload) != `{"result":"ok"}` {
		t.Errorf("payload = %s", last.Payload)
	}
	if last.TaskId != "task-1" {
		t.Errorf("task_id = %s, want task-1", last.TaskId)
	}
	if last.Timestamp == nil {
		t.Error("timestamp must be set on every event")
	}
}

func TestExecuteCapability_4xx_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	stream := &fakeStream{ctx: context.Background()}
	_ = newServer(t, srv.URL).ExecuteCapability(
		&zynaxv1.ExecuteCapabilityRequest{TaskId: "t1", CapabilityName: "call_api"},
		stream,
	)
	last := stream.lastEvent()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Errorf("want FAILED, got %v", last.EventType)
	}
	if last.Error.Code != "UPSTREAM_ERROR" {
		t.Errorf("code = %s, want UPSTREAM_ERROR", last.Error.Code)
	}
}

func TestExecuteCapability_5xx_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	stream := &fakeStream{ctx: context.Background()}
	_ = newServer(t, srv.URL).ExecuteCapability(
		&zynaxv1.ExecuteCapabilityRequest{TaskId: "t1", CapabilityName: "call_api"},
		stream,
	)
	if stream.lastEvent().Error.Code != "UPSTREAM_ERROR" {
		t.Errorf("code = %s, want UPSTREAM_ERROR", stream.lastEvent().Error.Code)
	}
}

func TestExecuteCapability_ConnectionRefused(t *testing.T) {
	stream := &fakeStream{ctx: context.Background()}
	_ = newServer(t, "http://127.0.0.1:1").ExecuteCapability(
		&zynaxv1.ExecuteCapabilityRequest{TaskId: "t1", CapabilityName: "call_api"},
		stream,
	)
	last := stream.lastEvent()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Errorf("want FAILED, got %v", last.EventType)
	}
	if last.Error.Code != "UPSTREAM_ERROR" {
		t.Errorf("code = %s, want UPSTREAM_ERROR", last.Error.Code)
	}
}

func TestExecuteCapability_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	stream := &fakeStream{ctx: context.Background()}
	_ = newServer(t, srv.URL).ExecuteCapability(
		&zynaxv1.ExecuteCapabilityRequest{
			TaskId: "t1", CapabilityName: "call_api", TimeoutSeconds: 1,
		},
		stream,
	)
	last := stream.lastEvent()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Errorf("want FAILED, got %v", last.EventType)
	}
	if last.Error.Code != "TIMEOUT" {
		t.Errorf("code = %s, want TIMEOUT", last.Error.Code)
	}
}

func TestExecuteCapability_SlowUpstream_EmitsProgress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow upstream test")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	stream := &fakeStream{ctx: context.Background()}
	_ = newServer(t, srv.URL).ExecuteCapability(
		&zynaxv1.ExecuteCapabilityRequest{TaskId: "t1", CapabilityName: "call_api"},
		stream,
	)
	var progressCount int
	for _, e := range stream.events {
		if e.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS {
			progressCount++
		}
	}
	if progressCount == 0 {
		t.Error("expected at least one PROGRESS event for slow upstream (>2s)")
	}
}

func TestExecuteCapability_UnknownCapability(t *testing.T) {
	stream := &fakeStream{ctx: context.Background()}
	_ = newServer(t, "http://127.0.0.1:1").ExecuteCapability(
		&zynaxv1.ExecuteCapabilityRequest{TaskId: "t1", CapabilityName: "nonexistent"},
		stream,
	)
	if stream.lastEvent().Error.Code != "INVALID_INPUT" {
		t.Errorf("code = %s, want INVALID_INPUT", stream.lastEvent().Error.Code)
	}
}

func TestExecuteCapability_SchemaValidation_InvalidPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := adapter.NewAgentServer(&config.AdapterConfig{
		AgentID:          "test",
		Endpoint:         "0.0.0.0:8080",
		RegistryEndpoint: "registry:9090",
		Capabilities: []config.RouteConfig{{
			Name:            "typed_api",
			Method:          "POST",
			URL:             srv.URL,
			InputSchemaJSON: `{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}`,
		}},
	})
	stream := &fakeStream{ctx: context.Background()}
	_ = s.ExecuteCapability(
		&zynaxv1.ExecuteCapabilityRequest{
			TaskId:         "t1",
			CapabilityName: "typed_api",
			InputPayload:   []byte(`{"wrong_field":123}`),
		},
		stream,
	)
	last := stream.lastEvent()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Errorf("want FAILED, got %v", last.EventType)
	}
	if last.Error.Code != "INVALID_INPUT" {
		t.Errorf("code = %s, want INVALID_INPUT", last.Error.Code)
	}
}

func TestGetCapabilitySchema_Known(t *testing.T) {
	s := adapter.NewAgentServer(&config.AdapterConfig{
		AgentID:          "test",
		Endpoint:         "0.0.0.0:8080",
		RegistryEndpoint: "registry:9090",
		Capabilities: []config.RouteConfig{{
			Name:            "call_api",
			Method:          "POST",
			URL:             "https://example.com",
			InputSchemaJSON: `{"type":"object"}`,
			Description:     "test capability",
		}},
	})
	resp, err := s.GetCapabilitySchema(context.Background(),
		&zynaxv1.GetCapabilitySchemaRequest{CapabilityName: "call_api"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CapabilityName != "call_api" {
		t.Errorf("capability_name = %s", resp.CapabilityName)
	}
	if resp.InputSchemaJson != `{"type":"object"}` {
		t.Errorf("input_schema_json = %s", resp.InputSchemaJson)
	}
}

func TestGetCapabilitySchema_Unknown(t *testing.T) {
	s := newServer(t, "https://example.com")
	_, err := s.GetCapabilitySchema(context.Background(),
		&zynaxv1.GetCapabilitySchemaRequest{CapabilityName: "unknown"})
	if err == nil {
		t.Fatal("expected error for unknown capability")
	}
}
