# agents/adapters/git — Git Adapter

Go adapter service implementing `AgentService` for GitHub repository operations.

## Module

`github.com/zynax-io/zynax/agents/adapters/git`

## Capabilities

| Name | Description |
|------|-------------|
| `open_pr` | Open a pull request in the configured repository. Returns PR URL and number. |
| `request_review` | Add reviewers to an existing PR. Emits `PROGRESS` per poll cycle; `COMPLETED` on confirmation. |
| `get_diff` | Fetch the unified diff for a PR. Truncates at 4 MB and sets `truncated: true`. |

All three capabilities require `owner` and `repo` declared in config — never derived from `input_payload` (SSRF prevention).

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `ADAPTER_CONFIG` | ✓ | Path to the YAML config file (see `agent-def.yaml.example`). |
| `GITHUB_TOKEN` | ✓ | GitHub token. The name of the env var is declared via `git.auth_env` in the config YAML; the token value is read at startup from that env var. Use a least-privilege **fine-grained PAT** (see [Credential — token scope and lifecycle](#credential--token-scope-and-lifecycle)), not a classic `repo`-scoped PAT. |

## Configuration

YAML file at the path given by `ADAPTER_CONFIG`. Key fields:

| Field | Description |
|-------|-------------|
| `agent_id` | Unique identifier registered with agent-registry. |
| `endpoint` | gRPC bind address (default `:50060`). |
| `registry_endpoint` | agent-registry address (e.g. `agent-registry:50052`). |
| `git.provider` | `github` (GitLab returns `INTERNAL` "not implemented" in M5). |
| `git.auth_env` | Name of the env var holding the token (e.g. `GITHUB_TOKEN`). |
| `capabilities[].owner` | Static GitHub org or user — never from `input_payload`. |
| `capabilities[].repo` | Static repository name — never from `input_payload`. |

## Credential — token scope and lifecycle

### Recommended token: fine-grained PAT (least privilege)

Use a GitHub **fine-grained personal access token** scoped to **only** the
`owner/repo` declared in config — never a classic PAT with the broad `repo`
scope (full read/write control across every repository the token owner can
reach). The three capabilities only ever touch pull requests, so the token needs
just one repository permission:

| Capability | Required fine-grained permission |
|------------|----------------------------------|
| `open_pr` | Pull requests: **Read and write** |
| `request_review` | Pull requests: **Read and write** |
| `get_diff` | Pull requests: **Read** (Read and write also works) |

Grant **Pull requests: Read and write** on the single configured repository and
nothing else. No `Contents`, `Administration`, or org-wide access is required.

### Lifecycle: read once at startup, no refresh

The token is resolved **once at process start** from the env var named in
`git.auth_env` (`config.ResolveToken`) and is **never refreshed** while the
adapter runs. A short-lived credential — for example a GitHub App installation
token (~1 h TTL) — will **expire mid-process** and subsequent Git calls will
fail with no automatic re-resolution. Until refreshable credentials land
(epic G, step G.7 / #1262 — App installation tokens minted and re-resolved
before expiry), use a credential whose lifetime exceeds the process: a
fine-grained PAT (no expiry, or a long expiry you rotate manually).

### Defense-in-depth: static `owner/repo` pinning

Token scope and `owner/repo` pinning are **distinct, complementary** controls.
`capabilities[].owner` / `.repo` are declared in config and never derived from
`input_payload` (SSRF prevention), so even a token broader than recommended
cannot make the adapter reach a repository the config does not name. Pin the
config narrowly **and** scope the token narrowly — neither replaces the other.

## gRPC Port

Default: **50060** (set via `endpoint` in config YAML).

## MCP shim surface (ADR-032)

`git-adapter mcp` runs a thin Model Context Protocol stdio server over the **same**
capability handlers (one Git implementation, two surfaces — ADR-032). It binds no
port and needs no registry. MCP tools map 1:1 onto the capabilities above; the
exposed tool set is an explicit allow-list built from `capabilities[].name` — no
Git logic is reimplemented in `internal/mcp/`. The owner/repo target stays pinned
in config, so no caller-supplied owner/repo/remote reaches a Git call (SSRF guard).

### Credential injection + redaction (G.3 / #1199)

The token is **injected at process start** — resolved once from the env var named
in `git.auth_env` (see `config.ResolveToken`), held only in process memory, never
a config field, never a tool argument, never serialized. `internal/redact` is the
egress backstop: a `redact.Redactor` built from the injected token scrubs the
secret (replaced with `[REDACTED]`) from every caller-visible CapabilityError
message and COMPLETED payload, and again at the MCP prompt boundary where the
tool-result text becomes model context. The token value itself is never logged.

## Testing

```bash
GOWORK=off go test ./... -race -timeout 60s   # ADR-017: GOWORK=off required
```

## Reference

Canvas: `docs/spdd/381-git-adapter/canvas.md` · Operator example: `agent-def.yaml.example`
