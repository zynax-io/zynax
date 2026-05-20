<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax — Comprehensive Architectural Review

**Repository:** `github.com/zynax-io/zynax`
**Review date:** 2026-05-20
**Reviewer mandate:** Principal Software Engineer / Distinguished Architect / CNCF TOC perspective
**Branch reviewed:** `main` (HEAD: 205 commits, 6 releases, latest `proto-stubs-20260422`)

---

## Executive Summary

Zynax is an ambitious, well-positioned **declarative control plane for AI agent workflows** ("Kubernetes for AI workflows"). Its conceptual model — three-layer separation (Intent / Communication / Execution), engine-agnostic Workflow IR, capability-routed dispatch via a stable `AgentService` gRPC contract — is genuinely strong and consonant with CNCF design philosophy. The engineering hygiene (ADRs, BDD-first, hexagonal layering, ≥90% domain coverage, Renovate, DCO, Go workspaces, AsyncAPI) is well above the median for a v0.3 project.

**The single most important finding is a delivery-vs-narrative gap.** Documentation and CHANGELOG repeatedly assert capabilities (mTLS, SBOM, cosign signing, Helm charts, working CloudEvents publishing, a Python SDK, a working capability dispatch path) that **do not exist in the codebase today**. The maintainer has *acknowledged this* — the M5.A "Truth Pass" epic (#458) is explicitly removing inflated claims (#472 removed the CNCF Sandbox Candidate badge; #473 purged phantom CHANGELOG entries). This self-correction is excellent. But several gaps remain — most importantly the end-to-end dispatch path (`engine-adapter → task-broker → agent`) is **not wired up yet**, despite M3 and M4 being marked "Complete" in the README.

Architecturally, the system is **sound at the contract layer and at the small-scale of each implemented service**, but is **not yet a system** — three of the seven declared platform services (`agent-registry`, `event-bus`, `memory-service`) are zero-LoC stubs. The `IRInterpreterWorkflow` runs in Temporal but dispatches to a `task-broker` that was only merged in M5.C and is not in the local Docker-compose stack. So in production today, every workflow's first action will fail.

| Dimension | Score | Rationale |
|---|---|---|
| **Overall architecture** | **6.5 / 10** | Excellent design, partial execution, integration gap |
| Simplicity | 7.5 / 10 | Clean hexagonal services, lean deps, but 1,325-line `ci.yml` and 63-target Makefile |
| Performance | 4.0 / 10 | Zero benchmarks, zero load tests, unbounded in-memory IR store, bespoke string-based CEL evaluator |
| Security | 4.0 / 10 | Insecure-credentials gRPC everywhere; non-constant-time bearer compare; SECURITY.md asserts controls that don't exist |
| Maintainability | 8.0 / 10 | High test coverage, ADRs, hexagonal layering, AGENTS.md per service |
| Scalability | 4.5 / 10 | Workflow-compiler stateful (unbounded map); no horizontal-scale story documented; no rate limits |
| Product-market fit | 7.0 / 10 | Strong category framing, good Kubernetes analogy, but crowded space (Temporal, Restate, Argo, LangGraph, Dapr Workflows) |
| CNCF alignment | 6.5 / 10 | Apache-2.0, DCO, ADRs, OSSF Scorecard, governance doc — but no SBOM/cosign/mTLS/Helm yet |

### Top 5 Strengths
1. **Contract-first discipline.** 8 protobuf services with explicit invariants in comments; 140+ BDD scenarios written before implementation; bufconn-based in-process testing for parallel, port-free contract testing.
2. **Honest, ADR-driven decision record.** 19 numbered ADRs, governance doc with conflict resolution, CNCF alignment section. Rare for a pre-1.0 project.
3. **Hexagonal layering applied uniformly.** Every implemented service has `internal/{api,domain,infrastructure}`. Domain packages have zero proto or gRPC imports.
4. **Truth-pass culture.** The M5.A epic actively removes inflated marketing claims. This is healthier than most early-stage projects.
5. **Genuine engine abstraction.** The `WorkflowEngine` interface in `services/engine-adapter/internal/domain/engine.go` is a 6-method port that genuinely decouples the IR from Temporal. Adding `ArgoEngine` would be ~500 LoC, not a rewrite.

### Top 5 Weaknesses
1. **The system is not connected end-to-end.** `engine-adapter` dispatches capabilities, but `task-broker` was only just landed (M5.C, in-memory, not in compose) and `agent-registry` doesn't exist. A `zynax apply` of any workflow with actions will fail at the first dispatch.
2. **Documentation overstates security posture.** `SECURITY.md` claims mTLS, SBOM, cosign-signed images. None exist. All inter-service gRPC uses `insecure.NewCredentials()`. Bearer auth uses non-constant-time `!=` comparison.
3. **Stateful workflow-compiler with unbounded in-memory IR store.** `services/workflow-compiler/internal/api/server.go:31` — `map[string]*zynaxv1.WorkflowIR` with no eviction, no max size, no persistence. Issue #466/#490 acknowledges this but is in M6 backlog.
4. **Zero performance engineering.** No benchmarks, no load tests, no profiling targets in Makefile, no documented latency/throughput SLOs for a "control plane" that aspires to CNCF graduation.
5. **Bespoke CEL evaluator and bespoke template engine.** `evalGuard()` in `interpreter.go` is a string-parsing `==`/`!=` matcher with **fail-open** semantics for unrecognized expressions. `resolveTemplate()` is a naive string-replace. Issue #476 (replace with `cel-go`) is open. The fail-open behaviour is a correctness bug masquerading as graceful degradation.

### Top 7 Highest-Priority Recommendations (Critical / High only)

| # | Recommendation | Class | Effort | Risk |
|---|---|---|---|---|
| 1 | **Finish M5.C end-to-end dispatch** (`agent-registry` MVP + `task-broker` wiring in compose + an in-tree adapter that produces a green E2E for `code-review.yaml`) | Critical | M (3–4 wks) | Low |
| 2 | **Replace bespoke CEL with `cel-go`** (#476) and remove fail-open default | Critical | S (1 wk) | Medium |
| 3 | **Make workflow-compiler stateless** (#466/#490): IR persistence moves to an injected store interface; default → no-op or bounded LRU | High | S (3–5 days) | Low |
| 4 | **mTLS or SPIFFE between services** + remove all `insecure.NewCredentials()` from defaults (gate behind `ZYNAX_DEV_INSECURE=1`) | High | M (2 wks) | Low |
| 5 | **Cosign + SBOM + SLSA L2 in release pipeline** (#235, #239, #465, #489) — table stakes for CNCF Sandbox | High | S–M | Low |
| 6 | **OpenTelemetry tracing across all three live services + Prometheus metrics surface for api-gateway and engine-adapter** | High | M (2 wks) | Low |
| 7 | **Constant-time bearer compare + OIDC/JWT path for production** | High | S (2 days for hardening; M for OIDC) | Low |

---

## 1. Product & Market Assessment

### 1.1 Problem statement (as the project sees it)
> "Workflows defined for Temporal cannot run on LangGraph. Without a control plane, every engine requires a different workflow definition."

This is a **real problem** for enterprises adopting multiple agent frameworks. The Kubernetes analogy (Pod spec → AgentDef, Deployment → Workflow, kubelet → Engine Adapter) is intellectually crisp and immediately communicable.

### 1.2 Competitive landscape (May 2026)

| Competitor | Overlap with Zynax | Differentiator |
|---|---|---|
| **Temporal** | Largest overlap. Temporal *is* a workflow engine; Zynax wraps it. | Zynax's pitch only holds if multi-engine is real; today it's Temporal-only. |
| **Dapr Workflows** | CNCF-incubating sibling — declarative bindings, pub/sub, state mgmt. | Dapr is general-purpose; Zynax is AI-agent-specific. |
| **Restate** | Durable execution; competing with Temporal. | Same level — Zynax could plug it in via an adapter. |
| **LangGraph / CrewAI / AutoGen** | Frameworks Zynax wraps as capabilities, not competitors. | Zynax should be careful to remain a *control plane*, not drift into framework territory (ADR-011 enforces this — good). |
| **Argo Workflows** | DAG-based, K8s-native. Listed as planned adapter. | ADR-014's "state machines beat DAGs" argument is defensible but contested by the Argo community. |
| **Numaflow, Kestra, n8n, OpenFunction** | Adjacent control-plane plays. | Different abstraction layer; lower threat. |
| **Kagent (CNCF Sandbox, 2026)** | **Direct competitor.** Newly-accepted CNCF Sandbox project, also positioning as the "control plane for AI agents" on Kubernetes. | This is the most important strategic risk. Zynax must explicitly position against Kagent (engine-agnosticism vs K8s-nativeness, capability routing vs Pod-per-agent). See `2026-05-20-kagent-comparison.md` when available. |

**Strategic risk:** "Control plane for AI agent workflows" is now a *crowded* category. Zynax's unique angle — **declarative + engine-agnostic + adapter-first (no SDK required)** — is genuine but must be ruthlessly defended in messaging. The README's tagline currently overlaps almost word-for-word with Kagent's.

### 1.3 Adoption barriers
1. **Zero stars, zero forks** as of review date. No published Docker images, no published Go module on `pkg.go.dev` (the `gen/go/zynax/v1` import path is referenced but `go install` would fail today).
2. **Temporal dependency.** Many adopters will see "needs a Temporal cluster" and bounce. The local Compose stack is great but a *zero-Temporal* lightweight mode (e.g. an in-process engine) would massively reduce evaluation friction.
3. **YAML-only authoring.** Power users will want imperative composition. There is no Go/Python builder API on the roadmap.
4. **Documentation overhead for the AI-context budget**, REASONS canvas, SPDD, multiple AGENTS.md — fascinating internally, but **alienating** to a casual external contributor who just wants to fix a typo. CONTRIBUTING.md at 416 lines is intimidating.

### 1.4 Product-market fit verdict
**Promising category, premature claim.** The framing is excellent; the implementation needs an end-to-end demo (workflow YAML → Temporal → real capability execution → result) before it can credibly say "product." Time-to-first-working-workflow for a new user is currently *infinite* because `task-broker` and `agent-registry` aren't wired.

**Score: 7.0 / 10** — would be 8.5+ once one working end-to-end demo exists.

---

## 2. Current Architecture Overview

```
            ┌───────────────────────┐
            │  zynax CLI (Go)       │
            │  apply/get/delete     │
            └────────────┬──────────┘
                         │ HTTP/REST (port 7080)
            ┌────────────▼──────────┐
            │  api-gateway          │  ✅ implemented
            │  bearer-token auth    │  ⚠  non-const-time compare
            │  /apply /workflows    │
            └──────┬───────────┬────┘
                  │            │
        gRPC (insecure!)       │
                  │            │
        ┌─────────▼───┐   ┌───▼──────────────┐
        │ workflow-  │   │ engine-       │  ✅ implemented
        │ compiler   │   │ adapter       │
        │ ⚠  in-mem   │   │ Temporal wkr  │
        │   IR store │   │ ⚠  event pub   │
        └────────────┘   │   is a stub   │
                         └──────┬─────────┘
                               │ Activity: DispatchCapability
                               │ gRPC (insecure!)
                               ▼
                         ┌─────────────────┐
                         │ task-broker     │  🟡 partial (M5.C in-mem MVP)
                         │ in-memory repo  │  NOT in compose stack
                         │ round-robin     │
                         └──────┬──────────┘
                               │ FindByCapability gRPC
                               │ ExecuteCapability gRPC (insecure!)
                               ▼
              ┌────────────────────────────┬────────────────┐
              │ agent-registry  │ stub    │ Agents/Adapters │ 🟡 http only
              │ event-bus       │ stub    │ http/llm/git/ci │   (1 of 5)
              │ memory-service  │ stub    │ langgraph       │
              └────────────────────────────┴────────────────┘
```

### 2.1 Repository structure (verified)

```
├── protos/         5 service .protos + cloudevents + event-bus = 8 .protos, ~7k LoC generated Go, full Python stubs
├── spec/           5 JSON Schemas, 1 AsyncAPI, 5 workflow examples
├── services/       7 declared, 4 with go.mod (api-gateway, engine-adapter, task-broker, workflow-compiler), 3 stubs
├── agents/         1 SDK placeholder (3 LoC), 1 http adapter, 1 example feature file
├── cmd/            zynax (CLI) + zynax-ci (validator tool)
├── docs/           19 ADRs, 4 milestone reviews, REASONS canvas docs, SPDD methodology
├── infra/          Dockerfile.tools + 4 compose files; NO helm/ despite README references
└── .github/        12 workflows; ci.yml is 1,325 lines
```

### 2.2 Service-level LoC inventory (verified)

| Service | Code | Test | Test:Code | Status |
|---|---:|---:|---:|---|
| workflow-compiler | 1,390 | 1,855 | 1.33 | ✅ |
| engine-adapter | 1,143 | 1,209 | 1.06 | ✅ |
| api-gateway | 1,047 | 1,071 | 1.02 | ✅ |
| task-broker | 905 | 455 | 0.50 | 🟡 in-memory MVP |
| agent-registry | 0 | 0 | — | ❌ stub (BDD feature files only) |
| event-bus | 0 | 0 | — | ❌ stub |
| memory-service | 0 | 0 | — | ❌ stub |
| cmd/zynax | 1,470 | 1,200 | 0.82 | ✅ |
| cmd/zynax-ci | 854 | 729 | 0.85 | ✅ |
| **Total Go** | **~7,360** | **~14,500** | **1.97** | |

Test-to-code ratio of ~2:1 is excellent. Coverage gate of ≥90% on domain packages is excellent.

---

## 3. Architectural Strengths

### 3.1 Three-layer separation is real, not aspirational
A manual audit of `services/*/internal/domain/*.go` reveals **zero gRPC imports**, **zero proto imports** (except where domain explicitly re-exports types for the api layer). The `engine-adapter` domain package contains `IRInterpreter` whose `Run()` method depends only on two domain interfaces (`ActivityExecutor`, `EventPublisher`). The Temporal SDK appears only under `internal/infrastructure/`. This is **textbook hexagonal architecture** and very rare to see done correctly.

### 3.2 The Workflow IR proto envelope is forward-compatible
`workflow_compiler.proto` keeps the legacy `bytes ir_payload` field 4 alongside structured fields 5–9 (M2 additions). Comment #5 explicitly: *"WorkflowIR fields 1–6 are the M1 envelope. Fields 7–9 are the M2 structured IR. All are additive. Engine adapters SHOULD use structured fields when ir_version is present."* — this is exactly the right disciplined evolution model.

### 3.3 BDD-first with bufconn is genuinely good engineering
`protos/tests/` runs all 140+ scenarios in-process via `testserver.NewBufconnServer`. No port conflicts, no teardown races, parallel-safe. The `GOWORK=off` requirement is documented in ADR-017. Most Go projects half this age have flaky integration tests; Zynax does not.

### 3.4 Idempotent apply via canonical hash (#485)
`ManifestWorkflowID(yaml)` in `apply.go` parses, re-marshals, SHA-256s, takes first 16 hex chars. Resubmitting the same manifest returns the same `run_id`. Re-running a *completed* workflow appends a Unix timestamp suffix. This is genuinely elegant GitOps-friendly behaviour.

### 3.5 Per-service AGENTS.md
Every service directory has its own `AGENTS.md` (300-line cap), inheriting from the root. This is the only AI-agent-onboarding pattern seen in the wild that actually scales — much better than monolithic CONTRIBUTING.md.

### 3.6 Lean dependency graph
api-gateway has **5 direct dependencies**; engine-adapter has **5**; task-broker has **3**. Compare with many Go services that have 50+. Lean dep graphs mean smaller CVE surface, faster builds, and easier vendor licensing review (a CNCF requirement).

### 3.7 Honest engineering culture
The M5.A "Truth Pass" epic (#458) — *deliberately removing inflated claims from CHANGELOG, removing premature CNCF badges, debating whether to keep the Python SDK at all* — is the kind of self-correction that distinguishes serious open-source projects from vanity ones.

---

## 4. Architectural Weaknesses

### 4.1 The system is not connected end-to-end (Critical)
As of HEAD:
- `engine-adapter`'s `DispatchCapabilityActivity` calls into `task-broker` via gRPC (the existing infrastructure-layer code is correct).
- But `services/task-broker/` was only added in PRs #520, #522, #523 (M5.C), is **in-memory**, has **no Dockerfile in the compose stack**, and **no `agent-registry` to call to find agents**.
- `services/agent-registry/` is 0 LoC.
- `services/event-bus/` is 0 LoC.
- `services/memory-service/` is 0 LoC.

Yet the README says M3 and M4 are "**Complete**."

Execution flow today:
1. `zynax apply code-review.yaml` → api-gateway ✅
2. api-gateway → workflow-compiler → WorkflowIR ✅
3. WorkflowIR → engine-adapter → Temporal ✅
4. Temporal → `DispatchCapabilityActivity` → task-broker (gRPC) ❌ — task-broker not in compose
5. Even if it were: no `agent-registry` to look up agents ❌

This is **not just an MVP gap** — it is a credibility issue. Tracked by M5.C epic #460.

### 4.2 Security posture vs documentation gap (High)
| Asserted in SECURITY.md | Reality |
|---|---|
| "mTLS between all services" | `grpc.WithTransportCredentials(insecure.NewCredentials())` everywhere |
| "SBOM per release" | No SBOM generation in CI (issue #235 open) |
| "cosign-signed images" | No cosign step in CI (issue #239, #465, #489 open) |
| "Non-root containers" | ✅ verified, all Dockerfiles do `USER zynax` |
| "trivy CVE scanning in CI" | ✅ `.trivyignore` exists; trivy gate added in M5.F |

Beyond documentation: `requireBearer()` in `services/api-gateway/internal/api/auth.go` uses non-constant-time comparison. No `ReadHeaderTimeout` on HTTP server (Slowloris risk). No rate limiting on `POST /api/v1/apply`.

### 4.3 Workflow-compiler is stateful and unbounded (High)
`services/workflow-compiler/internal/api/server.go`:
```go
type Server struct {
    mu    sync.RWMutex
    store map[string]*zynaxv1.WorkflowIR  // ❌ unbounded, in-process
}
```
No eviction, no TTL, no LRU bound, no persistence. Over weeks of operation this map grows without bound. Prevents horizontal scaling. The proto contract comments promise *30 days of retention* but the implementation provides *unbounded retention or no retention if the pod restarts*. Tracked by #466/#490 (currently M6 backlog — should be promoted to M5).

### 4.4 Bespoke CEL evaluator with fail-open semantics (High)
`services/engine-adapter/internal/domain/interpreter.go`:
```go
func evalGuard(expr string, ctx map[string]string) bool {
    // ...string-parsing matcher...
    return true // fail-open for unrecognised expressions
}
```
Three issues: (1) fail-open is wrong — unrecognized expressions silently fire every transition; (2) operator precedence edge cases; (3) no AND/OR, parens, numeric, list, or null support. Tracked by #476/#538. **Replacement with `cel-go` is a 1-day job.**

### 4.5 Bespoke template engine (Medium)
`resolveTemplate()` does deterministic-key string substitution with no escape mechanism, no type coercion, no conditional expressions. Either commit to `text/template` or explicitly cap and document the contract.

### 4.6 CloudEvents publishing is a phantom feature (High)
`PublishLifecycleEventActivity` only emits a debug-level log entry and returns nil. The README claims working CloudEvents lifecycle publishing. This is fine *as a stub* if marked as such — M4 should be relabeled accordingly.

### 4.7 Operational complexity overstated (Medium)
63 Makefile targets, 1,325-line `ci.yml`, 12 separate GitHub Actions workflows, a bespoke `zynax-ci` binary. The docker-tools-image pattern is great but pulling a 1+ GB image for a docs PR is wasteful. Tracked by M5.F (#542) and the DRY/KISS refactor (#555).

### 4.8 Stub services as zero-LoC ghosts (Medium)
`services/agent-registry/`, `services/event-bus/`, `services/memory-service/` have `AGENTS.md` and feature files but no Go code, no `go.mod`, no `cmd/`. They appear in `SERVICE_LIST`, architecture diagrams, and docker-compose comments. **Recommendation: add a minimal `go.mod + cmd/main.go` that returns `Unimplemented` for every RPC** so the system is visibly partial rather than invisibly broken.

### 4.9 No horizontal-scale story (Medium)
workflow-compiler is stateful (#4.3). task-broker stores tasks in memory. agent-registry will need a real backing store. No leader-election story. No multi-tenancy isolation (namespace field is cosmetic). Should be addressed in ADR-021.

### 4.10 Observability mostly absent (High)
Only workflow-compiler has Prometheus + OTel hooks. api-gateway and engine-adapter have nothing. A control plane without metrics is undeployable in production. Tracked by M7.A (#467/#491). Should be promoted.

---

## 5. Code Quality Assessment

### 5.1 Readability — 8.5 / 10
Descriptive names, function-level comments are high-signal. The `//nolint:funlen // four sequential validation phases` in `manifest.go` is exemplary — the developer didn't just suppress the linter, they explained *why*.

### 5.2 Duplication — 6.5 / 10
- gRPC interceptor boilerplate copy-pasted across services. Extract to `pkg/grpcmw/`.
- gRPC dial code repeats in every service. Extract a `pkg/grpcclient/` with TLS-first dial.
- Error mapping follows the same pattern in each consumer. Extract `pkg/grpcerr/Map()`.

**Note:** A shared `pkg/` internal Go module would not violate ADR-008 (no-shared-database rule).

### 5.3 Anti-patterns
- **Background context for long-running goroutines** in `task-broker/service.go`: `go func() { s.executeAsync(context.Background(), ...) }()` — loses parent cancellation, distributed-tracing context, and request-ID propagation.
- **Fail-open guard evaluation** (§4.4).
- **Best-effort event publish silently swallowing errors** in `temporal_workflow.go` — now logged as warn in M5.B (#483).

### 5.4 Top refactoring opportunities (by impact/effort)
1. Extract `pkg/grpcmw/` for shared interceptors (50 LoC saved).
2. Replace `evalGuard` with `cel-go` (80 LoC removed, correctness bug fixed).
3. Extract `WorkflowEngine` registry pattern for multi-engine without `main.go` churn.
4. Split 1,325-line `ci.yml` into reusable workflow files.
5. Inject `Store` port into workflow-compiler (stateless service, horizontal scale unlocked).

---

## 6. Performance Analysis

### 6.1 Nothing has been measured
No `func Benchmark...`, no `*.bench`, no `pprof` Makefile target, no `go test -bench`, no load-test harness (`k6`, `vegeta`, `ghz`), no documented latency or throughput SLOs.

### 6.2 Likely bottlenecks (by code inspection)
1. **workflow-compiler IR storage under write lock** — single-replica throughput-bound.
2. **`resolveTemplate` linear scan + sort on every action dispatch** — compile the template once.
3. **task-broker `memoryRepo.List` full-table scan** — fine for MVP; production needs an index.
4. **gRPC streaming `Watch` polling** — `TemporalEngine.Watch` polls `DescribeWorkflowExecution` every 2s. 1,000 concurrent SSE clients = 500 RPS into Temporal Frontend. Use Temporal's `QueryWorkflow` or signal-based pushdown. Tracked by #492.

### 6.3 Memory & GC
workflow-compiler keeps every compiled IR forever. For a 10 KB IR × 100k workflows = 1 GB of heap that never frees. task-broker keeps every task forever. Both will OOM in production within days under moderate load.

**Performance score: 4.0 / 10** — the design is performance-friendly; measurement and SLOs are absent.

---

## 7. Security Review

| Finding | Severity | File | Recommendation |
|---|---|---|---|
| All inter-service gRPC uses `insecure.NewCredentials()` | **Critical** | `services/*/internal/infrastructure/clients.go` | TLS-by-default; gate insecure behind `ZYNAX_DEV_INSECURE=1` |
| Bearer-token compare is not constant-time | **High** | `services/api-gateway/internal/api/auth.go:18` | `crypto/subtle.ConstantTimeCompare` |
| Single static bearer token, no rotation, no scopes | **High** | `services/api-gateway/internal/api/auth.go` | OIDC/JWT path; ADR for token format |
| SECURITY.md asserts mTLS/SBOM/cosign that don't exist | **High** | `SECURITY.md` | Truth-pass: rewrite to match reality |
| No SBOM generation in CI | **High** | `.github/workflows/` | `anchore/sbom-action` (syft SPDX) |
| No image signing | **High** | release workflows | `cosign sign --keyless` (tracked by #557/#556) |
| No `ReadHeaderTimeout` on api-gateway HTTP server | Medium | `services/api-gateway/cmd/api-gateway/main.go` | Set to 5s |
| No global request-body size limit | Medium | `services/api-gateway/internal/api/handler.go` | Wrap all routes with `http.MaxBytesReader` |
| No rate limiting | Medium | api-gateway | `golang.org/x/time/rate` per-IP or per-token |
| `requireBearer` bypassed when `ZYNAX_API_KEY=""` | Low | `auth.go` | Refuse to start in production mode without a key |

### 7.1 Supply-chain security
- ✅ Apache-2.0 with SPDX headers
- ✅ Renovate with automerge for patch updates
- ✅ OSSF Scorecard badge
- ❌ No `govulncheck` in required CI gates (only `make audit`)
- ❌ No SLSA provenance
- ❌ No `goreleaser` or reproducible builds attested (tracked by #556–#566)

### 7.2 Container hardening
- ✅ Non-root user (`USER zynax`)
- ✅ Multi-stage build, distroless-ish (Alpine final)
- ✅ `CGO_ENABLED=0` and `-trimpath`
- ❌ No `HEALTHCHECK` directive in Dockerfiles
- ❌ No `read-only` rootfs hint in compose
- ❌ Consider `gcr.io/distroless/static-debian12:nonroot` over Alpine

**Security score: 4.0 / 10** — the actual posture is in line with a typical pre-1.0 Go project; the gap between claimed and actual posture is the primary deduction.

---

## 8. Scalability Review

### 8.1 Horizontal scalability barriers
1. workflow-compiler IR store (#4.3).
2. task-broker memory repo.
3. agent-registry (doesn't exist yet, but heartbeat tracking is intrinsically stateful).

### 8.2 Resource isolation
No multi-tenancy. Every workflow runs in the same Temporal namespace ("default"). The proto's `namespace` field exists everywhere but is functionally cosmetic — no isolation is enforced.

### 8.3 Capacity limits
**None documented.** No max workflow size, max states, max actions, max payload. Operators cannot capacity-plan.

**Scalability score: 4.5 / 10**

---

## 9. Reliability Review

### 9.1 Failure modes
- **workflow-compiler restart:** all stored IRs lost. `GetCompiledWorkflow` returns NOT_FOUND for previously compiled workflows — violates proto contract advertising 30-day retention.
- **task-broker restart:** all in-flight tasks lost.
- **Temporal unavailable:** returns `Unavailable` codes → HTTP 503 ✅.

### 9.2 Timeouts & retries
- No deadlines on outgoing gRPC calls in `clients.go`. `CompileWorkflow` has no `context.WithTimeout`.
- No explicit `RetryPolicy` on Temporal Activities. Temporal default = exponential backoff with no max attempts — infinite retry on permanent failures (capability not found).
- No per-RPC timeout on `stream.Recv()` in task-broker. A stuck agent holds the stream open indefinitely.

### 9.3 Circuit breakers
None. No "stop dispatching to this agent for 30s" mechanism.

**Reliability score: 6.0 / 10** — Temporal carries most of the load; reliability is fragile where Temporal isn't involved.

---

## 10. Dependency Analysis

**Total direct production dependencies across all Go services: ~25.** Exceptional. Argo Workflows has 80+, Kubernetes has 200+.

**Removal/replacement opportunities:**
- Replace bespoke CEL with `cel-go` (+1 dep, removes ~80 LoC of buggy code).
- Replace bespoke template engine with `text/template` (stdlib, +0 deps).
- The `gogo/protobuf` indirect dep (from Temporal) is unmaintained; track Temporal SDK migration.

---

## 11. Testing Assessment

### 11.1 Strengths
- 140+ BDD scenarios at the contract layer. Parallel-safe via bufconn.
- ≥90% domain coverage gate enforced in CI.
- Test-to-code ratio ~2:1.

### 11.2 Gaps
- **Zero benchmarks** (`func Benchmark...`).
- **Zero fuzz tests** (`func Fuzz...`). YAML parser, CEL evaluator, template substituter are prime targets.
- **Zero load/scale tests.**
- **No integration test exercising the full E2E path** — because the path isn't connected (§4.1).
- **No mutation testing** (`gremlins`).

**Testing score: 7.5 / 10** — coverage discipline excellent; breadth narrow.

---

## 12. CI/CD Assessment

### 12.1 Strengths
- 12 workflows, each scoped. Required status checks documented.
- Path-based skipping avoids running Go tests on docs PRs.
- Renovate weekly automerge for patch updates.
- AI-context budget workflow — novel and useful.

### 12.2 Gaps
- No SBOM, no image signing, no SLSA provenance (tracked by M5.F.R #556).
- No versioned release has ever been cut. Every install URL returns HTTP 404 (tracked by #557–#558).
- task-broker excluded from service-release matrix (#559).
- http-adapter has no release pipeline (#560).
- `ci.yml` at 1,325 lines — split into composite/reusable workflows (#555).
- `govulncheck`/`bandit` not in required CI gates.

**CI/CD score: 6.5 / 10**

---

## 13. Documentation Assessment

### 13.1 Strengths
- ARCHITECTURE.md (509 lines) is excellent — explains *why*, not just *what*.
- 19 ADRs. GOVERNANCE.md (451 lines) is thorough.
- Per-service AGENTS.md.

### 13.2 Gaps
- **No quickstart producing a successful E2E workflow** — first dispatch will fail.
- **SECURITY.md asserts non-existent controls** — highest-priority doc fix.
- **No API reference generated from protos** (`protoc-gen-doc`).
- **No upgrade/migration guide** between versions.
- **No CODEOWNERS file** at repo root.

**Documentation score: 7.5 / 10**

---

## 14. CNCF Ecosystem Fit

### 14.1 What aligns well
Apache-2.0 ✅, DCO ✅, open governance ✅, gRPC + Protobuf ✅, OCI containers ✅, OSSF Scorecard ✅.

### 14.2 Gaps for CNCF Sandbox

| CNCF Sandbox criterion | Status |
|---|---|
| Apache-2.0 license | ✅ |
| Code of Conduct | ✅ |
| Public roadmap | ✅ ROADMAP.md |
| ADRs / RFCs | ✅ 19 ADRs |
| OWNERS/MAINTAINERS file with ≥2 maintainers from different orgs | ❌ Single maintainer |
| At least one external user/adopter | ❌ (0 stars, 0 forks) |
| Public mailing list / Slack / forum | ❌ |
| Trademark policy | ❌ Not present |
| SBOM / cosign / SLSA | ❌ Tracked by M5.F.R + M6.C |
| CNCF TOC sponsor | ❌ Not yet recruited |

**CNCF score: 6.5 / 10** — structural alignment strong; community traction is the blocking gap.

---

## 15. Prioritized Recommendations

### Critical (next 30 days)

**C1. Close the end-to-end dispatch gap (M5.C)**
- Land `agent-registry` MVP (#480) with in-memory persistence.
- Wire `task-broker` + `agent-registry` into `make run-local` (#481).
- First green E2E test: `code-review.yaml` runs through one capability dispatch.
- **Done when:** `make run-local && zynax apply spec/workflows/examples/code-review.yaml` produces observable state transitions and (mock-data) capability execution.

**C2. Replace bespoke CEL evaluator with cel-go (#476 → #538)**
- Correctness; remove fail-open default; gain `&&`, `||`, parens, numeric, list, regex.
- 3–5 days including test migration.

**C3. Truth-pass on SECURITY.md and README**
- Remove mTLS/SBOM/cosign claims from SECURITY.md.
- Add per-service status table to README (✅ / 🟡 / ❌).
- 4 hours of work.

### High (next 90 days)

**H1. Make workflow-compiler stateless (#466 — promote from M6 to M5)**
**H2. Add Prometheus `/metrics` + OTel tracing to api-gateway and engine-adapter (#491)**
**H3. TLS-by-default for inter-service gRPC; insecure behind `ZYNAX_DEV_INSECURE=1` (#488)**
**H4. SBOM + SLSA provenance + cosign signing (#235, #239, #465, #489, #556 prerequisite)**
**H5. Constant-time bearer compare + ReadHeaderTimeout + request-size middleware (hardening)**
**H6. Persistent task-broker and agent-registry backing stores (Postgres or BoltDB)**
**H7. ADR-020 for zero-trust intra-service security plan (#240)**
**H8. ADR-021 for horizontal scale + multi-tenancy plan**
**H9. Unimplemented gRPC skeletons for agent-registry, event-bus, memory-service (make system visibly partial)**

---

## 16. Unplanned-Gap Analysis

Gaps not yet filed as issues (as of 2026-05-20):

| # | Gap | Severity | Suggested issue |
|---|---|---|---|
| G1 | Bearer-token compare is not constant-time | High | `fix(api-gateway): constant-time bearer-token compare` |
| G2 | No `ReadHeaderTimeout` (Slowloris risk) | Medium | `fix(api-gateway): set ReadHeaderTimeout and enforce MaxBytesReader` |
| G3 | No rate limiting on `POST /apply` | Medium | `feat(api-gateway): per-IP token-bucket rate limit` |
| G4 | No `RetryPolicy` on Temporal Activities | Medium | `fix(engine-adapter): explicit RetryPolicy for DispatchCapabilityActivity` |
| G5 | Polling Watch Temporal load (→ #492) | Medium | Already tracked |
| G6 | `resolveTemplate` bespoke + unsanitised | Low | `refactor(engine-adapter): replace resolveTemplate with text/template` |
| G7 | `mergePayload` silently drops non-strings | Low | `fix(engine-adapter): mergePayload must handle all JSON scalar types` |
| G8 | `Action.Output` parsed but never consumed | Low | `fix(workflow-compiler): implement Action.Output mapping or remove` |
| G9 | No CODEOWNERS file | Low | `chore: add CODEOWNERS for area-label review routing` |
| G10 | workflow-compiler retention contract violated | Medium | `fix(workflow-compiler): enforce retention TTL or correct contract docs` |
| G11 | No benchmarks (→ #493, M7.C) | Medium | Already tracked |
| G12 | No fuzz tests (→ #539 partial) | Medium | Expand #539 or new issue for ParseManifest fuzzing |
| G13 | No load test | Medium | Already tracked in M7 |
| G14 | ManifestWorkflowID collision domain undocumented | Low | `docs/adr: ADR on collision domain for 64-bit workflow ID truncation` |
| G15 | YAML canonicalization may differ between yaml.v3 patches | Low | `fix(workflow-compiler): pin or replace canonicaliseYAML` |
| G16 | Background-context goroutines lose request-ID | Medium | `fix(task-broker): propagate request-ID via derived ctx in executeAsync` |
| G17 | Stub services in SERVICE_LIST despite zero code | Low | `chore: remove or Unimplemented-skeleton stub services from SERVICE_LIST` |
| G18 | No public meeting cadence or community channel | Medium | Tracked in M8.A (#470) |
| G19 | No competitive positioning vs Kagent/Dapr | **High** | `docs: Zynax vs Kagent vs Dapr Workflows positioning page` |
| G20 | No `pkg.go.dev` published Go module reference | Medium | `docs: verify and document Go module consumption path` |
| G21 | Python SDK claims v0.1.0 but is a 3-line placeholder | High | Tracked by #474 — ensure version placeholder is removed |
| G22 | Summarizer example only has a feature file | Low | `chore: implement or remove summarizer example agent` |
| G23 | Phantom researcher/calculator agents in AGENT_LIST | Low | `chore: remove phantom agents from AGENT_LIST` |
| G24 | Local Docker Compose omits task-broker, agent-registry, event-bus, memory-service | High | Tracked by #481 — ensure E2E exit criteria in M5.C |

---

## 17. Architectural Alternatives

### 17.1 Engine dispatch: single-engine env var (current) vs multi-engine per-workflow
**Recommendation:** Stay with Option A (single engine via env var) through v1.0. Add per-workflow `engine_hint` routing in v1.x when a second engine actually ships.

### 17.2 Workflow IR transport: bytes + structured (current) vs pure structured
**Recommendation:** Remove `ir_payload` field 4 by v1.0. Update ADR-012.

### 17.3 Agent dispatch: push (current planned) vs pull (work-queue) vs event-driven
**Recommendation:** Push for v0.x. Add pull mode for LLM/Bedrock adapters (which don't accept inbound connections) as alternate mode — warrants a new ADR.

### 17.4 Multi-tenancy: none (current) vs namespace-isolated vs per-tenant clusters
**Recommendation:** Document operating model as "one Zynax cluster per Kubernetes namespace per tenant" until v1.x.

---

## 18. Risk Register

| ID | Risk | P | I | Mitigation |
|---|---|---|---|---|
| R1 | M5.C dispatch slips again, blocking external adoption | High | Critical | Promote #460 to highest priority; release minimal agent-registry |
| R2 | CNCF Sandbox rejected due to single-maintainer | High | High | Recruit external maintainers NOW via CNCF Slack, hackathons |
| R3 | Security issue in `insecure.NewCredentials()` once project gains adopters | Medium | Critical | TLS-first dial helper; gate insecure behind dev-flag |
| R4 | Production OOM from unbounded IR store | Medium | High | Fix #466 in M5, not M6 |
| R5 | Bespoke CEL guard silently miscalculates | Medium | High | Replace with cel-go (#476/#538) |
| R6 | Kagent absorbs the "AI control plane" category | Medium | Critical | Sharpen positioning vs Kagent in README; publish comparison doc (#G19) |
| R7 | Temporal license change forces rewrite | Low | High | Engine abstraction protects you — but ship ArgoEngine/LangGraphEngine |
| R8 | Documentation overstates → reputational damage | High | Medium | Continue M5.A truth pass; SECURITY.md fix (#C3) |
| R9 | Release pipeline never triggered → all install URLs 404 | High | High | Fix with M5.F.R (#556–#558) immediately |

---

## 19. 30-Day Action Plan

| Week | Theme | Deliverables |
|---|---|---|
| W1 | **CI unblock + truth pass** | Fix release race condition (#557), cut v0.4.0 (#558), fix SECURITY.md (#C3), add README status table, constant-time bearer (#G1), ReadHeaderTimeout (#G2) |
| W2 | **Engine correctness** | Replace evalGuard with cel-go (#538), add RetryPolicy (#G4), fix mergePayload (#G7), ADR for activity retry policy |
| W3 | **Capability dispatch E2E** | Land agent-registry MVP (#526→#527→#528), wire compose (#481), first green E2E test with summarizer adapter |
| W4 | **Observability + supply chain** | Add metrics/traces to api-gateway + engine-adapter (#491), SBOM (#235) and cosign (#239) in release, publish blog post with asciinema E2E demo |

**Exit criteria:**
1. `make run-local && zynax apply spec/workflows/examples/code-review.yaml` → real execution end-to-end.
2. SECURITY.md matches shipped reality.
3. All inter-service gRPC TLS-by-default (insecure only behind dev flag).
4. v0.4.0 tag pushed with at least one downloadable artifact.

---

## 20. Final Verdict

### What should NOT be changed?
- **The three-layer separation.** It's the most valuable architectural idea in the project.
- **The hexagonal `internal/{api,domain,infrastructure}` structure.** Exemplary and rare.
- **The proto-first / BDD-first discipline.** Don't drop it under deadline pressure.
- **The Apache-2.0 + DCO licensing posture.** Vendor-neutral, CNCF-friendly.
- **The ADR culture.** Keep doing this at ADR-019+.
- **The `WorkflowEngine` interface.** Six methods, all necessary, all well-named.
- **Per-service AGENTS.md.** Genuinely good engineering management.

### Shortest path to excellence
1. Truth-pass SECURITY.md and README (C3).
2. Close M5.C dispatch gap with a working E2E demo (C1).
3. Replace the bespoke CEL evaluator with cel-go (C2).
4. Add basic observability (metrics + tracing) to all live services (H2).
5. Fix the release pipeline and cut v0.4.0 (M5.F.R #556–#558).

These five unlock the next 6 months of growth.

---

## Appendix A — Score Card

| Dimension | Score | Evidence |
|---|---:|---|
| Architectural soundness | 7.5 / 10 | Three-layer model, hexagonal services, real proto contracts |
| Simplicity | 7.5 / 10 | Lean deps, but 1,325-line ci.yml, 63 Makefile targets |
| Performance engineering | 4.0 / 10 | No benchmarks, no SLOs, unbounded in-memory state |
| Security | 4.0 / 10 | insecure-by-default gRPC; SECURITY.md inflated |
| Maintainability | 8.0 / 10 | High coverage, ADRs, AGENTS.md, hexagonal |
| Scalability | 4.5 / 10 | Stateful compiler, no horizontal-scale story |
| Reliability | 6.0 / 10 | Temporal covers a lot; gaps elsewhere |
| Testing | 7.5 / 10 | 2:1 test:code, BDD-first, but no benchmarks/fuzz/load |
| CI/CD | 6.5 / 10 | Comprehensive but oversized; missing SBOM/cosign/releases |
| Documentation | 7.5 / 10 | ARCHITECTURE.md, 19 ADRs, governance — but user-facing thinner |
| CNCF alignment | 6.5 / 10 | Apache-2.0/DCO/governance ✅; community traction ❌ |
| Product-market fit | 7.0 / 10 | Strong category, premature claims, crowded space |
| Open-source governance | 7.0 / 10 | GOVERNANCE.md excellent; needs second maintainer + CODEOWNERS |
| **Overall** | **6.5 / 10** | Excellent design, partial execution, integration gap |

---

## Appendix B — Key File References

- `services/engine-adapter/internal/domain/interpreter.go` — `evalGuard` (fail-open), `resolveTemplate`, `Run`
- `services/engine-adapter/internal/infrastructure/temporal.go` — polling Watch, no RetryPolicy
- `services/engine-adapter/internal/infrastructure/activities.go` — `PublishLifecycleEventActivity` stub
- `services/api-gateway/internal/api/auth.go` — non-constant-time bearer compare
- `services/api-gateway/internal/api/handler.go` — `maxBodyBytes` only in helper
- `services/api-gateway/internal/infrastructure/clients.go` — `insecure.NewCredentials()` everywhere
- `services/api-gateway/internal/domain/apply.go` — `ManifestWorkflowID` (64-bit truncation)
- `services/workflow-compiler/internal/api/server.go:31` — unbounded `map[string]*WorkflowIR`
- `services/task-broker/internal/domain/service.go` — `context.Background()` in async goroutine
- `services/task-broker/internal/infrastructure/memory_repo.go` — in-memory repo
- `agents/sdk/src/zynax_sdk/__init__.py` — 3-line SDK placeholder
- `SECURITY.md` — claims mTLS/SBOM/cosign that don't exist
- `infra/docker-compose/docker-compose.yml` — omits agent-registry, task-broker, event-bus, memory-service
- `.github/workflows/ci.yml` — 1,325 lines

Issue cross-references: #146, #173, #205, #228–#229, #232, #235, #239–#246, #248, #358, #373–#377, #381–#384, #400–#418, #442, #445, #458–#466, #472–#492, #520, #522–#523, #526–#532, #535–#540, #542–#566.

---

*Review commissioned 2026-05-20. Reviewer: Principal Software Engineer / Distinguished Architect / CNCF TOC perspective.*
