# ADR-016: Layered Testing Strategy

**Status:** Accepted  **Date:** 2026-04-21
**Supersedes:** ADR-004 (BDD as Primary Testing Methodology)

---

## Context

Zynax is a distributed, event-driven control plane with async state machines,
eventual consistency, pluggable engine adapters, and dynamic agent topology.
ADR-004 established BDD as the *primary* testing methodology. In practice, this
creates two problems:

1. **Noise at the wrong layer.** Gherkin is a poor fit for internal domain logic
   (routing algorithms, state transitions, message handlers). Forcing `.feature`
   files there produces verbose specs that test implementation details, slow down
   iteration, and add no documentation value.

2. **A gap at the distributed layer.** Classic unit and BDD tests cannot express
   partial failures, retry storms, message drops, or dynamic topology changes.
   The most important correctness properties of a mesh system are untestable with
   the current approach.

---

## Decision

Adopt a **four-tier testing pyramid**. BDD remains mandatory at system
boundaries; the other tiers cover what BDD cannot.

| Tier | Volume | Scope |
|------|--------|-------|
| BDD | 10–15% | System boundaries: agent contracts, inter-service gRPC, E2E flows |
| Unit / property-based | ≥ 40% | Domain layer: routing, state transitions, message handling |
| Contract | CI gate | Proto breaking-change detection, YAML manifest schema |
| Simulation / fault injection | As needed | Failure modes, retries, topology changes |

### Where BDD applies

- **Agent capability contracts** — task delegation, retry logic, failure handling
- **Inter-service gRPC contracts** — expected responses, error codes, edge cases
- **End-to-end workflows** — YAML submitted → distributed → result aggregated

BDD scenarios at these boundaries double as human-readable contract documentation
and as a prompting tool when generating code with AI assistants (write the scenario
first, then ask for code that satisfies it).

### Where BDD does NOT apply

- Internal domain logic (routing algorithms, scheduler, state machine internals)
- Networking and transport layer
- Performance and throughput
- Low-level concurrency primitives

Use unit tests and property-based tests for all of the above.

### Property-based testing

State machines and message handlers have invariants that hold across the entire
input space, not just for the handful of examples a human writes. Property tests
express these invariants and let the framework find counterexamples.

Tools: `hypothesis` (Python), `rapid` or `go-fuzz` (Go).

### Simulation / fault-injection testing

Distributed robustness requires injecting failures that cannot be reproduced in
unit tests: agent timeouts, dropped NATS messages, retry storms, nodes
joining/leaving mid-workflow. A simulation harness using `testcontainers` with
controlled fault injection is introduced as a follow-up to this ADR.

---

## Rationale

| Concern | BDD-only | Layered |
|---------|----------|---------|
| Internal domain correctness | Verbose, fragile | Unit + property: fast, precise |
| Distributed fault tolerance | Cannot express | Simulation harness |
| Contract safety | Scenarios only | `buf breaking` + schema CI gate |
| AI-assisted development | Feature file as prompt | Feature file as boundary prompt |
| Documentation value | High at boundaries | High where BDD is used, zero noise elsewhere |

---

## Consequences

- **DoD updated (AGENTS.md §11):** `.feature` file required only for
  system-boundary features. Domain logic requires unit or property tests.
- **Hard constraint updated (AGENTS.md §12):** BDD-first applies at boundaries;
  domain code is TDD.
- **BDD-first CI gate:** The existing `bdd-first` gate checks all `.go`/`.py`
  files. A follow-up PR will scope it to boundary code only
  (paths: `api/`, `tests/features/`, adapter entry points).
- **Simulation harness:** Introduced in a follow-up PR under `tests/simulation/`.
- **ADR-004** status updated to Superseded. Its BDD-first workflow steps remain
  correct and apply at system boundaries as described here.
