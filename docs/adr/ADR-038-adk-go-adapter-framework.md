<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-038 — Google ADK Go as a Go-native AI-framework adapter

| Field | Value |
|-------|-------|
| **Status** | Accepted |
| **Date** | 2026-06-21 |
| **Deciders** | Oscar Gómez Manresa |
| **Scope** | `agents/adapters/adk/` — a new capability adapter embedding the Google Agent Development Kit (Go); refines ADR-035 |
| **Related** | ADR-013 (adapter-first, no mandatory SDK), ADR-035 (adapter language boundary), ADR-009 (Go for platform, Python for agents), ADR-010 (pluggable runtime), ADR-001 (gRPC inter-service contract), ADR-032 (git MCP shim) |

---

## Context

ADR-035 drew the adapter language line by **what the adapter wraps**:

- **Go** for *stateless provider/proxy* adapters (`http`, `git`, `ci`, `llm`).
- **Python** for *AI-framework* adapters — "adapters that embed or wrap a
  Python-only library (LangGraph, LangChain, AutoGen, CrewAI) **where no Go
  equivalent exists**" (`langgraph`).

That last clause assumed AI-agent frameworks are Python-only. That assumption no
longer holds. **Google's Agent Development Kit ships a native Go SDK**
(`google.golang.org/adk`, `go 1.25`) with a full agent-reasoning loop:
`llmagent.New(cfg)` (instruction + tools + sub-agents), a `runner.Runner`, an
in-memory `session.Service`, deterministic workflow agents
(`agent/workflowagents`), `remoteagent` for A2A delegation, and A2A/REST servers
(`server/adka2a`, `server/adkrest`). It is precisely the "Go equivalent" ADR-035
presumed did not exist — an **AI-framework that is natively Go**.

This matters because Zynax's whole adapter thesis (ADR-013) is that *any* system
serving the `AgentService` gRPC contract becomes a first-class capability. ADK Go
is a high-value thing to wrap: it brings tool-use, multi-step reasoning,
sub-agent delegation, and A2A — none of which the single-shot `llm-adapter`
offers — and it is the same runtime the competing kagent project builds on, so
supporting it is also competitive parity.

Two facts were verified against the ADK Go source before this ADR (not from docs):

1. **The execution seam is near 1:1.**
   `Runner.Run(ctx, userID, sessionID, *genai.Content, RunConfig) → iter.Seq2[*session.Event, error]`
   maps directly onto Zynax's `AgentService.ExecuteCapability → stream TaskEvent`:
   each ADK `session.Event` (it embeds `model.LLMResponse`) becomes a Zynax
   `PROGRESS` event; the final non-`Partial` event becomes the terminal
   `COMPLETED`. No engine, compiler, or workflow change is required.

2. **ADK Go ships no Ollama/OpenAI model provider** — only `gemini` and `apigee`,
   both built on `google.golang.org/genai` (Gemini wire format). Ollama does not
   speak that format and there is no base-URL shortcut. The `model.LLM` interface
   is, however, tiny:
   `Name() string` + `GenerateContent(ctx, *LLMRequest, stream bool) iter.Seq2[*LLMResponse, error]`.

Zynax's hero onboarding (EPIC #1370) is a **zero-secret** local-Ollama quickstart.
A demo that needed a cloud `GOOGLE_API_KEY` would contradict that positioning.

## Decision

1. **Adopt Google ADK Go as a supported adapter framework.** Add
   `agents/adapters/adk/`, a Go adapter implementing the `AgentService` gRPC
   contract, mirroring the existing Go adapters (`git`/`http`/`ci`/`llm`):
   `cmd/adk-adapter/main.go` (wiring + registry registration), `internal/config`,
   `internal/adapter/server.go` (the bridge), `internal/adk` (builds the
   `llmagent`), shipped as a single distroless binary.

2. **Refine ADR-035: decouple the language axis from the "what it wraps" axis.**
   Adapter language is now chosen by *whether a native-Go binding exists*, not by
   whether the adapter embeds a framework:
   - **Python** only when the framework is **Python-only** (LangGraph → `langgraph`).
   - **Go** when the adapter is a proxy **or** embeds a framework with a native-Go
     SDK (ADK → `adk`).
   ADK is the first **Go AI-framework adapter**: it embeds an agent-reasoning
   framework (so it is *not* a proxy) yet needs no Python.

3. **Ship a custom `model.LLM` over Ollama inside the adapter** so ADK agents run
   secret-free under the existing Ollama compose overlay. It translates
   `[]*genai.Content` ↔ Ollama `/api/chat` messages and adapts responses back into
   `*model.LLMResponse`. ADK's native `gemini` provider remains selectable by env
   for users who supply a key — Ollama is the default, keeping the quickstart
   secret-free.

4. **Reasoning lives in the adapter; the manifest stays a thin capability surface.**
   The ADK agent's `Instruction`, `Tools`, and sub-agents are wired in
   `internal/adk`; the `AgentDef` manifest declares only the capability name,
   JSON input/output schema, and runtime image. The wire contract is the unchanged
   `AgentService` proto (ADR-001/ADR-013) — the adapter's use of ADK is invisible
   to the platform and to the language-agnostic BDD `.feature`.

## Consequences

**Positive**

- ADK's tool-use, multi-step reasoning, sub-agent delegation, and A2A reach become
  dispatchable Zynax capabilities, referenced by name from any workflow state —
  with **zero** engine/compiler/workflow change (the AgentService seam pays off).
- Go-native: distroless static binary, no Python dependency tax, same lint/test/
  build path as the other four Go adapters (ADR-035's uniformity benefit extends).
- The custom Ollama `model.LLM` keeps `make demo` secret-free, preserving the
  EPIC #1370 quickstart while adding ADK's capabilities.
- Competitive parity: Zynax can host the same ADK-authored agents kagent runs,
  but behind its declarative workflow engine.

**Negative / accepted**

- The custom `model.LLM` is maintenance we own: the `genai.Content` ↔ Ollama
  translation, including tool-call round-trips, is non-trivial and must track ADK's
  `model` package. Bounded by an adapter-level `.feature` parity oracle (ADR-016).
- Small local models are weak at tool-calling, so the **first** demo agent stays
  simple (single-tool or single-shot); richer tool-using demos may need a stronger
  model (cloud, with a key) and are explicitly out of the zero-secret default.
- ADK Go is young: pin the `google.golang.org/adk` version and re-verify the
  `Runner`/`Session`/`model.LLM` signatures on each bump. Its `go 1.25` floor sets
  the adapter module's toolchain.
- Supply chain grows by `google.golang.org/adk` + `google.golang.org/genai`
  (govulncheck-gated via `make security`).

## Alternatives considered

- **Wrap ADK through the Python adapter path (adk-python).** Rejected: ADK has a
  first-class Go SDK; routing through Python would re-incur exactly the dependency/
  CVE tax ADR-035 removed, for no capability gain.
- **Reuse the existing `llm-adapter` instead of adding ADK.** Rejected: the
  `llm-adapter` is a single-shot prompt→completion proxy with no agent loop, tools,
  or sub-agents — it cannot express what ADK exists to provide.
- **Gemini-only demo (skip the custom Ollama model).** Rejected: requires a cloud
  `GOOGLE_API_KEY`, breaking the zero-secret #1370 quickstart and the local-first
  positioning.
- **An `AdkEngine` implementing `WorkflowEngine` (ADK workflow agents interpret
  WorkflowIR).** Deferred, not rejected: a much larger, orthogonal effort behind
  ADR-015's engine abstraction; recorded for a future milestone, not this adapter.
