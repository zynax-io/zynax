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

Goal: Production-ready code quality across all Go services and Python SDK, plus
security supply-chain gates (SBOM, SLSA). Sets the foundation for M6 Helm deployment.

- [ ] refactor(workflow-compiler): naming, zero-value structs, functional options audit (#222)
- [ ] refactor: context.Context propagation into domain layer across all Go services (#223)
- [ ] refactor: error chain consistency — %w wrapping, typed sentinels, errors.Is/As (#224)
- [ ] docs(agents): Google-style docstrings on all public symbols in agents/sdk (#228)
- [ ] refactor(agents): strip explanatory comments — self-documenting names (#229)
- [ ] docs: architecture fitness functions — document all CI gates (#232)
- [ ] ci: SBOM generation with syft — publish as release artifact (#235)
- [ ] ci: SLSA provenance — sign released artifacts with sigstore/cosign (#239)
- [ ] docs: AI-output review checklist (#248)
- [ ] ci: publish tools image to public GHCR registry securely (#358)

---

## Active PRs

None.

---

## Known Blockers

None.

---

## Recently Closed

- M1–M4 GitHub milestones closed (all issues complete).
- M4 delivered: api-gateway REST, `zynax` CLI, Docker Compose runner, GitOps watch.
  Step issues #315–#320 all merged. Canvas: `docs/spdd/314-yaml-system-cli/canvas.md`.
