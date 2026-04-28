# agents/adapters/ — AGENTS.md

> Adapter-First Integration. No SDK Required.
> ADR-013: any system becomes a capability by implementing `AgentService` gRPC.
> Inherits all rules from root `AGENTS.md` and `agents/AGENTS.md`.
> Full implementation patterns: `docs/patterns/python-agent-guide.md`.

---

## Core Principle

> Any system becomes a capability by implementing the `AgentService` gRPC contract.
> No language. No framework. No SDK import.

```
External System      Adapter             Zynax capability
─────────────────    ──────────────      ─────────────────
Bedrock API     →    llm/              → summarize
GitHub API      →    git/              → open_mr, request_review
Jenkins/CI      →    ci/               → run_tests, deploy
HTTP REST API   →    http/             → call_payments_api
LangGraph app   →    langgraph/        → research_topic
```

---

## Adapter Directory

```
agents/adapters/
├── http/          ← Wraps any REST API (config-only, M5)
├── llm/           ← Bedrock, Ollama, OpenAI (M5)
├── git/           ← GitHub/GitLab (M5)
├── ci/            ← Jenkins, GitHub Actions (M5)
└── langgraph/     ← LangGraph app as capability (M5)
```

All adapters are M5+ unless otherwise noted. BDD `.feature` file before implementation.

---

## The Two Adapter Interfaces

**Zynax-facing:** Implement `ExecuteCapability` gRPC (stream of `TaskEvent`).
Register capabilities in an `AgentDef` YAML on startup.

**System-facing:** Whatever protocol the wrapped system speaks (REST, gRPC, CLI).
The adapter translates between the two sides.

---

## Rules

- Adapters are stateless — no adapter-local state that survives restart.
- Register capabilities from `AgentDef` YAML, not hardcoded in Python.
- Capability names must be `snake_case` and match the registry entry.
- Emit at least one `TaskEvent.progress()` for tasks > 2 seconds.
- Always emit exactly one `TaskEvent.result()` or `TaskEvent.error()` as the final event.
