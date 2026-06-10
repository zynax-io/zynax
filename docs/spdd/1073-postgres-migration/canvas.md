# REASONS Canvas — Migrate Postgres off Deprecated Bitnami Images

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #1073
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-10
**Status:** Draft

---

## R — Requirements

> Problem statement: what breaks or is missing without this feature?

The `helm/charts/postgres` subchart pins `docker.io/bitnami/postgresql:16.4.0-debian-12-r12`.
In ~August 2025 Broadcom/Bitnami removed **all versioned tags** from the free
`docker.io/bitnami/*` namespace (only a rolling `latest` remains) and moved the archived
images to a frozen, read-only `docker.io/bitnamilegacy/*` mirror that receives no further
updates or CVE patches. The pinned tag now returns `NotFound`, so any fresh Postgres
deployment from the umbrella chart fails `ImagePullBackOff`.

An interim, e2e-only override to `bitnamilegacy/postgresql` shipped in PR #1069 to unblock the
`e2e smoke` gate, but `bitnamilegacy` is frozen and unpatched — unacceptable for production.
Staying on Bitnami and "just bumping the version" is not viable: every versioned tag is gone,
so a new pin also 404s and only the non-reproducible moving `:latest` remains.

> Definition of done: observable outcomes that confirm delivery.

- Production Postgres runs on a **maintained image source** with reproducible, digest-pinnable,
  CVE-patched tags (no `docker.io/bitnami/*` or `bitnamilegacy/*` references remain).
- Postgres is on a **current stable major version** (target: Postgres 17.x).
- Temporal schema bootstrap, per-service schema ownership (ADR-021), credential Secrets, and
  TLS wiring continue to function; persistence tests for task-broker / agent-registry /
  memory-service pass.
- The `e2e smoke` gate passes **without** the `bitnamilegacy` override (the override is removed
  from `scripts/e2e/values-e2e.yaml`).
- An ADR records the chosen Postgres distribution and version strategy.

## E — Entities

> Domain entities and their relationships. Tier 1 only.

```
PostgresDeployment
  ├── ImageSource        (maintained registry + digest-pinnable tag)
  ├── MajorVersion       (target: 17.x)
  ├── CredentialSecret   (admin password + per-service passwords)
  ├── SchemaOwnership    (ADR-021: task_broker / agent_registry / memory_service)
  └── consumed by:
        ├── TemporalDatastore   (default + visibility stores)
        ├── TaskBrokerRepo
        ├── AgentRegistryRepo
        └── MemoryServiceVectorStore
```

The migration replaces only **ImageSource** (and possibly the chart conventions around it);
the logical relationships to consumers must be preserved.

## A — Approach

> Solution strategy. What we WILL do and what we WON'T do. Reference governing ADRs.

**WILL:**
- Evaluate three distribution options in a new ADR and pick one:
  1. **CloudNativePG** operator (CNCF, production HA/backup, declarative).
  2. **Official `postgres:17`** Docker Official Image + a thin maintained chart.
  3. **Bitnami Secure Images** (paid registry, keeps Bitnami conventions).
- Re-wire the chosen image's credential / init / volume conventions so the existing consumers
  (Temporal datastores, the three Go services) connect unchanged at the DSN level.
- Land on a current stable Postgres major version and make the image digest-pinnable via the
  `images.yaml` SoT (ADR-024).
- Remove the e2e `bitnamilegacy` override once production is migrated.

**WON'T:**
- Introduce Postgres HA / multi-AZ / cross-region replication (defer unless the chosen option
  provides it for free).
- Change the per-service schema-ownership model (ADR-021 stays).
- Re-touch the interim e2e override beyond removing it at the end (already delivered in #1069).

Governing ADRs: ADR-021 (schema ownership), ADR-024 (image source-of-truth), plus a **new ADR**
this epic must produce for the Postgres distribution + version decision.

## S — Structure

> System placement. Services, packages, files, contracts touched.

- `helm/charts/postgres/` — image source, chart dependency (or replacement), value structure,
  secret keys, init/volume-permission wiring.
- `helm/charts/temporal/values.yaml` — datastore `connectAddr` / driver if service naming or
  auth conventions change.
- `helm/zynax-umbrella/` — dependency graph + packaged subchart `.tgz` if the postgres subchart
  is replaced.
- `images/images.yaml` — add the new (base) Postgres image to the SoT; `make sync-images`.
- `scripts/e2e/values-e2e.yaml` — remove the `bitnamilegacy` override (final step).
- `docs/adr/` — new ADR for the distribution + version decision.
- No gRPC contracts change; consumers connect via DSN/Secret as today.

## O — Operations

> Ordered, concrete, testable implementation steps. Each = one reviewable PR/commit.

1. **ADR: Postgres distribution + version decision.** Compare CloudNativePG vs official-image
   chart vs Bitnami Secure Images against maintenance, reproducibility, HA roadmap, and
   migration cost; record the choice and target major version (17.x). (`docs:` — no code.)
2. **Spike the chosen image in the postgres subchart** behind the existing Service name and
   Secret keys; render with `helm template` and prove the credential/init wiring is intact.
3. **Re-wire credential Secrets + init** for the new image (admin + per-service passwords,
   ADR-021 schemas), keeping consumer DSNs unchanged.
4. **Update Temporal datastore wiring** if the new Service name / auth differs; verify the
   schema Job (`useHelmHooks: false`, set in #1069) still bootstraps both stores.
5. **Register the new image in `images/images.yaml`** (base image) and run `make sync-images`;
   verify the drift gate.
6. **Bring up the full umbrella on kind** (reuse `scripts/e2e/cluster-up.sh`) and assert all
   services + Temporal reach a healthy rollout on the new Postgres.
7. **Remove the e2e `bitnamilegacy` override** from `scripts/e2e/values-e2e.yaml`; confirm the
   `e2e smoke` gate is green end-to-end on the migrated source.
8. **Document a major-version upgrade/migration note** (data considerations for stateful
   environments) in the postgres chart README / ops runbook.

## N — Norms

> Cross-cutting standards. Pull from AGENTS.md Hard Constraints + layer norms.

- Commit hygiene: `Signed-off-by` (DCO) + `Assisted-by` trailer; never `Co-Authored-By` for AI.
- Conventional commit types only (feat/fix/refactor/docs/test/ci/chore); scope = directory.
- PR size: ≤ 200 ideal / 201–400 acceptable; generated `.tgz`/lock files excluded.
- Image versions are managed via `images/images.yaml` (ADR-024) — no hand-edited banner regions;
  use `make sync-images` / `make check-images`.
- Helm: `helm lint` + `ct lint` clean; chart changes trigger the (gated) `e2e smoke` workflow.
- One commit per logical change; one PR per story; squash-merge with required signatures.

## S — Safeguards (second S)

> Non-negotiable constraints. Things that MUST NEVER happen.

### Context Security (mandatory before committing this Canvas)
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no email addresses
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file (2026-06-10 — verdict WARN: clean content, Status Draft pending human alignment)

### Feature Safeguards
- Never reintroduce a reference to `docker.io/bitnami/*` or `docker.io/bitnamilegacy/*` in any
  production chart path (the whole point of the epic).
- Never pin a Postgres image to a **moving** tag (`latest`); always a reproducible, digest-pinned
  version registered in `images.yaml` (ADR-024).
- Never break ADR-021 schema ownership: each service keeps its dedicated schema; no cross-schema
  access; no shared application role with superuser at runtime in production.
- Never store DB passwords/DSNs in chart `values.yaml`; always via K8s Secrets (ADR-021).
- Never perform an in-place major-version bump on a stateful production volume without a
  documented migration path (the e2e cluster is ephemeral and exempt).
