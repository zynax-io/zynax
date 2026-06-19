# REASONS Canvas — Declarative demo-scenario manifest (Workflow + AgentDef + context injection in one file)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/lib:spdd-security-review <path>` before committing.

**Issue:** #1385
**Epic:** #1370 (awesome-quickstart / first-run UX) — Operations step 14
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-19
**Status:** Aligned (v1 — maintainer-authorized 2026-06-19, approach A2 selected)
**Aligned:** 2026-06-19 (maintainer-authorized; approach A2 — manifest-set convention)

> **Provenance — own canvas required (ADR-019).** The #1370 canvas
> ([docs/spdd/1370-awesome-quickstart/canvas.md](../1370-awesome-quickstart/canvas.md), Reframe
> Addendum) lists #1385 and #1387 as new `feat:` stories that **cross a spec/gRPC boundary** and
> therefore **require their own REASONS Canvas before any implementation (ADR-019)**. This file is
> that canvas. It is **Draft only** — the maintainer reviews and sets `Status: Aligned`; `/deliver
> #1385` must refuse to run until then.

---

## R — Requirements

> Problem statement: what breaks or is missing without this feature?

Grounded in the live 2026-06-18 validation run (recorded in the #1370 canvas R section): getting a
scenario to run end-to-end required **manual, imperative edits** across three layers that a user
must not have to touch:

- Hand-editing the **adapter config** (which provider/model/endpoint an `AgentDef` points at).
- Hand-editing a **compose overlay** to bring up the right runtime alongside the platform.
- Hand-editing the **prompt / context** baked into the workflow action.

A user who wants to run *their own* scenario — a workflow, the agent(s) that supply its
capabilities, and the context to inject — has no single declarative artefact to express it. They
must read the repo, know which files to edit, and apply manifests one at a time in the right order
with no defined dependency or readiness semantics. This contradicts the declarative control-plane
mandate (ADR-011) and the first-run UX goal of the parent epic.

> Definition of done: the observable outcomes that confirm delivery.

- **One declarative artefact** (a `Scenario`, expressed as a manifest set under a single directory
  or document) wires together: the `Workflow`, the `AgentDef`(s) that provide its capabilities, and
  a **context-injection block** — and runs to a **terminal result with no imperative edits**.
- `zynax apply <scenario>` (and `make demo SCENARIO=<name>`) brings the scenario up end-to-end:
  registers the AgentDef(s), submits the Workflow, and reports the run to a terminal state.
- The scenario schema/convention is **documented** and **validated by `zynax`** (local
  `zynax validate` + `--dry-run`) before anything is submitted.
- The story ships a **human-validation guide** per the standard
  ([docs/contributing/human-validation-guide.md](../../contributing/human-validation-guide.md), #1388).
- Out of scope is explicit and enforced: no imperative scripting, no UI scenario builder.

---

## E — Entities

> Tier 1 abstractions only. Names are public-safe.

```
Scenario ──references──> Workflow            (kind: Workflow,  apiVersion: zynax.io/v1)
Scenario ──references──> AgentDef[1..N]      (kind: AgentDef,  apiVersion: zynax.io/v1alpha1)
Scenario ──carries────> ContextInjectionBlock (the scenario's declarative context slice)
ContextInjectionBlock ──bound at dispatch──> Workflow action `input` (Jinja2 {{ .ctx.* }} templates)

zynax apply <scenario> ──orders──> [ register AgentDef(s) → submit Workflow ] ──> run_id
make demo SCENARIO=<name> ──wraps──> zynax apply <scenario>  (+ compose bring-up + result + cleanup)
```

- **Scenario** — the composite artefact. It is **not** a new third schema kind (see A,
  Alternatives): the recommended shape is a **manifest set** — the existing `kind: Workflow` and
  `kind: AgentDef` manifests grouped under one directory plus a small `scenario.yaml` index that
  lists the member manifests and declares apply order + the context block. This honours ADR-028's
  "two kinds, no third schema" precedent.
- **ContextInjectionBlock** — declarative key/values injected into the workflow's action `input`
  templates. This is the **declarative analogue** of the manual prompt edit the live run needed. The
  *substantive* context-injection contract (bounded `{files[], max_tokens}`, strict isolation per
  ADR-028) is the subject of the **sibling canvas #1387**; #1385 provides only the **wiring slot**
  for it and defers the slice semantics there to avoid double-owning the contract.
- **Apply ordering** — AgentDef(s) must be registered (so the task broker can route the capability)
  **before** the Workflow is submitted; otherwise an unbacked capability fails fast (the #1370
  NotFound non-retryable fix, #1381). The scenario index makes this order declarative, not a thing
  the user sequences by hand.

Real contracts this composes (cited, not redefined):
- `spec/schemas/workflow.schema.json` — `kind: Workflow`; `spec.initial_state` + `spec.states[]`
  with `actions[].capability` + `actions[].input` (Jinja2 `{{ .ctx.* }}` / `{{ .trigger.* }}`).
- `spec/schemas/agent-def.schema.json` — `kind: AgentDef`; `spec.capabilities[]`
  (`input_schema`/`output_schema`) + `spec.runtime` (image, env, replicas).
- `spec/schemas/policy.schema.json` — the third `apply`-able kind today (Workflow, AgentDef, Policy).
- `spec/workflows/examples/code-review-ollama.yaml` + `agent-def-example.yaml` — the exact pair a
  first scenario composes.

---

## A — Approach

> What we WILL do AND what we WON'T do. ADRs that govern; one-way doors get an ADR citation.

### Alternatives considered

| Option | Tradeoff | Verdict |
|--------|----------|---------|
| **A1 — New `kind: Scenario` composite schema** (one file embeds/links a Workflow + AgentDefs + context). | Single tidy artefact, but it is a **third manifest schema** with its own validation, its own `apply` semantics, and a **new api-gateway response shape** (today `apply` returns *either* `run_id` (202, Workflow) *or* `agent_id` (201, AgentDef) — a Scenario produces *both*). That is a new spec **and** gRPC/REST boundary, a one-way door, and a direct repeat of the option ADR-028 §Option C **rejected** ("invent a third manifest schema"). Needs a `.feature` at the new boundary + likely an ADR. | **REJECTED** — one-way door, new gRPC boundary + dual result, needs own ADR + `.feature`; highest blast radius, contradicts ADR-028. |
| **A2 — Manifest-set convention** (existing `kind: Workflow` + `kind: AgentDef` files grouped under `spec/scenarios/<name>/`, plus a tiny `scenario.yaml` index declaring members + apply order + the context slot). `zynax apply <dir>` and `make demo` iterate the set client-side in dependency order. | Zero new server-side kind; reuses the two shipped, CI-enforced schemas and the existing `/api/v1/apply` per-manifest path (no new gateway response shape). The only **new** spec surface is a small, separately-validated `scenario.schema.json` for the *index* file, and a CLI ordering loop. Aligns with ADR-028 ("two kinds, no third schema") and keeps the user manifests identical to any hand-applied manifest — which is the point of declarative self-service. | **SELECTED** (maintainer-authorized 2026-06-19) — recommended; zero new boundary, composes only accepted ADRs. |
| **A3 — Do nothing** (keep documenting the manual edit sequence). | Violates ADR-011 (declarative control plane) and the parent-epic first-run UX goal; leaves the live-validation pain in place. | Rejected. |

### We will

- Add a **scenario manifest-set convention** (A2): `spec/scenarios/<name>/` containing the member
  `Workflow` + `AgentDef` manifests (unchanged shapes) and a `scenario.yaml` index file.
- Define a **small `scenario.schema.json`** for the index only: `kind: Scenario`,
  `apiVersion: zynax.io/v1alpha1`, `metadata.name/namespace`, `spec.members[]` (each a
  `{kind, file}` reference), `spec.apply_order[]`, and a `spec.context` slot that **points at** the
  context-injection block whose semantics live in #1387. The index is validated **locally by
  `zynax`**, mirroring the existing `kindToSchemaFile` map (`cmd/zynax/validate/manifest.go`).
- Teach `zynax apply` (and `zynax validate`/`--dry-run`) to accept a **scenario directory or index
  file**, expand it to its member manifests, validate each against its existing schema, and submit
  them over the existing `/api/v1/apply` REST path in `apply_order` — **no new gateway endpoint, no
  new response shape**. Each member is applied exactly as a hand-applied manifest is today.
- Wire `make demo SCENARIO=<name>` to: compose bring-up → `zynax apply` the scenario → poll to a
  terminal result → print the capability output → cleanup (reusing the #1360 `make demo` skeleton).
- Ship one **reference scenario** composing `code-review-ollama.yaml` + an Ollama-backed AgentDef,
  plus a **human-validation guide** (#1388 template).

### We will NOT

- **Won't** invent a `kind: Scenario` *server-side* composite that the api-gateway compiles/runs
  (A1) — the index is a **client-side** orchestration artefact; the platform only ever sees the two
  existing kinds. (This is the explicit boundary that keeps us out of the gRPC contract.)
- **Won't** define the context-slice *semantics* here — bounded `{files[], max_tokens}` + strict
  isolation belong to **#1387** (ADR-028 contract). #1385 owns only the wiring slot.
- **Won't** accept provider/model/endpoint/URL from runtime input — declarative config in the
  `AgentDef`/overlay only (ADR-013).
- **Won't** add gRPC or proto-generated types to the CLI (cmd/zynax AGENTS.md — HTTP REST only).
- **Won't** add imperative scripting hooks or a UI builder (→ M-UX).

**Positioning fit:** this advances the engine-portability / declarative-control-plane wedge — a user
declares *what* scenario to run, not *how* to wire it. CLI help and the scenario docs lead with
"declare your scenario in one place, `zynax apply` brings it up" rather than generic control-plane
framing. See [docs/product/positioning.md](../../product/positioning.md).

**Governing ADRs:** ADR-011 (declarative YAML control plane), ADR-028 (AgentDef-vs-Workflow split +
context-slice injection contract — "two kinds, no third schema"), ADR-029 (workflow data-flow), and
ADR-013 (config-only provider/model/endpoint). **An ADR may be required** if the maintainer prefers
A1 (new kind) — that is a one-way door; A2 (recommended) needs no new ADR because it composes only
existing accepted decisions.

---

## S — Structure (first S)

> Files created or modified, with a one-line purpose. Recommended approach (A2).

```
spec/schemas/scenario.schema.json          ← NEW: JSON Schema for the scenario INDEX (kind: Scenario, members[], apply_order[], context slot)
spec/scenarios/code-review/scenario.yaml   ← NEW: reference scenario index (members + order + context pointer)
spec/scenarios/code-review/workflow.yaml   ← reference Workflow member (reuses code-review-ollama shape)
spec/scenarios/code-review/agent.yaml      ← reference AgentDef member (Ollama-backed capability provider)
spec/tests/features/scenario_schema.feature ← BDD: index-schema validation contract (committed before impl, ADR-016)
cmd/zynax/validate/manifest.go             ← add "Scenario" → scenario.schema.json to kindToSchemaFile
cmd/zynax/cmd/apply.go                      ← accept a scenario dir/index; expand → validate → apply members in order (REST only)
cmd/zynax/validate/scenario.go             ← NEW: expand a scenario index into ordered member manifest paths
docs/scenarios/scenario-manifest.md        ← NEW: schema reference + authoring guide
docs/scenarios/code-review.validation.md    ← NEW: human-validation guide (per #1388 template)
Makefile                                    ← `make demo SCENARIO=<name>` wires compose → apply → result → cleanup (#1360 skeleton)
```

Config env prefix: none new (CLI is HTTP-REST only). Port: none new.
gRPC contracts: **none added** under A2 — the recommended approach introduces **no proto change and
no new api-gateway endpoint**. Any deviation toward A1 (a server-side `kind: Scenario`) would add a
gRPC/REST boundary and is gated by `buf breaking` (ADR-016) + a `.feature` at that boundary + an ADR.

---

## O — Operations

> Ordered, independently releasable steps (INVEST, ≤400 lines each). One step = one PR. Maps to
> candidate child issues (see Story-decomposition / Notes).

1. **O1 — `feat(spec): scenario index schema + BDD`** — add `spec/schemas/scenario.schema.json`
   (index only: `kind: Scenario`, `members[]`, `apply_order[]`, `context` slot) and the
   `scenario_schema.feature` contract; wire it into `make validate-spec`. Verified by:
   `make validate-spec` passes on a valid index and rejects a missing/cyclic `apply_order`. **(spec
   boundary — `.feature` committed before impl, ADR-016.)** Size: S.
2. **O2 — `feat(cli): validate scenario index locally`** — add `Scenario → scenario.schema.json` to
   `kindToSchemaFile`; add `validate/scenario.go` to expand an index into ordered member paths and
   validate each member against its existing schema. Verified by: `zynax validate <scenario>` and
   `--dry-run` report per-member errors; unit tests on expansion + ordering. Size: S.
3. **O3 — `feat(cli): apply a scenario end-to-end`** — `zynax apply <dir|index>` expands and submits
   members over the existing `/api/v1/apply` REST path in `apply_order` (AgentDefs before Workflow);
   prints each `agent_id`/`run_id`. **No new gateway endpoint.** Verified by: applying the reference
   scenario registers the AgentDef then submits the Workflow and returns a `run_id`; failure on an
   out-of-order/missing member is bounded and explained. Size: M.
4. **O4 — `feat(spec): reference code-review scenario`** — ship `spec/scenarios/code-review/` (index
   + Workflow + Ollama AgentDef) composing `code-review-ollama.yaml`. Verified by: `make validate-spec`
   passes; `zynax apply` runs it to a terminal result with the model output visible (depends on the
   #1370 cluster fixes #1371–#1381 already merged). Size: M.
5. **O5 — `feat(make): make demo SCENARIO=<name>`** — extend the #1360 `make demo` skeleton to take a
   `SCENARIO` and run compose-up → apply → poll-to-terminal → print-output → cleanup. Verified by:
   `make demo SCENARIO=code-review` reaches a terminal result from a clean checkout with no secrets.
   Size: S.
6. **O6 — `docs: scenario authoring + human-validation guide`** — `docs/scenarios/scenario-manifest.md`
   (schema reference + authoring guide) and `docs/scenarios/code-review.validation.md` (per the #1388
   template). Verified by: a fresh reader reaches a pass/fail verdict by following the guide; doc
   commands match the real CLI surface. Size: S. **(docs — SPDD-exempt.)**

> The `context` slot defined in O1 is a **pointer/placeholder**; its bounded-slice semantics land in
> sibling **#1387** and must not be implemented here.

---

## N — Norms

> Cross-cutting standards (root + layer AGENTS.md, docs/patterns).

- **Commit hygiene:** every commit carries `Signed-off-by:` (DCO) + `Assisted-by: Claude/<model>` —
  never `Co-Authored-By` for AI.
- **Conventional commits / PR titles:** one of feat/fix/refactor/docs/test/ci/chore; scope maps to
  directory; one logical change per commit, one PR per issue.
- **BDD before impl:** the index-schema `.feature` is committed before O1 implementation (ADR-016);
  the per-step `feat:` PRs carry it as the gate.
- **Spec authoring rules** (spec/AGENTS.md): every manifest has `kind` + `apiVersion` +
  `metadata.name`/`namespace`; capabilities are `snake_case`; **never reference agent names** in a
  Workflow — only capability names (routing is the platform's job); validate locally via
  `make validate-spec` before applying.
- **Go services/CLI:** `GOWORK=off` for all `go build`/`go test` in `cmd/zynax/` (ADR-017);
  `CGO_ENABLED=0`, `-trimpath`; the CLI stays HTTP-REST only — no gRPC/proto types (cmd/zynax
  AGENTS.md); exit codes 0/1/2 (2 = still running).
- **Human-validation guide:** required for every user-visible story (audience: zynax-user) per
  [docs/contributing/human-validation-guide.md](../../contributing/human-validation-guide.md).
- **PR size:** ≤ 200 ideal, > 900 blocked.

---

## S — Safeguards (second S)

> Things that MUST NEVER happen in this feature.

### Context Security (mandatory before committing this Canvas)
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics.
- [x] No PII: no personal names in sensitive context, no email addresses.
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules.
- [x] All entities in E are public-safe abstractions.
- [ ] `/lib:spdd-security-review` run by maintainer at alignment — **Draft**; see Security review
      below (self-assessed PASS, maintainer confirms at alignment).

### Tier-2 security review (in-canvas pass — see §Security review below)

- **Does this cross a gRPC/spec boundary?** Under the **recommended A2**, the only *new spec* surface
  is the small client-side `scenario.schema.json` index — validated locally; **no new gRPC/REST
  endpoint and no new api-gateway response shape** are introduced. Members are applied over the
  *existing* `/api/v1/apply` path exactly as hand-applied manifests are. Under the rejected A1, a
  `kind: Scenario` would cross the api-gateway boundary (a response shape returning *both* `run_id`
  and `agent_id`) and would require a `.feature` at that boundary + an ADR. The boundary risk is the
  decisive reason A2 is recommended.
- **Injection surface of the context block:** the context-injection block feeds the Workflow
  action `input` (Jinja2 `{{ .ctx.* }}` templates). The hard rule: **the context block may carry
  data only — never provider/model/endpoint/URL or any field that redirects where a capability runs**
  (ADR-013). Provider/model/endpoint stay in the `AgentDef`/overlay. Template evaluation is the
  compiler's existing, already-bounded Jinja2 surface — #1385 adds no new evaluator, only a
  declarative place to put values the user already controls. The *bounded* `{files[], max_tokens}` +
  strict-isolation guarantees are #1387's contract; #1385 must not widen them.

### Feature Safeguards
- **Never** invent a server-side third manifest schema — the scenario index is a **client-side**
  composition artefact; the platform sees only `kind: Workflow` / `kind: AgentDef` (ADR-028).
- **Never** add a new api-gateway endpoint or response shape for scenarios under A2 — reuse
  `/api/v1/apply` per member (cmd/zynax AGENTS.md, ADR-001).
- **Never** accept provider/model/endpoint/URL from the context-injection block or any runtime input
  — declarative `AgentDef`/overlay config only (ADR-013).
- **Never** reference agent names in a Workflow member — only capability names (spec/AGENTS.md).
- **Never** submit the Workflow before its AgentDef(s) are registered — apply in declared
  `apply_order` so unbacked capabilities fail fast, not retry forever (#1381).
- **Never** put gRPC or proto-generated types in the CLI (ADR-001 / cmd/zynax AGENTS.md).
- **Never** define or widen the bounded context-slice semantics here — that contract is owned by
  #1387 (ADR-028).
- **Never** require a paid API key or any secret on the default scenario path (parent-epic #1370
  safeguard).

---

## Security review (Tier-2 pass — in-canvas)

> Self-assessed at Draft authoring; the maintainer re-runs `/lib:spdd-security-review` at alignment.

**Result: PASS (with one design-gating CONCERN resolved by approach choice).**

- **Tier 1 cleanliness:** no internal hostnames/IPs, no credentials/tokens, no PII, no injection-style
  authority phrasing. All E entities are public-safe abstractions. → PASS.
- **gRPC/spec boundary (CONCERN → resolved):** a naive `kind: Scenario` (A1) *would* cross the
  api-gateway gRPC/REST boundary with a new dual-result response shape — a real new attack/contract
  surface. The **recommended A2 eliminates this**: scenarios are a client-side manifest set applied
  over the existing per-manifest `/api/v1/apply` path, so **no new boundary is introduced**. The
  concern is honestly real for A1 and is the reason A2 is recommended; if the maintainer chooses A1,
  this canvas's O-steps and Safeguards require a `.feature` at the new boundary + an ADR before impl.
- **Context-injection injection surface:** the context block writes into the existing Jinja2 action
  `input` evaluator only — no new template engine, no new eval path. The load-bearing mitigation is
  the **data-only** rule (ADR-013): the block must never carry provider/model/endpoint/URL or any
  routing-redirecting field. This is captured as a hard Safeguard. The *bounded/isolated* slice
  guarantees are deferred to #1387 (not hand-waved — explicitly out of scope here and owned there).
  → PASS, with the data-only rule as the enforced mitigation.

No unmitigated attack surface remains under the recommended approach.

---

## Notes / Story-decomposition

> Maps O-steps to candidate child issues for the maintainer to file at alignment. None created yet —
> this is a **Draft** canvas; `/lib:spdd-story #1385` runs only after `Status: Aligned`.

| O-step | Candidate child story (file at alignment) | Type | Size | Boundary / gate |
|--------|-------------------------------------------|------|------|-----------------|
| O1 | `feat(spec): scenario index schema + validation BDD (#1385, step 1)` | feature | S | spec boundary — `.feature` first |
| O2 | `feat(cli): validate scenario index + member expansion (#1385, step 2)` | feature | S | CLI-local, no boundary |
| O3 | `feat(cli): apply a scenario end-to-end over REST (#1385, step 3)` | feature | M | reuses existing `/api/v1/apply` |
| O4 | `feat(spec): reference code-review scenario manifest set (#1385, step 4)` | feature | M | depends on #1371–#1381 |
| O5 | `feat(make): make demo SCENARIO=<name> (#1385, step 5)` | feature | S | wraps #1360 skeleton |
| O6 | `docs: scenario authoring + human-validation guide (#1385, step 6)` | docs | S | SPDD-exempt |

**Dependencies & cross-links:**
- **Sibling #1387** (declarative context-injection) owns the bounded `{files[], max_tokens}` +
  strict-isolation slice contract (ADR-028). #1385 provides only the wiring slot; the two canvases
  must stay consistent — the `spec.context` slot here is the seam #1387 fills.
- **#1360** (`make demo` entry point) provides the skeleton O5 extends.
- **#1370** parent-epic fixes (#1371–#1381) must be merged for O4's reference scenario to run green.
- **#1388** human-validation-guide standard (on main) is the template O6 follows.

**Resolved (alignment decision, maintainer-authorized 2026-06-19):** **A2 (manifest-set convention)
is SELECTED**; **A1 (new `kind: Scenario`) is REJECTED** — A1 is a one-way door that crosses the
api-gateway contract and needs its own ADR + `.feature`; A2 needs neither. The rest of the canvas is
written against A2.
