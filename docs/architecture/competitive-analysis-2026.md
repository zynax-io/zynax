<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax — Architecture Review & Competitive Analysis
**Document type:** Architecture Review · Strategic Positioning  
**Date:** 2026-04-23 · **Updated:** 2026-04-30 (Kestra AI added) · **Author:** Engineering  
**Status:** Accepted — guides M2 through M8 roadmap decisions  
**Related issues:** #165 (this review), epic #101 (M2), epic #157 (security)

---

## Executive Summary

This document provides the strategic and technical foundation for every major
decision Zynax will make from M2 through its CNCF Sandbox submission (M8).

**Three findings drive everything that follows:**

1. **The market gap Zynax targets is real and unoccupied.** No existing tool
   combines declarative YAML workflows, a formal engine-agnostic intermediate
   representation, a capability/agent registry, and pluggable execution engines.
   The combination is Zynax's sole differentiator and must be protected.

2. **Serverless Workflow is not a competitor — it is a compatibility
   opportunity.** Adopting Serverless Workflow YAML as an optional Layer 1
   syntax in M4 would give Zynax CNCF ecosystem reach at low cost, while
   preserving the architectural identity that makes Zynax unique.

3. **The implementation plan is sound but has one existential gate.** M3
   (Temporal adapter) is the proof-of-concept for the entire value proposition.
   If M3 slips or the adapter produces different behaviour across engines, the
   whole three-layer model is invalidated. M3 must be the engineering team's
   highest concentration point.

---

## 1. The Problem Zynax Solves

### 1.1 The current state of AI workflow orchestration

Organizations building AI systems today face a forced choice:

| Option | Trade-off |
|--------|-----------|
| Use **LangGraph** | Python-only, in-process, no engine portability |
| Use **Temporal** | Durable but no declarative YAML, no AI-native semantics |
| Use **Argo Workflows** | Kubernetes-native batch jobs, not suited for long-running AI interactions |
| Use **Prefect / Flyte** | Strong for data pipelines, no multi-agent coordination |
| Use **Conductor** | LLM tasks emerging but single-runtime, no formal IR |
| Use **Kestra** | AI agents as tasks inside its own engine; no engine portability, no capability registry |
| Use **Serverless Workflow** | Excellent generic DSL but zero AI-native semantics (agents, capabilities, memory) |

Every option requires either accepting engine lock-in or accepting the absence
of AI-native primitives. **No tool does both declarative AI orchestration and
engine portability.**

### 1.2 The Zynax hypothesis

> A control plane layer between workflow intent and execution engines —
> analogous to what Kubernetes did between application intent and container
> runtimes — enables organizations to write AI workflows once and run them on
> any engine, today and in the future.

The three-layer architecture expresses this directly:

```
┌────────────────────────────────────────────────────┐
│  Layer 1 — Intent                                  │
│  YAML manifests (operator-facing, state machine)   │
│  kind: Workflow  ·  states  ·  capabilities  ·     │
│  transitions  ·  human_in_the_loop                 │
└──────────────────────┬─────────────────────────────┘
                       │ WorkflowCompilerService
                       │ parse → validate → compile
┌──────────────────────▼─────────────────────────────┐
│  Layer 2 — Control Plane (WorkflowIR)              │
│  Protobuf intermediate representation              │
│  Engine-agnostic · versioned · formally typed      │
│  Agent registry · Capability routing               │
│  Event bus (NATS JetStream + CloudEvents)          │
└──────────┬───────────────────┬──────────┬──────────┘
           │                   │          │
┌──────────▼──┐  ┌─────────────▼──┐  ┌───▼──────────┐
│  Temporal   │  │   LangGraph    │  │     Argo     │
│  Adapter    │  │   Adapter      │  │   Adapter    │
└─────────────┘  └────────────────┘  └──────────────┘
     Layer 3 — Execution Engines
```

**The compiler pattern is the key structural advantage:** compile once to
WorkflowIR; swap engines without touching the manifest. This is LLVM for
workflows.

---

## 2. Competitive Landscape

### 2.1 Tool overview

| Tool | Category | Primary strength | Primary weakness |
|------|----------|-----------------|-----------------|
| **Serverless Workflow** | DSL specification | Vendor-neutral standard, CloudEvents native | No AI semantics, no formal IR, no capability model |
| **Temporal** | Execution engine | Durable execution guarantees, multi-language | Code-first only, single engine, no declarative model |
| **Argo Workflows** | K8s job orchestrator | Kubernetes-native, massive parallelism, DAGs | Batch/CI oriented, not suited for long-running AI |
| **LangGraph** | AI agent framework | AI-native design, human-in-the-loop, debugging | Python-only, in-process, not distributed |
| **Flyte** | ML pipeline orchestrator | Data lineage, caching, type safety, reproducibility | Kubernetes-only, ML/data focus, not AI agents |
| **Prefect** | Data workflow | Python ergonomics, hybrid execution | No multi-agent, no capability registry |
| **Conductor** | Microservice orchestrator | LLM task types emerging, event-driven | Single runtime, no formal IR, no engine portability |
| **Kestra** | General-purpose orchestrator + AI tasks | 1300+ plugins, embedded editor, AI agent tasks | Single engine, agents are embedded tasks not external services, no capability registry |
| **Zynax** | AI workflow control plane | Three-layer IR, engine portability, AI-native | Pre-release, no production adapter yet (M3 pending) |

### 2.2 Kestra — detailed assessment

Kestra (v1.0, September 2025) is an open-source, general-purpose orchestration
platform that added AI Agent tasks as a first-class feature. With 1,300+ plugins,
an embedded UI editor, and Git/Terraform integration, it targets both data engineers
and platform teams. Kestra AI Agents are LLM-driven tasks within Kestra's own
execution engine, not a separate control plane.

**What it does:**

An AI Agent task in Kestra launches an autonomous LLM process inside a flow
(Kestra's term for a workflow). The agent is given:
- A system message + prompt (guides LLM reasoning)
- Memory (KV Store or Redis, persists context across runs)
- Tools (web search via Tavily/Google, code execution, dynamic Kestra task
  invocation, MCP client, file system via Docker bind-mounts)

Supported LLM providers: OpenAI, Anthropic Claude, Google Gemini, Mistral,
Amazon Bedrock, Azure OpenAI, DeepSeek, Ollama.

**Architecture model:**

```
Kestra Engine (single runtime)
  └─ Flow (YAML)
      ├─ Task: fetch data (plugin)
      ├─ Task: AI Agent ────────► LLM (OpenAI/Anthropic/...)
      │         ├─ memory: Redis   ◄─── KV store
      │         └─ tools:
      │              ├─ web search (Tavily)
      │              ├─ execute Kestra task (dynamic)
      │              └─ run code (Judge0)
      └─ Task: store results (plugin)
```

Agents are tasks. The flow engine is Kestra. There is one runtime.

**Where Kestra stops:**

| Limitation | Consequence |
|------------|-------------|
| Single execution engine (Kestra's own) | No engine portability; migrating off Kestra requires rewriting all flows |
| Agents are embedded tasks, not external services | Agents cannot be deployed independently; capability reuse across flows requires copy-paste |
| No capability registry | Routing is static (task definition); no dynamic discovery of what agents can do |
| No formal IR | YAML is interpreted at runtime; no compile-time validation of state machine topology |
| Memory is LLM context only | No vector store, no workflow-scoped shared namespace, no isolation guarantees |
| Tools are hardcoded plugin types | Adding a new tool requires a Kestra plugin (JVM/Go); no gRPC adapter pattern |
| No CloudEvents-native event model | Events are Kestra-internal; integrating with CNCF tooling requires adapters |

**Relationship to Zynax:** Partial overlap, mostly complementary. Kestra's 1300+
plugins are a significant integration catalogue — wrapping Kestra task execution
as a Zynax adapter (a Kestra-backed capability provider) would give Zynax access
to that catalogue without building native integrations. Kestra could be a Zynax
Layer 3 execution backend for task-heavy workflows.

**Differentiation from Kestra AI:**

| Dimension | Kestra | Zynax |
|-----------|--------|-------|
| Engine model | Single proprietary engine | Any engine (Temporal, LangGraph, Argo, Kestra) |
| Agent model | LLM task inside a flow | External service implementing gRPC contract |
| Capability routing | Static — task definition names the plugin | Dynamic — broker discovers capable agents at runtime |
| Memory | LLM conversation context (KV/Redis) | Full workflow-scoped KV + vector platform |
| YAML compilation | Runtime interpretation | Compile to formal IR; errors caught before deployment |
| Multi-agent | Agents share memory in same flow | Agents are independent services; capability routing is the coordination primitive |
| Language | Agents are Kestra plugins (JVM/Go) | Agents are any language implementing AgentService gRPC |
| Target persona | Platform engineer adding AI tasks to existing orchestration | Platform team building AI-native distributed systems |

**Key differentiating message vs Kestra:**

> "Kestra adds AI tasks to general-purpose orchestration. Zynax is purpose-built
> infrastructure for AI-native systems where agents are independent services,
> execution engines are pluggable, and workflows are compiled — not interpreted."

**Synergies to exploit:**

1. **Kestra as a Zynax capability adapter**: wrap Kestra task execution behind
   the `AgentService` gRPC contract. Instantly exposes 1300+ Kestra plugins as
   Zynax capabilities without building native integrations.
2. **Kestra plugin library as a compatibility signal**: document which Kestra
   plugins have Zynax-native equivalents and which are best delegated to a
   Kestra adapter. Reduces migration friction for Kestra users.
3. **Kestra as a Zynax Layer 3 engine**: for task-heavy, data-pipeline-oriented
   workflows, a KestiraEngine adapter would let users mix Temporal (long-running
   AI interactions) and Kestra (data integration tasks) within the same IR.

---

### 2.3 Serverless Workflow — detailed assessment

Serverless Workflow (SW) deserves special attention because it is the closest
in intent and the strongest candidate for strategic alignment.

**What it is:**  
A CNCF Sandbox specification (accepted July 2020, v1.0.0 released January
2025) for vendor-neutral workflow definitions. Eleven core task types
(`call`, `fork`, `for`, `listen`, `emit`, `try`, `switch`, `do`, `run`,
`set`, `raise`). Expression language: jq (mandatory), JavaScript/JEXL
(optional). Event model: CloudEvents v1.0 native.

**Production runtimes implementing the spec:**

| Runtime | Backer | Status |
|---------|--------|--------|
| SonataFlow (ex-Kogito) | Red Hat / Apache | Production, Kubernetes-native |
| Apache EventMesh Workflow | Apache Foundation | Active |
| Synapse | Open source | Reference implementation |
| Lemline | Community | Minimal, no external DB required |

**SDKs:** Go, Java, .NET, PHP, Python, Rust, TypeScript.

**Where SW stops:**

- No capability or agent registry concept
- No memory or persistent context model
- No AI/LLM-specific task semantics (only generic `call`)
- No formal intermediate representation — each runtime interprets independently
- Human-in-the-loop is a generic `suspend`, not a semantic state
- Runtime interoperability in practice varies; spec compliance is not verified

**Relationship to Zynax:** Complementary, not competing. SW is the
standardization layer (DSL governance, multi-vendor buy-in). Zynax is
the abstraction layer above engines (IR, capability routing, AI semantics).
They solve different problems and can be combined.

### 2.4 Full feature benchmark

The following matrix scores eight tools across fifteen capabilities that
matter specifically for AI workflow orchestration at scale.

| Feature | SW | Temporal | Argo | LangGraph | Flyte | Prefect | Conductor | **Kestra** | **Zynax** |
|---------|:--:|:--------:|:----:|:---------:|:-----:|:-------:|:---------:|:----------:|:---------:|
| Declarative YAML/DSL | ✅ | ⚠️ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ✅ |
| AI / LLM native | ❌ | ❌ | ❌ | ✅✅ | ❌ | ❌ | ✅ | ✅✅ | **✅✅✅** |
| Human-in-the-loop (semantic state) | ⚠️ | ✅ | ⚠️ | ✅✅ | ⚠️ | ⚠️ | ✅ | ✅ | **✅✅✅** |
| Event-driven execution | ✅✅ | ✅ | ⚠️ | ✅ | ⚠️ | ✅ | ✅✅ | ✅ | **✅✅✅** |
| Multi-agent coordination | ❌ | ⚠️ | ❌ | ✅✅ | ❌ | ❌ | ✅ | ✅ | **✅✅✅** |
| Pluggable execution engines | ⚠️ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | **✅✅✅** |
| gRPC-native contracts | ✅ | ✅✅ | ⚠️ | ❌ | ❌ | ❌ | ✅ | ❌ | **✅✅✅** |
| CloudEvents native | ✅✅ | ⚠️ | ⚠️ | ⚠️ | ❌ | ⚠️ | ✅ | ⚠️ | ✅✅ |
| CNCF-aligned | ✅ Sandbox | ❌ | ✅ Incubating | ❌ | ✅ Incubating | ❌ | ⚠️ | ❌ | 🎯 M8 |
| Language-agnostic agents | ✅ | ⚠️ | ✅ | ❌ | ✅ | ✅ | ✅ | ⚠️ | **✅✅✅** |
| Formal IR / compilation step | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ⚠️ | ❌ | **✅✅✅** |
| Capability / skill registry | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | **✅✅✅** |
| Memory / persistent context | ❌ | ✅✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅✅ |
| Open source + permissive licence | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Compile-time error detection | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | **✅✅✅** |
| Plugin / integration catalogue | ❌ | ❌ | ⚠️ | ⚠️ | ✅ | ✅ | ✅ | **✅✅✅** | ⚠️ via adapters |

**Key:** ✅✅✅ best-in-class / native · ✅✅ strong · ✅ supported ·
⚠️ partial / workaround · ❌ absent · 🎯 planned

**Reading the benchmark:**  
Zynax leads or ties on every AI-specific and engine-portability dimension.
It lags on maturity (pre-release vs production). Serverless Workflow leads on
standardisation and ecosystem. Temporal leads on durable execution guarantees.
No single tool dominates all dimensions — which confirms the market gap.

---

## 3. Strategic Positioning

### 3.1 Zynax's unique value proposition

Zynax occupies a position no other tool holds: **the control plane layer for
AI workflows that spans all execution engines**.

The three differentiators that must be communicated in every external context:

**1. Engine portability via formal IR**  
WorkflowIR is compiled once from YAML and executes identically on any
engine adapter. Switching from Temporal to LangGraph requires no manifest
changes — only replacing the adapter. No other tool offers this guarantee
because no other tool has a formal IR.

**2. AI-native capability model**  
Agents register capabilities; workflows reference capability names, not
agent identities. The task broker routes dynamically. This decoupling means
any agent can satisfy any capability, enabling horizontal scaling,
blue-green agent deploys, and capability-based load balancing — none of
which are possible in systems that hardcode agent references.

**3. State machine with semantic AI states**  
`human_in_the_loop` is a first-class state type, not a workaround. Terminal
states are validated at compile time. Orphan states are caught before
deployment. These are structural guarantees the spec produces; Temporal
and LangGraph offer them only as runtime conventions.

### 3.2 Positioning against each competitor

| vs | Zynax positioning |
|----|-------------------|
| Serverless Workflow | "SW defines the DSL standard; Zynax is the AI-native control plane that executes it" |
| Temporal | "Use Temporal as a Zynax engine; swap to LangGraph without rewriting a single YAML file" |
| LangGraph | "LangGraph is an excellent Zynax engine for Python-native agents; the control plane is language-agnostic" |
| Argo Workflows | "Argo excels at batch DAGs on Kubernetes; Zynax handles long-running AI interactions above any runtime" |
| Conductor | "Both target AI services; Zynax differentiates on engine portability, formal IR, and open governance" |
| Kestra | "Kestra adds AI tasks to general-purpose orchestration. Zynax is purpose-built infrastructure where agents are external services, engines are pluggable, and workflows are compiled — not interpreted" |

### 3.3 Target buyer

Zynax is for **platform engineering teams** at organisations that:

- Are running or planning multi-agent AI systems in production
- Have tried LangGraph or Temporal and found engine lock-in to be a risk
- Run on Kubernetes and value CNCF-aligned tooling
- Want workflow definitions in version-controlled YAML, not application code
- Need to route tasks to agents dynamically based on capability, not identity

Zynax is **not** for:

- Teams building one-shot LLM pipelines (LangChain is sufficient)
- Teams doing batch ML training jobs (Flyte/Argo is the right tool)
- Teams that have committed to a single workflow engine and see no reason to change

### 3.4 Serverless Workflow compatibility strategy

**Recommendation: adopt SW YAML as an optional Layer 1 syntax in M4.**

The workflow-compiler already has a parser architecture (M2). Adding a SW
YAML parser that transpiles to WorkflowIR is 1–2 sprints of work. The
strategic return:

- Zynax can be positioned as a Serverless Workflow-compatible runtime
- Organizations invested in SW tooling can adopt Zynax without rewriting manifests
- CNCF Sandbox reviewers will view SW compatibility as a strong ecosystem signal
- SonataFlow and EventMesh communities become potential contributors

**Impedance mismatch to manage:**  
SW is task-oriented (imperative); Zynax is state-oriented (declarative).
The transpiler maps SW task sequences to Zynax state chains. SW `fork`
maps to parallel transition targets. SW `listen` maps to event-triggered
state activation. AI-specific extensions (capability refs, memory
configuration) remain Zynax-native and are not expected in SW manifests.

**Proposed M4 dual-syntax:**

```yaml
# Zynax native (current — unchanged)
apiVersion: zynax.io/v1
kind: Workflow
metadata:
  name: code-review
spec:
  initial_state: await_review
  states:
    await_review:
      type: human_in_the_loop
      on:
        - event: review.approved
          goto: merge
        - event: review.rejected
          goto: revise
    merge:
      type: terminal
    revise:
      type: normal
      actions:
        - capability: request_changes
      on:
        - event: changes.ready
          goto: await_review
```

```yaml
# Serverless Workflow compatible (new in M4 — compiles to same IR)
apiVersion: workflows.serverlessworkflow.io/v1
kind: Workflow
metadata:
  name: code-review
spec:
  states:
    - name: await_review
      type: operation
      actions:
        - name: wait-for-human
          sleep:
            before: PT0S
      onEvents:
        - eventRef: review.approved
          transition: merge
        - eventRef: review.rejected
          transition: revise
    - name: merge
      type: end
    - name: revise
      type: operation
      actions:
        - functionRef:
            refName: request_changes
      transition: await_review
```

---

## 4. Implementation Roadmap

This section maps all open issues to a sequenced delivery plan. Each PR is
sized to stay within the 900-line limit (ADR decision 002).

### 4.1 Priority tiers

```
TIER 0 — In flight (PR #164, merging)
  #159  CI secret/PII scanning gate for AI context files
  #163  Retroactive audit of CLAUDE.md and AGENTS.md
  #132  DCO Signed-off-by in CLAUDE.md commit template

TIER 1 — M1 completion (parallelisable with M2)
  PR B:  #133 + #131   agents/sdk/pyproject.toml + stub_generation BDD steps
  PR C:  #152 + #153   post-merge proto auto-regen + freshness gate E2E

TIER 2 — M2 core (strict sequential order)
  PR D:  #84 + #85     structural + semantic validators
  PR E:  #86           WorkflowGraph → WorkflowIR serialization
  PR F:  #87 + #154    gRPC API layer + BDD step definitions
  PR G:  #155 + #142 + #88   coverage gate + make test + Makefile targets

TIER 3 — Security framework (before M3)
  PR H:  #158          ADR + CODEOWNERS authorization model
  PR I:  #161 + #160 + #162  policy + checklist + previsualization gate

TIER 4 — M3 Temporal adapter (M3 milestone, existential gate)
  [Issues to be created from epic #101 M3 scope]

TIER 5 — Serverless Workflow compatibility (M4)
  [New issue: SW YAML parser in workflow-compiler — see §3.4]

TIER 6 — KB work (post-M8, blocked on #157 + all milestones)
  #143 #144 #145 #146 #147 #148
```

### 4.2 PR B — M1 CI tooling

**Issues:** #133, #131  
**Size estimate:** S (< 200 lines)

`agents/sdk/pyproject.toml` is referenced in ci.yml but does not exist.
The Python test step is guarded by a `has_py` file count, so it does not
fail today — but will fail the moment any `.py` file is added to
`agents/sdk/`. This is a latent CI break.

`stub_generation.feature` exists in `protos/tests/features/` but has no
corresponding `steps_test.go`. All other eight feature files have Go
implementations; this one is silently skipped by godog.

**Acceptance criteria:**

- [ ] `agents/sdk/pyproject.toml` created with minimum viable config for ruff, mypy, pytest-bdd
- [ ] `protos/tests/stub_generation/steps_test.go` created
- [ ] All scenarios in `stub_generation.feature` pass with `GOWORK=off go test ./...`
- [ ] CI `test-unit` godog step runs and passes

### 4.3 PR C — M1 proto automation

**Issues:** #152, #153  
**Size estimate:** M (< 400 lines, mostly YAML)

Post-merge proto stub auto-regeneration closes the loop on the pre-merge
freshness gate that already exists in lint. Without it, `main` can have
updated `.proto` files and stale stubs after merge.

The freshness gate end-to-end validation confirms the existing `buf generate
+ git diff` step actually catches real drift — it has never been exercised
because no PR has modified a `.proto` file since stubs were committed.

**Acceptance criteria:**

- [ ] `.github/workflows/proto-generate.yml` created; triggers on proto/buf file changes to `main`
- [ ] Bot commits use `[skip ci]` to avoid CI loop
- [ ] Freshness gate end-to-end: changing a proto without regenerating fails CI
- [ ] BSR connectivity confirmed in CI runner logs

### 4.4 PR D — M2 structural and semantic validators

**Issues:** #84, #85  
**Size estimate:** M–L (< 700 lines)  
**Depends on:** M2 foundation (merged in M1: domain types, ParseManifest, WorkflowGraph)

**Structural validators** (#84) operate on the compiled `WorkflowGraph`:

| Validator | Error code | Description |
|-----------|-----------|-------------|
| `OrphanStateDetector` | `ORPHAN_STATE` | Every state reachable from `initial_state` |
| `TerminalStateValidator` | `NO_TERMINAL_STATE` | At least one `StateTypeTerminal` exists |
| `TransitionTargetValidator` | `UNKNOWN_STATE_REFERENCE` | All `goto` targets resolve to known states |
| `InitialStateValidator` | `NO_INITIAL_STATE` | `initial_state` resolves to a known state |

**Semantic validators** (#85) check business-level correctness:

| Validator | Error code | Description |
|-----------|-----------|-------------|
| `CapabilityRefValidator` | `INVALID_CAPABILITY_REF` | Capability names are snake_case with no reserved prefix |
| `EventNameValidator` | `INVALID_EVENT_NAME` | Event names follow `domain.resource.action` pattern |
| `NamespaceValidator` | `INVALID_FIELD_VALUE` | Namespace is a valid DNS label |
| `DuplicateTransitionValidator` | `DUPLICATE_STATE_NAME` | No two transitions have identical event+guard combinations |

**Architecture note:** validators form a pipeline; each receives the
`*WorkflowGraph` from the previous stage. Structural validators run first;
semantic validators run only if structural validation passes. This matches
the existing `ParseManifest` → `Build` chain.

**Acceptance criteria:**

- [ ] `internal/domain/validators.go` implements all validators as a `ValidatorPipeline`
- [ ] `internal/domain/validators_test.go` covers every error path (≥95% coverage)
- [ ] `ParseManifest → Build → Validate` integration test added to `manifest_test.go`
- [ ] GOWORK=off `go test ./internal/domain/...` passes

### 4.5 PR E — WorkflowGraph to WorkflowIR serialization

**Issue:** #86  
**Size estimate:** M (< 500 lines)  
**Depends on:** PR D (validated graph is the input)

The compiler step converts the validated internal `WorkflowGraph` into a
`WorkflowIR` proto message ready for gRPC transmission.

```
WorkflowGraph ──► Compiler.Compile() ──► pb.WorkflowIR
```

**Key decisions:**

- `workflow_id`: UUID v4, generated at compile time (not from manifest)
- `compiled_at`: RFC 3339 timestamp, set by compiler, not caller
- `ir_version`: constant `"v1"` until a breaking IR change ships
- State IDs in the IR are the YAML key names (no transformation)
- Transition guard expressions are passed through verbatim (evaluated by the engine adapter at runtime)

**Acceptance criteria:**

- [ ] `internal/domain/compiler.go`: `Compile(*WorkflowGraph, CompileOptions) (*pb.WorkflowIR, error)`
- [ ] All `WorkflowIR` proto fields populated correctly
- [ ] Round-trip test: `ParseManifest → Build → Validate → Compile → proto.Marshal → proto.Unmarshal` produces identical IR
- [ ] `workflow_id` is a valid UUID v4
- [ ] No proto imports in the domain layer (mapping happens in compiler.go, not manifest.go)

### 4.6 PR F — gRPC API layer and BDD contract tests

**Issues:** #87, #154  
**Size estimate:** L (< 900 lines)  
**Depends on:** PR E

The gRPC server wires all domain components:

```
CompileWorkflow(manifest_yaml)
  → domain.ParseManifest(yaml)
  → domain.Build(manifest)
  → domain.ValidatorPipeline.Run(graph)
  → domain.Compile(graph, opts)
  → pb.CompileWorkflowResponse{WorkflowIR}
```

`ValidateManifest` runs parse + validate only (no compile).
`GetCompiledWorkflow` retrieves a previously compiled IR by ID (in-memory
store; no database yet — M2 constraint).

`workflow_compiler_service.feature` already exists; step definitions are
missing. The in-memory test server pattern from `agent_registry_service/`
is the reference implementation.

**Acceptance criteria:**

- [ ] `api/grpc_server.go` implements `WorkflowCompilerServiceServer`
- [ ] `CompileWorkflow`, `ValidateManifest`, `GetCompiledWorkflow` all implemented
- [ ] `api/grpc_server_test.go`: unit tests for all three RPCs
- [ ] `protos/tests/workflow_compiler_service/steps_test.go` created
- [ ] All BDD scenarios in `workflow_compiler_service.feature` pass
- [ ] `GOWORK=off go test ./...` passes from `protos/tests/`

### 4.7 PR G — CI gates and Makefile targets

**Issues:** #155, #142, #88  
**Size estimate:** S–M (< 350 lines, mostly YAML + Makefile)  
**Depends on:** PR F (coverage gate needs a service with tests)

Coverage gate: `workflow-compiler` is at 96.2% coverage post-M2. The gate
enforces ≥90% so regressions fail before merge. Applied only to services
with a `go.mod`.

`make test` runs the complete local test suite inside Docker (Go unit,
godog BDD, Python pytest-bdd, spec validation). Today contributors must
run four separate commands and know which flags to use.

`make validate-spec` and `make dry-run FILE=path` mirror the CI spec
validation gate locally. `dry-run` is a stub in M2 (no running service
yet); it becomes functional in M3 when the gRPC server is live.

**Acceptance criteria:**

- [ ] `test-unit` job gains coverage gate step; fails at < 90%
- [ ] `make test` runs all tiers in Docker; exits non-zero on any failure
- [ ] `make validate-spec` passes on current `spec/`
- [ ] `make dry-run FILE=spec/workflows/examples/minimal.yaml` exits 0

### 4.8 M3 — Temporal adapter (existential milestone)

M3 is not yet broken into implementation issues. That decomposition happens
at the start of M3 planning. The following architecture decisions are
already made (ADR-015):

- `EngineAdapterService` implements a single gRPC contract
- `Submit(WorkflowIR)` translates IR states to Temporal workflow functions
- `Signal(run_id, event)` delivers events to running workflow instances
- `Cancel(run_id)` cancels a workflow execution
- `Watch(run_id)` streams execution events back as CloudEvents

**The M3 success criterion:** a workflow defined in YAML, compiled to IR,
submitted to Temporal via the adapter, completes execution, and produces
the same output as if executed by a local in-memory runner. This is the
proof-of-concept for the entire control plane model.

**Critical risk:** semantic translation between WorkflowIR and Temporal
workflow functions is harder than it appears. States map to workflow
coroutines; transitions map to signals; human_in_the_loop maps to
`workflow.GetSignalChannel`. Guard evaluation (CEL expressions) must happen
in the adapter, not in Temporal. This translation layer deserves a dedicated
design spike before implementation begins.

---

## 5. Security Architecture

### 5.1 AI context file security (active, #157)

All AI context files (`CLAUDE.md`, `AGENTS.md`, `docs/ai-assistant-setup.md`)
are in a public repository. A gitleaks CI gate (PR #164, merged) scans for
secrets, PII, local paths, and internal infrastructure details on every PR
that touches these files.

The full security framework (CODEOWNERS authorization, content policy,
previsualization gate) is deferred to post-M8 — the point at which active
knowledge base population begins. The scanning gate is in place now, before
any KB content is written.

### 5.2 Supply chain security

- `govulncheck` runs on every PR (pinned to `v1.1.4` until Go ≥ 1.25 in
  all environments; currently Go 1.25 is in use — upgrade pin in M3)
- `gitleaks` default ruleset runs as part of the AI context scan
- `pip-audit` is planned for M2 CI (#137)
- Proto backward-compatibility is enforced by `buf breaking` on every PR

### 5.3 Architecture invariants that must not be broken

Established in M1; enforced by the layer boundary CI gate:

1. **No shared database.** Cross-service reads via gRPC only.
2. **No Layer 1→3 coupling.** YAML manifests are never imported by Go services.
3. **Contracts before implementations.** Proto + feature files merge before service code.

---

## 6. Recommendations

### 6.1 Immediate actions (M2 window)

| Priority | Action | Rationale |
|----------|--------|-----------|
| Critical | Execute PR D → E → F → G in strict order | M2 completion unblocks M3 |
| Critical | Run PR B and PR C in parallel with PR D | M1 closure before M3 begins |
| High | Create M3 design spike issue for Temporal translation layer | Prevents M3 delay from late-discovered complexity |
| High | Open a GitHub Discussion: "Should Zynax accept Serverless Workflow YAML?" | Community signal before committing M4 scope |

### 6.2 M3 window (after M2 merges)

| Priority | Action | Rationale |
|----------|--------|-----------|
| Critical | Temporal adapter M3 — treat as the highest-risk deliverable | Engine portability proof is existential |
| High | Contact SonataFlow maintainers (Red Hat) re: Zynax as SW runtime | Strategic partnership signal for CNCF application |
| High | Identify first external adopter | CNCF Sandbox requires evidence of external use |
| Medium | Draft SW YAML parser design for M4 | Reduces M4 scope risk |

### 6.3 Pre-M8 CNCF requirements

CNCF Sandbox requires:

- [ ] At least one public production use case
- [ ] At least two active maintainers from different organisations
- [ ] Clear governance model (CODEOWNERS + MAINTAINERS.md)
- [ ] Security disclosure process (SECURITY.md)
- [ ] Alignment with CNCF TOC criteria (open, vendor-neutral, cloud-native)

Serverless Workflow compatibility (M4–M5) will significantly strengthen
the CNCF application by demonstrating ecosystem alignment.

### 6.4 What not to do

| Anti-pattern | Why |
|-------------|-----|
| Add Serverless Workflow support before M3 ships | Engine portability must be proven first; SW compatibility is a growth strategy, not a foundation |
| Expose WorkflowIR as a public API before it stabilises | IR is internal; premature stability promises constrain architectural evolution |
| Build a hosted cloud offering before CNCF Sandbox | Cloud hosting is a business decision that requires a stable open-source foundation |
| Add LLM-specific task types to the YAML manifest | Capabilities are the right abstraction; LLM details belong in agents, not in manifests |

---

## 7. Viability Assessment

### 7.1 Market timing

The AI agent orchestration market is in the infrastructure phase (2025–2027):
frameworks proliferate, production deployments begin, and the pain of
framework lock-in becomes visible. This is precisely when control plane
tools become valuable — after the "just use LangGraph" phase and before
the "we're stuck on LangGraph" renegotiation.

Zynax is correctly positioned for this window. M3 must ship before the
market consolidates around one or two frameworks, because consolidation
reduces the value of engine portability.

### 7.2 Strengths to leverage

| Strength | How to amplify |
|----------|---------------|
| Three-layer compiler model | Explicit documentation; position as "LLVM for AI workflows" |
| gRPC-native contracts | Cloud-native teams already speak gRPC; reduce adoption friction |
| AI-first YAML semantics | Publish comparison: "Zynax vs LangGraph for distributed agents" |
| CNCF trajectory | Reference CNCF in every external communication from M3 onward |
| BDD contract tests | "Spec before code" is a credibility signal; emphasise in contributor docs |

### 7.3 Risks and mitigations

| Risk | Probability | Mitigation |
|------|-------------|-----------|
| M3 Temporal translation is more complex than estimated | High | Design spike before implementation; timebox to 2 weeks |
| LangGraph consolidates as the de facto standard | Medium | LangGraph becomes a Zynax engine adapter — not a threat, a feature |
| Serverless Workflow moves to Incubation before Zynax reaches Sandbox | Low | Accelerates joint positioning; Zynax becomes "the AI-native SW runtime" |
| Community fragmentation across too many AI frameworks | Medium | Adapter pattern is the answer; each framework becomes an engine |
| No external adopter before M8 CNCF application | Medium | Identify one design partner at M3 (single org, non-public if needed) |

### 7.4 Verdict

**Zynax is viable.** The market gap is real. The architecture is sound
(proven pattern: LLVM, Kubernetes, TensorFlow all use the three-layer
compiler model). The roadmap is executable. The single existential
condition is M3: if the Temporal adapter ships and demonstrates semantic
equivalence across engines, the value proposition is proven. Everything
else is execution.

---

## Appendix A — Open Issues by Priority

See the priority order pinned on #157 for the live list. Reproduced here
at the time of this document for reference:

| Tier | Issues | Description |
|------|--------|-------------|
| 0 (in flight) | #159, #163, #132 | Security scan gate + DCO fix (PR #164) |
| 1 M1 completion | #133, #131, #152, #153 | PR B + PR C |
| 2 M2 core | #84, #85, #86, #87, #154, #155, #142, #88 | PR D → E → F → G |
| 3 Security framework | #158, #161, #160, #162 | PR H + PR I |
| 4 M3 Temporal | TBD | Engine adapter (existential milestone) |
| 5 SW compatibility | TBD new issue | M4 Serverless Workflow YAML parser |
| 6 KB work | #143–#148 | Post-M8, post-#157 |

---

## Appendix B — ADR Index (as of M1)

| ADR | Decision |
|-----|----------|
| ADR-001 | gRPC as inter-service protocol |
| ADR-002 | Python 3.12 for all agent adapters |
| ADR-003 | uv as Python package manager |
| ADR-004 | BDD-first testing strategy |
| ADR-005 | Apache-2.0 licence |
| ADR-006 | Monorepo structure |
| ADR-007 | Pydantic-settings for agent config |
| ADR-008 | No shared databases |
| ADR-009 | Go for platform services, Python for agents |
| ADR-010 | Pluggable agent runtime |
| ADR-011 | Declarative YAML control plane |
| ADR-012 | WorkflowIR as engine-agnostic IR |
| ADR-013 | Adapter-first, no SDK required for agents |
| ADR-014 | Event-driven state machine |
| ADR-015 | Pluggable workflow engines |
| ADR-016 | Layered testing strategy |
| ADR-017 | Contract test isolation (GOWORK=off) |

---

## Appendix C — External References

- [Serverless Workflow Specification](https://serverlessworkflow.io/)
- [Serverless Workflow GitHub](https://github.com/serverlessworkflow/specification)
- [CNCF Landscape — App Definition](https://landscape.cncf.io/card-mode?category=application-definition-image-build)
- [Temporal Documentation](https://docs.temporal.io/)
- [LangGraph Documentation](https://langchain-ai.github.io/langgraph/)
- [Argo Workflows Documentation](https://argo-workflows.readthedocs.io/)
- [Flyte Documentation](https://flyte.org/)
- [Conductor OSS](https://www.conductor-oss.org/)
- [Kestra Documentation](https://kestra.io/docs)
- [Kestra AI Agents — Introducing AI Agents blog post](https://kestra.io/blogs/introducing-ai-agents)
- [CloudEvents Specification](https://cloudevents.io/)
- [CNCF Sandbox Criteria](https://github.com/cncf/toc/blob/main/process/sandbox.md)
- [Zynax Execution Architecture](execution-architecture.md)
