<!-- SPDX-License-Identifier: Apache-2.0 -->

# Decisions Needed

**Date:** 2026-05-21  
**Context:** Output of the 2026-05-20 principal architect review and 2026-05-21 architecture overhaul.  
**Purpose:** Items that require a human decision before implementation can proceed.

Items are marked with a recommended default. If the recommendation is acceptable, simply proceed — no formal approval needed unless the item says "requires ADR."

---

## D1 — OTel promotion: M7 → M5 or M6? (Milestone planning)

**Issue:** [#491](https://github.com/zynax-io/zynax/issues/491) — OpenTelemetry baseline for api-gateway + engine-adapter  
**Gap:** H2 from `docs/reviews/04-architecture-gaps.md` — rated **High** by the 2026-05-20 review  
**Current milestone:** M7

**Context:** Today, a workflow run is completely untraceable end-to-end. api-gateway logs a request ID; engine-adapter logs Temporal IDs; nothing links them. The 2026-05-20 review rates observability absence as "High" — operators cannot diagnose failures in production.

**Options:**
1. **Promote to M5** — ship before v0.4.0. Adds ~1–2 weeks to M5, already the longest milestone.
2. **Promote to M6** *(recommended)* — OTel lands alongside the K8s deployment work where structured logs/traces are most needed. Low-risk — v0.4.0 ships on time.
3. **Leave in M7** — status quo. Acceptable if M6 scope is already large.

**Recommended:** Option 2 (M6). `gh issue edit 491 --milestone "K8s Production-Ready (M6)"`

---

## D2 — ADR-020: Zero-trust auth design (Security)

**Issue:** [#240](https://github.com/zynax-io/zynax/issues/240)  
**Gap:** H7 from `docs/reviews/04-architecture-gaps.md`

**Context:** Today, inter-service communication has no authentication — any process in the same Docker network can call any gRPC service without a token. Bearer-token auth exists only at the api-gateway HTTP boundary. For K8s deployment (M6), this is the primary security gap.

**Scope of ADR-020:**
- mTLS for gRPC inter-service (the obvious choice for K8s)
- SPIFFE/SPIRE for workload identity
- Token-based with service accounts as an alternative

**Decision needed:** Which authentication model for inter-service gRPC in K8s (M6)?

**Recommended starting point:** mTLS with cert-manager for K8s; skip for local Docker Compose (too much overhead for development). This is a one-way door that shapes the entire M6 security work.

**Action:** Author ADR-020 in `docs/adr/ADR-020-zero-trust-auth.md` before starting #488 (mTLS).

---

## D3 — ADR-021: Horizontal scale plan (Architecture)

**Issue:** [#578](https://github.com/zynax-io/zynax/issues/578)  
**Gap:** H8 from `docs/reviews/04-architecture-gaps.md`

**Context:** The 2026-05-20 review scores architecture 7.5/10 but flags no documented answer to: "how does Zynax scale beyond a single-node deployment?" Key questions:
- workflow-compiler: once #466 (stateless) merges, it scales horizontally. What's the target replica count?
- task-broker: in-memory repo — multiple replicas diverge. For M6, does this require a Redis-backed repo or Postgres?
- agent-registry: same in-memory problem as task-broker.
- api-gateway: stateless today. Scale freely.

**Decision needed:** What is the horizontal scale target for v0.5.0 (M6)?  
Specifically: does task-broker need a persistence layer in M6, or is single-replica acceptable?

**Action:** Author ADR-021 in `docs/adr/ADR-021-horizontal-scale.md`. Link from #578.

---

## D4 — `ZYNAX_DEV_INSECURE` env var name (Issue #623)

**Issue:** [#623](https://github.com/zynax-io/zynax/issues/623) — startup guard for empty API key  
**Context:** Issue #623 uses `ZYNAX_DEV_INSECURE` as the flag that allows empty `ZYNAX_GW_API_KEY` in development. This env var name does not currently exist in the codebase.

**Options:**
1. `ZYNAX_DEV_INSECURE=1` *(recommended)* — clear and consistent with the `INSECURE` naming seen in gRPC ecosystem
2. `ZYNAX_GW_DISABLE_AUTH=1` — gateway-scoped
3. `ZYNAX_NO_AUTH=1` — shortest

**Recommended:** Option 1. Update issue #623 with the chosen name before implementation starts.

**Action:** Reply to #623 confirming the env var name. No ADR needed — reversible choice.

---

## D5 — gRPC deadline default (Issue #622)

**Issue:** [#622](https://github.com/zynax-io/zynax/issues/622) — add `context.WithTimeout` to all outgoing gRPC calls  
**Context:** Issue #622 uses 30s as the default deadline. This is a reasonable starting point but may need tuning based on observed latency.

**Options:**
1. **30s fixed** — hardcoded, simple. Good for M5.
2. **Configurable per service** *(recommended)* — env var `ZYNAX_GRPC_TIMEOUT_S` with 30s default. Operators can tune without rebuild.
3. **Per-call timeouts** — finer-grained but complex.

**Recommended:** Option 2. Adds one env var per service, negligible complexity. Update issue #622 if you want this rather than Option 1 before implementation.

**Action:** Confirm preference in a comment on #622. No ADR needed.

---

## D6 — Close or supersede stale M6/M7 issues (Issue hygiene)

**Context:** `docs/reviews/05-action-plan.md` lists several issues that may need explicit closure or reclassification:

| Issue | Status | Action needed |
|-------|--------|---------------|
| [#358](https://github.com/zynax-io/zynax/issues/358) | Publish tools image (superseded by #563) | Close as superseded? |
| [#235](https://github.com/zynax-io/zynax/issues/235) | Standalone SBOM (superseded by #489 / M6.C) | Close when M6 goes active? |
| [#239](https://github.com/zynax-io/zynax/issues/239) | Similar SBOM/cosign item | Same as #235 |

**Recommended:** Close #358 now (superseded by #563, already merged). Defer #235/#239 until M6 planning.

**Action:** `gh issue close 358 --reason "not planned" --comment "Superseded by #563 (deduplication done)."` — only if you agree.

---

*Update this file as decisions are made. Remove resolved items or mark them ✅ Done.*
