<!-- SPDX-License-Identifier: Apache-2.0 -->
# ADR-043: Thin `Workflow` CRD front-end — GitOps authoring, zero run-state in etcd

**Status:** Accepted  **Date:** 2026-07-03
**Related:** ADR-040 (**implements** — §3 mandates a `Workflow` CRD *only* as a declarative/GitOps authoring surface whose controller reconciles through the existing compile→submit path; §1 new infra must be a K8s primitive; §5 admission/conversion webhooks deferred until `v1beta1`), ADR-039 (CRD precedent this mirrors — controller-runtime manager, namespaced cache, Lease leader election, least-privilege RBAC; the `Agent` CRD — note ADR-039 sited its controller in an *internal* service (agent-registry), **not** the edge), ADR-011 (`Workflow` is an existing manifest kind compiled by workflow-compiler and never imported by code — the CR reconciles *through* the compiler; breaking shape bumps `apiVersion`), ADR-034 (**Status: Proposed** — not yet Accepted-binding — with a **name collision** to flag: ADR-034 binds the name `ManifestWorkflowID` to the *compiler's random per-call* `workflow_id`, whereas the hash-derived dedup key this ADR leans on is api-gateway's *separate* value of the same name in `services/api-gateway/internal/domain/apply.go`; ADR-034's random-id contract is unchanged either way), ADR-020 (bearer-token auth guards the *external boundary* — this ADR embeds the controller in that edge service; see Consequences), ADR-041 (kind first-run resource floor "must stay light" — the embedded manager/informer cache adds to the gateway pod footprint; see Consequences), ADR-015 (pluggable engines — run-state stays in the engine), ADR-008 (no shared DB — etcd is never the run-state store), ADR-012 (additive proto evolution — this ADR needs no proto change). Implements EPIC #1573 (M8.E — the EPIC number and M8 sub-letter are staged in `docs/milestones/M8-planning.md` and remain **provisional** until M8 is formally opened); companion REASONS Canvas **to be created before implementation** (ADR-019 canvas-before-code): `docs/spdd/1573-workflow-crd-front-end/canvas.md`.

---

## Context

Workflows today are submitted **imperatively** over REST. `zynax apply` (`cmd/zynax/cmd/apply.go`)
POSTs a manifest to `/api/v1/apply`; the gateway (`services/api-gateway/internal/api/handler.go`)
routes on kind and calls `ApplyService.ApplyWorkflow`
(`services/api-gateway/internal/domain/apply.go`), which fans out
`compiler.CompileWorkflow` → `engine.SubmitWorkflow`. Run-state lives entirely in the **engine** —
Temporal's durable history or the Argo engine (`services/engine-adapter/internal/infrastructure/`) —
and the gateway is stateless: `GetWorkflowStatus`/`WatchWorkflow` query the engine live. **Nothing
persists workflow run-state in etcd.**

Two accepted decisions now require a declarative front-end. **ADR-040 §3** mandates a `Workflow`
CRD "*only as a declarative/GitOps authoring surface*: a controller reconciles it by calling the
existing compile→submit path … Zynax does **not** become the durable workflow store (etcd is not the
source of truth for run state)." **ADR-011** already names `Workflow` a first-class manifest kind
compiled by workflow-compiler. The imperative REST path cannot be `kubectl apply`-ed, diffed, or
Argo-CD-synced, so operators standardizing on GitOps have no declarative surface for the one kind
that most needs one.

The `Agent` CRD (ADR-039; its M8.C EPIC letter is staged in `docs/milestones/M8-planning.md` and
provisional until M8 opens) has just landed the precedent: a namespaced `zynax.io/v1alpha1`
CRD shipped in a chart `crds/` directory (`helm/zynax-agent-registry/crds/agents.zynax.io.yaml`),
a controller-runtime manager with a namespaced informer cache and Lease leader election
(`services/agent-registry/internal/infrastructure/crd/`), and a least-privilege namespaced Role
(`helm/zynax-agent-registry/templates/rbac.yaml`). A `Workflow` CRD can reuse that shape verbatim.

Several sharp edges must be settled *before* code, which is why ADR-040 §Neutral flagged this as a
future ADR and why EPIC #1573 makes this ADR its story 1:

- **Identity vs ADR-034, and the post-completion duplicate-run trap.** A level-triggered reconcile
  re-invokes compile→submit on every requeue — periodic resync, leader change, **and pod restart**
  (the Agent precedent does a *full re-List* on restart:
  `services/agent-registry/internal/infrastructure/crd/informer.go` — "Restart recovery is a free
  resync — the cache Lists"). The compiler mints a *random* `workflow_id` per call (ADR-034, **Status:
  Proposed** — which, note, applies the name `ManifestWorkflowID` to that random compiler value), so a
  naive loop would spawn duplicate runs. api-gateway's *separate*, hash-derived dedup key of the same
  name (`apply.go`) does **not** by itself close this: `submit()` returns the existing run only while
  the prior run is `RUNNING`; on a `COMPLETED` prior run it deliberately mints a fresh
  timestamp-suffixed id (`rerunWorkflowID`) and starts a **new** run. So every *post-completion*
  reconcile of an unchanged CR would spawn a duplicate. Reconcile idempotency across the completed
  window must therefore come from a `generation`/`observedGeneration` gate (Decision §4), not from the
  id derivation, and **not** from changing ADR-034.
- **`apiVersion` mismatch.** The manifest schema pins `zynax.io/v1`
  (`spec/schemas/workflow.schema.json`); the CRD, following the Agent precedent, serves
  `v1alpha1`. The mapping must be explicit.
- **Two-CRD collision.** The Argo engine already materializes an `argoproj.io` `Workflow` CRD at its
  execution boundary. The authoring CRD must not overlap it.
- **Webhooks.** ADR-040 §5 defers admission/conversion webhooks until a `v1beta1` exists.

---

## Decision

**We will introduce a thin `Workflow` Custom Resource (`zynax.io/v1alpha1`) as a GitOps authoring
front-end whose controller reconciles by calling the existing `ApplyWorkflow` compile→submit path —
and nothing more. Execution and run-state stay in the engine; the CR carries only a thin status
mirror. etcd never holds run-state.**

1. **Thin `Workflow` CRD as an authoring surface.** A namespaced CRD in group `zynax.io`, version
   `v1alpha1`, shipped in a chart `crds/` directory (mirroring
   `helm/zynax-agent-registry/crds/agents.zynax.io.yaml`), with a `status:{}` subresource and an
   OpenAPI v3 structural schema **ported from `spec/schemas/workflow.schema.json`**. `spec` carries
   exactly today's `Workflow` manifest body — it introduces **no new authoring semantics, no new
   proto, and no new gRPC RPC**.

2. **Reconcile *through* the existing compile→submit path — never re-implement it.** A
   controller-runtime manager embedded in **api-gateway** (new
   `services/api-gateway/internal/infrastructure/crd/`, mirroring the agent-registry manager) calls
   `ApplyService.ApplyWorkflow` directly. api-gateway is the only service holding both the compiler
   and engine gRPC clients plus the fan-out orchestration, so the reconciler produces **byte-identical
   IR to the REST path**. The controller does not compile, submit, interpret, or store execution — it
   is a translator from `kubectl apply` to the same `ApplyWorkflow` call REST already makes.

3. **Zero run-state in etcd — status is a thin mirror only.** Following the `Agent` status shape,
   the CR `status` holds only low-churn, reconciler-owned fields: `conditions[]` (`Compiled`,
   `Submitted`), `observedGeneration`, and the last-dispatched `workflowID`/`runID` **reference** so
   `kubectl get workflow` shows *what was dispatched*. These turn over **only on a real `spec` change**
   (a `generation` bump — §4's gate); an unchanged CR reconciled repeatedly writes no new
   `workflowID`/`runID`, so etcd is not churned. It **never** holds phase, progress, step
   outputs, or results — those are fetched live from the engine exactly as today (`zynax status` /
   `GetWorkflowStatus`). Run-state stays engine-agnostic IR in Temporal/Argo (ADR-012/015); etcd is
   not the workflow store (ADR-008/040 §3).

4. **Idempotency from an `observedGeneration` gate — the hash id alone is insufficient; ADR-034
   untouched.** api-gateway's hash-derived dedup key (`ManifestWorkflowID` in `apply.go`) dedups
   **only within the concurrent, still-`RUNNING` window**: `submit()` returns the existing run when the
   prior run is `RUNNING`, but on a `COMPLETED` prior run it deliberately mints a fresh
   timestamp-suffixed id (`rerunWorkflowID`) and starts a new run — a *re-run* is the intended REST
   behaviour. A level-triggered reconciler requeues an unchanged CR on every periodic resync, leader
   change, and pod restart, so leaning on the hash id alone would spawn a **duplicate run on every
   post-completion reconcile** (and churn `status.workflowID`/`runID` each time). **The reconcile loop
   therefore gates re-submission on generation: it calls `ApplyWorkflow` only when
   `metadata.generation != status.observedGeneration` (a real `spec` change), and is a no-op when the
   spec is unchanged and already `Submitted`** — independent of resync, leader change, or restart. Put
   plainly: the hash key covers the concurrent/still-`RUNNING` window, and `observedGeneration` covers
   the post-completion window. The compiler's *random* per-call `workflow_id` (ADR-034 — Status:
   Proposed) is **not** changed. The CR's DNS-1123 `metadata.name` is a third, human-facing identity
   that maps *onto* the manifest name, not onto either workflow id. A BDD scenario (see Consequences)
   must assert that reconciling an already-`COMPLETED`, spec-unchanged CR — **including after a
   controller restart** — starts no new run.

5. **`v1alpha1` served; explicit mapping to the pinned schema.** The CRD serves `zynax.io/v1alpha1`
   (Agent precedent); its OpenAPI schema is generated from `spec/schemas/workflow.schema.json` and the
   reconciler hands the compiler the same `Workflow` manifest body it accepts today. A breaking shape
   change bumps `apiVersion` per ADR-011; per ADR-040 §5 there is **no admission/conversion webhook**
   at launch — defaulting/validation beyond the OpenAPI structural schema waits for `v1beta1`.

6. **No collision with the Argo engine CRD.** The authoring CRD lives in group **`zynax.io`**; the
   Argo engine's execution CRD lives in group **`argoproj.io`**. They share only the noun `Workflow`,
   never a group/version, and operate at different layers (authoring vs execution). The controller
   runs **in-cluster only** (namespaced cache/RBAC), matching the Agent manager's gating.

7. **`zynax apply kind: Workflow` targets the CRD path on K8s; REST is retained.** On a Kubernetes
   runtime, `zynax apply` for `kind: Workflow` bridges to CR-apply (the GitOps front door). The REST
   path (`/api/v1/apply`, `ApplyService.ApplyWorkflow`) is **retained through M8** and is not
   deprecated by this ADR: it remains (a) the reconciler's own internal call target and (b) the direct
   submission path for non-GitOps callers and CI. Its longer-term end-state (whether CR-apply becomes
   the sole operator-facing surface) is left to a successor ADR once the CRD path has soaked — it is
   **not** decided here.

---

## Rationale

| Option | Assessment |
|--------|------------|
| **Thin `Workflow` CRD front-end; embedded controller reconciles via `ApplyWorkflow`; status = thin mirror; zero run-state in etcd** | ✅ **Chosen** — satisfies ADR-040 §3's explicit mandate; delivers GitOps-native workflow authoring (diffable, `kubectl`-able, Argo-syncable) with **no new authoring semantics and no new store**; reuses the compile→submit path so CR-apply yields byte-identical IR to REST across Temporal *and* Argo (portability wedge preserved); reuses the M8.C `Agent` CRD precedent (controller-runtime, namespaced cache, Lease election, least-privilege RBAC) at low marginal cost. |
| REST-only status quo (no CRD) | ✗ Rejected — leaves the one first-class manifest kind (ADR-011) with no declarative surface; workflows cannot be GitOps-synced; directly contradicts the ADR-040 §3 mandate. |
| Full execution-owning `Workflow` operator (run-state / phase / results in CR `status`, etcd as the store) | ✗ Rejected — couples the IR to Kubernetes and undercuts cross-engine portability; makes etcd the durable workflow store (ADR-040 §3, ADR-008); duplicates the durability Temporal/Argo already own; high-frequency status writes churn etcd. This is the anti-pattern ADR-040 names. |
| Standalone `workflow-controller` binary (separate module) | ✗ Deferred — would re-wire the compiler+engine clients and the fan-out that only api-gateway holds, duplicating `ApplyWorkflow` and risking IR drift from the REST path. Revisit only if controller/gateway lifecycle separation becomes a hard requirement. |
| Admission-webhook defaulting/validation at launch | ✗ Deferred — ADR-040 §5 defers webhooks until `v1beta1`; the OpenAPI structural schema (ported from `spec/schemas/workflow.schema.json`) is the launch validation surface. |
| Manifest-derived deterministic `workflow_id` (amend ADR-034) to get idempotency | ✗ Rejected — unnecessary *and* insufficient. Reconcile idempotency comes from the `generation == observedGeneration` gate (Decision §4), not from the id derivation: api-gateway's hash-derived dedup key covers only the still-`RUNNING` window (on `COMPLETED` it re-runs), so amending ADR-034's random-id contract would neither fix post-completion duplicates nor be required once the generation gate is in place. |

---

## Consequences

- **Positive:**
  - Operators author and manage workflows as plain `zynax.io/v1alpha1` CRs — `kubectl apply`,
    `git`-diff, and Argo-CD sync all work; GitOps convergence needs no manual apply and, via the §4
    `observedGeneration` gate, re-sync of an unchanged CR starts **no duplicate run**.
  - CR-apply reconciles through `ApplyWorkflow`, so it produces **identical IR** to `zynax apply` and
    runs unchanged on both `temporal` and `argo` (engine-portability wedge intact).
  - **No new store, no new proto, no new RPC, no new authoring semantics** — run-state stays in the
    engine; ADR-008/012/015/040 §3 all upheld.
  - Reuses the M8.C `Agent` CRD scaffolding (controller-runtime, namespaced cache, Lease election,
    least-privilege RBAC), and strengthens the M8 CNCF-idiomatic control-plane story.

- **Negative / trade-off (load-bearing):**
  - **api-gateway gains the full `controller-runtime`/`client-go` dependency tree**, plus a `crds/`
    directory and namespaced RBAC (Role: get/list/watch on `workflows`, update/patch on
    `workflows/status`, leases) in `helm/zynax-api-gateway` (or a shared CRD chart — see follow-up).
    api-gateway currently has **no** k8s client dependency; this is real weight (and a `GOWORK=off`
    build surface, ADR-017) added to a previously k8s-client-free service.
  - **Security: the k8s-API credential now lives at the front door (blast-radius expansion vs
    ADR-020).** ADR-020 frames api-gateway as the guarded, internet-facing boundary (its in-process
    bearer check protects the external boundary — a check the co-accepted ADR-044 moves out to the
    Envoy edge + NetworkPolicy lockdown, so the "front door" remains guarded, just not in-process).
    Embedding the controller here grants that
    externally-exposed edge service a Kubernetes ServiceAccount with `get/list/watch` +
    `update/patch` on `workflows`/`workflows/status` **and** `leases` — so a gateway compromise now
    also yields cluster-API write credentials. This is a genuine widening relative to ADR-020's model,
    and it diverges from ADR-039, which deliberately kept its controller in an *internal* service
    (agent-registry). "Least-privilege namespaced RBAC" narrows the *scope* of that credential but does
    not change the fact that it now sits at the front door. **Mitigation (must ship with the code):** a
    dedicated ServiceAccount (not the gateway's request-serving identity), a `NetworkPolicy`
    restricting gateway→API-server egress, and consideration of a separate deployment/port for the
    manager; if these prove insufficient, re-weigh the deferred standalone-binary option (Rationale) on
    this security basis, not only on dependency weight.
  - **First-run footprint vs ADR-041.** ADR-041 made the kind first-run resource floor
    (Docker + kind + kubectl + Helm + ~4 CPU / 8 GB, slower cold-start) a load-bearing trade-off that
    "must stay light." The embedded `controller-runtime` manager + informer cache + `client-go` tree
    raise the api-gateway pod's memory/CPU on exactly that first-run path the kind quickstart depends
    on — a cost the dependency-tree bullet above frames only as build surface, not runtime weight.
    **Mitigation:** a bounded namespaced informer cache, a lazy/optional manager start when no
    `Workflow` CRD is installed (so non-CRD first-runs pay nothing), and explicit gateway pod resource
    limits — so the first-run wedge ADR-041 committed to is not eroded.
  - The controller runs **in-cluster only** (namespaced cache/RBAC). On any non-Kubernetes caller the
    CR path is inert and REST remains the only route — the two surfaces are not at parity off-cluster.
  - **Two `Workflow`-named CRDs coexist** (`zynax.io` authoring vs `argoproj.io` execution). They
    never share a group/version, but the shared noun can confuse operators reading `kubectl get
    workflows`; docs must disambiguate.
  - Because `status` is a deliberately **thin mirror**, `kubectl get workflow` **cannot** show live run
    progress or results — users must fall back to `zynax status`/the engine. This is an intentional
    limitation (the whole point of "zero run-state in etcd"), not a gap to be closed by widening
    `status`.

- **Neutral / follow-up required:**
  - `.feature` contract committed before implementation (ADR-016). Required BDD scenarios: (a) **assert
    run-state is absent from the CR**; (b) CR-apply IR equals `zynax apply` IR on both engines; and
    (c) reconciling an already-`COMPLETED`, spec-unchanged CR — **including after a controller
    restart / re-List** — starts **no new run** (the §4 `observedGeneration` gate).
  - Decide CRD chart placement: `helm/zynax-api-gateway/crds/` vs a shared CRD chart alongside
    `helm/zynax-agent-registry/crds/`.
  - Weigh embedding the controller in api-gateway (chosen) against a separate Go module — the deferred
    standalone-binary option — on **both** bases: the k8s dependency weight/pod footprint on the
    gateway (ADR-041) **and** the security blast-radius of siting a cluster-API credential in the
    external-boundary service (ADR-020). If the dedicated-SA + `NetworkPolicy` mitigations prove
    insufficient, the standalone binary is the fallback.
  - Map `spec/schemas/workflow.schema.json` → CRD OpenAPI v3 and pin the explicit `v1alpha1`↔`v1`
    apiVersion mapping; `v1beta1` + webhooks (ADR-040 §5) remain out of scope.
  - Runtime smoke before claiming done: `kubectl apply` a `Workflow` CR on a kind cluster, confirm it
    reconciles to a dispatched run and that GitOps re-sync is idempotent (no duplicate runs) — including
    **after the run has `COMPLETED`** and **after a controller pod restart** (the windows where the hash
    id alone would re-run) — twice, on both `temporal` and `argo`. CI-green alone is insufficient.
  - The forthcoming M8.G admission-policy ADR (ADR-045, EPIC #1575) depends on this CR path;
    the REST end-state is left to a successor ADR once the CRD path has soaked.
