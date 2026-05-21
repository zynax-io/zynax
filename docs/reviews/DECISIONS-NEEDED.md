<!-- SPDX-License-Identifier: Apache-2.0 -->

# Decisions Needed

**Date:** 2026-05-21  
**Context:** Output of the 2026-05-20 principal architect review and 2026-05-21 architecture overhaul.  
**Status:** All 6 decisions resolved 2026-05-21.

---

## ✅ D1 — OTel promotion: M7 → M6

**Issue:** [#491](https://github.com/zynax-io/zynax/issues/491)  
**Decision:** Promote to **M6**.  
**Rationale:** The 2026-05-20 review rated observability absence as "High" gap (H2). A workflow run is completely untraceable end-to-end today. OTel baseline lands alongside K8s deployment work (M6) where structured traces are most needed. v0.4.0 ships on time.

**Actions taken:**
- #491 milestone changed M7 → M6 (`gh issue edit 491 --milestone "K8s Production-Ready (M6)"`)
- Comment added to #491 with rationale and scope

**Scope for M6:** api-gateway + engine-adapter export traces to an OTLP collector with `workflow_id` as root span attribute. Pattern: `docs/engineering/best-practices/architecture-patterns.md §OpenTelemetry`.

---

## ✅ D2 — ADR-020: Inter-service gRPC auth model

**Issue:** [#240](https://github.com/zynax-io/zynax/issues/240) (ADR-020 authoring) · [#488](https://github.com/zynax-io/zynax/issues/488) (implementation)  
**Decision:** **mTLS with cert-manager** for all inter-service gRPC in Kubernetes.

**Rationale:** Industry standard for K8s workload identity. Works with existing gRPC client code (add TLS credentials). cert-manager manages certificate lifecycle. Local Docker Compose stays insecure (dev convenience). SPIFFE/SPIRE deferred until CNCF Sandbox (M8).

**Actions taken:**
- Comment added to #240 with ADR-020 scope
- Comment added to #488 noting it implements this decision, blocked on #240

**ADR-020 must specify:**
- mTLS for all inter-service gRPC in K8s
- cert-manager as certificate authority
- Docker Compose exemption (insecure gRPC for local dev)
- Per-service identity via certificate SANs

---

## ✅ D3 — ADR-021: Horizontal scale plan

**Issue:** [#578](https://github.com/zynax-io/zynax/issues/578) (ADR-021) · [#626](https://github.com/zynax-io/zynax/issues/626) (M6.H epic)  
**Decision:** **Postgres-backed repositories** for task-broker + agent-registry in M6.

**Rationale:** Front-loads work required for CNCF Sandbox (M8) anyway. ACID guarantees unlock queryable task history and multi-tenant safety. Enables horizontal scaling before production. Effort: ~4–5 weeks.

**Actions taken:**
- Comment added to #578 with ADR-021 scope
- [#626](https://github.com/zynax-io/zynax/issues/626) — `epic(infra): M6.H — Postgres-backed repositories` created in M6

**ADR-021 must specify:**
- Postgres as persistence layer for task-broker + agent-registry in M6
- Hexagonal pattern: `infrastructure/postgres/` adapter per service; in-memory retained as test double
- Postgres in docker-compose + Helm chart
- Integration tests via testcontainers-go

---

## ✅ D4 — `ZYNAX_DEV_INSECURE` env var name (issue #623)

**Issue:** [#623](https://github.com/zynax-io/zynax/issues/623)  
**Decision:** `ZYNAX_DEV_INSECURE=1`.

**Rationale:** Clear intent, consistent with gRPC ecosystem conventions. Prefix matches all other `ZYNAX_` vars. One flag can cover future insecure-mode relaxations.

**Actions taken:**
- Comment added to #623 with full implementation spec (startup guard logic)

**Implementation:** `os.Exit(1)` if `ZYNAX_GW_API_KEY` empty and `ZYNAX_DEV_INSECURE` not set. `WARN` log + continue if `ZYNAX_DEV_INSECURE=1`.

---

## ✅ D5 — gRPC deadline default (issue #622)

**Issue:** [#622](https://github.com/zynax-io/zynax/issues/622)  
**Decision:** Configurable via `ZYNAX_GRPC_TIMEOUT_S`, default 30 seconds.

**Rationale:** Zero overhead vs hardcoded but gives production flexibility. Operators tune per deployment without rebuild.

**Actions taken:**
- Comment added to #622 with full implementation spec (config struct field + `context.WithTimeout` usage)

**Services:** api-gateway (→ compiler, → engine-adapter), engine-adapter (→ task-broker), task-broker (→ agent-registry).

---

## ✅ D6 — Stale issue hygiene

**Decision:** Close #358 now; defer #235 and #239 to M6 planning.

**Actions taken:**
- #358 closed as "not planned" — superseded by #563 (merged 2026-05-20)
- #235 and #239 left open; revisit at M6 kickoff when #489/#465 scope is confirmed

---

*All decisions resolved. This file is now a decision log, not a pending list.*
