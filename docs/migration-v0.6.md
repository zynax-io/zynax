<!-- SPDX-License-Identifier: Apache-2.0 -->

# Migration Guide: v0.5.0 → v0.6.0

v0.6.0 (milestone M7) makes workflows **usable and observable**. It is an **additive**
release: existing v0.5.0 manifests continue to apply unchanged, and no proto contract or
event schema breaks. This guide lists what is new, what you can opt into, and the few
configuration steps required to use the new capabilities.

> **TL;DR — nothing breaks.** Upgrade the platform images and CLI, and your v0.5.0
> workflows keep running. Everything below is opt-in.

---

## At a glance

| Area | v0.5.0 | v0.6.0 | ADR |
|------|--------|--------|-----|
| Cross-state data | Context strings / templates only | Explicit **output/input data-flow bindings** | ADR-029 |
| Run visibility | `zynax status` | `zynax logs` streaming + `/logs` REST (SSE) | — |
| Observability | None built in | **OTEL traces/metrics/logs + Uptrace** UI | ADR-030 |
| Context | Per-hop, ad hoc | **Trace / data / correlation propagation** across hops | ADR-031 |
| Git access | Direct adapter calls | **Git MCP shim** (`zynax mcp git`) | ADR-032 |
| Experts | Manifest only | **Expert-agent substrate** + reference agents | ADR-033 |
| Authoring | Hand-written manifests | **Reusable templates** + `zynax init` | — |
| LLM adapter | Python only | **Go entry point** alongside Python | — |

---

## 1. Workflow data-flow bindings (ADR-029)

You can now pass typed values between states explicitly instead of threading them through
context strings.

- **Publish** a result field from the producing action:
  ```yaml
  actions:
    - capability: summarize_feedback
      input: { feedback: "{{ .context.latest_feedback }}" }
      output:
        feedback_summary: summary   # expose the action's `summary` result as `feedback_summary`
  ```
- **Consume** it downstream with a JSONPath binding rooted at the producing state:
  ```yaml
  review_summary: "$.states.fix.output.feedback_summary"
  ```

**Migration:** optional. Existing `{{ .context.* }}` templates still work. Adopt bindings
where you currently round-trip data through context — see
[`code-review.yaml`](../spec/workflows/examples/code-review.yaml) and
[`ci-pipeline.yaml`](../spec/workflows/examples/ci-pipeline.yaml), and the
[authoring guide](authoring/workflows.md).

## 2. Execution log streaming

A new way to watch runs:

```bash
zynax logs <run-id>            # stream execution events
zynax logs <run-id> --follow   # tail until the run reaches a terminal state
zynax logs <run-id> --format json
```

The same stream is served over REST as Server-Sent Events at
`GET /api/v1/workflows/{id}/logs`. Each event carries `timestamp`, `eventType`,
`fromState`, `toState`, and `status`.

**Migration:** none. New, purely additive command and endpoint.

## 3. OpenTelemetry + Uptrace observability (ADR-030)

Go services emit OTEL traces and RED metrics; Python adapters emit traces and logs.
**Telemetry is off by default** — enable it by pointing services at an OTLP collector:

```bash
ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:7017
```

Run the Uptrace UI locally with the observability Compose overlay, or in-cluster with the
Uptrace Helm chart. Credentials live in a gitignored env file — never commit them.

**Migration:** opt-in. Set the env var and start the overlay. Full steps:
[OpenTelemetry](observability/opentelemetry.md) · [Running Uptrace](observability/uptrace.md) ·
[Sampling and Retention](observability/sampling.md) ·
[Troubleshooting](observability/troubleshooting.md).

## 4. Context propagation (ADR-031)

Three kinds of context now flow across every hop (gRPC, Temporal, NATS):

- **Trace context** — W3C `traceparent`, so a run is one trace end-to-end.
- **Correlation context** — `x-request-id` and `x-namespace` headers.
- **Data context** — scoped read/write workflow data (the basis for §1 bindings).

**Migration:** none for authors — propagation is automatic. If you wrote custom gRPC
clients, ensure they forward incoming metadata; the platform clients already do.

## 5. Git MCP shim (ADR-032)

Expose Git operations to MCP clients through the git-adapter:

```bash
zynax mcp git
```

It requires a **least-privilege** token scoped to only the repositories and operations
you need. Setup and the `.mcp.json` shape are in the [Git MCP guide](git-mcp/README.md).

**Migration:** none. New optional surface.

## 6. Reusable templates + scaffolding

Manifest scaffolds now ship under `spec/templates/` and the CLI can generate from them:

```bash
zynax init workflow
zynax init expert
```

**Migration:** none. Optional authoring convenience. See the
[Examples Index → Templates](examples/index.md#templates--spectemplates).

## 7. Expert-agent substrate (ADR-033)

Runtime experts are `kind: AgentDef` manifests dispatched by the engine, with reference
agents under `agents/examples/` (`echo`, `summarizer`, `go-review-expert`) and a
runtime-expert manifest example
([`agent-def-expert.yaml`](../spec/workflows/examples/agent-def-expert.yaml)).

**Migration:** none. See the [Expert Authoring Guide](authoring/experts.md).

## 8. Go entry point for the LLM adapter

The LLM adapter (`agents/adapters/llm/`) now ships a **Go** entry point
(`cmd/llm-adapter/`) in addition to its Python source, with a registration example at
[`agent-def.yaml.example`](../agents/adapters/llm/agent-def.yaml.example).

**Migration:** none. The Python adapter remains; the Go binary is an additional deployment
option.

---

## Upgrade checklist

1. Pull the v0.6.0 platform images and rebuild/reinstall the CLI (`make install-cli`).
2. Re-apply your existing workflows — they should apply unchanged.
3. (Optional) Enable observability: set `ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT` and start the
   Uptrace overlay.
4. (Optional) Adopt data-flow bindings where you currently round-trip values through context.
5. (Optional) Wire the Git MCP shim and/or the Go LLM adapter as needed.

No manifest rewrites are required to reach v0.6.0. See the [FAQ](faq.md) for common
upgrade questions.
