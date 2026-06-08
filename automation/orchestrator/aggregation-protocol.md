# Orchestrator Aggregation Protocol

> **Issue:** #876 (M6.DevAuto — DevAuto.3)
> **Plane:** near-term
> **Depends-on:** `automation/experts/*.yaml` (DevAuto.2 — #875 CLOSED)

This document describes how the DevAuto orchestrator weighs expert recommendations,
decides when to act vs escalate, records reasons, and manages per-expert context budgets.

---

## 1. How the Orchestrator Weighs Expert Recommendations

All 9 domain experts are invoked in **parallel** (fan-out mode, see `config.yaml`).
Once all outputs are collected (or `timeout_seconds: 300` is reached), the orchestrator
applies a **weighted consensus** strategy.

### Vote-weight formula

```
vote_weight = aggregation_weight × confidence_score
```

Where:

| Confidence | Score |
|------------|-------|
| `low`      | 1     |
| `medium`   | 2     |
| `high`     | 3     |

`aggregation_weight` is declared per expert in `automation/experts/<name>.yaml`
(field `aggregation_weight`). High-impact experts (security, arch, contract) carry `1.5`;
all others carry `1.0`.

### Effective weight table (snapshot)

| Expert               | aggregation_weight | Max vote_weight (high conf) |
|----------------------|--------------------|-----------------------------|
| security-supply-chain | 1.5               | 4.5                         |
| arch-adr             | 1.5                | 4.5                         |
| api-contract         | 1.5                | 4.5                         |
| qa-bdd               | 1.0                | 3.0                         |
| ci-release           | 1.0                | 3.0                         |
| persistence-state    | 1.0                | 3.0                         |
| docs-agents          | 1.0                | 3.0                         |
| planning-task-split  | 1.0                | 3.0                         |

An action must reach `aggregate_weight_minimum: 3.0` (from `config.yaml`) in summed
expert votes before it is included in the final verdict.

### Conflict resolution baseline

When two experts recommend contradictory actions, `conflict_resolution: highest_confidence_wins`
applies: the expert with the higher `vote_weight` wins and its action is included in the verdict.
This is the **default path**. Escalation overrides this when the thresholds below are reached.

---

## 2. When the Orchestrator Decides vs Escalates

Escalation is triggered when **any** of the following conditions hold
(from `config.yaml aggregation.escalation_threshold`):

| Condition | Threshold | Effect |
|-----------|-----------|--------|
| `conflicting_high_confidence` | ≥2 high-confidence experts disagree | Escalate |
| `any_critical_flag` | Any expert's `flags[]` is non-empty | Escalate |
| `top_action_confidence` | Aggregate confidence for top action is `low` | Escalate |

When none of these conditions hold, the orchestrator **decides autonomously** and,
in Wave 2+, may execute actions from `human_in_the_loop.auto_allowed`.

When escalation is triggered, `aggregated_verdict.escalated = true` and
`human_action_required.required = true` are written to the decision log. No auto-action
is taken. A PR comment is posted (this itself is in `auto_allowed`).

### Never-auto actions

Regardless of wave or confidence, the following actions are **never executed automatically**
(from `config.yaml human_in_the_loop.never_auto`):

- `merge`
- `push`
- `bump-dependency`
- `close-issue`
- `delete-branch`
- `force-push`

---

## 3. How Reasons Are Recorded (ADR vs Decision-Log Entry)

### Decision-log entry

Every orchestrator run — whether it decides or escalates — writes a JSON decision-log
document conforming to `decision-log-schema.yaml`. This records:

- Per-expert outputs (`expert_outputs[].reasons_decisions`)
- The aggregated verdict with full `expert_votes` map
- The `human_action_required` block
- `execution_result` once post-action is complete

Decision-log files live in `.automation/decision-logs/` (gitignored) and are uploaded
as GitHub Actions artifacts (`retention_days: 90`).

### ADR

An ADR is created only when the decision is a **one-way door**: a change that another
engineer would reverse without knowing the rationale (per `AGENTS.md`). Orchestrator
auto-actions do not by themselves require an ADR. A human reviewing an escalated run
may choose to create an ADR if the resolution establishes a precedent.

### Distinction

| Record type      | When created              | Who creates it      | Location                              |
|------------------|---------------------------|---------------------|---------------------------------------|
| Decision-log     | Every orchestrator run    | Orchestrator (auto) | `.automation/decision-logs/` + CI artifact |
| ADR              | One-way design decisions  | Human engineer      | `docs/adr/`                           |

---

## 4. Conflict Resolution Examples

### Example 1 — Single-expert flag, immediate escalation

**Scenario:** A PR modifies `.github/workflows/service-release.yml` to add a new
`env:` block. The `security-supply-chain` expert raises a tier-2 flag
`"secret_exposure_risk"` because the block references `${{ env.REGISTRY_PASSWORD }}`.

**Expert outputs:**

| Expert               | Confidence | Recommended action         | flags                    |
|----------------------|------------|----------------------------|--------------------------|
| security-supply-chain | high      | block-pr / request-changes | `["secret_exposure_risk"]` |
| ci-release           | medium     | request-changes            | `[]`                     |
| all others           | —          | no-action or not-applicable | `[]`                    |

**Aggregation:**

- `any_critical_flag: true` → `"secret_exposure_risk"` is non-empty → **escalate immediately**
- `conflicting_high_confidence` threshold: not reached (only one high-confidence expert)
- Decision: `escalated = true`, `human_action_required.required = true`
- Reason: `"Tier-2 flag raised by security-supply-chain: secret_exposure_risk"`
- No auto-action taken. A PR comment is posted with the flag details.

---

### Example 2 — Contradictory high-confidence experts, escalation

**Scenario:** A PR adds a new gRPC field to an existing proto message.
`api-contract` and `arch-adr` give contradictory high-confidence recommendations.

**Expert outputs:**

| Expert        | Confidence | vote_weight | Recommended action               |
|---------------|------------|-------------|----------------------------------|
| api-contract  | high       | 1.5 × 3 = 4.5 | block-pr: breaking-change detected |
| arch-adr      | high       | 1.5 × 3 = 4.5 | pass: field is additive, non-breaking |
| qa-bdd        | medium     | 1.0 × 2 = 2.0 | request-changes: missing BDD file |
| others        | low/medium | < 3.0       | no-action                        |

**Aggregation:**

- `conflicting_high_confidence` = 2 experts disagree (api-contract says block, arch-adr says pass)
- Threshold `conflicting_high_confidence: 2` is met → **escalate**
- `highest_confidence_wins` would pick neither because both are equal — escalation takes over
- Decision: `escalated = true`
- Human reviews the contradiction and decides: the proto field is in a new message (additive)
  → `arch-adr` reasoning was correct → `pass` applied manually

---

### Example 3 — Clear majority, no escalation (decision path)

**Scenario:** A PR updates a Python adapter's unit test file only (no proto, no workflow,
no security-sensitive path).

**Expert outputs:**

| Expert               | Confidence | vote_weight | Recommended action |
|----------------------|------------|-------------|--------------------|
| qa-bdd               | high       | 1.0 × 3 = 3.0 | pass              |
| docs-agents          | high       | 1.0 × 3 = 3.0 | pass              |
| planning-task-split  | medium     | 1.0 × 2 = 2.0 | pass              |
| ci-release           | low        | 1.0 × 1 = 1.0 | pass              |
| security-supply-chain | low       | 1.5 × 1 = 1.5 | pass              |
| api-contract         | low        | 1.5 × 1 = 1.5 | pass              |
| arch-adr             | low        | 1.5 × 1 = 1.5 | pass              |
| persistence-state    | low        | 1.0 × 1 = 1.0 | pass              |

**Aggregation:**

- All experts recommend `pass`; no flags; no contradictions
- `action = pass` total weight = 3.0 + 3.0 + 2.0 + 1.0 + 1.5 + 1.5 + 1.5 + 1.0 = 15.5
  → exceeds `aggregate_weight_minimum: 3.0`
- `conflicting_high_confidence` = 0 → no escalation
- `any_critical_flag` = false → no escalation
- Aggregate confidence = high (majority of weight from high-confidence votes)
- Decision: `escalated = false`, verdict `pass`, `human_action_required.required = false`
- Auto-actions executed (Wave 2+): `auto-label` → adds `approved` label

---

## 5. Context-Budget Management

Each expert's context window is **strictly isolated**: no expert sees another expert's
context files. Expert outputs (structured JSON) are collected by the orchestrator and
fed only to the orchestrator expert's context slice.

### Per-expert context slots

| Expert               | max_tokens | Context files (from expert YAML context_slice) |
|----------------------|------------|------------------------------------------------|
| api-contract         | 4,000      | protos, buf config, AsyncAPI spec, capability schemas |
| arch-adr             | 4,000      | ADRs, AGENTS.md, services/, docs/adr/         |
| ci-release           | 3,000      | .github/workflows/, images.yaml, Makefile, Dockerfiles |
| docs-agents          | 2,000      | docs/, AGENTS.md files, README patterns       |
| persistence-state    | 3,000      | DB migrations, state/, service data models    |
| planning-task-split  | 3,000      | state/current-milestone.md, issues, SPDD canvases |
| qa-bdd               | 3,000      | protos/tests/, *.feature files, BDD contracts |
| security-supply-chain | 3,000     | .github/workflows/, SECURITY.md, Dockerfiles, images.yaml |
| orchestrator         | 8,000      | expert outputs (JSON), config.yaml, decision-log-schema.yaml |

**Total maximum tokens per run:** 4000 + 4000 + 3000 + 2000 + 3000 + 3000 + 3000 + 3000 + 8000 = **33,000**

Each expert slot is independent: if an expert's declared context files exceed its
`max_tokens` budget, the `overflow_policy: truncate_oldest_files` in `config.yaml`
drops the least recently modified files first until the budget is met. This ensures
no expert exceeds its declared token cap.

The orchestrator expert receives the JSON-serialized outputs of all 8 domain experts
(not raw code) within its 8,000-token window, which is sufficient because each expert
output is bounded by its `summary` (≤200 words) plus structured action lists.
