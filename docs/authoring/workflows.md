<!-- SPDX-License-Identifier: Apache-2.0 -->
# Workflow Authoring Guide

> Audience: workflow authors. This guide explains how to write a Zynax
> `kind: Workflow` manifest — its states, transitions, CEL guards, and the
> cross-state data-flow bindings — and how to scaffold, validate, and apply it
> with the `zynax` CLI.
>
> Reference material: [`spec/schemas/workflow.schema.json`](../../spec/schemas/workflow.schema.json),
> the reusable template [`spec/templates/workflow/workflow.template.yaml`](../../spec/templates/workflow/workflow.template.yaml),
> and the runnable examples under [`spec/workflows/examples/`](../../spec/workflows/examples/).
> Governing ADRs: ADR-029 (data-flow bindings), ADR-015 (pluggable engine).

---

## 1. The shape of a Workflow

A workflow is a declarative state machine. Every manifest has four top-level keys
(all four are required by [`workflow.schema.json`](../../spec/schemas/workflow.schema.json)):

```yaml
kind: Workflow            # constant — distinguishes it from AgentDef
apiVersion: zynax.io/v1   # constant — the compiler rejects any other value
metadata: { ... }         # name, version, namespace, labels, annotations
spec: { ... }             # initial_state, optional triggers, states
```

The workflow compiler (`WorkflowCompilerService`) consumes this manifest; `make
validate-spec` and `zynax validate` check it against the schema before it ever
reaches the engine.

### metadata

| Field | Required | Rule |
|-------|----------|------|
| `name` | yes | lowercase DNS-style, `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, ≤253 chars, unique per namespace |
| `version` | no | SemVer 2.0.0 (e.g. `1.0.0`, `2.1.0-rc.1`). Omit for the unversioned baseline. See [Versioning](#6-versioning). |
| `namespace` | no | defaults to `default`; same DNS pattern, ≤63 chars |
| `labels` | no | arbitrary string key/value pairs for grouping and filtering |
| `annotations` | no | tooling metadata; conventionally carry a one-line `description` |

### spec

`spec` requires `initial_state` and a non-empty `states` map. `triggers` is
optional.

- **`initial_state`** — the state entered on submission. It **must** match a key
  under `states` (enforced by `zynax validate`'s data-flow pass, not the schema).
- **`triggers`** — CloudEvent patterns that auto-start a *new* instance. Omit for
  workflows you submit manually.
- **`states`** — the state machine itself (see [§2](#2-states)).

---

## 2. States

`spec.states` is a map of state name → state definition. State names are
lowercase and descriptive; they double as the routing targets of transitions.

```yaml
spec:
  initial_state: review
  states:
    review:                # ← a normal state
      actions: [ ... ]     # capability invocations run on entry
      on: [ ... ]          # outbound transitions
```

### State types

The optional `type` field (default `normal`) classifies behaviour:

| `type` | Meaning |
|--------|---------|
| `normal` | runs its actions, then waits for a transition event (default) |
| `terminal` | an end state — **must not** declare `on` transitions |
| `human_in_the_loop` | pauses execution until an external signal arrives |

**At least one terminal state must be reachable.** This is enforced by the
compiler. A typical workflow has two terminals — a success (`done`) and a
failure (`failed`/`abandon`) — as in
[`feature-implementation.yaml`](../../spec/workflows/examples/feature-implementation.yaml).

### Actions

Each state runs an ordered list of `actions` on entry. An action invokes one
**capability** — never an agent name. The task broker routes the capability to a
registered agent that advertises it (see the
[Expert Authoring Guide](experts.md) for how capabilities are declared).

```yaml
actions:
  - capability: request_review     # required — must match a registered AgentDef capability
    input:                         # template values passed to the agent
      pr_url: "{{ .trigger.pr_url }}"
    output:                        # write result fields back into context (see §4)
      feedback_summary: summary
    timeout: 24h                   # ISO-8601-style duration; absent → only the workflow TTL applies
```

- `input` values may reference workflow context (`{{ .context.key }}`), trigger
  data (`{{ .trigger.field }}`), or a cross-state binding (`$.states.<state>.output.<key>` — see [§4](#4-data-flow-cross-state-output--input)).
- `timeout` matches `^([0-9]+(ns|us|ms|s|m|h))+$` (e.g. `30m`, `1h`, `24h`).

### Transitions (`on`)

Outbound edges are evaluated **in order**; the first transition whose `event`
matches and whose `guard` (if any) is true fires. Terminal states have no `on`.

```yaml
on:
  - event: review.approved      # required — matched against the incoming event 'type'
    goto: merge                 # required — must be a defined state
  - event: review.needswork
    goto: fix
    set:                        # optional — write into context when this edge fires
      context.latest_feedback: "{{ .event.comments }}"
```

`zynax validate` confirms every `goto` (and `initial_state`) targets a defined
state — a typo like `goto: mrege` fails validation before deploy.

---

## 3. CEL guards

When two transitions share the same `event`, a `guard` disambiguates them. A
guard is a [CEL](https://github.com/google/cel-spec) expression evaluated in the
workflow context; the transition fires only when it returns `true`.

From [`code-review.yaml`](../../spec/workflows/examples/code-review.yaml):

```yaml
on:
  - event: review.timeout
    goto: escalate
    guard: "escalation_count < 2"     # escalate the first two timeouts
  - event: review.timeout
    goto: abandon
    guard: "escalation_count >= 2"    # give up after that
```

Guidelines:

- Reference context keys directly (`escalation_count`, not `.context.escalation_count`).
- Keep guards total — for a shared event, the guards should cover every case so
  the workflow never stalls with no matching transition.
- Guards only **select** an edge; to **mutate** context use `set:` on the
  transition or `output:` on an action.

---

## 4. Data-flow: cross-state `output:` / `input:`

Data-flow bindings (ADR-029) thread a result produced in one state into a later
state without round-tripping through a fragile template string.

**Publish** with an action's `output:` map — `<context-key>: <result-field>`:

```yaml
fix:
  actions:
    - capability: summarize_feedback
      input:
        feedback: "{{ .context.latest_feedback }}"
      output:
        feedback_summary: summary    # publish the action's `summary` result as `feedback_summary`
```

**Consume** in a later state with an `input:` binding rooted at
`$.states.<state>.output.<key>`:

```yaml
escalate:
  type: human_in_the_loop
  actions:
    - capability: notify_human
      input:
        review_summary: "$.states.fix.output.feedback_summary"   # ← cross-state binding
```

The binding path is literal — `$.states.<producing-state>.output.<published-key>`.
No `{{ }}` template indirection is needed (or used) for these bindings.

`feature-implementation.yaml` chains four stages this way:

```
plan      → publishes implementation_plan
implement → consumes the plan, publishes branch_name + diff_summary
verify    → consumes the branch,  publishes test_report
open_pr   → consumes branch + diff_summary to draft the PR body
```

```yaml
implement:
  actions:
    - capability: write_code
      input:
        plan: "$.states.plan.output.implementation_plan"   # consume
      output:
        branch_name: branch                                # publish
        diff_summary: summary
```

Contrast the binding styles:

| Reference | Use |
|-----------|-----|
| `{{ .trigger.field }}` | data from the event that started the instance |
| `{{ .context.key }}` | a value written earlier via `set:` or a same-key `output:` |
| `$.states.<state>.output.<key>` | a value published by another state's action `output:` |

---

## 5. Triggers

A trigger auto-starts a new instance when a matching CloudEvent arrives. Each
trigger needs an `event` type; `filter` adds key/value conditions where **all**
entries must match:

```yaml
triggers:
  - event: github.pull_request.opened
    filter:
      repo: "zynax-io/zynax"
      base_branch: "main"
```

Omit `triggers` entirely for workflows you submit by hand with `zynax apply`.

---

## 6. Versioning

`metadata.version` is an optional SemVer 2.0.0 string. The compiler and CLI
surface it so you can evolve a workflow without breaking in-flight instances:

- **Omit it** for the unversioned baseline (e.g. while iterating locally).
- **Bump it on every change** once the workflow is in use — `MAJOR` for a
  breaking state-machine change, `MINOR` for additive states/transitions,
  `PATCH` for fixes. The reusable template ships at `0.1.0`; the runnable
  examples pin `1.0.0`.

---

## 7. Authoring with the `zynax` CLI

### Scaffold from the reusable template

`zynax init workflow` copies the versioned baseline from
[`spec/templates/workflow/workflow.template.yaml`](../../spec/templates/workflow/workflow.template.yaml)
— a known-good starting point instead of a blank file. It runs entirely locally
(no api-gateway connection):

```bash
# Print a fresh manifest to stdout
zynax init workflow

# Name it and write it to a file
zynax init workflow my-release-pipeline -o my-release-pipeline.yaml
```

The optional `[name]` argument overrides `metadata.name`; `--output`/`-o` writes
to a file (otherwise stdout). `--template-dir` (default `spec/templates`) points
at the template tree.

### Validate before you apply

`zynax validate <file>` reads `kind:` from the YAML, loads the matching schema
from `spec/schemas/<kind>.schema.json`, and — for `kind: Workflow` — also runs
the **data-flow pass** that confirms `initial_state` and every transition `goto`
resolve to a defined state:

```bash
zynax validate my-release-pipeline.yaml          # human-readable; exits 1 on any error
zynax validate my-release-pipeline.yaml --format json
```

`make validate-spec` runs the same schema validation across the whole `spec/`
tree in CI.

### Apply

```bash
zynax apply spec/workflows/examples/code-review.yaml             # submit
zynax apply --dry-run spec/workflows/examples/code-review.yaml   # validate only, no submit
```

`--dry-run` validates the manifest without submitting; `--engine` forwards an
engine hint to `SubmitWorkflow` (ADR-015).

---

## 8. Worked examples

All of these live under [`spec/workflows/examples/`](../../spec/workflows/examples/)
and are validated in CI:

| File | Demonstrates |
|------|--------------|
| [`feature-implementation.yaml`](../../spec/workflows/examples/feature-implementation.yaml) | a four-stage data-flow chain (plan → implement → verify → open_pr) |
| [`code-review.yaml`](../../spec/workflows/examples/code-review.yaml) | loops, `human_in_the_loop`, timeouts, CEL guards, and `output:`/`input:` data-flow |
| [`research-task.yaml`](../../spec/workflows/examples/research-task.yaml) | a simpler linear task |
| [`agent-def-expert.yaml`](../../spec/workflows/examples/agent-def-expert.yaml) | a runtime expert manifest (see the [Expert Authoring Guide](experts.md)) |

---

## 9. Checklist before you ship

- [ ] `kind: Workflow` and `apiVersion: zynax.io/v1` are present and exact.
- [ ] `metadata.name` is lowercase DNS-style and unique in its namespace.
- [ ] `metadata.version` is bumped (SemVer) if the workflow is already in use.
- [ ] `initial_state` names a defined state; every `goto` targets a defined state.
- [ ] At least one terminal state is reachable; terminal states have no `on`.
- [ ] Shared-event transitions are disambiguated by total CEL guards.
- [ ] Every `capability` matches one declared by a registered AgentDef.
- [ ] Cross-state inputs use `$.states.<state>.output.<key>` against an action that
      actually publishes that key via `output:`.
- [ ] `zynax validate <file>` exits 0.

See also: the [Expert Authoring Guide](experts.md).
