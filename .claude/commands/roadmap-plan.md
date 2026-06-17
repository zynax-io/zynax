---
description: SPDD-native roadmap planner — reads the whole repo (ROADMAP, architecture/security/market reviews, ADRs, AGENTS.md, experts, code, live GH issues/labels/milestones, canvases, ai-learnings) and infers the next stories + unresolved fixes for the ACTIVE milestone. Routes every feat: through the SPDD pipeline (analysis→story→canvas→security-review→align), cross-links canvas↔issue, then leaves the repo orchestrate-ready and as repo-clean as possible. Keeps filling the active milestone until its exit criteria are met, then PROPOSES advancing to the next milestone. PLAN by default; mutations gated behind --execute.
argument-hint: "[--execute] [--area <scope>] [--max N]   default: PLAN only"
---

# /roadmap-plan — SPDD-Native Roadmap Planner & Orchestration-Readiness Pass

Answer the question **/milestone-plan cannot**: *what stories and fixes SHOULD exist for the
active milestone that don't yet* — then create them the SPDD way and leave the repo ready for
`/milestone-orchestrate`.

```
/roadmap-plan  →  infer stories + fixes from the whole repo (this command)
/milestone-plan        →  sequence the EXISTING issues into parallel batches
/milestone-orchestrate →  deliver batches via expert subagents
/milestone-learn       →  synthesize session learnings into expert guides
/repo-clean            →  reconcile status surfaces to live state
/milestone-close + /milestone-new →  advance to the next milestone
```

`/milestone-plan` sequences issues that already exist. **`/roadmap-plan` is the step before
it** — it mines the repo for work that is implied but unfiled, files it correctly (SPDD for
`feat:`), aligns the active milestone's canvases to their EPICs and issues, and runs a
`/repo-clean` reconcile so the next `/milestone-orchestrate` starts from a true, complete state.

> **This command is PLAN-by-default and read-mostly.** It prints a traceable plan and **stops**.
> It mutates GitHub (creates issues, runs SPDD canvases/security-reviews, cross-links,
> reconciles) **only** with `--execute` or an explicit "go" after the plan. It never edits
> `AGENTS.md`, ADRs, or any `.claude/commands/**` file — refinements to those are *proposed*
> into the human-gated `/milestone-learn` + `/repo-clean` loop (STEP 6).

> **Rules are not restated.** Domain + contribution rules live in [AGENTS.md](AGENTS.md),
> [CLAUDE.md](CLAUDE.md), [docs/git-workflow.md](docs/git-workflow.md), and the SPDD guide
> [docs/patterns/spdd-guide.md](docs/patterns/spdd-guide.md). The SPDD `feat:` contract is
> [ADR-019](docs/adr/INDEX.md). This file is the *inference + readiness loop* only.

---

## Operating contract (read before doing anything)

- **Active milestone is the unit of work.** Read it from `state/milestone.yaml` (SSoT). Never
  hardcode a milestone name/number/label. This command plans **into the active milestone** and
  keeps filling it until its exit criteria are met (STEP 7).
- **Live GitHub state is the source of truth** — never memory, never a stale doc. Every decision
  is driven by `gh issue list` / `gh pr list` / `gh api .../milestones` and the files read fresh.
- **SPDD for every `feat:`.** Inferred *capabilities* are never filed as bare issues. They go
  through `/spdd-analysis` → `/spdd-story` → `/spdd-reasons-canvas` → `/spdd-security-review` →
  human aligns the canvas. `fix:` / `refactor:` / `docs:` / `test:` / `ci:` / `chore:` are
  SPDD-exempt and may be filed directly.
- **Two phases.** PLAN (default) computes and prints; EXECUTE (`--execute` or "go") mutates.
  Never create an issue, run an SPDD canvas, cross-link, or reconcile in PLAN phase.
- **Hard constraints (mirror of [AGENTS.md §Hard Constraints](AGENTS.md#hard-constraints) + repo memory):**
  commit type is one of `feat|fix|refactor|docs|test|ci|chore` (never invent one); every commit
  carries a DCO `Signed-off-by:` (your configured git identity — see
  [docs/git-workflow.md](docs/git-workflow.md)) **and** `Assisted-by: Claude/<model>` (never
  `Co-Authored-By` for AI); merge is **squash-only**; never disable signing; never push `main`
  directly; never write a literal `[skip ci]` token (write "skip-ci marker"). Use repo-relative
  paths in any committed markdown; never put a literal email in a `.claude/commands/**` file.
- **Never writes `state/milestone.yaml`.** Only `/milestone-close` and `/milestone-new` may.
  When the active milestone is complete, this command *proposes* the transition and hands off.

---

## Issue / story / canvas / PR shape (templates · What-for · tags)

Everything this command files follows the canonical shapes — no ad-hoc bodies.

- **Issue body = the matching template** in [.github/ISSUE_TEMPLATE/](.github/ISSUE_TEMPLATE/):
  `feature_request.md` (`feat:`), `bug_report.md` (`fix:`/bug), `documentation.md` (`docs:`),
  `adr_proposal.md` (proposed ADR). Fill every template section; never freehand.
- **PR body = [docs/contributing/pr-templates.md](docs/contributing/pr-templates.md)** (the
  per-type skeleton), exactly as `/milestone-orchestrate` and `/issue-deliver` build it.
- **Stories = `/spdd-story`** output (INVEST stories mapped to the Canvas O section) — never a
  bare issue for a `feat:` capability.
- **Every issue, story, canvas, and PR carries a `## What for (user impact)` block** — see below.

### The `## What for (user impact)` block (mandatory — paste into body + canvas)

This is the product lens, distinct from the engineering "Why". It states *who benefits and how*,
with the adoption emphasis this project lives or dies on:

```markdown
## What for (user impact)
- **User type(s):** developer | operator | maintainer | product-owner | zynax-user | enterprise
  (pick all that apply — these map to the `audience:` labels)
- **Expected impact:** <the observable outcome for that user — what they can now do / no longer suffer>
- **Adoption lever:** <how this attracts developers, lowers Day-0 friction, or grows the community
  — REQUIRED when `product: adoption` or `product: dx` applies; "N/A" only if genuinely none>
- **Real use case:** <the concrete scenario it unlocks, tied to a hero workflow / example where possible>
```

> **Adoption-first prioritization.** Per [docs/product/strategy.md](docs/product/strategy.md),
> the binding constraint is **traction, not features**. Candidates that attract developers, cut
> Day-0 friction, or enable a real use case are **promoted** in the plan (higher `priority:`,
> tagged `product: adoption` / `product: dx` / `product: use-case`) and surfaced in their own
> "Adoption-driving" group in the STEP 5 output.

### Label policy (every created issue)

Apply, in addition to `type:` / `area:` / `priority:` / `status:` / `milestone:`:

| Label group | When |
|-------------|------|
| `product: strategy` | **Every issue this command files** (it is strategy/roadmap-inferred) |
| `product: adoption` | Drives user adoption / onboarding / Day-0 experience |
| `product: dx` | Developer experience / attracting contributors |
| `product: use-case` | Enables a concrete real-world use case / hero workflow |
| `audience: <type>` | One per user type in the What-for block (developer/operator/maintainer/product-owner/zynax-user/enterprise) |

The full `product:` and `audience:` taxonomy lives in [docs/labels.md](docs/labels.md). In the
EXECUTE phase, ensure each label exists before applying it (`gh label create … 2>/dev/null || true`,
mirroring `/milestone-orchestrate` STEP 4).

---

## STEP 0 — Isolated worktree (leave the user's checkout untouched)

Run all git/inference work in a throwaway worktree detached at `origin/main`, exactly like
`/repo-clean` and `/milestone-orchestrate` do.

```bash
RUN_ID="$(date +%s)-$$"
REPO=$(git rev-parse --show-toplevel)
WT="/tmp/zynax-roadmap-${RUN_ID}"
git -C "$REPO" worktree remove "$WT" --force 2>/dev/null || true
rm -rf "$WT" 2>/dev/null || true
git -C "$REPO" fetch origin --prune
git -C "$REPO" worktree add "$WT" origin/main
cd "$WT"
```

Parse args: `--execute` (skip the approval gate), `--area <scope>` (restrict inference to one
service/EPIC scope), `--max N` (cap proposed stories per run; default 8 to keep batches reviewable).

---

## STEP 1 — Load active-milestone config + planning state

```bash
# ── Active-milestone config (SSoT: state/milestone.yaml) — loaded at runtime ──
CFG=state/milestone.yaml
MILESTONE_NAME=$(awk '/^active:/{f=1} f && /^  name:/{print $2; exit}' "$CFG")
MILESTONE_TITLE=$(awk -F'"' '/^active:/{f=1} f && /^  title:/{print $2; exit}' "$CFG")
MILESTONE_NUMBER=$(awk '/^active:/{f=1} f && /^  github_milestone_number:/{print $2; exit}' "$CFG")
MILESTONE_VERSION=$(awk '/^active:/{f=1} f && /^  version:/{print $2; exit}' "$CFG")
PLANNING_DOC=$(awk '/^active:/{f=1} f && /^  planning_doc:/{print $2; exit}' "$CFG")
MILESTONE_LABEL=$(awk -F'"' '/^    milestone:/{print $2; exit}' "$CFG")
GH_MILESTONE="${MILESTONE_TITLE} (${MILESTONE_NAME})"
# ──────────────────────────────────────────────────────────────────────────────

cat state/current-milestone.md      # active blockers, exit criteria, per-EPIC status
cat "$PLANNING_DOC"                  # EPIC table + dependency table for the active milestone
sed -n '1,200p' CLAUDE.md           # per-milestone scope table (In scope / Out of scope)
```

Snapshot live GitHub state (the dedupe + routing baseline):

```bash
OPEN=$(gh issue list  --state open   --limit 400 --json number,title,body,labels,milestone)
CLOSED=$(gh issue list --state closed --limit 400 --json number,title,labels,milestone)
gh api repos/:owner/:repo/milestones --jq '.[] | "\(.title)\t#\(.number)\topen:\(.open_issues)\tclosed:\(.closed_issues)\tstate:\(.state)"'
```

---

## STEP 2 — Mine the inference corpus (delegate heavy reads to Explore subagents)

The "read the whole repo" job blows the planner's context. Fan out **read-only `Explore`
subagents** (max 3, in parallel), each mining one corpus and returning a structured candidate
list: `kind | title | evidence (file:line or doc#) | suggested type | suggested EPIC/milestone`.
The planner stays lean and only merges + dedupes the returned lists.

| Source | What to extract → candidate |
|--------|------------------------------|
| `docs/architecture/*-review.md`, `docs/reviews/*.md` | **Risk registers** (R1…Rn) and **gap analyses** (e.g. the principal review's §16 G1–G24 "unplanned-gap" list) — every row not yet a GH issue is a candidate |
| `docs/adr/INDEX.md` + ADRs `Status: Proposed`/`🟡` | An accepted-but-unimplemented or proposed ADR → an implementation candidate (route to SPDD if it adds capability) |
| `ROADMAP.md`, `state/current-milestone.md`, `$PLANNING_DOC` | Active-milestone EPICs with **no story children**, "blockers", deferred/"M?+" items, unmet exit criteria |
| `docs/product/strategy.md` + review docs | Product/strategy **recommendations** not yet filed (e.g. recommended features, Day-0 friction cuts) |
| Code markers | `grep -rnE 'TODO|FIXME|HACK|XXX' --include='*.go' --include='*.py'`; `t.Skip`/`xfail`/`pytest.mark.skip`; `Unimplemented`/`not yet implemented` comments — each is a candidate `fix:`/`feat:` |
| `docs/spdd/*/canvas.md` | Canvases `Status: Draft`/unaligned, or whose EPIC has open O-steps with no issue |
| `docs/ai-learnings/*.md` + open `[AUTO]` families | **Recurring** pain (≥2 sessions / repeated AUTO issue) → a systemic fix **or a drift-prevention guardrail** (STEP 6) |
| `AGENTS.md`, `services/*/AGENTS.md`, `.claude/commands/experts/*` | Documented known limitations / "defer" notes that imply unfiled work |

Example fan-out (one of up to three):

```
Agent({
  description: "Mine review/ADR risk+gap registers for unfiled work",
  subagent_type: "Explore",
  prompt: """
    Read every docs/architecture/*-review.md, docs/reviews/*.md, and docs/adr/INDEX.md in
    <REPO>. Extract every risk-register row, recommendation, and 'gap not yet filed' item.
    For each, report: a one-line title, the source (file:line), a suggested conventional-commit
    type (feat|fix|refactor|docs|test|ci|chore), and whether it adds a *capability* (→ SPDD feat)
    or is a fix/hardening. Do NOT cross-check GitHub — just return the candidate list with
    evidence. Read only; edit nothing.
  """
})
```

---

## STEP 3 — Dedupe, classify, and route each candidate

For every merged candidate:

1. **Dedupe vs live GH.** Drop it if an open or closed issue already covers it (fuzzy-match
   title + scope against `$OPEN`/`$CLOSED`). Cite the existing issue in the plan instead.
2. **Type it.** Assign one of the 7 commit types. A *new capability* ⇒ `feat:` ⇒ **SPDD path**.
3. **Route to a milestone.** Match the candidate's scope to the per-milestone scope table in
   [CLAUDE.md](CLAUDE.md) + `ROADMAP.md`:
   - In active-milestone scope → the **active milestone** (`$GH_MILESTONE`).
   - Clearly a later milestone → label for that milestone (or leave milestone-less as backlog
     with a note). Never stuff out-of-scope work into the active milestone to pad it.
4. **Attach to an EPIC.** If the active milestone organises work under EPICs, map each story to
   its parent EPIC (so `/milestone-plan` can sequence it). A story with no EPIC for a milestone
   that uses EPICs is itself a finding (propose an EPIC or a "misc" bucket).

---

## STEP 4 — Active-milestone canvas ⇄ issue ⇄ security-review audit

This is the orchestration-readiness core. For **each EPIC of the active milestone** (from
`$PLANNING_DOC` / `open_epics` in `state/milestone.yaml`), verify the full SPDD linkage and
record every gap:

```bash
for CANVAS in docs/spdd/*/canvas.md; do
  EPIC_N=$(basename "$(dirname "$CANVAS")" | grep -oP '^\d+')
  # Status may be its own line OR inline in the header (e.g. "**Author:** … · **Status:** Aligned").
  STATUS=$(grep -m1 -oiE 'Status:[^|·]*' "$CANVAS" 2>/dev/null)
  echo "EPIC #$EPIC_N  canvas=$CANVAS  ${STATUS:-<no status>}"
done
```

| Readiness check (per active-milestone EPIC) | Gap → action |
|---|---|
| A canvas exists at `docs/spdd/<N>-<slug>/canvas.md` | **missing** → `/spdd-reasons-canvas <N>` (EXECUTE phase) |
| Canvas `Status:` reflects reality (`Draft`→`Aligned`→`Implemented`) | **stale** → fold into the `/repo-clean` reconcile (STEP 6) |
| Canvas **links to its GH issue** (`#<N>` in the canvas header) | **missing** → add the issue ref to the canvas |
| GH issue **links back to the canvas** (`docs/spdd/<N>-…/canvas.md` in the issue body) | **missing** → `gh issue edit <N>` to add the canvas link |
| A `docs/spdd/<N>-<slug>/SECURITY-REVIEW.md` exists and **PASSES** (feat EPICs) | **missing/FAIL** → `/spdd-security-review <canvas>` then resolve before align |
| Canvas is `Aligned` before any implementation issue is `READY` | **not aligned** → human sets `Aligned` after security-review PASS |
| Every EPIC O-step has a story issue | **missing** → `/spdd-story <N>` to decompose |

The output of this step is the precise SPDD action list that makes the active milestone
orchestrate-ready: which EPICs need a canvas, which need stories, which need a security-review,
which need cross-links, which need an align.

---

## STEP 5 — PLAN output (then STOP unless --execute)

Print one traceable plan and wait for "go" (unless `--execute`).

```
## /roadmap-plan — <date>, active milestone <NAME> "<TITLE>" (#<num>, <version>)

### Milestone fill status
  EPICs: <done>/<total> complete · open EPICs without stories: #…, #…
  Exit criteria unmet: <list from $PLANNING_DOC / current-milestone.md>
  Verdict: FILLING (work remains) | COMPLETE (propose advance — STEP 7)

### Proposed stories (SPDD — feat:) — into <NAME>
| candidate | EPIC | SPDD actions needed | evidence |
|-----------|------|---------------------|----------|
| feat(engine-adapter): … | #1167 | /spdd-story → /spdd-reasons-canvas → /spdd-security-review → align | review §16 G8 |

### Proposed issues (SPDD-exempt — fix/refactor/docs/test/ci/chore) — into <NAME>
| # (new) | type(scope): title | evidence | labels |
|---------|--------------------|----------|--------|
| — | fix(api-gateway): set ReadHeaderTimeout | review §7 G2 / auth.go:NN | type: bug, milestone: <NAME> |

### Adoption-driving candidates (PROMOTED — product: adoption / dx / use-case)
| candidate | user type(s) | adoption lever | real use case | labels |
|-----------|--------------|----------------|---------------|--------|
(These are surfaced first and priority-bumped — traction is the binding constraint.)

### Canvas ⇄ issue ⇄ security-review gaps (orchestration-readiness)
| EPIC | canvas | issue link | back-link | security-review | align | action |
|------|--------|-----------|-----------|-----------------|-------|--------|

### Repo-clean delta (run after the above)
  <surfaces that will disagree once issues are created — handed to /repo-clean>

### Drift-prevention proposals (propose-only → APPLY_LOG.md / guardrails)
| pattern | recurrence | proposed guardrail (CI gate / command-step / expert rule) |
|---------|-----------|-----------------------------------------------------------|

### Out-of-active-milestone candidates (backlog / later milestone — NOT created into <NAME>)
| candidate | suggested milestone | why deferred |

### Handoff
  After --execute: run /milestone-plan, then /milestone-orchestrate.
```

---

## STEP 6 — EXECUTE (on approval / --execute)

Perform the mutations **in dependency order**. SPDD before issues-that-depend-on-canvas;
cross-link before align; reconcile last.

1. **SPDD-exempt issues** — create directly, into the active milestone. Body = the matching
   `.github/ISSUE_TEMPLATE/*` filled in (including the `## What for (user impact)` block).
   Ensure the product/audience labels exist, then apply them alongside the standard groups:
   ```bash
   for L in "product: strategy" "product: adoption" "product: dx" "product: use-case" \
            "audience: developer" "audience: operator" "audience: maintainer" \
            "audience: product-owner" "audience: zynax-user" "audience: enterprise"; do
     gh label create "$L" 2>/dev/null || true     # idempotent; colours/desc in docs/labels.md
   done
   gh issue create --title "fix(api-gateway): set ReadHeaderTimeout (Slowloris)" \
     --body-file /tmp/issue-body-${RUN_ID}.md \
     --milestone "$GH_MILESTONE" \
     --label "type: bug" --label "$MILESTONE_LABEL" \
     --label "product: strategy" --label "audience: operator"   # + any other applicable product:/audience:
   ```
2. **feat: stories — SPDD path only.** For each feat EPIC/story gap, invoke the SPDD skills
   (against the real checkout, not this `/tmp` worktree — they manage their own git):
   ```
   /spdd-analysis <issue-or-epic>      # research, risk table, Tier-2 flags
   /spdd-story <epic>                  # decompose into INVEST stories (→ Canvas O section)
   /spdd-reasons-canvas <epic>         # create docs/spdd/<N>-<slug>/canvas.md (Status: Draft)
   /spdd-security-review <canvas>      # Tier-2 scan — MUST PASS before align
   ```
   Then **stop for the human to set `Status: Aligned`** — `/spdd-generate` refuses an unaligned
   canvas by design, and alignment is a human decision (CLAUDE.md SPDD rule).
3. **Cross-link canvas ⇄ issue** for every gap from STEP 4:
   ```bash
   # issue → canvas
   gh issue edit <N> --body "<existing body>

   SPDD canvas: docs/spdd/<N>-<slug>/canvas.md"
   # canvas → issue: add the `Issue: #<N>` line to the canvas header (edit + commit in the PR below)
   ```
4. **Security-review** any feat canvas missing a PASSing `SECURITY-REVIEW.md` (step 2's
   `/spdd-security-review`); resolve findings before the canvas is aligned.
5. **Drift-prevention + command/expert refinements — propose only.** Write each recurring-pattern
   guardrail and any milestone-command refinement as a **PENDING** row in
   [docs/ai-learnings/APPLY_LOG.md](docs/ai-learnings/APPLY_LOG.md) (category `domain` /
   `env-constraint`), and surface it in the report. **Never** auto-edit `experts/*`,
   `milestone-*.md`, `AGENTS.md`, or ADRs — the human applies via `/milestone-learn --apply`
   (expert files) or a deliberate command-file PR (CODEOWNERS-gated). This is how `/repo-clean`'s
   reconcile rules migrate into CI gates / command guardrails over time, so drift stops at the
   source and `/repo-clean` is eventually needed only after big context-losing crashes.
6. **Reconcile (`/repo-clean`).** Once issues exist and canvases are linked, the status surfaces
   are stale by construction. Run `/repo-clean` to bring README/ROADMAP/ARCHITECTURE/CLAUDE/
   state/planning/canvas surfaces back to live state and dedup-triage any `[AUTO]` noise — leaving
   the repo "as repo-clean as possible". Let `/repo-clean` own its own PR; don't fold it here.
7. **Commit canvas/back-link edits** made in this worktree as one `docs:`/`chore:` PR (DCO `-s` +
   `Assisted-by`), squash-merge on the human's call.

---

## STEP 7 — Milestone-completion gate & advance proposal

After the fill pass, re-assess the active milestone against its exit criteria
(`$PLANNING_DOC` + `state/current-milestone.md`):

- **FILLING** (open EPICs, unmet exit criteria, or stories just created): report the
  orchestrate-ready batch and recommend `/milestone-plan` → `/milestone-orchestrate`. Re-run
  `/roadmap-plan` after the next delivery wave to keep filling.
- **COMPLETE** (all EPIC stories merged, exit criteria met, `gh` milestone has 0 open issues):
  do **not** create filler. **Propose the transition** (never execute it here):
  ```
  ## Active milestone <NAME> appears COMPLETE
    - Exit criteria: all met (cite each).
    - Recommended: /milestone-close   (tag <version>, GitHub Release, rotate milestone.yaml)
    - Then:        /milestone-new      (scaffold the next milestone + planning doc + active block)
    - Next milestone candidates inferred from ROADMAP/backlog: <list with rationale>
  ```
  `/milestone-close` and `/milestone-new` are the **only** sanctioned writers of
  `state/milestone.yaml`. `/roadmap-plan` stops at the proposal.

---

## STEP 8 — Verify & report, then clean up

```bash
# Issues created land in the active milestone with the milestone label:
gh issue list --milestone "$GH_MILESTONE" --state open --json number,title,labels --jq 'length'
# Canvas linkage spot-check:
for c in docs/spdd/*/canvas.md; do grep -Hm1 -E 'Issue:|#[0-9]+' "$c"; done
```

Report: stories created (via SPDD) + issues created (direct), with numbers; canvas/issue
cross-links added; security-reviews run + status; drift/refinement proposals written to
APPLY_LOG (count, PENDING); the `/repo-clean` result; and the milestone verdict (FILLING with
the next orchestrate batch, or COMPLETE with the advance proposal).

```bash
cd "$REPO" && git worktree remove "$WT" --force 2>/dev/null || true
```

---

## Guardrails

- **PLAN first, always.** No GitHub or repo mutation without `--execute` / explicit "go".
- **SPDD is mandatory for `feat:`.** Never file a capability as a bare issue; never skip the
  canvas → security-review → align gate. Alignment is the human's call.
- **Dedupe is mandatory.** Never create an issue that duplicates an open/closed one — cite the
  existing one instead.
- **Never** auto-edit `AGENTS.md`, ADRs, `.claude/commands/**` (experts or milestone-*), or
  `state/milestone.yaml`. Propose; the human applies through the sanctioned command.
- **Never** push `main`, bypass signing/DCO, invent a commit type, or write a literal skip-ci token.
- **Respect milestone scope.** Don't pad the active milestone with out-of-scope work to avoid
  declaring it complete — surface those as backlog/later-milestone candidates.
- If `gh`/Docker is unavailable, fall back to PLAN-only and say so — never guess live state.

---

## Lifecycle / integration map

| Stage | Command | This command's role |
|-------|---------|---------------------|
| Infer unfiled work | **`/roadmap-plan`** | mine repo → SPDD-file stories + fixes into active milestone |
| Make orchestrate-ready | **`/roadmap-plan`** STEP 4–6 | canvas⇄issue⇄security-review align + `/repo-clean` |
| Sequence existing issues | `/milestone-plan` | (downstream) parallel batches |
| Deliver | `/milestone-orchestrate`, `/issue-deliver` | (downstream) |
| Synthesize learnings | `/milestone-learn` | this command *proposes* refinements into its APPLY_LOG |
| Reconcile surfaces | `/repo-clean` | invoked in STEP 6; this command feeds it guardrail proposals to make it eventually unnecessary |
| Advance milestone | `/milestone-close` + `/milestone-new` | this command *proposes* the transition in STEP 7; never executes it |
