<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-024 — Container Image Reference Management

| Field | Value |
|-------|-------|
| **Status** | Accepted |
| **Date** | 2026-06-08 |
| **Deciders** | Oscar Gómez Manresa |
| **Scope** | `images/images.yaml`, all CI workflow files, Dockerfiles, `config/ci-runner-digest.txt` |

---

## Context

Container image references were split-brain across the repository:

- The `zynax/ci-runner` digest appeared **18 times** across 6 files:
  `config/ci-runner-digest.txt` (aspirational source) and 5 GitHub Actions
  workflow files (`.github/workflows/ci.yml`, `pr-checks.yml`,
  `_test-go.yml`, `_test-python.yml`, `ai-context-budget.yml`).
- Dockerfile base image digests (`golang-alpine`, `distroless-static`,
  `python-slim`) repeated across 7+ files with no drift-check gate.
- `config/ci-runner-digest.txt` and `scripts/bump-ci-runner.sh` were ~80% of
  a solution already — the missing 20% was a CI gate that made the source file
  provably canonical rather than aspirational. Proof: issue #853 was an open,
  unfulfilled ci-runner digest bump at the time EPIC #855 was opened.

### Why certain consumers must be generated

Dockerfile `FROM` lines, GitHub Actions `container:` fields, and
`config/ci-runner-digest.txt` are static files evaluated at build time or by
the GitHub Actions runner before any tool can run. They **cannot import a
value from `images.yaml` at runtime** — they must be pre-populated with the
correct value. This is the fundamental constraint that Option B accepts: the
remaining duplication is generated, not hand-maintained.

### Relationship to prior decisions

- **ADR-019** (SPDD / REASONS canvas) — EPIC #855 canvas
  (`docs/spdd/855-images-sot/canvas.md`) was authored and aligned before any
  implementation code was written, as required for `feat:` PRs.
- **EPIC #855** (M6.Images) implemented the solution in seven incremental
  O-steps (#856–#862); this ADR records the final architectural rationale.

---

## Decision

**Option B — `images/images.yaml` as source of truth, generator + CI drift-check.**

1. `images/images.yaml` is the **only place a human edits** pinned image
   digests. All other occurrences are generated.

2. `zynax-ci images sync` (`make sync-images`) stamps all consumer files from
   the values in `images.yaml`. Consumer files contain banner-marked generated
   regions (`# BEGIN zynax-ci:images:<name>` / `# END zynax-ci:images:<name>`)
   that `sync` owns; content outside the banners is owned by humans.

3. `zynax-ci images check` (`make check-images`) is wired as a required CI
   gate in `pr-checks.yml` and `ci.yml`. It exits non-zero when any consumer
   diverges from `images.yaml`. **This gate is the keystone invariant**: without
   it, `images.yaml` is aspirational rather than provably canonical.

4. A digest bump workflow:
   - Edit `images/images.yaml` only.
   - Run `make sync-images` to stamp all consumers.
   - Open one PR — no knowledge of which downstream files exist required.

5. Renovate continues to manage Dockerfile base image bumps for monthly group
   updates. **Renovate and `sync` are complementary, not competing**: Renovate
   opens the PR that updates `images.yaml`; `sync` stamps all consumers from
   that source. The CI gate (`check`) ensures consumers stay in sync between
   Renovate updates.

---

## Rationale

| Option | Assessment |
|--------|------------|
| **Option B — `images.yaml` + generator + CI gate** (chosen) | ✅ Chosen — single edit point, deterministic drift-check, scalable to any number of consumers, no hand-editing of generated regions required |
| Option A — keep manual bumps + shell script | ✗ Rejected — `scripts/bump-ci-runner.sh` only covered the ci-runner digest in workflow files; Dockerfile base images required separate manual effort; no CI gate meant drift was silent |
| Option C — Renovate for all images | ✗ Rejected for workflow file references — Renovate has no native support for updating arbitrary digest occurrences inside GitHub Actions `container:` fields in non-image-spec positions; it already handles Dockerfile base images well and continues to do so |

---

## Consequences

### Positive

- A release engineer edits **one file** (`images/images.yaml`) and runs one
  command (`make sync-images`) to propagate a digest bump to all consumers.
- CI fails immediately on any PR that hand-edits a banner region or forgets to
  run `sync` after updating `images.yaml`.
- The number of consumers can grow (new workflows, new Dockerfiles) without
  changing the bump procedure — only add the file to the `consumers:` list in
  `images.yaml`.
- `zynax-ci images check --dry-run` provides a human-readable diff for PR
  review.

### Negative / trade-offs

- Banner-marked regions in workflow files and Dockerfiles **must not be
  hand-edited**. An editor who ignores the banner comments will have their
  changes overwritten by the next `make sync-images` run and caught by CI.
- All digest bumps go through `make sync-images`; a bump that skips `sync`
  will fail the CI gate. This is intentional — the friction is the feature.
- `scripts/bump-ci-runner.sh` is deprecated (kept for backward compatibility
  in M6; scheduled for removal in M7 or when confirmed disused).

### Out of scope

The following image references are **intentionally excluded** from
`images.yaml` management:

| Reference type | Reason |
|----------------|--------|
| Docker Compose `:main` dev tags | Intentionally fluid — no single version to centralise |
| Helm `values.yaml` image fields | The native `image: {repository, tag}` pattern per M6.Helm canvas is already correct |
| README install table | Shows `:latest`/`:vX.Y.Z` — always correct as documentation |

### Follow-up required

| Action | Tracking |
|--------|---------|
| Remove `scripts/bump-ci-runner.sh` | M7 or when confirmed disused |
| Upgrade Renovate config to open PRs that edit `images/images.yaml` directly | Future Renovate config PR |
