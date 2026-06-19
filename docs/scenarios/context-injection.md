<!-- SPDX-License-Identifier: Apache-2.0 -->
# Declarative context-injection

> **Story:** #1387 · **Canvas:** `docs/spdd/1387-context-injection/canvas.md` (approach A2)
> **Schema:** `spec/schemas/context-injection.schema.json` · fills the reserved
> `spec.context` slot of a Scenario index (`spec/schemas/scenario.schema.json`, #1385).

Declare the content a scenario injects **once**, in its index — `zynax` injects it
into your workflow, **bounded and isolated**. No hand-pasting a diff into a prompt,
no re-pasting on every run.

## Why

Before #1387 the only way to ground a demo in real content was to paste it into the
workflow action by hand:

```yaml
actions:
  - capability: codereview
    input:
      prompt: |
        Review the following git diff ...
        ```diff
        <the entire diff pasted by hand, re-pasted every run, unbounded>
        ```
```

This mixes **instructions** (the prompt) with **data** (the diff), has no size
bound, and must be edited by hand for every new input. The context block makes the
injection **declarative**: name a file, set a budget, reference it from the prompt.

## How it works

1. A Scenario index declares a `spec.context` block: file-rooted `sources` plus a
   `max_tokens` budget.
2. A Workflow member references each source by key via the **existing**
   `{{ .ctx.<key> }}` template surface (the same one workflow actions already use —
   no new template engine).
3. At `zynax apply` (and `zynax validate`) time the CLI reads each source's files,
   applies the `max_tokens` cap, and binds the result into the Workflow's
   `{{ .ctx.<key> }}` references **before** the member is submitted over the
   existing `/api/v1/apply` REST path. Nothing new crosses the gRPC boundary.

## Authoring shape

```yaml
# spec/scenarios/<name>/scenario.yaml  (kind: Scenario)
spec:
  members: [ ... ]
  apply_order: [ ... ]
  context:
    sources:
      - key: diff               # referenced as {{ .ctx.diff }}
        files:
          - diff.patch          # relative to the scenario directory
      # multiple files are concatenated in declared order:
      # - key: notes
      #   files: [intro.md, details.md]
    max_tokens: 8000            # hard cap on combined source content
    overflow: truncate-oldest   # or: error
```

```yaml
# the Workflow member references the key:
actions:
  - capability: codereview
    input:
      prompt: |
        Review the following git diff ...
        ```diff
        {{ .ctx.diff }}
        ```
```

| Field | Meaning |
|-------|---------|
| `sources[].key` | The context key; referenced as `{{ .ctx.<key> }}`. Lowercase alphanumeric, `_`/`-` internal. |
| `sources[].files` | File paths **relative to the scenario directory**; concatenated in order. Absolute paths or paths escaping the directory are rejected. |
| `max_tokens` | Hard cap on combined content (≈4 chars/token heuristic). Required. |
| `overflow` | `truncate-oldest` (default) drops the earliest-declared sources first until the budget is met; `error` fails apply instead. |

## The data-only rule (security)

A context block is **strictly data-only** (ADR-013/ADR-035). It admits content
sources and a budget **only**. It can **never** carry `provider`, `model`,
`endpoint`, `url`, `base_url`, `api_key`, or any other field that could redirect
*where or how* a capability runs — those live in `AgentDef`/overlay config and are
never accepted from injected content. The CLI **rejects** any such field at compile
time (both the JSON Schema and the compiler enforce this), so injected content can
never repoint a capability at an attacker-controlled or paid provider.

Two further guarantees:

- **Bounded** — `max_tokens` caps the dispatched `input_payload`; injection can
  never silently inflate the payload (ADR-028).
- **Isolated** — one scenario's context is resolved from its own block only and
  never reaches another scenario (ADR-028 strict isolation).

## Validate locally

```bash
zynax validate spec/scenarios/code-review   # schema + per-member + context binding
make validate-spec                          # CI-equivalent spec gate
```

`zynax validate` fails fast on: a routing/provider field in the block, a missing
`max_tokens`, a source file that is missing or escapes the directory, or a
`{{ .ctx.<key> }}` reference the block does not supply.

## See also

- `docs/scenarios/scenario-manifest.md` — the scenario manifest-set convention (#1385).
- `docs/scenarios/code-review.context.validation.md` — human-validation guide for the worked example.
- ADR-028 — the context-slice injection contract (`{files[], max_tokens}` + isolation).
