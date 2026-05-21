<!-- SPDX-License-Identifier: Apache-2.0 -->

# 00 — Repository Inventory

**Date:** 2026-05-21  
**Branch:** docs/architecture-overhaul-m5  
**Purpose:** Phase 0 artifact — complete file/directory map, doc-artifact list,
agent-context census, and AI-context-budget baseline.

---

## 1. Repository Facts (verified against HEAD)

| Property | Value |
|---|---|
| Remote | `github.com/zynax-io/zynax` |
| Default branch | `main` |
| Working branch | `docs/architecture-overhaul-m5` |
| License | Apache-2.0 |
| Languages | Go (~68%), Gherkin/BDD (~17%), Python (~13%), Makefile |
| Go version | 1.26.3 (go.work) |
| Python version | 3.12 (ADR-002) |
| Package manager (Python) | uv (ADR-003) |
| Releases cut | v0.4.0 tag pending (CHANGELOG promoted, `git push origin v0.4.0` pending per state/current-milestone.md) |

---

## 2. Top-Level Directory Map

```
zynax/
├── spec/                   Layer 1 — YAML manifests + JSON schemas + AsyncAPI
│   ├── asyncapi/           AsyncAPI spec (11 event channels defined)
│   ├── schemas/            JSON Schema files (Workflow, AgentDef, Policy)
│   ├── tests/              Schema validation tests
│   └── workflows/examples/ 5 example workflow YAML files
│
├── protos/                 Layer 2 — gRPC contracts (source of truth)
│   ├── zynax/v1/           8 proto files (agent, workflow_compiler, engine_adapter,
│   │                       task_broker, agent_registry, event_bus, memory_service, cloudevents)
│   ├── buf.yaml / buf.gen.yaml
│   ├── generated/          Python stubs (auto-generated, committed)
│   ├── tests/              BDD contract test suites (godog, 140+ scenarios)
│   └── AGENTS.md
│
├── services/               Layer 3 — Go platform services
│   ├── workflow-compiler/  ✅ Implemented — YAML → WorkflowIR (go.mod, ~1390 LoC)
│   ├── engine-adapter/     ✅ Implemented — IR → Temporal (go.mod, ~1143 LoC)
│   ├── api-gateway/        ✅ Implemented — HTTP REST entry point (go.mod, ~1047 LoC)
│   ├── task-broker/        🟡 In-memory MVP (go.mod, ~905 LoC); not in compose yet
│   ├── agent-registry/     ❌ Stub — AGENTS.md + feature files only; no go.mod, 0 LoC
│   ├── event-bus/          ❌ Stub — AGENTS.md only; no go.mod, 0 LoC
│   ├── memory-service/     ❌ Stub — AGENTS.md only; no go.mod, 0 LoC
│   └── AGENTS.md
│
├── agents/                 Layer 4 — Python execution adapters + SDK
│   ├── sdk/                Python SDK — zynax-sdk (Agent base class, @capability routing)
│   ├── adapters/
│   │   └── http/           ✅ HTTP adapter — REST capability proxy
│   ├── examples/           Reference feature files (summarizer, etc.)
│   └── AGENTS.md
│
├── cmd/
│   ├── zynax/              ✅ CLI — apply/get/delete/status/logs (go.mod, ~1470 LoC)
│   └── zynax-ci/           ✅ CI toolchain — validate canvas/schema/manifests, check ai-context
│
├── infra/
│   ├── docker/             Dockerfile.tools, Dockerfile.ci-runner
│   └── docker-compose/     docker-compose.yml (canonical), overrides
│
├── gen/go/                 Go protobuf stubs (auto-generated, committed)
├── docs/                   All documentation (see §4 below)
├── state/                  current-milestone.md
├── tools/                  Tool scripts
├── .github/workflows/      14 CI workflow files
├── .claude/commands/       8 SPDD slash-command definitions
└── [root governance files] README.md ARCHITECTURE.md AGENTS.md CLAUDE.md
                            CHANGELOG.md ROADMAP.md SECURITY.md GOVERNANCE.md
                            CONTRIBUTING.md CODE_OF_CONDUCT.md LICENSE
                            Makefile go.work .pre-commit-config.yaml renovate.json
                            .trivyignore .dockerignore
```

---

## 3. GitHub Actions Workflows (14 files)

| File | Purpose | Size |
|---|---|---|
| `ci.yml` | Main CI — lint, test, coverage, BDD, security | 1,325 lines (⚠ oversized) |
| `pr-checks.yml` | PR validation — conventional-commit, PR-size, canvas | ~600 lines |
| `release.yml` | Unified tag-triggered release — CLI + zynax-ci + service images | ~550 lines |
| `cli-release.yml` | CLI binary release (also called from release.yml) | ~220 lines |
| `service-release.yml` | Service image release (also called from release.yml) | ~225 lines |
| `zynax-ci-release.yml` | zynax-ci binary release | ~115 lines |
| `tools-image.yml` | tools Docker image rebuild on Dockerfile.tools change | ~160 lines |
| `proto-generate.yml` | Auto-regenerate stubs on .proto change, post-merge | ~115 lines |
| `proto-stubs-publish.yml` | Publish proto stubs to BSR (Buf Schema Registry) | ~220 lines |
| `pr-size.yml` | PR size gate | ~55 lines |
| `ai-context-budget.yml` | Advisory AI context line count check | ~40 lines |
| `kb-preview.yml` | Knowledge-base preview builds | ~165 lines |

Required status checks (enforced): `CI / lint`, `CI / test-unit-go`, `CI / test-bdd`,
`CI / coverage-gate`, `PR Checks / conventional-commit`, `PR Checks / pr-size`,
`PR Checks / validate-canvas`.

---

## 4. Documentation Artifacts

### Root governance

| File | Purpose | State |
|---|---|---|
| `README.md` (449 lines) | Project overview, quickstart, milestone status | Mostly current (see §5) |
| `ARCHITECTURE.md` (510 lines) | Architecture design and rationale | ⚠ **Severely stale** — milestone table shows M2 as "Next", M3 as "Planned" |
| `AGENTS.md` (217 lines) | Engineering constitution for contributors + AI | Mostly current; 3 broken file links |
| `CLAUDE.md` (195 lines) | Session bootstrap for AI coding assistants | Current |
| `CHANGELOG.md` | Notable changes per milestone | Current (v0.4.0 accurate) |
| `ROADMAP.md` | Narrative roadmap M1–M8 | Mostly current; M5 section needs completion state |
| `SECURITY.md` | Security policy and controls | Truth-pass complete (2026-05-20) |
| `GOVERNANCE.md` | Contributor governance, DCO, conflict resolution | Current |
| `CONTRIBUTING.md` | Contribution guide | Current |
| `CODE_OF_CONDUCT.md` | CNCF CoC | Current |

### Architecture reviews (`docs/architecture/`)

| File | Date | Summary |
|---|---|---|
| `2026-04-30-competitive-analysis.md` | 2026-04-30 | Competitive landscape (Temporal, Dapr, Argo, LangGraph) |
| `2026-04-30-execution-architecture.md` | 2026-04-30 | Execution architecture deep-dive (M3 era) |
| `2026-05-18-external-architectural-review.md` | 2026-05-18 | External review — pre-M5.F |
| `2026-05-20-principal-architect-review.md` | 2026-05-20 | **Authoritative review** — 6.5/10 overall, G1-G24 gap list, 30-day plan |

### ADRs (`docs/adr/`)

| ADR | Title | Status |
|---|---|---|
| ADR-001 | gRPC as inter-service protocol | Accepted |
| ADR-002 | Python 3.12 | Accepted |
| ADR-003 | uv package manager | Accepted |
| ADR-004 | BDD testing | **Superseded by ADR-016** |
| ADR-005 | Apache 2.0 | Accepted |
| ADR-006 | Monorepo | Accepted |
| ADR-007 | Pydantic Settings | Accepted |
| ADR-008 | No shared databases | Accepted |
| ADR-009 | Language strategy (Go/Python) | Accepted |
| ADR-010 | Pluggable agent runtime | Accepted |
| ADR-011 | Declarative YAML control plane | Accepted |
| ADR-012 | Workflow IR | Accepted |
| ADR-013 | Adapter-first, no mandatory SDK | Accepted |
| ADR-014 | Event-driven state machine model | Accepted |
| ADR-015 | Pluggable workflow engines | Accepted |
| ADR-016 | Layered testing strategy | Accepted |
| ADR-017 | Contract test isolation (GOWORK=off) | Accepted |
| ADR-018 | AI KB authorization model | Accepted |
| ADR-019 | SPDD prompt governance | Accepted |
| ADR-020 | Zero-trust intra-service security | **Not yet filed** (planned; issue #240) |
| ADR-021 | Horizontal scale + multi-tenancy | **Not yet filed** (planned; issue #578) |

### Milestone docs (`docs/milestones/`)

| File | Purpose | State |
|---|---|---|
| `M1-engineering-review.md` | M1 engineering review | Complete |
| `M1-release-notes.md` | M1 release notes | Complete |
| `M5-plan.md` (494 lines) | M5 authoritative execution plan | Current (rev 30, 2026-05-21) |
| M2–M4 reviews | Missing | ⚠ **Not yet created** |

### SPDD Canvas artifacts (`docs/spdd/`)

27 canvas directories. Notable ones:

| Directory | Issue | Status |
|---|---|---|
| `214-temporal-execution/` | #214 | M3 epic |
| `314-yaml-system-cli/` | #314 | M4 epic |
| `377-adapter-library/` | #377 | M5 adapter library epic |
| `380–384-*` | #380–384 | Adapter canvases (http ✅, git/ci/llm/langgraph BDD done) |
| `458-truth-pass/` | #458 | M5.A truth pass |
| `459-engine-correctness/` | #459 | M5.B engine correctness |
| `460-capability-dispatch/` | #460 | M5.C capability dispatch E2E |
| `461-security-baseline/` | #461 | M5.D security baseline ✅ |
| `462-dx-polish/` | #462 | M5.E DX polish ✅ |
| `474-python-sdk/` | #474 | Python SDK ✅ |
| `476-guard-parser/` | #476 | cel-go guard ✅ |
| `479-task-broker/` | #479 | task-broker MVP ✅ |
| `480-agent-registry/` | #480 | agent-registry pending |

### Engineering docs (`docs/engineering/`, `docs/patterns/`, `docs/decisions/`)

| File | Purpose |
|---|---|
| `docs/engineering/dependency-strategy.md` | Dependency version policy |
| `docs/engineering/renovate-fix-sop.md` | Renovate CI failure SOP |
| `docs/patterns/bdd-contract-testing.md` | BDD contract testing guide |
| `docs/patterns/go-service-patterns.md` | Go service code templates |
| `docs/patterns/python-agent-guide.md` | Python agent patterns |
| `docs/patterns/proto-interop.md` | Multi-language proto guide |
| `docs/patterns/helm-charts.md` | Helm chart templates |
| `docs/patterns/spdd-guide.md` | Full SPDD workflow guide |
| `docs/decisions/001–004` | Minor scoped decisions |
| `docs/reviews/` | **This task's audit trail** (being created) |

---

## 5. Agent-Context Files (AI Context Budget)

The `zynax-ci check ai-context` command enforces advisory line-count budgets.

| File | Limit | **Current** | Delta | Notes |
|---|---|---|---|---|
| `CLAUDE.md` | 200 | **195** | -5 | ✅ Within budget |
| `AGENTS.md` (root) | 300 | **217** | -83 | ✅ Has 3 broken links (see §6) |
| `docs/ai-assistant-setup.md` | 150 | **139** | -11 | ✅ Within budget |
| `services/workflow-compiler/AGENTS.md` | 150 | **71** | -79 | ✅ |
| `services/engine-adapter/AGENTS.md` | 150 | **98** | -52 | ✅ |
| `services/api-gateway/AGENTS.md` | 150 | **70** | -80 | ✅ |
| `services/task-broker/AGENTS.md` | 150 | **122** | -28 | ✅ |
| `services/agent-registry/AGENTS.md` | 150 | **56** | -94 | ✅ |
| `services/event-bus/AGENTS.md` | 150 | **57** | -93 | ✅ |
| `services/memory-service/AGENTS.md` | 150 | **61** | -89 | ✅ |
| `agents/sdk/AGENTS.md` | 150 | **126** | -24 | ✅ |
| `agents/adapters/AGENTS.md` | 150 | **125** | -25 | ✅ |
| `cmd/zynax/AGENTS.md` | 150 | **73** | -77 | ✅ |
| `cmd/zynax-ci/AGENTS.md` | 150 | **81** | -69 | ✅ |
| `protos/AGENTS.md` | 150 | **91** | -59 | ✅ |
| `protos/tests/AGENTS.md` | 150 | **78** | -72 | ✅ |
| `spec/AGENTS.md` | 150 | **67** | -83 | ✅ |
| `infra/AGENTS.md` | 150 | **69** | -81 | ✅ |
| **Total** | **2000** | **~1796** | **-204** | ✅ Under budget |

---

## 6. Known Issues Found During Inventory

### Drift / Discrepancies

| # | Location | Issue | Severity |
|---|---|---|---|
| D1 | `ARCHITECTURE.md` §Milestone Status | Shows M2 as "Next", M3 as "Planned" — stale by 4 milestones | **Critical** |
| D2 | `ARCHITECTURE.md` §13 Milestones | Table still says M2 "Next" and M5 "Planned" | Critical |
| D3 | `AGENTS.md` §Knowledge Base Index | References `docs/architecture/execution-architecture.md` (wrong path) | Medium |
| D4 | `AGENTS.md` §Knowledge Base Index | References `docs/architecture/competitive-analysis-2026.md` (wrong path) | Medium |
| D5 | `AGENTS.md` §Knowledge Base Index | References `docs/architecture/2026-05-external-architectural-review.md` (wrong path) | Medium |
| D6 | `README.md` M4 description | Claims "kind: AgentDef routing via AgentRegistryService" — agent-registry doesn't exist | High |
| D7 | `README.md` Quickstart | No caveat that first dispatch will fail (no agent-registry wired) | High |
| D8 | `README.md` GHCR images table | Does not list agent-registry (no image exists — correct, but no explanation) | Low |
| D9 | `docs/milestones/` | M2, M3, M4 engineering reviews missing (only M1 has review + release-notes) | Medium |
| D10 | `AGENTS.md` | References `AGENTS.md §7` for security architecture — no §7 exists in AGENTS.md | Low |

### Missing Docs

- M2, M3, M4 engineering reviews and release notes
- ADR-020 (zero-trust intra-service security — planned)
- ADR-021 (horizontal scale + multi-tenancy — planned)
- `docs/engineering/best-practices/` directory (all standards files missing)
- `docs/reviews/DECISIONS-NEEDED.md` (created by this task)

---

## 7. CI Gates Summary

| Gate | Enforced By | Description |
|---|---|---|
| `proto-breaking` | `pr-checks.yml` | `buf breaking` against main |
| `stubs-freshness` | `pr-checks.yml` | Regenerate stubs and check no diff |
| `layer-boundaries` | `pr-checks.yml` | `zynax-ci validate` layer isolation |
| `conventional-commit` | `pr-checks.yml` | PR title format check |
| `pr-size` | `pr-size.yml` | ≤ 900 LOC (with exclusions) |
| `validate-canvas` | `pr-checks.yml` | `zynax-ci validate canvas` on docs/spdd/ |
| `coverage-gate` | `ci.yml` | ≥ 90% on domain packages |
| `ai-context-budget` | `ai-context-budget.yml` | Advisory (non-blocking); ≤ 2000 lines |
| `trivy-scan` | `release.yml` | Container CVE scan before GHCR push (#565) |
| `golangci-lint` | `ci.yml` | Go lint |
| `ruff/mypy/bandit` | `ci.yml` | Python lint + SAST |

---

*This inventory was produced at phase boundary 0 on branch `docs/architecture-overhaul-m5`.
Re-verify against HEAD before acting on specific file references.*
