// SPDX-License-Identifier: Apache-2.0

// Package model provides a custom ADK model.LLM backed by a local Ollama
// endpoint. It is the zero-secret reasoning backend for the adk-adapter
// (ADR-038 §3): ADK's native providers speak the genai/Gemini wire format and
// require credentials, whereas Ollama speaks its own /api/chat format over a
// plain host. This adapter translates genai.Content <-> Ollama chat messages so
// ADK agents run against `ollama serve` with no API key — keeping `make demo`
// secret-free.
package model

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"os"
	"strings"

	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

// Defaults used when the ModelConfig leaves a field blank.
const (
	// DefaultHost is the local Ollama endpoint when neither config nor
	// OLLAMA_HOST supplies one.
	DefaultHost = "http://localhost:11434"
	// DefaultModelName is a small, widely-available coder model — the same tag
	// the code-review-rank demo pulls, so `make demo` needs no extra download.
	DefaultModelName = "qwen2.5-coder:0.5b"

	roleSystem     = "system"
	roleUser       = "user"
	roleAssistant  = "assistant"
	genaiRoleModel = "model"

	maxErrBody = 2048
)

// Ollama implements adk model.LLM over an Ollama /api/chat endpoint. It is
// stateless and safe for concurrent use across capabilities.
type Ollama struct {
	host  string
	name  string
	httpc *http.Client
}

// NewOllama builds an Ollama model. host falls back to $OLLAMA_HOST then
// DefaultHost; name falls back to DefaultModelName. A bare host:port (as Ollama
// itself accepts in OLLAMA_HOST) is normalised to an http:// URL.
func NewOllama(host, name string) *Ollama {
	if host == "" {
		host = os.Getenv("OLLAMA_HOST")
	}
	if host == "" {
		host = DefaultHost
	}
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	host = strings.TrimRight(host, "/")
	if name == "" {
		name = DefaultModelName
	}
	return &Ollama{host: host, name: name, httpc: &http.Client{}}
}

// Name reports the model tag, satisfying model.LLM.
func (o *Ollama) Name() string { return o.name }

// chatMessage / chatRequest / chatResponse mirror the subset of the Ollama
// /api/chat wire format this adapter uses.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string         `json:"model"`
	Messages []chatMessage  `json:"messages"`
	Stream   bool           `json:"stream"`
	Options  map[string]any `json:"options,omitempty"`
}

type chatResponse struct {
	Message chatMessage `json:"message"`
	Done    bool        `json:"done"`
	Error   string      `json:"error,omitempty"`
}

// GenerateContent satisfies model.LLM. It POSTs the translated conversation to
// /api/chat and yields model.LLMResponse values: for stream=true, one Partial
// response per delta followed by a final non-Partial (TurnComplete) response
// carrying the full text; for stream=false, a single final response. Any
// transport, status, or decode failure is yielded as an error.
func (o *Ollama) GenerateContent(ctx context.Context, req *adkmodel.LLMRequest, stream bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		body, err := json.Marshal(chatRequest{
			Model:    o.modelFor(req),
			Messages: toOllamaMessages(req),
			Stream:   stream,
			Options:  optionsFor(req),
		})
		if err != nil {
			yield(nil, fmt.Errorf("ollama: marshal request: %w", err))
			return
		}
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.host+"/api/chat", bytes.NewReader(body))
		if err != nil {
			yield(nil, fmt.Errorf("ollama: new request: %w", err))
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := o.httpc.Do(httpReq)
		if err != nil {
			yield(nil, fmt.Errorf("ollama: chat request: %w", err))
			return
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			// Do not surface the raw upstream body: it is untrusted external bytes
			// that must not flow into a CapabilityError.message (adapter rule).
			// The status code is enough to classify; the body stays in Ollama's logs.
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrBody))
			yield(nil, fmt.Errorf("ollama: chat status %d", resp.StatusCode))
			return
		}

		if stream {
			streamChat(resp.Body, yield)
			return
		}
		var cr chatResponse
		if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
			yield(nil, fmt.Errorf("ollama: decode response: %w", err))
			return
		}
		if cr.Error != "" {
			yield(nil, fmt.Errorf("ollama: %s", cr.Error))
			return
		}
		yield(finalResponse(cr.Message.Content), nil)
	}
}

// streamChat decodes an NDJSON /api/chat stream, yielding a Partial response per
// non-empty delta and a final non-Partial response with the accumulated text.
func streamChat(body io.Reader, yield func(*adkmodel.LLMResponse, error) bool) {
	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var full strings.Builder
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var cr chatResponse
		if err := json.Unmarshal(line, &cr); err != nil {
			yield(nil, fmt.Errorf("ollama: decode stream chunk: %w", err))
			return
		}
		if cr.Error != "" {
			yield(nil, fmt.Errorf("ollama: %s", cr.Error))
			return
		}
		if cr.Done {
			// The done frame may itself carry the tail of the generation (some
			// servers flush it alongside the done marker); include it so the
			// aggregated final text is never truncated. Empty content (the
			// canonical case) is a no-op.
			if tail := cr.Message.Content; tail != "" {
				full.WriteString(tail)
				if !yield(partialResponse(tail), nil) {
					return
				}
			}
			yield(finalResponse(full.String()), nil)
			return
		}
		delta := cr.Message.Content
		if delta == "" {
			continue
		}
		full.WriteString(delta)
		if !yield(partialResponse(delta), nil) {
			return
		}
	}
	if err := sc.Err(); err != nil {
		yield(nil, fmt.Errorf("ollama: read stream: %w", err))
		return
	}
	// Stream ended without an explicit done marker — surface what we have.
	yield(finalResponse(full.String()), nil)
}

// modelFor prefers an explicit per-request model, else the configured tag.
func (o *Ollama) modelFor(req *adkmodel.LLMRequest) string {
	if req != nil && req.Model != "" {
		return req.Model
	}
	return o.name
}

// optionsFor forwards the sampling knobs ADK sets that Ollama understands.
func optionsFor(req *adkmodel.LLMRequest) map[string]any {
	if req == nil || req.Config == nil || req.Config.Temperature == nil {
		return nil
	}
	return map[string]any{"temperature": *req.Config.Temperature}
}

// toOllamaMessages flattens the system instruction and the genai.Content history
// into Ollama chat messages, mapping the genai "model" role to "assistant".
func toOllamaMessages(req *adkmodel.LLMRequest) []chatMessage {
	if req == nil {
		return nil
	}
	var msgs []chatMessage
	if req.Config != nil && req.Config.SystemInstruction != nil {
		if s := contentText(req.Config.SystemInstruction); s != "" {
			msgs = append(msgs, chatMessage{Role: roleSystem, Content: s})
		}
	}
	for _, c := range req.Contents {
		text := contentText(c)
		if text == "" {
			continue
		}
		msgs = append(msgs, chatMessage{Role: ollamaRole(c.Role), Content: text})
	}
	return msgs
}

// contentText concatenates the text parts of a genai.Content (non-text parts —
// function calls, media — are not supported by this zero-tool backend).
func contentText(c *genai.Content) string {
	if c == nil {
		return ""
	}
	var b strings.Builder
	for _, p := range c.Parts {
		if p != nil && p.Text != "" {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

func ollamaRole(genaiRole string) string {
	if genaiRole == genaiRoleModel || genaiRole == roleAssistant {
		return roleAssistant
	}
	return roleUser
}

func partialResponse(delta string) *adkmodel.LLMResponse {
	return &adkmodel.LLMResponse{Content: modelContent(delta), Partial: true}
}

func finalResponse(text string) *adkmodel.LLMResponse {
	return &adkmodel.LLMResponse{Content: modelContent(text), Partial: false, TurnComplete: true}
}

func modelContent(text string) *genai.Content {
	return &genai.Content{Role: genaiRoleModel, Parts: []*genai.Part{genai.NewPartFromText(text)}}
}
