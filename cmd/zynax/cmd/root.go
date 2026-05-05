// SPDX-License-Identifier: Apache-2.0

// Package cmd implements the zynax CLI commands.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax/client"
)

var rootCmd = &cobra.Command{
	Use:          "zynax",
	Short:        "Zynax CLI — apply, manage, and monitor Zynax workflows",
	SilenceUsage: true,
}

var (
	apiURL   string
	insecure bool
)

// Execute runs the root command and exits on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	defaultURL := os.Getenv("ZYNAX_API_URL")
	if defaultURL == "" {
		defaultURL = "http://localhost:8080"
	}
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", defaultURL, "api-gateway base URL ($ZYNAX_API_URL)")
	rootCmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "skip TLS certificate verification")
}

func newGateway() *client.Gateway {
	return client.New(apiURL, insecure)
}
