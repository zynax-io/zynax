// SPDX-License-Identifier: Apache-2.0

package cmd

import "github.com/spf13/cobra"

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate CI and documentation artifacts",
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
