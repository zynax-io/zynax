# ADR Index — Architecture Decision Records

All decisions that shape Zynax are recorded here. Before proposing a design change,
search this index for the ADR that already governs the area. One-way doors always
get ADRs; reversible implementation choices do not.

To add a new ADR, copy [TEMPLATE.md](TEMPLATE.md), assign the next number, and open
a PR against `docs/adr/`.

---

| ADR | Title | Status | Date | Governs |
|-----|-------|--------|------|---------|
| [ADR-001](ADR-001-grpc-inter-service-protocol.md) | gRPC as the inter-service protocol | Accepted | 2026-04-01 | `protos/`, all service-to-service calls |
| [ADR-002](ADR-002-python-312.md) | Python 3.12 as the agent runtime | Accepted | 2026-04-01 | `agents/`, Python toolchain |
| [ADR-003](ADR-003-uv-package-manager.md) | uv as the Python package manager | Accepted | 2026-04-01 | `agents/`, `pyproject.toml`, lock files |
| [ADR-004](ADR-004-bdd-testing.md) | BDD as the primary testing methodology | Superseded | 2026-04-01 | Superseded by ADR-016 |
| [ADR-005](ADR-005-apache-license.md) | Apache 2.0 license | Accepted | 2026-04-01 | `LICENSE`, SPDX headers on all files |
| [ADR-006](ADR-006-monorepo.md) | Monorepo structure | Accepted | 2026-04-01 | Repository layout, `go.work`, `buf.work.yaml` |
| [ADR-007](ADR-007-pydantic-settings.md) | Pydantic Settings for agent configuration | Accepted | 2026-04-01 | `agents/` config management |
| [ADR-008](ADR-008-no-shared-databases.md) | No shared databases across services | Accepted | 2026-04-01 | All services — cross-service data access |
| [ADR-009](ADR-009-language-strategy.md) | Language strategy: Go for services, Python for agents | Accepted | 2026-04-01 | `services/` (Go only), `agents/` (Python only) |
| [ADR-010](ADR-010-pluggable-agent-runtime.md) | Pluggable agent runtime | Accepted | 2026-04-01 | `agents/`, `AgentService` gRPC contract |
| [ADR-011](ADR-011-declarative-yaml-control-plane.md) | Declarative YAML control plane | Accepted | 2026-04-01 | `spec/`, Layer 1 → Layer 2 boundary |
| [ADR-012](ADR-012-workflow-ir.md) | Workflow IR as the engine-agnostic intermediate representation | Accepted | 2026-04-01 | `WorkflowIR` proto, `services/workflow-compiler/` |
| [ADR-013](ADR-013-adapter-first-no-sdk.md) | Adapter-first — no mandatory SDK | Accepted | 2026-04-01 | `agents/sdk/` (optional), `AgentService` gRPC |
| [ADR-014](ADR-014-event-driven-state-machine.md) | Event-driven state machine workflow model | Accepted | 2026-04-01 | `spec/workflows/`, `WorkflowIR` state model |
| [ADR-015](ADR-015-pluggable-workflow-engines.md) | Pluggable workflow engines | Accepted | 2026-04-01 | `services/engine-adapter/`, M3+ |
| [ADR-016](ADR-016-layered-testing-strategy.md) | Layered testing strategy | Accepted | 2026-04-21 | Test placement, BDD scope, coverage gates |
| [ADR-017](ADR-017-contract-test-isolation.md) | Contract test isolation — GOWORK=off | Accepted | 2026-04-21 | `protos/tests/`, `services/*/` — all `go test` invocations |
| [ADR-018](ADR-018-ai-kb-authorization-model.md) | AI knowledge base authorization model | Accepted | 2026-04-24 | `CLAUDE.md`, `AGENTS.md`, `.ai/`, `.claude/` — KB paths |

---

## Status definitions

| Status | Meaning |
|--------|---------|
| **Proposed** | Open for discussion — not yet binding |
| **Accepted** | Binding — all new code must comply |
| **Deprecated** | No longer recommended — existing code may still follow it |
| **Superseded** | Replaced by a newer ADR (noted in the entry) |
