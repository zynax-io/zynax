<!-- SPDX-License-Identifier: Apache-2.0 -->
# ADR-048: API versioning and deprecation policy for REST and gRPC surfaces

**Status:** Proposed  **Date:** 2026-07-08
**Related:** ADR-001 (gRPC inter-service protocol — the surface being governed), ADR-039/ADR-046 (removal-clause precedent this policy codifies), ADR-027 (release mechanics that carry deprecation notices)
**Proposal issue:** #1693 · **User-facing policy doc:** #1415

---

## Context

Zynax exposes three consumer-facing contract surfaces: gRPC packages under `zynax.v1`
(`protos/zynax/v1/`), the REST gateway under `/api/v1/`, and the AsyncAPI eventing spec
(`spec/asyncapi/zynax-events.yaml`, versioned via `zynaxschemarev`). The only guard on
change today is mechanical — `buf breaking` as a CI gate — plus per-case removal clauses
written into individual ADRs.

The M8→M9 wave exercised an implicit convention twice: mark deprecated in-band
(`option deprecated = true` on `EventBusService`; `x-zynax-deprecated` /
`x-zynax-removal-milestone` in AsyncAPI; `UNIMPLEMENTED` on the registry push RPCs), ship
the deprecation in a release (v0.7.0), remove on the next milestone boundary (M9). It
worked, but it lives in two ADR removal clauses and tribal knowledge.

The 2026-06-19 architecture review rates the missing policy its highest-priority unfiled
decision (R6/T3.2; gap analysis "High"). Issue #1415 requests the user-facing document.
Without a written policy, every future surface change is re-litigated, and v1.0.0 —
reserved for CNCF acceptance — would ship without a compatibility contract, the first
thing evaluators check.

---

## Decision

We will adopt the following policy for `zynax.v1` gRPC, `/api/v1` REST, and the AsyncAPI
eventing contract:

1. **Stability promise.** Surfaces under a `v1` major are stable: no breaking change
   (removal, rename, semantic change of an existing field/RPC/route/channel) lands within
   the same major surface. Additive changes remain free (guarded by `buf breaking`'s
   file-scoped rules and spec validation).
2. **In-band deprecation markers.** A surface scheduled for removal is marked where the
   consumer sees it: `option deprecated = true` (proto), `x-zynax-deprecated` +
   `x-zynax-removal-milestone` (AsyncAPI), `Deprecation` + `Sunset` headers (REST), plus a
   release-notes "Deprecations" section entry.
3. **Notice window.** A deprecation ships in at least one published release before its
   removal. Removals ride milestone boundaries and require an ADR removal clause naming
   the artifact, the gate (e.g. zero caller references), and the target milestone —
   exactly the ADR-039/ADR-046 shape.
4. **Mechanical + human gate.** `buf breaking` / `make validate-spec` remain the CI gates;
   an intentional break is only mergeable with (a) the governing ADR removal clause and
   (b) a documented, file-scoped exception in the same PR.
5. **Pre-1.0 escape hatch.** While the project is pre-v1.0.0, the notice window may be
   compressed to a single minor release when an accepted ADR explicitly says so — the
   policy exists to make such compressions loud, not impossible.

Issue #1415 renders this policy as user-facing documentation once accepted.

---

## Rationale

| Option | Assessment |
|--------|------------|
| Status quo — `buf breaking` only + per-ADR clauses | ✗ Rejected — no externally visible promise; consumers reconstruct guarantees from CHANGELOG archaeology; fails CNCF-maturity expectations. |
| K8s-style N/N-1/N-2 parallel major surfaces served indefinitely | ✗ Deferred — strongest guarantee, but serving parallel majors is unaffordable at single-maintainer scale and premature pre-v1.0.0; revisit at Incubation. |
| **Written policy: stable v1, in-band markers, ≥1-release notice, milestone-boundary removals via ADR clauses** | ✅ Chosen — codifies the proven ADR-039/046 practice at near-zero process cost; checkable by adopters; scales upward later. |

---

## Consequences

- **Positive:** adopters get a written compatibility contract (enterprise/CNCF table
  stakes); M9-style removals become routine, citable process; release notes gain a
  standard "Deprecations" section; the M9.B/M9.A removals retroactively demonstrate
  compliance.
- **Negative / trade-off:** breaking changes acquire a mandatory waiting period —
  intentional friction; deprecation must be marked in up to three places (proto, spec,
  REST headers), which invites drift until linted.
- **Neutral / follow-up required:** #1415 writes the user-facing policy doc; add a
  release-notes "Deprecations" template section; consider a CI lint asserting that any
  `option deprecated` proto also has a removal-milestone comment; align the conformance
  suite (#1692) so published matrices always reference the surface version they prove.
