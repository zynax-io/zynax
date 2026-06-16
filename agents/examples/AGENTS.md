<!-- SPDX-License-Identifier: Apache-2.0 -->

# agents/examples — AGENTS.md

> Canonical "write-your-own-agent" reference agents built on the Zynax SDK.
> Inherits all rules from root `AGENTS.md`, `agents/AGENTS.md`, and `agents/sdk/AGENTS.md`.

---

## Purpose

These are copy-me starting points for new SDK agents. Each one subclasses
`zynax_sdk.Agent`, exposes one `@capability(...)`, and ships a capability JSON Schema
plus a BDD `.feature`. They are deterministic and dependency-free so they build, lint,
and test offline — swap the handler body for an LLM or framework call to make one real.

| Agent | Capability | Shows |
|-------|-----------|-------|
| `echo` | `echo` | The smallest possible agent: progress + completed, payload round-trip |
| `summarizer` | `summarize` | Structured input, validation, `report_failed` on empty input |
| `go-review-expert` | `go_review` | An *expert*-style agent (ADR-033 runtime substrate), rule-based findings |

> Runtime `kind: AgentDef` registration that makes `go-review-expert` dispatchable
> inside a workflow is delivered separately (EPIC X step X.3, #1203). This directory
> is the SDK agents + schemas + BDD only.

---

## Layout (per agent)

```
agents/examples/<name>/
├── pyproject.toml          ← uv project; depends on zynax-sdk via [tool.uv.sources] path
├── capability.json         ← capability JSON Schema (input/output)
├── src/<pkg>/
│   ├── __init__.py
│   └── agent.py            ← Agent subclass + @capability handler
└── tests/
    ├── conftest.py         ← adds protos/generated/python to sys.path
    ├── features/<cap>.feature
    └── test_<cap>.py       ← pytest-bdd step definitions
```

The SDK is untyped (no `py.typed`), so the `Agent` subclass line and the `@capability`
decorator carry a single targeted `# type: ignore` each — this is expected under
`mypy --strict` and mirrors the SDK's own `agent.py`.

---

## Gates (run via Docker tools image)

```bash
make lint-agent AGENT=<name>        # ruff + mypy --strict
make test-unit-agent AGENT=<name>   # pytest (CI loop adds --cov-fail-under=90)
```

`make lint-agents` / `make test-unit-agents` discover every `agents/examples/*/pyproject.toml`
automatically — no manual list to update when adding a new agent.
