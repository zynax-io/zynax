---
description: Generate a dated, scored architecture review document grounded in the live repo — three-layer separation, ADR adherence, hexagonal services, scalability/security/reliability, CNCF fit — and tracked longitudinally against the previous architecture review (which prior risks/recommendations are now closed). Read-only synthesis; writes one dated doc under docs/architecture/ and opens a docs: PR with --pr.
argument-hint: "[--pr] [--since <prev-review-path>]   default: working draft, no PR"
---

# /architecture-review — Architecture Review Document Generator

Produce a principal-engineer-grade architecture review as a **dated document**, in the exact
shape and home of the existing reviews under [docs/architecture/](docs/architecture/) (e.g.
[2026-05-20-principal-architect-review.md](docs/architecture/2026-05-20-principal-architect-review.md)).
This is a point-in-time assessment, not a living doc — and it is **longitudinal**: it diffs
against the previous architecture review and records which risks/recommendations have since closed.

> **Read-mostly.** It reads the repo and live GitHub state and writes exactly one new dated
> Markdown file. With `--pr` it opens a `docs:` PR; otherwise it leaves the file as an
> uncommitted working draft for you to review. It never edits code, ADRs, or `AGENTS.md`.

> **Truth-pass discipline (non-negotiable).** Every score, strength, and weakness is **grounded
> in a cited repo artifact** (`file:line`, ADR#, issue#, or `gh` query result). No invented
> numbers. Separate **shipped / partial / aspirational** explicitly. This is the lesson the M5.A
> Truth Pass and [docs/product/strategy.md](docs/product/strategy.md) encode: an ungrounded
> review is worse than none.

> **Rules are not restated.** Architecture invariants live in [AGENTS.md](AGENTS.md) and
> [docs/adr/INDEX.md](docs/adr/INDEX.md). Contribution rules in [CLAUDE.md](CLAUDE.md). This file
> is the *review-doc generation loop* only.

---

## STEP 0 — Resolve output path + previous review (convention-following)

```bash
REPO=$(git rev-parse --show-toplevel)
DATE=$(date +%Y-%m-%d)
OUT="docs/architecture/${DATE}-architecture-review.md"
# Previous review to diff against (most recent existing review of this kind):
PREV=$(ls -1 docs/architecture/*review*.md 2>/dev/null | sort | tail -1)
echo "writing: $OUT   diffing-against: ${PREV:-<none>}"
```

Honour `--since <path>` to override the previous-review baseline.

---

## STEP 1 — Gather the grounding corpus (delegate heavy code reads)

The planner-style context budget applies: do not read every service file in this context.
Fan out up to **3 read-only `Explore` subagents** (parallel), each returning grounded findings
with `file:line` evidence. Suggested split:

| Subagent | Mines | Returns |
|----------|-------|---------|
| 1 — structure | `AGENTS.md`, `ARCHITECTURE.md`, `services/*/AGENTS.md`, `internal/{api,domain,infrastructure}` layout | three-layer/hexagonal adherence, layer-boundary violations, coupling, with file paths |
| 2 — decisions | `docs/adr/INDEX.md` + each ADR | which ADRs are honoured vs drifted; proposed-but-unimplemented; one-way-door integrity |
| 3 — quality | benchmarks/fuzz/load presence, error handling, timeouts/retries, observability hooks, dep graph | perf/reliability/scalability evidence + gaps |

The coordinator additionally reads live state itself (cheap):

```bash
sed -n '1,60p' state/current-milestone.md
gh api repos/:owner/:repo/milestones --jq '.[]|select(.state=="open")|"\(.title) open:\(.open_issues) closed:\(.closed_issues)"'
[ -n "$PREV" ] && sed -n '1,80p' "$PREV"        # prior scores + risk register to diff against
```

---

## STEP 2 — Synthesize the review (sections, mirroring the gold template)

Write `$OUT` with this structure (the proven shape of the 2026-05-20 review). Begin with the
SPDX header `<!-- SPDX-License-Identifier: Apache-2.0 -->` and use **repo-relative links** only.

1. **Header** — document type, date, reviewer-mandate, branch/HEAD reviewed.
2. **Executive summary** — the single most important finding; what is real vs narrative.
3. **Scorecard** — per-dimension 1–10 (Architecture, Simplicity, Performance, Security,
   Maintainability, Scalability, Reliability, Testing, CI/CD, Documentation, CNCF, PMF), **each
   with a one-line evidence rationale**. No bare numbers.
4. **Top strengths / Top weaknesses** — each citing `file:line` or ADR#.
5. **Per-dimension sections** — architecture, code quality, performance, security, scalability,
   reliability, testing, CI/CD, docs, CNCF fit (mirror the gold template's §2–§14).
6. **Risk register** — `ID | risk | P | I | mitigation | status`.
7. **Longitudinal delta vs `$PREV`** — a table: each prior risk/recommendation → `closed` /
   `partially-closed` / `open`, with the commit/PR/issue that closed it. (This is what makes the
   series valuable; cf. the post-M6 deltas in [docs/product/strategy.md](docs/product/strategy.md).)
8. **Prioritized recommendations** — Critical / High, mapped to issues where they exist.
9. **Gap analysis** — items not yet filed as issues (feed these to `/roadmap-plan`). Annotate
   each row with **user type(s)** (developer/operator/maintainer/product-owner/zynax-user/
   enterprise) and an **adoption lever** note, so `/roadmap-plan` files it with the right
   `product:` / `audience:` labels and a complete `## What for (user impact)` block.
10. **Appendix** — score card table + key file references.

> **Heavy-write tip.** If the corpus is large, draft each section from the subagents' returned
> findings; never paste raw code into the doc — cite it.

---

## STEP 3 — Deliver

Default: leave `$OUT` as an uncommitted working draft and print its path + a 10-line summary.

With `--pr`, open a `docs:` PR from an isolated worktree (never the user's checkout, never `main`):

```bash
RUN_ID="$(date +%s)-$$"; WT="/tmp/zynax-archreview-${RUN_ID}"
git -C "$REPO" worktree add "$WT" origin/main
# write $OUT inside $WT, then:
git -C "$WT" checkout -b "docs/architecture-review-${DATE}"
git -C "$WT" add "$OUT"
git -C "$WT" commit -s -F /tmp/archreview-msg-${RUN_ID}.txt   # docs(architecture): … + Assisted-by trailer
git -C "$WT" push -u origin "docs/architecture-review-${DATE}"
gh pr create --title "docs(architecture): architecture review ${DATE}" \
  --body "<exec summary + scorecard table + longitudinal delta>" --label "type: docs"
git -C "$REPO" worktree remove "$WT" --force 2>/dev/null || true
```

The review doc lands under `docs/` (PR-size-exempt, **not** CODEOWNERS-gated), so its PR can be
self-merged once CI is green — squash-only. Leave the merge to the human unless they say "go".

---

## Guardrails

- **Ground everything.** No score or claim without a cited artifact. Mark shipped/partial/aspirational.
- **Longitudinal, not amnesiac.** Always diff against the previous review; record what closed.
- **Read-only on the system under review.** Never edit code, ADRs, `AGENTS.md`, or `.feature` files.
- **One dated doc per run.** Don't overwrite a prior review — the series is the value.
- Commit (with `--pr`): `docs:` type, DCO `-s`, `Assisted-by: Claude/<model>`, squash-merge, signed; repo-relative links; no literal email; no skip-ci token.
- Pairs with `/roadmap-plan`: this review's **gap analysis** is exactly the input `/roadmap-plan` mines into issues.
