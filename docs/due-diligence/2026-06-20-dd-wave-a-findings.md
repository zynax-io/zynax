<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax Due-Diligence — Wave A (Ground-Truth) Findings

> **Run output of issue #1402** — the first execution wave of the investment-grade
> due-diligence framework ([2026-06-18-zynax-due-diligence-framework.md](2026-06-18-zynax-due-diligence-framework.md)).
> This document is a **findings artifact**, not a verdict: it carries the eight Wave A
> ground-truth agent packets that later waves consume. The investment recommendation is
> produced only after Wave D synthesis (#1405) and the final report (#1406).

| Field | Value |
|-------|-------|
| Wave | **A — Ground truth** (no upstream dependencies; framework §3.2) |
| Issue | #1402 — *DD execution: validate dispatch loop + run Wave A* |
| Date | 2026-06-20 |
| Repository HEAD audited | `main` @ `e3135a6` |
| Agents run | 8 — §5.1, 5.2, 5.5, 5.7, 5.9, 5.10, 5.12, 5.24 |
| Dispatch loop validated | **Yes** — §3.5 dry-run gate run on Security (5.2) first; PASS, then scaled to 7 more |
| Evidence discipline | §0.4 — every claim carries `path:line`, a command+output, or is marked `UNKNOWN`; roadmap/marketing = `CLAIMED`, never `VERIFIED` |

## How this wave was produced

Each agent was dispatched as an independent **read-only** investigator. It received, by
direct reference, the framework's shared context packet (Part 1), the binding scoring/evidence
rules (Part 2 §2.2–§2.6), the handoff schema (§3.4), and its own Part 5 prompt — then audited
the repository at HEAD and returned the §3.4 YAML packet plus a §6.2 prose section. Per the
framework's anti-overlap matrix (§3.1), each agent scored only its primary zone and recorded
cross-zone observations as cross-references rather than re-scoring them.

The **dispatch loop was validated before scaling** (framework §3.5): the Security agent (5.2)
was run alone as the dry-run gate. Acceptance was met — the returned packet had an
`overall_score`, per-sub-dimension scores each carrying ≥1 `path:line`/command evidence, a
completed drift test for contradiction-register rows C2–C4, and an explicit unknowns ledger —
so the remaining seven agents were dispatched.

## Wave A scorecard

> Provisional, un-weighted. The orchestrator (Part 4, run in Wave D) down-weights
> low-confidence scores and resolves contradictions before any aggregate is final.

| Agent | Dim | Score | Conf | Most severe red flag (evidence) | Strongest green flag (evidence) |
|-------|-----|:---:|:---:|----------------------------------|----------------------------------|
| 5.1 Architecture | D3 | 7 | High | Argo portability stubbed at execution — `argo_engine.go:62-98` never calls `IRInterpreter.Run` | Genuinely engine-neutral IR, no engine types — `workflow_compiler.proto:205-241` |
| 5.2 Security | D5 | 7 | Med | mTLS fails open; 2 prod overlays omit TLS vs ADR-020:50 claim — `tlscreds.go:20` | Supply-chain trifecta cosign+SBOM+SLSA — `release.yml:201,527,510` |
| 5.5 Engineering | D4 | 8 | High | ADR-010 "Protocol, never a base class" contradicted by `class Agent(...ABC)` — `agents/sdk/src/zynax_sdk/agent.py:59` | Strict 14-linter config passes **0 issues**, no blanket `//nolint` — `tools/golangci-lint.yml:17-33` |
| 5.7 Testing | D9 | 8 | High | Domain-coverage gate re-checks only *changed* services — `_test-go.yml:132-134` | Blocking ≥90% gate proven by execution on all 7 domains — `tools/coverage-gates.env` |
| 5.9 DevOps | D8 | 8 | High | Production service/adapter images are **amd64-only** — `ci.yml:861` | Build-once / promote-by-retag (scan == deploy) — `release.yml:160-204` |
| 5.10 Documentation | D11 | 7 | High | README self-contradicts its own milestone status — `README.md:333-337` vs `446-464` | §1.10 doc-vs-tooling lag already resolved at HEAD — `CLAUDE.md:86-110` |
| 5.12 AI Workflow | D10 | 7 | High | Canvas-before-code is a **soft** gate (passes if any canvas exists) — `pr-checks.yml:231-233` | Closed, traceable learnings loop — `APPLY_LOG.md:15-99` |
| 5.24 Repo Health | D16 | 7 | High | **Bus factor = 1** — `git shortlog -sne` (one human identity) | Strictly linear signed history, 0 merge commits — `git log --merges` → 0 |

Un-weighted mean ≈ **7.4 / 10**. Treat as directional only — final aggregation is the
orchestrator's (Wave D).

## Aggregate drift test — contradiction register & headline claims

Verification of the boldest claims against HEAD (framework §2.6; Part 1 §1.10 register C1–C8):

| Claim | Register | Result | Evidence (agent) |
|-------|:---:|:---:|------------------|
| Engine-agnostic — "runs on Temporal **or** Argo without a rewrite" | thesis / §1.5 | **PARTIAL** | Real at the IR/contract boundary; Argo path serialises IR to a no-op cluster stub, never interprets it (5.1) |
| mTLS enforced on all inter-service gRPC | C2 | **PARTIAL** | Real mTLS available but code fails open to `insecure.NewCredentials()`; 2 prod overlays omit `tlsSecretName` (5.2) |
| SBOM generated per release | C3 | **VERIFIED** | syft SPDX per-service digest in `release.yml:527`, attached to Release (5.2, 5.9) |
| cosign-signed images | C4 | **PARTIAL** | Signing + SLSA provenance wired in `release.yml`; GHCR signature existence `UNKNOWN` (no registry access) (5.2, 5.9) |
| ≥90% domain coverage, gate blocks | strategy / ADR-016 | **VERIFIED** | Executed on all 7 services (92.1–100%); gate is `exit 1` (5.5, 5.7) |
| "140+ BDD scenarios" / every RPC covered | M1 / ADR-016 | **VERIFIED** | 306 scenarios in 18 features; all 33 RPCs covered (5.7) |
| Go lint-clean | strategy | **VERIFIED** | `golangci-lint` → 0 issues across sampled services (5.5) |
| "21 workflows / build-once-promote-by-retag" | ADR-027 | **VERIFIED** | 21 workflow files; `release.yml` is retag-only, zero build steps (5.9) |
| ADR-010 "AgentRuntime is a Protocol — never a base class" | ADR-010 | **CONTRADICTED** | SDK ships `class Agent(AgentServiceServicer, ABC)`; no Protocol in source (5.5) |
| Canvas-before-code is an enforced gate (ADR-019) | ADR-019 | **PARTIAL** | Soft gate: passes whenever any canvas exists repo-wide; "Aligned" is human-review-only (5.12) |
| §1.10 doc-vs-tooling lag (CLAUDE.md/spdd-guide.md command names) | §1.10 | **RESOLVED at HEAD** | Both files now cite the live 5-verb surface; residual lag only in `pr-checks.yml:240`, `APPLY_LOG.md` (5.10, 5.12) |
| Single-maintainer bus factor (acknowledged) | §1.9 | **VERIFIED** | One human identity in `git shortlog`; MAINTAINERS.md still open (#494) (5.24) |

## Cross-cutting themes (provisional, for the orchestrator)

- **Portability moat is real at the interface, partial at execution.** The engine-neutral IR
  and the 5-method `WorkflowEngine` port are genuine; only Temporal actually interprets the IR
  today (5.1). This is the single most important diligence question and should gate the
  "category" claim.
- **Supply-chain & test rigor are the strongest verified assets.** Coverage, BDD breadth, lint,
  SBOM/provenance, and build-once-promote are all VERIFIED by execution or config — not
  narrative (5.5, 5.7, 5.9, 5.2).
- **The gaps are enforcement-shaped, not absence-shaped.** mTLS, the coverage re-gate, and
  canvas-alignment all *exist* but are opt-in / soft / partial rather than hard-enforced
  (5.2, 5.7, 5.12).
- **The binding constraint is social, not technical.** Bus factor 1 and zero named external
  adopters are the CNCF/acquisition gate; the engineering substrate is well ahead of the
  community substrate (5.24, and confirmed by Part 1 §1.9).

## Handoff

These eight packets are the dependency input for the later waves (framework §3.2):

- **Wave B (#1403)** — Performance, Technical Debt, Maintainability, Scalability, OpenSSF,
  Innovation — consume the Architecture (5.1), Engineering (5.5), Testing (5.7), and DevOps
  (5.9) findings.
- **Wave C (#1404)** — Product, Market, Competitive, Governance, Open Source, Enterprise, CNCF,
  Roadmap, DX — lightly consume the Documentation (5.10) and Repo Health (5.24) findings.
- **Wave D (#1405)** — Risk, Investment, Business Strategy + the Part 4 orchestrator consume
  **everything**, resolve the contradictions above, confidence-weight the scores, and write the
  executive summary.
- **Report (#1406)** — assembles the Part 10 document and Part 9 executive presentation.

---

# Per-Agent Findings Packets

> Each section below is the verbatim, read-only output of one Wave A agent: its §3.4 YAML
> handoff packet followed by its §6.2 prose section. Literal contributor email addresses were
> neutralised (gitleaks PII gate); no other content was altered.

---
# Agent 5.1 — Architecture Agent · Wave A (ground-truth)

> Issue #1402 · HEAD `e3135a60e4abb20886d51f81d6448b22fe04cb64` · READ-ONLY audit.
> Every claim is grounded in `path:line` / command-output, or marked `UNKNOWN`.
> Marketing/roadmap = `CLAIMED`; code/CI/contract-verified = `VERIFIED`.

---

## (a) §3.4 Handoff packet

```yaml
agent: "5.1 Architecture"
wave: "A"
dimension_groups: ["D3", "D6", "D16"]
overall_score: 7
overall_confidence: "High"

sub_scores:
  - dimension: "Three-layer separation (hexagonal domain isolation)"
    score: 8
    confidence: "High"
    justification: "domain imports zero api/infra packages (real rule holds); but boundary is convention-only — no depguard/import-linter CI gate."
    evidence:
      - "services/AGENTS.md:16 — '4. The domain/ layer has zero imports from api/ or infrastructure/.'"
      - "services/AGENTS.md:51 — 'domain: ZERO imports from api or infrastructure'"
      - "cmd: grep -rn 'internal/api|internal/infrastructure' services/*/internal/domain/ → (empty: no domain→api/infra import found)"
      - "tools/golangci-lint.yml:17-33 — enabled linters list has NO depguard/forbidigo/import-boundary linter → boundary unenforced by tooling"
      - "engine-adapter/internal/domain/engine.go:5 — 'It has zero imports from api or infrastructure layers.' (package doc, matches reality)"
  - dimension: "Domain purity vs proto coupling"
    score: 7
    confidence: "High"
    justification: "Domain depends only on proto MESSAGE types (data), not gRPC service/client stubs; Part-1's stricter 'zero proto/gRPC imports' paraphrase overstates the real contract."
    evidence:
      - "services/engine-adapter/internal/domain/engine.go:10 — imports zynaxv1 (WorkflowIR message), not grpc"
      - "services/workflow-compiler/internal/domain/ir/ir.go:11 — imports zynaxv1 message types only"
      - "cmd: grep zynaxv1/grpc in services/*/internal/domain → matches are proto MESSAGE types; no google.golang.org/grpc client/server import in domain"
      - "services/AGENTS.md:16,51 — actual rule is api/infra isolation, NOT proto-free (Part1 §1.3 paraphrase is stricter than the real contract)"
  - dimension: "Engine-neutral Workflow IR (no Temporal leakage in contract)"
    score: 9
    confidence: "High"
    justification: "WorkflowIR proto is a pure state-machine model (StateIR/TransitionIR/ActionIR); no Temporal/Argo types in the contract; HITL modeled generically."
    evidence:
      - "protos/zynax/v1/workflow_compiler.proto:205-241 — WorkflowIR: workflow_id/states/initial_state/ir_version; no engine types"
      - "protos/zynax/v1/workflow_compiler.proto:122-135 — StateType incl. STATE_TYPE_HUMAN_IN_THE_LOOP (ADR-014 HITL modeled in IR, engine-neutral)"
      - "protos/zynax/v1/engine_adapter.proto:107,157 — engine identity is a free-text string ('temporal'/'argo'/'langgraph'), not a typed enum binding the contract to one engine"
      - "services/engine-adapter/internal/domain/interpreter.go:42-54 — IRInterpreter is a plain Go struct depending on ActivityExecutor/EventPublisher interfaces, not Temporal SDK"
  - dimension: "WorkflowEngine interface & engine extensibility"
    score: 8
    confidence: "High"
    justification: "Single clean 5-method port; Temporal AND Argo both satisfy it with compile-time assertions; adding a 3rd engine = implement 5 methods + a switch case."
    evidence:
      - "services/engine-adapter/internal/domain/engine.go:17-41 — WorkflowEngine{Submit,Signal,Cancel,GetStatus,Watch}"
      - "services/engine-adapter/internal/infrastructure/argo_engine.go:314 — 'var _ domain.WorkflowEngine = (*ArgoEngine)(nil)' (compile-time conformance)"
      - "services/engine-adapter/cmd/engine-adapter/main.go:185-197 — buildEngine switch over temporal|argo (ADR-015 config-selected)"
      - "docs/adr/ADR-015-pluggable-workflow-engines.md (exists) — engines selected by config flag"
  - dimension: "Multi-engine portability — OPERATIONAL reality (drift core)"
    score: 4
    confidence: "High"
    justification: "Same IR is SUBMITTED to both engines, but only Temporal INTERPRETS it; the Argo 'interpreter' is a stub that asserts non-empty payload and exits 0 — capability-dispatch parity explicitly out of scope."
    evidence:
      - "scripts/e2e/manifests/argo-ir-interpreter.yaml:10-14 — 'asserts a non-empty IR payload arrived and exits 0 ... Full Argo-side IR interpretation (capability-dispatch parity with the Temporal IRInterpreterWorkflow) is deliberately out of scope for the smoke gate.'"
      - "scripts/e2e/manifests/argo-ir-interpreter.yaml:56-60 — sh: if IR empty exit 1 else 'payload OK — succeeding' (no state traversal, no dispatch)"
      - "services/engine-adapter/internal/infrastructure/argo_engine.go:62-98 — Submit serialises IR to JSON, hands it to a cluster WorkflowTemplate param; never calls IRInterpreter.Run"
      - "scripts/e2e/e2e-argo.sh:232-266 — argo leg asserts only Argo Workflow CR phase==Succeeded (vs temporal leg's capability+CloudEvents+memory roundtrip)"
  - dimension: "7-service dependency graph: gRPC-only, no shared DB (ADR-001/008)"
    score: 9
    confidence: "High"
    justification: "All 8 inter-service edges are gRPC; no REST between services; task-broker & agent-registry own SEPARATE Postgres databases; no cross-service internal/ import."
    evidence:
      - "services/api-gateway/internal/infrastructure/clients.go:77,81,86,92 — api-gateway dials compiler/engine/registry/event-bus via grpc.NewClient"
      - "services/engine-adapter/cmd/engine-adapter/main.go:291,302 — engine-adapter→task-broker, →event-bus (gRPC)"
      - "services/task-broker/internal/infrastructure/registry_client.go:28 + event_publisher.go:46 — task-broker→agent-registry, →event-bus (gRPC)"
      - "infra/docker-compose/postgres-zynax-init.sql:5-6 — 'CREATE DATABASE task_broker; CREATE DATABASE agent_registry;' (separate DBs)"
      - "docs/adr/ADR-008-no-shared-databases.md — each service owns its schema; cross-service reads via gRPC only"
      - "cmd: grep for services/<other>/internal cross-imports → none found"
  - dimension: "Contract versioning & buf-breaking discipline"
    score: 8
    confidence: "High"
    justification: "Single zynax.v1 package; buf-breaking runs on every proto PR against base branch (required check); explicit additive-only/reserved rules documented."
    evidence:
      - ".github/workflows/pr-checks.yml:129-135 — 'buf breaking --against ...#branch=<base.ref>,subdir=protos' on proto-changed PRs"
      - "protos/buf.yaml:22-24 — breaking: use: [FILE]"
      - "protos/AGENTS.md:56-62 — never remove/renumber/retype field; new fields & enum values additive; breaking → zynax/v2/"
      - "cmd: grep 'package zynax' protos/zynax/v1/*.proto → all 'package zynax.v1'; no v2 dir"
  - dimension: "Idempotent apply (ManifestWorkflowID) & error modeling"
    score: 7
    confidence: "Medium"
    justification: "Idempotency works (SHA-256 over canonicalised YAML → stable id; existing run reused); errors are structured CompilationError not gRPC metadata. BUT ADR-034 is 'Proposed' and describes RANDOM ids — spec contradicts the shipped idempotent code."
    evidence:
      - "services/api-gateway/internal/domain/apply.go:29-35 — ManifestWorkflowID = 'wf-'+sha256(canonicaliseYAML)[:16]"
      - "services/api-gateway/internal/domain/apply.go:111-131 — running id → returns existing run_id (idempotent)"
      - "protos/zynax/v1/workflow_compiler.proto:15-17,259-266 — errors are repeated CompilationError in response body, never gRPC metadata; INVALID_ARGUMENT on structural error"
      - "docs/adr/ADR-034-manifest-workflow-id-collision-domain.md — status PROPOSED; describes random 64-bit ids (contradicts shipped manifest-hash idempotency)"
  - dimension: "State management & migration strategy (Postgres repos)"
    score: 7
    confidence: "Medium"
    justification: "task-broker/agent-registry/memory-service Postgres-backed with per-service DBs & migrations; BUT workflow_compiler proto still documents an unbounded in-memory IR map (stale C7 comment at HEAD)."
    evidence:
      - "services/task-broker/internal/infrastructure/postgres/repository.go (exists) — Postgres repo behind a domain port"
      - "infra/docker-compose/postgres-zynax-init.sql:5-6 — per-service DB provisioning"
      - "protos/zynax/v1/workflow_compiler.proto:50-53 — GetCompiledWorkflow doc still says 'unbounded in-memory map ... planned for M6 (issue #466)' — stale vs C7 'fixed M6' claim"

drift_test:
  - claim: "Engine-agnostic — the same workflow runs on Temporal OR Argo without a rewrite."
    result: "PARTIAL"
    evidence:
      - "VERIFIED at submission/contract level: same YAML→same WorkflowIR→same WorkflowEngine port→both engines satisfy it (argo_engine.go:314; engine.go:17-41)."
      - "CONTRADICTED at execution level: Argo leg runs a stub template that only checks IR non-empty and exits 0 (argo-ir-interpreter.yaml:10-14,56-60); only Temporal's IRInterpreterWorkflow actually traverses states & dispatches capabilities (interpreter.go:42-70)."
      - "e2e-argo.sh:232-266 asserts only Argo CR phase==Succeeded; no capability-dispatch / state-transition parity assertion."
  - claim: "Two real engines validated per CI run (engine matrix)."
    result: "PARTIAL"
    evidence:
      - "VERIFIED matrix exists: .github/workflows/e2e-smoke.yml:60-61 'matrix: engine: [temporal, argo]', fail-fast:false."
      - "WEAKENED: e2e-smoke.yml:16-19 — gated, NOT a required PR gate; argo leg excluded from branch-protection required-check set; only triggers on helm/services/engine-adapter paths."
      - "Asymmetric assertions: temporal leg = happy+failure+helm rollback (e2e-happy.sh/e2e-failure.sh); argo leg = CR-phase only (e2e-argo.sh)."
  - claim: "Inter-service: gRPC-only, no shared DB (ADR-001/008)."
    result: "VERIFIED"
    evidence:
      - "All 8 edges via grpc.NewClient (clients.go:77,81,86,92; engine-adapter main.go:291,302; task-broker registry_client.go:28, event_publisher.go:46)."
      - "Separate Postgres DBs per service (postgres-zynax-init.sql:5-6); no cross-service internal/ import (grep→none)."
  - claim: "buf-breaking gate enforces backward-compat on PRs."
    result: "VERIFIED"
    evidence:
      - ".github/workflows/pr-checks.yml:129-135 runs 'buf breaking --against' base branch on proto-changed PRs; protos/buf.yaml:22-24 breaking:[FILE]."

red_flags:
  - severity: "High"
    finding: "Engine portability is asymmetric: only Temporal genuinely interprets the IR. The Argo engine serialises the IR to JSON and hands it to a cluster-side stub template that asserts the payload is non-empty and exits 0 — capability-dispatch parity is 'deliberately out of scope'. The headline moat ('runs on Temporal OR Argo without rewrite') is real at the interface/submission boundary but NOT at the execution boundary. A second engine is wired structurally, not functionally equivalent."
    evidence:
      - "scripts/e2e/manifests/argo-ir-interpreter.yaml:10-14,56-60"
      - "services/engine-adapter/internal/infrastructure/argo_engine.go:62-98 (no IRInterpreter.Run call on Argo path)"
      - "scripts/e2e/e2e-argo.sh:232-266 (CR-phase assertion only)"
  - severity: "Medium"
    finding: "Layer-boundary separation has NO automated enforcement (no depguard/import-linter/fitness function in lint or CI). The three-layer/hexagonal invariant holds today by code review + convention only; one careless import re-introduces coupling silently. Directly weakens the 'enforced, not just documented' claim."
    evidence:
      - "tools/golangci-lint.yml:17-33 (linter list has no import-boundary linter)"
      - "cmd: grep depguard/forbidigo/import-boundary across Makefile + golangci config → none found"
  - severity: "Medium"
    finding: "Spec/implementation drift on idempotent apply: ADR-034 is 'Proposed' and specifies RANDOM workflow ids, while shipped api-gateway derives a deterministic SHA-256 manifest hash (idempotent). The ADR contradicts the code at HEAD — an unreconciled one-way-door decision record."
    evidence:
      - "services/api-gateway/internal/domain/apply.go:29-35"
      - "docs/adr/ADR-034-manifest-workflow-id-collision-domain.md (status Proposed; random ids)"
  - severity: "Low"
    finding: "Dead contract surface: SubmitWorkflowRequest.engine_hint exists (engine_adapter.proto:158) and is plumbed end-to-end from ?engine= (api-gateway handler.go:89), but the engine-adapter handler IGNORES it — engine selection is process-wide via env (main.go:185-197, handler.go:36-48). Per-request engine routing is not implemented; multi-engine = multiple deployments."
    evidence:
      - "services/engine-adapter/internal/api/handler.go:36-48 (SubmitWorkflow never reads GetEngineHint)"
      - "services/engine-adapter/cmd/engine-adapter/main.go:185-197 (process-wide engine switch)"
  - severity: "Low"
    finding: "Stale durability comment in shipped contract: workflow_compiler.proto still documents the IR store as an 'unbounded in-memory map ... planned for M6' (the C7 row Part-1 claims fixed in M6). At HEAD the contract text contradicts the 'refactored M6' narrative."
    evidence:
      - "protos/zynax/v1/workflow_compiler.proto:50-53"

green_flags:
  - strength: "Genuinely engine-neutral IR contract: WorkflowIR is a clean state-machine model (StateIR/TransitionIR/ActionIR + HITL state type) with zero Temporal/Argo types; engine identity is a free-text string, not a typed binding."
    evidence: ["protos/zynax/v1/workflow_compiler.proto:205-241,122-135", "protos/zynax/v1/engine_adapter.proto:107,157"]
  - strength: "Textbook WorkflowEngine port: one 5-method interface, both engines carry compile-time conformance assertions, config-selected per ADR-015 — adding a 3rd engine is a bounded, well-defined task."
    evidence: ["services/engine-adapter/internal/domain/engine.go:17-41", "services/engine-adapter/internal/infrastructure/argo_engine.go:314"]
  - strength: "Clean service topology: 8 gRPC-only inter-service edges, no REST between services, separate Postgres DB per stateful service, no cross-service internal/ imports — ADR-001/008 hold in code, not just docs."
    evidence: ["services/api-gateway/internal/infrastructure/clients.go:77-92", "infra/docker-compose/postgres-zynax-init.sql:5-6"]
  - strength: "Strong contract discipline: single zynax.v1 namespace, buf-breaking as a required PR gate against the base branch, explicit additive-only/reserved-field rules, structured CompilationError error modeling (not gRPC metadata)."
    evidence: [".github/workflows/pr-checks.yml:129-135", "protos/AGENTS.md:56-62", "protos/zynax/v1/workflow_compiler.proto:15-17"]
  - strength: "Hexagonal domain is real where it counts: domain packages import zero api/infrastructure packages; engine-specific SDKs (Temporal) are confined to infrastructure behind ActivityExecutor/EventPublisher ports."
    evidence: ["services/engine-adapter/internal/domain/engine.go:5", "services/engine-adapter/internal/domain/interpreter.go:42-54", "grep internal/api|internal/infrastructure in */internal/domain → empty"]

open_questions:
  - "What is the true cost to bring the Argo engine to capability-dispatch parity with Temporal? Does it require a sidecar/operator that runs IRInterpreter inside the Argo pod, or a full re-implementation of the state machine in Argo DAG primitives?"
  - "Is per-request engine_hint intended to ever be honored (single engine-adapter routing to multiple backends), or is process-per-engine the permanent model? The contract field implies the former; the code implements the latter."
  - "Does any LangGraph engine adapter exist beyond the proto string mention, or is it capability-provider-only (per Part-1 non-goals)?"

unknowns:
  - "Whether the argo leg of e2e-smoke is actually GREEN at HEAD (CI run results not inspectable offline) — verified the harness exists and what it asserts, not the live pass/fail. (E1 executed-proof not available read-only/offline.)"
  - "Helm production DB topology: whether prod provisions truly separate Postgres INSTANCES (vs separate DBs in one instance as in docker-compose) — ADR-008 permits shared instance/separate schema; not independently confirmed in infra/helm at HEAD."
  - "10x-scale break points (workflow-compiler IR store durability, engine-adapter Temporal worker fan-out) — design-level inference only; no load test observed."

cross_references:
  - to_agent: "5.7 Testing"
    note: "Argo leg has materially weaker e2e assertions than Temporal (CR-phase only vs capability+events+memory). Coverage/parity gap is a testing-rigor finding too."
    evidence: ["scripts/e2e/e2e-argo.sh:232-266", "scripts/e2e/manifests/argo-ir-interpreter.yaml:10-14"]
  - to_agent: "5.9 DevOps"
    note: "e2e-smoke argo leg is NOT a required branch-protection check; engine matrix is path-gated/optional — affects 'multi-engine proven in CI' DevOps claim."
    evidence: [".github/workflows/e2e-smoke.yml:16-19,60-61"]
  - to_agent: "5.5 Engineering"
    note: "No automated layer-boundary linter (depguard/import-boundary) — hexagonal invariant is review-enforced only; relevant to code-quality enforcement scoring."
    evidence: ["tools/golangci-lint.yml:17-33"]
  - to_agent: "5.13 Governance / 5.10 Documentation"
    note: "ADR-034 'Proposed' contradicts shipped idempotent apply; workflow_compiler.proto:50-53 in-memory-map comment is stale vs C7. ADR-vs-code reconciliation debt."
    evidence: ["docs/adr/ADR-034-manifest-workflow-id-collision-domain.md", "protos/zynax/v1/workflow_compiler.proto:50-53"]
  - to_agent: "5.16 Scalability"
    note: "engine_hint dead field + process-per-engine model constrains horizontal multi-engine routing; IR store durability open at scale."
    evidence: ["services/engine-adapter/internal/api/handler.go:36-48"]

recommendations:
  - priority: "P0"
    action: "Stop marketing 'runs on Temporal OR Argo without rewrite' as a shipped/proven capability until the Argo path interprets the IR (capability dispatch + state transitions), OR re-label it precisely as 'engine-neutral IR with Temporal as the reference interpreter; Argo submission validated, execution parity in progress'."
    rationale: "The boldest moat claim is currently CONTRADICTED at the execution boundary (argo-ir-interpreter.yaml:10-14). This is exactly the delivery-vs-narrative drift class the diligence exists to catch (Part-1 §1.10)."
  - priority: "P1"
    action: "Add an automated layer-boundary fitness function (depguard or a go/analysis import-linter in golangci-lint + CI) asserting domain imports no api/infrastructure and no grpc client/server packages."
    rationale: "Converts the central hexagonal/three-layer invariant from convention to an enforced gate, hardening the headline architectural claim against silent regression."
  - priority: "P1"
    action: "Bring the Argo engine to IRInterpreter parity (e.g. run the existing engine-neutral IRInterpreter as an Argo pod/operator) and add a CROSS-engine parity test that submits the SAME IR to both engines and asserts equivalent state transitions + dispatch."
    rationale: "Makes the portability moat operationally real and testable; today no test enforces cross-engine equivalence."
  - priority: "P2"
    action: "Reconcile ADR-034 to Accepted matching the shipped manifest-hash idempotency (or revert code to the ADR); refresh the stale in-memory-map comment in workflow_compiler.proto:50-53."
    rationale: "Eliminates ADR-vs-code one-way-door drift and removes a stale contract comment that re-asserts a since-fixed limitation."
  - priority: "P2"
    action: "Either implement per-request engine_hint routing in the engine-adapter handler, or deprecate/document the field as advisory-only to avoid a misleading contract surface."
    rationale: "Closes the gap between the contract (per-request hint) and the runtime (process-wide engine)."
```

---

## (b) §6.2 Prose section

## 5.1 Architecture — Score: 7 (High)

**Mission recap:** Assess whether Zynax's three-layer separation and engine-agnostic IR are real or aspirational, and whether the "Kubernetes for AI workflows" control-plane architecture is sound, modular, and defensible.

**Verdict:** The architecture is genuinely well-built at the contract and topology level — a clean engine-neutral state-machine IR, a textbook 5-method `WorkflowEngine` port that two engines satisfy with compile-time assertions, strict gRPC-only inter-service edges with per-service Postgres databases, and a buf-breaking gate enforcing additive-only contract evolution. The design is not a monolith-in-disguise; the bones are sound and extensible. The single material weakness is that the headline moat — multi-engine portability — is real at the *submission/interface* boundary but only Temporal genuinely *interprets* the IR. The Argo path serialises the IR to JSON and hands it to a cluster-side stub that checks the payload is non-empty and exits 0, with "capability-dispatch parity ... deliberately out of scope." A second engine is therefore wired structurally, not yet functionally equivalent. That gap, plus the absence of any automated layer-boundary linter, keeps this a strong-7 rather than a 9.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Three-layer / hexagonal isolation | 8 | High | `services/AGENTS.md:16,51`; grep domain→api/infra empty; `tools/golangci-lint.yml:17-33` (no boundary linter) |
| Domain purity vs proto coupling | 7 | High | `engine-adapter/internal/domain/engine.go:10`; `workflow-compiler/internal/domain/ir/ir.go:11` (message types only) |
| Engine-neutral IR (no Temporal leak) | 9 | High | `workflow_compiler.proto:205-241,122-135`; `engine_adapter.proto:107,157` |
| WorkflowEngine interface & extensibility | 8 | High | `engine.go:17-41`; `argo_engine.go:314`; `main.go:185-197` |
| Portability — operational reality | 4 | High | `argo-ir-interpreter.yaml:10-14,56-60`; `argo_engine.go:62-98`; `e2e-argo.sh:232-266` |
| 7-service graph: gRPC-only, no shared DB | 9 | High | `clients.go:77-92`; `engine-adapter main.go:291,302`; `postgres-zynax-init.sql:5-6` |
| Contract versioning / buf-breaking | 8 | High | `pr-checks.yml:129-135`; `buf.yaml:22-24`; `protos/AGENTS.md:56-62` |
| Idempotent apply & error modeling | 7 | Medium | `apply.go:29-35,111-131`; `workflow_compiler.proto:15-17`; `ADR-034` (Proposed, contradicts) |
| State mgmt & migration | 7 | Medium | `task-broker/.../postgres/repository.go`; `workflow_compiler.proto:50-53` (stale) |

**Drift test:**
- *"Engine-agnostic — runs on Temporal OR Argo without rewrite"* → **PARTIAL.** VERIFIED at submission/contract (same YAML → same IR → same port, both engines conform); CONTRADICTED at execution (Argo runs a non-interpreting stub; only Temporal traverses the IR and dispatches capabilities — `argo-ir-interpreter.yaml:10-14`).
- *"Two real engines validated per CI run"* → **PARTIAL.** Matrix exists (`e2e-smoke.yml:60-61`) but the argo leg is non-required, path-gated, and asserts only CR-phase==Succeeded vs Temporal's full capability/event/memory assertions.
- *"gRPC-only, no shared DB"* → **VERIFIED** (`clients.go:77-92`, `postgres-zynax-init.sql:5-6`).
- *"buf-breaking enforces backward-compat on PRs"* → **VERIFIED** (`pr-checks.yml:129-135`).

**Red flags (severity-ordered):**
1. **High** — Argo engine does not interpret the IR; portability moat is structural, not functional (`argo-ir-interpreter.yaml:10-14`, `argo_engine.go:62-98`, `e2e-argo.sh:232-266`).
2. **Medium** — No automated layer-boundary enforcement; hexagonal invariant is convention/review-only (`tools/golangci-lint.yml:17-33`).
3. **Medium** — ADR-034 ("Proposed", random ids) contradicts shipped idempotent SHA-256 apply (`apply.go:29-35` vs `ADR-034`).
4. **Low** — `engine_hint` is a dead contract field; handler ignores it (`handler.go:36-48`, `main.go:185-197`).
5. **Low** — Stale "unbounded in-memory map / planned for M6" comment in the live contract (`workflow_compiler.proto:50-53`).

**Green flags:**
- Engine-neutral IR contract with HITL modeled generically and engine identity as free-text string (`workflow_compiler.proto:205-241,122-135`).
- Clean 5-method `WorkflowEngine` port with compile-time conformance for both engines (`engine.go:17-41`, `argo_engine.go:314`).
- gRPC-only topology, per-service DBs, no cross-service internal imports (`clients.go:77-92`, `postgres-zynax-init.sql:5-6`).
- Required buf-breaking PR gate + explicit additive-only/reserved rules + structured error modeling (`pr-checks.yml:129-135`, `protos/AGENTS.md:56-62`).

**Open questions / unknowns:** True cost to reach Argo IR-interpretation parity; whether per-request `engine_hint` is ever intended to route; whether the argo CI leg is actually green at HEAD (not inspectable offline); prod Helm DB-instance topology; 10x-scale break points (IR store durability, Temporal worker fan-out).

**Recommendations:** P0 — re-label the portability claim precisely until Argo interprets the IR; P1 — add a layer-boundary fitness function and a cross-engine IR-parity test (+ bring Argo to interpreter parity); P2 — reconcile ADR-034 and the stale proto comment, resolve the `engine_hint` dead-field gap.

**Cross-references:** 5.7 Testing (argo e2e assertion gap); 5.9 DevOps (argo leg non-required, matrix path-gated); 5.5 Engineering (no boundary linter); 5.13/5.10 (ADR-vs-code drift); 5.16 Scalability (process-per-engine, IR durability).
<!-- SPDX-License-Identifier: Apache-2.0 -->
# Agent 5.2 — Security — Wave A (ground-truth) — Issue #1402

> Scope: real security posture of Zynax at current HEAD (branch `main`, commit `e3135a6`).
> Evidence rule (§0.4): every factual claim carries `path:line` or a command+output, or is
> marked `UNKNOWN — not found`. Roadmap/marketing statements are labelled `CLAIMED`.
> Dimension groups: D5 (contributes D7/D8/D14). Read-only audit; no repo files modified.

---

## (a) §3.4 Handoff packet

```yaml
agent: "5.2 Security"
wave: "A"
dimension_groups: ["D5"]
overall_score: 7
overall_confidence: "Medium"

sub_scores:
  - dimension: "Inter-service transport / mTLS"
    score: 6
    confidence: "High"
    justification: "Real mTLS (RequireAndVerifyClientCert) via cert-manager, but enforcement is opt-in by overlay and 2 of 7 service overlays omit it; code falls open to insecure when certs absent."
    evidence:
      - "services/api-gateway/internal/infrastructure/tlscreds.go:20  (empty cert path -> insecure.NewCredentials())"
      - "services/api-gateway/internal/infrastructure/tlscreds.go:47  (tls.RequireAndVerifyClientCert — true mTLS when configured)"
      - "helm/zynax-engine-adapter/values.yaml:51  (tlsSecretName: \"\" — insecure by default)"
      - "helm/zynax-engine-adapter/values-production.yaml:26  (tlsSecretName set in prod overlay)"
      - "helm/zynax-api-gateway/values-production.yaml  (22 lines, NO tlsSecretName — gateway upstream stays insecure even in prod overlay)"
      - "docs/adr/ADR-020-zero-trust-auth.md:50  (claims 'mTLS enforced on all inter-service gRPC' K8s M6+)"
  - dimension: "api-gateway authN/authZ"
    score: 7
    confidence: "High"
    justification: "Bearer auth uses crypto/subtle.ConstantTimeCompare; fails closed on empty key unless explicit ZYNAX_GW_DEV_INSECURE=1. No RBAC/SSO/scopes (single shared key)."
    evidence:
      - "services/api-gateway/internal/api/auth.go:20  (subtle.ConstantTimeCompare — timing-safe)"
      - "services/api-gateway/cmd/api-gateway/main.go:45  (empty key -> refuse to start unless DevInsecure)"
      - "services/api-gateway/cmd/api-gateway/main.go:68  (Warn: auth disabled in dev-insecure mode)"
      - "services/api-gateway/internal/api/auth.go:14  (single shared static key; no per-caller identity / RBAC)"
  - dimension: "Supply chain: SBOM + signing + provenance"
    score: 8
    confidence: "Medium"
    justification: "release.yml runs cosign sign (keyless OIDC), syft SPDX SBOM per service, and actions/attest-build-provenance (SLSA L2). Signatures themselves not verifiable locally (cosign absent / no network) -> Medium."
    evidence:
      - ".github/workflows/release.yml:201  (cosign sign --yes per promoted digest, merge path)"
      - ".github/workflows/release.yml:504  (cosign sign on version tag path)"
      - ".github/workflows/release.yml:527  (syft Generate SBOM SPDX per service digest)"
      - ".github/workflows/release.yml:510  (actions/attest-build-provenance — SLSA L2, ADR-025)"
      - "cmd: `cosign version` -> 'orden no encontrada' (cosign not installed; GHCR verify not runnable locally — UNKNOWN)"
  - dimension: "Image digest pinning + drift gate"
    score: 9
    confidence: "High"
    justification: "images.yaml is single SoT of sha256 digests; check-images gate runs in pre-merge CI; Dockerfiles consume pinned digests via banner regions."
    evidence:
      - "images/images.yaml  (every base + service entry carries a sha256 digest pin)"
      - ".github/workflows/pr-checks.yml:336  (go run . images check — drift gate runs pre-merge)"
      - "infra/docker/Dockerfile.service:25  (DISTROLESS_DIGEST pinned via banner region)"
      - "infra/docker/Dockerfile.service:49  (FROM gcr.io/distroless/static:nonroot@${DISTROLESS_DIGEST})"
  - dimension: "Container hardening"
    score: 9
    confidence: "High"
    justification: "Distroless nonroot, static CGO_ENABLED=0 stripped binary; Helm enforces runAsNonRoot/1001, RuntimeDefault seccomp, readOnlyRootFilesystem, drop ALL caps, no priv-esc."
    evidence:
      - "infra/docker/Dockerfile.service:49  (gcr.io/distroless/static:nonroot)"
      - "infra/docker/Dockerfile.service:43  (CGO_ENABLED=0, -trimpath -ldflags -s -w)"
      - "helm/zynax-lib/templates/_helpers.tpl:76  (podSecurityContext: runAsNonRoot, runAsUser 1001, seccomp RuntimeDefault)"
      - "helm/zynax-lib/templates/_helpers.tpl:89  (containerSecurityContext: allowPrivilegeEscalation false, readOnlyRootFilesystem true, drop ALL caps)"
      - "helm/zynax-api-gateway/templates/deployment.yaml:31  (pod/containerSecurityContext actually wired into Deployment)"
  - dimension: "CI security gates (CVE / SAST / secrets)"
    score: 8
    confidence: "High"
    justification: "govulncheck (Go), bandit + pip-audit (Python), trivy CRITICAL/HIGH = fail with audited .trivyignore, hadolint, dependency-review (blocks HIGH CVEs), gitleaks, CodeQL SARIF upload. Some pip-audit/bandit steps use '|| failed=true' aggregate pattern (still blocking) and 2 named CVE ignores."
    evidence:
      - ".github/workflows/ci.yml:689  (govulncheck on changed Go services)"
      - ".github/workflows/ci.yml:716  (bandit + pip-audit across SDK + agents)"
      - ".github/workflows/ci.yml:870  (Trivy CRITICAL,HIGH = fail on staging image)"
      - ".github/workflows/pr-checks.yml:60  (dependency-review blocks new HIGH CVEs)"
      - ".github/workflows/pr-checks.yml:366  (gitleaks secret scan over PR commit range)"
      - ".trivyignore  (only DS002 root-tools-image, accepted-until 2026-11-01, documented)"
      - ".github/workflows/ci.yml:730  (pip-audit --ignore-vuln PYSEC-2026-196 — one suppressed Python CVE)"
  - dimension: "Network attack surface (K8s)"
    score: 7
    confidence: "Medium"
    justification: "Every service ships a NetworkPolicy with Ingress AND Egress policyTypes; egress restricted to DNS + named upstream gRPC ports. But ingress rules restrict ports only — no 'from' source selector — so any pod may reach those ports (not source-locked default-deny)."
    evidence:
      - "helm/zynax-api-gateway/templates/networkpolicy.yaml:12  (policyTypes Ingress + Egress)"
      - "helm/zynax-api-gateway/templates/networkpolicy.yaml:15  (ingress: ports only, no 'from' — any source on those ports)"
      - "helm/zynax-api-gateway/templates/networkpolicy.yaml:21  (egress limited to DNS + 50052/50054/50055)"
  - dimension: "Secrets hygiene"
    score: 8
    confidence: "Medium"
    justification: "gitleaks runs in both ci.yml and pr-checks.yml with a baseline; config from envconfig only; no plaintext secrets surfaced in audited paths. Did not exhaustively scan every tracked file."
    evidence:
      - ".github/workflows/ci.yml:286  (gitleaks detect --no-git --source . with baseline)"
      - ".github/workflows/pr-checks.yml:385  (gitleaks over full PR commit range)"
      - "services/api-gateway/cmd/api-gateway/main.go:56  (envconfig.Process — config via env, not embedded secrets)"

drift_test:
  - claim: "C2 — mTLS enforced between all services (early SECURITY.md / ADR-020:50)"
    result: "PARTIAL"
    evidence:
      - "services/api-gateway/internal/infrastructure/tlscreds.go:47  (real mTLS: RequireAndVerifyClientCert)"
      - "services/api-gateway/internal/infrastructure/tlscreds.go:20  (BUT empty cert path -> insecure.NewCredentials(); code fails OPEN, not closed)"
      - "helm/zynax-engine-adapter/values.yaml:51  (tlsSecretName default \"\" — insecure unless prod overlay applied)"
      - "helm/zynax-api-gateway/values-production.yaml  (NO tlsSecretName even in prod overlay — gateway upstream not mTLS)"
      - "docs/adr/ADR-020-zero-trust-auth.md:51  (ADR itself states Compose stays insecure)"
  - claim: "C3 — SBOM per release (early SECURITY.md / ADR-025)"
    result: "VERIFIED"
    evidence:
      - ".github/workflows/release.yml:527  (syft generates SPDX SBOM from each service version digest)"
      - ".github/workflows/release.yml:577  (sbom-* artifacts collected into the GitHub Release)"
      - "SECURITY.md:42  (claim corroborated by E3 CI evidence above)"
  - claim: "C4 — cosign-signed images (early SECURITY.md)"
    result: "PARTIAL"
    evidence:
      - ".github/workflows/release.yml:201  (cosign sign --yes wired on promote path — E3)"
      - ".github/workflows/release.yml:504  (cosign sign on version tag path — E3)"
      - "cmd: `cosign version` -> not installed; no network — could NOT run `cosign verify` against GHCR (E1 missing). Signing is configured; signature EXISTENCE in registry is UNKNOWN."

red_flags:
  - severity: "High"
    finding: "mTLS is NOT enforced platform-wide: services fall open to insecure.NewCredentials() when cert paths are unset, the chart DEFAULT is insecure, and the api-gateway + workflow-compiler production overlays do not set tlsSecretName — so the gateway's upstream gRPC and compiler can run plaintext even under the 'production' overlay. ADR-020:50 and SECURITY.md:89 assert 'mTLS enforced on all inter-service gRPC', which overstates the shipped default."
    evidence:
      - "services/api-gateway/internal/infrastructure/tlscreds.go:20"
      - "helm/zynax-api-gateway/values-production.yaml  (no tlsSecretName)"
      - "docs/adr/ADR-020-zero-trust-auth.md:50"
      - "SECURITY.md:89"
  - severity: "Medium"
    finding: "NetworkPolicy ingress is port-scoped, not source-scoped: no 'from' selector means any pod in the cluster can dial the service ports. Not a true zero-trust default-deny ingress."
    evidence:
      - "helm/zynax-api-gateway/templates/networkpolicy.yaml:15"
  - severity: "Medium"
    finding: "api-gateway authZ is a single shared static bearer key with no per-caller identity, scopes, RBAC, or SSO. Adequate for a control plane MVP, insufficient for an enterprise governance buyer."
    evidence:
      - "services/api-gateway/internal/api/auth.go:14"
  - severity: "Low"
    finding: "Two named dependency CVEs are suppressed (pip-audit PYSEC-2026-196; trivy DS002 root tools-image). Both documented/time-boxed, but they are standing exceptions to the blocking gates."
    evidence:
      - ".github/workflows/ci.yml:730"
      - ".trivyignore  (DS002, accepted-until 2026-11-01)"

green_flags:
  - strength: "Bearer auth is timing-safe (crypto/subtle.ConstantTimeCompare) and fails CLOSED — gateway refuses to start with an empty key unless ZYNAX_GW_DEV_INSECURE=1 is explicitly set."
    evidence:
      - "services/api-gateway/internal/api/auth.go:20"
      - "services/api-gateway/cmd/api-gateway/main.go:45"
  - strength: "End-to-end supply-chain controls in CI: cosign keyless signing + syft SPDX SBOM + SLSA L2 provenance attestation per image, all in release.yml."
    evidence:
      - ".github/workflows/release.yml:201"
      - ".github/workflows/release.yml:527"
      - ".github/workflows/release.yml:510"
  - strength: "Strong container hardening shipped: distroless/static:nonroot, static stripped binary, and Helm enforces runAsNonRoot/1001, RuntimeDefault seccomp, readOnlyRootFilesystem, drop ALL caps, no priv-esc — wired into actual Deployments."
    evidence:
      - "infra/docker/Dockerfile.service:49"
      - "helm/zynax-lib/templates/_helpers.tpl:76"
      - "helm/zynax-api-gateway/templates/deployment.yaml:31"
  - strength: "Digest pinning with a SoT (images.yaml) and a pre-merge drift gate (images check) plus auditable, time-boxed trivy exceptions."
    evidence:
      - "images/images.yaml"
      - ".github/workflows/pr-checks.yml:336"
      - ".trivyignore"
  - strength: "Layered blocking CVE/SAST/secret gates: govulncheck, bandit, pip-audit, trivy (CRITICAL/HIGH fail), dependency-review, gitleaks (x2), CodeQL SARIF."
    evidence:
      - ".github/workflows/ci.yml:689"
      - ".github/workflows/ci.yml:870"
      - ".github/workflows/pr-checks.yml:60"

open_questions:
  - "Are cosign signatures and SLSA provenance actually present on the published GHCR images? (cosign not installed / no network — could not run `cosign verify` or `gh attestation verify`.)"
  - "Is there any documented operational guard that REJECTS startup when a service is deployed without TLS in a production namespace (analogous to ZYNAX_GW_DEV_INSECURE for the bearer key)? None found in code."
  - "Why do api-gateway and workflow-compiler production overlays omit tlsSecretName — intentional (TLS-terminating ingress in front of gateway) or a gap?"

unknowns:
  - "GHCR image signature/attestation existence — cosign binary absent and no network access; marked UNKNOWN, not failed (per task instruction)."
  - "Runtime enforcement of NetworkPolicy depends on a CNI that honors it (e.g. Calico/Cilium) — not verifiable from repo."
  - "Did not exhaustively scan every tracked file for secrets beyond gitleaks gate config; relied on CI gate evidence (Medium confidence on secrets)."

cross_references:
  - to_agent: "5.x Architecture/Reliability"
    note: "mTLS fail-open default (insecure.NewCredentials) intersects the three-layer/zero-trust posture; flag the chart default vs ADR-020 claim divergence."
    evidence: ["services/api-gateway/internal/infrastructure/tlscreds.go:20", "docs/adr/ADR-020-zero-trust-auth.md:50"]
  - to_agent: "5.x Governance/CNCF-readiness"
    note: "Supply chain (cosign+SBOM+SLSA) and OpenSSF posture are CNCF-credible; the mTLS-enforcement gap is the kind of thing an external security audit (M8 gate) would flag."
    evidence: [".github/workflows/release.yml:201", "SECURITY.md:89"]
  - to_agent: "5.x Docs/Truth-pass"
    note: "SECURITY.md:89 and ADR-020:50 assert 'mTLS between all platform services / enforced'; reality is opt-in-by-overlay with two overlays missing it. Doc-vs-impl drift to reconcile."
    evidence: ["SECURITY.md:89", "helm/zynax-api-gateway/values-production.yaml"]

recommendations:
  - priority: "P0"
    action: "Make insecure transport fail-closed in production: have services refuse to start without TLS creds unless an explicit ZYNAX_DEV_INSECURE=1 flag is set (mirror the gateway's API-key guard at main.go:45), and add tlsSecretName to the api-gateway + workflow-compiler production overlays."
    rationale: "Today 'mTLS enforced' (ADR-020:50, SECURITY.md:89) is overstated; a misconfigured prod deploy silently runs plaintext gRPC. Fail-open + missing overlays is the single highest-severity gap."
  - priority: "P1"
    action: "Add source 'from' selectors (default-deny ingress) to the NetworkPolicies so only the api-gateway and named peer services may reach each port, not any pod."
    rationale: "Port-only ingress is not zero-trust; lateral movement is unconstrained inside the namespace."
  - priority: "P1"
    action: "Run `cosign verify` / `gh attestation verify` against a published GHCR image in CI (or document a verified run) to convert C4 from PARTIAL to VERIFIED, and reconcile SECURITY.md mTLS wording to 'mTLS supported via cert-manager; enforced when production overlay applied'."
    rationale: "Closes the supply-chain proof gap and removes the doc overstatement an external audit would flag."
  - priority: "P2"
    action: "Introduce per-caller identity / scoped tokens (or OIDC) at the api-gateway and time-box the two CVE suppressions to the quarterly review."
    rationale: "Single shared static key blocks the enterprise governance persona; standing CVE ignores erode the blocking-gate guarantee."
```

---

## (b) §6.2 Prose section

## Security Agent — Score: 7 (Medium)

**Mission recap:** Establish the real security posture — authn/authz, transport, secrets, supply
chain, container hardening, CI gates — and separate shipped controls from documented intent.

**Verdict:** Zynax has a genuinely strong, CNCF-credible *supply-chain and container-hardening*
posture (cosign keyless signing, syft SPDX SBOM, SLSA L2 provenance, digest pinning with a drift
gate, distroless-nonroot images, full Helm securityContext, and layered blocking CVE/SAST/secret
gates), and its gateway authentication is timing-safe and fails closed. The material weakness is
**transport enforcement**: mTLS is correctly implemented (`RequireAndVerifyClientCert`) and wired
through cert-manager, but it is **opt-in by Helm overlay and fails open in code** — the chart
default returns `insecure.NewCredentials()`, and the api-gateway and workflow-compiler production
overlays omit `tlsSecretName`. This makes "mTLS enforced on all inter-service gRPC"
(ADR-020:50 / SECURITY.md:89) an overstatement of the shipped default. A Fortune-500 security
review would pass the supply chain and containers but flag the fail-open transport default and the
single shared bearer key.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Key evidence |
|---|---|---|---|
| Inter-service transport / mTLS | 6 | High | `tlscreds.go:20` (insecure fallback), `tlscreds.go:47` (RequireAndVerifyClientCert), `zynax-api-gateway/values-production.yaml` (no tlsSecretName) |
| api-gateway authN/authZ | 7 | High | `auth.go:20` (ConstantTimeCompare), `main.go:45` (fail-closed) |
| Supply chain: SBOM + sign + provenance | 8 | Medium | `release.yml:201` cosign, `:527` syft SBOM, `:510` SLSA L2 |
| Image digest pinning + drift gate | 9 | High | `images/images.yaml`, `pr-checks.yml:336` |
| Container hardening | 9 | High | `Dockerfile.service:49`, `_helpers.tpl:76`/`:89` |
| CI security gates (CVE/SAST/secrets) | 8 | High | `ci.yml:689`/`:870`, `pr-checks.yml:60`/`:366` |
| Network attack surface (K8s) | 7 | Medium | `networkpolicy.yaml:12` (Ingress+Egress), `:15` (no from-selector) |
| Secrets hygiene | 8 | Medium | `ci.yml:286`, `pr-checks.yml:385` |

**Drift test:**

| Claim | Result | Evidence |
|---|---|---|
| C2 — mTLS enforced between all services | **PARTIAL** | Real mTLS (`tlscreds.go:47`) but fails open (`tlscreds.go:20`), default chart insecure (`engine-adapter/values.yaml:51`), gateway prod overlay omits TLS |
| C3 — SBOM per release | **VERIFIED** | `release.yml:527` syft SPDX per digest; `:577` attached to Release |
| C4 — cosign-signed images | **PARTIAL** | Signing wired (`release.yml:201`,`:504`); GHCR signature existence UNKNOWN — cosign absent, no network to run `cosign verify` |

**Red flags (severity-ordered):**
1. **High** — mTLS fails open: services default to `insecure.NewCredentials()` and api-gateway/workflow-compiler prod overlays omit `tlsSecretName`; ADR-020:50/SECURITY.md:89 overstate enforcement (`tlscreds.go:20`, `zynax-api-gateway/values-production.yaml`).
2. **Medium** — NetworkPolicy ingress is port-scoped, not source-scoped — any pod may dial service ports (`networkpolicy.yaml:15`).
3. **Medium** — Single shared static bearer key, no RBAC/scopes/SSO (`auth.go:14`).
4. **Low** — Two standing CVE suppressions (pip-audit PYSEC-2026-196; trivy DS002) — documented/time-boxed (`ci.yml:730`, `.trivyignore`).

**Green flags:**
- Timing-safe, fail-closed gateway auth (`auth.go:20`, `main.go:45`).
- Full supply-chain trifecta: cosign + SBOM + SLSA L2 in `release.yml`.
- Strong container hardening wired into Deployments (`Dockerfile.service:49`, `_helpers.tpl:76`/`:89`, `deployment.yaml:31`).
- Digest pinning SoT + pre-merge drift gate + auditable exceptions (`images/images.yaml`, `pr-checks.yml:336`, `.trivyignore`).
- Layered blocking CVE/SAST/secret gates (`ci.yml:689`/`:870`, `pr-checks.yml:60`/`:366`).

**Open questions / unknowns:** Are signatures/provenance actually present on published GHCR images (cosign/network unavailable)? Is there a startup guard that rejects plaintext in a prod namespace (none found)? Why do two prod overlays omit `tlsSecretName`? NetworkPolicy enforcement depends on a CNI not verifiable from repo.

**Recommendations:** P0 — fail-closed transport + add `tlsSecretName` to the two missing prod overlays; P1 — source-scoped default-deny NetworkPolicies, and run `cosign verify` in CI to close C4; P1 — reconcile SECURITY.md/ADR-020 mTLS wording to "supported, enforced under prod overlay"; P2 — per-caller identity/scoped tokens at the gateway.

**Cross-references:** Architecture/Reliability (mTLS fail-open vs zero-trust claim), Governance/CNCF (supply chain credible; mTLS gap is an M8 external-audit finding), Docs/Truth-pass (SECURITY.md:89 / ADR-020:50 doc-vs-impl drift).

---

## PRIVATE — do not publish (unfixed-issue detail; mirrors repo policy)

> This section is for the orchestrator only and must NOT appear in any public-facing report.

- **Fail-open transport exploitation path (config-dependent, not a code 0-day):** Because
  `tlsCreds()` returns `insecure.NewCredentials()` whenever any of cert/key/CA env paths is empty
  (`services/api-gateway/internal/infrastructure/tlscreds.go:20`, same pattern in
  engine-adapter/task-broker/agent-registry/memory-service/workflow-compiler/event-bus
  `internal/infrastructure/tlscreds.go`), an operator who deploys with the DEFAULT chart values
  (`tlsSecretName: ""`) — or who applies the api-gateway/workflow-compiler production overlays,
  which omit `tlsSecretName` — runs plaintext gRPC between services. Inside the namespace, the
  port-only NetworkPolicy (`helm/zynax-api-gateway/templates/networkpolicy.yaml:15`, no `from`
  selector) does not constrain which pod may connect, so a co-resident workload could connect to,
  observe, or inject inter-service gRPC traffic with no certificate challenge. There is no
  startup guard that refuses to run without TLS in production (unlike the bearer-key guard at
  `main.go:45`). Remediation: fail-closed transport flag + complete the prod overlays + source-
  scoped NetworkPolicies (see P0/P1).
- **No verified signature chain:** could not run `cosign verify` / `gh attestation verify`
  (cosign not installed; no network). If GHCR signatures are in fact absent or unverifiable, the
  C4 "cosign-signed images" claim would downgrade from PARTIAL to CONTRADICTED — flagged for the
  orchestrator to verify with registry access.
# Agent 5.5 — Engineering (Wave A, ground-truth) — Issue #1402

Audit target: `the repository root` @ HEAD `e3135a6` (read-only).
Method: Grep/Read + simple single bash commands. Go commands run `GOWORK=off` inside service dirs (ADR-017).
Executed proof (E1) obtained for the coverage and Go-lint drift tests; mypy execution blocked by a sandbox tool-version mismatch (marked UNKNOWN-by-execution, config still cited).

---

## (a) §3.4 Handoff packet

```yaml
agent: "5.5 Engineering"
wave: "A"
dimension_groups: ["D4", "D3", "D9"]
overall_score: 8
overall_confidence: "High"
sub_scores:
  - dimension: "Domain code quality / cohesion / complexity"
    score: 9
    confidence: "High"
    justification: "Sampled 6 domain pkgs: pure logic, hexagonal, sentinel-error + %w wrapping, scoped access control, tight cohesion; cyclop≤16 enforced and passing."
    evidence:
      - "services/task-broker/internal/domain/service.go:43 (DispatchTask: validate→save→detached async, clean)"
      - "services/engine-adapter/internal/domain/interpreter.go:49 (Run state machine, fail-closed guards)"
      - "services/engine-adapter/internal/domain/datacontext.go:44 (ScopeError — loud cross-run denial, no silent fallback)"
      - "services/workflow-compiler/internal/domain/manifest.go:126 (4-phase parser, collects all errors)"
      - "services/workflow-compiler/internal/domain/policy_gate.go:119 (PolicyGate.Check, stateless, documented)"
  - dimension: "Layer boundaries (no business logic in main.go/api)"
    score: 9
    confidence: "High"
    justification: "453-line engine-adapter main.go is pure wiring/lifecycle (config, gRPC/HTTP/admin servers, signals, graceful drain) — zero business logic; domain pkgs have zero proto imports per their headers."
    evidence:
      - "services/engine-adapter/cmd/engine-adapter/main.go:5 ('All business logic lives in internal/')"
      - "services/engine-adapter/cmd/engine-adapter/main.go:185 (buildEngine: only a config switch)"
      - "services/engine-adapter/internal/domain/interpreter.go:3 ('zero imports from api or infrastructure')"
      - "tools/golangci-lint.yml:32 (contextcheck enforces ctx-first across layers)"
  - dimension: "Error handling (no ignored errors, structured errors)"
    score: 8
    confidence: "High"
    justification: "errcheck enabled and passing 0 issues; `_ =` discards confined to tests + cleanup defers; domain returns typed errors (ScopeError, DataReferenceError, ParseErrors, PolicyGateError)."
    evidence:
      - "tools/golangci-lint.yml:20 (errcheck 'never discard errors') + lint run → '0 issues'"
      - "services/engine-adapter/internal/domain/datacontext.go:60 (DataReferenceError, 'no implicit fallback')"
      - "services/task-broker/internal/domain/service.go:309 (newTaskID panics on crypto/rand failure — fail-fast, justified)"
      - "services/workflow-compiler/internal/domain/policy_gate.go:184 (counter error → pass; FAIL-OPEN, documented tradeoff)"
  - dimension: "Lint strictness vs. blanket suppression"
    score: 9
    confidence: "High"
    justification: "14-linter v2 config (gosec, wrapcheck, errorlint, cyclop≤16, funlen=80, contextcheck...), default:none; ZERO blanket //nolint — every directive names a specific linter and almost all carry a justification."
    evidence:
      - "tools/golangci-lint.yml:17-33 (linter set, default: none)"
      - "grep nolint services/*.go → all scoped: //nolint:gosec G115 enum casts, //nolint:funlen w/ reason; none bare"
      - "services/task-broker/internal/domain/service.go:246 (//nolint:gosec // G115: idx bounded by len)"
      - "services/workflow-compiler/internal/domain/manifest.go:126 (//nolint:funlen w/ 4-phase rationale)"
  - dimension: "Cross-service consistency / shared libs"
    score: 8
    confidence: "High"
    justification: "Shared libs/zynaxobs (tracing+metrics+propagation) and libs/zynaxconfig used by every service; identical TLSCreds, Probes, getEnv helpers, SPDX + package-doc headers across services."
    evidence:
      - "libs/zynaxobs/tracing.go, metrics.go, propagation.go (shared obs lib)"
      - "services/engine-adapter/cmd/engine-adapter/main.go:31 (imports libs/zynaxobs)"
      - "grep tlscreds.go → 6 services share an identical TLSCreds //nolint:gosec ReadFile pattern"
  - dimension: "Python adapter/SDK quality (typing/async/Protocol)"
    score: 7
    confidence: "Medium"
    justification: "SDK + langgraph adapter: mypy strict=true, full annotations, correct grpc.aio async (no blocking-in-async), specific excepts, Google docstrings. BUT ADR-010's AgentRuntime Protocol is absent — SDK ships a concrete base class instead."
    evidence:
      - "agents/sdk/pyproject.toml:41 (strict = true)"
      - "agents/adapters/langgraph/src/langgraph_adapter/handler.py:73 (asyncio.wait_for ticker — correct async)"
      - "agents/sdk/src/zynax_sdk/agent.py:59 (class Agent(AgentServiceServicer, ABC) — concrete base class)"
      - "docs/adr/ADR-010-pluggable-agent-runtime.md:54 ('AgentRuntime is a Protocol — never a base class')"
      - "grep -rn 'AgentRuntime|runtime_checkable|class.*Protocol' agents/sdk/src → no match"
  - dimension: "Comment density / naming / readability"
    score: 9
    confidence: "High"
    justification: "Doc comments on every exported symbol with ADR/canvas citations explaining WHY; intent-revealing names; the 'why' (not 'what') comment style is consistently applied."
    evidence:
      - "services/task-broker/internal/domain/service.go:83 (detach: explains why goroutine outlives RPC)"
      - "services/engine-adapter/internal/domain/interpreter.go:191 (cel program cache rationale for Temporal replay determinism)"
      - "services/engine-adapter/internal/domain/datacontext.go:82 (WorkflowDataContext doc: scope/lifetime/ADR-029)"
drift_test:
  - claim: "≥90% domain coverage on internal/domain/ for every Go service"
    result: "VERIFIED"
    evidence:
      - "task-broker→92.1%; engine-adapter→92.8%; agent-registry→94.0%; api-gateway→96.7%"
      - "workflow-compiler→97.5%/92.7%/95.9% (domain, ir, validators); memory-service→100%; event-bus→100%"
      - "all 7 services run via `GOWORK=off go test ./internal/domain/... -cover` → every result ≥90%"
      - "Makefile:286-302 hard CI gate fails build below 90% (not just a doc target)"
  - claim: "Go code is lint-clean under the strict golangci-lint config"
    result: "VERIFIED"
    evidence:
      - "cd services/task-broker && golangci-lint run ./... --config ../../tools/golangci-lint.yml → '0 issues.'"
      - "cd services/engine-adapter && golangci-lint run → '0 issues.'"
      - "cd services/workflow-compiler && golangci-lint run → '0 issues.'"
  - claim: "Python is lint-clean under ruff + mypy --strict"
    result: "PARTIAL"
    evidence:
      - "agents/sdk/pyproject.toml:41 strict=true; ruff select includes E,F,W,I,D,UP,S,B in langgraph (E3 config)"
      - "executed `mypy src/ --strict` on agents/sdk → mypy 1.20.2 INTERNAL ERROR (sandbox tool-version ≠ pinned tools image); E1 not obtainable here"
      - "Makefile:324 enforces pytest --cov-fail-under=90 + mypy --strict in CI (gate exists)"
  - claim: "ADR-010 — agents use a Protocol-based pluggable runtime (structural subtyping, never a base class)"
    result: "CONTRADICTED"
    evidence:
      - "docs/adr/ADR-010-pluggable-agent-runtime.md:54 mandates Protocol, never base class; lines 30-40 define AgentRuntime + zynax-sdk[langgraph] extras"
      - "agents/sdk/src/zynax_sdk/agent.py:59 ships `class Agent(AgentServiceServicer, ABC)` with __init_subclass__ registration"
      - "no AgentRuntime/runtime_checkable/Protocol anywhere in agents/sdk/src; extras model not present"
red_flags:
  - severity: "Medium"
    finding: "ADR-010 intent-vs-implementation drift: the documented AgentRuntime Protocol + framework-extras model (zynax-sdk[langgraph]) is not implemented; the Python SDK uses a concrete `Agent` base class that the ADR explicitly forbids. Functional impact is low (the gRPC proto is the real boundary), but it is exactly the doc-vs-code drift class flagged in Part 1 §1.10."
    evidence:
      - "docs/adr/ADR-010-pluggable-agent-runtime.md:54"
      - "agents/sdk/src/zynax_sdk/agent.py:59"
  - severity: "Low"
    finding: "Two deliberate fail-open paths in domain code. PolicyGate quota check passes when the active-invocation counter errors; evalGuard is fail-CLOSED (safer) but the quota gate is fail-OPEN. Documented as availability tradeoffs but worth an explicit security sign-off (defer to 5.2)."
    evidence:
      - "services/workflow-compiler/internal/domain/policy_gate.go:184-190 (counter error → return nil)"
      - "services/engine-adapter/internal/domain/interpreter.go:220 (evalGuard fail-closed — for contrast)"
  - severity: "Low"
    finding: "Three //nolint:funlen suppressions on domain parsers/validators (ParseManifest, convertState, CircularTransitionDetector.Validate). All justified, but they are the longest domain funcs and the highest-leverage future refactor targets."
    evidence:
      - "services/workflow-compiler/internal/domain/manifest.go:126,215"
      - "services/workflow-compiler/internal/domain/validators/structural.go:58"
green_flags:
  - strength: "≥90% domain coverage is a real, enforced CI gate — independently reproduced on all 7 services (92.1%–100%), not a marketing number."
    evidence: ["Makefile:286-302", "7× `go test ./internal/domain/... -cover` ≥90%"]
  - strength: "Lint is genuinely strict (14 linters incl. gosec/wrapcheck/errorlint/cyclop/contextcheck) AND actually passes 0 issues — reproduced on 3 services. No blanket //nolint anywhere; every suppression is scoped + justified."
    evidence: ["tools/golangci-lint.yml:17-33", "3× golangci-lint run → '0 issues.'", "nolint census: all scoped"]
  - strength: "Clean hexagonal boundaries: even a 453-line main.go contains zero business logic; domain packages declare and honour zero api/infrastructure imports."
    evidence: ["services/engine-adapter/cmd/engine-adapter/main.go:5", "services/engine-adapter/internal/domain/interpreter.go:3"]
  - strength: "Shared platform libraries (zynaxobs, zynaxconfig) give real cross-service consistency in tracing, metrics, TLS, probes, and config — not copy-paste."
    evidence: ["libs/zynaxobs/*.go", "services/engine-adapter/cmd/engine-adapter/main.go:31"]
  - strength: "Comment discipline explains WHY (determinism, detached ctx, scope isolation) with ADR/canvas citations — a Staff+ habit."
    evidence: ["services/task-broker/internal/domain/service.go:83", "services/engine-adapter/internal/domain/interpreter.go:191"]
open_questions:
  - "Is ADR-010 stale (SDK design deliberately changed and ADR not updated), or is the Protocol runtime genuinely unbuilt? Needs maintainer confirmation — feeds the contradiction register."
  - "Does the PolicyGate fail-open quota path have an explicit security decision behind it, or is it incidental?"
unknowns:
  - "Python lint-clean at HEAD by EXECUTION — mypy 1.20.2 in the sandbox INTERNAL-ERRORs; only the pinned tools image is authoritative. Config (strict=true) and the CI gate are verified; the green result itself is UNKNOWN here."
  - "Whole-repo lint/coverage in one pass (only the documented 7 services + 3 lint runs were executed; remaining services' lint and adapter coverage not individually re-run, though the same Makefile gate covers them)."
cross_references:
  - to_agent: "5.14"
    note: "Highest-leverage debt items: (1) reconcile ADR-010 vs SDK base-class; (2) the 3 funlen-suppressed parsers as refactor candidates."
    evidence: ["docs/adr/ADR-010-pluggable-agent-runtime.md:54", "services/workflow-compiler/internal/domain/manifest.go:126"]
  - to_agent: "5.7"
    note: "Coverage gate reproduced ≥90% on all 7 domains (E1). Confirms 5.7's testing-tier claim at the domain layer."
    evidence: ["Makefile:286-302", "7× cover outputs 92.1–100%"]
  - to_agent: "5.2"
    note: "Two domain fail-open/fail-closed guard behaviours (quota fail-open; CEL guard fail-closed) need a security read — out of my scoring zone."
    evidence: ["services/workflow-compiler/internal/domain/policy_gate.go:184", "services/engine-adapter/internal/domain/interpreter.go:220"]
  - to_agent: "5.1"
    note: "C7 (stateless workflow-compiler) — PolicyGate.Check is stateless modulo injected counter (policy_gate.go:74); does not contradict, supports the refactor claim. Owner 5.1 to score."
    evidence: ["services/workflow-compiler/internal/domain/policy_gate.go:72-81"]
recommendations:
  - priority: "P1"
    action: "Reconcile ADR-010 with the shipped SDK: either implement the AgentRuntime Protocol + extras model, or update/supersede the ADR to record the concrete-base-class decision."
    rationale: "Removes a live doc-vs-code drift in exactly the class the project's own Truth Pass exists to prevent."
  - priority: "P2"
    action: "Add an explicit comment/ADR note on the PolicyGate fail-open quota path (counter error → allow) and confirm it is the intended availability/safety tradeoff."
    rationale: "A fail-open security control should be a recorded decision, not an inference."
  - priority: "P2"
    action: "Refactor the three funlen-suppressed parsers (ParseManifest, convertState, CircularTransitionDetector.Validate) into phase helpers."
    rationale: "Removes the only complexity suppressions in the domain layer; low risk, high readability payoff."
```

---

## (b) §6.2 Prose section

## Engineering — Score: 8 (High)

**Mission recap:** Assess Go-service + Python-adapter code quality — complexity, cohesion, error handling, naming, boundaries, anti-patterns, cross-service consistency, lint strictness — and drift-test the ≥90%-coverage and lint-clean claims at HEAD.

**Verdict:** This is strong, production-credible engineering that would largely pass a Staff+ review at a top org. Across six sampled domain packages the code is pure, cohesive, hexagonally clean, and consistently documents *why* (with ADR/canvas citations) rather than *what*. The two headline quality claims — ≥90% domain coverage and a strict lint that actually passes — are not marketing: I reproduced both with executed commands. The one real blemish is an ADR-vs-implementation drift in the Python SDK (ADR-010's Protocol runtime is unbuilt), which is low-impact functionally but is precisely the doc-vs-code drift class this diligence is built to catch.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Domain quality / cohesion / complexity | 9 | High | service.go:43, interpreter.go:49, datacontext.go:44, manifest.go:126 |
| Layer boundaries (no logic in main/api) | 9 | High | engine-adapter/main.go:5,185; interpreter.go:3 |
| Error handling | 8 | High | golangci:20 (errcheck, 0 issues); datacontext.go:60; policy_gate.go:184 (fail-open) |
| Lint strictness vs. suppression | 9 | High | golangci:17-33; nolint census all-scoped; service.go:246 |
| Cross-service consistency / shared libs | 8 | High | libs/zynaxobs/*; 6× shared tlscreds.go |
| Python adapter/SDK quality | 7 | Medium | sdk/pyproject.toml:41; agent.py:59 vs ADR-010:54 |
| Comments / naming / readability | 9 | High | service.go:83; interpreter.go:191; datacontext.go:82 |

**Drift test:**
- "≥90% domain coverage, every Go service" → **VERIFIED**. Reproduced on all 7: task-broker 92.1%, engine-adapter 92.8%, agent-registry 94.0%, api-gateway 96.7%, workflow-compiler 97.5/92.7/95.9%, memory-service 100%, event-bus 100%. It is a hard CI gate (`Makefile:286-302`), not a soft target.
- "Go lint-clean under strict config" → **VERIFIED**. `golangci-lint run` with the repo's 14-linter `tools/golangci-lint.yml` returns "0 issues." on task-broker, engine-adapter, and workflow-compiler.
- "Python lint-clean (ruff + mypy --strict)" → **PARTIAL**. Config (`strict=true`) and the CI gate (`--cov-fail-under=90`, `mypy --strict`) are verified; executing mypy here hit a sandbox tool-version INTERNAL ERROR, so the green *result* is UNKNOWN-by-execution.
- "ADR-010 Protocol-based pluggable runtime" → **CONTRADICTED**. ADR-010:54 says "Protocol — never a base class"; the SDK ships `class Agent(AgentServiceServicer, ABC)` and no `AgentRuntime`/`Protocol` exists in the source.

**Red flags (severity-ordered):**
1. **Medium — ADR-010 doc-vs-code drift.** Documented Protocol runtime + `zynax-sdk[langgraph]` extras are unimplemented; the SDK uses the forbidden base-class pattern (`agents/sdk/src/zynax_sdk/agent.py:59` vs `docs/adr/ADR-010-pluggable-agent-runtime.md:54`). Low functional risk (the gRPC contract is the true boundary) but a live drift.
2. **Low — fail-open quota gate.** `PolicyGate` allows compilation when its counter errors (`policy_gate.go:184-190`); contrast the fail-closed CEL guard (`interpreter.go:220`). Documented, but a fail-open control deserves an explicit decision (→ 5.2).
3. **Low — three `funlen`-suppressed parsers** are the domain layer's only complexity exceptions and the top refactor targets.

**Green flags:**
- Coverage gate is real and reproduced (92.1–100% across 7 services; `Makefile:286-302`).
- Lint is strict *and* passes 0 issues; no blanket `//nolint` anywhere — every suppression names its linter and is justified (`tools/golangci-lint.yml:17-33`).
- Clean hexagonal boundaries even in a 453-line wiring `main.go` (`engine-adapter/main.go:5`).
- Genuine shared platform libs (`libs/zynaxobs`, `libs/zynaxconfig`) drive cross-service consistency, not copy-paste.
- Comment discipline explains rationale with ADR/canvas references (`service.go:83`, `interpreter.go:191`).

**Open questions / unknowns:**
- Is ADR-010 stale or the runtime genuinely unbuilt? Needs maintainer confirmation.
- Python lint *result* at HEAD is unverifiable in this sandbox (mypy version mismatch); config and gate are verified.
- Whole-repo single-pass lint/coverage not executed; the same Makefile gate covers the unrun services.

**Recommendations:**
- **P1** — Reconcile ADR-010 with the shipped SDK (implement the Protocol, or supersede the ADR).
- **P2** — Document/sign-off the PolicyGate fail-open quota path.
- **P2** — Refactor the three `funlen`-suppressed parsers into phase helpers.

**Cross-references:** debt items → 5.14; coverage corroboration → 5.7; fail-open/fail-closed guard semantics → 5.2; stateless-compiler (C7) supporting evidence → 5.1.
# Agent 5.7 — Testing — Due-Diligence Findings (Wave A, ground-truth, issue #1402)

Audit target: `the repository root` @ HEAD `e3135a6` (branch `main`).
Read-only audit. Every claim carries `path:line` or a command→output, or is marked `UNKNOWN`.

---

## (a) §3.4 Handoff packet (YAML)

```yaml
agent: "5.7 Testing"
wave: "A"
dimension_groups: ["D9", "D6", "D8"]   # D9 primary; contributes D6 (eng quality) + D8 (reliability)
overall_score: 8
overall_confidence: "High"

sub_scores:
  - dimension: "BDD contract coverage (ADR-016: every RPC ≥1 scenario)"
    score: 9
    confidence: "High"
    justification: "All 33 gRPC RPCs across 7 services are referenced in feature files; 306 scenarios in 18 features; full proto BDD suite passes."
    evidence:
      - "find protos/tests -name '*.feature' → 18 files"
      - "grep -rhE '^\\s*Scenario:' protos/tests --include=*.feature | wc -l → 306"
      - "grep -rnE '^\\s*rpc ' protos/zynax --include=*.proto → 33 RPCs across 7 services"
      - "per-RPC grep loop: each of 33 RPC names appears in ≥1 feature file (CompileWorkflow..WatchWorkflow all ≥1)"
      - "cd protos/tests && GOWORK=off go test ./... → all 10 suites 'ok' (E1)"
  - dimension: "Domain unit-coverage gate (≥90%) — exists AND blocks"
    score: 9
    confidence: "High"
    justification: "Gate enforced in _test-go.yml (failed=true→exit 1); threshold in tools/coverage-gates.env; all 7 service domains measured ≥90% at HEAD (E1)."
    evidence:
      - ".github/workflows/_test-go.yml:155-181 (gate sets failed=true then `$failed && exit 1`)"
      - "tools/coverage-gates.env:4 (COVERAGE_DOMAIN_GATE=90)"
      - "GOWORK=off go test ./internal/domain/... → task-broker 92.1%, workflow-compiler 97.5/92.7/95.9%, engine-adapter 92.8%, agent-registry 94.0%, memory-service 100%, event-bus 100%, api-gateway 96.7% (E1)"
      - "Python gate: _test-python.yml:65,78 (--cov-fail-under=90)"
      - "adapter gate ≥85% _test-go.yml:265-280; cmd/zynax ≥79% :307-322; cmd/zynax-ci ≥80% :327-344 (all exit 1)"
  - dimension: "Domain coverage gate is per-changed-service, not a global floor"
    score: 6
    confidence: "High"
    justification: "Gate only measures+enforces services whose <SVC>_CHANGED==true; an unchanged service could drift sub-90 on a given PR without being gated."
    evidence:
      - ".github/workflows/_test-go.yml:132-134 (if CHANGED != true → 'SKIPPED — not changed'; continue)"
      - ".github/workflows/_test-go.yml:160 (gate only runs `if [ -f domain-coverage.out ]`, which exists only for changed svcs)"
  - dimension: "Integration tests (testcontainers, real backings, in CI)"
    score: 7
    confidence: "High"
    justification: "Real testcontainers/Docker integration; required CI job that FAILS (not skips) when empty + self-check guard; but only 2 of 4 testcontainer services are gated."
    evidence:
      - ".github/workflows/ci.yml:653-666 (go test -tags=integration -race on real Docker daemon, exit 1 on fail)"
      - ".github/workflows/ci.yml:636-651 (self-check: fails if an allowlisted suite loses its //go:build integration files — gate can't degrade to no-op)"
      - "//go:build integration in 4 svcs: task-broker, agent-registry, memory-service, event-bus (grep -rln //go:build integration services)"
      - "ci.yml:608-610,627 (INTEGRATION_SUITES='task-broker agent-registry'; event-bus+memory-service excluded for 'pre-existing deterministic failures')"
  - dimension: "E2E smoke actually applies a workflow + asserts execution"
    score: 8
    confidence: "High"
    justification: "e2e-happy.sh submits a real workflow via api-gateway, polls to terminal succeeded, asserts CloudEvents on NATS JetStream + memory KV roundtrip — genuine, not a no-op. But gated, not a true required gate (shim no-ops it for non-e2e PRs)."
    evidence:
      - "scripts/e2e/e2e-happy.sh:168-203 (POST /api/v1/apply, poll GET /workflows/{id} until succeeded, fail on terminal failure)"
      - "scripts/e2e/e2e-happy.sh:204-300 (REQUIRED CloudEvent assertions on JetStream stream ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW + grpcurl memory Set/Get roundtrip)"
      - ".github/workflows/e2e-smoke.yml:146-167 (happy+failure path temporal leg; argo leg asserts Workflow CR phase Succeeded)"
      - ".github/workflows/e2e-smoke.yml:16-19 (GATED — not a required PR gate; triggers only on helm/services/engine-adapter paths)"
      - ".github/workflows/e2e-smoke-skip.yml:39-46 (required check satisfied by a no-op shim for non-e2e PRs; argo leg intentionally NOT shimmed = advisory)"
  - dimension: "Fuzz / benchmark presence and gating"
    score: 5
    confidence: "High"
    justification: "2 functional fuzz targets + committed seed corpus + benchmarks with committed baseline exist and run, but NO CI workflow invokes make fuzz/make bench/benchstat — they are local-only, they do not gate."
    evidence:
      - "FuzzParseManifest (workflow-compiler) + FuzzEvalGuard (engine-adapter) — go test -list='^Fuzz' (E1)"
      - "GOWORK=off go test -fuzz=^FuzzParseManifest$ -fuzztime=5s → 84-seed baseline, 187 new-interesting, PASS, no crashers (E1)"
      - "services/workflow-compiler/internal/domain/testdata/fuzz_corpus/ (7 committed seeds)"
      - "Makefile:357-377 (bench + fuzz targets); tools/bench-baseline.txt committed"
      - "grep -rliE 'fuzz|bench|benchstat' .github/workflows/ci.yml weekly-audit.yml pr-checks.yml → no output (NOT in CI)"
  - dimension: "Test smells (over-mocking, internals, flaky-skip, GOWORK misuse)"
    score: 6
    confidence: "High"
    justification: "Proto BDD tests contract-stubs (by ADR-016 design, not prod code); 1 hard service-level engine-adapter BDD skip; remaining t.Skip are legit env guards; no property/mutation/chaos testing; GOWORK=off used correctly."
    evidence:
      - "protos/tests/task_broker_service/steps_test.go:245 (RegisterTaskBrokerServiceServer(srv, tc.stub) — in-test brokerStub, not services/task-broker domain code)"
      - "services/engine-adapter/tests/steps_test.go:17 (t.Skip — entire service-level engine-adapter BDD disabled: 'requires a running Temporal server — deferred to M6/M7')"
      - "other t.Skip are env guards: cmd/zynax/cmd/mcp_test.go:47 (windows), gitops/watcher_test.go:201 (root), validate/context_test.go:290 (fixture absent) — acceptable"
      - "pgregory.net/rapid in go.sum (4 svcs) but imported by 0 .go files → no property-based testing (grep -rln pgregory.net/rapid services --include=*.go → empty)"
      - "no rapid/gopter/testing/quick/toxiproxy/chaos usage found"

drift_test:
  - claim: "140+ BDD scenarios across all services (ROADMAP.md:78, README.md:412)"
    result: "VERIFIED"
    evidence:
      - "grep -rhE '^\\s*Scenario:' protos/tests --include=*.feature | wc -l → 306 (>140)"
      - "18 .feature files; full suite passes: cd protos/tests && GOWORK=off go test ./... → all ok (E1)"
  - claim: "Every proto method has a BDD scenario (ROADMAP.md:78; ADR-016)"
    result: "VERIFIED"
    evidence:
      - "33 RPCs (grep rpc protos/zynax) all referenced in feature files (per-RPC grep loop, each ≥1)"
  - claim: "≥90% coverage on internal/domain (AGENTS.md:108,123) — gate blocks"
    result: "VERIFIED"
    evidence:
      - "_test-go.yml:155-181 (failed=true→exit 1) + coverage-gates.env:4 (=90)"
      - "all 7 service domains measured ≥90% at HEAD (92.1/97.5/92.8/94.0/100/100/96.7%) (E1)"
    caveat: "Enforcement is per-changed-service (_test-go.yml:132-134), not a global per-PR floor across all services."

red_flags:
  - severity: "Medium"
    finding: "Domain coverage gate enforces only on CHANGED services per PR; an unchanged service can silently drift below 90% and never be re-gated until it is next touched. (Mitigant: ALL 7 domains currently ≥90% at HEAD, so no active drift.)"
    evidence: [".github/workflows/_test-go.yml:132-134", ".github/workflows/_test-go.yml:160"]
  - severity: "Medium"
    finding: "Fuzz + benchmarks exist and are functional but do NOT gate CI (no workflow runs make fuzz / make bench / benchstat); the bench-baseline.txt regression-guard described in the Makefile is never enforced in CI."
    evidence: ["grep -rliE 'fuzz|bench|benchstat' .github/workflows/{ci,weekly-audit,pr-checks}.yml → empty", "Makefile:357-377", "tools/bench-baseline.txt"]
  - severity: "Medium"
    finding: "engine-adapter — the core execution layer — has its entire SERVICE-level BDD suite hard-skipped (needs a Temporal server). Engine-adapter is covered only by proto-contract BDD (bufconn) + domain unit tests; the wired Temporal interpreter path is exercised only by the gated, non-required e2e-smoke."
    evidence: ["services/engine-adapter/tests/steps_test.go:17", ".github/workflows/e2e-smoke.yml:16-19"]
  - severity: "Medium"
    finding: "Integration (testcontainers) gate covers only 2 of 4 testcontainer-bearing services; event-bus + memory-service integration suites are excluded for 'pre-existing deterministic failures' (event-bus godog step arity + DLQ timeout; memory-service DeleteNamespace cascade) — honestly documented but a real coverage hole on persistence/eventing paths."
    evidence: [".github/workflows/ci.yml:608-610", ".github/workflows/ci.yml:627"]
  - severity: "Low"
    finding: "e2e-smoke 'temporal' is a required check only nominally: a no-op shim (e2e-smoke-skip.yml) satisfies it for any PR not touching e2e paths, so most PRs never run a real cluster E2E; the argo leg is advisory only."
    evidence: [".github/workflows/e2e-smoke-skip.yml:39-46"]
  - severity: "Low"
    finding: "Proto-contract BDD validates against in-test stubs (e.g. brokerStub), not the real service domain implementation; '306 BDD scenarios pass' attests contract semantics, not that production service code satisfies them (that is the unit + integration tiers' job). A reader could over-read the headline count."
    evidence: ["protos/tests/task_broker_service/steps_test.go:28,245"]
  - severity: "Low"
    finding: "No property-based, mutation, or chaos testing. pgregory.net/rapid is a dangling go.sum entry imported by zero source files."
    evidence: ["grep -rln pgregory.net/rapid services --include=*.go → empty", "no gopter/testing.quick/toxiproxy/chaos found"]

green_flags:
  - strength: "Genuine, blocking multi-tier coverage gates wired to a single source of truth (tools/coverage-gates.env): domain ≥90, adapter ≥85, CLI ≥79/80, Python ≥90 — all exit-1 on breach."
    evidence: ["tools/coverage-gates.env", ".github/workflows/_test-go.yml:155-181,265-344", ".github/workflows/_test-python.yml:65,78"]
  - strength: "All 7 service domains exceed 90% at HEAD by executed proof, several at 100% — the ≥90% claim is real, not aspirational."
    evidence: ["GOWORK=off go test ./internal/domain/... per service (E1): 92.1/97.5/92.8/94.0/100/100/96.7%"]
  - strength: "BDD-before-code discipline at gRPC boundaries: 306 scenarios (18 features) over bufconn in-process gRPC, every RPC covered, full suite green (E1)."
    evidence: ["protos/tests/testserver/server.go:19 (bufconn)", "cd protos/tests && GOWORK=off go test ./... → all ok"]
  - strength: "Integration gate is anti-erosion engineered: it FAILS (never silently skips) when its scope is empty, and a self-check fails if an allowlisted suite loses its //go:build integration files (#553)."
    evidence: [".github/workflows/ci.yml:600-602,636-651"]
  - strength: "E2E happy-path is a real end-to-end assertion (submit workflow → poll terminal succeeded → assert CloudEvents on NATS JetStream → memory KV roundtrip), with no skip path on the required assertions and a 2-engine (temporal+argo) matrix."
    evidence: ["scripts/e2e/e2e-happy.sh:168-300", ".github/workflows/e2e-smoke.yml:55-61,146-167"]
  - strength: "Functional fuzzing with committed seed corpus: FuzzParseManifest + FuzzEvalGuard run clean (187 new-interesting inputs, 0 crashers in 5s)."
    evidence: ["GOWORK=off go test -fuzz=^FuzzParseManifest$ -fuzztime=5s → PASS (E1)", "testdata/fuzz_corpus/ (7 seeds)"]

open_questions:
  - "Will the per-changed-service coverage gate catch a regression introduced by a wide refactor that touches many files but leaves an untouched low-coverage package below the floor?"
  - "When will event-bus + memory-service integration suites be fixed and promoted into INTEGRATION_SUITES (the documented follow-up)?"
  - "Is there a plan to wire fuzz/bench into a scheduled job (weekly-audit) so the committed bench-baseline regression-guard actually fires?"

unknowns:
  - "Whether the e2e-smoke kind-cluster run is actually green on recent PRs — could not execute a kind cluster in this sandbox; verified only that the scripts assert real behavior (E2/E3), not a live green run (no E1)."
  - "Live integration-test pass status — testcontainers needs a Docker daemon + network not exercised here; confirmed they COMPILE under -tags=integration but did not run the postgres-backed suites (no E1 on integration pass)."

cross_references:
  - to_agent: "5.9 DevOps"
    note: "CI gate mechanics (required-check set, e2e-smoke-skip shim, selective bdd-select fail-open) are owned by 5.9; I cite them for test-gate efficacy only."
    evidence: [".github/workflows/_test-go.yml", ".github/workflows/e2e-smoke-skip.yml", "cmd/zynax-ci/cmd/bdd_select.go:29-53"]
  - to_agent: "5.5 Engineering"
    note: "Domain code quality + the brokerStub-vs-real-impl layering (proto BDD tests contracts, not services/*/internal/domain) is 5.5's zone; the ≥90% domain coverage figures support their quality score."
    evidence: ["protos/tests/task_broker_service/steps_test.go:245", "per-service domain coverage (E1)"]
  - to_agent: "5.6 Performance"
    note: "Benchmarks (engine-adapter, workflow-compiler) + tools/bench-baseline.txt exist but do not gate; 5.6 should weigh whether perf regressions can land unnoticed."
    evidence: ["Makefile:357-365", "tools/bench-baseline.txt"]
  - to_agent: "5.1 Architecture"
    note: "ADR-016 layered-testing contract is materially honored (BDD-before-code at gRPC boundaries, buf-breaking as gate, contract stubs). policy.proto + cloudevents.proto are message-only (no RPCs) and are tested as schema/envelope features."
    evidence: ["protos/zynax/v1/policy.proto (no service/rpc)", "protos/tests/features/policy_enforcement.feature"]

recommendations:
  - priority: "P1"
    action: "Make the domain coverage gate a global per-PR floor (measure all 7 service domains every PR, not only CHANGED ones), or add a scheduled full-matrix coverage job."
    rationale: "Closes the silent-drift window; all domains are already ≥90%, so the change is low-risk and locks in the current state."
  - priority: "P1"
    action: "Wire make fuzz (short campaign) + make bench/benchstat-vs-baseline into a scheduled CI job (e.g. weekly-audit.yml) so the committed bench-baseline.txt regression-guard and fuzz corpus actually gate something."
    rationale: "Today they are local-only; a perf regression or a new parser crasher can land on main undetected."
  - priority: "P1"
    action: "Restore engine-adapter service-level BDD by wiring a Temporal dev-container into service-test CI (the skip already promises 'M6/M7'); promote event-bus + memory-service integration suites into INTEGRATION_SUITES once their deterministic failures are fixed."
    rationale: "The core execution + persistence/eventing paths are the highest-risk untested-in-required-CI surface."
  - priority: "P2"
    action: "Either adopt pgregory.net/rapid for real property tests on the IR/manifest parser or drop it from go.mod/go.sum."
    rationale: "A dangling test-dependency implies property testing that does not exist; tidy the supply-chain surface."
  - priority: "P2"
    action: "Promote the e2e-smoke argo leg from advisory to required once it has a stable green history (decision already recorded on #1092)."
    rationale: "ADR-015 multi-engine portability is a headline claim; only the temporal leg is currently (nominally) required."
```

---

## (b) §6.2 Prose section

## 5.7 Testing — Score: 8 (High)

Mission recap: Test architect / SDET — assess the pyramid (BDD, unit, integration, E2E, fuzz, bench, property, chaos) and whether the gates are real and blocking.

Verdict: Zynax has a genuinely strong, multi-tier test discipline that survives contact with executed proof: 306 BDD contract scenarios (every one of 33 gRPC RPCs covered) pass over in-process bufconn, and all seven service domains measure ≥90% statement coverage at HEAD against a real, exit-1 coverage gate. This is the opposite of "coverage theater." The score is held at 8 rather than 9 by four honest-but-real gaps: the domain gate enforces only on *changed* services per PR (drift window), fuzz/benchmarks are functional but local-only (they never gate CI), the core engine-adapter's service-level BDD is hard-skipped pending a Temporal dev-container, and integration (testcontainers) coverage gates only 2 of the 4 services that own integration suites.

Sub-dimension scores:

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| BDD contract coverage (every RPC ≥1 scenario) | 9 | High | 306 scenarios / 18 features; 33/33 RPCs covered; full suite green (E1) |
| Domain coverage gate ≥90% exists AND blocks | 9 | High | `_test-go.yml:155-181`; `coverage-gates.env:4`; 7/7 domains ≥90% (E1) |
| Coverage gate is per-changed-service (not global) | 6 | High | `_test-go.yml:132-134,160` |
| Integration tests (testcontainers, in CI) | 7 | High | `ci.yml:636-666`; only task-broker+agent-registry gated |
| E2E smoke applies workflow + asserts execution | 8 | High | `scripts/e2e/e2e-happy.sh:168-300`; gated not required |
| Fuzz / bench presence + gating | 5 | High | 2 fuzz targets + bench baseline exist; not in any CI workflow |
| Test smells | 6 | High | proto-BDD stubs by design; 1 hard engine-adapter BDD skip; no property/mutation/chaos |

Drift test:
- "140+ BDD scenarios across all services" (`ROADMAP.md:78`, `README.md:412`) → **VERIFIED** — 306 actual scenarios in 18 feature files; full `protos/tests` suite passes (E1).
- "Every proto method has a BDD scenario" (`ROADMAP.md:78`, ADR-016) → **VERIFIED** — all 33 RPCs referenced in features.
- "≥90% coverage on `internal/domain`, gate blocks" (`AGENTS.md:108,123`) → **VERIFIED** — gate is exit-1 (`_test-go.yml:155-181`) and all 7 domains are ≥90% at HEAD by executed proof (92.1/97.5/92.8/94.0/100/100/96.7%). Caveat: enforcement is per-changed-service, not a global per-PR floor.

Red flags (severity-ordered):
1. (Medium) Domain coverage gate enforces only on CHANGED services — silent-drift window (`_test-go.yml:132-134`). Mitigated today: all domains ≥90%.
2. (Medium) Fuzz + benchmarks do not gate CI — no workflow runs `make fuzz`/`make bench`/`benchstat`; the committed `bench-baseline.txt` regression-guard never fires (`grep` over workflows → empty; `Makefile:357-377`).
3. (Medium) engine-adapter service-level BDD entirely hard-skipped pending a Temporal server (`services/engine-adapter/tests/steps_test.go:17`); the wired Temporal path is only exercised by the gated, non-required e2e-smoke.
4. (Medium) Integration gate covers 2 of 4 testcontainer services; event-bus + memory-service excluded for "pre-existing deterministic failures" (`ci.yml:608-610`).
5. (Low) e2e-smoke "temporal" is required only nominally — a no-op shim satisfies it for non-e2e PRs (`e2e-smoke-skip.yml:39-46`); argo leg advisory.
6. (Low) Proto-contract BDD tests run against in-test stubs (`brokerStub`), not the real service code — headline scenario count attests contract semantics, not production-code conformance (`task_broker_service/steps_test.go:28,245`).
7. (Low) No property/mutation/chaos testing; `pgregory.net/rapid` is a dangling, unimported go.sum entry.

Green flags:
- Real, blocking, single-source-of-truth coverage gates across all tiers (domain ≥90 / adapter ≥85 / CLI ≥79–80 / Python ≥90), all exit-1 (`coverage-gates.env`; `_test-go.yml`; `_test-python.yml:65,78`).
- ≥90% domain coverage proven by execution on all 7 services — not aspirational.
- BDD-before-code at gRPC boundaries: 306 green scenarios over bufconn, every RPC covered.
- Integration gate is anti-erosion engineered — FAILS (never silently skips) when empty and self-checks that allowlisted suites still own their integration files (`ci.yml:600-602,636-651`).
- E2E happy-path is a true end-to-end assertion (submit → poll terminal succeeded → CloudEvents on NATS JetStream → memory KV roundtrip) on a temporal+argo matrix.
- Functional fuzzing with committed seed corpus (187 new-interesting inputs, 0 crashers in a 5s run).

Open questions / unknowns:
- Could not run a kind-cluster e2e or the postgres testcontainer suites in this sandbox — verified the scripts/configs assert real behavior (E2/E3) but have no E1 on a live green e2e/integration run.
- When will event-bus + memory-service integration suites be fixed and promoted into the required `INTEGRATION_SUITES`?
- Will the per-changed-service coverage gate be hardened into a global floor?

Recommendations:
- P1 — Make the domain coverage gate a global per-PR floor (or scheduled full matrix) to close the silent-drift window.
- P1 — Wire `make fuzz` + benchstat-vs-baseline into a scheduled CI job so they actually gate.
- P1 — Restore engine-adapter service BDD (Temporal dev-container) and promote event-bus/memory-service integration suites.
- P2 — Adopt `rapid` for real property tests on the IR parser, or remove the dangling dependency.
- P2 — Promote the e2e-smoke argo leg from advisory to required (per #1092).

Cross-references: 5.9 DevOps (CI required-check set, e2e-smoke-skip shim, bdd-select fail-open); 5.5 Engineering (domain quality + proto-BDD-stub-vs-real-impl layering); 5.6 Performance (benchmarks exist but do not gate); 5.1 Architecture (ADR-016 layered-testing contract honored; policy/cloudevents protos are message-only and tested as schema features).
# Agent 5.9 — DevOps / Release Engineering — DD Wave A

> REPO: the repository root · HEAD `e3135a6` (2026-06-20) · branch `main`
> READ-ONLY audit. Evidence cited as `path:line` (E2/E3/E4), command→output (E1), or marked UNKNOWN.
> Issue #1402 · Wave A (ground-truth) · Dimension group D8 (contributes D5/D9/D16).

---

## (a) §3.4 Handoff packet

```yaml
agent: "5.9 DevOps / Release Engineer"
wave: "A"
dimension_groups: ["D8", "D5", "D9", "D16"]
overall_score: 8
overall_confidence: "High"
sub_scores:
  - dimension: "Blocking quality gates (required vs advisory)"
    score: 9
    confidence: "High"
    justification: "12 required status checks enforced via modern ruleset incl. dco, security, lint x3, tests, integration, e2e, gitleaks; SKIPPED-bypass footgun explicitly defended."
    evidence:
      - "cmd→`gh api repos/zynax-io/zynax/rulesets/17547241 --jq .rules`→required_status_checks contexts: dco, test-unit, security, lint-proto, lint-go, lint-python, 'GitHub Actions workflow lint', 'Conventional Commit title', 'PR size label', 'Secret scan (gitleaks)', 'e2e smoke (temporal)', test-integration; strict_required_status_checks_policy:true"
      - ".github/workflows/ci.yml:573-591 (test-unit wrapper uses if: always() to defeat SKIPPED-bypass — #986)"
      - ".github/workflows/ci.yml:602-651 (test-integration self-check FAILS if allowlist degrades to no-op)"
  - dimension: "Reproducibility — build-once / promote-by-retag (ADR-027)"
    score: 9
    confidence: "High"
    justification: "Production service images built EXACTLY ONCE pre-merge in ci.yml, scan-gated, then promoted by manifest retag in release.yml; release.yml contains zero image build steps; one-way-door ADR."
    evidence:
      - ".github/workflows/ci.yml:789-913 (build-images: pre-merge build+Trivy+SBOM to staging lane)"
      - ".github/workflows/release.yml:160-204 (imagetools create retag staging→main-<sha>→latest, cosign sign; no docker build)"
      - "docs/adr/ADR-027-shift-left-pipeline.md:32-47,88-90 ('build exactly once… promote… never rebuild'; rebuild path explicitly rejected)"
  - dimension: "Digest pinning enforcement (ADR-024, check-images SoT)"
    score: 9
    confidence: "High"
    justification: "images/images.yaml is the digest SoT; drift gate runs in ci.yml lint-go AND pr-checks AND Makefile; all CI containers pinned by sha256 digest."
    evidence:
      - "cmd→`cd cmd/zynax-ci && GOWORK=off go run . images check`→'✅ All consumer files are aligned with images/images.yaml.' (E1, live pass)"
      - ".github/workflows/ci.yml:456-460 (lint-go: zynax-ci images check); .github/workflows/pr-checks.yml:321-337 (image-digest-alignment job)"
      - "images/images.yaml:1-13 ('Single source of truth for all pinned container image digests'); ci.yml:49,60 et al pin ci-runner @sha256:ede25504…"
  - dimension: "Release engineering (semver, signing, SBOM, provenance, PyPI OIDC)"
    score: 8
    confidence: "High"
    justification: "Semver tag-driven; cosign keyless sign at merge + version; SLSA L2 attestation + SPDX SBOM on version images; CLI cross-compiled 5 platforms; SDK→PyPI via OIDC Trusted Publisher (no stored keys). Gap: service container images are amd64-only; signatures not verifiable locally (no registry access)."
    evidence:
      - ".github/workflows/release.yml:46-47 (semver tag triggers v[0-9]+.[0-9]+.[0-9]+); :201,504 (cosign sign); :510-516 (attest-build-provenance SLSA L2); :518-533 (syft SPDX SBOM)"
      - ".github/workflows/sdk-publish.yml:31,107-111 (environment: pypi, pypa/gh-action-pypi-publish OIDC, no API key)"
      - ".github/workflows/release.yml:308-315 (CLI 5-platform matrix linux/darwin/windows × amd64/arm64)"
  - dimension: "Automation hygiene (digest bot, skip-ci loop-safety, proto regen)"
    score: 8
    confidence: "High"
    justification: "Post-merge digest bot commits images.yaml with [skip ci]; loop-safety reasoned in ADR-027 §3C (no CI run→no workflow_run→no retag→no loop), confirmed by 15+ live bot commits; proto stubs verified pre-merge + auto-regen PR post-merge."
    evidence:
      - "cmd→`git log --grep='skip ci'`→15 consecutive 'chore(images): sync digests after main-<sha> [skip ci]' commits incl. HEAD~1 5a26f51 (E1)"
      - ".github/workflows/release.yml:217-245 (digest bot commit '[skip ci]', BOT_GITHUB_TOKEN ruleset-bypass push); :18-23 (loop-safety rationale)"
      - ".github/workflows/proto-generate.yml:1-13,65-92 (post-merge regen opens PR; pre-merge gate in ci.yml:337-352 incl. 2-pass determinism check)"
  - dimension: "Build speed & caching, Docker-only model, GOWORK=off enforcement"
    score: 8
    confidence: "High"
    justification: "Change-detection skips lanes; gha layer cache shared PR↔main; PR build reuses cache; everything in pinned ci-runner container; GOWORK=off used 23× in Makefile and pervasively in every Go CI step (ADR-017)."
    evidence:
      - ".github/workflows/ci.yml:107-245 (changes job: per-service/path change detection gates every lane); :852,867-868 (cache-from/cache-to type=gha scope per service, shared with release)"
      - "cmd→`grep -c GOWORK=off Makefile`→23; ci.yml:336,453,459,521 etc. GOWORK=off on every go run"
      - ".github/workflows/ci.yml:594-598 (integration job documents Docker-only/testcontainers model; runs on bare host for DinD)"
  - dimension: "CI-as-tested-code (ADR-036, cmd/zynax-ci)"
    score: 8
    confidence: "High"
    justification: "CI logic consolidated into a Go CLI (zynax-ci) with 22 test files and an enforced ≥80% coverage gate; replaces brittle inline shell + python scripts (images digest, canvas/schema/milestone validation, deps/expert-mapping checks)."
    evidence:
      - "cmd→`find cmd/zynax-ci -name '*_test.go' | wc -l`→22"
      - "tools/coverage-gates.env:7 (COVERAGE_CLI_ZYNAX_CI_GATE=80); _test-go.yml:327,339 enforces it"
      - ".github/workflows/ci.yml:451-460,521; pr-checks.yml:274-279,363 (zynax-ci validate canvas/schema/milestone, images check, check deps, check expert-mapping)"
  - dimension: "Pipeline observability (failures actionable, flaky mgmt, weekly audit)"
    score: 7
    confidence: "Medium"
    justification: "Failures emit ::error:: + remediation hints; Trivy SARIF→Security tab; SBOM artifacts; weekly audit fails loudly (no [AUTO] issue spam). Gaps: no explicit flaky-test retry/quarantine harness; e2e argo leg advisory-only; no pipeline-DORA telemetry surfaced."
    evidence:
      - ".github/workflows/ci.yml:889-913 (Trivy SARIF→Security tab + 30-day SBOM artifact on blocked PRs)"
      - ".github/workflows/weekly-audit.yml:1-8 (fails loudly Monday instead of auto-filing skeleton issues); :207-232 (fan-in exits non-zero)"
      - ".github/workflows/e2e-smoke-skip.yml:19-21 (argo leg intentionally advisory until green history); UNKNOWN: no flaky-retry mechanism found"
drift_test:
  - claim: "21 workflows in .github/workflows/"
    result: "VERIFIED"
    evidence:
      - "cmd→`ls .github/workflows/ | wc -l`→21 (ai-context-budget, ci, cli-release, e2e-smoke-skip, e2e-smoke, helm-lint, kb-preview, pr-checks, pr-image-cleanup, proto-generate, proto-stubs-publish, pr-size, release, sdk-publish, service-release, _test-go, _test-python, tools-image, tools-publish, weekly-audit, zynax-ci-release)"
      - "Part 1 §1.11 (framework:296) claims '21 files' — matches exactly. (Note: prompt §5.9 example file list is illustrative, not literal — e.g. no 'tools-*.yml' pair; actual reusable workflows are _test-go.yml/_test-python.yml.)"
  - claim: "build-once-promote-by-retag is real (ADR-027) — production images never rebuilt after merge"
    result: "VERIFIED"
    evidence:
      - ".github/workflows/release.yml:160-204 (retag via imagetools create — zero build/build-push steps in the entire file)"
      - ".github/workflows/ci.yml:789-868 (single pre-merge build to staging lane); ADR-027:46-47,88-90"
      - "cmd→`git log --grep='skip ci'`→live digest-sync commits prove the merge→retag→commit loop runs in production (E1)"
  - claim: "The 12 listed required gates actually BLOCK merge"
    result: "VERIFIED"
    evidence:
      - "cmd→`gh api .../rulesets/17547241`→required_status_checks lists all 12 contexts; strict_required_status_checks_policy:true (up-to-date branch required); required_signatures + required_linear_history + squash-only enforced"
      - "NOTE: required_approving_review_count:0 → NO human approval required to merge (automation can self-merge on green) — recorded, not a gate failure but a governance nuance (cross-ref 5.x governance)"
  - claim: "Multi-arch (arm64) builds for shipped artifacts"
    result: "PARTIAL"
    evidence:
      - "VERIFIED arm64 for tools/ci-runner (.github/workflows/tools-image.yml:111-154 native arm64) and CLI/zynax-ci binaries (release.yml:308-376)"
      - "CONTRADICTED for production SERVICE container images: ci.yml:861 builds 'platforms: linux/amd64' ONLY; release.yml is pure retag (adds no platform). Deployed service/adapter images are amd64-only. service-release.yml:80 builds amd64+arm64 but is workflow_dispatch-only (manual, not the live release path)."
red_flags:
  - severity: "Medium"
    finding: "Production service & adapter container images are amd64-only. The pre-merge build-images gate (the single build per ADR-027) builds linux/amd64 exclusively, and release.yml only retags — so no arm64 manifest is ever produced for deployed images. Helm/K8s on arm64 nodes (and Apple-Silicon local Docker) cannot run the official service images. The 'multi-arch' supply-chain story holds for tools/CLI but not for the services themselves."
    evidence:
      - ".github/workflows/ci.yml:861 (platforms: linux/amd64)"
      - ".github/workflows/release.yml:160-204 (retag-only, no platform expansion)"
      - ".github/workflows/service-release.yml:80 (multi-arch exists but workflow_dispatch-only — not on the tag-release path)"
  - severity: "Low"
    finding: "Doc-vs-config drift: e2e-smoke.yml header asserts it is 'GATED, NOT a required PR gate… deliberately excluded from the branch-protection required-check set', but the live ruleset lists 'e2e smoke (temporal)' as a REQUIRED status check (correctly satisfied by the e2e-smoke-skip.yml shim). The stale comment, if trusted, would mislead an engineer about merge-blocking behavior — a Part 1 §1.10-class drift, low impact because the shim is correct."
    evidence:
      - ".github/workflows/e2e-smoke.yml:17-19 (comment: 'NOT a required PR gate… excluded')"
      - "cmd→ruleset required_status_checks includes 'e2e smoke (temporal)'"
      - ".github/workflows/e2e-smoke-skip.yml:1-21 (shim correctly satisfies the required check on docs-only PRs)"
  - severity: "Low"
    finding: "Self-contradicting advisory flag: ai-context-budget.yml comment says 'Advisory only — never fail CI' but the step sets continue-on-error:false, so a non-zero zynax-ci exit would fail the run. Cosmetic but a latent surprise."
    evidence: [".github/workflows/ai-context-budget.yml:44-45 (comment vs continue-on-error:false)"]
  - severity: "Low"
    finding: "Bus-factor / token concentration: retag+digest-push to main relies on BOT_GITHUB_TOKEN (admin-owned fine-grained PAT) as the only ruleset-bypass actor; PR creation in proto-generate.yml may be blocked by org policy (falls back to a printed manual link). Single-maintainer automation custody (echoes Part 1 §1.9 social gating gap)."
    evidence: [".github/workflows/release.yml:89-95,217-245", ".github/workflows/proto-generate.yml:84-92"]
green_flags:
  - strength: "Build-once / promote-by-retag (ADR-027): the artifact that is Trivy-scanned pre-merge is the EXACT bit-for-bit binary promoted to production — scan==deploy, no post-merge rebuild nondeterminism. Best-in-class supply-chain property, formalized as a one-way-door ADR."
    evidence: ["release.yml:160-204", "ci.yml:789-913", "docs/adr/ADR-027-shift-left-pipeline.md:88-90"]
  - strength: "Digest SoT enforced three ways with live green: images/images.yaml + zynax-ci images check in lint-go, in pr-checks (dedicated job), and Makefile check-images. Ran live: all consumers aligned."
    evidence: ["cmd→`zynax-ci images check`→'✅ All consumer files are aligned' (E1)", "images/images.yaml:1-13", "ci.yml:456-460"]
  - strength: "SKIPPED-required-check bypass footgun explicitly engineered against: test-unit and test-integration use if: always() and self-checks so a required gate can never silently pass as neutral/no-op (#986, #553); e2e required check has a paths-ignore shim so docs PRs aren't blocked forever."
    evidence: ["ci.yml:566-591", "ci.yml:602-651", "e2e-smoke-skip.yml:1-44"]
  - strength: "CI-as-tested-code (ADR-036): brittle inline shell/python replaced by a Go CLI (zynax-ci) with 22 test files and an enforced ≥80% coverage gate, used for digest sync, canvas/schema/milestone validation, dep & expert-mapping drift."
    evidence: ["cmd→`find cmd/zynax-ci -name '*_test.go'|wc -l`→22", "tools/coverage-gates.env:7", "pr-checks.yml:274-279"]
  - strength: "Shift-left security baked into the merge gate: pre-merge Hadolint→Trivy(CRITICAL,HIGH,exit1)→SARIF→CycloneDX SBOM; cosign keyless sign at merge; SLSA L2 provenance + SPDX SBOM at version; SDK→PyPI via OIDC Trusted Publisher (zero stored keys). Trivy DB pinned by digest."
    evidence: ["ci.yml:836-913", "release.yml:201,510-533", "sdk-publish.yml:107-111", "ci.yml:876 (TRIVY_DB pinned)"]
  - strength: "Enforced coverage gates as SoT: domain ≥90%, adapter ≥85%, CLI ≥79/80%, Python ≥90% — single file, read by both enforcement and PR report."
    evidence: ["tools/coverage-gates.env:4-8", "_test-go.yml:155,265,307,327", "_test-python.yml:65"]
open_questions:
  - "Are arm64 service images on any roadmap, or is amd64-only an accepted constraint for K8s deployment targets? (Helm charts ship but services are amd64-only.)"
  - "required_approving_review_count:0 means automation can self-merge on green CI — is that intentional for a CNCF-aspiring project, given single-maintainer bus factor (Part 1 §1.9)?"
  - "No flaky-test quarantine/retry harness found — how are intermittent failures (e.g. the noted event-bus DLQ-timeout, memory-service cascade) managed beyond exclusion from the integration allowlist?"
  - "Can this ship multiple times/day safely? Retag is ~10s and idempotent/restartable (release.yml:146-159), so yes mechanically — but every merge auto-promotes to :latest with no human approval gate."
unknowns:
  - "cosign signatures and SLSA attestations on live GHCR images NOT independently verified — cosign is not installed locally and the sandbox has no registry pull access (cmd→`cosign version`→command not found). Signing is VERIFIED at the config level (E3: release.yml:201,504; sdk-publish.yml) but the existence of signatures in GHCR is UNKNOWN (Part 1 §1.10 C4 'cosign-signed images — verify signatures exist in GHCR' remains config-VERIFIED / artifact-UNKNOWN)."
  - "Actual CI wall-clock / DORA lead-time numbers — no pipeline telemetry surfaced in-repo; build-speed score rests on caching+change-detection design (E3), not measured timings."
  - "Whether the post-merge retag job has ever silently failed to promote (the loop is no-op-on-missing-staging by design) — live run-history not inspected."
cross_references:
  - to_agent: "5.8 Security / Supply-chain"
    note: "Cosign/SBOM/SLSA/Trivy/mTLS supply-chain claims (Part 1 C2-C4) — I confirm the CI mechanics (E3) but defer GHCR artifact existence (cosign verify) to security agent with registry access."
    evidence: ["release.yml:510-533", "ci.yml:870-913"]
  - to_agent: "5.x Governance / Maintainership"
    note: "required_approving_review_count:0 + single BOT_GITHUB_TOKEN bypass actor → automation self-merge + bus-factor concentration; reinforces §1.9 social gating gap."
    evidence: ["ruleset 17547241 pull_request rule", "release.yml:89-95"]
  - to_agent: "5.7 / Infra-SRE (Helm/K8s)"
    note: "Helm charts deploy services whose images are amd64-only — arm64 K8s nodes cannot run official service images. Confirm cluster target arch assumptions."
    evidence: ["ci.yml:861", "release.yml:160-204"]
recommendations:
  - priority: "P1"
    action: "Make production service/adapter images multi-arch (add linux/arm64 to ci.yml build-images platforms, or a buildx matrix), OR explicitly document amd64-only as a supported-platform constraint in the Helm/quickstart docs."
    rationale: "Closes the largest delivery red flag; aligns 'multi-arch' narrative with shipped artifacts; unblocks arm64 K8s and Apple-Silicon evaluators (a CNCF/adoption friction point)."
  - priority: "P2"
    action: "Reconcile stale doc-vs-config drift: update e2e-smoke.yml header ('NOT a required gate') to match the ruleset, and fix ai-context-budget.yml comment/continue-on-error mismatch."
    rationale: "Part 1 §1.10-class drift is the project's known failure mode; cheap to fix and removes misleading merge-behavior signals."
  - priority: "P2"
    action: "Introduce a flaky-test quarantine/retry signal and surface basic pipeline DORA metrics (lead time, change-fail rate) so 'ship multiple times/day' is measured, not asserted."
    rationale: "Observability sub-dimension is the weakest; measured pipeline health strengthens the production-credibility case."
  - priority: "P2"
    action: "Consider requiring ≥1 review (or a second bypass actor) given CNCF-aspiration and single-maintainer bus factor, rather than required_approving_review_count:0."
    rationale: "Auto-self-merge on green is efficient for a solo maintainer but a governance liability at the M8 CNCF gate."
```

---

## (b) §6.2 Prose section

## Agent 5.9 DevOps / Release Engineering — Score: 8 (High)

**Mission recap:** Assess CI/CD, release engineering, automation, reproducibility, build speed, quality gates, tooling, versioning, and operational observability of the delivery pipeline itself.

**Verdict:** This is a genuinely strong, supply-chain-grade pipeline that is well above market norm for a project of this size. The standout is **build-once / promote-by-retag (ADR-027)**: production service images are built exactly once in pre-merge CI, scan-gated by Trivy, and then promoted to production purely by manifest retag — `release.yml` contains no image-build step, so the deployed binary is provably the scanned binary. Twelve real merge-blocking gates are enforced through a modern repository ruleset (not the legacy protection API, which is why a naive `branches/main/protection` call 404s), the SKIPPED-required-check bypass footgun is explicitly engineered against, and CI logic has been migrated into a unit-tested Go CLI (`zynax-ci`, 22 test files, ≥80% gate). The two material weaknesses are that the deployed **service images are amd64-only** despite a "multi-arch" narrative, and that supply-chain artifacts (cosign signatures, SLSA attestations) are config-verified but not independently confirmed in GHCR from this sandbox.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Blocking gates (required vs advisory) | 9 | High | ruleset 17547241 (12 required contexts, strict policy); ci.yml:573-591,602-651 |
| Reproducibility (build-once/retag, ADR-027) | 9 | High | ci.yml:789-913; release.yml:160-204; ADR-027:32-90 |
| Digest pinning (ADR-024, check-images) | 9 | High | `zynax-ci images check`→✅ (E1); images.yaml:1-13; ci.yml:456-460 |
| Release engineering (semver/sign/SBOM/PyPI OIDC) | 8 | High | release.yml:46-47,201,510-533; sdk-publish.yml:107-111 |
| Automation hygiene (digest bot, skip-ci, proto regen) | 8 | High | `git log --grep='skip ci'`→15 commits; release.yml:217-245; proto-generate.yml |
| Build speed/caching, Docker-only, GOWORK=off | 8 | High | ci.yml:107-245,852-868; `grep -c GOWORK=off Makefile`→23 |
| CI-as-tested-code (ADR-036) | 8 | High | 22 zynax-ci test files; coverage-gates.env:7 |
| Pipeline observability | 7 | Medium | ci.yml:889-913; weekly-audit.yml:1-8,207-232 |

**Drift test:**
- "21 workflows" → **VERIFIED** (`ls .github/workflows | wc -l`→21; matches §1.11). Prompt §5.9's example filename list is illustrative — actual reusable workflows are `_test-go.yml`/`_test-python.yml`, no literal `tools-*.yml` pair.
- "build-once-promote-by-retag" → **VERIFIED** (release.yml is retag-only; ADR-027:88-90; live `[skip ci]` digest-sync commit history proves the loop runs).
- "12 listed gates actually block" → **VERIFIED** (ruleset required_status_checks, strict policy on). Caveat: `required_approving_review_count:0` — no human approval needed.
- "Multi-arch builds" → **PARTIAL** (arm64 for tools/ci-runner + CLI binaries; **amd64-only for deployed service/adapter images** — ci.yml:861; release.yml retag-only).

**Red flags (severity-ordered):**
1. **[Medium] Production service/adapter images are amd64-only.** The single ADR-027 build (ci.yml:861) is `linux/amd64`; retag adds no arch. arm64 K8s/Apple-Silicon cannot run official service images. `service-release.yml:80` has multi-arch but is manual/dispatch-only.
2. **[Low] Doc-vs-config drift:** `e2e-smoke.yml:17-19` says it is "NOT a required PR gate," but the ruleset requires `e2e smoke (temporal)` (correctly shimmed by `e2e-smoke-skip.yml`). A §1.10-class drift.
3. **[Low] Self-contradicting flag:** `ai-context-budget.yml:44-45` "Advisory only — never fail CI" vs `continue-on-error:false`.
4. **[Low] Token/bus-factor concentration:** retag+digest push depend on a single admin-owned `BOT_GITHUB_TOKEN` as the sole ruleset-bypass actor.

**Green flags:**
- Build-once/promote-by-retag — scan==deploy, no post-merge rebuild nondeterminism (release.yml:160-204; ADR-027).
- Digest SoT enforced three ways, live-green (`zynax-ci images check`→✅).
- SKIPPED-required-check bypass explicitly defeated (ci.yml:566-591,602-651; e2e shim).
- CI-as-tested-code: Go CLI, 22 tests, ≥80% gate (ADR-036).
- Shift-left security in the merge gate: Hadolint→Trivy(exit1)→SARIF→CycloneDX SBOM; cosign sign; SLSA L2 + SPDX at version; SDK→PyPI OIDC trusted publisher, zero stored keys.
- Enforced coverage gates as SoT (domain ≥90%, python ≥90%, adapter ≥85%, CLI ≥79/80%).

**Open questions / unknowns:** arm64 service-image roadmap; intent of `required_approving_review_count:0` for a CNCF-aspiring repo; absence of a flaky-test quarantine harness; **cosign/SLSA artifacts in GHCR not independently verified** (cosign absent locally, no registry pull access — config-VERIFIED only, leaving Part 1 C4 artifact-UNKNOWN); no measured CI wall-clock/DORA numbers in-repo.

**Recommendations:** P1 — make service images multi-arch or document amd64-only as a supported constraint. P2 — reconcile the e2e-smoke / ai-context-budget doc-vs-config drifts; add flaky-test handling + DORA telemetry; reconsider zero-required-reviews given the M8 CNCF gate and single-maintainer bus factor.

**Cross-references:** 5.8 Security (cosign/SBOM/SLSA artifact existence — I confirm CI mechanics, defer GHCR verification); 5.x Governance (zero required reviews + single bypass-token → self-merge + bus factor); 5.7/Infra-SRE (amd64-only images vs Helm K8s targets).
<!-- Agent 5.10 — Documentation Agent · Wave A (ground-truth) · GitHub issue #1402 -->
<!-- Target repo: the repository root @ HEAD e3135a6 (e3135a60e4abb20886d51f81d6448b22fe04cb64) -->
<!-- Framework: docs/due-diligence/2026-06-18-zynax-due-diligence-framework.md -->

# Agent 5.10 — Documentation — §3.4 Handoff Packet

```yaml
agent: "5.10 Documentation"
wave: "A"
dimension_groups: ["D11", "D1", "D7"]
overall_score: 7
overall_confidence: "High"
sub_scores:
  - dimension: "Coverage map (install/quickstart/tutorials/examples/subsystems)"
    score: 8
    confidence: "High"
    justification: "Broad, layered coverage: install, quickstart, dev-guide, authoring, observability, 37 ADRs, per-service AGENTS.md, runnable examples; few dark subsystems."
    evidence:
      - "docs/quickstart.md (8-step clone→traced-run path)"
      - "docs/developer-guide.md:1-60 (daily workflow, make targets)"
      - "docs/authoring/workflows.md:1-30 (workflow authoring, points at real schema/template/examples)"
      - "ls docs/adr/ADR-*.md | wc -l → 37 ADRs"
      - "spec/workflows/examples/ → 10 example manifests incl. code-review-ollama.yaml, e2e-demo.yaml"
      - "docs/observability/ (opentelemetry.md, uptrace.md, sampling.md, troubleshooting.md, naming-conventions.md)"
  - dimension: "Accuracy of doc claims vs code/CI (core test, 8 samples)"
    score: 8
    confidence: "High"
    justification: "Sampled 8 concrete claims; 7 VERIFIED against source/config, 1 (make demo e2e) PARTIAL (wiring verified, runtime not executable here)."
    evidence:
      - "CLI default URL 8080: cmd/zynax/cmd/root.go:40 ✓ matches quickstart export of mapped 7080 (docker-compose.yml:282 maps 7080:8080)"
      - "All quickstart CLI subcommands exist: cmd/zynax/cmd/logs.go:25, result.go:19, init.go:35/44, events.go:20, validate.go:14"
      - "Default demo model qwen2.5-coder:3b: infra/docker-compose/ollama/llm-adapter.config.yaml:27 == Makefile:154 DEMO_MODEL (single source)"
      - "Temporal+Argo both implement WorkflowEngine: services/engine-adapter/internal/infrastructure/temporal.go + argo_engine.go"
      - "AgentService gRPC contract (no-SDK capability): protos/zynax/v1/agent.proto:34-46"
      - "Stateless workflow-compiler (C7): grep services/workflow-compiler/internal/ → no in-memory IR/compiled store, only legit graph maps"
  - dimension: "Cross-doc consistency (README/ROADMAP/CLAUDE/state)"
    score: 6
    confidence: "High"
    justification: "Top-line milestone status agrees across 4 surfaces; but README has internal staleness (quickstart 2 milestones behind its own service table) and CLAUDE.md omits the new M-UX milestone."
    evidence:
      - "AGREE: README.md:407, ROADMAP.md:160, state/current-milestone.md:15-21, CLAUDE.md:127-129 — M6 Complete v0.5.0 / M7 Active v0.6.0"
      - "STALE within README: lines 333-337 + 355 'capability dispatch pending M5.C' contradicts README Service Status table lines 446-464 (all ✅ Complete)"
      - "STALE: README.md:503 'Helm charts (planned, #241)' but Helm shipped M6 (state/current-milestone.md:240)"
      - "STALE: README.md:227 task-broker 'In-memory' vs README.md:451 'Postgres-backed pgx/v5'"
      - "GAP: ROADMAP.md:61 inserts M-UX (M7→M-UX→M-dx→M8); CLAUDE.md:129 still says M7→M-dx→M8"
  - dimension: "Maintenance burden / single-source-of-truth / drift risk"
    score: 6
    confidence: "Medium"
    justification: "Strong SoT mechanisms (images.yaml gate, DEMO_MODEL derived from config, state/current-milestone.md canonical) but high doc surface area (130+ md files, 6 dated review docs, ~30 canvases) creates recurring drift; README is the chronic laggard."
    evidence:
      - "SoT win: images/images.yaml + make check-images CI gate (README.md:168-179)"
      - "SoT win: Makefile:154 DEMO_MODEL := awk model: from llm-adapter.config.yaml (no duplication)"
      - "Burden: find docs -name '*.md' → 130+ files incl. docs/reviews/ (7 dated), docs/architecture/ (10 dated reviews), docs/product/ (7)"
      - "Drift: README.md:504 'ADR-001 – ADR-019' but 37 ADRs exist"
  - dimension: "Onboarding path from docs alone"
    score: 8
    confidence: "High"
    justification: "Clear linear path clone→bootstrap→install-cli→make demo / quickstart→authoring→examples; zero-secret Ollama path lowers the barrier; CLI command reference cross-links to source-of-record."
    evidence:
      - "README.md:24-33 'See it run — one command' (make demo, Docker+ollama pull only)"
      - "docs/quickstart.md:17-20 every command maps to a real subcommand, source-of-record cmd/zynax/cmd/"
      - "docs/developer-guide.md:3-4 routes newcomers to quickstart first"
      - "docs/authoring/workflows.md:9-12 reference material section links schema+template+examples"
  - dimension: "Drift-test honesty / Truth-Pass culture intact"
    score: 8
    confidence: "High"
    justification: "The §1.10 'known lag' (CLAUDE.md/spdd-guide cite pre-PR#1400 command names) is RESOLVED at this HEAD — docs already match the live 5-verb tree; honest stubs are labelled."
    evidence:
      - "CLAUDE.md:90-110 cites /plan, /deliver, /lib:spdd-* — matches .claude/commands/ (deliver.md, plan.md, review.md, reconcile.md, learn.md, milestone.md, lib/, experts/)"
      - "docs/patterns/spdd-guide.md:7-9 + 32-45 cite /plan, /deliver, /lib:spdd-* (new names)"
      - "Honest stub: README.md:37 asciinema PLACEHOLDER flagged openly in docs/casts/README.md:35"
drift_test:
  - claim: "Write your agent workflow once — run it on Temporal or Argo without a rewrite (engine portability)"
    result: "VERIFIED"
    evidence:
      - "services/engine-adapter/internal/infrastructure/temporal.go + argo_engine.go both implement the WorkflowEngine interface (handler.go)"
      - ".github/workflows/e2e-smoke.yml:61 engine: [temporal, argo] 2-leg CI matrix; fail-fast:false"
      - "ROADMAP.md:181 + state/current-milestone.md:116 ArgoEngine #766 delivered M6"
  - claim: "No SDK required — any system becomes a capability by implementing the AgentService gRPC contract"
    result: "VERIFIED"
    evidence:
      - "protos/zynax/v1/agent.proto:34 service AgentService; :39 ExecuteCapability stream; :46 GetCapabilitySchema"
      - "ADR-013 adapter-first-no-sdk; 5 adapters (http/git/ci/llm Go, langgraph Py) README.md:459-464"
  - claim: "make demo — one command boots a zero-secret local-LLM stack and runs the hero code-review workflow end-to-end with a real model"
    result: "PARTIAL"
    evidence:
      - "Wiring VERIFIED: Makefile:162-205 demo target boots COMPOSE_DEMO (Ollama overlay), zynax apply DEMO_TARGET, prints zynax result"
      - "spec/workflows/examples/code-review-ollama.yaml exists; llm-adapter.config.yaml:27 registers codereview capability on local model"
      - "Runtime NOT executed here (no Docker/Ollama in audit env) → end-to-end success is E2/E3 verified, not E1; asciinema proof is still a PLACEHOLDER (README.md:37)"
red_flags:
  - severity: "Medium"
    finding: "README internally self-contradicts and lags its own service table by ~2 milestones: the Quickstart 'M5 status note' (README.md:333-337,355) tells users capability dispatch is 'pending M5.C' and to register an adapter first, while the same file's Service Status table (446-464) and ROADMAP report dispatch shipped in v0.4.0 (M5.C) and end-to-end demo green. A new evaluator reading top-down hits the stale note first and may conclude the product is less complete than it is — the exact narrative-vs-delivery drift class §1.10 warns about, here in the *opposite* (under-claiming) direction."
    evidence:
      - "README.md:333-337 'M5 status note' + :355 'capability dispatch pending M5.C'"
      - "README.md:446-464 Service Status table all ✅ Complete (Postgres-backed)"
      - "README.md:227 task-broker 'In-memory' vs :451 'Postgres-backed pgx/v5'"
      - "README.md:503 'Helm charts (planned, #241)' vs M6 shipped (state/current-milestone.md:240)"
      - "README.md:504 'ADR-001 – ADR-019' vs 37 ADRs on disk"
  - severity: "Low"
    finding: "Hero 'See it run' asciinema cast in README is a non-functional PLACEHOLDER; the visual proof a first-time visitor expects is dead (text path works, and the stub is honestly labelled, so impact is cosmetic-trust not functional)."
    evidence:
      - "README.md:37 asciinema.org/a/PLACEHOLDER.svg"
      - "docs/casts/README.md:35 documents the placeholder as a maintainer follow-up"
  - severity: "Low"
    finding: "CLAUDE.md milestone-program map (M7→M-dx→M8) has not absorbed the 2026-06-18 M-UX insertion that ROADMAP.md already carries (M7→M-UX→M-dx→M8) — a fresh, small status-surface divergence."
    evidence:
      - "CLAUDE.md:129 defers to 'M-dx developer-experience program'"
      - "ROADMAP.md:57-61 + :226 M-UX milestone (#10) between M7 and M-dx"
green_flags:
  - strength: "The §1.10 'known doc-vs-tooling lag' (CLAUDE.md/spdd-guide still citing pre-PR#1400 command names) is already RESOLVED at this HEAD: both docs cite the live 5-verb surface, matching the .claude/commands/ tree exactly. Evidence the Truth-Pass reconciliation habit is operating, not just claimed."
    evidence:
      - "CLAUDE.md:86-110 + docs/patterns/spdd-guide.md:7-45 cite /plan //deliver //lib:spdd-*"
      - ".claude/commands/ = deliver.md plan.md review.md reconcile.md learn.md milestone.md lib/ experts/"
  - strength: "High doc accuracy on the sampled claims (7/8 VERIFIED): CLI defaults, every quickstart subcommand, ports, default model, statelessness, engine portability all check out against source/config — docs describe HEAD, not aspiration."
    evidence:
      - "cmd/zynax/cmd/root.go:40; docker-compose.yml:282; cmd/zynax/cmd/{logs,result,init,events,validate}.go"
      - "infra/docker-compose/ollama/llm-adapter.config.yaml:27 == Makefile:154"
  - strength: "Genuine single-source-of-truth mechanisms with CI enforcement reduce structural drift: images.yaml (+ make check-images gate), DEMO_MODEL derived from config via awk, canonical state/current-milestone.md."
    evidence:
      - "README.md:168-179 images.yaml SoT + make check-images CI gate"
      - "Makefile:154 DEMO_MODEL awk from llm-adapter.config.yaml"
  - strength: "Clear, low-friction onboarding path with a zero-secret local-LLM default; CLI command reference explicitly cites cmd/zynax/cmd/ as source-of-record and tells the reader how each command was verified."
    evidence:
      - "docs/quickstart.md:17-20, :237-254 command reference table"
      - "README.md:24-33 make demo one-command path"
open_questions:
  - "Does `make demo` actually reach WORKFLOW_STATUS_COMPLETED with a real model on a clean host? Wiring is verified but no E1 runtime proof was obtainable in the audit env (no Docker/Ollama). Cross-check with Agent owning runtime/e2e."
  - "Is the README the only chronically-lagging surface, or do other entry docs (faq.md, local-dev.md) carry similar stale milestone notes? Sampled README + quickstart + dev-guide + authoring only."
unknowns:
  - "Runtime end-to-end success of make demo / quickstart — UNKNOWN (E1 not executable in read-only audit env); verified to E2/E3 wiring only."
  - "Whether published GitHub Release CLI binaries match the documented curl install URLs — UNKNOWN (no network/GHCR access exercised)."
cross_references:
  - to_agent: "5.x (Runtime / E2E)"
    note: "make demo / quickstart end-to-end is wiring-verified but not runtime-proven here; needs an E1 run to close the PARTIAL drift-test result."
    evidence: ["Makefile:162-205", "spec/workflows/examples/code-review-ollama.yaml"]
  - to_agent: "5.x (Security)"
    note: "Doc agent VERIFIED §1.10 C7 (stateless compiler) from code; C2 mTLS / C3 SBOM / C4 cosign / C5 CloudEvents are doc-claimed fixed in M6 but require security/CI agent's E1 verification — out of doc scope."
    evidence: ["state/current-milestone.md:224-238 (mTLS/supply-chain COMPLETE claims)", "README.md:450 (CloudEvents via event-bus #827)"]
  - to_agent: "5.x (Architecture)"
    note: "Three-layer separation and ADR set are well-documented and internally consistent; ADR count claim in README (ADR-001–ADR-019) is the only stale architecture-doc reference found."
    evidence: ["README.md:504", "37 ADR files on disk"]
recommendations:
  - priority: "P1"
    action: "Truth-pass the README: delete/replace the stale 'M5 status note' (333-337,355), fix task-broker 'In-memory'→Postgres (227), 'Helm planned'→shipped (503), 'ADR-001–ADR-019'→ADR-001–ADR-037 (504)."
    rationale: "These under-claim shipped capability and self-contradict the same file's service table — the highest-leverage doc fix for evaluator trust."
  - priority: "P2"
    action: "Record and embed the make-demo asciinema cast to replace the README PLACEHOLDER (docs/casts/README.md already has the procedure)."
    rationale: "Restores the hero 'see it run' proof a first-time visitor expects."
  - priority: "P2"
    action: "Sync CLAUDE.md's milestone-program line to include M-UX (M7→M-UX→M-dx→M8) to match ROADMAP.md."
    rationale: "Keeps the four status surfaces fully aligned; cheap, prevents a fresh drift seed."
```

---

## Agent 5.10 — Documentation — Score: 7 (High)

**Mission recap:** Assess documentation coverage, accuracy vs. code, cross-doc consistency, maintenance burden, and onboarding path; run the drift test on README's boldest capability claims; verify whether the §1.10 doc-vs-tooling lag persists at HEAD.

**Verdict:** Zynax's documentation is broad, layered, and — on the sampled claims — accurate to HEAD: 7 of 8 sampled claims VERIFIED against source/config, the onboarding path is clean (clone → `make demo` → quickstart → authoring → examples), and genuine SoT mechanisms (images.yaml CI gate, config-derived demo model) curb structural drift. The Truth-Pass culture is demonstrably working: the framework's flagged "known lag" (CLAUDE.md/spdd-guide citing pre-PR#1400 command names) is already reconciled at this HEAD. The one persistent weakness is the **README itself**, which self-contradicts and under-claims shipped capability — its Quickstart still carries an "M5 status note" telling users capability dispatch is pending, two milestones behind its own service table, plus three smaller stale lines. This is the §1.10 drift class, here pointing the safe direction (under- not over-claiming), but it still misleads a top-down reader. Net: strong, honest, slightly aging at the front door.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Coverage map | 8 | High | quickstart.md, developer-guide.md, authoring/workflows.md, 37 ADRs, observability/, 10 example manifests |
| Accuracy vs code/CI (8 samples) | 8 | High | root.go:40; docker-compose.yml:282; cmd/zynax/cmd/*; llm-adapter.config.yaml:27==Makefile:154; agent.proto:34; temporal.go+argo_engine.go |
| Cross-doc consistency | 6 | High | README.md:407 / ROADMAP.md:160 / state/current-milestone.md:15-21 / CLAUDE.md:127-129 agree top-line; README 333-337,355,227,503,504 stale; CLAUDE.md:129 omits M-UX |
| Maintenance / SoT / drift | 6 | Medium | images.yaml gate (README:168-179); DEMO_MODEL awk (Makefile:154); 130+ md files; README:504 ADR count stale |
| Onboarding from docs alone | 8 | High | README:24-33; quickstart.md:17-20; developer-guide.md:3-4; authoring/workflows.md:9-12 |
| Truth-Pass honesty / drift-lag | 8 | High | CLAUDE.md:90-110 + spdd-guide.md:7-45 match .claude/commands/ 5-verb tree; PLACEHOLDER honestly labelled |

**Drift test (3 boldest README claims):**
- "Run on Temporal **or** Argo without a rewrite" → **VERIFIED** — temporal.go + argo_engine.go both implement WorkflowEngine; e2e-smoke.yml:61 runs a `[temporal, argo]` CI matrix.
- "No SDK required — any gRPC service is a capability via AgentService" → **VERIFIED** — agent.proto:34-46 (`ExecuteCapability`/`GetCapabilitySchema`); ADR-013; 5 adapters shipped.
- "`make demo` runs the hero workflow end-to-end with a real model" → **PARTIAL** — target fully wired (Makefile:162-205, code-review-ollama.yaml, llm-adapter.config.yaml:27) but runtime not executable in this read-only env (E2/E3, not E1); asciinema proof still a PLACEHOLDER.

**Red flags (severity-ordered):**
1. **Medium** — README under-claims & self-contradicts: stale "M5 status note" (README.md:333-337,355) vs its own Service Status table (446-464); task-broker "In-memory" (227) vs "Postgres-backed" (451); "Helm planned" (503); "ADR-001–ADR-019" vs 37 ADRs (504).
2. **Low** — Hero asciinema cast is a dead PLACEHOLDER (README.md:37; honestly flagged in docs/casts/README.md:35).
3. **Low** — CLAUDE.md:129 milestone program omits the M-UX insertion ROADMAP.md:57-61 carries.

**Green flags:**
- §1.10 known doc-vs-tooling lag already RESOLVED at HEAD (CLAUDE.md:86-110, spdd-guide.md:7-45 ↔ .claude/commands/).
- High sampled accuracy (7/8 VERIFIED) — docs describe HEAD, not aspiration.
- CI-enforced single-source-of-truth (images.yaml + check-images; DEMO_MODEL derived from config).
- Low-friction, zero-secret onboarding with source-of-record cross-links.

**Open questions / unknowns:** Does `make demo` actually reach COMPLETED on a clean host (E1 not runnable here)? Do other entry docs (faq.md, local-dev.md) carry similar stale notes? Do published Release binary URLs resolve (no network access)?

**Recommendations:** P1 — truth-pass the README (4 stale spots). P2 — record/embed the demo cast; sync CLAUDE.md to add M-UX.

**Cross-references:** Runtime/E2E agent (close the make-demo PARTIAL with an E1 run); Security agent (verify §1.10 C2–C5 M6 fixes — out of doc scope); Architecture agent (only stale architecture-doc ref is README's ADR count).
<!-- Agent 5.12 — AI Workflow · Wave A (ground-truth) · GitHub issue #1402 · READ-ONLY audit of the repository root at HEAD -->

# Agent 5.12 — AI Workflow / SPDD Pipeline

## (a) §3.4 Handoff packet

```yaml
agent: "5.12 AI Workflow"
wave: "A"
dimension_groups: ["D10", "D11", "D16"]
overall_score: 7
overall_confidence: "High"
sub_scores:
  - dimension: "SPDD enforcement — is canvas-before-code a GATE or a convention?"
    score: 6
    confidence: "High"
    justification: "A real CI gate exists, but its 'present' check passes if ANY canvas exists repo-wide, and Draft canvases only warn — so it is a soft schema gate, not a per-feature alignment gate."
    evidence:
      - ".github/workflows/pr-checks.yml:204 (canvas-freshness job, feat: only)"
      - ".github/workflows/pr-checks.yml:231-233 (passes when canvas_existing=find docs/spdd -name canvas.md | head -1 — ~30 canvases already exist, so this branch never fails)"
      - "cmd/zynax-ci/validate/canvas.go:124-129 (Draft → ValidationWarning, NOT ValidationError — a Draft canvas does not fail the gate)"
      - "docs/adr/ADR-019-spdd-prompt-governance.md:35-41,66-69 (Core rule 'Canvas before code'; Aligned required before generate)"
      - "Makefile:477-479 (validate-canvas → zynax-ci validate canvas)"
  - dimension: "Canvas validator correctness (executed proof)"
    score: 7
    confidence: "High"
    justification: "Validator runs, checks 7 REASONS sections + header fields + status enum + security marker + no committed private file; flags 2 real issues; but its status enum is already stale vs live lifecycle (Superseded unmodeled)."
    evidence:
      - "cmd: `GOWORK=off go run . validate canvas ../../docs/spdd/` (from cmd/zynax-ci) → most canvases OK, exit 1"
      - "output → FAIL docs/spdd/1359-zero-temporal-engine/canvas.md: invalid Status 'Superseded' — must be one of: Aligned, Draft, Implemented, Synced"
      - "output → FAIL docs/spdd/1370-awesome-quickstart/canvas.md: canvas.private.md found on disk (but .gitignore:53 = docs/spdd/**/canvas.private.md; git ls-files shows only canvas.md tracked → disk false-positive, leak control holds at git layer)"
      - "cmd/zynax-ci/validate/canvas.go:36-56 (validStatuses map + canvasSections)"
  - dimension: "Command surface quality — 5 verbs + lib/ + experts/ + README map"
    score: 8
    confidence: "High"
    justification: "Live tree exactly matches the claimed shape (5 verbs + milestone + README + 19 lib/ + 8 experts); verbs are well-engineered and encode hard-won operational rules; README is a faithful self-guided map."
    evidence:
      - "find .claude/commands -name '*.md' → plan/deliver/review/reconcile/learn + milestone.md + README.md + lib/ (19 files) + experts/ (8 personas)"
      - ".claude/commands/README.md:13-49 (five-verb decision tree, safe-by-default PLAN unless --execute)"
      - ".claude/commands/deliver.md:38-47 (encodes DCO Signed-off-by, Assisted-by not Co-Authored-By, squash-only because required_signatures blocks rebase, runtime-evidence-not-config, re-run stateful paths twice)"
      - "wc -l experts/*.md → 201-408 lines each (substantial, not stubs)"
  - dimension: "Cognitive-load reduction (5-verb consolidation vs prior 20+ sprawl)"
    score: 8
    confidence: "Medium"
    justification: "Consolidation to 5 milestone-agnostic verbs with lib/ building blocks is a genuine simplification; README explicitly frames it as 'fewer doors'. Medium because the prior 20+ sprawl is asserted, only partly observable in residual stale refs."
    evidence:
      - ".claude/commands/README.md:10-13,53 ('without memorizing 20 commands'; 'same as before — fewer doors')"
      - ".claude/commands/README.md:78-92 (lib/ grouped by SPDD/delivery/review/milestone; experts/ dispatched not invoked)"
      - "CLAUDE.md §SPDD lines 88-115 (two-verb daily driver: /plan + /deliver)"
  - dimension: "Context management / KB tiering (ADR-018) + leak controls"
    score: 8
    confidence: "High"
    justification: "Three-tier classification (ADR-018/ADR-019) backed by real, enforced controls: gitleaks AI-context config, ai-context-budget gate, CODEOWNERS on KB paths, gitignored private canvases."
    evidence:
      - "docs/adr/ADR-019-spdd-prompt-governance.md:72-81 (Tier 1/2/3 table; security-review must pass before commit)"
      - "docs/adr/ADR-018-ai-kb-authorization-model.md:42-76 (CODEOWNERS + branch protection + CI gating for KB paths)"
      - ".github/workflows/pr-checks.yml:385-388 (gitleaks detect --config tools/gitleaks-ai-context.toml)"
      - ".github/workflows/ai-context-budget.yml:7-13,37 (budget gate on CLAUDE.md + **/AGENTS.md via zynax-ci check ai-context)"
      - ".github/CODEOWNERS:15-20 (/CLAUDE.md, /AGENTS.md, **/AGENTS.md, /.claude/ → @zynax-io/maintainers)"
      - ".gitignore:53 (docs/spdd/**/canvas.private.md)"
  - dimension: "Hallucination/quality prevention (security-review, drift/sync, learnings loop)"
    score: 8
    confidence: "High"
    justification: "Multiple defense layers: spdd-security-review step, spdd-sync/prompt-update drift blocks, and a genuinely closed learnings loop with a Draft→applied/rejected lifecycle that rejects structural-workaround patterns and shows real LoC deltas."
    evidence:
      - "docs/ai-learnings/APPLY_LOG.md:15-99 (structured runs; e.g. ci-release.md +20L; rejects structural-workaround rows; 2 pending at 2026-06-18 → live loop)"
      - "wc -l docs/ai-learnings/*.md → ci-release.md 647, go-services.md 633 (accumulated real knowledge)"
      - "docs/patterns/spdd-guide.md:145-177 (security-review before every commit; sync/prompt-update for drift)"
      - ".github/PULL_REQUEST_TEMPLATE.md:78,188 (/spdd-security-review passed checkbox)"
      - ".claude/commands/learn.md (synthesizer writes PENDING proposals to APPLY_LOG; human-gated)"
  - dimension: "Self-hosting automation (automation/) — real or aspirational?"
    score: 6
    confidence: "High"
    justification: "Honestly delivered to a boundary: orchestrator+expert mesh authored as real Zynax manifests with schema tests passing, but the live `zynax apply` e2e is explicitly deferred to M7 (#1103) behind a cleanly-skipping platform gate — not yet running on itself."
    evidence:
      - "automation/README.md:16-43 (Wave 4 manifests delivered O1-O7+O9; live e2e deferred to M7 #1103; 'Do not wire workflows/*.yaml into main CI' until flip)"
      - "automation/tests/test_platform_readiness.py:48-57 (xfail marker GONE; zynax_client fixture skips cleanly unless ZYNAX_PLATFORM_E2E=1 — even more honest than the STATUS doc claims)"
      - "automation/workflows/ (issue-delivery.yaml, dev-advisory-orchestrator.yaml, learning-synthesizer.yaml, experts/*.yaml — real manifests, kind: counts confirm content)"
      - "docs/adr/ADR-028-...:7,15-23 (Accepted; AgentDef-vs-Workflow split)"
      - "docs/adr/ADR-033-expert-agent-substrate.md:3 (Status: Proposed — substrate not yet Accepted)"
  - dimension: "Defensible IP vs productivity multiplier vs process overhead"
    score: 7
    confidence: "Medium"
    justification: "The SPDD + REASONS Canvas + learnings-loop + KB-tiering system is a coherent, internally-consistent AI-native methodology that is a real productivity asset and partly novel; defensibility is process/discipline IP (copyable), and a single-maintainer bus factor limits durability."
    evidence:
      - "docs/adr/ADR-019-...:104-111 (rationale: 'the value of SPDD is in the gate, not the template')"
      - "37 'Implemented' canvases (grep Status across docs/spdd/*/canvas.md) → process actually ran at scale"
      - "ADR-028 deciders + ADR-028:9 single decider 'Oscar Gómez Manresa' (bus factor); cross-ref Part 1 §1.9"
drift_test:
  - claim: "Canvas-before-code is an ENFORCED gate (ADR-019), not a convention"
    result: "PARTIAL"
    evidence:
      - ".github/workflows/pr-checks.yml:204 (gate job exists, runs on feat: PRs) — real"
      - ".github/workflows/pr-checks.yml:231-233 (passes if any canvas exists anywhere → not per-feature)"
      - "cmd/zynax-ci/validate/canvas.go:124-129 (Draft canvas → warning only, not a hard fail) → Aligned-before-merge is NOT machine-enforced; relies on human review + branch protection"
  - claim: "Canvases map to shipped features (no canvas/feature drift)"
    result: "VERIFIED"
    evidence:
      - "grep Status across docs/spdd/*/canvas.md → 37 Implemented + 1 Synced + Aligned/Superseded"
      - "docs/spdd/214-temporal-execution/canvas.md:12 Status: Implemented ↔ services/engine-adapter/internal/infrastructure/temporal_workflow.go + internal/domain/interpreter.go exist (shipped)"
      - "docs/patterns/spdd-guide.md:254 (101-workflow-ir canvas cited as backed by real delivered code)"
  - claim: "CLAUDE.md §SPDD and docs/patterns/spdd-guide.md still lag the live .claude/commands surface (§1.10 note, post-PR#1400)"
    result: "CONTRADICTED (lag RESOLVED in those two files; lag persists elsewhere)"
    evidence:
      - "grep '/spdd-reasons|/spdd-generate|/m6-|/issue-deliver|/milestone-plan|/milestone-orchestrate' CLAUDE.md docs/patterns/spdd-guide.md → NO output (both now use /plan, /deliver, /lib:spdd-*)"
      - "CLAUDE.md:88-115 + docs/patterns/spdd-guide.md:7-9,49,266 (consolidated names live) → §1.10-flagged lag is FIXED at HEAD"
      - "RESIDUAL lag persists: pr-checks.yml:240 ('/spdd-reasons-canvas'); docs/ai-learnings/APPLY_LOG.md:3,8 ('/m6-learn'); .claude/commands/learn.md:193,352 + experts/go-services.md:287-288 (refs to retired files milestone-orchestrate.md / issue-deliver.md); ADR-019:87-94 (old un-namespaced names — acceptable as historical record)"
red_flags:
  - severity: "Medium"
    finding: "The canvas-freshness CI gate is a SOFT gate, not the hard per-feature alignment gate ADR-019 advertises: the 'Canvas present' step passes whenever ANY canvas exists in docs/spdd/ (≈30 do), so a feat: PR with no canvas for ITS issue still passes; and Draft canvases only emit a warning, so Aligned-before-merge is enforced only by human review, not by tooling."
    evidence:
      - ".github/workflows/pr-checks.yml:231-233"
      - "cmd/zynax-ci/validate/canvas.go:124-129"
  - severity: "Low"
    finding: "Pervasive command-name doc lag survives even inside the consolidated command files themselves: live .claude/commands/learn.md and experts/go-services.md still cross-reference retired command files (milestone-orchestrate.md, issue-deliver.md) that no longer exist; APPLY_LOG and the CI gate error string cite /m6-learn and /spdd-reasons-canvas. The PR#1400 rename was incomplete; the project's own /reconcile truth-pass has not yet swept these."
    evidence:
      - ".claude/commands/learn.md:193,200,352"
      - ".claude/commands/experts/go-services.md:287-288"
      - ".github/workflows/pr-checks.yml:240"
      - "docs/ai-learnings/APPLY_LOG.md:3,8"
  - severity: "Low"
    finding: "Validator status enum has already drifted from the live canvas lifecycle: 'Superseded' is a real status in use (1359) but rejected by the validator; the validator also disk-flags a correctly-gitignored canvas.private.md as a hard error, producing a non-actionable CI failure on local runs."
    evidence:
      - "cmd/zynax-ci/validate/canvas.go:36-38 (enum lacks Superseded)"
      - "docs/spdd/1359-zero-temporal-engine/canvas.md Status: Superseded"
      - "cmd/zynax-ci/validate/canvas.go:162-170 vs .gitignore:53"
  - severity: "Low"
    finding: "Durability/bus-factor: the methodology and its ADRs (019/028/033) are single-maintainer authored; ADR-033 (expert substrate) is still 'Proposed', and runtime experts beyond one reference are deferred to M-dx — the IP is real but thinly socialized."
    evidence:
      - "docs/adr/ADR-028-...:9 (single decider); docs/adr/ADR-033-...:3 (Proposed); ADR-033:28,57-58 (full library deferred to M-dx)"
green_flags:
  - strength: "A genuinely closed, disciplined learnings feedback loop: APPLY_LOG.md shows a Draft→applied/rejected lifecycle with source-session traceability and real committed LoC deltas to expert guides, and it deliberately REJECTS structural-workaround/env-constraint patterns rather than polluting the guides — this is a working flywheel, not ceremony."
    evidence:
      - "docs/ai-learnings/APPLY_LOG.md:15-99 (multiple dated runs; rows 9-11 re-routed; 3 structural rows rejected; 2 pending → live)"
      - "docs/ai-learnings/ci-release.md (647L) / go-services.md (633L)"
  - strength: "Strong, multi-layer KB-leak control aligned to ADR-018/ADR-019 tiering: gitleaks AI-context config, ai-context-budget gate, CODEOWNERS on every KB path, gitignored private canvases, and a security-review step before any canvas commit. Best-in-class for an AI-authored public repo."
    evidence:
      - ".github/workflows/pr-checks.yml:385-388; .github/workflows/ai-context-budget.yml:7-13; .github/CODEOWNERS:15-20; .gitignore:53; ADR-019:72-81"
  - strength: "Command verbs encode hard-won operational discipline directly into the prompt surface — DCO, Assisted-by-not-Co-Authored-By, squash-only (required_signatures blocks rebase), runtime-evidence-not-config-evidence, re-run-stateful-paths-twice — turning past failure modes into reusable guardrails."
    evidence:
      - ".claude/commands/deliver.md:33-47"
      - ".claude/commands/README.md:108-121 (Conventions enforced)"
  - strength: "Self-hosting automation is honestly bounded, not over-claimed: real Zynax manifests authored, schema tests pass, and the live-platform e2e is gated behind a cleanly-skipping test deferred to M7 — exactly the anti-pattern (green-looking automation on unbuilt capability) that the design doc warns against."
    evidence:
      - "automation/README.md:16-43; automation/tests/test_platform_readiness.py:48-57; automation/STATUS-AND-DIRECTION.md:159-194 (two-plane 'honest line' model)"
open_questions:
  - "Is the Aligned-before-merge requirement enforced anywhere in tooling, or only by human reviewer discipline + branch protection? (Found no machine check that THIS PR's canvas is Aligned.)"
  - "Does any CI step verify a feat: PR's canvas matches its OWN issue number/slug, or only that some canvas changed/exists?"
  - "How many of the 37 'Implemented' canvases have a matching merged PR vs. a back-filled status label? (Spot-checked 214/101 only.)"
unknowns:
  - "Productivity multiplier magnitude — no velocity/DORA baseline-vs-SPDD comparison is committed; the multiplier claim rests on E5 docs, not measured E1 (marked CLAIMED)."
  - "Whether /plan, /deliver, /learn actually run as designed end-to-end — assessed from prompt text (E2) only; not executed in this read-only audit."
cross_references:
  - to_agent: "5.26"
    note: "SPDD + REASONS Canvas + KB-tiering + learnings-loop is the strongest novelty candidate for Innovation/IP scoring; defensibility is process-discipline IP (copyable, single-maintainer-authored). ADR-019/028/033 are the IP artifacts."
    evidence: ["docs/adr/ADR-019-spdd-prompt-governance.md", "docs/adr/ADR-028-...", "docs/adr/ADR-033-...", "docs/ai-learnings/APPLY_LOG.md"]
  - to_agent: "5.5"
    note: "If heavy AI authorship correlates with quality issues, the canvas-freshness soft gate (Draft-passes, any-canvas-passes) is the most likely leak point — recommend correlating AI-authored PRs against post-merge defect/[AUTO]-issue rate."
    evidence: [".github/workflows/pr-checks.yml:231-233", "cmd/zynax-ci/validate/canvas.go:124-129"]
  - to_agent: "5.7"
    note: "Security-review step + gitleaks-ai-context + CODEOWNERS feed the security-posture picture; ADR-018 KB authorization is a shared control."
    evidence: ["docs/adr/ADR-018-ai-kb-authorization-model.md", ".github/workflows/pr-checks.yml:385-388"]
recommendations:
  - priority: "P1"
    action: "Harden the canvas-freshness gate: require a canvas whose directory matches the PR's issue number, and fail (not warn) if its Status is not Aligned at merge time."
    rationale: "Closes the gap between ADR-019's 'enforced gate' claim and the soft tooling reality; converts the PARTIAL drift result to VERIFIED."
  - priority: "P2"
    action: "Run /reconcile to sweep residual command-name lag (pr-checks.yml:240, APPLY_LOG /m6-learn, learn.md/go-services.md refs to retired milestone-orchestrate.md/issue-deliver.md) and add 'Superseded' to the validator status enum."
    rationale: "The project's own truth-pass tool exists; the lag is exactly its job and undermines the otherwise strong self-guided map."
  - priority: "P2"
    action: "Commit a velocity/quality baseline (pre- vs post-SPDD DORA or defect-rate) so the productivity-multiplier claim becomes VERIFIED (E1) rather than CLAIMED (E5)."
    rationale: "Turns the methodology's headline value-prop into evidenced IP for Innovation (5.26)."
```

## (b) §6.2 Prose section

## 5.12 AI Workflow — Score: 7 (High)

**Mission recap:** Assess whether Zynax's SPDD pipeline, command surface, context management, and learnings loop are a genuine asset or a liability.

**Verdict:** Zynax has built a coherent, unusually disciplined AI-native development methodology — REASONS Canvas governance (ADR-019), three-tier KB security (ADR-018), a consolidated 5-verb command surface, and a genuinely closed learnings flywheel — and it has run at scale (37 Implemented canvases, real expert-guide LoC deltas). The headline weakness is that the canvas-before-code **gate is softer than advertised**: the CI check passes whenever any canvas exists in the repo and Draft canvases only warn, so "Aligned before merge" rests on human review, not tooling. The system is a real productivity asset and a partly-novel process IP, tempered by single-maintainer bus factor and pervasive (if low-severity) command-name doc lag.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| SPDD enforcement (gate vs convention) | 6 | High | pr-checks.yml:231-233; canvas.go:124-129; ADR-019:35-41 |
| Canvas validator correctness (E1) | 7 | High | `go run . validate canvas` → 2 FAILs (Superseded, private.md); canvas.go:36-56 |
| Command surface quality | 8 | High | README.md:13-49; deliver.md:38-47; experts 201-408L |
| Cognitive-load reduction | 8 | Medium | README.md:10-13,53; CLAUDE.md:88-115 |
| Context mgmt / KB tiering (ADR-018) | 8 | High | ADR-018:42-76; pr-checks.yml:385-388; CODEOWNERS:15-20; .gitignore:53 |
| Hallucination/quality prevention | 8 | High | APPLY_LOG.md:15-99; spdd-guide.md:145-177 |
| Self-hosting automation real? | 6 | High | automation/README.md:16-43; test_platform_readiness.py:48-57 |
| Defensible IP vs overhead | 7 | Medium | ADR-019:104-111; 37 Implemented canvases; ADR-028:9 bus factor |

**Drift test:**
- *Canvas-before-code is an enforced gate* → **PARTIAL.** Gate job is real (pr-checks.yml:204) but passes if any canvas exists (231-233) and Draft only warns (canvas.go:124-129).
- *Canvases map to shipped features* → **VERIFIED.** 37 Implemented; 214 canvas ↔ temporal_workflow.go/interpreter.go shipped.
- *CLAUDE.md §SPDD + spdd-guide.md lag the live commands (§1.10)* → **CONTRADICTED — lag RESOLVED in those two files** (grep finds zero old names), but residual lag persists in pr-checks.yml:240, APPLY_LOG, and internal .claude/commands cross-refs to retired files.

**Red flags (severity-ordered):**
1. **Medium —** Soft canvas-freshness gate: any-canvas-passes + Draft-only-warns → "Aligned before merge" not machine-enforced (pr-checks.yml:231-233; canvas.go:124-129).
2. **Low —** Command-name lag survives inside the consolidated files themselves (learn.md:193,352; go-services.md:287-288 → retired milestone-orchestrate.md/issue-deliver.md; pr-checks.yml:240; APPLY_LOG:3,8).
3. **Low —** Validator enum drift: 'Superseded' rejected (canvas.go:36-38 vs 1359 canvas); gitignored private.md hard-flagged on disk (canvas.go:162-170 vs .gitignore:53).
4. **Low —** Single-maintainer-authored IP; ADR-033 still Proposed, full expert library deferred to M-dx.

**Green flags:**
- Closed, disciplined learnings loop that rejects structural-workaround patterns and shows real committed deltas (APPLY_LOG.md:15-99).
- Multi-layer KB-leak control (gitleaks-ai-context + budget gate + CODEOWNERS + gitignored private canvases + security-review) — ADR-018/019.
- Verbs encode operational scars as guardrails (deliver.md:33-47: DCO, Assisted-by, squash-only, runtime-evidence, run-twice).
- Self-hosting honestly bounded behind a cleanly-skipping platform gate, not over-claimed (automation/README.md:16-43; test_platform_readiness.py:48-57).

**Open questions / unknowns:** Is Aligned-before-merge enforced anywhere but human review? Does any check tie a canvas to its OWN issue? Productivity-multiplier magnitude is CLAIMED (E5), not measured (no committed DORA baseline). Pipeline behaviour assessed from prompt text (E2), not executed.

**Recommendations:** P1 — harden the gate (issue-matched canvas + fail-on-not-Aligned-at-merge). P2 — /reconcile sweep the residual command-name lag and add 'Superseded' to the validator. P2 — commit a velocity/quality baseline to upgrade the multiplier claim to VERIFIED.

**Cross-references:** 5.26 (SPDD/Canvas/learnings = primary novelty/IP candidate; defensibility is copyable process discipline); 5.5 (soft gate is the likeliest AI-quality leak point — correlate AI-authored PRs vs defect rate); 5.7 (security-review + gitleaks-ai-context + ADR-018 KB authorization are shared controls).
<!-- Zynax Investment-Grade Due-Diligence — Agent 5.24 Repository Health — Wave A (ground-truth) -->
<!-- Issue #1402 · target HEAD e3135a6 (2026-06-20) · READ-ONLY audit · all claims evidenced or marked UNKNOWN -->

# Agent 5.24 — Repository Health (Wave A, ground-truth)

## (a) §3.4 Handoff Packet

```yaml
agent: "5.24 Repository Health"
wave: "A"
dimension_groups: ["D16"]
overall_score: 7
overall_confidence: "High"

sub_scores:
  - dimension: "Commit cadence & recency (actively maintained?)"
    score: 9
    confidence: "High"
    justification: "Very high, accelerating cadence; HEAD commit hours old; 824 commits in ~2 months."
    evidence:
      - "git log -1 --format='%ci' HEAD → 2026-06-20 01:35:24 +0000 (audit date 2026-06-20)"
      - "git rev-list --count HEAD → 824"
      - "git log per-month: 2026-04=121, 2026-05=259, 2026-06=444 (first 20 days) → accelerating"
      - "first commit 2026-04-20 09:58:32 +0200 → project age ~2 months"

  - dimension: "Contributor distribution (bus factor)"
    score: 3
    confidence: "High"
    justification: "Effectively a single human committer; all non-bot work by one person (two emails)."
    evidence:
      - "git shortlog -sne HEAD → 769 Oscar Gómez <oscar.gomez-at-gmail-dot-com> + 3 Oscar Gómez Manresa <ogomezmanresa-at-gmail-dot-com> = 772 human; 45 github-actions[bot]; 7 renovate[bot]"
      - "git log --format='%G? %an' -50 | sort | uniq -c → 39 'Oscar Gómez', 11 'github-actions[bot]' (no other human author in last 50)"
      - "docs/product/strategy.md §8 (per §1.9 packet): 'single-maintainer bus factor' acknowledged CLAIMED"

  - dimension: "Branch / PR / merge hygiene & discipline (ADR-023 squash-only, signed)"
    score: 8
    confidence: "High"
    justification: "Strictly linear (zero merge commits), squash-only, no orphaned PRs; minor local merged-branch leftover."
    evidence:
      - "git log --oneline --merges main -100 | wc -l → 0 (no merge commits — squash-only, ADR-023)"
      - "git log --oneline -30 --merges → empty"
      - "gh pr list --state open → empty (no orphaned/stale PRs)"
      - "git for-each-ref refs/remotes/origin → all remote branches 2026-06-19/20 (no stale remote sprawl)"
      - "git branch (local) → ~28 squash-merged-but-unpruned local branches, all 2026-06-18..06-20 (hygiene nit, recent)"

  - dimension: "Commit signing posture (signed merges, ADR-023 / required_signatures)"
    score: 8
    confidence: "Medium"
    justification: "All human commits carry signatures; only bot digest-sync commits unsigned. Local trust-store lacks key so cannot cryptographically verify here."
    evidence:
      - "git log --format='%H %G?' -10 → human commits 'E' (signature present, key not in local store), bot 'N'"
      - "git log --show-signature -3 → 'gpg: Firmado ... usando RSA clave B5690EEEBB952194 / Imposible comprobar la firma: No public key' (signed, unverifiable locally)"
      - "git log --format='%G? %an' -50 → 39 E (Oscar), 11 N (github-actions[bot] digest-sync commits only)"
      - "CLAUDE.md memory: 'branch protection requires it' (SSH signing) — CLAIMED, enforcement not directly verifiable read-only here"

  - dimension: "[AUTO] drift-issue pile (digest-drift / smoke / size / security) & triage state"
    score: 8
    confidence: "High"
    justification: "Auto-issue mechanism was retired in favor of loud red runs; historical [AUTO] burst fully closed; ~0 open untriaged."
    evidence:
      - "gh issue list --search 'AUTO in:title' --state all → 23 total; --state open → 1, which is #244 'auto-deploy on main merge' (FALSE POSITIVE, real feature issue)"
      - "All real [AUTO] issues (#1035-#1054 digest-drift + post-merge smoke, 2026-06-09 burst) → CLOSED"
      - ".github/workflows/weekly-audit.yml:4-8 → 'Replaces the per-merge ... mesh: audits fail loudly as a red workflow run instead of auto-filing [AUTO] skeleton issues'"

  - dimension: "Repo cleanliness (committed build artifacts, gitignore hygiene)"
    score: 8
    confidence: "High"
    justification: "No committed build artifacts/binaries; comprehensive curated .gitignore; one uncommitted local go.work.sum drift in working tree (not committed)."
    evidence:
      - "git ls-files '*.out' '*coverage*' '*.test' '*.prof' → only legitimate Go/JS source (coveragecomment.go, coverage_test.go) — no coverage.out/binary artifacts"
      - "git ls-files '*.exe' '*.so' '*.o' '*.a' → empty (no committed binaries)"
      - ".gitignore:1-57 → curated: Go artifacts, Python caches, SBOM, secrets globs, AI config, scratch files, explicit 'generated proto stubs ARE committed' notes"
      - "git status --porcelain → ' M go.work.sum' (local toolchain drift, 148 insertions, uncommitted; last committed 2026-06-16 — NOT introduced by this audit)"

  - dimension: "Open-vs-closed issue ratio (project throughput / hygiene)"
    score: 8
    confidence: "High"
    justification: "39 open / 632 closed (~94% closure); open set is genuine roadmap/epic/DD work, not stale clutter."
    evidence:
      - "gh issue list --state open --json number --jq length → 39"
      - "gh issue list --state closed --jq length → 632 → ratio 632/(632+39)=94.2%"
      - "gh issue list --state open titles → all are epics/DD-execution/M7-M8 features (e.g. #1419 OIDC auth, #1417 fuzz, #1404 Wave C); no untriaged bot clutter"

  - dimension: "CHANGELOG / state-file currency (delivery-vs-narrative drift signal)"
    score: 6
    confidence: "High"
    justification: "state/current-milestone.md is current & detailed; CHANGELOG [Unreleased] lags far behind ~115 closed M7 issues (only 1 entry). Prior phantom-entry purge on record."
    evidence:
      - "CHANGELOG.md:15-20 → [Unreleased] holds only #1214; M7 has ~115 closed issues (state/current-milestone.md:63) → CHANGELOG lag"
      - "CHANGELOG.md:44 → '#473 CHANGELOG phantom entries removed (... features that do not exist in git)' — documented prior drift, since corrected"
      - "state/current-milestone.md:63 → 'GitHub milestone #7 — ~115 closed / ~10 open (~92% done)'"
      - "state/current-milestone.md:1 → 'Updated by /milestone-close and /repo-clean' — actively maintained canonical status file"

drift_test:
  - claim: "M7 ~115 closed / ~10 open (~92% done) as of 2026-06-19 (state/current-milestone.md:63)"
    result: "PARTIAL"
    evidence:
      - "gh issue list --state open --jq length → 39 repo-wide open (milestone-scoped count not isolated read-only; #1370 epic visibly delivered per PRs #1431-#1441 in git log)"
      - "git log → PRs #1431-#1441 (M7.K cluster) present on main → corroborates near-complete M7 delivery"
      - "Direction VERIFIED (high delivery), exact milestone-7 numerator not independently reproduced → PARTIAL"
  - claim: "Strict merge discipline: squash-only, signed (ADR-023)"
    result: "VERIFIED"
    evidence:
      - "git log --merges -100 → 0 merge commits (linear/squash-only)"
      - "git log --format='%G?' human commits → 'E' (signature present); bot-only commits 'N'"
  - claim: "Auto pipelines generate [AUTO] noise that must be triaged (prompt CONTEXT)"
    result: "VERIFIED (mechanism retired)"
    evidence:
      - ".github/workflows/weekly-audit.yml:4-8 → [AUTO] skeleton issues replaced by loud red runs"
      - "gh search 'AUTO in:title' → 23 total, all real ones CLOSED, 0 open untriaged"
  - claim: "Velocity/cadence is high and sustained (implied by active M7 narrative)"
    result: "VERIFIED"
    evidence:
      - "Per-month: 121 (Apr) → 259 (May) → 444 (Jun, first 20 days) → accelerating, matches active-milestone narrative"

red_flags:
  - severity: "High"
    finding: "Bus factor = 1. Effectively one human committer (Oscar, two emails = 772 commits); no second human author in the entire shortlog or last-50 window. A single point of failure for a project targeting CNCF (needs >=2 maintainers from 2+ orgs)."
    evidence:
      - "git shortlog -sne HEAD → only one human identity (772 commits across 2 emails); rest are github-actions[bot] (45) + renovate[bot] (7)"
      - "git log --format='%G? %an' -50 → no second human author"
      - "state/current-milestone.md / §1.9 packet: CNCF gate is 'social, not technical' — single-maintainer bus factor"
  - severity: "Low"
    finding: "~28 squash-merged local branches left unpruned; one uncommitted go.work.sum drift in working tree (148 insertions). Cosmetic local hygiene, not committed to history."
    evidence:
      - "git branch → ~28 local feat/fix/docs/refactor branches (all squash-merged, appear unmerged to git, 2026-06-18..06-20)"
      - "git status --porcelain → ' M go.work.sum' (uncommitted, last committed 2026-06-16)"
  - severity: "Low"
    finding: "Hand-curated CHANGELOG [Unreleased] lags ~115 closed M7 issues (1 entry). state file is current, but CHANGELOG is the public-facing surface most prone to the project's documented narrative-drift class."
    evidence:
      - "CHANGELOG.md:15-20 vs state/current-milestone.md:63 (~115 closed)"
      - "CHANGELOG.md:44 → prior phantom-entry purge (#473) shows this surface has drifted before"

green_flags:
  - strength: "Exceptionally active, accelerating cadence on a ~2-month-old repo (121→259→444 commits/month), HEAD hours old at audit."
    evidence:
      - "git log per-month uniq -c → 121/259/444"
      - "git log -1 --format='%ci' → 2026-06-20 01:35:24 (audit-day commit)"
  - strength: "Strict, clean merge discipline: 0 merge commits (squash-only/linear, ADR-023), all human commits signed, 0 open/orphaned PRs."
    evidence:
      - "git log --merges -100 → 0"
      - "gh pr list --state open → empty"
      - "git log --format='%G?' human → 'E'"
  - strength: "Mature automation-noise handling: [AUTO] auto-issue mechanism deliberately retired for loud red runs; historical burst (#1035-#1054) fully closed; ~0 untriaged auto-issues."
    evidence:
      - ".github/workflows/weekly-audit.yml:4-8"
      - "gh search AUTO in:title → 23 total, 0 genuinely open"
  - strength: "Clean repo: no committed build artifacts/binaries; comprehensive, well-commented .gitignore; 94% issue closure (39 open / 632 closed)."
    evidence:
      - "git ls-files binary/artifact globs → none committed"
      - ".gitignore:1-57"
      - "gh issue counts 39/632"
  - strength: "[skip ci] marker confined to bot digest-sync/ci-runner-bump commits only — never on human feature PRs (no CI-bypass on real work)."
    evidence:
      - "git log --grep='skip ci' -20 → 100% 'chore(images): sync digests' / 'bump ci-runner digest' bot commits"

open_questions:
  - "Exact M7-milestone-scoped open/closed split (gh milestone filtering not isolated here) — direction is clearly near-complete but the precise ~92% numerator is unconfirmed read-only."
  - "Is required_signatures branch protection actually enforced on main, or is signing merely configured? Not verifiable read-only from the clone."
  - "Who, if anyone, is the designated second maintainer / successor? MAINTAINERS.md is an OPEN issue (#494), implying none exists yet."

unknowns:
  - "Cryptographic signature validity: commits show 'E' / 'gpg ... Imposible comprobar la firma: No public key' — signatures present but the signing key is absent from this environment's trust store, so verification is UNKNOWN (Medium confidence on the signing sub-score)."
  - "Branch-protection rule contents (required status checks, required_signatures, up-to-date-before-merge) — server-side config not readable from the working tree."

cross_references:
  - to_agent: "5.8"
    note: "Bus factor = 1 is a primary governance/sustainability risk; feed into team/maintainer-depth scoring."
    evidence: ["git shortlog -sne → single human identity (772 commits)"]
  - to_agent: "5.15"
    note: "Single-maintainer concentration is the CNCF-gating social risk (>=2 maintainers from 2+ orgs unmet); MAINTAINERS.md still an open issue (#494)."
    evidence: ["gh issue list open → #494 'create MAINTAINERS.md ... single-maintainer reality'", "§1.9 packet"]
  - to_agent: "5.13"
    note: "Merge/CI discipline (squash-only, signed, [skip ci] confined to bots, weekly-audit replacing [AUTO] noise) corroborates CI/release health."
    evidence: [".github/workflows/weekly-audit.yml:4-8", "git log --merges -100 → 0"]
  - to_agent: "5.23"
    note: "CHANGELOG [Unreleased] lag vs ~115 closed M7 issues is the residual narrative-drift surface; truth-pass (#1445, #494/#496) is active but CHANGELOG trails."
    evidence: ["CHANGELOG.md:15-20", "CHANGELOG.md:44 (#473 phantom purge)", "state/current-milestone.md:63"]

recommendations:
  - priority: "P0"
    action: "Recruit and document a second maintainer (close #494 MAINTAINERS.md); establish bus-factor mitigation before any CNCF Sandbox submission."
    rationale: "Bus factor = 1 is the single most material repo-health/sustainability risk and the explicit CNCF social gate."
  - priority: "P2"
    action: "Prune squash-merged local branches and reconcile the [Unreleased] CHANGELOG against M7 closed issues at release cut (or auto-generate via `make changelog`)."
    rationale: "Closes the residual hygiene + narrative-drift surface; cheap, recurring."
  - priority: "P2"
    action: "Add a periodic working-tree-clean / generated-artifact (go.work.sum) check so local toolchain drift never reaches a PR."
    rationale: "Keeps the otherwise-clean tree clean and prevents accidental artifact commits."
```

## (b) §6.2 Prose Section

## Repository Health — Score: 7 (High)

Mission recap: Assess objective repo-health signals — commit activity, contributor distribution, branch/PR/merge hygiene, drift artifacts, automation noise, and repo cleanliness — and run the drift test on claimed velocity.

Verdict: This is a **living, exceptionally well-tended, and disciplined repository** with one dominant structural weakness. Cadence is high and accelerating (121→259→444 commits/month on a ~2-month-old, 824-commit repo), history is strictly linear (zero merge commits, squash-only per ADR-023), human commits are signed, there are no orphaned PRs, no committed build artifacts, a curated `.gitignore`, a 94% issue-closure ratio, and a deliberately matured automation-noise posture (the `[AUTO]` skeleton-issue mechanism was retired for loud red weekly-audit runs). The one thing dragging the score down hard is **bus factor = 1**: a single human contributor accounts for effectively 100% of non-bot commits. The repo is clean and the process is mature; the *people* dimension is the hidden risk, and it is the same risk that gates the project's CNCF ambition.

Sub-dimension scores:

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Commit cadence & recency | 9 | High | per-month 121/259/444; HEAD 2026-06-20; 824 commits |
| Contributor distribution (bus factor) | 3 | High | `git shortlog` → one human identity (772), bots 45+7 |
| Branch/PR/merge hygiene & discipline | 8 | High | `--merges -100` → 0; `gh pr list open` → empty |
| Commit signing posture | 8 | Medium | `%G?` → human 'E', bot 'N'; key absent locally |
| `[AUTO]` pile & triage | 8 | High | weekly-audit.yml:4-8; 23 AUTO total, 0 open |
| Repo cleanliness | 8 | High | no artifact/binary tracked; .gitignore:1-57 |
| Open-vs-closed issues | 8 | High | 39 open / 632 closed (94%) |
| CHANGELOG/state currency | 6 | High | CHANGELOG [Unreleased] lags ~115 M7 closes |

Drift test:
- "M7 ~92% done" → **PARTIAL** — direction VERIFIED via M7.K PRs #1431-#1441 on main; exact milestone numerator not isolated read-only.
- "Squash-only, signed (ADR-023)" → **VERIFIED** — 0 merge commits in 100; human commits signed.
- "[AUTO] noise must be triaged" → **VERIFIED (mechanism retired)** — auto-issues replaced by red runs; all real `[AUTO]` issues closed.
- "High, sustained velocity" → **VERIFIED** — 121→259→444 commits/month matches the active-milestone narrative.

Red flags (severity-ordered):
1. **High — Bus factor = 1.** `git shortlog -sne` shows a single human identity (772 commits across two emails); no second human author appears in the entire shortlog or the last 50 commits. Single point of failure; unmet CNCF `>=2 maintainers` gate; `MAINTAINERS.md` still an open issue (#494).
2. **Low — ~28 unpruned squash-merged local branches + one uncommitted `go.work.sum` drift** (148 insertions, last committed 2026-06-16). Cosmetic, local-only, not in committed history.
3. **Low — CHANGELOG `[Unreleased]` lags ~115 closed M7 issues** (1 entry). The state file is current; the public-facing CHANGELOG is the surface most prone to the project's documented narrative-drift class (a phantom-entry purge already happened, CHANGELOG.md:44 / #473).

Green flags:
- Exceptionally active, accelerating cadence; HEAD hours old at audit.
- Strict, clean merge discipline (0 merge commits, signed human commits, 0 open PRs).
- Mature automation-noise handling (`[AUTO]` retired for loud red runs; historical burst fully closed).
- Clean repo (no committed artifacts; curated `.gitignore`; 94% issue closure).
- `[skip ci]` confined to bot digest/ci-runner commits — never on human feature PRs.

Open questions / unknowns:
- Exact M7 milestone open/closed split not isolated read-only (direction clearly near-complete).
- Whether `required_signatures` is *enforced* on main or merely configured — not verifiable from the clone.
- Signature cryptographic validity is UNKNOWN locally (`No public key`); signatures are present but the key is absent from this environment's trust store.
- No designated second maintainer / successor appears to exist (MAINTAINERS.md is open issue #494).

Recommendations:
- **P0** — Recruit and document a second maintainer (close #494); mitigate bus factor before any CNCF submission. Rationale: the single most material sustainability risk and the explicit CNCF social gate.
- **P2** — Prune merged local branches; reconcile `[Unreleased]` CHANGELOG at each release cut (or auto-generate). Rationale: closes residual hygiene + narrative-drift surface.
- **P2** — Add a working-tree-clean / generated-artifact guard (e.g. `go.work.sum`) so local toolchain drift never reaches a PR. Rationale: keeps the otherwise-clean tree clean.

Cross-references:
- **5.8 / 5.15** — Bus factor = 1 is the primary governance/sustainability and CNCF-social-gate risk; `MAINTAINERS.md` still open (#494).
- **5.13** — Merge/CI discipline (squash-only, signed, `[skip ci]` bot-confined, weekly-audit) corroborates CI/release health.
- **5.23** — CHANGELOG lag vs M7 closes is the residual narrative-drift surface; truth-pass (#1445) active but CHANGELOG trails.
