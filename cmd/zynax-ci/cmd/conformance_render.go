// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/check"
)

var renderOpts struct {
	matrix   string
	out      string
	headline bool
}

var conformanceRenderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render an emitted ZECS matrix for humans (#1775)",
	Long: `Render a zecs-matrix.json document as Markdown — the table published with a
release and shown in the e2e job summary (docs/conformance/README.md §11).

It takes an EMITTED document and only reformats it: no verdict is recomputed, no
scenario is re-run, and nothing is read from the repository. The published JSON
and the published table are therefore two views of one artifact and cannot
disagree; a release note is never hand-maintained.

  --matrix <file>   the document to render ('-' reads stdin)
  --headline        print only the one-line claim, e.g.
                    'ZECS v0.8.0 — temporal PASS (required), argo PASS (advisory)
                     · required legs: PASS · all legs: PASS'

Every leg is rendered with its enforcement, and an incomplete run says so in its
first line — a published result must never claim more authority than the run
that produced it had.`,
	Args: cobra.NoArgs,
	RunE: runConformanceRender,
}

func init() {
	f := conformanceRenderCmd.Flags()
	f.StringVar(&renderOpts.matrix, "matrix", "", "path to an emitted zecs-matrix.json ('-' for stdin)")
	f.StringVar(&renderOpts.out, "out", "-", "write the rendering here ('-' for stdout)")
	f.BoolVar(&renderOpts.headline, "headline", false, "print only the one-line release-notes claim")
	_ = conformanceRenderCmd.MarkFlagRequired("matrix")

	conformanceCmd.AddCommand(conformanceRenderCmd)
}

func runConformanceRender(cmd *cobra.Command, _ []string) error {
	in := cmd.InOrStdin()
	if renderOpts.matrix != "-" {
		f, err := os.Open(renderOpts.matrix) //nolint:gosec // caller-supplied input path
		if err != nil {
			return fmt.Errorf("conformance render: open %q: %w", renderOpts.matrix, err)
		}
		defer func() { _ = f.Close() }()
		in = f
	}
	doc, err := check.DecodeMatrix(in)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if renderOpts.out != "-" {
		f, cerr := os.Create(renderOpts.out) //nolint:gosec // caller-supplied output path
		if cerr != nil {
			return fmt.Errorf("conformance render: create %q: %w", renderOpts.out, cerr)
		}
		defer func() { _ = f.Close() }()
		out = f
	}
	if renderOpts.headline {
		if _, werr := fmt.Fprintln(out, check.Headline(doc)); werr != nil {
			return fmt.Errorf("conformance render: write headline: %w", werr)
		}
		return nil
	}
	return check.RenderMarkdown(out, doc)
}
