<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — Consolidate Python CI scripts into zynax-ci (M7.Y)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #1421
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-18
**Status:** Draft

> Python counterpart to #1285 (M7.S — bash → zynax-ci, closed) and #1276 (llm-adapter
> Python → Go, ADR-035). Same destination binary, same ADR-036 pattern.

---

## R — Requirements

- **Problem:** ~270 LOC of first-party Python deterministic CI logic lives **outside** the tested
  `zynax-ci` binary: `scripts/update_image_digest.py` (image-digest upsert run on the push-to-main
  release path), `scripts/validate_milestone_state.py` (milestone-state schema gate), and
  `automation/scripts/check_expert_mapping.py` (the ADR-033 expert-mapping drift guard). This logic
  gets no unit tests, no `make lint`, and no `govulncheck` coverage, and it forces a `python3`
  runtime onto CI gates — including a digest commit that pushes to `main`.
- **Missing state:** each script reproduced as a tested `zynax-ci` subcommand, with every consuming
  workflow/Makefile step reduced to a single `zynax-ci <verb>` call — the pattern already used by
  `validate <schema|canvas|...>`, `images sync|check`, `check deps`.
- **Definition of done — observable outcomes:**
  - Each script's behaviour is reproduced by a `zynax-ci` subcommand with table-driven unit tests.
  - Output, exit code, and any file edit are **byte-for-byte parity** with the Python, verified
    before deletion.
  - Each call site is a single `zynax-ci <verb>` invocation; the `.py` is removed.
  - `cmd/zynax-ci` module coverage stays **≥ 80 %**; `make lint` + `govulncheck` green.
  - ~270 LOC of first-party Python removed; no Python left in `scripts/` root or `automation/scripts/`.
  - The Python that **stays** (SDK, langgraph, examples, pytest, e2e shell) is untouched — by design.

---

## E — Entities

### Existing entities consumed (unchanged)

- **`zynax-ci`** (`cmd/zynax-ci`) — the Cobra CLI that already hosts CI/dev verbs. This epic extends
  it with: `images digest-update`, `validate milestone`, `check expert-mapping`.
- **`images/images.yaml`** — image-reference SoT (ADR-024); the `images` group already reads/writes it.
- **`state/milestone.yaml`** + **`state/milestone.schema.json`** — the milestone SoT and its JSON Schema.
- **`automation/experts/runtime_mapping.yaml`** + authoring-expert / runtime-agent globs +
  the **ADR-033** mapping table — the three surfaces the drift guard reconciles.
- **`automation/tests/test_expert_mapping.py`** — pytest that imports the drift-guard script directly;
  reconciled in S3 (ported or repointed) so the gate keeps firing.

### New / extended entities (zynax-ci subcommands)

- **`images digest-update`** — line-based upsert of one entry's digest in `images.yaml`
  (reuses the existing `internal/images` package).
- **`validate milestone`** — JSON-Schema validation of `state/milestone.yaml`.
- **`check expert-mapping`** — reconcile the mapping against the experts/agents globs + ADR-033 table.

### Relationship

```
GitHub workflow / Makefile step  ──►  zynax-ci <verb>  ──►  (tested Go logic)
                                              │
                                              ├─ images digest-update → images/images.yaml (ADR-024)
                                              ├─ validate milestone    → state/milestone.{yaml,schema.json}
                                              └─ check expert-mapping   → runtime_mapping.yaml + ADR-033 table

Stays Python (deliberate, NOT migrated):
  agents/sdk (PyPI product, ADR-009) · agents/adapters/langgraph (library, ADR-035)
  agents/examples · automation/tests/* (pytest)
Stays bash (ADR-036):  scripts/e2e/*  (kind / kubectl / helm / docker drivers)  — covered by #1285
```

---

## A — Approach

**We will:**

- Add/extend `cmd/zynax-ci` subcommands for each of the three scripts, each with table-driven unit
  tests (`GOWORK=off go test ./...`), SPDX headers, functions ≤ 30 lines.
- Migrate one script at a time, proving **byte-for-byte parity** (output, exit code, file edit)
  against the Python before deleting it; the consuming step becomes a single `zynax-ci <verb>` call.
- Reuse the existing `internal/images` package for `images digest-update` rather than re-parsing YAML;
  use a maintained Go JSON-Schema library (e.g. `santhosh-tekuri/jsonschema`) for `validate milestone`.
- For S3, reconcile `automation/tests/test_expert_mapping.py` (which imports the Python directly) by
  porting its intent to a Go test under `cmd/zynax-ci/check`, then delete the Python.

**We will NOT:**

- Migrate the **deliberate** Python: `agents/sdk` (PyPI product, ADR-009), `agents/adapters/langgraph`
  (LangGraph library, ADR-035), `agents/examples`, or any `automation/tests/*` pytest suite.
- Port the e2e harness or any `scripts/e2e/*.sh` — orchestration over external CLIs; stays bash (ADR-036).
- Change what any gate asserts (schema, drift rules, digest format) — this is behaviour parity, not a
  policy change.
- Touch the user-facing `zynax` CLI or rewrite the Makefile wholesale.

**Governing ADRs:** ADR-036 (CI logic as a Go CLI), ADR-024 (images.yaml SoT), ADR-033 (expert/runtime
mapping), ADR-009 (Python for agents — the out-of-scope rationale), ADR-035 (adapter language boundary),
ADR-017 (`GOWORK=off`), ADR-019 (this Canvas before code).

---

## S — Structure

```
cmd/zynax-ci/
├── internal/images/
│   └── digest.go                ← images digest-update (S1, reuses Upsert/DigestRe) — DONE
├── validate/
│   └── milestone.go             ← validate milestone (S2); JSON-Schema vs state/milestone.schema.json
├── check/
│   └── expertmapping.go         ← check expert-mapping (S3); 3 drift rules + ADR-033 table parse
└── cmd/
    ├── images.go  validate.go  check.go   ← cobra wiring (one verb each) + *_test.go

Thinned (verb calls): .github/workflows/{tools-image,release,pr-checks,ci}.yml · Makefile recipes
Removed at each story's cutover: scripts/update_image_digest.py (S1),
  scripts/validate_milestone_state.py (S2), automation/scripts/check_expert_mapping.py (S3)
Unchanged: scripts/e2e/* (bash), agents/** Python, automation/tests/* (except the S3 reconcile)
```

Config: subcommands read env (`$GITHUB_OUTPUT`, refs) — 12-Factor; no new secrets.

---

## O — Operations

Each step is one reviewable PR, mapped 1:1 to a story issue:

1. **S1 — `images digest-update`** (#1422): subcommand + parity vs `update_image_digest.py`; swap
   `tools-image.yml` + `release.yml` call sites; delete the `.py`. **Implemented** on branch
   `refactor/images-digest-update-go` (the reference pattern for S2/S3).
2. **S2 — `validate milestone`** (#1423): subcommand validating `state/milestone.yaml` against its
   JSON Schema (valid + invalid fixture tests); swap `pr-checks.yml` + `Makefile` call sites; delete
   `validate_milestone_state.py`. (dep: S1 pattern)
3. **S3 — `check expert-mapping`** (#1424): subcommand enforcing the three drift rules + ADR-033 table
   parse (fixture tests); reconcile `automation/tests/test_expert_mapping.py`; swap the call site;
   delete `check_expert_mapping.py`. (dep: S1 pattern)

---

## N — Norms

Pulled from root `AGENTS.md` §Hard Constraints, `docs/engineering/best-practices/go.md`,
`docs/engineering/best-practices/github-ci.md`, `cmd/zynax/AGENTS.md`.

- Commit hygiene: subject ≤ 72 chars, imperative, no period, no emojis; `Signed-off-by:` +
  `Assisted-by: Claude/<model-id>` on every commit; never `Co-Authored-By:` for AI.
- One PR per story (S1–S3); ≤ 400 lines excluding generated code; `refactor:`/`ci:` PRs are valid here.
- SPDX header `// SPDX-License-Identifier: Apache-2.0` on every `.go` file.
- `GOWORK=off` for every `go` / `go test` command in `cmd/zynax-ci` (ADR-017).
- Go functions ≤ 30 lines; no `panic`; never discard errors; close resources via `defer`.
- Table-driven unit tests with fixtures; prove byte-for-byte parity before deleting any script.
- Image refs only via `images/images.yaml` (ADR-024); never hand-edit banner-marked regions.
- Workflow actions remain SHA-pinned, least-privilege (`permissions:`), with `concurrency` groups.
- Do not write literal email addresses in source or fixtures (gitleaks PII gate).

---

## S — Safeguards

### Context Security (complete before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no email literals; author name is the public maintainer of record
- [x] No prompt injection: no instruction-like phrasing that overrides AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] `/spdd-security-review` passed — result: PASS (2026-06-18)

### Feature Safeguards

- **Never** change what a gate asserts during migration — behaviour parity only (digest format,
  schema, drift rules are unchanged).
- **Never** delete a script before its replacement verb is green in CI (parity proven).
- **Never** migrate the deliberate Python — `agents/sdk` (ADR-009), `agents/adapters/langgraph`
  (ADR-035), `agents/examples`, `automation/tests/*` pytest — it stays Python.
- **Never** port the e2e harness in this epic — it stays bash (ADR-036).
- **Never** hand-edit `images/images.yaml` banner-marked regions — use the `images` internals (ADR-024).
- **Never** unpin a GitHub Action SHA or widen workflow `permissions:` while thinning a step.
- **Never** commit a code step before its parity is proven against the Python it replaces.
