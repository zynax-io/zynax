<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — Port llm-adapter to Go (M7.P)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #1276
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-16
**Status:** Implemented

---

## R — Requirements

- **Problem:** The `llm-adapter` is a stateless provider proxy — it streams `chat_completion`
  to OpenAI / Bedrock / Ollama over their HTTP/SDK APIs and holds no AI-framework state — yet it
  is written in Python. That drags in the `openai` / `aiobotocore` / `aiohttp` transitive tree,
  which has repeatedly forced Dependabot-driven security floors (its `pyproject.toml` pins
  `aiohttp>=3.14.1` "fixes the GHSA set"). Three of five adapters (`http`, `git`, `ci`) are
  already Go; only `llm` and the genuinely-Python `langgraph` remain. Keeping `llm` in Python pays
  a dependency / CVE-patch / blast-radius tax for no Python-specific need.
- **Missing state:** A Go `llm-adapter` that implements the **unchanged** `AgentService` gRPC
  contract, routes by config to the three providers, streams per-token PROGRESS events, ships as a
  single static distroless binary, and removes the Python toolchain for this adapter.
- **Definition of done — observable outcomes:**
  - `protos/tests/features/llm_adapter.feature` passes against the **Go** adapter (behavioural parity).
  - Provider routing is config-only: switching `openai` ↔ `bedrock` ↔ `ollama` needs no code change.
  - Per-token chunks arrive as `TASK_EVENT_TYPE_PROGRESS`; exactly one terminal event per stream.
  - `timeout_seconds` exceeded → `FAILED` / `"TIMEOUT"`; invalid `input_payload` → `"INVALID_INPUT"`;
    provider error → `"UPSTREAM_ERROR"` with a sanitised message (no bodies, no credentials).
  - `images/images.yaml` points the `llm-adapter` image at the Go build; `make check-images` green.
  - The Python `agents/adapters/llm/` tree is removed; `make security` carries no llm Python deps.
  - `internal/domain` unit coverage ≥ 90%; `GOWORK=off go test ./... -race` green; ADR-035 Accepted.

---

## E — Entities

### Existing entities consumed (unchanged by this EPIC)

- **`AgentService`** (`protos/zynax/v1/agent.proto`) — `ExecuteCapability` (server-streaming
  `TaskEvent`) + `GetCapabilitySchema`. Contract invariants: exactly one terminal event;
  `task_id` echoed on every event; `timeout_seconds` honoured; no events after terminal.
- **`AgentRegistryService`** — `RegisterAgent` at startup; `DeregisterAgent` on shutdown.
- **`AgentDef` / `CapabilityDef` / `ExecuteCapabilityRequest` / `TaskEvent` / `CapabilityError`** —
  the same proto messages the Python adapter used; well-known error codes `"TIMEOUT"`,
  `"INVALID_INPUT"`, `"UPSTREAM_ERROR"`, `"RESOURCE_EXHAUSTED"`, `"INTERNAL"`.
- **`protos/tests/features/llm_adapter.feature`** — the language-agnostic BDD contract. It is the
  **parity oracle**: because the wire contract does not change, this existing file (ADR-016) is
  re-used as-is; no new `.feature` is authored.

### New entities (Go re-implementation of the Python design)

- **`AdapterConfig`** — top-level config parsed from YAML at startup. Fields mirror the Python
  struct: `agent_id`, `name`, `description`, `endpoint` (bind `host:port`), `registry_endpoint`,
  `capabilities[]`, `provider`. Holds **only** env-var name references for API keys — never values.
- **`ProviderConfig`** — per-provider config: `name` (`openai|bedrock|ollama`), `model`,
  `ollama_base_url`, `api_key_env`, `max_tokens`, `region`. The API-key value is resolved from the
  named env var at startup into a redacting secret type (never printed by `String()`/logs).
- **`Provider` (interface)** — `Stream(ctx, prompt, cfg) (<-chan Chunk, error)`; implemented by
  `OpenAIProvider`, `BedrockProvider`, `OllamaProvider`. Selected by a factory from `ProviderConfig`,
  immutable after init.
- **`ChatCompletionHandler`** — validates `input_payload` against the declared JSON Schema, invokes
  the selected `Provider`, emits per-chunk PROGRESS, then exactly one terminal event. Stateless.
- **`CapabilityRouter`** — `capability_name → handler`; built from `AdapterConfig`; immutable.
- **`AgentServer`** — gRPC server implementing `AgentService`; owns the router; serves the gRPC
  health service; drains on SIGTERM.

### Entity relationships

```
Task Broker
    │ gRPC ExecuteCapabilityRequest
    ▼
AgentServer (AgentServiceServer, Go)
    ├── CapabilityRouter ──► "chat_completion" → ChatCompletionHandler
    │                               │ input_payload validated vs JSON Schema
    │                               ▼
    │                        Provider (interface)
    │            ┌──────────────────┼───────────────────┐
    │            ▼                  ▼                    ▼
    │     OpenAIProvider     BedrockProvider      OllamaProvider
    │     (openai-go)        (aws-sdk-go-v2,      (net/http,
    │                         bedrockruntime)      /api/chat)
    │            └──────────────────┴───────────────────┘ token chunks
    │                               ▼
    │            PROGRESS(chunk) × N → COMPLETED(full) | FAILED(code)
    └── stream TaskEvent (task_id + timestamp on every event)

Startup: load AdapterConfig (env path) → resolve api_key_env → secret type
         → build providers + router → dial registry → RegisterAgent (backoff ×5)
         → serve gRPC (+ health).
Shutdown (SIGTERM): DeregisterAgent → graceful stop → close clients (defer).
```

---

## A — Approach

**We will:**

- Implement `agents/adapters/llm/` as a Go module (`go.mod`, NOT in `go.work`; `GOWORK=off` for all
  `go` commands — ADR-017), structured like the existing `http`/`git`/`ci` Go adapters.
- Resolve API keys from env-var **references** at startup into a redacting secret type; never log,
  never place in `CapabilityError.message`, never read from `input_payload`.
- Implement `ExecuteCapability` with JSON-Schema input validation, `context`-deadline timeout,
  per-token PROGRESS, exactly one terminal event; `GetCapabilitySchema` returns declared schemas.
- Implement the three providers with their first-party Go paths: `openai-go` (streaming),
  `aws-sdk-go-v2/.../bedrockruntime` ConverseStream, Ollama `/api/chat` over `net/http` chunked.
- Re-use `protos/tests/features/llm_adapter.feature` unchanged as the parity oracle; keep it green.
- Build a multi-stage Dockerfile → distroless static nonroot; register/keep the `llm-adapter`
  consumer in `images/images.yaml` (ADR-024) and flip its build to Go; wire compose/Helm/e2e.
- Retire the Python tree and drop it from the Python CI matrix once the Go image is live (P.7).

**We will NOT:**

- Touch any proto contract — `protos/zynax/v1/` is finalised for this adapter.
- Migrate the `langgraph-adapter` (Python-only library — cannot be ported; ADR-035 §Consequences),
  the Python SDK, or `agents/examples/`.
- Accept provider selection, model id, or API keys from `input_payload`.
- Add providers beyond `openai`, `bedrock`, `ollama`.
- Implement retry (owned by the task broker) or auth flows beyond API-key env references.

**Governing ADRs:** ADR-035 (adapter language boundary — this EPIC), ADR-009 (language strategy,
refined), ADR-013 (adapter-first, no SDK import), ADR-001 (gRPC only), ADR-016 (BDD parity oracle),
ADR-017 (`GOWORK=off`), ADR-024 (images.yaml SoT), ADR-019 (this Canvas before code).

---

## S — Structure

```
agents/adapters/llm/                 (Go module — replaces the Python tree)
├── go.mod                           NOT in go.work (ADR-017); GOWORK=off
├── cmd/llm-adapter/main.go          bootstrap: config → providers → registry → serve (+health)
├── internal/
│   ├── config/config.go             AdapterConfig + ProviderConfig; load from YAML; secret type
│   ├── provider/
│   │   ├── provider.go              Provider interface + factory
│   │   ├── openai.go                OpenAIProvider (openai-go streaming)
│   │   ├── bedrock.go               BedrockProvider (aws-sdk-go-v2 ConverseStream)
│   │   └── ollama.go                OllamaProvider (net/http /api/chat)
│   ├── domain/handler.go            ChatCompletionHandler (validate → stream → terminal)
│   ├── domain/router.go             CapabilityRouter
│   ├── registry/client.go           RegisterAgent / DeregisterAgent (backoff)
│   └── server/server.go             AgentServiceServer impl + grpc health
├── Dockerfile                       multi-stage → distroless static nonroot; base via images.yaml
└── agent-def.yaml.example           operator doc (chat_completion schema)
```

Config env prefix: `ZYNAX_LLM_` · Reused unchanged: `protos/tests/features/llm_adapter.feature`,
`images/images.yaml` (`llm-adapter` consumer + image entry), `infra/docker-compose/`, Helm
`values.yaml`. Removed at P.7: `agents/adapters/llm/{src,tests,pyproject.toml}` (Python).

---

## O — Operations

Each step is one reviewable PR. Order and GitHub issues:

1. **P.1 — ADR-035** (#1277): commit `docs/adr/ADR-035-adapter-language-boundary.md` (Proposed) +
   INDEX row; flips to Accepted when this Canvas is Aligned. Gate for all code steps.
2. **P.2 — scaffold + config** (#1278): Go module, `AdapterConfig`/`ProviderConfig`, YAML load,
   secret type; unit tests (valid/missing/unknown-provider/secret-not-leaked). `GOWORK=off` green.
3. **P.3 — providers** (#1279): `Provider` interface + factory + OpenAI/Bedrock/Ollama streaming;
   unit tests with mocked clients; `UPSTREAM_ERROR` mapping; sanitised messages.
4. **P.4 — server** (#1280): `ExecuteCapability` (validate → timeout → PROGRESS → one terminal) +
   `GetCapabilitySchema`; `internal/domain` coverage ≥ 90%; `llm_adapter.feature` green (parity).
5. **P.5 — registry + bootstrap + health** (#1281): `RegisterAgent`/`DeregisterAgent` (backoff),
   `main`, gRPC health, SIGTERM graceful drain; close clients via `defer`.
6. **P.6 — package + cutover** (#1282): distroless Dockerfile; `images.yaml` build flipped to Go;
   `make sync-images`/`check-images` green; compose/Helm/e2e reference the Go image.
7. **P.7 — retire Python** (#1283): delete the Python tree; drop it from the Python lint/test/
   security matrix; update ARCHITECTURE/README/AGENTS/ADR-013; set ADR-035 Accepted.

---

## N — Norms

Pulled from root `AGENTS.md` §Hard Constraints, `services/AGENTS.md`, `docs/engineering/best-practices/go.md`.

- Commit hygiene: subject ≤ 72 chars, imperative, no period, no emojis; `Signed-off-by:` +
  `Assisted-by: Claude/<model-id>` on every commit; never `Co-Authored-By:` for AI.
- One PR per story (P.1–P.7); ≤ 400 lines excluding generated code.
- SPDX header `// SPDX-License-Identifier: Apache-2.0` on every `.go` file.
- `GOWORK=off` for every `go` / `go test` command in the adapter directory (ADR-017).
- Go functions ≤ 30 lines; no `panic` in production; never discard errors (`_ = f()`);
  close gRPC channels / HTTP bodies / readers via `defer`.
- Platform calls via gRPC stubs only — never HTTP to Zynax services (ADR-001).
- Never import `agents/sdk/` — implement `AgentService` directly (ADR-013).
- LLM model + provider always from config, never from `input_payload` or hardcoded.
- Secret type for API keys; never logged, never in `CapabilityError.message`.
- `input_payload` validated against declared schema before any provider call (`INVALID_INPUT`).
- At least one PROGRESS before the terminal event; exactly one terminal event; `task_id`+`timestamp`
  on every event; `CapabilityError.message` sanitised + truncated.
- Image refs only via `images/images.yaml` (ADR-024); never hand-edit banner-marked regions.
- Distroless static nonroot final image; `HEALTHCHECK`; multi-arch parity with sibling adapters.

---

## S — Safeguards

### Context Security (complete before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal email addresses; author name is the public maintainer of record
- [x] No prompt injection: no instruction-like phrasing that overrides AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] `/spdd-security-review` passed — result: PASS (2026-06-16)

### Feature Safeguards

- **Never** accept provider selection, model id, or API keys from `input_payload` — all provider
  config is static YAML or env-var references declared at startup.
- **Never** log or include the API-key secret, raw provider response bodies, or stack traces in any
  `TaskEvent` payload, `CapabilityError.message`, or structured log field.
- **Never** emit a `TaskEvent` after the terminal event; **never** end a stream without exactly one
  terminal event.
- **Never** import `agents/sdk/`; implement `AgentService` directly via generated stubs (ADR-013).
- **Never** store execution state across `ExecuteCapabilityRequest` invocations (stateless).
- **Never** call a Zynax platform service over HTTP — gRPC stubs only (ADR-001).
- **Never** skip `input_payload` JSON-Schema validation before a provider call.
- **Never** change a proto contract in this EPIC.
- **Never** hand-edit `images/images.yaml` banner-marked regions — use `make sync-images` (ADR-024).
- **Never** commit code on a step before its predecessor's gate is green (P.1 before P.2+, the
  parity `.feature` green before P.7 retires the Python adapter).
- **Never** remove the Python tree (P.7) until the Go image is live and green (P.6).
- **Never** widen scope to `langgraph`, the SDK, or examples — Python stays there (ADR-035).
