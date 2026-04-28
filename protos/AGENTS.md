# protos/ — Engineering Contract

> Proto files are the **API contract**. They are reviewed like production code.
> A bad proto is a breaking change waiting to happen.
> Language interoperability guide: `docs/patterns/proto-interop.md`.
> BDD contract testing guide: `docs/patterns/bdd-contract-testing.md`.

---

## File Layout

```
protos/
├── zynax/v1/                    ← One proto per service
│   ├── agent_registry.proto
│   ├── task_broker.proto
│   ├── memory.proto
│   └── ...
├── generated/                   ← Auto-generated. NEVER edit manually.
│   ├── go/zynax/v1/
│   └── python/zynax/v1/
└── tests/                       ← BDD contract tests (godog)
```

---

## Naming Conventions

```protobuf
// Package: reverse-DNS, versioned
syntax = "proto3";
package zynax.v1;
option go_package = "github.com/zynax-io/zynax/gen/go/zynax/v1";

// Service: PascalCase, singular
service AgentRegistryService { ... }

// RPC: PascalCase verb+noun, full request/response wrappers
rpc RegisterAgent(RegisterAgentRequest) returns (RegisterAgentResponse);
rpc WatchAgentEvents(WatchAgentEventsRequest) returns (stream AgentEvent);

// Enums: SCREAMING_SNAKE_CASE, prefixed with type name, 0 = UNSPECIFIED
enum AgentStatus {
  AGENT_STATUS_UNSPECIFIED = 0;
  AGENT_STATUS_ACTIVE = 1;
  AGENT_STATUS_OFFLINE = 2;
}
```

---

## Backward Compatibility Rules

Breaking a consumer is a production incident. These rules prevent it:

- **Never** remove a field — mark it `reserved` and add to `reserved` names.
- **Never** renumber a field — field numbers are permanent.
- **Never** change a field type.
- **Never** rename a package.
- Adding new fields is safe (they default to zero value).
- Adding new enum values is safe (unknown values are ignored).
- Breaking changes require a new version package: `zynax/v2/`.

```protobuf
// Removing a field correctly:
message AgentSpec {
  reserved 3;
  reserved "old_field_name";
  string id = 1;
  string display_name = 2;
}
```

---

## Generate Command

```bash
make generate-protos
# Runs: buf generate --template buf.gen.yaml
# Output: protos/generated/go/ and protos/generated/python/
# Always commit generated files — never edit them
```

---

## Contract Test Mandate

Every RPC method **must** have at least one BDD scenario in `protos/tests/`.
See `protos/tests/AGENTS.md` for running instructions.
See `docs/patterns/bdd-contract-testing.md` for authoring patterns.
