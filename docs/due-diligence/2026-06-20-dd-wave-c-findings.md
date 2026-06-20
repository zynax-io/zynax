<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax Due-Diligence — Wave C (Product / Market / Governance) Findings

> **Run output of issue #1404** — the third execution wave of the investment-grade due-diligence
> framework ([2026-06-18-zynax-due-diligence-framework.md](2026-06-18-zynax-due-diligence-framework.md)),
> lightly consuming the Wave A ground truth ([2026-06-20-dd-wave-a-findings.md](2026-06-20-dd-wave-a-findings.md)).
> A **findings artifact**, not a verdict: the recommendation is produced only after Wave D
> synthesis (#1405) and the final report (#1406).

| Field | Value |
|-------|-------|
| Wave | **C — Product / market / governance** (parallel; lightly consumes Wave A; framework §3.2) |
| Issue | #1404 — *DD execution: run Wave C (product / market / governance agents)* |
| Date | 2026-06-20 |
| Repository HEAD audited | `main` @ `e3135a6` (code state; Wave A doc merged later as `8926b28`) |
| Agents run | 9 — §5.3, 5.4, 5.8, 5.11, 5.13, 5.19, 5.20, 5.21, 5.25 |
| Evidence discipline | §0.4 — every claim carries `path:line`, a command+output, an external citation (E7), or `UNKNOWN`; roadmap/marketing = `CLAIMED` |

## Provenance & consumption note

Each agent ran **read-only** and returned the §3.4 YAML packet plus a §6.2 prose section, scoring
only its primary zone (anti-overlap §3.1). Wave C consumes Wave A's Documentation (5.10) and
Repository Health (5.24) lightly; market/competitive agents also cite external ecosystem data (E7,
sourced inline). The first batch (5.3, 5.4, 5.19) ran before the merged Wave A doc was locally
available and re-derived the relevant upstream findings from code; the remaining six consumed the
committed Wave A packet directly. Several Wave C agents have **intra-wave** dependencies (5.21←5.8/5.13;
5.25←5.13/5.3) which, being same-wave, are recorded as cross-references for the orchestrator rather
than consumed live. Contributor emails and absolute local paths were neutralised (secret/PII gate).

## Wave C scorecard

> Provisional, un-weighted. Final aggregation is the orchestrator's (Wave D).

| Agent | Dim | Score | Conf | Most severe red flag (evidence) | Strongest green flag (evidence) |
|-------|-----|:---:|:---:|----------------------------------|----------------------------------|
| 5.3 Product | D1 | 6 | Med | hero "run on Temporal **OR** Argo" is CONTRADICTED — Argo ships no IR interpreter — `README.md:7` vs `argo_engine.go` | runnable, zero-secret, one-command Temporal hero path; both M7 keystones (data-flow, OTEL) actually shipped |
| 5.4 Market | D2 | 6 | Med | CNCF-Sandbox-backed Kagent owns the identical "control plane for AI agents" framing while Zynax is at ~0 stars/forks/adopters | hard-to-copy, partially-proven engine-neutral IR moat (Temporal + Argo behind one port) |
| 5.8 Open Source | D7 | 6 | High | **no top-level LICENSE file** for an Apache-2.0/CNCF-bound OSS project (`git log --all -- LICENSE` empty); bus factor = 1 | exemplary Truth-Pass transparency + honest solo-maintainer governance matrix |
| 5.11 Developer Experience | D1 | 7 | High | "one command" demo drift — `make demo` needs 4 prerequisites and exits if the CLI isn't pre-installed — `README.md:24-33` vs `Makefile:162-175` | self-documenting Makefile (72 ★-annotated targets) + script-friendly Cobra CLI |
| 5.13 Governance | D12 | 7 | High | `MAINTAINERS.md` referenced by GOVERNANCE.md + CODEOWNERS but **absent** (#494 open); "no single company controls" contradicted | honest, self-correcting Truth-Pass habit (M3/M4 down-graded with reasons in the canonical state file) |
| 5.19 Competitive | D2 | 5 | Med | **Critical** — Kagent out-positions Zynax on CNCF backing, web UI, MCP discovery, HITL, GitOps, multi-LLM while Zynax has 0 adopters | compile-time structural IR validation — shipped, rare, hard-to-copy — `validators/structural.go:11-58` |
| 5.20 Enterprise Adoption | D14 | 4 | High | no enterprise identity layer — single shared static bearer key, no RBAC/SSO/OIDC, scoped-token authz a `pending` stub — `auth.go:13-26` | policy/quota/rate-limit enforcement is real and tested — `quota_check_test.go:35`, `ratelimit.go:17-69` |
| 5.21 CNCF Readiness | D7 | 5 | High | verdict **NOT YET** — missing LICENSE makes the project non-donatable as-is; bus factor 1 + no adopters stack on top | unusually honest self-assessment that itself says "don't file until prerequisites are real" |
| 5.25 Future Roadmap | D12 | 7 | High | the binding v1.0/CNCF constraint is **social** (≥2 cross-org maintainers, audit, TOC sponsor) vs bus factor = 1 — unbuyable by velocity | the data-flow keystone (ADR-029) is shipped and execution-verifiable; EPIC #1167 + W.2–W.5 closed |

Un-weighted mean ≈ **5.9 / 10**. Directional only.

## Aggregate drift test (Wave C contributions)

| Claim | Result | Evidence (agent) |
|-------|:---:|------------------|
| Hero promise: "Temporal OR Argo, traces in Uptrace, <15 min" | **PARTIAL** | Temporal run + Uptrace traces VERIFIED; "OR Argo" CONTRADICTED (5.3) |
| "<15 min / one command" demo | **PARTIAL** | `make demo` is a 4-prereq chain; "<15 min" is a CLAIMED target with no committed measurement (5.11) |
| "Complementary to Kagent" co-existence | **PARTIAL (a hedge)** | Mechanism real (`agent.proto`), but zero Kagent integration/test; framed defensively in strategy doc (5.4, 5.19) |
| Apache-2.0 LICENSE file present | **CONTRADICTED** | No LICENSE in repo or git history; README badge + GOVERNANCE "done" overstate it (5.8, 5.21) |
| "No single company controls" / MAINTAINERS exists | **CONTRADICTED** | `MAINTAINERS.md` absent; single human committer (5.13) |
| "Production-ready" (enterprise lens) | **PARTIAL / overstated** | True for K8s deploy mechanics; false for enterprise governance (no RBAC/SSO/audit) (5.20) |
| Past milestone "Complete" labels | **VERIFIED honest** | M3/M4 re-labelled Partial with reasons; optimism since corrected — docs now under-state reality (5.13, 5.25) |

## Cross-cutting themes (provisional, for the orchestrator)

- **The binding constraint is social, confirmed from three angles.** Bus factor 1, zero named
  adopters, and a CNCF-backed rival (Kagent) on the identical positioning — Market, Competitive,
  Open Source, CNCF, and Roadmap all converge here. Velocity cannot buy it.
- **Two hard, cheap-to-fix governance gaps gate CNCF:** the missing LICENSE file and the missing
  MAINTAINERS.md are independently flagged and are pure execution oversights, not design problems.
- **The product is real on its beachhead, narrow beyond it.** The Temporal hero path runs and the
  M7 keystones shipped (Product, DX, Roadmap green), but the "two engines" headline, enterprise
  identity, and multi-tenancy are not there yet.
- **Honesty culture is a genuine asset** (Open Source, Governance, Roadmap): the project's own docs
  under-state rather than over-state — the opposite of the §1.10 history, and a positive diligence
  signal.

## Handoff

Wave C feeds the synthesis waves: Product/Market/Competitive/Enterprise/Governance/CNCF/Roadmap
findings flow to **Wave D (#1405)** — Risk (5.23), Investment (5.17), Business Strategy (5.18) + the
Part 4 orchestrator — and into the **final report (#1406)**.

---

# Per-Agent Findings Packets

> Each section is the verbatim, read-only output of one Wave C agent: its §3.4 YAML handoff packet
> followed by its §6.2 prose section. Local absolute paths and contributor emails were neutralised
> (secret/PII gate); no other content was altered.

---
<!-- SPDX-License-Identifier: Apache-2.0 -->
<!-- Zynax Due-Diligence · Wave C · Agent 5.3 Product · issue #1404 · evaluated at HEAD (main @ e3135a6) 2026-06-20 -->

# (a) §3.4 Handoff Packet

```yaml
agent: "5.3 Product"
wave: "C"
dimension_groups: ["D1", "D14"]   # D1 Product (primary); contributes to D14 Market/GTM synthesis
overall_score: 6
overall_confidence: "Medium"

sub_scores:
  - dimension: "Hero-journey completability from docs alone (quickstart traced E2E)"
    score: 6
    confidence: "Medium"
    justification: "Temporal-backed code-review-ollama path is real, honest, and CLI-runnable; but '<15min, two engines, traces' is only ~1.5 of 3 — Argo cannot dispatch capabilities and the README/Examples-Index runnable claims are partly stale."
    evidence:
      - "docs/quickstart.md:3-20"      # make demo one-command + 8-step manual hero path
      - "docs/quickstart.md:146-163"   # code-review-ollama runs to completion; others wait on external events
      - "spec/workflows/examples/code-review-ollama.yaml:1-14"  # honest 'runs to completion from CLI alone' header
      - "Makefile:162-205"             # make demo target boots Ollama overlay + applies hero workflow + prints result
      - "cmd/zynax/cmd/ → apply.go result.go events.go logs.go validate.go status.go get.go init.go all present"
  - dimension: "Runnable example inventory (real compile+dispatch vs illustrative)"
    score: 5
    confidence: "High"
    justification: "Only 2 of 10 examples run to completion from the CLI (e2e-demo echo, code-review-ollama LLM); the 3 headlined 'real, runnable' event-driven workflows compile but HANG (no event source, no deployed agents)."
    evidence:
      - "spec/workflows/examples/e2e-demo.yaml:1-16"            # minimal echo dispatch fixture, runs
      - "spec/workflows/examples/code-review-ollama.yaml:5-8"   # runs to terminal via Ollama codereview
      - "spec/workflows/examples/code-review.yaml:2-9"          # '⚠ NOT a runnable CLI demo … just hangs'
      - "spec/workflows/examples/feature-implementation.yaml:32-36"  # github.* triggers + undeployed capabilities
      - "docs/examples/index.md:19-32"   # labels the 3 event-driven specs 'Real, runnable … verified against the running stack'
  - dimension: "Value-prop clarity vs Kagent (differentiation legible to a buyer)"
    score: 7
    confidence: "Medium"
    justification: "Crisp, repeatedly-stated engine-portability + adapter-first + GitOps wedge with an explicit Kagent comparison table and co-existence story; weakened by the Argo-portability claim being thinner than messaged."
    evidence:
      - "README.md:7-9"                # 'Write your agent workflow once — run on Temporal or Argo without a rewrite'
      - "README.md:45-59"              # wedge framing + Kagent co-existence
      - "docs/product/strategy.md:169-194"   # Zynax-vs-Kagent table + positioning takeaways
      - "docs/product/strategy.md:49-55"     # one-defensible-wedge statement
  - dimension: "Product completeness for hero persona (data-flow ADR-029, observability ADR-030)"
    score: 7
    confidence: "High"
    justification: "Both keystones are implemented at HEAD, not just ADR-decisions: output/input bindings parse+compile in the IR; OTEL is wired into 5 services with Uptrace compose overlay."
    evidence:
      - "protos/zynax/v1/workflow_compiler.proto:158-168"   # output_bindings / input_bindings IR fields
      - "services/workflow-compiler/internal/domain/manifest.go:262-269,319-335"  # convertOutputBindings — output: rejection LIFTED
      - "libs/zynaxobs/providers.go (InitProviders, OTLP/gRPC, env-gated)"  # real instrumentation, off by default
      - "infra/docker-compose/docker-compose.observability.yml (uptrace:1.7.0 :7020, otel-collector :7017)"
  - dimension: "Adoption barriers (Temporal dep · YAML-only · Day-0 friction · no web UI)"
    score: 5
    confidence: "Medium"
    justification: "make demo + zero-secret Ollama overlay materially cut Day-0 friction; but Temporal is still the only real engine, authoring is YAML-only, there is no web UI for runs (Temporal/Uptrace UIs only), and the zero-Temporal eval mode is deferred to M-dx."
    evidence:
      - "Makefile:160"                 # DEMO_SERVICES := api-gateway llm-adapter (Temporal in depends_on closure)
      - "docs/product/strategy.md:279-285"  # Day-0 friction audit (Temporal dep, YAML-only, no web UI)
      - "state/current-milestone.md:71"     # #1359 zero-Temporal Day-0 engine DEFERRED to M-dx
      - "docs/product/strategy.md:124"      # 'Web UI 📅 Aspirational — none today'
  - dimension: "Roadmap realism (M7/M8) vs delivery history"
    score: 7
    confidence: "Medium"
    justification: "M7 is ~92% delivered with honest shipped/partial/aspirational tables and a documented Truth-Pass culture; M8 (CNCF) gates are correctly framed as social (maintainer/adopter/traction), not technical — credible sequencing."
    evidence:
      - "state/current-milestone.md:63-73"   # M7 ~115 closed / ~10 open (~92%), explicit deferrals
      - "docs/product/strategy.md:113-124"   # honest shipped/partial/aspirational split
      - "docs/product/strategy.md:330-346"   # CNCF gate is community, not tech
      - "docs/product/strategy.md:126-130"   # Truth-Pass (M5.A) removed premature CNCF badge

drift_test:
  - claim: "Hero journey: ship YAML → run on Temporal OR Argo → traces in Uptrace → <15 min, completable from docs alone."
    result: "PARTIAL"
    evidence:
      - "docs/quickstart.md:1-220 — full 8-step path exists, commands map to real CLI subcommands (cmd/zynax/cmd/*.go all present)"
      - "VERIFIED leg: Temporal hero run — spec/workflows/examples/code-review-ollama.yaml:5-8 + Makefile:162-205"
      - "VERIFIED leg: traces in Uptrace — libs/zynaxobs wired into 5 services; docker-compose.observability.yml ships uptrace+collector; docs/quickstart.md:119-220"
      - "FAILED leg: 'OR Argo' — ArgoEngine only SUBMITS IR to an external WorkflowTemplate (argo_engine.go:62-98,280-289); the only shipped template scripts/e2e/manifests/argo-ir-interpreter.yaml is an alpine smoke-stub that validates payload≠empty and exits 0, with NO capability dispatch"
  - claim: "Two engines shipped & tested: 'run on Temporal OR Argo without a rewrite' (README.md:7; strategy.md:118)."
    result: "CONTRADICTED"
    evidence:
      - "services/engine-adapter/internal/infrastructure/argo_engine.go:25-28,280-283 — Argo path passes serialized IR JSON as a 'workflow-ir' param to a cluster WorkflowTemplate; no IRInterpreter/DispatchCapability for Argo (those exist only in temporal_workflow.go / activities.go / activity_dispatch.go)"
      - "scripts/e2e/manifests/argo-ir-interpreter.yaml — smoke template: 'Full Argo-side IR interpretation (capability-dispatch parity with the Temporal IRInterpreterWorkflow) is deliberately out of scope for the smoke gate'"
      - ".github/workflows/e2e-smoke.yml:56-61 — argo matrix leg runs with fail-fast:false (advisory; Temporal is the protected baseline). Argo proves only Workflow CR phase=Succeeded, which the stub trivially reaches."
  - claim: "Data-flow output/input bindings are implemented (ADR-029); compiler no longer rejects output:."
    result: "VERIFIED"
    evidence:
      - "protos/zynax/v1/workflow_compiler.proto:158-168 — output_bindings/input_bindings fields present (additive)"
      - "services/workflow-compiler/internal/domain/manifest.go:262-269,319-335 — convertOutputBindings parses output:; M6's 'output: rejection' lifted by M7 EPIC W (#1177/#1178)"
      - "NOTE drift: docs/product/strategy.md:122 still says '🟡 Partial … compiler still gates output:' — now STALE/CONTRADICTED at HEAD"

red_flags:
  - severity: "High"
    finding: "'Two engines without a rewrite' is the headline wedge but Argo cannot execute a real workflow — it only submits the IR to an external WorkflowTemplate that the repo does not ship in production form; the only shipped template is an alpine smoke-stub with zero capability dispatch. A buyer testing the differentiating claim on Argo gets a workflow that reaches 'Succeeded' without running anything."
    evidence:
      - "services/engine-adapter/internal/infrastructure/argo_engine.go:62-98,280-289"
      - "scripts/e2e/manifests/argo-ir-interpreter.yaml (smoke-only, out-of-scope note)"
      - "README.md:7 / docs/product/strategy.md:118,177 (claim asserted as ✅ Shipped)"
      - "cross-ref Wave A 5.1 Architecture: 'Argo execution stubbed'"
  - severity: "Medium"
    finding: "Doc-vs-reality drift on 'runnable' examples: docs/examples/index.md, docs/faq.md, and strategy.md headline three EVENT-DRIVEN workflows (code-review/feature-implementation/ci-pipeline) as 'real, runnable … verified against the running stack', but their own YAML headers say they HANG from the CLI (no event source, undeployed capabilities). 'Compile green' is conflated with 'runs end-to-end'. The actually-runnable hero (code-review-ollama) is absent from the Examples Index."
    evidence:
      - "docs/examples/index.md:5-8,19-32 (no code-review-ollama; no 'hangs' caveat)"
      - "spec/workflows/examples/code-review.yaml:2-9"
      - "docs/faq.md:24-28 (tells user to apply the non-runnable code-review.yaml as 'a real example')"
      - "docs/product/strategy.md:256-260"
  - severity: "Medium"
    finding: "Stale README Quickstart + placeholder hero asset undercut Day-0 conversion: README 'Try it with Docker' still narrates 'capability dispatch pending M5.C' and points users at the hanging code-review.yaml; the README hero asciinema cast is a literal PLACEHOLDER. First-impression surfaces lag the honest quickstart.md."
    evidence:
      - "README.md:333-355 (pending-M5.C language; applies code-review.yaml)"
      - "README.md:35-41 (asciicast PLACEHOLDER; cast not yet recorded)"
      - "state/current-milestone.md:73 (hero cast = human follow-up, docs/casts placeholder)"
  - severity: "Low"
    finding: "Reference agents are real (build/lint/test, ship capability JSON + BDD) but deterministic dependency-free stubs — 'swap the handler body for an LLM … to make one real'; go-review-expert's runtime AgentDef registration was delivered separately. They demonstrate the contract, not a working agentic review out of the box (the Ollama path covers that instead)."
    evidence:
      - "agents/examples/AGENTS.md:12-25"
      - "agents/examples/go-review-expert/capability.json (rule-based go_review)"

green_flags:
  - strength: "A genuinely runnable, zero-secret hero path exists at HEAD: `make demo` (one command) boots an Ollama overlay and a real local model reviews a git diff to a terminal state, printed straight from the CLI — the single most important conversion artifact, and it is honest about which examples do/don't run."
    evidence: ["Makefile:128-205", "docs/quickstart.md:3-20,146-163", "spec/workflows/examples/code-review-ollama.yaml:1-14"]
  - strength: "Both M7 usability keystones are actually implemented, not just ADR'd: data-flow output/input bindings compile in the IR, and OpenTelemetry is wired into 5 services with a ready Uptrace compose overlay (env-gated off-by-default) — the 'traces in Uptrace' leg of the hero promise is real."
    evidence: ["protos/zynax/v1/workflow_compiler.proto:158-168", "services/workflow-compiler/internal/domain/manifest.go:262-269", "libs/zynaxobs/providers.go", "infra/docker-compose/docker-compose.observability.yml"]
  - strength: "Differentiation vs Kagent is crisp and buyer-legible, carried consistently across README + strategy with an explicit comparison table and a co-existence (not-a-fight) story; the strategy doc is unusually honest (shipped/partial/aspirational split + Truth-Pass culture)."
    evidence: ["README.md:45-59", "docs/product/strategy.md:169-194,113-130"]
  - strength: "Quickstart commands all map to real CLI subcommands (apply/result/events publish/logs --follow/validate/status/get/init), and the quickstart explicitly documents which examples run vs which are reference-only — low-friction and trustworthy where it matters."
    evidence: ["cmd/zynax/cmd/{apply,result,events,logs,validate,status,get,init}.go", "docs/quickstart.md:160-163,237-254"]

open_questions:
  - "Wall-clock: does `make demo` (cold image build + Ollama pull + Temporal boot + LLM inference on qwen2.5-coder:3b) actually finish in <15 min on a typical evaluator laptop? Not measurable statically — runtime smoke needed."
  - "Who is the buyer today? Strategy names the hero persona (AI-forward dev team) but lists ZERO external adopters / 0 stars-forks baseline — is there any named pilot since the May review?"
  - "Will the stale Examples-Index/FAQ/README runnable claims be reconciled, or do they recur each milestone (this is the exact drift class the project's Truth-Pass was meant to kill)?"

unknowns:
  - "Wave A findings file (docs/due-diligence/2026-06-20-dd-wave-a-findings.md) DOES NOT EXIST at HEAD — could not directly consume the Documentation 5.10 packet; I independently traced quickstart accuracy and cite the 5.1 'Argo stubbed' note as relayed in my own prompt."
  - "End-to-end <15-min timing — UNKNOWN, static-only audit (no go/docker run permitted)."
  - "Whether any community channel cadence or external adopter materialized post-strategy-doc — out of repo scope."

cross_references:
  - to_agent: "5.1 Architecture"
    note: "Confirm/own the 'Argo execution stubbed' finding: ArgoEngine submits IR only; no Argo IRInterpreter/DispatchCapability; shipped argo-ir-interpreter.yaml is a smoke-stub. Drives my High red flag on the two-engine product claim."
    evidence: ["services/engine-adapter/internal/infrastructure/argo_engine.go:62-98", "scripts/e2e/manifests/argo-ir-interpreter.yaml"]
  - to_agent: "5.19 Competitive"
    note: "The Kagent 'engine portability' differentiator is weaker in practice than messaged — only Temporal truly executes; factor into the competitive 'wins/loses' analysis and the co-existence framing stress-test."
    evidence: ["docs/product/strategy.md:176-185", "argo_engine.go:280-289"]
  - to_agent: "5.10 Documentation"
    note: "examples/index.md:19-32, faq.md:24-28, and README.md:333-355 over-claim/stale-claim runnability of event-driven examples and point users at the hanging code-review.yaml; quickstart.md and YAML headers are the honest source. Doc-accuracy zone is yours to score."
    evidence: ["docs/examples/index.md:5-8", "docs/faq.md:24-28", "README.md:333-355"]
  - to_agent: "5.20 Enterprise Adoption / 5.18 Business Strategy"
    note: "Zero-Temporal eval mode deferred to M-dx; Temporal remains a mandatory dep for any real run — the #1 Day-0 evaluation friction the strategy itself names."
    evidence: ["state/current-milestone.md:71", "docs/product/strategy.md:281,407"]

recommendations:
  - priority: "P0"
    action: "Reconcile the 'runnable' claim across README 'Try it with Docker', docs/examples/index.md, and docs/faq.md to match quickstart.md + the YAML headers: feature the CLI-runnable code-review-ollama / e2e-demo as the runnable set, and clearly label code-review/feature-implementation/ci-pipeline as reference-only (compile-green, not CLI-runnable)."
    rationale: "Persona: hero AI-forward dev team. Lever: Day-0 trust. This is the exact delivery-vs-narrative drift class the project's own Truth-Pass culture exists to prevent; a first-time evaluator who applies the headlined code-review.yaml gets a hung run and bounces."
  - priority: "P0"
    action: "Stop asserting Argo as a co-equal shipped execution engine in product/marketing surfaces (README hero line, strategy '✅ Shipped (Temporal+Argo)'). Re-label as 'Temporal (full execution) + Argo (submission/portability-proof; full IR interpretation roadmap)' until an Argo-side IR interpreter with capability dispatch ships."
    rationale: "Persona: platform engineer evaluating engine-portability (the core wedge). Lever: differentiation honesty. The headline promise is the one a sophisticated buyer will test first, and it currently fails on Argo."
  - priority: "P1"
    action: "Record the hero asciinema cast (replace README PLACEHOLDER) and add a measured wall-clock to the <15-min claim from a clean clone on a reference laptop; surface the make-demo path as the literal first action in the README (it already is — keep it, but prove the timing)."
    rationale: "Persona: hero. Lever: time-to-first-value, the strategy's own #1 conversion metric (strategy.md:285,294). A believable first 15 minutes is the single highest-leverage adoption asset."
  - priority: "P1"
    action: "Add the code-review-ollama hero workflow (and the scenario diff-injection variant spec/scenarios/code-review/) to docs/examples/index.md as the headline runnable example."
    rationale: "Persona: hero. Lever: discoverability. The one example that actually runs end-to-end is currently absent from the canonical Examples Index."
  - priority: "P2"
    action: "Prioritize the deferred zero-Temporal Day-0 eval engine (#1359, M-dx) onto the near roadmap, or document a clearly-supported lightweight Temporal mode (EVAL_TEMPORAL=1 already exists) front-and-centre in the quickstart."
    rationale: "Persona: evaluator / platform engineer. Lever: Day-0 friction. 'Needs Temporal' is the named top evaluation deterrent; EVAL_TEMPORAL=1 partly answers it but is buried in a make-demo footnote (Makefile:204)."
```

# (b) §6.2 Prose

## 5.3 Product — Score: 6 (Medium)

**Mission recap:** Judge product vision, market fit, completeness, value proposition, adoption barriers, and whether the "usable today" story is real for the hero persona — can the hero journey be completed from docs alone, and is the "<15 min, two engines, traces" promise achievable at HEAD?

**Verdict:** Zynax has a genuinely real, honest, zero-secret hero path on **Temporal** — `make demo` boots an Ollama overlay and a local model reviews a git diff to completion, with traces available in a ready Uptrace overlay — and both M7 usability keystones (data-flow bindings, OTEL/Uptrace) are actually implemented, not merely ADR'd. That is a strong, conversion-ready core. But the headline "**run on Temporal OR Argo without a rewrite**" does **not** survive contact with HEAD: the Argo engine only *submits* the IR to an external WorkflowTemplate the repo never ships in production form, and the only committed template is an alpine smoke-stub that dispatches nothing. Compounding this, three first-impression docs (README "Try it with Docker", Examples Index, FAQ) over-claim the runnability of event-driven examples that their own YAML headers say *hang* from the CLI — the precise delivery-vs-narrative drift class the project's Truth-Pass was built to prevent. The product is credible-and-usable for one engine and one beachhead; the multi-engine wedge is thinner than messaged.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Hero-journey completability (docs alone, traced E2E) | 6 | Medium | `docs/quickstart.md:3-20,146-163`; `Makefile:162-205`; `spec/workflows/examples/code-review-ollama.yaml:1-14` |
| Runnable example inventory (real vs illustrative) | 5 | High | `e2e-demo.yaml:1-16`; `code-review-ollama.yaml:5-8`; `code-review.yaml:2-9`; `docs/examples/index.md:19-32` |
| Value-prop clarity vs Kagent | 7 | Medium | `README.md:7-9,45-59`; `docs/product/strategy.md:169-194` |
| Completeness for hero (data-flow ADR-029, obs ADR-030) | 7 | High | `workflow_compiler.proto:158-168`; `manifest.go:262-269`; `libs/zynaxobs/providers.go`; `docker-compose.observability.yml` |
| Adoption barriers (Temporal · YAML-only · Day-0 · no UI) | 5 | Medium | `Makefile:160`; `strategy.md:279-285`; `state/current-milestone.md:71`; `strategy.md:124` |
| Roadmap realism (M7/M8) vs history | 7 | Medium | `state/current-milestone.md:63-73`; `strategy.md:113-124,330-346` |

**Drift test:**
- Hero promise "<15 min, two engines, traces, from docs alone" → **PARTIAL**. Temporal hero run = VERIFIED; traces-in-Uptrace = VERIFIED; "OR Argo" = FAILED (submission-only, no dispatch). Quickstart commands all map to real CLI subcommands.
- "Two engines shipped & tested without a rewrite" (`README.md:7`, `strategy.md:118`) → **CONTRADICTED**. Argo has no IR interpreter / capability dispatch; shipped template is a smoke-stub; argo CI leg is advisory (`e2e-smoke.yml:56-61`).
- "Data-flow output/input bindings implemented; compiler no longer rejects `output:`" (ADR-029) → **VERIFIED** (`workflow_compiler.proto:158-168`, `manifest.go:262-335`). Note: `strategy.md:122` is now stale on this.

**Red flags (severity-ordered):**
1. **High** — Argo cannot execute a real workflow; the differentiating "two engines" claim is asserted as ✅ Shipped but is submission-only (`argo_engine.go:62-98`; `scripts/e2e/manifests/argo-ir-interpreter.yaml`). Cross-refs Wave A 5.1 "Argo stubbed."
2. **Medium** — "Runnable" doc drift: Examples Index / FAQ / strategy headline event-driven workflows that hang from the CLI as "real, runnable, verified" (`docs/examples/index.md:19-32`; `faq.md:24-28`; `code-review.yaml:2-9`); the actually-runnable hero is absent from the Index.
3. **Medium** — Stale README Quickstart ("pending M5.C") + PLACEHOLDER hero asciicast undercut first-impression conversion (`README.md:35-41,333-355`).
4. **Low** — Reference agents are real-but-deterministic stubs demonstrating the contract, not working agentic review (`agents/examples/AGENTS.md:12-25`).

**Green flags:**
- Real, honest, one-command zero-secret hero run on Temporal (`make demo` → LLM reviews a diff to terminal) — the highest-leverage conversion artifact (`Makefile:128-205`; `code-review-ollama.yaml`).
- Both M7 keystones implemented for real: data-flow bindings compile; OTEL wired into 5 services + Uptrace compose overlay, env-gated (`workflow_compiler.proto:158-168`; `libs/zynaxobs`; `docker-compose.observability.yml`).
- Crisp, consistent, buyer-legible Kagent differentiation + unusually honest shipped/partial/aspirational strategy and Truth-Pass culture (`README.md:45-59`; `strategy.md:113-130,169-194`).
- All quickstart commands map to real CLI subcommands; quickstart honestly flags runnable vs reference-only.

**Open questions / unknowns:** Actual wall-clock of `make demo` (<15 min?) is not statically measurable. Wave A findings file does not exist at HEAD — could not directly consume the 5.10 packet. Zero named external adopters / 0-star baseline per strategy; no traction signal verifiable in-repo. Static-only audit (no run permitted).

**Recommendations:** (P0) reconcile the "runnable" claim across README/Examples-Index/FAQ to match quickstart truth [hero · Day-0 trust]; (P0) stop asserting Argo as a co-equal shipped engine until Argo-side IR interpretation ships [platform engineer · differentiation honesty]; (P1) record the hero cast and prove the <15-min timing [hero · time-to-value]; (P1) add code-review-ollama to the Examples Index [hero · discoverability]; (P2) surface/advance a zero-Temporal eval mode [evaluator · Day-0 friction].

**Cross-references:** 5.1 Architecture (own the Argo-stubbed finding); 5.19 Competitive (engine-portability wedge weaker than messaged); 5.10 Documentation (runnable-example doc drift); 5.20 Enterprise / 5.18 Business (Temporal mandatory-dep friction).
<!-- SPDX-License-Identifier: Apache-2.0 -->
<!-- Zynax Due-Diligence · Wave C · Agent 5.4 — Market Agent · issue #1404 · evaluated at HEAD (main @ e3135a6) -->

# Agent 5.4 — Market Agent (Wave C)

> **Wave A consumption note.** The handoff path
> `docs/due-diligence/2026-06-20-dd-wave-a-findings.md` **does not exist** in the repo at HEAD
> (`ls docs/due-diligence/` → only `2026-06-18-zynax-due-diligence-framework.md`). I therefore
> could not consume the Architecture (5.1) packet directly. To honor the instruction to weigh the
> "portability-is-partial" finding, I re-derived it independently from the same ground truth the
> 5.1 agent would cite: the engine-adapter source, the CI engine matrix, and the prior architecture
> reviews — and I CITE Architecture (5.1) as the owner without re-scoring its zones (§3.1). This is
> logged as an `unknown` below.

---

## (a) §3.4 handoff packet

```yaml
agent: "5.4 Market"
wave: "C"
dimension_groups: ["D2"]          # contributes to D13 (Business Strategy) and D15 (Investment)
overall_score: 6
overall_confidence: "Medium"
sub_scores:
  - dimension: "Category definition & TAM/SAM/SOM framing"
    score: 6
    confidence: "Low"
    justification: "Real, articulable category (control plane above agent orchestration, across durable engines); TAM is assumption-based (E7), no bottom-up sizing in repo."
    evidence:
      - "docs/product/strategy.md:134-157 (three-market intersection + 'category Zynax is creating')"
      - "docs/due-diligence/2026-06-18-zynax-due-diligence-framework.md:174-177 (category triad)"
      - "E7: CNCF Cloud Native AI landscape (cncf.io/projects, landscape.cncf.io) — agent-orchestration is a 2025-26 emergent category, not yet a sized market"
  - dimension: "Comparables / competitive matrix (OSS, commercial, emerging)"
    score: 6
    confidence: "Medium"
    justification: "Honest, repo-grounded matrix exists; Temporal/Argo are wrapped-not-rivals, Kagent is the one true direct competitor and is CNCF-backed."
    evidence:
      - "docs/architecture/2026-05-28-competitive-positioning.md:24-36 (5-way matrix)"
      - "docs/product/strategy.md:196-208 (full landscape: Kagent direct; Restate/Dapr/Flyte adjacent)"
      - "E7: Kagent = CNCF Sandbox 2026 (cncf.io sandbox roster); Argo Workflows = CNCF Graduated; Dapr = CNCF Graduated; Temporal/Restate/Flyte/LangGraph not CNCF"
  - dimension: "Market timing (early/right/late)"
    score: 7
    confidence: "Medium"
    justification: "'Right, leaning early' — agent-orchestration demand is inflecting in 2025-26, but the engine-portability pain is not yet acute for most buyers (most run one engine)."
    evidence:
      - "README.md:7-9,47-49 (portability wedge is the lead message)"
      - "docs/product/strategy.md:152-157 ('crowded category … must be ruthlessly defended')"
      - "E7: LangGraph/CrewAI/AutoGen explosion 2024-25 → control-plane consolidation thesis is timely (Gartner/CNCF AI-agent commentary 2025-26)"
  - dimension: "SWOT"
    score: 6
    confidence: "Medium"
    justification: "Strengths real (engine-agnostic IR proven asymmetrically; integrity/Truth-Pass); fatal-class weakness is zero traction + single maintainer."
    evidence:
      - "services/engine-adapter/internal/domain/engine.go:17-41 (clean WorkflowEngine port, 2 real impls)"
      - "docs/product/strategy.md:62-64,337-338 (zero stars/forks, single maintainer baseline)"
      - "docs/due-diligence/2026-06-18-zynax-due-diligence-framework.md:252 (bus-factor + zero adopters = gating gap)"
  - dimension: "Moat & defensibility (pressure-tested vs §1.5)"
    score: 6
    confidence: "Medium"
    justification: "Two genuine moats (engine-agnostic IR; stable AgentService integration contract); both are narrow, copyable-with-effort, and unprotected by traction/ecosystem."
    evidence:
      - "protos/zynax/v1/agent.proto:1-47 (AgentService = no-SDK universal capability contract = integration moat)"
      - "services/engine-adapter/internal/infrastructure/argo_engine.go:58-203 + temporal.go (two real engines behind one port = IR moat, partially proven)"
      - "docs/product/strategy.md:218-229 (adapter library explicitly conceded as commoditized / low moat)"
  - dimension: "Network effects / platform strategy / expansion vectors"
    score: 5
    confidence: "Low"
    justification: "Adapter/capability model is a potential flywheel (every new gRPC capability is reusable) but no live network effect today — 0 community adapters; expansion vectors are roadmap/CLAIMED."
    evidence:
      - "ROADMAP.md:213 (reusable templates #1171), ROADMAP.md:233 (hosted playground epic #1389)"
      - "spec/templates/ (expert/task/workflow template dirs exist) + spec/workflows/examples/ (10 runnable workflows)"
      - "docs/product/strategy.md:296-297 (community-built adapters = 'first community adapter' is still a target, not achieved)"
drift_test:
  - claim: "'Complementary to Kagent, not rivals' — a real co-existence story (a Kagent agent registers as a Zynax capability via AgentService gRPC)."
    result: "PARTIAL"
    evidence:
      - "protos/zynax/v1/agent.proto:1-47 (AgentService is genuinely no-SDK/no-framework/any-language → the technical mechanism for co-existence IS real and shipped)"
      - "BUT: docs/product/strategy.md:190-194 + 2026-05-28-competitive-positioning.md:69-83 frame it as a way to avoid 'a fight Zynax cannot yet win on ecosystem size' — i.e. a hedge against a CNCF-backed rival, not a demonstrated joint deployment (no combined Kagent+Zynax integration test or reference in repo)"
      - "docs/due-diligence/2026-06-18-zynax-due-diligence-framework.md:208 (framework itself flags: 'stress-test whether this framing survives a buyer who has already adopted Kagent')"
  - claim: "Engine-agnostic: 'run it on Temporal OR Argo without a rewrite' (Temporal + Argo proven in CI matrix)."
    result: "PARTIAL"
    evidence:
      - "VERIFIED real: .github/workflows/e2e-smoke.yml:60-61 (matrix engine:[temporal,argo]); scripts/e2e/e2e-argo.sh:178-266 submits real code-review.yaml via api-gateway ?engine=argo and asserts Argo Workflow CR reaches Succeeded"
      - "BUT asymmetric: e2e-smoke.yml:147-173 — temporal leg runs happy-path AND failure-path; argo leg runs ONLY a single happy-path assertion. Portability is proven, not equally hardened."
      - "argo_engine.go:58-203 is a real implementation (314 LoC, argoproj.io/v1alpha1 CRs); contrast 2026-06-18-architecture-review.md:151 which still called ArgoEngine 'stubbed' — that doc lags HEAD; the engine is now real"
  - claim: "Defensible category — 'the control plane for AI agent workflows'."
    result: "PARTIAL"
    evidence:
      - "Category is articulable and the IR/AgentService moats are real (engine.go:17-41, agent.proto:1-47)"
      - "CONTRADICTED-as-defensible by its own docs: docs/product/strategy.md:57-64,152-157 admit a CNCF-backed rival (Kagent) holds the same tagline → category is contested, not owned"
red_flags:
  - severity: "High"
    finding: "A CNCF-Sandbox-backed direct competitor (Kagent) occupies the identical 'control plane for AI agents' framing while Zynax has zero stars/forks/named adopters and a single maintainer — the rival is winning the only dimension (distribution/ecosystem) that the market actually rewards for an OSS control plane."
    evidence:
      - "docs/product/strategy.md:57-64,200,337-338"
      - "docs/architecture/2026-05-28-competitive-positioning.md:35,57 (Kagent = CNCF Sandbox + growing ecosystem)"
      - "docs/due-diligence/2026-06-18-zynax-due-diligence-framework.md:194,252"
  - severity: "Medium"
    finding: "The 'complementary to Kagent' story is more hedge than proven co-existence — the AgentService mechanism is real but there is no integration test, reference deployment, or evidence a Kagent-adopting buyer would add Zynax rather than treat it as redundant control-plane overhead."
    evidence:
      - "2026-05-28-competitive-positioning.md:69-83 (narrative only, no test/reference)"
      - "docs/product/strategy.md:190-194 (explicitly 'a fight Zynax cannot yet win')"
  - severity: "Medium"
    finding: "TAM/SAM/SOM is entirely assumption-based (E7) with no bottom-up sizing in-repo; the portability pain that justifies the category is not yet acute for the majority of buyers who run a single engine — risk of a 'real but premature' market."
    evidence:
      - "No TAM artifact in docs/product/ (strategy.md:134-157 is qualitative only)"
      - "docs/product/strategy.md:240-242 (framework's own 'multi-engine portability actually valued by buyers' is an untested key assumption)"
  - severity: "Low"
    finding: "Network-effect flywheel is theoretical: the adapter/capability model could compound, but 0 community-built adapters exist and the expansion vectors (templates marketplace, hosted Zynax Cloud) are all roadmap-CLAIMED."
    evidence:
      - "docs/product/strategy.md:296-297; ROADMAP.md:213,233"
green_flags:
  - strength: "Genuine, hard-to-copy technical moat that is partially PROVEN, not just claimed: one engine-neutral IR executes on two real, independent engines (Temporal + Argo) behind a clean port, validated end-to-end in CI."
    evidence:
      - "services/engine-adapter/internal/domain/engine.go:17-41"
      - "services/engine-adapter/internal/infrastructure/argo_engine.go:58-266 + temporal.go"
      - ".github/workflows/e2e-smoke.yml:56-61; scripts/e2e/e2e-argo.sh:178-266"
  - strength: "Strong integration moat via AgentService: a single no-SDK, no-framework, any-language gRPC contract turns any external system (including a Kagent agent) into a Zynax capability — this is the structural basis for both co-existence and an ecosystem flywheel."
    evidence:
      - "protos/zynax/v1/agent.proto:1-47 (ADR-013 adapter-first)"
  - strength: "Unusual marketing/reality discipline ('Truth Pass' culture): the strategy doc itself separates shipped/partial/aspirational and concedes its own commoditized surfaces and zero-traction baseline — a credibility asset rare in this category and directly de-risks the diligence drift concern."
    evidence:
      - "docs/product/strategy.md:113-130,218-229,287-291"
  - strength: "Timing is favorable: agent-framework proliferation (LangGraph/CrewAI/AutoGen) in 2024-26 creates real demand for a consolidation/control-plane layer; Zynax is early-but-not-late into that wave."
    evidence:
      - "E7: CNCF Cloud Native AI landscape + agent-orchestration emergence 2025-26"
      - "README.md:7-9,47-49"
open_questions:
  - "Would a buyer who has already standardized on Kagent (or a single engine like Temporal) actually pay the added-control-plane cost, or treat Zynax as redundant? No reference deployment answers this."
  - "Is multi-engine portability a top-3 buying criterion for any identifiable segment today, or a 'nice-to-have' that loses to single-engine simplicity until a second engine is forced on a team?"
  - "What is a defensible bottom-up SAM/SOM? No sizing artifact exists to anchor it."
unknowns:
  - "Wave A 5.1 findings file (docs/due-diligence/2026-06-20-dd-wave-a-findings.md) absent at HEAD — could not consume the Architecture portability packet directly; re-derived independently from engine-adapter source + CI and cited 5.1 as owner."
  - "TAM/SAM/SOM dollar figures — not derivable from repo; flagged E7 assumption-based, not asserted."
  - "Live GitHub star/fork/adopter counts at HEAD — strategy.md states a 0/0/0 baseline (May review); not re-pulled live here, so treated as CLAIMED-baseline."
cross_references:
  - to_agent: "5.1 Architecture"
    note: "Portability-is-partial is the market-load-bearing finding: Argo CI leg is happy-path-only vs Temporal's happy+failure (e2e-smoke.yml:147-173); ArgoEngine real at HEAD though 2026-06-18 review still labels it 'stubbed' (that doc lags). 5.1 owns the score; I consume it for the 'engine-agnostic' market thesis."
    evidence: [".github/workflows/e2e-smoke.yml:147-173", "services/engine-adapter/internal/infrastructure/argo_engine.go:58-266", "docs/architecture/2026-06-18-architecture-review.md:151"]
  - to_agent: "5.19 Competitive"
    note: "Kagent matrix is the primary competitive zone (owned by 5.19). I cite it for moat/timing; the 'complementary' drift-test result (PARTIAL = hedge) should reconcile with 5.19's Kagent deep-dive."
    evidence: ["docs/architecture/2026-05-28-competitive-positioning.md:24-83"]
  - to_agent: "5.3 Product"
    note: "Beachhead (agentic software-engineering automation) is the only segment with runnable proof (10 example workflows); my SAM/SOM 'premature market' flag depends on 5.3's beachhead-validation reality."
    evidence: ["spec/workflows/examples/ (code-review.yaml, feature-implementation.yaml, ci-pipeline.yaml)", "docs/product/strategy.md:237-262"]
  - to_agent: "5.18 Business Strategy / 5.17 Investment"
    note: "Expansion/monetization vectors (open-core + Zynax Cloud; CNCF-donated core + services) are roadmap-CLAIMED; D2 feeds D13/D15 — defer monetization scoring to them."
    evidence: ["docs/product/strategy.md:357-391", "ROADMAP.md:233"]
recommendations:
  - priority: "P0"
    action: "Produce one published reference deployment that demonstrates the SAME workflow YAML running unchanged on Temporal AND Argo (and ideally alongside a Kagent-registered capability) with traces — converting the partially-proven portability moat and the hedge-grade co-existence story into demonstrated, citable market proof."
    rationale: "The two narrowest moats (IR portability, AgentService integration) are real but asymmetrically/abstractly proven; a single reference deployment is worth more market-positioning than any feature, and directly answers the Kagent-buyer stress test."
  - priority: "P1"
    action: "Author a bottom-up SAM/SOM and an explicit ICP (which segment feels engine-lock-in pain TODAY), replacing the qualitative three-market framing; label assumptions E7."
    rationale: "The category is real but the buyer with acute portability pain is unidentified — without it, timing risk ('right but premature') cannot be retired and the TAM stays assumption-only."
  - priority: "P1"
    action: "Land the first community-built adapter and ≥1 named external adopter before the M8 CNCF filing, and instrument an adoption funnel (time-to-first-workflow) rather than feature counts."
    rationale: "For an OSS control plane the moat IS the ecosystem flywheel; today it is 0-node. Distribution, not design, is the binding constraint and the thing the CNCF-backed rival is already winning."
  - priority: "P2"
    action: "Tighten the Argo CI leg to parity with Temporal (add a failure-path assertion) and reconcile lagging docs (2026-06-18 review calling ArgoEngine 'stubbed') so the portability claim is symmetrically defensible under buyer scrutiny."
    rationale: "Partial proof + a doc that contradicts HEAD is exactly the delivery-vs-narrative drift the diligence exists to catch; closing it protects the project's hard-won integrity asset."
```

---

## (b) §6.2 prose

## Market Agent — Score: 6 (Medium)

**Mission recap:** Size and characterize the market; map comparables; assess timing, SWOT, moat, network effects, platform strategy, and expansion — and stress-test the "complementary to Kagent" story.

**Verdict:** Zynax is in a **real, timely, but contested and unsized** market. It is creating a coherent category — a declarative control plane *above* AI-agent orchestration and *across* durable engines — and it backs that category with two genuine, partially-proven moats (an engine-neutral IR that runs on two real engines, and a no-SDK `AgentService` integration contract). But the category is occupied by a CNCF-Sandbox-backed direct rival (Kagent) with the same tagline and a real ecosystem, while Zynax carries a zero-traction, single-maintainer baseline. The market thesis is sound; the *defensibility* rests on distribution Zynax does not yet have. Score sits at the top of "adequate / real but crowded-and-uncertain-timing."

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Category & TAM/SAM/SOM (E7 assumption-based) | 6 | Low | `docs/product/strategy.md:134-157`; CNCF AI landscape (E7) |
| Comparables matrix (OSS/commercial/emerging) | 6 | Medium | `2026-05-28-competitive-positioning.md:24-36`; `strategy.md:196-208` |
| Market timing (right, leaning early) | 7 | Medium | `README.md:7-9,47-49`; agent-framework wave 2024-26 (E7) |
| SWOT | 6 | Medium | `engine.go:17-41`; `strategy.md:62-64,337-338` |
| Moat & defensibility (pressure-tested) | 6 | Medium | `agent.proto:1-47`; `argo_engine.go:58-203`; `strategy.md:218-229` |
| Network effects / platform / expansion | 5 | Low | `ROADMAP.md:213,233`; `strategy.md:296-297` |

**Drift test:**
- *"Complementary to Kagent, not rivals"* → **PARTIAL (hedge more than co-existence).** The mechanism is real and shipped (`agent.proto:1-47` — a Kagent agent genuinely *can* register as a capability), but there is no integration test or reference deployment, and the project's own docs frame it as a way to avoid "a fight Zynax cannot yet win on ecosystem size" (`strategy.md:190-194`). It is a defensible mechanism deployed as a defensive narrative.
- *"Run it on Temporal OR Argo without a rewrite"* → **PARTIAL.** Both engines are real and both CI legs run (`e2e-smoke.yml:60-61`; `e2e-argo.sh:178-266` submits a real workflow and asserts the Argo CR reaches `Succeeded`), but the Argo leg is happy-path-only vs Temporal's happy+failure (`e2e-smoke.yml:147-173`). Portability is proven, not equally hardened — and a prior review still calls ArgoEngine "stubbed" (`2026-06-18-architecture-review.md:151`, lags HEAD).
- *"Defensible category"* → **PARTIAL.** Articulable and moated, but contested by a CNCF-backed rival holding the same framing (`strategy.md:57-64,152-157`).

**Red flags (severity-ordered):**
1. **High** — CNCF-Sandbox rival (Kagent) owns the identical framing while Zynax has 0 stars/forks/adopters and one maintainer; the rival leads on the only dimension the market rewards (distribution). `strategy.md:57-64,337-338`; `2026-05-28-competitive-positioning.md:35,57`.
2. **Medium** — "Complementary" is a hedge, not a demonstrated joint deployment. `2026-05-28-competitive-positioning.md:69-83`.
3. **Medium** — TAM is E7 assumption-only with no bottom-up sizing; portability pain not yet acute for single-engine buyers ("real but premature" risk). `strategy.md:134-157,240-242`.
4. **Low** — Network-effect flywheel is theoretical; 0 community adapters; expansion vectors all roadmap-CLAIMED. `strategy.md:296-297`; `ROADMAP.md:213,233`.

**Green flags:**
- Hard-to-copy, partially-PROVEN technical moat: one IR → two real engines behind a clean port, validated e2e in CI. `engine.go:17-41`; `argo_engine.go:58-266`; `e2e-smoke.yml:56-61`.
- Strong integration moat: no-SDK `AgentService` makes any system (incl. Kagent) a capability — basis for both co-existence and an ecosystem flywheel. `agent.proto:1-47`.
- Marketing-vs-reality discipline ("Truth Pass"): docs concede commoditized surfaces and zero traction — a rare credibility asset. `strategy.md:113-130,218-229`.
- Favorable timing: agent-framework proliferation (2024-26) creates genuine consolidation/control-plane demand. README.md:7-9; CNCF AI landscape (E7).

**Open questions / unknowns:**
- Would a Kagent- or single-engine-committed buyer pay the added control-plane cost? No reference answers it.
- Defensible bottom-up SAM/SOM and the ICP that feels lock-in pain *today* — both absent.
- Wave A 5.1 findings file absent at HEAD; portability-is-partial re-derived independently and cited to 5.1.

**Recommendations:**
- **P0** — Publish one reference deployment: same workflow YAML on Temporal AND Argo (ideally beside a Kagent-registered capability), with traces. Converts both moats and the co-existence hedge into citable market proof.
- **P1** — Author a bottom-up SAM/SOM + explicit ICP (who feels engine lock-in now); label assumptions E7.
- **P1** — Land first community adapter + ≥1 named adopter before M8 filing; instrument time-to-first-workflow.
- **P2** — Bring the Argo CI leg to parity (failure-path) and reconcile the lagging "stubbed" doc.

**Cross-references:** 5.1 Architecture (portability-is-partial, owns the score); 5.19 Competitive (Kagent matrix, reconcile co-existence drift result); 5.3 Product (beachhead reality underpins the "premature market" flag); 5.18/5.17 (expansion/monetization vectors → D13/D15).
<!-- SPDX-License-Identifier: Apache-2.0 -->
<!-- Zynax Investment-Grade Due-Diligence — Agent 5.8 Open Source — Wave C (product/market/governance) -->

# Agent 5.8 — Open Source / OSPO (Wave C) — Issue #1404

> Scope: license hygiene, governance maturity, bus factor, contribution funnel, issue/PR
> management, release cadence, transparency / Truth-Pass culture. Repository at HEAD
> (`main` @ `e3135a6`). READ-ONLY audit; no repo files modified.
> Evidence rule (§0.4): every factual claim carries `path:line` or a command+output, or is
> marked `UNKNOWN — not found`. Roadmap/marketing statements are labelled `CLAIMED`.
> Dimension group: **D7** (contributes D12/D15; feeds 5.21 CNCF).
> Wave A consumed (not re-scored, per §3.1): Repo Health 5.24 (bus factor / cadence / merge
> discipline) and Security 5.2 (SECURITY.md disclosure) — cited inline.

---

## (a) §3.4 Handoff packet

```yaml
agent: "5.8 Open Source / OSPO"
wave: "C"
dimension_groups: ["D7"]
overall_score: 6
overall_confidence: "High"

sub_scores:
  - dimension: "License hygiene (Apache-2.0 LICENSE file, NOTICE, SPDX headers, third-party compliance)"
    score: 4
    confidence: "High"
    justification: "Apache-2.0 is the chosen license and SPDX headers are pervasive, but the top-level LICENSE file DOES NOT EXIST (never in git history); README badge + License section link to a non-existent LICENSE; ADR-005's SPDX-enforcement and CLA claims are not implemented. No vendored third-party code, so NOTICE absence is lower-risk."
    evidence:
      - "cmd: ls LICENSE (repo root) → 'No existe el fichero' (NO top-level LICENSE file)"
      - "cmd: git ls-files | grep -iE 'license|notice|copying' → only 'docs/adr/ADR-005-apache-license.md' (no LICENSE, no NOTICE tracked)"
      - "cmd: git log --oneline --all -- LICENSE → (empty: LICENSE never existed in history)"
      - "README.md:13 — '[![License: Apache 2.0](...)](LICENSE)' badge links to missing file; README.md:527 — 'see [LICENSE](LICENSE)' broken link"
      - "docs/adr/ADR-005-apache-license.md:6 — 'License Zynax under Apache License 2.0' (decision recorded; the artifact it mandates is absent)"
      - "cmd: grep -rn 'SPDX-License-Identifier' .github/workflows/ Makefile → headers present across all 21 workflows + Makefile (SPDX header DISCIPLINE is real)"
      - "cmd: find . -type d -iname vendor -o -iname third_party → (empty: no vendored third-party source → NOTICE not strictly required, though LICENSE still is)"
  - dimension: "ADR-005 enforcement drift (license-eye / CLA claims vs reality)"
    score: 3
    confidence: "High"
    justification: "ADR-005 makes two enforcement claims that are CONTRADICTED at HEAD: SPDX headers 'enforced by license-eye in CI' (no such tool anywhere) and 'Contributors must sign CLA (GitHub CLA bot)' (project uses DCO, no CLA bot)."
    evidence:
      - "docs/adr/ADR-005-apache-license.md:16 — 'SPDX header required in every source file (enforced by license-eye in CI)'"
      - "cmd: grep -rniE 'license-eye|skywalking|reuse|\\.licenserc' .github/workflows Makefile tools scripts → no match (license-eye NOT wired; SPDX headers unenforced by tooling)"
      - "docs/adr/ADR-005-apache-license.md:17 — 'Contributors must sign CLA (automated via GitHub CLA bot)'"
      - "GOVERNANCE.md:152 — 'The DCO bot enforced in CI blocks merges without it' (project uses DCO, NOT CLA); cmd grep 'CLA|cla-assistant' → no CLA bot found"
      - "ci.yml:56-74 — 'dco' job 'Verify Signed-off-by on every commit' (DCO is the real gate)"
  - dimension: "Governance maturity (decision-making documented & followable; maintainer roles)"
    score: 8
    confidence: "High"
    justification: "GOVERNANCE.md is best-in-class for a solo project: explicit role ladder, an honest 'Solo Maintainer Phase' decision matrix, RFC process, triage SLAs, conflict resolution, CNCF alignment checklist. Only blemish: 4 broken links to a non-existent MAINTAINERS.md."
    evidence:
      - "GOVERNANCE.md:27-94 — Contributor/Reviewer/Maintainer/Emeritus roles with how-to-become + rights + responsibilities"
      - "GOVERNANCE.md:104-118 — 'Solo Maintainer Phase (current)' explicitly documents self-merge-on-green for non-breaking + 5-day RFC comment for breaking changes (honest, followable)"
      - "GOVERNANCE.md:120-137 — Multi-Maintainer decision matrix + supermajority definition"
      - "GOVERNANCE.md:434-447 — CNCF alignment checklist honestly leaves '≥2 maintainers from ≥2 orgs' UNCHECKED (line 441)"
      - "GOVERNANCE.md:86,93,396,405 — references [MAINTAINERS.md] 4× — file does NOT exist (broken links; #494 still open per 5.24)"
  - dimension: "Bus factor (distinct human committers)"
    score: 3
    confidence: "High"
    justification: "Bus factor = 1. One human identity (two emails = 772 commits) accounts for 100% of non-bot work; no second human author. Single point of failure; unmet CNCF >=2-maintainers gate. (Consumes 5.24; not re-scoring their zone — recorded here for the D7 governance lens.)"
    evidence:
      - "cmd: git shortlog -sne HEAD → 769 Oscar Gómez + 3 Oscar Gómez Manresa = 772 human; 45 github-actions[bot]; 7 renovate[bot] (one human identity)"
      - "5.24 (Wave A): 'Bus factor = 1 ... no second human author in the entire shortlog or last-50 window' — VERIFIED"
      - "docs/product/strategy.md:82 — self-acknowledged 'single maintainer' (honest)"
      - ".github/CODEOWNERS:5-29 — every path → @zynax-io/maintainers team (effectively one person; no MAINTAINERS.md to enumerate it)"
  - dimension: "Contribution funnel (CONTRIBUTING approachable vs Day-0 wall)"
    score: 7
    confidence: "High"
    justification: "CONTRIBUTING is thorough and genuinely approachable — clear prerequisites table, one-command bootstrap, channel routing, 2-business-day SLA. The Day-0 friction is the heavy AGENTS.md/DCO/SPDD reading load demanded before a first PR, not the doc quality."
    evidence:
      - "CONTRIBUTING.md:13-24 — 'Before You Start' (read AGENTS.md, git-workflow, ADRs, open issue first, sign DCO)"
      - "CONTRIBUTING.md:48-66 — prerequisites table (Go/Python/uv/Docker/buf) + 'make bootstrap' one-command setup"
      - "CONTRIBUTING.md:28-42 — community channels + '2 business days' response SLA"
      - "CONTRIBUTING.md:15-16 — 'Read AGENTS.md — the full engineering contract. Required for all contributors' (substantial Day-0 reading load = friction)"
      - "docs/product/strategy.md:82 — 'Day-0 friction' named as a top adoption blocker (CLAIMED, self-aware)"
  - dimension: "Issue/PR management (templates, triage labels, responsiveness)"
    score: 8
    confidence: "High"
    justification: "7 structured issue templates + a detailed PR template + a config.yml that routes questions to Discussions, security to private advisories, CoC to email. Documented triage steps + priority defs + stale-bot policy. 94% issue-closure ratio (5.24)."
    evidence:
      - "cmd: ls .github/ISSUE_TEMPLATE/ → adr_proposal, bug_report, documentation, epic, feature_request, image_bump + config.yml (7 templates)"
      - ".github/ISSUE_TEMPLATE/config.yml:1-20 — blank_issues_enabled:false; routes Q&A→Discussions, security→advisories, CoC→email"
      - ".github/PULL_REQUEST_TEMPLATE.md (8123 bytes — detailed checklist)"
      - "GOVERNANCE.md:211-238 — 7-step triage process + priority definitions + 90-day stale-bot policy"
      - "5.24 (Wave A): 39 open / 632 closed = 94.2% closure; 0 open/orphaned PRs"
  - dimension: "Release cadence + tags (SemVer, signed tags, GitHub Releases, SECURITY disclosure)"
    score: 7
    confidence: "High"
    justification: "Real SemVer releases (v0.4.0, v0.5.0) with corresponding GitHub Releases carrying SBOM/cosign artifacts; documented milestone→version map + checklist; private security disclosure via GitHub Advisories with SLAs. Two narrative blemishes: GOVERNANCE milestone-version table drift vs CLAUDE.md, and CHANGELOG [Unreleased] lag (5.24)."
    evidence:
      - "cmd: git tag → v0.4.0, v0.4.0-verify-supply-chain, v0.5.0, v0.5.0-snapshot.1 (+ proto-stubs tags)"
      - "cmd: gh release list → 'v0.5.0: K8s Production-Ready' (2026-06-12), 'Zynax v0.4.0' (2026-05-28) — real GitHub Releases"
      - "SECURITY.md:14-26 — private disclosure via GitHub Security Advisories + 48h ack / 5-day assessment / severity-based fix SLAs"
      - "GOVERNANCE.md:242-290 — release process, milestone→version map, signed-tag checklist (git tag -s)"
      - "GOVERNANCE.md:251-258 — milestone table (M5=Observability/v0.5.0, M6=v1.0.0-rc.1) DRIFTS vs CLAUDE.md (M5=capability dispatch/v0.4.0, M6=K8s/v0.5.0, M7=observability)"
  - dimension: "Transparency (ADRs/RFCs public, honest status, Truth-Pass culture)"
    score: 8
    confidence: "High"
    justification: "Strong transparency culture: 39 public ADRs, a documented Truth-Pass that purged phantom claims, and a strategy doc that states 'zero stars/forks/adopters, single maintainer' to its own face. Caveat: the elaborate RFC process has 0 actual RFCs (template only)."
    evidence:
      - "cmd: ls docs/adr/*.md | wc -l → 39 public ADRs"
      - "cmd: ls docs/rfcs/ → RFC-000-template.md ONLY (0 RFCs filed despite GOVERNANCE.md:180-207 process)"
      - "docs/product/strategy.md:290 — '0 stars, 0 forks, 0 external adopters.' (self-stated, matches live signals)"
      - "docs/product/strategy.md:338 — '≥1 external adopter: ❌ (0 stars/forks baseline)' (honest CNCF-readiness self-assessment)"
      - "5.24 (Wave A): CHANGELOG.md:44 — prior '#473 CHANGELOG phantom entries removed' = documented Truth-Pass already executed"

drift_test:
  - claim: "Apache-2.0 licensed — LICENSE file present (§1.9; ADR-005; GOVERNANCE.md:442 'Apache 2.0 license (done)')"
    result: "CONTRADICTED"
    evidence:
      - "cmd: ls LICENSE → 'No existe el fichero'; git log --all -- LICENSE → empty (never existed)"
      - "README.md:13,527 — license badge + License section both link to a non-existent LICENSE file (broken)"
      - "GOVERNANCE.md:442 — 'Apache 2.0 license (done)' is OVERSTATED: the license is decided (ADR-005) and headers exist, but the legal LICENSE artifact is absent"
  - claim: "SPDX headers enforced by license-eye in CI; contributors sign a CLA (ADR-005:16-17)"
    result: "CONTRADICTED"
    evidence:
      - "cmd: grep -rniE 'license-eye|reuse|\\.licenserc' .github/workflows Makefile tools → no match (no SPDX-header CI gate)"
      - "GOVERNANCE.md:152 + ci.yml:56-74 — project enforces DCO, not a CLA; no CLA bot exists"
  - claim: "Healthy / growing community (implied by CNCF-aspiration framing)"
    result: "UNKNOWN (no false claim made — honest)"
    evidence:
      - "cmd: gh repo view zynax-io/zynax → stargazerCount:1, forkCount:0, watchers:0"
      - "docs/product/strategy.md:62,290,338 — repo's OWN docs state 'zero stars, zero forks, zero external adopters, single maintainer' — NO community-size drift; claims match reality (the desired Truth-Pass outcome)"
  - claim: "Documented, followable decision-making + maintainer governance (§1.9)"
    result: "VERIFIED"
    evidence:
      - "GOVERNANCE.md:98-137 — lazy-consensus + explicit Solo-Maintainer-Phase decision matrix"
      - "GOVERNANCE.md:27-94 — role ladder; GOVERNANCE.md:211-238 — triage SLAs"

red_flags:
  - severity: "High"
    finding: "No top-level LICENSE file exists (never has in git history) for a project whose entire positioning is Apache-2.0 open-source and CNCF-Sandbox-bound. The README license badge and License section both link to a missing LICENSE; ADR-005 records the decision but the legally operative artifact is absent. Apache-2.0 §4 requires the license text to accompany distribution; GOVERNANCE.md:442 calling the license 'done' is overstated. This is a one-line fix but a hard CNCF blocker and a genuine legal-hygiene gap (no NOTICE either, though lower-risk: no vendored third-party source)."
    evidence:
      - "cmd: ls LICENSE → 'No existe el fichero'; git log --all -- LICENSE → empty"
      - "README.md:13,527 (broken LICENSE links)"
      - "GOVERNANCE.md:442 ('Apache 2.0 license (done)' overstated)"
  - severity: "High"
    finding: "Bus factor = 1 (consumed from 5.24, recorded for the governance lens). One human identity = 772 commits; zero second human author; no MAINTAINERS.md (#494 open). Governance docs are written for a multi-maintainer org that does not yet exist — CODEOWNERS routes everything to a one-person team, and 4 GOVERNANCE.md links to MAINTAINERS.md are broken. This is the explicit CNCF social gate and the project's own #1 sustainability risk."
    evidence:
      - "cmd: git shortlog -sne HEAD → one human identity (772 commits), rest bots"
      - "GOVERNANCE.md:86,93,396,405 (broken MAINTAINERS.md links); GOVERNANCE.md:441 (CNCF ≥2-maintainer gate UNCHECKED)"
      - "5.24 (Wave A) red flag #1: bus factor = 1; #494 MAINTAINERS.md still open"
  - severity: "Medium"
    finding: "ADR-005 enforcement drift: it claims SPDX headers are 'enforced by license-eye in CI' (no such tool wired anywhere) and that contributors sign a 'CLA via GitHub CLA bot' (the project actually uses DCO; no CLA exists). An Accepted ADR describing controls that were never implemented is exactly the delivery-vs-narrative drift class the diligence targets."
    evidence:
      - "docs/adr/ADR-005-apache-license.md:16-17"
      - "cmd: grep -rniE 'license-eye|reuse' .github/workflows Makefile → none; ci.yml:56-74 (DCO, not CLA)"
  - severity: "Low"
    finding: "Process-vs-practice gap: an elaborate RFC process (GOVERNANCE.md:180-207, mandatory for any proto/architecture change) has produced 0 RFCs — only the template exists. With 39 ADRs filed, decisions are recorded as ADRs not RFCs; the RFC machinery is aspirational. GOVERNANCE milestone→version table (M5=Observability) also drifts vs CLAUDE.md (M5=capability dispatch; M7=observability)."
    evidence:
      - "cmd: ls docs/rfcs/ → RFC-000-template.md only"
      - "GOVERNANCE.md:251-258 vs CLAUDE.md milestone table"

green_flags:
  - strength: "Exemplary transparency / Truth-Pass culture: the project's own strategy doc states 'zero stars, zero forks, zero external adopters, single maintainer' and marks CNCF prerequisites ❌ to its own face — no community-size inflation anywhere. Live signals (1 star, 0 forks) match the docs. This is the rare case where self-reported status is more pessimistic than reality."
    evidence:
      - "docs/product/strategy.md:62,290,338"
      - "cmd: gh repo view → stargazerCount:1, forkCount:0; CHANGELOG.md:44 (prior phantom-entry purge)"
  - strength: "Best-in-class governance documentation for a solo project: explicit role ladder, an honest 'Solo Maintainer Phase' decision matrix (self-merge rules stated openly), RFC process, triage SLAs, conflict resolution, and a CNCF-alignment checklist that honestly tracks what's unmet."
    evidence:
      - "GOVERNANCE.md:27-137 (roles + decision-making)"
      - "GOVERNANCE.md:104-118 (Solo Maintainer Phase); GOVERNANCE.md:434-447 (CNCF checklist)"
  - strength: "Mature issue/PR funnel: 7 structured issue templates, an 8KB PR checklist, config.yml routing questions→Discussions / security→private advisories / CoC→email, documented triage + stale-bot policy, and a 94% issue-closure ratio with 0 orphaned PRs (5.24)."
    evidence:
      - ".github/ISSUE_TEMPLATE/config.yml:1-20"
      - "GOVERNANCE.md:211-238; 5.24 closure 632/671"
  - strength: "Real, SemVer'd, signed releases with supply-chain artifacts: v0.4.0 + v0.5.0 git tags AND GitHub Releases; SECURITY.md private-disclosure policy with explicit SLAs; SBOM/cosign attached per release (5.2)."
    evidence:
      - "cmd: git tag → v0.4.0, v0.5.0; gh release list → matching Releases"
      - "SECURITY.md:14-26,42-43; GOVERNANCE.md:242-290"
  - strength: "Pervasive SPDX-header discipline (every workflow + Makefile + source carries 'SPDX-License-Identifier: Apache-2.0'), so the license INTENT is unambiguous even with the LICENSE file missing — the gap is the artifact, not the discipline."
    evidence:
      - "cmd: grep -rn 'SPDX-License-Identifier' .github/workflows Makefile → 21 workflows + Makefile"

open_questions:
  - "Is the missing LICENSE file an oversight or a deliberate omission? Apache-2.0 §4 requires the text to accompany distribution — a published PyPI SDK / GHCR images without it could be a distribution-compliance gap. (Cannot inspect published artifact contents read-only.)"
  - "Who is the designated second maintainer / successor? MAINTAINERS.md is referenced 4× but does not exist (#494 open) — the entire multi-maintainer governance is currently unbacked by people."
  - "Will the RFC process ever be exercised, or is it dead machinery that ADRs have de facto replaced?"

unknowns:
  - "Whether published GHCR images / the PyPI zynax-sdk wheel bundle a LICENSE internally (separate from the repo root) — not inspectable read-only/offline; repo-root LICENSE is definitively absent."
  - "Whether branch protection actually enforces required reviews/signatures server-side (5.24 also flagged this UNKNOWN); CODEOWNERS exists but its enforcement is server-side config."
  - "GitHub repo-level Discussions activity / first-response latency in practice — SLA is documented (2 business days) but actual responsiveness is not measurable read-only with a single-author repo."

cross_references:
  - to_agent: "5.21 CNCF"
    note: "D7 feeds CNCF directly: missing LICENSE file, bus factor 1, no MAINTAINERS.md (#494), and 'zero external adopters' are all hard Sandbox prerequisites unmet. Governance docs are CNCF-shaped and honest, so the gap is people + the LICENSE artifact, not policy."
    evidence: ["cmd: ls LICENSE → absent", "GOVERNANCE.md:434-447", "docs/product/strategy.md:338"]
  - to_agent: "5.24 Repo Health"
    note: "Consumed bus factor / 94% closure / squash-only discipline; not re-scored. Add: governance docs presume a multi-maintainer org that doesn't exist (broken MAINTAINERS.md links)."
    evidence: ["git shortlog -sne", "GOVERNANCE.md:86"]
  - to_agent: "5.10 Documentation"
    note: "License/ADR drift (ADR-005 license-eye+CLA claims; GOVERNANCE.md:442 'license done'; milestone-table drift vs CLAUDE.md) is ADR-vs-reality reconciliation debt — same drift class 5.10 tracks."
    evidence: ["docs/adr/ADR-005-apache-license.md:16-17", "GOVERNANCE.md:251-258"]
  - to_agent: "5.2 Security"
    note: "SECURITY.md private-disclosure policy (GitHub Advisories + SLAs) is a strong OSS-governance signal corroborating the supply-chain posture 5.2 scored."
    evidence: ["SECURITY.md:14-26"]

recommendations:
  - priority: "P0"
    action: "Add a top-level LICENSE file containing the full Apache-2.0 text (and a NOTICE if/when third-party source is bundled). Fix the README badge/section links. Reconcile GOVERNANCE.md:442 wording."
    rationale: "Hard CNCF blocker + Apache-2.0 distribution-compliance gap; a one-line fix for the project's legal cornerstone whose absence undercuts the entire OSS positioning."
  - priority: "P0"
    action: "Recruit + document a second maintainer (create MAINTAINERS.md, close #494) to break bus factor = 1 before any CNCF Sandbox filing."
    rationale: "The explicit CNCF social gate and the project's own #1 sustainability risk; governance docs already presume this org structure."
  - priority: "P1"
    action: "Reconcile ADR-005: either implement an SPDX-header CI gate (license-eye/REUSE) or rewrite the ADR to describe the real DCO-based controls; drop the non-existent CLA claim."
    rationale: "Eliminates an Accepted-ADR-vs-implementation drift in the exact class this diligence exists to catch."
  - priority: "P2"
    action: "Either exercise the RFC process for the next breaking change or fold it into the ADR flow; fix the GOVERNANCE milestone→version table drift vs CLAUDE.md."
    rationale: "Removes dead governance machinery and a status-surface inconsistency; cheap transparency hygiene."
```

---

## (b) §6.2 Prose section

## 5.8 Open Source / OSPO — Score: 6 (High)

**Mission recap:** Assess license, governance, contribution model, bus factor, community health, onboarding, issue management, release cadence, transparency, and decision-making — and run the drift test on any "community" claims vs actual stars/forks/contributors.

**Verdict:** Zynax has the *governance and transparency posture of a project far more mature than its actual community*, paired with one embarrassing legal-hygiene gap and the same single-maintainer ceiling that gates everything social. GOVERNANCE.md, CONTRIBUTING.md, the issue/PR template set, and the SECURITY disclosure policy are all best-in-class for a solo CNCF-aspirant, and the transparency is genuinely exemplary — the project's own docs state "zero stars, zero forks, zero external adopters" rather than inflating a community. The two material defects: there is **no top-level LICENSE file** (never has been in git history) despite Apache-2.0 being the whole positioning and despite README/GOVERNANCE asserting the license is "done"; and **bus factor = 1**, with elaborate multi-maintainer governance written for an org that does not yet exist.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| License hygiene (LICENSE/NOTICE/SPDX/third-party) | 4 | High | `ls LICENSE`→absent; `git log --all -- LICENSE`→empty; `README.md:13,527`; SPDX headers present |
| ADR-005 enforcement drift (license-eye/CLA) | 3 | High | `ADR-005:16-17` vs grep (no license-eye); `ci.yml:56-74` (DCO not CLA) |
| Governance maturity (roles/decision-making) | 8 | High | `GOVERNANCE.md:27-137,434-447`; broken MAINTAINERS.md links `:86` |
| Bus factor (consumed 5.24) | 3 | High | `git shortlog -sne`→1 human (772); `#494` open |
| Contribution funnel (CONTRIBUTING) | 7 | High | `CONTRIBUTING.md:13-66`; AGENTS.md reading load = Day-0 friction |
| Issue/PR management (templates/triage) | 8 | High | `ISSUE_TEMPLATE/config.yml`; `GOVERNANCE.md:211-238`; 94% closure (5.24) |
| Release cadence + tags | 7 | High | `git tag`→v0.4.0/v0.5.0; `gh release list`; `SECURITY.md:14-26` |
| Transparency / Truth-Pass culture | 8 | High | 39 ADRs; `strategy.md:290,338`; 0 RFCs (`ls docs/rfcs/`) |

**Drift test:**
- *"Apache-2.0 — LICENSE file present (§1.9; GOVERNANCE.md:442 'license done')"* → **CONTRADICTED.** No LICENSE file exists or ever existed (`ls LICENSE`→absent; `git log --all -- LICENSE`→empty); README badge/section link to it (`README.md:13,527`).
- *"SPDX headers enforced by license-eye; contributors sign a CLA (ADR-005:16-17)"* → **CONTRADICTED.** No license-eye/REUSE gate anywhere; the real gate is DCO (`ci.yml:56-74`), no CLA bot.
- *"Healthy/growing community"* → **UNKNOWN / no false claim made.** `gh repo view`→1 star, 0 forks, 0 watchers; the repo's own docs state "zero stars/forks/adopters" (`strategy.md:62,290,338`) — *no community-size drift; claims match reality.* This is the desired Truth-Pass outcome.
- *"Documented, followable governance (§1.9)"* → **VERIFIED** (`GOVERNANCE.md:98-137,27-94`).

**Red flags (severity-ordered):**
1. **High** — No top-level LICENSE file for an Apache-2.0/CNCF-bound project; README links broken; GOVERNANCE.md:442 overstates it as "done" (`ls LICENSE`→absent, `README.md:13,527`).
2. **High** — Bus factor = 1; multi-maintainer governance + CODEOWNERS presume an org that doesn't exist; no MAINTAINERS.md (#494) (`git shortlog -sne`, `GOVERNANCE.md:86`).
3. **Medium** — ADR-005 claims license-eye SPDX enforcement + a CLA bot; neither exists (`ADR-005:16-17`, `ci.yml:56-74`).
4. **Low** — RFC process exists but 0 RFCs filed; GOVERNANCE milestone→version table drifts vs CLAUDE.md (`ls docs/rfcs/`, `GOVERNANCE.md:251-258`).

**Green flags:**
- Exemplary Truth-Pass culture — self-reported status is *more* pessimistic than reality; live signals match docs (`strategy.md:290`, `gh repo view`).
- Best-in-class solo-project governance: honest Solo-Maintainer-Phase matrix + CNCF checklist (`GOVERNANCE.md:104-118,434-447`).
- Mature issue/PR funnel: 7 templates, routing config, 94% closure, 0 orphaned PRs (`config.yml`, 5.24).
- Real SemVer signed releases + private security disclosure (`git tag`, `gh release list`, `SECURITY.md:14-26`).
- Pervasive SPDX-header discipline (21 workflows + Makefile + source) — license intent unambiguous even sans LICENSE file.

**Open questions / unknowns:** Is the missing LICENSE an oversight or deliberate (and do published GHCR/PyPI artifacts bundle one)? Who is the designated 2nd maintainer? Will the RFC process ever run? Server-side branch-protection enforcement is UNKNOWN read-only.

**Recommendations:** P0 — add a real LICENSE file + fix README/GOVERNANCE wording; P0 — recruit + document a 2nd maintainer (close #494); P1 — reconcile ADR-005 to the real DCO controls (drop the CLA/license-eye claims); P2 — exercise-or-fold the RFC process and fix the milestone-table drift.

**Cross-references:** 5.21 CNCF (missing LICENSE + bus factor + 0 adopters = unmet Sandbox gates; feeds 5.21 directly); 5.24 Repo Health (consumed bus factor/closure/merge discipline, not re-scored); 5.10 Documentation (ADR-005 + GOVERNANCE drift = reconciliation debt); 5.2 Security (SECURITY.md disclosure corroborates supply-chain posture).
<!-- SPDX-License-Identifier: Apache-2.0 -->
<!-- Agent 5.11 — Developer Experience · Wave C (product/market/governance) · GitHub issue #1404 -->
<!-- Target repo: the repository root @ HEAD e3135a6 (main) · READ-ONLY audit -->
<!-- Framework: docs/due-diligence/2026-06-18-zynax-due-diligence-framework.md -->
<!-- Evidence rule §0.4: every claim carries path:line / command-output, or is UNKNOWN. Roadmap/marketing = CLAIMED. -->

# Agent 5.11 — Developer Experience — §3.4 Handoff Packet

```yaml
agent: "5.11 Developer Experience"
wave: "C"
dimension_groups: ["D1", "D7", "D8"]   # D1 primary; contributes D7 (quality enablement) / D8 (DevOps inner loop)
overall_score: 7
overall_confidence: "High"

sub_scores:
  - dimension: "Time-to-first-success (clone → first runnable workflow): step count & footguns"
    score: 6
    confidence: "High"
    justification: "Real zero-secret happy path exists and every step maps to a real target, but the headline 'one command' (make demo) is actually a 4-prereq chain — CLI must be built & on PATH, host ollama + model pulled — and make demo only CHECKS for the CLI, it does not install it."
    evidence:
      - "README.md:24-33 — 'See it run — one command: make demo' (markets a single command)"
      - "Makefile:162-164 — demo target: `command -v $(ZYNAX) ... || (echo '❌ zynax CLI not found — run make install-cli ...' && exit 1)` — demo FAILS fast if CLI absent; it does not install it"
      - "Makefile:166-175 — demo needs host `ollama` on PATH; if present it pulls the model, else warns the codereview will 404"
      - "docs/quickstart.md:24-33,37-51,55-72 — manual path is 8 steps: clone→bootstrap→install-cli→version-check→ollama pull→compose up→export ZYNAX_API_URL→apply"
      - "docs/developer-guide.md:13-18 — one-time setup is explicitly TWO commands (make bootstrap + make install-cli), confirming demo is not standalone"
      - "Makefile:211-213 — install-cli builds with local Go ('requires Go 1.26.3') → contradicts the quickstart's 'nothing else needs Go' (quickstart.md:32-33); release-binary path (docs/local-dev.md:9-18) avoids Go but is a separate route"
  - dimension: "Local dev loop (compose up / logs / reset / inner-loop speed / Docker-only friction)"
    score: 8
    confidence: "High"
    justification: "Complete, well-aliased compose lifecycle (up/down/logs/ps/reset/restart-one), a guarded destructive reset, and a single-service rebuild target; Docker-only model removes toolchain drift but every lint/test shells into a container, taxing the inner loop."
    evidence:
      - "Makefile:85-99 — dev-up/dev-down/dev-logs/dev-ps/dev-reset/dev-restart full lifecycle; dev-reset:94-96 prompts before deleting volumes (footgun guard)"
      - "Makefile:97-99 — dev-restart SVC=<name> rebuilds one service (fast inner loop for a single change)"
      - "Makefile:115-120,122-126 — run-local/stop-local/logs-local print the port map (7080 gateway, 7088 Temporal UI)"
      - "Makefile:35-37,234-238 — TOOLS_RUN docker-run wrapper; every lint-go target execs into ghcr.io/zynax-io/zynax/tools per service in a for-loop (per-service container spin-up = slow inner loop)"
      - "Makefile:207-209 — demo-clean tears down base+overlay volumes in one target"
  - dimension: "Tooling ergonomics (make help quality / error messages / GOWORK=off trap / pre-commit)"
    score: 8
    confidence: "High"
    justification: "make help is self-documenting with 72 annotated targets and ★-marked entry points; error messages are actionable with emoji+remedy; GOWORK=off is consistently documented in 4 surfaces; pre-commit is auto-wired by bootstrap with an honest managed-vs-system split."
    evidence:
      - "Makefile:40-42 — help target greps `## ` annotations into a formatted list; cmd: `grep -cE '^[a-zA-Z_-]+:.*?## ' Makefile` → 72 documented targets"
      - "Makefile:45,115,162,227,264 — ★-prefixed help text flags the canonical entry points (bootstrap/run-local/demo/ci/test)"
      - "Makefile:54,62,98,164,524 — actionable error strings with the fix inline (e.g. '❌ Usage: make dry-run FILE=<path>')"
      - "CLAUDE.md:68; docs/local-dev.md:21,90; CONTRIBUTING.md:136; docs/developer-guide.md:124 — GOWORK=off trap documented in 4 surfaces with ADR-017 rationale"
      - ".pre-commit-config.yaml:1-11,16-54 — managed hooks (gitleaks/ruff auto-download) vs system hooks (gofmt/golangci-lint/mypy need local toolchain) split documented; CONTRIBUTING.md:139-171 mirrors it"
      - "Makefile:45-51 — bootstrap installs pre-commit hooks and warns (not fails) if pre-commit binary missing"
  - dimension: "Contributor friction (SPDD/canvas: quality moat vs casual-contributor wall)"
    score: 5
    confidence: "High"
    justification: "CONTRIBUTING is thorough but long (515 lines, 15 sections) and the SPDD canvas-before-code mandate is a real Day-0 wall for feat: PRs; mitigated by good-first-issue funnel and the fact that fix/docs/chore are SPDD-exempt — so the wall is scoped, not total."
    evidence:
      - "cmd: `wc -l CONTRIBUTING.md` → 515 lines, 15 numbered sections (long onboarding read)"
      - "CLAUDE.md:« SPDD »:'Every feat: PR requires a REASONS Canvas committed before any implementation code' — Day-0 gate for any new feature"
      - "CONTRIBUTING.md:9 — 'Read this document before opening your first PR' + CONTRIBUTING.md:13-24 four pre-reqs (AGENTS.md, git-workflow.md, ADRs, open-issue-first)"
      - "CONTRIBUTING.md:481-507 §15 First Contribution — good-first-issue funnel, 'You do not need to know the whole codebase' (lowers the wall)"
      - "Wave A 5.12 (final.md:1628-1631) — canvas gate is SOFT: pr-checks.yml:231-233 passes if ANY canvas exists; so for a casual contributor the canvas is more cultural-expectation than hard CI block"
      - "CLAUDE.md:« SPDD »:'Scope: feat: PRs only — fix:, refactor:, docs:, ci:, chore: are exempt' — most first PRs avoid the canvas entirely"
  - dimension: "CLI UX (cmd/zynax — discoverability / help / error handling)"
    score: 8
    confidence: "High"
    justification: "Cobra CLI with a complete, consistently-described verb set, source-of-record cross-links, dry-run/validate previews, meaningful exit codes, and structured errors with line numbers; weak spots are minor (no friendly 'is the stack up?' hint on connection-refused)."
    evidence:
      - "cmd: `grep -rn 'Short:' cmd/zynax/cmd/` → 21 commands/subcommands each with a clear one-line Short (apply/validate/init/status/logs/result/get/delete/events/mcp/gitops)"
      - "cmd/zynax/cmd/root.go:18-23,37-44 — SilenceUsage:true (errors don't dump usage spam); ZYNAX_API_URL env default; --insecure flag"
      - "cmd/zynax/cmd/apply.go:44-48,121-137 — --dry-run path prints compiler errors as 'error (line N): [CODE] message' (structured, actionable)"
      - "cmd/zynax/cmd/status.go:26 — 'Check workflow status (exits 0 if terminal, 2 if still running)' — script-friendly exit codes"
      - "cmd/zynax/client/gateway.go:100-103,132,159 — HTTP errors surfaced as 'zynax: apply: HTTP %d: %s' with server body; 21 ErrNotFound 404 mapping"
      - "GAP: cmd/zynax/client/gateway.go:330-332 — connection-refused returns raw 'zynax: post /api/v1/apply: <net error>'; no hint to run `make run-local` / check ZYNAX_API_URL (default 8080 vs stack's mapped 7080 is a known first-run trip — root.go:40 vs quickstart.md:94-99)"
  - dimension: "Documentation-as-DX (onboarding path coherence; consume 5.10)"
    score: 7
    confidence: "High"
    justification: "Layered, source-of-record-linked onboarding (README→quickstart→dev-guide→faq→local-dev) that 5.10 scored 8/High on onboarding; the front-door README under-claims via a stale 'M5 status note', which actively misleads a first-time DX evaluator into thinking dispatch is unfinished."
    evidence:
      - "Wave A 5.10 (final.md:1380-1388) — 'Onboarding from docs alone' scored 8/High; clear clone→demo→quickstart→authoring path"
      - "README.md:333-337,355 — stale 'M5 status note' tells users 'capability dispatch pending M5.C' / 'register an adapter first' — contradicts the same file's Service Status table (Wave A 5.10 final.md:1417-1423)"
      - "docs/faq.md:13-33 — concise getting-started Q&A; docs/quickstart.md:237-254 — command reference table cross-links cmd/zynax/cmd/ as source-of-record"
      - "docs/local-dev.md:60-117 — persona paths (Go/Python/proto contributor) give role-scoped command loops"

drift_test:
  - claim: "make demo — ONE command boots a zero-secret stack and runs the hero workflow end-to-end (README.md:24-33)"
    result: "PARTIAL"
    evidence:
      - "WIRING VERIFIED: Makefile:162-205 boots COMPOSE_DEMO Ollama overlay, applies DEMO_TARGET, prints zynax result (corroborates Wave A 5.10 final.md:1409-1414)"
      - "CONTRADICTED as 'one command': Makefile:163-164 demo EXITS if zynax CLI not on PATH (does not auto-install); Makefile:166-175 requires host ollama + a model pull; true first-run chain = bootstrap + install-cli + PATH + ollama pull + demo (docs/developer-guide.md:13-18 confirms 2-command setup)"
      - "RUNTIME end-to-end NOT executed here (no Docker/Ollama in audit env); E2/E3-verified, not E1 — same boundary Wave A 5.10 hit (final.md:1455)"
  - claim: "Time-to-first-working-workflow < 15 min, one command (docs/product/strategy.md:294)"
    result: "PARTIAL"   # CLAIMED target; tracing the documented steps it is plausible but unproven and gated on image pull + model pull
    evidence:
      - "docs/product/strategy.md:254,294 — '< 15 min, one command' is a STRATEGY/marketing target (CLAIMED, E5), not an instrumented measurement; no committed timing artifact found"
      - "docs/quickstart.md:7 — softer doc claim is 'in minutes' (not 15); step trace = 8 documented steps (quickstart.md §1-§5)"
      - "First-run wall-clock is dominated by uncontrolled downloads: GHCR tools image pull (make bootstrap), compose --build of platform images, and `ollama pull qwen2.5-coder:3b` (Makefile:131,170-171 note 'can take a few minutes') — none of which the 15-min figure accounts for; UNKNOWN whether a cold clone hits 15 min"
  - claim: "Docker-only — 'nothing else needs Go, Python, or buf installed locally' (quickstart.md:32-33)"
    result: "PARTIAL"
    evidence:
      - "VERIFIED for lint/test/generate: Makefile:35-37,234-238,381-382 run everything via the tools container (TOOLS_RUN)"
      - "CONTRADICTED for the CLI: make install-cli compiles with a LOCAL Go toolchain (Makefile:211-213 'requires Go 1.26.3'; quickstart.md:39-40 'requires the Go toolchain pinned in go.work'); the Docker-only promise holds only if you take the release-binary route (docs/local-dev.md:9-18), which the quickstart mentions secondarily"

red_flags:
  - severity: "Medium"
    finding: "The boldest DX claim — 'make demo, one command' (README front door) — is a multi-prerequisite chain in reality: the CLI must be built and on PATH (demo only checks, never installs), and the host needs ollama + a pulled model, or the run 404s. A first-time evaluator running the marketed one command on a clean machine hits an immediate `❌ zynax CLI not found` exit. This is the delivery-vs-narrative drift class §1.10 exists to catch, here at the highest-traffic surface."
    evidence:
      - "README.md:24-33 (one-command marketing)"
      - "Makefile:162-164 (demo exits, does not install, when CLI absent)"
      - "Makefile:166-175 (host ollama + model-pull prerequisite)"
      - "docs/developer-guide.md:13-18 (setup is documented elsewhere as two commands)"
  - severity: "Medium"
    finding: "README Quickstart carries a stale 'M5 status note' (capability dispatch 'pending M5.C', 'register an adapter first', 'capability dispatch pending M5.C' in the logs comment) that UNDER-claims shipped capability and contradicts the same file's Service Status table — the first thing a DX evaluator reads top-down tells them the product is less finished than it is. (Consumed from Wave A 5.10; re-flagged as a DX/onboarding harm, not re-scored.)"
    evidence:
      - "README.md:333-337,355"
      - "Wave A 5.10 (final.md:1417-1423) — same stale note, scored as Medium doc red flag"
  - severity: "Medium"
    finding: "Contributor onboarding is heavy for a casual external contributor: a 515-line CONTRIBUTING with 15 sections + four mandatory pre-reads (AGENTS.md, git-workflow.md, ADRs, open-issue-first), plus the SPDD canvas-before-code mandate for any feat: PR. The quality intent is sound (it IS a moat), but combined with bus-factor-1 (Wave A 5.24) and an enforced 2-day review SLA from a single volunteer maintainer, the practical barrier to a first feature contribution is high."
    evidence:
      - "cmd: `wc -l CONTRIBUTING.md` → 515 lines; CONTRIBUTING.md:13-24,41-42"
      - "CLAUDE.md «SPDD» (canvas-before-code for feat:); Wave A 5.12 (final.md:1699) soft-gate caveat"
      - "Wave A scorecard (final.md:50) — bus factor = 1"
  - severity: "Low"
    finding: "CLI gives no first-run-friendly hint on the most common failure (stack not up / wrong port): connection-refused surfaces as a raw Go net error, and the CLI default URL (8080) differs from the stack's mapped port (7080), a documented-but-easy-to-miss trip requiring the ZYNAX_API_URL export."
    evidence:
      - "cmd/zynax/client/gateway.go:330-332 (raw net error, no remedy hint)"
      - "cmd/zynax/cmd/root.go:40 (default 8080) vs docs/quickstart.md:94-99 (must export 7080)"
  - severity: "Low"
    finding: "Inner-loop latency: every make lint/test target spins a fresh tools container per service in a shell for-loop, so a single-service lint pays N container-startup costs. Acceptable for CI parity but slower than a native loop for tight iteration."
    evidence:
      - "Makefile:234-238 (lint-go for-loop, one TOOLS_RUN per service); Makefile:267-273 (test-unit-go same pattern)"

green_flags:
  - strength: "Self-documenting Makefile: `make help` renders 72 ★-annotated targets with a clean two-column format; entry points are visually marked, so discoverability of the whole DX surface is one command."
    evidence: ["Makefile:40-42", "cmd: grep -cE '^[a-zA-Z_-]+:.*?## ' Makefile → 72", "Makefile:45,115,162,227,264 (★ markers)"]
  - strength: "Genuinely zero-secret, zero-cloud runnable path: the Ollama overlay registers a real codereview capability against a local model, the demo model is single-sourced from config via awk (no drift), and observability is off-by-default and env-gated — the barrier to a first real run is Docker + one model pull, no API keys."
    evidence: ["Makefile:152-160 (DEMO_MODEL awk from llm-adapter.config.yaml)", "docs/quickstart.md:55-72,119-130", "Wave A 5.10 final.md:1357,1442-1450"]
  - strength: "Thoughtful, script-friendly CLI: SilenceUsage, env-default api-url, dry-run/validate previews before submit, meaningful exit codes (0 terminal / 2 running), structured compiler errors with line numbers, and a command reference that cross-links the CLI source as source-of-record."
    evidence: ["cmd/zynax/cmd/root.go:18-23", "cmd/zynax/cmd/apply.go:121-137", "cmd/zynax/cmd/status.go:26", "docs/quickstart.md:237-254"]
  - strength: "Complete, guard-railed local lifecycle and honest pre-commit story: full compose up/down/logs/ps/reset/restart-one set, a confirm-prompt on the destructive reset, and a pre-commit config that openly documents which hooks are auto-managed vs need a local toolchain (with the bypass + PR-note convention)."
    evidence: ["Makefile:85-99,207-209", ".pre-commit-config.yaml:1-11", "CONTRIBUTING.md:139-176"]
  - strength: "The GOWORK=off footgun is not hidden — it is documented in 4 separate surfaces (CLAUDE.md, CONTRIBUTING, local-dev, developer-guide) with the ADR-017 rationale, and most contributors never hit it because make targets set it for them."
    evidence: ["CLAUDE.md:68", "docs/local-dev.md:90", "CONTRIBUTING.md:136", "docs/developer-guide.md:124", "Makefile:212,271 (make sets GOWORK=off)"]

open_questions:
  - "What is the actual cold-clone wall-clock to first completed workflow? The '< 15 min' figure is a strategy target with no committed measurement, and the dominant costs (GHCR tools pull, compose --build, ollama model pull) are uninstrumented."
  - "Is the make-demo asciinema cast going to be recorded? The hero 'see it run' proof is a dead PLACEHOLDER (README.md:37) — the single highest-leverage DX trust artifact is missing (consumed from 5.10 final.md:1424-1428)."
  - "Would a `make quickstart` / `make first-run` umbrella target that chains bootstrap→install-cli→ollama-pull→demo close the gap between the 'one command' claim and the real 4-step prereq chain?"

unknowns:
  - "Runtime E1 proof of make demo / quickstart reaching a completed workflow — not executable in this read-only audit env (no Docker/Ollama); verified to E2/E3 wiring only (same boundary as Wave A 5.10)."
  - "Whether the documented GitHub Release CLI-binary URLs (docs/local-dev.md:9-18) resolve — no network access in audit env (UNKNOWN, cross-ref 5.10 final.md:1456)."
  - "Real onboarding-funnel conversion (issue → merged first PR) for external contributors — no contributor metric is committed; inferred from CONTRIBUTING + bus-factor-1, not measured."

cross_references:
  - to_agent: "5.10 Documentation"
    note: "Consumed 5.10's onboarding (8/High) and the stale README 'M5 status note' / make-demo PARTIAL drift; I re-flag the stale note and the dead asciinema cast as DX harms but do not re-score the doc zone (§3.1)."
    evidence: ["Wave A final.md:1380-1388,1409-1428"]
  - to_agent: "5.12 AI Workflow"
    note: "Consumed 5.12's soft-canvas-gate finding; it materially lowers the SPDD Day-0 wall for casual contributors (gate passes if any canvas exists, Draft only warns), so SPDD is a cultural quality bar more than a hard external-contributor blocker."
    evidence: ["Wave A final.md:1628-1631,1699"]
  - to_agent: "5.24 Repository Health"
    note: "Bus-factor-1 compounds DX contributor friction — a single volunteer maintainer with a 2-day SLA is the throughput ceiling on the otherwise-good onboarding funnel."
    evidence: ["Wave A final.md:50", "CONTRIBUTING.md:41-42"]
  - to_agent: "5.9 DevOps"
    note: "Docker-only inner loop (per-service container spin-up per lint/test) is a DX-speed cost that intersects DevOps build ergonomics; CI parity is the upside."
    evidence: ["Makefile:234-238,267-273"]

recommendations:
  - priority: "P0"
    action: "Make the 'one command' claim true OR re-label it. Either add a `make quickstart` umbrella target that chains bootstrap→install-cli→(ollama pull)→demo, or change README.md:24-33 to 'Three steps to see it run' and list the real prerequisites inline."
    rationale: "Closes the boldest DX drift at the highest-traffic surface (README front door). User type: new EVALUATOR. Adoption lever: time-to-first-success / first-impression trust."
  - priority: "P1"
    action: "Truth-pass the README Quickstart: delete the stale 'M5 status note' (333-337,355) and record/embed the make-demo asciinema cast to replace the PLACEHOLDER (README.md:37)."
    rationale: "Stops the front door under-claiming shipped capability and restores the visual 'see it run' proof. User type: EVALUATOR. Adoption lever: conversion / credibility."
  - priority: "P1"
    action: "Add a friendly CLI hint on connection-refused (e.g. 'cannot reach api-gateway at <url> — is the stack up? try make run-local and export ZYNAX_API_URL=http://localhost:7080') in cmd/zynax/client/gateway.go's post/Do error wrappers."
    rationale: "The 8080-vs-7080 default-port trip is the most common first-run failure; a one-line remedy hint removes a sharp edge. User type: new USER. Adoption lever: first-run success rate."
  - priority: "P2"
    action: "Add a 'casual contributor fast path' summary at the top of CONTRIBUTING (or a CONTRIBUTING-QUICK.md) for fix:/docs:/good-first-issue PRs that explicitly states the SPDD canvas is NOT required for non-feat work — so the 515-line read and the canvas mandate don't deter small first contributions."
    rationale: "Preserves SPDD as a quality moat for features while lowering the wall for the contributions that grow the bus factor. User type: external CONTRIBUTOR. Adoption lever: contributor funnel / bus factor."
  - priority: "P2"
    action: "Commit a measured cold-clone-to-first-result timing (and bake an instrumented `make demo --time` or a docs/casts timing note) to convert the '< 15 min' strategy target from CLAIMED to VERIFIED."
    rationale: "The 15-min figure is a core go-to-market conversion claim with no evidence; measuring it de-risks the headline. User type: EVALUATOR/maintainer. Adoption lever: defensible conversion metric."
```

---

# (b) §6.2 Prose section

## 5.11 Developer Experience — Score: 7 (High)

**Mission recap:** Assess Zynax's contributor and user experience — setup, build, local run, feedback loops, tooling ergonomics, error messages, CLI UX, and time-to-first-success — and drift-test the "<15 min" / one-command-demo claims.

**Verdict:** Zynax's DX is well-engineered and above market norm where it counts — a self-documenting 72-target Makefile, a thoughtful Cobra CLI with structured errors and script-friendly exit codes, a complete guard-railed compose lifecycle, an honest pre-commit story, and a genuinely zero-secret local-LLM path that needs only Docker plus one model pull. The substrate is strong. The headline weakness is a precise instance of the §1.10 drift class at the front door: the README markets `make demo` as "one command," but on a clean machine that single command exits immediately with `❌ zynax CLI not found` because `make demo` only checks for the CLI and never installs it, and it additionally needs a host `ollama` and a pulled model. The true first-run is a four-prerequisite chain, and the README Quickstart further *under*-claims shipped capability via a stale "M5 status note." Contributor friction is the other tension: the quality machinery (515-line CONTRIBUTING, SPDD canvas-before-code for features) is a real moat but a tall wall for casual contributors — softened by the fact that fix/docs/chore PRs are SPDD-exempt and the canvas gate is soft. Net: strong, fast where it matters, but the marketed first impression overstates the smoothness of the very first 60 seconds.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Time-to-first-success (steps & footguns) | 6 | High | README.md:24-33; Makefile:162-175; quickstart.md:24-72; developer-guide.md:13-18 |
| Local dev loop (compose/logs/reset/speed) | 8 | High | Makefile:85-99,115-126,207-209; TOOLS_RUN 35-37,234-238 |
| Tooling ergonomics (help/errors/GOWORK/hooks) | 8 | High | Makefile:40-42 (72 targets); CLAUDE.md:68; .pre-commit-config.yaml:1-54 |
| Contributor friction (SPDD moat vs wall) | 5 | High | wc -l CONTRIBUTING.md → 515; CLAUDE.md «SPDD»; Wave A 5.12 final.md:1628-1631 |
| CLI UX (discoverability/help/errors) | 8 | High | grep Short: → 21 cmds; root.go:18-23; apply.go:121-137; gateway.go:100-103,330-332 |
| Documentation-as-DX (consume 5.10) | 7 | High | Wave A 5.10 final.md:1380-1388,1417-1423; faq.md:13-33; local-dev.md:60-117 |

**Drift test:**
- *"make demo — one command end-to-end"* → **PARTIAL.** Wiring verified (Makefile:162-205) but it is a 4-prereq chain (CLI build + PATH + host ollama + model pull); demo exits if the CLI is absent (Makefile:163-164). Runtime E1 not executable here.
- *"Time-to-first-working-workflow < 15 min, one command"* → **PARTIAL (CLAIMED).** A strategy-doc target (strategy.md:294), not measured; the documented manual path is 8 steps and wall-clock is dominated by uninstrumented image + model pulls.
- *"Docker-only — nothing else needs Go locally"* → **PARTIAL.** True for lint/test/generate (tools container), but `make install-cli` compiles with a local Go toolchain (Makefile:211-213); Docker-only holds only via the release-binary route.

**Red flags (severity-ordered):**
1. **Medium —** "One command" is a multi-prereq chain; clean-machine `make demo` exits with `❌ zynax CLI not found` (README.md:24-33 vs Makefile:162-175). Highest-traffic surface, §1.10 drift class.
2. **Medium —** Stale README "M5 status note" under-claims shipped dispatch and contradicts its own service table (README.md:333-337,355; Wave A 5.10).
3. **Medium —** Heavy contributor onboarding (515-line CONTRIBUTING + SPDD canvas for feat:) over a bus-factor-1 maintainer with a 2-day SLA (CONTRIBUTING.md:13-24,41-42; Wave A 5.24).
4. **Low —** CLI gives no friendly remedy on connection-refused; default 8080 ≠ stack's 7080 (gateway.go:330-332; root.go:40 vs quickstart.md:94-99).
5. **Low —** Per-service container spin-up per lint/test taxes the inner loop (Makefile:234-238,267-273).

**Green flags:**
- Self-documenting Makefile: 72 ★-annotated targets via `make help` (Makefile:40-42).
- Zero-secret, zero-cloud runnable path; demo model single-sourced from config (Makefile:152-160; quickstart.md:55-72).
- Thoughtful CLI: SilenceUsage, env defaults, dry-run previews, exit codes 0/2, structured line-numbered errors (root.go:18-23; apply.go:121-137; status.go:26).
- Complete guard-railed compose lifecycle + honest pre-commit managed/system split (Makefile:85-99; .pre-commit-config.yaml:1-11).
- GOWORK=off trap documented in 4 surfaces and auto-set by make (CLAUDE.md:68; local-dev.md:90; CONTRIBUTING.md:136; developer-guide.md:124).

**Open questions / unknowns:** Actual cold-clone wall-clock to first result (uninstrumented); will the dead asciinema cast be recorded (README.md:37)?; would a `make quickstart` umbrella close the one-command gap?; runtime E1 of demo not executable here; release-binary URLs not network-verifiable; external contributor funnel unmeasured.

**Recommendations:** P0 — make the "one command" claim true (a `make quickstart` umbrella) or re-label it. P1 — truth-pass the README Quickstart + record the demo cast; add a friendly connection-refused CLI hint. P2 — add a casual-contributor fast path stating SPDD canvas is feat:-only; commit a measured <15-min timing to upgrade the claim from CLAIMED to VERIFIED.

**Cross-references:** 5.10 (onboarding 8/High + stale README note + dead cast — consumed, not re-scored); 5.12 (soft canvas gate lowers the SPDD wall for casual contributors); 5.24 (bus-factor-1 is the funnel throughput ceiling); 5.9 (Docker-only inner-loop speed cost vs CI-parity benefit).
<!-- SPDX-License-Identifier: Apache-2.0 -->
<!-- Agent 5.13 — Governance · Wave C (product/market/governance) · GitHub issue #1404 -->
<!-- READ-ONLY audit of the repository root at HEAD e3135a6 (2026-06-20). -->
<!-- Every factual claim carries path:line / command-output, or is marked UNKNOWN. -->
<!-- Roadmap/marketing = CLAIMED; code/CI/config/contract-verified = VERIFIED. -->
<!-- Consumes Wave A: Repository Health (5.24) + AI Workflow (5.12) from docs/due-diligence/2026-06-20-dd-wave-a-findings.md (cited, not re-scored — §3.1). -->

# Agent 5.13 — Governance (Wave C)

## (a) §3.4 Handoff packet

```yaml
agent: "5.13 Governance"
wave: "C"
dimension_groups: ["D12"]   # contributes D7/D15; feeds 5.25 Future Roadmap
overall_score: 7
overall_confidence: "High"

sub_scores:
  - dimension: "Decision discipline — are one-way doors ADR'd? Is the ADR process followed & current?"
    score: 8
    confidence: "High"
    justification: "37 ADRs, all present as files AND listed in INDEX.md with status/date/governs; one-way doors (license, no-shared-DB, gRPC, IR, merge policy, supply-chain) each have an ADR; a Rejected ADR (037) is recorded honestly. But the INDEX entry for ADR-023 contradicts the ADR body, and Deciders field is inconsistently applied."
    evidence:
      - "ls docs/adr/ → ADR-001..037 (37 files) + INDEX.md + TEMPLATE.md (no gaps in the number line)"
      - "docs/adr/INDEX.md:14-50 — every ADR-001..037 listed with Status/Date/Governs columns; statuses incl. Accepted/Proposed/Superseded/Rejected"
      - "docs/adr/INDEX.md:50 — ADR-037 'Rejected' (zero-Temporal engine, superseded by #1456) — negative decisions recorded, not hidden"
      - "docs/adr/INDEX.md:54-61 — status-definition legend (Proposed/Accepted/Deprecated/Superseded)"
      - "cmd: grep -rl Deciders docs/adr/ | wc -l → 13 of 37 ADRs carry a Deciders field (inconsistent; TEMPLATE.md:1-6 omits a Deciders field, so it is ad hoc)"
      - "docs/decisions/ — 4 lightweight 'decision records' (001-004) + README — a second, lighter decision tier below ADRs"
  - dimension: "Planning quality — milestone scoping, exit criteria, realism vs delivery cadence"
    score: 8
    confidence: "High"
    justification: "M7-planning.md is investment-grade program management: 22-deliverable brief-coverage map, explicit risk register, acceptance matrix with per-row done-checks, per-issue DoD, rollout/rollback. Machine-readable milestone state validated by schema. Delivery cadence (5.24) confirms plans predict reality (~92% M7 done)."
    evidence:
      - "docs/milestones/M7-planning.md:607-624 — §13 Acceptance Matrix: 12 criteria each with a concrete Done-check (e2e green / CI config / doc walkthrough)"
      - "docs/milestones/M7-planning.md:626-629 — explicit per-issue Definition of Done (canvas, AC, tests, make ci green, PR≤400, DCO, squash)"
      - "docs/milestones/M7-planning.md:686-712 — Appendix A maps all 22 brief deliverables to sections (incl. risk register, rollout, rollback)"
      - "state/milestone.schema.json:29-62 — strict JSON Schema (additionalProperties:false, semver/name patterns) validates state/milestone.yaml"
      - "state/milestone.yaml:4-9 — 'Updated ONLY by /milestone-close and /milestone-new'; validated by make validate-milestone-state + advisory pr-checks job"
      - "5.24 (Wave A): per-month cadence 121→259→444 commits; M7 ~115 closed/~10 open (~92%) → plans track reality (final.md:1758,1826)"
  - dimension: "Execution strategy — how is work claimed/tracked/closed? cross-machine safety? truth-pass habit"
    score: 8
    confidence: "High"
    justification: "Work flows through GitHub-label claiming + SPDD canvas-per-EPIC + squash-only signed merges; cross-machine safety is an explicit design goal of the delivery commands; a recurring documented 'Truth Pass' habit (M5.A #458, 2026-06-17 M7 truth-pass) actively reconciles narrative to reality."
    evidence:
      - "ROADMAP.md:138-146 — M5 Definition of Done: 7/7 criteria each [x] with the proof (e.g. '#1 zynax apply → WORKFLOW_STATUS_COMPLETED')"
      - "state/current-milestone.md:80-82 — '2026-06-17 truth-pass: verified architecture review committed ... milestone renamed + ROADMAP reconciled (#1299)'"
      - "ROADMAP.md:150 — 'M5.A — Truth Pass: docs aligned with shipped reality'; CHANGELOG phantom-entry purge #473 (5.24, final.md:1825)"
      - "5.12 (Wave A): /deliver.md:38-47 encodes cross-machine merge rules (squash-only because required_signatures blocks rebase; DCO; Assisted-by) (final.md:1558)"
      - "5.24 (Wave A): 0 merge commits in last 100 (squash-only/linear, ADR-023); 0 open/orphaned PRs; 94% issue closure (final.md:1775,1816)"
  - dimension: "Risk management — explicit risk register? how are blockers surfaced?"
    score: 7
    confidence: "High"
    justification: "Per-milestone risk register with likelihood/impact/mitigation/owner is genuine; blockers surfaced via 'Depends on #N' edges + state/current-milestone.md blocker notes + loud red CI runs (the [AUTO] issue-factory was deliberately retired). Gap: the register is per-milestone-plan, not a single living cross-cutting risk register, and is not refreshed mid-milestone."
    evidence:
      - "docs/milestones/M7-planning.md:485-496 — §8 Risk Register: 8 rows, each Likelihood/Impact/Mitigation/Owner-EPIC (e.g. R8 'scope creep — brief is ~3 milestones' High/High → program split M7/M-dx/M8)"
      - "docs/milestones/M7-planning.md:277 — blockers encoded as 'Depends on #N' so /milestone-orchestrate detects them without cross-referencing"
      - "state/current-milestone.md:71-73 — open blockers surfaced explicitly (#1359 deferred to M-dx; #1385/#1387 need own canvas)"
      - "5.24 (Wave A): weekly-audit.yml:4-8 — [AUTO] skeleton-issue factory retired for loud red runs (final.md:1798) — blockers surface as failing CI, not issue noise"
      - "GAP: no single cross-milestone risk register file; risk lives only inside each docs/milestones/M*-planning.md"
  - dimension: "Ownership & sustainability — CODEOWNERS, succession, founder dependency"
    score: 4
    confidence: "High"
    justification: "CODEOWNERS exists and routes every sensitive path to @zynax-io/maintainers; GOVERNANCE.md is a mature CNCF-shaped charter with an honest 'Solo Maintainer Phase'. BUT MAINTAINERS.md (referenced by GOVERNANCE + CODEOWNERS) does NOT exist, the @zynax-io/maintainers team is effectively one human (bus factor 1 per 5.24), and 13/13 ADR Deciders are the same single person — founder dependency is the dominant unmitigated risk."
    evidence:
      - ".github/CODEOWNERS:5-31 — '* → @zynax-io/maintainers'; KB paths (CLAUDE.md, AGENTS.md, .claude/) + every service dir owned (ADR-018)"
      - "cmd: ls MAINTAINERS.md → 'No existe el fichero' — MAINTAINERS.md MISSING despite GOVERNANCE.md:86 'Current maintainers: Listed in MAINTAINERS.md' + CODEOWNERS team refs"
      - "GOVERNANCE.md:104-118 — honest 'Solo Maintainer Phase (current)': self-merge after CI green; RFC + 5-day comment for breaking changes"
      - "GOVERNANCE.md:6 — 'Governance is neutral — no single company controls' (CLAIMED; contradicted by bus factor 1 + missing MAINTAINERS.md)"
      - "docs/adr/ADR-026:9 + ADR-029:9 — | Deciders | Oscar Gómez Manresa | (single decider on the ADRs that carry the field)"
      - "5.24 (Wave A): git shortlog -sne → one human identity (772 commits); MAINTAINERS.md is open issue #494 (final.md:1766,1907)"
  - dimension: "Governance-document currency (charter vs live reality)"
    score: 5
    confidence: "High"
    justification: "GOVERNANCE.md is structurally complete (roles, decision matrix, RFC, triage, release, AI-contributor policy) but §6 Milestone→Version Mapping is badly stale and self-contradicts the live ROADMAP/state; the RFC process is documented yet zero RFCs have ever been filed (decisions route through ADRs instead)."
    evidence:
      - "GOVERNANCE.md:247-258 — §6 maps M2→v0.2.0, M3→v0.3.0, M4→v0.4.0, M5→'Observability'/v0.5.0, M6→'Production Hardening'/v1.0.0-rc.1, M7→'Developer Experience'; references 'zynaxctl' CLI (line 254)"
      - "ROADMAP.md:267-275 + state/milestone.yaml:36-57 — actual: M2=v0.1.0, M5='Adapter Library'/v0.4.0, M6='K8s Production-Ready'/v0.5.0, M7='Usable Workflows + Observability'/v0.6.0 — DIRECTLY CONTRADICTS GOVERNANCE §6"
      - "GOVERNANCE.md:444 — CNCF checklist 'External security audit (target: v0.5.0)' unchecked though v0.5.0/M6 already shipped 2026-06-12 (stale target)"
      - "ls docs/rfcs/ → only RFC-000-template.md (zero filed RFCs); GOVERNANCE.md:180-207 documents a full RFC lifecycle that is unused in practice"
      - "docs/adr/INDEX.md:36 — ADR-023 INDEX row says 'rebase-merge only'; ADR-023:46-48 body says 'Merge strategy: squash-merge ... --rebase is rejected' — INDEX contradicts its own ADR (and the live squash-only reality, 5.24)"

drift_test:
  - claim: "M5 is Complete (v0.4.0) with 7/7 Definition-of-Done criteria met (ROADMAP.md:138-146) — a past milestone 'Complete' claim."
    result: "VERIFIED (with a minor register-timeline nuance)"
    evidence:
      - "DoD#3 'all 5 adapters merged' + DoD#5 'cel-go replaces bespoke guard evaluator': cel-go is a REAL dependency (engine-adapter/go.mod:7 'github.com/google/cel-go v0.28.1') and is used for guard eval (interpreter.go:17,201-215,226 celEnvironment/evalGuard) — shipped, not narrative."
      - "C6 register ('agent-registry 0 LoC M1-M4'): agent-registry/internal/{api,domain,infrastructure} all present (ls) — delivered under M5.C #460 as claimed (ROADMAP.md:152)."
      - "M5-plan.md:25,114 attributes cel-go to M5.B (#476/#538, ✅ Done) — so Part 1 §1.10 C8 ('claimed replaced M6') is the REGISTER's timeline imprecision, not an M5 over-claim; the milestone claim holds at HEAD."
  - claim: "M3 & M4 were honestly RE-LABELLED from 'Complete' to 'Partial' after the Truth Pass (Part 1 §1.10 C1)."
    result: "VERIFIED"
    evidence:
      - "state/current-milestone.md:17-18 — 'M3 ⚠ Partial / M4 ⚠ Partial'; :23 'M3/M4 are partial because task-broker and agent-registry were not delivered in those milestones. Both completed under M5.C (#460).'"
      - "CLAUDE.md §Per-Milestone Scope — M3 '(Partial)', M4 '(Partial)' carried in the canonical instruction file"
      - "This is the single strongest governance green flag: a self-correcting honest down-grade of a prior over-claim, exactly the behaviour §0.4/§1.10 exist to reward."
  - claim: "GOVERNANCE.md is current and 'no single company controls decisions' (GOVERNANCE.md:6)."
    result: "CONTRADICTED"
    evidence:
      - "GOVERNANCE.md:247-258 §6 milestone/version mapping is stale and contradicts ROADMAP.md:267-275 + state/milestone.yaml (M2 v0.2.0 vs actual v0.1.0; M6 'Production Hardening v1.0.0-rc.1' vs actual 'K8s Production-Ready v0.5.0'); references retired 'zynaxctl' name."
      - "ls MAINTAINERS.md → missing; git shortlog → one human (5.24, final.md:1766). 'Neutral, no single company' is aspirational (CLAIMED), not VERIFIED."
  - claim: "The ADR process is followed and the INDEX is complete & current."
    result: "PARTIAL"
    evidence:
      - "VERIFIED completeness: 37 ADR files = 37 INDEX rows, no gaps (ls + INDEX.md:14-50)."
      - "PARTIAL currency: INDEX.md:36 ADR-023 says 'rebase-merge only' but ADR-023 body + live reality are squash-only (ADR-023:46-48); 3 ADRs are still 'Proposed' on shipped/decided features (ADR-031 context-prop, ADR-033 expert-substrate, ADR-034 manifest-id — 5.1 flags ADR-034 contradicts shipped idempotent code)."

red_flags:
  - severity: "High"
    finding: "Founder dependency / succession is unmitigated AND the governance docs misrepresent it. MAINTAINERS.md — referenced by GOVERNANCE.md:86, CODEOWNERS, and the add/remove-maintainer process (§10) — does not exist; @zynax-io/maintainers is effectively one human (bus factor 1); every ADR Decider is the same person. GOVERNANCE.md:6 asserts 'neutral governance, no single company controls', which the repository state contradicts. The process is robust ON PAPER (solo-maintainer phase, RFC, supermajority) but has never been exercised by a second human, so its robustness-to-founder-leaving is untested and the named succession artifact is absent."
    evidence:
      - "cmd: ls MAINTAINERS.md → does not exist (GOVERNANCE.md:86 + CODEOWNERS reference it)"
      - "GOVERNANCE.md:6 'no single company controls' vs git shortlog one human (5.24, final.md:1766) + ADR-026:9/ADR-029:9 single Decider"
      - "5.24 (Wave A): MAINTAINERS.md is open issue #494; CNCF '≥2 maintainers from 2+ orgs' unmet (final.md:1907)"
  - severity: "Medium"
    finding: "GOVERNANCE.md §6 (Milestone→Version Mapping) is materially stale and self-contradicts the live ROADMAP and state/milestone.yaml — the single document that is supposed to be the project's authoritative charter carries an out-of-date version/scope table (M2 v0.2.0 vs v0.1.0; M5 'Observability'; M6 'v1.0.0-rc.1'; M7 'Developer Experience') and the retired 'zynaxctl' CLI name. For a project whose own thesis is 'truth pass against drift', the charter is itself drifted. The §12 CNCF 'external security audit target: v0.5.0' is also a stale target (v0.5.0 shipped)."
    evidence:
      - "GOVERNANCE.md:247-258, :444"
      - "ROADMAP.md:267-275; state/milestone.yaml:36-57"
  - severity: "Medium"
    finding: "The documented RFC process is unused: a full RFC lifecycle (GOVERNANCE.md:180-207, required for any proto/architecture/governance change) exists, but zero RFCs have ever been filed (only RFC-000-template.md). All real decisions route through ADRs (+ a lightweight docs/decisions tier). This is not wrong per se, but it means a load-bearing governance gate has never been executed — process maturity is partly theatrical until a second contributor triggers it."
    evidence:
      - "ls docs/rfcs/ → RFC-000-template.md only"
      - "GOVERNANCE.md:180-207 (RFC required for proto/arch/governance changes); GOVERNANCE.md:126-127 (RFC + PROTO REVIEWED for new contracts)"
  - severity: "Low"
    finding: "ADR-vs-INDEX and ADR-vs-code reconciliation debt: INDEX.md:36 mislabels ADR-023 as 'rebase-merge only' (body is squash-only); ADR-034 ('Proposed', random ids) contradicts the shipped deterministic idempotent apply (5.1 cross-ref). The Deciders field is applied to only 13/37 ADRs and is absent from TEMPLATE.md, so attribution discipline is ad hoc."
    evidence:
      - "docs/adr/INDEX.md:36 vs ADR-023-restrict-direct-pushes-to-main.md:46-48"
      - "5.1 (Wave A): ADR-034 Proposed/random-ids contradicts apply.go:29-35 (final.md:251-254)"
      - "cmd: grep -rl Deciders docs/adr/ | wc -l → 13; docs/adr/TEMPLATE.md:1-6 has no Deciders field"
  - severity: "Low"
    finding: "Residual command-name drift in planning/governance surfaces post PR#1400: M7-planning.md §9 SPDD runbook still lists pre-consolidation verbs (/spdd-reasons-canvas, /spdd-generate). state/milestone.yaml header cites /milestone-new while the live command set uses /milestone open|close + /lib:milestone-*. Corroborates 5.12's residual-lag finding; cosmetic but a truth-pass miss."
    evidence:
      - "docs/milestones/M7-planning.md:506-512 (/spdd-analysis, /spdd-reasons-canvas, /spdd-generate)"
      - "state/milestone.yaml:4 ('/milestone-close and /milestone-new')"
      - "5.12 (Wave A): pervasive residual command-name lag (final.md:1633-1638)"

green_flags:
  - strength: "Honest, self-correcting milestone labelling — the strongest governance signal. M3/M4 were down-graded 'Complete'→'Partial' with the explicit reason carried in the canonical state file and CLAUDE.md; a recurring 'Truth Pass' habit (M5.A #458; 2026-06-17 M7) actively reconciles narrative to shipped reality. This is the exact anti-drift behaviour §1.10 exists to reward."
    evidence:
      - "state/current-milestone.md:17-23"
      - "ROADMAP.md:150 (M5.A Truth Pass); state/current-milestone.md:80-82"
  - strength: "Investment-grade per-milestone planning: M7-planning.md carries a 22-deliverable brief-coverage map, an 8-row risk register (likelihood/impact/mitigation/owner), a 12-criterion acceptance matrix with concrete done-checks, per-issue DoD, and rollout/rollback — and delivered ~92% of it on a high, accelerating cadence (5.24)."
    evidence:
      - "docs/milestones/M7-planning.md:485-496,607-629,686-712"
      - "5.24 (Wave A): cadence 121→259→444/mo; M7 ~92% done (final.md:1758,1826)"
  - strength: "Strong, machine-checked decision substrate: 37 ADRs all present and indexed (incl. a recorded 'Rejected' ADR-037 — negative decisions kept); a JSON-Schema-validated milestone state machine; an engineering manifesto whose 15 principles each carry an *Enforced by:* line and honest ⏳ markers for not-yet-enforced gates."
    evidence:
      - "docs/adr/INDEX.md:14-50; state/milestone.schema.json:29-62"
      - "docs/contributing/engineering-manifesto.md:8-12,38-58 (P1 main-protection ruleset; P2 canvas-freshness ⏳ not-yet-required)"
  - strength: "Mature, enforced merge & contribution governance: ADR-023 squash-only + required signatures + linear history (verified 0 merge commits by 5.24); DCO enforced; conventional-commit 7-type gate; a documented AI-contributor policy (human sponsor of record, Assisted-by not Co-Authored-By); CODEOWNERS gating KB + every service path (ADR-018)."
    evidence:
      - "GOVERNANCE.md:141-176 (DCO + commit hygiene), :318-368 (AI contributor policy)"
      - ".github/CODEOWNERS:5-31; 5.24 git log --merges -100 → 0 (final.md:1775)"

open_questions:
  - "Does the documented governance survive the founder leaving? The solo-maintainer phase, RFC process, and supermajority votes have NEVER been exercised by a second human — robustness is asserted, not demonstrated (GOVERNANCE.md:104-118; zero RFCs filed)."
  - "Is required_signatures / main-protection ruleset actually enforced server-side, or only documented? Not verifiable read-only from the clone (also a 5.24 open question)."
  - "Why is MAINTAINERS.md (issue #494) still unfiled this late (37 ADRs, v0.5.0 shipped, CNCF-targeted) — is it deliberate honesty about bus factor 1, or an overdue gap blocking the CNCF social gate?"

unknowns:
  - "Whether GitHub branch-protection rule contents (required checks, required_signatures, up-to-date-before-merge) match the manifesto/ADR-023 claims — server-side config not readable from the working tree."
  - "Whether the lazy-consensus / 5-day RFC comment periods have ever been observed in practice — no filed RFC exists to inspect timing against."
  - "GitHub Project board state (GOVERNANCE.md:296-303 names it the authoritative execution roadmap) — not inspectable read-only/offline."

cross_references:
  - to_agent: "5.25 Future Roadmap"
    note: "M8 exit criteria are concrete and gated on the SAME unmet social items governance flags: ≥2 maintainers from 2+ orgs, external security audit, production reference deploy, CNCF TOC filing. Planning realism is high but M8 is blocked by bus factor 1 + missing MAINTAINERS.md, not by technical scope."
    evidence: ["ROADMAP.md:255-261", "GOVERNANCE.md:434-447", "MAINTAINERS.md missing (#494)"]
  - to_agent: "5.8 Open Source"
    note: "GOVERNANCE.md/CONTRIBUTING/CoC/SECURITY all present and CNCF-shaped, but MAINTAINERS.md missing and 'neutral governance' contradicted by bus factor 1 — score the social/community gate, not the charter prose."
    evidence: ["GOVERNANCE.md:6,86,104-118", "ls MAINTAINERS.md → missing"]
  - to_agent: "5.21 CNCF"
    note: "GOVERNANCE.md:434-447 CNCF checklist is partly stale (external-audit target v0.5.0 already shipped, unchecked); ≥2-maintainer + audit gates unmet. Decision/ADR/SBOM/cosign structural alignment is strong."
    evidence: ["GOVERNANCE.md:438-447", "docs/adr/INDEX.md:14-50"]
  - to_agent: "5.10 Documentation"
    note: "GOVERNANCE.md §6 milestone/version table + 'zynaxctl' name are stale vs ROADMAP/state; ADR-023 INDEX row contradicts its own ADR; M7-planning §9 + milestone.yaml header carry pre-PR#1400 command names — doc-currency reconciliation debt."
    evidence: ["GOVERNANCE.md:247-258", "docs/adr/INDEX.md:36", "docs/milestones/M7-planning.md:506-512"]
  - to_agent: "5.1 Architecture"
    note: "Re-confirms 5.1's ADR-034 finding from the governance angle: a 'Proposed' ADR contradicting shipped code is unreconciled decision-record drift; INDEX currency is the governance owner's responsibility."
    evidence: ["docs/adr/ADR-034-manifest-workflow-id-collision-domain.md (Proposed)", "5.1 packet (final.md:251-254)"]

recommendations:
  - priority: "P0"
    action: "Create MAINTAINERS.md (close #494) and either recruit a second maintainer from a second org or explicitly mark the project single-maintainer in GOVERNANCE.md:6 instead of asserting 'no single company controls'. Until then the charter overstates neutrality."
    rationale: "Founder dependency is the dominant sustainability risk and the explicit CNCF social gate; the named succession artifact is referenced everywhere yet absent. Aligns governance prose with verifiable reality."
  - priority: "P1"
    action: "Run a truth-pass over GOVERNANCE.md itself: rewrite §6 Milestone→Version Mapping to match ROADMAP/state (M2=v0.1.0 … M7=v0.6.0), drop 'zynaxctl', refresh the §12 CNCF external-audit target, and fix the ADR-023 INDEX row ('squash-merge only', not 'rebase-merge')."
    rationale: "The project's signature discipline is anti-drift truth-passing; its own charter and ADR index currently carry exactly the drift class it polices, undermining credibility with a diligence/CNCF reviewer."
  - priority: "P2"
    action: "Decide the RFC-vs-ADR-vs-docs/decisions tiering explicitly (RFC has never been used) and standardise the ADR Deciders field by adding it to TEMPLATE.md so attribution is consistent across all ADRs."
    rationale: "Removes a load-bearing-but-unused process (RFC) ambiguity and makes decision attribution uniform — both improve robustness-to-founder-leaving."
  - priority: "P2"
    action: "Promote a single living cross-milestone risk register (or link the per-milestone §8 registers from state/current-milestone.md) and refresh it mid-milestone, not only at planning time."
    rationale: "Risk registers exist per-plan but are not continuously surfaced; a living register closes the gap between planned and emergent risk."
```

---

## (b) §6.2 Prose section

## 5.13 Governance — Score: 7 (High)

**Mission recap:** Assess Zynax's decision discipline (ADRs / one-way doors), planning quality (milestones, exit criteria, realism vs cadence), execution strategy (claim/track/close, cross-machine safety, truth-pass), risk management, and ownership/sustainability — and run the drift test on a past milestone's "Complete" claim.

**Verdict:** Zynax has **unusually mature process governance for a two-month-old project, dragged down by one structural fact and a stale charter.** The decision substrate (37 ADRs, all indexed, including a recorded *Rejected* ADR), the program-management quality (a 22-deliverable M7 plan with risk register, acceptance matrix, per-issue DoD, rollout/rollback), the machine-validated milestone state machine, and an enforced engineering manifesto are all genuinely strong and verified in-repo. The signature behaviour — an honest, self-correcting "Truth Pass" that re-labelled M3/M4 from *Complete* to *Partial* — is exactly the anti-drift discipline this diligence rewards. The two material problems: (1) **founder dependency is unmitigated and partly misrepresented** — `MAINTAINERS.md` is referenced by the charter and CODEOWNERS but does not exist, the maintainer team is effectively one human, and GOVERNANCE.md still claims "neutral governance, no single company controls"; and (2) the **charter itself has drifted** — GOVERNANCE.md §6 carries a stale milestone/version table that contradicts the live ROADMAP and `state/milestone.yaml`, and the documented RFC process has never once been used. The process is robust on paper but its robustness-to-founder-leaving is untested.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Decision discipline (ADRs / one-way doors) | 8 | High | `docs/adr/INDEX.md:14-50` (37 indexed, incl. Rejected ADR-037); `grep Deciders → 13/37` |
| Planning quality (scoping / exit criteria / realism) | 8 | High | `M7-planning.md:607-629,686-712` (acceptance matrix + DoD + 22-deliverable map); `milestone.schema.json:29-62` |
| Execution strategy (claim/track/close, cross-machine, truth-pass) | 8 | High | `ROADMAP.md:138-146` (M5 7/7 DoD); `current-milestone.md:80-82` (truth-pass); 5.24 squash-only/0 merges |
| Risk management (register + blocker surfacing) | 7 | High | `M7-planning.md:485-496` (§8 register); `:277` ('Depends on #N'); weekly-audit (5.24) |
| Ownership & sustainability (CODEOWNERS / succession / founder dep) | 4 | High | `.github/CODEOWNERS:5-31`; `ls MAINTAINERS.md → missing`; one human (5.24); single ADR Decider |
| Governance-document currency | 5 | High | `GOVERNANCE.md:247-258,444` stale vs `ROADMAP.md:267-275`; zero RFCs filed; INDEX ADR-023 mislabel |

**Drift test:**
- *Past milestone "Complete" claim — "M5 Complete, 7/7 DoD met" (ROADMAP.md:138-146)* → **VERIFIED.** The boldest DoD items check out in code: cel-go is a real dependency driving guard evaluation (`engine-adapter/go.mod:7`, `interpreter.go:201-215`), and agent-registry shipped with full `internal/{api,domain,infrastructure}`. The Part 1 §1.10 C8 row ("cel-go claimed replaced M6") is the *register's* timeline imprecision — the M5 plan attributes cel-go to M5.B (#476/#538) — not a milestone over-claim. The M5 "Complete" claim holds at HEAD.
- *M3/M4 honestly re-labelled Complete→Partial (§1.10 C1)* → **VERIFIED** (`state/current-milestone.md:17-23`, CLAUDE.md). The strongest green flag.
- *GOVERNANCE.md current & "no single company controls" (line 6)* → **CONTRADICTED** (§6 stale version table vs ROADMAP/state; MAINTAINERS.md missing; bus factor 1).
- *ADR process followed & INDEX complete/current* → **PARTIAL** (37/37 indexed, but ADR-023 INDEX row contradicts its body, and 3 'Proposed' ADRs cover shipped/decided features).

**Red flags (severity-ordered):**
1. **High** — Founder dependency unmitigated and the charter misrepresents it: `MAINTAINERS.md` missing despite being referenced (GOVERNANCE.md:86, CODEOWNERS, §10), maintainer team is one human, every ADR Decider is the same person; GOVERNANCE.md:6 "no single company controls" is CLAIMED, not VERIFIED. Process robustness-to-founder-leaving is untested (#494).
2. **Medium** — GOVERNANCE.md §6 milestone/version mapping is stale and self-contradicts ROADMAP.md:267-275 / state/milestone.yaml (M2 v0.2.0 vs v0.1.0; M6 "Production Hardening v1.0.0-rc.1"; M7 "Developer Experience"; retired "zynaxctl"); §12 CNCF audit target v0.5.0 already shipped. The anti-drift project's own charter has drifted.
3. **Medium** — The documented RFC gate (required for proto/architecture/governance changes) has never been used; zero RFCs filed. A load-bearing governance process is untested.
4. **Low** — ADR reconciliation debt: INDEX.md:36 mislabels ADR-023 'rebase-merge only'; ADR-034 ('Proposed', random ids) contradicts shipped idempotent apply (5.1); Deciders applied to only 13/37 ADRs, absent from TEMPLATE.md.
5. **Low** — Residual pre-PR#1400 command names in M7-planning §9 and the milestone.yaml header (corroborates 5.12).

**Green flags:**
- Honest, self-correcting milestone labelling + a recurring Truth-Pass habit (M3/M4 down-graded; M5.A #458; 2026-06-17 M7 reconcile) — `state/current-milestone.md:17-23,80-82`.
- Investment-grade per-milestone planning (risk register + acceptance matrix + per-issue DoD + rollout/rollback) delivered at ~92% on accelerating cadence — `M7-planning.md:485-629`, 5.24.
- Strong machine-checked decision substrate: 37 indexed ADRs incl. a Rejected one, schema-validated milestone state, enforced engineering manifesto with honest ⏳ markers — `INDEX.md:14-50`, `milestone.schema.json`, `engineering-manifesto.md:38-58`.
- Mature merge/contribution governance: ADR-023 squash-only + signed + linear (0 merge commits, 5.24), DCO, conventional-commit gate, defined AI-contributor policy, CODEOWNERS over KB + every service — `GOVERNANCE.md:141-176,318-368`, `CODEOWNERS:5-31`.

**Open questions / unknowns:** Does the governance survive the founder leaving (solo-phase/RFC/supermajority never exercised by a 2nd human)? Is the main-protection ruleset enforced server-side or only documented (not readable read-only)? Why is MAINTAINERS.md (#494) still unfiled this late? RFC comment-period timing has no filed instance to inspect; GitHub Project board not inspectable offline.

**Recommendations:** **P0** — create MAINTAINERS.md (#494) and recruit a second-org maintainer, or stop asserting neutral governance until then. **P1** — truth-pass GOVERNANCE.md itself (fix §6 version table, drop `zynaxctl`, refresh CNCF audit target, fix ADR-023 INDEX row). **P2** — resolve RFC-vs-ADR tiering (RFC unused) and standardise the ADR Deciders field via TEMPLATE.md; promote a single living cross-milestone risk register.

**Cross-references:** 5.25 Future Roadmap (M8 gated on the same unmet social items governance flags); 5.8 Open Source (missing MAINTAINERS.md, neutrality claim); 5.21 CNCF (stale CNCF checklist, ≥2-maintainer/audit gates unmet); 5.10 Documentation (GOVERNANCE §6 + ADR-023 INDEX + command-name drift); 5.1 Architecture (ADR-034 Proposed-vs-shipped, INDEX currency).
# Agent 5.19 — Competitive Analysis · Wave C (product/market/governance)

> Issue #1404 · HEAD `e3135a6` · READ-ONLY audit.
> Every claim is grounded in `path:line` (Zynax), cited E7 (competitors), or marked `UNKNOWN`.
> Marketing/roadmap = `CLAIMED`; code/CI/contract-verified = `VERIFIED`.
> Consumes Wave A 5.1 Architecture packet (`docs/due-diligence/2026-06-20-dd-wave-a-findings.md`) — does NOT re-score 5.1.

---

## (a) §3.4 Handoff packet

```yaml
agent: "5.19 Competitive Analysis"
wave: "C"
dimension_groups: ["D2"]
overall_score: 5
overall_confidence: "Medium"

sub_scores:
  - dimension: "Differentiator: engine-agnostic IR + multi-engine portability (the headline moat)"
    score: 5
    confidence: "High"
    justification: "Real & structurally hard-to-copy at the CONTRACT boundary (engine-neutral IR + 5-method port), but only Temporal interprets the IR; Argo is a non-interpreting stub and LangGraph-as-engine does not exist. Portability is proven at submission, not execution — the moat is half-built."
    evidence:
      - "protos/zynax/v1/workflow_compiler.proto:205-241 — WorkflowIR is a pure state-machine model, zero engine types (Wave A 5.1: score 9)"
      - "services/engine-adapter/internal/domain/engine.go:17-41 — clean 5-method WorkflowEngine port; both engines compile-time conform (Wave A 5.1)"
      - "scripts/e2e/manifests/argo-ir-interpreter.yaml:10-14 — Argo leg 'asserts a non-empty IR payload arrived and exits 0 ... Full Argo-side IR interpretation ... deliberately out of scope' (Wave A 5.1 High red flag)"
      - "services/engine-adapter/cmd/engine-adapter/main.go:177-178 — engine switch is temporal|argo ONLY; no langgraph engine"
      - "services/engine-adapter/AGENTS.md:4 — 'LangGraph/Argo WorkflowEngine backends deferred to M6+'"
  - dimension: "Differentiator: no-SDK adapter-first capability contract (AgentService gRPC)"
    score: 7
    confidence: "High"
    justification: "Genuinely real and clean: any gRPC service implementing one 2-RPC contract becomes a capability — no SDK/language/framework. This is the structural basis of the Kagent 'complementary' story. Contestable: Restate/Dapr offer multi-language SDK paths that are nearly as low-friction in practice."
    evidence:
      - "protos/zynax/v1/agent.proto:3-7 — 'Any system that serves this single RPC becomes a first-class Zynax capability: no SDK required, no framework required, no language requirement.'"
      - "protos/zynax/v1/agent.proto:34-47 — AgentService{ExecuteCapability(stream), GetCapabilitySchema}"
      - "services/task-broker/internal/domain/service.go:18,48 — broker dispatches by capability_name (routing-by-capability, not identity)"
      - "E7: Restate 5-language SDKs (TS/Py/Java/Go/Rust), single binary — restate.dev/what-is-durable-execution"
  - dimension: "Differentiator: compile-time structural validation of the workflow state machine"
    score: 7
    confidence: "High"
    justification: "Strongest UNDER-marketed edge. Terminal/orphan/cycle/transition-target validation runs at compile time before any engine sees the IR — most rivals interpret YAML/code at runtime. Real, defensible, and not trivially copied without an IR."
    evidence:
      - "services/workflow-compiler/internal/domain/validators/structural.go:11-58 — TerminalStateValidator, OrphanStateValidator, CircularTransitionDetector (BFS/DFS at compile time)"
      - "docs/architecture/2026-04-30-competitive-analysis.md:261 — 'Compile-time error detection' column: only Zynax marked best-in-class across 8 tools"
      - "E7: LangGraph/Temporal/Dapr offer durability at RUNTIME; structural topology errors surface at execution, not compile (alphabold/langgraph-agents-in-production; docs.temporal.io)"
  - dimension: "Differentiator: GitOps-native YAML + deploy-anywhere (Compose AND K8s)"
    score: 5
    confidence: "Medium"
    justification: "`zynax apply` + gitops watcher are real and Compose-AND-K8s is a genuine deploy-surface edge over K8s-only Kagent. BUT GitOps is now contested: Kagent integrates with ArgoCD/GitOps (E7), and declarative YAML is table-stakes. Edge is real but narrowing."
    evidence:
      - "cmd/zynax/cmd/apply.go:60,104 — runApplyScenario / runApply (zynax apply implemented)"
      - "cmd/zynax/gitops/watcher.go:37,107 — file-watch → apply-on-change (GitOps watch)"
      - "docs/architecture/2026-05-28-competitive-positioning.md:26 — Compose (M5) + K8s Helm (M6) vs Kagent 'K8s-only (Kind required)'"
      - "E7: Kagent 'integrates naturally with ArgoCD and GitOps workflows' — thenewstack.io/meet-kagent; kagent.dev/docs"
  - dimension: "Named rival Kagent — head-to-head where each WINS"
    score: 4
    confidence: "High"
    justification: "Kagent decisively wins on the buyer-visible axes: CNCF Sandbox backing (Solo.io), shipped Web GUI+CLI, MCP/OpenAPI tool discovery, 7+ LLM providers, HITL, K8s-native, ArgoCD-integrated. Zynax wins ONLY on engine-portability (half-built) + compile-time IR validation + Compose deploy. A buyer-facing scorecard favors Kagent today."
    evidence:
      - "E7: Kagent CNCF Sandbox, Web GUI+CLI, MCP+OpenAPI discovery, HITL, Bedrock/Anthropic/Azure/Gemini/Vertex/Ollama/OpenAI — thenewstack.io; kagent.dev/docs; cncf/sandbox#360"
      - "docs/architecture/2026-05-28-competitive-positioning.md:32,35,36 — Zynax: Web UI ❌ (M8), CNCF ❌ (M8 target), Production-proven No (v0.4.0); Kagent: UI ✅, CNCF ✅ Sandbox, Production Partial"
      - "find <repo> -iname '*ui*'/'*web*'/'*dashboard*' → no Zynax web UI exists (M8 CLAIMED)"
      - "strategy.md:290 — Zynax '0 stars, 0 forks, 0 external adopters' vs Kagent growing ecosystem"
  - dimension: "'Complementary, not competitive' framing — buyer credibility"
    score: 3
    confidence: "Medium"
    justification: "Architecturally CREDIBLE (a Kagent agent CAN implement AgentService gRPC and register) but NOT DEMONSTRATED: zero Kagent adapter/example/test in code — the integration exists only in prose. A buyer already on Kagent gets the workflow layer they need from Kagent itself (Argo/GitOps/HITL); the 'just register Kagent under Zynax' story adds a second control plane for unproven portability. Reads as wishful from the smaller, un-adopted project."
    evidence:
      - "grep -rni kagent <repo> (excluding docs/) → ZERO hits in .go/.py/.yaml; Kagent appears ONLY in docs/marketing"
      - "docs/architecture/2026-05-28-competitive-positioning.md:75 — 'A Kagent agent can register as a Zynax capability via the AgentService gRPC contract' (CLAIMED; no adapter, no e2e proof)"
      - "protos/zynax/v1/agent.proto:3-7 — contract makes it POSSIBLE; no Kagent-shaped implementation/test demonstrates it"
      - "E7: Kagent ships its own HITL + ArgoCD/GitOps — overlaps the control-plane layer Zynax claims to add, weakening 'Kagent lacks a control plane'"
  - dimension: "Time-to-parity: how fast could a funded rival copy the IR/portability moat"
    score: 5
    confidence: "Medium"
    justification: "The COPYABLE part (IR + WorkflowEngine port + compile-time validators) is a few quarters of work for a funded team — the design is documented (ADR-012/015) and the proto is public. The HARD-to-copy part (genuine multi-engine execution parity) Zynax itself has NOT yet built. So the moat that exists is shallow, and the deep version is unbuilt by everyone."
    evidence:
      - "docs/adr/ADR-012-workflow-ir.md, ADR-015-pluggable-workflow-engines.md — design is public & documented (lowers copy cost)"
      - "Wave A 5.1 open_question — true cost of Argo IR-interpretation parity is UNKNOWN even to Zynax (docs/due-diligence/2026-06-20-dd-wave-a-findings.md:166)"
      - "E7: Restate & Dapr Agents already ship durable multi-language agent orchestration at v1.0/production — a funded rival starts from a production base, not zero (restate.dev; github.com/dapr/dapr-agents)"

drift_test:
  - claim: "Engine-agnostic IR — write a workflow once, run it on Temporal OR Argo (OR LangGraph) without a rewrite (boldest differentiator)."
    result: "PARTIAL"
    evidence:
      - "VERIFIED at contract: same YAML→same WorkflowIR→same 5-method port; Argo compile-time conforms (workflow_compiler.proto:205-241; engine.go:17-41; argo_engine.go:314 via Wave A 5.1)."
      - "CONTRADICTED at execution: Argo path is a non-interpreting stub that checks IR non-empty and exits 0 (argo-ir-interpreter.yaml:10-14); only Temporal traverses states & dispatches (Wave A 5.1 High red flag)."
      - "CONTRADICTED on LangGraph-as-engine: positioning doc says 'Temporal + LangGraph adapters' (2026-05-28-competitive-positioning.md:28) but engine switch is temporal|argo only (main.go:177-178) and AGENTS.md:4 defers LangGraph engine to M6+; LangGraph is a CAPABILITY adapter (agents/adapters/langgraph/), not an engine."
  - claim: "Zynax is complementary to Kagent — a Kagent agent registers as a Zynax capability; not a rival."
    result: "PARTIAL"
    evidence:
      - "VERIFIED-possible: AgentService gRPC contract genuinely accepts any gRPC service as a capability (agent.proto:3-7,34-47)."
      - "CONTRADICTED-as-shipped: no Kagent adapter/example/test anywhere in code (grep kagent excl docs → 0); integration is prose-only (positioning.md:75)."
      - "WEAKENED by competitor reality: Kagent now ships HITL + ArgoCD/GitOps + Web UI + MCP discovery (E7), occupying much of the 'control plane' Zynax claims Kagent lacks (positioning.md:77)."
  - claim: "No tool combines declarative YAML + formal engine-agnostic IR + capability registry + pluggable engines — the gap is real and unoccupied (competitive-analysis.md:22-25)."
    result: "PARTIAL"
    evidence:
      - "VERIFIED unique combination at the design level: the formal compile-step + IR is genuinely rare (competitive-analysis.md:257,261; structural.go:11-58)."
      - "WEAKENED: since that 2026-04-30 doc, Dapr Agents (v1.0, durable workflow engine + multi-agent + capability-ish components) and Restate (durable execution + AI-agent orchestration, 5 SDKs) have shipped production-grade adjacent stacks; the 'unoccupied gap' is now crowding (E7: github.com/dapr/dapr-agents; restate.dev/blog durable-orchestration-for-ai-agents)."

red_flags:
  - severity: "Critical"
    finding: "The named direct rival out-positions Zynax on every buyer-visible axis: CNCF Sandbox backing (Solo.io), shipped Web GUI+CLI, MCP/OpenAPI tool discovery, multi-LLM, HITL, K8s-native, ArgoCD/GitOps integration — while Zynax has 0 stars/forks/adopters, no UI, and no CNCF status. Zynax's one structural edge (engine portability) is only half-built (Temporal interprets; Argo stubs). A buyer comparison today favors Kagent; Zynax wins only on a moat it hasn't finished."
    evidence:
      - "E7: thenewstack.io/meet-kagent; kagent.dev/docs; cncf/sandbox#360 (Kagent UI/CLI/MCP/HITL/CNCF)"
      - "docs/architecture/2026-05-28-competitive-positioning.md:32,35,36 (Zynax UI/CNCF/prod all ❌/No)"
      - "strategy.md:290 (0 stars/0 forks/0 adopters)"
      - "scripts/e2e/manifests/argo-ir-interpreter.yaml:10-14 (Argo non-interpreting; Wave A 5.1)"
  - severity: "High"
    finding: "The boldest differentiator is overstated in Zynax's own positioning: the M5 positioning table claims 'Temporal + LangGraph adapters' for engine portability, but LangGraph-as-engine does not exist (engine switch is temporal|argo; LangGraph is a capability adapter) and Argo does not interpret the IR. This is the delivery-vs-narrative drift class (Part 1 §1.10) reappearing in the competitive doc."
    evidence:
      - "docs/architecture/2026-05-28-competitive-positioning.md:28 ('Temporal + LangGraph adapters')"
      - "services/engine-adapter/cmd/engine-adapter/main.go:177-178; services/engine-adapter/AGENTS.md:4"
      - "scripts/e2e/manifests/argo-ir-interpreter.yaml:10-14 (Wave A 5.1 High red flag)"
  - severity: "High"
    finding: "The 'complementary to Kagent' story is prose-only — zero Kagent adapter, example, or e2e test in the codebase — while the underlying premise ('Kagent lacks a control plane') is eroded by Kagent shipping HITL + ArgoCD/GitOss. To a buyer already running Kagent, adding a second, un-adopted control plane for unproven portability is a hard sell."
    evidence:
      - "grep -rni kagent <repo> excl docs → 0 code hits"
      - "docs/architecture/2026-05-28-competitive-positioning.md:75-78 (CLAIMED integration; CLAIMED 'control plane Kagent lacks')"
      - "E7: kagent.dev/docs (HITL); thenewstack.io/meet-kagent (ArgoCD/GitOps)"
  - severity: "Medium"
    finding: "Time-to-parity on the copyable portion of the moat is short. The IR design, WorkflowEngine port, and validators are public (ADR-012/015 + open proto); a funded rival could replicate the contract layer in a couple of quarters. Meanwhile Restate and Dapr Agents already ship production durable multi-agent orchestration, so a fast-follower starts ahead of Zynax on maturity."
    evidence:
      - "docs/adr/ADR-012-workflow-ir.md; docs/adr/ADR-015-pluggable-workflow-engines.md; protos/zynax/v1/*.proto (public design)"
      - "E7: github.com/dapr/dapr-agents (v1.0 production); restate.dev/blog (durable AI-agent orchestration)"
  - severity: "Low"
    finding: "The strongest genuinely-defensible edge (compile-time structural IR validation) is under-marketed — buried in the feature matrix rather than led with. Zynax leads with portability (half-built) instead of validation (fully shipped and rare)."
    evidence:
      - "services/workflow-compiler/internal/domain/validators/structural.go:11-58 (shipped)"
      - "docs/architecture/2026-04-30-competitive-analysis.md:261 (only-Zynax column, under-emphasized in positioning.md)"

green_flags:
  - strength: "Compile-time structural validation is a real, rare, hard-to-copy edge: terminal/orphan/cycle/transition-target checks run before any engine executes the IR — a structural guarantee runtime-interpreting rivals (Temporal, LangGraph, Dapr) do not provide."
    evidence: ["services/workflow-compiler/internal/domain/validators/structural.go:11-58", "docs/architecture/2026-04-30-competitive-analysis.md:261"]
  - strength: "The no-SDK AgentService contract is genuinely clean and minimal (2 RPCs, no language/framework requirement) — the most legible part of the value prop and the only credible technical basis for any future 'complementary' integration."
    evidence: ["protos/zynax/v1/agent.proto:3-7,34-47", "services/task-broker/internal/domain/service.go:18,48"]
  - strength: "Engine-neutral IR + WorkflowEngine port are textbook-clean (Wave A 5.1 scored 9 and 8): the STRUCTURE for a real portability moat exists; if Argo/LangGraph reach interpreter parity the differentiator becomes defensible."
    evidence: ["protos/zynax/v1/workflow_compiler.proto:205-241", "services/engine-adapter/internal/domain/engine.go:17-41", "docs/due-diligence/2026-06-20-dd-wave-a-findings.md:38-55"]

open_questions:
  - "Why does Zynax win a head-to-head against Kagent TODAY, given Kagent has CNCF + UI + ArgoCD/GitOps + HITL + multi-LLM and Zynax's portability is half-built? Current honest answer: only for a buyer who specifically needs non-Temporal/non-K8s engine portability AND compile-time topology validation AND zero adopters is acceptable."
  - "How long does the moat last? The copyable layer (IR/port/validators) is short-fuse (public design); the deep layer (true multi-engine execution parity) is unbuilt by Zynax and everyone — so 'lasting moat' depends on Zynax SHIPPING parity before a funded rival or Kagent adds an IR layer."
  - "Is 'complementary to Kagent' a go-to-market or a fallback? If Kagent keeps absorbing control-plane features, the complementary story collapses into 'subset of Kagent'."

unknowns:
  - "Whether Kagent has, or is building, any engine-portability / formal-IR layer that would directly contest Zynax's last structural edge — not found in E7 within this search; UNKNOWN."
  - "Live GitHub traction deltas (Kagent stars/contributors vs Zynax) at June 2026 — not independently fetched; strategy.md asserts Zynax at 0 baseline."
  - "Whether any design-partner / private adopter exists that would change the 'no adopter' competitive posture — none found in repo; UNKNOWN."

cross_references:
  - to_agent: "5.1 Architecture"
    note: "Consumed verbatim: portability is real at the IR interface, stubbed at Argo execution. The competitive moat inherits this exact half-built status. Did NOT re-score 5.1."
    evidence: ["docs/due-diligence/2026-06-20-dd-wave-a-findings.md:56-64,128-132"]
  - to_agent: "5.4 Product / 5.17 Market-fit / 5.18 Positioning"
    note: "Per §5.19 RETURN, D2 feeds 5.4/5.17/5.18: lead with compile-time validation (shipped, rare), NOT portability (half-built); 'complementary to Kagent' needs a real Kagent adapter+demo before it is sayable to a buyer; CNCF/UI/adopter gap is the dominant competitive deficit."
    evidence: ["services/workflow-compiler/internal/domain/validators/structural.go:11-58", "grep kagent excl docs → 0", "strategy.md:290"]
  - to_agent: "5.13 Governance"
    note: "Positioning doc (2026-05-28) claims 'Temporal + LangGraph adapters' — a competitive-doc drift that should be reconciled to code reality (temporal+argo; argo stubbed; langgraph=capability)."
    evidence: ["docs/architecture/2026-05-28-competitive-positioning.md:28", "services/engine-adapter/AGENTS.md:4"]

recommendations:
  - priority: "P0"
    action: "Re-lead the competitive narrative on the SHIPPED, rare edge (compile-time structural IR validation + no-SDK capability contract) and stop leading with multi-engine portability until Argo/a 2nd engine actually interprets the IR. Correct the positioning table's 'LangGraph adapter = engine portability' claim."
    rationale: "The current lead claim is CONTRADICTED at the execution boundary and in the positioning doc itself — exactly the drift class the diligence exists to catch (Part 1 §1.10). Leading with a half-built moat against a CNCF-backed rival is the worst possible footing."
  - priority: "P1"
    action: "Either BUILD a real Kagent capability adapter + e2e demo (Kagent agent registered via AgentService, dispatched inside a zynax Workflow) or DROP the 'complementary to Kagent' claim. Prose-only complementarity is not buyer-credible."
    rationale: "The story's only credibility comes from working code; absent that, a Kagent buyer sees a smaller un-adopted rival, not a complement (positioning.md:75 vs grep kagent → 0)."
  - priority: "P1"
    action: "Ship multi-engine EXECUTION parity (Argo interprets the IR + a cross-engine parity test) before a competitor copies the contract layer — this is the only thing that converts a shallow, copyable moat into a durable one."
    rationale: "The copyable part of the moat is public (ADR-012/015); the durable part is unbuilt. The window to own it closes as Restate/Dapr/Kagent mature (E7)."
  - priority: "P2"
    action: "Add Restate and Dapr Agents as first-class entries in the competitive matrix and refresh it for June 2026 (Kagent now has HITL/ArgoCD/GitOps; Dapr Agents at v1.0; Restate ships AI-agent orchestration). The 2026-04-30/05-28 docs predate these shifts."
    rationale: "The current matrices understate live rivals and overstate the 'unoccupied gap' (competitive-analysis.md:22-25), risking credibility with a technical buyer who knows the 2026 landscape."
```

---

## (b) §6.2 Prose section

## 5.19 Competitive Analysis — Score: 5 (Medium)

**Mission recap:** Assess the competitive landscape — especially the named rival Kagent — and judge whether Zynax's differentiation is defensible, legible, and survives contact with a buyer.

**Verdict:** Zynax has one genuinely rare, shipped, hard-to-copy edge (compile-time structural validation of the workflow IR) and a clean no-SDK capability contract — but it leads its marketing with the differentiator that is only half-built (multi-engine portability: Temporal interprets the IR, Argo is a non-interpreting stub, LangGraph-as-engine does not exist). Against the named rival Kagent, Zynax loses on every buyer-visible axis — CNCF Sandbox backing, web UI, MCP/OpenAPI tool discovery, HITL, multi-LLM, K8s-native, and now ArgoCD/GitOps integration — while carrying 0 stars/forks/adopters. The "complementary to Kagent" story is architecturally possible but prose-only (zero Kagent code), and Kagent has since absorbed much of the "control plane" Zynax claims it lacks. A 5 (Adequate / differentiated but contestable, with a backed rival ahead on legibility).

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Engine-agnostic IR + multi-engine portability (headline moat) | 5 | High | `workflow_compiler.proto:205-241`; `engine.go:17-41`; `argo-ir-interpreter.yaml:10-14`; `main.go:177-178`; `engine-adapter/AGENTS.md:4` |
| No-SDK AgentService capability contract | 7 | High | `agent.proto:3-7,34-47`; `task-broker/.../service.go:18,48`; E7 Restate SDKs |
| Compile-time structural IR validation | 7 | High | `validators/structural.go:11-58`; `competitive-analysis.md:261` |
| GitOps-native + Compose-AND-K8s deploy | 5 | Medium | `cmd/zynax/cmd/apply.go:60,104`; `gitops/watcher.go:37,107`; `positioning.md:26`; E7 Kagent+ArgoCD |
| Kagent head-to-head (where each wins) | 4 | High | `positioning.md:32,35,36`; E7 thenewstack/kagent.dev/cncf#360; `strategy.md:290` |
| "Complementary to Kagent" buyer credibility | 3 | Medium | grep kagent excl docs → 0; `positioning.md:75-78`; E7 Kagent HITL/ArgoCD |
| Time-to-parity of the moat | 5 | Medium | `ADR-012`/`ADR-015` (public design); E7 Dapr v1.0 / Restate AI-agent orchestration |

**Drift test:**
- *"Engine-agnostic IR — run on Temporal OR Argo (OR LangGraph) without rewrite" (boldest differentiator)* → **PARTIAL.** VERIFIED at the contract/submission boundary (one IR, one port, both engines conform); CONTRADICTED at execution (Argo is a non-interpreting stub — `argo-ir-interpreter.yaml:10-14`) and CONTRADICTED on LangGraph (positioning doc says "Temporal + LangGraph adapters" `positioning.md:28`, but the engine switch is temporal|argo `main.go:177-178` and LangGraph is a capability adapter, not an engine — `engine-adapter/AGENTS.md:4`).
- *"Complementary, not competitive with Kagent"* → **PARTIAL.** The AgentService contract makes it possible (`agent.proto:3-7`), but no Kagent adapter/example/test exists (grep → 0), and Kagent shipping HITL + ArgoCD/GitOps erodes the "Kagent lacks a control plane" premise (E7; `positioning.md:77`).
- *"No tool combines YAML + formal IR + capability registry + pluggable engines — gap is real and unoccupied"* → **PARTIAL.** The formal compile-step is genuinely rare (`structural.go:11-58`; `competitive-analysis.md:261`), but Dapr Agents (v1.0) and Restate (AI-agent orchestration) now crowd the adjacent space the 2026-04-30 doc called empty (E7).

**Positioning matrix (June 2026, code-grounded · E7-grounded):**

| Dimension | Zynax | Kagent | Temporal | Argo Workflows | LangGraph | Restate | Dapr (Workflows/Agents) |
|---|---|---|---|---|---|---|---|
| Core abstraction | Engine-neutral Workflow **IR** (`workflow_compiler.proto:205-241`) | K8s **CRDs**, pod-per-agent (E7) | Durable code workflows (E7) | K8s DAG jobs (E7) | Python `StateGraph` (E7) | Durable code + journal (E7) | Durable workflow engine + agents (E7) |
| Engine portability | ✅ at IR contract / ❌ at execution (Temporal-only interpret; Argo stub — `argo-ir-interpreter.yaml:10-14`) | ❌ ADK/K8s lock-in (E7) | N/A (is the engine) | N/A | N/A (is the framework) | ❌ Restate runtime | ❌ Dapr runtime |
| Compile-time topology validation | ✅✅ shipped (`validators/structural.go:11-58`) | ❌ runtime | ❌ runtime | ⚠️ DAG-shape only | ❌ runtime | ❌ runtime | ❌ runtime |
| No-SDK / language-agnostic agents | ✅✅ any gRPC service (`agent.proto:3-7`) | ✅ any container (E7) | ✅ multi-lang SDK | ✅ container | ❌ Python-first (E7) | ✅ 5-lang SDK (E7) | ✅ any language (E7) |
| LLM-native built-in | ⚠️ via llm-adapter (`agents/adapters/llm/`) | ✅ ModelConfig, 7+ providers (E7) | ❌ | ❌ | ✅ built-in (E7) | ⚠️ via OpenAI SDK (E7) | ✅ built-in (E7) |
| Human-in-the-loop | ✅ semantic IR state (`workflow_compiler.proto:122-135` STATE_TYPE_HUMAN_IN_THE_LOOP) | ✅ shipped (E7) | ✅ signals | ⚠️ | ✅ interrupts (E7) | ✅ resilient approvals (E7) | ✅ |
| Web UI | ❌ none (M8 CLAIMED; no `*ui*` in repo) | ✅ GUI+CLI (E7) | ✅ Temporal UI | ✅ Argo UI | ⚠️ LangSmith (E7) | ⚠️ Restate UI (E7) | ⚠️ |
| GitOps-native | ✅ `zynax apply` + watcher (`apply.go:60`; `watcher.go:37`) | ✅ ArgoCD-integrated (E7) | ❌ | ✅ (Argo CD family) | ❌ | ⚠️ | ⚠️ |
| Deploy surface | ✅ Compose **and** K8s (`positioning.md:26`) | ❌ K8s-only (E7) | Any | K8s-only | Library / Platform (E7) | Single binary / K8s (E7) | K8s-native |
| CNCF status | ❌ M8 target (`positioning.md:35`) | ✅ Sandbox (cncf#360) | ❌ | ✅ Graduated family | ❌ | ❌ | ✅ Incubating |
| Production-proven / adopters | ❌ v0.4-0.6, **0 adopters** (`strategy.md:290`) | Partial, growing (E7) | ✅ | ✅ | ✅ | ✅ (early) | ✅ v1.0 (E7) |

**Reading:** Zynax leads only two rows defensibly — compile-time topology validation (uniquely ✅✅) and the no-SDK gRPC contract (tied) — plus the Compose-AND-K8s deploy surface vs Kagent's K8s-only. On every maturity/legibility row (UI, CNCF, adopters, LLM-native) a backed rival is ahead. The portability row, its headline, is the one marked half-built.

**Red flags (severity-ordered):**
1. **Critical** — Kagent out-positions Zynax on every buyer-visible axis (CNCF, UI, MCP/OpenAPI, HITL, multi-LLM, ArgoCD/GitOps) while Zynax has 0 adopters and a half-built portability moat (E7; `positioning.md:32,35,36`; `strategy.md:290`; `argo-ir-interpreter.yaml:10-14`).
2. **High** — The boldest differentiator is overstated in Zynax's own positioning doc ("Temporal + LangGraph adapters") vs code (temporal|argo; Argo stubbed; LangGraph=capability) (`positioning.md:28`; `main.go:177-178`; `engine-adapter/AGENTS.md:4`).
3. **High** — "Complementary to Kagent" is prose-only (zero Kagent code) and undercut by Kagent absorbing control-plane features (grep → 0; `positioning.md:75-78`; E7).
4. **Medium** — Short time-to-parity on the copyable layer (public IR/port/validators) while Restate/Dapr already ship production durable agent orchestration (`ADR-012/015`; E7).
5. **Low** — The strongest shipped edge (compile-time validation) is under-marketed vs the half-built portability lead (`structural.go:11-58`; `competitive-analysis.md:261`).

**Green flags:**
- Compile-time structural IR validation — rare, shipped, hard-to-copy without an IR (`validators/structural.go:11-58`).
- Minimal, legible no-SDK AgentService contract — the only credible technical basis for any future complementary integration (`agent.proto:3-7,34-47`).
- Engine-neutral IR + clean WorkflowEngine port (Wave A 5.1 scored 9/8) — the structure for a real moat exists if execution parity ships (`workflow_compiler.proto:205-241`; `docs/due-diligence/2026-06-20-dd-wave-a-findings.md:38-55`).

**Open questions / unknowns:** Why Zynax wins a head-to-head today (honest answer: only a narrow buyer who needs non-Temporal/non-K8s portability + compile-time validation and tolerates zero adopters); how long the moat lasts (copyable layer is short-fuse, deep layer unbuilt by all); whether Kagent is building an IR/portability layer (UNKNOWN); live traction deltas (UNKNOWN); any private design-partner (none found).

**Recommendations:** P0 — re-lead on shipped validation + no-SDK contract, correct the "LangGraph adapter = portability" claim, stop marketing portability as proven until a 2nd engine interprets the IR. P1 — build a real Kagent adapter+demo or drop the complementary claim; ship multi-engine execution parity + a cross-engine test before the contract layer is copied. P2 — add Restate/Dapr to the matrix and refresh it for June 2026.

**Cross-references:** 5.1 Architecture (consumed: portability real at IR, stubbed at Argo — not re-scored); 5.4/5.17/5.18 (D2 feeds them: lead with validation, fix Kagent story, CNCF/UI/adopter gap is the dominant deficit); 5.13 Governance (positioning-doc drift on "LangGraph adapter").

---

### E7 sources cited

- Kagent: thenewstack.io/meet-kagent-open-source-framework-for-ai-agents-in-kubernetes; kagent.dev/docs/kagent/introduction/what-is-kagent; github.com/cncf/sandbox/issues/360
- Restate: restate.dev/what-is-durable-execution; restate.dev/blog/durable-orchestration-for-ai-agents-with-restate-and-openai-sdk
- Dapr Agents: github.com/dapr/dapr-agents; docs.dapr.io/developing-ai/dapr-agents/dapr-agents-why
- LangGraph: github.com/langchain-ai/langgraph; docs.langchain.com/oss/python/langgraph/durable-execution; alphabold.com/langgraph-agents-in-production
- Temporal / Argo: docs.temporal.io; argo-workflows.readthedocs.io (baseline, per repo competitive-analysis.md:805-807)
<!-- SPDX-License-Identifier: Apache-2.0 -->
# Agent 5.20 — Enterprise Adoption — Wave C (product/market/governance) — Issue #1404

> ROLE: Enterprise architect / buyer-side platform evaluator (Fortune-500 lens).
> REPO: the repository root · HEAD `e3135a6` · branch `main` · READ-ONLY audit.
> Evidence rule (§0.4): every factual claim carries `path:line` / command→output, or is
> marked `UNKNOWN`. Roadmap/marketing statements are labelled `CLAIMED`, never `VERIFIED`.
> Dimension group D14 (feeds D15). Consumes Wave A Security (5.2) and DevOps (5.9) per §3.1 —
> cited, not re-scored.

---

## (a) §3.4 Handoff packet

```yaml
agent: "5.20 Enterprise Adoption"
wave: "C"
dimension_groups: ["D14"]
overall_score: 4
overall_confidence: "High"

sub_scores:
  - dimension: "Enterprise authN/Z — RBAC, SSO/OIDC, identity"
    score: 2
    confidence: "High"
    justification: "No RBAC, no SSO, no OIDC, no per-caller identity. AuthN is a single shared static bearer key; authZ is binary (valid key = full mutate access). OIDC/JWT is roadmap-only and the BDD step for permission scopes is an unimplemented `pending` stub."
    evidence:
      - "services/api-gateway/internal/api/auth.go:13-26 — requireBearer: one shared static key, no identity, no scopes, no roles"
      - "services/api-gateway/tests/steps_test.go:242 — permission step `pending` (// M6 OIDC/JWT) — scoped-token authZ NOT implemented"
      - "SECURITY.md:90 — 'OIDC/JWT authentication replacing static bearer token' listed under 'Planned (M6+)' (CLAIMED, not shipped)"
      - "docs/product/strategy.md:321,360 — 'Enterprise-readiness (RBAC/SSO/multi-tenancy)' scheduled Post-M8 (CLAIMED)"
      - "cmd: grep -rniE 'rbac|oidc|sso|saml|oauth' services/ protos/ helm/ → only a pending BDD stub; zero implementing code"
  - dimension: "Multi-tenant isolation"
    score: 2
    confidence: "High"
    justification: "Namespace is correlation/quota metadata only, not an isolation boundary. All Temporal workflows execute in one process-wide configured namespace regardless of the manifest namespace; the 'namespace field is cosmetic' architecture finding still holds at HEAD for execution isolation."
    evidence:
      - "services/engine-adapter/internal/infrastructure/temporal.go:54-58,76 — Temporal namespace is a single config value (e.namespace); NOT derived per-workflow from the IR"
      - "services/api-gateway/internal/api/handler.go:66-72 — ?namespace attached as correlation metadata on gRPC hops only (canvas C.2 comment), not an isolation primitive"
      - "docs/architecture/2026-05-20-principal-architect-review.md:350 — 'namespace field exists everywhere but is functionally cosmetic — no isolation is enforced'"
      - "docs/reviews/04-architecture-gaps.md:148 — Multi-tenancy: '❌ Cosmetic only; all workflows share Temporal default namespace'; ADR-021 still PROPOSED (01-decision-ledger.md:208)"
      - "due-diligence framework:238 — 'true multi-tenant isolation deferred post-M8' (project's own non-goal)"
  - dimension: "Compliance & audit — audit log, traceability, certifications, data residency"
    score: 3
    confidence: "High"
    justification: "No audit-log middleware on mutating endpoints; mutations are bearer+rate-limit gated only. No SOC2/HIPAA/GDPR/FedRAMP posture. Data residency undocumented. Partial offsets: GitOps YAML-in-Git audit trail, X-Request-ID/W3C trace correlation, and a Postgres+CloudEvents history exist (CLAIMED as audit substrate)."
    evidence:
      - "services/api-gateway/internal/api/handler.go:41-50 — RegisterRoutes wraps requireBearer + rate-limit only; NO audit-write on apply/delete/events"
      - "docs/architecture/2026-05-18-external-architectural-review.md:331 — '...no audit log, no request-level logging middleware on POST /apply, DELETE /workflows/{id}...'; real audit log is a future item (:795)"
      - "cmd: grep -rniE 'audit log|audit trail' services/ → zero audit-logging implementation in service code"
      - "docs/product/2026-06-18-market-fit-review.md:108 — 'audit trail via Postgres + CloudEvents' is the CLAIMED substrate; enterprise governance marked Partial, Post-M8"
      - "cmd: grep -rniE 'data residency|gdpr|hipaa|soc2|fedramp' docs/ SECURITY.md README.md → only aspirational mentions in architecture reviews; no shipped posture (UNKNOWN/absent)"
      - "ROADMAP.md:160 — M6 == 'Kubernetes Production-Ready'; no compliance-cert scope anywhere"
  - dimension: "Policy enforcement"
    score: 6
    confidence: "High"
    justification: "Real and shipped — but via env-config, not the Policy CRD. workflow-compiler runs a PolicyGate (routing+quota); engine-adapter has a QuotaChecker (RESOURCE_EXHAUSTED); api-gateway has per-IP token-bucket rate limiting. The committed policy.schema.json is validated but the operator that consumes the Policy manifest is still 'implemented in M6' per the schema's own doc."
    evidence:
      - "services/workflow-compiler/cmd/workflow-compiler/main.go:143-155 — buildPolicyGate(routing,quotas) wired from env config"
      - "services/engine-adapter/internal/infrastructure/quota_check_test.go:35 — 'quota exceeded returns RESOURCE_EXHAUSTED' (executed-tested)"
      - "services/api-gateway/internal/api/ratelimit.go:17-69 — per-IP token-bucket Middleware on /apply and /events"
      - "spec/schemas/policy.schema.json:5 — 'Policy enforcement is implemented in M6 (Kubernetes operator); this schema is committed in M2 so the compiler can validate policy references' — CRD-driven enforcement is the operator's job"
  - dimension: "Operability — observability, runbooks, upgrade/migration, LTS"
    score: 6
    confidence: "High"
    justification: "Strong OTEL/Uptrace wiring (off by default) + real Helm topology + a genuine Postgres backup/restore/upgrade README. BUT no operational runbooks (incident/DR), no published SLOs, no LTS branch, single-instance Postgres SPOF (HA = future CloudNativePG)."
    evidence:
      - "services/api-gateway/cmd/api-gateway/main.go:90 — zynaxobs.InitTracer wired in service code (OTEL real, not doc-only)"
      - "docs/migration-v0.6.md:5-12 — v0.5→v0.6 is additive, no contract breaks; clear opt-in upgrade checklist"
      - "helm/charts/postgres/README.md:43-68 — pg_dump/restore, logical-replication, pg_upgrade major-version runbook (real)"
      - "helm/zynax-api-gateway/templates/pdb.yaml:3 + networkpolicy.yaml:3 — PDB + NetworkPolicy shipped per service"
      - "cmd: ls docs/runbooks docs/operations docs/ops → none exist; no incident/DR/on-call runbook in repo"
      - "helm/charts/postgres/README.md:67-68 — HA successor (CloudNativePG, ADR-026) is the 'documented HA successor' — i.e. single-instance Postgres SPOF today; SECURITY.md:7-12 supported-versions is rolling (main/latest/previous), no LTS"
      - "Wave A 5.9 (final.md:1185-1189,1251) — no DORA/pipeline telemetry, no flaky-test harness; SLO *values* unpublished (CLAIMED M8)"
  - dimension: "Multi-platform — cloud portability, air-gapped/on-prem, K8s + compose"
    score: 4
    confidence: "High"
    justification: "Vendor-neutral K8s (Helm umbrella + cert-manager/postgres/nats/temporal/uptrace subcharts) and Docker-Compose both ship → cloud-agnostic and self-hostable in principle. But production SERVICE images are amd64-only (no arm64 K8s / Apple-Silicon), and no air-gapped/private-registry install guide exists."
    evidence:
      - "cmd: find helm infra -maxdepth 2 -type d → zynax-umbrella + 8 service charts + charts/{cert-manager,postgres,nats,temporal,uptrace}; infra/docker-compose present"
      - "Wave A 5.9 (final.md:1208-1218) — production service/adapter container images are amd64-only (ci.yml:861 builds linux/amd64; release.yml retag-only); arm64 K8s nodes cannot run official images"
      - "cmd: grep -rniE 'air.?gap|on-prem|private registry|offline install' docs/ README.md infra/ → no air-gapped/on-prem install runbook found (UNKNOWN/absent)"
      - "docs/migration-v0.6.md:79-80 — Uptrace runnable via Compose overlay OR in-cluster Helm (portability point)"
  - dimension: "Supportability — who answers a 2am production incident"
    score: 3
    confidence: "High"
    justification: "A genuine SECURITY-vulnerability response SLA exists (48h ack), but there is NO production-incident support model: no SUPPORT.md, no MAINTAINERS.md, no on-call/escalation, no commercial SLA. Bus factor = 1 (Wave A 5.24). A Fortune-500 buyer has no contracted entity to call at 2am."
    evidence:
      - "SECURITY.md:23-26 — vuln-disclosure SLA (ack 48h; Critical 7d / High 30d) — security-only, not operational support"
      - "cmd: ls SUPPORT.md MAINTAINERS.md → neither exists"
      - "cmd: grep -rniE 'on-call|escalation|incident response|2am|enterprise support' docs/ → only SPDD/dev-advisory orchestrator escalation; no production support model"
      - "Wave A 5.24 (final.md:50) — 'Bus factor = 1' (one human identity in git shortlog); MAINTAINERS.md open (#494)"
      - "docs/product/strategy.md:360 — 'SLAs' are an Enterprise-tier roadmap item (CLAIMED)"

drift_test:
  - claim: "Zynax is 'K8s Production-Ready' (M6 / v0.5.0 milestone title)."
    result: "PARTIAL / OVERSTATED for ENTERPRISE production"
    evidence:
      - "ROADMAP.md:160 + README.md:407 — 'K8s Production-Ready' is the shipped-milestone label."
      - "Production-credible for the platform substrate (Helm, mTLS available, container hardening, supply-chain — Wave A 5.2/5.9), BUT NOT enterprise-production-ready: no RBAC/SSO (auth.go:13-26), cosmetic multi-tenancy (temporal.go:54-58), no audit log (handler.go:41-50), no support model (no SUPPORT.md/MAINTAINERS.md), single-Postgres SPOF (postgres README:67-68)."
      - "The project itself scopes RBAC/SSO/multi-tenancy/SLAs/operator-runbooks to Post-M8 (strategy.md:321,360) — i.e. 'production-ready' is true for the K8s deploy mechanics, not for enterprise governance."
  - claim: "SECURITY.md:44 — 'Multi-arch container images (linux/amd64 + linux/arm64) for ALL service + tools images (✅ #489)'."
    result: "CONTRADICTED (for service images)"
    evidence:
      - "SECURITY.md:44 asserts arm64 for all service images as shipped."
      - "Wave A 5.9 VERIFIED (final.md:1208-1218): production service/adapter images are amd64-only (ci.yml:861; release.yml is retag-only and adds no platform). Multi-arch holds for tools/CLI, NOT services. SECURITY.md overstates the shipped artifact — a Part-1 §1.10-class doc-vs-artifact drift."
  - claim: "SECURITY.md:89 — 'mTLS between all platform services — ✅ shipped #488'."
    result: "CONTRADICTED / fails-open"
    evidence:
      - "Wave A 5.2 (final.md:491-497): mTLS code falls open to insecure.NewCredentials() when certs unset (tlscreds.go:20); chart default is insecure; api-gateway + workflow-compiler production overlays omit tlsSecretName. 'mTLS on all services' overstates the shipped default."

red_flags:
  - severity: "Critical"
    finding: "No enterprise identity layer: authN is a single shared static bearer key, authZ is binary, and there is NO RBAC, SSO, or OIDC. A Fortune-500 IdP cannot be integrated; there is no per-user identity to audit or revoke. This is a hard procurement blocker — it fails the first checkbox of nearly every enterprise security questionnaire."
    evidence:
      - "services/api-gateway/internal/api/auth.go:13-26"
      - "services/api-gateway/tests/steps_test.go:242 (OIDC/JWT permission step still `pending`)"
      - "SECURITY.md:90 (OIDC 'Planned M6+'); docs/product/strategy.md:321,360 (RBAC/SSO Post-M8)"
  - severity: "Critical"
    finding: "No multi-tenant isolation: namespace is metadata only; all workflows execute in one shared Temporal namespace. A regulated buyer cannot isolate teams/customers — noisy-neighbour and data-bleed risk. Marketed as 'Kubernetes for AI workflows' yet lacks K8s's foundational tenancy primitive."
    evidence:
      - "services/engine-adapter/internal/infrastructure/temporal.go:54-58,76"
      - "docs/architecture/2026-05-20-principal-architect-review.md:350; docs/reviews/04-architecture-gaps.md:148 (ADR-021 still PROPOSED)"
  - severity: "High"
    finding: "No audit log on mutating control-plane operations (apply / delete / publish-event). Bearer+rate-limit only; no who-did-what record. Combined with the shared key, an action cannot be attributed to a principal — fails SOC2 CC6/CC7 audit-trail expectations."
    evidence:
      - "services/api-gateway/internal/api/handler.go:41-50"
      - "docs/architecture/2026-05-18-external-architectural-review.md:331,795 (real audit log is future work)"
  - severity: "High"
    finding: "No production support / incident model: no SUPPORT.md, no MAINTAINERS.md, no on-call/escalation/SLA, bus factor = 1. There is no entity a buyer can contract for a 2am outage. (The 48h SLA in SECURITY.md is vuln-disclosure only, not operational.)"
    evidence:
      - "cmd: ls SUPPORT.md MAINTAINERS.md → absent"
      - "SECURITY.md:23-26 (vuln-only SLA); Wave A 5.24 final.md:50 (bus factor 1)"
  - severity: "Medium"
    finding: "Single-instance Postgres SPOF + no operational runbooks (incident/DR/restore-drill). A credible backup/upgrade README exists, but HA is explicitly a future CloudNativePG migration (ADR-026) and there are no docs/runbooks for failure scenarios."
    evidence:
      - "helm/charts/postgres/README.md:67-68 (HA is the documented future successor)"
      - "cmd: ls docs/runbooks docs/operations → none"
  - severity: "Medium"
    finding: "Doc-vs-artifact drift undermines a security questionnaire: SECURITY.md:44 claims arm64 service images (false — amd64-only, Wave A 5.9) and SECURITY.md:89 claims mTLS shipped on all services (fails-open, Wave A 5.2). These overstated controls are exactly what an enterprise security review will probe and find wanting."
    evidence: ["SECURITY.md:44,89", "Wave A 5.9 final.md:1208-1218", "Wave A 5.2 final.md:491-497"]

green_flags:
  - strength: "Policy enforcement is real and tested: PolicyGate (routing+quota) in workflow-compiler, QuotaChecker returning RESOURCE_EXHAUSTED in engine-adapter, per-IP rate-limiting at the gateway — more than most pre-1.0 control planes ship."
    evidence: ["services/workflow-compiler/cmd/workflow-compiler/main.go:143-155", "services/engine-adapter/internal/infrastructure/quota_check_test.go:35", "services/api-gateway/internal/api/ratelimit.go:17-69"]
  - strength: "Genuine observability stack: OTEL traces/metrics/logs wired in service code (off by default) + Uptrace deployable via Compose overlay or Helm subchart — an operability foundation enterprises expect."
    evidence: ["services/api-gateway/cmd/api-gateway/main.go:90", "docs/observability/opentelemetry.md:13", "docs/migration-v0.6.md:70-85"]
  - strength: "Clean, additive upgrade story + real DB lifecycle: migration-v0.6.md (no contract breaks, opt-in checklist) and a concrete Postgres pg_dump/logical-replication/pg_upgrade runbook."
    evidence: ["docs/migration-v0.6.md:5-12,143-153", "helm/charts/postgres/README.md:43-68"]
  - strength: "Cloud-neutral deployment: vendor-agnostic Helm umbrella (cert-manager/postgres/nats/temporal/uptrace subcharts) plus Docker-Compose — self-hostable on any conformant K8s; no managed-cloud lock-in."
    evidence: ["cmd: find helm/charts -maxdepth 2 -type d → cert-manager,postgres,nats,temporal,uptrace", "helm/zynax-api-gateway/templates/{pdb,networkpolicy}.yaml:3"]

open_questions:
  - "Is per-workflow Temporal-namespace isolation (or per-tenant task-queue) on the M8 roadmap, or does multi-tenancy remain a hard Post-M8 deferral with no design (ADR-021 still PROPOSED)?"
  - "Will the audit-log substrate (Postgres + CloudEvents, claimed in the market-fit review) ever surface a queryable, tamper-evident audit API, or is it incidental event storage?"
  - "Is an air-gapped / private-registry install path intended (digest pinning + Helm make it feasible), or is a public GHCR pull a hard dependency?"
  - "Who is the contractual support entity for a paying enterprise — is a commercial/SLA tier real or strategy-doc aspiration (strategy.md:360)?"

unknowns:
  - "Data-residency posture — no documentation found in code, SECURITY.md, or observability/infra docs. Marked absent/UNKNOWN, not failed."
  - "Whether any external enterprise pilot has deployed Zynax in production (no named adopters; bus factor 1 per Wave A 5.24) — adoption signal not verifiable from repo."
  - "Live GHCR signature/attestation existence inherited as UNKNOWN from Wave A (5.2/5.9 — cosign not installed, no registry access)."

cross_references:
  - to_agent: "5.2 Security"
    note: "I rely on (do not re-score) the mTLS fail-open finding and the single-shared-key/no-RBAC finding; both directly drive my authN/Z (2) and compliance (3) scores and the SECURITY.md:89 drift contradiction."
    evidence: ["Wave A final.md:491-497,503-505 (tlscreds.go:20; auth.go:14)"]
  - to_agent: "5.9 DevOps"
    note: "I rely on the amd64-only service-image finding (multi-platform sub-score) and the no-DORA-telemetry/no-SLO finding (operability sub-score); the amd64 fact contradicts SECURITY.md:44."
    evidence: ["Wave A final.md:1208-1218,1185-1189"]
  - to_agent: "5.16 Scalability (Wave B)"
    note: "Single-instance Postgres SPOF and cosmetic namespace constrain both tenancy and scale; relevant to the scalability zone's data-layer SPOF assessment."
    evidence: ["helm/charts/postgres/README.md:67-68", "services/engine-adapter/internal/infrastructure/temporal.go:54-58"]
  - to_agent: "5.13 Governance / 5.24 Repo Health"
    note: "Absence of SUPPORT.md / MAINTAINERS.md and bus factor 1 are simultaneously a governance gap and the root of the supportability red flag."
    evidence: ["cmd: ls SUPPORT.md MAINTAINERS.md → absent", "Wave A final.md:50"]

recommendations:
  - priority: "P0"
    action: "Replace the static bearer key with OIDC/JWT + role claims (the architecture review's R2 / ADR-020 'Planned' work) and add per-principal authZ. This unblocks the single largest procurement gate."
    rationale: "No enterprise can integrate its IdP or attribute actions today; this fails the first page of every security questionnaire (auth.go:13-26)."
  - priority: "P0"
    action: "Stop labelling SECURITY.md controls as shipped when they are not: correct line 44 (services are amd64-only, not arm64) and line 89 (mTLS fails-open, not enforced-on-all). Add a 'not for enterprise production' caveat alongside the 'K8s Production-Ready' milestone label."
    rationale: "Overstated controls are the project's documented failure mode (Part-1 §1.10) and will be caught in diligence, eroding trust on otherwise strong supply-chain work."
  - priority: "P1"
    action: "Implement real multi-tenant isolation (per-namespace Temporal namespace or per-tenant task-queue) and finalize ADR-021 (still PROPOSED)."
    rationale: "Tenancy is the K8s primitive the product's own positioning invokes; cosmetic namespaces block any shared/regulated deployment (temporal.go:54-58)."
  - priority: "P1"
    action: "Add an audit log on all mutating control-plane operations and publish SUPPORT.md + MAINTAINERS.md with an escalation/incident model (even community-tier)."
    rationale: "Closes the SOC2 audit-trail gap and gives a buyer a named support contact — both procurement prerequisites (handler.go:41-50; no SUPPORT.md)."
  - priority: "P2"
    action: "Author operator runbooks (incident/DR/restore-drill) and ship arm64 service images or document amd64-only as a supported-platform constraint; publish SLO values."
    rationale: "Turns the existing Helm/OTEL/Postgres-README foundation into an operable, portable enterprise package."
```

---

## (b) §6.2 Prose section

## 5.20 Enterprise Adoption — Score: 4 (High)

**Mission recap:** From a Fortune-500 buyer-side lens, assess whether a large enterprise could adopt and operate Zynax — compliance/audit, enterprise authN/Z, operability, multi-platform support, and supportability — and drift-test the "production-ready" claim against the gaps.

**Verdict:** Zynax has a strong *platform* substrate but is **not enterprise-adoptable today** — it is pilot-able, not procurement-approvable. The three controls a Fortune-500 security review checks first are all missing or cosmetic: there is no RBAC/SSO/OIDC (a single shared static bearer key, with the scoped-token BDD test still a `pending` stub), no real multi-tenant isolation (all workflows run in one Temporal namespace; the manifest namespace is correlation metadata only), and no audit log on mutating operations. Real strengths exist around the edges — shipped policy/quota/rate-limit enforcement, genuine OTEL/Uptrace observability, a clean additive upgrade story with a credible Postgres backup/restore runbook, and vendor-neutral Helm + Compose deployment — but they sit on top of an identity and tenancy layer that an enterprise cannot accept. The project itself scopes RBAC/SSO/multi-tenancy/SLAs to Post-M8, so the gaps are acknowledged, not hidden; the diligence concern is the doc-vs-artifact drift in SECURITY.md (arm64 and mTLS claimed as shipped when they are not) layered under a "K8s Production-Ready" milestone label.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Enterprise authN/Z (RBAC/SSO/OIDC) | 2 | High | `auth.go:13-26`; `steps_test.go:242` (pending); `SECURITY.md:90`; `strategy.md:321,360` |
| Multi-tenant isolation | 2 | High | `temporal.go:54-58,76`; `handler.go:66-72`; principal-architect-review:350; reviews/04:148 |
| Compliance & audit (audit log / certs / residency) | 3 | High | `handler.go:41-50`; external-review:331,795; grep audit/SOC2/GDPR → absent |
| Policy enforcement | 6 | High | `workflow-compiler main.go:143-155`; `quota_check_test.go:35`; `ratelimit.go:17-69`; `policy.schema.json:5` |
| Operability (OTEL / runbooks / upgrade / LTS) | 6 | High | `api-gateway main.go:90`; `migration-v0.6.md:5-12`; `postgres README:43-68`; no runbooks dir; Wave A 5.9 |
| Multi-platform (cloud / air-gap / K8s+compose) | 4 | High | `find helm/charts`; Wave A 5.9 (amd64-only); no air-gap guide |
| Supportability (2am incident / SLA / escalation) | 3 | High | `SECURITY.md:23-26` (vuln-only); no SUPPORT.md/MAINTAINERS.md; Wave A 5.24 (bus factor 1) |

**Drift test:**
- *"K8s Production-Ready" (M6/v0.5.0 milestone)* → **PARTIAL / OVERSTATED for enterprise.** True for the K8s deploy mechanics; false for enterprise governance — no RBAC/SSO, cosmetic tenancy, no audit log, no support model, single-Postgres SPOF (`ROADMAP.md:160`, `README.md:407` vs `auth.go:13-26`, `temporal.go:54-58`).
- *SECURITY.md:44 "multi-arch arm64 for all service images (✅)"* → **CONTRADICTED.** Service images are amd64-only (Wave A 5.9, `ci.yml:861`).
- *SECURITY.md:89 "mTLS between all platform services (✅ shipped)"* → **CONTRADICTED / fails-open.** Code falls open to insecure creds; prod overlays omit `tlsSecretName` (Wave A 5.2).

**Red flags (severity-ordered):**
1. **Critical** — No enterprise identity layer (RBAC/SSO/OIDC); shared static bearer key (`auth.go:13-26`, `steps_test.go:242`).
2. **Critical** — No multi-tenant isolation; namespace cosmetic; all workflows share one Temporal namespace (`temporal.go:54-58`, reviews/04:148).
3. **High** — No audit log on mutating control-plane operations (`handler.go:41-50`).
4. **High** — No production support/incident model; no SUPPORT.md/MAINTAINERS.md; bus factor 1 (Wave A 5.24).
5. **Medium** — Single-instance Postgres SPOF + no operational runbooks (`postgres README:67-68`).
6. **Medium** — SECURITY.md overstates arm64 and mTLS as shipped (`SECURITY.md:44,89` vs Wave A 5.2/5.9).

**Green flags:**
- Shipped, tested policy/quota/rate-limit enforcement (`workflow-compiler main.go:143-155`, `quota_check_test.go:35`, `ratelimit.go:17-69`).
- Real OTEL/Uptrace observability, off by default (`api-gateway main.go:90`, `opentelemetry.md:13`).
- Clean additive upgrade + concrete Postgres lifecycle runbook (`migration-v0.6.md:5-12`, `postgres README:43-68`).
- Cloud-neutral Helm umbrella + Compose; PDB/NetworkPolicy per service (`find helm/charts`, `pdb.yaml:3`).

**Open questions / unknowns:** Per-tenant isolation on the roadmap? Will the Postgres+CloudEvents "audit substrate" become a queryable audit API? Air-gapped install path intended? Is a commercial/SLA support tier real? Data-residency posture is absent/UNKNOWN; no named production adopter; GHCR signature existence inherited UNKNOWN from Wave A.

**Recommendations:** P0 — ship OIDC/JWT + role claims and correct the overstated SECURITY.md controls; P1 — implement real namespace tenancy (finalize ADR-021) and add an audit log + SUPPORT.md/MAINTAINERS.md; P2 — author operator runbooks, ship arm64 (or document amd64-only), publish SLOs.

**Cross-references:** 5.2 Security (mTLS fail-open, shared key/no-RBAC — drives authN/Z + compliance scores); 5.9 DevOps (amd64-only, no DORA/SLO — drives multi-platform + operability); 5.16 Scalability (Postgres SPOF, cosmetic namespace); 5.13/5.24 Governance/Repo Health (no SUPPORT.md/MAINTAINERS.md, bus factor 1).
<!-- SPDX-License-Identifier: Apache-2.0 -->
# Agent 5.21 — CNCF Readiness — Wave C — Issue #1404

> Role: CNCF TOC reviewer. Map Zynax against CNCF Sandbox criteria at HEAD and judge readiness
> honestly. READ-ONLY audit. Consumes Wave A (5.2 Security, 5.24 Repo Health); OpenSSF/LICENSE
> findings cross-referenced to same-wave 5.8/5.22 (not re-scored).

## (a) §3.4 Handoff packet

```yaml
agent: "5.21 CNCF Readiness"
wave: "C"
dimension_groups: ["D7"]   # feeds D15
overall_score: 5
overall_confidence: "High"
sub_scores:
  - dimension: "Structural governance artifacts (CoC, GOVERNANCE, CONTRIBUTING, SECURITY, ADRs/RFCs)"
    score: 8
    confidence: "High"
    justification: "CoC adopts CNCF CoC verbatim; GOVERNANCE.md is a full 12-section neutral-governance doc incl. solo-maintainer phase + multi-maintainer matrix + CNCF alignment checklist; SECURITY.md has private-advisory disclosure + SLAs + cosign-verify recipe; 36+ public ADRs + RFC process. Docked for dangling MAINTAINERS.md references."
    evidence:
      - "CODE_OF_CONDUCT.md:5 (adopts CNCF Code of Conduct verbatim)"
      - "GOVERNANCE.md:1-451 (12 sections; §10 add/remove maintainers; §12 CNCF alignment checklist)"
      - "SECURITY.md:18-26 (GitHub private Security Advisories + 48h/5d/severity SLAs)"
      - "SECURITY.md:65-71 (cosign verify recipe — keyless OIDC)"
      - "CONTRIBUTING.md (20.7 KB present); docs/adr/ (36 ADRs + INDEX.md)"
  - dimension: "License (CNCF hard requirement: top-level LICENSE)"
    score: 2
    confidence: "High"
    justification: "NO top-level LICENSE file exists — never committed to git history. Intent is well-documented (ADR-005 Accepted, SPDX headers repo-wide, README badge) but the canonical Apache-2.0 full text is absent. README links [LICENSE](LICENSE) are dangling. This is a CNCF Sandbox hard blocker and a doc-vs-reality drift (Part 1 §1.9 wrongly asserts LICENSE present)."
    evidence:
      - "cmd: `ls LICENSE` -> No existe el fichero (absent at top level)"
      - "cmd: `find . -maxdepth 2 -iname 'LICENSE*'` -> no output (absent anywhere)"
      - "cmd: `git ls-files | grep -i LICENSE` -> empty (not tracked)"
      - "cmd: `git log --all -- LICENSE` -> exit 0, no commits (NEVER existed in history)"
      - "cmd: `git check-ignore LICENSE` -> exit 1 (not gitignored — genuinely missing, not hidden)"
      - "README.md:13 + README.md:527 link [LICENSE](LICENSE) — dangling; docs/adr/ADR-005-apache-license.md:3 Status Accepted; SPDX-License-Identifier: Apache-2.0 in README.md:1, Makefile:1, CONTRIBUTING.md:1"
  - dimension: "Public roadmap + versioning"
    score: 8
    confidence: "High"
    justification: "ROADMAP.md narrative M1-M8 with version plan; M8 CNCF section lists the social prerequisites as unchecked; GitHub Project board referenced as authoritative execution roadmap; semver in GOVERNANCE §6."
    evidence:
      - "ROADMAP.md:251-261 (M8 CNCF Sandbox checklist: >=2 maintainers, ext audit, TOC filed — all unchecked)"
      - "ROADMAP.md:267-275 (version plan v0.1.0->v1.0.0); GOVERNANCE.md:244-258 (semver + milestone->version map)"
  - dimension: "Maintainer prerequisite (>=2 maintainers from 2+ orgs) + OWNERS/MAINTAINERS file"
    score: 1
    confidence: "High"
    justification: "Bus factor = 1: a single human identity (Oscar, 2 emails = 867 commits); no MAINTAINERS.md and no OWNERS file exist. MAINTAINERS.md is an OPEN backlog issue (#494, milestone:M8). CODEOWNERS exists but points at a @zynax-io/maintainers team of one. CNCF social gate unmet."
    evidence:
      - "cmd: `git shortlog -sne --all` -> Oscar Gómez (789) + Oscar Gómez Manresa (78); only bots otherwise (github-actions 45, renovate 7)"
      - "cmd: `ls MAINTAINERS.md OWNERS` -> both No existe el fichero (absent)"
      - "gh issue view 494 -> state OPEN, 'docs: create MAINTAINERS.md ... single-maintainer reality', labels milestone:M8 + status:backlog"
      - ".github/CODEOWNERS:5 (* -> @zynax-io/maintainers — team of one); GOVERNANCE.md:86 'Current maintainers: Listed in MAINTAINERS.md' (dangling)"
      - "consume 5.24: 'Bus factor = 1 ... MAINTAINERS.md still open (#494)' (docs/due-diligence/2026-06-20-dd-wave-a-findings.md:1853,1906)"
  - dimension: "Named adopters (>=1 external adopter)"
    score: 0
    confidence: "High"
    justification: "Zero named adopters. No ADOPTERS.md file. Strategy doc self-reports 0 stars/0 forks/0 external adopters as the baseline. CNCF Sandbox does not strictly require adopters but TOC strongly weights traction; this is a Wave-C cross-cut blocker."
    evidence:
      - "cmd: `find . -iname 'ADOPTERS*'` -> no output (no ADOPTERS file)"
      - "docs/product/strategy.md:290 '0 stars, 0 forks, 0 external adopters'; :338 '>=1 external adopter | ❌ (0 stars/forks baseline)'"
  - dimension: "Security disclosure + supply-chain controls (CNCF-credible)"
    score: 8
    confidence: "Medium"
    justification: "Private advisory disclosure with SLAs (SECURITY.md). Supply-chain trifecta cosign keyless + syft SPDX SBOM + SLSA L2 wired in release.yml (consume 5.2). Medium because GHCR signature EXISTENCE is config-VERIFIED / artifact-UNKNOWN (5.2/5.9 could not run cosign verify), and mTLS fails open — the kind of finding an M8 external audit flags."
    evidence:
      - "SECURITY.md:18-26 (private Security Advisories + SLAs); SECURITY.md:42-43 (SBOM + cosign SLSA L2 shipped #489)"
      - "consume 5.2: 'Supply-chain trifecta cosign+SBOM+SLSA — release.yml:201,527,510' (docs/due-diligence/2026-06-20-dd-wave-a-findings.md:44)"
      - "consume 5.2: C4 cosign-signed images PARTIAL — GHCR signature existence UNKNOWN (final.md:64,614); mTLS fails open + 2 prod overlays omit TLS (final.md:44,62)"
  - dimension: "Healthy dev cadence"
    score: 8
    confidence: "High"
    justification: "Consume 5.24: high, accelerating cadence (121->259->444 commits/month on a ~2-month, 824-commit repo), strictly linear signed history, 94% issue-closure. Cadence is strong; durability is the bus-factor risk, not velocity."
    evidence:
      - "consume 5.24: 'Cadence is high and accelerating (121->259->444 commits/month)' + '94% issue-closure' (docs/due-diligence/2026-06-20-dd-wave-a-findings.md:1933)"
      - "consume 5.24: 'strictly linear signed history, 0 merge commits' (final.md:50)"
  - dimension: "Trademark / neutrality / clean donatability"
    score: 4
    confidence: "Medium"
    justification: "No trademark policy file (acknowledged ❌, deferred to M8.A). GOVERNANCE.md asserts vendor-neutral 'no single company controls' but de-facto control is one individual; clean to donate IP-wise (Apache intent, no obvious company entanglement) BUT the missing LICENSE text and absent trademark policy block a clean donation today. Open-core monetization (Scenario A) is flagged in-doc as a neutrality tension."
    evidence:
      - "cmd: `find . -iname '*trademark*'` -> no output; docs/product/strategy.md:340 'Trademark policy | ❌'"
      - "GOVERNANCE.md:6 'Governance is neutral — no single company controls decisions' (aspirational vs bus-factor=1 reality)"
      - "docs/product/strategy.md:365-366 (open-core 'vendor-capture optics complicate a neutral CNCF donation')"
  - dimension: "Differentiation from existing CNCF projects (incl. Kagent)"
    score: 7
    confidence: "Medium"
    justification: "Clear, articulated differentiation vs Kagent (CNCF Sandbox 2026): engine-agnostic Workflow IR + multi-engine portability (Temporal+Argo) + Compose-and-K8s + adapter-first no-SDK vs Kagent's K8s-CRD pod-per-agent ADK lock-in. The 'why not them' answer exists and is coherent; weakened because Kagent already holds the category framing WITH CNCF backing, and the 'complementary' co-existence story is unproven."
    evidence:
      - "Part 1 §1.6 table (framework:197-205) Zynax IR/Temporal+Argo/Compose+K8s vs Kagent CRD/ADK-lock-in/K8s-only"
      - "docs/product/strategy.md:59-64,184,200 (Kagent = direct competitor, CNCF-backed, same framing)"
      - "docs/architecture/2026-05-28-competitive-positioning.md (named-competitor source per Part 1 §1.6)"

drift_test:
  - claim: "Apache-2.0 license present (LICENSE file) — Part 1 §1.9 / GOVERNANCE §12 '✅ done'"
    result: "CONTRADICTED"
    evidence:
      - "cmd: `git ls-files | grep -i LICENSE` -> empty; `git log --all -- LICENSE` -> no commits; `ls LICENSE` -> absent"
      - "Intent documented (ADR-005:3 Accepted; SPDX headers) but the canonical LICENSE full-text file does NOT exist; README.md:13,527 links dangle; GOVERNANCE.md:442 'Apache 2.0 license (done)' overstated"
  - claim: "OpenSSF Scorecard ✅ — README badge + strategy §8 table"
    result: "PARTIAL"
    evidence:
      - "README.md:16 (OpenSSF Scorecard badge present, links to securityscorecards.dev API)"
      - "cmd: `grep -rln scorecard .github/workflows/` -> exit 1 (NO scorecard.yml workflow committed)"
      - "Badge is a live API render, NOT a project-run scheduled scan; no committed Scorecard automation. Score itself not fetchable from sandbox (no network) — magnitude UNKNOWN. strategy.md:336 marks it ✅ but the control is a badge, not an in-repo workflow."
  - claim: "CNCF structural alignment strong; gating gap is social not technical (Part 1 §1.9)"
    result: "PARTIAL"
    evidence:
      - "Structural strength VERIFIED for CoC/GOVERNANCE/SECURITY/roadmap/ADRs/supply-chain (above)"
      - "BUT two STRUCTURAL gaps remain that §1.9 understated: missing LICENSE file (hard blocker) and missing MAINTAINERS/OWNERS file (#494 open) — so the gap is NOT purely social; one hard technical/legal artifact (LICENSE) is also missing"

red_flags:
  - severity: "Critical"
    finding: "No top-level LICENSE file anywhere in the repo or git history, despite Apache-2.0 intent (ADR-005, SPDX headers, README badge). A canonical license file is a CNCF Sandbox hard requirement and a basic OSS-hygiene/legal expectation; its absence makes the project non-donatable as-is and contradicts §1.9/GOVERNANCE §12 'done'."
    evidence: ["`git log --all -- LICENSE` -> no commits", "`ls LICENSE` -> absent", "README.md:13 dangling [LICENSE](LICENSE)", "GOVERNANCE.md:442 'Apache 2.0 license (done)'"]
  - severity: "High"
    finding: "Single-org maintainership (bus factor = 1); no MAINTAINERS.md / OWNERS file. CNCF '>=2 maintainers from 2+ orgs' unmet; GOVERNANCE.md references a MAINTAINERS.md that does not exist. MAINTAINERS.md is an open backlog issue (#494)."
    evidence: ["`git shortlog -sne --all` -> one human identity", "`ls MAINTAINERS.md OWNERS` -> absent", "gh issue 494 OPEN milestone:M8", "GOVERNANCE.md:86 dangling ref"]
  - severity: "High"
    finding: "Zero named external adopters and 0 stars/0 forks baseline; no ADOPTERS.md. TOC weighs traction heavily; redundancy risk vs Kagent (already CNCF Sandbox with the same framing) sharpens the 'why not them' question."
    evidence: ["docs/product/strategy.md:290,338", "`find -iname ADOPTERS*` -> none", "Part 1 §1.6 Kagent = Sandbox 2026 direct competitor"]
  - severity: "Medium"
    finding: "No committed OpenSSF Scorecard workflow and no trademark policy; the Scorecard '✅' rests on a live badge, not an in-repo scan. Trademark policy deferred to M8.A."
    evidence: ["`grep -rln scorecard .github/workflows/` -> exit 1", "README.md:16 badge only", "strategy.md:340 'Trademark policy ❌'"]

green_flags:
  - strength: "Unusually honest self-assessment: strategy §8 already maps the Sandbox criteria with the social gaps marked ❌ and explicitly advises 'do not file until prerequisites are real' — exactly the integrity a TOC rewards."
    evidence: ["docs/product/strategy.md:330-346 (criteria table + honest verdict)"]
  - strength: "Strong, near-complete structural governance: CNCF CoC verbatim, a full neutral GOVERNANCE.md with a solo->multi maintainer model + CNCF alignment checklist, private-advisory SECURITY.md with SLAs, RFC process, 36+ public ADRs."
    evidence: ["CODE_OF_CONDUCT.md:5", "GOVERNANCE.md:1-451", "SECURITY.md:18-26", "docs/adr/ (36 ADRs)"]
  - strength: "CNCF-credible supply-chain posture (cosign keyless + syft SPDX SBOM + SLSA L2) and high, accelerating dev cadence with signed linear history (consume 5.2, 5.24)."
    evidence: ["docs/due-diligence/2026-06-20-dd-wave-a-findings.md:44 (release.yml:201,527,510)", "final.md:1933 (cadence + linear history)"]
  - strength: "Coherent differentiation vs Kagent: engine-agnostic IR + multi-engine portability + Compose-and-K8s + adapter-first no-SDK — a defensible 'why not them' answer exists."
    evidence: ["Part 1 §1.6 framework:197-205", "docs/product/strategy.md:184,200"]

open_questions:
  - "Was the LICENSE file ever intended to be committed (was it lost in a scaffolding gap) or is the project relying solely on SPDX headers? A TOC/legal review would reject SPDX-only."
  - "What is the actual OpenSSF Scorecard numeric score? (No network from sandbox; badge value not fetchable.)"
  - "Is there any company entity behind zynax-io that would complicate a neutral donation (open-core Scenario A optics)?"

unknowns:
  - "OpenSSF Scorecard numeric value — badge is a live API render; no network access and no committed scorecard.yml to infer from. UNKNOWN."
  - "GHCR cosign signature / SLSA attestation EXISTENCE — deferred to 5.2/5.9 (cosign absent, no registry access); config-VERIFIED only."
  - "Whether a TOC sponsor has been informally approached — strategy.md:341 says 'not yet recruited'; not independently verifiable."

cross_references:
  - to_agent: "5.2 Security"
    note: "Supply-chain (cosign+SBOM+SLSA) is CNCF-credible; mTLS fail-open + 2 prod overlays omitting TLS are exactly the class of finding an M8 external security audit (a Sandbox->Incubation gate) would flag."
    evidence: ["docs/due-diligence/2026-06-20-dd-wave-a-findings.md:44,62,64"]
  - to_agent: "5.24 Repository Health"
    note: "Bus factor = 1 and MAINTAINERS.md open (#494) is the shared root of my maintainer-prerequisite and cadence-durability scores."
    evidence: ["docs/due-diligence/2026-06-20-dd-wave-a-findings.md:1853,1906,1933"]
  - to_agent: "5.8 / 5.22 (Open Source / OpenSSF & LICENSE — same wave)"
    note: "The missing LICENSE file and badge-only OpenSSF Scorecard are owned cross-references; I record them as a Critical/Medium CNCF blocker but defer the primary OSS-hygiene/Scorecard scoring to 5.8/5.22."
    evidence: ["`git log --all -- LICENSE` -> none", "README.md:16", "`grep -rln scorecard .github/workflows/` -> exit 1"]
  - to_agent: "5.13 Governance"
    note: "GOVERNANCE.md is structurally strong but contains dangling MAINTAINERS.md references and a 'license done' overstatement; reconcile with the missing-file reality."
    evidence: ["GOVERNANCE.md:86,442"]

recommendations:   # critical-path ordered
  - priority: "P0"
    action: "Commit the canonical Apache-2.0 LICENSE full text at repo root and fix the dangling README/GOVERNANCE references."
    rationale: "Hard CNCF Sandbox requirement and basic legal hygiene; trivial to fix; its absence alone would bounce a TOC submission and currently makes the project non-donatable. Also closes the §1.9/GOVERNANCE §12 'done' contradiction."
  - priority: "P0"
    action: "Recruit and document a 2nd maintainer from a 2nd org; create MAINTAINERS.md (close #494)."
    rationale: "The binding social gate (>=2 maintainers from 2+ orgs) and the single largest sustainability risk (bus factor = 1); resolves the dangling GOVERNANCE.md maintainer references."
  - priority: "P1"
    action: "Land >=1 named external adopter (ADOPTERS.md) and establish a public community cadence before filing."
    rationale: "TOC weights traction; 0 stars/forks vs a CNCF-backed Kagent makes 'why not them' hard to answer on adoption."
  - priority: "P1"
    action: "Add a committed OpenSSF Scorecard workflow (scorecard.yml) and a TRADEMARK/branding policy."
    rationale: "Converts the Scorecard badge into a project-run control and closes the trademark ❌ (M8.A) — both expected at submission."
  - priority: "P2"
    action: "Do NOT file the Sandbox application until P0+P1 are real; recruit a TOC sponsor in parallel."
    rationale: "The project's own strategy doc (strategy.md:345) is correct — filing early and being rejected is worse than filing late; honor that."
```

## (b) §6.2 Prose

## CNCF Readiness — Score: 5 (High)
**Mission recap:** Map Zynax against CNCF Sandbox criteria at HEAD and judge readiness honestly.

**Verdict:** Zynax is **structurally close but not Sandbox-ready (do NOT file now)** — and the
single blocker is not the social gap everyone expected but a **missing top-level LICENSE file**:
there is no Apache-2.0 license text anywhere in the repo or git history despite ADR-005, SPDX
headers, and a README license badge that links to a dangling `LICENSE` path. Layered on top are
the well-known social gates: bus factor = 1, no MAINTAINERS/OWNERS file (#494 open), and zero
named adopters against a CNCF-backed Kagent occupying the same framing. The governance substrate
(CNCF CoC verbatim, a complete neutral GOVERNANCE.md, private-advisory SECURITY.md, public
roadmap, 36+ ADRs, cosign/SBOM/SLSA supply chain) is genuinely strong and the project's own
strategy §8 is admirably honest about what is unmet — which is the best signal a TOC could ask
for. Score 5: structurally close, socially far, with one hard legal artifact also missing.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Structural governance artifacts | 8 | High | GOVERNANCE.md:1-451; CODE_OF_CONDUCT.md:5; SECURITY.md:18-26 |
| License (LICENSE file) | 2 | High | `git log --all -- LICENSE` -> none; README.md:13 dangling |
| Public roadmap + versioning | 8 | High | ROADMAP.md:251-275 |
| Maintainer prereq + MAINTAINERS/OWNERS | 1 | High | `git shortlog` one human; `ls MAINTAINERS.md OWNERS` absent; #494 OPEN |
| Named adopters | 0 | High | strategy.md:290,338; no ADOPTERS file |
| Security disclosure + supply chain | 8 | Medium | SECURITY.md:18-26; 5.2 release.yml:201,527,510 |
| Healthy dev cadence | 8 | High | 5.24 final.md:1933 |
| Trademark / neutrality | 4 | Medium | no trademark file; strategy.md:340 |
| Differentiation vs Kagent | 7 | Medium | Part 1 §1.6; strategy.md:184,200 |

**Drift test:**
- "Apache-2.0 LICENSE present (§1.9 / GOVERNANCE §12 done)" → **CONTRADICTED** — file never existed in git (`git log --all -- LICENSE` empty); intent documented but canonical text absent; README links dangle.
- "OpenSSF Scorecard ✅" → **PARTIAL** — badge present (README.md:16) but no committed scorecard.yml workflow (`grep -rln scorecard .github/workflows/` exit 1); score value UNKNOWN (no network).
- "Structural alignment strong; gap is purely social (§1.9)" → **PARTIAL** — true for CoC/governance/supply-chain, but two structural artifacts (LICENSE, MAINTAINERS/OWNERS) are also missing, so the gap is not purely social.

**Red flags (severity-ordered):**
1. **Critical —** No LICENSE file anywhere (`git log --all -- LICENSE` empty; README.md:13 dangling; GOVERNANCE.md:442 "done"). CNCF hard blocker; non-donatable as-is.
2. **High —** Bus factor = 1, no MAINTAINERS/OWNERS (`git shortlog` one human; #494 OPEN). Unmet ">=2 maintainers from 2+ orgs".
3. **High —** Zero named adopters / 0 stars-forks (strategy.md:290,338) vs CNCF-backed Kagent — sharpens "why not them".
4. **Medium —** No committed Scorecard workflow; no trademark policy (`grep` exit 1; strategy.md:340).

**Green flags:**
- Unusually honest self-assessment with an explicit "don't file yet" verdict (strategy.md:330-346).
- Near-complete structural governance (GOVERNANCE.md:1-451; CODE_OF_CONDUCT.md:5; SECURITY.md:18-26; 36+ ADRs).
- CNCF-credible supply chain + high signed-linear cadence (5.2, 5.24).
- Coherent Kagent differentiation — defensible "why not them" answer (Part 1 §1.6).

**Open questions / unknowns:** Was LICENSE lost in scaffolding or never committed? Actual OpenSSF
Scorecard numeric value (no network)? GHCR signature existence (defer 5.2/5.9)? Any company
entity behind zynax-io complicating a neutral donation?

**Recommendations (critical-path):**
- **P0** — Commit Apache-2.0 LICENSE at root; fix dangling refs. (Hard gate, trivial fix.)
- **P0** — Recruit + document a 2nd-org maintainer; create MAINTAINERS.md (close #494).
- **P1** — Land >=1 named adopter (ADOPTERS.md) + public cadence before filing.
- **P1** — Add scorecard.yml + TRADEMARK policy.
- **P2** — Do not file until P0+P1 real; recruit TOC sponsor in parallel (honors strategy.md:345).

**Cross-references:** 5.2 (supply chain credible; mTLS fail-open = M8 audit finding); 5.24 (bus
factor = 1 / #494); 5.8 & 5.22 (LICENSE + OpenSSF — owned there, flagged here as CNCF blocker);
5.13 (GOVERNANCE.md dangling refs + "license done" overstatement); D15 (this D7 feeds the
governance/community roll-up).
<!-- SPDX-License-Identifier: Apache-2.0 -->
# Agent 5.25 — Future Roadmap · Wave C (product/market/governance)

> Issue #1404 · HEAD `main @ e3135a6` · READ-ONLY audit.
> Evidence rule §0.4: every claim carries `path:line` / command-output, or is `UNKNOWN`.
> Roadmap/marketing = `CLAIMED`; code/CI/GitHub-state-verified = `VERIFIED`.
> Consumes Wave A (`docs/due-diligence/2026-06-20-dd-wave-a-findings.md`): Repo Health 5.24 (velocity/cadence) and
> Architecture 5.1 (data-flow / portability keystone). Same-wave 5.13/5.3 are cross-references
> (not blocking; not re-scored per §3.1).

---

## (a) §3.4 Handoff packet

```yaml
agent: "5.25 Future Roadmap"
wave: "C"
dimension_groups: ["D12"]   # feeds D15 (Wave D investment synthesis)
overall_score: 7
overall_confidence: "High"

sub_scores:
  - dimension: "Keystone delivery — data-flow bindings (ADR-029) done?"
    score: 9
    confidence: "High"
    justification: "Keystone is fully delivered at HEAD: ADR-029 Accepted, proto binding fields present, interpreter threads a run-scoped data context, real example workflows use the bindings, and EPIC W + all 5 stories are CLOSED on GitHub."
    evidence:
      - "docs/adr/ADR-029-workflow-data-flow.md:6 — '**Status** | Accepted' (Date 2026-06-16)"
      - "protos/zynax/v1/workflow_compiler.proto:160,168 — output_bindings=5 / input_bindings=6 (additive IR fields)"
      - "services/engine-adapter/internal/domain/interpreter.go:66,131,152 — NewScopedWorkflowDataContext + data.ResolveInputs + data.WriteOutputs (run-scoped threading, ADR-029)"
      - "services/engine-adapter/internal/domain/datacontext.go (WorkflowDataContext type exists, with _test.go)"
      - "cmd: gh issue view 1167 → state CLOSED (epic M7.W); gh issue view 1176/1177/1178/1179 → all CLOSED (W.2 proto, W.3 compiler, W.4 data context, W.5 e2e)"
      - "spec/workflows/examples/{research-task,code-review,ci-pipeline,feature-implementation}.yaml — real manifests using input_bindings/output_bindings (grep -l)"
  - dimension: "M7 EPIC completion vs plan"
    score: 8
    confidence: "High"
    justification: "Live GitHub milestone is ~95% closed (114 closed / 6 open); of the 6 open, 5 are due-diligence meta-issues and only ONE (#1370 M7.K) is a real feature EPIC. Strong, near-complete delivery against an ambitious 12-EPIC plan."
    evidence:
      - "cmd: gh issue list --milestone 'Usable Workflows + Observability (M7)' --state closed → 114; --state open → 6"
      - "cmd: gh issue list ... --state open → #1370 (M7.K) is the only feature EPIC; #1399,#1403,#1404,#1405,#1406 are DD-framework meta-issues"
      - "state/current-milestone.md:63 — 'GitHub milestone #7 — ~115 closed / ~10 open (~92% done)' (CLAIMED, corroborated by live count above)"
      - "state/milestone.yaml:27 — open_epics list is stale (467,468,469,1167,1170,1172… all in fact CLOSED) — written only by /milestone-close"
  - dimension: "Sequencing realism (dependency ordering, observability/usability on track)"
    score: 8
    confidence: "High"
    justification: "Critical path Q→W→C→X→T→D is correctly ordered (the keystone gates the dependent EPICs); a documented dependency graph + 5-wave parallel plan exist; observability core (libs/zynaxobs OTEL) and Uptrace compose/Helm landed."
    evidence:
      - "docs/milestones/M7-planning.md:459 — 'Critical path: Q → W → C → X → T → D' with mermaid graph 426-457"
      - "docs/milestones/M7-planning.md:472-478 — 5-wave parallel plan, ≤3 concurrent, explicit gate-to-advance per wave"
      - "state/current-milestone.md:85-87 — O.4/O.7/O.8 (Uptrace exemplars + compose + Helm) shipped; Wave A 5.24 confirms libs/zynaxobs OTEL core wired in all 7 main.go"
      - "docs/milestones/M7-planning.md:485-497 — risk register: scope-creep risk #8 rated High/High with the program-split mitigation"
  - dimension: "Velocity vs ambition (does cadence support the M8 timeline?)"
    score: 6
    confidence: "Medium"
    justification: "Cadence is exceptional and accelerating (121→259→444 commits/mo; 94% issue-closure), which easily clears the *technical* M7→M-dx work; BUT M8/v1.0 gates are social (≥2 cross-org maintainers, external audit, TOC sponsor) which velocity cannot buy — and bus factor=1 is the binding constraint."
    evidence:
      - "5.24 (Wave A) docs/due-diligence/2026-06-20-dd-wave-a-findings.md:1758 — 'git log per-month: 2026-04=121, 2026-05=259, 2026-06=444 → accelerating'"
      - "5.24 (Wave A) final.md:1933 — '94% issue-closure ratio … zero merge commits, squash-only'"
      - "5.24 (Wave A) final.md:1766 — 'git shortlog -sne → 772 human commits one identity; bus factor=1' (score 3/10)"
      - "ROADMAP.md:257-261 — M8 gates: '≥2 maintainers from different organisations', 'External security audit', 'CNCF TOC application filed' (CLAIMED; social, not code)"
  - dimension: "Roadmap honesty (non-goals explicit, scope creep controlled)"
    score: 7
    confidence: "High"
    justification: "Non-goals are explicit and ADR-backed (LLM framework / DAG / required-SDK ruled out); risk register flags scope creep openly. BUT the v1.0 path quietly grew from 1 hop to 3-4 (M-UX + M-dx inserted) and the docs are internally inconsistent on it — mild, undisclosed scope expansion toward v1.0."
    evidence:
      - "ROADMAP.md:296-300 — 'What Is NOT on the Roadmap' table: LLM framework (ADR-011), DAG workflows (ADR-014), SDK-required (ADR-013)"
      - "docs/milestones/M7-planning.md:496 — risk #8 'Scope creep — brief is ~3 milestones … Likelihood High / Impact High' with program-split mitigation (honest disclosure)"
      - "ROADMAP.md:56-59 — path to v1.0 now M7(v0.6)→M-UX(v0.7)→M-dx(v0.8)→M8(v1.0): FOUR milestones"
      - "docs/milestones/M7-planning.md:53,55 — same plan still describes a THREE-milestone program 'M7 → M-dx → M8' with M-dx at v0.7.0 (conflicts ROADMAP M-dx=v0.8.0)"
      - "ROADMAP.md:265-275 — Version Plan table OMITS M-UX and M-dx, jumping v0.6.0 → v1.0.0 (internally inconsistent vs lines 56-59)"
  - dimension: "Proposed-ADR backlog (forward-commitment freshness)"
    score: 7
    confidence: "High"
    justification: "Keystone-class ADRs (029/030/032/035/036) are Accepted; only 031/033/034 remain Proposed though their issues are closed — a known, tracked promote-on-alignment debt, not a blocking slip."
    evidence:
      - "cmd: grep -niE 'Status' docs/adr/ADR-031/033/034 → all '**Status:** Proposed'; ADR-029 → Accepted (head -8)"
      - "docs/milestones/M7-planning.md:568-569 — 'ADR-029/030/032 are Accepted; 031/033/034 remain Proposed and should be promoted as canvases C/X/Q reach Aligned' (self-disclosed)"
      - "state/current-milestone.md:269-271 — same state-hygiene flag for ADRs 031/033/034"
  - dimension: "Critical path to v1.0 / CNCF (true blockers: technical vs social)"
    score: 6
    confidence: "High"
    justification: "Technical path to v1.0 is plausible given velocity + a delivered keystone. The TRUE blockers are social/community: bus factor=1, zero named external adopters, no recruited cross-org maintainer or TOC sponsor — none addressable by code cadence."
    evidence:
      - "ROADMAP.md:257-261 — M8 checklist is community/governance/audit, not features"
      - "Part 1 §1.9 (framework:251) — 'The gating gap is social, not technical: single-maintainer bus factor, zero named external adopters, no recruited TOC sponsor'"
      - "5.24 (Wave A) final.md:1933 — bus factor=1; MAINTAINERS.md still open (#494, per Wave A scorecard:72)"

drift_test:
  - claim: "DRIFT TEST (prior-milestone optimism calibration): a prior milestone was labeled Complete but had silently deferred scope (Part 1 §1.10 C1/C7)."
    result: "VERIFIED (historical optimism real; SINCE-CORRECTED by Truth Pass)"
    evidence:
      - "Part 1 §1.10 framework:263 — C1 'M3 & M4 Complete' (early README) but Partial; relabelled Partial — confirmed at HEAD by state/current-milestone.md:17-18,23 (M3/M4 ⚠ Partial)"
      - "docs/milestones/M5-plan.md:9 — 'GitHub milestone closed; 5 deferred issues moved to M6' (plan-vs-actual slip: #235 #239 #376 #466 #656 per current-milestone.md:26)"
      - "docs/milestones/M5-plan.md:718 — 'H1: Stateless workflow-compiler … M6 (deferred — not completed in M5)' (keystone-adjacent slip)"
      - "CALIBRATION: the same optimism class is NOT present at M7 HEAD — the keystone (ADR-029/EPIC W) is actually DONE (CLOSED + code-verified), and M7 status (95% closed) is corroborated by live GitHub, not just narrative. The project has moved from claim-ahead-of-delivery (M3/M4) to delivery-ahead-of-doc-update (M7 canvas/plan lag CLOSED issues — see red flag below)."
  - claim: "M7 keystone (data-flow bindings, ADR-029) is done."
    result: "VERIFIED"
    evidence:
      - "ADR-029 Accepted (ADR-029:6); proto fields (workflow_compiler.proto:160,168); interpreter threading (interpreter.go:66,131,152); EPIC #1167 + W.2-W.5 #1176-1179 all CLOSED (gh)"
  - claim: "M7 is ~92% complete (state doc)."
    result: "VERIFIED"
    evidence:
      - "live gh count 114 closed / 6 open = ~95%; 5 of 6 open are DD meta-issues, only #1370 is a feature EPIC"
  - claim: "The path to v1.0 is the stated single hop M7 → M-dx → M8."
    result: "CONTRADICTED (internally inconsistent; path grew)"
    evidence:
      - "ROADMAP.md:56-59 inserts M-UX (v0.7) + M-dx (v0.8) → 4 milestones to v1.0; M7-planning.md:53 still says 3 (M-dx=v0.7); Version Plan table ROADMAP.md:265-275 omits both"

red_flags:
  - severity: "High"
    finding: "The binding constraint on v1.0/CNCF is SOCIAL, not technical, and is not solvable by the project's (excellent) velocity. M8 requires ≥2 cross-org maintainers, an external security audit, and a filed TOC application; the repo is bus factor=1 with zero named external adopters and an open MAINTAINERS.md. The technical roadmap can land on time and v1.0/CNCF still stalls."
    evidence:
      - "ROADMAP.md:257-261 (M8 social/governance checklist)"
      - "5.24 (Wave A) docs/due-diligence/2026-06-20-dd-wave-a-findings.md:1766,1933 (bus factor=1; MAINTAINERS.md open #494)"
      - "Part 1 §1.9 framework:251-252 (gating gap is social)"
  - severity: "Medium"
    finding: "Status-surface drift in the OPPOSITE direction of the old M3/M4 problem: the keystone canvas and the M7 planning doc UNDER-state reality. Canvas W is still 'Status: Aligned' with W.2-W.5 acceptance boxes UNCHECKED, and M7-planning §4a lists W.2-W.5 as ⬜ open — yet all four are CLOSED and code-verified at HEAD. state/milestone.yaml open_epics still lists already-closed EPICs. Honest direction (delivery ahead of docs) but it is the same reconciliation-debt class the Truth Pass exists to catch."
    evidence:
      - "docs/spdd/1167-workflow-data-flow/canvas.md:9,92-122 (Status Aligned; unchecked W.2-W.5 boxes)"
      - "docs/milestones/M7-planning.md:173-176 (W.2-W.5 marked ⬜ open) vs gh issue view 1176-1179 → all CLOSED"
      - "state/milestone.yaml:27 (open_epics lists closed 467,468,469,1167,1170,1172)"
  - severity: "Medium"
    finding: "Roadmap version/sequence inconsistency toward v1.0: ROADMAP shows 4 milestones (M7→M-UX→M-dx→M8) but M7-planning still describes a 3-milestone program with a different M-dx version, and the Version Plan table omits M-UX/M-dx entirely. The number of hops to v1.0 is not stated consistently — a mild scope-expansion-without-clean-disclosure signal."
    evidence:
      - "ROADMAP.md:56-59 vs ROADMAP.md:265-275 vs docs/milestones/M7-planning.md:53,55"
  - severity: "Low"
    finding: "Three forward-commitment ADRs (031 context, 033 expert substrate, 034 manifest-id) remain Proposed though their issues are closed; promotion is tracked but pending. Architecture (5.1) separately flags ADR-034 'Proposed' as contradicting shipped idempotent apply."
    evidence:
      - "grep ADR-031/033/034 → Proposed; docs/milestones/M7-planning.md:568-569; cross-ref 5.1 final.md:251-254"

green_flags:
  - strength: "The keystone shipped, and is verifiable by execution + closed issues, not narrative — exactly the delivery posture the diligence rewards. ADR-029 Accepted, proto fields additive, interpreter threads the data context, real example workflows consume bindings."
    evidence: ["docs/adr/ADR-029-workflow-data-flow.md:6", "services/engine-adapter/internal/domain/interpreter.go:66,131,152", "gh issue view 1167/1176-1179 → CLOSED"]
  - strength: "Near-complete M7 (~95% closed) delivered at exceptional, accelerating cadence — the technical execution capability is among the strongest signals in the audit; ambition is matched by demonstrated throughput on the in-scope (technical) work."
    evidence: ["gh: 114 closed / 6 open", "5.24 final.md:1758 (121→259→444 commits/mo)"]
  - strength: "Roadmap is calibrated by its own history and disciplined about scope: explicit ADR-backed non-goals, a scope-creep risk rated High/High with a concrete program-split mitigation, and an honest 'M7 targets local/dev; prod scale → M8' deferral on observability."
    evidence: ["ROADMAP.md:296-300", "docs/milestones/M7-planning.md:496", "docs/milestones/M7-planning.md:491,581"]
  - strength: "Correctly sequenced critical path with a real dependency graph and gated parallel waves; the keystone is sequenced first because everything depends on it — and that ordering held in actual delivery."
    evidence: ["docs/milestones/M7-planning.md:426-459,472-478"]

open_questions:
  - "Will any cross-org maintainer or TOC sponsor be recruited on a timeline that supports M8/v1.0? No evidence of an active recruitment thread in-repo (MAINTAINERS.md #494 open)."
  - "Is the v1.0 path 3 or 4 milestones (M-UX inclusion)? ROADMAP and M7-planning disagree; the true remaining hop-count to v1.0 is ambiguous."
  - "Does M7 formally CLOSE at v0.6.0 once #1370 lands, or does the residual #1359 (zero-Temporal engine) → M-dx deferral leave M7 'closing' indefinitely?"

unknowns:
  - "Actual calendar dates for M-UX / M-dx / M8 — no committed target dates found; all milestone timing is sequence-only (CLAIMED), so 'will v1.0 happen on the stated timeline' is UNKNOWN because no dated timeline exists to test against."
  - "Whether an external security audit has been initiated/scoped (M8 gate) — not determinable from repo."

cross_references:
  - to_agent: "5.1 Architecture"
    note: "Portability moat is real at the IR/contract boundary but Argo only stub-executes the IR — this caps the 'category' claim that the roadmap leans on for CNCF differentiation. Roadmap defers Argo-parity implicitly; flag as a forward risk."
    evidence: ["docs/due-diligence/2026-06-20-dd-wave-a-findings.md:240-244 (argo stub)", "docs/milestones/M7-planning.md (no Argo-parity EPIC in M7)"]
  - to_agent: "5.24 Repository Health"
    note: "Velocity (121→259→444 commits/mo) is the engine behind the credible technical roadmap; bus factor=1 is the engine-killer for the social M8 gate. Both consumed directly."
    evidence: ["docs/due-diligence/2026-06-20-dd-wave-a-findings.md:1758,1766"]
  - to_agent: "5.13 Governance (same-wave)"
    note: "M8 gates (≥2 cross-org maintainers, TOC application, CoC/governance maturity) sit in the Governance zone; do not re-score — Governance owns the structural-readiness judgement."
    evidence: ["ROADMAP.md:257-261"]
  - to_agent: "5.3 Product (same-wave)"
    note: "Roadmap's 'usable workflows' bet (data-flow keystone delivered) is the product-readiness pivot; Product owns adoption-funnel scoring."
    evidence: ["docs/milestones/M7-planning.md:27-39"]

recommendations:
  - priority: "P0"
    action: "Treat maintainer/community recruitment (cross-org co-maintainer, external-adopter capture, TOC sponsor) as the literal critical path to v1.0 — schedule it NOW as an M-UX/M-dx-parallel track, not an M8 line-item. Technical velocity will not move this gate."
    rationale: "Every M8 checkbox is social; bus factor=1 + zero named adopters is the single most severe forward risk and the slowest-moving variable in the plan."
  - priority: "P1"
    action: "Run a status-surface reconcile (the project's own /reconcile): tick canvas-W acceptance boxes, flip M7-planning §4a W.2-W.5 to ✅, prune state/milestone.yaml open_epics of closed EPICs, and resolve the M7→M-UX→M-dx→M8 version/hop inconsistency across ROADMAP + M7-planning + Version-Plan table."
    rationale: "Docs now LAG reality (under-claiming) — opposite polarity to the old M3/M4 over-claim, but the same reconciliation debt the Truth Pass exists to prevent; left unchecked it erodes the very delivery-vs-narrative trust the project rebuilt."
  - priority: "P2"
    action: "Promote ADRs 031/033/034 from Proposed to Accepted now their issues are closed (and reconcile ADR-034 to the shipped id scheme per 5.1)."
    rationale: "Closes the forward-commitment-freshness gap and removes an ADR-vs-code contradiction surfaced by Architecture."
  - priority: "P2"
    action: "Publish dated (even rough) targets for M-UX/M-dx/M8 so the v1.0/CNCF timeline becomes testable; absence of any committed date makes the timeline neither verifiable nor falsifiable."
    rationale: "Converts an UNKNOWN-by-construction timeline into a calibratable forward commitment."
```

---

## (b) §6.2 Prose section

## 5.25 Future Roadmap — Score: 7 (High)

**Mission recap:** Judge the realism, sequencing, and risk of the M7/M8 roadmap and the credibility of the path to v1.0 and CNCF, calibrated against the project's documented history of optimistic labeling.

**Verdict:** The forward roadmap is credible, well-sequenced, and — critically — calibrated by delivery, not narrative. The M7 keystone (workflow data-flow, ADR-029) is genuinely done: the ADR is Accepted, the additive proto binding fields exist, the engine-adapter interpreter threads a run-scoped data context (`ResolveInputs`/`WriteOutputs`), real example workflows consume the bindings, and EPIC W plus all five stories are CLOSED on GitHub. The milestone is ~95% complete on live state (114 closed / 6 open, and 5 of those 6 are due-diligence meta-issues), delivered at an exceptional, accelerating cadence. The reason this is a strong-7 and not a 9 is entirely the destination, not the engine: the true blocker to v1.0/CNCF is social (≥2 cross-org maintainers, external audit, TOC sponsor) against a bus-factor-of-1 repository — a gate no amount of commit velocity can clear.

**Sub-dimension scores:**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Keystone delivery — data-flow (ADR-029) | 9 | High | `ADR-029:6` Accepted; `workflow_compiler.proto:160,168`; `interpreter.go:66,131,152`; gh #1167/#1176-1179 CLOSED |
| M7 EPIC completion vs plan | 8 | High | gh 114 closed / 6 open (~95%); only #1370 is an open feature EPIC |
| Sequencing realism | 8 | High | `M7-planning.md:459` critical path Q→W→C→X→T→D; waves `472-478` |
| Velocity vs ambition | 6 | Medium | 5.24 `final.md:1758` 121→259→444 commits/mo vs social M8 gates |
| Roadmap honesty (non-goals, scope creep) | 7 | High | `ROADMAP.md:296-300`; `M7-planning.md:496`; version inconsistency `ROADMAP.md:56-59` vs `265-275` |
| Proposed-ADR backlog freshness | 7 | High | ADR-031/033/034 Proposed; `M7-planning.md:568-569` tracks promotion |
| Critical path to v1.0 / CNCF | 6 | High | `ROADMAP.md:257-261`; Part 1 §1.9 framework:251 (social gate) |

**Drift test:** *"A prior milestone was labeled Complete but silently deferred scope."* → **VERIFIED historically, SINCE-CORRECTED.** M3/M4 were labeled Complete then relabelled Partial (Part 1 §1.10 C1; `current-milestone.md:17-18,23`); M5 closed with 5 issues deferred to M6 (`M5-plan.md:9`) and the stateless-compiler keystone "deferred — not completed in M5" (`M5-plan.md:718`). The calibration result is the headline finding: **the optimism pattern has inverted at M7.** The keystone is actually done (code + closed issues), and M7's status is corroborated by live GitHub rather than asserted. The project moved from *claim-ahead-of-delivery* (M3/M4) to *delivery-ahead-of-doc-update* (the canvas and plan still list closed work as open). Subsidiary: *"v1.0 path is the stated M7→M-dx→M8"* → **CONTRADICTED** — ROADMAP now inserts M-UX + M-dx (4 hops) while the plan still says 3.

**Red flags (severity-ordered):**
1. **High** — v1.0/CNCF is gated by social factors (cross-org maintainers, external audit, TOC application) that velocity cannot buy, against bus factor = 1 and zero named adopters (`ROADMAP.md:257-261`; 5.24 `final.md:1766,1933`; framework §1.9:251).
2. **Medium** — Status-surface drift, opposite polarity to the old problem: canvas-W still "Aligned" with unchecked boxes and `M7-planning §4a` lists W.2-W.5 ⬜ open while all are CLOSED; `milestone.yaml` open_epics lists closed EPICs (`canvas.md:9,92-122`; `M7-planning.md:173-176`; `milestone.yaml:27`).
3. **Medium** — Roadmap version/hop inconsistency toward v1.0 (`ROADMAP.md:56-59` vs `265-275` vs `M7-planning.md:53`).
4. **Low** — ADRs 031/033/034 still Proposed though issues closed (`M7-planning.md:568-569`; cross-ref 5.1 on ADR-034).

**Green flags:**
- Keystone shipped and execution-verifiable, not narrative (`ADR-029:6`; `interpreter.go:66,131,152`; closed #1167/#1176-1179).
- ~95% M7 completion at accelerating cadence — demonstrated throughput matches stated ambition on in-scope technical work (gh counts; 5.24 `final.md:1758`).
- Scope discipline: ADR-backed explicit non-goals, scope-creep risk rated High/High with a program-split mitigation, honest local-vs-prod observability deferral (`ROADMAP.md:296-300`; `M7-planning.md:496,491`).
- Correctly sequenced, dependency-aware critical path that held in actual delivery (`M7-planning.md:426-459`).

**Open questions / unknowns:** No committed calendar dates for M-UX/M-dx/M8 — the v1.0 timeline is sequence-only, so "will it ship on the stated timeline" is UNKNOWN by construction. Whether a cross-org maintainer or external security audit is in motion is not determinable from the repo. The true hop-count to v1.0 (3 vs 4 milestones) is ambiguous in the docs.

**Recommendations:** P0 — schedule maintainer/adopter/TOC recruitment as the literal critical path to v1.0 now, parallel to M-UX/M-dx, because it is the slowest and only non-technical variable. P1 — run the project's own `/reconcile` to close the status-surface lag (canvas boxes, §4a, milestone.yaml, version inconsistency). P2 — promote ADRs 031/033/034 and reconcile ADR-034 to shipped code; publish dated milestone targets so the v1.0/CNCF timeline becomes testable.

**Cross-references:** 5.1 Architecture (Argo only stub-executes the IR — caps the category claim the roadmap leans on for CNCF differentiation); 5.24 Repo Health (velocity is the credible-roadmap engine, bus-factor=1 is the M8 engine-killer); 5.13 Governance (owns the M8 structural-readiness judgement — not re-scored here); 5.3 Product (owns adoption-funnel scoring of the "usable workflows" bet).
