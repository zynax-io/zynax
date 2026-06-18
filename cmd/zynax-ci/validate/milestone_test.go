// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax-ci/validate"
)

const milestoneSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["active", "history"],
  "properties": {
    "active": { "type": "object" },
    "history": { "type": "array" }
  }
}`

func writeMilestoneFixture(t *testing.T, stateYAML string) (stateFile, schemaFile string) {
	t.Helper()
	dir := t.TempDir()
	stateFile = filepath.Join(dir, "milestone.yaml")
	schemaFile = filepath.Join(dir, "milestone.schema.json")
	if err := os.WriteFile(stateFile, []byte(stateYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(schemaFile, []byte(milestoneSchema), 0o600); err != nil {
		t.Fatal(err)
	}
	return stateFile, schemaFile
}

func TestValidateMilestone_Conforms(t *testing.T) {
	stateFile, schemaFile := writeMilestoneFixture(t, "active: {}\nhistory: []\n")
	errs, err := validate.Milestone(stateFile, schemaFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected conforming, got errors: %+v", errs)
	}
}

func TestValidateMilestone_MissingRequired(t *testing.T) {
	stateFile, schemaFile := writeMilestoneFixture(t, "active: {}\n") // history missing
	errs, err := validate.Milestone(stateFile, schemaFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected a schema violation for missing 'history'")
	}
}

func TestValidateMilestone_BadYAML(t *testing.T) {
	stateFile, schemaFile := writeMilestoneFixture(t, "active: {{\n")
	errs, err := validate.Milestone(stateFile, schemaFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 1 || errs[0].Message == "" {
		t.Errorf("expected one YAML parse error, got: %+v", errs)
	}
}

func TestValidateMilestone_MissingFile(t *testing.T) {
	_, schemaFile := writeMilestoneFixture(t, "active: {}\nhistory: []\n")
	if _, err := validate.Milestone(filepath.Join(t.TempDir(), "nope.yaml"), schemaFile); err == nil {
		t.Error("expected an operational error for a missing state file")
	}
}
