// SPDX-License-Identifier: Apache-2.0

// Package client provides an HTTP client for the Zynax api-gateway.
package client

import (
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

// Sentinel errors returned by Gateway methods.
var ErrNotFound = errors.New("zynax: not found")

// WorkflowStatus is the api-gateway's workflow run summary.
type WorkflowStatus struct {
	RunID        string `json:"run_id"`
	WorkflowID   string `json:"workflow_id"`
	Status       string `json:"status"`
	CurrentState string `json:"current_state"`
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
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
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
