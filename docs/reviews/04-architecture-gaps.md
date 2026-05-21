<!-- SPDX-License-Identifier: Apache-2.0 -->

# 04 — Architecture Gap Analysis

**Date:** 2026-05-21  
**Branch:** docs/architecture-overhaul-m5  
**Source:** 2026-05-20 principal architect review (`docs/architecture/2026-05-20-principal-architect-review.md`)  
**Purpose:** Phase 2 artifact — verify each finding in the review against HEAD,
extend with framework-lens analysis (Fowler/Richardson/DDD), and rank all gaps.

> **Reading guide:** Each review finding (G1-G24, H1-H9, C1-C3, R1-R9) is verified
> below. Items marked ✅ are fixed; 🟡 partially addressed; ❌ still open.

---

## Part A: Verification of Review Findings Against HEAD (2026-05-21)

### Critical Findings

| Finding | Review says | HEAD state | Verified |
|---|---|---|---|
| **C1** End-to-end dispatch not wired | task-broker in-memory, not in compose; agent-registry 0 LoC | ❌ Still open — #526→#528→#481 pending | Confirmed |
| **C2** Bespoke CEL fail-open → cel-go | evalGuard() string-matching, fail-open | ✅ **Fixed** — #538 #539 #540 merged; cel-go integrated, fail-closed | Confirmed |
| **C3** Truth-pass SECURITY.md + README | SECURITY.md claimed mTLS/SBOM/cosign | ✅ **Fixed** — truth pass 2026-05-20; SECURITY.md rewritten; #579 (README status table) open | Mostly done |

### High-Priority Gaps (G1-G24, H1-H9 verified)

| # | Gap | Status | Issue | Milestone |
|---|---|---|---|---|
| G1 | Bearer-token non-constant-time compare | ❌ Open — `r.Header.Get("Authorization") != want` in auth.go | #567 | M5 |
| G2 | No `ReadHeaderTimeout` (Slowloris) | ❌ Open | #568 | M5 |
| G3 | No rate limiting on POST /apply | ❌ Open | #580 | M6 |
| G4 | No RetryPolicy on Temporal Activities | ❌ Open | #569 | M5 |
| G5 | Watch polling under load (500 RPS at 1k clients) | ❌ Open | #492 | M7 |
| G6 | resolveTemplate bespoke + unsanitized | ❌ Open | #584 | M6 |
| G7 | mergePayload silently drops non-strings | ❌ Open | #571 | M5 |
| G8 | Action.Output parsed but never consumed | ❌ Open | #581 | M6 |
| G9 | No CODEOWNERS | ✅ **Fixed** — file added | ~~#573~~ | — |
| G10 | workflow-compiler retention contract violated | ❌ Open — unbounded map | #572 (doc) + #466 (fix) | M5 |
| G11 | No benchmarks | ❌ Open | #493 | M7 |
| G12 | No fuzz tests | 🟡 Partial — fuzz seed for CEL guard added in #539 | #539 + expand | M5/M7 |
| G13 | No load tests | ❌ Open | M7 backlog | M7 |
| G14/G15 | ManifestWorkflowID collision domain undocumented | ❌ Open | #583 | M6 |
| G16 | Background-context goroutines in task-broker lose request-ID | ❌ Open | #570 | M5 |
| G17 | Stub services in SERVICE_LIST despite 0 LoC | ❌ Open | #574 | M5 |
| G18 | No public meeting cadence / community channel | ❌ Open | #470 | M8 |
| G19 | No competitive positioning vs Kagent/Dapr | ❌ Open | #575 | M5 |
| G20 | No pkg.go.dev published Go module reference | ❌ Open | #582 | M6 |
| G21 | Python SDK claims v0.1.0 but was 3-line placeholder | ✅ **Fixed** — full Agent base class in #535 | #474 | — |
| G22 | Summarizer example only has feature file | ❌ Open | #576 | M5 |
| G23 | Phantom researcher/calculator agents in AGENT_LIST | ❌ Open | #577 | M5 |
| G24 | Compose omits task-broker + agent-registry | ❌ Open | #481 | M5 |
| H1 | Make workflow-compiler stateless (#466) | ❌ Open | #466 | M5 (promoted) |
| H2 | Add OTel + Prometheus to api-gateway + engine-adapter | ❌ Open | #491 | M5/M6 |
| H3 | TLS-by-default for inter-service gRPC | ❌ Open | #488 (ADR-020) | M6 |
| H4 | SBOM + SLSA provenance + cosign | ❌ Open | #235 #239 #465 #489 | M6 |
| H5 | Constant-time bearer + ReadHeaderTimeout + request-size | ❌ Open (G1+G2 cover this) | #567 #568 | M5 |
| H6 | Persistent task-broker + agent-registry stores | ❌ Open | M6 backlog | M6 |
| H7 | ADR-020 zero-trust security | ❌ Open | #240 | M6 |
| H8 | ADR-021 horizontal scale + multi-tenancy | ❌ Open | #578 | M6 |
| H9 | Unimplemented gRPC skeletons for stub services | ❌ Open | #574 | M5 |

### Risk Register Verification

| Risk | P | I | Status |
|---|---|---|---|
| R1: M5.C dispatch slips again | High | Critical | Ongoing — critical path still pending |
| R2: CNCF Sandbox rejected (single maintainer) | High | High | Open — needs community recruitment |
| R3: insecure.NewCredentials() exploited by adopters | Medium | Critical | Open — no TLS yet |
| R4: Production OOM from unbounded IR store | Medium | High | Open — #466 pending (should be M5) |
| R5: Bespoke CEL silently miscalculates | Medium | High | ✅ **Fixed** — cel-go (#538) |
| R6: Kagent absorbs "AI control plane" category | Medium | Critical | Open — positioning doc #575 pending |
| R7: Temporal license change forces rewrite | Low | High | Mitigated — WorkflowEngine interface |
| R8: Docs overstate → reputational damage | High | Medium | 🟡 Mostly fixed — SECURITY.md done; README status table #579 pending |
| R9: Release pipeline never triggered → 404s | High | High | 🟡 Partial — pipeline fixed; tag push pending |

---

## Part B: Extended Gap Analysis (Framework Lenses)

### B1 — Event-Driven Pattern Classification (Fowler taxonomy)

The review raises CloudEvents publishing as a stub. Here is the complete Fowler classification
for every event/async flow in Zynax:

| Flow | Pattern | Assessment |
|---|---|---|
| `zynax.workflow.state.entered/exited/completed/failed` CloudEvents | **Event Notification** | *Intended* design; currently a log stub only. Events carry minimal data (workflow ID, state name, timestamp) — consumers must call back to query state. Correct pattern for control-plane events. |
| NATS JetStream channels (11 in AsyncAPI spec) | **Event Notification** + **Event-Carried State Transfer** | Notification for lifecycle events; ECST for task events that carry result payloads. AsyncAPI spec defines this correctly. NATS bus is 0 LoC today. |
| Temporal activity history | **Event Sourcing** (internal to Temporal) | Temporal maintains a full durable event log that enables workflow replay. This is Temporal's native event-sourcing model. Zynax inherits this by running `IRInterpreterWorkflow` in Temporal. |
| task-broker → agent ExecuteCapability | **Command** (not an event) | This is a direct gRPC call, correctly modeled as a command. No event pattern needed here. |
| CloudEvents `task.completed` | **Event-Carried State Transfer** | Task result events should carry the result payload so consumers don't need to re-query. Currently a stub. |

**Key finding:** The intended design is **Event Notification** for workflow lifecycle, **ECST** for task results. This is the *right* choice for a control plane. Fowler warns against the event-notification trap (implicit cross-service flows only visible at runtime) — Zynax mitigates this via the explicit WorkflowIR state machine, which is the actual contract, not the events.

**Passive-aggressive command risk:** The current stub `DispatchCapabilityActivity` looks like a command (it is one — gRPC call) but M3 docs describe it as emitting events. This is correctly a command; the events are callbacks. No risk here.

**Event schema evolution:** Temporal handles replay internally. CloudEvents schema evolution is undefined (no version field in the event body beyond the spec version). This is a gap for M6: define CloudEvents schema version strategy before publishing real events.

### B2 — Service Boundaries vs Business Capabilities (DDD / Fowler)

Applying the microservices characteristics check:

| Characteristic | Assessment |
|---|---|
| Componentization via services | ✅ gRPC-only Published Interfaces; protobuf as explicit contracts |
| Organized around business capabilities | ✅ workflow-compiler = "compile intent"; engine-adapter = "execute"; task-broker = "route capabilities"; agent-registry = "catalog agents" — these are genuine business capabilities |
| Smart endpoints, dumb pipes | ✅ NATS as dumb pipe; gRPC as typed pipe; no ESB routing logic |
| Decentralized data management | ✅ ADR-008; each service owns its store; no shared DB |
| Design for failure | 🟡 Temporal carries reliability; timeout/deadline/circuit-breaker absent elsewhere (G4, #569) |
| Evolutionary design | ✅ Proto backward compat; BDD-first allows refactoring with safety net |

**Potential distributed monolith risk:** The dispatch chain (api-gateway → workflow-compiler → engine-adapter → task-broker → agent-registry → agent) is 5 synchronous hops on the hot path. Fowler's First Law: "don't distribute your objects." Each hop multiplies failure probability. Mitigation: Temporal provides durability for the engine-adapter→task-broker hop (Activities with retry). The api-gateway→workflow-compiler→engine-adapter hops should each have explicit deadlines (G4 covers Activities; need deadline policy for the upstream gRPC calls too).

**Service split assessment:** The 7-service split is justified:
- workflow-compiler / engine-adapter / api-gateway: each has a distinct scaling profile and change pattern ✅
- task-broker / agent-registry: tightly coupled (task-broker queries registry every dispatch). Could be merged for v0.x. Keep separate to preserve clear bounded contexts — the operational cost is low since they're in the same compose stack.
- event-bus / memory-service: 0 LoC stubs with no scaling consideration yet. Could be kept as library facades until M6 justifies standalone services. **Recommendation:** Keep as planned; implement as thin gRPC wrappers over NATS/Redis. Modular-monolith-first principle doesn't apply here since they're already isolated.

### B3 — Distributed-Systems Patterns (Richardson)

| Pattern | Present | Partial | Missing | Recommendation |
|---|---|---|---|---|
| **Saga** (distributed transaction with compensation) | ❌ | | ✅ | For task dispatch failures, Temporal's Activity retry IS a saga-like mechanism. Explicit compensation steps should be documented when M5.C completes. |
| **Outbox** (reliable event publish) | ❌ | | | CloudEvents currently a stub. When event-bus is implemented, use transactional outbox to prevent dual-write hazard between Temporal state and NATS publish. Issue: document this in ADR when event-bus is built. |
| **Idempotency keys** | ✅ | | | `ManifestWorkflowID` (SHA-256 + first 16 chars) provides idempotent apply. Same manifest → same run_id. Resubmit after completion appends Unix timestamp. |
| **API Gateway** responsibilities | ✅ | | | Auth, routing, request-ID, apply → gRPC forwarding. Missing: rate limit (#580), ReadHeaderTimeout (#568). |
| **Circuit breaker / Bulkhead** | ❌ | | ✅ | No circuit breaker on gRPC clients. task-broker dispatch could pile up on slow agents. Recommend: add `connectTimeout` + retry budget at gRPC dial time (#569). |
| **Distributed tracing** | ❌ | | ✅ | Only workflow-compiler has OTel hooks. api-gateway and engine-adapter have none. A workflow run is not traceable end-to-end today (#491, H2). |
| **Dead-letter / retry** | 🟡 | | | Temporal's Activity retry is a dead-letter queue equivalent. No retry on NATS publishes (stubs). G4: no explicit RetryPolicy on Activities → Temporal default = infinite. |
| **Backpressure** | ❌ | | | No bounded queues anywhere. task-broker `executeAsync` goroutines are unbounded. agent-registry heartbeat channels unbounded. |
| **Service discovery** | 🟡 | | | In-memory registry (M5 MVP). No DNS-based discovery or Consul/Envoy. Sufficient for Docker Compose; insufficient for K8s. |
| **Dual-write hazard** | ❌ documented | | | When event-bus is implemented, the pattern "write to Temporal AND publish to NATS" is a dual-write hazard. Outbox pattern or transactional publish needed. |

### B4 — Operability (Cockcroft / CNCF / OTel)

| Dimension | Status | Gap |
|---|---|---|
| **Traces** | 🟡 workflow-compiler only | api-gateway, engine-adapter, task-broker have no OTel tracing. A workflow run is untraceable end-to-end. #491 |
| **Metrics** | 🟡 workflow-compiler only | api-gateway and engine-adapter have no Prometheus `/metrics`. #491 |
| **Structured logs** | ✅ All services use structured JSON logs via `log/slog` | |
| **Request correlation** | ✅ X-Request-ID propagated across api-gateway → compiler → engine-adapter (M5.D #484) | task-broker and agent-registry not yet included |
| **Health probes** | ❌ No HEALTHCHECK in Dockerfiles | #463 canvas exists; not yet implemented |
| **Config externalization** | ✅ 12-Factor: env vars + `.env` | |
| **Statelessness** | 🟡 workflow-compiler stateful (IR map) | #466 — must be stateless for horizontal scale |
| **Rollout/rollback** | ✅ Docker Compose; Helm charts in docs/patterns | Helm charts are templates only (no in-tree charts) |
| **SLOs** | ❌ None defined | No latency or error-rate targets for any service |
| **Multi-tenancy** | ❌ Cosmetic only | `namespace` field in proto is not enforced; all workflows share Temporal default namespace |

### B5 — Security and Supply Chain

| Control | Status | Evidence |
|---|---|---|
| TLS between services | ❌ | `insecure.NewCredentials()` everywhere |
| Bearer-token auth (api-gateway) | ✅ | M5.D #482 — but non-constant-time compare (G1 / #567) |
| Non-constant-time compare | ❌ | Confirmed in `auth.go` |
| ReadHeaderTimeout | ❌ | No timeout on HTTP server; Slowloris risk (#568) |
| Container non-root | ✅ | `USER zynax` in all Dockerfiles |
| HEALTHCHECK | ❌ | No HEALTHCHECK directive |
| SBOM per release | ❌ | #235 / #489 / M6 |
| cosign-signed images | ❌ | #239 / #489 / M6 |
| SLSA provenance | ❌ | M6 |
| Trivy CVE scan | 🟡 | Added to release pipeline in #565; not yet in PR checks |
| govulncheck | 🟡 | In `make audit` but not in required CI gates |
| OSSF Scorecard | ✅ | Badge in README |
| Renovate dependency updates | ✅ | Weekly; patch automerge |
| OIDC/JWT auth path | ❌ | Static bearer token only |
| Rate limiting | ❌ | #580 / M6 |

---

## Part C: Gaps Not Yet in the Tracker (NEW)

These items were identified during this review and are not in the M5-plan or any open issue:

| # | Gap | Severity | Recommended action |
|---|---|---|---|
| NEW-1 | No explicit gRPC deadline/timeout on api-gateway → workflow-compiler + engine-adapter calls | Medium | `context.WithTimeout` at dial site; 30s default per call. File as M5 issue. |
| NEW-2 | event-bus / memory-service stub services will crash if their gRPC port is dialed | Low | Add `Unimplemented` gRPC skeleton so the system fails gracefully. Tracked by #574 (H9) — verify it covers both services. |
| NEW-3 | CloudEvents schema version strategy undefined | Low | Define version field or use `spec_version` + envelope version before publishing real events. File as M6 issue. |
| NEW-4 | `ZYNAX_API_KEY=""` bypasses auth silently | Medium | Gateway logs a warning on startup but requests still pass. Add `os.Exit(1)` in production mode. Related to H5. |
| NEW-5 | go.work does not include cmd/zynax-ci (standalone) | Low | Known by design (docs/decisions/004). Document in ARCHITECTURE.md to avoid confusion. |
| NEW-6 | No CODEOWNERS coverage for `docs/` or `spec/` | Low | CODEOWNERS file exists (G9 fixed) but may not cover all areas. Verify. |
| NEW-7 | No `read-only` rootfs hint in docker-compose | Low | Add `read_only: true` to service containers and explicit `tmpfs` mounts for writable dirs. |

---

## Part D: Ranked Gap List (All Open Gaps, By Severity × Effort)

### P0 — Blocks M5 exit criteria

| Gap | Issue | Est. effort |
|---|---|---|
| M5.C: agent-registry domain (#527) | #527 | M (1 wk) |
| M5.C: agent-registry gRPC + go.work (#528) | #528 | S (3 days) |
| M5.C: compose wiring (#481) | #481 | S (1 day) |
| M5.C: task-broker handler tests (#532) | #532 | S (2 days) |
| M5.C: agent-registry BDD trim (#526) | #526 | XS (1 day) |

### P1 — M5 quality / security (do before v0.4.0 tag)

| Gap | Issue | Est. effort |
|---|---|---|
| G1: Bearer constant-time compare | #567 | XS (2h) |
| G2: ReadHeaderTimeout + MaxBytesReader | #568 | XS (2h) |
| G4: Temporal Activity RetryPolicy | #569 | S (1 day) |
| G16: Background context goroutines | #570 | S (1 day) |
| G10: workflow-compiler retention doc | #572 | XS (2h) |
| H1: stateless workflow-compiler (#466) | #466 | S (3-5 days) |
| G17/H9: stub service placeholders | #574 | XS (2h) |
| README status table | #579 | XS (1h) |
| NEW-4: ZYNAX_API_KEY empty behavior | new | XS (1h) |

### P2 — M5 documentation / cleanup

| Gap | Issue | Est. effort |
|---|---|---|
| G7: mergePayload non-strings | #571 | S (1 day) |
| G19: Kagent positioning doc | #575 | S (1 day) |
| G22: summarizer phantom | #576 | XS (1h) |
| G23: phantom AGENT_LIST | #577 | XS (1h) |
| G12: fuzz tests expansion | #539+ | M |

### P3 — M6 (post-M5)

TLS (#488 / ADR-020), SBOM (#235 / #489), OIDC (#488), rate limiting (#580),
Action.Output (#581), resolveTemplate (#584), hash ADR (#583),
pkg.go.dev (#582), OTel baseline (#491), persistent stores (H6).

### P4 — M7/M8

Benchmarks (#493), load tests, Watch polling (#492), CNCF community (#470),
ADR-021 scale (#578).

---

## Part E: "Do Not Change" Confirmation

The 2026-05-20 review explicitly lists architectural elements that must NOT be changed.
These are verified as still present and intact:

| Crown Jewel | File | Confirmed |
|---|---|---|
| Three-layer separation (Intent/Communication/Execution) | Root AGENTS.md, ARCHITECTURE.md, ADR-011 | ✅ |
| Hexagonal `internal/{api,domain,infrastructure}` | All 4 implemented services | ✅ |
| Proto-first + BDD-first discipline | ADR-016, CI gates | ✅ |
| Apache-2.0 + DCO | LICENSE, every file header | ✅ |
| ADR culture (ADR-001–ADR-019) | docs/adr/ | ✅ |
| `WorkflowEngine` 6-method interface | services/engine-adapter/internal/domain/engine.go | ✅ |
| Per-service AGENTS.md | All 9 service/cmd dirs | ✅ |

---

*This gap analysis was produced 2026-05-21 against HEAD on branch `docs/architecture-overhaul-m5`.
Severity ratings are the author's professional judgment informed by the 2026-05-20 review.*
