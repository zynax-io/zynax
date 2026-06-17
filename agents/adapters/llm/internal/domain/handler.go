// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/zynax-io/zynax/agents/adapters/llm/internal/provider"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

// maxErrMsgLen bounds any CapabilityError.message so a verbose upstream or
// validation error can never bloat a terminal event (canvas M7.P safeguards).
const maxErrMsgLen = 512

// Well-known capability error codes (canvas E / parity oracle).
const (
	codeInvalidInput = "INVALID_INPUT"
	codeTimeout      = "TIMEOUT"
	codeUpstream     = "UPSTREAM_ERROR"
)

// EventSink is the minimal surface of AgentService_ExecuteCapabilityServer the
// handler needs. Defining it here keeps the handler testable without a real gRPC
// server and decouples domain logic from the transport.
type EventSink interface {
	Send(*zynaxv1.TaskEvent) error
	Context() context.Context
}

// chatInput is the validated chat_completion input payload. Only prompt is
// required; provider, model, and credentials are never read from the payload
// (canvas safeguards) — they come from static config.
type chatInput struct {
	Prompt string `json:"prompt"`
}

// ChatCompletionHandler validates input_payload against the declared JSON Schema,
// invokes the configured Provider, streams per-token PROGRESS events, and emits
// exactly one terminal event. It is stateless and safe for concurrent use.
type ChatCompletionHandler struct {
	provider provider.Provider
	schema   *jsonschema.Schema // compiled input schema; nil when none declared
}

// newChatCompletionHandler binds a Provider and compiles the declared input
// schema once at startup. An empty schema string disables structural validation
// (prompt presence is still enforced at request time).
func newChatCompletionHandler(p provider.Provider, inputSchemaJSON string) (*ChatCompletionHandler, error) {
	h := &ChatCompletionHandler{provider: p}
	if strings.TrimSpace(inputSchemaJSON) == "" {
		return h, nil
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("input.json", strings.NewReader(inputSchemaJSON)); err != nil {
		return nil, fmt.Errorf("domain: add input schema: %w", err)
	}
	sch, err := compiler.Compile("input.json")
	if err != nil {
		return nil, fmt.Errorf("domain: compile input schema: %w", err)
	}
	h.schema = sch
	return h, nil
}

// Execute runs the capability and streams TaskEvents on sink: PROGRESS per token
// (at least one), then exactly one terminal COMPLETED or FAILED event. task_id
// and a timestamp are set on every event. timeoutSeconds bounds the provider
// call via a derived context deadline. The returned error is only a transport
// (Send) failure; capability failures are delivered as a FAILED event.
func (h *ChatCompletionHandler) Execute(taskID string, payload []byte, timeoutSeconds int32, snk EventSink) error {
	s := streamSink{EventSink: snk}
	prompt, err := h.parseInput(payload)
	if err != nil {
		return s.send(failedEvent(taskID, codeInvalidInput, err.Error()))
	}

	streamCtx := snk.Context()
	if timeoutSeconds > 0 {
		var cancel context.CancelFunc
		streamCtx, cancel = context.WithTimeout(streamCtx, time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	}

	chunks, err := h.provider.Stream(streamCtx, prompt)
	if err != nil {
		return s.terminalForError(streamCtx, taskID, err, false)
	}
	return h.relay(streamCtx, s, taskID, chunks)
}

// relay pumps provider chunks to PROGRESS events and emits the terminal event.
// It guarantees at least one PROGRESS even for an empty stream so the contract
// "at least one PROGRESS before terminal" always holds.
func (h *ChatCompletionHandler) relay(streamCtx context.Context, s streamSink, taskID string, chunks <-chan provider.Chunk) error {
	var b strings.Builder
	emitted := false
	for c := range chunks {
		if c.Err != nil {
			return s.terminalForError(streamCtx, taskID, c.Err, emitted)
		}
		if err := s.send(progressEvent(taskID, []byte(c.Text))); err != nil {
			return err
		}
		b.WriteString(c.Text)
		emitted = true
	}
	if streamCtx.Err() != nil {
		return s.terminalTimeout(taskID, emitted)
	}
	if err := s.ensureProgress(taskID, emitted); err != nil {
		return err
	}
	return s.send(completedEvent(taskID, b.String()))
}

// parseInput decodes and validates payload against the declared schema, then
// extracts the prompt. Any failure maps to INVALID_INPUT.
func (h *ChatCompletionHandler) parseInput(payload []byte) (string, error) {
	var v interface{}
	if err := json.Unmarshal(payload, &v); err != nil {
		return "", fmt.Errorf("invalid JSON in input_payload: %w", err)
	}
	if h.schema != nil {
		if err := h.schema.Validate(v); err != nil {
			return "", fmt.Errorf("input_payload schema validation: %w", err)
		}
	}
	var in chatInput
	if err := json.Unmarshal(payload, &in); err != nil {
		return "", fmt.Errorf("invalid input_payload: %w", err)
	}
	if in.Prompt == "" {
		return "", fmt.Errorf("input_payload field \"prompt\" is required")
	}
	return in.Prompt, nil
}
