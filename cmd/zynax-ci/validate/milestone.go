// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// Milestone validates the YAML file at stateFile against the JSON Schema
// at schemaFile (Draft 2020-12). It returns the validation errors sorted by
// instance path; an empty slice means the file conforms. A returned error is an
// operational failure (file unreadable, schema uncompilable), not a schema
// violation.
//
// This is the Go replacement for scripts/validate_milestone_state.py: it makes
// the same accept/reject decision (same schema, same draft). Error message
// wording comes from santhosh-tekuri/jsonschema and is not identical to the
// Python jsonschema text — only the decision and the OK output are parity-bound.
func Milestone(stateFile, schemaFile string) ([]ValidationError, error) {
	absSchema, err := filepath.Abs(schemaFile)
	if err != nil {
		return nil, fmt.Errorf("validate milestone: resolve schema path: %w", err)
	}
	sch, err := compileSchema(absSchema)
	if err != nil {
		return nil, fmt.Errorf("validate milestone: compile schema %q: %w", absSchema, err)
	}

	raw, err := os.ReadFile(stateFile) //nolint:gosec // caller-supplied repo path
	if err != nil {
		return nil, fmt.Errorf("validate milestone: read %q: %w", stateFile, err)
	}

	var doc interface{}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return single(stateFile, "", "YAML parse error: "+err.Error()), nil
	}

	jsonBytes, err := json.Marshal(normalise(doc))
	if err != nil {
		return nil, fmt.Errorf("validate milestone: marshal %q: %w", stateFile, err)
	}
	var instance interface{}
	if err := json.Unmarshal(jsonBytes, &instance); err != nil {
		return nil, fmt.Errorf("validate milestone: unmarshal %q: %w", stateFile, err)
	}

	valErr := sch.Validate(instance)
	if valErr == nil {
		return nil, nil
	}
	var ve *jsonschema.ValidationError
	if errors.As(valErr, &ve) {
		errs := flattenManifestErrors(stateFile, ve)
		sort.SliceStable(errs, func(i, j int) bool { return errs[i].Path < errs[j].Path })
		return errs, nil
	}
	return nil, fmt.Errorf("validate milestone: %w", valErr)
}
