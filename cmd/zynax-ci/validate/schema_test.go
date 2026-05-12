// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax-ci/validate"
)

func writeJSON(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSchema_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "good.json", `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object"
	}`)
	errs, err := validate.Schema(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestSchema_MissingSchemaField(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "no-schema.json", `{"type": "object"}`)
	errs, err := validate.Schema(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected error for missing $schema field")
	}
}

func TestSchema_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := writeJSON(t, dir, "bad.json", `{not json}`)
	errs, err := validate.Schema(path)
	if err != nil {
		t.Fatalf("unexpected I/O error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected error for invalid JSON")
	}
}

func TestSchema_FileNotFound(t *testing.T) {
	_, err := validate.Schema("/nonexistent/path/schema.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestSchemaDir_ValidDir(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "a.json", `{"$schema":"https://json-schema.org/draft/2020-12/schema"}`)
	writeJSON(t, dir, "b.json", `{"$schema":"https://json-schema.org/draft/2020-12/schema"}`)
	results, err := validate.SchemaDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if len(r.Errors) != 0 {
			t.Errorf("unexpected errors for %s: %v", r.File, r.Errors)
		}
	}
}

func TestSchemaDir_OneFailingFile(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "good.json", `{"$schema":"https://json-schema.org/draft/2020-12/schema"}`)
	writeJSON(t, dir, "bad.json", `{"type":"object"}`)
	results, err := validate.SchemaDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	failCount := 0
	for _, r := range results {
		if len(r.Errors) > 0 {
			failCount++
		}
	}
	if failCount != 1 {
		t.Errorf("expected 1 failing file, got %d", failCount)
	}
}

func TestSchemaDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	results, err := validate.SchemaDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results in empty dir, got %d", len(results))
	}
}
