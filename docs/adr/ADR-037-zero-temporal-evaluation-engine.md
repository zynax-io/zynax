# ADR-037: Zero-Temporal in-process evaluation engine

**Status:** Rejected  **Date:** 2026-06-19
**Rejected:** 2026-06-19 — superseded by #1456 (lightweight Temporal eval profile); see Update below.
**Related:** ADR-015 (pluggable workflow engines — this adds a third engine behind the same
port), ADR-012 (Workflow IR — the engine-agnostic IR this engine interprets), ADR-014
(event-driven state machine), ADR-034 (deterministic ManifestWorkflowID), ADR-022 (EventBusService)

> Scoping canvas: [docs/spdd/1359-zero-temporal-engine/canvas.md](../spdd/1359-zero-temporal-engine/canvas.md) · Issue #1359 (M7)

---

## Context

After the Day-0 onboarding work, **Temporal is the heaviest remaining prerequisite** to run a
first workflow. The `engine-adapter` ships two `WorkflowEngine` implementations selected by
`ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE` (ADR-015): `temporal` (default) and `argo`. Both require a
heavy external control plane — a Temporal server or an Argo Workflows cluster on Kubernetes —
before any workflow can reach a terminal state.

For a new evaluator on a laptop this is the top adoption barrier
([docs/product/strategy.md](../product/strategy.md) §7.1 / §10, and the 2026-06-19 reviews):
"needs a Temporal cluster" makes evaluators bounce and pushes time-to-first-workflow well past
the <15-minute target.

A key structural fact makes a lightweight engine cheap: the IR-interpretation logic is already
engine-agnostic. `domain.IRInterpreter.Run` is a pure state-machine loop that performs no I/O —
it delegates every side effect to two ports, `ActivityExecutor` (capability dispatch) and
`EventPublisher` (lifecycle events). The Temporal engine wires those ports through Temporal
activities; nothing about the interpreter requires a durable backend.

A new engine is a **one-way door**: once `eval` is a documented, selectable engine value with a
public evaluation/production boundary, removing or redefining it breaks user expectations and the
"same YAML graduates to Temporal" promise. It therefore warrants an ADR before implementation.

---

## Decision

We will add a third `WorkflowEngine` implementation, **`EvalEngine`**, that interprets the
compiled `WorkflowIR` **in-process** for evaluation, selectable via
`ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE=eval` alongside `temporal` and `argo`.

- **Reuse, do not fork, `domain.IRInterpreter`.** The eval engine drives `IRInterpreter.Run`
  with synchronous in-process implementations of `ActivityExecutor` (calling the existing
  `domain.CapabilityDispatcher` over the task-broker gRPC client) and `EventPublisher`
  (best-effort lifecycle events to the EventBusService, degrading to a no-op when unset). Both
  engines share one state-machine semantics, so "the same YAML runs on Temporal" stays literally true.
- **No durability.** Runs are tracked in an in-memory store keyed by run id; there is no
  persistence, no retry, and no crash-recovery. A process restart loses all runs.
- **Same contract.** `Submit` assigns `RunID = ir.WorkflowId` (deterministic, ADR-034);
  `GetStatus` / `Watch` / `Signal` / `Cancel` honour the existing `WorkflowEngine` port,
  including `ErrExecutionNotFound` and `ErrTerminalState`.
- **Opt-in.** The default `ACTIVE_ENGINE` stays `temporal`; `eval` is selected explicitly for Day-0.
- **No contract changes.** No change to `WorkflowIR`, workflow YAML, or any proto.

---

## Rationale

| Option | Assessment |
|--------|------------|
| **In-process eval engine behind `WorkflowEngine`** | ✗ **Deferred** — Sound, but only strictly needed for a *zero-Docker, zynax-binary-only* run. For the compose Day-0 path a stripped Temporal profile (next row) achieves the same with no new engine to maintain. |
| Lightweight Temporal profile — `temporal server start-dev` (in-memory, no external DB) + retries off (`ZYNAX_ENGINE_MAX_ACTIVITY_ATTEMPTS=1`) | ✅ **Chosen (#1456)** — Config-only: a compose overlay + 2 existing env vars, ZERO new engine code, reuses the proven `TemporalEngine`. Removes the database (the real Day-0 weight: today Temporal = auto-setup + dedicated Postgres + UI) and gives eval-grade fail-fast behavior. No semantic-drift risk. |
| Embedded Temporal / Templite (in-process Temporal-lite) | ✗ Rejected — heavier dependency surface, replay/persistence semantics we explicitly do not want for an evaluation engine, and a larger maintenance burden than the few hundred lines this engine needs. |
| Require Temporal (status quo — "do nothing") | ✗ Rejected — leaves the top adoption barrier in place; evaluators bounce before the first workflow runs. |
| Shell out to an external lightweight runner | ✗ Rejected — adds a process/IPC boundary and a new artifact to ship and version, for no benefit over an in-process Go implementation behind the existing port. |

---

## Consequences

- **Positive:** A new user runs a real workflow end-to-end on a laptop with **no Temporal/K8s
  cluster** — same declarative YAML, same IR, same capability dispatch path. Graduation to
  production is one env-var flip (`ACTIVE_ENGINE=temporal`). New code is small and isolated in
  `services/engine-adapter/internal/infrastructure/`; the engine-agnostic interpreter is reused
  unchanged, so the two engines cannot drift in semantics.
- **Negative / trade-off:** The eval engine is **evaluation-grade only** — no durability,
  persistence, retries, replay, or audit history. A restart loses runs. It must never be used as a
  production execution backend; this boundary is enforced by the default staying `temporal`, by
  adding no persistence config, and by mandatory docs/error copy pointing to the graduation path.
- **Neutral / follow-up required:** add an `eval` reference leg to the e2e-smoke matrix (no
  Temporal/Argo control plane), update the unsupported-engine error to list `eval`, and document
  the evaluation-vs-production boundary. No proto, YAML, or persistence-schema changes.

---

## Update — 2026-06-19 (Rejected)

This ADR's original alternatives analysis omitted a **stripped Temporal profile**: Temporal's
official `temporal server start-dev` runs an embedded in-memory database (no external Postgres),
and retries are already a config knob (`ZYNAX_ENGINE_MAX_ACTIVITY_ATTEMPTS=1`,
`services/engine-adapter/cmd/engine-adapter/main.go:80`). The engine-adapter is a Temporal *client*
that simply dials `ZYNAX_ENGINE_ADAPTER_TEMPORAL_HOST_PORT` (`main.go:68`), so pointing it at a
dev-server is a compose change, not code.

That option (tracked as **#1456**) achieves the Day-0 goal — no external database, evaluation-grade
fail-fast behavior — by **reusing the proven `TemporalEngine` with ~zero new code and nothing new to
maintain**, and with no eval-vs-production semantic-drift risk. It therefore dominates a third
in-process engine for the Docker/compose path.

**Decision:** Rejected. The in-process `EvalEngine` is **deferred** — revisit only if a truly
zero-Docker, `zynax`-binary-only run (no container at all) becomes a requirement. Issues #1359 and
its stories #1449–#1452 are closed as superseded by #1456.
