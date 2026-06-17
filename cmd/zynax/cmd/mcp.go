// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// gitAdapterBinEnv names the env var that overrides the git-adapter binary path.
// Default is "git-adapter" (resolved via $PATH).
const gitAdapterBinEnv = "GIT_ADAPTER_BIN"

// defaultGitAdapterBin is the binary launched when GIT_ADAPTER_BIN is unset.
const defaultGitAdapterBin = "git-adapter"

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run a Zynax MCP server for the authoring loop",
}

var mcpGitCmd = &cobra.Command{
	Use:   "git",
	Short: "Launch the Git MCP server (thin shim over git-adapter)",
	Long: `Launch the Git MCP stdio server so an authoring loop (e.g. Claude Code)
can perform Git operations — clone/branch/commit/PR/review — as MCP tools.

This is a thin launcher (ADR-032): it execs the git-adapter binary in its
"mcp" mode and wires the adapter's stdio to this process. The single Git
implementation lives in the git-adapter; no Git logic is duplicated here.

Configuration (injected at process start — never as a CLI flag, never from a
prompt):

  ADAPTER_CONFIG   path to the git-adapter YAML config (required)
  <auth_env>       the token env var named by git.auth_env in that config
                   (e.g. GITHUB_TOKEN) — a least-privilege, fine-grained PAT
  GIT_ADAPTER_BIN  override the git-adapter binary path (default: git-adapter)

The token is read once from the environment by the git-adapter at startup. It is
never accepted as a tool argument, never read from prompt content, and never
written to any committed config. Wire this command into .mcp.json (see
.mcp.json.example) and reference the token by env/secret-ref only.

Press Ctrl+C to stop.`,
	Args: cobra.NoArgs,
	RunE: func(c *cobra.Command, _ []string) error {
		return runMCPGit(c.Context(), c.OutOrStdout(), c.ErrOrStderr(), os.Stdin, os.Environ())
	},
}

// runMCPGit execs the git-adapter binary in MCP mode, wiring stdio through so it
// speaks the MCP stdio protocol to the caller. The git-adapter binary is
// resolved from $GIT_ADAPTER_BIN (default "git-adapter"); the token and config
// are inherited from the process environment (env) — never passed as arguments.
func runMCPGit(ctx context.Context, stdout, stderr io.Writer, stdin io.Reader, env []string) error {
	bin := defaultGitAdapterBin
	for _, kv := range env {
		if v, ok := envValue(kv, gitAdapterBinEnv); ok && v != "" {
			bin = v
		}
	}

	// #nosec G204 — bin is operator-controlled (GIT_ADAPTER_BIN or a fixed
	// default), never derived from prompt/tool input; the only argument is the
	// constant "mcp" subcommand.
	cmd := exec.CommandContext(ctx, bin, "mcp")
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mcp git: launch %q: %w", bin, err)
	}
	return nil
}

// envValue returns the value of key from a "KEY=VALUE" entry and whether it
// matched key. It avoids a strings.SplitN allocation on the hot path.
func envValue(entry, key string) (string, bool) {
	n := len(key)
	if len(entry) > n && entry[:n] == key && entry[n] == '=' {
		return entry[n+1:], true
	}
	return "", false
}

func init() {
	mcpCmd.AddCommand(mcpGitCmd)
	rootCmd.AddCommand(mcpCmd)
}
