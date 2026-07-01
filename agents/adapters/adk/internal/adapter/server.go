// SPDX-License-Identifier: Apache-2.0

// Package adapter implements the AgentService gRPC contract for the adk-adapter.
//
// ExecuteCapability is the ADR-038 bridge (S3, #1479): it validates input_payload
// against the capability's input schema, runs the ADK Runner for that capability
// (session keyed by workflow_id), and maps the Runner's *session.Event stream
// onto TaskEvents — a PROGRESS per streamed chunk, then exactly one terminal
// COMPLETED (final text coerced to the output schema) or FAILED (classified
// CapabilityError). timeout_seconds bounds the run via a context deadline.
// GetCapabilitySchema returns the declared schemas unchanged.
package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/zynax-io/zynax/agents/adapters/adk/internal/adk"
	"github.com/zynax-io/zynax/agents/adapters/adk/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/adk/internal/model"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// maxErrMsgLen bounds any CapabilityError.message so a verbose model or
// validation error can never bloat a terminal event.
const maxErrMsgLen = 512

// Well-known capability error codes (canvas E / parity oracle, shared with the
// other AI adapters).
const (
	codeInvalidInput = "INVALID_INPUT"
	codeTimeout      = "TIMEOUT"
	codeUpstream     = "UPSTREAM_ERROR"
)

// promptField is the required input_payload field carrying the user turn. It
// mirrors the llm-adapter "parity oracle": provider/model/credentials are never
// read from the payload — only the prompt is.
const promptField = "prompt"

// capRuntime is the immutable per-capability execution unit built once at
// startup: compiled schemas, the coercion target key, and the ADK Runner.
type capRuntime struct {
	cfg       config.CapabilityConfig
	inSchema  *jsonschema.Schema // compiled input schema; nil when none declared
	outSchema *jsonschema.Schema // compiled output schema; nil when none declared
	outKey    string             // string property to wrap raw text under; "" -> raw
	runner    *runner.Runner
}

// AgentServer implements AgentServiceServer. Its capability table is built once
// at construction from the validated config and is immutable thereafter.
type AgentServer struct {
	zynaxv1.UnimplementedAgentServiceServer
	appName string
	userID  string
	caps    map[string]*capRuntime
}

// NewAgentServer builds an AgentServer from a validated AdapterConfig, wiring one
// ADK Runner per capability over a shared, zero-secret Ollama model. It returns
// an error if any schema fails to compile or any agent/runner fails to build.
func NewAgentServer(cfg *config.AdapterConfig) (*AgentServer, error) {
	llm := model.NewOllama(cfg.Model.Host, cfg.Model.Name)
	return newAgentServer(cfg, llm)
}

// newAgentServer is the injectable core of NewAgentServer: tests pass a fake
// model.LLM to drive the real ADK Runner without an Ollama endpoint.
func newAgentServer(cfg *config.AdapterConfig, llm adkmodel.LLM) (*AgentServer, error) {
	sess := session.InMemoryService()
	appName := cfg.Name
	if appName == "" {
		appName = "adk-adapter"
	}
	userID := cfg.AgentID
	if userID == "" {
		userID = appName
	}

	caps := make(map[string]*capRuntime, len(cfg.Capabilities))
	for _, c := range cfg.Capabilities {
		inSchema, err := compileSchema(c.InputSchemaJSON)
		if err != nil {
			return nil, fmt.Errorf("adapter: capability %q input schema: %w", c.Name, err)
		}
		outSchema, err := compileSchema(c.OutputSchemaJSON)
		if err != nil {
			return nil, fmt.Errorf("adapter: capability %q output schema: %w", c.Name, err)
		}
		r, err := adk.NewRunner(appName, adk.AgentSpec{
			Name:        c.Name,
			Description: c.Description,
			Instruction: c.Instruction,
		}, llm, sess)
		if err != nil {
			return nil, fmt.Errorf("adapter: capability %q: %w", c.Name, err)
		}
		caps[c.Name] = &capRuntime{
			cfg:       c,
			inSchema:  inSchema,
			outSchema: outSchema,
			outKey:    outputTargetKey(c.OutputSchemaJSON),
			runner:    r,
		}
	}
	return &AgentServer{appName: appName, userID: userID, caps: caps}, nil
}

// ExecuteCapability runs the ADK bridge and streams TaskEvents. task_id and
// capability_name are required (INVALID_ARGUMENT gRPC error). An unknown
// capability, malformed input, or schema-invalid payload yields a terminal
// FAILED INVALID_INPUT; a deadline yields TIMEOUT; a model/runner failure yields
// UPSTREAM_ERROR. A successful run streams >=1 PROGRESS then one COMPLETED.
func (s *AgentServer) ExecuteCapability(req *zynaxv1.ExecuteCapabilityRequest, stream zynaxv1.AgentService_ExecuteCapabilityServer) error {
	if req.TaskId == "" {
		return status.Error(codes.InvalidArgument, "task_id is required")
	}
	if req.CapabilityName == "" {
		return status.Error(codes.InvalidArgument, "capability_name is required")
	}
	rt, ok := s.caps[req.CapabilityName]
	if !ok {
		return sendFailed(stream, req.TaskId, codeInvalidInput, fmt.Sprintf("unknown capability: %s", req.CapabilityName))
	}

	prompt, err := parseInput(req.InputPayload, rt.inSchema)
	if err != nil {
		return sendFailed(stream, req.TaskId, codeInvalidInput, err.Error())
	}

	ctx := stream.Context()
	if req.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	sessionID := req.WorkflowId
	if sessionID == "" {
		sessionID = req.TaskId
	}
	msg := &genai.Content{Role: "user", Parts: []*genai.Part{genai.NewPartFromText(prompt)}}

	runCfg := agent.RunConfig{StreamingMode: agent.StreamingModeSSE}
	return bridge(ctx, stream, req.TaskId, rt, rt.runner.Run(ctx, s.userID, sessionID, msg, runCfg))
}

// bridge maps an ADK event stream onto TaskEvents: a PROGRESS per streamed chunk,
// then one terminal COMPLETED (final text coerced to the output schema) or a
// classified FAILED. It guarantees at least one PROGRESS before COMPLETED. It is
// pure over the seq, so tests drive it with a fabricated stream.
func bridge(ctx context.Context, stream zynaxv1.AgentService_ExecuteCapabilityServer, taskID string, rt *capRuntime, seq iter.Seq2[*session.Event, error]) error {
	var finalText string
	emittedProgress := false
	for ev, evErr := range seq {
		if evErr != nil {
			return sendFailed(stream, taskID, classify(ctx, evErr), evErr.Error())
		}
		text := eventText(ev)
		if isPartial(ev) {
			if text == "" {
				continue
			}
			if err := sendProgress(stream, taskID, []byte(text)); err != nil {
				return err
			}
			emittedProgress = true
			continue
		}
		if text != "" {
			finalText = text
		}
	}
	if ctx.Err() != nil {
		return sendFailed(stream, taskID, codeTimeout, "request exceeded timeout")
	}
	// Guarantee the contract "at least one PROGRESS before terminal" even when the
	// model returned a single non-streamed turn.
	if !emittedProgress {
		if err := sendProgress(stream, taskID, []byte(finalText)); err != nil {
			return err
		}
	}
	payload, ok := coerceOutput(finalText, rt)
	if !ok {
		return sendFailed(stream, taskID, codeUpstream, "model output did not satisfy the declared output_schema")
	}
	return sendCompleted(stream, taskID, payload)
}

// GetCapabilitySchema returns the JSON Schema for a named capability.
func (s *AgentServer) GetCapabilitySchema(_ context.Context, req *zynaxv1.GetCapabilitySchemaRequest) (*zynaxv1.GetCapabilitySchemaResponse, error) {
	if req.CapabilityName == "" {
		return nil, status.Error(codes.InvalidArgument, "capability_name is required")
	}
	rt, ok := s.caps[req.CapabilityName]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "unknown capability: %s", req.CapabilityName)
	}
	return &zynaxv1.GetCapabilitySchemaResponse{
		CapabilityName:   rt.cfg.Name,
		InputSchemaJson:  rt.cfg.InputSchemaJSON,
		OutputSchemaJson: rt.cfg.OutputSchemaJSON,
		Description:      rt.cfg.Description,
	}, nil
}

// parseInput decodes payload, validates it against the declared schema, and
// extracts the prompt. Any failure is an INVALID_INPUT-worthy error.
func parseInput(payload []byte, schema *jsonschema.Schema) (string, error) {
	var v any
	if err := json.Unmarshal(payload, &v); err != nil {
		return "", fmt.Errorf("invalid JSON in input_payload: %w", err)
	}
	if schema != nil {
		if err := schema.Validate(v); err != nil {
			return "", fmt.Errorf("input_payload schema validation: %w", err)
		}
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return "", fmt.Errorf("input_payload must be a JSON object")
	}
	prompt, ok := obj[promptField].(string)
	if !ok || prompt == "" {
		return "", fmt.Errorf("input_payload field %q (string) is required", promptField)
	}
	return prompt, nil
}

// coerceOutput shapes the model's final text to the declared output schema and
// reports whether the result satisfies it. No schema -> raw bytes, ok. With a
// schema: text that already parses as a schema-valid JSON object is passed
// through; otherwise the text is wrapped under the schema's target string key.
// ok is true only when the produced bytes validate against the schema — the
// caller turns a false into a terminal FAILED, so a COMPLETED payload always
// honors the declared output_schema (the BDD invariant).
func coerceOutput(text string, rt *capRuntime) (payload []byte, ok bool) {
	if rt.outSchema == nil {
		return []byte(text), true
	}
	if trimmed := strings.TrimSpace(text); strings.HasPrefix(trimmed, "{") {
		var v any
		if json.Unmarshal([]byte(trimmed), &v) == nil && rt.outSchema.Validate(v) == nil {
			return []byte(trimmed), true
		}
	}
	if rt.outKey != "" {
		if b, err := json.Marshal(map[string]string{rt.outKey: text}); err == nil && validates(rt.outSchema, b) {
			return b, true
		}
	}
	return []byte(text), false
}

// validates reports whether payload is a JSON document satisfying schema.
func validates(schema *jsonschema.Schema, payload []byte) bool {
	var v any
	if json.Unmarshal(payload, &v) != nil {
		return false
	}
	return schema.Validate(v) == nil
}

// compileSchema compiles a JSON Schema string, returning (nil, nil) for an empty
// schema (structural validation disabled).
func compileSchema(schemaJSON string) (*jsonschema.Schema, error) {
	if strings.TrimSpace(schemaJSON) == "" {
		return nil, nil
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", strings.NewReader(schemaJSON)); err != nil {
		return nil, fmt.Errorf("add schema: %w", err)
	}
	sch, err := c.Compile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	return sch, nil
}

// outputTargetKey picks the string property the raw model text is wrapped under
// when coercing: the first required string property, else the first string
// property, else "" (no wrapping). It parses the schema leniently — a schema it
// cannot read simply yields "".
func outputTargetKey(schemaJSON string) string {
	if strings.TrimSpace(schemaJSON) == "" {
		return ""
	}
	var s struct {
		Properties map[string]struct {
			Type string `json:"type"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	if json.Unmarshal([]byte(schemaJSON), &s) != nil {
		return ""
	}
	for _, k := range s.Required {
		if p, ok := s.Properties[k]; ok && p.Type == "string" {
			return k
		}
	}
	for k, p := range s.Properties {
		if p.Type == "string" {
			return k
		}
	}
	return ""
}

// classify maps a Runner error to a capability error code: a deadline is
// TIMEOUT; anything else surfaced during the run is an upstream model failure.
func classify(ctx context.Context, err error) string {
	if errors.Is(err, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded {
		return codeTimeout
	}
	return codeUpstream
}

// eventText concatenates the text parts of an ADK event's content.
func eventText(ev *session.Event) string {
	if ev == nil || ev.Content == nil {
		return ""
	}
	var b strings.Builder
	for _, p := range ev.Content.Parts {
		if p != nil && p.Text != "" {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

func isPartial(ev *session.Event) bool { return ev != nil && ev.Partial }

// --- TaskEvent emitters (each sets task_id + timestamp) ---

func sendProgress(stream zynaxv1.AgentService_ExecuteCapabilityServer, taskID string, payload []byte) error {
	return stream.Send(&zynaxv1.TaskEvent{ //nolint:wrapcheck // direct stream send; gRPC error surfaced as-is
		TaskId:    taskID,
		EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS,
		Payload:   payload,
		Timestamp: timestamppb.Now(),
	})
}

func sendCompleted(stream zynaxv1.AgentService_ExecuteCapabilityServer, taskID string, payload []byte) error {
	return stream.Send(&zynaxv1.TaskEvent{ //nolint:wrapcheck // direct stream send; gRPC error surfaced as-is
		TaskId:    taskID,
		EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED,
		Payload:   payload,
		Timestamp: timestamppb.Now(),
	})
}

// sendFailed emits exactly one terminal FAILED TaskEvent echoing task_id, with a
// rune-bounded message.
func sendFailed(stream zynaxv1.AgentService_ExecuteCapabilityServer, taskID, code, msg string) error {
	return stream.Send(&zynaxv1.TaskEvent{ //nolint:wrapcheck // direct stream send; gRPC error surfaced as-is
		TaskId:    taskID,
		EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED,
		Timestamp: timestamppb.Now(),
		Error:     &zynaxv1.CapabilityError{Code: code, Message: boundMessage(msg)},
	})
}

// boundMessage truncates by runes (never mid-rune), so the result is always valid
// UTF-8 — a byte-boundary cut could produce an invalid proto3 string that makes
// stream.Send fail and drop the terminal event.
func boundMessage(msg string) string {
	if len(msg) <= maxErrMsgLen {
		return msg
	}
	r := []rune(msg)
	if len(r) > maxErrMsgLen {
		r = r[:maxErrMsgLen]
	}
	return string(r)
}
