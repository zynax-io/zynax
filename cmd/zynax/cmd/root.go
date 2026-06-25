// SPDX-License-Identifier: Apache-2.0

// Package cmd implements the zynax CLI commands.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax/client"
)

// Version is set at build time via -ldflags "-X ...cmd.Version=v0.3.0".
var Version = "dev"

const workflowRunIDUse = "workflow <run-id>"

// beginnerGroupID groups the small noun-first command set surfaced first in
// `zynax --help` for new users (canvas O20). Advanced verbs (apply, get, status,
// validate, …) stay available but fall under cobra's default "Additional
// Commands" heading. `doctor` (issue #1489) joins this group when it lands.
const beginnerGroupID = "beginner"

var rootCmd = &cobra.Command{
	Use:          "zynax",
	Short:        "Zynax CLI — apply, manage, and monitor Zynax workflows",
	Version:      Version,
	SilenceUsage: true,
}

var (
	apiURL   string
	apiKey   string
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
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", os.Getenv("ZYNAX_API_KEY"), "api-gateway bearer token sent as Authorization header ($ZYNAX_API_KEY)")
	rootCmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "skip TLS certificate verification")
	rootCmd.AddGroup(&cobra.Group{ID: beginnerGroupID, Title: "Getting started:"})
}

func newGateway() *client.Gateway {
	return client.New(apiURL, insecure, apiKey)
}
