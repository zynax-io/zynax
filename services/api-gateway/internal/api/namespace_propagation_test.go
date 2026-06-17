// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zynax-io/zynax/services/api-gateway/internal/api"
	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
)

// This file is the end-to-end namespace propagation audit for EPIC #767
// (canvas step O2 / D.4). It asserts that the namespace supplied as the HTTP
// `?namespace=` query parameter flows unchanged through every control-plane
// hop:
//
//	HTTP ?namespace=team-a
//	  → CompileWorkflowRequest.namespace   (CompilerPort.CompileWorkflow arg)
//	  → WorkflowIR.namespace               (echoed back as CompileResult.Namespace)
//	  → SubmitWorkflowRequest.namespace    (EnginePort.SubmitWorkflow arg)
//
// The recording stubs capture the namespace observed at each boundary so the
// test can assert continuity across all three hops in a single HTTP request.

// nsTeamA is the sample namespace used across the end-to-end propagation asserts.
const nsTeamA = "team-a"

// recordingCompiler captures the namespace it receives and echoes it back on
// the CompileResult, mirroring the real compiler embedding the namespace into
// WorkflowIR.namespace (proto field 3).
type recordingCompiler struct {
	gotNamespace    string
	gotCtxNamespace string
}

func (c *recordingCompiler) CompileWorkflow(ctx context.Context, _ []byte, namespace string, _ bool) (domain.CompileResult, error) {
	c.gotNamespace = namespace
	c.gotCtxNamespace = domain.NamespaceFromContext(ctx)
	return domain.CompileResult{IRBytes: []byte("ir"), Namespace: namespace}, nil
}

// recordingEngine captures the namespace it receives on SubmitWorkflow. Its
// GetWorkflowStatus reports ErrNotFound so the submit hop always fires (no
// idempotent short-circuit on a running workflow).
type recordingEngine struct {
	gotNamespace string
}

func (e *recordingEngine) SubmitWorkflow(_ context.Context, _ []byte, _, _, namespace string) (string, error) {
	e.gotNamespace = namespace
	return "run-ns", nil
}

func (e *recordingEngine) GetWorkflowStatus(_ context.Context, _ string) (domain.WorkflowRunSummary, error) {
	return domain.WorkflowRunSummary{}, domain.ErrNotFound
}

func (e *recordingEngine) CancelWorkflow(_ context.Context, _ string) error { return nil }

func (e *recordingEngine) WatchWorkflow(_ context.Context, _ string, _ func(domain.WatchEvent) error) error {
	return nil
}

func newRecordingServer(c domain.CompilerPort, e domain.EnginePort) *httptest.Server {
	svc := domain.NewApplyService(c, e, &stubRegistry{}, nil)
	h := api.NewHandler(svc, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	// Front the mux with RequestIDMiddleware (as main.go does) so the X-Namespace
	// header is read into the correlation context before the handler runs.
	return httptest.NewServer(api.RequestIDMiddleware(mux))
}

// TestNamespacePropagation_EndToEnd asserts the namespace from the HTTP query
// param reaches both the compile hop and the submit hop unchanged.
func TestNamespacePropagation_EndToEnd(t *testing.T) {
	compiler := &recordingCompiler{}
	engine := &recordingEngine{}
	srv := newRecordingServer(compiler, engine)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/apply?namespace="+nsTeamA, "application/yaml", bytes.NewBufferString(workflowYAML))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status: got %d, want 202", resp.StatusCode)
	}
	// Hop 1: HTTP ?namespace= → CompileWorkflowRequest.namespace
	if compiler.gotNamespace != nsTeamA {
		t.Errorf("compile hop: got namespace %q, want %s", compiler.gotNamespace, nsTeamA)
	}
	// Correlation context: the namespace must also ride on the call context so
	// the gRPC interceptors attach it as x-namespace metadata on every hop.
	if compiler.gotCtxNamespace != nsTeamA {
		t.Errorf("compile hop: ctx namespace %q, want %s", compiler.gotCtxNamespace, nsTeamA)
	}
	// Hop 3: WorkflowIR.namespace (compiled.Namespace) → SubmitWorkflowRequest.namespace
	if engine.gotNamespace != nsTeamA {
		t.Errorf("submit hop: got namespace %q, want %s", engine.gotNamespace, nsTeamA)
	}
}

// TestNamespacePropagation_HeaderWinsOverQuery asserts the X-Namespace header
// set on the request takes precedence over the ?namespace= query param when both
// are present, and that the header value rides the correlation context.
func TestNamespacePropagation_HeaderWinsOverQuery(t *testing.T) {
	compiler := &recordingCompiler{}
	engine := &recordingEngine{}
	srv := newRecordingServer(compiler, engine)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/apply?namespace=team-q", bytes.NewBufferString(workflowYAML))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Namespace", "team-h")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status: got %d, want 202", resp.StatusCode)
	}
	// The request field still reflects the query param (existing #767 contract),
	// but the correlation context carries the header-supplied namespace.
	if compiler.gotCtxNamespace != "team-h" {
		t.Errorf("ctx namespace %q, want team-h (header wins)", compiler.gotCtxNamespace)
	}
}

// TestNamespacePropagation_DefaultsWhenAbsent asserts backwards compatibility:
// when `?namespace=` is omitted, an empty string flows through unchanged. The
// downstream workflow-compiler is responsible for substituting "default"; the
// gateway must not invent a namespace of its own (canvas Safeguards).
func TestNamespacePropagation_DefaultsWhenAbsent(t *testing.T) {
	compiler := &recordingCompiler{}
	engine := &recordingEngine{}
	srv := newRecordingServer(compiler, engine)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/apply", "application/yaml", bytes.NewBufferString(workflowYAML))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status: got %d, want 202", resp.StatusCode)
	}
	if compiler.gotNamespace != "" {
		t.Errorf("compile hop: got namespace %q, want empty (gateway must not invent a namespace)", compiler.gotNamespace)
	}
	if engine.gotNamespace != "" {
		t.Errorf("submit hop: got namespace %q, want empty", engine.gotNamespace)
	}
}

// TestNamespacePropagation_DistinctNamespacesIsolated asserts two requests with
// different namespaces are routed independently — team-a and team-b never
// cross-contaminate at any hop.
func TestNamespacePropagation_DistinctNamespacesIsolated(t *testing.T) {
	for _, ns := range []string{"team-a", "team-b"} {
		compiler := &recordingCompiler{}
		engine := &recordingEngine{}
		srv := newRecordingServer(compiler, engine)

		resp, err := http.Post(srv.URL+"/api/v1/apply?namespace="+ns, "application/yaml", bytes.NewBufferString(workflowYAML))
		if err != nil {
			srv.Close()
			t.Fatal(err)
		}
		_ = resp.Body.Close()

		if compiler.gotNamespace != ns {
			t.Errorf("compile hop: got namespace %q, want %q", compiler.gotNamespace, ns)
		}
		if engine.gotNamespace != ns {
			t.Errorf("submit hop: got namespace %q, want %q", engine.gotNamespace, ns)
		}
		srv.Close()
	}
}
