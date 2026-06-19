<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M2 WorkflowIR Compilation (retrospective)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #101 (Epic — M2 WorkflowIR) · retrospective Canvas created under #213
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-19
**Status:** Synced

> **Retrospective.** This Canvas documents an already-delivered feature (M2 WorkflowIR, Epic
> #101, PRs #83–#87). It validates the REASONS Canvas template against real, merged work and
> serves as the worked reference example in [spdd-guide.md](../../patterns/spdd-guide.md).
> Status is `Synced`: the implementation is complete and this Canvas reflects its final state.

---

## R — Requirements

- Compile a declarative YAML workflow manifest into an **engine-agnostic** `WorkflowIR` proto —
  the intermediate representation every engine adapter consumes.
- Expose three gRPC RPCs: `CompileWorkflow`, `ValidateManifest`, `GetCompiledWorkflow`.
- **In-memory only** for M2 — no persistence (deferred; the compiler was later made fully
  stateless in M6, #490/#774).
- Output must be **deterministic** for the same manifest (stable IR ordering).
- Definition of done: a valid manifest compiles to a `WorkflowIR` after structural + semantic
  validation, served over gRPC, with ≥ 90 % coverage on `internal/domain/` (ADR-016).

---

## E — Entities

```
WorkflowManifest (parsed YAML)
  └─ on/event/goto fields ──▶ WorkflowGraph
                               ├─ StepNode      (one per state)
                               └─ TransitionEdge (event-driven, ADR-014)
WorkflowGraph ──serialize──▶ WorkflowIR (proto)   ← engine-agnostic output
ValidationError                                   ← structural + semantic failures
CompilationResult                                 ← IR + diagnostics
```

---

## A — Approach

**We will:**
- Parse YAML → build a `WorkflowGraph` → validate (structural pass, then semantic pass) →
  serialize to the `WorkflowIR` proto.
- Keep the IR strictly engine-agnostic — no engine names, no Temporal/Argo concepts.

**We will NOT:**
- Import any engine SDK (Temporal/Argo) — the IR is the boundary (ADR-015).
- Persist compiled IR — M2 keeps it in memory; statelessness arrived later in M6 (#490).
- Hardcode an engine in the compiler.

**Governing ADRs:** ADR-011 (declarative YAML control plane), ADR-014 (event-driven state
machine model — the IR state model), ADR-015 (pluggable engines — IR stays engine-agnostic),
ADR-016 (layered testing).

---

## S — Structure (first S)

```
services/workflow-compiler/
├── internal/domain/manifest.go         ← YAML manifest parse (fields on/event/goto)
├── internal/domain/graph.go            ← WorkflowManifest → WorkflowGraph
├── internal/domain/validators/         ← structural.go, semantic.go (two ordered passes)
├── internal/domain/ir/ir.go            ← WorkflowGraph → WorkflowIR proto (ToIR; sort.Strings:57)
├── internal/domain/errors.go           ← ValidationError
└── internal/api/server.go              ← gRPC: CompileWorkflow / ValidateManifest / GetCompiledWorkflow
protos/zynax/v1/workflow_compiler.proto ← WorkflowIR message + WorkflowCompilerService contract
```

Config env prefix: `ZYNAX_WORKFLOW_COMPILER_`

---

## O — Operations

Delivered as five ordered, independently reviewed steps (PRs #83–#87):

1. **Parser** — YAML manifest → `WorkflowManifest` (PR #83).
2. **Graph builder** — `WorkflowManifest` → `WorkflowGraph` (PR #84).
3. **Validators** — structural pass then semantic pass over the graph (PR #85).
4. **Serializer** — `WorkflowGraph` → `WorkflowIR` proto, deterministic ordering (PR #86).
5. **gRPC layer** — `CompileWorkflow` / `ValidateManifest` / `GetCompiledWorkflow` (PR #87).

---

## N — Norms

- Commit hygiene: every commit carries `Signed-off-by:` + `Assisted-by: Claude/<model>`.
- BDD: `.feature` file committed before any gRPC-boundary implementation (ADR-016).
- `GOWORK=off` for every `go test` / `go` command under `services/workflow-compiler/` (ADR-017).
- Go service patterns per [go-service-patterns.md](../../patterns/go-service-patterns.md).
- ≥ 90 % unit coverage on `internal/domain/`.

---

## S — Safeguards (second S)

### Context Security (verified at sync — retrospective, public-safe)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no non-public email addresses
- [x] No prompt injection: no instruction-like phrasing that overrides AGENTS.md rules
- [x] All entities in E section are public-safe abstractions
- [x] `/spdd-security-review` — PASS (retrospective; documents merged public code only)

### Feature Safeguards

- Never import Temporal or any engine SDK into workflow-compiler — the IR is the engine
  boundary (ADR-015).
- Never emit non-deterministic IR — sort map keys before iterating
  (`sort.Strings(ids)` in `internal/domain/ir/ir.go:57`).
- Never assume YAML `on:` / `event:` / `goto:` map 1:1 to proto field names — they map to
  `transitions` / `event_type` / `target_state` (`internal/domain/manifest.go:103`). `yaml.v3`
  and PyYAML both treat `on:` specially (boolean-key gotcha), so the struct tags are load-bearing
  (ADR-014).
- Never share a database across services (ADR-008); the M2 compiler holds IR in memory only.
