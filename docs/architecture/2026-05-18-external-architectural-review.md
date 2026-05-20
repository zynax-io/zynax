# Zynax — Architectural Review

**Repository:** `github.com/zynax-io/zynax`
**Commit reviewed:** `9c9927f` (2026-05-18, HEAD of `main`)
**Project age at review:** ~4 weeks (first commit 2026-04-20)
**Reviewer perspective:** Principal architect, CNCF maintainer mindset, performance & security lens

---

## 0. Methodological Note

This review is grounded in direct inspection of the source tree, proto contracts, CI workflows, Makefile, Helm/Compose definitions, ADRs, README/ARCHITECTURE/ROADMAP, the open issue list, and the full 205-commit git history. Where a claim is made about the codebase, a concrete file/line reference is cited so every conclusion is independently verifiable.

The review is intentionally adversarial. Sycophancy is not useful at architectural review boards. Where the project does excellent work, that is called out plainly; where claims do not survive contact with the source, that is called out equally plainly.

---

## 1. Executive Summary

### 1.1 Scores (0–10)

| Dimension | Score | One-line rationale |
|---|---|---|
| **Overall architectural soundness** | **6.0** | Sound *concept*, premature *execution claims*, healthy *internal structure* |
| Simplicity | 4.5 | Concept is simple; the surrounding process scaffolding is wildly disproportionate |
| Performance | 4.0 | No benchmarks; multiple polling loops; map-iteration determinism bug inside a Temporal workflow |
| Security | 3.5 | No auth on the control-plane front door; insecure inter-service gRPC; misleading "CEL" guard with silent fail-open |
| Maintainability | 6.5 | Hexagonal layout, good ADR habit, ≥90% domain coverage gate are real assets |
| Scalability | 4.0 | Polling-based dispatch and watch; no horizontal-scale story below M6 |
| Reliability | 4.5 | Best-effort event publish swallows errors; SSE stream dies at 30 s WriteTimeout; health probes are unconditional 200s |
| Product-market fit | 3.5 | Differentiation against Temporal/Argo/Kestra/LangGraph is not yet demonstrated |
| CNCF alignment | 4.0 | Topology and choices (gRPC, NATS, Temporal, OTel intent) are aligned; the "Sandbox Candidate" badge is self-applied and premature |
| Open-source sustainability | 2.5 | 1 human contributor, 0 stars/forks/watchers, no MAINTAINERS.md, governance written for an organisation that does not yet exist |
| Documentation density | 8.0 | ADRs 1–19, ARCHITECTURE.md, ROADMAP.md, governance, contributing — genuinely strong |
| Code quality (where it exists) | 7.0 | What's been written is clean, idiomatic Go |
| Test rigor (claimed) | 5.5 | 398 BDD scenarios sound impressive but most test in-test stubs, not real services |

### 1.2 Top Strengths

1. **Honest hexagonal architecture in the implemented services.** `services/{workflow-compiler,engine-adapter,api-gateway}` cleanly separate `internal/domain`, `internal/api`, `internal/infrastructure`. Domain code imports zero infrastructure. This is textbook and rare in young projects.
2. **Excellent dependency hygiene.** The fattest module (`engine-adapter`) has 5 direct dependencies. Most have 4–5. No leftpad-style transitive sprawl.
3. **Backward-compatible proto design discipline.** ADR-001 commits to "ordinals are permanent"; `WorkflowIR` correctly evolved with additive fields 7–9 retaining the M1 envelope. This is the right way.
4. **Multi-module Go workspace** properly used — every service is its own go.mod, with `go.work` orchestrating local development and `GOWORK=off` documented for hermetic CI.
5. **Sound conceptual frame.** The "control plane / data plane" split for AI agent workflows is a defensible idea. ADR-014 (state machines over DAGs) and ADR-015 (pluggable engines) are well-reasoned positions.
6. **CI is genuinely well-structured.** Path-based change detection, per-area lanes (lint-proto / lint-go / lint-python), DCO + conventional-commit + PR-size gates, pinned action SHAs. This is professional CI work.
7. **Stable, modern toolchain.** Go 1.26.3, buf, golangci-lint v2, govulncheck, gitleaks, OpenSSF Scorecard. The tooling story is mature.

### 1.3 Top Weaknesses

1. **Reality-claim divergence is severe.** The README and CLAUDE.md state M1–M4 are "Complete" and the platform runs end-to-end. In fact, **5 of 7 declared platform services have zero Go implementation** (agent-registry, task-broker, memory-service, event-bus, and the registry side of api-gateway's dependencies are stub-only). The Python SDK is a 5-line empty `__init__.py`. The "end-to-end" path collapses at the first capability dispatch.
2. **The `DispatchCapabilityActivity` dials a service that does not exist.** `services/engine-adapter/cmd/engine-adapter/main.go:120` opens a gRPC channel to `TaskBrokerService`. There is no task-broker server in the tree. `make run-local` will start successfully, accept a `zynax apply`, run `IRInterpreterWorkflow`, then fail at the first action.
3. **The "CEL guard evaluation" is a 24-line equality-only string parser with silent fail-open.** `services/engine-adapter/internal/domain/interpreter.go:178-198`. Supports `==` and `!=` only. No `&&`, `||`, `<`, `>`, function calls, numeric coercion, or null-handling. Worse, *unrecognised expressions return `true`* — the system silently fires every transition with a malformed guard.
4. **Determinism bug inside a Temporal workflow.** `interpreter.go:204-209` (`resolveTemplate`) iterates a `map[string]string` and `strings.ReplaceAll`s into a JSON template. Go map iteration order is randomised. Temporal workflows must be deterministic; replay after worker restart will diverge whenever two keys produce overlapping substitution effects. This is a latent production-incident generator.
5. **Workflow-compiler implementation violates its own proto contract.** The proto comment block (`workflow_compiler.proto:9-13`) explicitly says *"errors are expressed as a repeated CompilationError in CompileWorkflowResponse, **not in gRPC metadata**"* and *"all errors found are reported — not just the first"*. The implementation (`workflow-compiler/internal/api/server.go:46, 56, 60`) does exactly the opposite: it returns a gRPC `InvalidArgument` carrying *only the first* error's `Message` string and discards the structured list. Contract test stubs validate the stub, not the server, so this is invisible to CI.
6. **No authentication anywhere on the api-gateway.** `api-gateway/internal/api/handler.go` routes — `POST /api/v1/apply`, `DELETE /api/v1/workflows/{id}` — execute and cancel workflows with no auth middleware, no API key, no JWT, no OIDC. The control plane front door is wide open. This is acknowledged in the architecture diagram ("→ auth · rate limit") but never implemented.
7. **All three liveness probes are the same unconditional `w.WriteHeader(200)`.** `api-gateway/cmd/api-gateway/main.go:92-95` and `engine-adapter/cmd/engine-adapter/main.go:171-179`. K8s cannot distinguish startup from readiness from liveness. Restart logic, rolling updates, and traffic admission are all degraded.
8. **The SSE log streaming endpoint is broken by design.** `api-gateway/cmd/api-gateway/main.go:64` sets `WriteTimeout: 30 * time.Second`. The handler at `handler.go:117-150` is supposed to stream events for the lifetime of a workflow run (potentially hours/days per the README's "long-running (days)" claim). Every `zynax logs` will die at exactly 30 seconds.
9. **Best-effort event publication silently swallows all errors.** `engine-adapter/internal/domain/interpreter.go:65, 71, 75, 81` and `infrastructure/temporal_workflow.go:71-76` discard the publisher's return value with `_ = pub.Publish(...)`. Operational invisibility — there is no metric, no log, no alert when the event bus is down.
10. **"CNCF Sandbox Candidate" badge is self-applied and misleading.** The shield is hard-coded in `README.md:7`. The project is not listed on `landscape.cncf.io`, has not filed a TOC submission (M8 on the roadmap), has one contributor, and is four weeks old. To a reader who doesn't read carefully, the badge implies a status the project does not hold.
11. **Open-source governance scaffolding is theatrical for a one-person project.** `GOVERNANCE.md` (451 lines) details supermajority maintainer nominations, lazy-consensus voting, RFC processes, and triage rotations. `MAINTAINERS.md` does not exist. The 195 commits are by one human (plus 7 by Renovate). The 59 open issues are all opened by the same author. This is not governance; it is governance cosplay.
12. **Process complexity wildly exceeds project complexity.** 2,624 lines of GitHub Actions YAML, 359-line Makefile with 73 targets, 19 ADRs, "REASONS Canvas" methodology, "SPDD prompt governance" with Tier-1/Tier-2 secrecy, "AI context budget" CI gate — for ~7,000 lines of actual Go and an empty Python SDK. The ratio of process artefacts to shipping product is unhealthy.

### 1.4 Highest-Priority Recommendations (snapshot — full list in §17)

| # | Recommendation | Priority | Effort |
|---|---|---|---|
| 1 | Stop claiming milestones are "Complete" until the runnable system end-to-end actually runs end-to-end | **Critical** | Documentation only |
| 2 | Remove the "CNCF Sandbox Candidate" badge until a TOC submission is in flight | **Critical** | 1 line |
| 3 | Fix the workflow-compiler contract violation (return structured errors, not just the first) | **Critical** | ~50 LOC |
| 4 | Replace the bespoke guard parser with `cel-go` or remove the "CEL" claim from docs | **Critical** | ~200 LOC or 5-line doc change |
| 5 | Fix the `resolveTemplate` map-iteration determinism bug | **Critical** | ~10 LOC |
| 6 | Add minimum-viable auth (bearer-token or mTLS) at the api-gateway | **Critical** | ~150 LOC |
| 7 | Implement at least a stub task-broker so the advertised end-to-end path can run | **High** | ~500 LOC |
| 8 | Split health/ready/startup probes with real semantics | **High** | ~80 LOC |
| 9 | Fix the SSE WriteTimeout (use `http.NewResponseController` or run the stream on a separate listener) | **High** | ~30 LOC |
| 10 | Implode the governance/process surface to match project reality | **High** | Delete-only |

---

## 2. Product & Market Assessment

### 2.1 The Stated Positioning

> *"Zynax is to AI workflows what Kubernetes is to containers — a control plane that abstracts the execution layer behind a declarative, versionable API."* — README.md

### 2.2 The Concept, Tested

The analogy is more rhetorical than structural. Kubernetes succeeded because (a) containers were a standardised, low-level primitive (OCI image, runc) that pre-existed Kubernetes; (b) operators wanted multi-tenant orchestration with a uniform API across heterogeneous runtimes; (c) there was no incumbent control plane.

For AI agent workflows in 2026, the analogous primitive layer is **not** standardised. "An agent workflow" is not yet an OCI image. Temporal already abstracts durable execution; LangGraph already abstracts agent graphs; Argo Workflows already provides declarative Kubernetes-native YAML; Kestra and Airflow II already provide engine-agnostic orchestration. Each engine has its own SDK, its own state model, its own runtime expectations. The Workflow IR Zynax proposes (a state machine over capability dispatches, ADR-012, ADR-014) is one possible abstraction over those — but it is *not* the universal lowest common denominator that "container" was.

The market is not waiting for a universal control plane the way it was waiting for one in 2014. The competition is:

| Competitor | What it owns | Why Zynax is at a disadvantage |
|---|---|---|
| **Temporal** (incumbent durable execution) | Production-proven durable workflows | Zynax depends on it — it is below Zynax. Why use the wrapper when the engine is mature? |
| **LangGraph** (incumbent agent graphs) | Default in the LLM ecosystem | Network effects with LangSmith/LangChain; agent developers are already there |
| **Argo Workflows** (CNCF graduated) | Kubernetes-native declarative YAML workflows | Already CNCF graduated; already has the operator/community Zynax aspires to |
| **Kestra** | Declarative engine-agnostic orchestration | Same pitch, multi-year head start, real adopters |
| **Restate, Inngest, Trigger.dev** | Durable-execution-as-a-service | Aim at the same developer who would consume Zynax |

The roadmap's M5 (Adapter Library) actually reveals the strategic risk: every adapter Zynax adds (LLM, HTTP, Git, CI, LangGraph) is **commoditised infrastructure** — a thin shim. The unique value remaining is the IR + capability-routing model. That has to be uniquely good to justify adoption.

### 2.3 Likely Real Differentiator

The strongest under-developed thread in the project is **declarative-first agent definition with engine-agnostic guarantees**. If a workflow YAML can be authored once and demonstrably executed on Temporal *and* LangGraph with identical observable behaviour — including the human-in-the-loop semantics that DAG engines do not natively express — there is a niche. It is a small niche (the population of organisations that want to swap workflow engines is small), but it is real.

The risk is that the project is positioned as a generalist control plane when its credible niche is "the YAML manifest layer for hybrid agentic + durable workflows". The roadmap should reflect that narrowing.

### 2.4 Adoption Barriers

1. **No production users.** Zero stars, zero forks, zero external contributors as of the review date.
2. **No demo deployment.** A reader who finds the repo cannot follow `make run-local && zynax apply` to a working workflow because the task-broker is missing.
3. **Brand collision risk.** "Zynax" returns disambiguation noise in common web searches; the trademark posture isn't documented.
4. **CNCF Sandbox path requires 2+ maintainers from different organisations** (correctly noted in M8). Currently impossible to satisfy.

### 2.5 Verdict

Concept: defensible. Positioning: overreaching. Differentiation: latent but not yet demonstrated. The strongest move is to **narrow the pitch** to one workflow that no incumbent does well today and prove it end-to-end, before scaling the platform surface.

---

## 3. Current Architecture Overview

### 3.1 Topology (as designed)

```
 YAML manifests
       |
       | HTTP REST (POST /api/v1/apply)
       v
 +----------------+         gRPC                +--------------------+
 | api-gateway    |---------------------------->| workflow-compiler  |
 | (Go)           |                              +--------------------+
 |                |  gRPC                        +--------------------+
 |                |---------------------------->| engine-adapter     |
 |                |                              | (Go, Temporal)     |
 |                |  gRPC                        +----------+---------+
 |                |---------------------------->           |
 +----------------+                                        | Activity
                                                           v
                          +----------------------------+
                          |  task-broker (NOT IMPL)    | <- gap
                          +-----------+----------------+
                                      |
                                      v
                          +----------------------------+
                          |  agent-registry (NOT IMPL) | <- gap
                          +-----------+----------------+
                                      |
                                      v
                          +----------------------------+
                          |  agents/adapters/*         |
                          |  (Python SDK = empty;      |
                          |   http adapter = real)     |
                          +----------------------------+

                          +----------------------------+
                          |  event-bus (NOT IMPL)      | <- gap
                          +----------------------------+
                          +----------------------------+
                          |  memory-service (NOT IMPL) | <- gap
                          +----------------------------+
```

### 3.2 Topology (actually implemented)

| Component | Implementation status | LOC | Verdict |
|---|---|---|---|
| `services/api-gateway` | Real Go service | 1,576 | Clean, missing auth/metrics/middleware |
| `services/workflow-compiler` | Real Go service | 3,113 | Cleanest service in the tree; contract violation in error reporting |
| `services/engine-adapter` | Real Go service | 2,275 | Functional Temporal wiring; broken capability dispatch (no broker) |
| `services/agent-registry` | **Feature file only** | 0 | Stub — protos define the contract, BDD tests target in-memory stubs |
| `services/task-broker` | **Feature file only** | 0 | Stub — engine-adapter dials a service that isn't there |
| `services/memory-service` | **Feature file only** | 0 | Stub |
| `services/event-bus` | **Feature file only** | 0 | Stub |
| `cmd/zynax` | Real CLI | 1,892 | Cobra-based, clean |
| `cmd/zynax-ci` | Real validator CLI | 2,361 | JSON-schema validation, ai-context budget |
| `agents/adapters/http` | Real Go adapter | 939 | The only working executor |
| `agents/sdk/src/zynax_sdk/__init__.py` | **5-line empty package** | 5 | Claimed but not built |
| `protos/zynax/v1/*.proto` | Real | 1,700 | Excellent, additive evolution |
| `protos/generated/{go,python}` | Generated | ~16,500 | Committed, regenerated via buf |
| `protos/tests/*` | Real BDD harness | 7,174 | Tests **stub servers**, not real services |
| `infra/docker-compose/` | Real | — | Only starts the 3 implemented services + Temporal + NATS |
| `infra/helm/` | **Does not exist** | — | CHANGELOG claims "Helm chart templates"; only patterns exist |

### 3.3 The Three-Layer Frame (ADR-014 / ARCHITECTURE.md §2)

The "Intent (YAML) / Communication (Contracts) / Execution (Engines)" decomposition is conceptually sound and corresponds 1:1 to actual repo layout (`spec/`, `protos/`, `services/`+`agents/`). This is one of the project's strongest architectural moves.

### 3.4 Engine Adapter Pattern (ADR-015)

The `domain.WorkflowEngine` Go interface (`services/engine-adapter/internal/domain/engine.go:14-40`) is a textbook hexagonal port. The TemporalEngine implementation cleanly sits behind it. *Adding a second engine adapter is a credible 2-week effort* — the abstraction is honest, not leaky. This is genuinely well done.

---

## 4. Architectural Strengths (Detailed)

### 4.1 Hexagonal Layering, Enforced

Every implemented service follows:

```
services/<name>/
  cmd/<name>/main.go         <- composition root, env-var config
  internal/domain/           <- pure, no SDK imports
  internal/api/              <- gRPC/HTTP handlers, error mapping
  internal/infrastructure/   <- Temporal, NATS, DB clients
```

`go list -deps` would confirm `internal/domain` has zero imports of `internal/api` or `internal/infrastructure`. The `AGENTS.md` explicitly enforces this ("Never import from another service's `internal/`"). Junior contributors get the right shape by copying any existing service.

### 4.2 Proto Contract Discipline

`protos/zynax/v1/workflow_compiler.proto` is exemplary:
- Permanent enum ordinals declared in comments.
- Backward-compat policy referenced (ADR-001 §backward-compat).
- M1 envelope preserved as `bytes ir_payload` (field 6) when M2 added structured `states`, `initial_state`, `ir_version` (fields 7–9).
- gRPC `INVALID_ARGUMENT` semantics explicit.
- "Never returns a WorkflowIR" invariants documented per-method.

Across 8 proto files (1,700 lines), the style is consistent and reviewable.

### 4.3 Dependency Minimalism

The fattest direct dependency list is `engine-adapter`: Temporal SDK, gRPC, protobuf — that's it. `api-gateway` has 5 direct deps (envconfig, cobra-less, just stdlib HTTP). No web frameworks, no ORMs, no microservice frameworks. This is the right discipline.

### 4.4 CI Architecture

The `ci.yml` workflow is professional:
- DCO + change-detection job at the top so subsequent lanes can skip cheaply.
- Lint and test separated per language.
- All third-party actions pinned to SHA, not tag.
- Concurrency group keyed on `workflow + ref + sha` so back-to-back pushes don't pile up.
- ≥90% domain coverage gate on services, ≥80% on adapters and CLI tools.

### 4.5 Conscientious ADR Practice

19 ADRs, all dated, all with rationale. ADR-008 ("no shared databases") is the kind of constraint that prevents architectural rot two years out. ADR-001 (gRPC + backward-compat ordinals) governs proto evolution. ADRs are referenced from code comments — which means they're actually read.

### 4.6 Capability Routing as a Concept

Routing workflows to *capabilities* (`summarize`, `merge_pr`) rather than agent IDs (ADR-013) is a legitimately good model. It mirrors Kubernetes Services-over-Pods. When/if the agent-registry is built, this remains an asset.

---

## 5. Architectural Weaknesses (Detailed)

### 5.1 The Reality-Claim Gap

`README.md`, `CLAUDE.md`, `AGENTS.md`, and `ARCHITECTURE.md` all assert milestones M1–M4 are **Complete**, with v0.3.0 as the version. The "Try it with Docker" section advertises:

```
make run-local
zynax apply spec/workflows/examples/code-review.yaml
zynax logs wf-<hex>
# streams state-transition events
```

The reality:
1. `make run-local` starts api-gateway + workflow-compiler + engine-adapter + Temporal + NATS. **It does not start task-broker, agent-registry, memory-service, or event-bus** because they are not implemented.
2. `zynax apply code-review.yaml` → api-gateway → workflow-compiler → engine-adapter → `IRInterpreterWorkflow` → `DispatchCapabilityActivity` dials `localhost:50053` → connection refused. The workflow fails immediately.
3. `zynax logs` cannot stream state-transition events because `WatchWorkflow` polls `DescribeWorkflowExecution` every 2 s and only emits run-level status — not the IR `CurrentState` (which is never queried back from Temporal at all).

### 5.2 The "CEL" Misnomer with Silent Fail-Open

`services/engine-adapter/internal/domain/interpreter.go:178-198`:

```go
func evalGuard(expr string, ctx map[string]string) bool {
    expr = strings.TrimSpace(expr)
    for _, op := range []string{"!=", "=="} {
        idx := strings.Index(expr, op)
        if idx < 0 { continue }
        // ... split, trim, compare ...
        switch op {
        case "==": return lval == rval
        case "!=": return lval != rval
        }
    }
    return true // fail-open for unrecognised expressions
}
```

Three problems compounded:

1. **It is not CEL.** Real CEL supports types, logical operators, function calls, map/list indexing, and is sandboxed. This is a `strings.Index` match.
2. **Fail-open is the worst possible default.** A typo (`ctx.foo === "bar"`) silently returns `true` and the workflow advances through a gate it should have been blocked at.
3. **The README, ARCHITECTURE.md, and milestone notes all claim "CEL guards"** — this is a documentation-truth gap.

The `code-review.yaml` example uses `guard: "{{ .context.escalation_count }} < 2"`. That guard does not parse — the implementation only supports `==` and `!=`, returns `true` for everything else, and the workflow takes the transition unconditionally.

### 5.3 Determinism Bug in a Temporal Workflow

Temporal workflows must be **deterministic**. On worker restart, Temporal replays the workflow from event history and expects identical decisions. Any non-determinism causes a `nondeterminism panic` and the workflow becomes unrecoverable.

`services/engine-adapter/internal/domain/interpreter.go:204-209`:

```go
func resolveTemplate(template string, ctx map[string]string) []byte {
    result := template
    for k, v := range ctx {
        result = strings.ReplaceAll(result, "{{ .ctx."+k+" }}", v)
    }
    return []byte(result)
}
```

`for k, v := range ctx` over a `map[string]string` has **randomised iteration order** in Go (since Go 1.0, intentionally). If two `ctx` keys produce substitutions that overlap, the output depends on iteration order. Under replay this will diverge — a latent production-incident generator.

Similarly `mergePayload` (lines 222-237) iterates `map[string]interface{}` from `json.Unmarshal`. Both functions are invoked from inside `IRInterpreterWorkflow` (the registered Temporal workflow). Fix: sort keys before iterating.

### 5.4 Workflow-Compiler Contract Violation

`protos/zynax/v1/workflow_compiler.proto:13` (proto doc):

> *Errors are expressed as a repeated CompilationError in CompileWorkflowResponse, **not in gRPC metadata**. All errors found are reported — not just the first.*

`services/workflow-compiler/internal/api/server.go:46-60`:

```go
if len(parseErrs) > 0 {
    return nil, status.Error(codes.InvalidArgument, parseErrs[0].Message)  // only first; in status not response
}
```

The handler returns the first error in `status.Error` and discards the structured list. The api-gateway's compile-error response is therefore always empty in practice. `ValidateManifest` on the same file does it correctly; the two methods have inconsistent semantics.

### 5.5 No Auth, No Rate Limit, No Audit

The architecture diagram shows `→ auth · rate limit · REST-to-gRPC` next to API Gateway. The implementation has only the third. There is no authentication, no authorization, no rate limiting, no audit log, no request ID propagation, no request-level logging middleware on `POST /api/v1/apply`, `DELETE /api/v1/workflows/{id}`, or `GET /api/v1/workflows/{id}/logs`.

For a control plane this is critical-severity. A reachable Zynax api-gateway is a remote workflow-execution facility.

### 5.6 Polling Where Pushing Was Designed

1. `engine-adapter/internal/domain/activity.go:42` — `pollInterval: 500ms` on `GetTask` until terminal. With 1,000 concurrent workflows averaging 10 actions at 30 s each, polling at 500 ms produces 600,000 `GetTask` RPCs/second to the broker.
2. `engine-adapter/internal/infrastructure/temporal.go:107-127` — `Watch` polls `DescribeWorkflowExecution` every 2 s. Temporal's `GetWorkflowExecutionHistory` supports long-polling and history streaming. The current implementation emits an event every 2 s whether anything changed or not, with empty `FromState`/`ToState`.

### 5.7 Best-Effort Becomes Best-Forgotten

`engine-adapter/internal/domain/interpreter.go:65, 71, 75, 81`:

```go
_ = pub.Publish(ctx, "zynax.workflow.completed", ec.WorkflowID, ec.CurrentState)
```

Errors are discarded and the Go linter is silenced. There is no metric, no log, no alert. If the event bus is down, operators only discover via missing dashboards. Fix: two lines of `slog.Warn` + `metrics.IncCounter`.

### 5.8 SSE Handler vs Server WriteTimeout

`api-gateway/cmd/api-gateway/main.go:62-67`: `WriteTimeout: 30 * time.Second` is a hard deadline per connection. The SSE log streaming handler must remain open for the lifetime of a workflow run. Every `zynax logs` will die at exactly 30 seconds.

Fix: use `http.NewResponseController(w).SetWriteDeadline(time.Time{})` (Go 1.20+), or run the streaming endpoint on a second `http.Server` with `WriteTimeout: 0`.

### 5.9 Probes That Aren't Probes

All three probe handlers (`/healthz`, `/readyz`, `/startupz`) are identical `w.WriteHeader(200)`. K8s cannot distinguish startup from readiness from liveness. Restart logic, rolling updates, and traffic admission are all degraded.

Correct minimum: `/startupz` one-shot ready flag; `/readyz` checks gRPC client states; `/livez` checks the last-successful-work timestamp.

### 5.10 Compose-Time Configuration Bug

`infra/docker-compose/docker-compose.yml:128`: `ZYNAX_GW_REGISTRY_ADDR: "localhost:50052"`. `localhost` inside the api-gateway container resolves to the api-gateway container, not the agent-registry.

### 5.11 Mixed YAML Libraries

The workflow-compiler imports `gopkg.in/yaml.v3` but a transitive dep pulls `go.yaml.in/yaml/v2`. Two YAML packages in one binary is worth flagging for the next dependency cleanup.

### 5.12 Three Compose Files in Two Directories

```
infra/docker/docker-compose.yml
infra/docker/docker-compose.tools.yml
infra/docker/docker-compose.test.yml
infra/docker-compose/docker-compose.yml  <- different directory
```

Two `make` targets, two directories with confusable names, two compose files attempting to be canonical. Pick one.

---

## 6. Code Quality Assessment

### 6.1 General Quality

Within the implemented Go services (~7,000 LOC), code quality is **good**. Idiomatic Go. Short functions. Clear separation. Errors wrapped with `fmt.Errorf("... : %w", err)`. Test files alongside source files. No global mutable state. No `panic` in production paths.

### 6.2 Function Length Discipline (AGENTS.md: "Go functions ≤ 30 lines")

Sampled functions are at or near the limit with annotations where exceeded. The rule is followed in spirit.

### 6.3 Notable Code Smells

| Location | Smell | Severity |
|---|---|---|
| `engine-adapter/internal/domain/interpreter.go:178-198` | Bespoke parser of CEL-like expressions; should be a library | High |
| `engine-adapter/internal/domain/interpreter.go:204-209` | Map iteration in workflow-deterministic path | High |
| `workflow-compiler/internal/api/server.go:46, 56, 60` | Single-error contract violation | High |
| `engine-adapter/internal/infrastructure/temporal_workflow.go:71` | Silently swallowed activity error | Medium |
| `engine-adapter/internal/infrastructure/temporal.go:88` | `Cancel(ctx, runID, _)` discards reason | Low |
| `api-gateway/cmd/api-gateway/main.go:92-95` | Three identical probe handlers | High (K8s) |
| `agents/sdk/src/zynax_sdk/__init__.py` | 5-line empty SDK claimed in README | Documentation gap |

### 6.4 Technical Debt Concentration

The single largest debt item is in `engine-adapter/internal/domain/interpreter.go` — three serious correctness issues in 225 lines. This file should be the next refactor target before any new milestone work.

---

## 7. Performance Analysis

### 7.1 No Benchmarks Exist

There are no `*_test.go` files containing `func Benchmark…` anywhere in the implemented services. No `make bench`. No performance regression gate.

### 7.2 Hot-Path Observations

1. `workflow-compiler.CompileWorkflow` is `sync.RWMutex` over `map[string]*zynaxv1.WorkflowIR` — will become a contention point at >1k req/s.
2. `engine-adapter.DispatchCapabilityActivity` polls every 500 ms — at workflow execution scale, this becomes the dominant traffic source.
3. `resolveTemplate` allocates a new string with each `strings.ReplaceAll` call. A `strings.Builder` or single regex pass is ~5× cheaper.

### 7.3 Score Justification

**4/10.** No benchmarks; polling-heavy patterns; no pprof; documented performance claims absent.

---

## 8. Security Review

### 8.1 Threat Model

**Not documented.** `SECURITY.md` is 32 lines and only describes the vulnerability-reporting flow. There is no STRIDE analysis, no asset inventory, no trust-boundary diagram.

### 8.2 Findings

| # | Finding | Severity |
|---|---|---|
| S1 | No authentication on api-gateway | **Critical** |
| S2 | No authorization / multi-tenant isolation | **Critical** |
| S3 | Inter-service gRPC uses `insecure.NewCredentials()` | **High** |
| S4 | `reflection.Register(srv)` enabled in production | Medium |
| S5 | "CEL guards" fail-open on parse error | High (logical security) |
| S6 | `--insecure` flag with `InsecureSkipVerify` in production CLI build | Medium |
| S7 | No request ID / correlation ID — incident forensics are degraded | Medium |
| S8 | No SBOM published with releases | Medium |
| S9 | Container images built without `USER` directive (needs check) | Medium |
| S10 | `make run-local` doesn't enable TLS on api-gateway HTTP — users may copy to production | Medium |

### 8.3 Supply Chain

**Positive:** OpenSSF Scorecard badge, Renovate bot, SHA-pinned Actions, govulncheck + gitleaks + bandit + pip-audit integrated.
**Gaps:** No cosign signing, no SLSA provenance, no SBOM attached to releases.

### 8.4 YAML Injection Surface

`workflow-compiler` parses untrusted YAML. There is no `yaml.Decoder.KnownFields` enforcement and no document-size cap inside the parser. Harden with explicit max-document size and `KnownFields(true)`.

### 8.5 Score Justification

**3.5/10.** Strong supply-chain *posture*, but the actual application-level security surface is incomplete in critical ways (no auth, no authz, no mTLS).

---

## 9. Scalability Review

### 9.1 Horizontal Scalability

Not yet demonstrated. The workflow-compiler's in-memory IR store is the single largest blocker — multiple replicas would not share IRs. The engine-adapter consumes the IR on `SubmitWorkflow` and never needs to retrieve it from the compiler again, so making the compiler stateless (drop the store) is the correct path.

### 9.2 Bottlenecks (expected)

| Component | Predicted bottleneck | Mitigation |
|---|---|---|
| workflow-compiler | In-memory store, no persistence | Make stateless |
| engine-adapter (Watch) | Temporal `DescribeWorkflowExecution` polling | Switch to history long-poll |
| engine-adapter (DispatchCapability) | 500 ms polling of broker | Push-based completion |

### 9.3 Score Justification

**4/10.** Foundation is twelve-factor-compatible; the bottlenecks named above are real and unaddressed.

---

## 10. Reliability Review

### 10.1 Idempotency

`CompileWorkflow` is **not** idempotent — every call generates a fresh `wf-<hex>` ID. Resubmitting the same manifest produces a new workflow. `kubectl apply` is idempotent; `zynax apply` should be too.

### 10.2 Retries and Circuit Breakers

None present. The engine-adapter does not retry the broker dispatch; the api-gateway does not retry the compiler/engine.

### 10.3 Score Justification

**4.5/10.** The reliability primitives (Temporal, NATS) are excellent. The Zynax layer adds little reliability of its own and a few new failure modes (template determinism, swallowed events).

---

## 11. Testing Assessment

### 11.1 What Exists

- 23 `.feature` files with 398 scenarios (godog).
- 13 Go `*_test.go` files within `services/`.
- ≥90% domain coverage gate for services, ≥80% total for adapters and CLI.
- `buf breaking` + `buf lint` in CI.

### 11.2 What's Missing

- No benchmarks.
- No fuzz tests (YAML parser and guard parser are obvious targets).
- No integration tests against real services.
- No end-to-end test (impossible without task-broker).
- No chaos/fault-injection tests.
- No load tests.
- No mutation testing.

### 11.3 The BDD Stub Problem

`protos/tests/agent_registry_service/steps_test.go` defines a private `registryStub` implementing the agent-registry proto in-test. The BDD scenarios execute against this stub. There is no real agent-registry server. These are **contract-style tests** — they pin what the proto says, not what any server does. They are useful for guarding backward compatibility but should be clearly distinguished from "service tests".

### 11.4 Score Justification

**5.5/10.** Lots of BDD scaffolding, mature CI orchestration, but coverage of behaviour (not lines) is weak; no benchmarks/fuzz/E2E.

---

## 12. CI/CD Assessment

### 12.1 What's in Place

11 workflows (2,624 LOC). Quality of `ci.yml`:
- Action SHAs pinned (not tags).
- `concurrency:` with `cancel-in-progress: false` — appropriate for protected branches.
- `permissions: contents: read` — least-privilege.
- Path-based job skipping.
- Coverage gating and live coverage comment on every PR.

### 12.2 Gaps

- No multi-arch container builds in CI.
- No cosign signing of release artifacts (required for SLSA L3).
- No SBOM generation in release workflows.
- No release automation.

### 12.3 Score Justification

**6.5/10.** Above-average CI for an early-stage project; falls short of CNCF-graduated standards (signing, SBOM, multi-arch).

---

## 13. Documentation Assessment

### 13.1 Quality

The ADRs are unusually well-written. ADR-014 (state machines over DAGs) and ADR-015 (pluggable engines) are crisp and short. The README balances narrative and quickstart well.

### 13.2 Gaps

- No architectural diagrams beyond ASCII art.
- No troubleshooting guide.
- No tutorial that ends with a working workflow.
- No performance/capacity-planning documentation.

### 13.3 Score Justification

**8/10.** Genuinely the strongest dimension of the project.

---

## 14. Dependency Analysis

### 14.1 Concerns

| Dep | Concern | Action |
|---|---|---|
| `prometheus/client_golang` | Declared but no `/metrics` endpoint registered | Wire it, or remove |
| `otelgrpc` | Declared but no exporter/tracer initialization visible | Wire it, or remove |
| `gopkg.in/yaml.v3` + `go.yaml.in/yaml/v2` | Two YAML libraries in the same binary | Pick one |

### 14.2 Score Justification

**8/10.** Minimal, well-chosen, license-compatible. Could close out the "declared but unused" items.

---

## 15. CNCF Ecosystem Fit

### 15.1 Alignment

- gRPC + protobuf (CNCF graduated), NATS JetStream (CNCF incubating), CloudEvents (CNCF graduated), OTel intent declared, Apache 2.0 license, OpenSSF Scorecard.

### 15.2 Falls Short of Sandbox Bar

- ✗ Two maintainers from different organisations — 1 human author.
- ✗ Active community signals — 0 stars/forks/external issues.
- ✗ MAINTAINERS.md — not present.
- ✗ Public production usage — none.
- ✗ Project age — 4 weeks; Sandbox typically expects 6+ months.
- ✗ CNCF Landscape entry — not present.

The badge `CNCF-Sandbox_Candidate` is **self-applied** via shields.io and not an official CNCF designation.

### 15.3 Score Justification

**4/10.** Technical alignment is real; organisational alignment is not yet earned.

---

## 16. Open Source Governance Review

### 16.1 The Mismatch

`GOVERNANCE.md` (451 lines) describes supermajority maintainer nominations, lazy-consensus voting, RFC processes, and triage rotations for an organisation of 1 human contributor. The volume of governance scaffolding — combined with the SPDD "REASONS Canvas" methodology, AI-context budget gates, Tier-1/Tier-2 prompt-secrecy framework — suggests the project is **investing in process rather than product**.

### 16.2 Recommendation

Defer 80% of the governance surface until there are 5+ contributors. Keep: CODE_OF_CONDUCT.md, SECURITY.md, CONTRIBUTING.md, DCO, conventional commits. Defer: maintainer supermajority rules, RFC process, lazy-consensus voting.

### 16.3 Score Justification

**2.5/10** for actual community health; **6/10** for paper governance.

---

## 17. Refactoring Opportunities

Ranked by (impact × confidence) / effort:

| # | Refactor | Impact | Effort | Notes |
|---|---|---|---|---|
| R1 | Fix workflow-compiler error reporting to return structured `Errors[]` per contract | High | XS (~50 LOC) | One-day fix; immediate contract truth |
| R2 | Fix the `resolveTemplate` determinism bug | Critical | XS (~10 LOC) | Sort keys before iterating |
| R3 | Replace bespoke guard parser with `cel-go` (or remove "CEL" claim from docs) | High | S (~200 LOC) | Closes a security/correctness gap |
| R4 | Split probe handlers into three handlers with real semantics | High | S (~80 LOC) | Required for production K8s |
| R5 | Fix SSE WriteTimeout via `http.NewResponseController` | High | XS (~30 LOC) | Makes `zynax logs` actually work |
| R6 | Add minimum-viable auth (static bearer token via env var) at api-gateway | Critical | S (~150 LOC) | Bridge until proper OIDC/JWT |
| R7 | Wire Prometheus + OTel — declared in go.mod and never exposed | Medium | S (~200 LOC) | Closes a documentation-vs-reality gap |
| R8 | Implement minimum task-broker (in-memory, single-replica) | High | M (~500 LOC) | Makes the platform end-to-end actually run |
| R9 | Make workflow-compiler stateless (drop the in-memory store) | Medium | S | Required before M6 K8s story |
| R10 | Replace polling-based broker `GetTask` with streaming `WatchTask` (already in proto) | High | M | Proto contract already supports this |
| R11 | Add idempotency to `CompileWorkflow` (hash → ID) | Medium | S | UX win |
| R12 | Consolidate compose files to one canonical location | Medium | XS | Reduces confusion |
| R13 | Add benchmarks for the IRInterpreter and the workflow-compiler | Medium | M | Establishes a baseline before optimisations |
| R14 | Add fuzz tests for `domain.ParseManifest` and `domain.evalGuard` | High (security) | M | Both are untrusted-input parsers |
| R15 | Remove "CNCF Sandbox Candidate" badge and replace with a more honest signal | High | XS | One-line change |
| R16 | Audit the CHANGELOG, which lists "Helm chart templates" that do not exist | Medium | XS | Doc-truth |
| R17 | Empty `agents/sdk/src/zynax_sdk/__init__.py` — either implement or remove from docs | High | M (implement) / XS (remove) | Choose deliberately |

---

## 18. Architectural Alternatives

### 18.1 Workflow Execution Model

| Option | Description | Trade-offs |
|---|---|---|
| **A — Current** | Custom IR interpreter inside a Temporal workflow | Maximum portability; custom logic must remain deterministic |
| **B — Native Temporal workflows** | Compile IR to Go code generating a real Temporal workflow | Native Temporal features accessible; loses portability |
| **C — Temporal Search Attributes + Activities** | Each state is an Activity; transitions via Signal | Simpler; less generic |

**Recommendation:** stay on A, fix the determinism bug, replace the guard parser with `cel-go`.

### 18.2 Capability Dispatch

| Option | Description | Trade-offs |
|---|---|---|
| **A — Current** | engine-adapter calls task-broker, polls until terminal | Simple; high RPC pressure; latency floor of 500 ms |
| **B — Temporal async completion** | engine-adapter dispatches with completion token; broker calls back | Lower latency; lower RPC pressure; requires durable token storage |
| **C — Streaming completion** | engine-adapter opens server-streaming RPC; broker emits events | Real-time; broker becomes stateful per-task |

**Recommendation:** B is the right end-state; A is acceptable for v0.

### 18.3 IR Persistence

| Option | Description | Trade-offs |
|---|---|---|
| **A — Current** | In-memory `sync.RWMutex` map in workflow-compiler | Works for one replica; lost on restart |
| **B — Stateless** | Drop persistence; callers store the IR | Simpler; pushes responsibility outward; matches `kubectl apply` |
| **C — Persisted** | etcd or Postgres backing store | Most production-y; violates ADR-008 |

**Recommendation:** B. ADR-008 already points here; the proto's `GetCompiledWorkflow` returning NOT_FOUND is already permitted.

---

## 19. Risk Register

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| Determinism bug in `resolveTemplate` causes workflow panic in production | High (when scaled) | Severe (stuck workflows) | Sort map keys; add a determinism regression test |
| Guard parser fail-open silently advances workflows past intended gates | High | High (correctness) | Replace with cel-go OR fail-closed on parse error |
| Workflow-compiler error mis-reporting hides validation issues | Certain | Medium (UX) | Fix the error mapping |
| api-gateway exposed without auth | Certain on public deployment | Critical | Add bearer-token gate |
| SSE log stream dropped at 30 s | Certain | High (UX) | Fix WriteTimeout |
| BDD "contract tests" mistaken for proof of system correctness | Certain | Medium (engineering culture) | Rename, document, add real E2E |
| Premature CNCF Sandbox claim damages credibility | High | Medium (brand) | Remove badge |

---

## 20. Prioritized Recommendations

### 20.1 Critical (do first)

| # | Recommendation | Benefit | Effort |
|---|---|---|---|
| C1 | Align documentation with reality (update milestone status or implement task-broker) | Trust, truthfulness | XS (docs) — M (code) |
| C2 | Remove the self-applied "CNCF Sandbox Candidate" badge | Removes the most obvious credibility risk | XS |
| C3 | Fix `resolveTemplate` determinism bug (`interpreter.go:204-209`) | Prevents stuck workflows in production | XS |
| C4 | Replace guard parser with `cel-go` OR fail-closed + rename to "Simple Equality Guard" | Closes correctness/security gap | S |
| C5 | Fix the workflow-compiler error contract (return structured `CompilationError` list) | Honours the proto contract | XS |
| C6 | Add bearer-token auth on api-gateway (env-configured shared secret minimum) | Closes the most severe security gap | S |
| C7 | Empty Python SDK: implement it or remove the SDK promise from docs | Truthfulness | XS (remove) / M (implement) |

### 20.2 High (do within 30–90 days)

| # | Recommendation | Benefit | Effort |
|---|---|---|---|
| H1 | Implement a minimal in-memory task-broker | Unblocks end-to-end demo | M |
| H2 | Implement a minimal in-memory agent-registry | Unblocks AgentDef apply path | M |
| H3 | Split probes: `/startupz` (one-shot), `/readyz` (dep-check), `/livez` (deadlock-check) | Production-correct K8s posture | S |
| H4 | Fix SSE WriteTimeout for log streaming | Headline UX works | XS |
| H5 | Wire Prometheus `/metrics` and at least one OTel span per request | Honours observability promise | S |
| H6 | Replace polling-based `Watch` with Temporal `GetWorkflowExecutionHistory` long-poll | Cuts RPC pressure 10×; emits real state-transition events | M |
| H7 | Make workflow-compiler stateless (drop in-memory store) | Trivial horizontal scaling; matches ADR-008 | S |
| H8 | Idempotent `Apply` (manifest-hash → workflow ID) | `kubectl apply`-style UX | S |
| H9 | Audit and update CHANGELOG to match what was actually shipped | Doc-truth | XS |
| H10 | Consolidate to one canonical `infra/docker-compose/` | Reduces confusion | XS |

### 20.3 Medium (do within 90 days)

| # | Recommendation | Benefit | Effort |
|---|---|---|---|
| M1 | Add benchmarks for interpreter and compiler | Future-proofs perf | M |
| M2 | Add fuzz tests for ParseManifest and (future) CEL guard evaluator | Closes input-driven attack surface | M |
| M3 | Add mTLS between platform services | Defense-in-depth | M |
| M4 | Add request ID middleware + structured logging with correlation | Required for production incident response | S |
| M5 | Write a `docs/troubleshooting.md` | Reduces support burden | S |
| M6 | Add cosign signing and SBOM publication to release workflows | SLSA L3 path | M |
| M7 | Decommission half the open governance surface until >5 contributors | Less process theatre | S |
| M8 | Move generated stubs to a separate repo (`zynax-protos`) | Better DX for external consumers | M |
| M9 | Decide YAML library: `gopkg.in/yaml.v3` OR `go.yaml.in/yaml/v3`, not both | Build hygiene | XS |

---

## 21. 30-Day Action Plan

**Goal: turn documentation truth into product truth. Stop the credibility bleed.**

| Day | Action |
|---|---|
| 1 | Remove CNCF badge (C2). Audit README/CHANGELOG/ARCHITECTURE for claims that don't survive `git ls-files`. |
| 2 | Update milestone status: M1 ✓, M2 ✓, M3 ⚠ (partial — no broker), M4 ⚠ (partial — no agent registry). |
| 3 | Fix `resolveTemplate` (C3), add a regression test. |
| 4 | Fix workflow-compiler error contract (C5), update the BDD stub to expect structured errors. |
| 5 | Decision day: replace guard parser with cel-go (recommended) vs. rename to "Simple Equality Guard" + fail-closed (C4). |
| 6–7 | Implement chosen guard fix; add fuzz test. |
| 8 | Split probes (H3). Add baseline metrics (H5). |
| 9 | Fix SSE WriteTimeout (H4). |
| 10 | Add bearer-token middleware on api-gateway (C6). |
| 11–15 | Implement minimal in-memory task-broker (H1). Single replica. Round-robin assignment. |
| 16–17 | Implement minimal in-memory agent-registry (H2). |
| 18 | End-to-end test: `make run-local && zynax apply spec/workflows/examples/code-review.yaml` runs, dispatches to an HTTP adapter capability, completes. |
| 19 | Idempotent Apply (H8). |
| 20 | Consolidate compose files (H10). |
| 21 | Update CHANGELOG with what's actually shipped (H9). |
| 22–25 | Cut **v0.4.0-alpha** with honest release notes. |
| 26 | Decommission half the governance scaffolding (M7). Move SPDD/REASONS docs from "mandatory" to "experimental methodology". |
| 27 | Empty Python SDK decision (C7): implement a minimal agent class OR remove from docs. |
| 28 | Write `docs/troubleshooting.md` with the failure modes you now know exist. |
| 29 | Open 5 deliberately-scoped "good first issue"-style tasks. Write them like a tech lead, not like a step in your own canvas. |
| 30 | Post one external-facing artefact: a blog post, a HN/Lobsters/Reddit submission, or an X/Bluesky thread — proposing the project. **Get the first non-author star.** |

---

## 22. 90-Day Strategic Roadmap

**Goal: prove the differentiation with one credible end-to-end scenario.**

### Days 31–60

1. **Pick one differentiator workflow.** Recommend "code-review with human-in-the-loop and hybrid LangGraph + Temporal execution."
2. Implement enough of the LangGraph engine adapter to run that one workflow on LangGraph.
3. Implement enough of the Temporal interpreter to run the same YAML on Temporal.
4. Prove byte-identical state-transition events from both engines on the same input.
5. Write the demo as a 5-minute video. Publish.
6. Wire Prometheus + OTel for real.
7. Replace polling-based Watch with streaming (H6).

### Days 61–90

1. mTLS between services (M3).
2. Production-style Helm chart for the 4 implemented services.
3. Real audit log (write to NATS JetStream stream + persist).
4. Cosign + SBOM on release (M6).
5. First external contributor PR. Onboard them properly.
6. Cut **v0.4.0** with one demo deployment documented.
7. Quietly file the actual CNCF Landscape entry.

---

## 23. Long-Term Vision (1–3 years)

### Fork A — "The Honest YAML Layer"

Zynax becomes the **best declarative YAML layer for hybrid workflow engines**. It does not compete with Temporal, Argo, or LangGraph. It does the one thing none of them does: hold the workflow definition in a form that can run on any of them. Roadmap: v1.0 in ~12 months, two engines (Temporal + LangGraph) with proven semantic equivalence on a 20-workflow conformance suite. CNCF Sandbox submission at v1.0.

### Fork B — "The Cloud-Native Agent Platform"

Zynax becomes a full agent platform: registry + broker + memory + event bus + multi-engine + multi-tenant + multi-region. This is a 5+ year, multi-engineer, funded effort.

### Recommendation

**Fork A.** The codebase, the contributor base, and the architectural clarity all point there.

---

## 24. Final Verdict

> **Is this architecture fundamentally sound?**

The *concept* is sound. The *engine abstraction* is sound. The *layering* is sound. The *protocol* is sound. The *actual implementation* is ~25% complete and several core paths contain correctness or security bugs that should not pass a senior review (determinism, fail-open guards, no auth, contract violation in the only compiler). With ~3 focused weeks of work on §17 R1–R6 and a brutally honest documentation pass, the system would be a credible v0.3-alpha. As shipped today, the documentation overstates the engineering.

> **What should NOT be changed?**

1. The hexagonal layering — keep it.
2. The proto contract discipline — keep it.
3. The ADR habit — keep it.
4. The minimal-deps choices — keep them.
5. The state-machine-over-DAG decision (ADR-014) — it's the right call.
6. The pluggable-engine port (ADR-015) — it's the project's most valuable abstraction.
7. The Go workspace + per-service go.mod layout — keep it.
8. The change-detection CI pattern — keep it.

> **What should be changed immediately?**

1. The "CNCF Sandbox Candidate" badge.
2. The milestone-status claims.
3. The `resolveTemplate` determinism bug.
4. The workflow-compiler error contract.
5. The guard parser (or its name).
6. The "M3/M4 Complete" message until the task-broker exists.

The skeleton of this project is excellent. The flesh is partly real, partly imagined, partly broken. The gap is closable with weeks of work — but only if the project first stops pretending the gap doesn't exist.

---

*Reviewer's closing note: this project shows the unmistakable fingerprints of a careful, opinionated engineer who has read widely and thought hard about how systems should be built. That is rare and valuable. The advice above is offered in that spirit — not to diminish what's been done, but to point out that the marginal hour is now better spent shipping a small thing that works than scaffolding a large thing that is mostly described.*
