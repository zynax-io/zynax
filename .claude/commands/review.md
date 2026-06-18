---
description: Run the full review suite — architecture, security-posture, product-strategy, and market-fit reviews — in parallel read-only subagents, each producing its dated grounded document, then collect them into one docs: PR (or four, with --split). The single command to take a fresh, truthful read of where the project stands across engineering and product.
argument-hint: "[--pr] [--split] [--only architecture,security,product,market]   default: working drafts, one bundled PR with --pr"
---

# /review — Parallel Review Document Orchestrator

Fan out all four review-document generators at once, each in its own isolated subagent, and
collect their dated docs. Thin coordination layer: spawn → collect → deliver. The orchestrator
itself reads no code — each review's grounding work happens in its subagent's isolated context
(same discipline as `/deliver`).

```
/review
  ├─ /lib:architecture-review   → docs/architecture/<date>-architecture-review.md
  ├─ /lib:security-review-doc   → docs/architecture/<date>-security-review.md (+ .private.md, gitignored)
  ├─ /lib:product-review        → docs/product/<date>-product-strategy-review.md
  └─ /lib:market-fit-review     → docs/product/<date>-market-fit-review.md
```

> **Read-mostly.** Subagents read the repo + live GitHub state and return their doc CONTENT;
> the coordinator writes the files and (with `--pr`) opens the PR(s). Nothing edits code, ADRs,
> or `AGENTS.md`. Each review keeps its own truth-pass discipline (grounded claims, longitudinal
> delta, shipped/partial/aspirational split).

> **Rules are not restated.** Each review command (`/lib:architecture-review`, `/lib:security-review-doc`,
> `/lib:product-review`, `/lib:market-fit-review`) owns its own contract and guardrails. This file is the
> *parallel coordination loop* only.

---

## STEP 0 — Coordinator worktree + arg parse

```bash
RUN_ID="$(date +%s)-$$"
REPO=$(git rev-parse --show-toplevel)
COORD_WT="/tmp/zynax-reviewsuite-${RUN_ID}"
git -C "$REPO" worktree remove "$COORD_WT" --force 2>/dev/null || true
git -C "$REPO" fetch origin --prune
git -C "$REPO" worktree add "$COORD_WT" origin/main
cd "$COORD_WT"
DATE=$(date +%Y-%m-%d)
```

Parse `--only` (subset of `architecture,security,product,market`; default all four), `--pr`
(open PR after collecting), `--split` (one PR per review instead of one bundled PR).

---

## STEP 1 — Dispatch the review subagents in parallel (read-only)

Spawn one background `Agent` per selected review. Each runs that review's procedure end-to-end
against the **real checkout** (read-only) and **returns the finished Markdown** in its result —
it does NOT commit (the coordinator writes one branch to avoid four agents racing one tree).

For each selected review R with its command file F (e.g. `architecture` → `architecture-review.md`):

```
Agent({
  description: "<R> review document",
  subagent_type: "Explore",          // read-only; no edits
  run_in_background: true,
  prompt: """
    You are generating the <R> review document for Zynax. Follow this command's procedure
    exactly (STEP 0–2 — gather grounding corpus, synthesize), but DO NOT write a file, create a
    branch, or open a PR. Instead RETURN the complete review Markdown as your final message,
    starting with the SPDX header and using repo-relative links only.

    <full content of .claude/commands/<F>>

    Constraints:
    - Read-only. Ground every claim in a cited artifact (file:line / doc / gh result). No invented numbers.
    - Mark shipped / partial / aspirational. Include the longitudinal delta vs the prior review.
    - Annotate recommendations/gaps with user type(s) + adoption lever (for the /plan handoff).
    - Pull live signals (stars/forks/contributors/alerts/milestones) via gh yourself.
    - End with: a line `OUTPUT_PATH: <the dated path this review uses>` so the coordinator can place it.
  """
})
```

> **Why subagents return content rather than commit:** four agents committing to one branch
> collide; four separate worktrees + four PRs is noisier than one review snapshot. Returning
> content lets the coordinator assemble a single, reviewable PR. (`--split` overrides this — see
> STEP 3.) The security review's `.private.md`, if any, is returned separately and the coordinator
> writes it **only** if `.gitignore` covers it, and never stages it.

---

## STEP 2 — Collect results

As each subagent completes, extract its `OUTPUT_PATH` and Markdown body. Write each doc to its
path inside `$COORD_WT`. For the security review, write the public doc; write the `.private.md`
only if `git -C "$COORD_WT" check-ignore <priv>` succeeds, and never stage it.

```bash
# per returned review: write the file at its OUTPUT_PATH inside the coordinator worktree
# (the coordinator holds 4 large docs briefly — acceptable for a snapshot run)
```

Report any subagent that failed (with reason) and continue with the rest.

---

## STEP 3 — Deliver

**Default (no `--pr`):** leave all docs as working drafts in the coordinator worktree path and
print each `OUTPUT_PATH` + a 5-line summary per review, then clean up the worktree.

**`--pr` (bundled, default):** one `docs:` PR with all collected docs:

```bash
git -C "$COORD_WT" checkout -b "docs/review-suite-${DATE}"
git -C "$COORD_WT" add docs/architecture/${DATE}-*.md docs/product/${DATE}-*.md   # explicit paths; NEVER git add -A (private file)
git -C "$COORD_WT" commit -s -F /tmp/reviewsuite-msg-${RUN_ID}.txt   # docs: review suite <date> + Assisted-by
git -C "$COORD_WT" push -u origin "docs/review-suite-${DATE}"
gh pr create --title "docs: review suite — ${DATE} (architecture · security · product · market-fit)" \
  --body-file /tmp/reviewsuite-prbody-${RUN_ID}.md --label "type: docs" --label "product: strategy"
```

PR body is built from [docs/contributing/pr-templates.md](docs/contributing/pr-templates.md)
(docs variant) and links each review with its one-line headline verdict. DCO `-s` + `Assisted-by`;
squash-only; repo-relative links; no literal email; no skip-ci token.

**`--split`:** instead, have each subagent open its own `docs:` PR via its own worktree (the
standalone behaviour of each review command with `--pr`). Use when you want them reviewed/merged
independently.

`docs/` is PR-size-exempt and not CODEOWNERS-gated → self-mergeable on green CI (squash-only);
leave merge to the human unless they say "go".

```bash
cd "$REPO" && git worktree remove "$COORD_WT" --force 2>/dev/null || true
```

---

## STEP 4 — Report

```
=== Review Suite — <date> ===
| Review | Doc | Headline verdict | Top recommendation |
|--------|-----|------------------|--------------------|
| architecture | docs/architecture/<date>-architecture-review.md | <score / one-line> | <#1 rec> |
| security     | docs/architecture/<date>-security-review.md      | <posture>           | <#1 rec> |
| product      | docs/product/<date>-product-strategy-review.md   | <PMF / one-line>    | <#1 rec> |
| market-fit   | docs/product/<date>-market-fit-review.md         | <verdict>           | <#1 rec> |

PR: #<n> (bundled) or four PRs (--split).
Next: feed the reviews' gap-analyses/recommendations to /plan to file them as issues.
```

---

## Guardrails

- **Coordinator reads no code.** All grounding happens in subagents; if you find yourself reading
  a service file here, stop and let the subagent do it.
- **Never `git add -A`** in this command — it would stage a security `.private.md`. Stage explicit paths.
- **One dated doc per review per run.** Never overwrite a prior dated review.
- Each review keeps truth-pass discipline (grounded, longitudinal, shipped/partial/aspirational).
- Commit (with `--pr`): `docs:` type, DCO `-s`, `Assisted-by`, signed, squash-merge; repo-relative links.
- Pairs with `/plan`: the suite produces the assessments; `/plan` turns their
  gaps/recommendations into SPDD-filed, adoption-tagged issues for the active milestone.
