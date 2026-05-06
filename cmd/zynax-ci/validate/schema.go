// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SchemaResult holds the outcome of validating a single JSON Schema file.
type SchemaResult struct {
	File   string            `json:"file"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// Schema validates a single JSON Schema file: well-formed JSON + $schema field present.
func Schema(filePath string) ([]ValidationError, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("validate schema: read %q: %w", filePath, err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return single(filePath, "", "invalid JSON: "+err.Error()), nil
	}
	if _, ok := doc["$schema"]; !ok {
		return single(filePath, "/$schema", "missing $schema field"), nil
	}
	return nil, nil
}

// SchemaDir walks dir for *.json files and validates each with Schema.
func SchemaDir(dir string) ([]SchemaResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("validate schema: read dir %q: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)

	var results []SchemaResult
	for _, path := range files {
		errs, err := Schema(path)
		if err != nil {
			return nil, err
		}
		results = append(results, SchemaResult{File: path, Errors: errs})
	}
	return results, nil
}
