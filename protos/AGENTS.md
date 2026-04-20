# protos/ — AGENTS.md

> Proto files are the **API contract**. They are reviewed like production code.
> A bad proto is a breaking change waiting to happen.
> Read this entirely before touching any `.proto` file.

---

## Proto Authoring Rules

### 1. File Layout

```
protos/
├── zynax/
│   ├── v1/
│   │   ├── agent_registry.proto     ← One proto per service
│   │   ├── task_broker.proto
│   │   ├── memory.proto
│   │   └── api_gateway.proto
│   └── common/
│       ├── types.proto              ← AgentId, TaskId, Timestamp wrappers
│       ├── errors.proto             ← Standard error details
│       └── pagination.proto         ← PageRequest / PageResponse
└── generated/                       ← Auto-generated. Never edit manually.
    └── python/
        └── zynax/
            └── v1/
```

### 2. Naming Conventions

```protobuf
// ✅ Package: reverse-DNS, versioned
syntax = "proto3";
package zynax.v1;
option go_package = "github.com/zynax-io/zynax/gen/go/zynax/v1";
option java_package = "io.zynax.v1";

// ✅ Service: PascalCase, singular, descriptive
service AgentRegistryService { ... }

// ✅ RPC: PascalCase verb+noun, full request/response wrapper
rpc RegisterAgent(RegisterAgentRequest) returns (RegisterAgentResponse);
rpc GetAgent(GetAgentRequest) returns (GetAgentResponse);
rpc ListAgentsByCapability(ListAgentsByCapabilityRequest)
    returns (ListAgentsByCapabilityResponse);
rpc WatchAgentEvents(WatchAgentEventsRequest)
    returns (stream AgentEvent);  // streaming: explicitly named

// ❌ Never
rpc Register(AgentSpec) returns (string);  // bare type, no wrapper
rpc Do(Request) returns (Response);        // vague name
```

### 3. Message Conventions

```protobuf
// ✅ Every top-level request message includes:
message RegisterAgentRequest {
  string request_id = 1;               // Idempotency key (UUID)
  AgentSpec spec = 2;                  // Main payload
}

// ✅ Every top-level response message includes:
message RegisterAgentResponse {
  string agent_id = 1;                 // Result
  google.protobuf.Timestamp created_at = 2;
}

// ✅ Nested message for complex types
message AgentSpec {
  string id = 1;
  string display_name = 2;
  repeated string capabilities = 3;
  map<string, string> metadata = 4;
  AgentStatus status = 5;
}

// ✅ Enums: SCREAMING_SNAKE_CASE, prefixed with type name
enum AgentStatus {
  AGENT_STATUS_UNSPECIFIED = 0;   // Always have an UNSPECIFIED = 0
  AGENT_STATUS_ACTIVE = 1;
  AGENT_STATUS_DRAINING = 2;
  AGENT_STATUS_OFFLINE = 3;
}
```

### 4. Backward Compatibility Rules

These rules prevent breaking consumers:

- **Never** remove a field. Mark it `reserved` and add to `reserved` names.
- **Never** renumber a field. Field numbers are permanent.
- **Never** change a field type.
- **Never** rename a package.
- Adding new fields is safe (they default to zero value).
- Adding new enum values is safe (unknown values are ignored).
- Breaking changes require a new version package: `zynax/v2/`.

```protobuf
// Removing a field correctly:
message AgentSpec {
  reserved 3;                    // Old field number preserved
  reserved "old_field_name";     // Old field name reserved
  string id = 1;
  string display_name = 2;
  // capabilities removed in v1.5 — use AgentCapabilityService instead
}
```

### 5. Generate Command

```bash
make generate-protos
# Runs: buf generate --template buf.gen.yaml
# Output goes to protos/generated/python/
# Commit generated files — never edit them
```

### 6. buf.yaml Configuration

```yaml
# protos/buf.yaml
version: v2
modules:
  - path: zynax
lint:
  use:
    - DEFAULT
  except:
    - PACKAGE_VERSION_SUFFIX  # We handle versioning ourselves
breaking:
  use:
    - FILE  # Detect breaking changes between versions
```

---

## 8. Consuming These Contracts — Language Interoperability Guide

> The proto files in `protos/zynax/v1/` are the **universal integration boundary**
> for Zynax. Any language, framework, or runtime that can speak gRPC can call
> any Zynax service, implement any adapter, or build any agent — without the Zynax
> Python SDK and without importing any Zynax-specific library beyond the generated
> stubs.
>
> This section explains what that means concretely for each supported language tier.

---

### The Client vs Server Distinction

Before choosing how to consume these contracts, understand which role your code plays:

**Client role** — Your system calls Zynax services (registers an agent, queries capabilities,
submits a task, reads memory). You need the generated stubs for your language and a gRPC
channel. Nothing else. The generated stubs are all you need.

**Server role** — Your system receives and executes tasks from the task-broker. You are
implementing the `AgentService` contract. You need the generated stubs plus the logic
to handle `ExecuteCapability` RPCs, stream `TaskEvent` responses, and register your
capabilities on startup. The Python SDK automates this boilerplate; in other languages
you implement it directly against the generated stubs.

Most integrations are one or the other. Some are both (an SDK agent that also queries
the memory service is both a server receiving tasks and a client reading memory).

---

### Language Support Tiers

| Tier | Languages | Generated stubs committed? | Maintained by |
|------|-----------|---------------------------|---------------|
| **1 — Official** | Go (platform services), Python (SDK + adapters) | Yes — in `gen/go/` and `protos/generated/python/` | Core maintainers |
| **2 — Supported** | Any language with a gRPC plugin for `buf generate` | No — generate locally | Maintainers provide the proto source; consumers generate |
| **3 — Community** | Language-specific wrapper libraries built on Tier 2 | N/A | Community contributors |

**Tier 1** means Zynax CI validates the generated stubs on every proto change and
ships them alongside the source. Consumers import them directly.

**Tier 2** means you run `buf generate` with your language's plugin against the
`protos/zynax/v1/` source and get idiomatic stubs. Every major language has a gRPC
plugin: Java, TypeScript/Node.js, Rust, C#, Kotlin, Swift, Ruby, Dart, PHP, and more.
The proto source is the contract — stub generation is a build step, not a dependency.

**Tier 3** is community-contributed SDKs or client libraries built on top of Tier 2
stubs. The Zynax project welcomes these but does not maintain them.

---

### Go — Importing the Generated Stubs

Go platform services and any Go consumer import the generated stubs as a standard
Go module dependency. The import path follows the `go_package` option declared in
each proto file: `github.com/zynax-io/zynax/gen/go/zynax/v1`.

The generated stubs contain typed request and response structs, service client
interfaces, and service server interfaces — everything needed to call or implement
any Zynax service. There is no separate Go SDK because the generated stubs and a
gRPC channel are sufficient. Go's type system and the generated interfaces provide
the same safety guarantees that a higher-level SDK would add in a less statically
typed language.

Within the Zynax monorepo, Go services reference the generated stubs via the
`go.work` workspace without needing to declare an external module dependency.
External Go consumers declare the import as a standard `go.mod` dependency.

---

### Python — Two Paths

Python consumers have two options depending on the role they play:

**Path A — Raw generated stubs** (`protos/generated/python/`)

Use this path when:
- Your system calls Zynax services as a client (submitting tasks, querying agents, reading memory)
- You are building a non-SDK adapter and want full control over the gRPC lifecycle
- You are integrating Zynax into an existing Python service that already manages its own gRPC connections

The raw stubs give you typed request/response classes and service stubs for every
proto. You manage the channel, the connection lifecycle, and the serialisation
yourself. This is the lowest-level, most explicit path.

**Path B — The `zynax-sdk` Python package** (`agents/sdk/`)

Use this path when:
- You are building a new Python agent that receives and executes tasks (server role)
- You want the SDK to handle agent registration, heartbeat, task routing, `AgentContext`
  injection, streaming event delivery, and graceful shutdown
- You are building on top of an AI framework (LangGraph, AutoGen, CrewAI) and want
  the platform plumbing to disappear so you can focus on agent logic

The SDK wraps the raw stubs and adds the boilerplate that every agent needs. It does
not change the proto contract — it implements it. See `agents/sdk/AGENTS.md` for the
decision between these two paths in detail.

---

### Other Languages — Generating Stubs Locally

For any language not in Tier 1, the workflow is:

1. Install `buf` and the gRPC plugin for your target language.
2. Add a `buf.gen.yaml` template that points at the `protos/zynax/v1/` source.
3. Run `buf generate` to produce idiomatic stubs for your language.
4. Implement the client (if calling Zynax services) or the server (if implementing
   an adapter) against those stubs.

The generated stubs are entirely derived from the proto source. They contain no
Zynax business logic. They change only when the proto source changes. Treat them
as a build artifact, not a dependency to version separately.

When Zynax is published to the Buf Schema Registry (BSR), consumers will be able to
depend on the registry directly instead of running `buf generate` locally. This is
planned for Milestone 1.

---

### The Interoperability Guarantee

Because every integration point is defined in proto, the following is always true:

- A Go service can invoke a capability implemented by a Python agent. They communicate
  exclusively via the gRPC contracts in `protos/zynax/v1/`. Neither knows what language
  the other is written in.

- A TypeScript web client can call the API Gateway, submit a workflow, and receive
  streaming events — using stubs generated from the same proto source as the Go
  services that process the request.

- A Java-based enterprise system can register capabilities and receive tasks from
  the task-broker, making it a first-class participant in Zynax workflows without
  Python, Go, or any Zynax SDK.

- A Rust high-performance adapter can serve capabilities to the task-broker,
  implement the `AgentService` contract directly from generated Rust stubs, and
  be indistinguishable from a Python SDK agent from the platform's perspective.

The proto contract is the only thing that matters for interoperability. Zynax is
agnostic about what is on the other end of the gRPC connection.

---

### Contract Versioning and Consumer Impact

When a proto change is merged (see §4, Backward Compatibility Rules, and
`CONTRIBUTING.md §13`), the impact on consumers depends on the change type:

**Backward-compatible changes** (new fields, new methods, new enum values):
Existing generated stubs in all languages continue to work. Consumers can
regenerate to access new fields, but are not required to.

**Breaking changes** (new package version `zynax/v2/`):
A new import path is published. Consumers opt in to the new version on their
own schedule. The old version remains available until formally deprecated via
a migration timeline documented in `docs/migrations/`.

This means Tier 2 and Tier 3 consumers are never broken by a backward-compatible
proto change and have explicit advance notice before any breaking change is
finalised.

---

### 7. Contract Tests

Every RPC method in every proto **must** have at least one BDD scenario
in the consuming service's `tests/features/` directory that verifies the
contract is honoured.

```gherkin
# tests/features/agent_registry_contract.feature
Feature: AgentRegistryService Contract
  The agent registry gRPC contract as defined in zynax/v1/agent_registry.proto

  Scenario: RegisterAgent returns valid response for valid spec
    Given a valid RegisterAgentRequest with request_id and spec
    When RegisterAgent is called
    Then the response is a RegisterAgentResponse
    And response.agent_id is a non-empty string
    And response.created_at is a valid RFC3339 timestamp

  Scenario: GetAgent returns NOT_FOUND for unknown agent_id
    Given an agent_id that does not exist in the registry
    When GetAgent is called with that agent_id
    Then the gRPC status code is NOT_FOUND
    And the error message contains the agent_id
```
