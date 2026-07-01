// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/check"
)

var protobufRuntimeRoot string

var checkProtobufRuntimeCmd = &cobra.Command{
	Use:   "protobuf-runtime",
	Short: "Verify each Python uv.lock pins a protobuf runtime >= the gencode version",
	Long: `Read the protobuf gencode version from a generated Python stub header and
verify that every agents/**/uv.lock pins a protobuf runtime at least that
version.

A locked runtime older than gencode makes the generated stubs raise at import
via runtime_version.ValidateProtobufRuntimeVersion — the langgraph e2e crash the
#1550 re-lock fixed. This guard catches the drift before merge (e.g. a stub
regeneration that bumps gencode without a matching re-lock).

Exits 0 when every lockfile is at or above gencode; exits 1 on any that is below.`,
	Args: cobra.NoArgs,
	RunE: runCheckProtobufRuntime,
}

func init() {
	checkProtobufRuntimeCmd.Flags().StringVar(&protobufRuntimeRoot, "root", ".", "repository root directory")
	checkCmd.AddCommand(checkProtobufRuntimeCmd)
}

func runCheckProtobufRuntime(_ *cobra.Command, _ []string) error {
	root := protobufRuntimeRoot
	if root == "." {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("check protobuf-runtime: get working directory: %w", err)
		}
	}

	report, err := check.ProtobufRuntime(root)
	if err != nil {
		return err
	}
	if !check.PrintProtobufRuntimeReport(os.Stdout, report) {
		return fmt.Errorf("protobuf runtime/gencode alignment check failed")
	}
	return nil
}
