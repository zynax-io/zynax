---
description: Generate a dated market-fit review document grounded in the repo + live signals — category, competitive landscape (lead vs Kagent), TAM framing, persona/beachhead validation, adoption-funnel reality, and product-market-fit verdict — with recommendations classified by user type + adoption lever, tracked longitudinally. Read-only synthesis; writes one dated doc under docs/product/ and opens a docs: PR with --pr.
argument-hint: "[--pr] [--since <prev-review-path>]   default: working draft, no PR"
---

# /lib:market-fit-review — Market-Fit Review Document Generator

Produce a point-in-time **product-market-fit** assessment as a dated document under
[docs/product/](docs/product/), extending the competitive analysis in
[docs/architecture/2026-05-28-competitive-positioning.md](docs/architecture/2026-05-28-competitive-positioning.md)
and the market sections of [docs/product/strategy.md](docs/product/strategy.md).

> **Read-mostly.** Reads the repo + live GitHub signals, writes one dated Markdown file. `--pr`
> opens a `docs:` PR; otherwise a working draft. Never edits code/ADRs/`AGENTS.md`.

> **Truth-pass discipline.** Competitive claims cite the repo's own positioning docs; adoption
> numbers come live from `gh`. Mark shipped/partial/aspirational. No invented scores — reconcile
> any PMF score to the architect review's grounded basis.

> **Rules are not restated.** See `strategy.md`, the competitive-positioning doc, and
> [ROADMAP.md](ROADMAP.md). This file is the *review-doc generation loop* only.

---

## STEP 0 — Resolve output path + previous review

```bash
REPO=$(git rev-parse --show-toplevel); DATE=$(date +%Y-%m-%d)
OUT="docs/product/${DATE}-market-fit-review.md"
PREV=$(ls -1 docs/product/*market-fit-review.md 2>/dev/null | sort | tail -1)
echo "writing: $OUT   baseline: ${PREV:-docs/architecture/2026-05-28-competitive-positioning.md}"
```

Honour `--since <path>`.

---

## STEP 1 — Gather the grounding corpus

Coordinator pulls live adoption/market signals; a read-only `Explore` subagent mines the
positioning corpus (`strategy.md`, competitive-positioning, README, ROADMAP) for the current
category framing, the Kagent comparison, and the beachhead definition.

```bash
gh api repos/:owner/:repo --jq '{stars:.stargazers_count, forks:.forks_count, watchers:.subscribers_count}'
gh api repos/:owner/:repo/contributors --jq 'length'
# Optional external signal: maintainers may paste recent competitor/CNCF news for the subagent to weigh.
```

---

## STEP 2 — Synthesize the review (sections)

Write `$OUT` (SPDX header; repo-relative links):

1. **Header + executive summary** — the PMF verdict in one line; what changed since `$PREV`.
2. **Category** — the "control plane for AI agents" category in the current period; is it more/less crowded?
3. **Competitive landscape** — **lead with Zynax vs Kagent** (engine-agnostic vs K8s-locked;
   capability-routing vs Pod-per-agent; co-existence story), then Temporal/Dapr/Restate/LangGraph/
   Argo/Flyte-Kubeflow. Extend the matrix from the competitive-positioning doc with any deltas.
4. **TAM framing** — the three adjacent markets (agent orchestration / workflow engines / AI-ML
   platforms) and where Zynax's wedge sits.
5. **Personas & beachhead validation** — does agentic SW-engineering automation still hold as the
   beachhead? Evidence (shipped examples, demand signals).
6. **Adoption-funnel reality** — live metrics (stars/forks/contributors/adopters) vs the funnel
   targets in `strategy.md §7`. Honest baseline.
7. **PMF verdict + score** — reconciled to the architect review's grounded basis; no invented number.
8. **Recommendations (prioritized)** — each classified by **user type** + **adoption lever**, for
   the `/plan` handoff (`product:`/`audience:` labels + `## What for (user impact)` block).
9. **Longitudinal delta vs `$PREV`** — what moved (positioning, competitors, metrics).
10. **Appendix** — sources.

---

## STEP 3 — Deliver

Default: working draft + summary. With `--pr`, open a `docs:` PR from an isolated worktree (PR body
from [docs/contributing/pr-templates.md](docs/contributing/pr-templates.md), docs variant; DCO `-s`
+ `Assisted-by`; squash-only; repo-relative links; no literal email). Label `type: docs` +
`product: strategy`.

`docs/` is not CODEOWNERS-gated → self-mergeable on green CI (squash-only); leave merge to the human.

---

## Guardrails

- **Lead with Kagent.** The repo's own docs name it the direct competitor — never omit it (the
  cautionary precedent: an earlier informal analysis did, and was wrong).
- **Ground everything.** Cite positioning docs; pull adoption metrics live; no invented PMF score.
- **Longitudinal.** Diff against the prior market-fit review / competitive-positioning doc.
- **Recommendations carry user-type + adoption-lever annotations** (the `/plan` handoff).
- One dated doc per run; never overwrite a prior review.
- Pairs with `/lib:product-review` (internal product lens) and `/plan` (recommendations → issues).
