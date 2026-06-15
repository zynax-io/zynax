# REASONS Canvas — EPIC T: Reusable Templates + First Real Workflows

> Tier 1 (public-safe). Tier 2 → `canvas.private.md`. Run `/spdd-security-review` before committing.

**Issue:** #1171 · **Milestone:** M7 (v0.6.0)
**Author:** M7 program plan · **Date:** 2026-06-15 · **Status:** Draft

---

## R — Requirements
- **Problem:** there are validation example manifests but no **reusable templates** (workflow/task/
  expert) and no **production-quality real workflows** that run end-to-end with data-flow.
- **Done when:** three real workflows (`code-review`, `ci-pipeline`, `feature-implementation`) apply
  and run green locally with data-flow + traces; `zynax init` scaffolds from templates; templates versioned.

## E — Entities
```
WorkflowTemplate / TaskTemplate / ExpertTemplate  ← parameterized, versioned scaffolds
version: field                                      ← workflow versioning
zynax init workflow|expert                          ← scaffolder
Real workflows: code-review · ci-pipeline · feature-implementation
```

## A — Approach
**We will:** add a template mechanism (parameterized + `version:`); surface validation/versioning in
the CLI; ship three runnable real workflows built on EPIC W data-flow + EPIC X experts; add
`zynax init` scaffolding.
**We will NOT:** ship the full example catalog (K8s/Helm/GitOps/security/etc.) — **deferred to M-dx**;
no replay/visualization/debugging yet (M-dx).
**Governing ADRs:** ADR-011 (declarative YAML), ADR-014 (state machine), ADR-029 (data-flow).

## S — Structure (first S)
```
spec/templates/{workflow,task,expert}/   ← reusable templates (versioned)
spec/workflows/examples/                   ← code-review.yaml · ci-pipeline.yaml · feature-implementation.yaml (runnable)
cmd/zynax/                                  ← `zynax init`, `zynax validate`, `version:` surfacing
docs/authoring/                             ← template + authoring guide
```

## O — Operations (stories — `spdd-story` form)
**T.1 — Template mechanism (workflow/task/expert) + versioning** · M · `feat`
- As a `workflow author`, I want reusable, versioned templates so I don't author from scratch.
- AC: [ ] template format + `version:` field; [ ] schema validation; [ ] `.feature`/contract test. Deps: W.3.

**T.2 — CLI validate + versioning surfacing** · S · `feat`
- As a `developer`, I want `zynax validate` and visible workflow versions so I catch errors pre-apply.
- AC: [ ] `zynax validate <file>` reports schema/data-flow errors; [ ] `version:` shown in status. Deps: T.1.

**T.3 — Three real, runnable workflows** · M · `feat`
- As a `developer`, I want production-quality workflows that actually run so Zynax is usable.
- AC: [ ] `code-review`, `ci-pipeline`, `feature-implementation` apply + run to terminal with data-flow + traces. Deps: W.5, X.3.

**T.4 — `zynax init workflow|expert`** · S · `feat`
- As a `developer`, I want scaffolding so I can start a new workflow/expert from a template.
- AC: [ ] `zynax init workflow|expert` emits a valid, versioned starting manifest. Deps: T.1.

**Order:** T.1 → {T.2, T.4} → T.3.

## N — Norms
- Manifests validate against `spec/schemas`; `make validate-spec` gate.
- `Signed-off-by:` + `Assisted-by:`; one logical change per commit.

## S — Safeguards (second S)
### Context Security
- [ ] No Tier 2 content (templates use placeholder repos/values); [ ] no PII; [ ] no prompt-injection; [ ] `/spdd-security-review` — PENDING

### Feature Safeguards
- Never ship a real workflow that depends on unimplemented features — must run on M7 capabilities.
- Never bake secrets into templates — parameterize via inputs/secret-refs.
- Never break manifest schema backward-compatibility — `version:` gates evolution.
