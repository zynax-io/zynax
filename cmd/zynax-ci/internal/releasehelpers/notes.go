// SPDX-License-Identifier: Apache-2.0

package releasehelpers

import (
	"fmt"
	"strings"
)

// Version is the resolved release version and its pre-release classification,
// parity with the release.yml "Resolve version" block: a tag containing a hyphen
// (e.g. v0.6.0-rc.1) is a pre-release.
type Version struct {
	Tag        string
	Prerelease bool
}

// Resolve classifies a release tag. A hyphen in the tag marks a pre-release
// (parity with the bash `if [[ "${TAG}" == *-* ]]`).
func Resolve(tag string) (Version, error) {
	if tag == "" {
		return Version{}, fmt.Errorf("releasehelpers: notes: tag is required")
	}
	return Version{Tag: tag, Prerelease: strings.Contains(tag, "-")}, nil
}

// repoURL is the canonical download/clone URL base used throughout the notes.
const repoURL = "https://github.com/zynax-io/zynax"

// NotesInput holds the inputs to the release-notes body assembly.
type NotesInput struct {
	// Version is the release tag (e.g. v0.6.0).
	Version string
	// Services is the ordered list of service image names whose GHCR pull
	// lines are rendered in the notes (parity with the SERVICE_IMAGES env).
	Services []string
}

// Body assembles the GitHub Release notes markdown, parity with the static body
// block in the release.yml "Create GitHub Release" step. The version is
// substituted into every download URL; the GHCR pull block is rendered from the
// service list rather than hand-maintained, so it can never drift from
// SERVICE_IMAGES. The softprops generate_release_notes changelog is appended by
// the action itself — this is the prepended body only.
func Body(in NotesInput) (string, error) {
	v, err := Resolve(in.Version)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	writeHeader(&b, v.Tag)
	writeCLIInstall(&b, v.Tag)
	writeCIInstall(&b, v.Tag)
	writeServiceImages(&b, v.Tag, in.Services)
	return b.String(), nil
}

func writeHeader(b *strings.Builder, version string) {
	fmt.Fprintf(b, "## Zynax %s\n\n", version)
	b.WriteString("Declarative AI-workflow control plane — CLI, CI toolchain, and service images.\n\n")
	b.WriteString("All service images are retag-promoted from the pre-merge security ")
	b.WriteString("gate (ADR-027): the digests below are the exact binaries that ")
	b.WriteString("passed Trivy before merge.\n\n")
}

// cliTarget is one zynax CLI download row in the install section.
type cliTarget struct{ label, artifact string }

func writeCLIInstall(b *strings.Builder, version string) {
	b.WriteString("### Install zynax CLI\n\n")
	rows := []cliTarget{
		{"macOS (Apple Silicon)", "zynax_%s_darwin_arm64.tar.gz"},
		{"macOS (Intel)", "zynax_%s_darwin_amd64.tar.gz"},
		{"Linux (amd64)", "zynax_%s_linux_amd64.tar.gz"},
		{"Linux (arm64)", "zynax_%s_linux_arm64.tar.gz"},
	}
	for _, r := range rows {
		artifact := fmt.Sprintf(r.artifact, version)
		fmt.Fprintf(b, "**%s:**\n```bash\n", r.label)
		fmt.Fprintf(b, "curl -L %s/releases/download/%s/%s | tar xz && sudo mv zynax /usr/local/bin/\n```\n\n", repoURL, version, artifact)
	}
	fmt.Fprintf(b, "**Windows (amd64):** download `zynax_%s_windows_amd64.zip`\n\n", version)
	b.WriteString("**Verify checksums:**\n```bash\nsha256sum -c checksums-cli.txt\n```\n\n")
}

func writeCIInstall(b *strings.Builder, version string) {
	b.WriteString("### Install zynax-ci\n\n```bash\n# Linux (amd64)\n")
	fmt.Fprintf(b, "curl -fsSL %s/releases/download/%s/zynax-ci-linux-amd64 -o ~/bin/zynax-ci && chmod +x ~/bin/zynax-ci\n", repoURL, version)
	b.WriteString("# macOS (Apple Silicon)\n")
	fmt.Fprintf(b, "curl -fsSL %s/releases/download/%s/zynax-ci-darwin-arm64 -o ~/bin/zynax-ci && chmod +x ~/bin/zynax-ci\n```\n\n", repoURL, version)
}

func writeServiceImages(b *strings.Builder, version string, services []string) {
	b.WriteString("### Service images (GHCR)\n\n```bash\n")
	for _, svc := range services {
		fmt.Fprintf(b, "docker pull ghcr.io/zynax-io/zynax/%s:%s\n", svc, version)
	}
	b.WriteString("```\n\n")
	b.WriteString("SPDX SBOMs for all service and adapter images are attached to this ")
	b.WriteString("release. Verify image signatures with ")
	b.WriteString("`cosign verify ghcr.io/zynax-io/zynax/<service>:<version>`.\n")
}
