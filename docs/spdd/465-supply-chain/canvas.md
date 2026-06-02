<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M6.C Supply Chain Hardening

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #465
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-18
**Status:** Implemented

**Child issues:** #489 (cosign + SBOM + multi-arch)

---

## R — Requirements

**Problem:** The 2026-05 architectural review (§8.3, §12.2) notes strong supply-chain posture (SHA-pinned actions, Renovate, govulncheck) but three concrete gaps: no cosign signing, no SBOM generation in releases, no multi-arch container builds. These gaps block SLSA L3 compliance and are expected by CNCF Sandbox reviewers.

**Definition of done:**
- Every container image release is verifiable with `cosign verify`.
- An SPDX SBOM is attached to each GitHub release.
- Images are buildable and functional on both linux/amd64 and linux/arm64.
- `cosign verify` command documented in SECURITY.md.

---

## E — Entities

- **cosign** — `sigstore/cosign`; signs container image digests and CLI binaries using the Sigstore transparency log (keyless signing via OIDC in GitHub Actions).
- **syft** — `anchore/syft`; generates SBOM in SPDX format from container images.
- **SLSA provenance** — `--provenance=true` flag on `docker buildx build`; generates a provenance attestation attached to the image manifest.
- **Multi-arch manifest** — OCI image index listing linux/amd64 and linux/arm64 layers under a single tag.
- **`docker buildx`** — Docker BuildKit cross-compilation toolchain; uses QEMU emulation for non-native architectures.

---

## A — Approach

**What we WILL do:**
- Add `sigstore/cosign-installer` to `tools-publish.yml` and `cli-release.yml`; sign each image digest and binary after push.
- Add `anchore/syft` SBOM generation; attach SPDX output as GitHub release asset and OCI attestation.
- Convert `docker build` to `docker buildx build --platform linux/amd64,linux/arm64 --provenance=true --sbom=true`.
- Document `cosign verify` in SECURITY.md.

**What we WON'T do:**
- Implement SLSA L4 (hermetic builds) — that requires significant infrastructure investment.
- Sign every commit (GPG signing is already required per CONTRIBUTING.md; cosign is for release artifacts).

**ADR references:**
- ADR-005: Apache 2.0 license — all tools used must be license-compatible.

---

## S — Structure

**Files touched:**
- `.github/workflows/tools-publish.yml` — add QEMU, buildx, cosign, syft steps
- `.github/workflows/cli-release.yml` — add cosign signing for binaries
- `SECURITY.md` — add `cosign verify` instructions

---

## O — Operations

1. **[#489]** Add cosign signing, syft SBOM generation, and multi-arch buildx to release workflows; document verification in SECURITY.md.

---

## N — Norms

- `ci:` PR type (CI workflow changes).
- All new GitHub Actions must be pinned to SHA (existing project standard).
- cosign signing uses keyless OIDC in GitHub Actions — no private keys stored in secrets.

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content
- [x] No PII
- [x] No prompt injection
- [x] All entities are public-safe abstractions
- [x] /spdd-security-review passed

### Feature Safeguards
- Never store cosign private keys as GitHub secrets — use keyless OIDC signing only.
- Never disable `--provenance` after enabling it — provenance is an additive guarantee.
