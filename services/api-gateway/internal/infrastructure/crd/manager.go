// SPDX-License-Identifier: Apache-2.0

package crd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// leaderElectionID is the Lease name for the single-writer Workflow status
// reconciler election (ADR-043). One Lease per deployment namespace.
const leaderElectionID = "zynax-api-gateway-workflow-controller"

// NewManager builds a controller-runtime manager whose namespaced informer
// cache watches Workflow CRs, and registers the WorkflowReconciler. The
// manager's own metrics/probe servers are disabled — api-gateway already serves
// metrics and health probes over HTTP; a second bind would clash.
//
// namespace scopes BOTH the informer cache and the election Lease: the chart's
// RBAC is a namespaced Role (least privilege, no ClusterRole), so a
// cluster-scope watch would be forbidden. Required — an empty namespace is a
// hard error (mirrors the agent-registry scheduler, ADR-039).
func NewManager(restCfg *rest.Config, namespace string) (manager.Manager, error) {
	if namespace == "" {
		return nil, fmt.Errorf("crd: watch namespace is required (namespaced RBAC forbids cluster-scope watches)")
	}
	mgr, err := ctrl.NewManager(restCfg, manager.Options{
		Metrics:                 metricsserver.Options{BindAddress: "0"},
		HealthProbeBindAddress:  "0",
		LeaderElection:          true,
		LeaderElectionID:        leaderElectionID,
		LeaderElectionNamespace: namespace,
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{namespace: {}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("crd: create manager: %w", err)
	}
	watched := &unstructured.Unstructured{}
	watched.SetGroupVersionKind(WorkflowGVK)
	if err := ctrl.NewControllerManagedBy(mgr).For(watched).
		Complete(&WorkflowReconciler{Client: mgr.GetClient()}); err != nil {
		return nil, fmt.Errorf("crd: build controller: %w", err)
	}
	return mgr, nil
}

// StartController loads the in-cluster / kubeconfig rest config, builds the
// manager, and runs it in a goroutine until ctx is cancelled. A manager error
// is fatal to the controller only — it is logged, not propagated — so a
// controller failure never takes down the api-gateway REST apply path.
func StartController(ctx context.Context, namespace string) error {
	// controller-runtime demands a logger before manager construction; bridge
	// it into the service's structured slog output.
	ctrl.SetLogger(logr.FromSlogHandler(slog.Default().Handler()))
	restCfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("crd: load kubeconfig: %w", err)
	}
	mgr, err := NewManager(restCfg, namespace)
	if err != nil {
		return fmt.Errorf("crd: build manager: %w", err)
	}
	go func() {
		slog.Info("api-gateway: workflow controller started (Workflow CRs -> compile/submit; Lease-elected, namespaced)")
		if err := mgr.Start(ctx); err != nil {
			slog.Error("workflow controller exited", "err", err)
		}
	}()
	return nil
}
