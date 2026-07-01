// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

// envHasKey reports whether any entry sets KEY (i.e. "KEY=..."). Used to assert
// that --no-load-images omits KIND_LOAD_IMAGES entirely.
func envHasKey(env []string, key string) bool {
	for _, e := range env {
		if strings.HasPrefix(e, key+"=") {
			return true
		}
	}
	return false
}

// setUpDefaults resets the package-level up flag vars to their command defaults.
// runUp is called directly (no cobra flag parse), so tests must set them.
func setUpDefaults() {
	upProfile = "full"
	upEngine = "temporal"
	upNoLoad = false
	upClusterName = ""
	upNamespace = ""
	upRepoRoot = ""
}

func runUpRecording(t *testing.T) *recordingRunner {
	t.Helper()
	rr := &recordingRunner{}
	deps := clusterDeps{run: rr.run, findRoot: fixedRoot("/repo")}
	if err := runUp(context.Background(), io.Discard, io.Discard, deps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rr.called {
		t.Fatal("runner was not invoked")
	}
	if rr.script != filepath.Join("/repo", clusterUpRel) {
		t.Errorf("script = %q, want %q", rr.script, filepath.Join("/repo", clusterUpRel))
	}
	return rr
}

func TestRunUp_Defaults(t *testing.T) {
	setUpDefaults()
	rr := runUpRecording(t)
	for _, want := range []string{"PROFILE=full", "E2E_ENGINE=temporal", "KIND_LOAD_IMAGES=1"} {
		if !envHas(rr.env, want) {
			t.Errorf("default env missing %q", want)
		}
	}
	// No overrides → cluster name / namespace fall through to the script defaults.
	if envHasKey(rr.env, "CLUSTER_NAME") {
		t.Error("CLUSTER_NAME should be absent when --cluster-name is empty")
	}
	if envHasKey(rr.env, "NAMESPACE") {
		t.Error("NAMESPACE should be absent when --namespace is empty")
	}
}

func TestRunUp_LiteArgo(t *testing.T) {
	setUpDefaults()
	upProfile = "lite"
	upEngine = "argo"
	rr := runUpRecording(t)
	if !envHas(rr.env, "PROFILE=lite") || !envHas(rr.env, "E2E_ENGINE=argo") {
		t.Errorf("lite/argo not mapped; env tail=%v", rr.env[len(rr.env)-4:])
	}
}

func TestRunUp_NoLoadImages_OmitsVar(t *testing.T) {
	setUpDefaults()
	upNoLoad = true
	rr := runUpRecording(t)
	if envHasKey(rr.env, "KIND_LOAD_IMAGES") {
		t.Error("--no-load-images must omit KIND_LOAD_IMAGES")
	}
}

func TestRunUp_ClusterAndNamespaceOverrides(t *testing.T) {
	setUpDefaults()
	upClusterName = "my-kind"
	upNamespace = "zynax-dev"
	rr := runUpRecording(t)
	if !envHas(rr.env, "CLUSTER_NAME=my-kind") || !envHas(rr.env, "NAMESPACE=zynax-dev") {
		t.Errorf("cluster/namespace overrides not forwarded; env=%v", rr.env[len(rr.env)-4:])
	}
}

func TestRunUp_RepoRootError_RunnerNotCalled(t *testing.T) {
	setUpDefaults()
	rr := &recordingRunner{}
	deps := clusterDeps{run: rr.run, findRoot: func(string) (string, error) { return "", errNoRepoRoot }}
	if err := runUp(context.Background(), io.Discard, io.Discard, deps); err == nil {
		t.Fatal("expected error when repo root cannot be resolved")
	}
	if rr.called {
		t.Error("runner must not run when repo-root resolution fails")
	}
}

func TestUpCmd_Registered(t *testing.T) {
	c, _, err := rootCmd.Find([]string{"up"})
	if err != nil || c.Name() != "up" {
		t.Fatalf("up command not registered: cmd=%v err=%v", c, err)
	}
	if c.GroupID != beginnerGroupID {
		t.Errorf("up GroupID = %q, want %q", c.GroupID, beginnerGroupID)
	}
}
