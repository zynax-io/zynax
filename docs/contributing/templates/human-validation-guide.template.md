<!-- SPDX-License-Identifier: Apache-2.0 -->
<!--
  Human-Validation Guide TEMPLATE.
  Copy this file to docs/<area>/validation/<story-slug>.md (or alongside the feature),
  delete this comment, and replace every <…> placeholder with the real value.
  Standard this implements: ../human-validation-guide.md
  Rule: a stranger to Zynax must finish setup and reach a verdict in minutes.
-->
# Human-Validation Guide — <feature / story name>

> **Story:** #<issue>  ·  **Canvas:** `docs/spdd/<id>/canvas.md` — O-step <N>

## Purpose
<One or two sentences: what feature this validates and what "working" looks like.>

## Prerequisites
- <e.g. Docker Engine ≥ 24 running>
- <e.g. the `zynax` CLI on PATH (`make build` or the release binary)>
- <e.g. the demo model pulled: `docker compose … exec ollama ollama pull <model>`>

## Expected duration
<e.g. ~5 minutes after the model is pulled.>

## Setup
```bash
# Exact commands to bring the system into the starting state — copy verbatim.
<command>
```

## Steps
1. <First exact command the human runs.>
   ```bash
   <command>
   ```
2. <Second exact command.>
   ```bash
   <command>
   ```

## Expected observable result
```
<Quote the exact output / status line / rendered field the reader should see.>
```

## Pass / fail criteria
- [ ] **PASS** when <unambiguous observable condition, e.g. `zynax result` shows a non-empty summary>.
- [ ] **FAIL** when <the opposite, e.g. output is empty, errors, or the run never reaches a terminal state>.

<!-- One pass/fail criterion per user-observable acceptance criterion in the canvas / story. -->

## Teardown
```bash
# Return the machine to a clean state.
<command, e.g. docker compose -f … -f … down -v>
```

## Troubleshooting
| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| <what the reader sees> | <why> | <command or step to recover> |

## Feedback / bug reporting
If validation fails, capture and file:
- The exact command run and its full output.
- Expected vs observed result.
- Versions: `zynax version`, image tags, model name.
- Open an issue with the `area: docs` (or relevant) label and attach the above.
