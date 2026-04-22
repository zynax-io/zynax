# docs/decisions/ — Tactical Decision Records

This directory captures **tactical decisions** — the "why we did it this specific
way" for implementation choices that are not architectural in nature but would
surprise or block a future contributor without context.

## ADRs vs Decision Records

| | ADR (`docs/adr/`) | Decision Record (`docs/decisions/`) |
|--|---|---|
| **Scope** | Strategic, cross-cutting | Tactical, localised |
| **Examples** | "Use gRPC for all inter-service comms", "Go for platform services" | "Split EngineAdapter tests into two files", "PR size limit is 900" |
| **Longevity** | Stable — rarely changes | Can be superseded easily |
| **RFC required** | Yes (for new ADRs) | No |
| **Format** | Full ADR template | Short (see below) |

If a decision would affect multiple milestones or multiple teams, write an ADR.
If it's a one-time judgment call within a module or workflow, write a decision record.

## Format

```markdown
# NNN: Short Title

**Date:** YYYY-MM-DD  **Author:** name or "M1 Engineering"

## Context

One paragraph: what was the situation, what was the question being answered.

## Decision

What we chose and the specific parameters (numbers, names, patterns).

## Alternatives considered

Brief bullets on what else was evaluated and why it was not chosen.

## Consequences

What this means for contributors going forward.
```

## Index

| # | Title | Scope |
|---|-------|-------|
| [001](001-engine-adapter-test-split.md) | EngineAdapter test two-file split | `protos/tests/engine_adapter_service/` |
| [002](002-pr-size-900-limit.md) | PR size limit 900 lines | Repository-wide |
| [003](003-bufconn-for-contract-tests.md) | bufconn for in-process gRPC tests | `protos/tests/` |
| [004](004-gowork-off-isolation.md) | GOWORK=off for contract tests | `protos/tests/` |
