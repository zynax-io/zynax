<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax — Competitive Positioning

**Document type:** Competitive Analysis · Strategic Positioning
**Date:** 2026-05-28 · **Author:** Engineering
**Status:** Current — supersedes §3 of `2026-04-30-competitive-analysis.md`
**Related issues:** [#575](https://github.com/zynax-io/zynax/issues/575) (G19 strategic risk R6)

---

## Value Proposition

Zynax is the **declarative control plane for multi-agent AI workflows** — engine-agnostic,
adapter-first, and GitOps-native. Where Kagent runs Kubernetes-native agents and LangGraph
builds Python state machines, Zynax compiles YAML workflow manifests into an engine-neutral
IR that can execute on Temporal, LangGraph, or any future engine without rewriting the
workflow. A single `zynax apply` submits the same YAML regardless of which engine is active.

---

## Comparison Table (as of M5 / v0.4.0)

| Dimension | Zynax | Kagent | Dapr Workflows | Temporal | LangGraph |
|-----------|-------|--------|----------------|----------|-----------|
| **Deployment** | Docker Compose (M5) · K8s Helm (M6) | K8s-only (Kind required) | K8s-native Dapr runtime | Any | Library (no infra) |
| **Workflow authoring** | Declarative YAML (`kind: Workflow`) | Kubernetes CRDs (kubectl-native) | YAML or code (Dapr SDK) | Go / Java / Python code | Python code |
| **Engine portability** | ✅ Temporal + LangGraph adapters; Argo planned | ❌ ADK lock-in | ❌ Dapr runtime lock-in | N/A (IS the engine) | N/A (IS the framework) |
| **Capability registry** | ✅ agent-registry gRPC service | ✅ Kubernetes CRD-based | ❌ Not built-in | ❌ Not built-in | ❌ Not built-in |
| **LLM-native** | Via llm-adapter (`chat_completion`) | ✅ Built-in ModelConfig | ❌ Not built-in | ❌ Not built-in | ✅ Built-in |
| **Multi-language agents** | ✅ Go adapters + Python adapters, same control plane | ✅ Any container | ✅ Any language | ✅ Any language | Python-first |
| **Web UI** | ❌ None (M8 roadmap) | ✅ Included | ❌ None | ✅ Temporal UI | ❌ None |
| **MCP tools** | Via adapters (git/ci/llm/langgraph shipped) | ✅ kubectl, Helm, Argo, Prometheus | ❌ | ❌ | Via extensions |
| **GitOps-native** | ✅ Workflow YAML in git, `zynax apply` | ❌ Kubectl imperative | ❌ | ❌ | ❌ |
| **CNCF status** | Not yet (M8 target) | ✅ Sandbox 2026 | ✅ Incubating | Not CNCF | Not CNCF |
| **Production-proven** | No (v0.4.0 · MVP) | Partial | Yes | Yes | Yes |

---

## When to Choose Zynax

- **You need engine-agnostic portability** — you cannot be locked into a single execution engine
  (Temporal today, Argo tomorrow) and want to swap without rewriting workflows.
- **Your workflows are event-driven state machines** that benefit from compile-time structural
  validation before execution.
- **You want GitOps-native YAML workflows** reviewed in PRs like any other source code.
- **You have agents in multiple languages** (Go HTTP adapters, Python LLM/LangGraph adapters)
  that should share a single control plane.
- **You prefer no-SDK agent registration** — any gRPC service implementing `AgentService` is a
  capability without framework lock-in.

---

## When NOT to Choose Zynax (yet — be honest)

- **You need K8s-native agent management with kubectl/Helm/Argo integration** → use **Kagent**.
  Kagent is purpose-built for this and is a CNCF Sandbox project with a growing ecosystem.
- **You need multi-LLM provider support with a web UI out of the box** → use **Kagent** or
  wait for Zynax M8.
- **You need a mature, production-proven, CNCF-backed orchestrator** → use **Temporal** (durable
  workflows) or **Dapr Workflows** (K8s-first).
- **You are building a pure Python LangGraph application with no cross-language concerns** → stay
  with **LangGraph** directly; the Zynax LangGraph adapter adds overhead you don't need.
- **You need this in production today** → Zynax is at v0.4.0 / MVP stage; persistence, mTLS, rate
  limiting, and SBOM are M6 targets.

---

## How Zynax Complements Kagent / MCP

These are complementary, not competitive:

- **Kagent handles K8s operations** (kubectl, Helm, Argo, Prometheus via MCP tools); **Zynax
  orchestrates the workflow** that decides when and in what order to invoke those tools.
- A Kagent agent can register as a Zynax capability via the `AgentService` gRPC contract.
  Kagent's MCP tools become Zynax capabilities without modification.
- Zynax adds the **control plane layer** that Kagent lacks: declarative YAML workflows,
  compile-time IR validation, multi-engine dispatch, and cross-language adapter support.

In a combined deployment: Zynax dispatches to a Kagent-managed K8s agent for cluster ops while
simultaneously dispatching to a Python llm-adapter for LLM reasoning — all expressed as a single
`kind: Workflow` YAML file.

---

## See Also

- `docs/architecture/2026-04-30-competitive-analysis.md` — original 2026-04-23 analysis
  (Temporal, Argo, Kestra, Serverless Workflow positioning; still valid)
- `ARCHITECTURE.md` — current implementation status per milestone
- `ROADMAP.md` — when production features (mTLS, K8s, SBOM) land
