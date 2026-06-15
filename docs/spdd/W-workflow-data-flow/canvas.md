# REASONS Canvas — EPIC W: Workflow Data-Flow (output/input bindings)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content belongs in `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #1167 · **Milestone:** M7 (v0.6.0)
**Author:** M7 program plan
**Date:** 2026-06-15
**Status:** Draft

---

## R — Requirements

- **Problem:** the compiler rejects `output:` on actions/states (`"output: which is not yet
  implemented; … upgrade to M7+"`), so no workflow can pass data between states. Every real
  pipeline (research → summarize, build → test → deploy) is currently inexpressible.
- A state's action must be able to **publish named outputs**; a later state must be able to
  **consume** them as inputs by reference.
- Data must be **workflow-scoped** and **explicitly read/written** — no implicit global mutable state.
- **Done when:** `research-task.yaml` and `code-review.yaml` apply and run to a terminal state with
  a downstream state consuming an upstream output; a BDD scenario covers data-flow; domain cov ≥90%.

---

## E — Entities

```
WorkflowIR
├── State
│   ├── Action.output_bindings   ← named outputs an action publishes (key → source path)
│   └── Action.input_bindings    ← inputs an action consumes (key → reference into data context)
└── WorkflowDataContext          ← workflow-scoped key/value store, written by outputs, read by inputs
        └── DataReference          ← typed reference ("$.states.search.output.results")
```

Relationships: `Action` *writes* `WorkflowDataContext` via `output_bindings`; `Action` *reads* it
via `input_bindings`. The context lives for the workflow run only.

---

## A — Approach

**We will:**
- Add **additive** proto fields for `output_bindings` / `input_bindings` to `WorkflowIR`
  (backward-compatible; manifests without them compile unchanged).
- Define a **minimal binding model**: literal values + JSON-path references into the data context.
- Compile bindings to IR, validate that every input reference resolves to a declared upstream output.
- Thread a workflow-scoped data context through the `IRInterpreterWorkflow` in engine-adapter.

**We will NOT:**
- Add an expression/transform language (filters, math, templating) — **deferred to M-dx**.
- Allow cross-workflow or cross-namespace data sharing.
- Persist the data context beyond the run (no durable data store in M7).

**Governing ADRs:** ADR-029 (workflow data-flow semantics — this EPIC), ADR-012 (Workflow IR),
ADR-014 (event-driven state machine), ADR-015 (pluggable engines), ADR-016 (.feature before impl).

---

## S — Structure (first S)

```
protos/zynax/v1/workflow_compiler.proto   ← add output_bindings/input_bindings (additive)
services/workflow-compiler/internal/domain/ir/        ← compile + validate bindings
services/workflow-compiler/internal/domain/validators/ ← reference-resolution validation
services/engine-adapter/internal/domain/              ← WorkflowDataContext + interpreter threading
protos/tests/workflow_compiler_service/               ← .feature for data-flow (new behaviour)
```

Config env prefix: `ZYNAX_WC_` / `ZYNAX_ENGINE_ADAPTER_` · No new ports.

---

## O — Operations (stories — `spdd-story` form)

**GitHub issues:** W.1 #1175 · W.2 #1176 · W.3 #1177 · W.4 #1178 · W.5 #1179 (epic #1167)

**W.1 — ADR: data-flow semantics & scoping model**
- As a `maintainer`, I want a recorded decision on the binding model so that the contract is stable before code.
- Size: S · Type: `docs`/`adr-proposal`
- Acceptance:
  - [ ] ADR-029 committed (Proposed→Accepted) defining binding syntax, scoping, and non-goals
  - [ ] `buf breaking` implications documented (fields are additive)
- Out of scope: implementation. Dependencies: none (gates W.2).

**W.2 — Proto: output/input binding fields + `.feature`**
- As a `workflow author`, I want IR fields for published outputs and consumed inputs so that states can exchange data.
- Size: M · Type: `feat` (new gRPC behaviour → `/spdd-api-test` first)
- Acceptance:
  - [ ] `output_bindings`/`input_bindings` added to the IR proto; stubs regenerated
  - [ ] `.feature` scenario committed before implementation (ADR-016)
  - [ ] `buf breaking` passes (additive only)
- Out of scope: compiler logic. Dependencies: W.1.

**W.3 — Compiler: compile + validate bindings; lift the rejection**
- As a `workflow author`, I want `output:` accepted and validated so that valid manifests compile.
- Size: M · Type: `feat`
- Acceptance:
  - [ ] `output:` no longer rejected; bindings compiled into IR
  - [ ] unresolved input references produce a clear `COMPILATION_ERROR` with line number
  - [ ] domain coverage ≥90%
- Out of scope: execution. Dependencies: W.2.

**W.4 — Engine-adapter: workflow-scoped data context**
- As a `workflow run`, I want outputs stored and inputs resolved at execution so that data flows state→state.
- Size: M · Type: `feat`
- Acceptance:
  - [ ] interpreter writes action outputs into a run-scoped context and resolves inputs from it
  - [ ] missing/typed-mismatch reference fails the run with a structured error
  - [ ] domain coverage ≥90%
- Out of scope: persistence. Dependencies: W.3.

**W.5 — End-to-end: real workflows run green**
- As a `developer`, I want the example workflows to actually run so that data-flow is proven end-to-end.
- Size: S · Type: `test`
- Acceptance:
  - [ ] `apply research-task.yaml` reaches terminal with `summarize` consuming `search` output
  - [ ] `apply code-review.yaml` runs green
  - [ ] e2e assertion added to the suite
- Out of scope: new examples (EPIC T). Dependencies: W.4.

**Order:** W.1 → W.2 → W.3 → W.4 → W.5 (strictly sequential — keystone path).

---

## N — Norms

- `Signed-off-by:` + `Assisted-by: Claude/<model>` on every commit; one logical change per commit.
- `.feature` committed before any gRPC boundary implementation (ADR-016); `GOWORK=off` for all `go` in services (ADR-017).
- Proto changes additive only; `buf breaking` is a CI gate.
- Conventional commit types limited to feat/fix/refactor/docs/test/ci/chore.

## S — Safeguards (second S)

### Context Security (complete before committing this Canvas)
- [ ] No Tier 2 content (no hostnames/IPs/credentials/deployment specifics)
- [ ] No PII; all E-section entities are public-safe abstractions
- [ ] No prompt-injection phrasing
- [ ] `/spdd-security-review` passed — result: PENDING (run before Aligned)

### Feature Safeguards
- Never make proto changes non-additive — backward compatibility is mandatory (ADR-012, `buf breaking`).
- Never let one workflow read another's data context — strict run-scoping (ADR-008 spirit: no shared state).
- Never introduce an expression language in M7 — bindings are literal/path refs only (scope guard).
- Never hardcode engine behaviour — data context lives behind the `WorkflowEngine` interface (ADR-015).
