# Current Milestone State

> This file tracks the active execution state. Update it when milestones close,
> blockers change, or active work shifts. Do NOT use this file for architecture
> decisions — those belong in `docs/adr/`. Do NOT accumulate history here.

---

## Status Summary

| Milestone | Status | Version |
|-----------|--------|---------|
| M1 — Contracts Foundation | ✅ Complete | v0.1.0 |
| M2 — Workflow IR | ✅ Complete | v0.1.0 |
| M3 — Temporal Execution | ✅ Complete | v0.2.0 |
| M4 — YAML System + CLI | ✅ Complete | v0.3.0 |
| **M5 — Adapter Library** | **In Progress** | v0.4.0 |

---

## M5 — Progress

Goal: Deliver the Adapter Library (#377) — five production-ready adapter services
(http, git, ci, llm, langgraph) that turn any external system into a Zynax capability.
Accompanied by code quality, security supply-chain gates, and documentation that bring
the whole platform to production readiness. Sets the foundation for M6 Helm deployment.

Epic: [#377 — M5 Adapter Library](https://github.com/zynax-io/zynax/issues/377)
Canvas: `docs/spdd/377-adapter-library/canvas.md` (Status: Draft — PR #385)

### Track 1 — Adapter Library (the main M5 deliverable)

Each adapter requires a REASONS Canvas + BDD feature file before implementation.

- [ ] feat(adapters/http): REST capability proxy adapter (#380, step 1) — Go
- [ ] feat(adapters/git): GitHub/GitLab operations adapter (#381, step 2) — Go
- [ ] feat(adapters/ci): CI pipeline trigger adapter (#382, step 3) — Go
- [ ] feat(adapters/llm): LLM provider capability adapter (#383, step 4) — Python
- [ ] feat(adapters/langgraph): LangGraph capability adapter (#384, step 5) — Python

### Track 2 — Go Code Quality

- [ ] refactor(workflow-compiler): naming, zero-value structs, functional options (#222)
- [ ] refactor: context.Context propagation — parent (#223)
  - [ ] refactor(workflow-compiler): thread ctx from gRPC boundary (#373, step 1)
  - [ ] docs(services): ctx-first mandate + Temporal exemption in AGENTS.md (#374, step 2)
- [ ] refactor: error chain consistency — %w, typed sentinels, errors.Is/As (#224)

### Track 2 — Python SDK Quality

- [ ] ci(agents): enable ruff Google-style docstring enforcement (#375) ← unblocked
- [ ] feat(agents): SDK core modules — runtime, context, capability, server, platform, observability ← needs new issue
- [ ] docs(agents): Google-style docstrings on all public SDK symbols (#228 / #376) ← blocked on SDK impl
- [ ] refactor(agents): strip explanatory comments (#229) ← blocked on #376

### Track 2 — Supply-Chain Security

- [ ] ci: consolidate tools image publish to single workflow (#358) ← do first
- [ ] ci: SBOM generation with syft (#235) ← after #358
- [ ] ci: SLSA provenance — sign artifacts with cosign (#239) ← after #235

### Track 2 — Architecture & Docs

- [ ] docs: architecture fitness functions — document all CI gates (#232)
- [ ] docs: AI-output review checklist (#248)

---

## Active PRs

| PR | Title | Status |
|----|-------|--------|
| [#385](https://github.com/zynax-io/zynax/pull/385) | docs: REASONS Canvas for M5 Adapter Library (#377) | Open — pending merge |

---

## Known Blockers

- **#376 (docstrings) and #229 (strip comments)** are blocked until a `feat(agents):` issue is created and merged for the SDK core module implementation.
- **#239 (cosign)** depends on #358 (stable GHCR path) and #235 (SBOM).

---

## Recently Closed

- M1–M4 GitHub milestones closed (all issues complete).
- M4 delivered: api-gateway REST, `zynax` CLI, Docker Compose runner, GitOps watch.
  Step issues #315–#320 all merged. Canvas: `docs/spdd/314-yaml-system-cli/canvas.md`.
