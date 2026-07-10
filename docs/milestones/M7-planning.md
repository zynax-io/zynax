<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax M7 — Usable Workflows + Observability Planning

> **Shipped in:** v0.7.0 (jointly with M8, released 2026-07-10; the original v0.6.0 target was
> skipped to keep tags monotonic) · **GitHub milestone:** #7 (`Usable Workflows + Observability (M7)`) —
> closed at 0 open / 172 closed
> **Status:** ✅ Complete (opened 2026-06-15, closed 2026-07-10) · **Last updated:** 2026-07-10 (prior 2026-06-19 truth-pass: ~115 closed / ~10 open,
> ~92% done; **EPIC #1370 M7.K — First-run UX cluster delivered** — 11 stories merged via PRs #1431–#1441,
> canvas `docs/spdd/1370-awesome-quickstart/canvas.md` → Implemented (core path); remaining #1359→M-dx and
> #1385/#1387 own-canvas, hero asciinema cast a human follow-up. Earlier 2026-06-17 pass: milestone renamed
> "Full Observability" → "Usable Workflows + Observability", verified review `docs/reviews/06` committed,
> all 12 EPIC canvases Aligned, EPIC-letter scheme reconciled #467→O/#469→R; see
> [§4a Delivery Progress](#4a--delivery-progress-live-issue-state); the 2026-06-16 pass marked
> O.4/O.7/O.8/X.2/R.1/Q.4/Q.5 ✅ and aligned EPIC X #1170 + EPIC T #1171 canvases) · **Planning author:** SPDD program plan
> **Program context:** first of a three-milestone program M7 → M-dx → M8 (see [§1 Program Roadmap](#1--program-roadmap)).

This document is the **single planning source of truth** for M7. It is generated from
the milestone brief and reconciled against **live GitHub + live stack state** (the local
stack was started and exercised on 2026-06-15; findings are recorded in
[§4 Reality Check](#4--reality-check-what-zynax-can-and-cannot-do-today)). Every EPIC maps
to SPDD canvases and GitHub issues; every implementation issue traces back to a specification.

The brief asked for 22 deliverables and an acceptance matrix; the map from brief →
section is in [Appendix A](#appendix-a--brief-deliverable-coverage-map).

---

## 0 — TL;DR

M6 shipped a production-ready **platform** (K8s, mTLS, Postgres, EventBus, Helm, SDK on
PyPI). But a developer cannot yet do **real work** with it: workflows cannot pass data
between states, you cannot stream execution logs, and there is no telemetry. M7 closes
that gap with one **vertical slice** — *author a real workflow, run it locally, watch it
execute end-to-end with traces/metrics/logs* — and lays the substrate for the expert-agent
system and example catalog that M-dx scales out.

**M7 ships when:** a developer runs `docker compose up` (platform **+ Uptrace**), applies a
real multi-step workflow with data flowing state→state, streams its logs, and sees the full
distributed trace (api-gateway → compiler → engine → broker → registry → agent) in Uptrace —
with green `make ci` and the supply-chain/coverage gaps from the reality check closed.

---

## 1 — Program Roadmap

The brief describes ~3 milestones of work. Rather than one unbounded milestone (which fights
the Agentic Delivery Playbook's "thin vertical slices, small mergeable PRs" principle), it is
sequenced as a **program** over the three already-planned GitHub milestones. This plan defines
all three at EPIC altitude and **M7 in full executable detail**.

```mermaid
graph LR
  M7["M7 — Usable Workflows + Observability<br/>v0.6.0 · milestone #7<br/>data-flow · log streaming · OTEL+Uptrace<br/>context propagation · Git MCP shim<br/>expert substrate + agents/examples<br/>first real workflows · quick-start docs"]
  MDX["M-dx — Developer Experience<br/>v0.7.0 · milestone #9<br/>full 14-expert library · example catalog<br/>advanced context (RAG/memory/compression)<br/>replay · visualization · debugging<br/>WAVEs gates at scale · authoring docs"]
  M8["M8 — CNCF Sandbox Submission<br/>v1.0.0 · milestone #8<br/>prod observability deploy · chaos/load/perf<br/>sampling/retention at scale · migration guides<br/>full docs suite · CNCF governance artifacts"]
  M7 --> MDX --> M8
```

| Milestone | Version | GH # | Theme | Why it is a discrete slice |
|-----------|---------|------|-------|----------------------------|
| **M7** | v0.6.0 | #7 | **Usable Workflows + Observability** | The minimum to do *one real workflow* with full telemetry. Everything else depends on data-flow + observability existing first. |
| **M-dx** | v0.7.0 | #9 | **Developer Experience** | Scales experts + examples + context once the substrate exists. Pure additive; no platform contract changes. |
| **M8** | v1.0.0 | #8 | **CNCF Sandbox** | Production hardening, scale testing, governance — gated on a feature-complete, observable platform. |

> **Reframing note.** GitHub milestone #7 was originally titled *"Full Observability"*. M7 keeps
> observability as a pillar but is **broader**: observability is necessary but not sufficient for
> "usable". The GitHub milestone was **renamed to "Usable Workflows + Observability (M7)" on
> 2026-06-17** to match this scope (`state/milestone.yaml` synced in #1300).

---

## 2 — Vision & Problem Statement

**Vision.** Zynax is the **control plane for agentic software delivery**: declarative YAML
workflows dispatch capabilities to pluggable agents, executed durably on pluggable engines,
fully observable. A developer should author a workflow, run it locally in one command, and
watch every step — without writing orchestration code.

**Problem (today, empirically verified 2026-06-15).** The platform runs, but:

1. **Workflows can't pass data.** `output:` bindings are rejected at compile time
   (`"not yet implemented; … upgrade to M7+"`). Every state is isolated — no real pipeline
   is expressible. *This is the keystone blocker.*
2. **No execution visibility.** `GET /api/v1/workflows/{id}/logs` returns
   `{"error":"streaming not supported"}`. There is no OTEL, no Prometheus scrape, no trace.
3. **No reference agents.** `agents/examples/` does not exist; the SDK has no canonical
   "write-your-own-agent" example, and there is no runtime expert pattern.
4. **No authoring ergonomics.** No reusable workflow/task/expert templates; no Git MCP surface
   for agent/expert authoring; quick-start docs are missing.
5. **Quality debt blocks a clean `make ci`.** `security-agents` fails on a tools-image `pip`
   CVE; `test-coverage` fails on an interface-only package; PyPI Trusted Publisher history is
   undocumented.

**Outcome.** Close 1–5 so the [§0 acceptance scenario](#0--tldr) passes.

---

## 3 — SPDD Discipline (how every spec is generated)

Per ADR-019, every `feat:` issue is preceded by a **REASONS Canvas** (status `Aligned`) before
any implementation. This plan front-loads the SPDD artifacts:

- **One canvas per EPIC** is committed with this plan under `docs/spdd/<letter>-<slug>/canvas.md`
  (all 10: [W](../spdd/1167-workflow-data-flow/canvas.md), [L](../spdd/468-log-streaming/canvas.md),
  [O](../spdd/467-observability-otel-uptrace/canvas.md), [C](../spdd/1168-context-propagation/canvas.md),
  [G](../spdd/1169-git-mcp-shim/canvas.md), [X](../spdd/1170-expert-substrate/canvas.md),
  [T](../spdd/1171-templates-real-workflows/canvas.md), [R](../spdd/469-test-rigor/canvas.md),
  [Q](../spdd/1172-quality-supply-chain/canvas.md), [D](../spdd/1173-docs/canvas.md)). Each canvas's
  **O — Operations** section lists the EPIC's stories in `spdd-story` form (As-a / I-want / so-that,
  size, acceptance criteria, out-of-scope, dependencies) — ready to become one GitHub issue each.
  `feat:` canvases (W/L/O/C/G/X/T) are the binding SPDD artifact; R/Q/D canvases are committed for
  traceability even though those types are SPDD-exempt.
- **SPDD command runbook** for every EPIC is in [§9](#9--spdd-command-runbook).
- `fix:`, `refactor:`, `docs:`, `ci:`, `chore:`, `test:` issues are **SPDD-exempt** (no canvas).

Traceability chain enforced for every issue:

```
ROADMAP/brief  →  this plan (EPIC)  →  REASONS Canvas  →  ADR (if one-way door)
               →  GitHub issue (labels/DoD/AC)  →  PR (Canvas-linked)  →  test + telemetry validation
```

---

## 4 — Reality Check (what Zynax can and cannot do today)

Verified by booting the full stack (`make run-local`, 13 containers healthy) and the test
suite on 2026-06-15.

> **⚠ Reconciliation note (2026-06-16).** The original §4 (2026-06-15) was a point-in-time snapshot.
> Re-verified against the tree at HEAD, **three rows it listed as "cannot do" are already done** and
> several "OPEN" stories already have substantial groundwork. Corrected rows are struck/annotated
> below; the authoritative per-issue picture is [§4a Delivery Progress](#4a--delivery-progress-live-issue-state).

### Works today ✅
| Capability | Evidence |
|------------|----------|
| Full local stack boots | 13 containers healthy after compose-corruption fix (PR #1166) |
| Apply a (single-state) workflow | `POST /api/v1/apply` `e2e-demo.yaml` → `run_id`, runs to `WORKFLOW_STATUS_COMPLETED` |
| Query workflow status | `GET /api/v1/workflows/{id}` → status JSON |
| Spec validation | `make validate-spec` — all 5 validators pass |
| Contract + unit + SDK tests | `make test-bdd`, `test-unit-go`, `test-unit-agents` (100% SDK cov) green |
| Proto/Python lint, Go vuln scan | `make lint-protos`, `lint-agents`, `security-go` clean |
| **OTEL trace + RED-metric core (Go)** ✅ *(was listed as missing)* | `libs/zynaxobs/tracing.go` `InitTracer()` OTLP/gRPC + W3C `traceparent`; `metrics.go` `zynax_grpc_requests_total`/`_duration_seconds`; `TracingStatsHandler()` wired in all 7 services' `main.go`. Backend/UI (Uptrace) is what's missing — see EPIC O |
| **`/logs` SSE plumbing** ✅ *(was listed as stubbed)* | `services/api-gateway/internal/api/handler.go:133` `handleWorkflowLogs` streams `text/event-stream` via `WatchWorkflowLogs`; `"streaming not supported"` is only a defensive `http.Flusher` guard. Event-merge (engine history + capability events) still needs L.2 |
| **pip CVE fixed; coverage gate honest; PyPI provenance recorded** ✅ | `infra/docker/Dockerfile.tools:152` pins `pip==26.1.2`; Q.2 #1213 + Q.3 #1214 closed |

### Cannot do yet — **M7 scope** ❌ *(corrected)*
| Gap | Evidence | Owning EPIC |
|-----|----------|-------------|
| Workflow **state→state data flow** (`output:`) — **keystone** | compiler rejects `output:` at `services/workflow-compiler/internal/domain/manifest.go:255-263` ("not yet implemented; … upgrade to M7+"); `ActionIR` in `protos/zynax/v1/workflow_compiler.proto` has no output fields. Reference `research-task.yaml`/`code-review.yaml` ship `output:` and therefore **do not compile today** | **EPIC W** (#1167) |
| **Observability backend/UI** (Uptrace) + Python-adapter OTEL + log export + exemplars | Go OTEL core exists (above) but no Uptrace backend, no `infra/docker-compose/docker-compose.observability.yml`, **no `helm/charts/uptrace/`** (charts live at `helm/charts/`, *not* `infra/helm/`), no OTLP log export, no Python OTEL | **EPIC O** (#467) |
| `/logs` **event-merge + CLI follower** | SSE plumbing present; missing per-workflow event subscription (L.2 #1181) + `zynax logs --follow` (L.4 #1183) | **EPIC L** (#468) |
| `agents/examples/` reference agents | directory does not exist | **EPIC X** (#1170) |
| Git **MCP shim** | no `.mcp.json`, no MCP server, no `zynax mcp` | **EPIC G** (#1169) |
| **Benchmark / fuzz / required-integration / e2e** CI gates | no `Fuzz*.go`, no bench/fuzz workflow; integration advisory-only | **EPIC R** (#469) |
| ~~`make lint-go` / `make security-agents` (pip 26.1.1)~~ | **DONE** — pip `26.1.2` pinned; Q.1 #1212 closed | ~~EPIC Q~~ ✅ |
| ~~`make test-coverage` (interface-only event-bus)~~ | **DONE** — honest coverage gate; Q.2 #1213 closed | ~~EPIC Q~~ ✅ |
| ~~PyPI Trusted Publisher provenance history~~ | **DONE** — recorded (§14); Q.3 #1214 closed | ~~EPIC Q~~ ✅ |

The remaining open rows are tracked explicitly in the acceptance matrix ([§13](#13--acceptance-matrix)).

---

## 4a — Delivery Progress (live issue state)

> Reconciled to GitHub milestone #7 + tree at HEAD on **2026-06-16**. Every M7 EPIC and story issue
> already exists on GitHub (created 2026-06-15). Status legend: ✅ closed · ⬜ open · 🟡 open but
> substantial groundwork in tree (orchestrate as **verify + extend**, not green-field).

| Story | Issue | EPIC | Status | Notes |
|-------|-------|------|--------|-------|
| W.1 ADR-029 data-flow semantics | [#1175](https://github.com/zynax-io/zynax/issues/1175) | W #1167 | ✅ | ADR-029 Accepted |
| W.2 proto output/input fields + `.feature` | [#1176](https://github.com/zynax-io/zynax/issues/1176) | W #1167 | ⬜ | **keystone — claim first** |
| W.3 compiler compile/validate; lift rejection | [#1177](https://github.com/zynax-io/zynax/issues/1177) | W #1167 | ⬜ | removes `manifest.go:255-263` |
| W.4 engine workflow-scoped data context | [#1178](https://github.com/zynax-io/zynax/issues/1178) | W #1167 | ⬜ | dep W.2/W.3 |
| W.5 real workflows run green e2e | [#1179](https://github.com/zynax-io/zynax/issues/1179) | W #1167 | ⬜ | dep W.4 |
| L.1 Temporal history long-poll | [#1180](https://github.com/zynax-io/zynax/issues/1180) | L #468 | ✅ | PR #1237 (old dup #492 closed) |
| L.2 per-workflow event subscription stream | [#1181](https://github.com/zynax-io/zynax/issues/1181) | L #468 | ⬜ | unblocks L.3 event-merge |
| L.3 streaming `GET /workflows/{id}/logs` | [#1182](https://github.com/zynax-io/zynax/issues/1182) | L #468 | 🟡 | SSE plumbing in `handler.go:133`; needs L.2 merge |
| L.4 `zynax logs --follow` | [#1183](https://github.com/zynax-io/zynax/issues/1183) | L #468 | ⬜ | alias over existing SSE |
| O.1 ADR-030 OTEL + Uptrace | [#1184](https://github.com/zynax-io/zynax/issues/1184) | O #467 | ✅ | ADR-030 Accepted |
| O.2 shared OTEL providers package | [#1185](https://github.com/zynax-io/zynax/issues/1185) | O #467 | 🟡 | **exists as `libs/zynaxobs`** (planned name `zynaxotel` is wrong) — extend, don't recreate |
| O.3 instrument all 7 services | [#1186](https://github.com/zynax-io/zynax/issues/1186) | O #467 | 🟡 | `TracingStatsHandler()` already wired in every `main.go` — verify+gap-fill |
| O.4 RED metrics + exemplars | [#1187](https://github.com/zynax-io/zynax/issues/1187) | O #467 | ✅ | trace_id exemplars on RED metrics (PR #1268) |
| O.5 trace propagation Temporal + NATS | [#1188](https://github.com/zynax-io/zynax/issues/1188) | O #467 | ⬜ | W3C propagator installed; verify Temporal/NATS hops |
| O.6 Python adapter OTEL (traces+logs) | [#1189](https://github.com/zynax-io/zynax/issues/1189) | O #467 | ⬜ | |
| O.7 local Uptrace docker-compose stack | [#1190](https://github.com/zynax-io/zynax/issues/1190) | O #467 | ✅ | `infra/docker-compose/docker-compose.observability.yml` (PR #1259) |
| O.8 Uptrace Helm chart | [#1191](https://github.com/zynax-io/zynax/issues/1191) | O #467 | ✅ | **`helm/charts/uptrace/`** (not `infra/helm/`) (PR #1267) |
| O.9 ship logs to Uptrace via OTLP + naming | [#1192](https://github.com/zynax-io/zynax/issues/1192) | O #467 | ⬜ | |
| C.1 ADR-031 context model | [#1193](https://github.com/zynax-io/zynax/issues/1193) | C #1168 | ✅ | issue closed; **ADR-031 still `Proposed` → accept on canvas-C alignment** |
| C.2 propagate correlation across hops | [#1194](https://github.com/zynax-io/zynax/issues/1194) | C #1168 | 🟡 | request-id+traceparent partially flow (`requestid.go`, `clients.go:276-302`) |
| C.3 data-context read/write scoping | [#1195](https://github.com/zynax-io/zynax/issues/1195) | C #1168 | ⬜ | dep W |
| C.4 agent handoff context contract | [#1196](https://github.com/zynax-io/zynax/issues/1196) | C #1168 | ⬜ | |
| G.1 ADR-032 Git MCP shim | [#1197](https://github.com/zynax-io/zynax/issues/1197) | G #1169 | ✅ | ADR-032 Accepted |
| G.2 MCP server over git-adapter | [#1198](https://github.com/zynax-io/zynax/issues/1198) | G #1169 | ⬜ | |
| G.3 credential injection + redaction | [#1199](https://github.com/zynax-io/zynax/issues/1199) | G #1169 | ⬜ | Tier-2 security review |
| G.4 `zynax mcp git` + `.mcp.json` | [#1200](https://github.com/zynax-io/zynax/issues/1200) | G #1169 | ⬜ | |
| G.5 least-privilege token scope validation | [#1260](https://github.com/zynax-io/zynax/issues/1260) | G #1169 | ⬜ | git-adapter hardening; Tier-2 review |
| G.6 docs: fine-grained PAT + no-refresh note | [#1261](https://github.com/zynax-io/zynax/issues/1261) | G #1169 | ⬜ | SPDD-exempt docs; READY now |
| G.7 refreshable credentials (App tokens) | [#1262](https://github.com/zynax-io/zynax/issues/1262) | G #1169 | ⬜ | git-adapter hardening; Tier-2 review |
| X.1 ADR-033 expert substrate | [#1201](https://github.com/zynax-io/zynax/issues/1201) | X #1170 | ✅ | issue closed; **ADR-033 still `Proposed` → accept on canvas-X alignment** |
| X.2 `agents/examples/` reference agents | [#1202](https://github.com/zynax-io/zynax/issues/1202) | X #1170 | ✅ | echo, summarizer, go-review-expert (PR #1270) |
| X.3 runtime expert (`kind: AgentDef`) | [#1203](https://github.com/zynax-io/zynax/issues/1203) | X #1170 | ⬜ | |
| X.4 discover + test `agents/examples/*` | [#1204](https://github.com/zynax-io/zynax/issues/1204) | X #1170 | ⬜ | |
| X.5 authoring↔runtime mapping + CI check | [#1205](https://github.com/zynax-io/zynax/issues/1205) | X #1170 | ⬜ | |
| T.1 reusable templates + versioning | [#1206](https://github.com/zynax-io/zynax/issues/1206) | T #1171 | ⬜ | dep X |
| T.2 `zynax validate` + version surfacing | [#1207](https://github.com/zynax-io/zynax/issues/1207) | T #1171 | ⬜ | |
| T.3 three real runnable workflows | [#1208](https://github.com/zynax-io/zynax/issues/1208) | T #1171 | ⬜ | dep W,X |
| T.4 `zynax init workflow\|expert` | [#1209](https://github.com/zynax-io/zynax/issues/1209) | T #1171 | ⬜ | |
| R.1 benchmarks + regression gate | [#493](https://github.com/zynax-io/zynax/issues/493) | R #469 | ✅ | IRInterpreter+ParseManifest benches + benchstat gate (PR #1269) |
| R.2 fuzz YAML→IR compiler | [#1210](https://github.com/zynax-io/zynax/issues/1210) | R #469 | ⬜ | |
| R.3 integration suite as required gate | [#553](https://github.com/zynax-io/zynax/issues/553) | R #469 | ⬜ | |
| R.4 flip platform-readiness xfail to real e2e | [#1103](https://github.com/zynax-io/zynax/issues/1103) | R #469 | ⬜ | carried from M6 |
| R.5 observability-validation trace assertion | [#1211](https://github.com/zynax-io/zynax/issues/1211) | R #469 | ⬜ | dep O |
| Q.1 bump tools-image pip → 26.1.2 | [#1212](https://github.com/zynax-io/zynax/issues/1212) | Q #1172 | ✅ | |
| Q.2 honest coverage gate | [#1213](https://github.com/zynax-io/zynax/issues/1213) | Q #1172 | ✅ | |
| Q.3 PyPI Trusted Publisher provenance | [#1214](https://github.com/zynax-io/zynax/issues/1214) | Q #1172 | ✅ | §14 |
| Q.4 verify pkg.go.dev consumption path | [#1215](https://github.com/zynax-io/zynax/issues/1215) | Q #1172 | ✅ | old dup #582 closed (PR #1258) |
| Q.5 ADR-034 ManifestWorkflowID collision domain | [#1216](https://github.com/zynax-io/zynax/issues/1216) | Q #1172 | ✅ | old dup #583 closed; ADR-034 finalized to the real crypto/rand id scheme (PR #1257) |
| D.1 Quick Start + Developer Guide | [#1217](https://github.com/zynax-io/zynax/issues/1217) | D #1173 | ⬜ | dep T.3, O.7 |
| D.2 Workflow + Expert authoring guides | [#1218](https://github.com/zynax-io/zynax/issues/1218) | D #1173 | ⬜ | |
| D.3 Context + Git MCP guides | [#1219](https://github.com/zynax-io/zynax/issues/1219) | D #1173 | ⬜ | |
| D.4 Observability (OTEL + Uptrace) guides | [#1220](https://github.com/zynax-io/zynax/issues/1220) | D #1173 | ⬜ | |
| D.5 Examples index + best practices + FAQ + migration | [#1221](https://github.com/zynax-io/zynax/issues/1221) | D #1173 | ⬜ | |
| P.1 ADR-035 adapter language boundary | [#1277](https://github.com/zynax-io/zynax/issues/1277) | P #1276 | ✅ | ADR-035 Accepted; canvas-P Aligned |
| P.2 Go scaffold + config | [#1278](https://github.com/zynax-io/zynax/issues/1278) | P #1276 | ⬜ | `GOWORK=off` module; dep P.1 |
| P.3 Go providers (OpenAI/Bedrock/Ollama) | [#1279](https://github.com/zynax-io/zynax/issues/1279) | P #1276 | ⬜ | dep P.2 |
| P.4 Go AgentService server (parity) | [#1280](https://github.com/zynax-io/zynax/issues/1280) | P #1276 | ⬜ | reuses `llm_adapter.feature`; dep P.3 |
| P.5 registry + bootstrap + health | [#1281](https://github.com/zynax-io/zynax/issues/1281) | P #1276 | ⬜ | dep P.4 |
| P.6 Dockerfile + images.yaml cutover | [#1282](https://github.com/zynax-io/zynax/issues/1282) | P #1276 | ⬜ | dep P.5; ADR-024 |
| P.7 retire Python llm-adapter | [#1283](https://github.com/zynax-io/zynax/issues/1283) | P #1276 | ⬜ | dep P.6 |
| S.1 ADR-036 CI logic as a Go CLI | [#1286](https://github.com/zynax-io/zynax/issues/1286) | S #1285 | ✅ | ADR-036 Accepted; canvas-S Aligned |
| S.2 `zynax-ci coverage-comment` | [#1287](https://github.com/zynax-io/zynax/issues/1287) | S #1285 | ⬜ | ← build-coverage-comment.sh; dep S.1 |
| S.3 `zynax-ci bench-gate` + `bdd-select` | [#1288](https://github.com/zynax-io/zynax/issues/1288) | S #1285 | ⬜ | ← bench-regression + bdd-select scripts; dep S.1 |
| S.4 `zynax-ci bump-runner` | [#1289](https://github.com/zynax-io/zynax/issues/1289) | S #1285 | ⬜ | ← bump-ci-runner.sh; images.yaml SoT; dep S.1 |
| S.5 `zynax-ci images meta/cleanup/retag` | [#1290](https://github.com/zynax-io/zynax/issues/1290) | S #1285 | ⬜ | ← report-image-meta + cleanup/retag blocks; dep S.1 |
| S.6 `zynax-ci release` helpers | [#1291](https://github.com/zynax-io/zynax/issues/1291) | S #1285 | ⬜ | ← release.yml assembly; cosign stays shell; dep S.5 |
| S.7 retire scripts + thin workflows | [#1292](https://github.com/zynax-io/zynax/issues/1292) | S #1285 | ⬜ | dep S.2–S.6 |

**Tally:** 11 closed / 53 open across 12 EPICs.

### Ready to claim now for `/milestone-orchestrate` (priority order, ≤3 per batch)
**Strictly unblocked** = every dependency is closed/none. Each EPIC's `*.1` ADR step is closed, so:

| # | Issue | Why ready | Note |
|---|-------|-----------|------|
| 1 | **W.2 [#1176]** | dep W.1 #1175 ✅ | **keystone — claim first** (W/C/T/X data-flow all gate on it) |
| 2 | **O.2 [#1185]** | dep O.1 #1184 ✅ | 🟡 verify+extend `libs/zynaxobs`; unblocks O.4/O.6/O.7 |
| 3 | **L.2 [#1181]** | dep L.1 #1180 ✅ | unblocks `/logs` event-merge (L.3) |

Also strictly-ready for later batches (deps closed/none): **G.2 [#1198]**, **X.2 [#1202]**,
**R.1 [#493]**, **R.3 [#553]**, **R.4 [#1103]**, **Q.4 [#1215]**, **Q.5 [#1216]**,
**G.5 [#1260]**, **G.6 [#1261]** (docs, no canvas — claim anytime), **G.7 [#1262]**,
**P.1 [#1277]** (ADR/docs — accept ADR-035 on canvas-P alignment; gates P.2→P.7),
**P.2–P.6 [#1278–#1282]** (canvas-P Aligned; P.2 ready after P.1, then chain),
**S.1 [#1286]** (ADR/docs — accept ADR-036 on canvas-S alignment; gates S.2→S.7),
**S.2–S.5 [#1287–#1290]** (canvas-S Aligned; parallel after S.1; S.6 dep S.5; S.7 last).
Unblock after W.2 merges: **W.3 [#1177]** → then **R.2 [#1210]**, **T.1 [#1206]** (both dep W.3).
Do **not** front-load O.7 #1190 / O.8 #1191 — they depend on O.2 #1185.

### State-hygiene flags (for a maintainer; not auto-fixable here)
- **Closed duplicate issues** already superseded — harmless to orchestrate but worth closing-as-dup
  with a note: #492 (old L.1), #582 (old Q.4), #583 (old Q.5).
- **EPIC #468 is CLOSED on GitHub** but is listed in `state/milestone.yaml › open_epics` and remains
  the parent of open L.2–L.4. Its stories are live; only the EPIC umbrella is closed. `milestone.yaml`
  is mutated **only** by `/milestone-new` / `/milestone-close` — flag for the next milestone op.
- **EPIC #467 title** still reads the pre-absorption "Wire Prometheus + OTel"; scope is the broader
  OTEL+Uptrace EPIC O. Cosmetic; retitle on next touch.
- **ADRs 031 / 033 / 034 are still `Proposed`** while their docs issues are closed — promote to
  `Accepted` as each canvas (C / X / Q) reaches `Aligned`. (ADR-035 / ADR-036 are already `Accepted`
  — canvases P #1276 and S #1285 are Aligned.)
- **Issue dependency markers are inconsistent.** Many open stories express deps as bare step
  labels (e.g. #1177 `Dependencies: W.2`, #1186 `Dependencies: O.2`) with no issue number, while
  others use `#N` (e.g. #1195, #1208). The §4a table above is the authoritative step→`#N` map the
  orchestrator should use to classify BLOCKED vs READY. **Recommended one-time normalization**
  (a `gh issue edit` pass, awaiting maintainer go-ahead): rewrite each bare-label dep to
  `Depends on #N` so `/milestone-orchestrate` STEP 3 detects blockers without cross-referencing.
  Step-label-only issues needing it: #1177 #1178 #1179 #1182 #1183 #1186 #1187 #1188 #1189 #1190
  #1191 #1192 #1196 #1199 #1200 #1204 #1207 #1209.

---

## 5 — EPIC Decomposition (M7)

Ten original EPICs plus the **first-run UX closeout EPIC K** (#1370, added in the 2026-06-18 reframe).
Three pre-exist on GitHub (#467/#468/#469) and are **absorbed/extended** rather than
duplicated. New EPICs are created by this plan ([§10](#10--github-bootstrap)).

| EPIC | Title | Type | GitHub issue | Primary area |
|------|-------|------|--------------|--------------|
| **W** | Workflow data-flow (`output:`/input bindings) | feat | **#1167** (new) | workflow-compiler, engine-adapter, protos |
| **L** | Execution log/event streaming (`/logs`) | feat | extends **#468** | api-gateway, engine-adapter, event-bus |
| **O** | Observability: OTEL + Uptrace + Prometheus | feat | absorbs **#467** | all services + adapters, infra |
| **C** | Context propagation (trace + data + correlation) | feat | **#1168** (new) | protos, all services, engine-adapter |
| **G** | Git MCP shim over git-adapter | feat | **#1169** (new) | agents/adapters, cli |
| **X** | Expert-agent substrate + `agents/examples/` | feat | **#1170** (new) | agents/sdk, agents/examples, agent-registry |
| **T** | Reusable templates + first real workflows | feat | **#1171** (new) | spec, cli, docs |
| **R** | Test rigor: benchmarks, fuzz, integration/e2e gates | test | absorbs **#469** (+#553 #493 #1103) | ci, multiple |
| **Q** | Quality & supply-chain fixes (audit closeout) | chore/ci | **#1172** (new) | ci, infra, docs |
| **D** | Docs: quick-start + authoring + observability | docs | **#1173** (new) | docs |
| **K** | First-run User Experience — zero-secret Ollama quickstart (✅ core path delivered) | feat | **#1370** (reframe 2026-06-18) | api-gateway, engine-adapter, adapters, cli, spec, infra, docs |

> EPIC issues created 2026-06-15 (W/C/G/X/T/Q/D = #1167–#1173). Pre-existing #467/#468/#469
> annotated to point at canvases O/L/R. Story issues are created per-EPIC via `/spdd-story`
> ([§9](#9--spdd-command-runbook)).

### EPIC W — Workflow data-flow *(keystone; everything depends on it)*
**Why first:** real workflows and expert pipelines are impossible without state→state data.
The compiler explicitly rejects `output:` today.
- **W.1** [#1175 ✅] ADR: data-flow semantics & scoping model — [ADR-029](../adr/ADR-029-workflow-data-flow.md) (Accepted)
- **W.2** [#1176] Proto: `output`/`input` binding fields on `WorkflowIR` state/action + `.feature` (feat, new gRPC boundary → `/spdd-api-test`) — **keystone, claim first**
- **W.3** [#1177] Compiler: compile `output:` bindings to IR; validate references; **lift the rejection at `services/workflow-compiler/internal/domain/manifest.go:255-263`**
- **W.4** [#1178] Engine-adapter: interpreter threads outputs into a workflow-scoped data context; inputs resolve from it
- **W.5** [#1179] End-to-end: make `research-task.yaml` + `code-review.yaml` apply and run green (they ship `output:` today and therefore do not compile)
**DoD:** `apply research-task.yaml` reaches terminal state with `summarize` consuming `search`'s output; BDD scenario for data-flow; ≥90% domain cov.

### EPIC L — Execution log/event streaming
**Why:** the `/logs` endpoint is stubbed; developers can't watch a run. Couples with #468
(replace polling with Temporal history long-poll) on the engine side.
- **L.1** [#1180 ✅] Engine-adapter: history streaming (long-poll `GetWorkflowHistory`) — merged (PR #1237)
- **L.2** [#1181] Event-bus: per-workflow event subscription stream (reuse `Subscribe` w/ `WorkflowID` scope)
- **L.3** [#1182 🟡] api-gateway: SSE `GET /api/v1/workflows/{id}/logs` — **plumbing already in `handler.go:133`**; remaining work is merging engine history + L.2 capability events
- **L.4** [#1183] CLI: `zynax logs <run-id> --follow`
**DoD:** streaming logs for the e2e-demo run show each state transition + capability event; no `streaming not supported`.

### EPIC O — Observability: OTEL + **Uptrace (traces · metrics · logs · APM, with login UI)** + Prometheus *(absorbs #467)*
**Backend decision:** **Uptrace** as the single default OTEL backend for **traces, metrics, logs,
and APM**, with its **web UI (login) for viewing logs and service maps**; **OTLP/gRPC** default,
OTLP/HTTP optional. No Jaeger/Loki/Elasticsearch (see [ADR-030 stub](../adr/ADR-030-observability-uptrace.md)).
Uptrace ships in **both** the local `docker compose` stack **and** the Helm deployment, so a
developer always has a UI to see logs/traces — locally and in-cluster.
> **Reconciliation:** the Go OTEL **trace + RED-metric core already exists** in **`libs/zynaxobs`**
> (`tracing.go` OTLP/gRPC + W3C propagator, `metrics.go` RED counters, `TracingStatsHandler()` wired
> in all 7 `main.go`). O.2/O.3 are therefore **verify + extend on `libs/zynaxobs`** (the planned name
> `libs/zynaxotel` has zero references — do not create a parallel package). The net-new work is the
> **Uptrace backend + UI, the OTLP collector, log export, exemplars, and Python-adapter OTEL.**
- **O.1** [#1184 ✅] ADR: OTEL + Uptrace — [ADR-030](../adr/ADR-030-observability-uptrace.md) (Accepted)
- **O.2** [#1185 🟡] **Extend `libs/zynaxobs`** with the logger provider + log OTLP exporter (tracer/meter already present)
- **O.3** [#1186 🟡] Verify gRPC server/client interceptors across all 7 services (already wired) + add HTTP middleware on api-gateway
- **O.4** [#1187 🟡] RED metrics exist — add **exemplars**; forward to Uptrace metrics
- **O.5** [#1188] Trace propagation across Temporal activities + NATS headers (W3C `traceparent`)
- **O.6** [#1189] Python adapters: OTEL SDK (traces+logs) in `agents/sdk`; auto-instrument capability handlers
- **O.7** [#1190] **`docker compose up` Uptrace stack** (`infra/docker-compose/docker-compose.observability.yml`): Uptrace + its ClickHouse/Postgres deps + OTLP collector; **login UI on a 70xx host port** for logs/traces/APM
- **O.8** [#1191] **Uptrace Helm chart** (**`helm/charts/uptrace/`** — charts live at `helm/charts/`, *not* `infra/helm/`): Deployment/Service/Ingress + login UI, wired as the in-cluster OTLP endpoint; values toggle (`observability.enabled`)
- **O.9** [#1192] **Log export to Uptrace** (structured logs shipped via OTLP logs) so logs are viewable in the Uptrace UI alongside traces; span/metric naming conventions doc + `trace_id`/`span_id` in every log line
**DoD:** one workflow run produces a connected trace across all hops **and its logs**, viewable
in the **Uptrace login UI** (local compose **and** Helm); RED metrics scraped; APM/service-map populated.

### EPIC C — Context propagation
**Why:** experts and multi-step workflows need deterministic context (both **trace context**
and **workflow data context**) to flow across service, engine, and agent boundaries.
- **C.1** [#1193 ✅] ADR: context model — [ADR-031](../adr/ADR-031-context-propagation.md) (issue closed; **promote ADR `Proposed`→`Accepted`** on canvas alignment)
- **C.2** [#1194 🟡] Propagate `traceparent` + `x-request-id` + `x-namespace` through every gRPC hop and Temporal memo (request-id + traceparent partially flow via `requestid.go` + `clients.go:276-302`)
- **C.3** [#1195] Workflow-scoped data context store (builds on EPIC W) with explicit read/write scoping
- **C.4** [#1196] Documented handoff contract between agents (what context an agent receives/returns)
**DoD:** a request-id set at api-gateway appears in every downstream span and log line for that run; documented in the context guide.

### EPIC G — Git MCP shim over git-adapter
**Decision:** MCP is a **thin protocol shim over the existing git-adapter** (one Git
implementation, two surfaces) — see [ADR-032 stub](../adr/ADR-032-git-mcp-shim.md).
- **G.1** [#1197 ✅] ADR: MCP-shim-over-adapter — [ADR-032](../adr/ADR-032-git-mcp-shim.md) (Accepted)
- **G.2** [#1198] MCP server exposing git-adapter capabilities (clone/branch/commit/PR/review) as MCP tools
- **G.3** [#1199] Credential injection via env/secret ref at process start; redaction in logs/traces
- **G.4** [#1200] CLI/dev wiring: `zynax mcp git` + `.mcp.json` example for Claude Code authoring loop
- **G.5** [#1260] git-adapter **least-privilege token scope validation** at startup — fail-fast/warn when the token can reach repos beyond the configured `owner/repo` (`config.go:101` resolves the token but never checks scope; `owner/repo` pinning is an adapter guard, not a credential restriction)
- **G.6** [#1261] **docs**: recommend a fine-grained PAT scoped to the configured repo (the example + `AGENTS.md:24` currently recommend broad `repo` scope) and document that the token is read once at startup with no refresh — *(SPDD-exempt docs; independently mergeable now)*
- **G.7** [#1262] git-adapter **refreshable credentials** — re-resolve / mint GitHub App installation tokens so a short-lived (~1 h) token does not expire mid-process (`main.go:45` resolves once; no refresh or App flow today)
**DoD:** an authoring session can open a PR via MCP with a scoped token; no token ever serialized into a prompt; security review PASS.

> **G.5–G.7 provenance.** Surfaced during the EPIC-G credential review of the existing
> git-adapter (the substrate the MCP shim wraps): the adapter *supports* restricted tokens
> but does not *enforce* scope (G.5), its docs recommend an over-broad `repo` PAT (G.6), and it
> has no token refresh / App-token path (G.7). G.5/G.7 are `feat:` (canvas O-step required
> before `/spdd-generate` — the canvas is intentionally left `Aligned` here so the in-flight
> G.2–G.4 stories are not reset to `Draft`; `/milestone-orchestrate` routes them through
> `spdd-canvas` first per STEP 5). G.6 is SPDD-exempt.

### EPIC X — Expert-agent substrate + `agents/examples/`
**Decision (both substrates, mapped):** runtime **AgentDef** experts (registered, dispatched,
OTEL-traced) for in-workflow execution **and** Claude Code experts (extend
`automation/workflows/experts/*.yaml` + `.claude` skills) for the delivery/authoring loop, with
a documented mapping ([ADR-033 stub](../adr/ADR-033-expert-agent-substrate.md)).
- **X.1** [#1201 ✅] ADR: expert substrate + runtime↔authoring mapping — [ADR-033](../adr/ADR-033-expert-agent-substrate.md) (issue closed; **promote ADR `Proposed`→`Accepted`** on canvas alignment)
- **X.2** [#1202] Create `agents/examples/` with three reference agents (uses SDK): `echo`, `summarizer`, `go-review-expert`
- **X.3** [#1203] Runtime expert pattern: `kind: AgentDef` expert template + registration + capability schema
- **X.4** [#1204] Wire `make lint-agent`/`test-unit-agent` to discover `agents/examples/*`
- **X.5** [#1205] Map each authoring expert (`experts/*.yaml`) to its runtime counterpart (or "authoring-only")
**DoD:** `agents/examples/` builds, lints, tests; `go-review-expert` registers and is dispatchable in a workflow with a trace.

### EPIC T — Reusable templates + first real workflows
- **T.1** [#1206] Template mechanism: workflow templates + task templates + expert templates (with versioning field)
- **T.2** [#1207] Workflow validation + versioning fields surfaced in CLI (`zynax validate`, `version:`)
- **T.3** [#1208] First **production-quality** real workflows (runnable end-to-end): `code-review`, `ci-pipeline`, `feature-implementation`
- **T.4** [#1209] `zynax init workflow|expert` scaffolds from templates
**DoD:** the three real workflows apply and run green locally with data-flow + traces; templates documented.

### EPIC R — Test rigor *(absorbs #469; pulls in #553, #493, #1103)*
- **R.1** [#493] Benchmarks for IRInterpreter + workflow-compiler with regression gate
- **R.2** [#1210] Fuzz tests for the YAML→IR compiler
- **R.3** [#553] Activate integration test suite as a required CI gate
- **R.4** [#1103] Flip platform-readiness `xfail` to real `zynax apply` e2e
- **R.5** [#1211] Observability validation test: assert a run emits a connected trace
**DoD:** benchmarks + fuzz + integration + e2e + observability tests run in CI as gates.

### EPIC Q — Quality & supply-chain fixes (audit closeout)
*(Directly closes the [§4](#4--reality-check-what-zynax-can-and-cannot-do-today) red rows that aren't features.)*
- **Q.1** [#1212 ✅] Bump tools image `pip` → 26.1.2 (PYSEC-2026-196) — done (`infra/docker/Dockerfile.tools:152`)
- **Q.2** [#1213 ✅] Honest coverage gate for interface-only packages (`event-bus/internal/domain`) — done
- **Q.3** [#1214 ✅] Document **PyPI Trusted Publisher** provenance ([§14](#14--pypi-trusted-publisher-history)) — done
- **Q.4** [#1215] Verify & document Go module consumption path (pkg.go.dev) — *(supersedes closed #582)*
- **Q.5** [#1216] ADR for `ManifestWorkflowID` 64-bit collision domain — promote [ADR-034](../adr/ADR-034-manifest-workflow-id-collision-domain.md) `Proposed`→`Accepted` *(supersedes closed #583)*
**DoD:** `make ci` is green end-to-end on a clean checkout; PyPI Trusted Publisher history present in this doc ([§14](#14--pypi-trusted-publisher-history)).

### EPIC D — Docs: quick-start + authoring + observability
- **D.1** [#1217] Quick Start (`docker compose up` → apply → watch trace) + Developer Guide
- **D.2** [#1218] Workflow Authoring + Expert Authoring guides
- **D.3** [#1219] Context System + Git MCP guides
- **D.4** [#1220] Observability + OpenTelemetry + Uptrace guides (sampling/retention/troubleshooting)
- **D.5** [#1221] Examples index + Best Practices + FAQ + Migration notes (v0.5→v0.6)
**DoD:** a new developer can go from clone to a traced real-workflow run using only the docs.

---

## 6 — Dependency Graph & Critical Path

```mermaid
graph TD
  W["W · data-flow (keystone)"]
  C["C · context propagation"]
  O["O · OTEL + Uptrace"]
  L["L · log streaming"]
  G["G · Git MCP shim"]
  X["X · expert substrate + examples"]
  T["T · templates + real workflows"]
  R["R · test rigor"]
  Q["Q · quality/supply-chain fixes"]
  D["D · docs"]

  Q --> W
  Q --> O
  W --> C
  W --> T
  W --> X
  O --> C
  O --> L
  C --> X
  C --> T
  X --> T
  G --> X
  W --> R
  O --> R
  T --> R
  T --> D
  O --> D
  G --> D
  X --> D
```

- **Critical path:** `Q → W → C → X → T → D` (data-flow and context gate the expert/example/doc work).
- **Parallelizable from the start:** `Q` (all sub-tasks), `O` (after Q.1 unblocks clean CI), `G` (independent of W).
- **Q is the unblocker:** Q.1/Q.2 must merge early so every other EPIC's PR sees a green `make ci`.

---

## 7 — Parallel Execution Plan (waves)

Designed for autonomous/parallel agent execution (≤3 concurrent per `/milestone-orchestrate`).

> **Reconciliation (2026-06-16):** Wave 0 is **complete** (Q.1 #1212 / Q.2 #1213 / Q.3 #1214 closed)
> and every EPIC's W.1/O.1/C.1/G.1/X.1 ADR step is closed. Execution now starts at **Wave 1, W.2 #1176**.

| Wave | EPICs / tasks (parallel) | Gate to advance |
|------|--------------------------|-----------------|
| **0** ✅ | ~~Q.1 #1212 · Q.2 #1213 · Q.3 #1214~~ (CI green + provenance) — **done** | `make ci` green on clean checkout ✅ |
| **1** | **W.2 #1176→W.5 #1179** (data-flow) · O.2 #1185–O.4 #1187 (verify+extend OTEL core) · G.2 #1198 (MCP server) | data-flow e2e green; trace visible in Uptrace |
| **2** | C.2 #1194–C.4 #1196 (context) · O.5 #1188–O.9 #1192 (propagation + Uptrace compose/Helm) · L.2 #1181–L.4 #1183 (log streaming) | request-id end-to-end; `/logs` streams |
| **3** | X.2 #1202–X.5 #1205 (experts + examples) · T.1 #1206–T.4 #1209 (templates + real workflows) · G.3 #1199/G.4 #1200 · G.5 #1260/G.6 #1261/G.7 #1262 (git-adapter credential hardening) | 3 real workflows run; reference agents dispatchable |
| **4** | R.1 #493–R.5 #1211 (test rigor) · D.1 #1217–D.5 #1221 (docs) · Q.4 #1215/Q.5 #1216 | all CI gates active; docs complete; acceptance matrix green |

Each task = one small, independently mergeable PR (≤400 lines target; planning/docs/spec
exempt per CLAUDE.md PR-size rules).

---

## 8 — Risk Register

| # | Risk | Likelihood | Impact | Mitigation | Owner EPIC |
|---|------|-----------|--------|------------|------------|
| 1 | Data-flow scoping (W) becomes a sprawling expression language | Med | High | ADR-029 fixes a **minimal** binding model (no expressions in M7 — literal/path refs only); defer transforms to M-dx | W |
| 2 | OTEL adds latency/overhead in the hot path | Med | Med | Head-based parent sampling default; benchmark in R.5; document overhead | O/R |
| 3 | Uptrace single-binary not prod-grade at scale | Low | Med | M7 targets local/dev; prod deploy + retention is M8 | O |
| 4 | Git MCP token leakage into prompts/traces | Low | **Critical** | ADR-032 mandates injection-at-process-start + log/trace redaction; security review gate (Tier 2) | G |
| 5 | Expert dual-substrate causes drift between runtime & authoring experts | Med | Med | ADR-033 mapping table is the SoT; CI check that every authoring expert declares its mapping | X |
| 6 | Coverage-gate change (Q.2) masks real gaps | Low | Med | Exclude only packages with **zero executable statements**; assert via `go tool cover` count, not a blanket skip | Q |
| 7 | Trace context lost across Temporal/NATS boundaries | Med | High | C.2 propagates via Temporal memo + NATS headers; R.5 asserts a connected trace | C |
| 8 | Scope creep — brief is ~3 milestones | **High** | High | Program split (M7/M-dx/M8); M-dx absorbs full expert library + example catalog + RAG | program |

---

## 9 — SPDD Command Runbook

Run from repo root. `feat:` EPICs require the full pipeline; non-feat are exempt.

```bash
# ── EPIC W — data-flow (feat, new gRPC boundary) ──────────────────────────
/spdd-analysis <W.2-issue>
/spdd-story <W-epic-issue>
/spdd-reasons-canvas <W.2-issue>          # → docs/spdd/<id>-workflow-data-flow/canvas.md
/spdd-security-review docs/spdd/<id>-workflow-data-flow/canvas.md   # must PASS
/spdd-api-test docs/spdd/<id>-workflow-data-flow/canvas.md          # new gRPC boundary → .feature first
# [human sets status: Aligned]
/spdd-generate docs/spdd/<id>-workflow-data-flow/canvas.md          # one O-step; stop; review; repeat W.3→W.5

# ── EPIC O — OTEL+Uptrace (feat) ──────────────────────────────────────────
/spdd-analysis <O-issue>; /spdd-reasons-canvas <O-issue>; /spdd-security-review <canvas>; /spdd-generate <canvas>   # per O-step

# ── EPIC C / G / X / T — same feat pipeline, one canvas per EPIC ──────────
#   /spdd-analysis → /spdd-reasons-canvas → /spdd-security-review → [Aligned] → /spdd-generate (per O-step)
#   EPIC X/T also need /spdd-api-test where a new capability schema or gRPC field is introduced.

# ── EPIC L — log streaming (feat, extends existing endpoint) ──────────────
/spdd-analysis <L-issue>; /spdd-reasons-canvas <L-issue>; /spdd-security-review <canvas>; /spdd-generate <canvas>

# ── SPDD-EXEMPT EPICs (no canvas) ─────────────────────────────────────────
#   EPIC Q: chore/ci/docs   — Q.1 ci, Q.2 ci, Q.3 docs, Q.4 docs, Q.5 adr-proposal
#   EPIC R: test            — benchmarks/fuzz/integration/e2e/observability-validation
#   EPIC D: docs            — all guides
#   #468 (engine history streaming) is refactor:; reuse its existing context, no new canvas.
```

Whole-milestone orchestration: `/milestone-plan` → `/milestone-orchestrate` (≤3 issues/batch,
routed to domain experts) → `/milestone-learn` after each cluster.

---

## 10 — GitHub Bootstrap

Milestone #7 already exists (idempotent reuse). Commands to create the **new** EPIC issues are
in [§10a](#10a--epic-create-commands). Pre-existing EPICs #467/#468/#469 and stories
#493/#553/#582/#583/#1103 are **absorbed** (re-labeled/linked, not recreated).

### 10a — EPIC create commands

> Executed by this plan via `gh` (see PR). One representative shown; the rest follow the same shape.

```bash
gh issue create \
  --title "epic(workflow-compiler): M7.W — Workflow data-flow (output/input bindings)" \
  --label "type: epic,area: workflow-compiler,area: engine-adapter,area: protos,priority: high,milestone: M7" \
  --milestone "Usable Workflows + Observability (M7)" \
  --body "<scope · stories W.1–W.5 · ADR-029 · DoD · AC>  Assisted-by: Claude/claude-opus-4-8"
# … EPICs L, O, C, G, X, T, Q, D analogously (see §5 for scope/stories).
```

---

## 11 — ADRs Required

| ADR | Title | Status | One-way door? | EPIC |
|-----|-------|--------|---------------|------|
| ADR-029 | Workflow data-flow semantics (binding model, scoping) | **Accepted** | Yes — proto contract | W |
| ADR-030 | Observability: OTEL + Uptrace, OTLP/gRPC, head sampling | **Accepted** | Yes — backend choice | O |
| ADR-031 | Context propagation model (trace vs data vs correlation) | Proposed → accept on canvas-C alignment | Yes — cross-service contract | C |
| ADR-032 | Git MCP as a thin shim over git-adapter | **Accepted** | Partly — auth model | G |
| ADR-033 | Expert-agent substrate (runtime AgentDef + authoring experts) | Proposed → accept on canvas-X alignment | Yes — agent model | X |
| ADR-034 | `ManifestWorkflowID` 64-bit collision domain (from #583) | Proposed → accept in Q.5 #1216 | No | Q |

ADR-029/030/032 are **Accepted** (per [docs/adr/INDEX.md](../adr/INDEX.md)); 031/033/034 remain
`Proposed` and should be promoted to `Accepted` as canvases C/X/Q reach `Aligned`.

---

## 12 — Observability, Security & Testing Plans

### Observability plan (default DX = `Zynax → OpenTelemetry → Uptrace`)
- Instrumentation standard: OpenTelemetry; backend: **Uptrace** (single backend for **traces,
  metrics, logs, and APM**, with a **login web UI** + service map); transport: OTLP/gRPC (HTTP optional).
- Signals: distributed traces, RED metrics (+exemplars), **structured logs shipped via OTLP** and
  correlated by `trace_id`/`span_id` — all viewable together in the Uptrace UI.
- Span naming: `<service>.<rpc>` for gRPC, `workflow.<state>` / `capability.<name>` for execution.
- Sampling: head-based parent sampling (configurable ratio; 100% local dev). Retention: M7 = dev defaults; scale retention → M8.
- **Local:** `docker compose -f …observability.yml up` brings Uptrace (+ deps + OTLP collector) with the login UI on a 70xx port.
- **In-cluster:** `helm/charts/uptrace/` (charts live at `helm/charts/`, not `infra/helm/`) deploys Uptrace + UI behind an Ingress, toggled by `observability.enabled`, so logs/traces are visible in any environment — not just locally.
- Validation: R.5 asserts a connected end-to-end trace per run.

### Security plan (review BEFORE implementation — Tier 2 scan per canvas)
Every `feat:` canvas runs `/spdd-security-review`. Focus areas this milestone:
- **STRIDE** on the data-flow context (W/C): tampering/info-disclosure via cross-state data leakage → explicit read/write scoping.
- **Git MCP (G):** credential management (no secrets in prompts), least-privilege token, prompt-injection on PR/issue content, context-leakage via traces (redaction).
- **Experts (X):** agent isolation, capability least-privilege, MCP permission scoping.
- **Supply chain (Q):** pip CVE closeout, cosign/SBOM unchanged from M6, PyPI Trusted Publisher provenance recorded.

### Testing plan (ADR-016 tiers + new gates)
| Test type | Where | Gate |
|-----------|-------|------|
| Unit (≥90% domain) | `services/*/internal/domain` | `make test-coverage` (Q.2 fixes interface-only) |
| BDD contract | `protos/tests/` + `services/*/tests` | `make test-bdd` |
| Integration | `//go:build integration` | **R.3 — new required gate (#553)** |
| Benchmarks (regression) | compiler + interpreter | **R.1 (#493)** |
| Fuzz | YAML→IR compiler | **R.2** |
| E2E | `zynax apply` real workflow | **R.4 (#1103)** |
| Observability validation | trace assertion | **R.5** |
| Performance/load/chaos | — | **deferred to M8** |

---

## 13 — Acceptance Matrix

M7 is **DONE** when every row is green. Rows 1–7 are the [§4](#4--reality-check-what-zynax-can-and-cannot-do-today) gaps the user flagged as must-include.

| # | Acceptance criterion | Verifies | EPIC | Done-check |
|---|----------------------|----------|------|------------|
| 1 | `apply research-task.yaml` runs to terminal with state→state data flow | data-flow | W | e2e test green |
| 2 | `GET /workflows/{id}/logs` streams real execution events | log streaming | L | `zynax logs --follow` shows transitions |
| 3 | One run emits a connected trace across all hops in Uptrace + RED metrics | OTEL/Uptrace | O | R.5 trace assertion |
| 4 | `agents/examples/` builds/lints/tests; a reference expert is dispatchable | reference agents | X | `make test-unit-agent` green |
| 5 | `make security-agents` / `make lint-go` run clean (pip 26.1.2) | supply chain | Q | `make ci` green |
| 6 | `make test-coverage` passes honestly (interface-only handled) | coverage gate | Q | `make ci` green |
| 7 | PyPI Trusted Publisher provenance documented ([§14](#14--pypi-trusted-publisher-history)) | provenance | Q | this doc + release notes |
| 8 | request-id propagates to every downstream span+log | context | C | log/trace inspection |
| 9 | Git MCP opens a PR with a scoped token; no secret in any prompt/trace | Git MCP | G | security review PASS |
| 10 | 3 real workflows (code-review, ci-pipeline, feature-impl) run green | templates/examples | T | e2e |
| 11 | Quick-start takes a new dev from clone → traced real-workflow run | docs | D | doc walkthrough |
| 12 | All new CI gates (integration, benchmark, fuzz, e2e, observability) active | test rigor | R | CI config |

**Definition of Done (every issue):** linked Canvas (feat) or rationale (non-feat); labels +
priority + milestone set; AC met; tests written and green; `make ci` green; telemetry emitted
where applicable; docs updated; one logical commit; PR ≤400 lines or justified; DCO signed;
`Assisted-by` trailer; squash-merged.

---

## 14 — PyPI Trusted Publisher History

> Recorded here per the milestone brief — the program had no durable record of the SDK's PyPI
> Trusted Publisher (OIDC) configuration. This is the canonical history; release notes link here.

**Mechanism.** `zynax-sdk` is published to PyPI via **Trusted Publishing (OIDC)** — no long-lived
API token is stored. The GitHub Actions workflow `sdk-publish.yml` requests a short-lived OIDC
token that PyPI exchanges for an upload scope.

**Required PyPI Trusted Publisher entry.** These are the exact values to register under the
`zynax-sdk` project's *Publishing* settings on pypi.org. They are sourced directly from
`.github/workflows/sdk-publish.yml` (the canonical SoT) — do not transcribe by hand.

| Field | Value |
|-------|-------|
| Distribution (PyPI project) | `zynax-sdk` |
| SDK version (`agents/sdk/pyproject.toml`) | `0.1.0` |
| Publisher | GitHub Actions — owner/repo `zynax-io/zynax` |
| Workflow filename | `sdk-publish.yml` (path `.github/workflows/sdk-publish.yml`) |
| GitHub Environment | `pypi` (set on the `publish` job — `environment: pypi`) |
| Trigger | push of a version tag matching `v*.*.*` |
| OIDC claims | `id-token: write` permission; PyPI exchanges the GitHub OIDC token for an upload scope |
| Trust model | OIDC Trusted Publisher (no long-lived API token stored in secrets) |
| TestPyPI dry-run | `tools-publish.yml` (PRs touching `agents/sdk/` — #805, F.1) |
| Supply-chain artifacts per publish | SPDX SBOM (`syft`) + cosign keyless signature bundles, uploaded to the GitHub Release |

**First-publish provenance.** As of this record the SDK has **not yet been published to PyPI** — no
`v*.*.*` tag has triggered `sdk-publish.yml`, so there is no published version, release URL, or
provenance attestation to cite. The first publish is expected at the **v0.6.0** SDK release tag.
When that tag is pushed, update the table below with the observed values:

| First-publish field | Value |
|---------------------|-------|
| First published version | _pending — fill at first `v*.*.*` publish_ |
| Trigger tag | _pending_ |
| GitHub Release URL | _pending_ |
| Provenance attestation reference | _pending — PyPI attaches a publish attestation via the Trusted Publisher flow_ |

**PyPI-side registration is a manual account action and cannot be verified or performed from this
repository.** Whether the Trusted Publisher entry already exists on pypi.org cannot be determined
from the repo. **Before the next SDK publish**, a maintainer with access to the `zynax-sdk` PyPI
project must confirm — or create — a Trusted Publisher entry using the *exact* values in the table
above (owner `zynax-io`, repo `zynax`, workflow `sdk-publish.yml`, environment `pypi`). If the
entry is missing the `v*.*.*` publish job will fail at the OIDC token exchange.

**Action items in Q.3:** (a) maintainer confirms/creates the PyPI Trusted Publisher entry with the
values above before the next publish; (b) at first publish, fill the "First-publish provenance"
table with the observed version + Release URL + attestation reference; (c) link this section from
the v0.6.0 release notes — recorded as a forward pointer in `CHANGELOG.md` under `[Unreleased]`
until v0.6.0 ships, at which point the v0.6.0 entry links here.

---

## Appendix A — Brief deliverable coverage map

| # | Brief deliverable | Section |
|---|-------------------|---------|
| 1 | milestone description | §0, §2 |
| 2 | roadmap | §1 |
| 3 | dependency graph | §6 |
| 4 | GitHub issue hierarchy | §5, §10 |
| 5 | labels | §10 (existing label set) |
| 6 | milestones | §1 |
| 7 | epics | §5 |
| 8 | features | §5 (stories per EPIC) |
| 9 | implementation order | §6, §7 |
| 10 | critical path | §6 |
| 11 | parallel execution plan | §7 |
| 12 | risk register | §8 |
| 13 | ADR list | §11 |
| 14 | documentation plan | §5 EPIC D |
| 15 | testing plan | §12 |
| 16 | observability plan | §12 |
| 17 | security plan | §12 |
| 18 | rollout strategy | §15 |
| 19 | rollback strategy | §15 |
| 20 | acceptance matrix | §13 |
| 21 | DoD per issue | §13 |
| 22 | SPDD commands per spec | §9 |

---

## 15 — Rollout & Rollback

**Rollout.** Incremental, per-EPIC, behind feature gates where a contract changes:
- Data-flow (W): proto fields are **additive** (backward-compatible per `buf breaking`); the
  compiler accepts manifests without `output:` unchanged.
- OTEL (O): off unless `ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT` is set → zero impact when unset.
- Uptrace (O.7): separate compose file; never required for the core stack.
- Git MCP (G): opt-in process; no change to runtime workflow path.

**Rollback.** Each EPIC is independently revertible:
- Proto additions are additive → revert is a stub-regenerate, no data migration.
- Observability is env-gated → unset the endpoint to disable.
- Quality fixes (Q) are isolated CI/tooling changes.
- No schema migrations introduced in M7 beyond additive proto fields → no destructive rollback path.

**Version cut.** v0.6.0 tagged via `/milestone-close` after the acceptance matrix is green;
GitHub Release + signed tag; `state/milestone.yaml` rotates M7 → history.
