// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/llm/internal/provider"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

// fakeProvider drives the handler with a scripted chunk sequence. When delay is
// positive each chunk is held back so a context deadline can fire first.
type fakeProvider struct {
	chunks   []provider.Chunk
	startErr error
	delay    time.Duration
}

func (f *fakeProvider) Stream(ctx context.Context, _ string) (<-chan provider.Chunk, error) {
	if f.startErr != nil {
		return nil, f.startErr
	}
	out := make(chan provider.Chunk)
	go func() {
		defer close(out)
		for _, c := range f.chunks {
			if f.delay > 0 {
				select {
				case <-time.After(f.delay):
				case <-ctx.Done():
					return
				}
			}
			select {
			case out <- c:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

// recordSink captures every TaskEvent sent and exposes the stream context.
type recordSink struct {
	ctx    context.Context
	events []*zynaxv1.TaskEvent
}

func (r *recordSink) Send(ev *zynaxv1.TaskEvent) error {
	r.events = append(r.events, ev)
	return nil
}

func (r *recordSink) Context() context.Context {
	if r.ctx == nil {
		return context.Background()
	}
	return r.ctx
}

const chatSchema = `{"type":"object","required":["prompt"],"properties":{"prompt":{"type":"string","minLength":1}},"additionalProperties":false}`

func newHandler(t *testing.T, p provider.Provider) *ChatCompletionHandler {
	t.Helper()
	h, err := newChatCompletionHandler(p, chatSchema)
	if err != nil {
		t.Fatalf("newChatCompletionHandler: %v", err)
	}
	return h
}

func lastEvent(evs []*zynaxv1.TaskEvent) *zynaxv1.TaskEvent {
	return evs[len(evs)-1]
}

func TestExecuteStreamsProgressThenCompleted(t *testing.T) {
	p := &fakeProvider{chunks: []provider.Chunk{{Text: "Hel"}, {Text: "lo"}}}
	h := newHandler(t, p)
	snk := &recordSink{}

	if err := h.Execute("task-1", []byte(`{"prompt":"hi"}`), 0, snk); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(snk.events) != 3 {
		t.Fatalf("want 3 events, got %d", len(snk.events))
	}
	for i := 0; i < 2; i++ {
		if snk.events[i].GetEventType() != zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS {
			t.Errorf("event %d: want PROGRESS, got %v", i, snk.events[i].GetEventType())
		}
	}
	final := lastEvent(snk.events)
	if final.GetEventType() != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
		t.Fatalf("final: want COMPLETED, got %v", final.GetEventType())
	}
	var out map[string]string
	if err := json.Unmarshal(final.GetPayload(), &out); err != nil {
		t.Fatalf("unmarshal completed payload: %v", err)
	}
	if out["completion"] != "Hello" {
		t.Errorf("completion = %q, want %q", out["completion"], "Hello")
	}
	for _, ev := range snk.events {
		if ev.GetTaskId() != "task-1" {
			t.Errorf("task_id = %q, want task-1", ev.GetTaskId())
		}
		if ev.GetTimestamp() == nil {
			t.Errorf("event %v: missing timestamp", ev.GetEventType())
		}
	}
}

func TestExecuteEmptyStreamStillEmitsProgress(t *testing.T) {
	p := &fakeProvider{chunks: nil}
	h := newHandler(t, p)
	snk := &recordSink{}

	if err := h.Execute("t", []byte(`{"prompt":"hi"}`), 0, snk); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(snk.events) != 2 {
		t.Fatalf("want 2 events (PROGRESS+COMPLETED), got %d", len(snk.events))
	}
	if snk.events[0].GetEventType() != zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS {
		t.Errorf("first: want PROGRESS, got %v", snk.events[0].GetEventType())
	}
	if lastEvent(snk.events).GetEventType() != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
		t.Errorf("final: want COMPLETED")
	}
}

func TestExecuteInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{"not json", `{not json`},
		{"missing prompt", `{"foo":"bar"}`},
		{"empty prompt", `{"prompt":""}`},
		{"wrong type", `{"prompt":123}`},
		{"additional prop", `{"prompt":"hi","extra":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHandler(t, &fakeProvider{})
			snk := &recordSink{}
			if err := h.Execute("t", []byte(tt.payload), 0, snk); err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if len(snk.events) != 1 {
				t.Fatalf("want exactly 1 terminal event, got %d", len(snk.events))
			}
			final := lastEvent(snk.events)
			if final.GetEventType() != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
				t.Fatalf("want FAILED, got %v", final.GetEventType())
			}
			if final.GetError().GetCode() != codeInvalidInput {
				t.Errorf("code = %q, want %q", final.GetError().GetCode(), codeInvalidInput)
			}
		})
	}
}

func TestExecuteUpstreamErrorSanitised(t *testing.T) {
	p := &fakeProvider{chunks: []provider.Chunk{
		{Text: "partial"},
		{Err: errors.New("openai: upstream returned HTTP 429")},
	}}
	h := newHandler(t, p)
	snk := &recordSink{}

	if err := h.Execute("t", []byte(`{"prompt":"hi"}`), 0, snk); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	final := lastEvent(snk.events)
	if final.GetEventType() != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Fatalf("want FAILED, got %v", final.GetEventType())
	}
	if final.GetError().GetCode() != codeUpstream {
		t.Errorf("code = %q, want %q", final.GetError().GetCode(), codeUpstream)
	}
	if final.GetError().GetMessage() == "" {
		t.Errorf("message must be non-empty")
	}
}

func TestExecuteStartFailureMapsUpstream(t *testing.T) {
	p := &fakeProvider{startErr: errors.New("dial tcp: connection refused")}
	h := newHandler(t, p)
	snk := &recordSink{}

	if err := h.Execute("t", []byte(`{"prompt":"hi"}`), 0, snk); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if snk.events[0].GetEventType() != zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS {
		t.Errorf("first event: want leading PROGRESS, got %v", snk.events[0].GetEventType())
	}
	final := lastEvent(snk.events)
	if final.GetError().GetCode() != codeUpstream {
		t.Errorf("code = %q, want %q", final.GetError().GetCode(), codeUpstream)
	}
}

func TestExecuteTimeout(t *testing.T) {
	// Provider stalls past the 1s budget, so the deadline fires before any
	// token: a leading PROGRESS then a terminal TIMEOUT must be emitted.
	p := &fakeProvider{chunks: []provider.Chunk{{Text: "x"}}, delay: 3 * time.Second}
	h := newHandler(t, p)
	snk := &recordSink{}

	start := time.Now()
	if err := h.Execute("t", []byte(`{"prompt":"hi"}`), 1, snk); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(snk.events) < 2 || snk.events[0].GetEventType() != zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS {
		t.Fatalf("want leading PROGRESS + terminal, got %d events", len(snk.events))
	}
	final := lastEvent(snk.events)
	if final.GetEventType() != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED || final.GetError().GetCode() != codeTimeout {
		t.Fatalf("final = %v/%q, want FAILED/TIMEOUT", final.GetEventType(), final.GetError().GetCode())
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("timeout took too long: %v", time.Since(start))
	}
}

func TestNoSchemaStillEnforcesPrompt(t *testing.T) {
	h, err := newChatCompletionHandler(&fakeProvider{}, "")
	if err != nil {
		t.Fatalf("newChatCompletionHandler: %v", err)
	}
	snk := &recordSink{}
	if err := h.Execute("t", []byte(`{}`), 0, snk); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if lastEvent(snk.events).GetError().GetCode() != codeInvalidInput {
		t.Errorf("want INVALID_INPUT for missing prompt without schema")
	}
}

func TestNewChatCompletionHandlerBadSchema(t *testing.T) {
	if _, err := newChatCompletionHandler(&fakeProvider{}, `{not a schema`); err == nil {
		t.Fatal("want error for malformed schema")
	}
}

// failSink fails Send on the call indexed by failAt (0-based).
type failSink struct {
	calls  int
	failAt int
}

func (f *failSink) Send(*zynaxv1.TaskEvent) error {
	defer func() { f.calls++ }()
	if f.calls == f.failAt {
		return errors.New("transport closed")
	}
	return nil
}
func (f *failSink) Context() context.Context { return context.Background() }

func TestExecutePropagatesSendError(t *testing.T) {
	// Fail on the first PROGRESS send; Execute must return the transport error.
	p := &fakeProvider{chunks: []provider.Chunk{{Text: "a"}, {Text: "b"}}}
	h := newHandler(t, p)
	if err := h.Execute("t", []byte(`{"prompt":"hi"}`), 0, &failSink{failAt: 0}); err == nil {
		t.Fatal("want transport error propagated from Send")
	}
}

func TestExecuteSendErrorOnTerminal(t *testing.T) {
	// One chunk → PROGRESS (ok), COMPLETED send fails.
	p := &fakeProvider{chunks: []provider.Chunk{{Text: "a"}}}
	h := newHandler(t, p)
	if err := h.Execute("t", []byte(`{"prompt":"hi"}`), 0, &failSink{failAt: 1}); err == nil {
		t.Fatal("want transport error on terminal send")
	}
}

func TestSanitiseTruncates(t *testing.T) {
	long := strings.Repeat("x", maxErrMsgLen+50)
	got := sanitise(long)
	if len([]rune(got)) != maxErrMsgLen {
		t.Errorf("len = %d, want %d", len([]rune(got)), maxErrMsgLen)
	}
	if sanitise("short") != "short" {
		t.Errorf("short message must pass through unchanged")
	}
}
