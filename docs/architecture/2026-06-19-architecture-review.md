<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax — Architecture Review

**Repository:** `github.com/zynax-io/zynax` · Apache-2.0 · CNCF Sandbox candidate (M8 prep)
**Review date:** 2026-06-19
**Reviewer mandate:** Full-stack architecture assessment · Three-layer separation · ADR adherence · CNCF fit · Longitudinal delta vs. [2026-06-18 architecture review](2026-06-18-architecture-review.md)
**HEAD reviewed:** `origin/main` @ `55785e9` (`docs(quickstart): reconcile with real CLI surface; lead with Ollama (#1441)`) — 11 commits ahead of the local checkout (`ab53338`/#1430). All claims below are grounded against `origin/main`.
**Method:** Read-only synthesis — `git show`/`git grep` against `origin/main`, live `gh` GitHub state, ADRs, milestone state, three fan-out `Explore` subagents (findings reconciled against authoritative `origin` diffs where the local working tree disagreed).

---

## Executive Summary

The single most important finding of this review is a **one-day, high-leverage UX delta**: the M7.K "First-run User Experience" cluster (EPIC [#1370](https://github.com/zynax-io/zynax/issues/1370)) landed **11 merged PRs (#1431–#1441)** between the prior review and today. This was the **#1 weakness** in the 2026-06-18 review ("First-run developer experience friction … example workflows still require credentials"). It is now **substantially closed**: a new contributor can run `make demo` — the Makefile's `★ One command` hero target ([Makefile:132](../../Makefile)) — to boot a **zero-secret** local Ollama stack and run the hero `code-review-ollama` workflow with **no API key**.

Critically, the UX push did **not** dent the architecture. Every fix landed on the correct side of a layer boundary:

- The engine-adapter NotFound fix translates a gRPC `codes.NotFound` into a Temporal `NonRetryableApplicationError` **in `internal/infrastructure/`, explicitly keeping `internal/domain/` Temporal-free per ADR-015** ([activity_dispatch.go:31-59](../../services/engine-adapter/internal/infrastructure/activity_dispatch.go)).
- The SSE-logs 500 fix added `Flush()`/`Unwrap()` to the gateway's response wrapper so `http.ResponseController` can reach the embedded `Flusher` — a clean Go-idiom middleware fix, not a layer leak ([main.go:165-178](../../services/api-gateway/cmd/api-gateway/main.go)).
- Adapter graceful degradation (missing secret → no capability registration → `NOT_SERVING` readiness, drain on SIGTERM) reads as a **textbook reliability pattern**, not a hack ([llm-adapter/main.go:76-152](../../agents/adapters/llm/cmd/llm-adapter/main.go)).

**What is real vs. narrative.** Two items the prior review still listed as open gaps are now **shipped and verifiable in code**: **fuzz tests** (`FuzzParseManifest`, `FuzzEvalGuard`) and the entire first-run cluster. Two items the prior review claimed as shipped — **mTLS** and **Postgres-backed repos** — are confirmed real (`tlscreds.go` in all 7 services; `internal/infrastructure/postgres/` in agent-registry/task-broker/memory-service with integration tests). The honest **remaining** gap is no longer "Day-0 onboarding" — it is **scale validation** (no load tests / no published SLOs) and **advanced authz** (no RBAC/OIDC, no admission control). The Day-0 adoption-barrier risk drops from **High** to **Low-Medium**: the zero-credential path exists and is tested, but EPIC #1370 is **still OPEN** with residual declarative-scenario scope (#1385, #1387) and a zero-Temporal evaluation engine (#1359) not yet merged.

| Dimension | 2026-06-18 | 2026-06-19 | Δ | Rationale (grounded) |
|---|---|---|---|---|
| **Architecture** | 8.5 | **8.5** | 0 | Three-layer intact; UX fixes respected ADR-015 domain/infra boundary ([activity_dispatch.go:31](../../services/engine-adapter/internal/infrastructure/activity_dispatch.go)) |
| **Simplicity** | 8.0 | **7.5** | −0.5 | Makefile grew 63→**74 targets** ([Makefile](../../Makefile)); consolidation regressed while features shipped |
| **Performance** | 6.0 | **6.0** | 0 | Benchmarks unchanged; still no load tests |
| **Security** | 8.5 | **8.5** | 0 | mTLS confirmed (`tlscreds.go` ×7); no RBAC/admission yet |
| **Maintainability** | 9.0 | **9.0** | 0 | zynax-ci Go-CLI migration continued (#1426/#1428/#1430) |
| **Scalability** | 7.5 | **7.5** | 0 | Postgres repos confirmed; still no scale data |
| **Reliability** | 8.0 | **8.5** | +0.5 | NotFound fail-fast + adapter graceful-degradation `NOT_SERVING` shipped ([#1434](https://github.com/zynax-io/zynax/pull/1434), [#1432](https://github.com/zynax-io/zynax/pull/1432)) |
| **Testing** | 8.0 | **8.5** | +0.5 | Fuzz now real (`FuzzParseManifest`/`FuzzEvalGuard`); 18 `.feature` files |
| **CI/CD** | 9.0 | **9.0** | 0 | 3 more Python→Go CI ports landed; bash still present in workflows |
| **Documentation** | 7.5 | **8.0** | +0.5 | Quickstart reconciled to real CLI surface ([#1441](https://github.com/zynax-io/zynax/pull/1441)); validation-guide standard ([#1440](https://github.com/zynax-io/zynax/pull/1440)) |
| **CNCF alignment** | 9.0 | **9.0** | 0 | Table-stakes met; community still 1★/0 forks/3 contributors |
| **Product–market fit** | 7.5 | **8.0** | +0.5 | Zero-secret `make demo` removes the biggest Day-0 barrier |
| **Overall** | **8.2** | **8.3** | **+0.1** | UX cluster closes the top weakness without architectural cost |

> The small overall delta is correct and honest: the 2026-06-18 review already scored the architecture as mature. This 24-hour window did not change the *structure* — it removed the **last adoption barrier in the funnel** (first-run friction) and quietly closed two testing gaps. The value here is **de-risking adoption**, not raising the architectural ceiling.

---

## Scorecard — 2026-06-19

| Dimension | Score | Evidence | One-line rationale |
|---|---|---|---|
| **Architecture** | **8.5** | three-layer enforced; ADR-015 boundary held under the UX-fix pressure ([activity_dispatch.go:31-59](../../services/engine-adapter/internal/infrastructure/activity_dispatch.go)); domain has zero `go.temporal.io` imports | Separation proved; engine abstraction real; fixes stayed in the right layer |
| **Simplicity** | **7.5** | `libs/zynaxobs` stdlib live; **Makefile 74 targets** (was 63) — `make demo`/`make run-local` good entry points but target sprawl up | Config/obs centralized; Makefile consolidation (ADR-036) losing ground to feature velocity |
| **Performance** | **6.0** | `manifest_bench_test.go`, `interpreter_bench_test.go` present; no load harness (no k6/vegeta/locust) | Micro-benchmarks gated; macro scale untested |
| **Security** | **8.5** | `tlscreds.go` in all 7 services; constant-time bearer; cosign+SBOM; gitleaks; zero-secret Ollama overlay adds no new secret surface ([docker-compose.ollama.yml](../../infra/docker-compose/docker-compose.ollama.yml)) | mTLS live; supply-chain signed; no RBAC/admission |
| **Maintainability** | **9.0** | `cmd/zynax-ci/` Go CLI absorbed 3 more Python scripts (#1426/#1428/#1430); per-layer AGENTS.md; 36 ADRs | Decisions recorded; CI logic testable Go |
| **Scalability** | **7.5** | `agent-registry/.../postgres/repository.go`, `task-broker/.../postgres/`, `memory-service/.../postgres/pgvector.go` + integration tests; HPA/PDB in Helm | Stateless story real; scale-out untested at load |
| **Reliability** | **8.5** | NotFound→non-retryable fail-fast ([#1434](https://github.com/zynax-io/zynax/pull/1434)); adapter `SetServingStatus(NOT_SERVING)` + SIGTERM drain; gRPC deadlines in `api-gateway/.../clients.go`; NATS AckExplicit/MaxDeliver=5 | Fail-fast + graceful degradation + deadlines all present |
| **Testing** | **8.5** | `FuzzParseManifest` ([manifest_fuzz_test.go:40](../../services/workflow-compiler/internal/domain/manifest_fuzz_test.go)), `FuzzEvalGuard` ([interpreter_test.go:557](../../services/engine-adapter/internal/domain/interpreter_test.go)); **18 `.feature`** files; ≥90% domain coverage | Fuzz now real; BDD Tier 2; e2e runnable |
| **CI/CD** | **9.0** | ADR-027 shift-left; zynax-ci subcommands; multi-arch; DCO+SSH signing | Build-once/promote; testable gates; some bash remains in `.github/workflows/` |
| **Documentation** | **8.0** | quickstart reconciled to real CLI ([#1441](https://github.com/zynax-io/zynax/pull/1441)); human-validation guide standard ([#1440](https://github.com/zynax-io/zynax/pull/1440)); README hero + `make demo` ([#1439](https://github.com/zynax-io/zynax/pull/1439)) | First-run docs now match reality; contributor guide still dense |
| **CNCF alignment** | **9.0** | Apache-2.0+SPDX, DCO, cosign keyless+SBOM, mTLS, Helm, GOVERNANCE.md, 36 ADRs | Table-stakes met; community is the gap |
| **Product–market fit** | **8.0** | zero-secret `make demo` + Qwen2.5-Coder 3B default ([#1437](https://github.com/zynax-io/zynax/pull/1437)); `zynax events publish` + `zynax result` CLI ([#1436](https://github.com/zynax-io/zynax/pull/1436)/[#1438](https://github.com/zynax-io/zynax/pull/1438)) | Day-0 barrier removed; funnel still needs external validation |

---

## Top Strengths (Shipped 2026-06-18 → 2026-06-19)

1. **Zero-secret first-run path is real and tested.** `make demo` (`★ One command`, [Makefile:132](../../Makefile)) boots the Ollama overlay ([docker-compose.ollama.yml](../../infra/docker-compose/docker-compose.ollama.yml)) — `ollama/ollama:latest`, host models mounted **read-only**, **nothing exposed to host LAN**, no OpenAI key — and runs `spec/workflows/examples/code-review-ollama.yaml` ([#1433](https://github.com/zynax-io/zynax/pull/1433), [#1435](https://github.com/zynax-io/zynax/pull/1435), [#1439](https://github.com/zynax-io/zynax/pull/1439)).

2. **Fail-fast dispatch with the ADR-015 boundary intact.** A capability with no registered agent now fails the workflow immediately instead of retrying until timeout — and the gRPC→Temporal error translation lives in `infrastructure`, not `domain` ([activity_dispatch.go:31-59](../../services/engine-adapter/internal/infrastructure/activity_dispatch.go), [#1434](https://github.com/zynax-io/zynax/pull/1434)). This is the strongest single signal that velocity is not eroding the architecture.

3. **Adapter graceful degradation.** Missing API key → adapter starts **degraded**: registers no capabilities, reports `NOT_SERVING`, and never crash-loops ([llm-adapter/main.go:76-130](../../agents/adapters/llm/cmd/llm-adapter/main.go), [#1432](https://github.com/zynax-io/zynax/pull/1432)). Readiness now reflects true capability availability — a real reliability improvement, applied across ci/git/llm adapters.

4. **SSE logs streaming fixed at the right layer.** `statusRecorder.Flush()`/`Unwrap()` forward to the embedded `http.Flusher` so `zynax logs` SSE stops 500-ing ([main.go:165-178](../../services/api-gateway/cmd/api-gateway/main.go), [#1431](https://github.com/zynax-io/zynax/pull/1431)).

5. **CLI completes the observe-loop.** `zynax events publish <run-id> <event-type>` ([events.go](../../cmd/zynax/cmd/events.go)) and `zynax result <run-id>` ([result.go](../../cmd/zynax/cmd/result.go)) close the gap between "apply a workflow" and "see what it produced" ([#1436](https://github.com/zynax-io/zynax/pull/1436), [#1438](https://github.com/zynax-io/zynax/pull/1438)).

6. **Fuzz tests now exist** (prior review had this as an open gap). `FuzzParseManifest` ([manifest_fuzz_test.go:40](../../services/workflow-compiler/internal/domain/manifest_fuzz_test.go)) and `FuzzEvalGuard` ([interpreter_test.go:557](../../services/engine-adapter/internal/domain/interpreter_test.go)) guard the YAML parser and CEL guard evaluator against malformed input.

---

## Top Weaknesses (Still Open)

1. **Load testing / scale SLOs absent.** No k6/vegeta/locust harness anywhere; benchmarks cover micro-ops only. CNCF M8 will require capacity data. (**User type:** operator/product-owner · **Adoption lever:** M8 perf SLO doc + load harness · **Gap issue:** [#1403–#1406](https://github.com/zynax-io/zynax/issues/1403) DD execution)

2. **Makefile sprawl regressed: 63 → 74 targets.** Feature velocity is winning over the ADR-036 consolidation goal. `make demo`/`make run-local`/`make help` exist, but the long tail is unindexed. (**User type:** maintainer/developer · **Adoption lever:** ADR-036 follow-through · **Gap issue:** open)

3. **No RBAC/ABAC; bearer token is all-or-nothing.** OIDC/JWT remains ADR-020 §Planned (M8). (**User type:** security/operator · **Adoption lever:** ADR on OIDC + role claims · **Gap issue:** ADR-020 Planned)

4. **EPIC #1370 still OPEN with residual scope.** The 11 merged PRs are the runnable core, but declarative demo-scenario config ([#1385](https://github.com/zynax-io/zynax/issues/1385), [#1387](https://github.com/zynax-io/zynax/issues/1387)) and the zero-Temporal lightweight evaluation engine ([#1359](https://github.com/zynax-io/zynax/issues/1359)) are not merged. The "no-Docker / no-Temporal" Day-0 onboarding is **not yet there** — `make demo` still requires Docker + Temporal. (**User type:** developer/zynax-user · **Adoption lever:** #1359 zero-Temporal engine · **Gap issue:** [#1359](https://github.com/zynax-io/zynax/issues/1359))

5. **Admission control (Kyverno/Gatekeeper) not shipped.** ADR-020 /C3 calls for policy gates; not in Helm. (**User type:** operator/security · **Adoption lever:** M8 security baseline · **Gap issue:** [#465](https://github.com/zynax-io/zynax/issues/465))

6. **Rate limiting on REST not implemented.** `POST /api/v1/apply` has no throttle. (**User type:** operator/security · **Adoption lever:** M8 middleware · **Gap issue:** open)

7. **API versioning/deprecation strategy under-specified.** gRPC v1 / REST `/api/v1/`; no migration process documented. (**User type:** maintainer/zynax-user · **Adoption lever:** ADR on API evolution · **Gap issue:** open)

---

## Longitudinal Delta vs. 2026-06-18 Architecture Review

### Prior weaknesses — status today

| # | Prior weakness (2026-06-18) | Status today | Evidence |
|---|---|---|---|
| W1 | **First-run UX friction** — examples need GitHub/LLM secrets, no masked/fallback path; M7.K #1370 in-flight | **PARTIALLY CLOSED** | 11 PRs merged (#1431–#1441): zero-secret Ollama overlay + `make demo` + runnable `code-review-ollama.yaml`. Residual: #1385/#1387/#1359 still open; #1370 EPIC OPEN |
| W2 | Makefile 63 targets, unindexed (ADR-036) | **WORSENED** | Now **74 targets** ([Makefile](../../Makefile)); zynax-ci CLI absorbed 3 Python scripts but Makefile grew |
| W3 | Graceful shutdown not standardized | **CLOSED** | All adapters + services drain on SIGTERM, `NOT_SERVING` before `GracefulStop()` ([llm-adapter/main.go:127-152](../../agents/adapters/llm/cmd/llm-adapter/main.go)) |
| W4 | Load testing / scale SLOs absent | **OPEN** | No load harness found |
| W5 | **Fuzz testing not implemented** | **CLOSED** | `FuzzParseManifest`, `FuzzEvalGuard` shipped |
| W6 | Admission control (Kyverno) not shipped | **OPEN** | Not in Helm; [#465](https://github.com/zynax-io/zynax/issues/465) backlog |
| W7 | API versioning strategy under-specified | **OPEN** | No ADR / migration doc filed |

### Prior risk register — status today

| Prior risk | 2026-06-18 status | 2026-06-19 status | Note |
|---|---|---|---|
| R1 API versioning | Open | **Open** | unchanged |
| R2 Load testing / SLOs | In-flight (#1403–#1406) | **In-flight** | DD-execution issues still open |
| R3 Fuzz testing | Proposed | **CLOSED** | fuzz code merged (no ADR-037 was ever filed; closed by code, not decision) |
| R4 First-run UX needs credentials | In-flight | **Largely closed** | zero-secret `make demo` shipped; #1359 (zero-Temporal) remains |
| R5 Graceful shutdown not standardized | Proposed (ADR-031) | **CLOSED** | implemented across adapters/services |
| R6 Admission control | Backlog | **Open** | unchanged |
| R7 Rate limiting | Backlog | **Open** | unchanged |
| R8 RBAC/ABAC | In-flight (ADR-020 Planned) | **Open** | unchanged |

### Corrections to the 2026-06-18 review (truth-pass)

- Prior review cited **"38 ADRs"**; [docs/adr/INDEX.md](../../docs/adr/INDEX.md) actually carries **36** ADR references (32 Accepted, 3 Proposed, 1 Superseded). Corrected here.
- Prior review listed fuzz as an open weakness and referenced a **proposed ADR-037**; no ADR-037 exists on disk. The fuzz *code* shipped anyway — gap closed by implementation, not by an ADR. (Recommend filing the ADR retroactively for one-way-door integrity.)
- Prior review's Makefile-target count (63) is now **74** — the consolidation goal regressed.

---

## Per-Dimension Notes (delta-focused; see 2026-06-18 review for full baseline)

### Three-Layer Separation (ADR-015) — FULLY ENFORCED, stress-tested
The NotFound fix is the proof point: the *temptation* was to classify the Temporal retry behaviour inside the dispatch domain logic; instead the translation sits in `internal/infrastructure/activity_dispatch.go` and the comment explicitly names ADR-015 ("The domain layer is Temporal-free"). Domain layers carry only `protos/generated` imports (the Layer-2 contract), zero `go.temporal.io`. **Score: 9.5.**

### Reliability — IMPROVED (+0.5)
Three reliability primitives landed in one day: fail-fast dispatch (#1434), graceful degradation with honest readiness (#1432), and SSE streaming correctness (#1431). Combined with pre-existing gRPC deadlines (`api-gateway/.../clients.go`) and NATS AckExplicit/MaxDeliver=5, the in-cluster reliability story is now strong. Gap: no chaos/failure-injection testing. **Score: 8.5.**

### Testing — IMPROVED (+0.5)
Fuzz targets close the longest-standing quality gap (open since the 2026-05-20 review). 18 `.feature` BDD files at gRPC boundaries; ≥90% domain coverage gate. Remaining: no load tests, no fuzz on proto unmarshalling. **Score: 8.5.**

### Security — UNCHANGED (8.5)
mTLS confirmed live (`tlscreds.go` in all 7 services — this verifies, not just asserts, the prior review's claim). The Ollama overlay is security-positive: it removes the need to plumb a real LLM secret for the demo path. No RBAC, no admission control, no REST rate-limit. **Score: 8.5.**

### Simplicity — REGRESSED (−0.5)
The only dimension to move down. Feature velocity outran consolidation: Makefile 63→74. The `make demo`/`make help` ergonomics are good, but ADR-036's "retire bash, index everything" goal is slipping. **Score: 7.5.**

### CNCF Alignment — UNCHANGED (9.0)
All technical table-stakes met. The binding constraint is **community**: 1 star, 0 forks, 3 contributors ([gh api](https://github.com/zynax-io/zynax) 2026-06-19). M8.A/M8.B EPICs ([#470](https://github.com/zynax-io/zynax/issues/470)/[#471](https://github.com/zynax-io/zynax/issues/471)) are the governance/community on-ramp; `make demo` is the technical on-ramp that should now feed it. **Score: 9.0.**

---

## Risk Register

| ID | Risk | P | I | Mitigation | Status |
|---|---|---|---|---|---|
| **R1** | Load testing absent; scale SLOs unknown | High | High | M8 perf acceptance criteria; load harness; public SLOs | In-flight ([#1403–#1406](https://github.com/zynax-io/zynax/issues/1403)) |
| **R2** | RBAC/ABAC missing; bearer all-or-nothing | Medium | High | OIDC/JWT + role claims (M8) | Open (ADR-020 Planned) |
| **R3** | EPIC #1370 OPEN — no-Docker/no-Temporal Day-0 not delivered | Medium | Medium | #1359 zero-Temporal eval engine; #1385/#1387 scenario config | In-flight ([#1359](https://github.com/zynax-io/zynax/issues/1359)) |
| **R4** | Admission control not shipped | Medium | Medium | Kyverno in M8; local kind enforcement | Backlog ([#465](https://github.com/zynax-io/zynax/issues/465)) |
| **R5** | Rate limiting on REST absent | Medium | Medium | api-gateway middleware; Helm-configurable | Backlog |
| **R6** | API versioning/deprecation under-specified | Medium | High | ADR on API evolution + migration matrix | Open |
| **R7** | Makefile sprawl (74 targets) erodes contributor onboarding | Low | Medium | ADR-036 follow-through; group + index | Open |
| **R8** | Fuzz code shipped without an ADR (one-way-door gap) | Low | Low | File ADR retroactively recording fuzz strategy | Open |

---

## Prioritized Recommendations

### Tier 1 — Critical (block M8 / production)

| # | Recommendation | Effort | User type | Adoption lever | Issue |
|---|---|---|---|---|---|
| T1.1 | Stand up a load/stress harness and publish SLO targets (concurrent workflows, dispatch fan-out, NATS throughput, Postgres pool). | M (1–2 wk) | operator/product-owner | M8 perf SLO doc + dashboards | [#1403–#1406](https://github.com/zynax-io/zynax/issues/1403) |
| T1.2 | Extend OIDC/JWT auth + role claims to replace the static bearer token; add cert rotation. | M (2 wk) | security/operator | ADR on OIDC provider; cert-rotation SLA | ADR-020 Planned |
| T1.3 | Land the zero-Temporal lightweight evaluation engine so the Day-0 path needs neither Docker-Temporal nor a paid key. | M (1–2 wk) | developer/zynax-user | true no-infra first run | [#1359](https://github.com/zynax-io/zynax/issues/1359) |

### Tier 2 — High (adoption + production confidence)

| # | Recommendation | Effort | User type | Adoption lever | Issue |
|---|---|---|---|---|---|
| T2.1 | Add REST rate limiting on `POST /api/v1/apply` (Helm-configurable). | S (3–5 d) | operator/security | api-gateway middleware | open |
| T2.2 | Ship Kyverno admission policies (Helm optional; kind gate). | M (1–2 wk) | security/operator | M8 admission baseline | [#465](https://github.com/zynax-io/zynax/issues/465) |
| T2.3 | Close out EPIC #1370: declarative demo-scenario manifest (workflow + AgentDef + context in one file). | M (1 wk) | developer/zynax-user | one-file demo authoring | [#1385](https://github.com/zynax-io/zynax/issues/1385)/[#1387](https://github.com/zynax-io/zynax/issues/1387) |
| T2.4 | File the fuzz ADR retroactively + extend fuzz to proto unmarshalling. | S (2–3 d) | quality/security | one-way-door integrity | open |

### Tier 3 — Medium (maintainability / DX)

| # | Recommendation | Effort | User type | Adoption lever | Issue |
|---|---|---|---|---|---|
| T3.1 | Consolidate the 74-target Makefile into grouped namespaces; finish ADR-036 bash retirement. | S (2–3 d) | maintainer/developer | ADR-036 follow-through | open |
| T3.2 | Publish an API versioning + deprecation strategy (REST + gRPC). | S (2–3 d) | maintainer/zynax-user | ADR on API evolution | open |

---

## Gap Analysis — Not Yet Filed (for `/plan` intake)

| Gap | Category | Severity | User type(s) | Adoption lever | Recommended issue title |
|---|---|---|---|---|---|
| Load/SLO harness | Reliability/Scalability | High | operator, product-owner | M8 capacity story; CNCF gate | `test(load): SLO targets + load harness for M8 acceptance` |
| REST rate limiting | Security | Medium | operator, security | DoS resistance; production trust | `feat(security): rate limiting on POST /api/v1/apply` |
| API versioning strategy | Maintainability | High | maintainer, zynax-user | upgrade confidence for SDK authors | `docs(api): REST/gRPC versioning + deprecation timeline` |
| Fuzz ADR (retroactive) | Quality/Process | Low | maintainer, security | decision traceability | `docs(adr): record fuzz-testing strategy (post-hoc ADR)` |
| Makefile consolidation | Maintainability | Medium | maintainer, developer | contributor onboarding | `chore(infra): group 74 Makefile targets; finish ADR-036` |
| Chaos / failure-injection | Reliability | Medium | operator | production-readiness evidence | `test(chaos): pod-kill + dependency-loss injection suite` |
| Demo-scenario manifest | DX | Medium | developer, zynax-user | one-file demo authoring | `feat(spec): declarative demo-scenario manifest (close #1370)` |

---

## Appendix A — Key File References

| Artifact | Path | Relevance |
|---|---|---|
| Engineering constitution | [AGENTS.md](../../AGENTS.md) | three-layer rules, anti-patterns |
| ADR register (36) | [docs/adr/INDEX.md](../../docs/adr/INDEX.md) | all decisions; ADR-015 boundary |
| Milestone state | [state/current-milestone.md](../../state/current-milestone.md) | M7 active, v0.6.0 target |
| NotFound/ADR-015 boundary | [services/engine-adapter/internal/infrastructure/activity_dispatch.go](../../services/engine-adapter/internal/infrastructure/activity_dispatch.go) | gRPC→Temporal translation in infra, domain stays Temporal-free |
| SSE Flush fix | [services/api-gateway/cmd/api-gateway/main.go](../../services/api-gateway/cmd/api-gateway/main.go) | `Flush()`/`Unwrap()` for SSE logs |
| Adapter graceful degradation | [agents/adapters/llm/cmd/llm-adapter/main.go](../../agents/adapters/llm/cmd/llm-adapter/main.go) | `NOT_SERVING` on missing secret; SIGTERM drain |
| Zero-secret demo overlay | [infra/docker-compose/docker-compose.ollama.yml](../../infra/docker-compose/docker-compose.ollama.yml) | host-LAN-isolated Ollama, no API key |
| Hero example | [spec/workflows/examples/code-review-ollama.yaml](../../spec/workflows/examples/code-review-ollama.yaml) | runnable, CLI-completable |
| `make demo` hero target | [Makefile](../../Makefile) | one-command boot + run |
| Fuzz targets | [services/workflow-compiler/internal/domain/manifest_fuzz_test.go](../../services/workflow-compiler/internal/domain/manifest_fuzz_test.go), [services/engine-adapter/internal/domain/interpreter_test.go](../../services/engine-adapter/internal/domain/interpreter_test.go) | YAML + CEL-guard fuzzing |
| New CLI verbs | [cmd/zynax/cmd/events.go](../../cmd/zynax/cmd/events.go), [cmd/zynax/cmd/result.go](../../cmd/zynax/cmd/result.go) | `events publish`, `result` |
| Postgres repos | [services/agent-registry/internal/infrastructure/postgres/repository.go](../../services/agent-registry/internal/infrastructure/postgres/repository.go) | ADR-021 confirmed |
| mTLS creds | `services/*/internal/infrastructure/tlscreds.go` (×7) | ADR-020 confirmed |

## Appendix B — Scorecard Summary

| Dimension | Score | Trend | Confidence |
|---|---:|---|---|
| Architecture | 8.5 | → | High |
| Simplicity | 7.5 | ↓ −0.5 | High |
| Performance | 6.0 | → | Medium |
| Security | 8.5 | → | High |
| Maintainability | 9.0 | → | High |
| Scalability | 7.5 | → | Medium |
| Reliability | 8.5 | ↑ +0.5 | High |
| Testing | 8.5 | ↑ +0.5 | High |
| CI/CD | 9.0 | → | High |
| Documentation | 8.0 | ↑ +0.5 | High |
| CNCF alignment | 9.0 | → | High |
| Product–market fit | 8.0 | ↑ +0.5 | Medium |
| **OVERALL** | **8.3** | **↑ +0.1** | **High** |

---

## Closing Statement

The 24-hour window from 2026-06-18 to 2026-06-19 is a model of **disciplined, high-leverage execution**: 11 PRs eliminated the project's #1 adoption barrier (first-run friction) while every single fix landed on the correct side of a layer boundary — the NotFound translation in `infrastructure`, the SSE flush as idiomatic middleware, the graceful degradation as honest readiness. The architecture did not move because it did not need to; what moved was the **funnel**. A new user can now run `make demo` with no secrets and watch a real workflow execute.

Two quieter wins: fuzz testing — open since the 2026-05-20 review — is now real, and the prior review's mTLS/Postgres claims are verified in code rather than merely asserted.

The honest remaining work is **operational maturity, not architecture**: load/SLO validation (R1), RBAC/OIDC (R2), and the *truly* infra-free Day-0 path (#1359, the zero-Temporal engine) that would let someone try Zynax with neither Docker nor a key. EPIC #1370 should not be closed until that engine lands. The next review's axis of motion should be **community and external adoption signals** (1★/3 contributors today) — the technical on-ramp is now built; the question is whether anyone walks up it.

OUTPUT_PATH: docs/architecture/2026-06-19-architecture-review.md
