# agents/adapters/adk — ADK Go Adapter

Go adapter service implementing `AgentService` by embedding the **Google Agent
Development Kit (Go)** (`google.golang.org/adk`). It runs an ADK agent
(`llmagent` + tools + optional sub-agents) per declared capability and bridges
the ADK `Runner` event loop onto the Zynax `TaskEvent` stream.

> First **Go-native AI-framework adapter** (ADR-038, refines ADR-035). Unlike the
> single-shot `llm-adapter`, an ADK-backed capability can use tools, multiple
> reasoning steps, and sub-agent delegation.

## Module

`github.com/zynax-io/zynax/agents/adapters/adk`

## Capabilities

Capabilities are **declared in config** — each maps to one ADK `llmagent`
(instruction + tools). The shipped `agent-def.yaml.example` declares a starter
capability; add more by adding `capabilities[]` entries, no code change.

| Field per capability | Role |
|----------------------|------|
| `name` | Capability name routed on by the task broker (e.g. `triage`). |
| `instruction` | The ADK agent's system instruction (`llmagent.Config.Instruction`). |
| `input_schema_json` / `output_schema_json` | JSON Schema enforced at the adapter boundary (ADK itself is untyped text/tool-args). |
| `timeout_seconds` | Wall-clock budget; exceeded → `FAILED` code `TIMEOUT`. |

## Model backends (ADR-038 §3)

| Backend | When | Secret? |
|---------|------|---------|
| **Ollama** (default, **only value wired in S3 #1479**) | `model.provider: ollama` — a custom `model.LLM` (`internal/model`) over Ollama `/api/chat` | **None** — keeps `make demo` secret-free |
| Gemini (native ADK) | `model.provider: gemini` — ADK's `model/gemini`; **designed (ADR-038 §3) but not yet wired** — config load rejects it until a later story | `GOOGLE_API_KEY` (when wired) |

The model name and host always come from config/env — never hardcoded (12-factor).

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `ADAPTER_CONFIG` | ✓ | Path to the YAML config (see `agent-def.yaml.example`). |
| `OLLAMA_HOST` | when `provider: ollama` | Ollama base URL (default `http://localhost:11434`). |
| `GOOGLE_API_KEY` | when `provider: gemini` | Google API key for the native Gemini provider. |

## Configuration

YAML at the path in `ADAPTER_CONFIG`. Key fields:

| Field | Description |
|-------|-------------|
| `agent_id` | Unique id registered with agent-registry. |
| `endpoint` | gRPC **bind** address (default `:50080`). A hostless value binds all interfaces but is not routable. |
| `advertise_endpoint` | **Routable** address the task-broker dials (e.g. `adk-adapter:50080`). Required when `endpoint` is hostless (issue #1371); else falls back to `endpoint`. |
| `registry_endpoint` | agent-registry address (e.g. `agent-registry:50052`). |
| `model.provider` | `ollama` (default; only value wired). |
| `model.name` | Model id (e.g. `qwen2.5-coder:0.5b`). |
| `capabilities[]` | `name`, `instruction`, `input_schema_json`, `output_schema_json`, `timeout_seconds`. |

Runnable demo (secret-free, local Ollama): `spec/workflows/examples/adk-code-review-ollama.yaml`
dispatches the `review` capability declared in `agent-def.yaml.example`. Bring up the
Ollama overlay (`docker-compose.ollama.yml`), then `zynax apply … && zynax result <run>`.

## The bridge (`internal/adapter`)

`ExecuteCapability` builds a `genai.Content` from the validated `input_payload`,
gets/creates an ADK `session` keyed by `workflow_id`, runs `Runner.Run`, and maps
each `*session.Event` → `PROGRESS`; the final non-`Partial` event → `COMPLETED`
(coerced to `output_schema`); any error → a classified `CapabilityError`. The
`AgentService` proto contract is unchanged — ADK use is an internal detail.

## gRPC Port

Default: **50080** (set via `endpoint` in config YAML).

## Testing

```bash
GOWORK=off go test ./... -race -timeout 60s   # ADR-017: GOWORK=off required
```

BDD: `protos/tests/features/adk_adapter.feature` (committed before the bridge — ADR-016).

## Reference

Canvas: `docs/spdd/1476-adk-go-adapter/canvas.md` · Decision: `docs/adr/ADR-038-adk-go-adapter-framework.md` · Operator example: `agent-def.yaml.example`
