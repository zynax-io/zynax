<!-- SPDX-License-Identifier: Apache-2.0 -->

# Best Practices

Conventions that keep Zynax workflows, experts, and operations maintainable. These
distil the authoring and observability guides into actionable rules. Where a topic has a
full guide, this page links to it rather than repeating it.

> See also: [Workflow Authoring](authoring/workflows.md) · [Expert Authoring](authoring/experts.md) ·
> [Observability](observability/opentelemetry.md) · [Examples Index](examples/index.md).

---

## Workflow authoring

- **Name and version every manifest.** Set `metadata.name`, `metadata.namespace`, and a
  semver `metadata.version`. The namespace plus name is the workflow's identity
  (ADR-034); reusing a name across namespaces is fine, reusing it within one is a
  collision.
- **Keep states single-purpose.** One responsibility per state; branch with `on:` events
  and `guard:` expressions rather than packing logic into one state.
- **Always declare terminal states.** Mark end states `type: terminal` so the engine
  knows the run is complete. A workflow with no reachable terminal state never finishes.
- **Set timeouts on long-running actions.** Use `timeout:` (e.g. `24h`) and route the
  emitted timeout event to an escalation or abandon path, as `code-review.yaml` does.
- **Dry-run before you apply.** `zynax apply --dry-run <file>` validates structure and
  bindings without starting a run.

## Data flow over templating

Prefer explicit **data-flow bindings** (ADR-029) to passing values through context
strings:

- Publish a result field with `output:` on the producing action:
  ```yaml
  output:
    feedback_summary: summary   # publish the action's `summary` result as `feedback_summary`
  ```
- Consume it downstream with a JSONPath input binding rooted at the producing state:
  ```yaml
  review_summary: "$.states.fix.output.feedback_summary"
  ```
- Reserve `{{ .context.* }}` / `{{ .event.* }}` templates for trigger and event payloads.
  Bindings are typed and traceable; deep template indirection is not. The
  [`code-review.yaml`](../spec/workflows/examples/code-review.yaml) and
  [`ci-pipeline.yaml`](../spec/workflows/examples/ci-pipeline.yaml) examples show both
  styles in context.

## Experts and agents

- **Pick the right substrate.** Authoring experts live in `.claude/`; runtime experts are
  `kind: AgentDef` manifests dispatched by the engine. See
  [Expert Authoring](authoring/experts.md) for which to use.
- **Declare schemas on every capability.** Give each capability an `input_schema` and
  `output_schema` with `required` fields and descriptions, as in
  [`agent-def-example.yaml`](../spec/workflows/examples/agent-def-example.yaml). The
  schema is the contract dispatch validates against.
- **Set `timeout_seconds` and `max_retries` per capability.** Fast health-checks
  (`ping`) get a short timeout and zero retries; expensive calls get longer windows.
- **Start from a reference agent.** Copy `agents/examples/echo/` for the smallest working
  adapter and grow from there. Commit the BDD `.feature` before the implementation
  (ADR-016).

## Observability

- **Telemetry is off by default.** Enable it by setting the OTLP endpoint env var; see
  [OpenTelemetry](observability/opentelemetry.md). Do not hard-code endpoints in manifests.
- **Watch runs live.** Use `zynax logs <run-id> --follow` to tail execution events to a
  terminal state, and `zynax status` to check current state.
- **Tune sampling deliberately.** Full-trace sampling is fine locally; sample down in
  shared environments — see [Sampling and Retention](observability/sampling.md).
- **Never commit credentials.** Uptrace and OTLP credentials go in gitignored env files,
  not in YAML or workflow manifests (placeholders only).

## Git and contributions

- **Least-privilege Git access.** The [Git MCP server](git-mcp/README.md) requires a
  scoped token — grant only the repositories and scopes a workflow needs.
- **One commit per logical change, one PR per issue.** See `CLAUDE.md` and the contributing
  docs for commit-type and PR-size rules.
- **Keep examples true.** If you change a manifest field or CLI flag, update the affected
  example and guide in the same change — docs are verified against the running stack.
