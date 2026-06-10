<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-026 — Postgres Distribution and Target Major Version

| Field | Value |
|-------|-------|
| **Status** | Accepted |
| **Date** | 2026-06-10 |
| **Deciders** | Oscar Gómez Manresa |
| **Scope** | `helm/charts/postgres/`, `helm/charts/temporal/`, `helm/zynax-umbrella/`, `images/images.yaml`, all production Postgres deployments |
| **Related** | ADR-021 (Postgres-backed repos / schema ownership), ADR-024 (image source-of-truth) |

---

## Context

The `helm/charts/postgres` subchart pins
`docker.io/bitnami/postgresql:16.4.0-debian-12-r12`. In ~August 2025
Broadcom/Bitnami removed **all versioned tags** from the free
`docker.io/bitnami/*` namespace (only a rolling `latest` remains) and moved the
archived images to a frozen, read-only `docker.io/bitnamilegacy/*` mirror that
receives no further updates or CVE patches. The pinned tag now returns
`NotFound`, so any fresh Postgres deployment from the umbrella chart fails
`ImagePullBackOff`.

This forces a decision rather than a version bump:

- "Just bump the version" is not viable — every versioned tag under
  `docker.io/bitnami/*` is gone, so a new pin also 404s and only the
  non-reproducible moving `:latest` survives.
- An interim, **e2e-only** override to `bitnamilegacy/postgresql` shipped in
  PR #1069 to unblock the `e2e smoke` gate. `bitnamilegacy` is frozen and
  unpatched, so this is acceptable only for the ephemeral e2e cluster and is
  **removed at the end of the migration epic** (EPIC #1073, final O-step). It is
  not a production answer.

### Constraints that bound the choice

- **Reproducibility (ADR-024):** the production image must be digest-pinnable and
  registered in `images/images.yaml` as the single source of truth. A moving
  `:latest` tag is forbidden.
- **Schema ownership (ADR-021):** each consuming service keeps a dedicated schema
  (`task_broker`, `agent_registry`, `memory_service`); Temporal owns its default +
  visibility stores. The new distribution must preserve per-service credential
  Secrets and schema isolation — no shared superuser role at runtime in
  production. Consumers connect via DSN/Secret and must be unaffected at the DSN
  level.
- **Current stable major:** the project targets **Postgres 17.x** (current stable
  major series at decision time), moving off the end-of-the-road 16.x pin.
- **Maintenance posture:** the source must receive ongoing CVE patches without a
  paid subscription, to keep the project self-hostable and CNCF-aligned.
- **HA is out of scope now** (EPIC #1073 "WON'T"): no multi-AZ / cross-region
  replication is introduced in this migration, but the choice should not foreclose
  an HA roadmap.

### Relationship to prior decisions

- **EPIC #1073** (`docs/spdd/1073-postgres-migration/canvas.md`, Aligned) frames
  the three options and the trade-off axes; this ADR is its O-step 1 and records
  the binding choice for the remaining O-steps to build on.
- **ADR-021** governs schema ownership — unchanged by this migration.
- **ADR-024** governs image references — the chosen base image is added to
  `images/images.yaml` and propagated via `make sync-images`.

---

## Decision

**Adopt the official Postgres Docker Official Image (`postgres`) on the
current stable major series — target Postgres 17.x — consumed through a thin,
project-maintained `helm/charts/postgres` subchart, with the base image
digest-pinned via `images/images.yaml` (ADR-024).**

Concretely:

1. **Distribution:** the upstream **Docker Official Image** `postgres` (the
   `library/postgres` image maintained by the PostgreSQL Docker community), not
   an operator and not a paid registry. This image receives regular minor and
   CVE rebuilds and exposes the standard `POSTGRES_*` / `/docker-entrypoint-initdb.d`
   conventions that the existing init wiring can target.

2. **Target major version:** **Postgres 17.x** — the current stable major. The
   subchart pins the digest of a specific `17.x` tag; minor bumps flow through the
   normal `images.yaml` + Renovate path (ADR-024). No moving `:latest`.

3. **Chart:** keep a **thin, project-owned** `helm/charts/postgres` subchart
   (single `StatefulSet` + `Service` + `Secret` references) rather than depending
   on an external community chart. This avoids re-inheriting an upstream chart's
   release cadence and keeps the credential/init/volume conventions under our
   control, behind the existing Service name and Secret keys so consumer DSNs do
   not change.

4. **Reproducibility:** register the `postgres:17.x` base image in
   `images/images.yaml`; `make sync-images` stamps consumers; `make check-images`
   gates drift (ADR-024).

5. **Schema ownership unchanged (ADR-021):** per-service schemas and Secrets are
   preserved; the new image's init scripts create the same schemas/roles. No
   shared superuser role at runtime in production.

6. **Interim override removal:** the e2e `bitnamilegacy` override in
   `scripts/e2e/values-e2e.yaml` (PR #1069) is **temporary** and is removed in the
   final O-step of EPIC #1073 once production runs on the official image.

---

## Rationale

### Trade-off table

| Axis | (1) CloudNativePG operator | (2) Official `postgres:17` + thin chart **(chosen)** | (3) Bitnami Secure Images |
|------|-----------------------------|------------------------------------------------------|---------------------------|
| **Maintenance** | CNCF operator; maintained, but adds an operator+CRDs lifecycle to own and upgrade | ✅ Upstream Docker Official Image; community-maintained, regular CVE rebuilds; we own only a small chart | Maintained, but behind a **paid** subscription/registry — a recurring cost and a self-hosting barrier |
| **Reproducibility / digest-pinning (ADR-024)** | Image digest-pinnable, but the operator + CRD versions become an extra moving surface to pin | ✅ Single base image, cleanly digest-pinned in `images.yaml`; smallest reproducibility surface | Digest-pinnable, but pulls from a gated registry requiring auth in CI and clusters |
| **HA roadmap** | ✅ Best — built-in HA, failover, backups, PITR out of the box | Manual / future — single instance now; HA would need a later operator or external tooling | Conventions for HA exist but still tied to the paid product |
| **Migration cost (off current chart)** | High — replace the subchart with operator CRDs; re-wire Temporal + 3 services to operator-managed Services/Secrets; new failure modes | ✅ Lowest — keep the existing Service name and Secret keys; swap only `ImageSource` + init conventions; consumer DSNs unchanged | Low-moderate — keeps Bitnami chart conventions, but adds registry auth/licensing wiring everywhere |

### Why option 2 wins now

- **Lowest migration cost** matches the EPIC's explicit "WON'T introduce HA"
  scope: the migration replaces only `ImageSource`, keeping the Service name and
  Secret keys so Temporal datastores and the three Go services connect unchanged
  at the DSN level.
- **No paid dependency** keeps Zynax self-hostable and CNCF-aligned, ruling out
  option (3) despite its convenience of preserving Bitnami conventions.
- **Smallest reproducibility surface** under ADR-024 — one base image to pin,
  versus an operator plus CRD versions.
- **Does not foreclose HA:** CloudNativePG remains the documented future option if
  an HA roadmap is taken up; this ADR can be superseded at that point without
  reversing the data model, because schema ownership (ADR-021) and DSN-level
  contracts are unchanged.

### Why the alternatives were not chosen now

- **(1) CloudNativePG — deferred, not rejected.** It is the strongest HA answer
  and the natural successor if/when production HA is in scope. It is rejected
  *for this migration* because it imports an operator + CRD lifecycle and a larger
  re-wire than the deprecation fire-drill warrants, and HA is explicitly out of
  scope for EPIC #1073.
- **(3) Bitnami Secure Images — rejected.** Returning to the Bitnami ecosystem on
  a paid registry reintroduces a vendor/licensing dependency and CI/cluster
  registry-auth burden, conflicting with the self-hostable, CNCF-aligned posture.

---

## Consequences

### Positive

- The deprecation fire-drill is resolved with the **smallest possible change**:
  swap the image source and init conventions; consumers connect unchanged.
- Production runs on a **maintained, CVE-patched, digest-pinnable** image with no
  `docker.io/bitnami/*` or `bitnamilegacy/*` reference remaining.
- Postgres moves to the **current stable major (17.x)**.
- No new paid dependency, no operator/CRD lifecycle to own at this milestone.
- Minor/CVE bumps follow the established `images.yaml` + Renovate path (ADR-024).

### Negative / trade-offs

- **No built-in HA.** The thin chart deploys a single instance; high
  availability, failover, and managed backups are not provided. Stateful
  production environments must arrange backups/replication separately until an HA
  decision is made.
- **We own the chart.** A thin project-maintained subchart means we carry the init
  / volume-permission / probe wiring ourselves rather than inheriting it from an
  upstream chart.
- **Major-version upgrades are not in-place.** A future major bump on a stateful
  production volume requires a documented `pg_upgrade`/dump-restore migration path
  (the ephemeral e2e cluster is exempt). EPIC #1073 O-step 8 documents this.

### Out of scope

| Item | Reason |
|------|--------|
| Postgres HA / multi-AZ / cross-region replication | Deferred (EPIC #1073 "WON'T"); revisit via CloudNativePG if HA is taken up |
| Changes to per-service schema ownership | ADR-021 stands unchanged |
| Re-touching the interim e2e override beyond its removal | Already delivered in PR #1069; removed in EPIC #1073 final O-step |

### Follow-up required

| Action | Tracking |
|--------|---------|
| Spike `postgres:17` in the subchart behind existing Service/Secret keys | EPIC #1073 O-step 2 |
| Register the base image in `images/images.yaml`; `make sync-images` | EPIC #1073 O-step 5 (ADR-024) |
| Remove the e2e `bitnamilegacy` override | EPIC #1073 O-step 7 |
| Document the major-version upgrade/migration note | EPIC #1073 O-step 8 |
| Re-evaluate CloudNativePG if an HA roadmap is adopted | Supersedes this ADR if taken |
