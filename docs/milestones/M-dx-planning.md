<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax M-dx — Developer Experience Planning (skeleton)

> **Milestone:** M-dx · GitHub milestone **#9** "Developer Experience (M-dx)" · Target: see ROADMAP version plan
> **Label:** `milestone: M-dx` · **Status:** planned / recurring — not the active build milestone
> This is the planning **skeleton** the `/milestone open` flow expects to exist; it inventories
> the live GitHub state so scoping starts from truth. Scope decisions belong to a future
> `/plan --milestone M-dx` pass.

## 0 — Goal

Make contributing to and building on Zynax delightful — contributor fast-lane, SDK/adapter-author
experience, and the AI-methodology/tooling stack — distinct from end-user UX (M-UX).

## 1 — EPIC inventory (live GitHub state, 2026-07-08)

| EPIC | Issue | State | Notes |
|------|-------|-------|-------|
| Contributor Experience — fast-lane + PR ergonomics + automation | [#1391](https://github.com/zynax-io/zynax/issues/1391) | open, aggregator | children #1361, #1363, #1366, #1368, #1369 (all open) |
| SDK & Adapter-Author Experience | [#1392](https://github.com/zynax-io/zynax/issues/1392) | open, placeholder | no children filed yet |
| SPDD methodology | [#205](https://github.com/zynax-io/zynax/issues/205) | open, **delivered** | all 9 children closed — recommended close (see triage notes 2026-07-08) |
| Technical Excellence Modernization | [#173](https://github.com/zynax-io/zynax/issues/173) | open, ~88% done | open stragglers: #146, #233, #234, #244, #245 |
| AI agent knowledge base | [#148](https://github.com/zynax-io/zynax/issues/148) | open, near-done | only #146 open — recommended fold into #173 |

Loose stories currently in the milestone: #1361, #1363, #1366, #1368, #1369 (via #1391) ·
#146 (double-parented #148/#173). Good-first-issue pool routed here: #1657, #1660, #1661.

## 2 — Known scope-shaping facts

- Program order (2026-06-18 realignment): M7 → M-UX → M-dx → M8; M9 was inserted as the
  active build milestone after M8's thin-Zynax reduction. M-dx remains a recurring bucket
  until explicitly activated.
- The zero-Temporal Day-0 engine (#1359) was superseded by #1456 (lightweight Temporal eval
  profile; ADR-037 Rejected) — do not resurrect it here without a new ADR.
- Epic-consolidation triage (2026-07-08): collapse #205/#148 into #173 or a fresh thin
  tracker before activating this milestone (see the recommendation comments on those issues).

## 3 — Dependencies

- Nothing in M9 blocks M-dx planning; M-dx implementation competes with M9 for capacity.
- ADR proposals #1693 (API versioning) and #1695 (fuzz strategy) touch contributor-facing
  policy and should be settled before SDK/adapter-author guides freeze.

## 4 — Risks (skeleton)

| Risk | Note |
|------|------|
| Placeholder epics (#1392) activate without INVEST decomposition | run `/plan #1392` before labelling anything ready |
| Epic nesting (#173 ⊃ #205/#148/#146) double-counts progress | consolidate first (triage comments filed 2026-07-08) |

## 5 — Exit criteria

To be defined at `/milestone open M-dx` time. Seed: fast-lane merged (#1361), PR-size
split-not-possible honored (#1363), integration suites green in gate (#1368), SDK author
guide shipped (#1392 child).
