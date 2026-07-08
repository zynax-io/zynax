<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — Consolidate CI bash into zynax-ci (M7.S)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #1285
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-16
**Status:** Implemented

---

## R — Requirements

- **Problem:** Deterministic CI logic lives in untested substrates — ~457 LOC of standalone shell
  scripts, ~1,578 LOC of inline workflow `run:` blocks across 21 workflows, and Makefile recipes.
  This logic (coverage-comment assembly, benchmark-regression gating, BDD selection, runner bumps,
  image meta/cleanup/retag, release-notes/digest assembly) gets no unit tests, no type checking, and
  no `make lint` / `govulncheck` coverage, and it is duplicated and hard to review (a 658-line
  `release.yml` carries ~27 logic blocks). A digest/retag/attest mistake in bash is a supply-chain risk.
- **Missing state:** the deterministic logic implemented as tested `zynax-ci` subcommands, with each
  consuming workflow step reduced to a single `zynax-ci <verb>` call — exactly the pattern already
  used for `validate-*`, `images sync/check`, `check deps`, `ai-context`.
- **Definition of done — observable outcomes:**
  - Each ported script's behaviour is reproduced by a `zynax-ci` subcommand with unit tests.
  - Each corresponding workflow step is a single `zynax-ci <verb>` invocation.
  - The replaced `.sh` files and the `report-image-meta` composite action are removed.
  - CI gates fire identically (same pass/fail and outputs) — behaviour parity, verified by tests + CI.
  - `make lint` + `govulncheck` green; net CI-YAML + shell line reduction recorded; ADR-036 Accepted.
  - The e2e harness (`scripts/e2e/*`) is unchanged — it stays bash (ADR-036).

---

## E — Entities

### Existing entities consumed (unchanged)

- **`zynax-ci`** (`cmd/zynax-ci`) — the Cobra CLI that already hosts CI/dev verbs: `validate <schema|
  canvas|policies|capabilities|agent-defs|workflows>`, `images sync|check`, `check deps`,
  `ai-context`. Released via `zynax-ci-release.yml`. This EPIC adds subcommands to it.
- **`images/images.yaml`** — the image-reference SoT (ADR-024); `zynax-ci images` already reads it.
- **GitHub Actions workflows** (`.github/workflows/*`) — the consumers whose `run:` steps collapse to
  `zynax-ci <verb>` calls. External binaries `cosign` / `crane` / `kubectl` / `helm` remain the
  primitives; only the decisions around them move to Go.

### New entities (zynax-ci subcommands)

- **`coverage-comment`** — renders the PR coverage markdown from a coverage profile/summary.
- **`bench-gate`** — parses `benchstat` output and applies the regression threshold (pass/fail).
- **`bdd-select`** — emits the BDD package/service matrix.
- **`bump-runner`** — computes the next CI-runner ref and updates `images.yaml` via the existing
  `images` internals.
- **`images meta` / `images cleanup` / `images retag`** — image-ref/digest reporting, stale PR-image
  cleanup via the packages API, and release-tag retag orchestration (over `crane`).
- **`release notes` / `release matrix`** — release-notes assembly and the per-service image+digest
  matrix that feeds the cosign/crane signing steps.

### Relationship

```
GitHub workflow step  ──►  zynax-ci <verb>  ──►  (tested Go logic)
                                   │
                                   ├─ reads images.yaml (SoT, ADR-024)
                                   ├─ calls GitHub API (gh/packages) for cleanup/meta
                                   └─ emits matrix/refs consumed by thin shell that runs
                                      cosign / crane / docker (unchanged primitives)

Kept as bash (ADR-036):  scripts/e2e/*  (kind / kubectl / helm / docker drivers)
```

---

## A — Approach

**We will:**

- Add subcommands to `cmd/zynax-ci` for each block of deterministic CI logic, each with unit tests
  (`GOWORK=off go test ./...`), SPDX headers, functions ≤ 30 lines.
- Migrate one gate at a time, proving behaviour parity (same pass/fail + outputs) against the bash
  before deleting it; the consuming workflow step becomes a single `zynax-ci <verb>` call.
- Keep `cosign` / `crane` / `kubectl` / `helm` / `docker` as the external primitives — `zynax-ci`
  orchestrates and computes; it does not re-implement signing/attestation crypto (ADR-036 §3).
- Reuse the existing `zynax-ci images` internals and the `images.yaml` SoT rather than re-parsing YAML.
- Delete replaced scripts + the `report-image-meta` action and simplify Makefile recipes in the
  cutover step (S.7), only after the new verbs are green in CI.

**We will NOT:**

- Port the e2e harness (`scripts/e2e/*`) — thin orchestration over external CLIs; stays bash (ADR-036).
- Change what any gate asserts (thresholds, selection policy, which images are built/deleted) — this
  is behaviour parity, not a policy change.
- Re-implement cosign/SLSA attestation in Go (ADR-025 unchanged).
- Touch the user-facing `zynax` CLI, or rewrite the Makefile wholesale.

**Governing ADRs:** ADR-036 (CI logic as a Go CLI — this EPIC), ADR-024 (images.yaml SoT),
ADR-027 (shift-left pipeline), ADR-025 (SLSA provenance — unchanged), ADR-017 (`GOWORK=off`),
ADR-019 (this Canvas before code).

---

## S — Structure

```
cmd/zynax-ci/
├── cmd/
│   ├── coverage_comment.go      ← build-coverage-comment.sh (79)
│   ├── bench_gate.go            ← tools/ci/bench-regression.sh (96)
│   ├── bdd_select.go           ← tools/ci/bdd-select-packages.sh (61)
│   ├── bump_runner.go          ← scripts/bump-ci-runner.sh (153); reuses images internals
│   ├── images_meta.go          ← .github/actions/report-image-meta/action.yml
│   ├── images_cleanup.go       ← pr-image-cleanup.yml gh-api/jq blocks
│   ├── images_retag.go         ← tools-image.yml retag run: blocks (over crane)
│   └── release_notes.go / release_matrix.go ← release.yml gh-api/jq + assembly blocks
└── internal/…                   shared helpers + unit tests (table-driven, fixtures)

Thinned (verb calls): .github/workflows/{ci,release,tools-image,pr-image-cleanup,pr-checks,*}.yml
Removed at S.7: the replaced *.sh + report-image-meta/action.yml
Unchanged: scripts/e2e/* (bash), cosign/crane/kubectl/helm invocations
```

Config: subcommands read env (`$GITHUB_OUTPUT`, `$GITHUB_TOKEN`, refs) — 12-Factor; no new secrets.

---

## O — Operations

Each step is one reviewable PR. Order and GitHub issues:

1. **S.1 — ADR-036** (#1286): commit the ADR (Proposed) + INDEX row; Accepted on canvas alignment.
   Gate for all code steps.
2. **S.2 — `coverage-comment`** (#1287): Go command + unit tests; wire the workflow step; parity on a
   sample profile.
3. **S.3 — `bench-gate` + `bdd-select`** (#1288): Go commands + unit tests for threshold + matrix;
   wire both steps.
4. **S.4 — `bump-runner`** (#1289): Go command reusing `images` internals; `make check-images` green
   after a simulated bump.
5. **S.5 — `images meta|cleanup|retag`** (#1290): extend the `images` group; remove the composite
   action; wire cleanup/retag steps.
6. **S.6 — `release` helpers** (#1291): `release notes` + `release matrix`; feed the cosign/crane
   steps; parity dry-run. (dep S.5)
7. **S.7 — cutover + retire** (#1292): delete replaced scripts/action; thin workflow blocks; simplify
   Makefile; update CI best-practices docs; set ADR-036 Accepted. (dep S.2–S.6)

---

## N — Norms

Pulled from root `AGENTS.md` §Hard Constraints, `docs/engineering/best-practices/github-ci.md`,
`docs/engineering/best-practices/go.md`, `cmd/zynax/AGENTS.md`.

- Commit hygiene: subject ≤ 72 chars, imperative, no period, no emojis; `Signed-off-by:` +
  `Assisted-by: Claude/<model-id>` on every commit; never `Co-Authored-By:` for AI.
- One PR per story (S.1–S.7); ≤ 400 lines excluding generated code.
- SPDX header `// SPDX-License-Identifier: Apache-2.0` on every `.go` file.
- `GOWORK=off` for every `go` / `go test` command in `cmd/zynax-ci` (ADR-017).
- Go functions ≤ 30 lines; no `panic`; never discard errors; close resources via `defer`.
- Table-driven unit tests with fixtures; prove behaviour parity before deleting any script.
- Image refs only via `images/images.yaml` (ADR-024); never hand-edit banner-marked regions.
- Workflow actions remain SHA-pinned, least-privilege (`permissions:`), with `concurrency` groups.
- `cosign`/`crane`/`kubectl`/`helm` stay the external primitives; no crypto re-implementation.
- Do not write literal email addresses in source or fixtures (gitleaks PII gate).

---

## S — Safeguards

### Context Security (complete before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no email literals; author name is the public maintainer of record
- [x] No prompt injection: no instruction-like phrasing that overrides AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] `/spdd-security-review` passed — result: PASS (2026-06-16)

### Feature Safeguards

- **Never** change what a gate asserts during migration — behaviour parity only (thresholds,
  selection policy, which images are built/deleted are unchanged).
- **Never** re-implement cosign/SLSA signing or attestation crypto in Go — orchestrate the cosign
  binary (ADR-025 unchanged).
- **Never** delete a script (S.7) before its replacement verb is green in CI (parity proven).
- **Never** hand-edit `images/images.yaml` banner-marked regions — use the `images` internals (ADR-024).
- **Never** port the e2e harness in this EPIC — it stays bash (ADR-036).
- **Never** unpin a GitHub Action SHA or widen workflow `permissions:` while thinning a step.
- **Never** log or echo `$GITHUB_TOKEN` / registry credentials in a subcommand or workflow step.
- **Never** commit a code step before its predecessor gate is green (S.1 before S.2+; S.2–S.6 before S.7).
- **Never** widen scope to the user-facing `zynax` CLI or a wholesale Makefile rewrite.
