// SPDX-License-Identifier: Apache-2.0

package images

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
			after := replaceDigest(entry.Name, before, refPat, entry.Digest)
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

// replaceDigest replaces the sha256 digest for imageName with target.
// Priority order:
//  1. Banner region — if the file contains "# BEGIN zynax-ci:images:<name>" markers,
//     only the content between those markers is updated. This allows files with multiple
//     pinned digests (e.g. both golang-alpine and distroless-static) to be synced
//     independently without corrupting each other.
//  2. Ref pattern — replaces <ref>[:<tag>]@sha256:<old> with <ref>[:<tag>]@sha256:<new>.
//  3. Bare sha256 fallback — replaces all sha256:… occurrences (used for plain-text
//     files like config/ci-runner-digest.txt that contain only a raw digest).
func replaceDigest(imageName, content string, refPat *regexp.Regexp, target string) string {
	if result, ok := replaceInBannerRegion(imageName, content, target); ok {
		return result
	}
	if refPat.MatchString(content) {
		return refPat.ReplaceAllStringFunc(content, func(m string) string {
			return sha256Re.ReplaceAllString(m, target)
		})
	}
	return sha256Re.ReplaceAllString(content, target)
}

// replaceInBannerRegion replaces sha256 digests within a banner-delimited region.
// Returns (result, true) when a matching banner is found; (content, false) otherwise.
// Only content between the BEGIN and END marker lines is modified.
func replaceInBannerRegion(imageName, content, target string) (string, bool) {
	begin := "# BEGIN zynax-ci:images:" + imageName
	end := "# END zynax-ci:images:" + imageName

	beginIdx := strings.Index(content, begin)
	if beginIdx < 0 {
		return content, false
	}

	// Advance past the BEGIN line (to the start of the region body).
	beginLineEnd := strings.Index(content[beginIdx:], "\n")
	if beginLineEnd < 0 {
		return content, false
	}
	regionStart := beginIdx + beginLineEnd + 1

	endIdx := strings.Index(content[regionStart:], end)
	if endIdx < 0 {
		return content, false
	}

	region := content[regionStart : regionStart+endIdx]
	newRegion := sha256Re.ReplaceAllString(region, target)
	return content[:regionStart] + newRegion + content[regionStart+endIdx:], true
}
