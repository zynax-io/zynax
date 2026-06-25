// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// lastRunFile is the basename of the state file that records the most recent
// run id. It lives under <config-dir>/zynax/ so that bare `zynax logs` /
// `zynax result` (no id) can default to the user's latest run.
const lastRunFile = "last-run"

// zynaxConfigDir returns the directory that holds the CLI's local state. It
// honors ZYNAX_CONFIG_DIR (used by tests to point at a t.TempDir()) and
// otherwise falls back to <os.UserConfigDir()>/zynax. CLI-side only — no
// server or chart involvement.
func zynaxConfigDir() (string, error) {
	if dir := os.Getenv("ZYNAX_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, "zynax"), nil
}

// saveLastRun records runID as the most recent run so a later bare logs/result
// can target it. A failure to persist is non-fatal to the caller: recording the
// id is a convenience, never a reason to fail an otherwise-successful apply.
func saveLastRun(runID string) error {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil
	}
	dir, err := zynaxConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir %s: %w", dir, err)
	}
	path := filepath.Join(dir, lastRunFile)
	//nolint:gosec // G703: path is a fixed basename joined to the confined CLI config dir, not user-tainted
	if err := os.WriteFile(path, []byte(runID+"\n"), 0o600); err != nil {
		return fmt.Errorf("write last-run file %s: %w", path, err)
	}
	return nil
}

// loadLastRun returns the most recently recorded run id, or an empty string
// (no error) when none has been recorded yet. A missing state file is the
// expected "no prior run" case, not a failure.
func loadLastRun() (string, error) {
	dir, err := zynaxConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, lastRunFile)
	b, err := os.ReadFile(path) //nolint:gosec // path is confined to the CLI config dir
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read last-run file %s: %w", path, err)
	}
	return strings.TrimSpace(string(b)), nil
}

// resolveRunID picks the run id to act on: an explicit positional arg always
// wins; otherwise it falls back to the last recorded run. When neither is
// available it returns a clear, actionable error telling the user to start a
// run or pass an id. Used by `zynax logs` and `zynax result` so a bare
// invocation targets the user's most recent run (#1491, canvas O21).
func resolveRunID(args []string) (string, error) {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		return args[0], nil
	}
	last, err := loadLastRun()
	if err != nil {
		return "", err
	}
	if last == "" {
		return "", fmt.Errorf(
			"no run id given and no prior run recorded: start a run with " +
				"`zynax apply <file>` (or `zynax workflow run <name>`), or pass a run id explicitly")
	}
	return last, nil
}
