// SPDX-License-Identifier: Apache-2.0

// Package bumprunner computes the next ci-runner digest pin and drives the
// images/images.yaml source of truth so all consumer files re-stamp. It is the
// tested Go replacement for scripts/bump-ci-runner.sh (ADR-036, M7 EPIC S step
// S.4).
//
// The legacy script sed-replaced the ci-runner digest in each workflow file
// directly. The images.yaml flow inverts that: the digest is updated once in the
// SoT (ADR-024) and images.Sync re-stamps every consumer, so a single tested
// path keeps every reference aligned. This package reuses the images internals
// rather than re-parsing YAML by hand (issue #1289 scope).
package bumprunner

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/zynax-io/zynax/cmd/zynax-ci/internal/images"
)

// RunnerImageName is the images.yaml entry whose digest this verb bumps.
const RunnerImageName = "ci-runner"

// digestRe matches a well-formed digest pin: sha256:<64 lowercase hex>.
// Parity with the bash regex `^sha256:[0-9a-f]{64}$`.
var digestRe = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// ValidateDigest reports an error when digest is not sha256:<64 lowercase hex>.
func ValidateDigest(digest string) error {
	if !digestRe.MatchString(digest) {
		return fmt.Errorf("digest must match sha256:<64 lowercase hex chars>, got: %q", digest)
	}
	return nil
}

// Result reports the outcome of a bump: the digest before/after on the SoT entry
// and the consumer files that were (or would be) re-stamped.
type Result struct {
	Image        string
	Before       string
	After        string
	YAMLChanged  bool
	ChangedFiles []string
}

// Bump validates digest, writes it onto the ci-runner entry in
// images/images.yaml, and re-stamps every consumer via images.Sync. When dryRun
// is true nothing is written; the Result still reflects what would change.
func Bump(repoRoot, digest string, dryRun bool) (Result, error) {
	if err := ValidateDigest(digest); err != nil {
		return Result{}, err
	}
	before, err := currentDigest(repoRoot)
	if err != nil {
		return Result{}, err
	}
	res := Result{Image: RunnerImageName, Before: before, After: digest, YAMLChanged: before != digest}
	if res.YAMLChanged && !dryRun {
		if err := setDigest(repoRoot, digest); err != nil {
			return res, err
		}
	}
	if err := syncConsumers(&res, repoRoot, digest, dryRun); err != nil {
		return res, err
	}
	return res, nil
}

// syncConsumers loads images.yaml (with digest applied in-memory so dry-run is
// accurate) and records which consumer files change.
func syncConsumers(res *Result, repoRoot, digest string, dryRun bool) error {
	f, err := images.Load(repoRoot)
	if err != nil {
		return err
	}
	applyDigest(&f, digest)
	results, err := images.Sync(f, repoRoot, dryRun)
	if err != nil {
		return err
	}
	for _, r := range results {
		if r.Changed {
			res.ChangedFiles = append(res.ChangedFiles, r.File)
		}
	}
	return nil
}

// applyDigest overwrites the in-memory ci-runner digest so a dry-run Sync diffs
// against the target value even before images.yaml is written.
func applyDigest(f *images.File, digest string) {
	for i := range f.Images {
		if f.Images[i].Name == RunnerImageName {
			f.Images[i].Digest = digest
		}
	}
}

// currentDigest returns the ci-runner digest recorded in images/images.yaml.
func currentDigest(repoRoot string) (string, error) {
	f, err := images.Load(repoRoot)
	if err != nil {
		return "", err
	}
	for _, e := range f.Images {
		if e.Name == RunnerImageName {
			return e.Digest, nil
		}
	}
	return "", fmt.Errorf("bump-runner: %q not found in images/images.yaml", RunnerImageName)
}

// entryDigestRe matches the digest line of the ci-runner entry only: the entry
// header followed by its ref and digest lines. The replacement preserves all
// surrounding comments and formatting (a YAML re-marshal would strip them).
var entryDigestRe = regexp.MustCompile(
	`(?m)(^  - name: ` + RunnerImageName + `\n(?:    [^\n]*\n)*?    digest: )sha256:[0-9a-f]{64}`,
)

// setDigest rewrites only the ci-runner digest line in images/images.yaml,
// leaving every other entry, comment, and consumer list untouched.
func setDigest(repoRoot, digest string) error {
	path := filepath.Join(repoRoot, "images", "images.yaml")
	data, err := os.ReadFile(path) //nolint:gosec // path is the repo SoT
	if err != nil {
		return fmt.Errorf("bump-runner: read %s: %w", path, err)
	}
	if !entryDigestRe.Match(data) {
		return fmt.Errorf("bump-runner: ci-runner digest line not found in %s", path)
	}
	out := entryDigestRe.ReplaceAll(data, []byte("${1}"+digest))
	if err := os.WriteFile(path, out, 0o600); err != nil { //nolint:gosec // path is the repo SoT (images/images.yaml under repoRoot)
		return fmt.Errorf("bump-runner: write %s: %w", path, err)
	}
	return nil
}
