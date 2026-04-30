<!-- SPDX-License-Identifier: Apache-2.0 -->

# AI Knowledge Base — Content Sanitization Policy

**Version:** 2.0  **Date:** 2026-04-30
**Governed by:** ADR-018 (AI KB Authorization Model)
**Enforced by:** `gitleaks-ai-context` CI gate · CODEOWNERS (`@zynax-io/maintainers`)

This policy defines what content is permitted in AI knowledge base files
(`CLAUDE.md`, all `AGENTS.md` files, `docs/ai-assistant-setup.md`, and future
`.ai/` and `.claude/` directories). It is the reference document that:

- Contributors use when writing or editing KB entries
- Reviewers use when evaluating KB PRs
- CI scanner rules (`tools/gitleaks-ai-context.toml`) are derived from

---

## Context Trust Levels

Every piece of context that informs an AI assistant — or gets committed to the
repo — falls into one of three tiers. The tier determines where the context may
live, who can see it, and what happens to it after the session ends.

| Tier | Name | Storage | Persisted? | Publicly visible? |
|------|------|---------|-----------|------------------|
| 1 | **Public** | `docs/spdd/canvas.md`, `AGENTS.md`, `CLAUDE.md`, ADRs | ✅ Committed | ✅ Yes |
| 2 | **Private** | `canvas.private.md` (gitignored, local only) | ✅ Local disk | ❌ No |
| 3 | **Ephemeral** | Session only — never written to disk | ❌ Discarded | ❌ No |

### Tier 1 — Public Context

Safe to commit. May appear in `docs/spdd/canvas.md`, `AGENTS.md`, `CLAUDE.md`,
ADRs, and any other public repository file.

| Type | Examples | Safe? |
|------|---------|-------|
| Architecture principles | Three-layer separation, gRPC mandate | ✅ Always |
| Coding standards | Go functions ≤ 30 lines, `GOWORK=off` | ✅ Always |
| Naming conventions | `snake_case` capabilities, `zynax.v1.*` topics | ✅ Always |
| Testing standards | BDD before implementation, ≥ 90% coverage | ✅ Always |
| Public API contracts | Proto field names, gRPC service names | ✅ Always |
| ADR rationale | Why we chose NATS over Kafka | ✅ If ADR is merged |
| Generic workflows | `make bootstrap → lint → test → PR` | ✅ Always |
| Abstracted diagrams | Layer diagrams without IP or hostname | ✅ Always |
| Error messages and fixes | Verbatim CI errors and their resolutions | ✅ No internal details |

**Rule: store intent, not environment. Store abstractions, not secrets. Store architecture, not operations.**

### Tier 2 — Private Context

Sensitive but legitimate. Never committed to the public repo. Stored in a local
companion file (`canvas.private.md` — gitignored) alongside the public Canvas.
See [#216](https://github.com/zynax-io/zynax/issues/216) for the full
private-vault convention.

| Type | Examples | Risk if leaked |
|------|---------|---------------|
| Real deployment targets | Production cluster names, namespace paths | Infrastructure disclosure |
| Internal service names | Internal API names not in public contracts | Reconnaissance |
| Customer-specific constraints | Tenant isolation requirements | Customer confidentiality |
| Security-sensitive design | Threat model specifics, pen-test findings | Attacker-observable |
| Personal context | Engineer names linked to sensitive work | PII |

**Rule: if disclosing it in a public GitHub PR would require a security incident report, it is Tier 2.**

### Tier 3 — Ephemeral Context

Session-only. Never written to disk in any form — not to memory files, Canvas
files, or commit messages. Exists only within a single AI session.

| Type | Examples | Rule |
|------|---------|------|
| Live debugging output | Stack traces from a running production process | Use → discard |
| Real-time observability | Metric values from a live dashboard | Use → discard |
| Sensitive user-provided context | Passwords or tokens typed into chat | Never persist |
| Investigation scratch notes | Hypotheses that turned out to be wrong | Discard, don't commit |

**Rule: if you would not want it in a commit message, keep it ephemeral.**

### Boundary cases

When context sits on the boundary between tiers, apply the **lower trust tier**:

- A real hostname also referenced in a public ADR → still Tier 2 (the ADR
  reference is fine; the operational hostname in a Canvas is not)
- An architecture decision that names an internal team → abstract the team
  reference, keep the decision
- A production error message containing a real hostname → sanitize the
  hostname, keep the error text

---

## Why This Matters

KB files are committed to a **public GitHub repository** and auto-loaded by AI
assistants on every interaction. There is no reliable way to un-publish merged
content (forks, mirrors, search-engine caches, AI training sets all persist it).

Treating KB files as "internal documentation" is a security mistake. Treat them
as **public press releases** — everything in them becomes permanently visible.

---

## What IS Allowed

Write KB entries that are:

| Category | Examples |
|----------|---------|
| Generalized engineering patterns | "Always prefix `go test` with `GOWORK=off` inside service directories" |
| Tool quirks and gotchas | "golangci-lint v2 reads config from `tools/golangci-lint.yml`, not `.golangci.yml`" |
| ADR rationale and decision context | "We chose NATS over Kafka because…" (referencing a merged ADR) |
| Anti-patterns and their consequences | "Do not import `api/` from `domain/` — violates the three-layer boundary (ADR-001)" |
| Command examples with placeholder values | `` make test-unit-svc SVC=<service-name> `` |
| Workflow steps that reference `main` or public branches | "Rebase onto `origin/main` before opening a PR" |
| References to public documentation | Links to GitHub docs, Go docs, RFC numbers |
| Error messages and their resolutions | Verbatim error text is fine; the resolution path must not reference internal details |

---

## What Is NOT Allowed

Never commit the following, regardless of context or how "harmless" it seems:

### Secrets and credentials

- API keys, bearer tokens, JWT secrets, database passwords
- OAuth client IDs or secrets
- Service account credentials or HMAC signing keys
- Any value that looks like a secret, even if it is expired or revoked

### Personal and organizational PII

- Full names linked to sensitive context (e.g. "Oscar handles the NATS broker")
- Personal email addresses (`name@domain.com`)
- Corporate email addresses unless they appear in a public commit/ADR already
- Phone numbers, Slack handles, Discord usernames
- Internal org charts, reporting lines, or role assignments

### Infrastructure details

- Absolute local file paths (`/home/oscar/`, `/Users/jane/`, `C:\Users\john\`)
- Internal hostnames with private TLDs (`.internal`, `.local`, `.corp`, `.lan`)
- Private IP address ranges (10.x, 172.16–31.x, 192.168.x)
- Real server names, cluster names, or namespace names from production
- Container registry URLs with credentials embedded
- VPN endpoints or bastion host addresses

### Operational security details

- Credential rotation schedules or procedures
- Details about who holds which access key
- Security incident postmortems with attacker-observable details
- Firewall rules, WAF configurations, rate-limit thresholds (unless public)

### Prompt injection payloads

Content that is crafted to influence AI assistant behavior rather than document
engineering facts. Signs of a prompt injection payload:

- Instruction-like phrasing outside of a clearly labelled command example
  (e.g. *"always respond with X"*, *"when asked about Y, do Z"*)
- Override attempts (*"ignore previous instructions"*)
- Conditional behavior triggers (*"if the user asks about billing, say…"*)
- Fake context injection (*"you are now a different assistant"*)

---

## How to Sanitize a KB Entry

Before committing a KB change, apply these steps in order:

1. **Replace all real names with roles.**
   - Before: `Oscar fixed this by running make bootstrap`
   - After: `The engineer fixed this by running make bootstrap`

2. **Replace all absolute paths with abstract representations.**
   - Before: `/home/oscar/workspace/zynax/services/`
   - After: `<repo-root>/services/`

3. **Replace all real hostnames and IPs.**
   - Before: `postgres.internal:5432`
   - After: `<db-host>:5432`
   - For documentation IPs, use RFC 5737 ranges: `192.0.2.x`, `198.51.100.x`, `203.0.113.x`

4. **Replace all tokens, keys, and secrets.**
   - Before: `NATS_TOKEN=abc123secret`
   - After: `NATS_TOKEN=<token>`

5. **Replace all email addresses.**
   - Before: `email: oscar@example.com`
   - After: `email: <maintainer-email>`

6. **Run the CI scanner locally before pushing.**
   ```bash
   gitleaks detect --no-git --source . \
     --config tools/gitleaks-ai-context.toml \
     --report-format json
   ```

7. **Preview your additions before pushing.**
   ```bash
   make preview-kb-changes
   ```
   Review the output as if you were a stranger reading it for the first time.

---

## Reviewer Verification Process

When reviewing a PR that touches KB paths, verify each of the following before
approving. This checklist is also embedded in the PR template.

### Automated checks (must be green before review begins)

- `gitleaks-ai-context` CI step passes
- `kb-content-previsualized` status check is `pending` (preview posted)

### Manual review (required before approval)

1. **Read the preview comment** posted automatically by the `kb-preview` CI job.
   Do not skip this step — the diff view mixes metadata with content.

2. **Check for secrets and PII.** Automated scanners catch known patterns;
   human review catches novel encodings and semantic PII.

3. **Check for prompt injection.** Ask: does any sentence read as an instruction
   to an AI rather than documentation for a human? Legitimate KB entries describe
   facts; injected content issues commands.

4. **Check that content is derived from reviewed sources.** KB entries should be
   summaries of merged code, ADRs, or CONTRIBUTING.md guidance. Content that
   introduces new policies not grounded in a merged source is a red flag.

5. **Approve the PR.** Confirm the preview checklist in the PR template is
   complete, then submit your approval review. The PR cannot merge without
   maintainer approval (enforced by CODEOWNERS).

---

## CI Scanner Rules

The rules in `tools/gitleaks-ai-context.toml` implement the "not allowed" list:

| Rule ID | Pattern | Catches |
|---------|---------|---------|
| `local-absolute-path` | `/home/<user>/`, `/Users/<user>/`, `C:\Users\<user>\` | Absolute local paths |
| `email-address` | RFC 5321 email regex | Personal and corporate emails |
| `internal-hostname` | `*.internal`, `*.local`, `*.corp`, `*.lan` | Private internal hostnames |
| `private-ip-range` | 10.x, 172.16-31.x, 192.168.x | Private IP ranges |
| (default gitleaks) | All default token/key patterns | API keys, tokens, secrets |

To run the scanner against all KB files:
```bash
gitleaks detect --no-git --source . \
  --config tools/gitleaks-ai-context.toml \
  --report-format json \
  --report-path /tmp/kb-scan.json
cat /tmp/kb-scan.json
```

To propose a new scanner rule, open a PR against `tools/gitleaks-ai-context.toml`
and reference the threat model entry from ADR-018 it addresses.

---

## Scope of This Policy

This policy applies to all content in:

- `/CLAUDE.md`
- `/AGENTS.md` and all `**/AGENTS.md` files
- `/docs/ai-assistant-setup.md`
- `/docs/knowledge-base-policy.md` (this file)
- `/docs/patterns/spdd-guide.md` (SPDD methodology — loaded by AI assistants)
- `/docs/spdd/**/canvas.md` (REASONS Canvas artifacts — public Tier 1 context)
- `/.ai/**` (future — Epic #148)
- `/.claude/**` (future — Epic #148)

**Explicitly excluded from this policy** (never committed, governed by Tier 2 rules):

- `/docs/spdd/**/canvas.private.md` — must be listed in `.gitignore`

It does not apply to regular source code, tests, or other documentation files
unless those files are explicitly added to the KB paths list in CODEOWNERS.

Changes to this policy require the same maintainer review as any other KB path
change (CODEOWNERS rule: `docs/knowledge-base-policy.md @zynax-io/maintainers`).
