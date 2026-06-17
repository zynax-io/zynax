---
description: Generate a dated product-strategy review document grounded in the live repo — positioning, real-vs-aspirational capability split, adoption funnel status, beachhead validation, and prioritized recommendations classified by user type + adoption lever — tracked longitudinally against the previous product review / docs/product/strategy.md. Read-only synthesis; writes one dated doc under docs/product/ and opens a docs: PR with --pr.
argument-hint: "[--pr] [--since <prev-review-path>]   default: working draft, no PR"
---

# /product-review — Product Strategy Review Document Generator

Produce a point-in-time product-strategy review as a **dated document** under
[docs/product/](docs/product/), building on the living
[docs/product/strategy.md](docs/product/strategy.md) and the competitive/architecture reviews.
Longitudinal: it diffs against the prior product review (and `strategy.md`) and records what
moved — adoption metrics, positioning, recommendations closed.

> **Read-mostly.** Reads the repo + live GitHub state, writes one dated Markdown file. With
> `--pr` opens a `docs:` PR; otherwise leaves a working draft. Never edits code/ADRs/`AGENTS.md`.

> **Truth-pass discipline (non-negotiable).** Every claim is **grounded in a cited artifact**
> (`file:line`, doc, issue#, or `gh` result). No invented numbers — the draft that became
> `strategy.md` was hardened precisely by removing ungrounded scores. Separate **shipped /
> partial / aspirational** explicitly.

> **Rules are not restated.** See [AGENTS.md](AGENTS.md), [CLAUDE.md](CLAUDE.md), and
> `strategy.md` for the canonical positioning. This file is the *review-doc generation loop* only.

---

## STEP 0 — Resolve output path + previous review

```bash
REPO=$(git rev-parse --show-toplevel); DATE=$(date +%Y-%m-%d)
OUT="docs/product/${DATE}-product-strategy-review.md"
PREV=$(ls -1 docs/product/*review*.md 2>/dev/null | sort | tail -1)   # else diff vs strategy.md
echo "writing: $OUT   baseline: ${PREV:-docs/product/strategy.md}"
```

Honour `--since <path>` to override the baseline.

---

## STEP 1 — Gather the grounding corpus (delegate heavy reads)

Fan out up to 3 read-only `Explore` subagents (parallel), returning grounded findings:

| Subagent | Mines | Returns |
|----------|-------|---------|
| 1 — positioning | `README.md`, `ROADMAP.md`, `docs/product/strategy.md`, `docs/architecture/*positioning*` | current tagline, value prop, real-vs-aspirational split (with refs) |
| 2 — delivery vs narrative | `state/current-milestone.md`, `docs/milestones/*`, ADR statuses, shipped capabilities | what is actually shipped this milestone vs claimed |
| 3 — adoption signals | (coordinator does this via `gh`) | stars/forks/discussions, external adopters, contributor count |

Coordinator reads cheap live signals itself:

```bash
gh api repos/:owner/:repo --jq '{stars:.stargazers_count, forks:.forks_count, watchers:.subscribers_count, open_issues:.open_issues_count}'
gh api repos/:owner/:repo/contributors --jq 'length'
gh api repos/:owner/:repo/milestones --jq '.[]|select(.state=="open")|"\(.title) open:\(.open_issues) closed:\(.closed_issues)"'
sed -n '1,60p' state/current-milestone.md
```

---

## STEP 2 — Synthesize the review (sections)

Write `$OUT` (SPDX header first; repo-relative links only):

1. **Header** — type, date, baseline diffed against.
2. **Executive summary** — the single most important product finding this period.
3. **Positioning check** — is the messaging still differentiated (esp. vs Kagent)? Drift from `strategy.md`?
4. **Real vs aspirational** — shipped / partial / aspirational table, reconciled to milestone state.
5. **Adoption funnel** — time-to-first-workflow, stars/forks/discussions, external adopters,
   contributors — **the numbers from STEP 1**, each with its source. Compare to the baseline.
6. **Beachhead validation** — is the hero use case (agentic SW-engineering automation) holding?
   Evidence from shipped examples / real workflows.
7. **Recommendations (prioritized)** — each classified by **user type** (developer/operator/
   maintainer/product-owner/zynax-user/enterprise) and **adoption lever**, so `/roadmap-plan` can
   file them with the right `product:` / `audience:` labels + `## What for (user impact)` block.
8. **Longitudinal delta** — prior recommendations → closed / open, with the PR/issue that moved them.
9. **Appendix** — sources + glossary.

---

## STEP 3 — Deliver

Default: leave `$OUT` as a working draft + print a summary. With `--pr`, open a `docs:` PR from an
isolated worktree (PR body built from [docs/contributing/pr-templates.md](docs/contributing/pr-templates.md),
docs-emphasis variant; DCO `-s` + `Assisted-by`; squash-only; repo-relative links; no literal email):

```bash
RUN_ID="$(date +%s)-$$"; WT="/tmp/zynax-prodreview-${RUN_ID}"
git -C "$REPO" worktree add "$WT" origin/main
# write $OUT inside $WT, then commit/push from $WT and:
gh pr create --title "docs(product): product strategy review ${DATE}" \
  --body-file /tmp/prodreview-prbody-${RUN_ID}.md --label "type: docs" --label "product: strategy"
git -C "$REPO" worktree remove "$WT" --force 2>/dev/null || true
```

`docs/` is PR-size-exempt and not CODEOWNERS-gated → self-mergeable once CI is green (squash-only);
leave merge to the human unless they say "go".

---

## Guardrails

- **Ground everything; no invented numbers.** Mark shipped/partial/aspirational. Pull adoption
  metrics live from `gh`, not memory.
- **Longitudinal.** Always diff against the prior review / `strategy.md`; record what moved.
- **Recommendations carry user-type + adoption-lever annotations** (the `/roadmap-plan` handoff).
- **Read-only on the product under review.** One dated doc per run; never overwrite a prior review.
- Commit (with `--pr`): `docs:` type, DCO `-s`, `Assisted-by`, signed, squash-merge.
- Pairs with `/roadmap-plan` (recommendations → issues) and `/market-fit-review` (competitive/TAM depth).
