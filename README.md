<!-- SPDX-License-Identifier: Apache-2.0 -->

<div align="center">

# Zynax

**Write your agent workflow once — run it on Temporal or Argo without a rewrite.**

_The engine-portability layer for agentic automation._

[![CI](https://github.com/zynax-io/zynax/actions/workflows/ci.yml/badge.svg)](https://github.com/zynax-io/zynax/actions/workflows/ci.yml)
[![AI Context Budget](https://github.com/zynax-io/zynax/actions/workflows/ai-context-budget.yml/badge.svg)](https://github.com/zynax-io/zynax/actions/workflows/ai-context-budget.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Built with CNCF-graduated technologies](https://img.shields.io/badge/built_with-CNCF_graduated_tech-026be0.svg)](https://www.cncf.io/projects/)
[![Go version](https://img.shields.io/github/go-mod/go-version/zynax-io/zynax?filename=go.work&label=go&color=00add8)](go.work)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/zynax-io/zynax/badge)](https://securityscorecards.dev)

[Quickstart](#-quickstart) · [Architecture](ARCHITECTURE.md) · [Documentation](docs/) · [Contributing](CONTRIBUTING.md) · [Roadmap](ROADMAP.md)

</div>

---

## See it run — one command

```bash
make demo
```

`make demo` boots the minimal stack with a zero-secret local-LLM ([Ollama](https://ollama.com)) overlay
and runs the hero code-review workflow end-to-end — a real model reviews a git diff and prints its
verdict, straight from the CLI. The only prerequisites are Docker and pulling the demo model once
(`ollama pull qwen2.5-coder:3b`). Tear it down with `make demo-clean`.

> **Zero-dependency first win** — want a success before any model or secret? Run the
> [`hello-world`](spec/workflows/examples/hello-world.yaml) workflow on the same kind cluster:
> `zynax --api-url http://localhost:8080 apply spec/workflows/examples/hello-world.yaml` (the built-in
> `echo` capability, no Ollama, no API key).

> **Switch engines, same workflow** — the demo runs on Temporal by default; run the *same*
> manifest on Argo with one flag: `ENGINE=argo make demo` (or `E2E_ENGINE=argo make demo`). That
> portability — write once, run on Temporal **or** Argo, no rewrite — is the wedge ([ADR-041](docs/adr/), [#1370](https://github.com/zynax-io/zynax/issues/1370)).

<!-- asciinema cast — record locally with `make demo` and `asciinema rec docs/casts/make-demo.cast`.
     See docs/casts/README.md. Once recorded, replace the placeholder below with the upload embed. -->
[![asciicast — make demo](https://asciinema.org/a/PLACEHOLDER.svg)](https://asciinema.org/a/PLACEHOLDER)

> The cast above is a placeholder until a maintainer records `docs/casts/make-demo.cast` from a live
> `make demo` run — see [docs/casts/README.md](docs/casts/README.md). New here? Jump to the
> [Quickstart](#-quickstart).

---

## What is Zynax?

> **Define an AI agent workflow once, as a YAML manifest, and run it on whichever engine your org
> already operates — Temporal or Argo — without changing the workflow.** That portability is the
> wedge: a Kubernetes-locked agent platform cannot move one workflow across engines.

Like Kubernetes for containers, Zynax is a control plane that abstracts the execution layer behind
a declarative, versionable API — but the point you reach for it is **engine portability** and
**adapter-first capabilities** (any gRPC service is a capability; no SDK required). It runs on
Docker Compose **and** Kubernetes/Helm.

**Already standardized on a K8s-native agent tool?** Keep it — those agents register as Zynax
capabilities, so Zynax orchestrates across your engines rather than replacing them.

> Positioning & messaging principle (lead with the wedge, everywhere): [docs/product/positioning.md](docs/product/positioning.md).

```yaml
# code-review.yaml
kind: Workflow
apiVersion: zynax.io/v1

metadata:
  name: code-review
  namespace: engineering

spec:
  initial_state: review

  states:
    review:
      actions:
        - capability: request_review
      on:
        - event: review.approved
          goto: merge
        - event: review.needswork
          goto: fix

    fix:
      on:
        - event: push
          goto: review

    merge:
      actions:
        - capability: merge_pr
      on:
        - event: merge.success
          goto: done

    done:
      type: terminal
```

```bash
zynax apply code-review.yaml
# run_id: wf-236c478f00eb68ce

zynax status workflow wf-236c478f00eb68ce
# status: Running  current_state: review

zynax logs wf-236c478f00eb68ce
# state.entered  review
# state.exited   review → merge
```

---

## Published Artifacts

Zynax publishes three kinds of artifacts on every release and on every merge to `main`.

### zynax CLI — user-facing binary

Download from the [latest GitHub Release](https://github.com/zynax-io/zynax/releases/latest):

| Platform | Command |
|----------|---------|
| macOS (Apple Silicon) | `curl -fsSL https://github.com/zynax-io/zynax/releases/latest/download/zynax_darwin_arm64.tar.gz \| tar xz && sudo mv zynax /usr/local/bin/` |
| macOS (Intel) | `curl -fsSL https://github.com/zynax-io/zynax/releases/latest/download/zynax_darwin_amd64.tar.gz \| tar xz && sudo mv zynax /usr/local/bin/` |
| Linux (amd64) | `curl -fsSL https://github.com/zynax-io/zynax/releases/latest/download/zynax_linux_amd64.tar.gz \| tar xz && sudo mv zynax /usr/local/bin/` |
| Linux (arm64) | `curl -fsSL https://github.com/zynax-io/zynax/releases/latest/download/zynax_linux_arm64.tar.gz \| tar xz && sudo mv zynax /usr/local/bin/` |

**From source** (requires the Go toolchain pinned in [`go.work`](go.work)): `cd cmd/zynax && GOWORK=off go build -o ~/bin/zynax .`
**Makefile shortcut:** `make install-cli`

Verify: `zynax --version`

---

### zynax-ci — CI and developer toolchain

`zynax-ci` is a standalone Go binary that replaces all Python validation scripts. It contains:
`validate canvas`, `validate schema`, `validate workflows`, `validate agent-defs`,
`validate capabilities`, `validate policies`, and `check ai-context`.

Download from the [latest GitHub Release](https://github.com/zynax-io/zynax/releases/latest):

```bash
# Linux (amd64)
curl -fsSL https://github.com/zynax-io/zynax/releases/latest/download/zynax-ci-linux-amd64 \
  -o ~/bin/zynax-ci && chmod +x ~/bin/zynax-ci

# macOS (Apple Silicon)
curl -fsSL https://github.com/zynax-io/zynax/releases/latest/download/zynax-ci-darwin-arm64 \
  -o ~/bin/zynax-ci && chmod +x ~/bin/zynax-ci

# macOS (Intel)
curl -fsSL https://github.com/zynax-io/zynax/releases/latest/download/zynax-ci-darwin-amd64 \
  -o ~/bin/zynax-ci && chmod +x ~/bin/zynax-ci
```

**From source** (requires the Go toolchain pinned in [`go.work`](go.work)): `cd cmd/zynax-ci && GOWORK=off go build -o ~/bin/zynax-ci .`
**Makefile shortcut:** `make install-ci-tools`

---

### Image versions

All container image references in workflow files and Dockerfiles are **generated** from
`images/images.yaml` — the single source of truth for every digest and tag used across the
repository.

```bash
make sync-images    # update all banner-marked regions from images/images.yaml
make check-images   # verify all banner-marked regions match images/images.yaml (CI gate)
```

**Do not hand-edit** any region delimited by `# BEGIN generated by zynax-ci images sync` /
`# END generated by zynax-ci images sync` banners — your changes will be overwritten on the
next `make sync-images` run.

See [`cmd/zynax-ci/AGENTS.md`](cmd/zynax-ci/AGENTS.md) for the `images sync` and
`images check` subcommand reference, and [`images/images.yaml`](images/images.yaml) for the
canonical digest list.

---

### Developer tools Docker image

`ghcr.io/zynax-io/zynax/tools:latest` is the canonical developer tools image. It ships:
`golangci-lint`, `buf`, `govulncheck`, `godog`, `mockery`, `migrate`,
`grpc_health_probe`, `protoc-gen-go`, `protoc-gen-go-grpc`, `gitleaks`, `uv`,
`ruff`, `mypy`, `bandit`, `pip-audit`, `pytest`, and **`zynax-ci`**.

The image is rebuilt and pushed to GHCR automatically on every merge to `main` that
changes `infra/docker/Dockerfile.tools` (via
[`.github/workflows/tools-image.yml`](.github/workflows/tools-image.yml)).

**Default behaviour — pull from GHCR (recommended):**

`make bootstrap` pulls the published image automatically on first use. No manual step needed:
```bash
make bootstrap   # pulls ghcr.io/zynax-io/zynax/tools:latest, installs pre-commit hooks
make lint        # uses cached image on all subsequent runs
```

**Build locally** (only needed when editing `Dockerfile.tools` itself):
```bash
make TOOLS_IMAGE=zynax/tools:local build-tools   # build once
make TOOLS_IMAGE=zynax/tools:local lint          # use local image for this run
# or export for the whole shell session:
export TOOLS_IMAGE=zynax/tools:local
make lint test validate-spec
```

**Pin to a specific SHA** (for reproducible CI references):
```bash
make TOOLS_IMAGE=ghcr.io/zynax-io/zynax/tools:main-<sha> lint
```

---

## Docker Images

All service and adapter images are published to GitHub Container Registry (GHCR) and are publicly readable. No authentication required to pull.

| Image | Description |
|-------|-------------|
| `ghcr.io/zynax-io/zynax/api-gateway` | REST gateway — receives `zynax apply` requests |
| `ghcr.io/zynax-io/zynax/workflow-compiler` | YAML → WorkflowIR compiler |
| `ghcr.io/zynax-io/zynax/engine-adapter` | Temporal execution adapter |
| `ghcr.io/zynax-io/zynax/task-broker` | In-memory capability dispatcher |
| `ghcr.io/zynax-io/zynax/http-adapter` | HTTP REST capability adapter |
| `ghcr.io/zynax-io/zynax/tools` | Dev toolchain image (Go + Python + proto tools) |

Images are rebuilt on every merge to `main` (`:main` tag) and on every versioned release (`:v<version>` and `:latest` tags).

**Pull the latest main build:**
```bash
docker pull ghcr.io/zynax-io/zynax/api-gateway:main
```

**Pull a specific version:**
```bash
docker pull ghcr.io/zynax-io/zynax/api-gateway:v0.4.0
```

Replace `api-gateway` with any image name from the table above.

---

## Key Principles

**Declarative-first** — workflows are YAML manifests, not code. Versionable, diffable, GitOps-ready.

**Engine-agnostic** — Temporal, LangGraph, or Argo are plugins. Swap the engine without changing the workflow.

**Capability routing** — workflows route to capabilities (`summarize`, `run_tests`, `open_mr`), not to named agents. Swap the executor without changing the workflow.

**No SDK required** — any system becomes a capability by implementing the `AgentService` gRPC contract. Wrap HTTP APIs, LLMs, Git providers, CI systems — all as capabilities.

**Event-driven state machines** — not DAGs. Supports loops, human-in-the-loop, long-running workflows, and async events natively.

> **How does Zynax compare to Kagent, Dapr, Temporal, or LangGraph?**
> See [docs/architecture/2026-05-28-competitive-positioning.md](docs/architecture/2026-05-28-competitive-positioning.md).

---

## Architecture

```
     YAML Manifests (Intent)
              ↓
       API Gateway (Go)
              ↓
     Workflow Compiler (Go)    ← YAML → Canonical IR
              ↓
      Engine Adapter (Go)      ← IR → Temporal / LangGraph / Argo
              ↓
        Task Broker (Go)       ← Capability routing
              ↓
    Execution Adapters Layer   ← LLM / HTTP / Git / CI / LangGraph
              ↓
     Event Bus — NATS (Go)     ← All events
              ↓
    Memory Service (Go)        ← KV + Vector context
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for the full design.

---

## Dependency Map by Layer

```
Layer 1 — Spec / YAML (intent)
    spec/workflows/      workflow.yaml, agent-def.yaml, policy.yaml
    spec/schemas/        JSON Schema validators
         │
         │  validated manifests (files only — never imported by services)
         ▼
Layer 2 — WorkflowIR / Compiler (canonical representation)
    services/workflow-compiler/   YAML → WorkflowIR (protobuf struct)
         │
         │  gRPC: CompileWorkflow / ValidateManifest / GetCompiledWorkflow
         ▼
Layer 3 — Platform Services (Go, gRPC-only cross-service)
    services/agent-registry/   capability catalogue
    services/task-broker/      capability routing + dispatch
    services/engine-adapter/   IR → Temporal / LangGraph / Argo
    services/memory-service/   KV + vector context store
    services/event-bus/        NATS JetStream async pub/sub
         │
         │  gRPC: AgentService contract (protos/zynax/v1/agent.proto)
         ▼
Layer 4 — Agents / SDK (Python execution adapters)
    agents/sdk/          zynax-sdk — Agent base class + @capability routing
    agents/examples/     reference agent implementations
```

| Layer | Package | Owns | Communicates via |
|-------|---------|------|-----------------|
| 1 — Spec | `spec/` | YAML manifests, JSON schemas | Filesystem — read at compile time |
| 2 — Compiler | `services/workflow-compiler/` | WorkflowIR (protobuf) | gRPC API to callers |
| 3 — Services | `services/*/` | Domain logic, state | gRPC between services; NATS events |
| 4 — Agents | `agents/` | Python adapters | gRPC stubs from `protos/generated/` |

Layer 1 YAML is never imported by Go services. Cross-service reads always go through gRPC — no shared packages, no shared databases.

---

## Quickstart

> **Prefer raw `docker compose` (no `make`)?** See
> [docs/running-with-docker-compose.md](docs/running-with-docker-compose.md) — three commands to a
> result, then optional depth (full platform, observability).

### Try it with Docker

**Prerequisites:** Docker Desktop + the `zynax` CLI (see [Install](#install-the-zynax-cli) above).

> **M5 status note:** `make run-local` starts all five platform services: api-gateway,
> workflow-compiler, engine-adapter, task-broker, and agent-registry, plus Temporal and NATS.
> Workflows submit, dispatch capabilities to registered agents, and produce state-transition logs.
> To exercise end-to-end capability dispatch, register an adapter (e.g. http-adapter) with the
> agent-registry before running `zynax apply`.

```bash
git clone https://github.com/zynax-io/zynax.git
cd zynax

# Start the full local stack (all 5 platform services + Temporal + NATS)
make run-local

# Apply an example workflow manifest
export ZYNAX_API_URL=http://localhost:7080
zynax apply spec/workflows/examples/code-review.yaml
# run_id: wf-<hex>

zynax status workflow wf-<hex>
# status: Running   current_state: review

zynax logs wf-<hex>
# streams state-transition events (capability dispatch pending M5.C)

# Stop the stack when done
make stop-local
```

Port map while the stack is running:

| Endpoint | URL |
|----------|-----|
| api-gateway HTTP | http://localhost:7080 |
| Temporal Web UI | http://localhost:7088 |
| Temporal gRPC | localhost:7233 |
| NATS | localhost:7422 |

### Develop locally

**Prerequisites:** Docker Desktop only (Go, Python, buf are not needed locally).

```bash
make bootstrap   # one-time per clone: pulls ghcr.io/zynax-io/zynax/tools:latest + installs pre-commit hooks
make lint        # proto + Go + Python lint
make test        # full suite (unit + BDD + coverage gate)
```

> **Pre-commit hooks** — `make bootstrap` wires `gofmt`, `golangci-lint`, `ruff`,
> `mypy`, and `gitleaks` to run automatically on every `git commit`. You must run
> `make bootstrap` once per clone to activate them.
> See [CONTRIBUTING.md §Pre-commit hooks](CONTRIBUTING.md#pre-commit-hooks) for details.

### Key make commands

| Command | What it does |
|---------|-------------|
| `make test` | Full test suite — spec validation + Go unit tests + BDD contracts + Python tests |
| `make test-unit-go` | Go unit tests with coverage report for all services |
| `make test-bdd` | Godog BDD contract tests for all `protos/tests/` packages |
| `make lint` | Proto + Go + Python lint |
| `make audit` | Dependency vulnerability audit — govulncheck (Go) + pip-audit (Python); exits 1 on any finding |
| `make security` | Full security scan — adds bandit SAST to the audit checks |
| `make validate-spec` | Validate all YAML manifests against JSON schemas (via `zynax-ci`) |
| `make validate-canvas` | Validate REASONS Canvas files under `docs/spdd/` (via `zynax-ci`) |
| `make generate-protos` | Regenerate Go + Python stubs from `.proto` files |
| `make build-tools` | Build `zynax/tools:local` from source — use when editing `Dockerfile.tools` |
| `make pull-tools` | Pull `ghcr.io/zynax-io/zynax/tools:latest` from GHCR explicitly (bootstrap does this automatically) |
| `make install-cli` | Build and install `zynax` CLI to `~/bin/zynax` |
| `make install-ci-tools` | Build and install `zynax-ci` toolchain to `~/bin/zynax-ci` |

---

## Milestone Status

Latest release: **v0.5.0 — K8s Production-Ready (M6 ✅)** · active milestone: **M7 — Usable Workflows + Observability** 🚧 (target v0.6.0) — live status in
[state/current-milestone.md](state/current-milestone.md) · milestone goals and sequence in
[ROADMAP.md](ROADMAP.md).

**M1** delivered the contracts-only foundation: 8 gRPC services defined as protobuf contracts,
AsyncAPI spec covering 11 event channels, generated Go + Python stubs, 140+ BDD contract
test scenarios across all services, and 5 CI gates (proto-breaking, stubs-freshness,
layer-boundaries, conventional-commit, PR-size).

**M2** delivered the Workflow IR compiler — the first running service in the platform:
YAML parser + WorkflowGraph builder, structural and semantic validators (no orphan states,
terminal state required, valid capability refs), WorkflowGraph → WorkflowIR serialisation
(protobuf), gRPC API layer (`CompileWorkflow` / `ValidateManifest` / `GetCompiledWorkflow`),
JSON Schemas for `Workflow`, `AgentDef`, and `Policy` manifest kinds, `make validate-spec`
target, and reference workflow YAML examples. Coverage gate ≥ 90% on all domain packages.

**M3** delivered the Temporal execution engine — the first live runtime: `WorkflowEngine`
Go interface decoupling the gRPC layer from any engine backend, `TemporalEngine` wrapping
the Temporal Go SDK, `IRInterpreterWorkflow` state machine interpreter driving
`DispatchCapabilityActivity` → `TaskBrokerService` gRPC, cel-go guard evaluation, and all 5
`EngineAdapterService` gRPC methods (`Submit`, `Signal`, `Cancel`, `GetWorkflowStatus`,
`WatchWorkflow`). **Partial:** CloudEvents lifecycle events are a log-stub (NATS not yet wired);
task-broker + agent-registry were delivered in M5.C (#479, #480).

**M4** delivered the YAML system and CLI: api-gateway HTTP REST layer
(`POST /api/v1/apply`, `GET /api/v1/workflows/{id}`, `DELETE /api/v1/workflows/{id}`),
the `zynax` CLI (`apply`, `get`,
`delete`, `status`, `logs`), local Docker Compose runner (`make run-local`),
pre-built CLI binaries published to GitHub Releases for five platforms, and GitOps integration.
Canvas: `docs/spdd/314-yaml-system-cli/canvas.md`.

---

## Service Status

Current implementation status per service and adapter (as of M7 active development; M6 shipped as v0.5.0):

### Platform Services

| Service | Status | Notes |
|---------|--------|-------|
| api-gateway | ✅ Implemented | REST → gRPC translation, bearer auth, context deadline propagation |
| workflow-compiler | ✅ Implemented | YAML → WorkflowIR; stateless — no in-memory IR store ([#466](https://github.com/zynax-io/zynax/issues/466)); multi-namespace support ([#767](https://github.com/zynax-io/zynax/issues/767)) |
| engine-adapter | ✅ Implemented | Temporal + Argo backends ([#766](https://github.com/zynax-io/zynax/issues/766)); cel-go guard evaluation; CloudEvents lifecycle events published via event-bus (#827) |
| task-broker | ✅ Implemented | Postgres-backed TaskRepository on pgx/v5 (EPIC [#626](https://github.com/zynax-io/zynax/issues/626)); gRPC wired in compose + Helm |
| agent-registry | ✅ Implemented | Postgres-backed AgentRepository on pgx/v5 (EPIC [#626](https://github.com/zynax-io/zynax/issues/626)); round-robin with heartbeat |
| event-bus | ✅ Implemented | NATS JetStream gRPC wrapper (EPIC #772: Publish/Subscribe/Unsubscribe + DLQ, ADR-022); engine-adapter lifecycle wiring merged (#827) |
| memory-service | ✅ Implemented | M6 — Redis KV + pgvector; all 10 RPCs, namespace TTL isolation, integration + BDD tests (EPIC #773) |

### Execution Adapters

| Adapter | Language | Status | Notes |
|---------|----------|--------|-------|
| http-adapter | Go | ✅ Complete | Generic HTTP capability bridge; SSRF-safe static routing |
| git-adapter | Go | ✅ Complete | `open_pr`, `request_review`, `get_diff` via GitHub REST API |
| ci-adapter | Go | ✅ Complete | `trigger_workflow`, `get_run_status` via GitHub Actions REST API |
| llm-adapter | Go | ✅ Complete | `chat_completion` — OpenAI, AWS Bedrock, Ollama (ported from Python, ADR-035) |
| langgraph-adapter | Python | ✅ Complete | LangGraph `StateGraph` as Zynax capabilities; wired in e2e-demo |

**Legend:** ✅ Complete · 🟡 MVP (in-memory, ephemeral) · 🔄 In Progress · 📋 Planned

---

## AI Context Architecture

AI assistants working in this repo load context in layers. Smaller budgets = higher signal density.

| File | Role | Limit |
|------|------|-------|
| `CLAUDE.md` | Session bootstrap — milestone status, dev workflow, anti-patterns | 200 lines |
| `AGENTS.md` (root) | Engineering constitution — immutable principles, hard constraints | 300 lines |
| `docs/ai-assistant-setup.md` | Onboarding guide for AI contributors | 150 lines |
| `services/*/AGENTS.md` | Per-service rules (layout, tests, service-specific mistakes) | 150 lines each |
| `agents/*/AGENTS.md` | Per-adapter rules (Python patterns, gRPC stub usage) | 150 lines each |

Total budget: **2000 lines** across all files. The [AI Context Budget](https://github.com/zynax-io/zynax/actions/workflows/ai-context-budget.yml) workflow reports current totals on every relevant PR (advisory, non-blocking). Counted by `zynax-ci check ai-context`.

---

## Repository Structure

```
spec/                Workflow YAML manifests and JSON schemas (Workflow, AgentDef, Policy)
services/            Go platform services
  workflow-compiler/ YAML → WorkflowIR compiler (M2 — complete)
  engine-adapter/    Temporal execution engine bridge (M3 — complete)
  api-gateway/       HTTP REST entry point: /api/v1/apply + /api/v1/workflows (M4 — complete)
  agent-registry/    Capability catalogue service (M4+)
  task-broker/       Capability routing service (M4+)
  memory-service/    KV + vector context store (M4+)
  event-bus/         NATS JetStream async pub/sub (M4+)
cmd/zynax/           zynax CLI — apply, get, delete, status (M4 — complete)
cmd/zynax-ci/        zynax-ci CI toolchain — validate canvas/schema/manifests, check ai-context
agents/              Python execution adapters + zynax-sdk
protos/              gRPC contracts (Go + Python stubs generated)
protos/tests/        BDD contract test suites (godog)
infra/               Docker-first dev environment · Helm charts (planned, #241)
docs/adr/            Architecture Decision Records (ADR-001 – ADR-019)
docs/milestones/     Per-milestone engineering reviews and release notes
docs/spdd/           REASONS Canvas artifacts — one canvas.md per feat: issue
```

---

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a PR.
Key rules: `.feature` file before any code, BDD-first, Docker-only dev.

---

## Security

Report vulnerabilities via [GitHub Security Advisories](https://github.com/zynax-io/zynax/security/advisories/new).
See [SECURITY.md](SECURITY.md).

---

## License

Apache License 2.0 — see [LICENSE](LICENSE).

SPDX-License-Identifier: Apache-2.0

---

<div align="center">
<sub>Zynax is built with CNCF-graduated technologies (Temporal, gRPC, OpenTelemetry). It is not an official CNCF project.</sub>
</div>
