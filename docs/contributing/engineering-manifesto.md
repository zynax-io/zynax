# Zynax Engineering Manifesto

> **Status:** Living document (not an ADR — mutable; see locked decision in EPIC #1107).
> **Last verified against the pipeline:** 2026-06-11.
> Issue: [#1122](https://github.com/zynax-io/zynax/issues/1122) · Part 6 of EPIC #1107 ·
> Spec: [`ci-delivery-overhaul-prompt.md` §Part 6](ci-delivery-overhaul-prompt.md)

This document is the engineering constitution of Zynax. It applies to all contributors —
human and automated. It is not aspirational: every principle listed here is structurally
enforced by the CI pipeline, the `main-protection` repository ruleset, or pre-merge checks.
When a principle is not yet (or not fully) enforced, it is marked ⏳ with a pointer to the
work that will add the enforcement.

These principles are derived from DORA research, CNCF project patterns (Kubernetes, Flux,
ArgoCD, Helm, Prometheus), and the Google DevOps Handbook. They are calibrated for a
project that ships to production continuously, values correctness over speed, and is
developed by a combination of human engineers and AI agents.

**Every *Enforced by:* line below was verified against the live repository configuration
on the date above.** If you find a drift between this document and the pipeline, the
pipeline wins — fix this document (it is a living doc; `docs/` changes are PR-size-exempt).

The required status checks referenced throughout are the ones in the **`main-protection`
repository ruleset**: `dco`, `test-unit`, `security`, `lint-proto`, `lint-go`,
`lint-python`, `GitHub Actions workflow lint`, `Conventional Commit title`,
`PR size label`, `Secret scan (gitleaks)`.

---

## Principles

### P1 — Main is production, always.

The `main` branch is the deployment artifact. Every commit on `main` is deployable,
right now, without any stabilization period, merge window, or manual step. If you would
not be comfortable deploying it at 2am on a Sunday, it must not be on `main`.

*Enforced by:* the **`main-protection` repository ruleset** (migrated from classic branch
protection on 2026-06-11): all required status checks listed above, squash-merge as the
only allowed merge method, required linear history, required signed commits, no deletions,
no force pushes (ADR-023). One recorded bypass exists — the repository-admin role, used
exclusively by the pipeline's digest bot commit; this is the deliberate, documented
exception in [ADR-027 §Explicit exception to ADR-023](../adr/ADR-027-shift-left-pipeline.md#explicit-exception-to-adr-023).

---

### P2 — The PR is the unit of correctness.

A PR contains everything needed to verify the change: production code, tests, updated
documentation, updated configuration, updated digests. There are no "fix CI" PRs, no
"update digest" issues generated after the fact, no "I'll add tests in the next PR."
If the change is not verifiable from the PR alone, it is not ready.

*Enforced by:* the required pre-merge checks (`ci.yml`, `pr-checks.yml`, `pr-size.yml`).
Digest sync is part of the merge pipeline itself (`release.yml` retag job — ADR-027), not
a follow-up. The `canvas-freshness` check (`pr-checks.yml`) enforces SPDD Canvas alignment
on every `feat:` PR (ADR-019). ⏳ `canvas-freshness` runs on every PR but is not yet in
the ruleset's required set — it gates by convention and review until promoted.

---

### P3 — Build once. Promote by tag.

A container image is built exactly once, during the PR lifecycle. It is scanned for CVEs
before merge. On merge, it is retagged — not rebuilt. On version release, it is retagged
again. The image in production is the exact binary that passed the security gate.

Rebuilding after merge introduces nondeterminism: base layer updates, network fetches,
toolchain differences. "It built the same way" is not a supply-chain guarantee.

*Enforced by:* the `build-images` job in `ci.yml` builds every image pre-merge into the
GHCR staging lane (`staging/<svc>:pr-<head-sha>`) — issue #1118, PR #1132. `release.yml`
is retag-only: it contains **no** `docker/build-push-action` step; promotion uses
`docker buildx imagetools create` (PR #1135, made restartable after partial failure in
PR #1139). Decision record: [ADR-027](../adr/ADR-027-shift-left-pipeline.md).

---

### P4 — Shift security left. Everything that can fail pre-merge, must.

CVE scanning, Dockerfile linting, secret scanning, dependency review, supply-chain
attestation — all happen before merge. Post-merge security is a weekly drift audit for
new CVEs disclosed after merge, not a gate for things we could have caught earlier.

The weekly audit finding a HIGH CVE is a signal to open a properly-scoped PR. It never
triggers automated issue factories.

*Enforced by:* Hadolint + Trivy image scan (CRITICAL,HIGH = fail) + SBOM in the
`build-images` job (`ci.yml`, #1118); `Secret scan (gitleaks)` in `pr-checks.yml`
(**required**); `govulncheck`/`bandit`/`pip-audit` + Trivy filesystem and Dockerfile
misconfiguration scans in the `security` job of `ci.yml` (**required**);
`dependency-review` (HIGH CVEs) in `pr-checks.yml`; `weekly-audit.yml` (schedule-only,
#1113). ⏳ The `Build + scan: *` checks are **not yet required** in the `main-protection`
ruleset: the matrix is path-conditional (it only runs when docker-relevant paths change),
so promoting it needs a fan-in gate job first — same pattern as `test-unit`. Follow-up to
EPIC #1107; until then a red `Build + scan` check blocks by review convention, not
structurally.

---

### P5 — Small batches ship faster and safer.

Optimal PR size is ≤200 net lines of production code. Hard limit is 900 lines. Not
because large changes are wrong in principle, but because review quality degrades with
size, bisection complexity grows quadratically, and rollback risk compounds. DORA data
consistently shows: batch size is the primary driver of deployment frequency and change
failure rate. High performers ship 46× more frequently with 5× lower failure rates.

*Enforced by:* `PR size label` check in `pr-size.yml` (**required** — fails any PR over
the 900-line hard limit, after the documented exclusions). Policy: `CLAUDE.md §PR size`
and [CONTRIBUTING.md §6](../../CONTRIBUTING.md).

---

### P6 — One issue. One PR. One logical change.

Each GitHub issue maps to exactly one PR. Each PR contains exactly one logical change —
a change that can be described with a single conventional-commit subject line. Each commit
within the PR is independently revertible.

This makes `git bisect` O(log n). It makes rollback a one-command operation. It makes
review focused. It makes the changelog meaningful.

*Enforced by:* `Conventional Commit title` check in `pr-checks.yml` (**required**).
`dco` check in `ci.yml` (**required** — DCO sign-off on every commit). Squash-only merge
and required signed commits in the `main-protection` ruleset.

---

### P7 — No post-merge corrective actions.

Post-merge has exactly two responsibilities:

1. Retag the verified staging image to its production names (`main-<sha>`, `latest`).
2. Commit the updated digest to `images/images.yaml` (a skip-ci-marked bot commit).

That is all. No re-running tests. No re-running security scans. No issue factories.
No "completeness meshes." If a post-merge check fails, it means a pre-merge check was
missing — fix the pre-merge check, don't add more post-merge band-aids.

*Enforced by:* `release.yml` (retag-only — PR #1135/#1139, ADR-027). `weekly-audit.yml`
is schedule-only and creates no issues (#1113 — it replaced the per-merge "Post-Merge
Completeness" mesh, which auto-filed `[AUTO]` skeleton issues).

---

### P8 — No patches. Fix the root cause.

A "patch" is any change that works around a broken system rather than fixing it:
a `continue-on-error: true` on a flaky test, an `|| true` in a script to silence an
error, an `xfail` with no time-bounded tracking issue, a `.trivyignore` entry with no
expiry date, a skip-ci marker on a non-automated commit.

When a patch is genuinely necessary (upstream unfixable CVE, known-flaky external
dependency), it must have: (1) a dated expiry annotation, (2) a tracking issue, and
(3) a comment explaining the root cause and the upstream fix timeline.

*Enforced by:* `GitHub Actions workflow lint` (actionlint, **required**) catches undefined
variables and shell anti-patterns in workflows. `.trivyignore` entries require
`accepted-until:` (max 6 months), a reason, and maintainer approval — the convention and
quarterly-review process are documented in the file header (#1116). ⏳ The
`accepted-until` expiry check is review-based; an automated expiry linter does not exist
yet.

---

### P9 — Operations are idempotent.

Every CI job, every script, every Make target can be run twice and produce the same
result. The retag job promotes by digest — re-running it recreates the same tags from the
same staging digest. The bot commit is skipped when `images/images.yaml` has no changes
(`git diff --quiet` guard in `release.yml`).

Idempotency means re-running a failed job is always safe. It is the difference between
a pipeline that can be recovered in 5 minutes and one that requires manual cleanup.

*Enforced by:* `docker buildx imagetools create` is idempotent (overwrites the tag).
The retag promotion was made explicitly restartable after partial failure (PR #1139), and
staging cleanup is ordered after promotion so a re-run always finds its source (PR #1138).
Make targets are designed to be re-entrant.

---

### P10 — Operations are atomic.

A change either completes fully or leaves the system in its previous state. There are
no partial states: no "digest partially updated", no "image pushed but not signed",
no "release created but SBOM missing."

If an operation must be atomic across multiple steps, those steps run in a single job
with `set -euo pipefail`. If a step fails, the job fails and no subsequent step runs.

*Enforced by:* every multi-step shell block in `release.yml` uses `set -euo pipefail`;
the workflow contains zero `continue-on-error`. Promote → sign → digest commit → staging
cleanup run in a single `retag-on-merge` job in dependency order (ADR-027 — digest drift
is structurally impossible because the digest commit happens in the same transaction as
the promotion).

---

### P11 — Actions are pinned to SHAs. Tags are not immutable.

A GitHub Actions tag (`@v4`) is mutable. An attacker who compromises the upstream action
repository can push a new commit to that tag and execute arbitrary code in your pipeline.
Pinning to a commit SHA guarantees you run the exact code you reviewed.

Every third-party `uses:` must be pinned to a full commit SHA with the version as a
comment: `uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2`

*Enforced by:* repo-wide SHA pinning applied in #1116 (PR #1130) — as of 2026-06-11 there
are zero tag-pinned third-party actions in `.github/workflows/`. `GitHub Actions workflow
lint` (actionlint, **required**) lints every workflow change. ⏳ actionlint does not
itself reject an unpinned action — pin compliance on new `uses:` lines is held by code
review until an automated pin checker is added.

---

### P12 — Least privilege everywhere.

Every workflow declares only the permissions it needs. The default at the workflow
level is `permissions: contents: read`. Write permissions are escalated per-job only
when the job genuinely needs them, with a comment explaining why (e.g. `packages: write
# pushes staging-lane images to GHCR` on `build-images`).

No global `permissions: write-all`. No ambient secrets accessed by jobs that don't need
them. OIDC tokens (`id-token: write`) only in signing/publishing jobs (cosign keyless,
PyPI Trusted Publisher).

*Enforced by:* every workflow file carries an explicit top-level `permissions:` block
(audited and hardened in #1116 / PR #1130); per-job escalations carry justification
comments. `GitHub Actions workflow lint` (**required**) gates workflow changes.
Zero-trust posture: [ADR-020](../adr/ADR-020-zero-trust-auth.md).

---

### P13 — Signals are binary. Advisory output is not a gate.

A CI check either passes or fails. There is no third state. `continue-on-error: true`
on a gate is a contradiction — it is not a gate. An "advisory" check that never blocks
anything is noise that erodes trust in the entire pipeline.

The pipeline has two classes of jobs:

- **Gates:** required status checks in the `main-protection` ruleset. Failure blocks
  merge. Zero tolerance for flakiness — fix the flakiness.
- **Audits:** scheduled weekly runs. Failure is visible as a red workflow run.
  Not required checks. Never on PR triggers.

There is no third class. "Advisory on PR" was an experiment. It is retired.

*Enforced by:* `required_status_checks` in the `main-protection` ruleset.
`weekly-audit.yml` is schedule-only (`schedule` + `workflow_dispatch`, #1113).
`dev-advisory.yml` was deleted and its LLM configs archived to `docs/archive/` (#1129).

---

### P14 — Automated releases from tags. No release checklists.

A release is created by pushing a version tag: `git tag -s v0.5.0`. The tag triggers
`release.yml` (version retag + cosign signing + SLSA provenance attestation + SPDX SBOM +
CLI binaries) and `sdk-publish.yml` (zynax-sdk to PyPI via OIDC Trusted Publisher — no
API keys). The GitHub Release is created with auto-generated notes. No human steps after
pushing the tag.

Release notes are generated from PR titles since the last tag. The quality of release
notes is a function of title quality — which is enforced by P6.

*Enforced by:* `release.yml` `on: push: tags` (retag-version → release jobs;
`generate_release_notes: true`). `sdk-publish.yml` `on: push: tags`.
`Conventional Commit title` check (**required** — ensures the squash-merge subjects that
become release notes are parseable).

---

### P15 — The pipeline is documentation. Keep it simple.

The `.github/workflows/` directory is the authoritative specification of how the project
is built, tested, and released. A new contributor should be able to understand the full
pipeline in 30 minutes. Each workflow file has one responsibility, a clear name, and a
comment header explaining what it does and why it exists.

Workflows are DRY: reusable workflows (`_test-go.yml`, `_test-python.yml`) eliminate
copy-paste. Image references have a single source of truth (`images/images.yaml`,
ADR-024) with a drift gate. The build matrix appears in `build-images` (`ci.yml`),
`pr-image-cleanup.yml`, and the retag job — kept in sync by explicit cross-reference
comments at each site. ⏳ Extracting the matrix into one shared config that all consumers
read is a known simplification, not yet done.

Complexity is a cost. Every job added is a job that can fail, must be maintained, and
must be understood by contributors. Prefer removing jobs to adding them.

*Not formally enforced — enforced by code review and by the principle that a workflow
file should fit on one screen.*

---

## DORA targets

| Metric | Target | Structural enabler |
|--------|--------|--------------------|
| Deployment frequency | Multiple per day | Tag-triggered release, retag-not-rebuild promotion, no merge windows |
| Lead time for changes | <1 day p50 | PR size limit, path-aware CI lanes, merge on green |
| Change failure rate | <5% | Pre-merge image scan, BDD contract tests, e2e-smoke gate (⏳ advisory today — promotion to required tracked in #1092) |
| MTTR | <1 hour | Linear squash-merge history, `git bisect`, revert-as-PR |

## CNCF reference patterns

| Project | Pattern we adopt |
|---------|-----------------|
| Kubernetes | Path-aware change detection matrix; reusable workflows |
| Flux | Image automation with skip-ci-marked bot digest commits; GitOps reconciliation |
| Helm | Conventional commits → auto-generated release notes; chart lint in CI (`helm-lint.yml`) |
| Prometheus | Multi-arch native builds (no QEMU — #837); minimal alpine/distroless base images |
| ArgoCD | Cosign keyless OIDC signing; SBOM on every release; SLSA provenance (ADR-025) |
| containerd | buf-based proto contract testing; strict layer boundary enforcement |

---

## Related documents

- [ADR-027 — shift-left pipeline model](../adr/ADR-027-shift-left-pipeline.md) (build once / retag / digest sync)
- [ADR-023 — restrict direct pushes to main](../adr/ADR-023-restrict-direct-pushes-to-main.md)
- [ADR-024 — image reference management](../adr/ADR-024-image-reference-management.md) · [ADR-025 — SLSA provenance](../adr/ADR-025-slsa-provenance-attestation.md)
- [CONTRIBUTING.md](../../CONTRIBUTING.md) — the practical how-to that these principles govern
- [Spec: CI/delivery overhaul prompt](ci-delivery-overhaul-prompt.md) — the original draft this document was distilled from
