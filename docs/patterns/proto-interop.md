# Proto Interoperability Guide

> The proto files in `protos/zynax/v1/` are the **universal integration boundary**.
> Any language with gRPC support can call any Zynax service or implement any
> capability — without the Python SDK and without any Zynax-specific library.

---

## Client vs Server Role

**Client role** — Your system calls Zynax services (registers an agent, submits a task,
reads memory). You need generated stubs for your language and a gRPC channel.

**Server role** — Your system receives and executes tasks from the task-broker. You implement
the `AgentService` contract. The Python SDK automates this boilerplate; in other languages
you implement it directly against generated stubs.

---

## Language Support Tiers

| Tier | Languages | Generated stubs committed? | Maintained by |
|------|-----------|---------------------------|---------------|
| **1 — Official** | Go (platform), Python (SDK + adapters) | Yes — `protos/generated/go/` and `protos/generated/python/` | Core maintainers |
| **2 — Supported** | Any language with a gRPC `buf generate` plugin | No — generate locally | Maintainers provide the proto source |
| **3 — Community** | Language-specific wrapper libraries on Tier 2 | N/A | Community contributors |

---

## Go — Importing Generated Stubs

Go services import the generated stubs as a standard Go module dependency.
Import path: `github.com/zynax-io/zynax/gen/go/zynax/v1`.

Within the monorepo, services reference stubs via `go.work` without declaring
an external dependency. External Go consumers use a standard `go.mod` dependency.

There is no separate Go SDK — the generated stubs and a gRPC channel are sufficient.

---

## Python — Two Paths

**Path A — Raw generated stubs** (`protos/generated/python/`)

Use when:
- Calling Zynax services as a client
- Building a non-SDK adapter with full control over the gRPC lifecycle
- Integrating Zynax into an existing Python service

**Path B — `zynax-sdk`** (`agents/sdk/`)

Use when:
- Building a new Python agent that receives and executes tasks (server role)
- You want registration, heartbeat, task routing, `AgentContext` injection, and
  graceful shutdown handled automatically
- Building on top of an AI framework (LangGraph, AutoGen, CrewAI)

---

## Other Languages — Generating Stubs Locally

1. Install `buf` and the gRPC plugin for your target language.
2. Add a `buf.gen.yaml` pointing at `protos/zynax/v1/`.
3. Run `buf generate` to produce idiomatic stubs.
4. Implement the client or server role against those stubs.

Every major language has a gRPC plugin: Java, TypeScript/Node.js, Rust, C#, Kotlin,
Swift, Ruby, Dart, PHP, and more.

---

## The Interoperability Guarantee

- A Go service can invoke a capability implemented by a Python agent via gRPC contracts alone.
- A TypeScript web client can submit a workflow to the API Gateway using stubs generated
  from the same proto source as the Go services.
- A Java enterprise system can register capabilities and receive tasks from the task-broker,
  becoming a first-class participant without Python, Go, or any Zynax SDK.
- A Rust adapter is indistinguishable from a Python SDK agent from the platform's perspective.

The proto contract is the only thing that matters for interoperability.

---

## Consumer Impact of Proto Changes

**Backward-compatible changes** (new fields, methods, enum values):
Existing generated stubs in all languages continue to work without regeneration.

**Breaking changes** (new package version `zynax/v2/`):
A new import path is published. Consumers opt in on their own schedule.
The old version remains available until formally deprecated with a migration timeline
in `docs/migrations/`.
