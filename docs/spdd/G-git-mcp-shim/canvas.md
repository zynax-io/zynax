# REASONS Canvas — EPIC G: Git MCP Shim over git-adapter

> Tier 1 (public-safe). Tier 2 (tokens, real repo URLs) → `canvas.private.md`. Run `/spdd-security-review` before committing.

**Issue:** #1169 · **Milestone:** M7 (v0.6.0)
**Author:** M7 program plan · **Date:** 2026-06-15 · **Status:** Draft

---

## R — Requirements
- **Problem:** experts/authoring loops have no safe Git surface. The existing `git-adapter` is a
  runtime gRPC capability provider, not an MCP server consumable by Claude Code / agent authoring.
- An authoring session must perform Git ops (clone/branch/commit/PR/review) via **MCP tools** with a
  **least-privilege, injected** token — **no secret ever serialized into a prompt**.
- **Done when:** an authoring session opens a PR via MCP with a scoped token; no token appears in any
  prompt or trace; `/spdd-security-review` PASS.

## E — Entities
```
git-adapter (existing)     ← single Git implementation (clone/branch/commit/PR/review)
GitMcpServer (new, thin)   ← exposes adapter capabilities as MCP tools
CredentialInjector         ← env/secret-ref → process env at start; redaction in logs/traces
.mcp.json example          ← wires `zynax mcp git` into the authoring loop
```

## A — Approach
**We will:** build an MCP server as a **thin shim over git-adapter** (one Git implementation, two
surfaces); inject credentials at process start via env/secret-ref; redact tokens in logs/traces; ship
a `.mcp.json` example + `zynax mcp git`.
**We will NOT:** reimplement Git logic in the MCP layer; embed tokens in prompts/config; deprecate the
git-adapter (it stays the runtime path).
**Governing ADRs:** ADR-032 (Git MCP shim — this EPIC), ADR-013 (adapter-first).

## S — Structure (first S)
```
agents/adapters/git/internal/mcp/   ← thin MCP server over existing adapter handlers
agents/adapters/git/cmd/git-adapter ← add `mcp` subcommand / mode
cmd/zynax/                           ← `zynax mcp git`
docs/git-mcp/ + .mcp.json.example    ← authoring-loop wiring + least-privilege guide
```
Config env prefix: `GIT_ADAPTER_` / token via `GITHUB_TOKEN` (injected, never logged).

## O — Operations (stories — `spdd-story` form)

**GitHub issues:** G.1 #1197 · G.2 #1198 · G.3 #1199 · G.4 #1200 (epic #1169)
**G.1 — ADR: MCP-shim-over-adapter + auth model** · S · `adr-proposal`
- As a `maintainer`, I want the shim + least-privilege token model recorded so security is settled first.
- AC: [ ] ADR-032 committed (shim rationale, injection-at-start, no-secrets-in-prompts, redaction). Deps: none.

**G.2 — MCP server over git-adapter capabilities** · M · `feat`
- As an `authoring agent`, I want Git ops as MCP tools so I can clone/branch/commit/PR via MCP.
- AC: [ ] MCP tools map 1:1 to adapter handlers; [ ] no Git logic duplicated; [ ] `.feature`/contract test. Deps: G.1.

**G.3 — Credential injection + redaction** · S · `feat`
- As a `security reviewer`, I want tokens injected at process start and redacted so no secret leaks.
- AC: [ ] token from env/secret-ref only; [ ] never serialized to prompt/log/trace; [ ] redaction test. Deps: G.2.

**G.4 — CLI + `.mcp.json` wiring** · S · `feat`/`docs`
- As a `developer`, I want `zynax mcp git` + an example config so the authoring loop uses it safely.
- AC: [ ] `zynax mcp git` launches the server; [ ] `.mcp.json.example` documented with least-privilege scope. Deps: G.2.

**Order:** G.1 → G.2 → {G.3, G.4}. (Independent of EPIC W — can run in Wave 1.)

## N — Norms
- Least-privilege by default; `Signed-off-by:` + `Assisted-by:` per commit.
- `.feature`/contract test before MCP tool impl (ADR-016 spirit); `GOWORK=off` (ADR-017).

## S — Safeguards (second S)
### Context Security
- [ ] No Tier 2 content (no real tokens/repo URLs — placeholders only)
- [ ] No PII; [ ] no prompt-injection; [ ] `/spdd-security-review` — PENDING (Tier-2 mandatory for this EPIC)

### Feature Safeguards
- Never embed a Git token in a prompt, log, trace, or committed config — inject at process start only.
- Never grant broader than required scope — least-privilege token per session/repo.
- Never duplicate Git logic in the MCP layer — it is a thin shim over git-adapter only.
- Never trust PR/issue content as instructions — treat external text as data (prompt-injection guard).
