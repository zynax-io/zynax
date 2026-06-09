<!-- SPDX-License-Identifier: Apache-2.0 -->

# automation/ — Dev-Automation Orchestrator + Expert Mesh

> **This folder is NOT ambient context.** It is not auto-loaded by any `AGENTS.md` glob.
> It is consumed **explicitly** by GitHub Actions workflows (which read `experts/*.yaml`)
> and (aspirationally) by the Zynax agent runtime (which will read `workflows/*.yaml`).
>
> Full architecture direction: [`STATUS-AND-DIRECTION.md`](STATUS-AND-DIRECTION.md)
> EPIC: [#873](https://github.com/zynax-io/zynax/issues/873) M6.DevAuto

---

## What this folder IS

A **two-plane automation system** that brings expert AI review into the CI/CD loop:

- **Near-term plane (Waves 0–3):** GitHub Actions + Claude Code subagents. Runnable
  today. Zero Zynax runtime dependency.
- **Aspirational plane (Wave 4):** Orchestrator + experts running as Zynax `kind: AgentDef`
  workflows on the Zynax platform itself. Blocked until M6.H (#626) and M6.I (#772) are
  both complete.

The **honest dividing line** between the two planes is a failing test:
`automation/tests/test_platform_readiness.py` — it is marked `@pytest.mark.xfail(strict=True)`
and fails today for three independent reasons (no `workflows/` AgentDefs, in-memory repos,
stub EventBusService). When M6.H + M6.I land and `workflows/` is authored (#881), that test
flips to pass and Wave 4 becomes viable.

## What this folder is NOT

- Not a Zynax service — no `go.mod`, no `pyproject.toml`, no gRPC handler
- Not a CI workflow — the `.github/workflows/` directory contains the actual workflow YAML
- Not ambient context — no `AGENTS.md` in any layer auto-includes this path
- Not safe to hand-edit banner-marked regions — use `make sync-images` for image refs

---

## Folder Structure

```
automation/
├── STATUS-AND-DIRECTION.md    ← Living architecture doc (full two-plane model, wave specs)
├── README.md                  ← This file — entry point for contributors
│
├── experts/                   ← Expert YAML configs (DevAuto.2, #875)
│   ├── schema.yaml            ← JSON Schema for all expert config files
│   ├── orchestrator.yaml      ← Orchestrator expert (aggregation, escalation)
│   ├── arch-adr.yaml          ← Architecture / ADR expert
│   ├── persistence-state.yaml ← Persistence / state expert
│   ├── api-contract.yaml      ← API / contract expert
│   ├── security-supply-chain.yaml ← Security + supply-chain expert
│   ├── qa-bdd.yaml            ← QA / BDD expert
│   ├── docs-agents.yaml       ← Docs / AGENTS expert
│   ├── ci-release.yaml        ← CI / release expert
│   └── planning-task-split.yaml   ← Planning / task-split expert
│
├── orchestrator/              ← Orchestrator config (DevAuto.3, #876)
│   ├── config.yaml            ← Aggregation thresholds, escalation rules, auto_allowed list
│   ├── decision-log-schema.yaml   ← JSON Schema for per-run decision-log artifact
│   └── aggregation-protocol.md   ← Human-readable description of the weighting algorithm
│
├── workflows/                 ← AgentDef YAMLs — ASPIRATIONAL (DevAuto.8, #881)
│   └── (not yet authored — blocked on M6.H #626 + M6.I #772)
│
└── tests/                     ← Platform-readiness gate (DevAuto.9, #882)
    ├── test_platform_readiness.py ← xfail test — the honest Wave 4 gate
    ├── conftest.py
    └── requirements.txt
```

---

## How to Read / Consume

### GitHub Actions (near-term plane)

GH Actions reads `automation/experts/*.yaml` to configure expert subagent invocations.
Each expert file declares:
- `context_slice` — which files/paths to feed as context
- `max_tokens` — hard budget for this expert's context window
- `system_prompt` — expert role and output contract

The `.github/workflows/dev-advisory.yml` workflow (DevAuto.4, #877) fans out to all 9
experts in parallel, then the orchestrator aggregates their outputs into a single PR comment.

### Zynax Runtime (aspirational plane — Wave 4 only)

When M6.H (#626) and M6.I (#772) are complete, the Zynax agent runtime will read
`automation/workflows/*.yaml` as `kind: AgentDef` manifests and execute the
orchestrator + expert mesh as native Zynax workflows.

**Do not wire `workflows/*.yaml` into main CI** until
`automation/tests/test_platform_readiness.py` flips from `xfail` to a clean pass.

---

## Two-Plane Summary

```
┌─────────────────────────────────────────────────────────────┐
│  NEAR-TERM PLANE (Waves 0–3)                                │
│  GitHub Actions + Claude Code subagents                     │
│  Runnable today. No Zynax runtime dependency.               │
│                                                             │
│  Wave 0: Advisory — expert subagents on PR events           │
│  Wave 1: Orchestrated advisory — aggregated PR comment      │
│  Wave 2: Gated automation — non-destructive actions only    │
│  Wave 3: Post-merge completeness mesh                       │
└────────────────────────────┬────────────────────────────────┘
                             │
         automation/tests/test_platform_readiness.py
         @pytest.mark.xfail(strict=True)
         ← THE HONEST LINE BETWEEN PLANES →
         Fails today. Passes only when M6.H + M6.I complete.
                             │
┌────────────────────────────▼────────────────────────────────┐
│  ASPIRATIONAL PLANE (Wave 4)                                │
│  Zynax AgentDef workflows running on Zynax itself           │
│  BLOCKED: needs M6.H (Postgres) + M6.I (event-bus)         │
│  GATED: failing test must flip to pass first                │
└─────────────────────────────────────────────────────────────┘
```

---

## Links

- Full architecture direction and wave specs: [`STATUS-AND-DIRECTION.md`](STATUS-AND-DIRECTION.md)
- EPIC: [#873 M6.DevAuto](https://github.com/zynax-io/zynax/issues/873)
- Platform-readiness gate: `automation/tests/test_platform_readiness.py` ([#882](https://github.com/zynax-io/zynax/issues/882))
- Root `AGENTS.md` pointer: see Knowledge Base Index table (`automation/` row)

---

*Zynax — automation/README.md · Apache 2.0*
*Assisted-by: Claude/claude-sonnet-4-6*
