// SPDX-License-Identifier: Apache-2.0

// Package provider implements the LLM provider clients (OpenAI, Bedrock,
// Ollama) behind a single streaming interface. The active implementation is
// selected from config.ProviderConfig by New and is immutable after init.
//
// Providers stream per-token Chunks over a channel. Upstream failures surface
// as a Chunk with a non-nil, sanitised Err — the value the handler maps to the
// "UPSTREAM_ERROR" capability error code. Raw response bodies and credentials
// never appear in an error message (canvas M7.P safeguards).
package provider

import (
	"context"
	"fmt"

	"github.com/zynax-io/zynax/agents/adapters/llm/internal/config"
)

// maxErrMsgLen bounds any provider error message placed on a Chunk so a verbose
// upstream error can never bloat a CapabilityError.message.
const maxErrMsgLen = 512

// Chunk is one unit streamed by a Provider. Exactly one of Text / Err is set on
// any given Chunk: a token-bearing Chunk has Text and a nil Err; a terminal
// failure Chunk has an Err and empty Text. The channel closes when the stream
// ends normally.
type Chunk struct {
	// Text is the token text for this chunk (UTF-8). Empty on an error Chunk.
	Text string
	// Err is a sanitised upstream error. Non-nil only on a terminal failure
	// Chunk, after which the channel is closed and no further Chunks are sent.
	Err error
}

// Provider streams a chat completion as token Chunks. Implementations are
// stateless and safe for concurrent use; each Stream call is independent.
type Provider interface {
	// Stream sends the prompt to the upstream model and returns a channel of
	// token Chunks. The channel is closed when the stream ends (normally or
	// after a single error Chunk). Honouring ctx cancellation/deadline is
	// mandatory: a cancelled ctx must stop production promptly and close the
	// channel.
	Stream(ctx context.Context, prompt string) (<-chan Chunk, error)
}

// New builds the Provider selected by cfg.Name, binding cfg (and the resolved
// API-key Secret) at construction time. The selection is immutable: callers
// build one Provider at startup and reuse it. An unknown provider name is a
// programming error guarded by config validation, but is still rejected here.
func New(cfg config.ProviderConfig, secret config.Secret) (Provider, error) {
	switch cfg.Name {
	case "openai":
		return newOpenAIProvider(cfg, secret), nil
	case "bedrock":
		return newBedrockProvider(cfg)
	case "ollama":
		return newOllamaProvider(cfg), nil
	default:
		return nil, fmt.Errorf("provider: unknown provider %q", cfg.Name)
	}
}

// sanitiseErr truncates a message to maxErrMsgLen runes so a verbose upstream
// error never bloats a CapabilityError.message. Callers must already have
// stripped credentials and raw bodies before passing a message here.
func sanitiseErr(msg string) string {
	r := []rune(msg)
	if len(r) > maxErrMsgLen {
		return string(r[:maxErrMsgLen])
	}
	return msg
}

// effectiveMaxTokens returns a sane token ceiling: the configured value when
// positive, else a conservative default shared by all providers.
func effectiveMaxTokens(cfg config.ProviderConfig) int {
	const defaultMaxTokens = 1024
	if cfg.MaxTokens > 0 {
		return cfg.MaxTokens
	}
	return defaultMaxTokens
}

// send delivers a Chunk on out, returning false if ctx is cancelled first so a
// producer goroutine stops promptly on a deadline rather than blocking forever.
func send(ctx context.Context, out chan<- Chunk, c Chunk) bool {
	select {
	case out <- c:
		return true
	case <-ctx.Done():
		return false
	}
}

// sendErr delivers a terminal error Chunk on out. Unlike send it does not
// abandon delivery on ctx cancellation: a deadline-induced upstream failure is
// itself the terminal event the reader is waiting for, so it must always be
// delivered. The producer goroutine is the sole writer and the reader always
// drains until the channel closes, so this blocking send cannot deadlock.
func sendErr(out chan<- Chunk, err error) {
	out <- Chunk{Err: err}
}
