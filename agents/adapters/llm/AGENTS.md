# agents/adapters/llm — LLM Provider Capability Adapter

Go adapter service implementing `AgentService` for LLM inference via OpenAI, AWS Bedrock, and Ollama.

Ported from Python to Go under ADR-035 (M7 EPIC P / #1276): the adapter is a stateless
provider proxy with no Python-specific dependency, so it ships as a single static distroless
binary — dropping the `openai` / `aiobotocore` / `aiohttp` supply-chain tree.

## Module

`github.com/zynax-io/zynax/agents/adapters/llm` · `cmd/llm-adapter/` + `internal/`

## Capabilities

| Name | Description |
|------|-------------|
| `chat_completion` | Stream a chat completion from the configured provider. Returns streamed `PROGRESS` events during generation and a final `COMPLETED` event with the full response text. |

## Supported Providers

| `provider.name` | Backend | Required config |
|-----------------|---------|-----------------|
| `openai` | OpenAI HTTP API | `provider.api_key_env` → env var holding the API key |
| `bedrock` | AWS Bedrock runtime | `provider.region` (+ ambient AWS credentials) |
| `ollama` | Ollama REST API | `provider.ollama_base_url` |

## Configuration

Provider, model, and limits are declared in a YAML config file named by the
`ZYNAX_LLM_CONFIG` env var (default: the baked example at `/etc/llm-adapter/config.yaml`).
The provider and model are **always** declared in config — never in `input_payload`.

See `agent-def.yaml.example` for the full schema. Key fields:

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `provider.name` | ✓ | — | Active provider: `openai`, `bedrock`, or `ollama`. |
| `provider.model` | ✓ | — | Model name (provider-specific). |
| `provider.api_key_env` | ✓ if openai | — | Name of the env var holding the API key (never logged or echoed). |
| `provider.ollama_base_url` | ✓ if ollama | — | Ollama REST base URL, e.g. `http://ollama:11434`. |
| `provider.region` | ✓ if bedrock | — | AWS region for the Bedrock endpoint. |
| `provider.max_tokens` | — | `4096` | Maximum token ceiling enforced before calling the provider. |
| `endpoint` | ✓ | `:50070` | gRPC address the adapter's server binds to. |
| `registry_endpoint` | ✓ | — | agent-registry gRPC address, e.g. `agent-registry:50052`. |

The API-key value is resolved at startup from the named env var and is never stored in
config, repr, logs, or error messages.

## gRPC Port

Default: **50070** (override via the `endpoint` field in the config file).

## Testing

```bash
cd agents/adapters/llm
GOWORK=off go test ./... -race -timeout 60s
```

## Reference

ADR-035 (adapter language boundary) · Canvas: `docs/spdd/383-llm-adapter/canvas.md`
