<!-- SPDX-License-Identifier: Apache-2.0 -->

<div align="center">

# Zynax

**The declarative control plane for AI agent workflows**

[![CI](https://github.com/zynax-io/zynax/actions/workflows/ci.yml/badge.svg)](https://github.com/zynax-io/zynax/actions/workflows/ci.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![CNCF Sandbox Candidate](https://img.shields.io/badge/CNCF-Sandbox_Candidate-026be0.svg)](https://cncf.io)
[![Go 1.22+](https://img.shields.io/badge/go-1.22+-00add8.svg)](https://go.dev)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/zynax-io/zynax/badge)](https://securityscorecards.dev)

[Quickstart](#-quickstart) · [Architecture](ARCHITECTURE.md) · [Documentation](docs/) · [Contributing](CONTRIBUTING.md) · [Roadmap](ROADMAP.md)

</div>

---

## What is Zynax?

> Zynax is to AI workflows what Kubernetes is to containers — a control plane
> that abstracts the execution layer behind a declarative, versionable API.

Zynax lets you define AI agent workflows as YAML manifests and execute them
on any workflow engine (Temporal, LangGraph, Argo) without changing your
workflow definition.

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
        - event: review.changes_requested
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
```

---

## Key Principles

**Declarative-first** — workflows are YAML manifests, not code. Versionable, diffable, GitOps-ready.

**Engine-agnostic** — Temporal, LangGraph, or Argo are plugins. Swap the engine without changing the workflow.

**Capability routing** — workflows route to capabilities (`summarize`, `run_tests`, `open_mr`), not to named agents. Swap the executor without changing the workflow.

**No SDK required** — any system becomes a capability by implementing the `AgentService` gRPC contract. Wrap HTTP APIs, LLMs, Git providers, CI systems — all as capabilities.

**Event-driven state machines** — not DAGs. Supports loops, human-in-the-loop, long-running workflows, and async events natively.

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
    agents/sdk/          zynax-sdk — gRPC stub wrapper + base adapter
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

**Prerequisites:** Docker Desktop. Nothing else.

```bash
git clone https://github.com/zynax-io/zynax.git
cd zynax
make bootstrap   # one-time: build the zynax-tools Docker image
make dev-up      # start the full local stack
```

### Key make commands

| Command | What it does |
|---------|-------------|
| `make test` | Full test suite — spec validation + Go unit tests + BDD contracts + Python tests |
| `make test-unit-go` | Go unit tests with coverage report for all services |
| `make test-bdd` | Godog BDD contract tests for all `protos/tests/` packages |
| `make lint` | Proto + Go + Python lint |
| `make audit` | Dependency vulnerability audit — govulncheck (Go) + pip-audit (Python); exits 1 on any finding |
| `make security` | Full security scan — adds bandit SAST to the audit checks |
| `make validate-spec` | Validate all YAML manifests against JSON schemas |
| `make generate-protos` | Regenerate Go + Python stubs from `.proto` files |
| `make build-tools` | Build the `zynax-tools:local` Docker image (run once after clone) |

---

## Milestone Status

| Milestone | Status | Version | Docs |
|-----------|--------|---------|------|
| **M1 — Contracts Foundation** | **Complete** | v0.1.0 | [Engineering Review](docs/milestones/M1-engineering-review.md) · [Release Notes](docs/milestones/M1-release-notes.md) |
| **M2 — Workflow IR** | **Complete** | v0.1.0 | [Epic #101](https://github.com/zynax-io/zynax/issues/101) |
| M3 — Temporal Execution | Planned | v0.2.0 | — |
| M4 — YAML System + CLI | Planned | v0.3.0 | — |

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

---

## Repository Structure

```
spec/                Workflow YAML manifests and JSON schemas (Workflow, AgentDef, Policy)
services/            Go platform services
  workflow-compiler/ YAML → WorkflowIR compiler (M2 — complete)
  agent-registry/    Capability catalogue service (M3+)
  task-broker/       Capability routing service (M3+)
  memory-service/    KV + vector context store (M3+)
  event-bus/         NATS JetStream async pub/sub (M3+)
  engine-adapter/    Pluggable execution engine bridge (M3+)
  api-gateway/       Public HTTP API surface (M4+)
agents/              Python execution adapters + zynax-sdk
protos/              gRPC contracts (Go + Python stubs generated)
protos/tests/        BDD contract test suites (godog)
infra/               Docker-first dev environment + Helm charts
docs/adr/            Architecture Decision Records (ADR-001 – ADR-018)
docs/milestones/     Per-milestone engineering reviews and release notes
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
<sub>Zynax is a CNCF Sandbox candidate project.</sub>
</div>
