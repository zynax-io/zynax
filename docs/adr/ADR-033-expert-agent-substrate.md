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

## Consequences

- **Positive:** a unified, coherent expert system; reference agents unblock `agents/examples/`; experts
  become observable when run as runtime AgentDefs.
- **Negative / trade-off:** two surfaces to maintain — mitigated by the mandatory mapping table + CI check.
- **Neutral / follow-up:** the full expert library, RAG/memory strategies, and escalation rules are M-dx.
