# ADR-039: CRD-native Scheduler — the `Agent` CRD as the single source of truth

**Status:** Accepted  **Date:** 2026-06-22
**Related:** ADR-021 (**amends** — reverses the agent-registry half: agent state moves
from Postgres to etcd via the `Agent` CRD; task-broker's Postgres `TaskRepository` is
untouched), ADR-028 (**amends** — preserves the AgentDef-vs-Workflow split and the
context-slice injection contract verbatim; only the *transport of identity* changes from
`RegisterAgent` push to `kubectl apply`), ADR-001 (the Kubernetes API server is external
infrastructure, like Temporal/NATS — calling it from `infrastructure/` is compliant),
ADR-008 (etcd is K8s control-plane infrastructure, not a shared application database),
ADR-011 (a CRD is the canonical declarative control surface), ADR-015 (pluggable-behind-a-port
precedent), ADR-033 (expert-agent substrate consumes the registry), ADR-016/ADR-017
(BDD at gRPC boundaries · `GOWORK=off`)

> **Sequencing:** ADR Proposed and a de-risking spike land in **M7** (the first-run UX
> ships unchanged on the current gRPC/Postgres registry); the full cutover is **M8**
> (CNCF-aligned). A REASONS Canvas + story issues are created when the M8 `feat:` work
> begins.

---

## Context

Zynax agent identity lives behind a clean hexagonal port — `domain.AgentRepository`
(`services/agent-registry/internal/domain/repository.go`, 5 methods) — with two
interchangeable adapters (`memory_repo.go`, `postgres/repository.go`). It is populated by
**push**: every adapter calls `RegisterAgent` on boot (with exponential-backoff retry) and
deregisters on `SIGTERM`, and operators additionally `zynax apply kind: AgentDef`, which
api-gateway parses and forwards as `RegisterAgent`. The task-broker reads it via
`FindByCapability` and dispatches with blind **round-robin** (`idx % len(agents)`,
`services/task-broker/internal/domain/service.go`).

Three structural problems follow from a push registry:

1. **Two sources of truth.** The declarative `AgentDef` manifest an operator applies and the
   runtime record an agent pushes can drift; nothing reconciles them.
2. **Stale liveness.** An agent that crashes without calling `DeregisterAgent` leaves a
   `REGISTERED` row. The registry cannot authoritatively know liveness, and task-broker has
   no readiness signal — so it round-robins requests into dead endpoints.
3. **Off-idiom for Kubernetes.** The rest of the platform is already K8s-native (Helm,
   cert-manager mTLS, Deployments, HPA, NetworkPolicy), yet discovery is a bespoke push
   registry that re-implements lifecycle, health, watches, and caching that the Kubernetes
   control plane already provides. This is an operational liability for HA and a credibility
   gap for the M8 CNCF Sandbox submission.

The category is converging on the opposite pattern. **Kagent** stores agents as Kubernetes
CRDs — the API server *is* the registry — and a controller reconciles them; there is no
separate registry service. Capability metadata rides the agent's A2A Agent Card. The broader
control-plane ecosystem (Argo CD, Crossplane, cert-manager, Flux, Istio, Gateway API) does
the same: it treats Kubernetes as the database rather than building a registry on top of it.

A key structural fact makes the change cheap for Zynax. ADR-021 already established that
"the repository port is unchanged; only the infrastructure adapter changes." The capability
lookup in the memory adapter is an in-memory secondary index (`capIndex: capability → {agentID}`)
— exactly the shape a Kubernetes informer cache would maintain from watch events. The
domain/service/gRPC layers never learn where the data comes from.

This is a **one-way door**: once the `Agent` CRD is the public, GitOps-managed source of
truth and push-registration is removed, reverting means restoring self-registration across
every adapter, the api-gateway apply path, Helm, and the BDD suite. It therefore warrants an
ADR before implementation, and a spike before the M8 build.

---

## Decision

We will make a Kubernetes **`Agent` Custom Resource** the **single source of truth** for
agent identity and capabilities, and evolve `agent-registry` from a push-based discovery
store into a **stateless, informer-backed Scheduler**.

1. **`Agent` CRD as the source of truth.** A namespaced CRD in group `zynax.io`, version
   `v1alpha1`, carries identity + capabilities (today's `AgentDef`) plus scheduler-scoring
   hints (`selectors`, `cost`, `resources`, `models`, `protocols`). Its OpenAPI v3 structural
   schema ports the constraints from `spec/schemas/agent-def.schema.json`. **Push
   registration (`RegisterAgent`/`DeregisterAgent`) and the memory + Postgres
   `AgentRepository` adapters are removed from the production path.** The CRD `spec` describes
   identity, capabilities, and an `endpointRef` to an existing Service — it does **not**
   subsume the agent's Deployment.

2. **Registry → stateless Scheduler.** The service runs a controller-runtime manager with an
   informer cache over `Agent` CRs, maintaining an in-memory capability index mirroring
   today's `capIndex`. It exposes a new `SelectAgent(capability, policy, constraints)` RPC
   (new file `protos/zynax/v1/scheduler.proto`, additive — passes the file-scoped
   `buf breaking` gate) that returns **exactly one** chosen agent with a selection rationale.
   task-broker stops calling `FindByCapability` + round-robin and calls `SelectAgent`. The
   scheduler is stateless: on restart the cache `List`s from the API server and rebuilds the
   index for free — no persistence, no migration.

3. **Readiness in CRD status, dynamic metrics from Prometheus at schedule time.** The CRD
   `status` holds only low-churn, reconciler-owned fields (`ready`, `replicas`, conditions)
   derived from the backing Service's EndpointSlice. Live `load`/`latency`/`queueDepth` are
   queried from the existing (M6) Prometheus **at selection time** — never stored in CRD
   status (that would churn etcd). If Prometheus is unavailable, selection degrades to
   readiness-filtered round-robin and never fails.

4. **The scoring pipeline is ordered and short-circuiting:** capability match (O(1) index) →
   hard constraints (tags/language/model/gpu/protocols) → readiness (the stale-liveness fix)
   → expert target (ADR-028 strict isolation, no fallback) → Prometheus-weighted load/latency
   → cost/gpu/model tie-break → locality. Expert-targeting moves into the request; the
   broker keeps the ADR-028 context-slice binding (it reads the `input_schema` carried back
   in the response).

---

## Rationale

| Option | Assessment |
|--------|------------|
| **`Agent` CRD as sole source of truth + stateless informer-backed Scheduler** | ✅ **Chosen** — eliminates the dual source of truth and the stale-liveness class of bug (readiness is reconciled from EndpointSlice, not self-asserted); statelessness makes the read path trivially N-replica and restart-recovery a free resync; aligns the control plane with the K8s idiom the rest of the platform already uses (strengthens the M8 CNCF story); upgrades blind round-robin to metrics-aware selection. |
| CRD adapter *alongside* memory/Postgres (keep the port, keep push as fallback) | ✗ Rejected — preserves two sources of truth (the original disease) and the stale-liveness bug, and the "return one scored agent" requirement does not fit the `FindByCapability`-returns-a-list port without contortion. Pays the dependency cost without the architectural payoff. |
| Keep round-robin selection (informer-backed candidate list, dumb `idx % len` pick) | ✗ Rejected — defeats the explicit goal of metrics-aware selection; ignores readiness, load, cost, and GPU/model fit. |
| Bare `client-go` informers without controller-runtime | ✗ Rejected — controller-runtime provides manager, leader election, cache, and webhook scaffolding we would otherwise hand-roll; the marginal dependency weight over bare `client-go` is small and is de-risked by the spike. |
| Store dynamic metrics (`load`/`latency`/`queueDepth`) in CRD `status` | ✗ Rejected — high-frequency status writes churn etcd and can throttle the API server; metrics belong in the existing Prometheus, queried at selection time. |
| Do nothing / defer to M9 | ✗ Rejected — leaves the dual-source-of-truth and stale-liveness liabilities in place and compounds the migration cost as more adapters self-register; misses the M8 CNCF window. |

---

## Consequences

- **Positive:** One source of truth, GitOps-native (`Agent` CRs are diffable, `kubectl`-able,
  Argo-syncable — aligns with the existing `cmd/zynax/gitops` direction). The stale-liveness
  bug class disappears. The scheduler is stateless, so the read path scales horizontally and
  a restart recovers by resyncing from the API server with zero persisted state (the ADR-021
  reversal payoff). Dispatch becomes load/latency/cost/GPU-aware instead of blind
  round-robin. The operator pattern + least-privilege RBAC strengthens the M8 CNCF Sandbox
  submission.

- **Negative / trade-off:** **This removes the Docker-Compose discovery path** and narrows the
  production control plane to Kubernetes-only. That directly erodes the EPIC #1370 first-run
  wedge ("zero-secret Ollama quickstart, runs on plain Compose vs K8s-only Kagent"). The M8
  cutover **must** discharge this by retargeting the one-command first-run to a local-K8s
  (k3d) bootstrap — preserving "clone → one command → result" on a laptop — or by explicitly
  keeping a documented Compose "lite" discovery mode behind a flag. **This is the load-bearing
  trade-off of this ADR; M7's first-run UX is unaffected because the cutover is M8.** The
  change is also a one-way door (push-registration removal touches every adapter, the
  api-gateway apply path, Helm, and the BDD suite), and it adds a live API-server dependency,
  new RBAC, the heavy `controller-runtime`/`client-go` dependencies, and a soft Prometheus
  dependency on the dispatch hot path.

- **Neutral / follow-up required:** Create the M8 REASONS Canvas + story issues. Run the M7
  spike to confirm `controller-runtime` builds under `GOWORK=off` (the heaviest unknown;
  fallback is to pull the scheduler out of `go.work` as a standalone module like `cmd/zynax`)
  and to prove resync-on-restart + Prometheus-down degradation. Sequence the proto change as
  **deprecation, not deletion** — `buf breaking` is file-scoped, so the new `scheduler.proto`
  is safe to add, but the `AgentRegistryService` RPCs in `agent_registry.proto` are marked
  `deprecated` and return `UNIMPLEMENTED` in M8 (their `AgentDef`/`CapabilityDef` messages are
  reused by `scheduler.proto`), with hard removal scheduled for M9. Enable leader election for
  the single-writer status reconciler while keeping the select path always-serving. Cache
  Prometheus reads with a short TTL to protect the hot path. Settle the multi-runtime
  positioning question (k3d-for-local recommended) in the M8 first-run work.
