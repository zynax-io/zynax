<!-- SPDX-License-Identifier: Apache-2.0 -->

# FAQ

Common questions about running, authoring, and operating Zynax. For step-by-step setup
see the [Quick Start](quickstart.md); for the full developer workflow see the
[Developer Guide](developer-guide.md).

---

## Getting started

**What do I need installed?**
Only Docker (Desktop, or Engine + the Compose plugin). Go, Python, and `buf` all run
inside containers. Run `make bootstrap` once after cloning. See the
[Quick Start](quickstart.md).

**How do I install the `zynax` CLI?**
`make install-cli` builds it to `~/bin/zynax` (ensure `~/bin` is on your `PATH`), or grab
a release binary — see [local-dev.md](local-dev.md). The CLI talks to the api-gateway over
HTTP REST only.

**How do I run a workflow?**
Bring the stack up (`zynax up`, or `make demo`), then `zynax apply <file>`. Try a real example:
```bash
zynax apply spec/workflows/examples/code-review.yaml
```
The [Examples Index](examples/index.md) lists every runnable manifest.

**How do I watch a run?**
`zynax status` shows the current state; `zynax logs <run-id> --follow` tails execution
events until the run reaches a terminal state. Add `--format json` for machine-readable
output. The same stream is available over REST at `GET /api/v1/workflows/{id}/logs` (SSE).

---

## Authoring

**Data-flow binding vs `{{ template }}` — which do I use?**
Use **data-flow bindings** (`output:` to publish, `$.states.<state>.output.<field>` to
consume) for values that flow between states; reserve templates for trigger/event
payloads. See [Best Practices → Data flow](best-practices.md#data-flow-over-templating)
and ADR-029.

**My workflow never finishes — why?**
Likely no reachable `type: terminal` state, or a `guard:` that no event satisfies. Every
path must end in a terminal state. Validate with `zynax apply --dry-run <file>` first.

**What's the difference between an authoring expert and a runtime expert?**
Authoring experts live in `.claude/` and assist development; runtime experts are
`kind: AgentDef` manifests the engine dispatches at run time. The
[Expert Authoring Guide](authoring/experts.md) covers both substrates and the mapping.

**How do I add a new capability provider?**
Copy a reference agent from `agents/examples/` (start with `echo/`), declare its
capabilities with `input_schema`/`output_schema`, and register it. See the
[Python Agent Guide](patterns/python-agent-guide.md).

**Is there a scaffolder?**
Yes — `zynax init workflow` and `zynax init expert` generate from the templates under
`spec/templates/`.

---

## Observability

**I enabled nothing and see no traces — is that a bug?**
No. Telemetry is **off by default**. Set the OTLP exporter endpoint env var to turn it on.
See [OpenTelemetry](observability/opentelemetry.md).

**How do I run the Uptrace UI locally?**
Use the observability Compose overlay; create the env file (credentials are never
committed) and bring it up. Step-by-step in [Running Uptrace](observability/uptrace.md).

**Traces appear but logs/metrics don't (or the UI is empty).**
See [Observability Troubleshooting](observability/troubleshooting.md) — it covers an empty
UI, a stack that won't start, and signal-by-signal checks. Sampling and retention tuning
is in [Sampling and Retention](observability/sampling.md).

---

## Git and operations

**How do I expose Git to an MCP client?**
Run `zynax mcp git`. It requires a **least-privilege** token scoped to only the
repositories and operations you need. Setup in the
[Git MCP server guide](git-mcp/README.md).

**Where are the architectural decisions recorded?**
In [docs/adr/INDEX.md](adr/INDEX.md). The M7 features (data-flow, observability, context
propagation, Git MCP, experts) map to ADR-029 through ADR-033.

---

## Upgrading

**How do I move from v0.5.0 to v0.6.0?**
Follow the [v0.5.0 → v0.6.0 Migration Guide](migration-v0.6.md). v0.6.0 is additive — no
breaking manifest changes — but adds data-flow bindings, log streaming, OTEL + Uptrace,
context propagation, the Git MCP shim, reusable templates, and a Go entry point for the
LLM adapter.
