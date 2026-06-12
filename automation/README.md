<!-- SPDX-License-Identifier: Apache-2.0 -->

# automation/ — Dev-Automation: Wave 4 Platform Manifests

> **This folder is NOT ambient context.** It is not auto-loaded by any `AGENTS.md` glob.
> It is consumed **explicitly** by `zynax apply` (which reads `workflows/*.yaml`) and by
> `make validate-spec` / pytest (which validate the manifests and their contracts).
>
> Full architecture direction: [`STATUS-AND-DIRECTION.md`](STATUS-AND-DIRECTION.md)
> EPICs: [#873](https://github.com/zynax-io/zynax/issues/873) M6.DevAuto ·
> [#881](https://github.com/zynax-io/zynax/issues/881) Wave 4 ·
> ADR-028 (AgentDef-vs-Workflow split)

---

## Status (2026-06-12) — Wave 4 delivered to the platform-readiness boundary

The **two-plane model** still holds, but both planes moved:

- **Near-term plane:** Waves 0–3 (GitHub Actions + advisory LLM CI) were **superseded**
  and retired in #1129. Their configs are archived under `docs/archive/dev-advisory/`;
  the learnings in `docs/ai-learnings/` remain real. The near-term automation layer today
  is the generalized Claude Code delivery commands (EPIC #1108: `/milestone-plan`,
  `/issue-deliver`, `/milestone-orchestrate`, `/milestone-learn` + the
  `.claude/commands/experts/*` knowledge base).
- **Platform plane (Wave 4, EPIC #881):** the orchestrator + expert mesh **as Zynax
  manifests** is authored and delivered — O1–O7 + O9 merged (#1096–#1102, #1104):
  ADR-028, 9 expert AgentDefs, the orchestrator Workflow, the issue-delivery Workflow
  (intake→plan→route→inject→implement→verify→decide), the context-slice injection
  binding in task-broker, and the learning-synthesizer AgentDef.

**The honest dividing line is still a test:** `automation/tests/test_platform_readiness.py`.
The schema-validation tests pass; the live `zynax apply` e2e remains
`@pytest.mark.xfail(strict=True)` because of four code-verified platform gaps
(workflow-compiler rejects `output:` — deferred to M7+ in `manifest.go`; Go-template
guards vs CEL evaluation, fail-closed; no capability providers for
review/aggregate/act/notify/record; no gateway outputs/decision-log read path).
That flip is **O8 (#1103), deferred to M7** — full gap analysis in the issue comments.
EPICs #873/#881 closed at this boundary.

**Do not wire `workflows/*.yaml` into main CI** until #1103 flips the readiness e2e to a
clean pass on a running platform.

---

## Folder Structure

```
automation/
├── STATUS-AND-DIRECTION.md    ← Living architecture doc (two-plane model, wave history)
├── README.md                  ← This file — entry point for contributors
│
├── workflows/                 ← Wave 4 platform manifests (EPIC #881, ADR-028)
│   ├── dev-advisory-orchestrator.yaml   ← kind: Workflow — fan_out → aggregate → act/escalate (O3, #1098)
│   ├── issue-delivery.yaml              ← kind: Workflow — intake→plan→route→inject→implement→verify→decide (O4 #1099, O6 #1101)
│   ├── learning-synthesizer.yaml        ← kind: AgentDef — synthesize_learnings, human-gated (O7, #1102)
│   └── experts/                         ← 9 × kind: AgentDef (8 domain experts + planner) (O2, #1097)
│       ├── arch-adr.yaml
│       ├── persistence-state.yaml
│       ├── api-contract.yaml
│       ├── security-supply-chain.yaml
│       ├── qa-bdd.yaml
│       ├── docs-agents.yaml
│       ├── ci-release.yaml
│       ├── planning-task-split.yaml
│       └── planner.yaml                 ← identify_next_issue capability (single provider)
│
└── tests/                     ← Manifest contract tests + the platform-readiness gate
    ├── test_platform_readiness.py ← schema tests pass; live-apply e2e stays xfail (O8 → M7, #1103)
    ├── test_expert_agentdefs.py
    ├── test_orchestrator_workflow.py
    ├── test_issue_delivery.py
    ├── test_learning_synthesizer.py
    ├── features/              ← BDD contracts (ADR-016)
    └── conftest.py
```

The retired near-term configs (`experts/*.yaml`, `orchestrator/*`) live at
`docs/archive/dev-advisory/` — they remain the **source of truth** each Wave 4 manifest
was translated from (context slices, I/O contracts, aggregation weights).

---

## How to Read / Consume

### Zynax Runtime (platform plane)

The Zynax runtime reads `automation/workflows/*.yaml`: experts and the synthesizer are
`kind: AgentDef` (capability providers), all orchestration is `kind: Workflow`
(state machines) — the ADR-028 split. `make validate-spec` validates every manifest
against `spec/schemas/agent-def.schema.json` / `spec/schemas/workflow.schema.json`.

Safeguards (inherited verbatim from the Wave 2 config, see canvas §Safeguards):
- **`never_auto`:** merge, push, bump-dependency, close-issue, delete-branch, force-push.
- **Strict context isolation:** task-broker binds each expert's registry-declared
  context slice at dispatch (O5, #1100) — a caller can never plant a foreign slice.
- **Human-gated learning loop:** the synthesizer can only propose
  `pending-human-review` updates; it declares no apply/edit/write capability.

### Claude Code delivery commands (near-term plane)

Day-to-day issue delivery runs through the generalized commands (EPIC #1108), which are
also Wave 4's specification — the manifests re-express that same orchestrator/expert
logic natively. See `docs/spdd/881-self-hosted-issue-delivery/canvas.md` Appendix B for
the command→manifest traceability table.

---

## Links

- Full architecture direction and wave history: [`STATUS-AND-DIRECTION.md`](STATUS-AND-DIRECTION.md)
- Canvas: [`docs/spdd/881-self-hosted-issue-delivery/canvas.md`](../docs/spdd/881-self-hosted-issue-delivery/canvas.md)
- ADR-028: [`docs/adr/ADR-028-agentdef-vs-workflow-self-hosted-automation.md`](../docs/adr/ADR-028-agentdef-vs-workflow-self-hosted-automation.md)
- EPICs: [#873 M6.DevAuto](https://github.com/zynax-io/zynax/issues/873) · [#881 Wave 4](https://github.com/zynax-io/zynax/issues/881)
- M7 continuation (readiness e2e flip): [#1103](https://github.com/zynax-io/zynax/issues/1103)
- Archived near-term plane: `docs/archive/dev-advisory/` (retired in #1129)
- Root `AGENTS.md` pointer: see Knowledge Base Index table (`automation/` row)

---

*Zynax — automation/README.md · Apache 2.0*
*Assisted-by: Claude/claude-sonnet-4-6*
*Assisted-by: Claude/claude-fable-5*
