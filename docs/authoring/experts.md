<!-- SPDX-License-Identifier: Apache-2.0 -->
# Expert Authoring Guide

> Audience: expert and task authors. This guide explains how to write a
> `kind: AgentDef` manifest — its capabilities and runtime — how an **expert**
> differs from a plain capability provider, and how the authoring ↔ runtime
> expert mapping ties a Claude Code authoring expert to a dispatchable runtime
> agent.
>
> Reference material: [`spec/schemas/agent-def.schema.json`](../../spec/schemas/agent-def.schema.json),
> the reusable templates
> [`spec/templates/expert/expert.template.yaml`](../../spec/templates/expert/expert.template.yaml)
> and [`spec/templates/task/task.template.yaml`](../../spec/templates/task/task.template.yaml),
> the reference runtime expert [`agents/examples/go-review-expert/`](../../agents/examples/go-review-expert),
> and the [expert mapping](../experts/mapping.md).
> Governing ADRs: ADR-028 (context-bounded experts), ADR-033 (expert-agent substrate).

---

## 1. Two substrates: authoring vs runtime experts

Zynax experts exist on **two substrates** ([mapping](../experts/mapping.md),
ADR-033):

- **Authoring experts** — `.claude/commands/experts/<slug>.md`. These drive the
  SPDD authoring/delivery loop (a Claude Code expert that ships a story). They do
  not run inside a workflow.
- **Runtime experts** — `kind: AgentDef` agents (under `agents/examples/`) that
  register in agent-registry and are dispatchable by the task broker inside a
  workflow.

This guide is primarily about **runtime** agents — the `AgentDef` manifest you
author so a capability becomes routable. The two substrates are reconciled by the
mapping table in [§5](#5-the-authoring--runtime-expert-mapping).

---

## 2. The shape of an AgentDef

Every manifest has four required top-level keys
([`agent-def.schema.json`](../../spec/schemas/agent-def.schema.json)):

```yaml
kind: AgentDef                 # constant — distinguishes it from Workflow
apiVersion: zynax.io/v1alpha1  # constant — the registry rejects any other value
metadata: { ... }              # name, version, namespace, labels
spec: { ... }                  # capabilities (required) + runtime (optional)
```

The registry derives its `AgentDef` record from this manifest when you submit it
with `zynax apply`.

### metadata

| Field | Required | Rule |
|-------|----------|------|
| `name` | yes | lowercase DNS-style, `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, ≤253; used as `agent_id` in the registry |
| `version` | no | SemVer 2.0.0; omit for the unversioned baseline. Bump on every **contract** change. |
| `namespace` | no | defaults to `default` |
| `labels` | no | Kubernetes label syntax; used by `ListAgents` selectors |

---

## 3. Capabilities

`spec.capabilities` is a non-empty, ordered list. Each capability is the unit the
task broker routes work to: when a workflow action names `capability: go_review`,
the broker dispatches it to an agent that declares `go_review`.

| Field | Required | Rule |
|-------|----------|------|
| `name` | yes | the routing key — `^[a-z][a-z0-9_]*$` (snake_case), 1–64 chars |
| `description` | no | shown in the registry UI and generated docs |
| `input_schema` | no | draft-07-subset JSON Schema; the broker validates dispatch payloads against it |
| `output_schema` | no | draft-07-subset JSON Schema for the COMPLETED event payload |
| `timeout_seconds` | no | positive integer; broker marks the task FAILED past this wall-clock limit |
| `max_retries` | no | ≥0; re-queues on failure (`0` = no retries) |

`input_schema`/`output_schema` must declare `type: object` at the top level. The
capability `name` is the contract the workflow author references — keep it stable.

Example, from the reference runtime expert
[`agent-def-expert.yaml`](../../spec/workflows/examples/agent-def-expert.yaml):

```yaml
spec:
  capabilities:
    - name: go_review
      description: >
        Rule-based Go code review. Inspects a Go diff or source file and returns
        structured findings (line, severity, message), a finding count, and an
        approval flag that is false when any finding has severity 'error'.
      input_schema:
        type: object
        required: [diff]
        properties:
          diff: { type: string, description: "Go source or unified-diff text." }
      output_schema:
        type: object
        required: [findings, finding_count, approved]
        properties:
          findings:      { type: array }
          finding_count: { type: integer, minimum: 0 }
          approved:      { type: boolean }
      timeout_seconds: 30
      max_retries: 1
```

---

## 4. Expert vs task: contracts and context isolation

Both experts and tasks are `AgentDef` manifests. The difference is convention,
labels, and contract shape:

- A **task** ([task template](../../spec/templates/task/task.template.yaml))
  declares a single capability with a minimal input/output contract and a
  `runtime` block. Label `kind: task`.
- An **expert** ([expert template](../../spec/templates/expert/expert.template.yaml))
  exposes an advisory `review` capability over a **strictly isolated context
  slice** (ADR-028). The slice is bound at dispatch time and hard-capped at
  `max_tokens`. Two hard rules:
  - The slice contains **only** the declared `files` (glob patterns), capped at
    `max_tokens`.
  - **Never** reference another expert's slice or output — strict isolation.

The expert template's `review` capability encodes this contract directly: its
`input_schema` requires a `context_slice` object with `files` and `max_tokens`,
and its `output_schema` requires structured `summary`, `recommended_actions`,
`reasons_decisions`, `confidence`, and `flags`.

### Labels that mark a runtime expert

The reference runtime expert distinguishes itself with labels so operators and
the mapping can select it via a `ListAgents` selector:

```yaml
metadata:
  labels:
    agent.zynax.io/kind: expert        # marks this AgentDef as a runtime expert
    agent.zynax.io/expert: go-review
```

### runtime block

The optional `runtime` block tells the control plane how to run the agent's gRPC
server:

```yaml
runtime:
  image: ghcr.io/zynax-io/zynax/go-review-expert:latest   # required — tag or digest
  env:
    LOG_LEVEL: info
    GRPC_PORT: "50051"
  resources:
    requests: { cpu: "100m", memory: "128Mi" }
    limits:   { cpu: "500m", memory: "512Mi" }
  replicas: 1
```

Keep secrets out of `env` — reference Kubernetes Secrets via the CLI rather than
inlining sensitive values.

---

## 5. The authoring ↔ runtime expert mapping

Each authoring expert (`.claude/commands/experts/<slug>.md`) declares a
`runtime_mapping` pointing at its runtime `AgentDef` counterpart — or the literal
`authoring-only` when none exists yet (X.5, #1205; ADR-033). The full table and
rules live in [`docs/experts/mapping.md`](../experts/mapping.md); the key facts
for authors:

- **Source of truth:** `automation/experts/runtime_mapping.yaml` (machine-readable,
  one entry per authoring expert). The table in the mapping doc and the ADR-033
  table mirror it.
- **Why it lives outside `.claude/**`:** that tree is CODEOWNERS-gated; the drift
  guard reads the authoring experts there read-only.
- **In M7 only `go-services → go-review-expert` is dual-substrate** (capability
  `code.review.go`, runtime at `agents/examples/go-review-expert`). The other
  authoring experts are explicitly `authoring-only` until the full library lands
  in M-dx — a deliberate, reviewable declaration, not an omission.

### Drift guard (CI)

`automation/scripts/check_expert_mapping.py` enforces three ADR-033 rules:

1. **Declared mapping is mandatory** — every authoring expert appears in the
   mapping file with a non-empty `runtime_mapping`.
2. **Runtime reference must resolve** — a named `runtime_mapping` resolves to
   `agents/examples/<name>`.
3. **Table reconciliation** — the mapping file stays identical to ADR-033's table.

Run it locally with `make check-expert-mapping` (CI runs it in `lint-python`).
**Adding a new authoring expert without updating the mapping file and ADR-033's
table fails the build.** So when you add a runtime expert that backs an authoring
expert, update the mapping in the same change.

---

## 6. Authoring with the `zynax` CLI

### Scaffold from the reusable template

`zynax init expert` copies the versioned baseline from
[`spec/templates/expert/expert.template.yaml`](../../spec/templates/expert/expert.template.yaml).
It runs entirely locally (no api-gateway connection):

```bash
# Print a fresh expert manifest to stdout
zynax init expert

# Name it and write it to a file
zynax init expert my-review-expert -o my-review-expert.yaml
```

The optional `[name]` argument overrides `metadata.name`; `--output`/`-o` writes
to a file; `--template-dir` (default `spec/templates`) points at the template
tree. For a plain task agent, copy `spec/templates/task/task.template.yaml`
directly (the CLI scaffolds the `workflow` and `expert` kinds).

### Validate

`zynax validate <file>` reads `kind:` from the YAML and validates against
`spec/schemas/agent-def.schema.json`:

```bash
zynax validate my-review-expert.yaml             # exits 1 on any error
zynax validate my-review-expert.yaml --format json
```

`make validate-spec` runs the same checks across `spec/` in CI.

### Apply (register)

```bash
zynax apply my-review-expert.yaml             # submit to agent-registry
zynax apply --dry-run my-review-expert.yaml   # validate only, no submit
```

---

## 7. Reference runtime expert

[`agents/examples/go-review-expert/`](../../agents/examples/go-review-expert) is
the M7 reference implementation — a Python adapter whose `go_review` capability
matches the `agent-def-expert.yaml` manifest's routing key. Use it as the
end-to-end model for a new runtime expert:

- `capability.json` — the declared capability contract
- `src/go_review_expert/agent.py` — the gRPC adapter implementation
- `tests/features/go_review.feature` — the BDD contract at the gRPC boundary

---

## 8. Checklist before you ship

- [ ] `kind: AgentDef` and `apiVersion: zynax.io/v1alpha1` are exact.
- [ ] `metadata.name` is lowercase DNS-style and unique in its namespace.
- [ ] `metadata.version` is bumped (SemVer) on any capability-contract change.
- [ ] Every capability `name` is snake_case (`^[a-z][a-z0-9_]*$`), 1–64 chars,
      and stable (workflows reference it).
- [ ] `input_schema`/`output_schema` declare `type: object` at the top level.
- [ ] For an **expert**: the slice declares only its own `files`, capped at
      `max_tokens`; it never reads another expert's slice or output (ADR-028).
- [ ] Runtime experts carry the `agent.zynax.io/kind: expert` label.
- [ ] `runtime.image` is fully qualified (tag or digest); no secrets inlined in `env`.
- [ ] If this runtime expert backs an authoring expert, the mapping file and
      ADR-033 table are updated in the same change (`make check-expert-mapping`).
- [ ] `zynax validate <file>` exits 0.

See also: the [Workflow Authoring Guide](workflows.md) and the
[expert mapping](../experts/mapping.md).
