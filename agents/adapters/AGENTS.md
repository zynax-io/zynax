# agents/adapters/ — AGENTS.md

> Adapter-First Integration. No SDK Required.
> ADR-013: any system becomes a capability by implementing `AgentService` gRPC.
> Inherits all rules from root `AGENTS.md` and `agents/AGENTS.md`.
> Full implementation patterns: `docs/patterns/python-agent-guide.md` (Python adapters)
> and `docs/patterns/go-service-patterns.md` (Go adapters).

---

## Core Principle

> Any system becomes a capability by implementing the `AgentService` gRPC contract.
> No language. No framework. No SDK import.

```
External System      Adapter             Zynax capability
─────────────────    ──────────────      ─────────────────
REST API        →    http/    (Go)     → call_payments_api
GitHub API      →    git/     (Go)     → open_pr, request_review
Jenkins/CI      →    ci/      (Go)     → trigger_workflow, get_run_status
Bedrock/OpenAI  →    llm/     (Go)     → chat_completion
LangGraph app   →    langgraph/ (Py)   → research_topic
```

---

## Adapter Directory

```
agents/adapters/
├── http/          ← Wraps any REST API (config-only) — Go, #380
├── git/           ← GitHub/GitLab operations          — Go, #381
├── ci/            ← GitHub Actions / Jenkins triggers  — Go, #382
├── llm/           ← Bedrock, Ollama, OpenAI            — Go, #383 (ported #1276)
└── langgraph/     ← LangGraph app as capability        — Python, #384
```

All adapters are M5 deliverables. Parent epic: #377. Canvas: `docs/spdd/377-adapter-library/canvas.md`.
BDD `.feature` file required before any implementation (ADR-016).
Individual REASONS Canvas required before any implementation (ADR-019).

---

## Language Strategy

| Adapter | Language | Reason |
|---------|----------|--------|
| `http/` | **Go** | Stateless REST proxy — no ML deps. Single binary, small image. |
| `git/` | **Go** | GitHub/GitLab API calls over HTTP — same profile as http. |
| `ci/` | **Go** | CI API calls (GitHub Actions, Jenkins REST) — same profile. |
| `llm/` | **Go** | Stateless provider proxy over OpenAI / Bedrock / Ollama HTTP APIs — no framework state. Ported from Python under ADR-035. |
| `langgraph/` | **Python** | LangGraph is a Python framework — adapter must import graph code. |

**Rule:** Default to Go for any adapter that is a stateless HTTP/gRPC proxy. Only choose Python
when the wrapped system's primary SDK or framework is Python-native (AI/ML exclusively).

---

## The Two Adapter Interfaces

**Zynax-facing:** Implement `ExecuteCapability` gRPC (stream of `TaskEvent`).
Register capabilities in an `AgentDef` YAML on startup via `AgentRegistryService.RegisterAgent`.
Deregister via `AgentRegistryService.DeregisterAgent` on graceful shutdown.

**System-facing:** Whatever protocol the wrapped system speaks (REST, gRPC, CLI).
The adapter translates between the two sides.

---

## Go Adapter Module Layout

```
agents/adapters/<name>/
├── go.mod                   module github.com/zynax-io/zynax/agents/adapters/<name>
├── cmd/<name>-adapter/
│   └── main.go              gRPC server bootstrap + graceful shutdown
├── internal/
│   ├── adapter.go           CapabilityRouter + system-facing CapabilityHandler
│   └── config.go            AdapterConfig YAML parsing
├── Dockerfile               two-stage Alpine (golang:*-alpine → alpine:*)
└── agent-def.yaml.example   operator documentation
```

All Go adapter modules are added to `go.work` with `use agents/adapters/<name>`.

## Python Adapter Module Layout

```
agents/adapters/<name>/
├── pyproject.toml           uv-managed; grpcio, protobuf, pydantic-settings in deps
├── src/<name>_adapter/
│   ├── server.py            gRPC server; ExecuteCapability; GetCapabilitySchema
│   ├── router.py            CapabilityRouter
│   ├── config.py            AdapterConfig + provider-specific config
│   └── ...                  provider modules, graph loader, etc.
├── Dockerfile               two-stage Python image (python:3.12-slim)
└── agent-def.yaml.example
```

---

## Rules

- Adapters are **stateless** — no adapter-local state that survives restart.
- Capabilities declared in `AgentDef` YAML, not hardcoded in source.
- Capability names must be `snake_case`, 1–64 characters, matching the registry entry exactly.
- Emit at least one `TASK_EVENT_TYPE_PROGRESS` event for tasks >2 seconds.
- Always emit exactly one `TASK_EVENT_TYPE_COMPLETED` or `TASK_EVENT_TYPE_FAILED` as the final event.
- Never emit events after the terminal event.
- Echo `task_id` on every `TaskEvent`. Populate `timestamp` on every event.
- Honour `timeout_seconds` — emit `TASK_EVENT_TYPE_FAILED` with code `"TIMEOUT"` on breach.
- `CapabilityError.message` is human-readable and sanitised — no raw API responses, no stack traces, no credential values.
- Never import `agents/sdk/` — adapters implement `AgentService` directly via generated stubs (ADR-013).
- Never call Zynax platform services via HTTP (Python adapters) — gRPC stubs only.
- Go adapters: `GOWORK=off` for all `go test` and `go build` (ADR-017). `CGO_ENABLED=0`, `-trimpath`.
- Python adapters: `asyncio` throughout; no blocking calls on the event loop.

---

## SSRF Prevention (http-adapter)

The http-adapter maps capability names to HTTP routes via static config only. No URL fields are
accepted in `input_payload` — all HTTP endpoints are declared in `AgentDef` YAML at startup.
This is a hard requirement enforced by the Canvas Safeguards (ADR-019).
