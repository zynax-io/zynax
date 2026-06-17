// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax/validate"
)

const validWorkflow = `kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: wf
spec:
  initial_state: start
  states:
    start:
      on:
        - event: done
          goto: finish
    finish:
      type: terminal
`

func writeTmp(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	f := filepath.Join(dir, "wf.yaml")
	if err := os.WriteFile(f, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestDataFlow_Valid(t *testing.T) {
	errs, err := validate.DataFlow(writeTmp(t, validWorkflow))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("expected no data-flow errors, got %v", errs)
	}
}

func TestDataFlow_UndefinedInitialState(t *testing.T) {
	body := `kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: wf
spec:
  initial_state: nope
  states:
    start:
      type: terminal
`
	errs, err := validate.DataFlow(writeTmp(t, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 1 || errs[0].Path != "/spec/initial_state" {
		t.Fatalf("expected initial_state error, got %v", errs)
	}
}

func TestDataFlow_UndefinedGotoTarget(t *testing.T) {
	body := `kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: wf
spec:
  initial_state: start
  states:
    start:
      on:
        - event: done
          goto: missing
`
	errs, err := validate.DataFlow(writeTmp(t, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 1 || errs[0].Path != "/spec/states/start/on/0/goto" {
		t.Fatalf("expected goto error, got %v", errs)
	}
}

func TestDataFlow_NonWorkflowKindSkipped(t *testing.T) {
	body := "kind: AgentDef\napiVersion: zynax.io/v1\n"
	errs, err := validate.DataFlow(writeTmp(t, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("expected non-Workflow to be skipped, got %v", errs)
	}
}

func TestDataFlow_FileNotFound(t *testing.T) {
	if _, err := validate.DataFlow("/nonexistent/wf.yaml"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestDataFlow_InvalidYAMLSkipped(t *testing.T) {
	// Parse errors are owned by the schema pass; DataFlow returns no errors.
	errs, err := validate.DataFlow(writeTmp(t, ":\t bad yaml {{{"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("expected parse errors to be skipped, got %v", errs)
	}
}

func TestFile_MissingFile(t *testing.T) {
	if _, err := validate.File("/nonexistent/wf.yaml", "spec/schemas"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFile_SchemaAndDataFlowCombined(t *testing.T) {
	// initial_state references an undefined state — data-flow catches it even
	// though the manifest is schema-valid.
	body := `kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: wf
spec:
  initial_state: ghost
  states:
    start:
      type: terminal
`
	_, file, _, _ := runtime.Caller(0)
	schemaDir := filepath.Join(filepath.Dir(file), "../../../spec/schemas")
	errs, err := validate.File(writeTmp(t, body), schemaDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("expected combined validation to report the data-flow error")
	}
}
