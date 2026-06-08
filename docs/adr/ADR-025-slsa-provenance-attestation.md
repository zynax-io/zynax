<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-025 — SLSA Provenance Attestation: Keep vs Disable

| Field | Value |
|-------|-------|
| **Status** | Accepted |
| **Date** | 2026-06-08 |
| **Deciders** | Oscar Gómez Manresa |
| **Scope** | Container image builds — `tools-image.yml`, `release.yml`, all `docker/build-push-action` steps |

---

## Context

`docker buildx` (via `docker/build-push-action`) attaches a SLSA provenance
attestation to every multi-arch build by default. In GHCR this appears as two
`unknown/unknown` rows in the Packages UI — one per platform variant of the
provenance manifest.

These rows are **not** broken or stray images. They are the expected SLSA Build
L1 provenance attestations produced by `buildkit` when `--provenance=mode=min`
(the default) is in effect. Contributors occasionally see them and assume something
is wrong.

### What provenance attestations are

A SLSA provenance attestation is an OCI image manifest of media type
`application/vnd.oci.image.manifest.v1+json` whose subject references the main
image digest. It encodes:

- The build invocation (GitHub Actions run ID, workflow ref, actor)
- The source repository and commit SHA
- The build platform (SLSA Build L1)

Because it is a separate OCI manifest rather than an image layer, GHCR displays
it as a row with no platform tag — hence `unknown/unknown` in the UI.

### Why this is a decision worth recording

Disabling provenance (`provenance: false` in the `docker/build-push-action` step)
is a one-way door: once the attestations stop appearing in GHCR, OpenSSF Scorecard
and other tools that verify them will begin penalising the project's score. Re-
enabling requires waiting for the next Scorecard evaluation cycle. This is exactly
the kind of irreversible architectural choice that warrants an ADR.

### Relationship to existing decisions

- **M6.C (#465, ADR-020)** — supply-chain hardening: cosign signing and SBOM
  generation were added to `release.yml` and `tools-image.yml`. Provenance
  attestations are a complementary (automatically generated) supply-chain artefact
  produced by the same build steps.
- **ADR-019 (SPDD)** — this issue (#868) is `docs:` type and SPDD-exempt.
- **#865** — depends on this decision: OCI manifest annotations (`org.opencontainers.image.*`)
  will be added to fix the missing-description issue in GHCR. That change is
  orthogonal to provenance attestations.
- **#869** — documentation of `unknown/unknown` rows for contributors; gated on
  this ADR.

---

## Decision

**Keep SLSA provenance attestations.** Do not set `provenance: false` anywhere
in the repository.

The `unknown/unknown` rows in the GHCR UI are a cosmetic issue. The fix is to
document them (via #869) and add OCI manifest annotations that give each real
image a description (via #865). Neither fix requires removing provenance.

---

## Rationale

| Option | Assessment |
|--------|------------|
| **Keep attestations** (chosen) | ✅ SLSA Build L1 provenance retained; `cosign verify-attestation` works; OpenSSF Scorecard benefits; `unknown/unknown` rows explained by documentation |
| Disable via `provenance: false` | ✗ Rejected — drops SLSA provenance; weakens Scorecard posture; `cosign verify-attestation` fails; one-way door with multi-cycle recovery |

Specific reasons for rejecting `provenance: false`:

1. This repo explicitly advertises OpenSSF Scorecard compliance and uses `cosign`
   for keyless signing. SLSA provenance attestations are exactly what those
   programmes reward.
2. Disabling provenance is a one-way door: once dropped and Scorecard score
   settles, recovering it requires re-opting in and waiting for the next score
   cycle.
3. The `unknown/unknown` confusion is a documentation problem, not a packaging
   defect. Fix it with docs and OCI annotations, not by removing the attestation.
4. SLSA provenance is generated for free by `docker/build-push-action`; there is
   no build-time or storage cost that would justify removing it.

---

## Consequences

### Positive

- SLSA Build L1 provenance remains on all images produced by `release.yml` and
  `tools-image.yml`.
- `cosign verify-attestation --type slsaprovenance <image>` continues to work for
  any image built after the M6.C supply-chain hardening PR (#489).
- OpenSSF Scorecard "Signed-Releases" and "SLSA" checks continue to benefit from
  the attestations.
- No workflow changes are required — the default `build-push-action` behaviour is
  already correct.

### Negative / trade-offs

- GHCR continues to display `unknown/unknown` rows until OCI annotations are
  added (#865) and/or GitHub improves their UI treatment of provenance manifests.
- Contributors may still be confused by the rows until #869 is merged (docs).

### Follow-up required

| Action | Issue |
|--------|-------|
| Add OCI manifest annotations to fix "no description" in GHCR | #865 |
| Document `unknown/unknown` rows as expected SLSA provenance | #869 |
| Add inline comments in workflow files pointing to this ADR | Done in this PR |
