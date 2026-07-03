// SPDX-License-Identifier: Apache-2.0

// Error-path and option-forwarding tests for the Ollama model adapter —
// transport failures, malformed bodies, stream error frames, scanner limits,
// early consumer stops, and per-request model/temperature overrides.
package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

func TestGenerateContent_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close() // connection refused from here on

	o := NewOllama(srv.URL, "m")
	if _, err := collect(o.GenerateContent(context.Background(), userReq("hi"), false)); err == nil {
		t.Fatal("expected a transport error against a closed server")
	}
}

func TestGenerateContent_InvalidRequestURL(t *testing.T) {
	// "%zz" is an invalid URL escape — request construction itself must fail.
	o := NewOllama("http://%zz", "m")
	if _, err := collect(o.GenerateContent(context.Background(), userReq("hi"), false)); err == nil {
		t.Fatal("expected an error building the request for an invalid host URL")
	}
}

func TestGenerateContent_DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("{not json"))
	}))
	defer srv.Close()

	o := NewOllama(srv.URL, "m")
	if _, err := collect(o.GenerateContent(context.Background(), userReq("hi"), false)); err == nil {
		t.Fatal("expected a decode error for a malformed non-stream body")
	}
}

// TestGenerateContent_ModelOverrideAndTemperature covers the per-request model
// override (modelFor) and the temperature forwarding (optionsFor).
func TestGenerateContent_ModelOverrideAndTemperature(t *testing.T) {
	var gotReq chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		_ = json.NewEncoder(w).Encode(chatResponse{Message: chatMessage{Role: roleAssistant, Content: "ok"}, Done: true})
	}))
	defer srv.Close()

	temp := float32(0.25)
	req := userReq("hi")
	req.Model = "override-model"
	req.Config.Temperature = &temp

	o := NewOllama(srv.URL, "configured-model")
	if _, err := collect(o.GenerateContent(context.Background(), req, false)); err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}
	if gotReq.Model != "override-model" {
		t.Errorf("model = %q, want the per-request override", gotReq.Model)
	}
	if gotReq.Options == nil || gotReq.Options["temperature"] == nil {
		t.Errorf("options = %+v, want temperature forwarded", gotReq.Options)
	}
}

func TestGenerateContent_StreamBadChunk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("{bad json\n"))
	}))
	defer srv.Close()

	o := NewOllama(srv.URL, "m")
	if _, err := collect(o.GenerateContent(context.Background(), userReq("hi"), true)); err == nil {
		t.Fatal("expected a decode error for a malformed stream chunk")
	}
}

func TestGenerateContent_StreamErrorFrame(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		enc := json.NewEncoder(w)
		_ = enc.Encode(chatResponse{Message: chatMessage{Content: "Hel"}})
		_ = enc.Encode(chatResponse{Error: "model exploded"})
	}))
	defer srv.Close()

	o := NewOllama(srv.URL, "m")
	if _, err := collect(o.GenerateContent(context.Background(), userReq("hi"), true)); err == nil {
		t.Fatal("expected the stream error frame to surface as an error")
	}
}

// TestGenerateContent_StreamNoDoneMarker covers a stream that ends at EOF
// without an explicit done frame — the aggregate must still be surfaced.
func TestGenerateContent_StreamNoDoneMarker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		enc := json.NewEncoder(w)
		_ = enc.Encode(chatResponse{Message: chatMessage{Content: "Wor"}})
		_ = enc.Encode(chatResponse{Message: chatMessage{Content: "ld"}})
	}))
	defer srv.Close()

	o := NewOllama(srv.URL, "m")
	got, err := collect(o.GenerateContent(context.Background(), userReq("hi"), true))
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}
	final := got[len(got)-1]
	if final.Partial || !final.TurnComplete || contentText(final.Content) != "World" {
		t.Errorf("final = %+v text=%q, want aggregated %q", final, contentText(final.Content), "World")
	}
}

// TestStreamChat_ScannerError covers a stream line exceeding the 1 MiB scanner
// buffer — the read error must surface, not hang or truncate silently.
func TestStreamChat_ScannerError(t *testing.T) {
	huge := strings.Repeat("a", 2*1024*1024)
	var gotErr error
	streamChat(strings.NewReader(huge), func(_ *adkmodel.LLMResponse, err error) bool {
		gotErr = err
		return true
	})
	if gotErr == nil {
		t.Fatal("expected a scanner error for an oversized stream line")
	}
}

// TestStreamChat_ConsumerStops covers the consumer breaking out mid-stream —
// streamChat must stop yielding immediately (delta branch).
func TestStreamChat_ConsumerStops(t *testing.T) {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	_ = enc.Encode(chatResponse{Message: chatMessage{Content: "one"}})
	_ = enc.Encode(chatResponse{Message: chatMessage{Content: "two"}})
	_ = enc.Encode(chatResponse{Done: true})

	yields := 0
	streamChat(strings.NewReader(b.String()), func(*adkmodel.LLMResponse, error) bool {
		yields++
		return false // stop after the first delta
	})
	if yields != 1 {
		t.Fatalf("yields = %d, want 1 (consumer stopped)", yields)
	}
}

// TestStreamChat_ConsumerStopsOnDoneTail covers the consumer breaking on the
// partial yielded from a done frame that carries tail content.
func TestStreamChat_ConsumerStopsOnDoneTail(t *testing.T) {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	_ = enc.Encode(chatResponse{Message: chatMessage{Content: "tail"}, Done: true})

	yields := 0
	streamChat(strings.NewReader(b.String()), func(*adkmodel.LLMResponse, error) bool {
		yields++
		return false // refuse the done-frame tail partial
	})
	if yields != 1 {
		t.Fatalf("yields = %d, want 1 (stopped on the done-frame tail)", yields)
	}
}

// TestStreamChat_SkipsBlankLines covers the empty-line continue branch.
func TestStreamChat_SkipsBlankLines(t *testing.T) {
	var b strings.Builder
	b.WriteString("\n\n")
	enc := json.NewEncoder(&b)
	_ = enc.Encode(chatResponse{Message: chatMessage{Content: "hi"}})
	_ = enc.Encode(chatResponse{Done: true})

	var texts []string
	streamChat(strings.NewReader(b.String()), func(r *adkmodel.LLMResponse, err error) bool {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		texts = append(texts, contentText(r.Content))
		return true
	})
	if len(texts) != 2 || texts[1] != "hi" {
		t.Fatalf("texts = %v, want one delta and the final aggregate", texts)
	}
}

// TestToOllamaMessages_NilAndEmpty covers the nil-request and empty-content
// guards in the translation helpers.
func TestToOllamaMessages_NilAndEmpty(t *testing.T) {
	if got := toOllamaMessages(nil); got != nil {
		t.Errorf("toOllamaMessages(nil) = %v, want nil", got)
	}
	// Content with no text parts is skipped entirely.
	req := &adkmodel.LLMRequest{Contents: []*genai.Content{{Role: "user", Parts: []*genai.Part{nil}}}}
	if got := toOllamaMessages(req); len(got) != 0 {
		t.Errorf("empty content should produce no messages, got %v", got)
	}
	if got := contentText(nil); got != "" {
		t.Errorf("contentText(nil) = %q, want empty", got)
	}
}

// TestOptionsFor_Nil covers the unconstrained branches directly.
func TestOptionsFor_Nil(t *testing.T) {
	if optionsFor(nil) != nil {
		t.Error("optionsFor(nil) should be nil")
	}
	if optionsFor(&adkmodel.LLMRequest{}) != nil {
		t.Error("optionsFor without config should be nil")
	}
}
