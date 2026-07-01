// SPDX-License-Identifier: Apache-2.0

package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/zynax-io/zynax/agents/adapters/adk/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	triageInputSchema  = `{"type":"object","properties":{"prompt":{"type":"string"}},"required":["prompt"]}`
	triageOutputSchema = `{"type":"object","properties":{"response":{"type":"string"}},"required":["response"]}`
)

// fakeStream captures sent TaskEvents. Embedding the generated stream interface
// supplies the gRPC plumbing methods the bridge never calls.
type fakeStream struct {
	zynaxv1.AgentService_ExecuteCapabilityServer
	ctx    context.Context
	events []*zynaxv1.TaskEvent
}

func (f *fakeStream) Send(e *zynaxv1.TaskEvent) error { f.events = append(f.events, e); return nil }
func (f *fakeStream) Context() context.Context {
	if f.ctx != nil {
		return f.ctx
	}
	return context.Background()
}

// stubLLM is a deterministic adk model.LLM: it yields Partial responses for each
// chunk, then a final non-Partial response, exercising the real ADK Runner with
// no Ollama endpoint. A non-nil err is yielded instead, simulating a model fault.
type stubLLM struct {
	chunks []string
	final  string
	err    error
}

func (stubLLM) Name() string { return "stub" }

func (s stubLLM) GenerateContent(_ context.Context, _ *adkmodel.LLMRequest, _ bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		if s.err != nil {
			yield(nil, s.err)
			return
		}
		acc := strings.Builder{}
		for _, c := range s.chunks {
			acc.WriteString(c)
			if !yield(&adkmodel.LLMResponse{Content: textContent(c), Partial: true}, nil) {
				return
			}
		}
		final := s.final
		if final == "" {
			final = acc.String()
		}
		yield(&adkmodel.LLMResponse{Content: textContent(final), Partial: false, TurnComplete: true}, nil)
	}
}

func textContent(t string) *genai.Content {
	return &genai.Content{Role: "model", Parts: []*genai.Part{genai.NewPartFromText(t)}}
}

func triageServer(t *testing.T, llm adkmodel.LLM) *AgentServer {
	t.Helper()
	srv, err := newAgentServer(&config.AdapterConfig{
		Name:    "adk-adapter",
		AgentID: "adk-1",
		Model:   config.ModelConfig{Provider: config.ProviderOllama},
		Capabilities: []config.CapabilityConfig{{
			Name:             "triage",
			Description:      "classify",
			Instruction:      "You classify tickets.",
			InputSchemaJSON:  triageInputSchema,
			OutputSchemaJSON: triageOutputSchema,
		}},
	}, llm)
	if err != nil {
		t.Fatalf("newAgentServer: %v", err)
	}
	return srv
}

// TestExecuteCapability_Validation covers the request-guard paths that never
// reach the ADK Runner, so they are fully deterministic.
func TestExecuteCapability_Validation(t *testing.T) {
	srv := triageServer(t, stubLLM{final: "ok"})
	cases := []struct {
		name      string
		req       *zynaxv1.ExecuteCapabilityRequest
		wantGRPC  codes.Code // OK => expect a terminal FAILED event instead
		wantEvent string     // CapabilityError code on the terminal FAILED event
	}{
		{"empty task_id", &zynaxv1.ExecuteCapabilityRequest{CapabilityName: "triage"}, codes.InvalidArgument, ""},
		{"empty capability", &zynaxv1.ExecuteCapabilityRequest{TaskId: "t1"}, codes.InvalidArgument, ""},
		{"unknown capability", &zynaxv1.ExecuteCapabilityRequest{TaskId: "t1", CapabilityName: "missing"}, codes.OK, codeInvalidInput},
		{"malformed input", &zynaxv1.ExecuteCapabilityRequest{TaskId: "t1", CapabilityName: "triage", InputPayload: []byte("not json")}, codes.OK, codeInvalidInput},
		{"schema-invalid input (no prompt)", &zynaxv1.ExecuteCapabilityRequest{TaskId: "t1", CapabilityName: "triage", InputPayload: []byte(`{"foo":"bar"}`)}, codes.OK, codeInvalidInput},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stream := &fakeStream{}
			err := srv.ExecuteCapability(tc.req, stream)
			if tc.wantGRPC != codes.OK {
				if status.Code(err) != tc.wantGRPC {
					t.Fatalf("code = %v, want %v", status.Code(err), tc.wantGRPC)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			last := stream.events[len(stream.events)-1]
			if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
				t.Fatalf("event_type = %v, want FAILED", last.EventType)
			}
			if last.GetError().GetCode() != tc.wantEvent {
				t.Errorf("code = %q, want %q", last.GetError().GetCode(), tc.wantEvent)
			}
			if last.TaskId != "t1" || last.Timestamp == nil {
				t.Errorf("task_id/timestamp not populated: %+v", last)
			}
		})
	}
}

// TestExecuteCapability_Dispatch drives the real ADK Runner (with a stub model)
// end-to-end: a known capability must stream >=1 PROGRESS then one COMPLETED
// whose payload validates against the declared output schema (ADR-038 bridge).
func TestExecuteCapability_Dispatch(t *testing.T) {
	srv := triageServer(t, stubLLM{chunks: []string{"urgent"}, final: "urgent: page on-call"})
	stream := &fakeStream{}
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t1",
		WorkflowId:     "wf-1",
		CapabilityName: "triage",
		InputPayload:   []byte(`{"prompt":"server down"}`),
	}
	if err := srv.ExecuteCapability(req, stream); err != nil {
		t.Fatalf("ExecuteCapability: %v", err)
	}
	if len(stream.events) < 2 {
		t.Fatalf("want >=2 events (PROGRESS+COMPLETED), got %d", len(stream.events))
	}
	progress, completed := 0, 0
	var completedPayload []byte
	for _, ev := range stream.events {
		if ev.TaskId != "t1" {
			t.Errorf("task_id = %q, want t1", ev.TaskId)
		}
		switch ev.EventType {
		case zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS:
			progress++
		case zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED:
			completed++
			completedPayload = ev.Payload
		default:
			t.Errorf("unexpected terminal event: %v %v", ev.EventType, ev.GetError())
		}
	}
	if progress < 1 || completed != 1 {
		t.Fatalf("progress=%d completed=%d, want >=1 and exactly 1", progress, completed)
	}
	if stream.events[len(stream.events)-1].EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
		t.Fatal("COMPLETED must be the final event")
	}
	assertValid(t, triageOutputSchema, completedPayload)
	// The model's actual final text must reach the payload — a dropped/empty
	// final would still pass schema validation, so assert the value too.
	var got map[string]string
	if err := json.Unmarshal(completedPayload, &got); err != nil || got["response"] != "urgent: page on-call" {
		t.Errorf("COMPLETED payload = %s, want response=%q", completedPayload, "urgent: page on-call")
	}
}

// --- bridge() unit tests: deterministic over a fabricated event stream ---

func seqOf(events []*session.Event, tail error) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		for _, e := range events {
			if !yield(e, nil) {
				return
			}
		}
		if tail != nil {
			yield(nil, tail)
		}
	}
}

func evText(text string, partial bool) *session.Event {
	return &session.Event{LLMResponse: adkmodel.LLMResponse{Content: textContent(text), Partial: partial}}
}

func triageRuntime(t *testing.T) *capRuntime {
	t.Helper()
	out, err := compileSchema(triageOutputSchema)
	if err != nil {
		t.Fatal(err)
	}
	return &capRuntime{outSchema: out, outKey: outputTargetKey(triageOutputSchema)}
}

func TestBridge_ProgressThenCompleted(t *testing.T) {
	stream := &fakeStream{}
	seq := seqOf([]*session.Event{evText("part-", true), evText("final answer", false)}, nil)
	if err := bridge(context.Background(), stream, "t1", triageRuntime(t), seq); err != nil {
		t.Fatalf("bridge: %v", err)
	}
	if len(stream.events) != 2 {
		t.Fatalf("want 2 events, got %d", len(stream.events))
	}
	if stream.events[0].EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS {
		t.Errorf("first event not PROGRESS: %v", stream.events[0].EventType)
	}
	last := stream.events[1]
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
		t.Fatalf("last event not COMPLETED: %v", last.EventType)
	}
	assertValid(t, triageOutputSchema, last.Payload)
	var got map[string]string
	if err := json.Unmarshal(last.Payload, &got); err != nil || got["response"] != "final answer" {
		t.Errorf("coerced payload = %s, want response=final answer", last.Payload)
	}
}

func TestBridge_NoPartialStillEmitsProgress(t *testing.T) {
	stream := &fakeStream{}
	// Only a final non-partial event — the bridge must still emit >=1 PROGRESS.
	seq := seqOf([]*session.Event{evText("single turn", false)}, nil)
	if err := bridge(context.Background(), stream, "t1", triageRuntime(t), seq); err != nil {
		t.Fatalf("bridge: %v", err)
	}
	if len(stream.events) != 2 || stream.events[0].EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS {
		t.Fatalf("want PROGRESS then COMPLETED, got %+v", stream.events)
	}
}

func TestBridge_ModelErrorIsUpstream(t *testing.T) {
	stream := &fakeStream{}
	seq := seqOf(nil, errors.New("connection refused"))
	if err := bridge(context.Background(), stream, "t1", triageRuntime(t), seq); err != nil {
		t.Fatalf("bridge: %v", err)
	}
	last := stream.events[len(stream.events)-1]
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED || last.GetError().GetCode() != codeUpstream {
		t.Fatalf("want FAILED UPSTREAM_ERROR, got %v %v", last.EventType, last.GetError())
	}
}

func TestBridge_DeadlineIsTimeout(t *testing.T) {
	// A wrapped context.DeadlineExceeded surfaced by the runner classifies as TIMEOUT.
	stream := &fakeStream{}
	seq := seqOf(nil, context.DeadlineExceeded)
	if err := bridge(context.Background(), stream, "t1", triageRuntime(t), seq); err != nil {
		t.Fatalf("bridge: %v", err)
	}
	last := stream.events[len(stream.events)-1]
	if last.GetError().GetCode() != codeTimeout {
		t.Fatalf("want TIMEOUT, got %v", last.GetError())
	}
}

func TestBridge_ExpiredContextIsTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // ctx.Err() != nil after the loop drains
	stream := &fakeStream{}
	seq := seqOf([]*session.Event{evText("done", false)}, nil)
	if err := bridge(ctx, stream, "t1", triageRuntime(t), seq); err != nil {
		t.Fatalf("bridge: %v", err)
	}
	last := stream.events[len(stream.events)-1]
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED || last.GetError().GetCode() != codeTimeout {
		t.Fatalf("want FAILED TIMEOUT, got %v %v", last.EventType, last.GetError())
	}
}

// --- pure helper tests ---

func TestCoerceOutput(t *testing.T) {
	rt := triageRuntime(t)
	// Plain text is wrapped under the schema's required string key (ok=true).
	if got, ok := coerceOutput("hello", rt); !ok || string(got) != `{"response":"hello"}` {
		t.Errorf("wrap: got %s ok=%v", got, ok)
	}
	// A schema-valid JSON object passes through unchanged.
	passthrough := `{"response":"already shaped"}` //nolint:gosec // G101 false positive: JSON test fixture, not a credential
	if got, ok := coerceOutput(passthrough, rt); !ok || string(got) != passthrough {
		t.Errorf("passthrough: got %s ok=%v", got, ok)
	}
	// No output schema -> raw bytes, ok.
	if got, ok := coerceOutput("raw", &capRuntime{}); !ok || string(got) != "raw" {
		t.Errorf("raw: got %s ok=%v", got, ok)
	}
	// A schema the text cannot satisfy (extra required field) -> ok=false so the
	// caller emits FAILED instead of a schema-invalid COMPLETED.
	strict, err := compileSchema(`{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}},"required":["a","b"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := coerceOutput("plain text", &capRuntime{outSchema: strict, outKey: "a"}); ok {
		t.Error("expected ok=false when the wrapped text cannot satisfy a multi-field schema")
	}
}

// TestBridge_SchemaInvalidOutputFails proves a run whose text cannot be coerced
// to the output schema ends in FAILED UPSTREAM_ERROR, not a bad COMPLETED — so a
// COMPLETED payload always validates (the BDD invariant).
func TestBridge_SchemaInvalidOutputFails(t *testing.T) {
	strict, err := compileSchema(`{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}},"required":["a","b"]}`)
	if err != nil {
		t.Fatal(err)
	}
	rt := &capRuntime{outSchema: strict, outKey: "a"}
	stream := &fakeStream{}
	if err := bridge(context.Background(), stream, "t1", rt, seqOf([]*session.Event{evText("just prose", false)}, nil)); err != nil {
		t.Fatalf("bridge: %v", err)
	}
	last := stream.events[len(stream.events)-1]
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED || last.GetError().GetCode() != codeUpstream {
		t.Fatalf("want FAILED UPSTREAM_ERROR, got %v %v", last.EventType, last.GetError())
	}
}

func TestBoundMessage(t *testing.T) {
	// Truncation must never split a multi-byte rune (proto3 requires valid UTF-8).
	long := strings.Repeat("a", maxErrMsgLen-1) + "é" + strings.Repeat("z", 100)
	got := boundMessage(long)
	if !utf8.ValidString(got) {
		t.Fatalf("boundMessage produced invalid UTF-8")
	}
	if len([]rune(got)) > maxErrMsgLen {
		t.Errorf("rune length %d exceeds %d", len([]rune(got)), maxErrMsgLen)
	}
	// A short message is returned unchanged.
	if boundMessage("short") != "short" {
		t.Error("short message altered")
	}
}

func TestOutputTargetKey(t *testing.T) {
	cases := map[string]string{
		triageOutputSchema:  "response",
		`{"type":"string"}`: "",
		``:                  "",
		`{"properties":{"n":{"type":"integer"}}}`: "",
	}
	for schema, want := range cases {
		if got := outputTargetKey(schema); got != want {
			t.Errorf("outputTargetKey(%q) = %q, want %q", schema, got, want)
		}
	}
}

func TestParseInput(t *testing.T) {
	sch, err := compileSchema(triageInputSchema)
	if err != nil {
		t.Fatal(err)
	}
	if p, err := parseInput([]byte(`{"prompt":"hi"}`), sch); err != nil || p != "hi" {
		t.Errorf("valid: p=%q err=%v", p, err)
	}
	if _, err := parseInput([]byte(`{"other":"x"}`), sch); err == nil {
		t.Error("expected schema validation error for missing prompt")
	}
	if _, err := parseInput([]byte(`nope`), sch); err == nil {
		t.Error("expected JSON parse error")
	}
	// Nil schema still enforces prompt presence.
	if _, err := parseInput([]byte(`{}`), nil); err == nil {
		t.Error("expected missing-prompt error with nil schema")
	}
}

func TestGetCapabilitySchema(t *testing.T) {
	srv := triageServer(t, stubLLM{final: "ok"})
	resp, err := srv.GetCapabilitySchema(context.Background(), &zynaxv1.GetCapabilitySchemaRequest{CapabilityName: "triage"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CapabilityName != "triage" || resp.InputSchemaJson != triageInputSchema || resp.OutputSchemaJson != triageOutputSchema {
		t.Errorf("resp = %+v", resp)
	}
	if _, err := srv.GetCapabilitySchema(context.Background(), &zynaxv1.GetCapabilitySchemaRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("empty name code = %v, want InvalidArgument", status.Code(err))
	}
	if _, err := srv.GetCapabilitySchema(context.Background(), &zynaxv1.GetCapabilitySchemaRequest{CapabilityName: "missing"}); status.Code(err) != codes.NotFound {
		t.Errorf("unknown code = %v, want NotFound", status.Code(err))
	}
}

// assertValid fails if payload does not validate against schemaJSON.
func assertValid(t *testing.T, schemaJSON string, payload []byte) {
	t.Helper()
	c := jsonschema.NewCompiler()
	if err := c.AddResource("s.json", strings.NewReader(schemaJSON)); err != nil {
		t.Fatal(err)
	}
	sch, err := c.Compile("s.json")
	if err != nil {
		t.Fatal(err)
	}
	var v any
	if err := json.Unmarshal(payload, &v); err != nil {
		t.Fatalf("payload not JSON: %s", payload)
	}
	if err := sch.Validate(v); err != nil {
		t.Fatalf("payload %s does not validate: %v", payload, err)
	}
}
