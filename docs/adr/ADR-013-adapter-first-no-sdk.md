# ADR-013: Adapter-First Integration — No SDK Required

**Status:** Accepted  **Date:** 2025-04-01

## Decision
Any system can become an Zynax capability by implementing the
`AgentService` gRPC contract. No SDK is required.

## Rationale
SDK-required architectures create friction:
- Language-limited (only SDK languages)
- Framework-coupled (SDK upgrades propagate)
- High-friction for non-engineering teams

Adapter-first means:
- Any language that supports gRPC = instant capability provider
- Existing HTTP APIs wrapped in minutes (http-adapter)
- LLM providers, CI systems, Git platforms become capabilities

## SDK Position
The SDK (`agents/sdk/`) is OPTIONAL. It provides convenience for Python agents
that want the full AgentRuntime Protocol + LangGraph/AutoGen integration.
It is never required. Zero features require SDK adoption.

## Adapters Provided
- `http-adapter` (Go) — wrap any REST API
- `llm-adapter` (Go) — Bedrock, Ollama, OpenAI (ported from Python, ADR-035)
- `git-adapter` (Go) — GitHub/GitLab capabilities + webhook integration
- `langgraph-adapter` (Python) — LangGraph app as capability
