# Learnings: SPDD Canvas Expert

> Format: each entry has `Seen in:` (issue/session) and `Date:` (YYYY-MM-DD).
> Updated by `/m6-learn` after each batch. Human-reviewed before merging.

---

## Effective patterns

- **The O section (Operations) is the most important canvas section — every O-step must be
  independently releasable and ≤400 lines.**
  Canvases that describe O-steps too broadly produce PRs that are too large and get rejected.
  Each O-step should describe exactly what one PR will do, in enough detail that
  `/spdd-generate` can implement it without further design decisions.
  Seen in: M6 EPIC canvas reviews broadly. Date: 2026-06-06.

- **The R section must reference K8s DoD criteria by name, not just "it should work".**
  "Service passes Kubernetes liveness + readiness probes" is a valid R section.
  "Improve health checking" is not — it has no observable outcome to verify.
  Seen in: M6.A #463 canvas. Date: 2026-06-06.

- **Tier 2 violations are always false positives on hostnames — grep before reviewing.**
  The security scanner flags strings that look like hostnames. Run `grep -E '\b[a-z0-9-]+\.[a-z]{2,}\b'`
  on the canvas before the review to identify any hostname-shaped strings that need to move
  to `canvas.private.md`.
  Seen in: M6.H #626 canvas security review. Date: 2026-06-06.

- **Always cross-check the ADR index before proposing a design in the canvas.**
  Multiple ADRs have already decided key questions (engine pluggability, no shared DB,
  gRPC-only inter-service, mTLS). Proposing a canvas that contradicts an Accepted ADR
  triggers a human rejection. Read `docs/adr/INDEX.md` first.
  Seen in: M6.I #772 canvas (event-bus ADR-022 decision). Date: 2026-06-06.

---

## Edge cases discovered

- **SPDD-exempt issues (fix:/ci:/chore:/docs:) still need story issues with acceptance criteria.**
  "SPDD-exempt" means no canvas is required, not that there are no story issues.
  Create story issues via `gh issue create` with the standard test-plan template.
  Seen in: M6.F #670 (Config convergence). Date: 2026-06-06.

- **Canvas O-steps that share proto types with adjacent O-steps are NOT independent (INVEST).**
  If O-step 2 defines a proto message that O-step 3 uses, they must be sequenced —
  O-step 2's PR must merge before O-step 3's branch is created.
  The canvas should make this dependency explicit in the O-step description.
  Seen in: M6.Argo #766 canvas. Date: 2026-06-06.

- **`/spdd-security-review` auto-alignment only works when the canvas has a clear Status: Draft line.**
  If the Status line is missing or malformed, the sed substitution silently fails.
  Always verify after auto-alignment: `grep "^Status:" docs/spdd/<N>-*/canvas.md`.
  Seen in: /m6-issue-generate STEP 4-CANVAS design. Date: 2026-06-06.

---

## Failed approaches

- **Writing O-steps as "implement X service" without file-level scope.**
  Ambiguous O-steps produce PRs that are either too large (everything) or too small
  (only one file). O-steps must name the specific files to create or modify.
  Seen in: M2 canvas early drafts. Date: 2026-06-06.

---

## Proposed expert prompt updates

*(none yet — populate after first batch of SPDD canvas expert sessions)*

## Session — 2026-06-09 (documentation consistency — M6 state reconciliation, EPIC #1001)

### Effective patterns
- **Live issue/PR state is the source of truth for "done"** — not the planning doc's Status column or a canvas `Status:` field. Reconcile docs against `gh issue list --milestone … --state open` + `gh pr list --state merged`, never against memory or the previous doc snapshot.

### Edge cases discovered
- **Doc drift accumulates because per-story delivery updates only the *local* status surfaces** (the M6-planning row + the canvas O-step checkbox) and not the *cross-cutting* ones. Found stale in one pass: `ARCHITECTURE.md` milestone table said M6 "📅 Planned" while 92 issues were closed; `README.md` had no M6 milestone row and listed event-bus/memory-service "📋 Planned" though EPICs #772/#773 had merged stories; `M6-planning.md` showed ADR-024 (#862), event-bus I.2–I.6, and images O5–O7 as Open/Pending though all merged; the `855-images-sot` canvas was still `Aligned` though every O-step (incl. ADR-024) merged; `state/current-milestone.md` had no M6 progress section.
- **An EPIC canvas `Status:` never flips `Aligned`→`Implemented` automatically** when its last O-step merges — it must be flipped explicitly, and was missed for `855-images-sot`. No CI gate catches a stale milestone label, so drift is silent.

### Proposed expert prompt update
- Rule: At every story delivery, reconcile **all** status surfaces in the same PR, driven by live issue/PR state — not just the immediate two. The "update state files" step must additionally: (a) flip the EPIC canvas `Status:` `Aligned`→`Implemented` when the issue closing the last O-step merges; (b) update the milestone tables in `README.md`, `ROADMAP.md`, `ARCHITECTURE.md`, `CLAUDE.md` (and the README per-service status table) whenever an EPIC completes or a service's implementation status changes; (c) refresh `state/current-milestone.md` progress + its "as of" date. Before opening the PR, run a consistency check: `grep` the milestone/status markers across README/ROADMAP/ARCHITECTURE/CLAUDE/current-milestone/M6-planning and confirm they agree on each milestone's state and each service's status.
  Category: domain
  Reason: doc drift is silent (no CI gate flags a stale milestone label) and compounds every iteration; delivery time is the only point where the author knows exactly what changed.
