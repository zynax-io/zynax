// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// doctorNamespace is the Helm release namespace the kind-native runtime deploys
// the Zynax stack into (scripts/e2e/cluster-up.sh: NAMESPACE=zynax).
const doctorNamespace = "zynax"

// doctorRelease is the umbrella Helm release name (cluster-up.sh: RELEASE_NAME=zynax).
const doctorRelease = "zynax"

// defaultModel is the reference local model the quickstart pulls (DEMO_MODEL /
// infra/docker-compose/ollama/llm-adapter.config.yaml). Kept in lockstep with
// the demo script so the doctor pre-flight and runtime config never drift.
const defaultModel = "qwen2.5-coder:3b"

// doctorComponent names a platform Deployment/StatefulSet doctor checks for
// readiness, paired with the user-facing label and a remediation hint.
type doctorComponent struct {
	label    string // user-facing component name (wedge/help copy)
	workload string // kubectl workload, e.g. "deployment/zynax-api-gateway"
	hint     string // one-line remediation shown when the component is not Ready
}

// doctorComponents are the platform workloads doctor asserts are Ready. Names
// match scripts/e2e/cluster-up.sh (namespace zynax, release zynax) exactly:
// the 5 Gherkin lines API/Registry/Runtime/Event Bus/Storage.
var doctorComponents = []doctorComponent{
	{"API gateway", "deployment/zynax-api-gateway", "kubectl -n " + doctorNamespace + " rollout status deployment/zynax-api-gateway"},
	{"Agent registry", "deployment/zynax-agent-registry", "kubectl -n " + doctorNamespace + " rollout status deployment/zynax-agent-registry"},
	{"Runtime (engine-adapter)", "deployment/zynax-engine-adapter", "kubectl -n " + doctorNamespace + " rollout status deployment/zynax-engine-adapter"},
	{"Event bus", "deployment/zynax-event-bus", "kubectl -n " + doctorNamespace + " rollout status deployment/zynax-event-bus"},
	{"Storage (Postgres)", "statefulset/zynax-postgresql", "kubectl -n " + doctorNamespace + " rollout status statefulset/zynax-postgresql"},
}

// commander runs an external command and returns its combined stdout. Injected
// so unit tests run with a fake — no real kubectl/helm/ollama or cluster.
type commander func(ctx context.Context, name string, args ...string) (string, error)

// healthProber probes the api-gateway liveness endpoint. Injected so tests use
// an httptest.Server (or a stub) instead of a real gateway.
type healthProber func(ctx context.Context) (string, error)

// execCommander is the production commander: shells out via os/exec and returns
// combined output. Per cmd/zynax/AGENTS.md the CLI shells out to the user's
// kubectl/helm/ollama binaries (no client-go/helm SDK deps).
func execCommander(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // G204: name is a fixed binary (kubectl/helm/ollama); args are static
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// doctorDeps bundles the injectable collaborators so tests are hermetic.
type doctorDeps struct {
	run    commander
	health healthProber
	model  string
}

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	GroupID: beginnerGroupID,
	Short:   "Check the platform is healthy — write once, run on Temporal or Argo",
	Long: `Confirm your local platform is ready before your first workflow.

Write your workflow ONCE — it runs on Temporal OR Argo, on the same kind
cluster that mirrors production. ` + "`zynax doctor`" + ` validates that runtime with
one read-only checklist instead of hand-running kubectl, helm, and curl:

  • Cluster      — the current kubecontext is reachable
  • Helm release — the zynax umbrella deployed cleanly
  • Pods         — API, Registry, Runtime, Event Bus and Storage are Ready
  • API gateway  — /healthz answers on --api-url
  • Model        — the default reference model is available (host-side)

Exit 0 only when the cluster, release, pods and gateway are all healthy
(scriptable); a missing local model is a warning, not a failure.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		deps := doctorDeps{run: execCommander, health: newGateway().Health, model: defaultModel}
		ok := runDoctor(cmd.Context(), cmd.OutOrStdout(), deps)
		if !ok {
			// Non-zero, scriptable exit (AC3). Mirrors status.go's os.Exit(2)
			// precedent: a clean RunE return would exit 0 and hide the failure.
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

// runDoctor runs every check, prints a ✓/✗ checklist, and returns true only
// when the cluster, release, pods and gateway are all healthy. A missing model
// prints a ⚠ warning line but does NOT flip the result (AC2/AC3): model-backed
// workflows need it, but the kind hero workflow uses the in-cluster echo
// capability, so a missing host model must not block a healthy platform.
func runDoctor(ctx context.Context, w io.Writer, deps doctorDeps) bool {
	// Wedge-first header (AC4): lead with engine portability, never "control plane".
	_, _ = fmt.Fprintln(w, "zynax doctor — write your workflow once, run it on Temporal or Argo.")
	_, _ = fmt.Fprintln(w, "Checking the kind runtime that mirrors production:")
	_, _ = fmt.Fprintln(w)

	healthy := true

	// 1. Cluster reachability.
	if err := checkCluster(ctx, deps.run); err != nil {
		printFail(w, "Cluster", "kubecontext not reachable — start it with `make kind-up` (or `kubectl config use-context kind-zynax-e2e`)")
		healthy = false
	} else {
		printOK(w, "Cluster")
	}

	// 2. Helm release status.
	if err := checkRelease(ctx, deps.run); err != nil {
		printFail(w, "Helm release", "umbrella not deployed — run `make kind-up` to `helm upgrade --install` it")
		healthy = false
	} else {
		printOK(w, "Helm release")
	}

	// 3. Pod/Deployment readiness, per component.
	for _, c := range doctorComponents {
		if err := checkWorkloadReady(ctx, deps.run, c.workload); err != nil {
			printFail(w, "Pods: "+c.label, c.hint)
			healthy = false
		} else {
			printOK(w, "Pods: "+c.label)
		}
	}

	// 4. api-gateway liveness over the configured --api-url.
	if _, err := deps.health(ctx); err != nil {
		printFail(w, "API gateway", "/healthz did not answer — check --api-url and `kubectl -n "+doctorNamespace+" port-forward svc/zynax-api-gateway 8080:8080`")
		healthy = false
	} else {
		printOK(w, "API gateway")
	}

	// 5. Default reference model (host-side, informational — AC2). Missing model
	// warns with remediation but never flips `healthy`.
	if checkModel(ctx, deps.run, deps.model) {
		printOK(w, "Model ("+deps.model+")")
	} else {
		printWarn(w, "Model ("+deps.model+")", "not found on host — `ollama pull "+deps.model+"` to run model-backed workflows (the kind demo's echo capability needs no model)")
	}

	_, _ = fmt.Fprintln(w)
	if healthy {
		_, _ = fmt.Fprintln(w, "✅  Platform healthy — run: zynax apply spec/workflows/examples/e2e-demo.yaml")
	} else {
		_, _ = fmt.Fprintln(w, "❌  Platform not ready — fix the ✗ lines above, then re-run `zynax doctor`.")
	}
	return healthy
}

// checkCluster verifies the current kubecontext is reachable.
func checkCluster(ctx context.Context, run commander) error {
	_, err := run(ctx, "kubectl", "cluster-info")
	return err
}

// checkRelease verifies the umbrella Helm release is deployed and its last
// operation succeeded (status "deployed").
func checkRelease(ctx context.Context, run commander) error {
	out, err := run(ctx, "helm", "-n", doctorNamespace, "status", doctorRelease, "-o", "json")
	if err != nil {
		return fmt.Errorf("helm status: %w", err)
	}
	var st struct {
		Info struct {
			Status string `json:"status"`
		} `json:"info"`
	}
	if err := json.Unmarshal([]byte(out), &st); err != nil {
		return fmt.Errorf("decode helm status: %w", err)
	}
	if st.Info.Status != "deployed" {
		return fmt.Errorf("release status %q (want deployed)", st.Info.Status)
	}
	return nil
}

// checkWorkloadReady verifies a Deployment/StatefulSet has all replicas Ready by
// parsing `kubectl get <workload> -o json` status counters.
func checkWorkloadReady(ctx context.Context, run commander, workload string) error {
	out, err := run(ctx, "kubectl", "-n", doctorNamespace, "get", workload, "-o", "json")
	if err != nil {
		return fmt.Errorf("kubectl get %s: %w", workload, err)
	}
	var obj struct {
		Spec struct {
			Replicas *int `json:"replicas"`
		} `json:"spec"`
		Status struct {
			ReadyReplicas int `json:"readyReplicas"`
		} `json:"status"`
	}
	if err := json.Unmarshal([]byte(out), &obj); err != nil {
		return fmt.Errorf("decode %s: %w", workload, err)
	}
	want := 1
	if obj.Spec.Replicas != nil {
		want = *obj.Spec.Replicas
	}
	if obj.Status.ReadyReplicas < want {
		return fmt.Errorf("%d/%d replicas ready", obj.Status.ReadyReplicas, want)
	}
	return nil
}

// checkModel reports whether the default reference model is present on the host.
// Host-side and informational: a missing `ollama` binary or model is not fatal.
func checkModel(ctx context.Context, run commander, model string) bool {
	out, err := run(ctx, "ollama", "list")
	if err != nil {
		return false
	}
	// `ollama list` first column is NAME (e.g. "qwen2.5-coder:3b").
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == model {
			return true
		}
	}
	return false
}

func printOK(w io.Writer, label string) {
	_, _ = fmt.Fprintf(w, "  ✓ %s\n", label)
}

func printFail(w io.Writer, label, hint string) {
	_, _ = fmt.Fprintf(w, "  ✗ %s — %s\n", label, hint)
}

func printWarn(w io.Writer, label, hint string) {
	_, _ = fmt.Fprintf(w, "  ⚠ %s — %s\n", label, hint)
}
