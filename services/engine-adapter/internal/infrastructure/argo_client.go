// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides concrete implementations of domain ports backed by
// external services and SDKs. Only this package may import engine-specific dependencies
// such as the Temporal Go SDK (ADR-015).
package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ArgoWorkflow is a minimal representation of an Argo Workflows Workflow resource
// submitted via the Argo Workflows REST API. Only the fields required for Submit
// and Signal are included here; B.3 will add status-query fields.
type ArgoWorkflow struct {
	APIVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Metadata   ArgoObjectMeta   `json:"metadata"`
	Spec       ArgoWorkflowSpec `json:"spec"`
}

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

// ArgoWorkflowEventBinding is a simplified representation of the payload used
// when delivering an external event to a running Argo Workflow via the
// /api/v1/events/{namespace}/{discriminator} endpoint.
type ArgoWorkflowEventBinding struct {
	Payload json.RawMessage `json:"payload,omitempty"`
}

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

// SubmitWorkflow POSTs the Argo Workflow resource to the Argo REST API.
func (c *httpArgoClient) SubmitWorkflow(ctx context.Context, namespace string, wf *ArgoWorkflow) error {
	body, err := json.Marshal(wf)
	if err != nil {
		return fmt.Errorf("argo_client: marshal workflow: %w", err)
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

// SendEvent POSTs an external event payload to the Argo Workflows event endpoint.
func (c *httpArgoClient) SendEvent(ctx context.Context, namespace, discriminator string, payload []byte) error {
	eventBody := ArgoWorkflowEventBinding{}
	if len(payload) > 0 {
		eventBody.Payload = json.RawMessage(payload)
	}

	body, err := json.Marshal(eventBody)
	if err != nil {
		return fmt.Errorf("argo_client: marshal event payload: %w", err)
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
