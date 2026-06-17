// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// fakeProbe returns a fixed header (or error) and never touches the network or
// any token — the scope policy is pure given the probe result.
type fakeProbe struct {
	hdr http.Header
	err error
}

func (f fakeProbe) Probe(_ context.Context) (http.Header, error) { return f.hdr, f.err }

func hdr(scopes string, set bool) http.Header {
	h := http.Header{}
	if set {
		h.Set("X-OAuth-Scopes", scopes)
	}
	return h
}

func TestParseMode(t *testing.T) {
	cases := []struct {
		in   string
		want Mode
	}{
		{"warn", ModeWarn},
		{"WARN", ModeWarn},
		{"  warn ", ModeWarn}, // surrounding whitespace is trimmed
		{"enforce", ModeEnforce},
		{"", ModeEnforce},
		{"bogus", ModeEnforce},
	}
	for _, c := range cases {
		if got := ParseMode(c.in); got != c.want {
			t.Errorf("ParseMode(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestModeString(t *testing.T) {
	const (
		warn    = "warn"
		enforce = "enforce"
	)
	if ModeWarn.String() != warn {
		t.Errorf("ModeWarn.String() = %q", ModeWarn.String())
	}
	if ModeEnforce.String() != enforce {
		t.Errorf("ModeEnforce.String() = %q", ModeEnforce.String())
	}
}

func TestInspect(t *testing.T) {
	tests := []struct {
		name       string
		probe      fakeProbe
		wantClass  string
		wantScopes []string
		wantOver   []string
		wantErr    bool
	}{
		{
			name:      "fine-grained token has no scope header",
			probe:     fakeProbe{hdr: hdr("", false)},
			wantClass: "fine-grained-or-app",
		},
		{
			name:       "classic over-broad repo scope",
			probe:      fakeProbe{hdr: hdr("repo, read:org", true)},
			wantClass:  "classic",
			wantScopes: []string{"read:org", "repo"},
			wantOver:   []string{"repo"},
		},
		{
			name:       "classic narrow scope passes",
			probe:      fakeProbe{hdr: hdr("read:user, public_repo", true)},
			wantClass:  "classic",
			wantScopes: []string{"public_repo", "read:user"},
			wantOver:   []string{},
		},
		{
			name:       "classic empty scope set present",
			probe:      fakeProbe{hdr: hdr("", true)},
			wantClass:  "classic",
			wantScopes: []string{},
			wantOver:   []string{},
		},
		{
			name:       "multiple over-broad scopes sorted+deduped",
			probe:      fakeProbe{hdr: hdr("admin:org, repo, repo, delete_repo", true)},
			wantClass:  "classic",
			wantScopes: []string{"admin:org", "delete_repo", "repo"},
			wantOver:   []string{"admin:org", "delete_repo", "repo"},
		},
		{
			name:    "probe error propagates",
			probe:   fakeProbe{err: errors.New("boom")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Inspect(context.Background(), tt.probe)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.TokenClass != tt.wantClass {
				t.Errorf("TokenClass = %q, want %q", res.TokenClass, tt.wantClass)
			}
			if tt.wantScopes != nil && !reflect.DeepEqual(res.Scopes, tt.wantScopes) {
				t.Errorf("Scopes = %v, want %v", res.Scopes, tt.wantScopes)
			}
			if tt.wantOver != nil && !reflect.DeepEqual(res.OverBroad, tt.wantOver) {
				t.Errorf("OverBroad = %v, want %v", res.OverBroad, tt.wantOver)
			}
		})
	}
}

func TestValidate_EnforceRejectsOverBroad(t *testing.T) {
	p := fakeProbe{hdr: hdr("repo", true)}
	res, err := Validate(context.Background(), p, ModeEnforce)
	if !errors.Is(err, ErrOverBroadScope) {
		t.Fatalf("expected ErrOverBroadScope, got %v", err)
	}
	// Result is still returned (safe to log) and lists the offending scope.
	if !reflect.DeepEqual(res.OverBroad, []string{"repo"}) {
		t.Errorf("OverBroad = %v, want [repo]", res.OverBroad)
	}
	// The error message must carry scope metadata, never a token value.
	if !strings.Contains(err.Error(), "repo") {
		t.Errorf("error should name the over-broad scope: %v", err)
	}
}

func TestValidate_WarnAllowsOverBroad(t *testing.T) {
	p := fakeProbe{hdr: hdr("repo", true)}
	res, err := Validate(context.Background(), p, ModeWarn)
	if err != nil {
		t.Fatalf("warn mode must not error: %v", err)
	}
	if len(res.OverBroad) != 1 {
		t.Errorf("warn mode should still report over-broad scopes: %v", res.OverBroad)
	}
}

func TestValidate_FineGrainedPasses(t *testing.T) {
	p := fakeProbe{hdr: hdr("", false)}
	res, err := Validate(context.Background(), p, ModeEnforce)
	if err != nil {
		t.Fatalf("fine-grained token must pass: %v", err)
	}
	if res.TokenClass != "fine-grained-or-app" {
		t.Errorf("TokenClass = %q", res.TokenClass)
	}
	if len(res.OverBroad) != 0 {
		t.Errorf("fine-grained token has no over-broad scopes: %v", res.OverBroad)
	}
}

func TestValidate_ProbeErrorPropagates(t *testing.T) {
	p := fakeProbe{err: errors.New("net down")}
	if _, err := Validate(context.Background(), p, ModeEnforce); err == nil {
		t.Fatal("expected probe error to propagate")
	}
}

func TestScopeHeaderAbsentVsEmpty(t *testing.T) {
	// Nil header → absent.
	if _, ok := scopeHeader(nil); ok {
		t.Error("nil header should report absent")
	}
	// Present but empty → present.
	h := http.Header{}
	h.Set("X-OAuth-Scopes", "")
	if v, ok := scopeHeader(h); !ok || v != "" {
		t.Errorf("present-empty header: got (%q,%v)", v, ok)
	}
}

func TestParseScopes(t *testing.T) {
	got := parseScopes(" repo ,, read:org, repo ")
	want := []string{"read:org", "repo"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseScopes = %v, want %v", got, want)
	}
	if len(parseScopes("")) != 0 {
		t.Error("empty input should yield no scopes")
	}
}
