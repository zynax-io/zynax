# Learnings: CI / Release Engineer

> Format: each entry has `Seen in:` (issue/session) and `Date:` (YYYY-MM-DD).
> Updated by `/m6-learn` after each batch. Human-reviewed before merging.

---

## Effective patterns

- **`images/images.yaml` is the SoT for all image references — never hardcode tags in workflows.**
  The drift-check gate (`cmd/zynax-ci images check`) runs on every PR and fails if any
  workflow file references an image tag that diverges from `images.yaml`.
  Seen in: M6.Images #856–#858. Date: 2026-06-06.

- **`cache-from: type=gha` on all `docker/build-push-action` steps cuts build time by 60–80%.**
  Without the GHA cache, each CI run rebuilds all layers from scratch.
  Always set `cache-to: type=gha,mode=max` as well.
  Seen in: M5.F CI sprint #542. Date: 2026-06-06.

- **All CI jobs run in the `ci-runner` container — never install tools in `run:` steps.**
  The container has Go, buf, cosign, yq, docker, helm, and Python pre-installed.
  Installing tools in `run:` adds 2–5 min per job and diverges from the container spec.
  Seen in: #552 (switch to ci-runner container mode) PR #850. Date: 2026-06-06.

- **`provenance: true` + `sbom: true` on `docker/build-push-action` generates SLSA attestations.**
  These appear in GHCR as `unknown/unknown` manifests — this is correct and expected (ADR-025).
  Do NOT add `provenance: false` to suppress them.
  Seen in: M6.C #489 PR #833. Date: 2026-06-06.

- **buf breaking against `main` catches proto backward-incompatibility before merge.**
  Field removals and number changes fail immediately. Adding fields (new field numbers) is safe.
  Always set `against: https://github.com/zynax-io/zynax.git#branch=main`.
  Seen in: M1 proto contracts gate. Date: 2026-06-06.

---

## Edge cases discovered

- **GHCR package names with slashes need URL encoding (`%2F`) in `gh api` calls.**
  `gh api /orgs/zynax-io/packages/container/zynax%2Fapi-gateway/versions` works.
  `gh api /orgs/zynax-io/packages/container/zynax/api-gateway/versions` returns 404.
  Seen in: M6.Images verification step design. Date: 2026-06-06.

- **`buf breaking` produces exit code 1 for any breaking change, including deprecations.**
  Deprecating a field (marking it `deprecated = true`) without removing it is NOT breaking,
  but buf may still warn. Use `buf breaking --error-format=json` to distinguish errors from warnings.
  Seen in: M1 proto gate. Date: 2026-06-06.

- **`cosign sign` requires `COSIGN_EXPERIMENTAL=1` and the workflow must run with `id-token: write`.**
  Missing the permission causes a cryptic OIDC error: "failed to get OIDC token".
  The permission must be on the job-level `permissions:` block, not just the workflow-level one.
  Seen in: M6.C #489 PR #833. Date: 2026-06-06.

- **Parallel matrix jobs that all push to the same GHCR tag race and corrupt manifests.**
  When building multi-arch images, always use `docker/build-push-action` with
  `platforms: linux/amd64,linux/arm64` in a SINGLE step — not two separate matrix jobs.
  Seen in: M5.F release pipeline. Date: 2026-06-06.

---

## Failed approaches

- **Using `merge-queue` as a required CI gate.**
  Removed in #544 because it caused false positives and blocked valid PRs.
  The current gate model is: branch protection + required checks + auto-merge.
  Seen in: M5 BATCH 0. Date: 2026-06-06.

---

## Proposed expert prompt updates

*(none yet — populate after first batch of CI/release expert sessions)*
