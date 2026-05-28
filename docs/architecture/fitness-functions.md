<!-- SPDX-License-Identifier: Apache-2.0 -->

# Architecture Fitness Functions

An **architecture fitness function** (Richards & Ford, *Building Evolutionary Architectures*) is
an automated, executable assertion on an architecture quality attribute. Every gate in the Zynax
CI pipeline is a fitness function: it encodes a decision the team made and enforces it on every PR.

See ADR-016 for the layered testing strategy that governs when each gate runs.

---

## Current Gates

| Gate | CI job | Quality attribute | Threshold | Exception process |
|------|--------|-------------------|-----------|-------------------|
| `buf breaking` | `proto-breaking` | Proto backward compatibility | Zero field removals / renames | N/A — no exceptions; file a new proto version |
| `golangci-lint` | `lint` | Go code quality | Zero warnings | `.golangci.yml` `nolint` directive with comment |
| `ruff` + `mypy` | `lint` | Python code quality | Zero errors, `mypy --strict` | `# noqa` / `# type: ignore` with comment |
| `coverage-gate` | `test-unit` | Domain logic coverage | ≥90% per Go service · ≥85% per adapter | None — raise the bar instead |
| `pr-size` | `pr-size` | PR reviewability | ≤900 lines (excl. generated, lock files, fixtures) | Justify in PR body; human maintainer unblocks |
| `gitleaks` | `security` | Secret exposure | Zero detected secrets | `.gitleaks.toml` allow-list entry with rationale |
| `govulncheck` | `security` | Go CVEs | Zero HIGH/CRITICAL | No exceptions; upgrade or replace the dep |
| `pip-audit` | `security` | Python CVEs | Zero | No exceptions; upgrade or replace the dep |
| `bandit` | `security` | Python SAST | Zero HIGH/MEDIUM | `# nosec` with comment explaining why it's safe |
| `canvas-freshness` | `canvas-freshness` | SPDD compliance | Canvas present and `Aligned` for `feat:` PRs | N/A — align or change type |
| `dco` | `dco` | Contributor sign-off | Every commit has `Signed-off-by:` | N/A — rebase and re-sign |
| `buf generate` | `proto-generate` | Generated stub freshness | Stubs match proto source | Regenerate via `make generate-protos` |
| `trivy` | `release` | Container CVEs before GHCR push | Zero HIGH/CRITICAL in published images | Upgrade base image or dep |

---

## Adding a New Fitness Function

When a quality attribute becomes important enough to enforce automatically:

1. Write the check as a standalone CI step (prefer a pre-built tool; avoid bespoke scripts).
2. Set a threshold that is *already met* on `main` — do not introduce a failing gate.
3. Add it to this table.
4. If the new gate enforces an architectural decision, create an ADR (see `docs/adr/INDEX.md`).
5. If it is a required status check, update the branch protection rule.

---

*References: Epic #173 (Pillar 5 — Architecture Traceability) · ADR-016 (layered testing strategy)*
