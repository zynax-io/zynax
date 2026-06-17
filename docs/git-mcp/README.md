<!-- SPDX-License-Identifier: Apache-2.0 -->

# Git MCP server (`zynax mcp git`)

`zynax mcp git` launches a Git **MCP server** so an authoring loop (Claude Code,
or any MCP client) can perform Git operations — clone / branch / commit / PR /
review — as MCP tools, using a **least-privilege, injected** token.

It is a thin shim over the existing `git-adapter` (ADR-032): one Git
implementation, two surfaces. The adapter remains the runtime gRPC capability
path; MCP is an additional authoring surface. No Git logic is duplicated in the
shim.

## How it works

`zynax mcp git` execs the `git-adapter` binary in its `mcp` mode and wires the
adapter's stdio through, so it speaks the MCP stdio protocol to the MCP client.

```
MCP client  <--stdio-->  zynax mcp git  --exec-->  git-adapter mcp
                                                     (1:1 tool ↔ capability)
```

## Configuration

All configuration is injected into the process environment **at start** — never
passed as a CLI flag and never read from prompt content:

| Variable          | Required | Purpose |
|-------------------|----------|---------|
| `ADAPTER_CONFIG`  | yes      | Path to the git-adapter YAML config (declares provider, `git.auth_env`, and the per-capability `owner`/`repo`). |
| `<auth_env>`      | yes      | The token env var named by `git.auth_env` in that config (e.g. `GITHUB_TOKEN`). |
| `GIT_ADAPTER_BIN` | no       | Override the `git-adapter` binary path (default: `git-adapter`, resolved via `$PATH`). |

## Least-privilege token (required)

The token is read **once** from the environment by the git-adapter at startup.
It is **never** accepted as a tool argument, **never** read from a prompt, and
**never** written to any committed config (ADR-032).

Use a **fine-grained Personal Access Token** scoped to exactly the `owner/repo`
declared in `ADAPTER_CONFIG`, with the **minimum** permissions the authoring
task needs:

- **Repository access:** only the specific repo(s) in the config — not "all
  repositories".
- **Permissions:** `Contents: Read and write` (clone/commit/push) and
  `Pull requests: Read and write` (open/review PRs). Grant nothing else.
- Do **not** use a classic `repo`-scoped PAT — it is org-wide and over-broad.

The token lifetime is the operator's responsibility. The git-adapter reads it
once at startup and does not refresh it (see canvas EPIC G.7 for refreshable /
GitHub App credentials).

## Wiring into an MCP client

Copy [`.mcp.json.example`](../../.mcp.json.example) to `.mcp.json` and adjust the
config path. Reference the token **by env name only** — never inline a literal:

```json
{
  "mcpServers": {
    "zynax-git": {
      "command": "zynax",
      "args": ["mcp", "git"],
      "env": {
        "ADAPTER_CONFIG": "agents/adapters/git/config.yaml",
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    }
  }
}
```

Export the token in the shell that launches the MCP client (or source it from a
secret manager); `${GITHUB_TOKEN}` resolves from that environment:

```bash
export GITHUB_TOKEN="$(your-secret-fetch ...)"   # fine-grained PAT, least-privilege
```

## Run it directly

```bash
ADAPTER_CONFIG=agents/adapters/git/config.yaml \
GITHUB_TOKEN="$GITHUB_TOKEN" \
zynax mcp git
```

The server speaks the MCP stdio protocol on stdin/stdout. Press Ctrl+C to stop.

## See also

- [ADR-032 — Git MCP shim over git-adapter](../adr/ADR-032-git-mcp-shim.md)
- Canvas: `docs/spdd/1169-git-mcp-shim/canvas.md` (EPIC G)
