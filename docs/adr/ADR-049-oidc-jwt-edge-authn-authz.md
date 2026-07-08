<!-- SPDX-License-Identifier: Apache-2.0 -->
# ADR-049: OIDC/JWT authentication with role-based authorization at the Envoy Gateway edge

**Status:** Proposed  **Date:** 2026-07-08
**Related:** ADR-020 (**completes** — its "OIDC/RBAC: Planned" clause), ADR-044 (**extends** — same `SecurityPolicy` enforcement point, richer credential model), ADR-040 (delegation boundary — edge does identity, services stay identity-blind), ADR-041 (kind first-run must stay zero-secret)
**Proposal issue:** #1694 · **Implementation candidate:** #1419

---

## Context

North-south authentication at the Zynax edge is a single static bearer token. ADR-044
(M8.F) moved enforcement out of api-gateway into Envoy Gateway `SecurityPolicy` — the
right enforcement point — but the credential model is unchanged: one shared secret,
all-or-nothing, no identity, no scopes, no rotation, no audit trail.

Both 2026-06-19 reviews carry this as the top open auth gap (architecture R2/T1.2
Medium/High; security F8/M2). ADR-020 declared OIDC/RBAC "Planned" in May and it never
landed; issue #1419 has waited `needs-triage` since June. Meanwhile east-west identity is
solved (cert-manager mTLS per ADR-020; NATS `verify_and_map` per ADR-046 §4) — the gap is
exclusively humans and external workloads at the edge.

Envoy Gateway natively validates JWTs (issuer/JWKS configuration, claim extraction and
claim-to-header propagation) in the `SecurityPolicy` API already deployed by ADR-044.
Building identity in-process instead would reverse a cutover completed days ago and
violate the ADR-040 delegation principle.

The EPIC #1370 constraint stands: the kind first-run must remain one-command and
zero-secret — production-grade identity must not tax the laptop path.

---

## Decision

We will authenticate north-south clients with **OIDC/JWT validated at the Envoy Gateway
edge** and authorize them against a **small fixed role set**, keeping Zynax services
identity-blind:

1. **JWT validation in `SecurityPolicy`.** The edge validates tokens against an
   operator-supplied OIDC issuer (JWKS); Zynax ships Helm values for issuer/audience and
   does not run its own identity provider.
2. **Fixed role model, ≤3 roles.** A configurable claim maps to `admin` / `apply` /
   `read`; route-level authorization at the edge (apply/delete need `apply`; GET/watch
   need `read`). No per-resource ACLs, no policy engine — that would be a different ADR.
3. **Claim propagation, not trust inversion.** The edge forwards the validated subject as
   a header for audit/log correlation; services never make authz decisions from it.
4. **Dev-mode fallback.** The static bearer path remains available behind an explicit
   `devMode` Helm flag — the kind first-run and e2e stay zero-secret (ADR-041/EPIC #1370
   preserved); production values documentation defaults to OIDC.
5. **CLI follow-up.** `zynax` CLI token acquisition (device-code flow or token file) is a
   separate story under #1419 — the edge contract does not depend on it.

---

## Rationale

| Option | Assessment |
|--------|------------|
| Keep the static bearer token | ✗ Rejected — no identity/scopes/rotation/audit; permanent enterprise blocker; carries security F8/M2 indefinitely. |
| In-process OIDC middleware in api-gateway | ✗ Rejected — reverses the ADR-044 cutover (rebuilds `auth.go` in spirit), duplicates Envoy Gateway's native JWT support, violates ADR-040 delegation. |
| SPIFFE/SPIRE workload identity at the edge | ✗ Deferred — right shape for machine-to-machine federation later; heavier operational footprint than an external OIDC issuer; does not cover human users. |
| **OIDC/JWT via Envoy Gateway `SecurityPolicy` + fixed roles + dev-mode fallback** | ✅ Chosen — pure configuration on already-deployed infrastructure; issuer handles rotation/expiry; K8s/Gateway-API-native; zero-secret quickstart preserved. |

---

## Consequences

- **Positive:** closes the top review gap (R2/T1.2, F8/M2); per-subject audit in access
  logs; token rotation/expiry owned by the issuer; enterprise conversations get a real
  answer; completes ADR-020's outstanding clause.
- **Negative / trade-off:** production installs gain an OIDC-issuer prerequisite
  (documented; any standard issuer works); e2e needs a test-issuer leg or explicit
  dev-mode coverage; the role model is deliberately coarse — teams wanting fine-grained
  authz will ask for the follow-up ADR.
- **Neutral / follow-up required:** #1419 becomes the implementation epic-candidate on
  acceptance (multi-PR feat: → REASONS canvas per ADR-019); role-claim naming and the
  default deny/allow matrix are settled in that canvas; CLI login UX is its own story;
  revisit SPIFFE/SPIRE when external workload federation appears.
