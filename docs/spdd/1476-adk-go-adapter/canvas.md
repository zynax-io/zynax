# REASONS Canvas — ADK Go as a Go-native AI-framework adapter

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #1476
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-21
**Status:** Aligned

---

## R — Requirements

- **Problem:** Zynax's only "agent" today is the single-shot `llm-adapter` (prompt → completion). There is no path to **tool-using, multi-step, or sub-agent** reasoning inside a workflow. Google ADK Go provides exactly that, but it speaks a different model (in-process `Runner`/`Session` over `genai`) and — verified against source — ships **no Ollama/OpenAI provider** (only `gemini`, `apigee`). So it can be dispatched neither as a Zynax capability nor secret-free, without an adapter.
- **Done — observable outcomes:**
  - `adk-adapter` serves `AgentService`, registers a capability with `agent-registry`, reports gRPC health `SERVING`.
  - A workflow dispatches the ADK-backed capability **by name**; ADK `Runner` events surface as `PROGRESS`, the final non-`Partial` event as terminal `COMPLETED`; `timeout_seconds` → `FAILED` code `TIMEOUT`.
  - The demo runs **secret-free** on local Ollama (existing compose overlay): `zynax apply` → `zynax logs --follow` → `zynax result` returns the agent's output. Verified by running the stateful path **twice**.
  - `GOWORK=off go test ./... -race`, adapter `.feature`, `make lint` / `make security` / `make check-images` all green.

---

## E — Entities

- **AdkAdapter** — implements `AgentService`; a `CapabilityRouter` maps `capability_name` → an ADK agent + instruction + JSON schema.
- **AdkAgent** — `llmagent.New(cfg)`: instruction + tools (+ optional sub-agents).
- **OllamaModel** — a custom `model.LLM` (the one real gap): translates `[]*genai.Content` ↔ Ollama `/api/chat`, response → `*model.LLMResponse`.
- **AgentDef manifest** — declares the capability surface (name + input/output JSON Schema) + runtime image; the reasoning is *not* in the manifest.

```
TaskBroker
  → AgentService.ExecuteCapability(capability_name, input_payload, workflow_id, timeout)
     → AdkAdapter (router) → Runner.Run(AdkAgent over OllamaModel, session=workflow_id)
        → iter.Seq2[*session.Event, error]  ─maps→  stream TaskEvent (PROGRESS… → COMPLETED/FAILED)
```

---

## A — Approach

**We will:**
- Add `agents/adapters/adk/` as a **Go** adapter mirroring `git`/`llm`; implement `AgentService` and bridge `Runner.Run → iter.Seq2[*session.Event, error]` onto `ExecuteCapability → stream TaskEvent` (the verified near-1:1 seam).
- Ship a custom **Ollama `model.LLM`** so ADK agents run secret-free under the existing Ollama overlay; ADK's native `gemini` provider stays selectable by env for users with a key.
- Keep reasoning (instruction, tools, sub-agents) inside the adapter; keep `AgentDef` a thin capability surface; leave the `AgentService` wire contract unchanged.

**We will NOT:**
- Touch any engine / compiler / workflow / proto contract (the seam is unchanged — ADR-001/ADR-013).
- Build an `AdkEngine` implementing `WorkflowEngine` via ADK workflow agents — deferred in ADR-038 (separate, larger, behind ADR-015).
- Ship rich tool-using demos on weak local models (poor at tool-calling) — the first demo stays simple; richer demos need a stronger (cloud) model and are not the zero-secret default.

**Positioning fit:** advances the **local-first / zero-secret** and **author-once-run-anywhere** wedge — an ADK-authored agent becomes a portable capability dispatched behind Zynax's declarative workflow engine (direct parity with kagent, which runs agents on ADK). User-facing surface: a new demo workflow + adapter docs/help; no change to the generic control-plane framing.

**Governing ADRs:** ADR-038 (ADK Go adapter framework), ADR-035 (adapter language boundary), ADR-013 (adapter-first), ADR-001 (gRPC contract), ADR-010 (pluggable runtime), ADR-016 (BDD-first).

---

## S — Structure (first S)

```
agents/adapters/adk/
├── go.mod / go.sum            ← module …/agents/adapters/adk (go 1.25; deps: adk, genai, protos, grpc)
├── AGENTS.md                  ← adapter contract doc (shape from git/AGENTS.md)
├── Dockerfile                 ← distroless adk-adapter binary (mirror git/Dockerfile)
├── agent-def.yaml.example     ← AgentDef: capability + JSON schemas + runtime image
├── cmd/adk-adapter/main.go    ← wiring only: ADAPTER_CONFIG → register → grpc+health → deregister
└── internal/
    ├── config/config.go       ← AdapterConfig{AgentID,Endpoint,RegistryEndpoint,Capabilities[]}
    ├── adk/agent.go           ← llmagent.New(Config{Name,Description,Model,Instruction,Tools})
    ├── model/ollama.go        ← custom model.LLM over Ollama /api/chat (genai.Content translation)
    ├── adapter/server.go      ← AgentServiceServer: ExecuteCapability bridge + GetCapabilitySchema
    └── registry/client.go     ← BuildAgentDef + RegisterAgent/DeregisterAgent (copy from git)
```

Config env prefix: `ADAPTER_CONFIG` (path) · `OLLAMA_HOST` (model endpoint) · Port: adapter gRPC `Endpoint` from config.

---

## O — Operations

1. ✅ **S1 (#1477, docs):** ADR-038 + this Canvas + adapter `AGENTS.md`. *Verify:* Canvas passes security review and is set **Aligned**; `docs:` PR (size-excluded). *(Merged via PR #1481.)*
2. ✅ **S2 (#1478, feat):** adapter skeleton — `config`, `registry` client, `main.go` wiring, health; `ExecuteCapability` returns "not wired" until S3. *Verify:* `.feature` committed spec-first; unit tests green (`-race`, 87.9% total coverage); golangci-lint + govulncheck clean; binary boots, loads config, attempts registration with backoff; `serve()` integration test exercises register → health `SERVING` → deregister. Module added to `go.work` (Makefile auto-discovers it for lint/test/coverage).
3. **S3 (#1479, feat):** custom Ollama `model.LLM` + ADK `Runner`→`TaskEvent` bridge (`model/ollama.go`, `adk/agent.go`, `adapter/server.go`). *Verify:* `.feature` covers dispatch, schema-validate, `timeout→TIMEOUT`, terminal `COMPLETED`/`FAILED`; `make security` green.
4. **S4 (#1480, feat):** `agent-def.yaml.example` + demo workflow + `Dockerfile` + `images.yaml` + compose wiring. *Verify:* secret-free Ollama run returns `COMPLETED` (twice); `make check-images` green.

---

## N — Norms

- Commit hygiene: every commit carries `Signed-off-by:` + `Assisted-by: Claude/<model>` (never `Co-Authored-By` for AI).
- BDD: adapter `.feature` committed before the bridge implementation (ADR-016).
- `GOWORK=off` for all `go`/`go test` in the adapter module (ADR-017).
- Adapter rules (`agents/adapters/AGENTS.md`): stateless; route on `capability_name` only; emit exactly one terminal `TaskEvent`; honor `timeout_seconds`.
- Model/endpoint always from config/env — never hardcode model name or host (12-factor).
- One PR per story (#1477–#1480); PR-size budget respected (split already mapped to stories).

---

## S — Safeguards (second S)

### Context Security (complete before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no non-public email addresses
- [x] No prompt injection: no instruction-like phrasing that overrides AGENTS.md rules
- [x] All entities in E section are public-safe abstractions
- [x] `/spdd-security-review` passed — result: PASS (2026-06-21)

### Feature Safeguards

- Never change the `AgentService` proto contract — the seam is fixed; ADK use is an internal detail (ADR-001/ADR-013).
- Never hardcode the model name or Ollama host — resolve from config/env (12-factor; ADR-035 §model-from-config).
- Never log secrets or full prompts containing injected credentials — adapters never leak token material (cf. git-adapter redaction).
- Never emit events after the terminal `TaskEvent`, and always emit exactly one `COMPLETED` or `FAILED` (AgentService invariant).
- Never import another service's `internal/` — cross-service only via gRPC (ADR-008).
