// SPDX-License-Identifier: Apache-2.0

package images

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var sha256Re = regexp.MustCompile(`sha256:[a-f0-9]{64}`)

// SyncResult records the outcome of syncing one consumer file.
type SyncResult struct {
	File    string
	Image   string
	Changed bool
	Before  string
	After   string
}

// Sync updates all consumer files so their pinned digests match the values in f.
// When dryRun is true, files are not written; results still reflect what would change.
func Sync(f File, repoRoot string, dryRun bool) ([]SyncResult, error) {
	var results []SyncResult
	for _, entry := range f.Images {
		refPat := buildRefPattern(entry.Ref)
		for _, rel := range entry.Consumers {
			full := filepath.Join(repoRoot, rel)
			data, err := os.ReadFile(full) //nolint:gosec
			if err != nil {
				return results, fmt.Errorf("images sync: read %s: %w", rel, err)
			}
			before := string(data)
			after := replaceDigest(before, refPat, entry.Digest)
			r := SyncResult{File: rel, Image: entry.Name, Before: before, After: after, Changed: before != after}
			results = append(results, r)
			if r.Changed && !dryRun {
				if err := os.WriteFile(full, []byte(after), 0o600); err != nil { //nolint:gosec
					return results, fmt.Errorf("images sync: write %s: %w", rel, err)
				}
			}
		}
	}
	return results, nil
}

// buildRefPattern matches <ref>[optional :suffix]@sha256:[a-f0-9]{64}.
func buildRefPattern(ref string) *regexp.Regexp {
	return regexp.MustCompile(regexp.QuoteMeta(ref) + `(?::[^\s@"]*)?@sha256:[a-f0-9]{64}`)
}

// replaceDigest replaces the sha256 portion matched by refPat with target.
// Falls back to bare sha256 replacement when refPat finds no match
// (e.g. config/ci-runner-digest.txt which contains only the raw digest).
func replaceDigest(content string, refPat *regexp.Regexp, target string) string {
	if refPat.MatchString(content) {
		return refPat.ReplaceAllStringFunc(content, func(m string) string {
			return sha256Re.ReplaceAllString(m, target)
		})
	}
	return sha256Re.ReplaceAllString(content, target)
}
