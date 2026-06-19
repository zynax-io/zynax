<!-- SPDX-License-Identifier: Apache-2.0 -->
# Declarative Scenario Manifests

A **scenario** wires together everything a demo or use-case needs — a `Workflow`,
the `AgentDef`(s) that supply its capabilities, and a reserved context slot — in
**one declarative place**, so you run your own scenario without hand-editing
compose files or adapter configs.

> Issue #1385 · Epic #1370 · Canvas `docs/spdd/1385-scenario-manifest/canvas.md`
> (approach **A2 — manifest-set convention**, ADR-028 "two kinds, no third schema").

---

## What a scenario is (and is not)

A scenario is **not** a new server-side manifest kind. It is a **client-side
manifest-set convention**: a directory under `spec/scenarios/<name>/` holding the
existing `kind: Workflow` and `kind: AgentDef` manifests (unchanged shapes) plus a
small `scenario.yaml` **index** that declares the members, the order to apply them
in, and a context slot.

`zynax apply <dir>` expands the index and submits each member over the **existing**
`/api/v1/apply` REST path — one member at a time, exactly as a hand-applied
manifest is. There is **no new api-gateway endpoint and no new response shape**:
the platform only ever sees the two kinds it already understands.

```
spec/scenarios/code-review/
├── scenario.yaml   ← the index (kind: Scenario)
├── agent.yaml      ← member: kind: AgentDef (the capability provider)
└── workflow.yaml   ← member: kind: Workflow (the runnable workflow)
```

---

## The index schema (`kind: Scenario`)

Validated against [`spec/schemas/scenario.schema.json`](../../spec/schemas/scenario.schema.json).

```yaml
kind: Scenario
apiVersion: zynax.io/v1alpha1
metadata:
  name: code-review            # lowercase DNS label; matches the directory name
  namespace: demo
  annotations:
    description: "..."
spec:
  members:                     # the member manifests this scenario composes
    - id: llm-agent            # stable id, referenced from apply_order
      kind: AgentDef           # only Workflow / AgentDef are valid members
      file: agent.yaml         # path RELATIVE to this index (no '..' escapes)
    - id: review-workflow
      kind: Workflow
      file: workflow.yaml
  apply_order:                 # member ids, in the order they are applied
    - llm-agent                # AgentDef(s) FIRST — see "apply order" below
    - review-workflow
  context:                     # RESERVED slot (semantics owned by #1387)
    language: go               # data only — never provider/model/endpoint/URL
```

| Field | Required | Notes |
|-------|----------|-------|
| `kind` | yes | Must be `Scenario`. |
| `apiVersion` | yes | Must be `zynax.io/v1alpha1`. |
| `metadata.name` | yes | Lowercase DNS label; conventionally the directory name. |
| `spec.members[]` | yes | Each `{id, kind, file}`; `kind` ∈ {Workflow, AgentDef}; `file` is relative. |
| `spec.apply_order[]` | yes | Member ids in apply order; each declared member referenced once. |
| `spec.context` | no | **Reserved** pass-through; see below. |

### Apply order

AgentDef members **must precede** the Workflow that consumes their capabilities.
Registering the AgentDef first means the task broker can route the capability when
the Workflow dispatches it; an unbacked capability then fails fast rather than
retrying forever (#1381). The index makes this order **declarative** — you never
sequence applies by hand.

### The reserved `context` slot

`spec.context` is a **wiring placeholder only** in #1385. The CLI does **not**
evaluate its values yet. The bounded `{files[], max_tokens}` slice semantics and
strict isolation are owned by sibling issue **#1387** (ADR-028); once that lands,
context values bind into a Workflow action's `input` Jinja2 `{{ .ctx.* }}`
templates.

**Hard rule (ADR-013):** the context slot may carry **data only** — never
provider/model/endpoint/URL or any field that redirects where a capability runs.
Provider/model/endpoint live in the `AgentDef`/compose overlay, never here.

---

## Authoring and running a scenario

```bash
# 1. Validate the whole set locally (index schema + each member's own schema):
zynax validate spec/scenarios/code-review
make validate-spec                         # the CI gate runs the same checks

# 2. Apply it — members submit in apply_order over the existing REST path:
zynax apply spec/scenarios/code-review
#   applying llm-agent (AgentDef)...
#   agent_id: ...
#   applying review-workflow (Workflow)...
#   run_id: ...

# 3. One-command demo (compose-up → apply → result → cleanup):
make demo SCENARIO=code-review
```

`zynax apply` accepts either the scenario **directory** or the `scenario.yaml`
index file directly. A plain manifest path still works exactly as before — the CLI
only treats a target as a scenario when it is a directory containing
`scenario.yaml`, or a file whose `kind` is `Scenario`.

---

## Authoring rules (recap)

- Member manifests follow the normal spec rules (`spec/AGENTS.md`): every manifest
  has `kind` + `apiVersion` + `metadata.name`; capabilities are `snake_case`;
  **never reference agent names in a Workflow** — only capability names.
- Member `file` paths are relative to the index and must not escape the scenario
  directory (no absolute paths, no `..`).
- Keep provider/model/endpoint out of manifests and out of the context slot
  (ADR-013) — they belong in the adapter config / compose overlay.

See also: [`docs/scenarios/code-review.validation.md`](code-review.validation.md)
for the human-validation walkthrough of the reference scenario.
