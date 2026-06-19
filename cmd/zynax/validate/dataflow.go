// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// DataFlow validates the state-machine data-flow of a Workflow manifest beyond
// what the JSON Schema can express: it checks that spec.initial_state resolves
// to a defined state and that every transition 'goto' targets a defined state.
//
// These are the compiler-enforced invariants the workflow.schema.json explicitly
// defers (see its 'states'/'initial_state' descriptions). DataFlow only inspects
// kind: Workflow manifests; for any other kind it returns (nil, nil).
//
// Returns (nil, nil) when the data-flow is sound; ([errors…], nil) on data-flow
// violations; (nil, err) on I/O or structural failures.
func DataFlow(filePath string) ([]ValidationError, error) {
	raw, err := os.ReadFile(filePath) //nolint:gosec // filePath is caller-supplied manifest path
	if err != nil {
		return nil, fmt.Errorf("validate: read %q: %w", filePath, err)
	}
	return DataFlowBytes(filePath, raw)
}

// DataFlowBytes runs the state-machine data-flow checks over already-loaded
// manifest bytes, attributing errors to filePath. It backs both DataFlow (file
// on disk) and the context-bound validation path (#1387).
func DataFlowBytes(filePath string, raw []byte) ([]ValidationError, error) {
	var doc struct {
		Kind string `yaml:"kind"`
		Spec struct {
			InitialState string `yaml:"initial_state"`
			States       map[string]struct {
				On []struct {
					Goto string `yaml:"goto"`
				} `yaml:"on"`
			} `yaml:"states"`
		} `yaml:"spec"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		// Structural/YAML errors are surfaced by Manifest's schema pass; skip here.
		return nil, nil //nolint:nilerr // schema validation owns parse-error reporting
	}
	if doc.Kind != kindWorkflow {
		return nil, nil
	}

	var errs []ValidationError

	if doc.Spec.InitialState != "" {
		if _, ok := doc.Spec.States[doc.Spec.InitialState]; !ok {
			errs = append(errs, ValidationError{
				File:    filePath,
				Path:    "/spec/initial_state",
				Message: fmt.Sprintf("initial_state %q is not a defined state", doc.Spec.InitialState),
			})
		}
	}

	// Iterate states in sorted order for deterministic error output.
	names := make([]string, 0, len(doc.Spec.States))
	for name := range doc.Spec.States {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		state := doc.Spec.States[name]
		for i, tr := range state.On {
			if tr.Goto == "" {
				continue
			}
			if _, ok := doc.Spec.States[tr.Goto]; !ok {
				errs = append(errs, ValidationError{
					File:    filePath,
					Path:    fmt.Sprintf("/spec/states/%s/on/%d/goto", name, i),
					Message: fmt.Sprintf("transition target %q is not a defined state", tr.Goto),
				})
			}
		}
	}

	return errs, nil
}
