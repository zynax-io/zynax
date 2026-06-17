// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/internal/imagesreport"
)

var (
	metaLabel   string
	metaDigest  string
	metaExtra   string
	metaIndex   string
	metaSummary string

	cleanupTag     string
	cleanupVerJSON string
	cleanupKeep    int
	cleanupPrune   bool

	retagRef    string
	retagSHA    string
	retagGitRef string
)

var imagesMetaCmd = &cobra.Command{
	Use:   "meta",
	Short: "Report OCI image metadata + size budget from an index manifest",
	Long: `Read a GHCR index manifest (JSON, from stdin or --index) and report the
OCI annotations, attestation count, and platform rows. Fails when the index
resolved but carries no description annotation; warns on a missing title.

Replaces .github/actions/report-image-meta (ADR-036). The workflow still fetches
the manifest with curl (the external primitive); this verb makes the
annotation/budget decisions testable. Writes the markdown report to
$GITHUB_STEP_SUMMARY (or --summary) and echoes it.`,
	Args: cobra.NoArgs,
	RunE: runImagesMeta,
}

var imagesCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Select GHCR package version ids to delete (PR-image or prune)",
	Long: `Read a GitHub packages-API versions list (JSON, from stdin or --versions)
and print the version ids to delete, one per line, for the shell DELETE loop.

Default mode selects versions tagged exactly --tag (the pr-image-cleanup
selection). With --prune it selects stale per-commit (main-<sha>) builds that
carry no latest/main/v* tag, newest-first, keeping --keep (the tools-image
prune). Replaces the gh/jq selection blocks (ADR-036); gh stays the API client.`,
	Args: cobra.NoArgs,
	RunE: runImagesCleanup,
}

var imagesRetagCmd = &cobra.Command{
	Use:   "retag",
	Short: "Compute the release tags to apply when promoting a manifest",
	Long: `Print the fully-qualified tags to apply when promoting a multi-arch
manifest: always <ref>:main-<sha> and <ref>:latest, plus <ref>:<version> when
--git-ref is a refs/tags/v* ref. One tag per line for the crane/imagetools loop.

Replaces the FINAL_TAGS bash in tools-image.yml (ADR-036); crane/imagetools
stays the copy primitive. zynax-ci only computes which tags to apply.`,
	Args: cobra.NoArgs,
	RunE: runImagesRetag,
}

func init() {
	imagesMetaCmd.Flags().StringVar(&metaLabel, "label", "", "human-readable image label for the summary")
	imagesMetaCmd.Flags().StringVar(&metaDigest, "digest", "", "index digest (sha256:...)")
	imagesMetaCmd.Flags().StringVar(&metaExtra, "extra", "", "extra markdown appended after the digest line")
	imagesMetaCmd.Flags().StringVar(&metaIndex, "index", "", "index manifest JSON file (default stdin)")
	imagesMetaCmd.Flags().StringVar(&metaSummary, "summary", "", "summary output file ($GITHUB_STEP_SUMMARY)")

	imagesCleanupCmd.Flags().StringVar(&cleanupTag, "tag", "", "exact tag to select (pr-<sha>)")
	imagesCleanupCmd.Flags().StringVar(&cleanupVerJSON, "versions", "", "versions-list JSON file (default stdin)")
	imagesCleanupCmd.Flags().BoolVar(&cleanupPrune, "prune", false, "prune mode: select stale main-<sha> builds")
	imagesCleanupCmd.Flags().IntVar(&cleanupKeep, "keep", 0, "prune mode: number of newest builds to keep")

	imagesRetagCmd.Flags().StringVar(&retagRef, "ref", "", "image repository without tag (ghcr.io/...)")
	imagesRetagCmd.Flags().StringVar(&retagSHA, "sha", "", "commit sha being published (github.sha)")
	imagesRetagCmd.Flags().StringVar(&retagGitRef, "git-ref", "", "triggering git ref (github.ref)")

	imagesCmd.AddCommand(imagesMetaCmd)
	imagesCmd.AddCommand(imagesCleanupCmd)
	imagesCmd.AddCommand(imagesRetagCmd)
}

func runImagesMeta(cmd *cobra.Command, _ []string) error {
	r, closeFn, err := openOrStdin(metaIndex, cmd)
	if err != nil {
		return err
	}
	defer closeFn()
	meta, err := imagesreport.ParseMeta(r)
	if err != nil {
		return err
	}
	if err := emitMetaSummary(cmd, meta); err != nil {
		return err
	}
	return metaDiagnostics(cmd, meta)
}

// emitMetaSummary renders and writes the step-summary markdown.
func emitMetaSummary(cmd *cobra.Command, meta imagesreport.Meta) error {
	md := meta.Summary(metaLabel, metaDigest, metaExtra)
	dest := pick(metaSummary, os.Getenv("GITHUB_STEP_SUMMARY"))
	if dest != "" {
		if err := appendFile(dest, md); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(cmd.OutOrStdout(), md); err != nil {
		return fmt.Errorf("images meta: write stdout: %w", err)
	}
	return nil
}

// metaDiagnostics emits warnings/notices and fails on a missing description.
func metaDiagnostics(cmd *cobra.Command, meta imagesreport.Meta) error {
	if meta.MissingDescription() {
		return fmt.Errorf("images meta: %s@%s is missing the org.opencontainers.image.description annotation", metaLabel, metaDigest)
	}
	errOut := cmd.ErrOrStderr()
	if meta.IndexResolved && meta.Title == "" {
		_, _ = fmt.Fprintf(errOut, "::warning::%s@%s is missing the org.opencontainers.image.title annotation\n", metaLabel, metaDigest)
	}
	if meta.IndexResolved && meta.Attestations != imagesreport.ExpectedAttestations {
		_, _ = fmt.Fprintf(errOut, "::notice::%s attestation manifests: %d (expected %d, ADR-025)\n", metaLabel, meta.Attestations, imagesreport.ExpectedAttestations)
	}
	return nil
}

func runImagesCleanup(cmd *cobra.Command, _ []string) error {
	r, closeFn, err := openOrStdin(cleanupVerJSON, cmd)
	if err != nil {
		return err
	}
	defer closeFn()
	ids, err := selectCleanupIDs(r)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprint(cmd.OutOrStdout(), imagesreport.FormatIDs(ids)); err != nil {
		return fmt.Errorf("images cleanup: write ids: %w", err)
	}
	return nil
}

// selectCleanupIDs dispatches to prune or exact-tag selection.
func selectCleanupIDs(r io.Reader) ([]int64, error) {
	if cleanupPrune {
		return imagesreport.SelectPrunable(r, cleanupKeep)
	}
	if cleanupTag == "" {
		return nil, fmt.Errorf("images cleanup: --tag is required unless --prune is set")
	}
	return imagesreport.SelectByTag(r, cleanupTag)
}

func runImagesRetag(cmd *cobra.Command, _ []string) error {
	tags, err := imagesreport.FinalTags(imagesreport.RetagInput{Ref: retagRef, SHA: retagSHA, GitRef: retagGitRef})
	if err != nil {
		return err
	}
	for _, t := range tags {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), t); err != nil {
			return fmt.Errorf("images retag: write tag: %w", err)
		}
	}
	return nil
}

// openOrStdin opens path, or returns the command's stdin when path is empty.
func openOrStdin(path string, cmd *cobra.Command) (io.Reader, func(), error) {
	if path == "" {
		return cmd.InOrStdin(), func() {}, nil
	}
	f, err := os.Open(path) //nolint:gosec // path is CI-controlled (flag)
	if err != nil {
		return nil, nil, fmt.Errorf("images: open %s: %w", path, err)
	}
	return f, func() { _ = f.Close() }, nil
}

// appendFile appends s to the file at path (created if absent).
func appendFile(path, s string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // CI-controlled
	if err != nil {
		return fmt.Errorf("images meta: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(s); err != nil {
		return fmt.Errorf("images meta: append %s: %w", path, err)
	}
	return nil
}
