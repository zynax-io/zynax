<!-- SPDX-License-Identifier: Apache-2.0 -->
# Human-Validation Guide — Declarative context-injection (code-review)

> **Story:** #1387 · **Canvas:** `docs/spdd/1387-context-injection/canvas.md` — O-step 4 & 5

## Purpose
Validates that a scenario injects **real content declaratively** through the
`spec.context` block — the code-review demo reviews a git diff that lives in a
**file**, bound into the workflow prompt via `{{ .ctx.diff }}`, with **no prompt
hand-edit**. "Working" means you can swap in your own diff by editing one file and
re-applying, and the run reviews exactly that diff.

## Prerequisites
- Docker Engine ≥ 24 running (`docker compose` v2).
- The `zynax` CLI on PATH (`make install-cli`, then ensure `~/bin` is on PATH).
- The demo model pulled once on the host: `ollama pull qwen2.5-coder:3b`.

## Expected duration
~5 minutes after the model is pulled.

## Setup
```bash
# From the repo root. Confirm the scenario (and its context block) validates first:
zynax validate spec/scenarios/code-review
```

## Steps
1. Confirm the diff is injected from a file, not pasted in the workflow:
   ```bash
   cat spec/scenarios/code-review/workflow.yaml   # prompt references {{ .ctx.diff }}
   cat spec/scenarios/code-review/scenario.yaml    # spec.context sources -> diff.patch
   cat spec/scenarios/code-review/diff.patch        # the actual diff under review
   ```
2. Bring the scenario up end-to-end:
   ```bash
   export ZYNAX_API_URL=http://localhost:7080
   zynax apply spec/scenarios/code-review
   zynax result <run-id-printed-above>
   ```
3. **Inject your own diff with no prompt edit** — replace only the data file:
   ```bash
   git diff > spec/scenarios/code-review/diff.patch   # your own changes
   zynax apply spec/scenarios/code-review
   zynax result <new-run-id>
   ```

## Expected observable result
```
applying llm-agent (AgentDef)...
agent_id: <id>
applying review-workflow (Workflow)...
run_id: <id>
── review ──────────────────────────────────────────────
<a non-empty review that refers to the SPECIFIC code in diff.patch
 (e.g. the Refund/Charge functions in the shipped example)>
────────────────────────────────────────────────────────
```

## Pass / fail criteria
- [ ] **PASS** — `zynax validate spec/scenarios/code-review` exits 0 with the context block present.
- [ ] **PASS** — the workflow prompt contains `{{ .ctx.diff }}` and **no** inline diff; the diff lives only in `diff.patch`.
- [ ] **PASS** — the review refers to the actual functions in `diff.patch` (the injected content reached the model).
- [ ] **PASS** — replacing `diff.patch` (Step 3) and re-applying reviews the **new** diff, with no edit to `workflow.yaml`.
- [ ] **PASS** — a context block carrying a `provider`/`model`/`endpoint`/`url` field is rejected by `zynax validate` (data-only safeguard). Try it:
  ```bash
  # temporarily add `endpoint: http://evil` under spec.context, then:
  zynax validate spec/scenarios/code-review   # MUST fail citing the forbidden field
  ```
- [ ] **FAIL** — any unresolved `{{ .ctx.* }}` reference, an empty review, or a routing field accepted silently.

## Teardown
```bash
git checkout spec/scenarios/code-review/diff.patch   # restore the shipped example diff
make demo-clean
```

## Troubleshooting
| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `context: ... is forbidden` | a routing field crept into `spec.context` | remove `provider`/`model`/`endpoint`/`url`; those belong in the AgentDef. |
| `{{ .ctx.diff }}` left unresolved | the `diff` source key is missing or misnamed | confirm `spec.context.sources[].key` matches the `{{ .ctx.<key> }}` reference. |
| `source file ... escapes the scenario directory` | a `files:` path used `../` or an absolute path | keep source files inside `spec/scenarios/code-review/`. |
| review ignores the diff | the diff exceeded `max_tokens` and was truncated | raise `max_tokens` or shorten the diff. |

## Feedback / bug reporting
If validation fails, capture the exact command + full output, expected vs observed,
and versions (`zynax version`, model `qwen2.5-coder:3b`). Open an issue referencing
#1387 with the `area: spec` label.
