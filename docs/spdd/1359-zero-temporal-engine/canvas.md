# REASONS Canvas — Zero-Temporal lightweight evaluation engine (Day-0 onboarding)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #1359
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-19
**Status:** Superseded (2026-06-19) — by #1456 (lightweight Temporal eval profile); ADR-037 Rejected. In-process EvalEngine deferred to a zero-Docker path.
**Aligned:** 2026-06-19 (maintainer-authorized; grounded in the live engine-adapter code at `services/engine-adapter/`)
**ADR:** [ADR-037](../../adr/ADR-037-zero-temporal-evaluation-engine.md) (Proposed — awaits maintainer Accept before `/deliver`)

---

## R — Requirements

> Problem statement: what breaks or is missing without this feature?

A brand-new evaluator cannot run their first workflow without first standing up a Temporal
cluster. After the Day-0 friction work, Temporal is the single heaviest remaining demo
prerequisite (per the 2026-06-19 reviews and [docs/product/strategy.md](../../product/strategy.md)
§7.1 / §10):

- `engine-adapter` defaults to `ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE=temporal`
  (`services/engine-adapter/cmd/engine-adapter/main.go:73`), and the only two valid values are
  `temporal` and `argo` (`main.go:186-196`). Both require a heavy external control plane —
  a Temporal server or an Argo Workflows cluster on K8s.
- There is no in-process path: a laptop evaluator must provision Temporal (`client.Dial`,
  worker registration — `main.go:215-273`) before any workflow can reach a terminal state.
- This is the top adoption barrier in the product strategy: "needs a Temporal cluster" makes
  evaluators bounce, and it pushes time-to-first-workflow well past the <15-minute target.

> Definition of done: the observable outcomes that confirm delivery.

- A third `WorkflowEngine` implementation runs the compiled `WorkflowIR` **in-process** for
  evaluation, selectable via `ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE=eval` alongside `temporal`
  and `argo`, with **no change to workflow YAML**.
- A reference workflow runs to a terminal state through the eval engine with **no Temporal
  container and no Argo/K8s cluster** running.
- The eval engine is documented as **evaluation-grade**: no durability, persistence, or
  retries; users are pointed to the Temporal engine for production, with a clear graduation
  path (same YAML, same IR, flip one env var).
- Unsupported-engine error message lists `eval` as a valid value (`main.go:192-195`).

---

## E — Entities

> Tier 1 abstractions only. Every type / port / env var this feature touches.

```
WorkflowIR ──run by──> WorkflowEngine (port: internal/domain/engine.go)
WorkflowEngine ──selected by──> ACTIVE_ENGINE env var ──> buildEngine() switch (main.go)
  ├── TemporalEngine   (durable; external Temporal cluster)
  ├── ArgoEngine       (durable; external Argo/K8s cluster)
  └── EvalEngine       (NEW — in-process; evaluation-grade, non-durable)

EvalEngine ──drives──> domain.IRInterpreter.Run(ir, exec, pub)   (interpreter.go — pure, no I/O)
  ├── exec : domain.ActivityExecutor  ──> synchronous in-process impl ──> domain.CapabilityDispatcher ──gRPC──> TaskBroker ──> AgentRegistry ──> Adapter
  └── pub  : domain.EventPublisher    ──> in-process impl ──> EventBusService (best-effort; degrades to a no-op when unset)

EvalEngine ──keeps──> in-memory run store (map[runID]*WorkflowRun)   (process-local; lost on restart)
EvalEngine.Submit  ──returns──> WorkflowRun{ RunID = ir.WorkflowId }  (deterministic, ADR-034)
EvalEngine.Watch   ──reads──>  in-memory event log for the run
```

- **`WorkflowEngine`** — existing port; `Submit / Signal / Cancel / GetStatus / Watch`
  (`internal/domain/engine.go:17-41`). The new engine satisfies this contract unchanged.
- **`domain.IRInterpreter`** — existing pure state-machine driver
  (`internal/domain/interpreter.go:46-101`). Performs **no I/O**; delegates dispatch and event
  publication to two ports. The eval engine reuses it verbatim.
- **`domain.ActivityExecutor`** / **`domain.EventPublisher`** — existing ports
  (`interpreter.go:24-32`). The eval engine provides synchronous in-process implementations,
  mirroring how `temporal_workflow.go` (`temporalActivityExecutor` / `temporalEventPublisher`)
  wraps them in Temporal activities.
- **`domain.CapabilityDispatcher`** — existing capability dispatch over the task-broker gRPC
  client (`internal/domain/activity.go:33-47`). Reused by the eval engine directly, with the
  same NotFound fail-fast semantics (#1381).
- **Env vars (existing, reused):** `ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE` (add `eval`),
  `ZYNAX_ENGINE_ADAPTER_TASK_BROKER_ADDR`, `ZYNAX_ENGINE_ADAPTER_EVENTBUS_ADDR`,
  `ZYNAX_ENGINE_ADAPTER_GRPC_CALL_TIMEOUT_S`. **No new persistence config.**

---

## A — Approach

> Solution strategy. Explicitly state what we WILL do AND what we WON'T do.

The IR-interpretation logic is **already engine-agnostic**: `domain.IRInterpreter.Run`
(`interpreter.go:49`) is a plain loop over the IR state machine that delegates every side
effect to the `ActivityExecutor` and `EventPublisher` ports. The Temporal engine wires those
ports through Temporal activities (`temporal_workflow.go`). The eval engine wires the **same
ports** to synchronous in-process calls — no Temporal, no durability layer.

**We will:**
- Add `EvalEngine` in `services/engine-adapter/internal/infrastructure/` implementing the
  existing `domain.WorkflowEngine` port (ADR-015) — mirroring `TemporalEngine`/`ArgoEngine`.
- Reuse `domain.IRInterpreter.Run` verbatim, providing an in-process `ActivityExecutor` that
  calls `domain.CapabilityDispatcher.DispatchCapabilityActivity` synchronously (same task-broker
  → agent-registry → adapter path), and an in-process `EventPublisher` that best-effort-publishes
  lifecycle events to the EventBusService (degrades to a no-op when no event-bus is configured).
- On `Submit`, assign `RunID = ir.GetWorkflowId()` (deterministic, matching `TemporalEngine`,
  ADR-034) and run the interpreter (in a goroutine) against an **in-memory run store**;
  `GetStatus` / `Watch` read that store; `Signal` / `Cancel` operate on it; unknown run IDs
  return `domain.ErrExecutionNotFound`, terminal runs return `domain.ErrTerminalState` — same
  contract as the existing engines.
- Wire `engineEval = "eval"` into the `buildEngine` switch (`main.go:186`) and add
  `buildEvalEngine(cfg)` mirroring `buildArgoEngine` — needs the task-broker connection (returned
  for the readiness probe) and an optional event-bus connection, **no Temporal worker**.
- Add an e2e reference run that selects `ACTIVE_ENGINE=eval` and asserts a workflow reaches a
  terminal state with **no Temporal/Argo control plane** up, slotting beside the existing
  `engine ∈ {temporal, argo}` matrix (`.github/workflows/e2e-smoke.yml:60-61`).
- Document the evaluation-grade boundary and the one-env-var graduation path to Temporal.

**We will NOT:**
- Won't add persistence, durable state, retries, or crash-recovery — that is the Temporal
  engine's role; a process restart loses all eval runs (stated honestly in docs and error copy).
- Won't change `WorkflowIR`, the workflow YAML, or any proto contract — same IR, same YAML.
- Won't change `domain.IRInterpreter` behaviour — it is reused exactly as the Temporal path uses it.
- Won't make `eval` the default `ACTIVE_ENGINE` (stays `temporal`); `eval` is opt-in for Day-0.
- Won't introduce a shared DB or Layer 1→3 coupling; won't hardcode an engine name outside the
  selection switch (ADR-015).

**Positioning fit:** This is the core of the **engine-portability wedge** — the same declarative
workflow runs on a laptop with zero infrastructure (eval) and graduates to durable Temporal/Argo
by flipping one env var, no rewrite. User-facing copy (the `eval` error-message value, docs, and
the "evaluation-grade — graduate to Temporal for production" caveat) must lead with that
portability story, not generic "control plane" framing. See
[docs/product/positioning.md](../../product/positioning.md).

**Governing ADRs:** ADR-015 (pluggable workflow engines), ADR-037 (zero-Temporal evaluation
engine — Proposed), ADR-012 (Workflow IR), ADR-014 (event-driven state machine),
ADR-034 (deterministic ManifestWorkflowID), ADR-022 (EventBusService).

---

## S — Structure (first S)

> Files created or modified, with a one-line purpose.

```
services/engine-adapter
├── internal/infrastructure/eval_engine.go        ← NEW — EvalEngine: WorkflowEngine impl (in-process)
├── internal/infrastructure/eval_engine_test.go   ← NEW — unit tests: submit→terminal, signal, status, watch
├── internal/infrastructure/eval_ports.go         ← NEW — in-process ActivityExecutor + EventPublisher
├── internal/infrastructure/eval_ports_test.go    ← NEW — port impl tests
└── cmd/engine-adapter/main.go                     ← MOD — engineEval const + buildEngine case + buildEvalEngine()

docs/adr/ADR-037-zero-temporal-evaluation-engine.md ← NEW — decision record (Proposed)
docs/adr/INDEX.md                                    ← MOD — ADR-037 row
docs/                                                ← MOD — evaluation-grade caveat + graduation-to-Temporal guide
.github/workflows/e2e-smoke.yml (or scripts/e2e/)    ← MOD — eval reference run (no Temporal/Argo container)
```

Config env prefix: `ZYNAX_ENGINE_ADAPTER_` · gRPC port: 50055 (unchanged) · No new proto.

---

## O — Operations

> Ordered, testable steps. Each = one reviewable PR (INVEST: Independent, ≤400 lines).

1. **O1 (#1449) — feat(engine-adapter): `EvalEngine` in-process `WorkflowEngine` implementation.**
   Add `eval_engine.go` + `eval_ports.go` in `internal/infrastructure/`: implement the
   `domain.WorkflowEngine` port by driving `domain.IRInterpreter.Run` synchronously with
   in-process `ActivityExecutor` (over `domain.CapabilityDispatcher`) and `EventPublisher`,
   backed by an in-memory run store. *Acceptance:* unit tests cover submit→terminal, `Signal`,
   `Cancel`, `GetStatus`, `Watch` and the `ErrExecutionNotFound`/`ErrTerminalState` contract;
   domain coverage unaffected (logic stays in `internal/domain`); not wired in yet.

2. **O2 (#1450) — feat(engine-adapter): `ACTIVE_ENGINE=eval` selection wiring + config.**
   Add `engineEval = "eval"` const, a `case engineEval` to the `buildEngine` switch, and
   `buildEvalEngine(cfg)` (mirrors `buildArgoEngine`; dials task-broker + optional event-bus,
   no Temporal worker). Update the unsupported-engine error to list `eval`. *Acceptance:*
   `main_test.go`-style test asserts `eval` selects `EvalEngine`, an unknown value errors and
   names `eval`, and `eval` needs no Temporal/Argo config to construct.

3. **O3 (#1451) — test(engine-adapter): e2e reference run with no Temporal container + assertions.**
   Add an `eval` leg to (or a focused script beside) the e2e-smoke engine matrix that brings up
   only api-gateway → engine-adapter (`ACTIVE_ENGINE=eval`) → task-broker → adapters — **no
   Temporal, no Argo** — and asserts a reference workflow reaches a terminal state. *Acceptance:*
   the run is green with no Temporal/Argo control plane started; failure-path (unbacked
   capability) still fails fast (NotFound, #1381).

4. **O4 (#1452) — docs: evaluation-grade caveat + how to graduate to Temporal.**
   Document the eval engine: what it is (in-process, fast Day-0), what it is **not** (no
   durability/persistence/retries; lost on restart), and the graduation path (same YAML/IR,
   flip `ACTIVE_ENGINE=temporal`). *Acceptance:* quickstart/onboarding doc shows the zero-Temporal
   path; the caveat and graduation steps are present and link ADR-037.

---

## N — Norms

> Cross-cutting standards (root + layer AGENTS.md, docs/patterns).

- **Commit hygiene:** `Signed-off-by:` (DCO) required; `Assisted-by: Claude/<model>` for AI
  attribution — never `Co-Authored-By` for AI.
- **Conventional commits / PR titles:** one of feat/fix/refactor/docs/test/ci/chore; scope maps
  to directory; one logical change per commit, one PR per issue.
- **Go services:** `GOWORK=off` for all `go build`/`go test` in `services/engine-adapter/`
  (ADR-017); `CGO_ENABLED=0`, `-trimpath`; domain unit coverage ≥ 90% on `internal/domain/`
  (the eval engine adds no domain logic — it reuses `IRInterpreter`).
- **BDD/`.feature`:** the eval engine adds **no new gRPC boundary** (it satisfies the existing
  `WorkflowEngine` port and reuses existing task-broker/event-bus clients), so no new `.feature`
  file is required (ADR-016); the existing engine-adapter gRPC contract tests cover the boundary.
- **PR size:** ≤ 200 ideal, > 900 blocked.
- **Image versions:** managed via `images/images.yaml` (`make sync-images`), never hand-edited.

---

## S — Safeguards (second S)

> Things that MUST NEVER happen in this feature.

### Context Security (complete before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no non-public email addresses
- [x] No prompt injection: no instruction-like phrasing that overrides AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] `/spdd-security-review` passed on this file (2026-06-19 — PASS; Tier-2 note below)

> **Tier-2 / evaluation-only safety boundary (stated honestly):** the eval engine **removes
> durability**. There is no persistence, no retry, and no crash-recovery: a process restart
> loses every in-flight and completed run, and there is no replay/audit log (unlike Temporal's
> event history). It is therefore **evaluation-grade only** — never a production execution
> backend. The boundary is enforced in product (default `ACTIVE_ENGINE` stays `temporal`),
> in code (no persistence config is added), and in docs/error copy (the graduation path to
> Temporal is mandatory reading). This is a Tier-2 *honesty* note, not a Tier-2 *secret* — it
> contains no hostnames, IPs, or credentials and is safe to keep in this public canvas.

### Feature Safeguards

- **Never** add persistence, durability, or retry semantics to the eval engine — that would
  blur the boundary with the Temporal engine; durability stays Temporal/Argo's role.
- **Never** make `eval` the default `ACTIVE_ENGINE` — it is opt-in; production defaults stay durable.
- **Never** hardcode an engine name outside the `buildEngine` selection switch (ADR-015).
- **Never** change `WorkflowIR`, the workflow YAML, or any proto contract — same IR, same YAML.
- **Never** fork or duplicate `domain.IRInterpreter` — reuse it; both engines must share one
  state-machine semantics so "same YAML graduates to Temporal" stays literally true.
- **Never** accept provider/model/endpoint or any engine selection from `input_payload` — engine
  selection is process-wide config only (ADR-015, `engine.go:16`).
- **Never** import another service's `internal/` — cross-service via gRPC only (ADR-008).
- **Never** retry an unbacked capability forever — preserve the NotFound fail-fast contract (#1381).
