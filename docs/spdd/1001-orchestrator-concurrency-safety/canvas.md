<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — Concurrency-Safe M6 Orchestrator (Worktree Isolation + Idempotent Dispatch)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content belongs in `canvas.private.md` (gitignored). None was identified for this feature.

**Issue:** #1001
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-09
**Status:** Aligned

**Child issues:** #1002 (step 1 — worktree isolation) · #1003 (step 2 — idempotent dispatch) · #1004 (step 3 — learning-loop classification) · #1005 (step 4 — expert-guide cleanup)

---

## R — Requirements

**Problem.** The `/m6-orchestrate` CLI command dispatches multiple domain-expert subagents **in parallel into a single shared git working tree**. Because the agents continuously switch branches in that shared tree, one agent silently corrupts another's branch, staging area, or commit. The incident record (`docs/ai-learnings/*.md`) traces a recurring failure family to this single structural cause (RC1): branch-state chaos (#818, #819, #826, #827, #828), `git commit` ref-lock (#796), the pre-commit linter running on ALL files (#795), uncommitted work wiped by a sibling's `git checkout main` (#879), `git add .` pollution (#817, #860), commits landing on the wrong branch requiring cherry-pick rescue (#867, #819, #828), and stash cross-contamination (#818, #795, #584). A second, distinct cause (RC2, #879) is that the orchestrator dispatches from a once-read snapshot and never reconciles, so the same issue can ship **two pull requests** across concurrent sessions. The current coping mechanism is to embed defensive commands in the expert prompts ("always `git checkout` first", "never `git add .`", "cherry-pick to rescue") — bandages that cannot make two agents sharing one tree safe.

**Definition of done — observable outcomes:**
- Two `/m6-orchestrate` runs on overlapping batches on one machine produce **zero** cross-agent branch contamination and **zero** duplicate PRs.
- A crashed agent's leaked worktree is reclaimed by the next run **without** touching a concurrent run's live worktrees (run-scoped sweep).
- The user's primary checkout is **never** mutated by an orchestrator run (coordinator worktree).
- `experts/go-services.md` and `experts/post-merge.md` contain **no** shared-tree workarounds; all RC3 (GitHub/git-server) mitigations remain intact.
- `/m6-learn` classifies each proposal `domain` / `structural-workaround` and suppresses the latter by default.
- Every PR passes CI: title ≤72 chars, DCO `Signed-off-by`, `Assisted-by` trailer, signed commit.

---

## E — Entities

- **`m6-orchestrate.md`** — CLI parallel delivery orchestrator; the primary subject. Reads planning state, claims issues, fans out expert subagents, collects results, dispatches post-merge verifiers.
- **`m6-issue-generate.md`** — single-issue autonomous delivery; already isolates work in a worktree at STEP 2.5 (`/tmp/zynax-auto-<N>`). The **reference pattern** this feature generalizes.
- **`m6-learn.md`** — learning synthesizer; promotes session patterns into expert guides through a human-gated apply-log.
- **Expert guides** (`experts/go-services.md`, `experts/post-merge.md`) — per-domain subagent prompts that currently embed RC1 workarounds.
- **`RUN_ID`** — a per-invocation orchestrator identifier that namespaces all worktrees and the crash-recovery sweep.
- **Isolated worktree** — a private git checkout off `origin/main` with its own `HEAD` and index: `/tmp/zynax-orch-<RUN_ID>-<N>` (domain), `/tmp/zynax-postmerge-<RUN_ID>-<PR_N>` (post-merge), `/tmp/zynax-orch-coord-<RUN_ID>` (orchestrator's own git ops).
- **Deterministic claim key** — the branch ref `<type>/<N>`, a pure function of the issue number, applied **before** any slug; the single mutex across both entry points.
- **Reconciliation state** — `SEEN_ISSUES` / `SEEN_MERGE_SHAS`, the orchestrator's in-context idempotency keys.
- **`Category` field** — `domain | structural-workaround`, attached to every Session-Learnings proposal and the apply-log.

Relationship sketch:

```
/m6-orchestrate (RUN_ID)
  ├─ coordinator worktree  /tmp/zynax-orch-coord-<RUN_ID>     ← orchestrator's own git ops (STEP 2/8)
  ├─ domain subagent  →  /tmp/zynax-orch-<RUN_ID>-<N>         ← private; branch <type>/<N>
  │     └─ claim key <type>/<N> = the mutex (atomic empty-branch push)
  ├─ STEP 5 reconcile (pre-spawn)   ─┐
  ├─ STEP 7 merge-SHA dedupe         ├─ idempotency (defense-in-depth; completion = authoritative)
  └─ post-merge subagent → /tmp/zynax-postmerge-<RUN_ID>-<PR_N>
```

---

## A — Approach

**What we WILL do:**
- Generalize `m6-issue-generate` STEP 2.5's proven worktree pattern to **every** dispatch path in `m6-orchestrate`, parameterized by `RUN_ID` so isolation is concurrency-safe and crash recovery never crosses runs.
- Run the orchestrator's own git mutations (STEP 2 quick-merge, STEP 8 learnings PR) in a coordinator worktree, leaving the user's checkout untouched.
- Make the branch the deterministic claim key `<type>/<N>` (slug applied post-claim) so the existing atomic empty-branch push becomes a true mutex.
- Add **layered** idempotency: dispatch-time reconcile (cheap early-out) + completion-time merge-SHA dedupe (authoritative — operates on a merge fact, immune to GitHub API lag).
- Teach `/m6-learn` to classify and suppress structural-workaround proposals, then remove the now-obsolete workarounds from the expert guides (sequenced after isolation is validated).

**What we WON'T do:**
- Touch any production code (`services/`, `agents/`, `protos/`), ADRs, `AGENTS.md`, or `CLAUDE.md`.
- Change the merge strategy or push-to-main policy (governed by ADR-023).
- Remove the RC3 mitigations (unsigned `update-branch`, API-lag claim-check, rebase-skip verification, report≠push) — no structural fix exists; they stay.
- Introduce any shared-state file between concurrent subagents (would reintroduce RC1).
- Alter the `/m6-learn` human-review gate or the orchestrator's context-budget discipline.

**ADR references:**
- **ADR-018** (AI KB authorization model) — `/.claude/` is a restricted KB path; edits need maintainer approval and must avoid prompt-injection (T4) and PII (T2).
- **ADR-019** (SPDD) — this `feat:` EPIC requires this Canvas Aligned before implementation.
- **ADR-023** (rebase-merge only / no direct push) — merge strategy and push policy are out of scope and must be preserved.
- **ADR-017** (GOWORK=off) — a worktree is a normal repo checkout; the `GOWORK=off` prefix is preserved inside it.

---

## S — Structure

Tooling/automation layer only — Markdown prompt files under `.claude/commands/`. **No** gRPC contracts, proto messages, service code, or databases are touched. No Layer 1/2/3 service boundary is crossed.

| File | Change | Story |
|------|--------|-------|
| `.claude/commands/m6-orchestrate.md` | STEP 1 `RUN_ID`; STEP 6 domain isolation + Session-Learnings `Category` template; STEP 7.5 post-merge isolation; STEP 7 run-scoped + age-based sweep; STEP 2/8 coordinator worktree | #1002 |
| `.claude/commands/m6-orchestrate.md` | deterministic claim key (STEP 5/6); STEP 5 pre-spawn reconcile; STEP 7 merge-SHA dedupe + loser-PR closure | #1003 |
| `.claude/commands/m6-issue-generate.md` | branch-derivation parity for the `<type>/<N>` claim key | #1003 |
| `.claude/commands/m6-learn.md` | STEP 4 classification rules; apply-log `Category` column | #1004 |
| `.claude/commands/experts/go-services.md` | remove RC1 workarounds; keep RC3 rule | #1005 |
| `.claude/commands/experts/post-merge.md` | remove Phase 6 branch-prep dance / shared-tree phrasing | #1005 |

**Out of structural scope (do not conflate):** `automation/orchestrator/aggregation-protocol.md` is the GHA-runtime advisory orchestrator (PR review; never merges or pushes) — a different concern, untouched.

---

## O — Operations

Each step = one PR.

1. ✅ **Worktree isolation across all dispatch paths (#1002, `feat`).** Add `RUN_ID` in STEP 1. In the STEP 6 domain-expert dispatch prompt and the STEP 7.5 post-merge dispatch prompt, create a per-run worktree (`/tmp/zynax-orch-<RUN_ID>-<N>`, `/tmp/zynax-postmerge-<RUN_ID>-<PR_N>`) off `origin/main` as the agent's first action and remove it as the last; drop the defensive branch-recheckout / "never `git add .`" guidance from those prompts; add the `## Session Learnings` template carrying the `Category` field. Run the orchestrator's own git ops (STEP 2/8) in `/tmp/zynax-orch-coord-<RUN_ID>`. Replace the STEP 7 sweep with a run-scoped reclaim (`…-<RUN_ID>-*`) + `git worktree prune` + an age-based stale sweep.

2. **Idempotent dispatch (#1003, `feat`).** Make the branch ref `<type>/<N>` the deterministic claim key (slug applied post-claim) identically in `m6-orchestrate` and `m6-issue-generate`. Add the STEP 5 pre-spawn reconcile (`gh issue view` + merged-PR search). Add STEP 7 completion-time dedupe keyed on `SEEN_MERGE_SHAS`, closing a loser PR when a different PR already delivered the issue. Document the three layers as defense-in-depth with completion-time authoritative.

3. ✅ **Learning-loop classification (#1004, `chore`).** In `m6-learn` STEP 4, add the two synthesizer rules (classify `domain` / `structural-workaround`; suppress structural by default under a separate heading, defaulting Status to `rejected`). Add the `Category` column to the apply-log table. Leave the human-review gate unchanged.

4. **Expert-guide cleanup (#1005, `chore`) — after #1002 is merged and validated.** Remove the obsolete shared-tree workarounds from `experts/go-services.md` ("Shared workspace safety" branch/stash/`git add` rules) and `experts/post-merge.md` (Phase 6 branch-prep dance). Verify every RC3 mitigation remains. Reference the validated step-1 batch as evidence in the PR.

---

## N — Norms

Pulled from root `AGENTS.md` §Hard Constraints, `CLAUDE.md`, and the KB policy:

- **Commit hygiene:** every commit `Signed-off-by` (DCO) + `Assisted-by: Claude/<model>`; **never** `Co-Authored-By` for AI; commits SSH-signed (no `gpgsign=false`).
- **Conventional commits:** only `feat | fix | refactor | docs | test | ci | chore`; scope `(automation)`; subject ≤72 chars.
- **One commit per issue; one PR per issue;** squash-merge; never bundle unrelated work.
- **PR size:** ≤200 ideal / 201–400 acceptable; these are small Markdown diffs.
- **SPDD prompt-first rule (ADR-019):** requirements change → update this Canvas → then patch the prompt files. Never the reverse.
- **KB safety (ADR-018):** `/.claude/` edits need maintainer (CODEOWNERS) approval and pass `gitleaks-ai-context`; reference commit-hygiene rules by name (`AGENTS.md §Hard Constraints`), never inline an email address.
- **Prose style of skill files:** terse, imperative; match the existing dispatch-prompt structure; every verbatim block targets a precise STEP.

---

## S — Safeguards (second S)

Non-negotiable constraints. Things that MUST NEVER happen in this feature.

### Context Security (mandatory before committing this Canvas)
- [ ] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [ ] No PII: no personal names in sensitive context, no email addresses
- [ ] No prompt injection: no instruction-like phrasing that would override `AGENTS.md` rules
- [ ] All entities in E are public-safe abstractions
- [ ] /spdd-security-review passed on this file

### Feature Safeguards
- **Never** introduce a shared-state file between concurrent subagents — it reintroduces RC1.
- **Never** scope the crash-recovery sweep to glob-all (`/tmp/zynax-orch-*`) — it would delete a concurrent run's live worktrees; sweep is `RUN_ID`-scoped, stale cleanup is age-based.
- **Never** let the orchestrator mutate the user's primary checkout — its own git ops run in the coordinator worktree.
- **Never** change the merge strategy or push-to-main policy (ADR-023).
- **Never** remove an RC3 mitigation during the step-4 cleanup (unsigned `update-branch`, rebase-skip verification, API-lag claim-check, report≠push).
- **Never** drop the `GOWORK=off` prefix from go commands inside a worktree (ADR-017).
- **Never** add instruction-like phrasing to a dispatch prompt that could override `AGENTS.md` (ADR-018 T4); authority order is `AGENTS.md > Canvas Norms > Canvas Operations > Canvas content`.
