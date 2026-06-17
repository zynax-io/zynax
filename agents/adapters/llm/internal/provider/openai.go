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

// OpenAI Chat Completions streaming endpoint and SSE markers.
const (
	openAIDefaultBaseURL = "https://api.openai.com/v1"
	openAIChatPath       = "/chat/completions"
	sseDataPrefix        = "data: "
	sseDoneToken         = "[DONE]"
)

// openAIProvider streams token chunks from the OpenAI Chat Completions API over
// net/http using the Server-Sent-Events streaming protocol. The credential is
// held in a redacting Secret and only revealed onto the Authorization header at
// request time — never logged and never placed in an error message.
type openAIProvider struct {
	model     string
	maxTokens int
	baseURL   string
	secret    config.Secret
	client    *http.Client
}

// newOpenAIProvider binds OpenAI config + the resolved API key into a Provider.
func newOpenAIProvider(cfg config.ProviderConfig, secret config.Secret) *openAIProvider {
	return &openAIProvider{
		model:     cfg.Model,
		maxTokens: effectiveMaxTokens(cfg),
		baseURL:   openAIDefaultBaseURL,
		secret:    secret,
		client:    &http.Client{Timeout: 120 * time.Second},
	}
}

// openAIMessage is one role/content turn in the chat request.
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIRequest is the JSON body posted to the chat completions endpoint.
type openAIRequest struct {
	Model     string          `json:"model"`
	Messages  []openAIMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
	Stream    bool            `json:"stream"`
}

// openAIStreamChunk is one SSE frame from the streaming response.
type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// Stream posts the prompt and relays each SSE delta.content as a Chunk.
func (p *openAIProvider) Stream(ctx context.Context, prompt string) (<-chan Chunk, error) {
	body, err := json.Marshal(openAIRequest{
		Model:     p.model,
		Messages:  []openAIMessage{{Role: "user", Content: prompt}},
		MaxTokens: p.maxTokens,
		Stream:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+openAIChatPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("openai: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if !p.secret.IsZero() {
		req.Header.Set("Authorization", "Bearer "+p.secret.Reveal())
	}

	out := make(chan Chunk)
	go p.run(req, out)
	return out, nil
}

// run executes the request and pumps SSE frames into out, closing it when the
// stream ends or fails. Error messages never carry the response body or key.
func (p *openAIProvider) run(req *http.Request, out chan<- Chunk) {
	defer close(out)

	resp, err := p.client.Do(req) //nolint:gosec // URL host is operator-controlled config (provider base URL), not request input
	if err != nil {
		sendErr(out, fmt.Errorf("openai: %s", sanitiseErr(err.Error())))
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		sendErr(out, fmt.Errorf("openai: upstream returned HTTP %d", resp.StatusCode))
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, sseDataPrefix) {
			continue
		}
		payload := strings.TrimPrefix(line, sseDataPrefix)
		if payload == sseDoneToken {
			return
		}
		var frame openAIStreamChunk
		if err := json.Unmarshal([]byte(payload), &frame); err != nil {
			sendErr(out, fmt.Errorf("openai: decode stream frame: %s", sanitiseErr(err.Error())))
			return
		}
		if len(frame.Choices) > 0 && frame.Choices[0].Delta.Content != "" {
			if !send(req.Context(), out, Chunk{Text: frame.Choices[0].Delta.Content}) {
				return
			}
		}
	}
	if err := scanner.Err(); err != nil {
		sendErr(out, fmt.Errorf("openai: read stream: %s", sanitiseErr(err.Error())))
	}
}
