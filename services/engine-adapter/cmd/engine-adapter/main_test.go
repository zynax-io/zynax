// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"testing"

	"github.com/zynax-io/zynax/services/engine-adapter/internal/domain"
	"github.com/zynax-io/zynax/services/engine-adapter/internal/infrastructure"
)

// TestBuildEngine_ArgoSelected verifies that ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE=argo
// injects an *infrastructure.ArgoEngine (ADR-015 — engine chosen by config flag).
func TestBuildEngine_ArgoSelected(t *testing.T) {
	cfg := loadConfig()
	cfg.ActiveEngine = engineArgo

	engine, cleanup, brokerConn, err := buildEngine(cfg)
	if err != nil {
		t.Fatalf("buildEngine(argo) returned error: %v", err)
	}
	t.Cleanup(cleanup)

	if _, ok := engine.(*infrastructure.ArgoEngine); !ok {
		t.Fatalf("buildEngine(argo) = %T; want *infrastructure.ArgoEngine", engine)
	}
	// Argo does not dispatch through the task-broker, so no broker connection.
	if brokerConn != nil {
		t.Errorf("buildEngine(argo) brokerConn = %v; want nil", brokerConn)
	}
}

// TestBuildEngine_UnrecognisedFatal verifies that an unknown engine name is a
// startup error (run() turns this into os.Exit(1)).
func TestBuildEngine_UnrecognisedFatal(t *testing.T) {
	cfg := loadConfig()
	cfg.ActiveEngine = "kubernetes-batch"

	engine, _, _, err := buildEngine(cfg)
	if err == nil {
		t.Fatalf("buildEngine(unrecognised) returned nil error; want startup failure")
	}
	if engine != nil {
		t.Errorf("buildEngine(unrecognised) engine = %v; want nil", engine)
	}
}

// TestArgoEngine_UnknownRunID is the smoke test required by canvas O4: the argo
// engine must surface domain.ErrExecutionNotFound for an unknown run ID. It uses a
// stub ArgoClient so no live Argo server is required.
func TestArgoEngine_UnknownRunID(t *testing.T) {
	cfg := loadConfig()
	cfg.ActiveEngine = engineArgo
	engine := infrastructure.NewArgoEngine(
		stubArgoClient{},
		infrastructure.ArgoConfig{Namespace: cfg.ArgoNamespace},
	)

	_, err := engine.GetStatus(context.Background(), "no-such-run")
	if !errors.Is(err, domain.ErrExecutionNotFound) {
		t.Fatalf("GetStatus(unknown) error = %v; want domain.ErrExecutionNotFound", err)
	}
}

// stubArgoClient is a not-found ArgoClient: every lookup reports the workflow is
// absent, exercising the engine's ErrExecutionNotFound mapping.
type stubArgoClient struct{}

func (stubArgoClient) SubmitWorkflow(context.Context, string, *infrastructure.ArgoWorkflow) error {
	return nil
}

func (stubArgoClient) SendEvent(context.Context, string, string, []byte) error {
	return nil
}

func (stubArgoClient) GetWorkflow(context.Context, string, string) (*infrastructure.ArgoWorkflow, error) {
	return nil, domain.ErrExecutionNotFound
}

func (stubArgoClient) DeleteWorkflow(context.Context, string, string) error {
	return domain.ErrExecutionNotFound
}
