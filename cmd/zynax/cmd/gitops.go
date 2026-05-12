// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax/gitops"
)

var gitopsCmd = &cobra.Command{
	Use:   "gitops",
	Short: "GitOps sub-commands",
}

var gitopsWatchCmd = &cobra.Command{
	Use:   "watch <dir>",
	Short: "Watch a directory for YAML changes and re-apply them",
	Long: `Watch <dir> recursively for YAML manifest changes.

On each change zynax computes a SHA-256 of the file content and calls
zynax apply only when the content has changed since the last run.
State is persisted to <dir>/.zynax-watch.state so restarts are idempotent.

Press Ctrl+C to stop (exits 0).`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		dir := args[0]
		info, err := os.Stat(dir)
		if err != nil {
			return fmt.Errorf("watch: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("watch: %q is not a directory", dir)
		}

		gw := newGateway()
		applyFn := func(ctx context.Context, _ string, content []byte) (string, error) {
			runID, _, _, err := gw.Apply(ctx, content, "")
			if err != nil {
				return "", err
			}
			return runID, nil
		}

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		slog.Info("gitops: starting", "dir", dir, "api_url", apiURL)
		w := gitops.New(dir, applyFn)
		if err := w.Run(ctx); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	gitopsCmd.AddCommand(gitopsWatchCmd)
	rootCmd.AddCommand(gitopsCmd)
}
