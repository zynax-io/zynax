// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/check"
)

var depsRoot string

var checkDepsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Verify all go.mod files agree on shared dependency versions",
	Long: `Walk the repository and check that each module in the must-match list
pins the same version across every go.mod file that declares it.

Must-match modules:
  go (toolchain directive)
  github.com/kelseyhightower/envconfig
  google.golang.org/grpc
  google.golang.org/protobuf
  gopkg.in/yaml.v3

Exits 0 when all versions agree; exits 1 on any divergence.`,
	Args: cobra.NoArgs,
	RunE: runCheckDeps,
}

func init() {
	checkDepsCmd.Flags().StringVar(&depsRoot, "root", ".", "repository root directory")
	checkCmd.AddCommand(checkDepsCmd)
}

func runCheckDeps(_ *cobra.Command, _ []string) error {
	root := depsRoot
	if root == "." {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("check deps: get working directory: %w", err)
		}
	}

	report, err := check.Deps(root)
	if err != nil {
		return err
	}

	ok := check.PrintDepsReport(os.Stdout, report)
	if !ok {
		return fmt.Errorf("go.mod version alignment check failed")
	}
	return nil
}
