# ADR-009: Language Strategy — Go for Platform, Python for Agents

**Status:** Accepted
**Date:** 2025-04-01

## Decision

- **Go 1.22+** for all five platform services (agent-registry, task-broker, memory-service, event-bus, api-gateway).
- **Python 3.12+** for all AI agents, powered by the Zynax SDK.

## Rationale

| Layer | Language | Justification |
|-------|----------|--------------|
| Platform | Go | High concurrency (goroutines), low memory footprint, static binary, excellent gRPC ecosystem, cloud-native tooling (distroless, K8s probes) |
| Agents | Python | Best AI/ML ecosystem (LangChain, LangGraph, AutoGen, CrewAI, HuggingFace, OpenAI SDK, etc.). No Go equivalent. |

## Contract

The boundary between Go and Python is the gRPC proto contract.
`buf generate` produces both Go and Python stubs from the same `.proto` files.
Neither layer knows the other's implementation language.

## Consequences

+ Platform services benefit from Go's concurrency and performance profile.
+ Agents benefit from Python's AI/ML ecosystem without compromise.
+ Adding a new AI framework (whatever comes after LangGraph) requires only a new `AgentRuntime` adapter in Python — zero Go changes.
- Contributors need familiarity with both languages. Mitigated by clear layer boundaries and per-layer AGENTS.md files.
