// SPDX-License-Identifier: Apache-2.0

// Package check implements advisory checks for the zynax-ci toolchain.
package check

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileReport is the result for a single AI context file.
type FileReport struct {
	Path      string
	Lines     int
	Threshold int
	Warn      bool
}

// ContextReport is the full report from an AI context check.
type ContextReport struct {
	Files      []FileReport
	Total      int
	TotalLimit int
	Warnings   int
}

// thresholds mirrors the logic in tools/count-ai-context.sh.
var (
	thresholdClaude     = 200
	thresholdRootAgents = 310
	thresholdAISetup    = 150
	thresholdDirAgents  = 150
	thresholdTotal      = 2200
)

// Context scans repoRoot for CLAUDE.md, AGENTS.md files, and
// docs/ai-assistant-setup.md, counting lines and comparing against
// per-file thresholds. Always returns without error even when thresholds
// are exceeded — this check is advisory only.
func Context(repoRoot string) (ContextReport, error) {
	var report ContextReport
	report.TotalLimit = thresholdTotal

	add := func(path string, threshold int) {
		n, err := countLines(path)
		if err != nil {
			return // file absent — skip silently
		}
		warn := n > threshold
		if warn {
			report.Warnings++
		}
		report.Files = append(report.Files, FileReport{
			Path:      relPath(repoRoot, path),
			Lines:     n,
			Threshold: threshold,
			Warn:      warn,
		})
		report.Total += n
	}

	add(filepath.Join(repoRoot, "CLAUDE.md"), thresholdClaude)
	add(filepath.Join(repoRoot, "AGENTS.md"), thresholdRootAgents)
	add(filepath.Join(repoRoot, "docs", "ai-assistant-setup.md"), thresholdAISetup)

	// Per-directory AGENTS.md files (all except root).
	var subdirAgents []string
	_ = filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() != "AGENTS.md" {
			return nil
		}
		if path == filepath.Join(repoRoot, "AGENTS.md") {
			return nil
		}
		subdirAgents = append(subdirAgents, path)
		return nil
	})
	sort.Strings(subdirAgents)
	for _, path := range subdirAgents {
		add(path, thresholdDirAgents)
	}

	if report.Total > thresholdTotal {
		report.Warnings++
	}
	return report, nil
}

// Print writes the report in the same table format as count-ai-context.sh.
func (r ContextReport) Print(w *os.File) {
	_, _ = fmt.Fprintf(w, "| %-55s | %5s | %5s | %-6s |\n", "File", "Lines", "Limit", "Status")
	_, _ = fmt.Fprintln(w, "|----------------------------------------------------------|-------|-------|--------|")
	for _, f := range r.Files {
		status := "OK"
		if f.Warn {
			status = "WARN"
		}
		_, _ = fmt.Fprintf(w, "| %-55s | %5d | %5d | %-6s |\n", f.Path, f.Lines, f.Threshold, status)
	}
	_, _ = fmt.Fprintln(w, "|----------------------------------------------------------|-------|-------|--------|")
	totalStatus := "OK"
	if r.Total > r.TotalLimit {
		totalStatus = "WARN"
	}
	_, _ = fmt.Fprintf(w, "| %-55s | %5d | %5d | %-6s |\n", "TOTAL", r.Total, r.TotalLimit, totalStatus)
	_, _ = fmt.Fprintln(w, "")
	if r.Warnings > 0 {
		_, _ = fmt.Fprintf(w, "WARNING: %d file(s) exceed their line threshold.\n", r.Warnings)
		_, _ = fmt.Fprintln(w, "Consider trimming AI context files — smaller budgets improve signal density.")
	} else {
		_, _ = fmt.Fprintln(w, "All AI context files are within budget.")
	}
}

func countLines(path string) (int, error) {
	f, err := os.Open(path) //nolint:gosec // path is from filepath.WalkDir on the repo root
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	s := bufio.NewScanner(f)
	n := 0
	for s.Scan() {
		n++
	}
	return n, s.Err()
}

func relPath(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return strings.TrimPrefix(rel, "./")
}
