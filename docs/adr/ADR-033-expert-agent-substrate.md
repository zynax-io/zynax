# ADR-033: Expert-agent substrate (runtime AgentDef + authoring experts)

**Status:** Proposed  **Date:** 2026-06-15
**Related:** ADR-010 (pluggable agent runtime), ADR-013 (adapter-first), ADR-019 (SPDD)

---

## Context

The M7 brief asks for a system of "expert agents" (Go, Kubernetes, Security, Docs, …). Zynax already
has **two** distinct expert-like surfaces: runtime agents (`kind: AgentDef`, registered in
agent-registry and dispatched as capabilities) and **Claude Code delivery experts**
(`automation/workflows/experts/*.yaml` + `.claude` skills) used in the SPDD authoring loop. Picking only
one substrate would either prevent experts from running inside workflows (authoring-only) or lose the
delivery-loop experts that already drive autonomous milestone work. Letting them evolve independently
would cause silent drift.

## Decision

1. Experts exist on **both substrates**, with a documented mapping:
   - **Runtime experts** (`kind: AgentDef`) — registered, dispatched, and OTEL-traced; run inside workflows.
   - **Authoring experts** (`automation/workflows/experts/*.yaml` + `.claude` skills) — drive the SPDD
     delivery/authoring loop.
2. A **mapping table** is the single source of truth linking each authoring expert to its runtime
   counterpart (or marking it "authoring-only"). A CI check requires every authoring expert to declare a
   `runtime_mapping:`.
3. `agents/examples/` holds canonical SDK-based reference agents (incl. a runtime expert) — created in M7.
4. The full 14-expert library is **deferred to M-dx**; M7 establishes the substrate + the mapping + a few references.

## Rationale

| Option | Assessment |
|--------|------------|
| Both substrates + mapping (chosen) | ✅ Experts run in workflows AND drive delivery; drift controlled by a CI-checked mapping |
| Runtime AgentDefs only | ✗ Rejected — loses the existing authoring-loop experts |
| Claude Code experts only | ✗ Rejected — experts couldn't run inside Zynax workflows |

## Mapping table (single source of truth)

Each authoring expert (`.claude/commands/experts/*.md`, surfaced as an
`automation/workflows/experts/*.yaml` entry) declares a `runtime_mapping:` to its runtime
counterpart (`kind: AgentDef` in agent-registry), or the literal `authoring-only`. The table below is
the canonical seed; the machine-readable source remains the `runtime_mapping:` field on each authoring
expert (the CI drift guard reconciles the two — see below).

| Authoring expert (`.claude/commands/experts/`) | Runtime counterpart (`kind: AgentDef`) | Capability | Status |
|-----------------------------------------------|----------------------------------------|------------|--------|
| `go-services`   | `go-review-expert`   | `code.review.go`        | runtime (M7, `agents/examples/go-review-expert`) |
| `bdd-contract`  | `authoring-only`     | —                       | authoring-only (deferred to M-dx) |
| `ci-release`    | `authoring-only`     | —                       | authoring-only (deferred to M-dx) |
| `git-ops`       | `authoring-only`     | —                       | authoring-only (deferred to M-dx) |
| `infra-helm`    | `authoring-only`     | —                       | authoring-only (deferred to M-dx) |
| `post-merge`    | `authoring-only`     | —                       | authoring-only (deferred to M-dx) |
| `python-adapters` | `authoring-only`   | —                       | authoring-only (deferred to M-dx) |
| `spdd-canvas`   | `authoring-only`     | —                       | authoring-only (deferred to M-dx) |

Only `go-services → go-review-expert` is dual-substrate in M7 (the reference runtime expert); the
remaining authoring experts are explicitly `authoring-only` until the full expert library lands in M-dx.
Marking a row `authoring-only` is a deliberate, reviewable declaration — not an omission.

## Drift guard

Authoring and runtime experts must never diverge silently. The guard has three parts:

1. **Declared mapping is mandatory.** Every authoring expert MUST carry a `runtime_mapping:` field
   whose value is either a runtime AgentDef name or the literal `authoring-only`. A missing or empty
   field is a hard CI failure.
2. **Runtime reference must resolve.** When `runtime_mapping:` names an AgentDef, that AgentDef MUST
   exist as a registerable expert under `agents/examples/` (and, once running, in agent-registry). A
   dangling reference fails CI.
3. **Table reconciliation.** A CI check (added in EPIC X step X.5) compares the `runtime_mapping:`
   fields against this ADR's mapping table; any disagreement — an unlisted expert, a stale row, or a
   changed counterpart — fails the build. The `runtime_mapping:` fields are the machine source of truth;
   this table is the human-readable mirror, and CI keeps them identical.

This keeps the two substrates coherent: a new authoring expert cannot merge without declaring (and a
reviewer cannot miss) whether it has a runtime counterpart.

## Consequences

- **Positive:** a unified, coherent expert system; reference agents unblock `agents/examples/`; experts
  become observable when run as runtime AgentDefs.
- **Negative / trade-off:** two surfaces to maintain — mitigated by the mandatory mapping table + CI check.
- **Neutral / follow-up:** the full expert library, RAG/memory strategies, and escalation rules are M-dx.
