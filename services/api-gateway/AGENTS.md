# services/api-gateway — AGENTS.md

> Go 1.22+. Inherits rules from root `AGENTS.md` and `services/AGENTS.md`.
> **Status: M4+ (not yet implemented).** No BDD contract tests yet.

---

## Purpose

The API Gateway is the **single external entry point** to the Zynax platform.
It translates HTTP REST requests from external clients into gRPC calls to internal
services via `grpc-gateway`.

- Exposes a versioned REST API (`/api/v1/`).
- Authenticates and authorizes external callers (API keys + JWT).
- Rate limits per client using token bucket algorithm.
- Transcodes REST ↔ gRPC via `grpc-gateway` annotations.
- Validates and sanitizes all external inputs before forwarding.
- Returns consistent error responses — never leaks internal gRPC details.

Does NOT: implement business logic · store data · call backing services directly except via gRPC.

---

## Internal Layout

```
services/api-gateway/
├── cmd/api-gateway/main.go
├── internal/
│   ├── api/
│   │   ├── gateway.go          ← grpc-gateway mux registration
│   │   ├── auth.go             ← JWT + API key middleware
│   │   └── ratelimit.go        ← token bucket per client
│   ├── domain/
│   │   └── (minimal — gateway has no domain logic)
│   └── infrastructure/
│       └── clients.go          ← gRPC clients for internal services
├── go.mod
└── Dockerfile
```

Config env prefix: `ZYNAX_GW_` · HTTP port: 8080 · gRPC port: 50057

---

## Running Tests

```bash
cd services/api-gateway
GOWORK=off go test ./... -race -timeout 60s
```
