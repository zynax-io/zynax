// SPDX-License-Identifier: Apache-2.0

package check

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Human renderings of an emitted ZECS result document (#1775).
//
// Every rendering here is a pure function of a MatrixDoc — the exact document
// published as zecs-matrix.json — and none of them recomputes a verdict. That is
// the property publication rests on: the table an adopter reads in a release and
// the JSON a script parses are two views of ONE artifact, so they cannot drift,
// and no release note is hand-maintained.
//
// Two things survive rendering intact, because they are what a published result
// is most likely to overstate:
//   - every leg is named with its enforcement, since a PASS on an advisory leg is
//     not a merge-blocking guarantee (gap G6, #1778), and the required legs
//     aggregate separately;
//   - `complete: false` is stated in the first line, so a run that skipped a leg
//     can never read as a complete conformance result.
const (
	noLegsClaim = "no engine legs recorded"
	noRequired  = "none declared"

	incompleteWarning = "> **Incomplete run — not a conformance result for the suite.** A leg, or a " +
		"scenario a leg was meant to run, produced no observation. This document reports what happened."

	skippedNotInLeg    = "SKIPPED — not in leg"
	skippedNotExecuted = "SKIPPED — not executed"
	absentCell         = "—"
)

// DecodeMatrix reads a document emitted by `zynax-ci conformance matrix`.
//
// A schema_version this build does not render is a hard error, not a
// best-effort read: a release must fail loudly rather than publish a table that
// silently drops meaning the document carried. Render such a document with the
// zynax-ci built from the revision that produced it.
func DecodeMatrix(r io.Reader) (MatrixDoc, error) {
	var doc MatrixDoc
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return MatrixDoc{}, fmt.Errorf("conformance render: decode matrix: %w", err)
	}
	if doc.SchemaVersion != matrixSchemaVersion {
		return MatrixDoc{}, fmt.Errorf(
			"conformance render: matrix schema_version %q, this build renders %q — render it with the zynax-ci built from the revision that emitted it",
			doc.SchemaVersion, matrixSchemaVersion)
	}
	return doc, nil
}

// Headline is the one-line conformance claim for release notes: the suite, its
// version, every leg with its result AND its enforcement, and both aggregates.
//
// It reads correctly whichever way the open enforcement decision (#1778) goes:
// the enforcement word and the `required legs` aggregate come from the document,
// so promoting a leg to required changes this line with no template edit.
func Headline(doc MatrixDoc) string {
	claim := noLegsClaim
	if cells := legClaims(doc); len(cells) > 0 {
		claim = strings.Join(cells, ", ")
	}
	line := fmt.Sprintf("%s — %s · required legs: %s · all legs: %s",
		suiteTitle(doc), claim, requiredVerdict(doc), doc.Result)
	if !doc.Complete {
		line += " · INCOMPLETE run — not a conformance result"
	}
	return line
}

// suiteTitle is the suite's short name and version as one label ("ZECS v0.8.0").
func suiteTitle(doc MatrixDoc) string {
	return strings.TrimSpace(doc.ShortName + " " + doc.Version)
}

func legClaims(doc MatrixDoc) []string {
	cells := make([]string, 0, len(doc.Legs))
	for _, leg := range doc.Legs {
		cells = append(cells, fmt.Sprintf("%s %s (%s)", leg.Engine, leg.Result, leg.Enforcement))
	}
	return cells
}

// requiredVerdict never renders the "no required legs" sentinel as a result: a
// suite whose every leg is advisory has no merge-blocking verdict to report.
func requiredVerdict(doc MatrixDoc) string {
	if doc.EnforcedResult == "" || doc.EnforcedResult == ResultNone {
		return noRequired
	}
	return doc.EnforcedResult
}

// RenderMarkdown writes the release-notes section for a matrix document: the
// headline claim, the per-scenario table, provenance, and the document's own
// notes. Generated on every publication — never edited by hand.
func RenderMarkdown(w io.Writer, doc MatrixDoc) error {
	var b strings.Builder
	fmt.Fprintf(&b, "### Engine conformance — %s\n\n%s\n\n", suiteTitle(doc), Headline(doc))
	if !doc.Complete {
		b.WriteString(incompleteWarning + "\n\n")
	}
	writeMatrixTable(&b, doc)
	if p := provenance(doc); p != "" {
		fmt.Fprintf(&b, "\n%s\n", p)
	}
	for _, note := range doc.Notes {
		fmt.Fprintf(&b, "\n- %s", collapse(note))
	}
	if len(doc.Notes) > 0 {
		b.WriteString("\n")
	}
	if doc.Definition != "" {
		fmt.Fprintf(&b,
			"\nWhat each leg actually asserts (§5) and the suite's known gaps (§7): `%s`.\n", doc.Definition)
	}
	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("conformance render: write: %w", err)
	}
	return nil
}

// writeMatrixTable renders one column per leg, headed by that leg's enforcement,
// so a reader cannot see a PASS without seeing what it is worth.
func writeMatrixTable(b *strings.Builder, doc MatrixDoc) {
	if len(doc.Legs) == 0 {
		b.WriteString(noLegsClaim + ".\n")
		return
	}
	header := make([]string, 0, len(doc.Legs)+1)
	header = append(header, "Scenario")
	divider := []string{"---"}
	for _, leg := range doc.Legs {
		header = append(header, fmt.Sprintf("%s (%s)", leg.Engine, leg.Enforcement))
		divider = append(divider, "---")
	}
	writeRow(b, header)
	writeRow(b, divider)
	for _, sc := range doc.Legs[0].Scenarios {
		row := make([]string, 0, len(doc.Legs)+1)
		row = append(row, "`"+sc.ID+"`")
		for _, leg := range doc.Legs {
			row = append(row, cellText(leg, sc.ID))
		}
		writeRow(b, row)
	}
}

func writeRow(b *strings.Builder, cells []string) {
	fmt.Fprintf(b, "| %s |\n", strings.Join(cells, " | "))
}

// cellText renders one cell. A skip says WHICH skip it is: `not_in_leg` carries
// the manifest's gap id and leaves the leg complete, while `not_executed` is the
// one that made the leg INCOMPLETE.
func cellText(leg LegMatrix, id string) string {
	for _, sc := range leg.Scenarios {
		if sc.ID != id {
			continue
		}
		if sc.Result != ResultSkipped {
			return sc.Result
		}
		switch sc.SkipKind {
		case SkipNotInLeg:
			if sc.Gap != "" {
				return fmt.Sprintf("%s (%s)", skippedNotInLeg, sc.Gap)
			}
			return skippedNotInLeg
		case SkipNotExecuted:
			return skippedNotExecuted
		default:
			return ResultSkipped
		}
	}
	return absentCell
}

// provenance states what was measured and where, so a published table can be
// traced back to the run that produced it.
func provenance(doc MatrixDoc) string {
	var parts []string
	if doc.Revision != "" {
		parts = append(parts, fmt.Sprintf("Revision `%s`", doc.Revision))
	}
	if doc.RunURL != "" {
		parts = append(parts, fmt.Sprintf("[run](%s)", doc.RunURL))
	}
	if doc.GeneratedAt != "" {
		parts = append(parts, fmt.Sprintf("generated %s", doc.GeneratedAt))
	}
	return strings.Join(parts, " · ")
}
