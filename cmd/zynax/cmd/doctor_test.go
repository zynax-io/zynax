// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeCommander builds a commander that returns canned output keyed by the
// command + first arg pattern, so doctor's checks run with no real binaries.
// Each rule maps a substring of the full command line to (output, err).
type cmdRule struct {
	match string // substring of "<name> <args...>"
	out   string
	err   error
}

func fakeCommander(rules []cmdRule) commander {
	return func(_ context.Context, name string, args ...string) (string, error) {
		full := name + " " + strings.Join(args, " ")
		for _, r := range rules {
			if strings.Contains(full, r.match) {
				return r.out, r.err
			}
		}
		return "", errors.New("fakeCommander: no rule for: " + full)
	}
}

// healthyRules returns commander rules where every kubectl/helm/ollama call
// reports a healthy platform with the default model present.
func healthyRules() []cmdRule {
	return []cmdRule{
		{match: "kubectl cluster-info", out: "Kubernetes control plane is running\n"},
		{match: "helm -n zynax status zynax", out: `{"info":{"status":"deployed"}}`},
		{match: "get deployment/", out: `{"spec":{"replicas":1},"status":{"readyReplicas":1}}`},
		{match: "get statefulset/", out: `{"spec":{"replicas":1},"status":{"readyReplicas":1}}`},
		{match: "ollama list", out: "NAME\t\t\tID\nqwen2.5-coder:3b\tabc123\n"},
	}
}

// okHealth is a healthProber stub reporting a healthy gateway.
func okHealth(_ context.Context) (string, error) { return "ok", nil }

// failHealth is a healthProber stub reporting an unreachable gateway.
func failHealth(_ context.Context) (string, error) { return "", errors.New("connection refused") }

func runDoctorTest(deps doctorDeps) (string, bool) {
	var out bytes.Buffer
	ok := runDoctor(context.Background(), &out, deps)
	return out.String(), ok
}

// TestDoctor_AllOK asserts AC1/AC3/AC4: a fully healthy platform prints every
// component OK and returns true (→ exit 0), and the output leads with the
// engine-portability wedge.
func TestDoctor_AllOK(t *testing.T) {
	out, ok := runDoctorTest(doctorDeps{run: fakeCommander(healthyRules()), health: okHealth, model: defaultModel})
	if !ok {
		t.Fatalf("expected healthy=true, got false; output:\n%s", out)
	}
	// AC4: wedge-first framing.
	if !strings.Contains(out, "Temporal or Argo") {
		t.Errorf("output missing engine-portability wedge:\n%s", out)
	}
	// AC1: every component line present and OK.
	for _, want := range []string{
		"✓ Cluster", "✓ Helm release",
		"✓ Pods: API gateway", "✓ Pods: Agent registry", "✓ Pods: Runtime (engine-adapter)",
		"✓ Pods: Event bus", "✓ Pods: Storage (Postgres)",
		"✓ API gateway", "✓ Model (" + defaultModel + ")",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "Platform healthy") {
		t.Errorf("expected healthy summary, got:\n%s", out)
	}
}

// TestDoctor_PodNotReady asserts the Gherkin failing path: a required pod not
// Ready flips the line to ✗ with a one-line hint and returns false (→ non-zero).
func TestDoctor_PodNotReady(t *testing.T) {
	rules := healthyRules()
	// engine-adapter reports 0/1 replicas ready.
	rules = append([]cmdRule{
		{match: "get deployment/zynax-engine-adapter", out: `{"spec":{"replicas":1},"status":{"readyReplicas":0}}`},
	}, rules...)

	out, ok := runDoctorTest(doctorDeps{run: fakeCommander(rules), health: okHealth, model: defaultModel})
	if ok {
		t.Fatalf("expected healthy=false for not-ready pod, got true; output:\n%s", out)
	}
	if !strings.Contains(out, "✗ Pods: Runtime (engine-adapter)") {
		t.Errorf("expected failing runtime line, got:\n%s", out)
	}
	// One-line hint present (remediation).
	if !strings.Contains(out, "rollout status deployment/zynax-engine-adapter") {
		t.Errorf("expected remediation hint, got:\n%s", out)
	}
	if !strings.Contains(out, "Platform not ready") {
		t.Errorf("expected not-ready summary, got:\n%s", out)
	}
}

// TestDoctor_ModelMissing asserts AC2: a missing default model warns with a
// remediation hint but does NOT flip the platform to unhealthy (exit stays 0).
func TestDoctor_ModelMissing(t *testing.T) {
	rules := healthyRules()
	rules = append([]cmdRule{
		{match: "ollama list", out: "NAME\tID\nllama3.2:3b\txyz\n"},
	}, rules...)

	out, ok := runDoctorTest(doctorDeps{run: fakeCommander(rules), health: okHealth, model: defaultModel})
	if !ok {
		t.Fatalf("expected healthy=true with model missing (warning only), got false:\n%s", out)
	}
	if !strings.Contains(out, "⚠ Model ("+defaultModel+")") {
		t.Errorf("expected model warning line, got:\n%s", out)
	}
	if !strings.Contains(out, "ollama pull "+defaultModel) {
		t.Errorf("expected ollama pull remediation hint, got:\n%s", out)
	}
}

// TestDoctor_ModelBinaryMissing asserts a missing `ollama` binary (commander
// error) is treated as model-absent: a warning, not a hard failure.
func TestDoctor_ModelBinaryMissing(t *testing.T) {
	rules := healthyRules()
	rules = append([]cmdRule{
		{match: "ollama list", err: errors.New("exec: ollama not found")},
	}, rules...)

	out, ok := runDoctorTest(doctorDeps{run: fakeCommander(rules), health: okHealth, model: defaultModel})
	if !ok {
		t.Fatalf("expected healthy=true when ollama binary absent, got false:\n%s", out)
	}
	if !strings.Contains(out, "⚠ Model (") {
		t.Errorf("expected model warning when ollama absent, got:\n%s", out)
	}
}

// TestDoctor_GatewayUnreachable asserts an unreachable api-gateway fails the
// gateway line with a hint and flips the platform to unhealthy.
func TestDoctor_GatewayUnreachable(t *testing.T) {
	out, ok := runDoctorTest(doctorDeps{run: fakeCommander(healthyRules()), health: failHealth, model: defaultModel})
	if ok {
		t.Fatalf("expected healthy=false for unreachable gateway, got true:\n%s", out)
	}
	if !strings.Contains(out, "✗ API gateway") {
		t.Errorf("expected failing gateway line, got:\n%s", out)
	}
	if !strings.Contains(out, "--api-url") {
		t.Errorf("expected gateway remediation hint, got:\n%s", out)
	}
}

// TestDoctor_ClusterUnreachable asserts an unreachable cluster fails the Cluster
// line and the whole command.
func TestDoctor_ClusterUnreachable(t *testing.T) {
	rules := []cmdRule{
		{match: "kubectl cluster-info", err: errors.New("connection refused")},
		{match: "helm -n zynax status zynax", out: `{"info":{"status":"deployed"}}`},
		{match: "get deployment/", out: `{"spec":{"replicas":1},"status":{"readyReplicas":1}}`},
		{match: "get statefulset/", out: `{"spec":{"replicas":1},"status":{"readyReplicas":1}}`},
		{match: "ollama list", out: "qwen2.5-coder:3b\tabc\n"},
	}
	out, ok := runDoctorTest(doctorDeps{run: fakeCommander(rules), health: okHealth, model: defaultModel})
	if ok {
		t.Fatalf("expected healthy=false for unreachable cluster, got true:\n%s", out)
	}
	if !strings.Contains(out, "✗ Cluster") {
		t.Errorf("expected failing cluster line, got:\n%s", out)
	}
}

// TestDoctor_ReleaseNotDeployed asserts a Helm release in a non-"deployed"
// state (e.g. failed/pending) fails the release line.
func TestDoctor_ReleaseNotDeployed(t *testing.T) {
	rules := healthyRules()
	rules = append([]cmdRule{
		{match: "helm -n zynax status zynax", out: `{"info":{"status":"failed"}}`},
	}, rules...)

	out, ok := runDoctorTest(doctorDeps{run: fakeCommander(rules), health: okHealth, model: defaultModel})
	if ok {
		t.Fatalf("expected healthy=false for failed release, got true:\n%s", out)
	}
	if !strings.Contains(out, "✗ Helm release") {
		t.Errorf("expected failing release line, got:\n%s", out)
	}
}

// TestDoctor_GatewayHealth_RealProbe wires the real Gateway.Health against an
// httptest.Server so the client probe path is covered end-to-end (no cluster).
func TestDoctor_GatewayHealth_RealProbe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	apiURL = srv.URL

	out, ok := runDoctorTest(doctorDeps{run: fakeCommander(healthyRules()), health: newGateway().Health, model: defaultModel})
	if !ok {
		t.Fatalf("expected healthy=true with live healthz probe, got false:\n%s", out)
	}
	if !strings.Contains(out, "✓ API gateway") {
		t.Errorf("expected gateway OK from live probe, got:\n%s", out)
	}
}

// TestDoctor_Registered asserts the doctor command is wired under root in the
// beginner group (canvas O19/O20).
func TestDoctor_Registered(t *testing.T) {
	c, _, err := rootCmd.Find([]string{"doctor"})
	if err != nil {
		t.Fatalf("doctor command does not resolve: %v", err)
	}
	if c != doctorCmd {
		t.Fatalf("resolved %q, want doctorCmd", c.Use)
	}
	if c.GroupID != beginnerGroupID {
		t.Errorf("doctor GroupID = %q, want beginner group %q", c.GroupID, beginnerGroupID)
	}
}
