<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — Wave 4: Self-Hosted Issue-Delivery Engine (Orchestrator + Experts as Zynax Manifests)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). None was identified for this feature.
> Run `/spdd-security-review docs/spdd/881-self-hosted-issue-delivery/canvas.md` before committing.

**Issue:** #881 (EPIC — M6.DevAuto, Wave 4)
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-10
**Status:** Aligned

**Parent EPIC:** #873 (M6.DevAuto) · **Prerequisites (all now CLOSED):** #626 (M6.H Postgres repos), #772 (M6.I EventBus), #874–#882 (Waves 0–3 + readiness test)

**Child issues (O-steps):** #1096 (O1) · #1097 (O2) · #1098 (O3) · #1099 (O4) · #1100 (O5) · #1101 (O6) · #1102 (O7) · #1103 (O8) · #1104 (O9) — see §O. Per ADR-019, `feat:` O-steps wait for **Aligned**.

---

## Why this Canvas exists (the real value being proven)

The maintainer currently delivers Zynax issues with a set of **Claude Code slash commands**:

| Command | What it does today (off-platform) |
|---------|-----------------------------------|
| `/m6-plan` | Reads live GitHub + canvas state, computes dependency-aware parallel groups, **identifies which issue goes next**. |
| `/m6-issue-generate <N>` | **Reads issue details**, claims it, runs SPDD, implements, waits for CI, verifies artifacts, merges — **gets one issue implemented end-to-end**. |
| `/m6-orchestrate` | Thin coordinator: claims a batch, **routes each issue to the right expert**, fans out expert subagents **in parallel with isolated context**, collects results. |
| `/m6-learn` | Synthesizes `docs/ai-learnings/*.md` into **expert-prompt updates** (human-gated). |
| `.claude/commands/experts/*.md` | Eight **domain expert prompts** injected into subagents at dispatch — the split-context knowledge base. |

These commands prove a *pattern*: **a thin orchestrator with a tiny context budget routes work to domain
experts, each of which starts fresh with only its bounded context slice, does deep work, and reports a
structured result; learnings feed back into the experts.** That pattern is exactly the Zynax paradigm —
declarative agents, capability dispatch, bounded context, event fan-out.

**Wave 4 makes that pattern self-hosted.** Instead of living in Claude Code, the orchestrator and experts
become **Zynax manifests** that run on the Zynax platform. The headline demo: *Zynax reads one of its own
GitHub issues, updates its plan, routes the issue to the right expert with an injected context slice,
drives it to an implemented change, and records the decision — using its own task-broker, agent-registry,
EventBus, and Postgres.* Zynax developing Zynax.

This Canvas configures **every agent's capabilities and context injection** and the **orchestration** that
turns them into one delivered issue.

---

## R — Requirements

**Problem.** Issue #881 was filed as "aspirational, BLOCKED" — gated on M6.H (#626), M6.I (#772), and the
Wave 0–3 chain. **All of those are now CLOSED.** The gate has lifted, but three problems remain before the
self-hosting demo is real:

1. **The manifest model is undecided.** The original #881 sketch used `spec.steps` / `spec.trigger` /
   `parallel: true`. That matches **neither** shipped schema: `spec/schemas/agent-def.schema.json` is a
   *capability provider* (`spec.capabilities[]` + `spec.runtime`), and `spec/schemas/workflow.schema.json`
   is a *state machine* (`spec.initial_state` + `spec.triggers` + `spec.states`). We must pick the honest
   mapping, not invent a third schema.
2. **The near-term YAMLs (`automation/experts/*.yaml`) are not platform manifests.** They are
   schema-`automation/experts/schema.yaml`-shaped context-slice configs consumed by GitHub Actions — not
   loadable by `zynax apply`. They must be *translated*, with the YAML as the source of truth for each
   agent's context slice and I/O contract.
3. **The readiness gate is still xfail and points at the wrong schema path.**
   `automation/tests/test_platform_readiness.py` references `spec/schemas/agent-def.json` (does not exist;
   the real file is `agent-def.schema.json`) and is `@pytest.mark.xfail(strict=True)`. It must flip to a
   genuine pass driven by a running platform.

**Definition of done — observable outcomes:**

- `automation/workflows/experts/*.yaml` — **9** `kind: AgentDef` manifests (8 domain experts + planner),
  each validating against `spec/schemas/agent-def.schema.json`, each carrying its bounded **context_slice**
  and **review/advise capability** I/O contract translated from `automation/experts/<name>.yaml`.
- `automation/workflows/dev-advisory-orchestrator.yaml` — a `kind: Workflow` state machine
  (fan-out → aggregate → act) that dispatches to the expert capabilities and validates against
  `spec/schemas/workflow.schema.json`.
- `automation/workflows/issue-delivery.yaml` — a `kind: Workflow` that, given a GitHub issue number,
  **reads the issue, updates the plan, identifies the next issue, routes to one expert, injects that
  expert's context slice, and drives it to an implemented change** — the self-hosting headline.
- A learning-loop `kind: AgentDef` (synthesizer) that turns session results into proposed expert-manifest
  updates, **human-gated** (mirrors `/m6-learn`).
- `automation/tests/test_platform_readiness.py` — **xfail removed**, correct schema path, and an e2e
  assertion that `zynax apply` of the orchestrator workflow on a running platform (Postgres + EventBus)
  produces an **aggregated verdict** and a **decision-log entry** for one real issue.
- Wave 0–3 GitHub Actions workflows remain **in place and unchanged** — both planes coexist.
- Every PR is ≤ the size budget, signed (DCO + SSH + `Assisted-by`), one commit per O-step.

---

## E — Entities

**Manifests (the new declarative assets, under `automation/workflows/`):**

- **Expert AgentDef** ×8 — `arch-adr`, `persistence-state`, `api-contract`, `security-supply-chain`,
  `qa-bdd`, `docs-agents`, `ci-release`, `planning-task-split`. Each is a `kind: AgentDef` exposing one
  **`review` capability** (`input: {trigger, diff_summary, changed_files, context_slice}` →
  `output: {summary, recommended_actions, reasons_decisions, confidence, flags, extra_fields}`). The
  capability I/O is the `output_contract` already declared in `automation/experts/<name>.yaml`.
- **Planner AgentDef** — `planning-task-split` extended with an **`identify_next_issue` capability**
  (`input: {milestone, open_issues, in_progress, dependency_table}` → `output: {next_issue, blocked_by,
  ready_batch, rationale}`). This is the `/m6-plan` brain.
- **Orchestrator Workflow** — `kind: Workflow`, `dev-advisory-orchestrator`. States: `fan_out` (dispatch
  all expert `review` capabilities in parallel) → `aggregate` (weighted-consensus verdict) → `act`
  (execute only `auto_allowed` non-destructive actions; else `escalate`). The **only** agent that sees all
  expert outputs.
- **Issue-Delivery Workflow** — `kind: Workflow`, `issue-delivery`. States: `intake` (read issue, classify
  type/expert) → `plan` (call planner `identify_next_issue`) → `route` (select expert) → `inject` (bind the
  expert's `context_slice`) → `implement` (drive the expert capability to a change) → `verify` (local gates)
  → `decide` (record decision-log; emit next-issue event). The self-hosting headline path.
- **Synthesizer AgentDef** — `kind: AgentDef`, `learning-synthesizer`, capability `synthesize_learnings`
  (`input: session_results[]` → `output: proposed_manifest_updates[]`). Human-gated apply (mirrors
  `/m6-learn` APPLY_LOG.md). Feeds back into the expert AgentDefs.

**Bridging / runtime entities (already shipped — Wave 4 consumes them):**

- **Context-slice** — the bounded `{files[], max_tokens}` set fed to exactly one expert. The on-platform
  analogue of "inject the expert `.md` into a fresh subagent." Strict isolation: experts never see each
  other's slice (mirrors `automation/orchestrator/config.yaml → fan_out.context_budget.isolation: strict`).
- **Decision-log** — JSON artifact per orchestrator run
  (`automation/orchestrator/decision-log-schema.yaml`); on-platform it is a durable record, not a CI artifact.
- **Capability dispatch** — task-broker (#479/#480, #626 Postgres-backed) routes each expert capability call.
- **EventBus** (#772, NATS JetStream) — durable fan-out of the 9 parallel `review` dispatches and collection
  of their results (replaces the orchestrator's in-process `Agent(run_in_background)` fan-out).
- **Postgres repos** (#626) — durable workflow/agent state so a restart does not lose an in-flight delivery.
- **Schemas** — `agent-def.schema.json` (experts), `workflow.schema.json` (orchestration), validated by
  `make validate-spec`.

Relationship sketch:

```
issue-delivery (kind: Workflow)
  intake → plan ──────────────► planning-task-split.identify_next_issue   (the /m6-plan brain)
              │                         (reads milestone + open issues + deps)
              ▼
            route → inject(context_slice) → implement ──► one expert AgentDef.review
              │                                              (bounded slice; strict isolation)
              ▼
            verify → decide → decision-log (durable)  + emit "next-issue" CloudEvent

dev-advisory-orchestrator (kind: Workflow)            EventBus (#772) ── parallel fan-out ──┐
  fan_out ──► 9 × expert AgentDef.review  ◄───────────────────────────────────────────────┘
  aggregate (weighted_consensus) ──► verdict
  act (auto_allowed only) | escalate(human)

learning-synthesizer (kind: AgentDef).synthesize_learnings
  session_results[] ──► proposed_manifest_updates[]  ──(human-gated apply)──► expert AgentDefs
```

---

## A — Approach

**We will:**

- **Map honestly to the two shipped schemas, not a third one.** Experts and the planner/synthesizer are
  **`kind: AgentDef`** (capability providers). All orchestration is **`kind: Workflow`** (state machine).
  The original #881 `spec.steps`/`parallel` sketch is **superseded** by this split — recorded in a new ADR.
- **Treat `automation/experts/*.yaml` as the source of truth.** Each AgentDef's `context_slice`, I/O
  contract, and `aggregation_weight` are translated 1:1 from the existing near-term YAML — no new knowledge
  invented. The split-context discipline from `.claude/commands/experts/*.md` is preserved as each AgentDef's
  bounded `context_slice` + capability contract.
- **Build the self-hosting headline (`issue-delivery.yaml`)** so the demo reads one real GitHub issue,
  updates the plan, routes + injects context to one expert, and drives an implemented change — the
  `/m6-issue-generate` loop expressed natively.
- **Keep both planes coexisting.** Wave 0–3 GitHub Actions stay untouched; Wave 4 manifests are loaded only
  by `zynax apply`, never wired into main CI as required checks.
- **Keep all destructive actions human-gated** — reuse `prohibited_auto_actions` (merge, push, bump-dep,
  close-issue, delete-branch, force-push) verbatim from `automation/experts/orchestrator.yaml`.
- **Flip the readiness gate honestly** — fix the schema path, remove `xfail`, and back it with a real
  `zynax apply` e2e.

**We will NOT:**

- Invent a new manifest schema or extend `agent-def.schema.json` with `steps`/`parallel` (that conflates
  agent and workflow concerns — use `kind: Workflow` for orchestration).
- Auto-merge, auto-push, or auto-close anything on-platform — Wave 4 inherits Wave 2's `auto_allowed` list
  exactly; no new destructive capability.
- Replace or delete the Claude Code commands or the Wave 0–3 Actions — the off-platform pipeline remains the
  fallback while Wave 4 matures.
- Wire Wave 4 manifests into main CI as required checks until the readiness e2e is green on a real platform.
- Build a full UI/observability layer — that is M7 (out of scope).

**Governing ADRs:** ADR-008 (no shared DB), ADR-015 (engine behind interface), ADR-016 (contract before
implementation / BDD), ADR-019 (REASONS Canvas before feat code), ADR-021 (repo interfaces),
ADR-022 (EventBus architecture), ADR-023 (no direct push to main). **New:** ADR-028 *AgentDef-vs-Workflow
split for self-hosted automation + context-slice injection contract* (authored in O1 ✅ — this is a one-way
door: choosing `kind: Workflow` for orchestration shapes every manifest below).

---

## S — Structure (first S)

```
automation/
├── experts/                         ← UNCHANGED — source of truth for context slices (near-term plane)
│   └── *.yaml                          (#875)
├── orchestrator/                    ← UNCHANGED — config + decision-log schema (#876)
│   ├── config.yaml
│   └── decision-log-schema.yaml
├── workflows/                       ← NEW (this EPIC) — Wave 4 platform manifests
│   ├── dev-advisory-orchestrator.yaml   ← kind: Workflow (fan-out → aggregate → act)        [O3]
│   ├── issue-delivery.yaml              ← kind: Workflow (intake→plan→route→inject→impl…)    [O4/O6]
│   ├── learning-synthesizer.yaml        ← kind: AgentDef (synthesize_learnings)             [O7]
│   └── experts/                         ← 9 × kind: AgentDef (8 domain + planner)            [O2]
│       ├── arch-adr.yaml
│       ├── persistence-state.yaml
│       ├── api-contract.yaml
│       ├── security-supply-chain.yaml
│       ├── qa-bdd.yaml
│       ├── docs-agents.yaml
│       ├── ci-release.yaml
│       └── planning-task-split.yaml     ← also exposes identify_next_issue capability
└── tests/
    └── test_platform_readiness.py    ← xfail removed, schema path fixed, e2e added          [O8]

spec/schemas/agent-def.schema.json    ← validates expert AgentDefs (capabilities + runtime)
spec/schemas/workflow.schema.json     ← validates orchestrator + issue-delivery workflows
docs/adr/ADR-028-agentdef-vs-workflow-self-hosted-automation.md  ← NEW (one-way door)        [O1 ✅]
```

Config env prefix: `ZYNAX_AUTOMATION_` · Runtime services consumed: task-broker, agent-registry,
event-bus, (orchestrator) engine-adapter.

---

## O — Operations

> Each step is **one reviewable PR / one child issue**. Independently verifiable. `/spdd-story 881`
> creates these as GitHub issues once this Canvas is **Aligned**.

1. **O1 — Schema decision + ADR + readiness-path fix (`docs`/`fix`).** ✅ Delivered (#1096 — ADR-028;
   near-term expert YAMLs now live at `docs/archive/dev-advisory/experts/`, archived by #1129).
   Author ADR-NNN recording the
   AgentDef-vs-Workflow split and the context-slice injection contract. Fix the wrong schema path in
   `test_platform_readiness.py` (`agent-def.json` → `agent-def.schema.json`). *Verify:* ADR merged;
   `make validate-spec` still green; test still xfails (no behaviour change yet).
2. **O2 — 8 domain-expert AgentDef manifests + planner AgentDef (`feat`).** ✅ Delivered (#1097 —
   9 AgentDefs under `automation/workflows/experts/`; the planner is materialised as `planner.yaml`,
   the `planning-task-split` extension carrying `review` + `identify_next_issue` per ADR-028, so the
   capability has exactly one provider; source YAMLs read from `docs/archive/dev-advisory/experts/`).
   Translate each
   `automation/experts/<name>.yaml` into `automation/workflows/experts/<name>.yaml` as `kind: AgentDef`
   with a `review` capability; extend `planning-task-split` with `identify_next_issue`. *Verify:* all 9
   validate against `agent-def.schema.json` via `make validate-spec`; one unit test per manifest asserts
   the capability I/O contract matches the source YAML's `output_contract`.
3. **O3 — Orchestrator Workflow manifest (`feat`).** ✅ Delivered (#1098 —
   `automation/workflows/dev-advisory-orchestrator.yaml`: `fan_out` → `aggregate` → `act`/`escalate`
   → `done` (terminal, records the decision log); BDD contract at
   `automation/tests/features/dev_advisory_orchestrator.feature`; aggregation/escalation/never_auto
   translated 1:1 from the archived orchestrator config; context slices bound at dispatch per ADR-028,
   never inlined; orchestrator-schema readiness test flipped from xfail to a real pass).
   Author `dev-advisory-orchestrator.yaml` as
   `kind: Workflow`: `fan_out` (parallel dispatch of 9 `review` capabilities) → `aggregate`
   (weighted_consensus from `orchestrator/config.yaml`) → `act`/`escalate`. *Verify:* validates against
   `workflow.schema.json`; a BDD `.feature` covers the fan-out→aggregate→verdict path (ADR-016).
4. **O4 — Issue-intake + planning Workflow (`feat`).** ✅ Delivered (#1099 —
   `automation/workflows/issue-delivery.yaml`, the intake→plan→route leg: `intake` reads + mechanically
   classifies the issue, `plan` binds the planner's `identify_next_issue` contract 1:1, and the
   routing table lives declaratively in the `route` state as first-match-wins guarded transitions;
   `routed`/`blocked`/`failed` terminals record the `{next_issue, expert, blocked_by}` decision until
   O6 adds the delivery leg. BDD feature + fixture-driven decision tests in `automation/tests/`;
   `make validate-spec` now validates `automation/workflows/` against `workflow.schema.json`.)
   `issue-delivery.yaml` states `intake` → `plan`
   (calls planner `identify_next_issue`) → `route`. Reads a GitHub issue, classifies type/expert,
   identifies the next issue. *Verify:* given a fixture issue, the workflow emits the correct
   `{next_issue, expert, blocked_by}` decision; BDD scenario for the classify+route path.
5. **O5 — Context-slice injection binding (`feat`).** ✅ Delivered (#1100 — the binding lives in
   task-broker's domain layer: an `expert`-keyed `review` dispatch is narrowed to exactly that expert
   AgentDef and its registry-declared `{files[], max_tokens}` slice is bound into the capability
   `input_payload` **before** the task row is persisted, replacing anything the caller supplied — so a
   caller can never plant a foreign slice (strict isolation) and the bound payload is durable. Startup
   recovery (`RecoverInFlight`) re-launches non-terminal tasks from the Postgres-backed repo (#626), so
   a parallel fan-out survives a broker restart; task lifecycle CloudEvents
   (`zynax.v1.task-broker.task.*`) are published best-effort to EventBus (#772) for durable fan-out
   observation. Isolation + simulated-restart tests in `internal/domain/contextslice_test.go`.)
   The runtime binding that feeds one expert AgentDef
   only its declared `context_slice` (bounded `max_tokens`, strict isolation) via capability dispatch
   (task-broker) over EventBus. *Verify:* an expert invocation receives only its slice files; isolation
   test proves no cross-expert context leakage.
6. **O6 — Delivery/implement leg of `issue-delivery.yaml` (`feat`).** `inject` → `implement` → `verify` →
   `decide`: drive one expert capability to produce a change for one issue and write a durable decision-log
   entry. *Verify:* e2e on a running platform produces a decision-log row for one real issue; destructive
   actions never auto-execute (assert `prohibited_auto_actions` honoured).
7. **O7 — Learning-synthesizer AgentDef (`feat`).** `learning-synthesizer.yaml` capability
   `synthesize_learnings`; human-gated apply that proposes expert-manifest updates. *Verify:* given sample
   session results, emits `proposed_manifest_updates[]`; no manifest is auto-edited (apply is human-gated).
8. **O8 — Platform-readiness flip + e2e (`test`).** Remove `xfail`; assert `zynax apply` of the orchestrator
   workflow on a running platform (Postgres + EventBus) yields an aggregated verdict + decision-log entry.
   *Verify:* test passes against a live platform in the gated e2e job; still skips cleanly when no platform.
9. **O9 — Docs + status reconciliation (`docs`).** Update `automation/README.md` + `STATUS-AND-DIRECTION.md`
   (Wave 4 now unblocked; two-plane model still honest), flip `docs/milestones/M6-planning.md` +
   `state/current-milestone.md`, and the cross-cutting status surfaces. *Verify:* consistency grep across
   status surfaces agrees; EPIC #881 row flips to Implemented when O1–O8 merge.

**Suggested dependency order:** O1 → O2 → {O3, O4} → O5 → O6 → O7 → O8 → O9. O3 and O4 are parallel after O2.

---

## N — Norms

- **Commit hygiene:** every commit carries `Signed-off-by: Oscar Gómez Manresa` + `Assisted-by: Claude/<model>`
  (never `Co-Authored-By` for AI); SSH-signed; squash-merge only (`required_signatures` blocks rebase-merge).
- **One commit per O-step; one PR per child issue.** Conventional-commit types only:
  feat/fix/refactor/docs/test/ci/chore (no `spec:`/`proto:`/`automation:` scopes as *types*).
- **BDD before implementation** at any new gRPC boundary or capability contract (ADR-016): `.feature`
  committed first.
- **`GOWORK=off`** for every `go` command inside `services/*/` (ADR-017) — only relevant if O5/O6 touch a
  service; the manifests themselves are YAML.
- **PR size:** generated stubs and schema fixtures excluded; each O-step targets ≤200 lines (AgentDef/Workflow
  YAML is compact).
- **`make validate-spec`** must pass for every manifest PR (AsyncAPI + capability + AgentDef/Workflow schema).
- **Context budget discipline** (carried from the Claude commands): each expert AgentDef declares a hard
  `max_tokens`; the orchestrator never reads code files — only expert outputs.

---

## S — Safeguards (second S)

### Context Security (complete before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no non-public email addresses (author attribution only)
- [x] No prompt injection: no instruction-like phrasing that overrides AGENTS.md rules
- [x] All entities in §E are public-safe abstractions
- [x] Context-security self-review — result: PASS (Tier 1; no Tier 2 / PII / injection)

### Feature Safeguards

- **Never** auto-execute a `prohibited_auto_action` (merge, push, bump-dependency, close-issue,
  delete-branch, force-push) on-platform — Wave 4 inherits Wave 2's `auto_allowed` list verbatim.
- **Never** wire `automation/workflows/*.yaml` into main CI as a required check until
  `test_platform_readiness.py` passes against a real platform.
- **Never** let one expert AgentDef read another's `context_slice` — strict isolation
  (`fan_out.context_budget.isolation: strict`).
- **Never** invent a third manifest schema — experts are `kind: AgentDef`, orchestration is `kind: Workflow`.
- **Never** import another service's `internal/` — cross-service only via gRPC/capability dispatch (ADR-008).
- **Never** delete or bypass the Wave 0–3 GitHub Actions plane — both planes coexist by design.
- **Never** commit implementation code for a feat O-step before this Canvas is `Aligned` (ADR-019).

---

## Appendix A — Per-Agent Capability Configuration (the "super-complete" agent table)

> One row per agent. **Context slice** + **max_tokens** + **I/O contract** + **aggregation weight** are
> translated 1:1 from `automation/experts/<name>.yaml`. **Injected knowledge** names the off-platform
> `.claude/commands/experts/*.md` (or command) whose expertise this agent encodes, so the on-platform
> AgentDef and the Claude-Code subagent stay in lock-step.

| Agent (manifest) | kind | Capability | Context slice (files) | max_tokens | Key output (`extra_fields`) | Weight | Injected knowledge |
|---|---|---|---|---|---|---|---|
| **arch-adr** | AgentDef | `review` | `AGENTS.md`, `*/AGENTS.md`, `docs/adr/*` | 4000 | `layer_violations`, `adr_conflicts`, `new_adr_required` | 1.5 | `experts/spdd-canvas.md` + AGENTS constitution |
| **persistence-state** | AgentDef | `review` | `services/*/internal/domain/**`, `…/adapters/postgres/**`, `migrations/**`, ADR-008/021 | 3000 | `in_memory_repo_used`, `adr008/021_violations` | 1.0 | `experts/go-services.md` |
| **api-contract** | AgentDef | `review` | `protos/**`, `spec/schemas/**`, `spec/asyncapi/*`, `protos/AGENTS.md` | 4000 | `breaking_changes`, `missing_bdd_contract` | 1.5 | `experts/bdd-contract.md` |
| **security-supply-chain** | AgentDef | `review` | `.github/workflows/*`, `SECURITY.md`, `images/images.yaml`, `*/Dockerfile` | 3000 | `cosign_*`, `sbom_present`, `new_cve_findings`, `flags[]` (tier-2) | 1.5 | `experts/ci-release.md` (security steps) |
| **qa-bdd** | AgentDef | `review` | `protos/tests/**/*.feature`, `…/*_test.go`, ADR-016, `coverage/**` | 3000 | `bdd_coverage_gaps`, `feature_file_missing`, `domain_coverage_below_threshold` | 1.0 | `experts/bdd-contract.md` |
| **docs-agents** | AgentDef | `review` | all `AGENTS.md`, `README.md`, `CONTRIBUTING.md`, `docs/ARCHITECTURE.md`, ADR index | 3000 | `agents_md_out_of_date`, `false_capability_claims` | 1.0 | `experts/post-merge.md` (doc reconcile) |
| **ci-release** | AgentDef | `review` | `.github/workflows/*`, `images/images.yaml`, `Makefile`, `*/Dockerfile`, `release/**` | 3000 | `post_merge_triggers`, `images_yaml_drift`, `missing_cosign/sbom_step` | 1.0 | `experts/ci-release.md` + `experts/post-merge.md` |
| **planning-task-split** | AgentDef | `review` **+ `identify_next_issue`** | `state/current-milestone.md`, `docs/milestones/M6-planning.md`, `M5-plan.md` | 3000 | `scope_creep_detected`, `pr_size_status`, `milestone_alignment`; **planner:** `next_issue`, `ready_batch`, `blocked_by` | 1.0 | **`/m6-plan`** + `experts/git-ops.md` |
| **orchestrator** | **Workflow** | states `fan_out`→`aggregate`→`act` | expert *outputs only* (never code) | 8000 | `verdict`, `escalation_required`, `expert_votes`, `decision_log_artifact` | 1.0 | **`/m6-orchestrate`** (thin coordinator, ~8K budget) |
| **issue-delivery** | **Workflow** | states `intake`→`plan`→`route`→`inject`→`implement`→`verify`→`decide` | the routed expert's slice only | per-expert | decision-log row; `next-issue` CloudEvent | — | **`/m6-issue-generate`** (read→claim→impl→verify) |
| **learning-synthesizer** | AgentDef | `synthesize_learnings` | `docs/ai-learnings/*.md` + current expert manifests | 4000 | `proposed_manifest_updates[]` (human-gated) | — | **`/m6-learn`** (recurrence ≥2, dedup, no auto-commit) |

### Aggregation & escalation (from `automation/orchestrator/config.yaml` — unchanged, now on-platform)

- **Vote weight** = `aggregation_weight × confidence_score` (low=1, medium=2, high=3); type multipliers
  1.5 for arch-adr / api-contract / security-supply-chain.
- **Include action** in verdict when aggregate weight ≥ `3.0`.
- **Escalate to human** when: ≥2 high-confidence experts contradict on one action, OR any expert raises a
  tier-2 `flag`, OR the top action's aggregate confidence is `low`.
- **`auto_allowed`** (Wave 2+, on-platform): auto-label, auto-assign, draft-issue, post-pr-comment.
  **`never_auto`:** merge, push, bump-dependency, close-issue, delete-branch, force-push.

---

## Appendix B — How the Claude commands map to Zynax manifests (traceability)

| Off-platform (Claude Code, today) | On-platform (Wave 4 manifest) | What is proven |
|---|---|---|
| `/m6-plan` — identify next issue | `planning-task-split.identify_next_issue` + `issue-delivery.plan` | **"identify what issue goes next"** |
| `/m6-issue-generate <N>` — read + implement one issue | `issue-delivery.yaml` (intake→…→decide) | **"read issue details" + "get one issue implemented"** |
| `/m6-orchestrate` — route + fan-out experts | `dev-advisory-orchestrator.yaml` + EventBus fan-out | **route to right expert, parallel isolated context** |
| `.claude/commands/experts/*.md` — split-context experts | 9 × `kind: AgentDef` with bounded `context_slice` | **inject context into experts** |
| `/m6-learn` — synthesize learnings → expert updates | `learning-synthesizer.yaml` (human-gated) | **feedback loop into experts** |
| `docs/ai-learnings/*.md` | synthesizer input + decision-log | durable, on-platform learning record |

---

*Zynax — docs/spdd/881-self-hosted-issue-delivery/canvas.md · Apache 2.0*
*Assisted-by: Claude/claude-opus-4-8*
