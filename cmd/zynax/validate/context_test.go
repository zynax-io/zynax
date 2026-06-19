// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// writeFile writes content into dir/name and returns dir for chaining.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// writeContextIndex writes a scenario index whose spec.context is the given YAML
// block (already indented under "  context:") into a fresh temp dir and returns
// the index path and its dir.
func writeContextIndex(t *testing.T, contextYAML string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	body := `kind: Scenario
apiVersion: zynax.io/v1alpha1
metadata:
  name: demo
spec:
  members:
    - id: workflow
      kind: Workflow
      file: workflow.yaml
  apply_order:
    - workflow
` + contextYAML
	idx := filepath.Join(dir, ScenarioIndexFile)
	if err := os.WriteFile(idx, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return idx, dir
}

// --- Decisive test 1: a valid context block parses, resolves, and substitutes
// into a Workflow's {{ .ctx.* }} surface.

func TestResolveContext_ValidBlockSubstitutes(t *testing.T) {
	idx, dir := writeContextIndex(t, `  context:
    sources:
      - key: diff
        files:
          - diff.patch
    max_tokens: 8000
`)
	writeFile(t, dir, "diff.patch", "DIFF-BODY-LINE-1\nDIFF-BODY-LINE-2\n")

	block, err := ParseContextBlock(idx)
	if err != nil {
		t.Fatalf("ParseContextBlock: %v", err)
	}
	if block == nil {
		t.Fatal("expected a context block, got nil")
	}
	values, err := ResolveContext(block, dir)
	if err != nil {
		t.Fatalf("ResolveContext: %v", err)
	}
	if !strings.Contains(values["diff"], "DIFF-BODY-LINE-1") {
		t.Fatalf("resolved diff missing file content: %q", values["diff"])
	}

	wf := []byte("kind: Workflow\nspec:\n  states:\n    review:\n      actions:\n        - capability: codereview\n          input:\n            prompt: |\n              ```diff\n              {{ .ctx.diff }}\n              ```\n")
	bound, err := BindContextIntoWorkflow(wf, values)
	if err != nil {
		t.Fatalf("BindContextIntoWorkflow: %v", err)
	}
	if !strings.Contains(string(bound), "DIFF-BODY-LINE-1") {
		t.Fatalf("bound workflow did not substitute {{ .ctx.diff }}:\n%s", bound)
	}
	if strings.Contains(string(bound), "{{") {
		t.Fatalf("bound workflow still contains a template reference:\n%s", bound)
	}
}

// --- Decisive test 2: a context block carrying a routing/provider field is
// REJECTED at compile time (data-only safeguard, ADR-013/035).

func TestParseContextBlock_RejectsRoutingFields(t *testing.T) {
	for _, field := range []string{"provider", "model", "endpoint", "url", "base_url", "api_key"} {
		t.Run(field, func(t *testing.T) {
			idx, dir := writeContextIndex(t, `  context:
    sources:
      - key: diff
        files:
          - diff.patch
    max_tokens: 8000
    `+field+`: anything
`)
			writeFile(t, dir, "diff.patch", "x")
			_, err := ParseContextBlock(idx)
			if err == nil {
				t.Fatalf("expected rejection of routing field %q, got nil", field)
			}
			if !strings.Contains(err.Error(), field) {
				t.Fatalf("error should cite the field %q: %v", field, err)
			}
		})
	}
}

// A routing field nested inside a source entry is also rejected.
func TestParseContextBlock_RejectsRoutingFieldInSource(t *testing.T) {
	idx, dir := writeContextIndex(t, `  context:
    sources:
      - key: diff
        files:
          - diff.patch
        endpoint: http://evil.example
    max_tokens: 8000
`)
	writeFile(t, dir, "diff.patch", "x")
	_, err := ParseContextBlock(idx)
	if err == nil {
		t.Fatal("expected rejection of routing field nested in a source, got nil")
	}
}

// An unknown top-level field is rejected (data-only: only sources/max_tokens/overflow).
func TestParseContextBlock_RejectsUnknownField(t *testing.T) {
	idx, dir := writeContextIndex(t, `  context:
    sources:
      - key: diff
        files:
          - diff.patch
    max_tokens: 8000
    language: go
`)
	writeFile(t, dir, "diff.patch", "x")
	_, err := ParseContextBlock(idx)
	if err == nil {
		t.Fatal("expected rejection of unknown field, got nil")
	}
}

// max_tokens is required.
func TestParseContextBlock_RequiresMaxTokens(t *testing.T) {
	idx, dir := writeContextIndex(t, `  context:
    sources:
      - key: diff
        files:
          - diff.patch
`)
	writeFile(t, dir, "diff.patch", "x")
	_, err := ParseContextBlock(idx)
	if err == nil {
		t.Fatal("expected error for missing max_tokens, got nil")
	}
}

// --- Decisive test 3: the max_tokens cap is enforced.

func TestResolveContext_TruncatesOldestOverBudget(t *testing.T) {
	// max_tokens 1 → budget = 4 chars. "old" (3) + "new12345" (8) = 11 > 4.
	idx, dir := writeContextIndex(t, `  context:
    sources:
      - key: old
        files:
          - old.txt
      - key: new
        files:
          - new.txt
    max_tokens: 1
    overflow: truncate-oldest
`)
	writeFile(t, dir, "old.txt", "old")
	writeFile(t, dir, "new.txt", "new12345")

	block, err := ParseContextBlock(idx)
	if err != nil {
		t.Fatalf("ParseContextBlock: %v", err)
	}
	values, err := ResolveContext(block, dir)
	if err != nil {
		t.Fatalf("ResolveContext: %v", err)
	}
	total := len(values["old"]) + len(values["new"])
	if total > block.MaxTokens*tokenCharsPerToken {
		t.Fatalf("combined content %d chars exceeds budget %d", total, block.MaxTokens*tokenCharsPerToken)
	}
	if values["old"] != "" {
		t.Errorf("oldest source should be dropped first, got %q", values["old"])
	}
}

func TestResolveContext_ErrorOverflowFailsOverBudget(t *testing.T) {
	idx, dir := writeContextIndex(t, `  context:
    sources:
      - key: big
        files:
          - big.txt
    max_tokens: 1
    overflow: error
`)
	writeFile(t, dir, "big.txt", strings.Repeat("x", 100))

	block, err := ParseContextBlock(idx)
	if err != nil {
		t.Fatalf("ParseContextBlock: %v", err)
	}
	if _, err := ResolveContext(block, dir); err == nil {
		t.Fatal("expected over-budget error under overflow: error, got nil")
	}
}

// --- Strict isolation: each ResolveContext call returns a fresh map built only
// from its own block; one scenario's values never appear in another's.

func TestResolveContext_IsolatesScenarios(t *testing.T) {
	idxA, dirA := writeContextIndex(t, `  context:
    sources:
      - key: diff
        files:
          - diff.patch
    max_tokens: 8000
`)
	writeFile(t, dirA, "diff.patch", "SCENARIO-A-CONTENT")
	idxB, dirB := writeContextIndex(t, `  context:
    sources:
      - key: diff
        files:
          - diff.patch
    max_tokens: 8000
`)
	writeFile(t, dirB, "diff.patch", "SCENARIO-B-CONTENT")

	blockA, _ := ParseContextBlock(idxA)
	blockB, _ := ParseContextBlock(idxB)
	valsA, err := ResolveContext(blockA, dirA)
	if err != nil {
		t.Fatalf("ResolveContext A: %v", err)
	}
	valsB, err := ResolveContext(blockB, dirB)
	if err != nil {
		t.Fatalf("ResolveContext B: %v", err)
	}
	if strings.Contains(valsA["diff"], "SCENARIO-B") || strings.Contains(valsB["diff"], "SCENARIO-A") {
		t.Fatal("context leaked across scenarios")
	}
}

// --- A source file that escapes the scenario directory is rejected (traversal).

func TestResolveContext_RejectsPathTraversal(t *testing.T) {
	idx, dir := writeContextIndex(t, `  context:
    sources:
      - key: secret
        files:
          - ../../etc/passwd
    max_tokens: 8000
`)
	block, err := ParseContextBlock(idx)
	if err != nil {
		t.Fatalf("ParseContextBlock: %v", err)
	}
	if _, err := ResolveContext(block, dir); err == nil {
		t.Fatal("expected path-traversal rejection, got nil")
	}
}

// --- An {{ .ctx.<key> }} reference the block does not supply fails loudly.

func TestBindContextIntoWorkflow_FailsOnUnresolvedKey(t *testing.T) {
	wf := []byte("kind: Workflow\nspec:\n  prompt: \"{{ .ctx.missing }}\"\n")
	if _, err := BindContextIntoWorkflow(wf, map[string]string{"diff": "x"}); err == nil {
		t.Fatal("expected error for unresolved {{ .ctx.missing }}, got nil")
	}
}

// --- The shipped code-review scenario fixture binds end-to-end: its declared
// context block resolves, substitutes into the Workflow member's {{ .ctx.diff }},
// and the bound member validates against the schema with no unresolved key.

func TestScenarioFile_CodeReviewFixtureBindsAndValidates(t *testing.T) {
	root := repoRoot(t)
	idx := filepath.Join(root, "spec", "scenarios", "code-review", "scenario.yaml")
	if _, err := os.Stat(idx); err != nil {
		t.Skipf("fixture not present: %v", err)
	}
	schemaDir := filepath.Join(root, "spec", "schemas")

	errs, err := ScenarioFile(idx, schemaDir)
	if err != nil {
		t.Fatalf("ScenarioFile: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("expected zero validation errors for the code-review fixture, got: %v", errs)
	}

	// Prove the diff actually lands in the bound workflow.
	block, err := ParseContextBlock(idx)
	if err != nil {
		t.Fatalf("ParseContextBlock: %v", err)
	}
	values, err := ResolveContext(block, filepath.Dir(idx))
	if err != nil {
		t.Fatalf("ResolveContext: %v", err)
	}
	wf, err := os.ReadFile(filepath.Join(root, "spec", "scenarios", "code-review", "workflow.yaml")) //nolint:gosec // fixed repo-relative fixture path in a test
	if err != nil {
		t.Fatal(err)
	}
	bound, err := BindContextIntoWorkflow(wf, values)
	if err != nil {
		t.Fatalf("BindContextIntoWorkflow: %v", err)
	}
	if !strings.Contains(string(bound), "func Refund") {
		t.Fatalf("bound workflow did not contain the injected diff content:\n%s", bound)
	}
	// No unresolved reference should remain in any scalar VALUE (comments, which
	// are documentation, are intentionally not templated).
	if scalarContainsTemplate(t, bound) {
		t.Fatalf("a scalar value in the bound workflow still contains a {{ ... }} reference:\n%s", bound)
	}
}

// scalarContainsTemplate reports whether any scalar string node in the YAML bytes
// still contains a template reference (ignoring comments).
func scalarContainsTemplate(t *testing.T, raw []byte) bool {
	t.Helper()
	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		t.Fatalf("re-parse bound YAML: %v", err)
	}
	return anyScalarHasTemplate(&root)
}

func anyScalarHasTemplate(n *yaml.Node) bool {
	if n.Kind == yaml.ScalarNode && strings.Contains(n.Value, "{{") {
		return true
	}
	for _, c := range n.Content {
		if anyScalarHasTemplate(c) {
			return true
		}
	}
	return false
}

// --- No context block declared → (nil, nil); binding is a no-op.

func TestParseContextBlock_NoBlock(t *testing.T) {
	idx := writeIndex(t, validIndex)
	block, err := ParseContextBlock(idx)
	if err != nil {
		t.Fatalf("ParseContextBlock: %v", err)
	}
	if block != nil {
		t.Fatalf("expected nil block when none declared, got %+v", block)
	}
}
