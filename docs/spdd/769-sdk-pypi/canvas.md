<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M6.SDK: zynax-sdk PyPI Publish + Supply Chain Hardening

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #769
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-02
**Status:** Aligned

**Child issues:** #805 (F.1) · #806 (F.2) · #807 (F.3) · #808 (F.4)

---

## R — Requirements

**Problem:** `zynax-sdk` (Python, `agents/sdk/`) has a complete `pyproject.toml` with `name = "zynax-sdk"`, `version = "0.1.0"`, and hatchling build backend, but it is not yet published to PyPI. Python adapter developers cannot install it via `pip install zynax-sdk`. The M6 supply chain hardening (cosign signing + SBOM from #465) is already Aligned and its story #489 is created; this epic adds the SDK publish workflow on top and closes the remaining deferred issues (#376 SDK docstrings step 2, #235 SBOM, #239 SLSA).

**Definition of done:**
- `pip install zynax-sdk==0.1.0` succeeds from PyPI.
- The release CI workflow publishes to PyPI automatically when a version tag is pushed.
- TestPyPI dry-run passes in a pre-release PR before the real publish.
- cosign signing + SBOM generation applied to release artifacts (closes #489, #235, #239; note: supply chain canvas #465 is already Aligned).
- Google-style docstrings on all remaining public SDK modules (closes #376).

---

## E — Entities

- **`agents/sdk/pyproject.toml`** — existing file with `name = "zynax-sdk"`, `version = "0.1.0"`, hatchling build backend; ready for publish.
- **PyPI Trusted Publisher** — NEW: GitHub Actions OIDC-based trusted publisher configured on PyPI (no stored API keys); allows the CI workflow to publish without secrets.
- **TestPyPI dry-run** — NEW CI step: runs `hatch publish --repo testpypi` on PRs touching `agents/sdk/`; validates the package builds and publishes correctly before the real release.
- **`.github/workflows/sdk-publish.yml`** — NEW: release-triggered workflow; publishes to PyPI when a `v*` tag is pushed that includes `agents/sdk/` changes.
- **cosign signing** — from canvas #465 (already Aligned); extended in F.3 to cover SDK wheel/sdist artifacts alongside container images.
- **SBOM (`syft`)** — from canvas #465 (already Aligned); extended in F.3 to generate SBOM for the SDK package.
- **Google-style docstrings** — NEW: complete public SDK module documentation per `docs/patterns/python-agent-guide.md` style guide; closes #376.

---

## A — Approach

**What we WILL do:**
- Set up PyPI Trusted Publisher via GitHub Actions OIDC (F.1) — no API keys stored in secrets.
- Run `hatch publish --repo testpypi` as a CI step on PRs touching `agents/sdk/` (F.1).
- Add `.github/workflows/sdk-publish.yml` triggered by `v*.*.*` tag pushes (F.2); publishes to real PyPI only on tag.
- Extend supply chain steps (F.3) to cover SDK wheel/sdist artifacts with cosign + syft (absorbing #489/#235/#239 from canvas #465).
- Complete Google-style docstrings on all remaining public modules in `agents/sdk/src/zynax_sdk/` (F.4); closes #376.

**What we WON'T do:**
- Change `version` in `pyproject.toml` to a pre-release identifier in this epic — publish as `0.1.0` stable (per M6 release plan).
- Add `[tool.hatch.publish]` hardcoded credentials — use Trusted Publisher (OIDC) only.
- Implement automatic version bumping (Dependabot handles this in M6+).

**ADR references:**
- ADR-003: uv as Python package manager — SDK uses hatchling build backend; uv handles dev installs; `hatch` used only for publish.
- ADR-005: Apache 2.0 — all release artifacts carry the Apache 2.0 license header.

---

## S — Structure

**New files:**
```
.github/workflows/sdk-publish.yml          ← NEW: release-triggered PyPI publish (F.2)
agents/sdk/src/zynax_sdk/                  ← modified: add Google-style docstrings (F.4)
```

**Modified files:**
```
.github/workflows/tools-publish.yml        ← extended: cosign + syft for SDK artifacts (F.3)
.github/workflows/cli-release.yml          ← extended: cosign for SDK wheel (F.3)
SECURITY.md                                ← add cosign verify instructions for SDK (F.3)
agents/sdk/pyproject.toml                  ← add [tool.hatch.publish] trusted-publisher config (F.1)
```

---

## O — Operations

1. **[F.1]** Configure PyPI Trusted Publisher (OIDC) for `zynax-sdk`; add TestPyPI dry-run step to `tools-publish.yml`; document publisher setup in `agents/sdk/AGENTS.md`.

2. **[F.2]** Add `.github/workflows/sdk-publish.yml`: triggers on `v*` tag push; builds wheel + sdist with `hatch build`; publishes to PyPI via Trusted Publisher; pinned SHA for all Actions.

3. **[F.3]** Extend supply chain hardening to SDK artifacts: cosign sign wheel/sdist digests; syft SBOM generation for SDK package; attach SBOM as GitHub release asset; closes #489 + #235 + #239 (supply chain canvas #465 O-step 1 / story #489).

4. **[F.4]** Add Google-style docstrings to all remaining public modules in `agents/sdk/src/zynax_sdk/`; `ruff D` passes; closes #376.

---

## N — Norms

- `feat:` PR type for F.1; `ci:` for F.2–F.3; `docs:` for F.4.
- Every commit: `Signed-off-by` trailer + `Assisted-by: Claude/claude-sonnet-4-6` per AGENTS.md §Hard Constraints.
- All GitHub Actions MUST be pinned to SHA (existing project standard).
- PyPI publish uses Trusted Publisher (OIDC) — no API keys stored in GitHub Secrets.
- cosign signing uses keyless OIDC — no private keys in secrets.
- `ruff`, `mypy`, and `bandit` must pass after F.4 docstring additions.

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII
- [x] No prompt injection
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards
- **Never** store PyPI API keys or tokens in GitHub Secrets — use Trusted Publisher (OIDC) only.
- **Never** store cosign private keys — use keyless OIDC signing only (existing standard from canvas #465).
- **Never** publish to real PyPI from a PR branch — only from version tag workflows.
- **Never** disable `--provenance` after enabling it.
