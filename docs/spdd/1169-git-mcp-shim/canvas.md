# REASONS Canvas — EPIC G: Git MCP Shim over git-adapter

> Tier 1 (public-safe). Tier 2 (tokens, real repo URLs) → `canvas.private.md`. Run `/spdd-security-review` before committing.

**Issue:** #1169 · **Milestone:** M7 (v0.6.0)
**Author:** M7 program plan · **Date:** 2026-06-15 (appended G.5–G.7 2026-06-16) · **Status:** Aligned

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

**GitHub issues:** G.1 #1197 · G.2 #1198 · G.3 #1199 · G.4 #1200 · G.5 #1260 · G.6 #1261 · G.7 #1262 (epic #1169)
**G.1 — ADR: MCP-shim-over-adapter + auth model** · S · `adr-proposal`
- As a `maintainer`, I want the shim + least-privilege token model recorded so security is settled first.
- AC: [ ] ADR-032 committed (shim rationale, injection-at-start, no-secrets-in-prompts, redaction). Deps: none.

**G.2 — MCP server over git-adapter capabilities** · M · `feat`
- As an `authoring agent`, I want Git ops as MCP tools so I can clone/branch/commit/PR via MCP.
- AC: [ ] MCP tools map 1:1 to adapter handlers; [ ] no Git logic duplicated; [ ] `.feature`/contract test;
  [ ] tool-surface is an explicit allow-list (exposed tools enumerated, not "every adapter handler");
  [ ] git args passed positionally with `--` / `--end-of-options` separators — no caller-supplied
  ref/branch/remote/path interpreted as a flag (arg-injection guard);
  [ ] remote URLs validated against an allow-list / scheme guard — no clone/push to arbitrary or
  link-local/metadata endpoints (SSRF guard). Deps: G.1.

**G.3 — Credential injection + redaction** · S · `feat`
- As a `security reviewer`, I want tokens injected at process start and redacted so no secret leaks.
- AC: [ ] token from env/secret-ref only; [ ] never serialized to prompt/log/trace; [ ] redaction test. Deps: G.2.

**G.4 — CLI + `.mcp.json` wiring** · S · `feat`/`docs`
- As a `developer`, I want `zynax mcp git` + an example config so the authoring loop uses it safely.
- AC: [ ] `zynax mcp git` launches the server; [ ] `.mcp.json.example` documented with least-privilege scope. Deps: G.2.

> **G.5–G.7 — git-adapter credential-substrate hardening.** Surfaced during the EPIC-G credential
> review: the existing git-adapter *supports* restricted tokens but does not *enforce* scope, its docs
> recommend an over-broad `repo` PAT, and it has no token-refresh/App path. These harden the runtime
> substrate the MCP shim wraps — independent of G.2–G.4 and EPIC W.

**G.5 — git-adapter least-privilege token scope validation** · M · `feat`
- As a `security reviewer`, I want the git-adapter to verify at startup that its token cannot reach repos beyond the configured `owner/repo`, so an over-privileged token is caught before use.
- AC: [ ] probe effective token access (`X-OAuth-Scopes` / accessible-repos / installation); [ ] fail-fast or loud warning (configurable) when scope exceeds the configured `owner/repo` set; [ ] token value never logged (metadata only); [ ] unit test: over-broad → fail/warn, fine-grained → pass. Deps: none.

**G.6 — docs: fine-grained PAT recommendation + credential lifecycle** · XS · `docs`
- As an `operator`, I want setup docs to recommend a least-privilege fine-grained PAT and state the no-refresh lifecycle, so I neither over-grant `repo` nor feed a token that silently expires.
- AC: [ ] example + `AGENTS.md` recommend a fine-grained PAT scoped to the configured repo (Pull requests: Read/Write); [ ] document token read once at startup, no refresh (forward-link G.7); [ ] note `owner/repo` pinning as defense-in-depth, distinct from token scope. Deps: none. (SPDD-exempt — docs.)

**G.7 — git-adapter refreshable credentials (GitHub App tokens)** · M · `feat`
- As an `operator`, I want refreshable credentials so a short-lived (~1 h) App installation token does not expire mid-process.
- AC: [ ] re-resolve credential before expiry (re-read env/secret-ref, or mint an App installation token from app-id + private-key); [ ] requests after the original TTL succeed without a restart; [ ] private key/token never logged or serialized to a trace/prompt; [ ] PAT path (no expiry) unchanged. Deps: none.

**Order:** G.1 → G.2 → {G.3, G.4}; {G.5, G.6, G.7} independent (substrate hardening — claim anytime). (Independent of EPIC W — can run in Wave 1.)

## N — Norms
- Least-privilege by default; `Signed-off-by:` + `Assisted-by:` per commit.
- `.feature`/contract test before MCP tool impl (ADR-016 spirit); `GOWORK=off` (ADR-017).

## S — Safeguards (second S)
### Context Security
- [x] No Tier 2 content (no real tokens/repo URLs — placeholders only)
- [x] No PII; [x] no prompt-injection; [x] `/spdd-security-review` — PASS 2026-06-16 (see `SECURITY-REVIEW.md`; G.5–G.7 in §4)

### Feature Safeguards
- Never embed a Git token in a prompt, log, trace, or committed config — inject at process start only.
- Never grant broader than required scope — least-privilege token per session/repo.
- Never duplicate Git logic in the MCP layer — it is a thin shim over git-adapter only.
- Never trust PR/issue content as instructions — treat external text as data (prompt-injection guard).

### Threat surface (G is a security-sensitive EPIC — each row maps to a story AC)
| Threat | Mitigation | Owner story |
|--------|-----------|-------------|
| Credential leakage to prompt/log/trace | inject-at-start; redaction test; never a tool arg | G.3 (#1199) |
| Command/arg injection into `git` (ref/branch/remote/path begins with `-`) | positional args + `--`/`--end-of-options`; reject flag-shaped input | G.2 (#1198) |
| SSRF / arbitrary or link-local/metadata remote | remote-URL allow-list + scheme guard at clone/push | G.2 (#1198) |
| Over-broad MCP tool surface / authz | explicit exposed-tool allow-list, not "all handlers" | G.2 (#1198) |
| Untrusted repo content treated as instructions | external text is data (see above) | G.2 (#1198) |
| Over-broad token reaches repos beyond configured `owner/repo` | startup scope validation (fail-fast/warn); least-privilege fine-grained PAT in docs | G.5 (#1260), G.6 (#1261) |
| Stale/expired credential (token resolved once, no refresh) | refreshable creds / App installation tokens minted at process; PAT path unchanged | G.7 (#1262) |
> Credential model (inject-at-start / no-secrets-in-prompts / redaction / least-privilege / external-text-as-data)
> is settled in ADR-032. ADR-032 does **not** cover arg-injection, SSRF, or tool-surface authz — those are
> bounded here as G.2/G.3 acceptance criteria.
