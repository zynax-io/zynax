# ADR-040: Kubernetes-native delegation boundary (thin-Zynax)

**Status:** Proposed  **Date:** 2026-06-22
**Related:** ADR-039 (the first concrete migration under this principle — agent-registry → `Agent` CRD), ADR-020 (mTLS via cert-manager — auth already delegated), ADR-030 (OpenTelemetry + Uptrace — metrics/tracing/logging already delegated), ADR-026 (Postgres distribution), ADR-012 (engine-agnostic Workflow IR — bounds the Workflow-CRD decision), ADR-015 (pluggable workflow engines), ADR-011 (declarative YAML / CRD control surface), ADR-016/017 (testing · `GOWORK=off`)

> **Sequencing:** This is a governing **principle** ADR. It records the delegation boundary and the
> custom core; it does not itself change code. The one open migration it points to (agent-registry →
> CRD) is ADR-039 (spike M7, build M8). The one new component it sanctions (a thin `Workflow` CRD
> front-end) gets its own ADR + Canvas when scheduled.

---

## Context

A recurring design question for Zynax is *where the line sits* between what the project builds and
what it delegates to Kubernetes and the cloud-native ecosystem — sharpened by the CNCF-Sandbox goal
(M8), where "don't re-implement what Kubernetes already gives you" is a maturity signal.

A repo audit (2026-06-22) against the candidate list of infrastructure responsibilities shows that
**Zynax is already thin**: most generic orchestration concerns are already delegated, and several
others were never built as Zynax components at all. The audit findings, with evidence:

- **Metrics** — already Prometheus `/metrics` on every service (`libs/zynaxobs/metrics.go`, M6).
- **Tracing / logging** — already OpenTelemetry → Uptrace and structured JSON to stdout
  (`libs/zynaxobs`, ADR-030); no Zynax log-aggregation service exists.
- **Auth** — already mTLS via cert-manager + ServiceAccounts/RBAC in Helm (ADR-020).
- **Config / secrets** — already env (`libs/zynaxconfig`) + Helm ConfigMaps/Secrets; no config service.
- **Service discovery** — already K8s DNS + `ZYNAX_*_ADDR` env; no discovery service.
- **Scaling / networking** — already HPA + NetworkPolicy per service (Helm, M6).
- **Leader election, feature flags, config service, tool/model/provider registries** — **do not exist**
  as Zynax components; there is nothing to remove.
- **Workflows** — **not persisted by Zynax**: `kind: Workflow` compiles on-the-fly to an
  engine-agnostic `WorkflowIR` (ADR-012) and is handed to an engine; the Argo engine already
  materializes an Argo Workflow CRD at the boundary (`services/engine-adapter/internal/infrastructure/argo_engine.go`).
- **Agent identity** — the one genuinely Zynax-owned infrastructure responsibility left: a push-based
  `agent-registry`. This is being migrated to a CRD by ADR-039.

So the "thin-Zynax migration" is mostly an **affirmation of an existing principle** plus one
in-flight migration. The value of writing it down is to (a) prevent drift (future proposals to build
a metrics pipeline, a config service, a feature-flag registry, etc.), and (b) record the few
genuine decisions — especially the Workflow boundary, which has a real tension with the
engine-agnostic IR.

---

## Decision

**Principle: Zynax builds only what is unique to it — the AI scheduling/intelligence layer — and
delegates every generic orchestration primitive to Kubernetes and the cloud-native ecosystem.**

Concretely:

1. **Delegated — do not build a Zynax service for these.** Metrics (Prometheus), tracing/logging
   (OpenTelemetry/Uptrace; **stdout for logs**), auth (cert-manager mTLS + RBAC), config/secrets
   (ConfigMaps/Secrets + env), service discovery (K8s DNS), scaling (HPA), networking
   (NetworkPolicy), leader election (`coordination.k8s.io` Lease), watches/reconciliation/informer
   caches (controller-runtime). New infrastructure needs **must** use a Kubernetes primitive, not a
   new Zynax service — a proposal to build one is an anti-pattern this ADR rejects.

2. **The one open migration: agent identity → `Agent` CRD (ADR-039).** The push registry, the
   in-memory/Postgres adapters, and the bespoke `capIndex` are replaced by a CRD + controller-runtime
   informer cache + a stateless scheduler. controller-runtime adoption (informers, watches,
   reconciliation, Lease-based leader election) rides along here — it is **adoption**, not a separate
   migration.

3. **Workflows: a thin `Workflow` CRD front-end, not a workflow store.** We will (in a future ADR)
   introduce a `Workflow` CRD **only as a declarative/GitOps authoring surface**: a controller
   reconciles it by calling the existing compile→submit path. **Execution stays engine-agnostic IR
   (ADR-012/015)** — the CRD is a front-end, the IR remains portable across Temporal/Argo/eval, and
   Zynax does **not** become the durable workflow store (etcd is not the source of truth for run
   state). This preserves the "same workflow graduates across engines" differentiator while giving
   K8s-native authoring.

4. **Not introduced (nothing to migrate today).** Tool/Model/Provider registries and feature flags
   do not exist as Zynax components. If such declarative configuration is ever needed, it is added as
   **CRDs or ConfigMaps reconciled by a controller** — never as a new bespoke registry service.

5. **Explicitly out of scope.** Migrating logging to a log-aggregation backend (Loki/Elastic/etc.) is
   **not** part of this program — logs stay structured-stdout; collection is the cluster's concern.
   Admission/conversion webhooks for CRDs are deferred until a `v1beta1` exists (per ADR-039).

6. **Stays custom — the Zynax intelligence core (never delegated):**
   - *Built today:* capability matching (`FindByCapability` + task-broker selection), task dispatch
     and the agent execution path (`AgentService.ExecuteCapability`), workflow execution via the IR +
     pluggable engines, model/provider routing (llm adapter), MCP integration (ADR-032), the
     memory-service, and context-slice / expert isolation (ADR-028).
   - *Planned differentiators (to be built as Zynax IP, not delegated):* the metrics-aware AI
     scheduler (ADR-039 M8), cost-aware and GPU-aware scheduling, trust/reputation scoring,
     multi-agent planning, multi-cluster federation, and distributed-inference coordination.

---

## Rationale

| Option | Assessment |
|--------|------------|
| **Codify a thin-Zynax delegation boundary; one migration (ADR-039); Workflow = thin CRD front-end** | ✅ **Chosen** — matches the audited reality (Zynax is already thin), prevents future drift toward re-building K8s primitives, strengthens the CNCF posture, and resolves the one real tension (Workflow) without sacrificing IR portability. |
| Migrate "everything" on the candidate list | ✗ Rejected — most items are already delegated or were never Zynax components; there is nothing to migrate. Treating them as open work would be busywork and could mislead. |
| Workflow as a full CRD source of truth (workflow state in etcd) | ✗ Rejected — couples the IR to Kubernetes and undercuts cross-engine portability (Temporal/eval), the stated differentiator; the Argo engine already owns the CRD at its boundary. |
| Migrate logging to Loki/Elastic | ✗ Rejected (out of scope by direction) — logs are already structured-stdout; aggregation is the cluster's job, and ADR-030 already rejected running a separate log stack. |
| Delegate scheduling / capability matching to Kubernetes | ✗ Rejected — Kubernetes schedules pods on CPU/RAM; it cannot reason about capabilities, model fit, protocol, GPU, cost, trust, or latency. This is the core Zynax IP and must stay custom. |

---

## Consequences

- **Positive:** A written boundary that keeps Zynax lean and CNCF-idiomatic — engineering effort
  concentrates on the AI scheduling/intelligence core, while Kubernetes and proven projects
  (cert-manager, Prometheus, OTel/Uptrace, NATS, controller-runtime) own the generic primitives.
  Future "let's build a service for X" proposals have a clear test to fail against. Confirms that the
  control plane is already mostly delegated — a strong maturity signal for M8.

- **Negative / trade-off:** The production control plane is **Kubernetes-coupled** (already largely
  true; ADR-039 makes it explicit for agent identity). The principle must not be over-applied to
  strip genuine domain logic — the custom core in Decision §6 is explicitly protected. The thin
  `Workflow` CRD front-end (Decision §3) adds a controller to build and maintain; it is justified
  only as GitOps-native authoring and must never absorb run-state.

- **Neutral / follow-up required:** Each concrete change lands behind its own ADR + Canvas, not this
  principle ADR — agent-registry → CRD is ADR-039; the `Workflow` CRD front-end is a future ADR
  (M8+). No code changes ship with this ADR. Revisit the boundary when introducing any new
  declarative surface (tool/model/provider/flags) to ensure it lands as a CRD/ConfigMap, not a new
  service.
