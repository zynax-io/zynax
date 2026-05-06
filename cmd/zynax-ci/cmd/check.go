// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Advisory checks for the Zynax repository",
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
