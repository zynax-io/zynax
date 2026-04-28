# services/memory-service — AGENTS.md

> Go 1.22+. Inherits rules from root `AGENTS.md` and `services/AGENTS.md`.
> **Status: M3+ (not yet implemented).** BDD contract tests exist in `protos/tests/`.

---

## Purpose

The Memory Service provides **namespaced, shared memory** for agents.
It is the only persistent context store in the mesh.

| Type | Backend | Use case |
|------|---------|---------|
| Key-Value | Redis | Fast ephemeral state, agent scratch-pads, session data |
| Vector | PostgreSQL + pgvector | Semantic similarity search, RAG, long-term memory |

- Creates and deletes namespaces with optional TTL.
- Set, get, delete, and list key-value entries per namespace.
- Upserts and deletes vector embeddings per namespace.
- Semantic similarity search (ANN via pgvector).
- Namespace isolation: agents cannot read each other's namespaces without permission.

Does NOT: generate embeddings (caller provides the vector) · store task results.

---

## Internal Layout

```
services/memory-service/
├── cmd/memory-service/main.go
├── internal/
│   ├── api/
│   │   └── handler.go          ← All 10 RPCs mapped to domain
│   ├── domain/
│   │   ├── kv.go               ← KVStore interface + KVEntry model
│   │   ├── vector.go           ← VectorStore interface + Embedding model
│   │   ├── namespace.go        ← Namespace model + NamespaceRepository
│   │   └── errors.go           ← ErrNamespaceNotFound, ErrKeyNotFound
│   └── infrastructure/
│       ├── redis_kv.go         ← RedisKVStore
│       └── pgvector.go         ← PgvectorStore
├── go.mod
└── Dockerfile
```

Config env prefix: `ZYNAX_MEMORY_` · gRPC port: 50053

---

## Running Tests

```bash
cd services/memory-service
GOWORK=off go test ./... -race -timeout 60s

# BDD contract tests
cd protos/tests
GOWORK=off go test ./memory_service/... -v -timeout 60s
```
