// SPDX-License-Identifier: Apache-2.0

// Package client provides an HTTP client for the Zynax api-gateway.
package client

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ErrNotFound is returned when the api-gateway responds with HTTP 404.
var ErrNotFound = errors.New("zynax: not found")

// WorkflowStatus is the api-gateway's workflow run summary.
type WorkflowStatus struct {
	RunID        string `json:"run_id"`
	WorkflowID   string `json:"workflow_id"`
	Status       string `json:"status"`
	CurrentState string `json:"current_state"`
	// Version is the manifest's metadata.version (SemVer) when set. Optional and
	// backward-compatible: older gateways omit it and the CLI hides it then.
	Version string `json:"version,omitempty"`
}

// CompileError is a single diagnostic from the workflow compiler.
type CompileError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Line    int32  `json:"line,omitempty"`
}

// Gateway is an HTTP client for the Zynax api-gateway REST API.
type Gateway struct {
	base   string
	client *http.Client
}

// New creates a Gateway pointing at baseURL.
// When insecure is true TLS certificate verification is skipped (local dev only).
func New(baseURL string, insecure bool) *Gateway {
	tr := http.DefaultTransport
	if insecure {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // G402: intentional for --insecure local-dev flag; never used in production
		}
	}
	return &Gateway{
		base:   strings.TrimRight(baseURL, "/"),
		client: &http.Client{Transport: tr},
	}
}

// Apply submits a manifest for execution. Returns run_id (Workflow) or
// agent_id (AgentDef) and any compiler warnings on success.
// Returns a descriptive error for 4xx/5xx responses.
func (g *Gateway) Apply(ctx context.Context, body []byte, engineHint string) (string, string, []string, error) {
	q := url.Values{}
	if engineHint != "" {
		q.Set("engine", engineHint)
	}
	resp, err := g.post(ctx, "/api/v1/apply", q, body)
	if err != nil {
		return "", "", nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusAccepted:
		var r struct {
			RunID    string   `json:"run_id"`
			Warnings []string `json:"warnings"`
		}
		if err := json.Unmarshal(raw, &r); err != nil {
			return "", "", nil, fmt.Errorf("zynax: decode apply response: %w", err)
		}
		return r.RunID, "", r.Warnings, nil
	case http.StatusCreated:
		var r struct {
			AgentID string `json:"agent_id"`
		}
		if err := json.Unmarshal(raw, &r); err != nil {
			return "", "", nil, fmt.Errorf("zynax: decode apply response: %w", err)
		}
		return "", r.AgentID, nil, nil
	case http.StatusUnprocessableEntity:
		var r struct {
			Errors []CompileError `json:"errors"`
		}
		_ = json.Unmarshal(raw, &r)
		return "", "", nil, fmt.Errorf("zynax: compilation failed with %d error(s)", len(r.Errors))
	default:
		return "", "", nil, fmt.Errorf("zynax: apply: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
}

// ApplyDryRun validates a manifest without submitting.
// Returns (nil, warnings, nil) when valid; (errors, nil, nil) when the
// compiler finds issues; (nil, nil, err) for transport/server errors.
func (g *Gateway) ApplyDryRun(ctx context.Context, body []byte) ([]CompileError, []string, error) {
	q := url.Values{"dry_run": []string{"true"}}
	resp, err := g.post(ctx, "/api/v1/apply", q, body)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		var r struct {
			Warnings []string `json:"warnings"`
		}
		_ = json.Unmarshal(raw, &r)
		return nil, r.Warnings, nil
	case http.StatusUnprocessableEntity:
		var r struct {
			Errors []CompileError `json:"errors"`
		}
		_ = json.Unmarshal(raw, &r)
		return r.Errors, nil, nil
	default:
		return nil, nil, fmt.Errorf("zynax: dry-run: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
}

// Health probes the api-gateway liveness endpoint GET /healthz (issue #1380:
// returns 200 + a {"status":"..."} JSON body when serving, 503 when not). It
// returns the reported status string on success, or a descriptive error when
// the gateway is unreachable or unhealthy. Used by `zynax doctor`.
func (g *Gateway) Health(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.base+"/healthz", nil)
	if err != nil {
		return "", fmt.Errorf("zynax: build request: %w", err)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("zynax: health: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("zynax: health: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	// Body is {"status":"ok"} since #1380; tolerate an empty body too.
	var s struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(raw, &s)
	if s.Status == "" {
		s.Status = "ok"
	}
	return s.Status, nil
}

// Ready probes the api-gateway readiness endpoint GET /readyz (200 when the
// gateway's dependencies are reachable, 503 otherwise; no body either way).
// Returns nil when ready, a descriptive error otherwise. Used by `zynax doctor`.
func (g *Gateway) Ready(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.base+"/readyz", nil)
	if err != nil {
		return fmt.Errorf("zynax: build request: %w", err)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("zynax: ready: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("zynax: ready: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

// GetWorkflow returns the current status of a workflow run.
func (g *Gateway) GetWorkflow(ctx context.Context, runID string) (*WorkflowStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.base+"/api/v1/workflows/"+runID, nil)
	if err != nil {
		return nil, fmt.Errorf("zynax: build request: %w", err)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zynax: get workflow: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		var s WorkflowStatus
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, fmt.Errorf("zynax: decode workflow status: %w", err)
		}
		return &s, nil
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		return nil, fmt.Errorf("zynax: get workflow: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
}

// DeleteWorkflow cancels a running workflow run.
func (g *Gateway) DeleteWorkflow(ctx context.Context, runID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, g.base+"/api/v1/workflows/"+runID, nil)
	if err != nil {
		return fmt.Errorf("zynax: build request: %w", err)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("zynax: delete workflow: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return ErrNotFound
	default:
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("zynax: delete workflow: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
}

// PublishEvent injects a business/lifecycle event into the run identified by
// runID so an event-driven workflow can advance. data is forwarded verbatim as
// the CloudEvent payload (may be nil). Returns the bus-assigned event_id.
func (g *Gateway) PublishEvent(ctx context.Context, runID, eventType string, data map[string]string) (string, error) {
	payload := struct {
		EventType string            `json:"event_type"`
		Data      map[string]string `json:"data,omitempty"`
	}{EventType: eventType, Data: data}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("zynax: encode event: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.base+"/api/v1/workflows/"+runID+"/events", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("zynax: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("zynax: publish event: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusAccepted, http.StatusOK:
		var r struct {
			EventID string `json:"event_id"`
		}
		_ = json.Unmarshal(raw, &r)
		return r.EventID, nil
	case http.StatusNotFound:
		return "", ErrNotFound
	default:
		return "", fmt.Errorf("zynax: publish event: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
}

// LogEvent is a single workflow lifecycle event received from the SSE stream.
type LogEvent struct {
	RunID     string `json:"run_id"`
	EventType string `json:"event_type"`
	FromState string `json:"from_state,omitempty"`
	ToState   string `json:"to_state,omitempty"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp,omitempty"`
	Payload   string `json:"payload,omitempty"`
}

// CompletionText extracts the capability result text from a log event's JSON
// payload. Task-broker completion events carry the result under a nested
// `result_payload` string whose value is itself JSON ({"completion": "..."});
// this unwraps one level and returns the `completion` field. It also accepts a
// bare {"completion": "..."} payload. Returns "" when no completion is present
// or the payload is not JSON, so callers can skip empty results silently.
func CompletionText(payload string) string {
	if payload == "" {
		return ""
	}
	var outer map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &outer); err != nil {
		return ""
	}
	// Capability events wrap the executor output in a result_payload string.
	if rp, ok := outer["result_payload"]; ok {
		var inner string
		if err := json.Unmarshal(rp, &inner); err == nil {
			return completionField(inner)
		}
	}
	if c, ok := outer["completion"]; ok {
		var s string
		if err := json.Unmarshal(c, &s); err == nil {
			return s
		}
	}
	return ""
}

// completionField parses a {"completion": "..."} JSON object and returns the
// completion string, or "" when absent or unparseable.
func completionField(s string) string {
	var inner struct {
		Completion string `json:"completion"`
	}
	if err := json.Unmarshal([]byte(s), &inner); err != nil {
		return ""
	}
	return inner.Completion
}

// WatchWorkflowLogs streams SSE events from GET /api/v1/workflows/{id}/logs,
// calling send for each event. Returns when the stream closes or ctx is cancelled.
func (g *Gateway) WatchWorkflowLogs(ctx context.Context, runID string, send func(LogEvent) error) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.base+"/api/v1/workflows/"+runID+"/logs", nil)
	if err != nil {
		return fmt.Errorf("zynax: build request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("zynax: watch workflow: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("zynax: watch workflow: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return readSSEStream(resp.Body, send)
}

func readSSEStream(r io.Reader, send func(LogEvent) error) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev LogEvent
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err != nil {
			return fmt.Errorf("zynax: decode SSE event: %w", err)
		}
		if err := send(ev); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (g *Gateway) post(ctx context.Context, path string, q url.Values, body []byte) (*http.Response, error) {
	u := g.base + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("zynax: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/yaml")
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zynax: post %s: %w", path, err)
	}
	return resp, nil
}
