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

// ManifestResult holds the outcome of validating a single manifest file.
type ManifestResult struct {
	File   string            `json:"file"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// BatchManifests walks dir for *.yaml files, filters by kind, and validates each
// against the JSON Schema at schemaPath. Returns one result per matching file.
func BatchManifests(dir, kind, schemaPath string) ([]ManifestResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("validate manifests: read dir %q: %w", dir, err)
	}

	absSchema, err := filepath.Abs(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("validate manifests: resolve schema path: %w", err)
	}
	sch, err := compileSchema(absSchema)
	if err != nil {
		return nil, fmt.Errorf("validate manifests: compile schema %q: %w", absSchema, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)

	var results []ManifestResult
	for _, path := range files {
		r, err := validateManifestFile(path, kind, sch)
		if err != nil {
			return nil, err
		}
		if r != nil {
			results = append(results, *r)
		}
	}
	return results, nil
}

// validateManifestFile validates one YAML file against the schema for the given kind.
// Returns nil (no error) when the file should be skipped (wrong kind or not a mapping).
func validateManifestFile(path, kind string, sch *jsonschema.Schema) (*ManifestResult, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is from filepath.ReadDir on caller-supplied dir
	if err != nil {
		return nil, fmt.Errorf("validate manifests: read %q: %w", path, err)
	}

	var doc interface{}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		r := ManifestResult{File: path, Errors: single(path, "", "YAML parse error: "+err.Error())}
		return &r, nil
	}

	m, ok := normaliseMap(doc)
	if !ok {
		return nil, nil // not a mapping — skip silently
	}
	if docKind, _ := m["kind"].(string); docKind != kind {
		return nil, nil // wrong kind — skip
	}

	jsonBytes, err := json.Marshal(normalise(doc))
	if err != nil {
		return nil, fmt.Errorf("validate manifests: marshal %q: %w", path, err)
	}

	var instance interface{}
	if err := json.Unmarshal(jsonBytes, &instance); err != nil {
		return nil, fmt.Errorf("validate manifests: unmarshal %q: %w", path, err)
	}

	if valErr := sch.Validate(instance); valErr != nil {
		var ve *jsonschema.ValidationError
		if errors.As(valErr, &ve) {
			r := ManifestResult{File: path, Errors: flattenManifestErrors(path, ve)}
			return &r, nil
		}
		return nil, fmt.Errorf("validate manifests: %w", valErr)
	}
	r := ManifestResult{File: path}
	return &r, nil
}

func compileSchema(absPath string) (*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft2020
	schema, err := c.Compile("file://" + absPath)
	if err != nil {
		return nil, fmt.Errorf("validate: compile schema %q: %w", absPath, err)
	}
	return schema, nil
}

func flattenManifestErrors(file string, ve *jsonschema.ValidationError) []ValidationError {
	if len(ve.Causes) == 0 {
		return single(file, ve.InstanceLocation, ve.Message)
	}
	var out []ValidationError
	for _, c := range ve.Causes {
		out = append(out, flattenManifestErrors(file, c)...)
	}
	return out
}

// normalise converts map[interface{}]interface{} produced by yaml.v3 edge
// cases to map[string]interface{} so json.Marshal succeeds on all valid YAML.
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

func normaliseMap(v interface{}) (map[string]interface{}, bool) {
	n := normalise(v)
	m, ok := n.(map[string]interface{})
	return m, ok
}
