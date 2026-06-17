// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/internal/releasehelpers"
)

var (
	matrixPrefix   string
	matrixService  string
	matrixVersion  string
	matrixExisting string
	matrixCandSHAs string

	notesVersion  string
	notesServices string
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Release-pipeline decision helpers (release.yml)",
	Long: `Compute the release.yml decisions that feed the cosign/crane signing
steps: which promoted main-<sha> image a release tag promotes (matrix) and the
GitHub Release notes body (notes).

Replaces the gh-api/jq + assembly run: blocks (ADR-036). git/gh/docker/cosign/
crane stay the external primitives — this group only makes their decisions
testable; no signing or attestation crypto moves into Go (ADR-025 unchanged).`,
}

var releaseMatrixCmd = &cobra.Command{
	Use:   "matrix",
	Short: "Resolve the source→target image refs a release tag promotes",
	Long: `Pick the promoted source tag a release promotes for one service and
print "<src>:<source-image> <tgt>:<target-image>" for the crane/cosign loop.

Walk --candidate-shas (first-parent commit SHAs from git rev-list, newest-first)
and select the first main-<sha> (or main-<sha[:8]>) present in --existing-tags
(the GHCR tag list from gh api). On no match the service is excluded: nothing is
printed and the command succeeds (parity with the retag-version src='' guard).`,
	Args: cobra.NoArgs,
	RunE: runReleaseMatrix,
}

var releaseNotesCmd = &cobra.Command{
	Use:   "notes",
	Short: "Assemble the GitHub Release notes body for a version",
	Long: `Render the release-notes markdown (install + GHCR pull blocks) for
--version and print it to stdout for the softprops release body. The service
pull block is rendered from --services so it cannot drift from SERVICE_IMAGES.

Replaces the static body block in release.yml (ADR-036); the softprops action
still appends its generated changelog. Use $GITHUB_OUTPUT redirection in the
workflow to capture the body.`,
	Args: cobra.NoArgs,
	RunE: runReleaseNotes,
}

func init() {
	releaseMatrixCmd.Flags().StringVar(&matrixPrefix, "prefix", "ghcr.io/zynax-io/zynax", "image repository prefix")
	releaseMatrixCmd.Flags().StringVar(&matrixService, "service", "", "service image name (e.g. api-gateway)")
	releaseMatrixCmd.Flags().StringVar(&matrixVersion, "version", "", "release version tag (e.g. v0.6.0)")
	releaseMatrixCmd.Flags().StringVar(&matrixExisting, "existing-tags", "", "newline/space-separated GHCR tag list (gh api)")
	releaseMatrixCmd.Flags().StringVar(&matrixCandSHAs, "candidate-shas", "", "newline/space-separated first-parent SHAs, newest-first")

	releaseNotesCmd.Flags().StringVar(&notesVersion, "version", "", "release version tag (e.g. v0.6.0)")
	releaseNotesCmd.Flags().StringVar(&notesServices, "services", "", "space/newline-separated service image names")

	releaseCmd.AddCommand(releaseMatrixCmd)
	releaseCmd.AddCommand(releaseNotesCmd)
	rootCmd.AddCommand(releaseCmd)
}

func runReleaseMatrix(cmd *cobra.Command, _ []string) error {
	src := releasehelpers.SelectSourceTag(fields(matrixCandSHAs), fields(matrixExisting))
	ref, ok, err := releasehelpers.PromoteRef(matrixPrefix, matrixService, src, matrixVersion)
	if err != nil {
		return err
	}
	if !ok {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "::notice::%s: no promoted main-<sha> source — excluded from %s\n", matrixService, matrixVersion)
		return nil
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "src=%s\ntgt=%s\n", ref.Source, ref.Target); err != nil {
		return fmt.Errorf("release matrix: write refs: %w", err)
	}
	return nil
}

func runReleaseNotes(cmd *cobra.Command, _ []string) error {
	body, err := releasehelpers.Body(releasehelpers.NotesInput{
		Version:  notesVersion,
		Services: fields(notesServices),
	})
	if err != nil {
		return err
	}
	if _, err := fmt.Fprint(cmd.OutOrStdout(), body); err != nil {
		return fmt.Errorf("release notes: write body: %w", err)
	}
	return nil
}

// fields splits a space/newline-separated flag value into non-empty tokens,
// preserving order (the candidate-sha walk and SERVICE_IMAGES are order-sensitive).
func fields(s string) []string {
	return strings.Fields(s)
}
