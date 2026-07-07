<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax M-UX — User Experience Planning (skeleton)

> **Milestone:** M-UX · GitHub milestone **#10** "User Experience (M-UX)" · Target: see ROADMAP version plan
> **Label:** `milestone: M-UX` · **Status:** planned — not the active build milestone
> Skeleton for the `/milestone open` flow; inventories live GitHub state (2026-07-08).
> Scope decisions belong to a future `/plan --milestone M-UX` pass.

## 0 — Goal

The **forward** User-Experience program: experience Zynax's value with **no clone** (hosted
try-it), intelligent context-loading at scale, and a discoverable Documentation Portal —
distinct from the M7 first-run closeout (delivered) and from contributor DX (M-dx).

## 1 — EPIC inventory (live GitHub state)

| EPIC | Issue | State | Notes |
|------|-------|-------|-------|
| Forward User Experience — no-clone try-it + intelligent context-loading | [#1389](https://github.com/zynax-io/zynax/issues/1389) | open, placeholder | priority: high; no children filed |
| Documentation Portal (Diátaxis restructure) | [#1390](https://github.com/zynax-io/zynax/issues/1390) | open, placeholder | no children filed |

Milestone #10 currently holds 0 closed issues — purely forward-looking.

## 2 — Known scope-shaping facts

- Program order (2026-06-18 realignment): M7 → **M-UX** → M-dx → M8; overtaken by events —
  M8 (thin-Zynax) and M9 (hard removals) were executed first. Re-sequencing M-UX is a
  ROADMAP decision to take explicitly, not to drift into.
- Product/market reviews (2026-06-19) put the binding constraint at **awareness/distribution**:
  hero asciinema cast, public launch, named external adopter. Those are maintainer actions
  tracked in the product docs, not repo epics — but M-UX's no-clone path is the highest-leverage
  repo-side complement.
- Deferred-with-own-canvas items adjacent to this milestone: #1385 (scenario manifest),
  #1387 (context injection) — canvases exist (`docs/spdd/1385-*`, `docs/spdd/1387-*`), Aligned,
  not started.

## 3 — Dependencies

- Documentation Portal (#1390) benefits from landing after the M9 doc truth-passes (fewer
  pages to restructure twice).
- No-clone try-it (#1389) depends on a hosting decision (one-way door → ADR before build).

## 4 — Risks (skeleton)

| Risk | Note |
|------|------|
| Both epics are placeholders | run `/plan #1389` / `/plan #1390` before any story work |
| Hosted playground cost/abuse surface | needs its own ADR + security review before implementation |

## 5 — Exit criteria

To be defined at `/milestone open M-UX` time. Seed: a visitor reaches a meaningful Zynax
result with zero local setup; docs restructured per Diátaxis with a working information
architecture.
