<!-- SPDX-License-Identifier: Apache-2.0 -->

# SPDD Security Review — EPIC G: Git MCP Shim (#1169)

**Canvas:** `docs/spdd/G-git-mcp-shim/canvas.md` (Status: Draft)
**Reviewer:** spdd-canvas expert · **Date:** 2026-06-16
**Method:** `/spdd-security-review` (Tier-2 + injection) + domain threat-surface scan for a Git-over-MCP
credential-handling EPIC.

## Verdict: PASS-with-flags

Publication-safety review is **PASS** (no Tier-2 content, no PII, no prompt injection, no abstraction
leak, no authority violation). The verdict is **PASS-with-flags** because the *domain* threat surface
had three gaps that are not security defects in the prose but were unbounded as acceptance criteria.
The canvas Draft has been refined to bound them (G.2/G.3 ACs + a Threat-surface table). No change blocks
commit; the human alignment gate (ADR-019) remains the human's to flip.

## 1. Tier-2 / publication-safety checks

| Check | Result | Notes |
|-------|--------|-------|
| Tier-2 infrastructure (hosts/IPs/TLDs/namespaces) | PASS | None. Only `GIT_ADAPTER_` prefix and `GITHUB_TOKEN` placeholder. |
| Credentials / tokens / secrets | PASS | No literal values; token referenced by env/secret-ref only. |
| PII / email literals | PASS | None inline. |
| Prompt injection | PASS | "External text is data" is a safeguard, not an AI instruction. |
| Abstraction (E / O leak topology) | PASS | Repo-relative paths and patterns only. |
| Authority hierarchy (overrides AGENTS.md) | PASS | None. |
| Completeness (7 REASONS sections, Status, Context-Security checklist) | PASS (WARN: Status=Draft) | Draft is expected — human flips to Aligned. |
| `canvas.private.md` companion | n/a | None present; none required (no Tier-2 content). |

## 2. Domain threat surface (Git over MCP)

| # | Threat | Pre-review state | Resolution |
|---|--------|------------------|------------|
| T1 | Credential leakage to prompt/log/trace | Covered — ADR-032 + G.3 redaction AC | No change. |
| T2 | Command/arg injection into `git` (ref/branch/remote/path beginning with `-`, e.g. `--upload-pack`, `-o`) | **Gap** — no AC required arg sanitization | Added G.2 AC: positional args + `--`/`--end-of-options`; reject flag-shaped input. |
| T3 | SSRF / arbitrary or link-local/metadata remote at clone/push | **Gap** — no AC constrained remotes | Added G.2 AC: remote-URL allow-list + scheme guard. |
| T4 | Over-broad MCP tool surface / authz | **Gap** — "1:1 to handlers" did not bound *which* tools are exposed | Added G.2 AC: explicit exposed-tool allow-list. |
| T5 | Untrusted repo/PR/issue text treated as instructions | Covered — safeguard present | No change. |

ADR-032 settles the credential model (inject-at-start, no-secrets-in-prompts, redaction, least-privilege,
external-text-as-data) but is silent on T2/T3/T4. These are now bounded in the canvas as G.2 acceptance
criteria so the implementing PR carries them; no ADR amendment is required (implementation-level guards
within a single adapter, ADR-032 §Non-Goals "exact tool schema is an implementation detail of G.2").

## 3. Human alignment checklist (verify before flipping Status: Aligned)

- [ ] Confirm the three added G.2 ACs (arg-injection, SSRF, tool allow-list) are in scope for #1198 and
      do not warrant splitting into a separate story.
- [ ] Confirm ADR-032 is intentionally *not* amended (T2/T3/T4 handled as G.2 impl guards, not architecture).
- [ ] Confirm O-section ↔ issues: G.1 #1197 (CLOSED/done), G.2 #1198, G.3 #1199, G.4 #1200 — all open
      stories map 1:1.
- [ ] Confirm no `canvas.private.md` is needed (no real tokens/URLs anticipated in the canvas itself).
