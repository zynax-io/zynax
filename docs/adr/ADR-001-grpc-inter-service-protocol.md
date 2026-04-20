# ADR-001: gRPC as Inter-Service Communication Protocol

**Status:** Accepted
**Date:** 2025-04-01
**Deciders:** Zynax Maintainers

---

## Context

Zynax is a multi-service platform where services must communicate
frequently and with strong type guarantees. We need to choose an
inter-service communication protocol.

Candidates evaluated:
- REST/HTTP with JSON
- gRPC with Protocol Buffers
- GraphQL
- Message queues (async-only)

## Decision

**Use gRPC with Protocol Buffers for all synchronous inter-service communication.**

## Rationale

| Criterion | REST+JSON | gRPC+Protobuf |
|-----------|-----------|----------------|
| Strong contracts | ❌ Optional (OpenAPI) | ✅ Enforced by .proto |
| Breaking change detection | ❌ Manual | ✅ `buf breaking` |
| Performance | Baseline | ~7x smaller payload, ~10x faster serialization |
| Streaming | ❌ Limited (SSE) | ✅ Native bidirectional |
| Code generation | ❌ Client-specific | ✅ Multi-language |
| Cloud-native ecosystem | ✅ | ✅ (Envoy, Istio, K8s native) |

## Consequences

**Positive:**
- Proto files serve as living API documentation.
- Breaking changes are caught automatically by `buf breaking` in CI.
- Generated client stubs eliminate handwritten HTTP clients.
- Native streaming enables real-time agent event subscription.

**Negative:**
- gRPC is not human-readable in the browser (mitigated by grpc-gateway in api-gateway).
- Requires proto toolchain setup (mitigated by `make generate-protos`).
- Learning curve for contributors unfamiliar with proto3.
