// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// CapabilityResult holds the outcome of validating a single capability entry.
type CapabilityResult struct {
	File       string            `json:"file"`
	Capability string            `json:"capability"`
	Errors     []ValidationError `json:"errors,omitempty"`
}

// Capabilities walks dir for *.yaml files, extracts capability declarations
// from AgentDef manifests, and validates each against the schema at schemaPath.
// Both spec.capabilities and top-level capabilities arrays are supported.
func Capabilities(dir, schemaPath string) ([]CapabilityResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("validate capabilities: read dir %q: %w", dir, err)
	}

	absSchema, err := filepath.Abs(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("validate capabilities: resolve schema: %w", err)
	}
	sch, err := compileSchema(absSchema)
	if err != nil {
		return nil, fmt.Errorf("validate capabilities: compile schema %q: %w", absSchema, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)

	var results []CapabilityResult
	for _, path := range files {
		fileResults, err := validateCapabilitiesInFile(path, sch)
		if err != nil {
			return nil, err
		}
		results = append(results, fileResults...)
	}
	return results, nil
}

func validateCapabilitiesInFile(path string, sch *jsonschema.Schema) ([]CapabilityResult, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is from os.ReadDir on caller-supplied dir
	if err != nil {
		return nil, fmt.Errorf("validate capabilities: read %q: %w", path, err)
	}

	var doc interface{}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, nil // not a valid YAML doc — skip
	}

	m, ok := normaliseMap(doc)
	if !ok {
		return nil, nil
	}

	var results []CapabilityResult
	for _, capMap := range extractCapabilities(m) {
		r, err := validateOneCapability(path, capMap, sch)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

func validateOneCapability(path string, capMap map[string]interface{}, sch *jsonschema.Schema) (CapabilityResult, error) {
	name, _ := capMap["name"].(string)
	if name == "" {
		name = "?"
	}

	jsonBytes, err := json.Marshal(capMap)
	if err != nil {
		return CapabilityResult{}, fmt.Errorf("validate capabilities: marshal capability in %q: %w", path, err)
	}
	var instance interface{}
	if err := json.Unmarshal(jsonBytes, &instance); err != nil {
		return CapabilityResult{}, fmt.Errorf("validate capabilities: unmarshal capability in %q: %w", path, err)
	}

	if valErr := sch.Validate(instance); valErr != nil {
		var ve *jsonschema.ValidationError
		if errors.As(valErr, &ve) {
			return CapabilityResult{File: path, Capability: name, Errors: flattenManifestErrors(path, ve)}, nil
		}
		return CapabilityResult{}, fmt.Errorf("validate capabilities: %w", valErr)
	}
	return CapabilityResult{File: path, Capability: name}, nil
}

// extractCapabilities pulls capabilities from spec.capabilities or top-level capabilities.
func extractCapabilities(m map[string]interface{}) []map[string]interface{} {
	var raw []interface{}
	if spec, ok := m["spec"].(map[string]interface{}); ok {
		if caps, ok := spec["capabilities"].([]interface{}); ok {
			raw = caps
		}
	}
	if raw == nil {
		if caps, ok := m["capabilities"].([]interface{}); ok {
			raw = caps
		}
	}

	var out []map[string]interface{}
	for _, item := range raw {
		if capMap, ok := item.(map[string]interface{}); ok {
			out = append(out, capMap)
		}
	}
	return out
}
