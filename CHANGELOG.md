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

## [0.8.0] — M9 (Hard Removals + Conformance) — 2026-08-20

### Added
- **Dead-letter forwarding in [`libs/zynaxevents`](libs/zynaxevents)** — the conventions
  provisioned `DLQ_<src>` and stopped redelivery at `MaxDeliver=5`, but nothing ever moved an
  exhausted message into it (true of the retired facade too). The new opt-in
  `Client.StartDLQForwarder` consumes the JetStream max-deliveries advisory, fetches the
  exhausted message from its source stream and republishes it byte-for-byte on the reserved
  exact `zynax.dlq.<prefix>.dead` subject with `Zynax-Dlq-Source-*` forensic headers. It never
  deletes the source message, and a deterministic `DLQ_<stream>:<sequence>` message id makes a
  replayed advisory, a retried publish or a second forwarder collapse onto one DLQ message.
  Opt-in by design: `Subscribe` does not start one — see
  [docs/patterns/direct-jetstream-events.md](docs/patterns/direct-jetstream-events.md). (#1653)

### Removed
- **BREAKING — `EventBusService` gRPC contract (`protos/zynax/v1/event_bus.proto`) and its
  generated Go/Python stubs are deleted.** Deprecated in v0.7.0 (`option deprecated = true`,
  ADR-046 Decision 6), removed on the M9 milestone boundary per the ADR-048 removal policy.
  **Replacement:** publish/subscribe over NATS JetStream through the shared
  [`libs/zynaxevents`](libs/zynaxevents) client; the contract of record is
  [`spec/asyncapi/zynax-events.yaml`](spec/asyncapi/zynax-events.yaml). No in-repo caller
  referenced the stubs — the facade Deployment and source tree were retired first
  (#1700, #1701). The `buf breaking` gate carries a documented exception scoped to this one
  file in `protos/buf.yaml`. (#1702)

### Supply chain
- **SDK provenance:** PyPI Trusted Publisher (OIDC) configuration for `zynax-sdk` is recorded in
  [M7-planning §14 — PyPI Trusted Publisher History](docs/milestones/M7-planning.md#14--pypi-trusted-publisher-history).
  The v0.6.0 release notes (first SDK publish) must link this section. (#1214)

### Security
- **Go toolchain moved to 1.26.6 across every build path**, clearing five Go standard-library
  advisories that were failing the scheduled Weekly Audit's `security rescan`:
  `GO-2026-6091` (html/template), `GO-2026-6090` (crypto/tls), `GO-2026-6089` (net/http),
  `GO-2026-6088` (encoding/xml) and `GO-2026-5972` (encoding/asn1). `images/images.yaml` moves
  the `golang-alpine` pin to `1.26.6-alpine` at its multi-arch OCI **index** digest, which
  re-stamps the eight banner-marked consumers; the version tags are hand-edited alongside
  because `images sync` rewrites only the digest inside a banner and `images check` only
  substring-tests it — neither reads `tag:`, so a digest-only bump would leave every Dockerfile
  claiming 1.26.5 while building 1.26.6. `govulncheck` now reports no vulnerabilities for all
  six services and all five Go adapters. (#1781)
- **Release binaries pinned to the same toolchain** via a `toolchain go1.26.6` directive in
  `go.work` and all 19 `go.mod` files. The release workflows resolve their Go version from
  `go-version-file`, which reads the `go` directive — still `1.26.4` — so the shipped `zynax`
  and `zynax-ci` binaries were being built on the vulnerable toolchain even after #1781.
  A `toolchain` directive rather than a `go` bump, because `go` is the floor imposed on
  *consumers* and `protos/generated/go` is published as a downloadable stub archive. (#1782)

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
