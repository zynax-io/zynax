# REASONS Canvas — agent-registry push-path hard-removal (M9.A)

> **All content in this Canvas is Tier 1 (public-safe).**
> Run `/lib:spdd-security-review <path>` before committing.

**Issue:** #1674
**Author:** Oscar Gómez Manresa
**Date:** 2026-07-08
**Status:** Aligned

> Story issues: step 1 → #1697 · step 2 → #1698 · step 3 → #1598 · step 4 → #1699

---

## R — Requirements

- **Problem.** ADR-039 (Accepted 2026-06-22) carries an explicit removal clause: push
  registration was deprecated in M8 — the `AgentRegistryService` RPCs return `UNIMPLEMENTED`
  since M8.C step 7 (#1584) and the Agent CRD → informer → `SelectAgent` path is the sole
  production path — with **hard removal scheduled for M9**. The dead surface still ships:
  api-gateway carries the `kind: AgentDef` push client (SA1019-marked), agent-registry carries
  the memory + Postgres `AgentRepository` adapters and their chart/umbrella DB wiring, and
  `agent_registry.proto` still declares the deprecated RPCs.
- **Done — the push path is physically gone, in caller→implementation→contract order:**
  - `zynax apply` of a `kind: AgentDef` manifest returns the documented retirement error
    naming the Agent CRD replacement; no `RegisterAgent`/`DeregisterAgent` client references
    remain in `services/api-gateway/` or `cmd/zynax/` (#1697).
  - agent-registry runs stateless: no repository adapters, no push handler shims, no database
    configuration in `helm/zynax-agent-registry` or the umbrella; restart recovery is a pure
    informer resync, verified live (#1698).
  - The deprecated RPCs are deleted from `agent_registry.proto` with regenerated stubs and a
    documented-intentional `buf breaking` exception; the `AgentDef`/`CapabilityDef` messages
    reused by `scheduler.proto` are untouched (#1598).
  - Docs/status surfaces reconciled and `spike/adr-039-crd-scheduler-proof` retired (#1699);
    epic #1674 boxes ticked; this canvas advanced to Implemented.
  - e2e matrix green on BOTH engine legs after every step (registration only via Agent CRD).

## E — Entities

- **Agent CRD** (`zynax.io/v1alpha1`) — the single source of truth for agent identity +
  capabilities (ADR-039 §1); unaffected by this epic, becomes the only registration surface.
- **AgentRegistryService (deprecated RPCs)** — `RegisterAgent`/`DeregisterAgent` in
  `protos/zynax/v1/agent_registry.proto`; `UNIMPLEMENTED` since #1584; deleted in step 3.
- **AgentRepository adapters** — `memory_repo.go` + `postgres/repository.go` behind the
  hexagonal port; production-dead since M8.C; deleted with the DB wiring in step 2.
- **AgentDef push path (api-gateway)** — manifest parse → `RegisterAgent` client forward +
  the CLI apply surface for `kind: AgentDef`; replaced by a retirement error in step 1.
- **Retirement error** — the documented, user-facing rejection text pointing to the Agent CRD
  migration guide (`docs/patterns/agent-crd-migration.md`).
- **Spike branch** — `spike/adr-039-crd-scheduler-proof`, the ADR's de-risk artifact; kept by
  standing instruction until this removal build lands; retired in step 4.

```
caller removal          implementation removal        contract removal        truth pass
#1697 api-gateway  -->  #1698 agent-registry     -->  #1598 proto+stubs  -->  #1699 docs+spike
(AgentDef → error)      (repos+DB+chart gone)         (buf breaking doc'd)    (epic closes)
```

## A — Approach

- **WILL:** delete in strict caller → implementation → contract order so every intermediate
  state compiles and deploys (the ADR-046 zero-caller-references discipline applied to
  ADR-039's clause); keep `AgentDef`/`CapabilityDef` proto messages (reused by
  `scheduler.proto`); ship a retirement error that names the replacement; verify the
  stateless-scheduler claim at runtime (restart → informer resync) before calling step 2 done;
  reconcile docs/status surfaces and retire the spike branch as the epic's closing step.
- **WON'T:** no changes to the scheduler's `SelectAgent` path, scoring pipeline, or CRD
  schema (delivered in M8.C, out of scope); no adapter-side changes expected — M8.C step 7
  removed adapter self-registration; step 1 re-verifies with a grep and, if any live adapter
  reference surfaces, extends scope explicitly via `/lib:spdd-prompt-update` rather than
  ad-hoc patching; no Compose-path work (runtime is kind-native per ADR-041); no new ADR —
  this epic *executes* an accepted removal clause.
- **Positioning fit:** internal removal; the only user-facing copy is the retirement error,
  which must point to the Agent CRD path (GitOps-native registration — the K8s-idiom story
  that backs the portability wedge's credibility).
- Governing ADRs: ADR-039 (removal clause — this epic), ADR-021 (amended half completes),
  ADR-028 (context-slice contract preserved verbatim), ADR-016 (BDD suites updated in the
  same PR as each surface they cover), ADR-019 (refactor/chore epic — canvas by maintainer
  request for traceability).

## S — Structure

- `services/api-gateway/internal/` — AgentDef parse/forward path + retirement error (step 1).
- `cmd/zynax/` — CLI apply surface for `kind: AgentDef` (step 1).
- `services/agent-registry/internal/` — repository adapters, push handler shims, their tests
  (step 2).
- `helm/zynax-agent-registry/` + `helm/zynax-umbrella/` — DB values/wiring removal (step 2).
- `protos/zynax/v1/agent_registry.proto` + generated Go/Python stubs (step 3).
- `protos/tests/agent-registry/features/` — push-registration scenarios retired alongside the
  surface they test; scheduler scenarios untouched (steps 1–3, per ADR-016).
- `docs/patterns/agent-crd-migration.md`, README service table,
  `state/current-milestone.md`, `docs/milestones/M9-planning.md`, this canvas (step 4).
- Config env prefix: `ZYNAX_AGENT_REGISTRY_` (DB-related vars removed in step 2).

## O — Operations

> Each step = one PR. Order is load-bearing (caller → implementation → contract → docs):
> #1697 → #1698 → #1598 → #1699.

1. **api-gateway AgentDef push path deleted** (#1697, `chore:`) — remove the `RegisterAgent`
   client forward (SA1019 marker), handler routing, and CLI surface; add the documented
   retirement error; grep gate: zero push-client references outside generated stubs; BDD
   scenarios updated; e2e green both legs.
2. **agent-registry stateless closeout** (#1698, `chore:`) — delete memory/Postgres
   `AgentRepository` adapters + push handler shims + their tests; drop chart/umbrella DB
   wiring; domain coverage ≥ 90% holds on remaining code; runtime smoke ×2 on the same kind
   cluster: pod delete → Ready via informer resync, no Postgres connection attempted.
3. **proto hard-removal** (#1598, `chore:`) — delete the deprecated RPCs from
   `agent_registry.proto`; regenerate Go+Python stubs; documented-intentional `buf breaking`
   exception scoped to these RPCs; `AgentDef`/`CapabilityDef` messages stay (scheduler.proto
   reuse); release-notes line.
4. **migration sweep + spike retirement** (#1699, `docs:`) — agent-crd-migration.md to
   post-removal tense; repo-wide push-path reference sweep (historical mentions only remain);
   status surfaces in the same diff; delete `spike/adr-039-crd-scheduler-proof` with a
   traceable PR-body note; tick epic boxes; canvas → Implemented.

## N — Norms

- Commit hygiene: DCO `Signed-off-by` + `Assisted-by: Claude/<model>` (never `Co-Authored-By`
  for AI); SSH-signed; one PR per story; subjects ≤ 72 chars; PR bodies from
  docs/contributing/pr-templates.md with runtime evidence.
- `GOWORK=off` for all `go` commands in service dirs (ADR-017); stubs via
  `make generate-protos` only (output committed); images/banner regions never hand-edited.
- Runtime evidence rule: steps 1–2 touch `services/*` — boot the documented kind path and run
  stateful flows twice on the same cluster before claiming done; CI-green is not runtime
  evidence.
- Milestone gate: merge only while M9 is the active milestone; do not merge any step before
  the v0.7.0 release (which ships the deprecation) is published.
- Requirements change → `/lib:spdd-prompt-update` first (Status back to Draft); never patch
  code ahead of the canvas.

## S — Safeguards (second S)

### Context Security (mandatory before committing this Canvas)
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no email addresses
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /lib:spdd-security-review passed on this file

### Feature Safeguards
- Never remove the proto RPCs (#1598) while any client reference exists — steps 1–2 are the
  grep-gated prerequisite; zero-caller-references is the merge gate.
- Never touch `AgentDef`/`CapabilityDef` messages — `scheduler.proto` reuses them (ADR-039
  §Consequences); deleting them is a second, unsanctioned breaking change.
- Never let the scheduler acquire persistence on the way out — statelessness (informer resync
  on restart) is the ADR-039 end state and must be verified live in step 2.
- Never delete the spike branch before steps 1–3 are merged — it is the ADR's verification
  artifact until the removal build exists (standing instruction).
- Never weaken the e2e gate: both engine legs must pass after every step, not only at epic
  end.
- Never hardcode engine or milestone names in surviving code/docs paths (ADR-015 / milestone
  tooling contract).
