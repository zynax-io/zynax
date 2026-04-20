# agents/adapters/ — AGENTS.md

> **Adapter-First Integration. No SDK Required.**
> See `ARCHITECTURE.md §7` and `docs/adr/ADR-013-adapter-first.md`.
>
> This directory contains execution adapters — thin wrappers that expose
> external systems as Keel capabilities WITHOUT requiring the SDK.

---

## Core Principle

> Any system becomes a capability by implementing the `AgentService` gRPC contract.

That is the ONLY requirement. No language. No framework. No SDK import.

```
External System      Adapter             Keel
─────────────────    ──────────────      ─────────────────
Bedrock API     →    llm/              → capability: summarize
GitHub API      →    git/              → capability: open_mr, request_review
Jenkins/CI      →    ci/               → capability: run_tests, deploy
HTTP REST API   →    http/             → capability: call_payments_api
LangGraph app   →    langgraph/        → capability: research_topic
Ollama          →    llm/              → capability: generate_code
```

---

## The Two Interfaces of Every Adapter

An adapter sits between two worlds. These two sides are completely independent
of each other, and understanding them separately is the key to adapter design.

### The Keel-Facing Side

Every adapter, in every language, implements the `AgentService` gRPC contract
defined in `protos/keel/v1/`. This side is always identical regardless of what
the adapter wraps or what language it is written in. The task-broker, which
dispatches tasks to adapters, sees exactly the same contract whether it is
talking to a Python HTTP adapter, a Go database adapter, or a Java enterprise
system adapter. The Keel platform is entirely unaware of what is on the other
side of this interface.

Adapters implement this side using **raw generated proto stubs** — not the
Python SDK. Rule 6 in this file states this explicitly: "Never import from
agents/sdk." The stubs for Go, Python, and any other language are generated
from the same proto source in `protos/keel/v1/`. See `protos/AGENTS.md §8`
for how to obtain stubs in any language.

### The External-Facing Side

This side speaks whatever protocol the wrapped system requires. It could be:
- HTTP/REST (the `http-adapter` uses `httpx`)
- A Python AI framework library (the `langgraph-adapter` imports LangGraph directly)
- A cloud provider SDK (the `llm-adapter` imports `boto3` for Bedrock)
- A Git hosting API (the `git-adapter` calls the GitHub REST API)
- A database driver, a message queue client, a gRPC client to another service
- Any other protocol the external system speaks

The language of the adapter is determined entirely by this side. The right
language for an adapter is the language that has the best client library or
most natural integration with the external system being wrapped.

---

## Adapter Language — Any Language Is Valid

### Why the Built-in Adapters Are Python

The built-in adapters in this directory are Python because their target
ecosystems have rich, mature Python client libraries:

| Adapter | External system | Why Python |
|---------|----------------|------------|
| `http/` | Any REST API | `httpx` async HTTP client |
| `llm/` | Bedrock, Ollama, OpenAI | `boto3`, `openai`, `ollama` Python SDKs |
| `git/` | GitHub, GitLab | `PyGitHub`, webhook handling libraries |
| `langgraph/` | LangGraph apps | LangGraph is a Python library |

This is an ecosystem decision, not an architectural requirement. These adapters
are Python because wrapping Python libraries from Python is natural. If LangGraph
released a Go client, a Go LangGraph adapter would be equally valid.

### Custom Adapters in Any Language

When you build an adapter to wrap your own system, the right language is the
one that fits the external system:

| External system | Natural adapter language |
|----------------|--------------------------|
| Java enterprise service (JAX-RS, Spring) | Java — use the system's own client classes |
| Go microservice (existing gRPC service) | Go — generated stubs are the natural client |
| Rust inference engine | Rust — call the engine natively, expose gRPC |
| TypeScript Node.js API | TypeScript — use the existing JS ecosystem |
| .NET / C# service | C# — use the .NET gRPC server libraries |
| Any language with gRPC support | That language |

For any language, the workflow is:
1. Generate `AgentService` stubs from `protos/keel/v1/` using `buf generate`
   with your language's gRPC plugin. See `protos/AGENTS.md §8`.
2. Implement `ExecuteCapability` and `GetCapabilities` against those stubs.
3. Start a gRPC server on the adapter's port.
4. Register capabilities via `AgentDef` YAML (the same format used by Python
   adapters — the YAML is language-agnostic by definition).
5. Deploy as a container alongside the wrapped system.

The `AgentDef` YAML, the capability names, and the event streaming format are
identical to Python adapters. The platform cannot distinguish a Go adapter from
a Python adapter at runtime.

### The Relationship Between Adapters and the SDK

Adapters and SDK agents both implement `AgentService`, but they are different
things:

| | Adapter | SDK Agent |
|---|---------|-----------|
| **Purpose** | Wrap an existing external system | Build new Keel-native intelligence |
| **SDK dependency** | None — raw stubs only | `keel-sdk` package |
| **AgentContext injection** | Not applicable — no platform context needed | Core feature — SDK injects context |
| **Platform service access** | Minimal — translate and forward | Full — memory, registry, broker via context |
| **Lifecycle management** | Self-managed or minimal | SDK manages registration, heartbeat, shutdown |
| **Language** | Any language that speaks to the external system | Python (SDK is Python-only) |
| **Preferred for** | Integrating existing systems, any language | New Python agent logic with AI frameworks |

The task-broker treats them identically. Both register capabilities. Both
receive `ExecuteCapabilityRequest`. Both return a stream of `CapabilityEvent`.
The internal architecture is their own concern.

---

## Adapter Layout

```
agents/adapters/
├── AGENTS.md                      ← This file
├── http/                          ← Wrap any HTTP REST API as a capability
│   ├── AGENTS.md
│   ├── pyproject.toml
│   └── src/keel_http_adapter/
│       ├── adapter.py             ← gRPC AgentService implementation
│       ├── config.py              ← env vars: target URL, auth, capability mapping
│       └── main.py
├── llm/                           ← Wrap LLM providers (Bedrock, Ollama, OpenAI)
│   ├── AGENTS.md
│   ├── pyproject.toml
│   └── src/keel_llm_adapter/
│       ├── adapter.py
│       ├── providers/             ← bedrock.py, ollama.py, openai.py
│       └── config.py
├── git/                           ← GitHub/GitLab events + operations
│   ├── AGENTS.md
│   ├── pyproject.toml
│   └── src/keel_git_adapter/
│       ├── adapter.py             ← Handles: open_mr, request_review, merge_pr
│       ├── webhook.py             ← Receives GitHub webhooks → emits to event-bus
│       └── config.py
└── langgraph/                     ← Wrap a LangGraph app as a capability
    ├── AGENTS.md
    ├── pyproject.toml
    └── src/keel_langgraph_adapter/
        ├── adapter.py
        └── config.py
```

---

## The AgentService gRPC Contract (What Adapters Implement)

Every adapter implements exactly these two RPCs. Nothing else.

```protobuf
service AgentService {
    // Execute a capability and stream events back
    rpc ExecuteCapability(ExecuteCapabilityRequest)
        returns (stream CapabilityEvent);

    // Declare what capabilities this adapter provides
    rpc GetCapabilities(GetCapabilitiesRequest)
        returns (GetCapabilitiesResponse);
}

message ExecuteCapabilityRequest {
    string request_id  = 1;  // idempotency key
    string capability  = 2;  // e.g. "summarize"
    string task_id     = 3;
    bytes  input_json  = 4;  // JSON-encoded input (matches capability input_schema)
}

message CapabilityEvent {
    enum Type {
        PROGRESS = 0;   // intermediate update
        RESULT   = 1;   // final result (terminal)
        ERROR    = 2;   // failure (terminal)
    }
    Type  type      = 1;
    bytes payload   = 2;  // JSON
    string message  = 3;
}
```

---

## HTTP Adapter Pattern

```python
# agents/adapters/http/src/keel_http_adapter/adapter.py

"""HTTP Adapter — wraps any REST API as an Keel capability.

Configuration via env vars (no config files):
  KEEL_HTTP_TARGET_URL=https://api.example.com
  KEEL_HTTP_CAPABILITIES={"call_payments": {"path": "/payments", "method": "POST"}}
  KEEL_HTTP_AUTH_HEADER=Authorization
  KEEL_HTTP_AUTH_VALUE=Bearer ${SECRET_TOKEN}
"""

import json
import grpc
import httpx
import structlog
from keel.v1 import agent_pb2, agent_pb2_grpc
from keel_http_adapter.config import settings

logger = structlog.get_logger(__name__)

class HTTPAdapter(agent_pb2_grpc.AgentServiceServicer):
    """Wraps a REST API as an Keel capability.

    No business logic here — pure protocol translation:
    gRPC ExecuteCapability → HTTP call → stream CapabilityEvent responses.
    """

    def __init__(self) -> None:
        self._client = httpx.AsyncClient(
            base_url=settings.target_url,
            headers={settings.auth_header: settings.auth_value.get_secret_value()},
            timeout=30.0,
        )
        self._capability_map: dict[str, dict] = json.loads(settings.capabilities_json)

    async def ExecuteCapability(
        self,
        request: agent_pb2.ExecuteCapabilityRequest,
        context: grpc.aio.ServicerContext,
    ):
        cap = self._capability_map.get(request.capability)
        if cap is None:
            await context.abort(grpc.StatusCode.NOT_FOUND,
                                f"capability {request.capability!r} not configured")
            return

        yield agent_pb2.CapabilityEvent(
            type=agent_pb2.CapabilityEvent.PROGRESS,
            message=f"calling {settings.target_url}{cap['path']}",
        )

        try:
            response = await self._client.request(
                method=cap["method"],
                url=cap["path"],
                json=json.loads(request.input_json),
            )
            response.raise_for_status()
        except httpx.HTTPStatusError as exc:
            yield agent_pb2.CapabilityEvent(
                type=agent_pb2.CapabilityEvent.ERROR,
                message=f"HTTP {exc.response.status_code}: {exc.response.text[:200]}",
            )
            return

        yield agent_pb2.CapabilityEvent(
            type=agent_pb2.CapabilityEvent.RESULT,
            payload=response.content,
        )

    async def GetCapabilities(self, request, context):
        caps = [
            agent_pb2.CapabilitySpec(name=name, description=cfg.get("description", ""))
            for name, cfg in self._capability_map.items()
        ]
        return agent_pb2.GetCapabilitiesResponse(capabilities=caps)
```

---

## LLM Adapter Pattern

```python
# agents/adapters/llm/src/keel_llm_adapter/adapter.py

"""LLM Adapter — wraps LLM providers as Keel capabilities.

Supported providers (via KEEL_LLM_PROVIDER env var):
  - bedrock  (Amazon Bedrock)
  - ollama   (local models)
  - openai   (OpenAI API)

Capabilities registered: summarize, generate_code, answer_question, extract_data
"""

class LLMAdapter(agent_pb2_grpc.AgentServiceServicer):

    def __init__(self) -> None:
        self._provider = self._build_provider(settings.provider)

    async def ExecuteCapability(self, request, context):
        input_data = json.loads(request.input_json)

        yield agent_pb2.CapabilityEvent(
            type=agent_pb2.CapabilityEvent.PROGRESS,
            message=f"calling {settings.provider} for {request.capability}",
        )

        prompt = self._build_prompt(request.capability, input_data)

        try:
            result = await self._provider.generate(
                prompt=prompt,
                model=settings.model,
                max_tokens=settings.max_tokens,
            )
        except Exception as exc:
            logger.error("llm_error", capability=request.capability, err=str(exc))
            yield agent_pb2.CapabilityEvent(
                type=agent_pb2.CapabilityEvent.ERROR,
                message=str(exc),
            )
            return

        yield agent_pb2.CapabilityEvent(
            type=agent_pb2.CapabilityEvent.RESULT,
            payload=json.dumps({request.capability: result}).encode(),
        )

    def _build_prompt(self, capability: str, input_data: dict) -> str:
        """Map capability name to a prompt template.
        Templates are loaded from config — never hardcoded here.
        """
        template = settings.prompt_templates.get(capability)
        if template is None:
            raise ValueError(f"No prompt template for capability: {capability!r}")
        return template.format(**input_data)
```

---

## Git Adapter — Event Integration

```python
# agents/adapters/git/src/keel_git_adapter/webhook.py

"""GitHub webhook receiver → Keel event-bus publisher.

GitHub events are translated to Keel workflow events and published
to event-bus. Running workflows that trigger on these events are signaled.

Event mapping:
  github.pull_request.opened    → workflow trigger: start code-review-workflow
  github.pull_request.reviewed  → workflow signal: review.approved / review.changes_requested
  github.push                   → workflow signal: push
"""

class GitHubWebhookHandler:
    def __init__(self, event_bus_client) -> None:
        self._bus = event_bus_client

    async def handle(self, event_type: str, payload: dict) -> None:
        keel_event = self._translate(event_type, payload)
        if keel_event is None:
            return  # not an event Keel cares about

        await self._bus.Publish(PublishRequest(
            topic=keel_event.topic,
            payload=json.dumps(keel_event.payload).encode(),
            correlation_id=payload.get("pull_request", {}).get("node_id", ""),
        ))

    def _translate(self, event_type: str, payload: dict) -> KeelEvent | None:
        MAPPING = {
            "pull_request.opened":                 "github.pull_request.opened",
            "pull_request_review.submitted":       self._map_review_event,
            "push":                                "github.push",
        }
        handler = MAPPING.get(event_type)
        if handler is None: return None
        if callable(handler): return handler(payload)
        return KeelEvent(topic=f"keel.v1.git.{handler}", payload=payload)
```

---

## AgentDef YAML for Adapters (same format as SDK agents)

```yaml
# Deploy the LLM adapter and register its capabilities:
kind: AgentDef
apiVersion: keel.io/v1

metadata:
  name: llm-bedrock-adapter
  namespace: shared

spec:
  endpoint: "llm-bedrock-adapter:50060"
  capabilities:
    - name: summarize
      description: "Summarise documents using Amazon Bedrock Claude."
    - name: generate_code
      description: "Generate code from a specification using Bedrock."
    - name: answer_question
      description: "Answer a factual question using Bedrock."
```

---

## Rules for Adapter Authoring

1. **Zero business logic** — adapters translate protocols, nothing else.
2. **Pure env var config** — `pydantic-settings`, prefix `KEEL_<ADAPTER>_`.
3. **Stream events, don't block** — always yield `PROGRESS` before `RESULT`.
4. **Handle errors gracefully** — yield `ERROR` event, never raise from the gRPC method.
5. **Never import from other adapters** — each adapter is independent.
6. **Never import from agents/sdk** — adapters have zero SDK dependency.
7. **One `AgentDef` YAML per adapter** — capabilities declared in YAML, not code.
8. **Language follows the external system** — choose the language that best fits
   what you are wrapping, not the language of the built-in adapters.

---

## Interoperability Guarantee

The same guarantee that applies to the full platform applies to adapters:

A workflow running on the Temporal engine, compiled by the Go workflow-compiler,
routing through the Go task-broker, can dispatch a capability to a Java adapter
wrapping a Java enterprise service, while another action in the same workflow
dispatches to a Python LangGraph adapter — and neither adapter knows the other
exists. The broker sees only the `AgentService` contract. The workflow YAML sees
only capability names.

The two built-in adapter types that most clearly demonstrate this:

- The `langgraph-adapter` (Python) and a hypothetical Go adapter for an in-house
  Go microservice would register identical `AgentDef` YAML and receive identical
  `ExecuteCapabilityRequest` messages. The platform treats them identically.

- The `http-adapter` is config-only — it can wrap any REST API in any language on
  any platform by changing environment variables, with zero code changes to the
  adapter itself. The wrapped API does not need to know it is connected to Keel.

The adapter pattern is the fullest expression of the architecture's language
neutrality. The proto contract is the only thing that matters. Everything else
— language, framework, runtime, deployment model — is an implementation detail
that Keel is unaware of and indifferent to.
