# Keel Platform — Engineering Contract

> **Authoritative engineering contract for contributors and AI assistants.**
> Read entirely before writing a single line of code.
> Every decision here is backed by an ADR. When in doubt, check `docs/adr/`.
>
> This file is read automatically by AI coding assistants (Claude Code, Cursor,
> Copilot, Gemini Code Assist, and others). It is equally authoritative for
> human contributors. There is one standard, not two.

---

## 0. What Is Keel?

> **Keel is a declarative, cloud-native, engine-agnostic control plane
> for AI agent workflows.**

It is NOT:
- An LLM framework (it does not call LLMs)
- A workflow engine (it does not execute workflows)
- A DevOps tool (it does not replace CI/CD)

It IS:
- The **Kubernetes of AI workflows** — a control plane that abstracts execution
- A **declarative intent layer** — workflows defined in YAML, not code
- An **engine-agnostic adapter** — Temporal, LangGraph, Argo are plugins
- A **capability router** — agents are capabilities, not identities

The core insight:
> Kubernetes won by abstracting containers.
> Keel wins by abstracting intelligence workflows.

---

## 1. The Three-Layer Separation (Non-Negotiable)

Every decision in this codebase must respect this separation:

```
┌─────────────────────────────────────────────────────────┐
│  LAYER 1 — INTENT                                       │
│  YAML manifests (Kubernetes-style)                      │
│  What should happen. Declarative. Versionable.          │
│                                                         │
│  workflows/ · policies/ · agent-defs/ · routing-rules/  │
├─────────────────────────────────────────────────────────┤
│  LAYER 2 — COMMUNICATION                                │
│  Contracts: gRPC (sync) + AsyncAPI/NATS (async)         │
│  How things talk. Typed. Multi-language.                │
│                                                         │
│  protos/keel/v1/ · spec/asyncapi/                       │
├─────────────────────────────────────────────────────────┤
│  LAYER 3 — EXECUTION                                    │
│  Workflow Engine Plugins (Temporal / LangGraph / Argo)  │
│  How it runs. Pluggable. Swappable.                     │
│                                                         │
│  services/engine-adapter/ · agents/adapters/            │
└─────────────────────────────────────────────────────────┘
```

**Violations of this separation are hard blockers at code review.**

- Layer 1 (YAML) must never import from Layer 3.
- Layer 2 (contracts) must never contain business logic.
- Layer 3 (execution) must never be a hard dependency — always behind an interface.

---

## 2. Full Architecture

```
     ┌──────────────────────────────────────────────┐
     │        YAML Manifests (Intent)               │
     │  kind: Workflow · AgentDef · Policy · Route   │
     └──────────────────┬───────────────────────────┘
                        │ keel apply
     ┌──────────────────▼───────────────────────────┐
     │         API Gateway (Go)                     │
     │  REST + gRPC-gateway · auth · rate limit     │
     └──────────────────┬───────────────────────────┘
                        │
     ┌──────────────────▼───────────────────────────┐
     │      Workflow Compiler (Go)                  │
     │  YAML → Canonical IR (Intermediate Rep.)     │
     │  Validates · Normalises · Routes to engine   │
     └─────┬──────────────────────┬────────────────┘
           │                      │
     ┌─────▼──────┐        ┌──────▼──────────────────┐
     │  Agent     │        │  Engine Adapter (Go)    │
     │  Registry  │        │  Temporal / LangGraph /  │
     │  (Go)      │        │  Argo — pluggable        │
     └─────┬──────┘        └──────────────────────────┘
           │
     ┌─────▼──────────────────────────────────────────┐
     │              Task Broker (Go)                   │
     │  Capability routing · assignment · retry        │
     └─────┬──────────────────────────────────────────┘
           │ capability dispatch
     ┌─────▼──────────────────────────────────────────┐
     │         Execution Adapters Layer                │
     │  LLM · HTTP API · Git · CI/CD · LangGraph agent │
     │  No SDK required — wrap anything                │
     └─────────────────────────────────────────────────┘
           │
     ┌─────▼──────────────────────────────────────────┐
     │   Event Bus — NATS JetStream (Go)               │
     │   AsyncAPI spec · all events flow here          │
     └─────────────────────────────────────────────────┘
           │
     ┌─────▼──────────────────────────────────────────┐
     │   Memory Service (Go)                           │
     │   KV + Vector · shared context across workflow  │
     └─────────────────────────────────────────────────┘
```

---

## 3. Development Model — Docker-First

> **Nothing runs on the host machine except Docker Desktop.**

```bash
make build-tools      # Build dev image (Go 1.22 + Python 3.12 + buf + all tools)
make dev-up           # Full platform stack
make lint             # Lint everything inside Docker
make test-unit        # All tests inside Docker
make generate-protos  # Go + Python stubs from .proto
```

---

## 4. The Four Non-Negotiable Mandates

### 4.1 Amazon API Mandate
All services expose capabilities ONLY via versioned gRPC. No shared databases.
No cross-service imports. Contracts are proto files — reviewed like production code.

### 4.2 Twelve-Factor
Config via env vars. Stateless processes. Logs to stdout. Port binding.
Backing services as attached resources.

### 4.3 Clean Code
- Go: functions ≤ 30 lines. No `panic` in production. All errors wrapped.
- Python (agents/adapters): functions ≤ 20 lines. No `print()`. Types 100%.
- Both: no magic numbers. No dead code. Comments explain WHY.

### 4.4 BDD-First
`.feature` file written before any implementation. Gherkin for both Go (godog)
and Python (pytest-bdd). A feature without a passing scenario is not done.

---

## 5. Repository Layout

```
keel/
├── AGENTS.md                      ← You are here
├── ARCHITECTURE.md                ← Deep dive: layers, IR, adapters
├── go.work                        ← Go workspace (all platform services)
├── Makefile                       ← ALL commands go through here
│
├── spec/                          ← Declarative intent layer
│   ├── AGENTS.md                  ← YAML schema rules
│   ├── schemas/                   ← JSON Schema for YAML manifests
│   │   ├── workflow.schema.json
│   │   ├── agent-def.schema.json
│   │   └── policy.schema.json
│   └── workflows/examples/        ← Reference YAML manifests
│       ├── code-review.yaml
│       ├── ci-pipeline.yaml
│       └── research-task.yaml
│
├── services/                      ← Go platform services
│   ├── AGENTS.md
│   ├── agent-registry/            ← Agent identity + capability registry
│   ├── task-broker/               ← Capability routing + task dispatch
│   ├── memory-service/            ← Shared KV + vector memory
│   ├── event-bus/                 ← NATS + AsyncAPI event backbone
│   ├── api-gateway/               ← REST + YAML apply endpoint
│   ├── workflow-compiler/         ← YAML → IR compiler
│   └── engine-adapter/            ← Temporal/LangGraph/Argo adapter
│
├── agents/                        ← Python pluggable agent layer
│   ├── AGENTS.md
│   ├── sdk/                       ← keel-sdk (optional, no SDK required)
│   └── adapters/                  ← Execution adapters (no SDK required)
│       ├── AGENTS.md
│       ├── http/                  ← Wrap any HTTP API as an agent
│       ├── llm/                   ← Wrap Bedrock, Ollama, OpenAI
│       ├── git/                   ← GitHub/GitLab event-driven adapter
│       └── langgraph/             ← LangGraph workflow adapter
│
├── protos/keel/v1/                ← gRPC contracts
├── infra/docker/                  ← Docker-first dev environment
├── docs/adr/                      ← ADR-001 through ADR-015
└── tools/                         ← golangci-lint, ruff, mypy configs
```

---

## 6. The Workflow Model

Keel workflows are **event-driven state machines**, not DAGs.
This is a deliberate choice (see ADR-014).

```yaml
# spec/workflows/examples/code-review.yaml
kind: Workflow
apiVersion: keel.io/v1

metadata:
  name: code-review-workflow
  namespace: engineering

spec:
  initial_state: review

  states:
    review:
      actions:
        - capability: request_review
          timeout: 24h
      on:
        - event: review.approved
          goto: merge
        - event: review.changes_requested
          goto: fix
        - event: review.timeout
          goto: escalate

    fix:
      actions:
        - capability: summarize_feedback
      on:
        - event: push
          goto: review

    merge:
      actions:
        - capability: merge_pr
      on:
        - event: merge.success
          goto: done
        - event: merge.conflict
          goto: fix

    escalate:
      actions:
        - capability: notify_human
      type: human_in_the_loop

    done:
      type: terminal
```

Key properties supported by state machine model:
- ✅ Loops (`fix → review → fix`)
- ✅ Async events (`review.approved`, `push`)
- ✅ Human-in-the-loop (`escalate` state)
- ✅ Long-running (days, not seconds)
- ✅ Timeout handling

---

## 7. The Capability Model

Everything executable in Keel is a **capability**, not a named agent.

```
summarize          request_review      run_tests
open_mr            review_code         merge_pr
notify_human       search_web          execute_sql
```

Capabilities are:
- Declared in `AgentDef` YAML manifests.
- Registered in `agent-registry`.
- Routed by `task-broker` based on capability match.
- Executed by whatever adapter/agent has registered that capability.

**The workflow YAML never names an agent directly. It names a capability.**
This decouples intent from implementation — you can swap the executor
without changing the workflow definition.

---

## 8. Adapter-First Integration (No SDK Required)

Agents do NOT need the Keel SDK. Any system can become a capability
by deploying an adapter. See `agents/adapters/AGENTS.md`.

```
HTTP API   →  http-adapter        →  capability: call_api
Bedrock    →  llm-adapter         →  capability: summarize
GitHub     →  git-adapter         →  capability: open_mr, request_review
LangGraph  →  langgraph-adapter   →  capability: research_topic
CI system  →  ci-adapter          →  capability: run_tests
```

Adapters implement the gRPC `AgentService` contract. That is the ONLY
requirement. No language. No framework. No SDK.

### How to Connect — by Role and Language

The right integration path depends on what role your code plays and what language
it is in. Three paths exist and they are all equal from the platform's perspective:

| Your situation | Path | Where to start |
|---------------|------|---------------|
| Wrapping an existing system in any language | Adapter | `agents/adapters/AGENTS.md` |
| Building a new Python agent | Python SDK | `agents/sdk/AGENTS.md` |
| Connecting from Go, TypeScript, Java, Rust, or any other language | Raw proto stubs | `protos/AGENTS.md §8` |

The proto files in `protos/keel/v1/` are the universal boundary. Any language
that can speak gRPC can call any Keel service or implement any capability. See
`ARCHITECTURE.md §11` for the full interoperability picture and `protos/AGENTS.md §8`
for language-specific consuming instructions.

---

## 9. Platform Services — Go Standards

See `services/AGENTS.md` for full patterns.

### Key layer rule (enforced by import analysis in CI)
```
api → domain ← infrastructure
       ↑
  domain: ZERO imports from api or infrastructure
```

### Key Go patterns
```go
// Errors: always wrap with context
return nil, fmt.Errorf("find agent %s: %w", id, domain.ErrAgentNotFound)

// Error mapping: ONLY in api layer
case errors.Is(err, domain.ErrAgentNotFound):
    return status.Errorf(codes.NotFound, err.Error())

// Context: first arg on every I/O function
func (r *repo) FindByID(ctx context.Context, id AgentID) (*Agent, error)

// Logging: slog, structured, contextual
slog.InfoContext(ctx, "workflow compiled",
    "workflow_id", id, "target_engine", engine, "states", len(states))

// Config: envconfig, fail fast on startup
func Load() (*Config, error) { envconfig.Process("KEEL_<SVC>", &cfg) }
```

---

## 10. Agent/Adapter Layer — Python Standards

See `agents/AGENTS.md` for full patterns.

Two ways to add execution capability:

**A — Adapter (preferred, no SDK):**
```python
# Implement one gRPC method: ExecuteCapability(request) → stream of events
# Declare capabilities in an AgentDef YAML
# Deploy as a container — done
```

**B — SDK Agent (full control):**
```python
# Implement AgentRuntime Protocol: execute(task, context) → AsyncIterator[TaskEvent]
# Pluggable runtime: LangGraph, AutoGen, Direct, Custom
# AgentContext injected — never constructed
```

---

## 11. Definition of Done

A feature is DONE when **ALL** are true:

- [ ] `.feature` file written before implementation
- [ ] All unit tests pass (`make test-unit`)
- [ ] Go: `golangci-lint` clean. Python: `ruff` + `mypy --strict` clean.
- [ ] `make security` clean
- [ ] Health probes correct
- [ ] Structured logs + metrics + traces for new behaviour
- [ ] YAML schema updated if new manifest kind added
- [ ] Proto change: backward-compatible OR new version + migration guide
- [ ] ADR created if architectural decision was made
- [ ] Required approvals obtained (see `GOVERNANCE.md §2`)

---

## 12. Hard Constraints — Contributors and AI Assistants

These rules apply equally to every contributor — human or AI.
Breaking any of them is a hard blocker at code review.

**Both layers:**
- Never install tools on host — everything in Docker
- Never commit secrets, tokens, or credentials
- Never skip the `.feature` file — no feature exists without a passing scenario
- Never share a database between services
- Never couple Layer 1 (YAML) to Layer 3 (engines)
- Never make changes outside the stated scope of an issue or task

**Commit hygiene (human and AI contributors):**
- Subject line ≤ 72 characters, imperative mood, capitalized, no period at end
- No `@mentions` anywhere in commit messages — issue references go in footer only (`Closes #123`)
- No emojis in commit messages
- Never merge `main` into a feature branch — always rebase (`git rebase origin/main`)
- Use `--force-with-lease` when pushing after a rebase, never bare `--force`
- `Assisted-by: ToolName/model-id` for AI attribution — never `Co-Authored-By:` for AI tools

**Go services:**
- Never `panic` in production code paths
- Never discard errors (`_ = f()`) — wrap all errors with `fmt.Errorf("context: %w", err)`
- Never import from another service's `internal/`
- Never hardcode engine names — always behind an interface
- Never expose credentials, tokens, or auth URLs in logs or structured output
- Never disable TLS verification — it must be on by default
- Never use shell execution (`exec.Command`) for git, kubectl, helm — use Go libraries
- Close HTTP response bodies, file handles, and archive readers via `defer`
- Delete temporary files on all code paths — success and error alike
- Machine-readable output → `stdout`; human-readable status messages → `stderr`

**Python agents/adapters:**
- Never call platform services via HTTP — only gRPC stubs
- Never instantiate platform clients in Runtime — use `context.*`
- Never require SDK adoption — adapters work without it
- Never hardcode LLM model names — env var always
- Never expose credentials or auth tokens in logs
- Close all I/O resources (HTTP clients, file handles, streams) in `finally` blocks or via context managers

---

*Keel — The control plane for AI-driven systems*
*Apache 2.0 · CNCF Sandbox Candidate*
