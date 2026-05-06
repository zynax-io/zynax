// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax/validate"
)

// repoSchemaDir resolves spec/schemas/ relative to this source file,
// so tests work regardless of the CWD they are invoked from.
func repoSchemaDir() string {
	_, file, _, _ := runtime.Caller(0)
	// file = cmd/zynax/validate/manifest_test.go (absolute)
	// three levels up reaches the repo root
	return filepath.Join(filepath.Dir(file), "../../../spec/schemas")
}

func TestManifest_ValidWorkflow(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	fixture := filepath.Join(filepath.Dir(file), "../../../spec/workflows/examples/code-review.yaml")

	errs, err := validate.Manifest(fixture, repoSchemaDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("unexpected validation error: %s", e)
		}
	}
}

func TestManifest_ValidAgentDef(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	fixture := filepath.Join(filepath.Dir(file), "../../../spec/workflows/examples/agent-def-example.yaml")

	errs, err := validate.Manifest(fixture, repoSchemaDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("unexpected validation error: %s", e)
		}
	}
}

func TestManifest_MissingRequiredField(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.yaml")
	// Missing 'spec' — required by the Workflow schema.
	if err := os.WriteFile(f, []byte("kind: Workflow\napiVersion: zynax.io/v1\nmetadata:\n  name: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	errs, err := validate.Manifest(f, repoSchemaDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("expected validation errors for missing 'spec', got none")
	}
}

func TestManifest_UnknownKind(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "unknown.yaml")
	if err := os.WriteFile(f, []byte("kind: Unknown\napiVersion: zynax.io/v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	errs, err := validate.Manifest(f, repoSchemaDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("expected error for unknown kind, got none")
	}
	if errs[0].Path != "/kind" {
		t.Errorf("expected error path '/kind', got %q", errs[0].Path)
	}
}

func TestManifest_MissingKind(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "nokind.yaml")
	if err := os.WriteFile(f, []byte("apiVersion: zynax.io/v1\nmetadata:\n  name: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	errs, err := validate.Manifest(f, repoSchemaDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("expected error for missing kind, got none")
	}
}

func TestManifest_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "broken.yaml")
	if err := os.WriteFile(f, []byte(":\t invalid yaml {{{"), 0o644); err != nil {
		t.Fatal(err)
	}

	errs, err := validate.Manifest(f, repoSchemaDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("expected error for invalid YAML, got none")
	}
}

func TestManifest_FileNotFound(t *testing.T) {
	_, err := validate.Manifest("/nonexistent/path/manifest.yaml", repoSchemaDir())
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
