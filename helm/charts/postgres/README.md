<!-- SPDX-License-Identifier: Apache-2.0 -->

# zynax-postgres

Thin, project-owned Postgres subchart around the **Docker Official Image**
`postgres` (ADR-026). Provisions a single cluster-level StatefulSet behind the
Service `<release>-postgresql`. task-broker, agent-registry, and memory-service
each use a dedicated schema (ADR-008 + ADR-021); Temporal owns its `temporal`
and `temporal_visibility` stores. No cross-schema access is permitted.

## Version pinning (ADR-024)

The chart runs **Postgres 17.10**, digest-pinned. The single source of truth is
`images/images.yaml`; the banner-marked region in `values.yaml`
(`postgresql.image.digest`) is stamped by tooling — **never edit it by hand**.

### Bumping a minor / patch version (e.g. 17.10 → 17.11)

Minor releases are in-place safe: the on-disk data format does not change
within a major series.

1. Update the `postgres` entry (tag + multi-arch index digest) in
   `images/images.yaml`.
2. Run `make sync-images` to stamp `values.yaml` (CI gates drift via
   `make check-images`).
3. Commit and let the gated `e2e smoke` workflow validate the rollout; the
   StatefulSet restarts onto the new image against the existing PVC.

Renovate proposes these bumps automatically through the same path.

## Major-version upgrades (e.g. 17.x → 18.x) — read before bumping

**Never bump the major version by editing `images.yaml` alone.** Postgres
majors change the on-disk data format: the new binary will refuse to start on
(or worse, could corrupt) a data directory initialized by an older major. The
PVC-backed StatefulSet means the old data directory survives pod restarts, so
an in-place image swap is **not** a migration. (Safeguard recorded in ADR-026
and the EPIC #1073 canvas; the ephemeral e2e cluster is exempt because every
gate run starts from an empty PVC.)

Safe paths, in order of preference for the current single-instance topology:

### 1. `pg_dump` / restore (recommended — simplest, brief downtime)

1. Take a full backup and stop writers (scale the consuming services and
   Temporal to zero).
2. Dump everything with the **new** major's client tools:
   `pg_dumpall -h <release>-postgresql -U postgres > dump.sql`
   (or per-database `pg_dump -Fc` for `task_broker` + the Temporal stores).
3. Deploy the new major into a **fresh PVC** (new release or delete the old
   PVC after the dump is verified), then restore:
   `psql -f dump.sql` / `pg_restore`.
4. Verify per-service schemas and roles (ADR-021), then scale services back up.

### 2. Logical replication (near-zero downtime)

Run old and new majors side by side (two releases of this chart), create a
`PUBLICATION` on the old primary and a `SUBSCRIPTION` on the new one, wait for
catch-up, then cut consumer DSNs over via the Secret/Service indirection.
Sequences and DDL are not replicated — sync them at cutover. Use when downtime
matters more than operational simplicity.

### `pg_upgrade` (in-place, use with care)

`pg_upgrade --link` is fast but requires both major binaries in one filesystem
namespace plus a second data directory — awkward inside the official image and
this thin chart. Prefer paths 1–2 until an operator (e.g. CloudNativePG, the
documented HA successor in ADR-026) owns upgrades.

### Temporal compatibility

The Temporal schema lives in this instance. Before any major bump, confirm the
pinned Temporal server release (see `helm/charts/temporal/Chart.yaml`) supports
the target Postgres major, and re-run the Temporal schema Job
(`temporal-sql-tool`) against the restored stores; it is idempotent and will
apply any pending schema migrations.

### After any upgrade

- Run `ANALYZE` (and `REINDEX` if collation/ICU changed) on restored databases.
- Confirm the `e2e smoke` gate is green and the per-service persistence tests
  pass before promoting beyond staging.
