# agents/adapters/llm — LLM Provider Capability Adapter

Python adapter service implementing `AgentService` for LLM inference via OpenAI, AWS Bedrock, and Ollama.

## Module

`llm-adapter` · `src/llm_adapter/`

## Capabilities

| Name | Description |
|------|-------------|
| `chat_completion` | Stream a multi-turn chat completion. Provider is selected by `LLM_PROVIDER`. Returns streamed `PROGRESS` events during generation and a final `COMPLETED` event with the full response text. |

## Supported Providers

| `LLM_PROVIDER` value | SDK | Required env vars |
|----------------------|-----|-------------------|
| `openai` | `openai` | `OPENAI_API_KEY` |
| `bedrock` | `aiobotocore` | `AWS_REGION` (+ ambient AWS credentials) |
| `ollama` | `httpx` (REST) | `OLLAMA_BASE_URL` |

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LLM_PROVIDER` | ✓ | — | Active provider: `openai`, `bedrock`, or `ollama`. |
| `AGENT_ID` | ✓ | — | Unique agent identifier registered with agent-registry. |
| `ADAPTER_ENDPOINT` | ✓ | — | gRPC address the task-broker dials, e.g. `llm-adapter:50057`. |
| `REGISTRY_ADDR` | ✓ | — | agent-registry gRPC address, e.g. `agent-registry:50052`. |
| `OPENAI_API_KEY` | ✓ if openai | — | OpenAI API key (never log or echo). |
| `OLLAMA_BASE_URL` | ✓ if ollama | — | Ollama REST base URL, e.g. `http://ollama:11434`. |
| `AWS_REGION` | ✓ if bedrock | — | AWS region for the Bedrock endpoint. |
| `LLM_MODEL` | — | `gpt-4o` / `anthropic.claude-3-5-sonnet-20241022-v2:0` / `llama3.2` | Model name; provider-specific default applies when unset. |
| `LLM_MAX_TOKENS` | — | `4096` | Maximum token ceiling enforced before calling the provider. |
| `ZYNAX_LLM_ADAPTER_GRPC_PORT` | — | `50057` | TCP port the adapter's gRPC server binds to. |

API keys are stored as `pydantic.SecretStr` — they never appear in repr, logs, or error messages.

## gRPC Port

Default: **50057** (override via `ZYNAX_LLM_ADAPTER_GRPC_PORT`).

## Docker Compose (local dev — Ollama)

Add to your `.env` (values never in `docker-compose.yml`):
```
LLM_PROVIDER=ollama
OLLAMA_BASE_URL=http://host.docker.internal:11434
LLM_MODEL=llama3.2
```

Ollama must be running on the Docker host. With Docker Desktop, `host.docker.internal` resolves to the host machine automatically. On Linux, add `--add-host=host.docker.internal:host-gateway` or use your host IP.

## Testing

```bash
cd agents/adapters/llm
uv run pytest tests/ -v
uv run pytest tests/ --cov=src --cov-fail-under=80 -v
```

## Reference

Canvas: `docs/spdd/383-llm-adapter/canvas.md`
