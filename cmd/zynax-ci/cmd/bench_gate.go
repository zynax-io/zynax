// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/benchgate"
)

var (
	benchReportPath string
	benchThreshold  float64
	benchEnforce    bool
)

var benchGateCmd = &cobra.Command{
	Use:   "bench-gate",
	Short: "Gate on benchmark regression from a benchstat report",
	Long: `Parse a benchstat report (baseline → new) and fail/pass on the regression
threshold. A positive "vs base" delta on a benchmark row whose magnitude exceeds
the threshold percent is a regression.

Replaces the parsing/gating half of tools/ci/bench-regression.sh (ADR-036). The
benchmarks themselves are still run by the workflow / Makefile; this verb reads
the benchstat output it produces and applies the decision.

Fail-open by default (EPIC R safeguard): a regression WARNS but exits 0. Set
--enforce (or BENCH_GATE_ENFORCE=true) to fail the build on a regression.

Inputs (flags override env, env overrides defaults):
  --report     $BENCH_REPORT   (default: stdin)
  --threshold  $THRESHOLD_PCT  (default: 20)
  --enforce    $BENCH_GATE_ENFORCE=true`,
	Args: cobra.NoArgs,
	RunE: runBenchGate,
}

func init() {
	benchGateCmd.Flags().StringVar(&benchReportPath, "report", "", "benchstat report file ($BENCH_REPORT; default stdin)")
	benchGateCmd.Flags().Float64Var(&benchThreshold, "threshold", 0, "regression threshold percent ($THRESHOLD_PCT; default 20)")
	benchGateCmd.Flags().BoolVar(&benchEnforce, "enforce", false, "fail the build on regression ($BENCH_GATE_ENFORCE=true)")
	rootCmd.AddCommand(benchGateCmd)
}

func runBenchGate(cmd *cobra.Command, _ []string) error {
	r, closeFn, err := benchReport(benchReportPath, cmd)
	if err != nil {
		return err
	}
	defer closeFn()

	res, err := benchgate.Evaluate(r, benchThresholdPct())
	if err != nil {
		return err
	}
	enforce := benchEnforce || os.Getenv("BENCH_GATE_ENFORCE") == "true"
	if _, err := fmt.Fprint(cmd.OutOrStdout(), benchgate.Summary(res, enforce)); err != nil {
		return fmt.Errorf("bench-gate: write summary: %w", err)
	}
	if res.Regressed() && enforce {
		return fmt.Errorf("bench-gate: %d benchmark(s) regressed beyond %g%%", len(res.Regressions), res.Threshold)
	}
	return nil
}

// benchReport opens the report file, or returns stdin when none is given.
func benchReport(p string, cmd *cobra.Command) (io.Reader, func(), error) {
	path := pick(p, os.Getenv("BENCH_REPORT"))
	if path == "" {
		return cmd.InOrStdin(), func() {}, nil
	}
	f, err := os.Open(path) //nolint:gosec // report path is CI-controlled (BENCH_REPORT env / flag)
	if err != nil {
		return nil, nil, fmt.Errorf("bench-gate: open report: %w", err)
	}
	return f, func() { _ = f.Close() }, nil
}

// benchThresholdPct resolves the threshold from flag, env, then default.
func benchThresholdPct() float64 {
	if benchThreshold > 0 {
		return benchThreshold
	}
	if v := os.Getenv("THRESHOLD_PCT"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return benchgate.DefaultThresholdPct
}
