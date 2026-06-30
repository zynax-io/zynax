# Runbook — see your workflow result

> **Watch your workflow run and see its result the same way whether it executed on Temporal or
> Argo.** This page is the reliable, copy-pasteable way to read a completed run's output **today**.

## When to use this

You ran a workflow, it reached `WORKFLOW_STATUS_COMPLETED`, and you want its result — but
`zynax result <run-id>` printed:

```
no result payload for run wf-<hex>
```

That error is a **known interim limitation**, not a failed run. `zynax result` hard-errors on a
COMPLETED run whose capabilities emit no `completion` field (for example a plain echo/hello-world
step), even though the run succeeded — see
[cmd/zynax/cmd/result.go](../../cmd/zynax/cmd/result.go). The permanent fix — declared
workflow-level outputs surfaced by `zynax result`, and a graceful exit-0 on empty — lands in
**M7.U O.9** ([#1538](https://github.com/zynax-io/zynax/issues/1538)), part of epic
[#1529](https://github.com/zynax-io/zynax/issues/1529). Until then, use any of the four methods
below. They all read the **same** lifecycle stream, so the result is identical no matter which
engine executed the run.

> Every command takes an explicit `<run-id>` (the `run_id: wf-<hex>` line printed by `zynax apply`).
> With **no** id, `zynax logs` / `zynax result` / `zynax status` target your most recent run
> (recorded by the last `zynax apply`). An explicit id always overrides.

---

## Method 1 — live tail (recommended)

Stream the run live; it exits automatically when the workflow reaches a terminal state. Capability
output is printed inline on an indented `output:` line — no parsing required.

```bash
zynax logs <run-id> --follow
```

Example tail of a completed run:

```
[2026-06-30T12:00:01Z] StateTransition                start → review (WORKFLOW_STATUS_RUNNING)
[2026-06-30T12:00:09Z] CapabilityCompleted            (WORKFLOW_STATUS_RUNNING)
    output: <the model's review text>
[2026-06-30T12:00:09Z] StateTransition                review → done (WORKFLOW_STATUS_COMPLETED)
```

If the run is already finished, the stream replays its events and then exits — `--follow` still
works after completion (within the engine's run-retention window).

---

## Method 2 — re-submit and stream from the start

If you missed the run or want a clean capture, submit it again and follow the new run id:

```bash
zynax apply spec/workflows/examples/code-review-ollama.yaml
# run_id: wf-<new-hex>
zynax logs <new-run-id> --follow
```

(Path is repo-relative; see [code-review-ollama.yaml](../../spec/workflows/examples/code-review-ollama.yaml).)

---

## Method 3 — replay after completion (scripting / JSON)

For programmatic extraction, read the JSON event stream and pull the completion text with `jq`. This
handles both payload shapes the gateway emits — a bare `{"completion": "..."}` and a capability event
that wraps it in `result_payload`:

```bash
zynax logs <run-id> --format json \
  | jq -r 'select(.payload != null and .payload != "")
           | (.payload | fromjson) as $p
           | ($p.result_payload // null) as $rp
           | (if $rp != null then ($rp | fromjson | .completion) else $p.completion end) // empty'
```

The last non-empty line is your result. Runs with no completion field (e.g. echo/hello-world) print
nothing here — that is expected, and is exactly the case the interim `zynax result` mishandles.

---

## Method 4 — prove the run reached a terminal state

Confirm the run actually completed (rather than still running or failed):

```bash
zynax status workflow <run-id>
# WORKFLOW_STATUS_COMPLETED
```

`zynax status workflow` exits `0` when the run is terminal (completed/failed/cancelled) and `2` while
it is still running — handy in scripts that wait for completion before reading the result.

---

## Summary

| Method | Command | Best for |
|--------|---------|----------|
| 1 — live tail | `zynax logs <id> --follow` | Watching a run and reading its output (recommended) |
| 2 — re-submit | `zynax apply <file>` → `zynax logs <new-id> --follow` | A clean capture from the start |
| 3 — JSON replay | `zynax logs <id> --format json \| jq …` | Scripting / extracting the completion text |
| 4 — status check | `zynax status workflow <id>` | Proving the run is terminal |

All four are engine-agnostic: the result is the same whether the run executed on Temporal or Argo.

---

## Related

- Quickstart golden path: [docs/quickstart.md](../quickstart.md)
- Epic M7.U (workflow-level output capture): [#1529](https://github.com/zynax-io/zynax/issues/1529)
- Permanent fix — `zynax result` reads declared outputs (O.9): [#1538](https://github.com/zynax-io/zynax/issues/1538)
- Canvas: [docs/spdd/1529-workflow-output-capture/canvas.md](../spdd/1529-workflow-output-capture/canvas.md)
