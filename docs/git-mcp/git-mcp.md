<!-- SPDX-License-Identifier: Apache-2.0 -->

# Git MCP guide — least-privilege Git for authoring loops

`zynax mcp git` exposes the Zynax `git-adapter` as a **Model Context Protocol
(MCP) server**, so an authoring loop (Claude Code, or any MCP client) can perform
Git operations as MCP tools using a **least-privilege, injected** token. It is a
thin shim over the existing `git-adapter` (ADR-032): one Git implementation, two
surfaces — the adapter remains the runtime gRPC capability path, and MCP is an
additional authoring surface. No Git logic is duplicated in the shim.

> For a quick-start (copy `.mcp.json.example`, export a token, run it), see
> [`docs/git-mcp/README.md`](README.md). **This guide** covers the tool surface,
> the credential model (injection, redaction, token scope, refreshable creds),
> and a complete `.mcp.json` wiring.

---

## The `zynax mcp git` surface (#1200)

```
MCP client  <--stdio-->  zynax mcp git  --exec-->  git-adapter mcp
                                                    (1:1 tool ↔ capability)
```

`zynax mcp git` (`cmd/zynax/cmd/mcp.go`) execs the `git-adapter` binary in its
`mcp` mode and wires the adapter's stdio through, so it speaks the MCP stdio
protocol to the MCP client. The binary path resolves from `GIT_ADAPTER_BIN`
(default `git-adapter`, found via `$PATH`). It reads its configuration from the
process **environment** — never a CLI flag, never prompt content:

| Variable | Required | Purpose |
|----------|----------|---------|
| `ADAPTER_CONFIG` | yes | Path to the git-adapter YAML config (declares `git.provider`, `git.auth_env`, and each capability's static `owner`/`repo`). |
| `<auth_env>` | yes | The token env var **named by** `git.auth_env` in that config (e.g. `GITHUB_TOKEN`). |
| `GIT_ADAPTER_BIN` | no | Override the `git-adapter` binary path (default `git-adapter`). |

### Exposed MCP tools (allow-list, 1:1 with gRPC capabilities)

The MCP tool set is an **explicit allow-list** built from `capabilities[].name`
in the config — the same handlers the runtime gRPC path uses. Three tools are
exposed:

| MCP tool | Description |
|----------|-------------|
| `open_pr` | Open a pull request in the configured repository. Returns PR URL and number. |
| `request_review` | Add reviewers to an existing PR. |
| `get_diff` | Fetch the unified diff for a PR. Truncates at 4 MB and sets `truncated: true`. |

All three require `owner` and `repo` declared in config — **never** derived from
tool input (SSRF prevention, see below).

---

## Configuration

The YAML file at `ADAPTER_CONFIG` declares the provider, the token env-var name,
and each capability's pinned target. Key fields (full example:
`agents/adapters/git/agent-def.yaml.example`):

| Field | Description |
|-------|-------------|
| `git.provider` | `github` (GitLab returns "not implemented" in M5). |
| `git.auth_env` | Name of the env var holding the token (e.g. `GITHUB_TOKEN`). |
| `capabilities[].name` | One of `open_pr`, `request_review`, `get_diff`. |
| `capabilities[].owner` | Static GitHub org/user — **never** from tool input. |
| `capabilities[].repo` | Static repository name — **never** from tool input. |

---

## Credential model

### Injection — read once at start, never an argument (#1199)

The token is **injected into the process environment at start** and resolved
**once** by the git-adapter from the env var named in `git.auth_env`
(`config.ResolveToken`). It is held only in process memory: it is never a config
field, never a tool argument, never read from a prompt, and never serialized.

### Redaction — the egress backstop (#1199)

`internal/redact` is the last line of defence against a token leaking into model
context or logs. A `redact.Redactor` built from the injected token scrubs the
secret — replaced with the literal `[REDACTED]` — from:

- every caller-visible `CapabilityError` message,
- the `COMPLETED` tool-result payload,
- and again at the **MCP prompt boundary**, where tool-result text becomes model
  context.

The token value itself is never logged. (A trivially short secret is not
redacted — there is an 8-byte minimum guard against masking common substrings.)

### Token scope — least privilege (#1260)

Use a GitHub **fine-grained PAT** scoped to **only** the `owner/repo` declared in
config — never a classic PAT with the broad `repo` scope. The three capabilities
only ever touch pull requests, so the token needs just one permission:

| Capability | Required fine-grained permission |
|------------|----------------------------------|
| `open_pr` | Pull requests: **Read and write** |
| `request_review` | Pull requests: **Read and write** |
| `get_diff` | Pull requests: **Read** |

Grant **Pull requests: Read and write** on the single configured repository and
nothing else — no `Contents`, `Administration`, or org-wide access is required.

**Defense in depth:** token scope and `owner/repo` pinning are *distinct,
complementary* controls. `capabilities[].owner`/`.repo` are config-declared and
never derived from tool input (SSRF prevention), so even a token broader than
recommended cannot make the adapter reach a repository the config does not name.
Pin the config narrowly **and** scope the token narrowly — neither replaces the
other.

### Lifecycle — read once, no refresh (#1262)

The token is resolved **once at process start** and is **never refreshed** while
the adapter runs. A short-lived credential — for example a GitHub App
installation token (~1 h TTL) — would **expire mid-process** and subsequent Git
calls would then fail with no automatic re-resolution. Until refreshable
credentials land (epic G, step G.7 / #1262 — App installation tokens minted and
re-resolved before expiry), use a credential whose lifetime exceeds the process:
a fine-grained PAT (no expiry, or a long expiry you rotate manually).

---

## `.mcp.json` example

Copy [`.mcp.json.example`](../../.mcp.json.example) to `.mcp.json` and adjust the
config path. Reference the token **by env name only** — never inline a literal
(a literal would land in a committed file and reach a prompt):

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

### Run it directly

```bash
ADAPTER_CONFIG=agents/adapters/git/config.yaml \
GITHUB_TOKEN="$GITHUB_TOKEN" \
zynax mcp git
```

The server speaks the MCP stdio protocol on stdin/stdout. Press Ctrl+C to stop.

---

## See also

- Quick-start + wiring: [`docs/git-mcp/README.md`](README.md)
- Context System (the handoff contract Git operations participate in): [`docs/context/context-system.md`](../context/context-system.md)
- ADR-032 — Git MCP shim over git-adapter: [`docs/adr/ADR-032-git-mcp-shim.md`](../adr/ADR-032-git-mcp-shim.md)
- Adapter contract + credential details: `agents/adapters/git/AGENTS.md`
- Operator config example: `agents/adapters/git/agent-def.yaml.example`
