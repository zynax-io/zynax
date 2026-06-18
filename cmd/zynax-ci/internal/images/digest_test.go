// SPDX-License-Identifier: Apache-2.0

package images_test

import (
	"strings"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax-ci/internal/images"
)

const (
	digA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	digC = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

// twoEntryYAML returns an images.yaml body with two entries whose digests are
// apiDigest and ciDigest respectively, plus a comment to prove it survives.
func twoEntryYAML(apiDigest, ciDigest string) string {
	return `# pinned digests (ADR-024)
images:
  - name: api-gateway
    ref: ghcr.io/example/api-gateway
    digest: ` + apiDigest + `
    consumers: []
  - name: ci-runner
    ref: ghcr.io/example/ci-runner
    digest: ` + ciDigest + `
    consumers:
      - workflow.yml
`
}

func TestUpsertUpdatesNamedEntryOnly(t *testing.T) {
	in := twoEntryYAML(digA, digB)
	out, action, err := images.Upsert(in, "ci-runner", "", digC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "updated" {
		t.Fatalf("action = %q, want updated", action)
	}
	if !strings.Contains(out, "digest: "+digC) {
		t.Errorf("ci-runner digest not updated to %s:\n%s", digC, out)
	}
	if !strings.Contains(out, "digest: "+digA) {
		t.Errorf("api-gateway digest %s should be untouched:\n%s", digA, out)
	}
	if strings.Contains(out, digB) {
		t.Errorf("old ci-runner digest %s should be gone:\n%s", digB, out)
	}
	// Comment and structure preserved.
	if !strings.HasPrefix(out, "# pinned digests (ADR-024)\n") {
		t.Errorf("leading comment not preserved:\n%s", out)
	}
}

func TestUpsertUnchangedWhenDigestMatches(t *testing.T) {
	in := twoEntryYAML(digA, digB)
	out, action, err := images.Upsert(in, "ci-runner", "", digB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "unchanged" {
		t.Fatalf("action = %q, want unchanged", action)
	}
	if out != in {
		t.Errorf("unchanged upsert mutated the text:\ngot:\n%s\nwant:\n%s", out, in)
	}
}

func TestUpsertAppendsNewEntry(t *testing.T) {
	in := twoEntryYAML(digA, digB)
	out, action, err := images.Upsert(in, "memory-service", "ghcr.io/example/memory-service", digC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "added" {
		t.Fatalf("action = %q, want added", action)
	}
	want := "\n  - name: memory-service\n    ref: ghcr.io/example/memory-service\n    digest: " + digC + "\n    consumers: []\n"
	if !strings.HasSuffix(out, want) {
		t.Errorf("new entry not appended as expected:\n%s", out)
	}
	if !strings.Contains(out, "digest: "+digA) || !strings.Contains(out, "digest: "+digB) {
		t.Errorf("existing entries lost when appending:\n%s", out)
	}
}

func TestUpsertNewEntryWithoutRefErrors(t *testing.T) {
	in := twoEntryYAML(digA, digB)
	if _, _, err := images.Upsert(in, "memory-service", "", digC); err == nil {
		t.Fatal("expected error for new entry without --ref, got nil")
	}
}

func TestDigestReValidation(t *testing.T) {
	valid := []string{digA, digB, digC}
	for _, d := range valid {
		if !images.DigestRe.MatchString(d) {
			t.Errorf("DigestRe rejected valid digest %q", d)
		}
	}
	invalid := []string{
		"",
		"sha256:tooshort",
		"sha256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", // uppercase
		"sha512:" + strings.Repeat("a", 64),
		digA + "extra",
	}
	for _, d := range invalid {
		if images.DigestRe.MatchString(d) {
			t.Errorf("DigestRe accepted invalid digest %q", d)
		}
	}
}
