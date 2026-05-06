// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags "-X ...cmd.Version=v0.3.0".
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:          "zynax-ci",
	Short:        "zynax-ci — CI and developer toolchain for Zynax",
	Version:      Version,
	SilenceUsage: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
