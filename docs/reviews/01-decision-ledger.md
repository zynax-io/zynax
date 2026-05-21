<!-- SPDX-License-Identifier: Apache-2.0 -->

# 01 — Decision Ledger

**Date:** 2026-05-21  
**Branch:** docs/architecture-overhaul-m5  
**Purpose:** Phase 1 artifact — complete record of every architectural decision,
its rationale, current status, and whether the code still reflects it.

Each entry answers: What was decided? When? Why? Is it still in force?
Does the code match? Evidence.

---

## Formal ADRs (docs/adr/)

### ADR-001 — gRPC as inter-service protocol
| Field | Value |
|---|---|
| Date | 2026-04-01 |
| Status | **Accepted** |
| Decision | All synchronous service-to-service communication uses gRPC + protobuf |
| Why | Language-agnostic, strongly typed, `buf breaking` CI gate, generated stubs for Go + Python |
| Code reflects? | ✅ Yes — all services use generated stubs from `gen/go/zynax/v1/`; no HTTP between services |
| Still valid? | Yes — reinforced by the 2026-05-20 review which rates the contract layer highly |

### ADR-002 — Python 3.12 as agent runtime
| Field | Value |
|---|---|
| Date | 2026-04-01 |
| Status | **Accepted** |
| Decision | Python 3.12 for all agent/adapter code (`agents/`) |
| Why | Latest stable with improved performance + typing |
| Code reflects? | ✅ Yes — `agents/sdk/pyproject.toml` requires Python ≥ 3.12 |
| Still valid? | Yes |

### ADR-003 — uv as Python package manager
| Field | Value |
|---|---|
| Date | 2026-04-01 |
| Status | **Accepted** |
| Decision | `uv` replaces `pip/poetry` for Python agent environments |
| Why | 10–100× faster than pip; reproducible via `uv.lock` |
| Code reflects? | ✅ Yes — `agents/sdk/uv.lock` exists; `pyproject.toml` uses uv conventions |
| Still valid? | Yes |

### ADR-004 — BDD as primary testing methodology
| Field | Value |
|---|---|
| Date | 2026-04-01 |
| Status | **Superseded by ADR-016** |
| Decision | (original) BDD scenarios as primary test form |
| Superseded because | ADR-016 introduces a tiered testing strategy (BDD at gRPC boundaries, unit ≥90% on domain, `buf breaking` as CI gate) — more precise scope than ADR-004 |
| Code reflects? | ✅ Yes — ADR-016 governs; ADR-004 is informational history only |

### ADR-005 — Apache 2.0 license
| Field | Value |
|---|---|
| Date | 2026-04-01 |
| Status | **Accepted** |
| Decision | Apache-2.0 with SPDX headers on every source file |
| Why | CNCF-compatible vendor-neutral license; DCO for copyright attribution |
| Code reflects? | ✅ Yes — every .go/.py/.md/.proto file carries `<!-- SPDX-License-Identifier: Apache-2.0 -->` |
| Still valid? | Yes |

### ADR-006 — Monorepo structure
| Field | Value |
|---|---|
| Date | 2026-04-01 |
| Status | **Accepted** |
| Decision | Single repo; `go.work` workspace; `buf.work.yaml`; GOWORK=off in service dirs |
| Why | Atomic cross-service changes; single CI pipeline; all stubs co-located |
| Code reflects? | ✅ Yes — `go.work` lists 6 modules |
| Notes | `cmd/zynax` and `cmd/zynax-ci` are **not** in `go.work` — they are standalone modules (see docs/decisions/004-gowork-off-isolation.md) |

### ADR-007 — Pydantic Settings for agent configuration
| Field | Value |
|---|---|
| Date | 2026-04-01 |
| Status | **Accepted** |
| Decision | Pydantic `BaseSettings` for all Python agent config (env-var + `.env` file) |
| Why | Type-safe config, 12-Factor compliance, `.env` for local dev |
| Code reflects? | ✅ Yes — `agents/adapters/http/` uses Pydantic Settings |

### ADR-008 — No shared databases
| Field | Value |
|---|---|
| Date | 2026-04-01 |
| Status | **Accepted** |
| Decision | Each service owns its own schema/namespace; cross-service data flows via gRPC only |
| Why | Prevent distributed monolith; each service can use the right store for its domain |
| Code reflects? | ✅ Yes — no shared ORM models; `services/*/internal/infrastructure/` each manage their own |
| Notes | task-broker uses in-memory store; agent-registry will use in-memory for MVP, Postgres in M6 |

### ADR-009 — Language strategy (Go for services, Python for agents)
| Field | Value |
|---|---|
| Date | 2026-04-01 |
| Status | **Accepted** |
| Decision | All platform services (`services/`, `cmd/`) in Go; all agents/adapters (`agents/`) in Python (or Go with gRPC contract) |
| Why | Go for control plane performance/concurrency; Python for AI/ML ecosystem access |
| Code reflects? | ✅ Yes — no Python in `services/`; no agent code in Go services |

### ADR-010 — Pluggable agent runtime
| Field | Value |
|---|---|
| Date | 2026-04-01 |
| Status | **Accepted** |
| Decision | Agents register capabilities via `AgentService.Register` gRPC; no SDK required |
| Why | Zero lock-in; any language, any framework |
| Code reflects? | ✅ Contractually — `AgentService` in proto; Python SDK provides optional ergonomic wrapper |

### ADR-011 — Declarative YAML control plane
| Field | Value |
|---|---|
| Date | 2026-04-01 |
| Status | **Accepted** |
| Decision | Workflows, AgentDefs, Policies are YAML. YAML is compiled to WorkflowIR — never imported by Go services |
| Why | Kubernetes analogy; GitOps; no programming language required for intent expression |
| Code reflects? | ✅ Yes — `spec/` never imported by `services/`; `layer-boundaries` CI gate enforces this |

### ADR-012 — Workflow IR
| Field | Value |
|---|---|
| Date | 2026-04-01 |
| Status | **Accepted** |
| Decision | `WorkflowIR` protobuf message is the canonical engine-agnostic representation |
| Why | Decouple intent (YAML) from execution (Temporal/LangGraph/Argo) |
| Code reflects? | ✅ Yes — `workflow_compiler.proto` has structured IR fields (M2 additions) |
| Notes | `bytes ir_payload` field 4 kept for backward-compat alongside structured fields; review recommends removing by v1.0 (ADR-012 should be updated) |

### ADR-013 — Adapter-first, no mandatory SDK
| Field | Value |
|---|---|
| Date | 2026-04-01 |
| Status | **Accepted** |
| Decision | Any system can become a capability by implementing `AgentService` gRPC — no SDK required |
| Why | Low adoption friction; language-neutral; Option A (minimal SDK) later chosen for Python ergonomics |
| Code reflects? | ✅ Yes — Python SDK (`agents/sdk/`) is optional helper, not a requirement |
| Notes | Python SDK Agent base class implemented in #535 (M5.A BATCH 3) |

### ADR-014 — Event-driven state machine model
| Field | Value |
|---|---|
| Date | 2026-04-01 |
| Status | **Accepted** |
| Decision | Workflows are finite state machines (states + events + transitions), NOT DAGs |
| Why | Loops, human-in-the-loop, long-running async workflows are natural; DAGs force workarounds |
| Code reflects? | ✅ Yes — `WorkflowIR` proto uses `StateIR` + `TransitionIR`; `IRInterpreterWorkflow` implements FSM |
| Event classification | CloudEvents are **Event Notification** type (not sourcing); NATS is the dumb pipe (see §17 of review) |

### ADR-015 — Pluggable workflow engines
| Field | Value |
|---|---|
| Date | 2026-04-01 |
| Status | **Accepted** |
| Decision | `WorkflowEngine` Go interface (6 methods) decouples gRPC layer from any engine |
| Why | Temporal is v0 default; swap in LangGraph/Argo without rewriting dispatch logic |
| Code reflects? | ✅ Yes — `services/engine-adapter/internal/domain/engine.go` has the 6-method interface; `TemporalEngine` is only implementation |
| Notes | The review calls the `WorkflowEngine` interface one of the "crown jewels" — preserve it |

### ADR-016 — Layered testing strategy
| Field | Value |
|---|---|
| Date | 2026-04-21 |
| Status | **Accepted** (supersedes ADR-004) |
| Decision | Three tiers: (1) BDD at gRPC boundaries (`protos/tests/`); (2) unit ≥90% on `internal/domain/`; (3) `buf breaking` as CI gate |
| Code reflects? | ✅ Yes — all services hit ≥90% domain coverage; 140+ BDD scenarios pass |

### ADR-017 — Contract test isolation (GOWORK=off)
| Field | Value |
|---|---|
| Date | 2026-04-21 |
| Status | **Accepted** |
| Decision | All `go test` and `go` commands in `services/*/` and `protos/tests/` require `GOWORK=off` |
| Why | `go.work` references modules that don't exist in early milestones; workspace causes resolution failures |
| Code reflects? | ✅ Yes — CI enforces this; AGENTS.md and CLAUDE.md document it |

### ADR-018 — AI KB authorization model
| Field | Value |
|---|---|
| Date | 2026-04-24 |
| Status | **Accepted** |
| Decision | Context files are Tier 1 (public-safe only); sensitive context in `canvas.private.md` (gitignored) |
| Code reflects? | ✅ Yes — enforced by ADR-019 SPDD process |

### ADR-019 — SPDD prompt governance
| Field | Value |
|---|---|
| Date | 2026-04-30 |
| Status | **Accepted** |
| Decision | Every `feat:` PR requires a REASONS Canvas committed before any implementation; CI validates canvas before code |
| Code reflects? | ✅ Yes — `validate-canvas` CI gate; 27 canvas directories in `docs/spdd/` |
| Scope | `feat:` PRs only — `fix:/refactor:/docs:/ci:/chore:` are exempt |

---

## Pending ADRs (decisions made in practice; not yet formalized)

### ADR-020 — Zero-trust intra-service security [PROPOSED]
| Field | Value |
|---|---|
| Linked issue | [#240](https://github.com/zynax-io/zynax/issues/240) + [#488](https://github.com/zynax-io/zynax/issues/488) |
| Decision (proposed) | All inter-service gRPC uses TLS-by-default; `ZYNAX_DEV_INSECURE=1` gates plain-text in dev |
| Current state | ❌ All services use `insecure.NewCredentials()` — no TLS anywhere |
| Priority | High (review §7) |

### ADR-021 — Horizontal scale + multi-tenancy [PROPOSED]
| Field | Value |
|---|---|
| Linked issue | [#578](https://github.com/zynax-io/zynax/issues/578) |
| Decision (proposed) | Document operating model as "one Zynax cluster per Kubernetes namespace per tenant" until v1.x |
| Current state | No multi-tenancy; `namespace` field cosmetic |
| Priority | High (review §8) |

---

## Significant Scoped Decisions (docs/decisions/)

| File | Decision | Still valid? |
|---|---|---|
| `001-engine-adapter-test-split.md` | Engine-adapter tests split from main CI for parallel execution | ✅ Yes |
| `002-pr-size-900-limit.md` | PR size limit 900 LOC (with exclusions); above 400 requires justification | ✅ Yes |
| `003-bufconn-for-contract-tests.md` | All BDD contract tests use in-memory bufconn (no network ports) | ✅ Yes |
| `004-gowork-off-isolation.md` | cmd/zynax and cmd/zynax-ci are not in go.work; standalone modules | ✅ Yes |

---

## Architectural Directions from Review (not yet ADRs)

### §17.1 — Single-engine via env var (current) vs multi-engine per-workflow
**Current:** `ZYNAX_ENGINE` env var selects one engine globally.  
**Review recommendation:** Keep single-engine through v1.0; add `engine_hint` per-workflow in v1.x.  
**Status:** Still current; no action needed until second engine ships.

### §17.2 — IR transport: bytes+structured vs pure structured
**Current:** `ir_payload` bytes (field 4) kept alongside structured fields (M2).  
**Review recommendation:** Remove `ir_payload` field 4 by v1.0.  
**Status:** Proposed; tracked as part of ADR-012 update.

### §17.3 — Agent dispatch: push (planned) vs pull vs event-driven
**Current:** Engine-adapter → task-broker → agent push model planned.  
**Review recommendation:** Push for v0.x; pull mode for LLM/Bedrock adapters in alternate mode.  
**Status:** Proposed; warrants a new ADR when second dispatch mode is designed.

### §17.4 — Multi-tenancy model
**Review recommendation:** "one Zynax cluster per Kubernetes namespace per tenant" until v1.x.  
**Status:** To be formalized in ADR-021.

---

## Rejected / Not-merged Directions

These are preserved from closed-unmerged PRs and issue threads to avoid re-litigating:

| Direction | Why rejected | Evidence |
|---|---|---|
| CNCF Sandbox Candidate badge in README | Premature — no adopters, no community | PR #472 removed it |
| Phantom CHANGELOG entries (Helm charts, Argo engine, working SDK) | Inflated claims removed in truth-pass | PR #473 removed them |
| Merge queue + `strict: false` | Merge queue caused complexity; reverted to `allow_auto_merge + strict: true` | Issue #544, #589 |
| Python SDK as separate published package | Not yet ready; minimal Agent base class approach chosen | ADR-013 + #474 Option A decision |

---

*All ADR source files are in `docs/adr/ADR-NNN-*.md`. Decisions in this ledger that trace
to code should be re-verified against HEAD before acting on them.*
