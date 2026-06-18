<!-- SPDX-License-Identifier: Apache-2.0 -->
# Human-Validation Guide Standard

The single source of truth for **what a human-validation guide is** and **when a story
needs one**. A human-validation guide lets a person who has never seen Zynax execute a
user-visible feature by hand and report a clear pass or fail — without reading the code,
the tests, or the PR diff.

It complements, never replaces, automated tests. Unit/BDD tests prove the contract holds
in CI; a human-validation guide proves a *real human* can reach the advertised outcome on
their own machine. Both are required for user-visible work.

> Template to copy: [`templates/human-validation-guide.template.md`](templates/human-validation-guide.template.md)

---

## When a story needs one

A story **must** ship a human-validation guide when it is *user-visible* — i.e. it carries
an `audience: zynax-user` or `audience: developer` label, or its acceptance criteria
describe something a human observes (a CLI command, a rendered output, a demo path, an
error message). EPIC #1370 (the awesome-quickstart cluster) made this a standing safeguard:
*every user-visible story ships a human-validation guide.*

A story does **not** need one when it is purely internal — a refactor, a proto field with
no surfaced behaviour, a CI gate, or a library change with no user-observable effect.
Internal stories validate through their automated tests alone.

| Story is… | Needs a human-validation guide? |
|-----------|---------------------------------|
| `audience: zynax-user` / `audience: developer`, or user-observable behaviour | **Yes** |
| Pure internal refactor, CI gate, non-surfaced proto/library change | No |

---

## Required sections

A guide is a short, copy-runnable document. Keep it tight — a reader should finish setup
and reach a verdict in minutes, not hours. Every guide MUST contain these sections, in order:

1. **Purpose** — one or two sentences: what feature this validates and what "working" means.
2. **Prerequisites / preconditions** — exactly what must be installed and running before the
   reader starts (Docker, a model pulled, a compose stack up). State versions where they matter.
3. **Expected duration** — a realistic estimate (e.g. "~5 minutes after the model is pulled").
4. **Setup** — the exact commands that bring the system into the starting state. No prose
   substitutes; the reader copies these verbatim.
5. **Steps** — the numbered, exact commands a human runs to exercise the feature. One action
   per step. No "now configure X" hand-waving — show the command.
6. **Expected observable result** — what the reader should *see* after the steps: the exact
   terminal output, the rendered field, the status line. Quote it so the reader can compare.
7. **Pass / fail criteria** — an unambiguous checklist. "PASS if the `zynax result` output
   shows a non-empty review summary; FAIL otherwise." No interpretation required.
8. **Teardown** — the commands that return the machine to a clean state (stop the stack,
   remove volumes), so the reader leaves nothing running.
9. **Troubleshooting** — the two or three failure modes a first-timer hits, each with a fix.
10. **Feedback / bug-reporting** — what to capture (command, observed vs expected output,
    versions) and where to file it, so a failed validation becomes an actionable report.

Sections 1–8 are mandatory. Sections 9–10 are strongly recommended and required for any
guide on a default first-run or demo path.

---

## How it ties into SPDD and the PR body

A human-validation guide is the human-executable mirror of two existing artefacts:

- **The SPDD Canvas Acceptance Criteria.** Each *user-observable* acceptance criterion in
  the canvas O-step (or the story's `## Acceptance criteria`) should map to one **Pass / fail
  criterion** in the guide. If a criterion cannot be turned into a step a human runs and a
  result they observe, it is not yet testable — tighten the criterion, not the guide.
- **The PR "Test plan & acceptance" section.** The PR template
  ([`pr-templates.md`](pr-templates.md)) requires one matrix row per acceptance criterion with
  the *exact command* used to verify and the observed result. A user-visible PR cites its
  human-validation guide as the verification method for the user-observable rows — the guide's
  Steps + Expected observable result *are* the "How verified" column for a human path.

So the chain is: **canvas acceptance criterion → guide pass/fail criterion → PR test-plan row.**
The same observable outcome appears in all three, worded consistently.

---

## Worked grounding — the Ollama quickstart

The required sections are not abstract: they mirror how EPIC #1370's first runnable workflow
actually validated. A human brings up the compose stack with the Ollama overlay, applies the
demo workflow, and observes the terminal result:

```bash
# Setup
docker compose -f infra/docker-compose/docker-compose.yml \
  -f infra/docker-compose/docker-compose.ollama.yml up -d

# Steps
zynax apply spec/workflows/examples/code-review-ollama.yaml
zynax logs <run-id> --follow

# Expected observable result
zynax result <run-id>      # shows a non-empty code-review summary
```

PASS if `zynax result` prints a code-review summary; FAIL if it is empty or errors. Teardown
brings the stack down with `docker compose … down -v`. Copy the template, fill these in for
your story, and you have a complete guide.

---

*See also:* [`pr-templates.md`](pr-templates.md) · [`../patterns/spdd-guide.md`](../patterns/spdd-guide.md) · [`templates/human-validation-guide.template.md`](templates/human-validation-guide.template.md)
