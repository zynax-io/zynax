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
| [ADR-019](ADR-019-spdd-prompt-governance.md) | Structured Prompt-Driven Development (SPDD) | Accepted | 2026-04-30 | `feat:` PRs — Canvas before code, `docs/spdd/`, slash commands |
| [ADR-020](ADR-020-zero-trust-auth.md) | mTLS with cert-manager for inter-service gRPC auth | Accepted | 2026-05-21 | All inter-service gRPC in K8s — #240 #488 |
| [ADR-021](ADR-021-horizontal-scale.md) | Postgres-backed repositories for horizontal scaling | Accepted | 2026-05-21 | task-broker + agent-registry repos — #578 #626 |
| [ADR-022](ADR-022-event-bus-architecture.md) | EventBusService gRPC wrapper over NATS JetStream | Accepted | 2026-06-02 | `services/event-bus/`, `protos/zynax/v1/event_bus.proto`, EPIC I (#772) |
| [ADR-023](ADR-023-restrict-direct-pushes-to-main.md) | Restrict direct pushes to `main`; rebase-merge only | Accepted | 2026-06-03 | All branches, PRs, and merge operations |
| [ADR-024](ADR-024-image-reference-management.md) | Container image reference management — `images.yaml` source of truth | Accepted | 2026-06-08 | `images/images.yaml`, all CI workflow files, Dockerfiles, `config/ci-runner-digest.txt` |
| [ADR-025](ADR-025-slsa-provenance-attestation.md) | SLSA provenance attestation: keep vs disable | Accepted | 2026-06-08 | `tools-image.yml`, `release.yml` — all `docker/build-push-action` steps |
| [ADR-026](ADR-026-postgres-distribution.md) | Postgres distribution + target major version (official `postgres:17` + thin chart) | Accepted | 2026-06-10 | `helm/charts/postgres/`, `images/images.yaml`, all production Postgres — EPIC #1073 |
| [ADR-027](ADR-027-shift-left-pipeline.md) | Shift-left pipeline model — build once pre-merge, promote by retag | Accepted | 2026-06-11 | `ci.yml` build-images, `release.yml` retag jobs, GHCR staging lane — EPIC #1109 |
| [ADR-028](ADR-028-agentdef-vs-workflow-self-hosted-automation.md) | AgentDef-vs-Workflow split for self-hosted automation + context-slice injection contract | Accepted | 2026-06-11 | `automation/workflows/**`, `spec/schemas/agent-def.schema.json`, `spec/schemas/workflow.schema.json` — EPIC #881 |
| [ADR-029](ADR-029-workflow-data-flow.md) | Workflow data-flow semantics (output/input bindings) | Accepted | 2026-06-16 | `WorkflowIR` proto, workflow-compiler, engine-adapter — M7 EPIC W |
| [ADR-030](ADR-030-observability-uptrace.md) | Observability — OpenTelemetry + Uptrace backend | Accepted | 2026-06-16 | all services + adapters, `libs/zynaxobs`, observability compose + Helm — M7 EPIC O |
| [ADR-031](ADR-031-context-propagation.md) | Context propagation model (trace · data · correlation) | Proposed | 2026-06-15 | all services, engine-adapter, agents/sdk — M7 EPIC C |
| [ADR-032](ADR-032-git-mcp-shim.md) | Git MCP as a thin shim over the git-adapter | Accepted | 2026-06-15 | `agents/adapters/git/`, cli — M7 EPIC G |
| [ADR-033](ADR-033-expert-agent-substrate.md) | Expert-agent substrate (runtime AgentDef + authoring experts) | Proposed | 2026-06-15 | `agents/examples/`, `automation/workflows/experts/`, agent-registry — M7 EPIC X |
| [ADR-034](ADR-034-manifest-workflow-id-collision-domain.md) | ManifestWorkflowID 64-bit collision domain + canonicalization stability | Proposed | 2026-06-15 | workflow-compiler — M7 EPIC Q (#583) |
| [ADR-035](ADR-035-adapter-language-boundary.md) | Adapter language boundary — Go for provider/proxy adapters, Python for AI-framework adapters | Accepted | 2026-06-16 | `agents/adapters/*` — refines ADR-009 — M7 EPIC P (#1276) |
| [ADR-036](ADR-036-ci-logic-as-go-cli.md) | CI logic belongs in a tested Go CLI (zynax-ci), not inline workflow bash | Accepted | 2026-06-16 | `cmd/zynax-ci/`, `.github/workflows/*`, `scripts/`, `Makefile` — M7 EPIC S (#1285) |
| [ADR-037](ADR-037-zero-temporal-evaluation-engine.md) | Zero-Temporal in-process evaluation engine (Day-0 onboarding) | Rejected | 2026-06-19 | `services/engine-adapter/` — third engine behind ADR-015 — M7 (#1359) (superseded by #1456) |
| [ADR-038](ADR-038-adk-go-adapter-framework.md) | Google ADK Go as a Go-native AI-framework adapter | Accepted | 2026-06-21 | `agents/adapters/adk/` — refines ADR-035 — M7 |
| [ADR-039](ADR-039-crd-native-scheduler.md) | CRD-native Scheduler — `Agent` CRD as the single source of truth; registry → stateless scheduler | Accepted | 2026-06-22 | `services/agent-registry/` (→ scheduler), `protos/zynax/v1/scheduler.proto`, task-broker — amends ADR-021/028 — spike M7, build M8 |
| [ADR-040](ADR-040-kubernetes-native-delegation-boundary.md) | Kubernetes-native delegation boundary (thin-Zynax) — build only the AI-scheduling core, delegate generic primitives | Proposed | 2026-06-22 | Repo-wide design principle — delegation vs custom core; Workflow=thin CRD front-end; Loki out of scope — relates ADR-039/020/030/012/015 |

---

## Status definitions

| Status | Meaning |
|--------|---------|
| **Proposed** | Open for discussion — not yet binding |
| **Accepted** | Binding — all new code must comply |
| **Deprecated** | No longer recommended — existing code may still follow it |
| **Superseded** | Replaced by a newer ADR (noted in the entry) |
