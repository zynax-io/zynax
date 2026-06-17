// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// cmdWithIO builds a command with buffered stdin/stdout/stderr for assertions.
func cmdWithIO(t *testing.T, stdin string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	c := &cobra.Command{}
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&bytes.Buffer{})
	c.SetIn(strings.NewReader(stdin))
	c.SetContext(context.Background())
	return c, &out
}

func writeReport(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunBenchGate_NoRegression_Stdin(t *testing.T) {
	benchReportPath, benchThreshold, benchEnforce = "", 0, false
	t.Setenv("BENCH_REPORT", "")
	t.Setenv("THRESHOLD_PCT", "")
	t.Setenv("BENCH_GATE_ENFORCE", "")
	c, out := cmdWithIO(t, "Compile-8  +5.00% (p=0.00)\n")
	if err := runBenchGate(c, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "✅ No benchmark regressed beyond 20%.") {
		t.Fatalf("want pass summary, got:\n%s", out.String())
	}
}

func TestRunBenchGate_Regression_FailOpen_File(t *testing.T) {
	benchReportPath = writeReport(t, "Compile-8  +24.10% (p=0.00)\n")
	benchThreshold, benchEnforce = 0, false
	t.Setenv("BENCH_REPORT", "")
	t.Setenv("THRESHOLD_PCT", "")
	t.Setenv("BENCH_GATE_ENFORCE", "")
	defer func() { benchReportPath = "" }()
	c, out := cmdWithIO(t, "")
	if err := runBenchGate(c, nil); err != nil {
		t.Fatalf("fail-open must not error: %v", err)
	}
	if !strings.Contains(out.String(), "fail-open mode") {
		t.Fatalf("want fail-open note, got:\n%s", out.String())
	}
}

func TestRunBenchGate_Regression_Enforce_Errors(t *testing.T) {
	benchReportPath, benchThreshold, benchEnforce = "", 0, true
	t.Setenv("BENCH_REPORT", "")
	t.Setenv("THRESHOLD_PCT", "")
	t.Setenv("BENCH_GATE_ENFORCE", "")
	defer func() { benchEnforce = false }()
	c, _ := cmdWithIO(t, "Compile-8  +24.10% (p=0.00)\n")
	if err := runBenchGate(c, nil); err == nil {
		t.Fatal("enforce must return an error on regression")
	}
}

func TestRunBenchGate_ThresholdFromEnv(t *testing.T) {
	benchReportPath, benchThreshold, benchEnforce = "", 0, false
	t.Setenv("BENCH_REPORT", "")
	t.Setenv("THRESHOLD_PCT", "10")
	t.Setenv("BENCH_GATE_ENFORCE", "")
	c, out := cmdWithIO(t, "Compile-8  +15.00% (p=0.00)\n")
	if err := runBenchGate(c, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "REGRESSION") || !strings.Contains(out.String(), "10%") {
		t.Fatalf("env threshold not applied, got:\n%s", out.String())
	}
}

func TestRunBenchGate_OpenError(t *testing.T) {
	benchReportPath = filepath.Join(t.TempDir(), "missing.txt")
	benchThreshold, benchEnforce = 0, false
	t.Setenv("BENCH_REPORT", "")
	defer func() { benchReportPath = "" }()
	c, _ := cmdWithIO(t, "")
	if err := runBenchGate(c, nil); err == nil {
		t.Fatal("want error opening a missing report file")
	}
}

func TestRunBDDSelect_FailOpen_BadRef(t *testing.T) {
	bddBase, bddHead = "", ""
	t.Setenv("BASE", "zzz-no-such-ref")
	t.Setenv("HEAD", "also-missing")
	c, out := cmdWithIO(t, "")
	// A bogus diff range makes git fail → fail-open prints ALL, exits nil.
	if err := runBDDSelect(c, nil); err != nil {
		t.Fatalf("fail-open must not error: %v", err)
	}
	if strings.TrimSpace(out.String()) != "ALL" {
		t.Fatalf("want ALL on diff failure, got: %q", out.String())
	}
}

func TestRunBDDSelect_EmptyRange(t *testing.T) {
	// HEAD..HEAD is a valid, empty diff → no proto changes → empty selection.
	bddBase, bddHead = "HEAD", "HEAD"
	t.Setenv("BASE", "")
	t.Setenv("HEAD", "")
	defer func() { bddBase, bddHead = "", "" }()
	c, out := cmdWithIO(t, "")
	if err := runBDDSelect(c, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Fatalf("want empty selection for HEAD..HEAD, got: %q", out.String())
	}
}
