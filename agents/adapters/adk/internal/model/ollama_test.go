// SPDX-License-Identifier: Apache-2.0

package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

// collect drains a GenerateContent iterator into slices for assertion.
func collect(seq func(func(*adkmodel.LLMResponse, error) bool)) ([]*adkmodel.LLMResponse, error) {
	var out []*adkmodel.LLMResponse
	var firstErr error
	for resp, err := range seq {
		if err != nil {
			firstErr = err
			break
		}
		out = append(out, resp)
	}
	return out, firstErr
}

func userReq(text string) *adkmodel.LLMRequest {
	return &adkmodel.LLMRequest{
		Contents: []*genai.Content{{Role: "user", Parts: []*genai.Part{genai.NewPartFromText(text)}}},
		Config: &genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{Parts: []*genai.Part{genai.NewPartFromText("be terse")}},
		},
	}
}

func TestNewOllama_Defaults(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "")
	o := NewOllama("", "")
	if o.host != DefaultHost || o.name != DefaultModelName {
		t.Errorf("defaults: host=%q name=%q", o.host, o.name)
	}
	// Bare host:port is normalised to an http:// URL; trailing slash trimmed.
	if got := NewOllama("127.0.0.1:11434/", "m").host; got != "http://127.0.0.1:11434" {
		t.Errorf("normalise: got %q", got)
	}
	// OLLAMA_HOST is the fallback when config omits the host.
	t.Setenv("OLLAMA_HOST", "http://ollama:11434")
	if got := NewOllama("", "m").host; got != "http://ollama:11434" {
		t.Errorf("env fallback: got %q", got)
	}
	if NewOllama("", "").Name() != DefaultModelName {
		t.Error("Name() should report the model tag")
	}
}

func TestGenerateContent_NonStream(t *testing.T) {
	var gotReq chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		_ = json.NewEncoder(w).Encode(chatResponse{Message: chatMessage{Role: roleAssistant, Content: "hello world"}, Done: true})
	}))
	defer srv.Close()

	o := NewOllama(srv.URL, "m")
	got, err := collect(o.GenerateContent(context.Background(), userReq("hi"), false))
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 response, got %d", len(got))
	}
	if !got[0].TurnComplete || got[0].Partial {
		t.Errorf("final flags: TurnComplete=%v Partial=%v", got[0].TurnComplete, got[0].Partial)
	}
	if contentText(got[0].Content) != "hello world" {
		t.Errorf("text = %q", contentText(got[0].Content))
	}
	// Request translation: system instruction first, then the user turn.
	if gotReq.Stream || gotReq.Model != "m" {
		t.Errorf("request: stream=%v model=%q", gotReq.Stream, gotReq.Model)
	}
	if len(gotReq.Messages) != 2 ||
		gotReq.Messages[0].Role != roleSystem || gotReq.Messages[0].Content != "be terse" ||
		gotReq.Messages[1].Role != roleUser || gotReq.Messages[1].Content != "hi" {
		t.Errorf("messages = %+v", gotReq.Messages)
	}
}

func TestGenerateContent_Stream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		enc := json.NewEncoder(w)
		_ = enc.Encode(chatResponse{Message: chatMessage{Content: "Hel"}})
		_ = enc.Encode(chatResponse{Message: chatMessage{Content: "lo"}})
		_ = enc.Encode(chatResponse{Done: true})
	}))
	defer srv.Close()

	o := NewOllama(srv.URL, "m")
	got, err := collect(o.GenerateContent(context.Background(), userReq("hi"), true))
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}
	// Two partial deltas + one final aggregated response.
	if len(got) != 3 {
		t.Fatalf("want 3 responses, got %d", len(got))
	}
	if !got[0].Partial || !got[1].Partial {
		t.Errorf("first two responses should be Partial")
	}
	final := got[2]
	if final.Partial || !final.TurnComplete || contentText(final.Content) != "Hello" {
		t.Errorf("final = %+v text=%q", final, contentText(final.Content))
	}
}

// TestGenerateContent_StreamContentOnDone covers a server that flushes the tail
// of the generation alongside the done marker — the tail must land in the final
// aggregated text, not be dropped.
func TestGenerateContent_StreamContentOnDone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		enc := json.NewEncoder(w)
		_ = enc.Encode(chatResponse{Message: chatMessage{Content: "Hel"}})
		_ = enc.Encode(chatResponse{Message: chatMessage{Content: "lo"}, Done: true}) // content ON the done frame
	}))
	defer srv.Close()

	o := NewOllama(srv.URL, "m")
	got, err := collect(o.GenerateContent(context.Background(), userReq("hi"), true))
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}
	final := got[len(got)-1]
	if final.Partial || !final.TurnComplete || contentText(final.Content) != "Hello" {
		t.Errorf("final aggregated text = %q, want %q (done-frame tail must not be dropped)", contentText(final.Content), "Hello")
	}
}

func TestGenerateContent_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "model not found", http.StatusNotFound)
	}))
	defer srv.Close()

	o := NewOllama(srv.URL, "m")
	if _, err := collect(o.GenerateContent(context.Background(), userReq("hi"), false)); err == nil {
		t.Fatal("expected an error for a non-200 status")
	}
}

func TestGenerateContent_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(chatResponse{Error: "out of memory", Done: true})
	}))
	defer srv.Close()

	o := NewOllama(srv.URL, "m")
	if _, err := collect(o.GenerateContent(context.Background(), userReq("hi"), false)); err == nil {
		t.Fatal("expected an error surfaced from the response body")
	}
}

func TestOllamaRole(t *testing.T) {
	for in, want := range map[string]string{"model": roleAssistant, "assistant": roleAssistant, "user": roleUser, "": roleUser} {
		if got := ollamaRole(in); got != want {
			t.Errorf("ollamaRole(%q) = %q, want %q", in, got, want)
		}
	}
}
