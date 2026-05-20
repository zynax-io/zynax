<!-- SPDX-License-Identifier: Apache-2.0 -->
<!-- This file tracks notable changes per milestone. -->
<!-- Entries are hand-curated; run `make changelog` for a raw Conventional-Commit log. -->

# Changelog

All notable changes to Zynax are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com) +
[Conventional Commits](https://conventionalcommits.org).
Versioning follows [Semantic Versioning](https://semver.org).

---

## [Unreleased]

---

## [0.4.0] — M5 (Adapter Library) — 2026-05-20

### Added
- `task-broker` service: in-memory `TaskBrokerService` with 5 RPCs (`DispatchTask`, `AcknowledgeTask`, `GetTask`, `ListTasks`, `CancelTask`), hexagonal layout, async dispatch, round-robin agent selection, 92.7% domain coverage (#479 / PRs #520 #522 #523)
- Bearer-token auth middleware for api-gateway mutating endpoints (`ZYNAX_GW_API_KEY`) (#482)
- X-Request-ID HTTP middleware with gRPC metadata propagation across api-gateway, workflow-compiler, and engine-adapter (#484)
- Idempotent `zynax apply` — manifest hash derives stable workflow ID; resubmitting the same YAML returns the same `run_id` (#485)
- HTTP adapter (`agents/adapters/http/`) — REST capability proxy, config-only route mapping, registry client with backoff (#380)
- Unified release workflow (`release.yml`) — single tag-triggered job fans out to parallel CLI binary, zynax-ci binary, and service image builds, then creates one GitHub Release with all assets (#557)

### Fixed
- `resolveTemplate` map-iteration non-determinism in `IRInterpreterWorkflow` — sorted-key iteration ensures Temporal determinism (#475)
- `CompileWorkflow` now returns the full `CompilationError` list instead of only the first error (#477)
- SSE `WriteTimeout` extended in api-gateway — `zynax logs` no longer disconnects at 30 s (#478)
- Event publish errors in engine-adapter now log as `WARN` instead of being silently discarded (#483)
- Release race condition between three legacy tag-triggered workflows eliminated — replaced by unified coordinator (#557)

### Changed
- Docker Compose files consolidated to one canonical `infra/docker-compose/docker-compose.yml`; `ZYNAX_GW_REGISTRY_ADDR` corrected to use service name (#486)
- CNCF Sandbox Candidate badge removed; replaced with "Built with CNCF-graduated technologies" (#472)
- CHANGELOG phantom entries removed (Helm charts, argo engine, features that do not exist in git) (#473)
- CI concurrency fixed — stale runs per branch cancelled; `merge_group` trigger removed (#545 #589)
- Push-to-main forced-true override removed from change-detection job (#546)

---

## [0.3.0] — M4 Partial (YAML System + CLI)

### Shipped
- api-gateway REST layer: `POST /api/v1/apply`, `GET /api/v1/workflows/{id}`, `DELETE /api/v1/workflows/{id}`, `GET /api/v1/workflows/{id}/logs` (SSE streaming)
- `zynax` CLI: `apply`, `get`, `delete`, `status`, `logs` subcommands
- `kind: AgentDef` routing via api-gateway → agent-registry
- Docker Compose local runner (`infra/docker-compose/`)
- GitOps watch mode in `zynax apply --watch`

### Not shipped in M4 (blocked by M5.C #460)
- Agent registry service (planned: #480)
- Task broker service (planned: #479)

---

## [0.2.0] — M3 Partial (Temporal Execution)

### Shipped
- `WorkflowEngine` interface + `TemporalEngine` implementation
- `IRInterpreterWorkflow` state machine (CEL guards, CloudEvents lifecycle)
- `DispatchCapabilityActivity` Temporal activity bridge
- All five `EngineAdapterService` gRPC methods: `SubmitWorkflow`, `GetWorkflowStatus`, `CancelWorkflow`, `WatchWorkflow`, `SendSignal`
- `slog`-based structured logging; CEL equality guards (`==`, `!=`)

### Not shipped in M3 (blocked by M5.C #460)
- Task broker for capability dispatch (planned: #479)

---

## [0.1.0] — M1 + M2 (Contracts + Workflow IR)

### Added
- Repository bootstrap and toolchain (`make bootstrap`, `make lint`, `make test`)
- 8 gRPC contracts: agent-registry, task-broker, memory-service, event-bus, workflow-compiler, engine-adapter, capability-agent, stub-generation
- AsyncAPI 3.0 spec for event topology (`spec/asyncapi.yaml`)
- JSON Schema validation for capability manifests (`spec/schemas/`)
- Go + Python generated stubs committed to `protos/generated/`
- 140+ BDD scenarios across all services (`protos/tests/`)
- 5 CI gates: `lint-proto`, `test-unit`, `test-integration`, `security`, `dco`
- YAML manifest parser + `WorkflowGraph` builder (`services/workflow-compiler/`)
- Structural and semantic validators for workflow manifests
- `WorkflowGraph → WorkflowIR` serialization
- `WorkflowCompilerService` gRPC API: `CompileWorkflow`, `ValidateManifest`, `GetCompiledWorkflow`
- ADRs 001–019 documenting architectural decisions (`docs/adr/`)
- CONTRIBUTING.md, SECURITY.md, GOVERNANCE.md, ROADMAP.md
- AGENTS.md engineering contracts (root, per-service, per-layer)
- Helm chart template patterns documented in `docs/patterns/helm-charts.md` (planned: `infra/helm/` charts not yet generated — see #458)
