// SPDX-License-Identifier: Apache-2.0

package releasehelpers

import (
	"strings"
	"testing"
)

func TestSelectSourceTag(t *testing.T) {
	tests := []struct {
		name      string
		candidate []string
		existing  []string
		want      string
	}{
		{
			name:      "full sha match newest-first",
			candidate: []string{"aaaa1111", "bbbb2222"},
			existing:  []string{"latest", "main", "main-bbbb2222"},
			want:      "main-bbbb2222",
		},
		{
			name:      "first matching candidate wins",
			candidate: []string{"cccc3333", "bbbb2222"},
			existing:  []string{"main-cccc3333", "main-bbbb2222"},
			want:      "main-cccc3333",
		},
		{
			name:      "short-sha fallback",
			candidate: []string{"abcdef0123456789"},
			existing:  []string{"main-abcdef01"},
			want:      "main-abcdef01",
		},
		{
			name:      "no promoted tags",
			candidate: []string{"aaaa1111"},
			existing:  []string{"latest", "v1.0.0"},
			want:      "",
		},
		{
			name:      "no candidate match",
			candidate: []string{"zzzz9999"},
			existing:  []string{"main-aaaa1111"},
			want:      "",
		},
		{
			name:      "empty candidate skipped",
			candidate: []string{"", "aaaa1111"},
			existing:  []string{"main-aaaa1111"},
			want:      "main-aaaa1111",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SelectSourceTag(tt.candidate, tt.existing); got != tt.want {
				t.Errorf("SelectSourceTag = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPromoteRef(t *testing.T) {
	ref, ok, err := PromoteRef("ghcr.io/zynax-io/zynax", "api-gateway", "main-abc123", "v0.6.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("want ok=true for a matched source tag")
	}
	if ref.Source != "ghcr.io/zynax-io/zynax/api-gateway:main-abc123" {
		t.Errorf("Source = %q", ref.Source)
	}
	if ref.Target != "ghcr.io/zynax-io/zynax/api-gateway:v0.6.0" {
		t.Errorf("Target = %q", ref.Target)
	}
	if ref.Service != "api-gateway" {
		t.Errorf("Service = %q", ref.Service)
	}
}

func TestPromoteRef_EmptySourceExcludes(t *testing.T) {
	_, ok, err := PromoteRef("ghcr.io/zynax-io/zynax", "api-gateway", "", "v0.6.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("want ok=false (service excluded) when source tag is empty")
	}
}

func TestPromoteRef_Validation(t *testing.T) {
	cases := []struct{ prefix, service, version string }{
		{"", "svc", "v1"},
		{"p", "", "v1"},
		{"p", "svc", ""},
	}
	for _, c := range cases {
		if _, _, err := PromoteRef(c.prefix, c.service, "main-x", c.version); err == nil {
			t.Errorf("want error for prefix=%q service=%q version=%q", c.prefix, c.service, c.version)
		}
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		tag        string
		prerelease bool
	}{
		{"v0.6.0", false},
		{"v0.6.0-rc.1", true},
		{"v1.2.3-beta", true},
	}
	for _, tt := range tests {
		got, err := Resolve(tt.tag)
		if err != nil {
			t.Fatalf("Resolve(%q): unexpected error: %v", tt.tag, err)
		}
		if got.Tag != tt.tag {
			t.Errorf("Tag = %q, want %q", got.Tag, tt.tag)
		}
		if got.Prerelease != tt.prerelease {
			t.Errorf("Resolve(%q).Prerelease = %v, want %v", tt.tag, got.Prerelease, tt.prerelease)
		}
	}
}

func TestResolve_EmptyTag(t *testing.T) {
	if _, err := Resolve(""); err == nil {
		t.Error("want error for empty tag")
	}
}

func TestBody(t *testing.T) {
	body, err := Body(NotesInput{
		Version:  "v0.6.0",
		Services: []string{"api-gateway", "git-adapter"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wants := []string{
		"## Zynax v0.6.0",
		"zynax_v0.6.0_darwin_arm64.tar.gz",
		"zynax_v0.6.0_linux_amd64.tar.gz",
		"zynax-ci-linux-amd64",
		"docker pull ghcr.io/zynax-io/zynax/api-gateway:v0.6.0",
		"docker pull ghcr.io/zynax-io/zynax/git-adapter:v0.6.0",
		"cosign verify",
	}
	for _, w := range wants {
		if !strings.Contains(body, w) {
			t.Errorf("body missing %q\n--- body ---\n%s", w, body)
		}
	}
}

func TestBody_EmptyVersion(t *testing.T) {
	if _, err := Body(NotesInput{Version: ""}); err == nil {
		t.Error("want error for empty version")
	}
}
