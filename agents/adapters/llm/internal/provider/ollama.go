// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/llm/internal/config"
)

// ollamaChatPath is the Ollama REST streaming chat endpoint.
const ollamaChatPath = "/api/chat"

// ollamaProvider streams token chunks from an Ollama server over net/http using
// the /api/chat newline-delimited-JSON streaming protocol. It needs no
// credential. The http.Client is injectable so tests drive it with httptest.
type ollamaProvider struct {
	model     string
	baseURL   string
	maxTokens int
	client    *http.Client
}

// newOllamaProvider binds Ollama config into a Provider. A nil-free default
// client with a generous read budget is used; tests replace client directly.
func newOllamaProvider(cfg config.ProviderConfig) *ollamaProvider {
	return &ollamaProvider{
		model:     cfg.Model,
		baseURL:   strings.TrimRight(cfg.OllamaBaseURL, "/"),
		maxTokens: effectiveMaxTokens(cfg),
		client:    &http.Client{Timeout: 120 * time.Second},
	}
}

// ollamaMessage is one role/content turn in the chat request.
type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaRequest is the JSON body posted to /api/chat.
type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  ollamaOptions   `json:"options"`
}

// ollamaOptions carries inference parameters Ollama accepts.
type ollamaOptions struct {
	NumPredict int `json:"num_predict"`
}

// ollamaStreamLine is one NDJSON frame from the streaming response.
type ollamaStreamLine struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

// Stream posts the prompt and relays each NDJSON message.content as a Chunk.
func (p *ollamaProvider) Stream(ctx context.Context, prompt string) (<-chan Chunk, error) {
	body, err := json.Marshal(ollamaRequest{
		Model:    p.model,
		Messages: []ollamaMessage{{Role: "user", Content: prompt}},
		Stream:   true,
		Options:  ollamaOptions{NumPredict: p.maxTokens},
	})
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+ollamaChatPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	out := make(chan Chunk)
	go p.run(req, out)
	return out, nil
}

// run executes the request and pumps the NDJSON body into out, closing it when
// the stream ends or fails. It never leaks credentials or raw bodies.
func (p *ollamaProvider) run(req *http.Request, out chan<- Chunk) {
	defer close(out)

	resp, err := p.client.Do(req) //nolint:gosec // URL host is operator-controlled config (ollama_base_url), not request input
	if err != nil {
		sendErr(out, fmt.Errorf("ollama: %s", sanitiseErr(err.Error())))
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		sendErr(out, fmt.Errorf("ollama: upstream returned HTTP %d", resp.StatusCode))
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var frame ollamaStreamLine
		if err := json.Unmarshal(line, &frame); err != nil {
			sendErr(out, fmt.Errorf("ollama: decode stream frame: %s", sanitiseErr(err.Error())))
			return
		}
		if frame.Message.Content != "" {
			if !send(req.Context(), out, Chunk{Text: frame.Message.Content}) {
				return
			}
		}
		if frame.Done {
			return
		}
	}
	if err := scanner.Err(); err != nil {
		sendErr(out, fmt.Errorf("ollama: read stream: %s", sanitiseErr(err.Error())))
	}
}
