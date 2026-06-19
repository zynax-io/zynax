<!-- SPDX-License-Identifier: Apache-2.0 -->
# Human-Validation Guide — Declarative code-review scenario

> **Story:** #1385  ·  **Canvas:** `docs/spdd/1385-scenario-manifest/canvas.md` — O-step 4 & 6

## Purpose
Validates that **one declarative scenario manifest set** runs a code-review demo to
a terminal result with **no imperative edits** — the AgentDef and Workflow are
wired together in `spec/scenarios/code-review/` and brought up by a single command.
"Working" means `zynax apply` (or `make demo SCENARIO=code-review`) registers the
AgentDef, submits the Workflow, and the run reaches a terminal state with a
non-empty model review.

## Prerequisites
- Docker Engine ≥ 24 running (`docker compose` v2).
- The `zynax` CLI on PATH (`make install-cli`, then ensure `~/bin` is on PATH).
- The demo model pulled once on the host: `ollama pull qwen2.5-coder:3b`
  (the Ollama overlay reuses host-pulled models read-only — nothing auto-downloads).

## Expected duration
~5 minutes after the model is pulled (most of it is the first model load).

## Setup
```bash
# From the repo root. Confirm the scenario validates locally first (no stack needed):
zynax validate spec/scenarios/code-review
```

## Steps
1. Validate the whole manifest set against its schemas (index + each member):
   ```bash
   zynax validate spec/scenarios/code-review
   ```
2. Bring the scenario up end-to-end with one command:
   ```bash
   make demo SCENARIO=code-review
   ```
   (Equivalent manual form, if you prefer to drive it yourself:)
   ```bash
   export ZYNAX_API_URL=http://localhost:7080
   zynax apply spec/scenarios/code-review
   zynax result <run-id-printed-above>
   ```

## Expected observable result
```
ok: spec/scenarios/code-review/scenario.yaml
...
applying llm-agent (AgentDef)...
agent_id: <id>
applying review-workflow (Workflow)...
run_id: <id>
── review ──────────────────────────────────────────────
<a non-empty, numbered code review ending in APPROVE or REQUEST-CHANGES>
────────────────────────────────────────────────────────
✅ Demo complete.
```

## Pass / fail criteria
- [ ] **PASS** — `zynax validate spec/scenarios/code-review` exits 0 (AC: "validated by `zynax`").
- [ ] **PASS** — `make validate-spec` passes including the scenario index (AC: "validated by `make validate-spec`").
- [ ] **PASS** — applying the scenario registers the AgentDef (prints `agent_id:`) **then** submits the Workflow (prints `run_id:`), in that order, with no manual edits to any compose/adapter/prompt file (AC: "one declarative manifest runs a scenario to a terminal result with no imperative edits").
- [ ] **PASS** — `zynax result <run-id>` prints a non-empty review and the run reaches a terminal state.
- [ ] **FAIL** — any of the above errors, the output is empty, members apply out of order, or the run never reaches a terminal state.

## Teardown
```bash
make demo-clean
```

## Troubleshooting
| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `zynax CLI not found` | `~/bin` not on PATH | `make install-cli`; add `~/bin` to PATH. |
| `apply did not return a run_id` | AgentDef not registered before Workflow, or model not pulled | Check `apply_order` lists the AgentDef first; run `ollama pull qwen2.5-coder:3b`. |
| Empty review / connection refused | adapter endpoint unreachable | Confirm the Ollama overlay is healthy: `docker compose ps`; see `infra/docker-compose/ollama/`. |

## Feedback / bug reporting
If validation fails, capture and file:
- The exact command run and its full output.
- Expected vs observed result.
- Versions: `zynax version`, image tags, model name (`qwen2.5-coder:3b`).
- Open an issue referencing #1385 with the `area: spec` label and attach the above.
