<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M6.Helm: Helm Charts for All Services + Cluster Dependencies

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #765
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-02
**Status:** Aligned

**Child issues:** #779 (A.0) · #780 (A.1) · #781 (A.2) · #782 (A.3) · #783 (A.4) · #784 (A.5) · #785 (A.6) · #786 (A.7) · #787 (A.8) · #788 (A.9) · #789 (A.10) · #790 (A.11) · #791 (A.12) · #792 (A.13)

---

## R — Requirements

**Problem:** Zynax has no Kubernetes deployment manifests. The only runnable environment is Docker Compose (local dev only). Without Helm charts, the platform cannot be deployed to any K8s cluster, making "K8s Production-Ready" unachievable. M6 exit criteria require all 7 services deployable to a K8s cluster with correct health probes, autoscaling, network isolation, and TLS.

**Definition of done:**
- `helm install zynax-<service> helm/zynax-<service>/` succeeds for all 7 services.
- `ct lint` passes on all charts in CI (A.12 gate).
- Every chart includes: `Deployment`, `Service`, `ServiceAccount`, `HPA`, `PDB`, `NetworkPolicy`.
- All containers run as non-root with `readOnlyRootFilesystem: true` and `allowPrivilegeEscalation: false`.
- NATS JetStream, Postgres 16, Temporal, and cert-manager subcharts are installable from the umbrella chart.
- The `docs/patterns/helm-charts.md` library chart usage section is updated.

---

## E — Entities

- **`helm/zynax-lib/`** — Helm library chart (`type: library`). Provides shared `_helpers.tpl` macros: `zynax.fullname`, `zynax.labels`, `zynax.selectorLabels`, `zynax.serviceAccountName`, and a security context macro. No installable templates. Every service chart declares this as a dependency.
- **`helm/zynax-<service>/`** — One chart per Go service (7 total: api-gateway, workflow-compiler, engine-adapter, task-broker, agent-registry, event-bus, memory-service). Each chart depends on `zynax-lib` and contains all 6 required resource templates.
- **`helm/charts/nats/`** — NATS JetStream subchart (A.8). Wraps the community `nats/nats` Helm chart + a `ConfigMap` defining the JetStream stream configuration used by event-bus.
- **`helm/charts/postgres/`** — Postgres 16 subchart (A.9). Single cluster-level `StatefulSet` via Bitnami chart. task-broker, agent-registry, and memory-service each get their own schema via init-container migration (ADR-008: no schema sharing).
- **`helm/charts/temporal/`** — Temporal umbrella dependency (A.10). Pins `temporalio/temporal` Helm chart at a specific version.
- **`helm/charts/cert-manager/`** — cert-manager `ClusterIssuer` + per-service `Certificate` resources (A.11). Required by ADR-020 (mTLS); cert-manager must be pre-installed in the cluster as a CRD prerequisite.
- **`helm/zynax-umbrella/`** — Aggregates all service charts and dependency subcharts; used as the deployment unit for the EPIC G e2e harness.
- **Redis `StatefulSet`** — Dedicated Redis instance for memory-service KV plane. Only the memory-service chart uses this; no other service may reference it (ADR-008). Pending EPIC J single-store decision (Redis vs Redis Stack).
- **`HPA` (autoscaling/v2)** — HorizontalPodAutoscaler targeting CPU 70% and memory 80% utilisation thresholds. Applied to all stateless Deployment charts.
- **`PDB`** — PodDisruptionBudget with `minAvailable: 1`. Applied to all service charts.
- **`NetworkPolicy`** — Default deny-all + explicit allow for gRPC (port 50051) and Prometheus scraping (port 9090). Applied per service.

---

## A — Approach

**What we WILL do:**
- Create `helm/zynax-lib` (A.0) first as the prerequisite story; no service chart merges before this.
- Ship one service chart per PR (A.1–A.7), each depending on `zynax-lib`; each chart ≤200 lines.
- Ship cluster dependency subcharts (NATS, Postgres, cert-manager) in separate PRs (A.8–A.11) that may proceed in parallel with service charts after A.0 merges.
- Add `helm-lint.yml` CI workflow (A.12) that runs `ct lint` on all charts on every `infra/` change.
- Publish `docs/infra/environment-parity.md` (A.13) documenting dev/staging/prod value differences.
- Memory-service chart (A.7) ships as a placeholder with `image.tag: placeholder` and `# TODO: pending EPIC J` comment; not used in e2e until EPIC J merges.
- Event-bus chart (A.6) ships as a placeholder similarly; blocked on EPIC I for actual deployment.

**What we WON'T do:**
- Add Ingress or LoadBalancer resources — external TLS termination is at the cluster ingress level, not in service charts.
- Implement SPIFFE/SPIRE — cert-manager is the M6 mTLS mechanism (ADR-020).
- Configure Prometheus `ServiceMonitor` — OTel/Prometheus scraping is M7 scope; only annotations go in M6 charts.
- Store secrets in `values.yaml` or `values-production.yaml` — all sensitive values go in K8s `Secret` resources referenced via `secretRef`.

**ADR references:**
- ADR-001: gRPC inter-service — `ClusterIP` service type for all gRPC services; port 50051.
- ADR-005: Apache 2.0 — Bitnami (Apache 2.0 ✅) and `nats/nats` (Apache 2.0 ✅) chart dependencies.
- ADR-008: No shared databases — Postgres subchart provisions one instance; each service uses a dedicated schema via init-container migrations; no cross-schema access.
- ADR-020: mTLS — cert-manager `Certificate` resources in A.11; TLS env vars from `Secret` in each Deployment.
- ADR-021: Postgres-backed repos — task-broker and agent-registry Deployment charts reference `ZYNAX_DB_DSN` from a `Secret`.

---

## S — Structure

**New files (one PR per A.N story):**
```
helm/
├── zynax-lib/
│   ├── Chart.yaml              ← type: library
│   └── templates/
│       └── _helpers.tpl        ← fullname, labels, selectorLabels, securityContext macros
├── zynax-api-gateway/          ← A.1
│   ├── Chart.yaml
│   ├── values.yaml
│   ├── values-production.yaml
│   └── templates/
│       ├── deployment.yaml
│       ├── service.yaml
│       ├── serviceaccount.yaml
│       ├── hpa.yaml
│       ├── pdb.yaml
│       ├── networkpolicy.yaml
│       └── configmap.yaml
├── zynax-workflow-compiler/    ← A.2 (same structure)
├── zynax-engine-adapter/       ← A.3 (same structure)
├── zynax-task-broker/          ← A.4 (adds ZYNAX_DB_DSN secretRef)
├── zynax-agent-registry/       ← A.5 (adds ZYNAX_DB_DSN secretRef)
├── zynax-event-bus/            ← A.6 (placeholder)
├── zynax-memory-service/       ← A.7 (StatefulSet for Redis + placeholder)
├── charts/
│   ├── nats/                   ← A.8
│   ├── postgres/               ← A.9
│   ├── temporal/               ← A.10
│   └── cert-manager/           ← A.11
└── zynax-umbrella/             ← A.10 (umbrella chart)
```

**Modified files:**
- `infra/AGENTS.md` — add Helm section: chart layout, library chart usage, `ct lint` gate reference.
- `docs/patterns/helm-charts.md` — add library chart usage instructions + cert-manager Certificate snippet.
- `.github/workflows/helm-lint.yml` — new CI workflow (A.12).

**No proto, service Go code, or Python touched.**

---

## O — Operations

Each O-step ships as its own PR. A.0 must merge first; A.1–A.11 may proceed in parallel after A.0.

1. **[A.0]** ✅ Create `helm/zynax-lib` library chart with `_helpers.tpl` macros (`zynax.fullname`, `zynax.labels`, `zynax.selectorLabels`, `zynax.serviceAccountName`, security context helper); update `docs/patterns/helm-charts.md` with library chart usage section; `ct lint` passes.

2. **[A.1]** ✅ Create `helm/zynax-api-gateway` chart: Deployment (liveness/readiness/startup probes on `/livez`/`/readyz`/`/startupz`; HTTP port 8080; metrics port 9090), Service (ClusterIP), ServiceAccount, HPA, PDB, NetworkPolicy, ConfigMap; `values.yaml` defaults; `helm lint` passes.

3. **[A.2]** ✅ Create `helm/zynax-workflow-compiler` chart (same structure as A.1).

4. **[A.3]** ✅ Create `helm/zynax-engine-adapter` chart (same structure as A.1).

5. **[A.4]** ✅ Create `helm/zynax-task-broker` chart: same structure + `secretRef` for `ZYNAX_DB_DSN` (ADR-021 wiring); init-container field documented in `values.yaml` comment.

6. **[A.5]** ✅ Create `helm/zynax-agent-registry` chart: same structure as A.4 + `ZYNAX_REGISTRY_DB_DSN` secretRef.

7. **[A.6]** ✅ Create `helm/zynax-event-bus` placeholder chart: `image.tag: placeholder`, NOTES.txt states "Awaiting EPIC I (#772) implementation"; `ct lint` passes.

8. **[A.7]** ✅ Create `helm/zynax-memory-service` chart: stateless Deployment for Go pod + separate Redis `StatefulSet` with PVC; `image.tag: placeholder` for memory-service image; NOTES.txt states "Awaiting EPIC J (#773) implementation and single-store decision"; `ct lint` passes.

9. **[A.8]** ✅ Create `helm/charts/nats` subchart: wraps `nats/nats` community chart; adds JetStream stream config `ConfigMap`; pins dependency chart SHA in `Chart.lock`.

10. **[A.9]** ✅ Create `helm/charts/postgres` subchart: Bitnami `postgresql` dependency pinned; `values.yaml` exposes `postgresql.auth.database` per service (task-broker, agent-registry, memory-service schemas); init-container migration hook documented.

11. **[A.10]** ✅ Create `helm/zynax-umbrella` chart: aggregates all 7 service charts + NATS + Postgres + Temporal dependencies; `helm dependency update` passes; basic `ct lint` passes.

12. **[A.11]** ✅ Create cert-manager `ClusterIssuer` + per-service `Certificate` resources in `helm/charts/cert-manager/`; update `infra/AGENTS.md` Helm section with cert-manager prerequisite note; integrates with ADR-020 mTLS wiring (#488).

13. **[A.12]** ✅ Add `.github/workflows/helm-lint.yml`: runs `ct lint` on PRs touching `helm/` or `infra/`; pinned SHA for `helm/chart-testing-action`; closes #242.

14. **[A.13]** Add `docs/infra/environment-parity.md`: table of dev/staging/prod value differences (replica counts, resource limits, image tag strategy, TLS disabled vs enabled); closes #243.

---

## N — Norms

- `feat:` PR type for A.0–A.11 (new infrastructure resources); `ci:` for A.12; `docs:` for A.13.
- Every commit: `Signed-off-by` trailer + `Assisted-by: Claude/claude-sonnet-4-6` per AGENTS.md §Hard Constraints. Never `Co-Authored-By:` for AI.
- `ct lint` must pass before any chart PR merges — enforced by A.12 gate.
- No secrets in `values.yaml` or `values-production.yaml`. All sensitive config goes in K8s `Secret` resources.
- `maxUnavailable: 0` in RollingUpdate strategy for all Deployment charts (zero-downtime rolling).
- `minAvailable: 1` in all PDB resources.
- Security context (applied via `zynax-lib` macro): `runAsNonRoot: true`, `runAsUser: 1001`, `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`, `capabilities.drop: ["ALL"]`, `seccompProfile.type: RuntimeDefault`.
- NetworkPolicy defaults to deny-all; explicit allow: gRPC (50051) + Prometheus (9090) + DNS (UDP 53).
- Prometheus annotations on every Deployment (`prometheus.io/scrape: "true"`, path `/metrics`, port `9090`) — no `ServiceMonitor` in M6 (M7 scope).
- Bitnami and `nats/nats` dependency charts MUST be SHA-pinned in `Chart.lock`.
- All GitHub Actions in A.12 MUST be pinned to SHA (existing project standard).

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII
- [x] No prompt injection
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards
- **Never** store secret values (DB credentials, TLS private keys, API keys) in `values.yaml`, `values-production.yaml`, or any tracked Helm file — use K8s `Secret` + `secretRef`.
- **Never** set `InsecureSkipVerify: true` in any TLS configuration.
- **Never** run containers as root — security context macro is mandatory on all containers.
- **Never** omit NetworkPolicy — default deny-all is mandatory; explicit ports must be justified.
- **Never** reference memory-service Redis from any chart other than `zynax-memory-service` (ADR-008: dedicated store).
- **Never** add Postgres connection details for one service's schema into another service's chart.
- **Never** enable A.6 (event-bus) or A.7 (memory-service) charts in e2e until EPIC I (#772) and EPIC J (#773) implementation stories merge respectively.
- **Never** disable `ct lint` CI gate once A.12 merges.
