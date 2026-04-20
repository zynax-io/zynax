# spec/ — AGENTS.md

> The `spec/` directory contains the **declarative intent layer** of Zynax.
> This is Layer 1 of the three-layer model. See `ARCHITECTURE.md §2`.
>
> Everything in this directory is YAML. There is no Go or Python here.
> YAML in `spec/` is never imported by code — it is compiled by the Workflow Compiler.

---

## What Lives Here

```
spec/
├── schemas/                  ← JSON Schema definitions for all manifest kinds
│   ├── workflow.schema.json  ← Validates Workflow manifests
│   ├── agent-def.schema.json ← Validates AgentDef manifests
│   ├── policy.schema.json    ← Validates Policy manifests
│   └── routing-rule.schema.json
└── workflows/examples/       ← Reference YAML manifests
    ├── code-review.yaml
    ├── ci-pipeline.yaml
    └── research-task.yaml
```

---

## Manifest Kinds

### Workflow
Defines an event-driven state machine. The primary user-facing primitive.

```yaml
kind: Workflow
apiVersion: zynax.io/v1

metadata:
  name: code-review-workflow
  namespace: engineering
  labels:
    team: platform
    tier: production

spec:
  # The state the workflow enters when submitted
  initial_state: review

  # Optional: events that trigger this workflow automatically
  triggers:
    - event: github.pull_request.opened
      filter:
        repo: "zynax/zynax"

  states:
    review:
      # Actions are capability invocations — never agent names
      actions:
        - capability: request_review
          input:
            pr_url: "{{ .event.pr_url }}"
          timeout: 24h
      on:
        - event: review.approved
          goto: merge
        - event: review.changes_requested
          goto: fix
        - event: review.timeout
          goto: escalate
          guard: "{{ .attempts }} < 3"

    fix:
      actions:
        - capability: summarize_feedback
          input:
            feedback: "{{ .event.comments }}"
      on:
        - event: push
          goto: review

    merge:
      actions:
        - capability: merge_pr
      on:
        - event: merge.success
          goto: done
        - event: merge.conflict
          goto: fix

    escalate:
      type: human_in_the_loop      # Pauses until a human signals continuation
      actions:
        - capability: notify_human
          input:
            message: "Review timeout — needs human intervention"

    done:
      type: terminal
```

### AgentDef
Declares a capability provider — an agent or adapter.
Does NOT describe how the capability runs — only what it can do.

```yaml
kind: AgentDef
apiVersion: zynax.io/v1

metadata:
  name: summarizer-agent
  namespace: engineering

spec:
  # The gRPC endpoint this agent/adapter listens on
  endpoint: "summarizer-agent:50060"

  # Declared capabilities — these are registered in agent-registry
  capabilities:
    - name: summarize
      description: "Summarise one or more documents into a concise paragraph."
      input_schema:
        type: object
        properties:
          documents:
            type: array
            items:
              type: string
        required: [documents]
      output_schema:
        type: object
        properties:
          summary:
            type: string

    - name: extract_keywords
      description: "Extract key topics from a document."
      input_schema:
        type: object
        properties:
          document: { type: string }
        required: [document]
      output_schema:
        type: object
        properties:
          keywords:
            type: array
            items: { type: string }

  # Optional: runtime hints for the scheduler
  resources:
    replicas: 2
    cpu: "500m"
    memory: "512Mi"
```

### Policy
Routing and scheduling policies for capability dispatch.

```yaml
kind: Policy
apiVersion: zynax.io/v1

metadata:
  name: summarize-routing-policy

spec:
  capability: summarize

  # Strategy: round_robin | least_loaded | affinity | priority
  strategy: affinity

  # Prefer the agent that last handled a task from the same workflow
  affinity:
    prefer_same_workflow: true

  # Fallback if preferred agent is unavailable
  fallback: round_robin

  # SLA
  timeout: 30s
  max_retries: 3
  retry_backoff: exponential
```

---

## YAML Authoring Rules

1. **Always include `kind` and `apiVersion`** — required for schema validation.
2. **Always include `metadata.name` and `metadata.namespace`** — namespaces are mandatory.
3. **Capabilities are lowercase, hyphenated** — `request_review`, not `RequestReview`.
4. **Never reference agent names** — only capability names. Routing is the platform's job.
5. **State names are lowercase, descriptive** — `review`, `fix`, `merge`, not `s1`, `s2`.
6. **Use `{{ .event.field }}` for input templates** — Jinja2-style, evaluated by compiler.
7. **Terminal states must be declared** — at least one state of `type: terminal` required.
8. **Validate locally before applying** — `make validate-spec` runs JSON Schema validation.

---

## Validation

```bash
# Validate all manifests in spec/ against their JSON Schemas
make validate-spec

# Validate a single file
make validate-spec FILE=spec/workflows/examples/code-review.yaml

# Dry-run a workflow (compile to IR without executing)
make dry-run WORKFLOW=spec/workflows/examples/code-review.yaml
```

CI blocks merge if any `.yaml` in `spec/` fails schema validation.
