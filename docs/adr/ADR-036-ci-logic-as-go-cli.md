<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-036 — CI logic belongs in a tested Go CLI (zynax-ci), not inline workflow bash

| Field | Value |
|-------|-------|
| **Status** | Accepted |
| **Date** | 2026-06-16 |
| **Deciders** | Oscar Gómez Manresa |
| **Scope** | `cmd/zynax-ci/`, `.github/workflows/*`, `scripts/`, `tools/ci/`, `Makefile` — M7 EPIC S (#1285) |
| **Related** | ADR-024 (images.yaml SoT), ADR-027 (shift-left pipeline), ADR-017 (GOWORK=off / contract-test isolation), ADR-006 (monorepo) |

---

## Context

Zynax CI logic is spread across three substrates that get no test coverage and no type
checking:

- **Standalone shell scripts** (~457 LOC of deterministic logic: coverage-comment assembly,
  benchmark-regression gating, BDD package selection, CI-runner version bumping).
- **Inline workflow `run:` blocks** (~1,578 LOC across 21 workflows: `gh api`/`jq` parsing,
  `crane`/`cosign` digest + attestation orchestration, image retag, image-meta reporting,
  release-notes assembly, validation gates).
- **Makefile recipes** (~234 shell lines).

Bash here is a maintenance and supply-chain liability: it is untested, it duplicates parsing
logic across workflows, it is hard to review (a 658-line `release.yml` carries ~27 logic blocks),
and a mistake in a digest/retag/attest step is a supply-chain risk. The repository already proved
the alternative: **`cmd/zynax-ci`** is a Cobra CLI that hosts exactly this class of logic
(`validate-*`, `images sync/check`, `check deps`, `ai-context`) with unit tests and `make lint` /
`govulncheck` coverage. New CI logic, however, keeps landing as inline bash by default.

The e2e harness (`scripts/e2e/*`, ~1,617 LOC) is a different animal: it is thin orchestration over
`kind` / `kubectl` / `helm` / `docker` and carries little decision logic of its own. Porting it to
Go would add `client-go` / Helm-SDK dependencies for rarely-run test scaffolding — cost without a
proportional maintenance win.

## Decision

1. **Deterministic CI logic is implemented as a tested `zynax-ci` subcommand**, not as inline
   workflow `run:` bash or a standalone script. "Deterministic logic" = parsing, gating,
   assembly, transformation, and orchestration decisions (what to retag, which packages to run,
   pass/fail thresholds, comment/notes rendering).

2. **Workflow `run:` steps stay thin** — ideally a single `zynax-ci <verb>` invocation plus
   environment wiring. Multi-line logic blocks are a smell to be extracted.

3. **External-tool invocation that carries no decision logic stays shell** — calling `cosign`,
   `crane`, `docker buildx`, `kubectl`, `helm` as binaries is fine; the *decisions around* those
   calls move to Go. `zynax-ci` does not re-implement signing/attestation crypto; it orchestrates
   the cosign/crane binaries.

4. **The e2e harness stays bash** (ADR-036 §Context). De-bashing stops at the test-driver boundary.

5. Behaviour parity is mandatory: a ported gate must fire identically (same pass/fail, same
   outputs) — verified by unit tests on the Go command and by the unchanged CI result.

## Consequences

**Positive**

- Every CI gate gains unit tests, types, and `make lint` / `govulncheck` coverage.
- Workflow YAML shrinks (~1,000+ inline `run:` lines collapse to `zynax-ci` calls); reviewers
  audit Go with tests instead of unreviewed bash.
- One versioned binary (released via `zynax-ci-release.yml`) replaces N copies of parsing logic.
- Supply-chain-sensitive steps (digest, retag, release assembly) become testable and reviewable.

**Negative / accepted**

- The e2e harness remains bash — Shell does not drop to zero; this is a deliberate cost/value line.
- Migrating supply-chain steps (digest/retag/attest orchestration) must be done carefully with
  parity tests; a regression there is high-impact. Mitigated by S-step sequencing and keeping
  cosign/crane as the actual signing/copy primitives.
- `zynax-ci` grows in surface; offset by deleting the scripts and inline blocks it replaces.

## Alternatives considered

- **Keep CI logic in bash (status quo).** Rejected: untested, duplicated, hard to review, a
  supply-chain liability — against the grain of the existing `zynax-ci` commands.
- **A new dedicated tool (`cmd/zynax-dev`).** Rejected for now: adds a second module, release
  pipeline, and versioning stream; `zynax-ci` is already the CI/dev-tooling home.
- **Port everything including the e2e harness to Go.** Rejected: adds heavy k8s/Helm client deps
  for rarely-run test glue with little decision logic — cost without proportional benefit.
