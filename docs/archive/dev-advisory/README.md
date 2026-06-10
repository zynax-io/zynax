<!-- SPDX-License-Identifier: Apache-2.0 -->

# docs/archive/dev-advisory/ — Retired LLM Advisory Mesh (Waves 0–2)

**Status: ARCHIVED** — retired by issue
[#1112](https://github.com/zynax-io/zynax/issues/1112)
(spec: `docs/contributing/ci-delivery-overhaul-prompt.md` §1A, part of #1109).
Nothing in this directory runs anywhere. It is preserved for reference only.

## What this was

The **dev-advisory mesh** (M6.DevAuto Waves 0–2, EPIC #873): a GitHub Actions
workflow (`.github/workflows/dev-advisory.yml`, now deleted) that fanned out to
9 LLM "expert" reviewers on every PR — each configured by an `experts/*.yaml`
file — then aggregated their outputs through the orchestrator configuration in
`orchestrator/` and posted a single advisory comment on the PR. `invoke-llm.sh`
was the shared script that called the LLM API for each expert.

Contents:

| Path | Original location | Role |
|------|-------------------|------|
| `experts/` | `automation/experts/` | 9 expert configs + schema (context slice, token budget, system prompt) |
| `orchestrator/` | `automation/orchestrator/` | Aggregation thresholds, decision-log schema, weighting protocol |
| `invoke-llm.sh` | `scripts/invoke-llm.sh` | LLM API invocation script used by the workflow |

## Why it was retired

- **Non-actionable output:** the per-PR advisory comments were not gates — they
  could not block a merge and rarely changed reviewer behaviour.
- **API quota:** every PR consumed LLM API quota across 9 expert calls plus
  aggregation, for advisory-only signal.
- **Duplicated signal:** the deterministic checks in `ci.yml` and
  `pr-checks.yml` already cover the failure modes the mesh commented on.

The decision was to **archive, not delete** — the Wave 0–2 design work is
preserved here (and in git history) as input for the Wave 4 re-expression.

## The living vision: Wave 4 (#881)

The idea of an orchestrated expert mesh is not dead. Issue
[#881](https://github.com/zynax-io/zynax/issues/881) (M6.DevAuto Wave 4,
canvas: `docs/spdd/881-self-hosted-issue-delivery/canvas.md`) re-expresses it
as **Zynax-native `kind: AgentDef` workflows running on the Zynax platform
itself** — not GitHub Actions LLM calls. See
`automation/tests/test_platform_readiness.py` for the readiness gate and
`automation/STATUS-AND-DIRECTION.md` for the two-plane architecture history.
