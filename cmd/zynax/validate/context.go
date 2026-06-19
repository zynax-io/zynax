// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// routingFields are the keys a context-injection block may NEVER carry: any of
// them could redirect WHERE or HOW a capability runs. Context injection is
// strictly data-only (ADR-013/ADR-035) — provider/model/endpoint stay in
// AgentDef/overlay config and are never accepted from input_payload. The schema
// forbids these structurally (additionalProperties:false); this Go check is the
// load-bearing belt-and-suspenders compile-time rejection the canvas mandates
// (O3 safeguard), so a routing field is caught even if the schema is bypassed.
var routingFields = map[string]struct{}{
	"provider": {},
	"model":    {},
	"endpoint": {},
	"url":      {},
	"uri":      {},
	"base_url": {},
	"baseurl":  {},
	"host":     {},
	"api_base": {},
	"api_key":  {},
}

// tokenCharsPerToken is the approximate character-to-token ratio used to enforce
// the max_tokens budget. ~4 chars/token is the widely-used rough heuristic for
// English/code; the cap is a bound, not an exact tokenizer (which is provider
// specific and out of scope for a data-only injection contract).
const tokenCharsPerToken = 4

// ContextSource is one resolved entry of a context-injection block: the context
// key and the files (relative to the scenario directory) that source its value.
type ContextSource struct {
	Key   string
	Files []string
}

// ContextBlock is the parsed spec.context block of a scenario index. It is the
// declarative analogue of a hand-pasted prompt blob: bounded, file-rooted, and
// data-only.
type ContextBlock struct {
	Sources   []ContextSource
	MaxTokens int
	Overflow  string
}

// ContextError is a structured failure resolving or binding a context block. It
// fails loudly — there is no implicit fallback to empty values.
type ContextError struct {
	// Key is the offending context key (empty for block-level errors).
	Key string
	// Reason is a human-readable explanation.
	Reason string
}

func (e *ContextError) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("context: key %q: %s", e.Key, e.Reason)
	}
	return fmt.Sprintf("context: %s", e.Reason)
}

// contextIndex decodes only the spec.context block of a scenario index. The
// block is decoded into a generic map first so a routing field can be rejected
// before it is interpreted, then into the typed shape.
type contextIndex struct {
	Spec struct {
		Context map[string]any `yaml:"context"`
	} `yaml:"spec"`
}

// ParseContextBlock reads the scenario index at indexPath and returns its parsed
// spec.context block, or (nil, nil) when no context block is declared. It
// rejects any routing/provider field at the block or source level (data-only,
// ADR-013) before interpreting the block, so a malicious block can never be
// silently accepted.
func ParseContextBlock(indexPath string) (*ContextBlock, error) {
	raw, err := os.ReadFile(indexPath) //nolint:gosec // indexPath is caller-supplied scenario path
	if err != nil {
		return nil, fmt.Errorf("context: read %q: %w", indexPath, err)
	}
	var idx contextIndex
	if err := yaml.Unmarshal(raw, &idx); err != nil {
		return nil, fmt.Errorf("context: parse %q: %w", indexPath, err)
	}
	if len(idx.Spec.Context) == 0 {
		return nil, nil
	}

	// Data-only rejection (load-bearing safeguard): no routing field at the top
	// level of the block, and only the known data keys are accepted.
	if err := rejectRoutingFields(idx.Spec.Context); err != nil {
		return nil, err
	}
	for k := range idx.Spec.Context {
		switch k {
		case "sources", "max_tokens", "overflow":
		default:
			return nil, &ContextError{Reason: fmt.Sprintf("unknown field %q — a context block carries sources/max_tokens/overflow only (data-only)", k)}
		}
	}

	block, err := decodeContextBlock(idx.Spec.Context)
	if err != nil {
		return nil, err
	}
	return block, nil
}

// decodeContextBlock converts the generic map into the typed ContextBlock,
// re-checking each source for routing fields and validating the budget.
func decodeContextBlock(m map[string]any) (*ContextBlock, error) {
	block := &ContextBlock{Overflow: "truncate-oldest"}

	if ov, ok := m["overflow"].(string); ok && ov != "" {
		if ov != "truncate-oldest" && ov != "error" {
			return nil, &ContextError{Reason: fmt.Sprintf("overflow %q must be truncate-oldest or error", ov)}
		}
		block.Overflow = ov
	}

	mt, ok := toInt(m["max_tokens"])
	if !ok || mt <= 0 {
		return nil, &ContextError{Reason: "max_tokens is required and must be a positive integer"}
	}
	block.MaxTokens = mt

	rawSources, ok := m["sources"].([]any)
	if !ok || len(rawSources) == 0 {
		return nil, &ContextError{Reason: "sources is required and must list at least one {key, files} entry"}
	}
	for i, rs := range rawSources {
		sm, ok := toStringMap(rs)
		if !ok {
			return nil, &ContextError{Reason: fmt.Sprintf("sources[%d] must be a mapping", i)}
		}
		if err := rejectRoutingFields(sm); err != nil {
			return nil, err
		}
		key, _ := sm["key"].(string)
		if key == "" {
			return nil, &ContextError{Reason: fmt.Sprintf("sources[%d] is missing a key", i)}
		}
		files, err := toStringSlice(sm["files"])
		if err != nil || len(files) == 0 {
			return nil, &ContextError{Key: key, Reason: "files is required and must list at least one path"}
		}
		block.Sources = append(block.Sources, ContextSource{Key: key, Files: files})
	}
	return block, nil
}

// rejectRoutingFields fails when any reserved routing/provider key is present.
// This is the data-only enforcement the canvas requires in O3 — independent of
// the JSON Schema so the rule holds even when validation is skipped.
func rejectRoutingFields(m map[string]any) error {
	for k := range m {
		if _, banned := routingFields[strings.ToLower(strings.TrimSpace(k))]; banned {
			return &ContextError{Reason: fmt.Sprintf("field %q is forbidden — a context block is data-only and may never carry provider/model/endpoint/URL (ADR-013/ADR-035)", k)}
		}
	}
	return nil
}

// ResolveContext reads each source's files (relative to dir, confined to it),
// applies the max_tokens hard cap with the block's overflow policy, and returns
// a key→value map ready to bind into a Workflow action's {{ .ctx.* }} template.
// Strict isolation is structural: the returned map is freshly built per call
// from one scenario's own block, so it can never carry another scenario's data.
func ResolveContext(block *ContextBlock, dir string) (map[string]string, error) {
	if block == nil {
		return nil, nil
	}
	values := make(map[string]string, len(block.Sources))
	seen := make(map[string]bool, len(block.Sources))
	budget := block.MaxTokens * tokenCharsPerToken

	for _, src := range block.Sources {
		if seen[src.Key] {
			return nil, &ContextError{Key: src.Key, Reason: "duplicate context key"}
		}
		seen[src.Key] = true

		var b strings.Builder
		for _, file := range src.Files {
			abs, err := confinePath(dir, file)
			if err != nil {
				return nil, &ContextError{Key: src.Key, Reason: err.Error()}
			}
			content, err := os.ReadFile(abs) //nolint:gosec // abs is confined to dir by confinePath
			if err != nil {
				return nil, &ContextError{Key: src.Key, Reason: fmt.Sprintf("read source file %q: %v", file, err)}
			}
			b.Write(content)
		}
		values[src.Key] = b.String()
	}

	if err := enforceBudget(block, values, budget); err != nil {
		return nil, err
	}
	return values, nil
}

// enforceBudget applies the max_tokens cap to the combined content. Under the
// 'error' policy an over-budget block fails; under 'truncate-oldest' the
// earliest-declared sources are dropped/trimmed first until the budget is met,
// so the most recently declared (typically most relevant) content survives.
func enforceBudget(block *ContextBlock, values map[string]string, budget int) error {
	total := 0
	for _, v := range values {
		total += len(v)
	}
	if total <= budget {
		return nil
	}
	if block.Overflow == "error" {
		return &ContextError{Reason: fmt.Sprintf("combined context ~%d tokens exceeds max_tokens %d", total/tokenCharsPerToken, block.MaxTokens)}
	}
	// truncate-oldest: walk sources in declared order, trimming from the front
	// until the remaining content fits the budget.
	overage := total - budget
	for _, src := range block.Sources {
		if overage <= 0 {
			break
		}
		v := values[src.Key]
		if len(v) <= overage {
			overage -= len(v)
			values[src.Key] = ""
			continue
		}
		values[src.Key] = v[overage:]
		overage = 0
	}
	return nil
}

// BindContextIntoWorkflow renders every {{ .ctx.<key> }} reference in the
// Workflow manifest using the resolved context values and returns the rendered
// manifest. Binding operates on the PARSED YAML tree (each scalar string node is
// rendered individually) rather than on the raw bytes, so injected multi-line
// content (a git diff) stays valid YAML regardless of block-scalar indentation.
// Each scalar is rendered with the SAME text/template engine the engine-adapter
// uses for the {{ .ctx.* }} surface (no new template engine, A2). An unresolved
// {{ .ctx.<key> }} reference (a key the block does not supply) fails loudly
// rather than substituting an empty value.
func BindContextIntoWorkflow(manifest []byte, values map[string]string) ([]byte, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(manifest, &root); err != nil {
		return nil, &ContextError{Reason: fmt.Sprintf("parse workflow YAML: %v", err)}
	}
	if err := renderScalars(&root, values); err != nil {
		return nil, err
	}
	out, err := yaml.Marshal(&root)
	if err != nil {
		return nil, &ContextError{Reason: fmt.Sprintf("re-encode workflow YAML: %v", err)}
	}
	return out, nil
}

// renderScalars walks a YAML node tree and renders every scalar string that
// contains a template reference, replacing the node's value (and forcing a
// literal block style when the result is multi-line so the re-encoded YAML stays
// readable and valid).
func renderScalars(n *yaml.Node, values map[string]string) error {
	if n.Kind == yaml.ScalarNode && strings.Contains(n.Value, "{{") {
		rendered, err := renderScalar(n.Value, values)
		if err != nil {
			return err
		}
		n.Value = rendered
		n.Tag = "!!str"
		if strings.Contains(rendered, "\n") {
			n.Style = yaml.LiteralStyle
		} else {
			n.Style = 0
		}
		return nil
	}
	for _, child := range n.Content {
		if err := renderScalars(child, values); err != nil {
			return err
		}
	}
	return nil
}

// renderScalar renders one scalar string through text/template with the ctx
// data root. missingkey=error makes an unresolved {{ .ctx.<key> }} fail loudly.
func renderScalar(s string, values map[string]string) (string, error) {
	t, err := template.New("ctx").Option("missingkey=error").Parse(s)
	if err != nil {
		return "", &ContextError{Reason: fmt.Sprintf("parse template %q: %v", s, err)}
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, map[string]any{"ctx": values}); err != nil {
		return "", &ContextError{Reason: fmt.Sprintf("bind context into workflow: %v", err)}
	}
	return buf.String(), nil
}

// confinePath joins file to dir and rejects any path that escapes dir or is
// absolute (path traversal). Mirrors resolveMemberPath so context sources obey
// the same containment rule as scenario members.
func confinePath(dir, file string) (string, error) {
	if file == "" {
		return "", fmt.Errorf("source file is empty")
	}
	if filepath.IsAbs(file) {
		return "", fmt.Errorf("source file %q must be relative to the scenario directory", file)
	}
	cleanDir := filepath.Clean(dir)
	abs := filepath.Clean(filepath.Join(cleanDir, file))
	rel, err := filepath.Rel(cleanDir, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("source file %q escapes the scenario directory", file)
	}
	return abs, nil
}

// toInt coerces a YAML-decoded numeric value to an int.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

// toStringMap coerces a YAML-decoded mapping (string or interface keys) to a
// map[string]any.
func toStringMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			out[fmt.Sprintf("%v", k)] = val
		}
		return out, true
	default:
		return nil, false
	}
}

// toStringSlice coerces a YAML-decoded sequence of strings to []string.
func toStringSlice(v any) ([]string, error) {
	raw, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("expected a list")
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok || s == "" {
			return nil, fmt.Errorf("expected a list of non-empty strings")
		}
		out = append(out, s)
	}
	return out, nil
}
