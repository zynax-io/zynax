<!-- SPDX-License-Identifier: Apache-2.0 -->

<div align="center">

# Keel

**The declarative control plane for AI agent workflows**

[![CI](https://github.com/keel-io/keel/actions/workflows/ci.yml/badge.svg)](https://github.com/keel-io/keel/actions/workflows/ci.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![CNCF Sandbox Candidate](https://img.shields.io/badge/CNCF-Sandbox_Candidate-026be0.svg)](https://cncf.io)
[![Go 1.22+](https://img.shields.io/badge/go-1.22+-00add8.svg)](https://go.dev)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/keel-io/keel/badge)](https://securityscorecards.dev)

[Quickstart](#-quickstart) · [Architecture](ARCHITECTURE.md) · [Documentation](docs/) · [Contributing](CONTRIBUTING.md) · [Roadmap](ROADMAP.md)

</div>

---

## What is Keel?

> Keel is to AI workflows what Kubernetes is to containers — a control plane
> that abstracts the execution layer behind a declarative, versionable API.

Keel lets you define AI agent workflows as YAML manifests and execute them
on any workflow engine (Temporal, LangGraph, Argo) without changing your
workflow definition.

```yaml
# code-review.yaml
kind: Workflow
apiVersion: keel.io/v1

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
keel apply code-review.yaml
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

## Quickstart

**Prerequisites:** Docker Desktop. Nothing else.

```bash
git clone https://github.com/keel-io/keel.git
cd keel

# Build the dev tools image (Go + Python + all tooling)
make build-tools

# Start the full local stack
make dev-up

# Apply a workflow
keel apply spec/workflows/examples/code-review.yaml

# Check status
keel get workflows -n engineering
```

---

## Repository Structure

```
spec/           Workflow YAML manifests and schemas
services/       Go platform services (7 services)
agents/         Python execution adapters + optional SDK
protos/         gRPC contracts (Go + Python stubs generated)
infra/          Docker-first dev environment + Helm charts
docs/adr/       Architecture Decision Records (ADR-001 – ADR-015)
```

---

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a PR.
Key rules: `.feature` file before any code, BDD-first, Docker-only dev.

---

## Security

Report vulnerabilities via [GitHub Security Advisories](https://github.com/keel-io/keel/security/advisories/new).
See [SECURITY.md](SECURITY.md).

---

## License

Apache License 2.0 — see [LICENSE](LICENSE).

SPDX-License-Identifier: Apache-2.0

---

<div align="center">
<sub>Keel is a CNCF Sandbox candidate project.</sub>
</div>
