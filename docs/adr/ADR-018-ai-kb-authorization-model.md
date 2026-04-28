# ADR-018: AI Knowledge Base Authorization Model

**Status:** Accepted  **Date:** 2026-04-24
**Related:** ADR-016 (Layered Testing Strategy), Epic #157 (Public KB Security)

---

## Context

Zynax stores engineering context in files that AI assistants auto-load on every
interaction:

| Path | Auto-loaded by |
|------|----------------|
| `CLAUDE.md` | Claude Code (all contributors) |
| `AGENTS.md` (root + every subdirectory) | Claude Code, Copilot, custom agents |
| `docs/ai-assistant-setup.md` | Referenced from CLAUDE.md |
| `.ai/` (future — Epic #148) | Claude Code, custom agents |
| `.claude/` (future — Epic #148) | Claude Code |

These files sit in a **public GitHub repository**. Every byte committed is
visible to the internet immediately on merge. This is a one-way door — once
content is published, it cannot be reliably unpublished (forks, mirrors, search
indexes, AI training sets).

Without an explicit authorization model, any contributor with PR rights can
modify KB files and silently influence AI assistant behavior for every engineer
in the repo. The threat vectors are concrete:

| ID | Threat |
|----|--------|
| T1 | Accidental secret exposure — API keys, tokens, or local paths in KB files |
| T2 | PII leakage — engineer names, emails, or roles committed to a public repo |
| T3 | Infrastructure fingerprinting — internal topology or credential details |
| T4 | Prompt injection via PR — content that looks like documentation but steers AI assistant behavior |
| T5 | Authorization bypass — contributor silently shifts AI behavior over time via KB PRs |

---

## Decision

AI knowledge base files are **restricted paths** governed by a mandatory
authorization policy:

1. **CODEOWNERS** — all KB paths require explicit approval from
   `@zynax-io/maintainers`. Even if branch protection already routes all PRs
   through maintainers, listing KB paths explicitly makes the security intent
   legible to contributors and tooling.

2. **Branch protection** — `Require review from Code Owners` must be enabled on
   `main`. A PR author cannot self-approve a KB change.

3. **Dedicated review checklist** — KB PRs require additional scrutiny beyond the
   standard engineering checklist. The checklist (ADR-related issue #160) covers:
   - Secret / PII scan passed
   - No prompt-injection payloads
   - Content matches reviewed source material
   - Previsualization approved (issue #162)

4. **CI gating** — the `gitleaks-ai-context` CI step (already in place since
   PR #164) scans KB files for secrets and PII on every PR that touches them.
   Policy document issue #161 formalizes what the scanner must catch.

### Canonical KB paths

The following paths are designated AI knowledge base paths for CODEOWNERS and
CI scanning purposes:

```
/CLAUDE.md
/AGENTS.md
**/AGENTS.md
/docs/ai-assistant-setup.md
/.ai/
/.claude/
```

Note: `.ai/` and `.claude/` do not yet exist on disk. They are pre-declared
here so that when Epic #148 adds them, protection is already in effect.

---

## Rationale

| Option | Assessment |
|--------|------------|
| No access control (anyone with PR rights) | ✗ T4 and T5 threats are open; a single malicious or careless PR shifts AI behavior globally |
| `@zynax-io/maintainers` required for KB paths | ✅ Minimal blast radius; maintainers are already the approval authority for architecture decisions |
| Separate `@zynax-io/kb-owners` team | ✗ Adds team management overhead; until KB is a major workstream, a dedicated team is premature |
| CI-only enforcement (no CODEOWNERS) | ✗ CI cannot block a determined maintainer who self-approves a PR; CODEOWNERS + branch protection is the correct control plane |
| Require two maintainer approvals for KB paths | Acceptable escalation path if T4 proves a real attack vector; deferred until a KB incident or policy review (issue #160 can revisit) |

The wildcard `*` already points all changes to `@zynax-io/maintainers` in the
current CODEOWNERS. Adding explicit KB path entries does not change enforcement
today — it documents intent, enables tooling to identify KB PRs, and allows a
future stricter rule (e.g., a two-maintainer requirement) to be applied only to
KB paths without touching general code review policy.

---

## Consequences

- **`CODEOWNERS`** — explicit entries added for all KB paths.
- **`CONTRIBUTING.md §11`** — AI-Assisted Contributions section references this
  ADR and the KB authorization policy so contributors understand why KB PRs
  require additional steps.
- **`PULL_REQUEST_TEMPLATE.md`** — a KB-specific review checklist will be added
  in issue #160 (S4 — PR reviewer checklist for prompt injection).
- **`#161` (S5)** and **`#162` (previsualization gate)** depend on this ADR
  being merged first.
- **Knowledge base issues (#143–#148)** remain blocked until Epic #157 is fully
  closed (all controls merged and verified).
- When `.ai/` and `.claude/` are created (Epic #148), no CODEOWNERS change is
  needed — the entries declared here take effect automatically.
