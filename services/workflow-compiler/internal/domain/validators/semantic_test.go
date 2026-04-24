package validators_test

import (
	"testing"

	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain/validators"
)

// CapabilityRefValidator ──────────────────────────────────────────────────────

func TestCapabilityRefValidator(t *testing.T) {
	cases := []struct {
		name       string
		capability string
		wantErr    bool
	}{
		{"simple", "summarize", false},
		{"snake_case", "send_email", false},
		{"multi_word", "fetch_and_parse", false},
		{"with_digits", "run_step_2", false},
		{"single_char", "a", false},
		{"empty", "", true},
		{"uppercase", "Summarize", true},
		{"leading_underscore", "_summarize", true},
		{"trailing_underscore", "summarize_", true},
		{"double_underscore", "send__email", true},
		{"hyphen", "send-email", true},
		{"reserved_zynax", "zynax_exec", true},
		{"reserved_system", "system_log", true},
		{"reserved_internal", "internal_ping", true},
	}

	v := validators.CapabilityRefValidator{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := graphWith("s", map[string]*domain.State{
				"s": {
					ID:   "s",
					Type: domain.StateTypeTerminal,
					Actions: []domain.Action{
						{Capability: tc.capability},
					},
				},
			})
			errs := v.Validate(g)
			if tc.wantErr && len(errs) == 0 {
				t.Errorf("capability %q: expected error, got none", tc.capability)
			}
			if !tc.wantErr && len(errs) != 0 {
				t.Errorf("capability %q: unexpected error: %v", tc.capability, errs)
			}
		})
	}
}

func TestCapabilityRefValidator_NoActions(t *testing.T) {
	g := graphWith("s", map[string]*domain.State{
		"s": {ID: "s", Type: domain.StateTypeTerminal},
	})
	if errs := (validators.CapabilityRefValidator{}).Validate(g); len(errs) != 0 {
		t.Errorf("expected no errors for state with no actions, got %v", errs)
	}
}

// EventNameValidator ──────────────────────────────────────────────────────────

func TestEventNameValidator(t *testing.T) {
	cases := []struct {
		name      string
		eventType string
		wantErr   bool
	}{
		{"simple", "push", false},
		{"dot_separated", "review.approved", false},
		{"multi_dot", "pr.merge.conflict", false},
		{"empty", "", true},
		{"uppercase", "Review.Approved", true},
		{"underscore", "review_approved", true},
		{"leading_dot", ".approved", true},
		{"trailing_dot", "review.", true},
		{"double_dot", "review..approved", true},
		{"spaces", "review approved", true},
		{"digit_start", "1push", true},
	}

	v := validators.EventNameValidator{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := graphWith("a", map[string]*domain.State{
				"a": normalState("a", tr(tc.eventType, "b")),
				"b": terminalState(),
			})
			errs := v.Validate(g)
			if tc.wantErr && len(errs) == 0 {
				t.Errorf("event_type %q: expected error, got none", tc.eventType)
			}
			if !tc.wantErr && len(errs) != 0 {
				t.Errorf("event_type %q: unexpected error: %v", tc.eventType, errs)
			}
		})
	}
}

func TestEventNameValidator_NoTransitions(t *testing.T) {
	g := graphWith("s", map[string]*domain.State{
		"s": {ID: "s", Type: domain.StateTypeTerminal},
	})
	if errs := (validators.EventNameValidator{}).Validate(g); len(errs) != 0 {
		t.Errorf("expected no errors for state with no transitions, got %v", errs)
	}
}

// NamespaceValidator ──────────────────────────────────────────────────────────

func TestNamespaceValidator(t *testing.T) {
	cases := []struct {
		name      string
		namespace string
		wantErr   bool
	}{
		{"default", "default", false},
		{"single_char", "a", false},
		{"hyphenated", "team-alpha", false},
		{"alphanumeric", "platform2", false},
		{"max_length", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false}, // 63 chars
		{"empty", "", false}, // empty is allowed (caught by manifest parser)
		{"uppercase", "Team", true},
		{"leading_hyphen", "-alpha", true},
		{"trailing_hyphen", "alpha-", true},
		{"underscore", "team_alpha", true},
		{"too_long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true}, // 64 chars
		{"spaces", "my namespace", true},
	}

	v := validators.NamespaceValidator{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := graphWith("s", map[string]*domain.State{
				"s": terminalState(),
			})
			g.Namespace = tc.namespace
			errs := v.Validate(g)
			if tc.wantErr && len(errs) == 0 {
				t.Errorf("namespace %q: expected error, got none", tc.namespace)
			}
			if !tc.wantErr && len(errs) != 0 {
				t.Errorf("namespace %q: unexpected error: %v", tc.namespace, errs)
			}
		})
	}
}

// DuplicateTransitionValidator ────────────────────────────────────────────────

func TestDuplicateTransitionValidator(t *testing.T) {
	cases := []struct {
		name    string
		g       *domain.WorkflowGraph
		wantErr bool
	}{
		{
			name: "unique events",
			g: graphWith("a", map[string]*domain.State{
				"a": normalState("a", tr("push", "b"), tr("merge", "b")),
				"b": terminalState(),
			}),
		},
		{
			name: "duplicate unguarded",
			g: graphWith("a", map[string]*domain.State{
				"a": normalState("a", tr("push", "b"), tr("push", "b")),
				"b": terminalState(),
			}),
			wantErr: true,
		},
		{
			name: "guarded duplicate allowed",
			g: graphWith("a", map[string]*domain.State{
				"a": {
					ID:   "a",
					Type: domain.StateTypeNormal,
					Transitions: []domain.Transition{
						{EventType: "push", TargetState: "b", Guard: "{{ .ctx.fast }}"},
						{EventType: "push", TargetState: "c", Guard: "{{ .ctx.slow }}"},
					},
				},
				"b": terminalState(),
				"c": terminalState(),
			}),
		},
		{
			name: "mixed: one guarded, one unguarded same event",
			g: graphWith("a", map[string]*domain.State{
				"a": {
					ID:   "a",
					Type: domain.StateTypeNormal,
					Transitions: []domain.Transition{
						{EventType: "push", TargetState: "b", Guard: "{{ .ctx.ok }}"},
						{EventType: "push", TargetState: "b"}, // unguarded
					},
				},
				"b": terminalState(),
			}),
			// only unguarded ones are checked; one unguarded = no dup
		},
	}

	v := validators.DuplicateTransitionValidator{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := v.Validate(tc.g)
			if tc.wantErr && len(errs) == 0 {
				t.Error("expected error, got none")
			}
			if !tc.wantErr && len(errs) != 0 {
				t.Errorf("expected no error, got %v", errs)
			}
		})
	}
}
