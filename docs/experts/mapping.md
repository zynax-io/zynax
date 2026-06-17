<!-- SPDX-License-Identifier: Apache-2.0 -->
# Authoring ↔ Runtime Expert Mapping

> Governed by [ADR-033](../adr/ADR-033-expert-agent-substrate.md) · EPIC X (#1170), step X.5 (#1205).

Zynax experts live on **two substrates**:

- **Authoring experts** — `.claude/commands/experts/<slug>.md`, used in the SPDD
  delivery/authoring loop (the Claude Code expert that ships a story).
- **Runtime experts** — `kind: AgentDef` agents (under `agents/examples/`) that
  register in agent-registry and are dispatchable inside a workflow.

Each authoring expert declares a `runtime_mapping` pointing at its runtime
counterpart, or the literal `authoring-only` when none exists yet.

## Source of truth

| Surface | Role |
|---------|------|
| `automation/experts/runtime_mapping.yaml` | **Machine-readable source of truth** (one entry per authoring expert) |
| This table | Human-readable mirror |
| ADR-033 mapping table | Canonical seed; CI keeps it identical to the manifest |

The mapping file lives outside `.claude/**` because that tree is CODEOWNERS-gated;
the drift guard reads the authoring experts there read-only.

## Mapping table

| Authoring expert (`.claude/commands/experts/`) | Runtime counterpart (`kind: AgentDef`) | Capability | Status |
|-----------------------------------------------|----------------------------------------|------------|--------|
| `go-services`     | `go-review-expert` | `code.review.go` | runtime (M7, `agents/examples/go-review-expert`) |
| `bdd-contract`    | `authoring-only`   | —                | authoring-only (deferred to M-dx) |
| `ci-release`      | `authoring-only`   | —                | authoring-only (deferred to M-dx) |
| `git-ops`         | `authoring-only`   | —                | authoring-only (deferred to M-dx) |
| `infra-helm`      | `authoring-only`   | —                | authoring-only (deferred to M-dx) |
| `post-merge`      | `authoring-only`   | —                | authoring-only (deferred to M-dx) |
| `python-adapters` | `authoring-only`   | —                | authoring-only (deferred to M-dx) |
| `spdd-canvas`     | `authoring-only`   | —                | authoring-only (deferred to M-dx) |

Only `go-services → go-review-expert` is dual-substrate in M7 (the reference
runtime expert). The rest are explicitly `authoring-only` until the full expert
library lands in M-dx — a deliberate, reviewable declaration, not an omission.

## Drift guard (CI)

`automation/scripts/check_expert_mapping.py` enforces ADR-033's three rules:

1. **Declared mapping is mandatory** — every authoring expert appears in the
   mapping file with a non-empty `runtime_mapping`.
2. **Runtime reference must resolve** — a named `runtime_mapping` resolves to
   `agents/examples/<name>`.
3. **Table reconciliation** — the mapping file stays identical to ADR-033's
   table.

Run locally with `make check-expert-mapping`; CI runs it in the `lint-python` job.
Adding an authoring expert without updating the mapping (and ADR-033's table)
fails the build.
