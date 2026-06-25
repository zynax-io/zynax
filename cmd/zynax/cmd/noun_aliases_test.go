// SPDX-License-Identifier: Apache-2.0

// Tests for the noun-first aliases (agent/workflow groups + publish verb).
// They assert command resolution and RunE wiring (no network) plus one
// behavioural round-trip per alias through httptest, mirroring cmd_test.go.
package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Manifest-kind literals shared across the CLI tests (satisfies goconst).
const (
	kindWorkflow = "Workflow"
	kindAgentDef = "AgentDef"
)

// runEPtr returns the comparable pointer of a command's RunE for identity asserts.
func runEPtr(c *cobra.Command) uintptr {
	if c == nil || c.RunE == nil {
		return 0
	}
	return reflect.ValueOf(c.RunE).Pointer()
}

// ── command resolution ────────────────────────────────────────────────────────

// TestNounAliases_Resolve proves each new noun-verb form resolves to a real
// command under rootCmd (back-compat verbs are covered separately below).
func TestNounAliases_Resolve(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want *cobra.Command
	}{
		{"agent init", []string{"agent", "init"}, agentInitCmd},
		{"agent publish", []string{"agent", "publish"}, agentPublishCmd},
		{"workflow init", []string{"workflow", "init"}, workflowInitCmd},
		{"workflow run", []string{"workflow", "run"}, workflowRunCmd},
		{"workflow publish", []string{"workflow", "publish"}, workflowPublishCmd},
		{"publish", []string{"publish"}, publishCmd},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _, err := rootCmd.Find(tc.args)
			if err != nil {
				t.Fatalf("Find(%v) error: %v", tc.args, err)
			}
			if c != tc.want {
				t.Errorf("resolved %q, want %q", c.Use, tc.want.Use)
			}
		})
	}
}

// TestNounAliases_DelegateToVerbRunE proves the aliases reuse the existing verb
// commands' RunE (no duplicated logic), per canvas O20.
func TestNounAliases_DelegateToVerbRunE(t *testing.T) {
	cases := []struct {
		name  string
		alias *cobra.Command
		verb  *cobra.Command
	}{
		{"agent init → init expert", agentInitCmd, initExpertCmd},
		{"agent publish → apply", agentPublishCmd, applyCmd},
		{"workflow init → init workflow", workflowInitCmd, initWorkflowCmd},
		{"workflow run → apply", workflowRunCmd, applyCmd},
		{"workflow publish → apply", workflowPublishCmd, applyCmd},
		{"publish → apply", publishCmd, applyCmd},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if runEPtr(tc.alias) == 0 {
				t.Fatalf("alias %q has nil RunE", tc.alias.Use)
			}
			if runEPtr(tc.alias) != runEPtr(tc.verb) {
				t.Errorf("alias %q RunE does not delegate to verb %q", tc.alias.Use, tc.verb.Use)
			}
		})
	}
}

// ── back-compat: existing verbs still registered ──────────────────────────────

func TestBackCompat_VerbsStillResolve(t *testing.T) {
	cases := [][]string{
		{"apply"},
		{"init"},
		{"init", "workflow"},
		{"init", "expert"},
	}
	for _, args := range cases {
		c, _, err := rootCmd.Find(args)
		if err != nil {
			t.Fatalf("back-compat verb %v no longer resolves: %v", args, err)
		}
		if c == rootCmd {
			t.Errorf("back-compat verb %v resolved to root (not registered)", args)
		}
	}
}

// ── beginner help group ───────────────────────────────────────────────────────

// TestBeginnerGroup_Registered proves the beginner group exists and that the
// existing beginner commands are tagged into it. `doctor` (#1489) is not
// asserted — it joins the group only once that command is registered.
func TestBeginnerGroup_Registered(t *testing.T) {
	var found bool
	for _, g := range rootCmd.Groups() {
		if g.ID == beginnerGroupID {
			found = true
		}
	}
	if !found {
		t.Fatalf("beginner group %q not registered on root", beginnerGroupID)
	}

	want := map[string]bool{"workflow": true, "agent": true, "publish": true, "logs": true, "result": true}
	for _, c := range rootCmd.Commands() {
		if want[c.Name()] && c.GroupID != beginnerGroupID {
			t.Errorf("command %q GroupID = %q, want beginner group %q", c.Name(), c.GroupID, beginnerGroupID)
		}
	}
}

// TestRootHelp_ListsBeginnerCommands proves the noun-first commands appear in
// `zynax --help` output under the beginner heading.
func TestRootHelp_ListsBeginnerCommands(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"--help"})
	defer rootCmd.SetArgs(nil)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("--help error: %v", err)
	}
	help := out.String()
	for _, name := range []string{"agent", "workflow", "publish"} {
		if !strings.Contains(help, name) {
			t.Errorf("--help missing beginner command %q:\n%s", name, help)
		}
	}
}

// ── behavioural parity: aliases hit the same apply path ───────────────────────

// applyAliasServer accepts an apply POST and records the kind it observed.
func applyAliasServer(t *testing.T, gotKind *string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b bytes.Buffer
		_, _ = b.ReadFrom(r.Body)
		body := b.String()
		switch {
		case strings.Contains(body, "kind: "+kindAgentDef):
			*gotKind = kindAgentDef
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"agent_id": "agent-1"})
		default:
			*gotKind = kindWorkflow
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"run_id": "wf-1"})
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestPublishAlias_AutoDetectsKind(t *testing.T) {
	cases := []struct {
		name     string
		cmd      *cobra.Command
		manifest string
		wantKind string
	}{
		{"publish workflow", publishCmd, "kind: " + kindWorkflow, kindWorkflow},
		{"publish agentdef", publishCmd, "kind: " + kindAgentDef, kindAgentDef},
		{"workflow run", workflowRunCmd, "kind: " + kindWorkflow, kindWorkflow},
		{"agent publish", agentPublishCmd, "kind: " + kindAgentDef, kindAgentDef},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotKind string
			srv := applyAliasServer(t, &gotKind)
			apiURL = srv.URL
			applyDryRun = false
			applyEngine = ""

			dir := t.TempDir()
			f := filepath.Join(dir, "m.yaml")
			if err := os.WriteFile(f, []byte(tc.manifest), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := fakeCmd(t)
			if err := tc.cmd.RunE(cmd, []string{f}); err != nil {
				t.Fatalf("alias RunE error: %v", err)
			}
			if gotKind != tc.wantKind {
				t.Errorf("server observed kind %q, want %q", gotKind, tc.wantKind)
			}
		})
	}
}

// TestNounInitAliases_ScaffoldSameAsVerb proves `agent init`/`workflow init`
// emit the same manifest kind as the underlying `init expert`/`init workflow`.
func TestNounInitAliases_ScaffoldSameAsVerb(t *testing.T) {
	root := repoRoot(t)
	initTemplateDir = filepath.Join(root, "spec/templates")
	initOutput = ""

	cases := []struct {
		name     string
		cmd      *cobra.Command
		wantKind string
	}{
		{"workflow init", workflowInitCmd, "kind: Workflow"},
		{"agent init", agentInitCmd, "kind: AgentDef"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := fakeCmd(t)
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := tc.cmd.RunE(cmd, nil); err != nil {
				t.Fatalf("alias init error: %v", err)
			}
			if !strings.Contains(out.String(), tc.wantKind) {
				t.Errorf("scaffold missing %q, got:\n%s", tc.wantKind, out.String())
			}
		})
	}
}
