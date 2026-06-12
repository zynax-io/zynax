// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides concrete implementations of domain ports backed by
// external services and SDKs. Only this package may import engine-specific dependencies
// such as the Temporal Go SDK (ADR-015).
package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ArgoWorkflow is a minimal representation of an Argo Workflows Workflow resource
// submitted via the Argo Workflows REST API.
type ArgoWorkflow struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Metadata   ArgoObjectMeta     `json:"metadata"`
	Spec       ArgoWorkflowSpec   `json:"spec"`
	Status     ArgoWorkflowStatus `json:"status,omitempty"`
}

// ArgoWorkflowStatus holds the runtime status returned by the Argo Workflows API.
// Only the fields needed for GetStatus and Watch are mapped here.
type ArgoWorkflowStatus struct {
	// Phase is the overall phase of the workflow: Pending, Running, Succeeded,
	// Failed, Error, or Skipped.
	Phase string `json:"phase,omitempty"`

	// Message contains a human-readable error or completion message.
	Message string `json:"message,omitempty"`

	// StartedAt is the RFC3339 timestamp when the workflow entered Running.
	StartedAt string `json:"startedAt,omitempty"`

	// FinishedAt is the RFC3339 timestamp when the workflow reached a terminal phase.
	FinishedAt string `json:"finishedAt,omitempty"`
}

// Argo phase constants mirror the Argo Workflows NodePhase string values.
// These are the only values the Argo REST API returns for Workflow.Status.Phase.
const (
	ArgoPhaseRunning   = "Running"
	ArgoPhasePending   = "Pending"
	ArgoPhaseSucceeded = "Succeeded"
	ArgoPhaseFailed    = "Failed"
	ArgoPhaseError     = "Error"
	ArgoPhaseSkipped   = "Skipped"
)

// ArgoObjectMeta holds the identifying metadata for an Argo resource.
type ArgoObjectMeta struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ArgoWorkflowSpec holds the specification for an Argo Workflow resource.
// WorkflowTemplateRef points to a pre-installed WorkflowTemplate in the namespace.
type ArgoWorkflowSpec struct {
	WorkflowTemplateRef *ArgoWorkflowTemplateRef `json:"workflowTemplateRef,omitempty"`
	Arguments           *ArgoArguments           `json:"arguments,omitempty"`
	ServiceAccountName  string                   `json:"serviceAccountName,omitempty"`
}

// ArgoWorkflowTemplateRef references a named WorkflowTemplate.
type ArgoWorkflowTemplateRef struct {
	Name string `json:"name"`
}

// ArgoArguments holds the top-level workflow arguments (parameters).
type ArgoArguments struct {
	Parameters []ArgoParameter `json:"parameters,omitempty"`
}

// ArgoParameter is a single name/value pair passed to a WorkflowTemplate.
type ArgoParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ArgoWorkflowCreateRequest is the envelope the Argo Workflows server API
// requires on POST /api/v1/workflows/{namespace} (WorkflowService_CreateWorkflow,
// definition io.argoproj.workflow.v1alpha1.WorkflowCreateRequest). Posting a
// bare Workflow manifest is rejected with HTTP 422 (#1157).
//
// The upstream schema also defines createOptions, instanceID (deprecated), and
// serverDryRun; they are intentionally omitted here because the engine never
// sets them.
type ArgoWorkflowCreateRequest struct {
	Namespace string        `json:"namespace,omitempty"`
	Workflow  *ArgoWorkflow `json:"workflow"`
}

// errArgoNotFound is a sentinel error used by httpArgoClient to signal 404 responses.
// ArgoEngine wraps this in domain.ErrExecutionNotFound.
var errArgoNotFound = errors.New("workflow not found")

// ArgoClient is the port that ArgoEngine uses to communicate with the Argo
// Workflows REST API. It is defined as an interface so ArgoEngine can be
// tested with a mock without a live Argo server.
//
// Implementations must be safe for concurrent use.
type ArgoClient interface {
	// SubmitWorkflow creates an Argo Workflow resource in the given namespace.
	SubmitWorkflow(ctx context.Context, namespace string, wf *ArgoWorkflow) error

	// SendEvent delivers an external event to a running workflow.
	// discriminator must match the WorkflowEventBinding selector configured in
	// the Argo Workflow resource.
	SendEvent(ctx context.Context, namespace, discriminator string, payload []byte) error

	// GetWorkflow retrieves an existing Argo Workflow resource by name.
	// Returns (nil, errArgoNotFound) if the workflow does not exist.
	GetWorkflow(ctx context.Context, namespace, name string) (*ArgoWorkflow, error)

	// DeleteWorkflow deletes an Argo Workflow resource by name, which causes Argo
	// to stop all running pods.
	DeleteWorkflow(ctx context.Context, namespace, name string) error
}

// httpArgoClient is the production ArgoClient that talks to the Argo Workflows
// REST API over HTTP/HTTPS. The server URL and token are read from config/env
// at construction time — never hardcoded.
type httpArgoClient struct {
	serverURL  string // Argo Workflows server base URL; read from env at startup
	token      string // Bearer token; empty if the server is unauthenticated
	httpClient *http.Client
}

// NewHTTPArgoClient constructs a production ArgoClient.
// serverURL must not have a trailing slash.
// token may be empty for unauthenticated Argo servers (local dev only).
func NewHTTPArgoClient(serverURL, token string, httpClient *http.Client) ArgoClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &httpArgoClient{
		serverURL:  serverURL,
		token:      token,
		httpClient: httpClient,
	}
}

// SubmitWorkflow POSTs the Argo Workflow resource to the Argo REST API,
// wrapped in the WorkflowCreateRequest envelope the server expects.
func (c *httpArgoClient) SubmitWorkflow(ctx context.Context, namespace string, wf *ArgoWorkflow) error {
	body, err := json.Marshal(ArgoWorkflowCreateRequest{
		Namespace: namespace,
		Workflow:  wf,
	})
	if err != nil {
		return fmt.Errorf("argo_client: marshal workflow create request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/workflows/%s", c.serverURL, namespace)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("argo_client: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("argo_client: submit workflow %q: %w", wf.Metadata.Name, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("argo_client: submit workflow %q: HTTP %d: %s", wf.Metadata.Name, resp.StatusCode, string(b))
	}
	return nil
}

// GetWorkflow retrieves an Argo Workflow resource via the Argo REST API.
// Returns a wrapped errArgoNotFound on HTTP 404.
func (c *httpArgoClient) GetWorkflow(ctx context.Context, namespace, name string) (*ArgoWorkflow, error) {
	url := fmt.Sprintf("%s/api/v1/workflows/%s/%s", c.serverURL, namespace, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("argo_client: create get-workflow request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("argo_client: get workflow %q: %w", name, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("argo_client: get workflow %q: %w", name, errArgoNotFound)
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("argo_client: get workflow %q: HTTP %d: %s", name, resp.StatusCode, string(b))
	}

	var wf ArgoWorkflow
	if err := json.NewDecoder(resp.Body).Decode(&wf); err != nil {
		return nil, fmt.Errorf("argo_client: decode workflow %q: %w", name, err)
	}
	return &wf, nil
}

// DeleteWorkflow sends a DELETE request to the Argo REST API to stop a workflow.
// Returns a wrapped errArgoNotFound on HTTP 404.
func (c *httpArgoClient) DeleteWorkflow(ctx context.Context, namespace, name string) error {
	url := fmt.Sprintf("%s/api/v1/workflows/%s/%s", c.serverURL, namespace, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("argo_client: create delete-workflow request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("argo_client: delete workflow %q: %w", name, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("argo_client: delete workflow %q: %w", name, errArgoNotFound)
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("argo_client: delete workflow %q: HTTP %d: %s", name, resp.StatusCode, string(b))
	}
	return nil
}

// SendEvent POSTs an external event payload to the Argo Workflows event endpoint.
// Per the Argo server API (EventService_ReceiveEvent), the request body IS the
// event payload itself (io.argoproj.workflow.v1alpha1.Item — arbitrary JSON),
// not an envelope. An empty payload is sent as "{}" so the body stays valid JSON.
func (c *httpArgoClient) SendEvent(ctx context.Context, namespace, discriminator string, payload []byte) error {
	body := payload
	if len(body) == 0 {
		body = []byte("{}")
	}

	url := fmt.Sprintf("%s/api/v1/events/%s/%s", c.serverURL, namespace, discriminator)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("argo_client: create event request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("argo_client: send event discriminator=%q: %w", discriminator, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("argo_client: send event discriminator=%q: HTTP %d: %s", discriminator, resp.StatusCode, string(b))
	}
	return nil
}
