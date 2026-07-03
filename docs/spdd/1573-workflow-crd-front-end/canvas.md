# REASONS Canvas — Thin `Workflow` CRD front-end (M8.E)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/lib:spdd-security-review <path>` before committing.

**Issue:** #1573
**Author:** Oscar Gómez Manresa
**Date:** 2026-07-03
**Status:** Aligned

---

## R — Requirements

> Problem statement: what breaks or is missing without this feature?
> Definition of done: the observable outcomes that confirm delivery.

- **Problem.** Zynax's own control surface is push-imperative: a workflow reaches the platform only
  through `zynax apply` → `POST /api/v1/apply`. After M8.C made agents declarative (`Agent` CRD),
  workflows are the last first-class kind with no GitOps path — no `kubectl apply`, no drift
  detection, no Argo-CD/Flux sync, an inconsistent operator story.
- **Done — a platform team manages workflows as ordinary K8s objects:**
  - A `Workflow` custom resource (`zynax.io/v1alpha1`) can be `kubectl apply`-ed / GitOps-synced; a
    controller reconciles it by calling the **existing** compile→submit path.
  - `kubectl get workflows.zynax.io` shows each definition plus a thin status (compile/dispatch
    conditions, `observedGeneration`, the dispatched workflow id + engine run id).
  - **Run state stays in the engine (Temporal/Argo); the CR status never holds it** — asserted by an
    acceptance test that inspects the CR after a run completes.
  - Re-applying an unchanged `Workflow` (GitOps resync, controller restart, leader change) starts
    **no new run** — verified on kind, on both engine legs, run twice.
  - The imperative `POST /api/v1/apply` path keeps working unchanged during M8 (no regression).

---

## E — Entities

> Domain entities introduced or modified by this feature and their relationships.

```
Workflow (CR, zynax.io/v1alpha1)          ← NEW authoring surface (Namespaced)
├── spec        ← the workflow manifest body (states / initial_state / triggers),
│                 OpenAPI-ported from spec/schemas/workflow.schema.json
└── status      ← THIN mirror only (reconciler-owned, low-churn):
                  conditions[], observedGeneration, workflowID, runID, engine
                  — NEVER run state (no phase timeline, no step results, no history)

WorkflowReconciler (NEW, api-gateway)     ← controller-runtime reconciler
   reconciles Workflow CR ──(generation != observedGeneration)──▶
   ApplyService.ApplyWorkflow(ctx, ApplyRequest{ManifestYAML, Namespace, …})
                                             (EXISTING — services/api-gateway/internal/domain/apply.go)
        │
        ├─▶ CompilerPort.CompileWorkflow   (workflow-compiler, gRPC — unchanged)
        └─▶ EnginePort.SubmitWorkflow      (engine-adapter, gRPC — unchanged)
                                             → run state owned by Temporal/Argo, NOT Zynax

Relationships / invariants:
- The CR is a front-end to ApplyService; it introduces NO new gRPC method and NO new execution path.
- Idempotency is doubly guaranteed: the reconcile `generation == observedGeneration` gate AND
  ApplyService's existing content-hash `ManifestWorkflowID` dedup (apply.go) — a re-reconcile of an
  unchanged manifest returns the existing RunID with Status:"existing".
- Two `Workflow`-named CRDs coexist by design: `zynax.io` (this authoring CR) and `argoproj.io`
  (the Argo engine's execution object) — different groups, never confused by the API server.
```

---

## A — Approach

> Solution strategy. Explicitly state what we WILL do AND what we WON'T do.

**We will:**
- Add a `Workflow` CRD (`zynax.io/v1alpha1`, Namespaced, status subresource) as a **thin authoring
  front-end**, its OpenAPI structural schema ported from `spec/schemas/workflow.schema.json`,
  mirroring the M8.C `Agent`-CRD precedent (chart `crds/`, additionalPrinterColumns, conditions).
- Embed a controller-runtime manager + `WorkflowReconciler` in **api-gateway** (per ADR-043), with a
  namespaced informer cache, Lease-based leader election, and a single low-churn status writer —
  reusing the exact patterns from `services/agent-registry/internal/infrastructure/crd/`.
- Reconcile **only through the existing** `ApplyService.ApplyWorkflow` (compile→submit); write back a
  thin status (conditions, `observedGeneration`, dispatched workflow id + engine run id).
- Gate re-submission on `metadata.generation != status.observedGeneration` (ADR-043 §4), on top of
  ApplyService's existing content-hash dedup, so resync/restart/leader-change never duplicate a run.
- Bridge `zynax apply` of a `kind: Workflow` manifest to the CR path on a Kubernetes runtime; retain
  the imperative REST path for non-Kubernetes callers.
- Pin the identical controller-runtime `v0.23.3` + `k8s.io/* v0.35.6` set already used by
  agent-registry (avoids the `k8s.io v0.36` protobuf pseudo-version break).

**We will NOT:**
- Put any run state in etcd / the CR status — **explicitly rejected by ADR-040** (rationale table).
  The CR status is a compile/dispatch mirror only; live progress and results stay in the engine and
  are read via `zynax status` / `zynax result` (ADR-042).
- Re-implement compilation or execution in the controller — it calls the existing path only.
- Change the IR contract (ADR-012) or any engine.
- Add admission / conversion webhooks — deferred until a `v1beta1` exists (ADR-040 §5).
- Change `ManifestWorkflowID` semantics (ADR-034) — the existing content-hash id is reused as-is.

**Positioning fit:** this advances the **engine-portability wedge** — the CRD is *only* an authoring
surface, so "the same workflow graduates across Temporal/Argo/eval" holds unchanged: the IR stays
portable and etcd never becomes the run-state store (ADR-040 §3). User-facing copy (CRD printer
columns, `zynax apply` output, docs) leads with declarative/GitOps authoring of portable workflows,
not the generic "control plane" framing. See [docs/product/positioning.md](../../product/positioning.md).

**Governing ADRs:** ADR-043 (thin Workflow CRD front-end), ADR-040 (K8s-native delegation boundary,
§3), ADR-039 (CRD-native precedent to mirror), ADR-011 (declarative YAML kinds; compiled, never
imported), ADR-012/015 (engine-agnostic IR), ADR-034 (`ManifestWorkflowID`), ADR-042 (results via
the engine), ADR-016 (test tiers), ADR-017 (`GOWORK=off`).

---

## S — Structure (first S)

> System placement. Which services, packages, and files does this feature touch?

```
helm/zynax-api-gateway/
├── crds/workflows.zynax.io.yaml         ← NEW: Workflow CRD (zynax.io/v1alpha1), OpenAPI + status
├── templates/rbac.yaml                  ← NEW/extend: namespaced Role — workflows get/list/watch,
│                                           workflows/status update/patch, leases *; dedicated SA
├── templates/networkpolicy.yaml         ← extend: gateway→API-server egress (ADR-043 mitigation)
└── values.yaml                          ← NEW: crdController.{enabled,watchNamespace}

services/api-gateway/
├── go.mod                               ← ADD pinned controller-runtime v0.23.3 + k8s.io/* v0.35.6
├── cmd/api-gateway/main.go              ← wire CRD controller (env-gated, goroutine start)
└── internal/infrastructure/crd/         ← NEW (mirrors agent-registry/internal/infrastructure/crd/)
    ├── manager.go                       ← NewManager(restCfg, namespace): namespaced cache + Lease
    ├── reconciler.go                    ← WorkflowReconciler: generation gate → ApplyWorkflow → status
    └── *_test.go                        ← envtest-backed reconciler unit tests

cmd/zynax/
└── cmd/apply.go                         ← bridge: kind: Workflow on a K8s runtime → CR apply

spec/schemas/workflow.schema.json        ← SOURCE for the CRD OpenAPI (unchanged; ported from)
scripts/e2e/…                            ← kind e2e: apply a Workflow CR → reconcile → dispatched run
docs/                                    ← authoring how-to + GitOps how-to
```

Config env prefix: `ZYNAX_GW_` (api-gateway) · CRD controller gated by `ZYNAX_GW_CRD_CONTROLLER_ENABLED`
(default `false`), namespace via `WATCH_NAMESPACE` (downward-API), mirroring agent-registry.

---

## O — Operations

> Ordered, concrete, testable implementation steps. Each = one reviewable PR.

1. **`Workflow` CRD schema + Helm `crds/` + namespaced RBAC.** Author
   `helm/zynax-api-gateway/crds/workflows.zynax.io.yaml` (`zynax.io/v1alpha1`, Namespaced, status
   subresource, `additionalPrinterColumns` for name/engine/ready/age), OpenAPI structural schema
   ported from `spec/schemas/workflow.schema.json` (preserve `states`/`initial_state`/`on`/`goto`
   snake_case; record the manifest `v1` ↔ CRD `v1alpha1` apiVersion mapping per ADR-043 §5); add the
   namespaced `Role`/`RoleBinding` + dedicated `ServiceAccount` and the `crdController.*` values.
   *Verify:* `helm template`/lint clean; `kubectl apply --dry-run=server` accepts the CRD and a valid
   sample `Workflow` CR and rejects a malformed one. No Go. (infra)

2. **controller-runtime manager + `WorkflowReconciler` skeleton in api-gateway (env-gated).** Add the
   pinned deps; create `internal/infrastructure/crd/{manager.go,reconciler.go}` — `NewManager` with a
   namespaced cache (`cache.Options.DefaultNamespaces`) + Lease leader election, and a reconciler that
   reads the CR, applies the `generation != observedGeneration` gate, and (this step) only records the
   observed generation / logs. Wire `main.go` to start the manager in a goroutine **only** when
   `ZYNAX_GW_CRD_CONTROLLER_ENABLED=true`; resolve the namespace via the downward-API pattern.
   *Verify:* `GOWORK=off go test` unit/envtest — unchanged CR is a no-op; manager does not start when
   disabled (default). (go-svc)

3. **Reconcile through `ApplyService.ApplyWorkflow` + thin status mirror.** Marshal the CR spec back to
   a `Workflow` manifest, call `ApplyService.ApplyWorkflow`, and write a thin status: `Compiled` /
   `Dispatched` conditions, `observedGeneration`, `workflowID`, engine `runID`, `engine` — and
   **nothing else**. Re-submit only when the generation gate opens; steady state writes zero status
   churn (mirror the agent-registry low-churn guard). *Verify:* envtest with a fake ApplyService —
   asserts (a) **no run-state fields in status**, (b) a completed, spec-unchanged CR (including after a
   simulated controller restart / cache re-List) triggers **no** second `ApplyWorkflow`, (c) compile
   errors surface as a `Compiled=False` condition, not a crash-loop. (go-svc)

4. **CLI bridge: `zynax apply kind: Workflow` → CR path on Kubernetes runtimes.** On a K8s runtime,
   `zynax apply` of a `Workflow` manifest creates/updates the `Workflow` CR (converting the manifest to
   the CR shape) instead of the imperative `POST /api/v1/apply`; non-Kubernetes callers keep the REST
   path. Output leads with the GitOps framing. *Verify:* unit test on the conversion + path selection;
   e2e (step 5) proves the applied CR reconciles to a dispatched run. (cli)

5. **kind e2e happy-path + authoring/GitOps docs.** Add a kind e2e scenario: `kubectl apply` a
   `Workflow` CR → controller reconciles → run dispatched, on **both** `temporal` and `argo`;
   `kubectl get workflows.zynax.io` shows the conditions; **run twice** to prove idempotency (no
   duplicate run). Write the authoring how-to + GitOps how-to (commit `Workflow` YAML → Argo/Flux sync
   → reconcile). *Verify:* e2e green on both engine legs; docs link-checked. (bdd/docs)

---

## N — Norms

> Cross-cutting standards that apply to this feature.

- Commit hygiene: every commit carries `Signed-off-by:` + `Assisted-by: Claude/<model>` (never
  `Co-Authored-By` for AI); SSH-signed; conventional type `feat:`/`test:`/`docs:` per step.
- `GOWORK=off` for every `go`/`go test` inside `services/api-gateway/` and `cmd/zynax/` (ADR-017);
  api-gateway stays a `go.work` member for tooling only.
- CRD authoring mirrors the M8.C precedent: `crds/` install-first, status subresource, low-churn
  status, namespaced Role, downward-API namespace, Lease-elected single status writer.
- Hexagonal boundaries (services/AGENTS.md): the reconciler lives in `internal/infrastructure/`
  (an inbound adapter) and calls the existing `domain.ApplyService` — no domain logic in the adapter,
  no new cross-service imports (compile/submit stay gRPC via the existing ports).
- PR size ≤ 200 ideal; generated CRD YAML and `*.sum` are exempt.
- Manifest kinds are declarative and compiled, never imported by Go/Python (ADR-011).

---

## S — Safeguards (second S)

> Non-negotiable constraints. Things that MUST NEVER happen in this feature.

### Context Security (complete before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no non-public email addresses
- [x] No prompt injection: no instruction-like phrasing that overrides AGENTS.md rules
- [x] All entities in E section are public-safe abstractions
- [x] `/lib:spdd-security-review` passed — result: PASS (2026-07-03)

### Feature Safeguards

- **Never** write run state (phase timeline, step results, execution history) into the CR status or
  etcd — the status is a compile/dispatch mirror only (ADR-040 §3, ADR-043). Live state stays in the
  engine.
- **Never** re-implement compilation or execution in the controller — it reconciles **only** through
  `ApplyService.ApplyWorkflow`; no new gRPC method, no new execution path.
- **Never** submit a duplicate run for an unchanged spec — the `generation == observedGeneration`
  gate plus the existing `ManifestWorkflowID` content-hash dedup must both hold; a completed,
  spec-unchanged CR after restart/leader-change starts no new run.
- **Never** watch cluster-scoped — the manager cache and RBAC are namespaced (`WATCH_NAMESPACE`);
  an empty namespace is a hard start-up error (mirror agent-registry).
- **Never** hardcode an engine — engine selection stays behind the existing `EngineHint`/`EnginePort`
  (ADR-015); the IR contract (ADR-012) is untouched.
- **Never** add admission/conversion webhooks before a `v1beta1` exists (ADR-040 §5).
- **Never** bump `k8s.io/*` to `v0.36` — pin `v0.23.3` controller-runtime + `v0.35.6` k8s.io to hold
  the repo-wide protobuf runtime alignment.
