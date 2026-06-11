<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-028 — AgentDef-vs-Workflow Split for Self-Hosted Automation + Context-Slice Injection Contract

| Field | Value |
|-------|-------|
| **Status** | Accepted |
| **Date** | 2026-06-11 |
| **Deciders** | Oscar Gómez Manresa |
| **Scope** | `automation/workflows/**` (Wave 4 platform manifests), `spec/schemas/agent-def.schema.json`, `spec/schemas/workflow.schema.json`, `automation/tests/test_platform_readiness.py` — EPIC #881 |
| **Related** | ADR-011 (declarative YAML control plane), ADR-014 (event-driven state machine), ADR-019 (SPDD governance), ADR-021 (Postgres-backed repos), ADR-022 (EventBus architecture) |

---

## Context

EPIC #881 (M6.DevAuto Wave 4) makes the maintainer's issue-delivery pipeline
**self-hosted**: the orchestrator and the domain experts that today live as
Claude Code slash commands and subagent prompts become Zynax manifests that run
on the Zynax platform itself — task-broker, agent-registry, EventBus, Postgres.
The headline demo is Zynax reading one of its own GitHub issues, routing it to
the right expert with a bounded context slice, and driving it to an implemented
change.

The blocker is that **the manifest model was undecided**:

1. **The original #881 sketch matches neither shipped schema.** It used
   `spec.steps`, `spec.trigger`, and `parallel: true` — a hybrid that conflates
   *what an agent can do* with *how work is orchestrated*. Meanwhile two schemas
   shipped and are enforced by `make validate-spec`:
   - `spec/schemas/agent-def.schema.json` — a **capability provider**
     (`kind: AgentDef`): `spec.capabilities[]` with per-capability
     `input_schema`/`output_schema`, plus `spec.runtime` (image, env, replicas).
     Registered with AgentRegistryService; dispatched by the task broker.
   - `spec/schemas/workflow.schema.json` — a **state machine**
     (`kind: Workflow`): `spec.initial_state` + `spec.triggers` + `spec.states`,
     where states invoke capabilities via `actions[]` and transition on
     CloudEvents. Compiled by WorkflowCompilerService.
2. **The near-term expert configs are not platform manifests.** The Wave 1–3
   expert YAMLs (delivered under `automation/experts/` in #875/#876, archived to
   `docs/archive/dev-advisory/experts/` when `dev-advisory.yml` was retired in
   #1129) each declare a `context_slice` (`{files[], max_tokens}`), an
   `input_contract`/`output_contract`, and an `aggregation_weight`. They are the
   source of truth for each expert's knowledge boundary, but they are consumed
   by GitHub Actions, not loadable by `zynax apply`.
3. **This is a one-way door.** Every Wave 4 manifest (9 expert AgentDefs, the
   orchestrator workflow, the issue-delivery workflow, the learning
   synthesizer) is shaped by which `kind` each concept maps to. Reversing the
   mapping after O2–O8 land would mean rewriting every manifest and its tests.

---

## Decision

**1. Two kinds, no third schema.** Experts, the planner, and the learning
synthesizer are **`kind: AgentDef`** — capability providers validated by
`spec/schemas/agent-def.schema.json`. **All orchestration** (the dev-advisory
fan-out→aggregate→act loop and the issue-delivery
intake→plan→route→inject→implement→verify→decide loop) is **`kind: Workflow`**
— state machines validated by `spec/schemas/workflow.schema.json`. The original
#881 `spec.steps`/`spec.trigger`/`parallel: true` sketch is **superseded**. We
will not invent a third manifest schema and will not extend
`agent-def.schema.json` with orchestration fields.

Concretely:

| Concept | kind | Capability / states |
|---------|------|---------------------|
| 8 domain experts (`arch-adr`, `persistence-state`, `api-contract`, `security-supply-chain`, `qa-bdd`, `docs-agents`, `ci-release`, `planning-task-split`) | AgentDef | `review` |
| Planner (`planning-task-split`, extended) | AgentDef | `review` + `identify_next_issue` |
| Learning synthesizer | AgentDef | `synthesize_learnings` (human-gated apply) |
| Dev-advisory orchestrator | Workflow | `fan_out` → `aggregate` → `act`/`escalate` |
| Issue-delivery engine | Workflow | `intake` → `plan` → `route` → `inject` → `implement` → `verify` → `decide` |

Parallelism is expressed the way `workflow.schema.json` already defines it: all
`actions[]` of a state start before the state machine waits for a transition —
the `fan_out` state lists the expert `review` invocations and the EventBus
(ADR-022) carries the durable fan-out/collection. No `parallel:` flag is needed.

**2. Context-slice injection contract.** Each expert AgentDef declares a
bounded context slice — `{files[], max_tokens}` — translated 1:1 from its
near-term YAML (now `docs/archive/dev-advisory/experts/<name>.yaml`). This is
the on-platform analogue of injecting a `.claude/commands/experts/*.md` prompt
into a fresh Claude Code subagent. The contract:

- **Bounded:** an expert receives only its declared `files[]`, hard-capped at
  its declared `max_tokens` (overflow policy: truncate oldest files).
- **Strictly isolated:** no expert ever sees another expert's slice or output
  (`fan_out.context_budget.isolation: strict`, carried over verbatim from the
  Wave 2 orchestrator config).
- **Orchestrator exception:** the orchestrator workflow aggregates expert
  *outputs only* — never raw code files — within its own token budget.
- **Transport:** the slice is bound at dispatch time into the capability
  `input_payload` (the `context_slice` field of the `review` capability's
  `input_schema`), routed by the task broker over the EventBus.

**3. Both planes coexist.** The Wave 0–3 GitHub Actions plane remains in place
and unchanged; Wave 4 manifests are loaded only by `zynax apply` and are never
wired into main CI as required checks until `test_platform_readiness.py`
passes against a real platform.

---

## Rationale

| Option | Assessment |
|--------|------------|
| **A — Split across the two shipped kinds** (AgentDef = capability provider, Workflow = orchestration) | ✅ Chosen — maps honestly onto schemas that already exist, are CI-enforced (`make validate-spec`), and are consumed by shipped services (agent-registry, workflow-compiler, task-broker, engine-adapter). Zero schema changes; Wave 4 exercises the platform exactly as any user manifest would — which is the point of self-hosting. |
| **B — Extend `agent-def.schema.json` with `steps`/`trigger`/`parallel` (the original #881 sketch)** | ✗ Rejected — conflates agent and workflow concerns in one kind; duplicates the state-machine semantics ADR-014 already standardised; requires schema + registry + compiler changes for a capability the Workflow kind already provides; the demo would then prove a bespoke path, not the real platform. |
| **C — Invent a third manifest schema for "automation" resources** | ✗ Rejected — a third source of truth with its own validation, compiler, and lifecycle; violates the §Safeguards rule of the #881 canvas ("never invent a third manifest schema") and adds maintenance surface for no expressive gain. |

The context-slice contract is recorded here (not in a separate ADR) because it
is the load-bearing half of the same decision: AgentDefs are only safe to fan
out in parallel *because* their context is bounded and isolated — that is what
makes the thin-orchestrator pattern work both off-platform (Claude Code
subagents) and on-platform (capability dispatch).

---

## Consequences

### Positive

- O2–O8 of EPIC #881 have a fixed target: 9 AgentDefs + 2 Workflows + 1
  synthesizer AgentDef, all validating against shipped schemas with no schema
  PRs in the critical path.
- The self-hosting demo doubles as a real integration test of agent-registry,
  task-broker, EventBus, and workflow-compiler — the manifests use only public,
  documented surface.
- The expert knowledge boundary stays in one place: the archived near-term
  YAMLs remain the source of truth for slices/contracts, translated 1:1 into
  AgentDef capabilities.

### Negative / trade-offs

- The expressive limits of `workflow.schema.json` are now binding for
  automation: anything the state machine cannot express (e.g. dynamic expert
  sets) needs a schema evolution PR first, not an automation-side workaround.
- Context slices are declared per-expert in two related shapes (archived
  near-term YAML and AgentDef `input_schema`) until the GitHub Actions plane is
  eventually retired — drift between them must be checked in O2's unit tests.

### Neutral / follow-up required

| Action | Tracking |
|--------|---------|
| Translate 9 expert YAMLs → `automation/workflows/experts/*.yaml` AgentDefs | EPIC #881 O2 (#1097) |
| Author orchestrator + issue-delivery Workflows | EPIC #881 O3/O4 (#1098, #1099) |
| Runtime context-slice binding with isolation test | EPIC #881 O5 (#1100) |
| Flip `test_platform_readiness.py` from xfail to a real e2e | EPIC #881 O8 (#1103) |
