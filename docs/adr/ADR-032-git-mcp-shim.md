<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-032 — Git MCP as a thin shim over the git-adapter

| Field | Value |
|-------|-------|
| **Status** | Accepted |
| **Date** | 2026-06-15 |
| **Deciders** | Oscar Gómez Manresa |
| **Scope** | `agents/adapters/git/` (existing runtime adapter + new `mcp` surface), `cmd/zynax/` (`zynax mcp git`), `docs/git-mcp/`, `.mcp.json.example` — M7 EPIC G (#1169) |
| **Related** | ADR-013 (adapter-first, no mandatory SDK), ADR-010 (pluggable agent runtime), ADR-007 (Pydantic Settings for config), ADR-018 (AI knowledge base authorization model), ADR-019 (SPDD prompt governance) |

---

## Context

Zynax already ships a **git-adapter** — a runtime gRPC capability provider that
performs clone / branch / commit / PR / review on behalf of workflow steps. It
is the single, audited Git implementation in the platform and is dispatched by
the task broker like any other capability (ADR-013).

EPIC G (#1169) requires a **Git surface for the authoring loop**: Claude Code
and agent-authoring sessions need Git tools exposed over the **Model Context
Protocol (MCP)** so an authoring agent can clone a repo, open a branch, commit,
and raise a PR while it is being driven interactively.

There are two ways to provide that surface:

1. Build a **second, independent Git implementation** behind an MCP server, or
2. Expose MCP as **another surface over the existing git-adapter**.

This is effectively a one-way door for the project's Git story. A second
implementation would mean two codepaths that clone, push, and authenticate to
remotes — and therefore **two credential-handling surfaces to secure and audit**.
Git behaviour would drift between the runtime path and the authoring path, and
every security fix would have to be applied twice. Because the security model
(how a token enters the process and how it is kept out of model context) is the
load-bearing concern of this EPIC, it must be settled **before** any
implementation lands — which is the purpose of this ADR (canvas step G.1).

## Decision

**1. One Git implementation, two surfaces.** The MCP Git server is a **thin
protocol shim over the existing git-adapter** — it translates MCP tool calls into
the adapter's existing handlers and translates results back. The git-adapter
remains the runtime workflow path; MCP is an additional surface for the authoring
loop. **No Git logic is reimplemented in the MCP layer.** MCP tools map 1:1 onto
adapter capabilities so there is a single place to audit clone/push/auth.

**2. Credentials are injected at process start.** The Git token is supplied to
the server process via **environment variable or secret-reference** at launch
(env prefix `GIT_ADAPTER_`; token via `GITHUB_TOKEN`). The token is read once
from the process environment at startup; it is **never accepted as a tool
argument, never read from prompt content, and never written to any committed
config**. `.mcp.json` references the token by env/secret-ref only — never a
literal value.

**3. No secret is ever serialized into a prompt.** A token (or any secret-shaped
value) is **never placed into model-visible context** — not in a tool
description, tool argument, tool result, system prompt, or example. The MCP
boundary is treated as a trust boundary: the model proposes Git *intent*; the
shim holds the credential and performs the privileged call. This keeps the
secret out of the LLM context window entirely, which is the only durable defence
against accidental exfiltration via prompts, screenshots, or saved transcripts.

**4. Redaction in logs and traces is mandatory.** Any value matching the token
shape is redacted before it reaches logs, traces, spans, or error messages.
Redaction failures are treated as security defects, not cosmetic bugs. This pairs
with the no-secrets-in-prompts rule: the secret must not leak through the
*observability* path any more than through the *model* path.

**5. Least-privilege by default.** Tokens are scoped **per session / per repo**
to the minimum capability the authoring task requires. No broad, long-lived,
org-wide token is used for the authoring loop. Scope is the operator's
responsibility at injection time; the shim does not widen scope.

**6. External text is data, not instructions.** PR bodies, issue text, diffs, and
review comments surfaced through the MCP tools are treated as **untrusted data**,
never as instructions to the authoring agent (prompt-injection guard, consistent
with ADR-018's authorization posture for AI-readable content).

## Non-Goals

- **Implementation of the MCP server, CLI, or wiring.** This ADR records the
  decision and the security model only. The server (G.2), credential injection +
  redaction code (G.3), and `zynax mcp git` + `.mcp.json.example` (G.4) are
  delivered by later stories in EPIC G.
- **Deprecating or replacing the git-adapter.** Runtime workflows continue to use
  the gRPC capability adapter as the primary path. A future consolidation onto
  MCP is not decided here and remains explicitly out of scope.
- **A new credential store or secrets manager.** Token sourcing relies on the
  existing env/secret-ref injection model; this ADR does not introduce a vault or
  rotation mechanism.
- **Defining the exact MCP tool schema.** The 1:1 mapping to adapter handlers is
  mandated; the concrete tool names and argument shapes are an implementation
  detail of G.2.

## Rationale

| Option | Assessment |
|--------|------------|
| **A — MCP shim over the git-adapter** (chosen) | ✅ Single Git implementation; **one** credential/security surface to audit and harden; no behavioural drift between runtime and authoring paths; reuses ADR-013 adapter-first posture; security model can be fixed once and inherited by both surfaces. |
| **B — Independent MCP Git implementation** | ✗ Rejected — duplicates Git logic *and* the credential-handling surface; doubles the attack surface for token leakage; guarantees long-term drift; every security fix must be applied twice. |
| **C — MCP everywhere, deprecate the adapter** | ✗ Deferred — runtime workflows depend on the gRPC capability adapter today; ripping it out is a far larger one-way door than this EPIC scopes and is unnecessary to deliver an authoring Git surface. |

The security model (injection-at-start, no-secrets-in-prompts, redaction,
least-privilege) is recorded **in this ADR rather than a separate one** because
it is inseparable from the shim decision: the thin-shim design is what makes a
*single* credential surface possible, and a single surface is what makes the
no-secrets-in-prompts guarantee auditable.

## Consequences

### Positive

- Authoring agents gain Git tools without a second Git codebase to build or
  maintain.
- There is exactly **one** place to audit how a Git credential enters a process
  and how it is kept out of model context, logs, and traces.
- Behaviour is consistent across the runtime (gRPC) and authoring (MCP) surfaces,
  since both call the same adapter handlers.
- The security posture is settled before any implementation, satisfying the EPIC
  requirement that "security is settled first".

### Negative / trade-offs

- The shim **couples the MCP server to the adapter's interface** — adapter
  changes must keep the MCP tool mapping in sync (a contract/`.feature` test in
  G.2 guards this).
- A correctly-scoped token still grants real write access for the session;
  least-privilege reduces blast radius but does not eliminate it — operator
  discipline at injection time remains essential.

### Neutral / follow-up required

| Action | Tracking |
|--------|---------|
| MCP server over git-adapter capabilities (1:1 tool mapping, contract test) | EPIC G — G.2 (#1198) |
| Credential injection at start + redaction in logs/traces (redaction test) | EPIC G — G.3 (#1199) |
| `zynax mcp git` CLI + `.mcp.json.example` with least-privilege scope guide | EPIC G — G.4 (#1200) |
| Tier-2 `/spdd-security-review` PASS for the EPIC G canvas | Canvas `docs/spdd/1169-git-mcp-shim/canvas.md` |
