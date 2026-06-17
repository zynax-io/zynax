<!-- SPDX-License-Identifier: Apache-2.0 -->

# Examples Index

A catalogue of the runnable examples that ship with Zynax: real workflows, manifest
templates, and reference agents. Everything here is verified against the running stack —
follow the [Quick Start](../quickstart.md) to bring the platform up, then `zynax apply`
any workflow below.

> **New here?** Start with the [Quick Start](../quickstart.md), then read the
> [Workflow Authoring Guide](../authoring/workflows.md) and
> [Expert Authoring Guide](../authoring/experts.md). This page is the map; those guides
> explain the concepts.

---

## Workflows — `spec/workflows/examples/`

### Real, runnable reference workflows

These three carry the `real, runnable reference (EPIC T.3, #1208)` marker and compile
green end-to-end. They are the recommended starting points for authoring your own.

| File | `metadata.name` | Demonstrates |
|------|-----------------|--------------|
| [`code-review.yaml`](../../spec/workflows/examples/code-review.yaml) | `code-review-workflow` | Loops, async events, human-in-the-loop, timeout handling, and **data-flow bindings** — `fix` publishes `feedback_summary` via `output:`, `escalate` consumes it via `$.states.fix.output.feedback_summary`. |
| [`ci-pipeline.yaml`](../../spec/workflows/examples/ci-pipeline.yaml) | `ci-pipeline-workflow` | Test → scan → build → deploy pipeline. `build` publishes `built_image` via `output:`; the deploy state consumes it with an input binding. |
| [`feature-implementation.yaml`](../../spec/workflows/examples/feature-implementation.yaml) | `feature-implementation-workflow` | Plan → implement → verify → open-PR flow for a tracked feature issue, with data-flow handoff between states. |

```bash
zynax apply spec/workflows/examples/code-review.yaml
zynax apply --dry-run spec/workflows/examples/ci-pipeline.yaml   # validate without running
```

### Additional workflow examples

| File | `kind` / name | Purpose |
|------|---------------|---------|
| [`research-task.yaml`](../../spec/workflows/examples/research-task.yaml) | Workflow · `research-task-workflow` | Iterative web research with human review and approval. |
| [`e2e-demo.yaml`](../../spec/workflows/examples/e2e-demo.yaml) | Workflow · `e2e-demo` | Minimal capability-dispatch fixture exercising the full dispatch chain end-to-end. |

### Manifest examples (non-Workflow kinds)

| File | `kind` / name | Purpose |
|------|---------------|---------|
| [`agent-def-example.yaml`](../../spec/workflows/examples/agent-def-example.yaml) | AgentDef · `code-review-agent` | A capability provider declaring `summarize`, `score_complexity`, and `ping` with input/output schemas and a runtime block. |
| [`agent-def-expert.yaml`](../../spec/workflows/examples/agent-def-expert.yaml) | AgentDef · `go-review-expert` | A **runtime expert** template; mirrors the `go-review-expert` reference agent. See [Expert Authoring](../authoring/experts.md). |
| [`policy-example.yaml`](../../spec/workflows/examples/policy-example.yaml) | Policy · `platform-policy` | Capability routing, rate-limiting, and quota rules. |

---

## Templates — `spec/templates/`

Copy-and-fill scaffolds for new manifests. `zynax init workflow` and `zynax init expert`
generate from these.

| File | For |
|------|-----|
| [`workflow/workflow.template.yaml`](../../spec/templates/workflow/workflow.template.yaml) | New `kind: Workflow` manifests. |
| [`expert/expert.template.yaml`](../../spec/templates/expert/expert.template.yaml) | New runtime-expert `kind: AgentDef` manifests. |
| [`task/task.template.yaml`](../../spec/templates/task/task.template.yaml) | Single-task manifests. |

---

## Reference agents — `agents/examples/`

Runnable adapter agents you can build, test, and register as capability providers. All
three are Python SDK adapters (see the [Python Agent Guide](../patterns/python-agent-guide.md)).

| Directory | Capability | What it does |
|-----------|-----------|--------------|
| [`echo/`](../../agents/examples/echo/) | `echo` | Copies its input payload to its output payload — the smallest possible adapter. |
| [`summarizer/`](../../agents/examples/summarizer/) | `summarize` | Produces a short extractive summary from a list of documents. |
| [`go-review-expert/`](../../agents/examples/go-review-expert/) | `go_review` | Rule-based Go code review; returns structured findings and an approval flag. The reference **runtime expert**. |

Each agent directory contains a `capability.json`, a `pyproject.toml`, the adapter source
under `src/`, and BDD `.feature` tests under `tests/features/`.

### LLM adapter — `agents/adapters/llm/`

The [`llm`](../../agents/adapters/llm/) adapter is the production capability provider for
LLM calls. As of v0.6.0 it ships a **Go** entry point (`cmd/llm-adapter/`) alongside the
Python source, with an [`agent-def.yaml.example`](../../agents/adapters/llm/agent-def.yaml.example)
showing how to register it. See the [v0.5.0 → v0.6.0 migration guide](../migration-v0.6.md).

---

## Where to go next

- [Best Practices](../best-practices.md) — conventions for authoring, data-flow, and observability.
- [FAQ](../faq.md) — common questions and gotchas.
- [Observability guides](../observability/opentelemetry.md) — trace and log your runs in Uptrace.
- [Git MCP server](../git-mcp/README.md) — expose Git operations to MCP clients.
