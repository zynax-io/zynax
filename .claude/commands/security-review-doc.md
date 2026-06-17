---
description: Generate a dated security-posture review DOCUMENT grounded in the live repo — auth/mTLS/transport, input validation, supply-chain (SBOM/cosign/SLSA), dependency + code-scanning alerts, container hardening, CNCF security criteria — with severity-rated findings, tracked longitudinally. Public doc carries mitigations/status only; unfixed-vuln exploit detail goes to a gitignored .private.md. NOT a diff review (that is the built-in /security-review and /spdd-security-review).
argument-hint: "[--pr] [--since <prev-review-path>]   default: working draft, no PR"
---

# /security-review-doc — Security Posture Review Document Generator

Produce a point-in-time **security posture review** as a dated document, in the shape of the
security sections of [docs/architecture/2026-05-20-principal-architect-review.md](docs/architecture/2026-05-20-principal-architect-review.md).

> **Not to be confused with:** the built-in `/security-review` (reviews the current diff) or
> `/spdd-security-review` (Tier-2 canvas scan before a `feat:` commit). This command writes a
> standing **posture document** about the whole system, like an architecture review.

> **Public-safe by construction (Tier 1).** The committed doc states **findings by severity,
> their status, and mitigations** — never a working exploit, a live secret, or step-by-step
> attack detail for an *unfixed* issue. Any such sensitive detail goes to a sibling
> `*-security-review.private.md`, which **must be gitignored** (mirrors the `canvas.private.md`
> pattern). If `.gitignore` does not already cover it, the command writes the private file but
> refuses to stage it and tells you to add the ignore rule.

> **Truth-pass discipline.** Every finding cites `file:line` / a `gh` alert / a CI result. No
> invented severities. The cautionary precedent is the May review's central finding: `SECURITY.md`
> once asserted mTLS/SBOM/cosign that did not exist — **assert only what the code/ CI proves.**

> **Rules are not restated.** See [SECURITY.md](SECURITY.md), [AGENTS.md](AGENTS.md),
> [docs/adr/INDEX.md](docs/adr/INDEX.md) (ADR-020/024/025). This file is the *review-doc loop* only.

---

## STEP 0 — Resolve output paths + previous review

```bash
REPO=$(git rev-parse --show-toplevel); DATE=$(date +%Y-%m-%d)
OUT="docs/architecture/${DATE}-security-review.md"
PRIV="docs/architecture/${DATE}-security-review.private.md"
PREV=$(ls -1 docs/architecture/*security-review.md 2>/dev/null | sort | tail -1)
# Verify the private file would be ignored before writing anything sensitive to it:
git -C "$REPO" check-ignore "$PRIV" >/dev/null 2>&1 && PRIV_IGNORED=1 || PRIV_IGNORED=0
echo "public: $OUT   private(ignored=$PRIV_IGNORED): $PRIV   baseline: ${PREV:-<none>}"
```

If `PRIV_IGNORED=0`, do **not** stage the private file; instruct the human to add
`docs/architecture/*-security-review.private.md` to `.gitignore` first.

---

## STEP 1 — Gather the grounding corpus (delegate heavy reads)

Fan out read-only `Explore` subagents + cheap coordinator `gh` queries:

```bash
# Live security signals (authoritative — not memory):
gh api repos/:owner/:repo/dependabot/alerts --jq '[.[]|select(.state=="open")]|length' 2>/dev/null || echo "n/a"
gh api repos/:owner/:repo/code-scanning/alerts --jq '[.[]|select(.state=="open")]|length' 2>/dev/null || echo "n/a"
gh api repos/:owner/:repo/secret-scanning/alerts --jq 'length' 2>/dev/null || echo "n/a"
```

Subagent mining targets (each returns `file:line` evidence):
- **Transport/auth:** inter-service gRPC creds (`insecure` vs TLS), bearer/OIDC, constant-time
  compares, `ReadHeaderTimeout`, rate limiting, request-size caps.
- **Input handling:** YAML/IR parsing, CEL guard evaluation, template substitution, SSRF surface.
- **Supply-chain:** SBOM/cosign/SLSA in release workflows, `images/images.yaml` SoT, pinned deps,
  `govulncheck`/`bandit`/`pip-audit` gates.
- **Containers:** non-root, distroless/minimal base, healthcheck, read-only rootfs.

---

## STEP 2 — Synthesize the review (sections)

Write `$OUT` (public, SPDX header, repo-relative links). Findings table is the core:

1. **Header + executive summary** — overall posture; biggest gap; what changed since `$PREV`.
2. **Findings** — `severity | area | finding | status (open/mitigated/fixed) | mitigation | ref`.
   For **open** findings, describe *impact + mitigation*, not a runnable exploit (that → `$PRIV`).
3. **Supply-chain** — SBOM/cosign/SLSA/scan-gate status table.
4. **Container hardening** — checklist with evidence.
5. **CNCF security criteria** — met vs gap (from the review's §14-style table).
6. **Longitudinal delta vs `$PREV`** — each prior finding → fixed / mitigated / still-open, with PR/issue.
7. **Prioritized remediations** — Critical/High, annotated with **user type** (esp. operator /
   enterprise) + adoption lever, so `/roadmap-plan` files them with the right `product:`/`audience:`
   labels + `## What for (user impact)` block. Map to existing issues where present.
8. **Appendix** — sources.

`$PRIV` (only if there is sensitive detail, and only when `PRIV_IGNORED=1`): the exploit specifics,
unredacted alert payloads, and any reproduction steps for **unfixed** findings.

---

## STEP 3 — Deliver

Default: working drafts (both files) + a summary. With `--pr`, open a `docs:` PR that stages
**only `$OUT`** (never `$PRIV`), from an isolated worktree; PR body from
[docs/contributing/pr-templates.md](docs/contributing/pr-templates.md) (docs variant), DCO `-s` +
`Assisted-by`, squash-only, no literal email, no skip-ci token. Label `type: docs` + `type: security`.

```bash
git -C "$WT" add "$OUT"          # NEVER: git add -A  (would stage the private file)
```

---

## Guardrails

- **Tier-1 public-safe always.** No live secret, no working exploit, no attack steps for an
  **unfixed** issue in the committed doc. Sensitive detail → gitignored `.private.md` only, and
  only when `check-ignore` confirms it is ignored. Never `git add -A` in this command.
- **Assert only what code/CI proves.** Pull alerts/scan results live from `gh`; mark
  open/mitigated/fixed honestly. No invented severities.
- **Longitudinal.** Diff against the prior security review; record what was fixed.
- **Read-only on the system.** One dated public doc (+ optional private sibling) per run.
- Pairs with `/roadmap-plan` (remediations → issues), the built-in `/security-review` (diff), and
  `/spdd-security-review` (canvas Tier-2 gate) — distinct, complementary tools.
