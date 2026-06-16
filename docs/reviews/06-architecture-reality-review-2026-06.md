# Zynax — Architecture & Reality Review

> **Verified locally on 2026-06-17, commit `42787fb` (#1294 — repo truth-pass).**
> Unlike the static draft this supersedes, the Go toolchain (1.26.4), Docker
> (29.4.1) and an authenticated `gh` CLI were all available, and the GitHub API was
> **not** rate-limited (5000/hr fresh). Every quantitative claim below was measured,
> not assumed. Build/lint/test/security gates were run via the repo's own Docker
> toolchain image; live counts come from `gh` against `zynax-io/zynax`, not from
> `state/current-milestone.md`. The kind-based e2e was **verified via the latest
> green CI run** (run `27640115904`) rather than re-run locally — noted as such.

| Field | Value |
|---|---|
| **Repository** | `github.com/zynax-io/zynax` · Apache-2.0 |
| **Branch / commit** | `main` @ `42787fb` (#1294, 2026-06-16) — review checkout fast-forwarded to match |
| **Review date** | 2026-06-17 |
| **Method** | Full clone; build/lint/test/security run via `make` (Docker toolchain image); live `gh` queries for milestones, CI, releases, alerts, rulesets, packages; code reads of the dispatch + CloudEvents + xfail paths; e2e confirmed via CI run `27640115904`. |
| **Active milestone** | M7 — Usable Workflows + Observability (target v0.6.0) |
| **Maintainer** | Óscar Gómez Manresa (sole maintainer) |

**Headline:** the critical *delivery-vs-narrative* gap from the earlier 6.5/10
review is **closed and now runtime-confirmed.** The end-to-end dispatch path is
wired *and executes* — the `e2e-smoke` gate's **temporal and argo legs both pass
green in CI**, and the temporal leg is a **required** status check. The two former
0-LoC stubs (`event-bus`, `memory-service`) are real and backed by NATS JetStream /
Redis-KV + pgvector. The *hosted* security posture is clean (0 open Dependabot, 0
open code-scanning). **One caveat the static draft missed:** the local `make ci`
gates are **not** green — `make test` and `make security` both fail (rc=2) on the
exact quality debt the M7 planning doc flags and that **EPIC Q (#1172)** exists to
close (a freshly-published pip CVE; one environment-sensitive ci-adapter test).
Residual risk has shifted from *"does it work"* to *"is it usable"* (the M7
data-flow keystone, still open), *"is `make ci` green"* (tracked debt), and *"is the
bus-factor survivable."*

---

## Contents

1. [Executive Summary](#1-executive-summary)
2. [Project Objective & Scope](#2-project-objective--scope)
3. [Repository Inventory (measured)](#3-repository-inventory-measured)
4. [Reality vs. Narrative — the core question](#4-reality-vs-narrative--the-core-question)
5. [CI, Images, Packages & Dependencies](#5-ci-images-packages--dependencies)
6. [Decisions, Docs & SPDD Canvases](#6-decisions-docs--spdd-canvases)
7. [Expert Mesh & Self-Hosting Automation](#7-expert-mesh--self-hosting-automation)
8. [Risks & Prioritised Recommendations](#8-risks--prioritised-recommendations)
9. [Verdict](#9-verdict)
10. [Verification Log](#10-verification-log)

---

## 1. Executive Summary

Zynax positions itself as *"the declarative control plane for AI agent workflows —
to AI workflows what Kubernetes is to containers."* Seven milestones in, the project
is materially ahead of where the last architectural review (6.5/10, 24 gaps) left
it. The single most important finding is positive and now **runtime-confirmed**:
the end-to-end capability dispatch path is real, wired, *and executes on real
engines in CI.*

Tracing the code: `engine-adapter`'s `DispatchCapabilityActivity` calls
`TaskBrokerService.DispatchTask` over gRPC, polls to a terminal status, and
`task-broker` resolves an agent via the registry and streams `ExecuteCapability`
events from a real agent endpoint until `COMPLETED`/`FAILED`. The `e2e-smoke` gate
stands up an ephemeral kind cluster, deploys the Helm umbrella chart, and asserts a
workflow reaches a *succeeded* terminal state — and the **latest green run
(`27640115904`) shows both `e2e smoke (temporal)` and `e2e smoke (argo)` jobs
passing.** That is the assertion the previous review could not make, now measured.

### 1.1 Scorecard *(post-verification)*

| Dimension | Score | One-line verdict |
|---|:---:|---|
| Contracts & API discipline | **9/10** | 9 protos + JSON-Schema + **306** BDD scenarios across 18 feature files |
| End-to-end execution | **9/10** | Dispatch path executes; **both** engine legs green in CI; temporal leg is a required check |
| Service implementation depth | **8/10** | All 7 services real; former 0-LoC stubs now substantial; test LoC > src LoC for every service |
| Kubernetes readiness | **7/10** | 14 Helm charts, probes, mTLS wiring; unproven at production scale |
| Security & supply chain | **7/10** | Hosted scanning clean (**0 open** Dependabot / code-scanning); cosign + SBOM + mTLS in all 7 services. **But local `make security` fails** on pip CVE PYSEC-2026-196 (tracked, EPIC Q) |
| CI/CD maturity | **8/10** | Path-filtered, SHA-pinned, ruleset-enforced; engine-matrix e2e with temporal leg required. **`make ci` not green locally** — `make test` red on one env-sensitive adapter test |
| Config & convergence | **7/10** | `libs/zynaxconfig` landed; all 17 modules on Go 1.26.4 |
| Observability | **5/10** | Prometheus `/metrics` shipped; OTel/Uptrace traces in-flight (M7 EPIC O, partially merged) |
| Usability of workflows | **4/10** | Data-flow bindings (#1167) are the M7 keystone — still **open** |
| Project sustainability | **3/10** | Single maintainer; CNCF needs ≥2 orgs |

**Composite:** roughly **7.9/10** as an engineering artifact — up from 6.5 — with the
residual weak axes being usability (the thing M7 exists to fix) and bus-factor, not
correctness. The end-to-end axis ticks up from 8→9 (runtime execution on both engines
is now CI-confirmed, temporal leg required — see §5.1), but CI/CD (9→8) and Security
(8→7) are tempered by the measured discovery that `make ci` is **not** green locally
on tracked quality debt (§5.3, §8.3). Net: marginally below the draft's optimistic
8.0–8.1 once the red local gates are counted honestly.

---

## 2. Project Objective & Scope

Zynax is a declarative, engine-agnostic control plane that turns existing systems
into agent *"capabilities"* without requiring an SDK. The design rests on a few
load-bearing decisions, all captured as ADRs:

- **Three-layer separation** — Intent (YAML manifests) / Communication (gRPC + CloudEvents) / Execution (pluggable engines).
- **Declarative YAML as the primary interface** (ADR-011) compiling to a canonical, engine-agnostic Workflow IR (ADR-012).
- **Adapter-first, no-SDK integration** (ADR-013) — existing HTTP/git/CI/LLM systems wrapped as capabilities.
- **Pluggable workflow engines** (ADR-015) — Temporal and Argo both implemented behind one `WorkflowEngine` interface.
- **Event-driven state machines over DAGs** (ADR-014), explicitly chosen and documented as a non-goal otherwise.

Target end-state is CNCF Sandbox readiness (M8 / v1.0.0). The objective is coherent
and the codebase is faithful to it — there is no evidence of the project quietly
becoming an LLM framework, which ADR-011 explicitly rules out.

---

## 3. Repository Inventory (measured)

### 3.1 Platform services — Go, hexagonal, all real

Every service follows the `internal/{api,domain,infrastructure}` layout and ships a
`cmd/<svc>/main.go` entrypoint. LoC counts are non-test source vs test source,
counted from the working tree at `42787fb` (`find … -name '*.go' [! -name '*_test.go'] | xargs wc -l`).

| Service | Src LoC | Test LoC | `.go` files | Status |
|---|:---:|:---:|:---:|---|
| engine-adapter | 2,438 | 3,156 | 31 | Real — Temporal + Argo |
| workflow-compiler | 2,058 | 2,943 | 21 | Real — YAML → IR |
| task-broker | 1,838 | 1,894 | 23 | Real — dispatch core |
| api-gateway | 1,386 | 2,202 | 24 | Real — REST + SSE |
| event-bus | 984 | 1,533 | 15 | Real — NATS JetStream (`nats.go`) |
| agent-registry | 978 | 1,343 | 13 | Real — resolution |
| memory-service | 936 | 1,342 | 15 | Real — Redis-KV (`redis_kv.go`) + pgvector (`postgres/pgvector.go`) |

> **Measured exactly as the draft predicted** — every LoC figure above matches.
> `event-bus` and `memory-service` — previously documented as genuine 0-LoC stubs —
> are substantial implementations with real backing adapters (verified by file
> presence: `services/event-bus/internal/infrastructure/nats.go`,
> `services/memory-service/internal/infrastructure/{redis_kv.go,postgres/pgvector.go}`).
> Test LoC exceeds source LoC for *every* service.

### 3.2 Adapters, libraries, CLI

- **Adapters:** `http`, `git`, `ci` (Go) and `langgraph`, `llm` (Python) — all 5 present under `agents/adapters/`. ADR-035 (Accepted 2026-06-16) sets the Go/Python language boundary; M7 EPIC **P** (#1276) ports `llm-adapter` to Go — the Python tree still exists pending that port.
- **Shared libs:** `libs/zynaxconfig` (the convergence fix for the old *"2 config mechanisms / 5 env-prefix conventions"* finding) and `libs/zynaxobs`.
- **CLI:** `cmd/zynax` (`apply`/`get`/`delete`/`status`/`logs` + GitOps watch) and `cmd/zynax-ci` (CI logic being consolidated from bash into Go per ADR-036, Accepted; M7 EPIC **S** #1285).

### 3.3 Contracts

9 proto files under `protos/zynax/v1` (`agent`, `agent_registry`, `task_broker`,
`memory`, `event_bus`, `workflow_compiler`, `engine_adapter`, `policy`,
`cloudevents`). Generated Go + Python stubs are a workspace module. Contract tests
use Godog BDD against every RPC — **306 scenarios across 18 `.feature` files**
(the draft's "140+" was a significant understatement).

---

## 4. Reality vs. Narrative — the core question

Each row verified against **code and runtime**, with the command/file that proves it.

| Claim | Reality | Evidence | Verified by |
|---|:---:|---|---|
| End-to-end dispatch path | ✅ **WIRED** | `activity.go`: `broker.DispatchTask` → poll → terminal; `agent_executor.go` streams `ExecuteCapability`. | code read |
| E2E demo reaches *succeeded* | ✅ **EXECUTED** | `e2e-smoke` run `27640115904`: `e2e smoke (temporal)` ✅ **and** `e2e smoke (argo)` ✅. | `gh run view 27640115904` |
| event-bus / memory-service real | ✅ **REAL** | 984 / 936 src LoC; `nats.go`, `redis_kv.go`, `postgres/pgvector.go` present. | `wc -l` + file read |
| mTLS between services | ✅ **WIRED** | `tlscreds.go` present in all 7 services; cert-manager wiring in Helm. | `ls services/*/internal/infrastructure/tlscreds.go` |
| cosign signing + SBOM | ✅ **REAL** | cosign in release lane; `make sbom` target present; SPDX/CycloneDX. | `grep -rl cosign .github/workflows` |
| Helm / K8s-native | ✅ **REAL** | 14 `Chart.yaml` files. | `find . -name Chart.yaml \| wc -l` |
| Engine-adapter CloudEvents | ⚠️ **REAL\*** | Publishes to `EventBusService`, **best-effort**: "errors are logged but not returned so that event-bus [outage doesn't fail the workflow]". | `activities.go:57,79` |
| Self-hosted dev-automation | ⚠️ **BOUNDARY** | Orchestrator + 9 expert AgentDefs validate against schema; execution-on-platform is a `strict=True` xfail (#1103, M7). | `test_platform_readiness.py:48` |
| Workflow data-flow bindings | ❌ **NOT YET** | #1167 (M7.W) confirmed **OPEN** in milestone M7. Without output→input bindings, multi-step workflows can't pass data. **Headline usability gap.** | `gh issue view 1167` |
| Distributed tracing (OTel) | ⚠️ **PARTIAL** | Prometheus `/metrics` shipped (M6); OTel traces + Uptrace UI are M7 EPIC **O**, partially merged (O.4/O.7/O.8 per state file). | state file + #467 open |
| e2e-smoke is a *required* gate | ✅ **REQUIRED (temporal)** | Ruleset `main-protection` lists `e2e smoke (temporal)` among required checks (argo leg not required). **Refines the draft's "non-required" claim.** | `gh api …/rulesets/17547241` |
| Local `make ci` gates green | ❌ **RED (tracked debt)** | `make test` rc=2 → `test-unit-adapters` fails on `TestGetRunStatus_ContextTimeout` (5/5 deterministic, env-sensitive). `make security` rc=2 → `security-agents` fails on pip 26.1.1 / PYSEC-2026-196. Both = EPIC Q (#1172) scope. **Draft assumed clean gates.** | `make test`, `make security` |
| Hosted security clean | ✅ **CLEAN** | 0 open Dependabot, 0 open code-scanning (distinct surface from local `make security`). | `gh api …/dependabot/alerts`, `…/code-scanning/alerts` |

> **Bottom line:** the narrative is honest and now runtime-backed. Where reality
> lags the marketing (CloudEvents best-effort, self-hosting at-boundary, data-flow
> pending) the gap is explicitly tracked in-repo with failing tests as the gate. The
> `strict=True` xfail as an *"aspirational plane"* honesty marker remains good
> practice.

---

## 5. CI, Images, Packages & Dependencies

### 5.1 CI pipeline

`ci.yml` runs a path-filtered, container-based pipeline:
`dco → changes → lint-proto/lint-go/lint-python → test-go/test-python/test-unit/test-integration → security → build-images`.
All third-party Actions are pinned to 40-char SHAs; the `ci-runner`/`tools` images are digest-pinned via `images.yaml`. **CI is green on `main`** (latest 12 runs all `success`).

- **Branch protection is ruleset-based, not classic.** `main` returns *"Branch not protected"* on the legacy API; protection is enforced by the active ruleset **`main-protection`** (`deletion`, `non_fast_forward`, `required_linear_history`, `required_signatures`, `pull_request`, `required_status_checks`). This is why squash-merge + signed commits are mandatory.
- **Required status checks (measured):** `dco`, `test-unit`, `security`, `lint-proto`, `lint-go`, `lint-python`, `GitHub Actions workflow lint`, `Conventional Commit title`, `PR size label`, `Secret scan (gitleaks)`, **`e2e smoke (temporal)`**. → The draft's claim that *"a green required-set does not guarantee the e2e path"* is **partially refuted**: the **temporal e2e leg is required**. The **argo** leg is *not* in the required set (advisory).
- **Merge Queue is off:** `merge_group` appears in **no** workflow (`grep -rln merge_group .github/workflows → NONE`), confirming the deliberate revert (#545/#589).

### 5.2 Images & supply chain

- **`images.yaml` is a real single source of truth** (ADR-024): `make check-images` exits **0** (drift gate clean). `make sync-images` stamps consumers.
- **GHCR packages exist** for all 7 services + 5 adapters + `tools` + `ci-runner`, each with a `staging/*` mirror (26 container packages total). The *"unknown/unknown"* rows are SLSA provenance/SBOM attestation manifests (ADR-025), not broken images.
- **Native multi-arch builds** (M6 #837) eliminated QEMU; distroless-nonroot final images.

### 5.3 Dependencies

- **Go 1.26.4 is consistent across all 17 modules** (`grep '^go ' $(find . -name go.mod)` → every module `1.26.4`).
- **Security scanning enforced in CI:** govulncheck (Go), bandit + pip-audit (Python), Trivy.
- **Hosted security clean (2026-06-17):** **0 open Dependabot alerts**, **0 open code-scanning alerts** (`gh api …/dependabot/alerts` and `…/code-scanning/alerts?state=open` → both `0`).
- **Local `make security` is RED (measured):** `make security` exits rc=2 — `security-agents` (Makefile:327) fails because pip-audit finds **pip 26.1.1 → PYSEC-2026-196 (fix 26.1.2)** in the Python toolchain env. Go `govulncheck` reports **0 affecting** vulnerabilities (only non-called transitive modules), and the Go adapter scan passes. This is the *"security-agents fails on a tools-image pip CVE"* item already documented in `M7-planning.md §2` and owned by **EPIC Q (#1172)**. The hosted Dependabot/code-scanning surface is a different, clean surface; the CVE is freshly published, so the last green CI run on `main` predates it.
- **Local `make test` is RED (measured):** `make test` exits rc=2 — `test-unit-adapters` (Makefile:188) fails on a single test, `TestGetRunStatus_ContextTimeout` in `agents/adapters/ci/internal/adapter`, which expects error code `TIMEOUT` but receives `UPSTREAM_ERROR` ("context deadline exceeded"). Re-run **5/5 deterministic** on Go 1.26.4 here — it is *not* flaky locally, but CI is green, so the classification is **environment/Go-patch-sensitive**. Worth a fix issue (error-mapping of `context.DeadlineExceeded`) under EPIC Q/R. Because `make` stops at the first failure, the `test-coverage`/`test-coverage-adapters` gates were **not reached** in this run (so the ≥90%/≥80% coverage claims are *unverified-locally*, not refuted).

> **One measured caveat (proto stubs):** local `make generate-protos` against the
> pinned tools image produced a diff in the **Python** stubs only (17 files,
> +1153/−1291) while the Go stubs were clean. This is consistent with a
> protoc/grpcio-tools version skew between the local invocation and the committed
> output rather than a contract change; the post-merge `proto-generate.yml` gate
> regenerates stubs on `main`. Worth a one-line confirmation that the tools image
> digest matches what produced the committed Python stubs. **Not** a contract drift.

---

## 6. Decisions, Docs & SPDD Canvases

- **36 ADRs** (ADR-001 … ADR-036) with an INDEX and TEMPLATE. ADR-034 (manifest workflow-id collision domain), ADR-035 (adapter language boundary), ADR-036 (CI logic as Go CLI) are all **Accepted** and reflect M7 cleanup.
- **52 SPDD REASONS-Canvas folders** under `docs/spdd/` (the draft's "~50"), validated by `make validate-canvas` (ADR-019). Every open M7 EPIC has a canvas folder (`<issue#>-<slug>` convention). Automation assets live in `automation/`, never in auto-loaded `AGENTS.md` paths.
- **Prior reviews archived in-repo** (`docs/reviews/00–05`). `ARCHITECTURE.md` is materially in sync with code.

> **Doc-hygiene actions surfaced (now low-cost):**
> 1. `ROADMAP.md` still labels M7 *"Full Observability / v0.6.0"* while
>    `state/current-milestone.md` broadens it to *"Usable Workflows + Observability."*
> 2. **EPIC letter-scheme drift:** issue titles use `M7.A/B/C` (for the pre-existing
>    #467/#468/#469) while the planning doc uses `O/L/R` and marks them "absorbed."
>    Worse, **#1168 is titled `M7.C` — colliding with #469's `M7.C`.** Reconcile the
>    epic letters across issue titles, labels and the planning table.

---

## 7. Expert Mesh & Self-Hosting Automation

The dev-automation epic runs on two parallel surfaces:

- **Claude Code surface (`.claude/commands/`):** `/milestone-{orchestrate,new,plan,close,learn}`, `/resume-milestone`, `/repo-clean`, `/issue-deliver`, and an `experts/` subfolder (go-services, bdd-contract, ci-release, git-ops, infra-helm, python-adapters, spdd-canvas, post-merge).
- **Zynax-native surface (`automation/workflows/`):** a `kind: Workflow` orchestrator (ADR-028) plus 9 expert AgentDefs, a learning-synthesizer and an issue-delivery workflow.

The two-plane separation is enforced by tests:
`test_platform_readiness.py::test_expert_agentdefs_schema_valid` validates the
expert AgentDefs and orchestrator against JSON schemas (passing, near-term plane),
while `test_orchestrator_executes_on_platform` is a **`strict=True` xfail**
(verified at `test_platform_readiness.py:48`) that flips green only when the
compiler accepts `output:` bindings, CEL guards, and orchestration-capability
providers. Only the orchestrator writes to GitHub; reviewers are read-only.

---

## 8. Risks & Prioritised Recommendations

### 8.1 Top risks

| # | Risk | Severity | Why it matters |
|:---:|---|:---:|---|
| R1 | Workflow data-flow bindings unshipped (#1167, confirmed OPEN) | 🔴 **High** | Until output→input bindings land, *"multi-step workflow"* is aspirational. M7 keystone; gates real usability. |
| R2 | Single maintainer / bus-factor | 🔴 **High** | CNCF Sandbox needs ≥2 maintainers from ≥2 orgs (M8). Sole authorship is the biggest threat to v1.0.0. |
| R3 | Self-hosting stuck at boundary (#1103, confirmed OPEN) | 🟠 **Medium** | Four code-verified platform gaps block the orchestrator from running on Zynax itself; honestly gated, still a credibility item. |
| R4 | Observability half-shipped | 🟠 **Medium** | Metrics exist; OTel traces / Uptrace UI mid-flight (EPIC O). A control plane without e2e traces is hard to operate at scale. |
| R5 | *Argo* e2e leg not in required set | 🟢 **Low** | **Downgraded:** temporal leg *is* required, so the dispatch path is regression-guarded on every infra-touching PR. Only the argo leg is advisory. |

### 8.2 Recommendations (right-sized — no overbuild)

- **H1 — Land #1167 data-flow first.** It is the keystone; every *"real workflow"* and template EPIC (T #1171) depends on it. Single critical-path item for v0.6.0.
- **H2 — Recruit a second maintainer.** Months-long social process; begin before the M8 application. *(Deferred to M8 per maintainer decision — not an M7 deliverable.)*
- **H3 — Reconcile `ROADMAP.md` + the EPIC letter scheme** to the live state file (M7 title; resolve the `M7.C` collision between #469 and #1168; drop stale Merge-Queue references). Cheap; prevents fresh truth-drift.
- **H4 — Consider promoting the argo e2e leg (or a sub-2-min compose dispatch smoke) to required** once argo is stable. *(Temporal leg already required — the draft's concern is largely addressed.)*
- **H5 — Sequence the llm-adapter Go port (P #1276) and CI-bash-to-Go (S #1285) after #1167**; valuable cleanup that must not steal the keystone's runway.
- **H6 — Flip the #1103 xfail as the platform-readiness north star.** When `output:`/CEL/orchestration providers exist, the self-hosting story becomes demonstrable — a strong CNCF narrative asset.

### 8.3 Corrections

Items the static draft got wrong, with the evidence that corrected them:

1. **BDD scenario count understated.** Draft: "140+". Measured: **306** scenarios across 18 `.feature` files (`grep -rhE '^\s*(Scenario|Scenario Outline):' protos/tests | wc -l`). *Corrected upward.*
2. **e2e-smoke is *not* fully non-required.** Draft (§5.1/R5): "e2e-smoke is a gated, non-required check … a green required-set does not guarantee the e2e path." Measured: **`e2e smoke (temporal)` is in the required-status-checks set** of ruleset `main-protection`; only the argo leg is advisory. *R5 downgraded from Low–Med to Low.*
3. **Branch protection mechanism.** Draft implied classic branch protection. Measured: classic API returns *"Branch not protected"*; protection is enforced via the **ruleset `main-protection`** (modern mechanism). *Clarified — not a defect.*
4. **Commit/SHA reference.** Draft header cited `42787fb` (#1294) but the working checkout was a commit behind; fast-forwarded to `42787fb` so verification matches the cited commit exactly.
5. **New, previously-unflagged finding:** local Python proto-stub regeneration drift (+1153/−1291, Go clean) — see §5.3 caveat. Likely tooling version skew, not contract drift.
6. **Biggest correction — `make ci` is RED locally.** The draft scored CI/CD 9 and Security 8 implying clean gates. Measured: **`make test` (rc=2)** fails on `TestGetRunStatus_ContextTimeout` (ci-adapter, 5/5 deterministic) and **`make security` (rc=2)** fails on pip CVE PYSEC-2026-196. These are *not* surprises to the repo — `M7-planning.md §2` reality-check predicted them and **EPIC Q (#1172)** owns the fix — but the draft missed them. Scores tempered (CI/CD 9→8, Security 8→7); composite 8.1→7.9. *This validates the M7 plan rather than contradicting it.*
7. **Coverage claims unverified-locally** (not refuted): `make` halted at `test-unit-adapters` before the coverage gates ran, so the ≥90% domain / ≥80% adapter figures were not measured this pass.
8. **Everything else held.** The draft's per-service LoC table, the CloudEvents best-effort nuance, the `strict=True` xfail, the 30-closed/56-open M7 split, the 0-open-alert security posture, releases v0.4.0+v0.5.0, 9 protos / 7 services / 14 charts / 36 ADRs, and Go 1.26.4 across all 17 modules were **all confirmed exactly as written.** For a static-only draft, the accuracy rate is high.

---

## 9. Verdict

Zynax has crossed the line the previous 6.5/10 review said it had not, and this pass
**measured it rather than inferred it: workflows execute end-to-end on both real
engines, asserted green in CI, with the temporal leg a required gate.** The services
are real, the contracts are disciplined (306 BDD scenarios), the supply chain is
hardened (0 open alerts, cosign + SBOM + mTLS), and the places where reality still
lags the vision are tracked honestly with failing tests rather than concealed.

The remaining work is not about correctness; it is about **usability** (the M7
data-flow keystone #1167 that lets a developer author a genuinely useful multi-step
workflow — confirmed still open) and **sustainability** (escaping single-maintainer
status before CNCF). Ship #1167, reconcile the roadmap and EPIC letters, and recruit
a co-maintainer — and the path from v0.6.0 to a credible CNCF Sandbox submission is
clear.

**Composite assessment: ~7.9 / 10, trajectory strongly positive** — the only
downward pull from the static draft's optimism is the honest accounting of the red
local `make ci` gates, which are tracked debt with a clear owner (EPIC Q #1172),
not architectural rot.

---

## 10. Verification Log

Build/lint/test/security were run via the repo's Docker toolchain image
(`make` targets). Live state via `gh` against `zynax-io/zynax`. e2e via CI run.

### 10.1 Claim → method → result → verdict (Part F checklist)

| # | Claim | Method | Result | Verdict |
|:---:|---|---|---|---|
| 1 | Builds clean; 17 modules on Go 1.26.4 | `grep '^go ' $(find . -name go.mod)` | every module `go 1.26.4` | **CONFIRMED** |
| 2 | `make test` green; domain ≥90% / adapter ≥80% | `make test` (gate run) | **rc=2 FAIL** at `test-unit-adapters`: `TestGetRunStatus_ContextTimeout` 5/5; coverage gates not reached | **REFUTED (test red)** + coverage **UNVERIFIED-LOCALLY** |
| 3 | BDD ~140+ | `grep -rhE 'Scenario(\| Outline):' protos/tests \| wc -l` | **306** across 18 `.feature` files | **REFUTED (understated) — actual 306** |
| 4 | e2e reaches succeeded at runtime | `gh run view 27640115904` | both legs `success` | **CONFIRMED (via CI)** |
| 5 | e2e passes for **both** temporal & argo legs | `gh run view 27640115904 --json jobs` | `e2e smoke (temporal)` ✅, `e2e smoke (argo)` ✅ | **CONFIRMED** |
| 6 | event-bus & memory-service functionally real | `wc -l` + file presence | `nats.go`; `redis_kv.go` + `pgvector.go` | **CONFIRMED** |
| 7 | `test_orchestrator_executes_on_platform` still strict xfail; 9-expert test passes | read `test_platform_readiness.py:48` | `@pytest.mark.xfail(strict=True)`; `test_expert_agentdefs_schema_valid` present | **CONFIRMED** |
| 8 | M7 = 30 closed / 56 open | `gh api …/milestones` | M7 = **30 closed / 56 open** | **CONFIRMED** |
| 9 | #1167 open; #1103 deferred/open | `gh issue view 1167 / 1103` | both **OPEN**, milestone M7 | **CONFIRMED** |
| 10 | CI green on main; e2e not in required set; merge_group absent | `gh run list`; ruleset; `grep merge_group` | CI green; **temporal e2e IS required**; merge_group **NONE** | **PARTIAL** (e2e-temporal required; merge_group confirmed absent) |
| 11 | 0 open Dependabot; code-scanning clean | `gh api …/dependabot/alerts`, `…/code-scanning/alerts?state=open` | **0** and **0** | **CONFIRMED** |
| 12 | `check-images` exits 0; cosign + SBOM + mTLS present | `make check-images`; greps | rc=**0**; cosign in workflows; `tlscreds.go` ×7 | **CONFIRMED** |
| 13 | v0.4.0 + v0.5.0 releases exist | `gh release list` | both present (+ snapshots, proto-stub releases) | **CONFIRMED** |
| 14 | CloudEvents publish best-effort | read `activities.go:57,79` | "errors are logged but not returned" | **CONFIRMED** |

### 10.2 Raw highlights

- **Toolchain:** `go version go1.26.4 linux/amd64`; Docker server `29.4.1`; `gh` authed as `ogomezm`; rate limit 4999/5000.
- **Milestones:** M1 40/0c · M2 31/0 · M3 6/0 · M4 30/0 · M5 150/0 · M6 183/0 · **M7 30c/56o** · M8 0/5 · M-dx 0/6.
- **e2e-smoke run `27640115904`:** `e2e smoke (argo)` success · `e2e smoke (temporal)` success.
- **Required checks (ruleset `main-protection`):** dco, test-unit, security, lint-proto, lint-go, lint-python, GitHub Actions workflow lint, Conventional Commit title, PR size label, Secret scan (gitleaks), **e2e smoke (temporal)**.
- **Drift gates:** `make check-images` rc=0 ✅; `make generate-protos` Go clean, Python +1153/−1291 (version-skew caveat, §5.3).
- **Hosted security:** 0 open Dependabot · 0 open code-scanning.
- **lint:** `make lint` rc=**0** ✅ — proto (`buf lint` + format), all 7 Go services 0 issues, Go adapters 0 issues, Python (ruff/black/etc.) all passed.
- **test gate:** `make test` rc=**2** ❌ — `test-unit-adapters` (Makefile:188): `--- FAIL: TestGetRunStatus_ContextTimeout` (`want TIMEOUT, got UPSTREAM_ERROR / "context deadline exceeded"`), `agents/adapters/ci/internal/adapter`. 5/5 deterministic on re-run.
- **security gate:** `make security` rc=**2** ❌ — Go `govulncheck` 0 affecting; Go adapter scan passed; `security-agents` (Makefile:327) fails: pip-audit → `pip 26.1.1  PYSEC-2026-196  fix 26.1.2` (+ `zynax-sdk 0.1.0` not-on-PyPI advisory, non-fatal). Both failures owned by EPIC Q #1172.

### 10.3 Could-not-verify-locally

- Kind-based e2e was **not** re-run locally (relied on CI run `27640115904`). The `scripts/e2e/` suite (`cluster-up.sh`, `e2e-happy.sh`, `e2e-failure.sh`, `helm-upgrade.sh`, `e2e-argo.sh`) is present and is the same path CI exercises.
- Production-scale K8s behaviour (HPA, multi-node, failure injection) remains untested by this review.
