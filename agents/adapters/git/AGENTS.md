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
| `GITHUB_TOKEN` | ✓ | GitHub PAT or App token. The name of the env var is declared via `git.auth_env` in the config YAML; the token value is read at startup from that env var. |

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

## gRPC Port

Default: **50060** (set via `endpoint` in config YAML).

## MCP shim surface (ADR-032)

`git-adapter mcp` runs a thin Model Context Protocol stdio server over the **same**
capability handlers (one Git implementation, two surfaces — ADR-032). It binds no
port and needs no registry. MCP tools map 1:1 onto the capabilities above; the
exposed tool set is an explicit allow-list built from `capabilities[].name` — no
Git logic is reimplemented in `internal/mcp/`. The owner/repo target stays pinned
in config, so no caller-supplied owner/repo/remote reaches a Git call (SSRF guard).
Credential injection + redaction are delivered by G.3 (#1199), not this layer.

## Testing

```bash
GOWORK=off go test ./... -race -timeout 60s   # ADR-017: GOWORK=off required
```

## Reference

Canvas: `docs/spdd/381-git-adapter/canvas.md` · Operator example: `agent-def.yaml.example`
