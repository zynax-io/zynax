// SPDX-License-Identifier: Apache-2.0

// Package validate provides local validation for Zynax manifests,
// REASONS Canvases, and JSON Schema documents.
package validate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// ValidationError is a single schema validation failure.
type ValidationError struct {
	File    string `json:"file"`
	Path    string `json:"path,omitempty"` // JSON Pointer to the failing element
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: at %s: %s", e.File, e.Path, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.File, e.Message)
}

// Manifest validates the YAML manifest at filePath against the JSON Schema
// for its kind. schemaDir is the directory containing <kind>.schema.json
// files (default: "spec/schemas" relative to CWD).
//
// Returns (nil, nil) on success; ([errors…], nil) on schema violations;
// (nil, err) on I/O or structural failures.
func Manifest(filePath, schemaDir string) ([]ValidationError, error) {
	raw, err := os.ReadFile(filePath) //nolint:gosec // filePath is caller-supplied manifest path
	if err != nil {
		return nil, fmt.Errorf("validate: read %q: %w", filePath, err)
	}

	var doc interface{}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return single(filePath, "", "YAML parse error: "+err.Error()), nil
	}

	m, ok := doc.(map[string]interface{})
	if !ok {
		return single(filePath, "", "manifest must be a YAML mapping"), nil
	}

	kind, _ := m["kind"].(string)
	if kind == "" {
		return single(filePath, "/kind", "missing or empty 'kind' field"), nil
	}

	schemaFile, err := kindToSchemaFile(kind)
	if err != nil {
		return single(filePath, "/kind", err.Error()), nil
	}

	absSchema, err := filepath.Abs(filepath.Join(schemaDir, schemaFile))
	if err != nil {
		return nil, fmt.Errorf("validate: resolve schema path: %w", err)
	}

	sch, err := compileSchema(absSchema)
	if err != nil {
		return nil, fmt.Errorf("validate: compile schema %q: %w", absSchema, err)
	}

	jsonBytes, err := json.Marshal(normalise(doc))
	if err != nil {
		return nil, fmt.Errorf("validate: marshal to JSON: %w", err)
	}

	var instance interface{}
	if err := json.Unmarshal(jsonBytes, &instance); err != nil {
		return nil, fmt.Errorf("validate: unmarshal JSON: %w", err)
	}

	if err := sch.Validate(instance); err != nil {
		var ve *jsonschema.ValidationError
		if errors.As(err, &ve) {
			return flattenErrors(filePath, ve), nil
		}
		return nil, fmt.Errorf("validate: %w", err)
	}
	return nil, nil
}

// File runs the full local validation pipeline over the manifest at filePath:
// JSON Schema validation (Manifest) followed by state-machine data-flow checks
// (DataFlow) for Workflow manifests. It returns the combined list of errors.
//
// Returns (nil, nil) when the manifest is valid; ([errors…], nil) on schema or
// data-flow violations; (nil, err) on I/O or structural failures.
func File(filePath, schemaDir string) ([]ValidationError, error) {
	schemaErrs, err := Manifest(filePath, schemaDir)
	if err != nil {
		return nil, err
	}
	flowErrs, err := DataFlow(filePath)
	if err != nil {
		return nil, err
	}
	combined := append(schemaErrs, flowErrs...) //nolint:gocritic // intentional new slice
	return combined, nil
}

// compileSchema compiles a JSON Schema from an absolute file path.
func compileSchema(absPath string) (*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft2020
	schema, err := c.Compile("file://" + absPath)
	if err != nil {
		return nil, fmt.Errorf("validate: compile schema %q: %w", absPath, err)
	}
	return schema, nil
}

// flattenErrors collects leaf validation errors from the error tree.
func flattenErrors(file string, ve *jsonschema.ValidationError) []ValidationError {
	if len(ve.Causes) == 0 {
		return single(file, ve.InstanceLocation, ve.Message)
	}
	var out []ValidationError
	for _, c := range ve.Causes {
		out = append(out, flattenErrors(file, c)...)
	}
	return out
}

// normalise converts map[interface{}]interface{} (yaml.v3 edge case) to
// map[string]interface{} so json.Marshal succeeds on all valid YAML.
func normalise(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, v2 := range val {
			out[k] = normalise(v2)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, v2 := range val {
			out[fmt.Sprintf("%v", k)] = normalise(v2)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, v2 := range val {
			out[i] = normalise(v2)
		}
		return out
	default:
		return val
	}
}

// kindToSchemaFile maps a manifest kind to its JSON Schema filename.
func kindToSchemaFile(kind string) (string, error) {
	switch kind {
	case "Workflow":
		return "workflow.schema.json", nil
	case "AgentDef":
		return "agent-def.schema.json", nil
	case "Policy":
		return "policy.schema.json", nil
	case "Scenario":
		return "scenario.schema.json", nil
	default:
		return "", fmt.Errorf("no JSON schema for kind %q — supported: Workflow, AgentDef, Policy, Scenario", kind)
	}
}

func single(file, path, msg string) []ValidationError {
	return []ValidationError{{File: file, Path: path, Message: msg}}
}
