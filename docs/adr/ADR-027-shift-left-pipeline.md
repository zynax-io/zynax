<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-027 — Shift-Left Pipeline Model: Build Once Pre-Merge, Promote by Retag

| Field | Value |
|-------|-------|
| **Status** | Accepted |
| **Date** | 2026-06-11 |
| **Deciders** | Oscar Gómez Manresa |
| **Scope** | `ci.yml` build-images job, `release.yml` retag jobs, GHCR staging lane — EPIC #1109 |
| **Related** | ADR-023 (direct-push restriction — explicit exception recorded here), ADR-024 (image source of truth), ADR-025 (SLSA provenance) |

---

## Context

Today container images are built **after** merge: `release.yml` rebuilds every
service from the merge commit, and the security scan (`make security`, Trivy)
runs against artefacts that are *not* the artefacts that ship. This creates
three structural problems:

1. **Scan ≠ deploy.** The binary that passed the pre-merge security gate is not
   the binary that ends up in production — the post-merge rebuild produces a
   different artefact (new timestamps, possibly newer base layers, possibly a
   newer transitive dependency resolved at build time).
2. **Rebuild nondeterminism.** Two builds of the same commit are not guaranteed
   to be byte-identical. Any post-merge build is an unreviewed variable.
3. **Digest drift (ADR-024).** The `images/images.yaml` digest update after a
   release is a separate, human-driven corrective action. Between the rebuild
   and the digest bump, the source of truth is stale.

The fix is to shift the build left of the merge: build exactly once, gate the
merge on the scan of that exact artefact, and *promote* it — never rebuild it.

This decision is Part 2 of the CI/CD + delivery overhaul
(`docs/contributing/ci-delivery-overhaul-prompt.md` §Part 2, EPIC #1109,
parent #1107) and is recorded **before any workflow code is written**, because
the retag model is a one-way door: once `release.yml` no longer contains a
build step, reversing the model means re-introducing the rebuild
nondeterminism this ADR exists to eliminate.

---

## Decision

Container images are **built exactly once**, in the PR's pre-merge CI, and
promoted through their lifecycle by **retagging — never rebuilding**.

1. **Pre-merge build (ci.yml `build-images`).** Every Docker-touching PR builds
   each affected service image and pushes it to a **public staging lane** in
   GHCR: `ghcr.io/zynax-io/zynax/staging/<svc>:pr-<sha>`.

2. **Pre-merge security gate.** The staging image is scanned with **Trivy at
   CRITICAL,HIGH severity** with `exit-code: 1` — a finding blocks the merge.
   Accepted-risk upstream CVEs go in a `.trivyignore` file where every entry
   carries a dated expiry (`accepted-until: YYYY-MM-DD reason: …`), reviewed
   quarterly. There are no undated exceptions.

3. **On merge: retag, not rebuild.** A `workflow_run`-triggered job (fires on
   the PR CI workflow completing against `main`) retags the staging image to
   `<svc>:main-<sha>` and `<svc>:latest`. The PR head SHA needed to locate the
   staging image is recovered from **`github.event.workflow_run.head_sha`**.
   The alternative — reading an OCI label stamped on the staging image — was
   **rejected as circular**: locating the image to read its label already
   requires the SHA the label would provide.

4. **On version tag: retag again.** A `v*.*.*` tag retags the already-promoted
   image to the release version. At no point after the pre-merge build is any
   image rebuilt. **The image in production is the exact binary that passed
   the pre-merge security gate.**

### Locked operator decisions (#1107, pre-flight 2026-06-11)

| # | Question | Decision |
|---|----------|----------|
| 1 | GHCR staging-lane visibility | **Public** — consistent with existing packages |
| 2 | Trivy blocking severity | **CRITICAL,HIGH** — matches `make security` standard |
| 3 | `.trivyignore` exceptions | **Yes, with `accepted-until: YYYY-MM-DD reason:` expiry**, quarterly review |
| 4 | Digest update on merge | **Direct `[skip ci]` bot commit to main** (Flux/ArgoCD model) — needs ADR-023 exception for the Actions bot |
| 8 | PR head SHA recovery in retag | **`workflow_run` trigger** (`github.event.workflow_run.head_sha`) — the OCI-label option is circular and was rejected |

---

## Rationale

| Option | Assessment |
|--------|------------|
| **Build once pre-merge, promote by retag** (chosen) | ✅ Scan == deploy; no rebuild nondeterminism; post-merge promotion is a ~10 s manifest operation instead of a ~5 min build |
| Keep post-merge rebuild in `release.yml` | ✗ Rejected — the shipped artefact is never the scanned artefact; every release re-introduces an unreviewed build variable |
| Build pre-merge *and* rebuild post-merge ("verify twice") | ✗ Rejected — doubles CI cost and still ships the second, unscanned build; the duplication is the problem, not a mitigation |
| OCI label on the staging image for head-SHA recovery | ✗ Rejected — circular: finding the image to read the label already requires the SHA; `workflow_run.head_sha` provides it directly from the event |

---

## Consequences

### Positive

- **Supply-chain integrity:** the scanned image and the deployed image are the
  same digest — scan == deploy, by construction.
- **No rebuild nondeterminism:** nothing is built after the merge gate; the
  artefact lineage is a single build with an auditable promotion chain
  (`pr-<sha>` → `main-<sha>`/`latest` → `v*.*.*`).
- **Faster post-merge pipeline:** a retag is a manifest-only operation (~10 s)
  versus a full multi-arch build (~5 min per service).

### Negative / trade-offs

- **Staging accumulation:** every Docker-touching PR pushes images to the
  staging lane. A PR-close cleanup job (spec §3B, issue #1119) is required to
  delete `staging/<svc>:pr-<sha>` images when their PR closes.
- **`workflow_run` trigger dependency:** the retag job depends on GitHub's
  `workflow_run` event semantics (it fires from the default branch and must
  filter for successful runs on `main`). A change in those semantics, or a
  skipped CI run, leaves a merge without a promoted image until re-run.

---

## Relationship to ADR-024 (image reference management)

The retag model **replaces the post-merge build**. The `images/images.yaml`
digest update moves from a separate, human-driven step to an **atomic
`[skip ci]` bot commit on every merge**: the same `workflow_run` job that
retags the image immediately commits the new digest to `images.yaml` (and runs
the sync stamp), so the source of truth is updated in the same promotion
transaction. **Digest drift becomes structurally impossible** — there is no
window in which a promoted image exists without its digest recorded.

### Explicit exception to ADR-023

ADR-023 forbids direct pushes to `main` (all changes: branch → PR → CI green →
squash-merge). The digest bot commit is a **deliberate, recorded exception**
to that rule:

- **Who:** the `github-actions[bot]` identity only, from the retag
  `workflow_run` job.
- **What:** only the `images/images.yaml` digest update and its banner-stamped
  consumer regions (`make sync-images` output) — never source code.
- **Why it is safe:** the commit is mechanically derived from an image that
  already passed the full pre-merge gate; it carries `[skip ci]` because
  re-running CI on a generated digest stamp adds no information. This is the
  standard Flux/ArgoCD image-automation model.

ADR-023 otherwise stands unchanged for all human and AI contributors.

---

## Relationship to ADR-025 (SLSA provenance)

SLSA Build L1 provenance attestations continue to be generated by
`docker/build-push-action` defaults on the **pre-merge** build (ADR-025
forbids `provenance: false`). The retag step **must preserve the staging
image's attestation manifests**: promotion uses
`docker buildx imagetools create`, which copies the **full manifest list** —
platform manifests *and* attestation manifests — to the new tag. A retag
mechanism that copies only the platform images (dropping the attestations)
would silently break `cosign verify-attestation` on promoted tags and is not
acceptable.

---

### Follow-up required

| Action | Tracking |
|--------|---------|
| Pre-merge `build-images` job (staging + Hadolint + Trivy + SBOM) | #1118 (spec §3A) |
| PR-close staging-image cleanup workflow | #1119 (spec §3B) |
| Convert `release.yml` to the retag model + atomic digest sync | #1120 (spec §3C) |
| Supply-chain hardening (SHA pins, permissions, SLSA L2) | #1116 (spec §3D) |
| Reconcile existing M6 issues with the new model | #1121 (spec §Part 4) |
