<!-- SPDX-License-Identifier: Apache-2.0 -->
# ADR-051: Load-testing harness, SLO acceptance criteria, and failure-injection strategy

**Status:** Proposed  **Date:** 2026-07-08
**Related:** ADR-039 (Prometheus-down degradation promise — verified here), ADR-046 (best-effort publish/DLQ semantics — verified here), ADR-044 (edge rate-limit under burst — verified here), ADR-030 (OTel/Uptrace — the measurement substrate), ADR-016 (**extends** — adds a per-release performance/failure tier)
**Proposal issue:** #1696 · **Implementation candidate:** #1420

---

## Context

The 2026-06-19 architecture review's single highest risk (R1/T1.1, High/High) is that no
load-test harness or written SLOs exist; it separately flags the absence of any
failure-injection suite. Issue #1420 has carried the ask since June, unmilestoned.

The substrate is now in place: a kind-based e2e harness with dual engine legs, Prometheus
metrics in every service, and OTel/Uptrace tracing. More importantly, **accepted ADRs
already promise specific degradation behaviour that nothing verifies**: the scheduler
falls back to readiness-filtered round-robin when Prometheus is down (ADR-039 §3);
JetStream publishing is best-effort with `MaxDeliver=5` + DLQ (ADR-046); the edge
rate-limit `BackendTrafficPolicy` bounds burst traffic (ADR-044). These are contracts
without tests.

Without a decision there is no definition of "fast enough", no repeatable harness, no
verified failure story — and no capacity data for CNCF or enterprise evaluation.

---

## Decision

We will define SLOs, build a repeatable load harness on the existing e2e stack, and
verify the ADR-promised degradation modes by targeted failure injection:

1. **Written SLOs for the golden paths** — apply→submit latency, dispatch latency, event
   fan-out lag, watch-stream time-to-first-event — at a declared reference scale on the
   reference environment (kind on CI hardware, stated honestly). Concrete numbers are
   baselined by the first harness runs and recorded as an amendment to this ADR.
2. **Repeatable load harness** reusing the kind e2e stack: k6 (or vegeta) for the REST
   edge plus a small Go driver for gRPC/dispatch load; emits a machine-readable report.
   No new infrastructure services.
3. **Targeted failure injection of promised behaviour** — kill Prometheus (scheduler must
   degrade to readiness-filtered selection, never fail); kill/restart NATS (publishers
   stay best-effort, DLQ semantics hold, watch terminal-close survives); kill an engine
   leg (the other leg is unaffected); burst the edge (rate-limit engages). Plain
   `kubectl delete pod` + harness assertions — no chaos platform.
4. **Per-release cadence.** The suite runs per release (not per PR) and publishes its
   report alongside the conformance matrix (#1692); an SLO miss is a release blocker by
   policy.

---

## Rationale

| Option | Assessment |
|--------|------------|
| Status quo — no load tests, anecdotal capacity claims | ✗ Rejected — top review risk stays open; ADR-promised fallbacks remain unverified; no answer for CNCF/enterprise due diligence. |
| Full chaos platform (Chaos Mesh / LitmusChaos) + continuous soak | ✗ Deferred — CNCF-native and rich, but heavy operational footprint at single-maintainer scale; the highest-value scenarios are the few ADR-promised ones, coverable with pod deletion; revisit post-CNCF-acceptance. |
| **Written SLOs + per-release harness on the existing kind stack + targeted injection of ADR-promised fallbacks** | ✅ Chosen — reuses the e2e substrate; tests exactly the promises already made; bounded cost; produces publishable capacity data. |

---

## Consequences

- **Positive:** closes review R1/T1.1; every ADR-promised degradation mode gets a test;
  honest capacity/SLO numbers for docs, CNCF, and due diligence; regressions caught at
  release boundaries; #1420 gains a Definition-of-Done.
- **Negative / trade-off:** kind-on-CI numbers are indicative, not production benchmarks
  — every report must state the reference environment; per-release cadence lets a
  regression live up to one milestone; releases gain a time-boxed load+chaos gate.
- **Neutral / follow-up required:** #1420 becomes the implementation epic-candidate on
  acceptance (canvas per ADR-019); first-run baselines set the SLO numbers (ADR
  amendment); graceful-shutdown draining (#1418) joins the injection scenarios once
  implemented; publication wiring shared with the conformance suite (#1692).
