# REASONS Canvas — Declarative context-injection for demo scenarios

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/lib:spdd-security-review <path>` before committing.

**Issue:** #1387
**Epic:** #1370 (awesome-quickstart / first-run UX) — Operations step 16
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-19
**Status:** Aligned (v1 — maintainer-authorized 2026-06-19, approach A2 selected)
**Aligned:** 2026-06-19 (maintainer-authorized; approach A2 — context: block on the existing template surface)

> **Provenance — own canvas required (ADR-019).** The #1370 canvas
> ([docs/spdd/1370-awesome-quickstart/canvas.md](../1370-awesome-quickstart/canvas.md), Reframe
> Addendum) lists #1385 and #1387 as new `feat:` stories that **cross a spec/gRPC boundary** and
> therefore **require their own REASONS Canvas before any implementation (ADR-019)**. This file is
> that canvas. It is **Draft only** — the maintainer reviews and sets `Status: Aligned`; `/deliver
> #1387` must refuse to run until then.

> **Pairing with #1385.** The sibling scenario-manifest canvas
> ([docs/spdd/1385-scenario-manifest/canvas.md](../1385-scenario-manifest/canvas.md), Draft on PR
> #1447) defines a client-side **manifest-set convention** whose scenario index carries a
> `spec.context` **slot** and **explicitly defers the slice semantics to this issue (#1387)**. #1385
> owns the *wiring slot*; **#1387 owns the *context-injection contract* that fills it.** The two
> canvases share one seam and must stay consistent.

---

## R — Requirements

> Problem statement: what breaks or is missing without this feature?

Grounded in the live 2026-06-18 validation run (recorded in the #1370 canvas R section) and in the
shipped reference example `spec/workflows/examples/code-review-ollama.yaml`. Today the only way to
ground a demo scenario in real content is to **hand-paste the data into the workflow action**. In
the shipped example, the git diff under review is embedded as a **literal multiline string** inside
`actions[].input.prompt`:

```yaml
actions:
  - capability: codereview
    input:
      prompt: |
        You are a senior Go code reviewer. Review the following git diff ...
        ```diff
        <the entire git diff pasted by hand here>
        ```
```

This means a user who wants to review *their own* diff (or feed *their own* document/data into any
demo scenario) must:

- Open the workflow YAML and **edit the prompt body by hand**, re-pasting content on every run.
- Manage the size of the pasted blob themselves — there is **no bound** on how much context gets
  inlined, and an over-large paste silently inflates the dispatched `input_payload`.
- Mix **instructions** (the prompt) and **data** (the diff) in one opaque string, with no declared
  provenance, no truncation policy, and no isolation guarantee between scenarios.

There is no **declarative** way to say "inject *this* content into *this* action's input." The
workflow data-flow keystone (EPIC W / ADR-029) solves a different problem — passing one **state's
output** to a **later state's input** *within a run* (`$.states.<state>.output.<key>`); it does
**not** provide a way to inject **external/initial content** (a diff, a file, a prompt fragment) at
the scenario boundary. The context-propagation work (EPIC C / ADR-031) standardises *how* trace,
data, and correlation move across boundaries and what an agent receives on dispatch, but it does
**not** define the *declarative authoring shape* for the initial content a scenario supplies. That
shape is the gap #1387 fills, and the slot #1385 left open.

> Definition of done: the observable outcomes that confirm delivery.

- A scenario declares its injected context **declaratively** in a bounded
  **context-injection block** — `{files[], max_tokens}` per the ADR-028 context-slice contract —
  instead of an inlined literal blob. The block names file sources, not pasted bodies.
- At dispatch, the block resolves into the workflow action `input` through the **existing** template
  surface (`{{ .ctx.* }}` / `{{ .trigger.* }}`) — **no new template engine, no new evaluator**.
- The injected context is **bounded** (hard-capped at the declared `max_tokens`, documented overflow
  policy) and **strictly isolated** (one scenario's context never leaks into another), honouring the
  ADR-028 contract verbatim.
- The reference `code-review` scenario injects a **real git diff** via the block — the live-run pain
  point — with **no hand-edit of the workflow prompt**.
- The block is **data-only**: it can never carry provider/model/endpoint/URL or any field that
  redirects where or how a capability runs (ADR-013/ADR-035). Those stay in `AgentDef`/overlay
  config and are never accepted from `input_payload`.
- The schema/convention is **documented** and **validated locally by `zynax`** before submission;
  the story ships a **human-validation guide** (#1388 template).

---

## E — Entities

> Tier 1 abstractions only. Names are public-safe.

```
ContextInjectionBlock ──declares──> ContextSource[1..N]   ({ key, files[] } — file-rooted, not pasted bodies)
ContextInjectionBlock ──bounds────> ContextBudget         ({ max_tokens, overflow: truncate-oldest })
ContextInjectionBlock ──isolated by──> scenario scope     (one scenario's context never reaches another)

ContextInjectionBlock ──resolved at dispatch──> WorkflowAction.input  ({{ .ctx.<key> }} template, EXISTING surface)
WorkflowAction.input  ──compiled──> ActionIR.input_template_json      (workflow_compiler.proto — EXISTING)
ActionIR ──dispatched──> ExecuteCapabilityRequest.input_payload       (task_broker.proto — EXISTING, data-only)

Scenario(#1385) ──carries `spec.context` slot──> ContextInjectionBlock   (the seam: #1385 wires, #1387 fills)
```

- **ContextInjectionBlock** — the declarative artefact #1387 introduces: a bounded, isolated
  description of the content a scenario injects into a workflow action's `input`. It is the
  *declarative analogue* of the hand-pasted diff in today's reference example. Its **shape and
  semantics** (`{files[], max_tokens}`, strict isolation, truncate-oldest overflow) are the ADR-028
  context-slice contract, applied at the scenario boundary rather than only per-expert.
- **ContextSource** — one named entry in the block: a context **key** plus the **file(s)** that
  source its value. The key is what a workflow action references via `{{ .ctx.<key> }}`. Sources are
  **file-rooted, not inline bodies** — that is what makes injection declarative and re-runnable.
- **ContextBudget** — the bound: `max_tokens` (hard cap) + an explicit overflow policy
  (truncate-oldest files, per ADR-028). Without this, injection is an unbounded `input_payload`
  inflation surface.

**Real contracts this composes (cited, NOT redefined):**

- `spec/schemas/workflow.schema.json` — `actions[].input` is a "Key-value input template passed to
  the agent. Values may reference workflow context with `{{ .ctx.key }}` or trigger data
  `{{ .trigger.field }}`." **#1387 reuses this exact surface** — the block resolves into existing
  `{{ .ctx.* }}` references; it adds no new template syntax.
- `protos/zynax/v1/workflow_compiler.proto` — `ActionIR.input_template_json` (the compiled,
  context-referencing input template). The injected values flow through here unchanged.
- `protos/zynax/v1/task_broker.proto` — `ExecuteCapabilityRequest.input_payload` (the JSON-encoded
  input dispatched to a capability). The block feeds **data** here; it may **never** feed
  routing/provider fields (ADR-013).
- `spec/schemas/agent-def.schema.json` — `input_schema` (validates `input_payload` structure). The
  injected context must satisfy the target capability's declared `input_schema`.
- ADR-028 — the `{files[], max_tokens}` + strict-isolation **context-slice injection contract**
  (recorded there as "the load-bearing half" of the AgentDef-vs-Workflow decision).
- ADR-031 §4 — the agent-handoff contract: "exactly what context an agent receives on dispatch." The
  injected block is content delivered **through** that handoff, not a new handoff.

---

## A — Approach

> What we WILL do AND what we WON'T do. ADRs that govern; one-way doors get an ADR citation.

### Alternatives considered

| Option | Tradeoff | Verdict |
|--------|----------|---------|
| **A1 — Extend the existing data-flow `input_bindings`** (ADR-029) to also accept an *external/initial* source (file/literal) alongside the `$.states.<state>.output.<key>` JSON-path references. | Reuses one binding surface; *but* ADR-029 deliberately scopes `input_bindings` to **state→state, run-scoped** references with **no expression/transform/templating** and "no typed schema for context values in M7". Adding an external file source widens an **accepted, additive-only proto contract** beyond its stated non-goals, conflates initial-context injection with intermediate data-flow, and would need a `buf breaking`-gated proto edit + ADR-029 amendment. It also has **no place** for the bounded `{files[], max_tokens}` / isolation semantics ADR-028 already owns. | **REJECTED** — reopens an accepted additive-only proto contract; broader blast radius; re-opens a fixed one-way-door contract for the wrong concern. |
| **A2 — New top-level `context:` block resolved through the EXISTING `{{ .ctx.* }}` template surface, with ADR-028 `{files[], max_tokens}` + strict-isolation semantics.** The block declares file-rooted sources + a token budget; the compiler binds them into the existing action `input` template (`ActionIR.input_template_json`) at dispatch. No new template engine, no new proto field on the dispatch path. | The block is a **declarative spec surface** (a small schema + the `spec.context` slot #1385 already reserves) that fills the existing `{{ .ctx.<key> }}` references workflow actions can already write. It applies ADR-028's contract at the scenario boundary, keeps initial-context injection cleanly separate from ADR-029 data-flow, and is **data-only** by construction (ADR-013). The only **new** surface is the context-block schema + the binding logic that reads files and caps tokens before resolution. | **SELECTED** (maintainer-authorized 2026-06-19) — recommended; data-only per ADR-013/035, no proto-contract reopening. |
| **A3 — Do nothing** (keep hand-pasting the diff/prompt into `actions[].input`). | Re-creates the live-run pain on every run; unbounded `input_payload`; mixes data and instructions; no isolation. Violates ADR-011 (declarative control plane) and the parent-epic first-run UX goal. | Rejected. |

### We will

- Add a **declarative context-injection block** (A2) — the contract that fills the `spec.context`
  slot #1385 reserves. Its shape and semantics are the **ADR-028 context-slice contract**: bounded
  `{files[], max_tokens}`, strict isolation (one scenario's context never reaches another), and a
  documented overflow policy (truncate-oldest files).
- Resolve the block into a workflow action's `input` through the **existing** `{{ .ctx.<key> }}`
  template surface (`spec/schemas/workflow.schema.json` → `ActionIR.input_template_json`). The
  block reads its declared files, applies the token bound, and binds the result to the named context
  keys **at dispatch** — adding **no new template engine and no new evaluator**.
- Define a **small JSON Schema** for the block (`context` source list + `max_tokens` budget), wired
  into `make validate-spec` and validated **locally by `zynax`** before submission.
- Convert the reference `code-review` scenario to **inject a real git diff via the block** instead of
  the hand-pasted multiline literal — the exact live-run pain point.
- Keep the block **data-only** by construction: the schema admits content sources only — never
  `provider`, `model`, `endpoint`, `url`, or any routing-redirecting field (ADR-013/ADR-035).
- Ship a **human-validation guide** per the #1388 template.

### We will NOT

- **Won't** widen ADR-029 `input_bindings` to carry external/initial context (A1) — that contract is
  fixed, additive-only, and scoped to **state→state, run-scoped** data-flow. #1387 is initial-context
  injection, a different concern.
- **Won't** re-spec what EPIC W (ADR-029, data-flow `output→input`) or EPIC C (ADR-031, context
  propagation / dispatch handoff) already deliver — #1387 **uses** the existing `{{ .ctx.* }}`
  template surface and the existing dispatch path; it adds the **declarative authoring shape** for
  initial content only.
- **Won't** define the *scenario manifest-set* convention or the `spec.context` slot itself — that is
  #1385's job. #1387 fills the slot; it does not own the index.
- **Won't** introduce a new template/expression language — no CEL/JSONata/transforms (ADR-029
  non-goal stands).
- **Won't** accept provider/model/endpoint/URL or any field that redirects a capability's execution
  from the context block or any `input_payload` — declarative `AgentDef`/overlay config only
  (ADR-013, http-adapter rule; ADR-035).
- **Won't** add gRPC or proto-generated types to the CLI (cmd/zynax AGENTS.md — HTTP REST only); the
  recommended approach adds **no new proto field on the dispatch path**.

**Positioning fit:** this advances the declarative-control-plane wedge — a user declares *what*
content grounds a scenario, not *how* to paste it into a prompt. CLI help and the context-injection
docs lead with "declare your context once; `zynax` injects it, bounded and isolated" rather than
generic control-plane framing. See [docs/product/positioning.md](../../product/positioning.md).

**Governing ADRs:** ADR-011 (declarative YAML control plane), ADR-028 (AgentDef-vs-Workflow split +
**context-slice injection contract** — the `{files[], max_tokens}` + strict-isolation semantics this
block applies), ADR-029 (workflow data-flow — the **boundary** #1387 must not re-open), ADR-031
(context propagation / dispatch handoff — the carrier #1387 rides), ADR-013 + ADR-035 (config-only
provider/model/endpoint; never `input_payload`). **A new ADR may be required** only if the maintainer
prefers A1 (amend ADR-029) — that re-opens a one-way door; A2 (recommended) composes only existing
accepted decisions and needs no new ADR.

---

## S — Structure (first S)

> Files created or modified, with a one-line purpose. Recommended approach (A2).

```
spec/schemas/context-injection.schema.json   ← NEW: JSON Schema for the context-injection block (sources[], max_tokens, overflow) — data-only, no routing fields
spec/tests/features/context_injection.feature ← BDD: block-schema + binding/bounding contract (committed before impl, ADR-016)
spec/workflows/examples/code-review-ollama.yaml ← MODIFY: replace the hand-pasted diff literal with a `{{ .ctx.diff }}` reference fed by the block
spec/scenarios/code-review/scenario.yaml      ← MODIFY (from #1385): fill the reserved `spec.context` slot with a real ContextInjectionBlock
services/workflow-compiler/...                ← bind context sources → action `input` template at compile/dispatch; enforce the token bound + isolation; reject routing fields
cmd/zynax/validate/manifest.go                ← add the context block to local validation (kind/schema mapping, reuse existing kindToSchemaFile pattern)
docs/scenarios/context-injection.md           ← NEW: schema reference + authoring guide (declarative context, bounds, isolation, data-only rule)
docs/scenarios/code-review.context.validation.md ← NEW: human-validation guide (per #1388 template)
```

Config env prefix: none new (CLI is HTTP-REST only). Port: none new.
gRPC contracts: **no new proto field on the dispatch path** under A2 — the block resolves through the
existing `{{ .ctx.* }}` template surface into `ActionIR.input_template_json` /
`ExecuteCapabilityRequest.input_payload`. Any boundary change (e.g. A1's ADR-029 amendment, or a new
dispatch field) is gated by `buf breaking` (ADR-012/016) **and** a `.feature` at that boundary **and**
an ADR. The token-bound + isolation enforcement lives in the compiler/engine, not in a new contract.

---

## O — Operations

> Ordered, independently releasable steps (INVEST, ≤400 lines each). One step = one PR. Maps to
> candidate child issues (see Notes / Story-decomposition).

1. **O1 — `feat(spec): context-injection block schema + BDD`** — add
   `spec/schemas/context-injection.schema.json` (sources `{key, files[]}` + `max_tokens` budget +
   overflow policy; **data-only — no `provider`/`model`/`endpoint`/`url` properties admitted**) and
   the `context_injection.feature` contract; wire it into `make validate-spec`. Verified by:
   `make validate-spec` passes a valid block, rejects a missing `max_tokens`, and rejects any
   routing/provider field. **(spec boundary — `.feature` committed before impl, ADR-016.)** Size: S.
2. **O2 — `feat(cli): validate the context-injection block locally`** — teach `zynax validate` /
   `--dry-run` to validate a scenario's context block against the schema and to fail fast on an
   unresolved `{{ .ctx.<key> }}` reference or a missing source file. Verified by: unit tests on
   key↔reference resolution and on the data-only rejection path. Size: S.
3. **O3 — `feat(workflow-compiler): bind + bound context into the action input`** — at
   compile/dispatch, read each source's files, apply the `max_tokens` hard cap with the
   truncate-oldest overflow policy, and bind the result into the action `input` template
   (`{{ .ctx.<key> }}` → `ActionIR.input_template_json`); enforce **strict scenario isolation** (one
   block never reaches another scenario). Verified by: a block over-budget is truncated deterministically;
   an isolation test proves cross-scenario non-leakage; a routing field in the block is rejected at
   compile time. **(spec/compiler boundary — covered by the O1 `.feature`.)** Size: M.
4. **O4 — `feat(spec): code-review scenario injects a real git diff`** — modify
   `code-review-ollama.yaml` (and the #1385 scenario index's `spec.context` slot) to inject a real
   diff through the block, replacing the hand-pasted literal. Verified by: `zynax apply` runs the
   scenario to a terminal result with the model reviewing the injected diff, **no prompt hand-edit**
   (depends on #1385 O1–O3 + the #1370 cluster fixes already merged). Size: M.
5. **O5 — `docs: context-injection authoring + human-validation guide`** —
   `docs/scenarios/context-injection.md` (schema reference, bounds, isolation, the data-only rule) and
   `docs/scenarios/code-review.context.validation.md` (per the #1388 template). Verified by: a fresh
   reader injects their own diff by following the guide; doc commands match the real CLI surface.
   Size: S. **(docs — SPDD-exempt.)**

> #1387 fills the `spec.context` slot #1385 reserves. O4 must land **after** #1385 O1–O3 (the slot +
> apply path exist) — sequence at alignment so the two clusters interleave cleanly.

---

## N — Norms

> Cross-cutting standards (root + layer AGENTS.md, docs/patterns).

- **Commit hygiene:** every commit carries `Signed-off-by:` (DCO) + `Assisted-by: Claude/<model>` —
  never `Co-Authored-By` for AI.
- **Conventional commits / PR titles:** one of feat/fix/refactor/docs/test/ci/chore; scope maps to
  directory; one logical change per commit, one PR per issue.
- **BDD before impl:** the context-block `.feature` is committed before O1/O3 implementation
  (ADR-016); the per-step `feat:` PRs carry it as the gate.
- **Spec authoring rules** (spec/AGENTS.md): every manifest has `kind` + `apiVersion` +
  `metadata.name`/`namespace`; capabilities are `snake_case`; **never reference agent names** in a
  Workflow — only capability names (routing is the platform's job); input templates use the
  `{{ .ctx.key }}` / `{{ .trigger.field }}` / `{{ .event.field }}` syntax evaluated by the compiler;
  validate locally via `make validate-spec` before applying.
- **Go services/compiler/CLI:** `GOWORK=off` for all `go build`/`go test` (ADR-017);
  `CGO_ENABLED=0`, `-trimpath`; domain unit coverage ≥ 90% on `internal/domain/`; the CLI stays
  HTTP-REST only — no gRPC/proto types (cmd/zynax AGENTS.md).
- **Proto changes (if any) are additive only** and pass `buf breaking` (ADR-012/016); the
  recommended approach adds none on the dispatch path.
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
- [ ] `/lib:spdd-security-review` run by maintainer at alignment — **Draft**; see §Security review
      below (self-assessed PASS with a load-bearing CONCERN, maintainer confirms at alignment).

### Tier-2 security review (in-canvas pass — see §Security review below)

- **Does this cross a gRPC/spec boundary?** Yes — a **new spec surface** (the context-block schema)
  and the compiler binding logic. Under the **recommended A2** there is **no new proto field on the
  dispatch path** (the block resolves through the existing `{{ .ctx.* }}` template surface into the
  existing `ActionIR.input_template_json` / `ExecuteCapabilityRequest.input_payload`). The spec
  boundary is gated by a `.feature` (O1, ADR-016); A1 (amending ADR-029) would additionally cross the
  proto contract and require `buf breaking` clearance + an ADR.
- **Injection surface — the decisive concern.** Context-injection is, by definition, putting
  externally-sourced content (a git diff, a file) into the `input` that reaches a capability's
  execution. Two distinct risks, each with a hard mitigation:
  1. **Routing redirection (data-vs-config leak).** If the block could carry
     `provider`/`model`/`endpoint`/`url`, an attacker who controls the injected content could
     redirect *where/how* a capability runs (e.g. to a paid or attacker-controlled provider). **Hard
     rule:** the block is **data-only** — the schema admits content sources only, the compiler
     **rejects any routing/provider field at compile time**, and provider/model/endpoint stay in
     `AgentDef`/overlay config and are never accepted from `input_payload` (ADR-013 http-adapter rule,
     ADR-035). This is enforced in O1 (schema) **and** O3 (compiler rejection) — not just documented.
  2. **Unsanitised content reaching execution.** The injected diff/file is **data**, delivered to a
     capability whose `input_schema` (`agent-def.schema.json`) validates structure. #1387 adds **no
     new template engine** — it binds values into the compiler's **existing, already-bounded**
     `{{ .ctx.* }}` evaluator, so no new injection-into-template path is created. The new guardrails
     it *adds* are the `max_tokens` hard cap (prevents unbounded `input_payload` inflation) and
     **strict scenario isolation** (one scenario's context never reaches another) — both the ADR-028
     contract. Prompt-injection *of the model by its own input content* is an inherent property of any
     LLM capability and is out of scope for the injection *contract*; the contract's job is to keep
     the content bounded, isolated, and data-only — which it does.

### Feature Safeguards
- **Never** accept provider/model/endpoint/URL — or any field that redirects where/how a capability
  runs — from the context-injection block or any `input_payload`. Declarative `AgentDef`/overlay
  config only (ADR-013, ADR-035). Enforced by schema **and** compiler rejection.
- **Never** inject context unbounded — every block declares a `max_tokens` cap with a documented
  overflow policy; the compiler enforces it (ADR-028).
- **Never** let one scenario's injected context reach another — strict isolation per ADR-028.
- **Never** widen the ADR-029 `input_bindings` / data-flow contract to carry external context — that
  contract is state→state, run-scoped, additive-only; #1387 is a separate spec surface.
- **Never** introduce a new template/expression language (CEL/JSONata/transforms) — reuse the
  existing `{{ .ctx.* }}` surface (ADR-029 non-goal stands).
- **Never** put gRPC or proto-generated types in the CLI (ADR-001 / cmd/zynax AGENTS.md).
- **Never** introduce a shared database or Layer 1→3 coupling (root AGENTS.md mandates).
- **Never** re-define the `spec.context` slot or the scenario manifest-set convention — those belong
  to #1385; #1387 only fills the slot.

---

## Security review (Tier-2 pass — in-canvas)

> Self-assessed at Draft authoring; the maintainer re-runs `/lib:spdd-security-review` at alignment.

**Result: PASS (one load-bearing CONCERN — the data-only routing rule — fully mitigated by schema +
compiler enforcement).**

- **Tier 1 cleanliness:** no internal hostnames/IPs, no credentials/tokens, no PII, no
  injection-style authority phrasing. All E entities are public-safe abstractions. → PASS.
- **gRPC/spec boundary:** the recommended A2 introduces a new **spec** surface (the context-block
  schema) but **no new proto field on the dispatch path** — it rides the existing `{{ .ctx.* }}`
  template surface and existing `input_payload`. The spec boundary is gated by the O1 `.feature`
  (ADR-016). If the maintainer chooses A1 (amend ADR-029), that crosses the proto contract and
  requires `buf breaking` clearance + a `.feature` + an ADR before impl. → PASS for A2.
- **Injection surface (CONCERN → mitigated).** This is a genuine injection surface: external content
  flows into a capability's execution input. The decisive risk is **routing redirection** — injected
  content must never be able to carry `provider`/`model`/`endpoint`/`url` and redirect where a
  capability runs. The mitigation is **enforced in two places**, not hand-waved: the schema (O1)
  admits content sources only, and the compiler (O3) rejects any routing/provider field at compile
  time; provider/model/endpoint remain config-only (ADR-013/ADR-035, never `input_payload`). The
  secondary risks — unbounded `input_payload` inflation and cross-scenario leakage — are closed by the
  ADR-028 `max_tokens` cap and strict-isolation guarantees, both enforced in O3. No new template
  evaluator is introduced. → PASS, with the data-only rule as the load-bearing, code-enforced
  mitigation.

No unmitigated attack surface remains under the recommended approach. Honest caveat: prompt-injection
*of an LLM by its own legitimate input content* is an inherent property of LLM capabilities, not a
property of this injection *contract* — the contract's job (bounded, isolated, data-only delivery) is
fully met; defending the model against its own input is a separate, capability-level concern.

---

## Notes / Story-decomposition

> Maps O-steps to candidate child issues for the maintainer to file at alignment. None created yet —
> this is a **Draft** canvas; `/lib:spdd-story #1387` runs only after `Status: Aligned`.

| O-step | Candidate child story (file at alignment) | Type | Size | Boundary / gate |
|--------|-------------------------------------------|------|------|-----------------|
| O1 | `feat(spec): context-injection block schema + BDD (#1387, step 1)` | feature | S | spec boundary — `.feature` first |
| O2 | `feat(cli): validate the context-injection block locally (#1387, step 2)` | feature | S | CLI-local, no boundary |
| O3 | `feat(workflow-compiler): bind + bound context into action input (#1387, step 3)` | feature | M | compiler; covered by O1 `.feature`; enforces ADR-028 bounds + isolation |
| O4 | `feat(spec): code-review scenario injects a real git diff (#1387, step 4)` | feature | M | depends on #1385 O1–O3 + #1370 cluster |
| O5 | `docs: context-injection authoring + human-validation guide (#1387, step 5)` | docs | S | SPDD-exempt |

**Dependencies & cross-links:**
- **Sibling #1385** (scenario manifest-set, Draft on PR #1447) reserves the `spec.context` **slot**;
  #1387 owns the **contract** that fills it. The two canvases share one seam — the `spec.context`
  slot here is exactly where #1387's `ContextInjectionBlock` lands. They must stay consistent.
- **EPIC W / ADR-029** (data-flow `output→input`, shipped) is the **boundary line**: #1387 must not
  widen `input_bindings`; it adds a separate initial-context surface.
- **EPIC C / ADR-031** (context propagation + dispatch handoff) is the **carrier** #1387 rides; #1387
  does not re-spec it.
- **ADR-028** owns the `{files[], max_tokens}` + strict-isolation context-slice contract that #1387
  applies at the scenario boundary.
- **#1370** parent-epic fixes (#1371–#1381, merged) and **#1388** human-validation-guide template are
  prerequisites for O4/O5.

**Resolved (alignment decision, maintainer-authorized 2026-06-19):** **A2 (new `context:` block on the
existing `{{ .ctx.* }}` surface) is SELECTED**; **A1 (extend ADR-029 `input_bindings`) is REJECTED** —
A1 re-opens an accepted, additive-only proto contract for the wrong concern and needs an ADR-029
amendment + `buf breaking` clearance; A2 needs neither. The rest of the canvas is written against A2.
