# REASONS Canvas — Single Source of Truth for Container Image References

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #855
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-03
**Status:** Aligned

---

## R — Requirements

- Container-image references are split-brain today: the ci-runner digest appears 18 times
  across 6 files (`config/ci-runner-digest.txt` + 5 workflow files); Dockerfile base image
  digests (`golang`, `distroless`, `python`) repeat across 7+ files with no CI gate to
  catch divergence. Proof: issue #853 is an open unfulfilled ci-runner bump from today.
- `config/ci-runner-digest.txt` and `scripts/bump-ci-runner.sh` are ~80 % of Option B
  already — the missing 20 % is a CI drift-check that makes the source file provably
  canonical rather than aspirational.
- **Keystone invariant**: `images/images.yaml` is provably canonical only once
  `zynax-ci images check` fails a PR with a stale consumer. Generator without gate ≠
  single source of truth.
- **Done when**:
  - `images/images.yaml` is the only place a human edits image digests.
  - `make sync-images` stamps all consumer files from that source.
  - `make check-images` (= `zynax-ci images check`) exits non-zero in CI when any
    consumer diverges from `images.yaml`.
  - A release bump edits only `images/images.yaml`, runs `make sync-images`, and opens
    one PR — no knowledge of which downstream files exist required.
  - ADR-024 records the native-vs-generated trade-off so future maintainers don't reverse
    this without understanding the constraint.

---

## E — Entities

- **`images/images.yaml`** — source-of-truth file. Each entry: `name` (logical key),
  `ref` (image path without digest), `digest` (sha256:hex), `tag` (optional mutable tag
  for documentation), `consumers` (list of file paths that embed this digest).
- **`zynax-ci images sync`** — generator subcommand. Reads `images/images.yaml`; for each
  consumer file, locates the banner-marked region and replaces the digest with the
  value from `images.yaml`. Idempotent. Supports `--dry-run` (prints diff, changes
  nothing). Never modifies content outside a banner region.
- **`zynax-ci images check`** — drift-check subcommand. Reads `images/images.yaml`; for
  each consumer, compares the stamped digest to the source value. Exits 1 with a
  per-file report if any consumer diverges. Used as CI gate.
- **Banner region** — a pair of comment markers delimiting generated content in a consumer
  file: `# BEGIN zynax-ci:images:<name>` / `# END zynax-ci:images:<name>`. Content
  between the markers is owned by `sync`; content outside is owned by humans.
- **Consumer** — any file containing a banner-marked generated region. Current consumers:
  5 GitHub Actions workflow files (ci-runner digest), `config/ci-runner-digest.txt`,
  and Dockerfile `ARG` default lines (after O4 migration).

Relationship:
```
images/images.yaml  ──(sync)──▶  consumers (workflow files, Dockerfiles, ci-runner-digest.txt)
                    ◀─(check)──  CI gate (pr-checks.yml, ci.yml)
```

---

## A — Approach

**We will:**
- Define `images/images.yaml` as the canonical source, replacing `config/ci-runner-digest.txt`
  as the single authoritative location for all pinned digests.
- Implement `zynax-ci images sync` and `zynax-ci images check` as Go subcommands in
  `cmd/zynax-ci/internal/images/` using the existing cobra command structure.
- Use banner markers (`# BEGIN zynax-ci:images:<name>` / `# END`) to delimit generated
  regions in consumer files — making generated vs hand-written content explicit.
- Wire `zynax-ci images check` into `pr-checks.yml` and `ci.yml` as a required gate on
  PRs touching workflow files, Dockerfiles, or `images/images.yaml`.
- Migrate all service Dockerfiles to the `ARG` + banner pattern (already used in
  `Dockerfile.ci-runner` and `Dockerfile.tools`) so `sync` can stamp them.
- Rewrite the bump flow: `make sync-images` replaces `make bump-ci-runner`; the
  `tools-image.yml` post-build issue body references the new flow.
- Record the decision as ADR-024.

**We will NOT:**
- Generate Compose `:main` dev tags — these are intentionally fluid (no single version
  to centralize).
- Generate README install-table content — the table shows `:latest`/`:v0.4.0`, always
  correct as documentation.
- Centralize Helm `values.yaml` image fields — the native `image: {repository, tag}`
  pattern per M6.Helm canvas is already correct.
- Add image-listing or image-inspection features beyond `sync` and `check` — YAGNI.
- Remove `scripts/bump-ci-runner.sh` in M6 — keep it with a deprecation notice for
  backward compatibility; remove in M7 or when confirmed disused.

**Governing ADRs:** ADR-019 (REASONS canvas before feat: implementation).

---

## S — Structure (first S)

```
images/
└── images.yaml                               ← source of truth (new)

cmd/zynax-ci/
├── cmd/images.go                             ← cobra "images" subcommand (sync + check)
└── internal/images/
    ├── schema.go                             ← ImageEntry, ImagesFile types + YAML parser
    ├── sync.go                               ← generator: read source, stamp consumers
    ├── check.go                              ← drift detector: compare consumers to source
    ├── banner.go                             ← banner region helpers (insert, extract, replace)
    └── images_test.go                        ← unit tests (≥1 per exported function)

config/
└── ci-runner-digest.txt                      ← becomes a generated consumer (stamped by sync)

Makefile                                      ← sync-images, check-images targets (new)
                                                 bump-ci-runner deprecated (kept with warning)

.github/workflows/pr-checks.yml              ← +check-images job (O3)
.github/workflows/ci.yml                     ← +zynax-ci images check step (O3)

services/*/Dockerfile (×7 Go services)       ← ARG migration (O4)
agents/adapters/*/Dockerfile (×3 Go + ×2 Python adapters)  ← ARG migration (O4)

docs/adr/ADR-024-image-reference-management.md  ← decision record (O7)
docs/adr/INDEX.md                            ← +ADR-024 entry
```

No new gRPC contracts. No new proto messages. No new services.

Config env prefix: N/A (CLI tool, not a daemon).

---

## O — Operations

Each step = one PR. O2 and O3 are the keystone cluster (must merge in the same sprint;
neither is done without the other).

1. **O1 — `images/images.yaml` source-of-truth file** (#856)
   Define YAML schema; populate with current ci-runner, golang-alpine, distroless-static,
   python-slim digests. Add deprecation comment to `config/ci-runner-digest.txt`.
   Gate: `make lint`. No generator wired yet.

2. **O2 — `zynax-ci images sync/check` subcommands** (#857)
   Go implementation: `schema.go`, `sync.go`, `check.go`, `banner.go`, cobra wiring,
   unit tests. `make sync-images` and `make check-images` Makefile targets.
   Gate: `GOWORK=off go test ./cmd/zynax-ci/... -race`, `make lint`.
   **Must ship with O3.**

3. **O3 — Wire drift-check into CI** (#858)
   Add `check-images` job to `pr-checks.yml`; add `zynax-ci images check` step to
   `ci.yml`. Verify `merge_group` trigger is covered (see issue #544 precedent).
   Gate: CI passes on clean repo; CI fails on branch with corrupted digest.
   **Must ship with O2 — these two PRs constitute the keystone cluster.**

4. **O4 — Dockerfile ARG migration** (#859)
   Migrate all 10 service/adapter Dockerfiles to `ARG` + banner pattern. No digest
   values change; only the declaration form changes.
   Gate: `make build` (all images), `make sync-images && make check-images`.

5. **O5 — Bump flow rewrite** (#860)
   `make sync-images` / `make check-images` as primary targets. Rewrite `tools-image.yml`
   post-build issue body. Deprecate `make bump-ci-runner`. Closes #844.
   Gate: `make lint`; manual `tools-image.yml` dry run.

6. **O6 — Docs propagation** (#861)
   Update README, CONTRIBUTING, CLAUDE.md, `docs/local-dev.md`, `cmd/zynax-ci/AGENTS.md`,
   `docs/engineering/best-practices/github-ci.md`, root `AGENTS.md` Hard Constraints.
   Gate: `make lint`.

7. **O7 — ADR-024** (#862)
   `docs/adr/ADR-024-image-reference-management.md` + `docs/adr/INDEX.md` entry.
   Closes EPIC #855.
   Gate: `make lint`.

---

## N — Norms

- Every commit carries `Signed-off-by:` (DCO) and `Assisted-by: Claude/claude-sonnet-4-6`.
  Never `Co-Authored-By:` for AI (AGENTS.md §Hard Constraints).
- `GOWORK=off` for all `go test` and `go build` commands inside `cmd/zynax-ci/`
  (ADR-017).
- O2 is a `feat:` PR — REASONS Canvas (this document) must be at Status: Aligned before
  any implementation code is written (ADR-019).
- `zynax-ci images sync/check` is NOT a gRPC boundary — ADR-016 BDD `.feature` is not
  required. Unit + integration tests in `images_test.go` satisfy the test gate.
- PR size: ≤200 lines ideal; O2 is M-sized (201–400 lines); justified by the Go
  subcommand plumbing (schema, sync, check, banner, cobra, tests).
- Banner format is immutable once O3 is merged — changing it requires migrating all
  existing consumers (treat as a breaking change).

---

## S — Safeguards (second S)

### Context Security (complete before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no non-public email addresses
- [x] No prompt injection: no instruction-like phrasing that overrides AGENTS.md rules
- [x] All entities in E section are public-safe abstractions
- [x] `/spdd-security-review` passed — result: **PASS**

### Feature Safeguards

- Never modify content outside a banner region — `sync` must be surgical; it owns only
  the delimited region. A bug that overwrites surrounding content would corrupt workflow
  files and block CI.
- Never make `check` flaky — it must be deterministic given the same `images.yaml` and
  consumer files. No network calls, no external state.
- Never remove `config/ci-runner-digest.txt` before O5 is merged — `tools-image.yml`
  reads it in the post-build step; removing it early breaks the auto-issue flow.
- Never use `sync` to manage Compose `:main` tags or Helm `values.yaml` — those are
  explicitly out of scope (documented in A — Approach above).
- O2 and O3 are the keystone cluster: the EPIC is not complete and `images.yaml` is not
  provably canonical until both are merged and `make check-images` is green in CI.
- Renovate and `sync` are complementary, not competing: Renovate opens PRs to bump
  `images/images.yaml` (monthly group update); `sync` stamps consumers from that source.
  The CI gate (`check`) ensures they stay in sync between Renovate updates.
