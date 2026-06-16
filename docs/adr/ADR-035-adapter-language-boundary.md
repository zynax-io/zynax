<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-035 — Adapter language boundary: Go for provider/proxy adapters, Python for AI-framework adapters

| Field | Value |
|-------|-------|
| **Status** | Accepted |
| **Date** | 2026-06-16 |
| **Deciders** | Oscar Gómez Manresa |
| **Scope** | `agents/adapters/*` — language selection for capability adapters; refines (does not reverse) ADR-009 — M7 EPIC P (#1276) |
| **Related** | ADR-009 (Go for platform, Python for agents), ADR-013 (adapter-first, no mandatory SDK), ADR-002 (Python 3.12 runtime), ADR-003 (uv package manager) |

---

## Context

ADR-009 set the language strategy at the *layer* granularity: **Go for the platform
services, Python for agents** — justified by Python's AI/ML ecosystem (LangChain,
LangGraph, AutoGen, CrewAI). That rule is correct for AI *agents*, but capability
**adapters** are a different animal, and the repository has already drifted to a finer
rule that ADR-009 never wrote down:

| Adapter | Language (today) | What it actually is |
|---------|------------------|---------------------|
| `http`  | **Go**    | stateless proxy over an arbitrary REST API |
| `git`   | **Go**    | stateless proxy over GitHub/GitLab APIs |
| `ci`    | **Go**    | stateless proxy over CI systems |
| `llm`   | **Python** | stateless proxy over OpenAI / Bedrock / Ollama HTTP APIs |
| `langgraph` | **Python** | mounts a Python-only `StateGraph` as a capability |

Three of five adapters are already Go. The split is not "platform vs agents" — it is
**stateless provider proxy vs AI-framework integration**. The `llm-adapter` is on the
wrong side of that line: it holds no framework state, runs no Python-only library, and
performs the same send-prompt / stream-tokens / return-response pattern as the
http-adapter. Yet by living in Python it drags in the `openai` / `aiobotocore` /
`aiohttp` transitive tree, which has repeatedly required Dependabot-driven security
floors (the adapter's `pyproject.toml` pins `aiohttp>=3.14.1` "fixes the GHSA set").

Maintaining a Python toolchain for an adapter that has no Python-specific reason to
exist multiplies the dependency surface, the CVE-patch cadence, and the blast radius
for no capability gain.

## Decision

Choose adapter implementation language by **what the adapter wraps**, not by which
layer it sits in:

1. **Go** is the default for **stateless provider/proxy adapters** — adapters whose job
   is to translate a Zynax capability into calls against an external HTTP/gRPC/SDK API
   and stream results back. This covers `http`, `git`, `ci`, **and `llm`**.

2. **Python** is reserved for **AI-framework adapters** — adapters that embed or wrap a
   Python-only library (LangGraph, LangChain, AutoGen, CrewAI) where no Go equivalent
   exists. This covers `langgraph` and any future framework adapter.

3. The **Python SDK** (`agents/sdk/`) and **`agents/examples/`** remain Python, unchanged
   — agent *authoring* keeps Python's AI ecosystem (ADR-009 §Rationale stands for agents).

The boundary between Go and Python is unchanged at the wire: the gRPC `AgentService`
proto contract (ADR-001/ADR-013). An adapter's language is an internal implementation
detail invisible to the platform and to the language-agnostic BDD `.feature` contract.

## Consequences

**Positive**

- `llm-adapter` ports to Go (M7 EPIC P / #1276): one fewer Python deployable, the
  `openai` / `aiobotocore` / `aiohttp` tree leaves the supply chain, and the adapter
  ships as a single static distroless binary — smaller image, smaller attack surface.
- The Go↔Python split is now stated by an explicit, testable rule rather than tribal
  knowledge, so the next adapter lands in the right language by default.
- Uniform tooling for 4 of 5 adapters (one lint/test/build path) lowers maintenance and
  contributor context-switching.

**Negative / accepted**

- `langgraph-adapter` **cannot** be ported to Go — LangGraph is Python-only. This ADR
  makes that explicit: de-Pythonization stops at the framework boundary; it is not a path
  to a Python-free repo. Python stays in CI for `langgraph`, the SDK, and examples.
- The port is behavioural-equivalence work, not new capability. It is justified by
  supply-chain/maintenance reduction, and is gated on the existing
  `protos/tests/features/llm_adapter.feature` staying green (the parity oracle) — so the
  contract risk is bounded.
- ADR-009's one-line "Python for agents" summary now needs reading together with this
  finer adapter rule; ADR-009 is **refined, not reversed**, and its agent rationale is
  untouched.

## Alternatives considered

- **Keep `llm-adapter` in Python (status quo).** Rejected: pays the Python dependency/CVE
  tax for an adapter with no Python-specific need, against the grain of the three Go
  adapters already shipped.
- **Port everything including `langgraph` to Go.** Rejected: impossible without deleting
  LangGraph support, which is a deliberate capability (ADR-013) and a competitive
  differentiator.
- **Remove all Python (SDK + examples + adapters).** Rejected: reverses the agent-authoring
  thesis of ADR-009/010/013/033 — a strategic product pivot, not a maintenance change.
