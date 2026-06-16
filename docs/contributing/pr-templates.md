# PR Body Templates

The single source of truth for **how a pull request body is structured** in Zynax.

Use it for every PR — opened by a human or by the milestone automation
(`/milestone-orchestrate`, `/issue-deliver`, and the expert subagents). The goal is a
review that is *fast and complete*: a reviewer can see **what to expect, why, how it was
verified, and the evidence** without reading the whole diff, and nothing gets lost.

> **Why this exists separately from `.github/PULL_REQUEST_TEMPLATE.md`:** GitHub only
> auto-applies that native template when a PR is opened *interactively* (web UI, or
> `gh pr create` with no body flag). The automation opens PRs with
> `gh pr create --body-file …`, and an explicit body **silently overrides** the native
> template. So the automation must *build* the body from this document. The native
> template mirrors the same skeleton so human and automated PRs look identical.

---

## The skeleton (every PR, every type)

Fill every section. Delete a section only if the per-type table below says it does not apply.

```markdown
## <type>(<scope>): <subject>

Closes #<issue>          <!-- REQUIRED: auto-closes the issue on merge.
                              For a canvas-step add: "Part of #<epic>" -->

### Why  (problem & intent)
<1–3 sentences: what is missing or broken, and why this change is needed.
 Do NOT restate what the code does — the diff shows that. For feat:, link the canvas O-step.>

### What you'll get  (deliverables ↔ what changed)
<Pair each deliverable with the concrete change that provides it — "expect X because Y changed".
 This is the "what to expect to be delivered and why" view, derived from the diff.>
- <deliverable a reviewer/operator will observe> ← `path/to/change`
- …

### Scope & boundaries
- **In scope:** <…>
- **Out of scope (deferred):** <… → #<issue> / M-dx / N/A>

### Test plan & acceptance
<The heart of the review. One row per Acceptance Criterion from the issue/canvas.
 Show the EXACT command used to verify and the observed result — not "tested manually".>

| Acceptance criterion | How verified (command) | Result |
|----------------------|------------------------|--------|
| <AC 1 verbatim>      | `<exact command>`      | ✅ <number / output snippet> |
| <AC 2 verbatim>      | `<exact command>`      | ✅ <…> |

**Local gates:** `make lint` ✅ · `GOWORK=off go test ./… -race` ✅ · `make security` ✅ · `make validate-spec` ✅
<list only the gates that apply to this change>

### Evidence
- **CI:** <all required checks green / link to the run>
- **Local output:** <coverage %, benchmark `ns/op`, `helm lint`, render preview — paste the decisive line>
- **Artifacts / images:** <`service:tag@sha256:…` if images were built, or "none — no service source changed">
- **Post-merge digest sync → main:** <`chore(images): sync digests after main-<sha>` commit SHA, or "N/A — no image rebuild">
  <!-- Left as a placeholder by the author; the post-merge verifier fills it after the
       release pipeline runs (see /milestone-orchestrate STEP 7.5). -->

### Risk & rollback
<Blast radius · how to revert (revert this PR? flip a flag?) · any data/migration concern.
 "Low — additive only, no behaviour change to existing paths" is an acceptable answer.>

### Review aids
<Make the review easy: suggested reading order, the ONE key file, any subtlety a reviewer
 would otherwise miss, anything intentionally left for a follow-up.>

<!-- feat: only --> **SPDD:** Canvas `docs/spdd/<id>/canvas.md` — Status: **Aligned** · `/spdd-security-review` **PASS**
**AI:** `Assisted-by: Claude/<model-id>`   (and set the `ai-assisted` label)
```

---

## Per-type emphasis

The skeleton is the same; each type *emphasises* different sections. Keep the others, but spend
your words where the table says.

| Type | Emphasis / required extras | Sections that are usually `N/A` |
|------|----------------------------|---------------------------------|
| **feat** | Full template. **SPDD line required** (canvas Aligned + security-review PASS). Test plan maps every canvas AC. Evidence shows new observability (log event / metric / trace). | — |
| **fix** | **Why = symptom + root cause** (not just "fixes bug"). A **regression test is mandatory** in the matrix: the test that fails on `main` and passes here. Evidence shows the repro **before → after**. | SPDD |
| **docs** | Test plan = render/preview check, link-check, and `make validate-spec` when specs/schemas change. Evidence = preview link. | Artifacts/images; SPDD; observability |
| **test** | "What you'll get" = the scenarios/coverage added. Evidence = coverage % and/or benchmark baseline numbers + the exact command to reproduce. | SPDD (unless the issue is `feat:`) |
| **ci** | Test plan = a **green run of the changed workflow** (paste the link). Evidence = that run + what it now gates. | SPDD |
| **chore** | Why = the maintenance rationale (CVE id, dep bump reason, tooling). Evidence = the dep/digest diff and that build+lint stay green. | SPDD; observability |

---

## Filled example (a `feat:` canvas-step)

```markdown
## feat(observability): trace_id exemplars on RED metrics

Closes #1187
Part of #467 (EPIC O — Observability)

### Why
Dashboards show rate/error/duration but cannot jump from a spiking metric to the trace that
caused it. O.4 adds OpenTelemetry exemplars so a metric carries its originating `trace_id`.

### What you'll get
- a `trace_id` exemplar on every RED metric observation ← `libs/zynaxobs/metrics.go`
- exemplars surfaced on the HTTP path ← `services/api-gateway/cmd/api-gateway/main.go`

### Scope & boundaries
- In scope: exemplars on existing gRPC + HTTP RED metrics, exposed at `/metrics`.
- Out of scope (deferred): log export → O.9 #1192.

### Test plan & acceptance
| Acceptance criterion | How verified | Result |
|---|---|---|
| RED metrics on gRPC + HTTP carry exemplars | `GOWORK=off go test ./... -race` (libs/zynaxobs) | ✅ ok 1.031s |
| exemplars carry `trace_id` | unit asserts exemplar label `trace_id` present | ✅ pass |
| labels stay low-cardinality | review: labels = service/method/status only | ✅ |

**Local gates:** `make lint` ✅ (1 wrapcheck nolint on the transparent ResponseWriter passthrough) · `go test -race` ✅

### Evidence
- CI: all required checks green.
- Local output: `libs/zynaxobs` tests ok 1.031s.
- Artifacts / images: api-gateway image rebuilt.
- Post-merge digest sync → main: `a53cb12 chore(images): sync digests after main-93d2dff`.

### Risk & rollback
Low — additive instrumentation; revert this PR to remove. No change to existing metric labels.

### Review aids
Start at `libs/zynaxobs/metrics.go` (the exemplar attach point); `main.go` is just the HTTP wiring.

**SPDD:** Canvas `docs/spdd/467-observability-otel-uptrace/canvas.md` — Status: Aligned · `/spdd-security-review` PASS
**AI:** `Assisted-by: Claude/claude-opus-4-8`  (label: ai-assisted)
```

---

## Rules that make the body work

- **`Closes #N` is mandatory.** Without a closing keyword the issue stays open after merge —
  the automation must include it (a bare `#N` mention does not auto-close).
- **The test-plan matrix mirrors the issue/canvas Acceptance Criteria verbatim** — one row each —
  so a reviewer can check delivery against the contract, not against the author's paraphrase.
- **Evidence is concrete:** paste the decisive line (coverage %, `ns/op`, `ok … s`, a run link),
  not "tested locally".
- **The post-merge digest line is a placeholder the author leaves and the post-merge verifier
  fills** once the release pipeline produces the `chore(images): sync digests …` commit on `main`
  (see `/milestone-orchestrate` STEP 7.5). It closes the loop from "PR merged" to "main is
  digest-consistent".
- Keep it tight. Long is not the goal — *complete and scannable* is.
