# ADR-032: Git MCP as a thin shim over the git-adapter

**Status:** Proposed  **Date:** 2026-06-15
**Related:** ADR-013 (adapter-first, no mandatory SDK), ADR-010 (pluggable agent runtime)

---

## Context

Zynax already ships a Go **git-adapter** — a runtime gRPC capability provider for clone/branch/commit/
PR/review. The M7 brief requires **Git MCP integration** so Claude Code and agent-authoring loops can
use Git tools. We could build a second, independent Git implementation behind MCP, or expose MCP as a
surface over the existing adapter. Duplicating Git logic would create drift and double the security
surface for credential handling — a costly long-term coupling decision.

## Decision

1. The MCP Git surface is a **thin shim over the existing git-adapter** — one Git implementation, two
   surfaces (runtime gRPC capability + MCP tools). No Git logic is reimplemented in the MCP layer.
2. Credentials (`GITHUB_TOKEN`) are **injected at process start** via env/secret-ref. A secret is
   **never serialized into a prompt, log, or trace** — redaction is mandatory.
3. Tokens follow **least-privilege** — scoped per session/repo.
4. The git-adapter remains the runtime workflow path; MCP is for the authoring loop.

## Rationale

| Option | Assessment |
|--------|------------|
| MCP shim over git-adapter (chosen) | ✅ Single implementation; one credential/security surface; no drift |
| Independent MCP Git implementation | ✗ Rejected — duplicate logic + duplicate credential surface; drift risk |
| MCP everywhere, deprecate adapter | ✗ Deferred — runtime workflows depend on the capability adapter today |

## Consequences

- **Positive:** authoring agents get Git tools without a second codebase; one place to audit credential
  handling; consistent behaviour across surfaces.
- **Negative / trade-off:** the shim couples the MCP server to the adapter's interface — adapter changes
  must keep the MCP mapping in sync.
- **Neutral / follow-up:** a future consolidation (MCP as the primary path) remains possible but is out
  of scope; Tier-2 security review is mandatory for this EPIC.
