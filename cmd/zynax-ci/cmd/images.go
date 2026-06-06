// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/internal/images"
)

var imagesRoot string

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Manage pinned container image digests (images/images.yaml)",
}

var imagesSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Stamp all consumer files with digests from images/images.yaml",
	Long: `Read images/images.yaml and update every listed consumer file so its
pinned digest matches the canonical value. Pass --dry-run to preview changes
without writing any files.`,
	Args: cobra.NoArgs,
	RunE: runImagesSync,
}

var imagesCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify all consumer files match images/images.yaml digests",
	Long: `Read images/images.yaml and verify that every listed consumer file
contains the canonical digest. Exits 1 if any file diverges.`,
	Args: cobra.NoArgs,
	RunE: runImagesCheck,
}

var imagesDryRun bool

func init() {
	imagesSyncCmd.Flags().StringVar(&imagesRoot, "root", ".", "repository root directory")
	imagesSyncCmd.Flags().BoolVar(&imagesDryRun, "dry-run", false, "print diff only; do not write files")
	imagesCheckCmd.Flags().StringVar(&imagesRoot, "root", ".", "repository root directory")
	imagesCmd.AddCommand(imagesSyncCmd)
	imagesCmd.AddCommand(imagesCheckCmd)
	rootCmd.AddCommand(imagesCmd)
}

func resolveRoot(root string) (string, error) {
	if root != "." {
		return root, nil
	}
	return os.Getwd()
}

func runImagesSync(_ *cobra.Command, _ []string) error {
	root, err := resolveRoot(imagesRoot)
	if err != nil {
		return fmt.Errorf("images sync: %w", err)
	}
	f, err := images.Load(root)
	if err != nil {
		return err
	}
	results, err := images.Sync(f, root, imagesDryRun)
	if err != nil {
		return err
	}
	anyChanged := false
	for _, r := range results {
		if r.Changed {
			anyChanged = true
			if imagesDryRun {
				fmt.Printf("── would update %s (image: %s)\n", r.File, r.Image)
				printDiff(r.Before, r.After)
			} else {
				fmt.Printf("✅  updated %s (image: %s)\n", r.File, r.Image)
			}
		}
	}
	if !anyChanged {
		fmt.Println("✅  All consumer files already match images/images.yaml.")
	}
	return nil
}

func runImagesCheck(_ *cobra.Command, _ []string) error {
	root, err := resolveRoot(imagesRoot)
	if err != nil {
		return fmt.Errorf("images check: %w", err)
	}
	f, err := images.Load(root)
	if err != nil {
		return err
	}
	report, err := images.Check(f, root)
	if err != nil {
		return err
	}
	if images.PrintCheckReport(os.Stdout, report) {
		return nil
	}
	return fmt.Errorf("image digest drift detected — run 'make sync-images' then commit")
}

// printDiff prints a simple before/after line diff (lines that changed).
func printDiff(before, after string) {
	bLines := strings.Split(before, "\n")
	aLines := strings.Split(after, "\n")
	for i := 0; i < len(bLines) && i < len(aLines); i++ {
		if bLines[i] != aLines[i] {
			fmt.Printf("  - %s\n", bLines[i])
			fmt.Printf("  + %s\n", aLines[i])
		}
	}
}
