<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax — Investment-Grade Technical & Strategic Due-Diligence Report

| Field | Value |
|-------|-------|
| **Title** | Zynax — Investment-Grade Technical & Strategic Due-Diligence Report |
| **Date** | 2026-06-21 |
| **Prepared by** | Lead Diligence Partner — orchestration of 26 specialised agents (Waves A–D) |
| **Repository HEAD audited** | `main` @ `e3135a6` |
| **Methodology** | 2026-06-18 Zynax due-diligence framework (Parts 1–10) |
| **Classification** | **CONFIDENTIAL** |
| **Overall verdict** | Conditional / Watch · Overall score **5.4 / 10** (raw weighted mean 6.4) · Confidence **Medium** · Risk profile **High** (4 Critical · 9 High · 11 Medium · 3 Low) |

---

## Confidentiality notice

This report is **CONFIDENTIAL** and is prepared solely for the named recipient in connection with the evaluation of a potential transaction involving Zynax. It contains the diligence team's analysis, opinions, and synthesis of the subject's public source repository as of the audited commit, and must not be reproduced, distributed, quoted, or disclosed — in whole or in part — to any third party without the prior written consent of the Lead Diligence Partner. It is an opinion of relative merit, not a guarantee, audit certificate, fairness opinion, or legal/financial advice; the recipient must form its own independent judgment and complete the confirmatory-diligence steps enumerated herein before relying on any conclusion.

---

## Methodology & evidence standard

This report was produced by a **multi-agent diligence orchestration**: 26 specialised analyst agents executed across four non-overlapping waves and were synthesised by a master orchestrator into the verdict, the confidence-weighted scorecard, the contradiction register, and the risk profile that this document presents. Every conclusion is traceable to a specific agent packet and a specific piece of evidence; the four wave findings documents (Appendix A) carry the full per-agent packets.

**Scoring scale (0–10, binding on all agents).** Each dimension is scored against a single rubric — 9–10 Exceptional (a competitive asset), 7–8 Strong (above market norm, production-credible), 5–6 Adequate (meets baseline, notable gaps), 3–4 Weak (material gaps that block adoption or scale), 1–2 Poor (fundamentally deficient), 0 Absent/Misrepresented. Every score carries a one-line justification and 3–7 evidence citations.

**Confidence bands.** Each score is reported with a confidence band: **High** (directly verified in code/CI/config, reproducible), **Medium** (inferred from strong but indirect evidence such as docs plus partial code), **Low** (resting on claims not independently verifiable). The orchestrator **down-weights low-confidence inputs** in aggregation — High ×1.0, Medium ×0.7, Low ×0.4 — so the headline scorecard reflects evidentiary strength, not assertion volume.

**Evidence taxonomy (E1–E7, strongest to weakest).** Every factual claim is anchored to one of seven tiers: **E1** executed proof (a command run with its output), **E2** source code (`repo-path:line`), **E3** configuration/CI (workflow YAML, Makefile, Helm values), **E4** contract/schema (proto, JSON Schema, AsyncAPI), **E5** first-party doc (ADR, AGENTS.md — records intent, not delivery), **E6** marketing/README claim (lowest weight), **E7** external sources (CVE DBs, competitor docs, ecosystem data). A fact is labelled **VERIFIED** only when it rests on E1–E4; a claim resting only on E5–E6 is held strictly separate as **CLAIMED** and is never reported as verified.

**Mandatory drift test.** Because the subject has a documented history of documentation outrunning delivery, every agent ran a drift test within its scope: take the boldest claims in the area, attempt to verify each against code/CI, and report **VERIFIED / PARTIAL / CONTRADICTED / UNKNOWN**. Contradictions were escalated to the orchestrator's contradiction register and resolved by the evidence hierarchy (executed proof and code beat docs and README). This is what surfaced, for example, the half-built portability finding and the missing-LICENSE finding.

**Anti-overlap wave model (A–D).** Scope was partitioned so no two agents owned the same evidence, preventing double-counting: **Wave A** (8 agents) established ground truth from code, CI, and contracts; **Wave B** (6 agents) covered derived quality, performance, and maintainability dimensions; **Wave C** (9 agents) covered strategic, market, governance, and ecosystem dimensions; **Wave D** (3 agents plus the orchestrator) synthesised everything into risk, investment, and business-strategy verdicts. The orchestrator applied single-ownership boundaries, confidence-weighted aggregation, and VERIFIED-vs-CLAIMED separation throughout, and treated drift itself as a finding.

---

## Scope & limits

This is a **read-only static audit at a single point in time** — `main` @ `e3135a6`, audited 2026-06-20 by Waves A–C and synthesised 2026-06-21. It is one analyst orchestration pass, not a recurring engagement, and it did not include a management session, code-author interview, or access to private financials.

Items requiring a **live runtime or registry** could not be confirmed offline and are explicitly carried as **UNKNOWN** in the unknowns ledger rather than asserted: notably the **existence of cosign signatures / SLSA attestations on published GHCR images** (signing is configured in CI, presence unverified — no live `cosign verify`); **end-to-end runtime success and wall-clock of the demo path** (no Docker/Ollama in the diligence environment); and **live kind-cluster e2e and CI leg status at HEAD** (not observable offline). These gaps bound the confidence of the security and product dimensions and are the first items for confirmatory diligence.

Findings are synthesis-only: this front matter and the report it heads introduce **no new claims** beyond those the wave packets already established. The complete per-agent evidence — all 26 packets — lives in **Appendix A**, the four wave findings documents (`docs/due-diligence/2026-06-20-dd-wave-a-findings.md`, `-b`, `-c`, and `2026-06-20-dd-wave-d-findings.md`).

---

## Glossary

A full glossary of terms, dimension labels (D1–D16), evidence tiers (E1–E7), risk identifiers (R1–R27), and contradiction-register identifiers (C1–C8, C-ARGO) appears in **Appendix E**.
# 1. Executive Summary

> **Evidence standard (Part 2 §2.4).** Every factual claim below carries an evidence
> citation — a `repo-path:line` (E2/E3/E4), an executed command and its output (E1), or an
> external source (E7) — or is explicitly marked `UNKNOWN`. **VERIFIED** facts (E1–E4)
> are kept strictly separate from **CLAIMED** material (E5–E6 / roadmap). This section
> synthesises the 26 agent packets produced across Waves A–D; the full per-agent findings
> are Appendix A (`docs/due-diligence/2026-06-20-dd-wave-a-findings.md`, `-b`, `-c`, `-d`).
> The verdict and its evidentiary spine are the Wave D synthesis capstone (issue #1405);
> the audited code state is `main @ e3135a6` (Waves A–C audited static; Wave D synthesised).

## 1.1 Verdict (Part 8 §8.5)

Zynax is a **best-in-class engineering substrate wrapped around a business that does not
yet exist.** Across 26 diligence agents the verified evidence is internally consistent:
the code, supply-chain, test, and contract substrate are above market norm and proven by
execution, while the commercial and social substrate — distribution, maintainer depth, the
finished portability moat, enterprise identity — is absent or aspirational. The result is a
high-conviction view of *what the asset is* (a CNCF-grade control plane built solo in
roughly two months) and an equally high-conviction view of *why it is not yet investable as
a product.*

```
RECOMMENDATION:     Conditional / Watch   (confidence: Medium)
OVERALL SCORE:      5.4 / 10              (raw confidence-weighted mean 6.4, §8.3-discounted)
RISK PROFILE:       High                  (4 Critical · 9 High · 11 Medium · 3 Low; 0 deal-breakers)
BEST-FIT STRUCTURE: Acqui-hire (primary) · milestone-gated seed (secondary) · CNCF incubate not-yet
THESIS (one line):  A CNCF-grade, execution-verified engineering substrate and a real-but-half-built
                    portability moat, built by an exceptional solo engineer — back the builder + IP,
                    not the product, and only against milestones.
```

The central, recurring diligence finding is that **the gaps are enforcement-shaped and
social, not absence-shaped** — capabilities exist but are opt-in, soft, or single-authored.
None of the four Critical risks is an absolute deal-breaker; each is a cheap or scoped
condition-precedent. But under Part 8 §8.3 three gating rules bind the verdict below any
"Proceed" regardless of the numeric mean (§1.6). The honest, Truth-Pass culture — docs that
*under-state* reality, the opposite of the §1.10 history — is the single most reassuring
diligence signal and the reason bus-factor risk is a condition-precedent rather than a kill.

| Maturity / readiness axis | Status |
|---|---|
| Technical maturity | **Strong** — production-credible compute substrate |
| Commercial / operational maturity | **Early / pre-seed** |
| Enterprise-ready | **No** — no RBAC/SSO/OIDC, audit, or multi-tenancy (5.20 `final.md:1606-1611`) |
| CNCF-donatable as-shipped | **No** — no LICENSE/MAINTAINERS, 0 adopters (5.21 `final.md:44`) |
| Overall (confidence-weighted, §8.3 caps applied) | **5.4 / 10 · Confidence: Medium** |
| Risk profile | **High** — ≥1 unmitigated Critical (R1 missing LICENSE, as-shipped; §7.4) |

## 1.2 Confidence-weighted scorecard (D1–D16)

Each dimension-group score is the confidence-weighted mean of its contributing agents
(Part 7 §7.4: **High ×1.0, Medium ×0.7, Low ×0.4**). The numeric mean and the §8.3-gated
recommendation are reported separately and honestly: the raw weighted overall is **6.4**,
but §8.3 caps override it to a reported **5.4** because a Critical is never averaged away.

| Dim | Group | Score | Conf. | One-line rationale (key evidence) |
|-----|-------|:----:|:----:|-----------------------------------|
| **D1** | Product | 5.5 | Medium | Hero path runs zero-secret, but the "one command" demo needs 4 prerequisites (5.3 `03-product.md:103-111`; 5.11 `Makefile:162-175`) |
| **D2** | Market | 5.5 | Medium | Kagent owns identical framing at 0 adopters; engine-neutral IR moat real-but-contested (5.19 `final.md:1348-1354`; 5.4 `final.md:393-398`) |
| **D3** | Architecture | 6.0 | High | Engine-neutral IR with no engine types, but Argo never calls `IRInterpreter.Run`; non-HA stateful SPOFs (5.1 `argo_engine.go:62-98`; 5.16 `final.md:879-891`) |
| **D4** | Engineering | 7.7 | High | 14-linter / 0-issues, no blanket `//nolint`; near-zero debt; 11 Go modules block cross-service coupling (5.5 `golangci-lint.yml:17-33`) |
| **D5** | Security | 6.4 | Med-High | cosign+SBOM+SLSA trifecta, but mTLS fails open across 5 services; OpenSSF badge "no data" + no LICENSE (5.2 `tlscreds.go:21`; 5.22 `final.md:1213-1218`) |
| **D6** | Performance | 5.6 | High | Hot-path benches beat targets 5–300×, but task-broker fan-out unbounded and zero load tests anywhere (5.6 `service.go:80-87`) |
| **D7** | Open Source | 6.0 | High | No top-level LICENSE; bus factor 1; CNCF "NOT YET" verdict (5.8 `final.md:647-652`; 5.13 `final.md:1112-1131`; 5.21 `final.md:44`) |
| **D8** | DevOps | 8.0 | High | Build-once / promote-by-retag (scan==deploy), 21 workflows; production images amd64-only (5.9 `release.yml:160-204`) |
| **D9** | Testing | 8.0 | High | ≥90% blocking gate executed 92.1–100% on 7 domains; 306 BDD scenarios over all RPCs (5.7 `coverage-gates.env`) |
| **D10** | AI Workflow | 6.7 | High | Closed, traceable learnings loop; canvas-before-code only a soft gate; adapter-first no-SDK routing genuinely embodied (5.12 `APPLY_LOG.md:15-99`) |
| **D11** | Documentation | 7.0 | High | §1.10 doc-vs-tooling lag resolved at HEAD, but README self-contradicts its milestone table (5.10 `README.md:333-337` vs `446-464`) |
| **D12** | Governance | 7.0 | High | Honest self-correcting Truth-Pass, but "no single company controls" contradicted by bus factor 1 (5.13 `final.md:1112-1131`) |
| **D13** | Financial | 5.0 | Medium | Replacement cost 8–14 eng-years / ~$2.0–4.5M floor; monetization roadmap-CLAIMED only, 0 revenue (5.17 `17-investment.md:229-253`) |
| **D14** | Enterprise Readiness | 4.0 | High | No RBAC/SSO/OIDC (single static bearer key); no multi-tenant isolation; no audit log on mutating ops (5.20 `auth.go:13-26`) |
| **D15** | Acquisition Readiness | 5.0\* | Med-High | Acqui-hire fit (team/IP > product); 4 Critical condition-precedent risks, 0 deal-breakers; incubate blocked (5.23 `23-risk.md:23-32`) |
| **D16** | Repo Health & Innovation | 6.6 | High | Linear signed history, 0 merge commits, but bus factor 1 (one identity, 772 commits); innovation candidates reframe self-cited prior art (5.24 `git shortlog`) |

\* D15 is synthesized, not a weighted input. **Raw weighted overall = 6.4/10**
(Proceed-Conditional band on §8.1); **reported overall = 5.4/10** after §8.3 caps. See §1.6.

## 1.3 Top 5 reasons to proceed

Distilled from the de-duplicated top-10 green flags across all 26 agents (full list in
Appendix A / Section 4).

1. **Execution-VERIFIED test rigor.** The ≥90% coverage gate is a blocking CI gate proven
   at 92.1–100% on all 7 domains, with 306 BDD scenarios covering every RPC — quality is
   demonstrated, not asserted (5.7 `coverage-gates.env`, `_test-go.yml`). [E1/E3]
2. **Full supply-chain trifecta VERIFIED.** cosign + SBOM (syft SPDX) + SLSA provenance,
   build-once / promote-by-retag so the scanned artifact is the deployed artifact
   (5.2/5.9 `release.yml:160-204,201,510,527`). [E3]
3. **A genuinely engine-neutral IR contract + clean 5-method `WorkflowEngine` port.** No
   engine types leak into the IR — the moat is real and hard-to-copy *at the interface*
   (5.1 `workflow_compiler.proto:205-241`). [E2/E4]
4. **Substantial, high-quality embodied IP at near-zero debt.** ~40k non-test LOC, 9
   protos, 37 ADRs, 9 Helm charts, 21 CI workflows; 11 separate Go modules make
   cross-service coupling mechanically impossible; ~8–14 eng-years (~$2.0–4.5M floor) to
   replicate at this bar (5.17 `17-investment.md:229-253`; 5.14/5.15 `final.md:665-670`). [E2/E7]
5. **Rare honesty / Truth-Pass culture.** Docs *under-state* reality (opposite of the
   §1.10 history); M3/M4 were down-graded with reasons; CVE suppressions are dated. This
   directly de-risks the thing diligence most fears, and the risks that remain are known,
   bounded, and issue-tagged (5.13/5.8/5.25; 5.23 `23-risk.md:116-119`). [E2/E5]

## 1.4 Top 5 reasons for caution

Distilled from the de-duplicated top-10 red flags; the four Critical items are surfaced here
as required by Part 6.3 ("a Critical red flag must appear in the Executive Summary").

1. **Bus factor = 1 with 0-review self-merge (Critical, R2).** 772 commits by one human
   identity, sole ADR Decider, `required_approving_review_count=0`, and the entire
   prompt/canvas IP single-authored — the root multiplier under R3/R5/R12/R18/R24
   (5.24 `git shortlog`; 5.8/5.22/5.25). [E1/E2]
2. **Zero distribution vs a CNCF-backed rival (Critical).** 0 stars/forks/named adopters
   while Kagent (CNCF Sandbox) owns the identical "control plane for AI agents" framing
   with UI/MCP/HITL/GitOps/multi-LLM. For an OSS control plane the ecosystem *is* the moat,
   and it is a 0-node graph today (5.19, Critical; 5.4/5.8/5.21/5.18). [E2/E7]
3. **No top-level LICENSE file (Critical, R1).** Despite Apache-2.0 intent, a README badge,
   and 347 SPDX headers, there is no LICENSE in tree or git history — an Apache §4
   distribution defect and a CNCF hard blocker, unmitigated as-shipped (5.8
   `final.md:647-652`; 5.22, `git log -- LICENSE` empty). [E1/E2]
4. **No enterprise identity + no multi-tenant isolation (Critical, R3/R4).** A single static
   bearer key, a shared Temporal namespace, and no principal-attributed audit log on
   mutating operations — the first checkboxes of enterprise procurement fail (5.20
   `final.md:1606-1616`, `auth.go:13-26`). [E2]
5. **Headline portability moat CONTRADICTED at execution (High, R5 / C-ARGO).** Only Temporal
   interprets the IR; the Argo path serialises the IR to JSON, hands it to a smoke-stub
   that asserts payload-not-empty and exits 0, and `argo_engine.go:62-98` never calls
   `IRInterpreter.Run`. A buyer who tests "run on Argo" gets a workflow that reports Success
   without running anything — the boldest claim fails when tested, resolved 7 agents to 1
   (5.1 `argo_engine.go:62-98`; `argo-ir-interpreter.yaml:10-14`; +5.3/5.6/5.14/5.19/5.26). [E1/E2]

> **Recurring drift theme (High, R8).** Several of the above are instances of one pattern —
> delivery-vs-narrative drift: Argo marketed "Shipped", decorative OpenSSF/bench gates,
> "runnable" examples that hang, a README self-contradiction, and a non-existent "LangGraph
> engine" claim — the exact §1.10 class the Truth-Pass exists to kill (5.3/5.6/5.22/5.10/5.19).

## 1.5 Swing factors

The verdict is unusually elastic: it is gated *down* by social and one-line-fixable gaps, so
a small number of de-risking events would move it materially. The four factors that would
change the recommendation:

1. **One named external adopter / first community adapter** (today 0/0/0) — retires the
   Critical distribution risk and flips Conditional toward a Proceed-track. [5.4, 5.19, 5.25]
2. **Argo `IRInterpreter` parity shipped + a cross-engine parity test** — removes the §8.3
   contradicted-core-thesis cap and makes the portability moat real rather than designed. [5.1, 5.3]
3. **A signed co-maintainer / key-person retention + IP assignment** — lifts the bus-factor
   cap on acquisition-readiness and unblocks the CNCF path. [5.24, 5.13]
4. **A committed LICENSE file + `MAINTAINERS.md`** — a cheap precedent that unblocks
   Apache/CNCF donatability for any clean structure (one-line / one-file fixes). [R1, #494]

## 1.6 How the §8.3 gating rules bound the verdict

The raw confidence-weighted mean (6.4) sits in the Proceed-Conditional band, yet the
*actionable* recommendation is gated lower. Three Part 8 §8.3 rules each independently bind
the verdict, and the reported overall (5.4) reflects the §8.3 "never average away a Critical"
instruction:

| # | Gating rule | Trigger at HEAD | Effect on verdict |
|---|-------------|-----------------|-------------------|
| 1 | ≥1 unmitigated Critical → cap at Conditional/Watch | R1 LICENSE (as-shipped); R3/R4 identity/tenancy; zero distribution | **Caps the ceiling** at Conditional/Watch regardless of the numeric mean |
| 2 | Contradicted core-thesis claim → cap at Pass (Revisit) | C-ARGO: "run on Temporal OR Argo without a rewrite" contradicted at execution (7:1 evidence) | **Pulls the floor** toward Pass-Revisit on any structure whose value rests on the portability moat, until parity ships or the claim is re-labelled (CP-4) |
| 3 | Bus factor = 1, no succession → cap acquisition-readiness | R2 | **Caps the D15 sub-verdict** at Conditional; makes key-person retention the load-bearing diligence item for any non-acqui-hire structure |

**Net result.** Conditional / Watch (Medium), leaning toward Pass (Revisit) on any structure
whose value rests on the portability moat or CNCF-donatability until those are fixed. This is
concordant with the 5.17 Investment draft, which independently lands Conditional/Watch
(Medium), acqui-hire primary, and applies the same three caps; the only delta is the overall
figure — 5.4 here (the §8.3-discounted weighted mean of all D1–D16) versus 5.17's 5.0 D15
input, a 0.4 spread inside Medium confidence and immaterial to the band.

**Best-fit structure: acqui-hire (primary).** Team and IP materially exceed product maturity
and bus-factor risk dominates (§8.4): the substrate transplants cleanly (11 Go modules, near-
zero debt) and the proven solo builder is the multiplier, while the contradicted portability
thesis and zero traction hurt least when an acquirer supplies its own distribution.
**Secondary: a small milestone-gated seed** if the two de-risking triggers (named adopter +
co-maintainer) begin to move. **Right long-run home but NOT YET available: CNCF
incubate/sponsor**, blocked by R1 + 0 adopters. Conditions precedent CP-1…CP-6 (LICENSE →
bus-factor/succession plan → enterprise identity/tenancy → portability honesty → traction
trigger → managed-service SPOF/SLO baseline) are detailed in Sections 8 and 10.

> **Material unknowns bounding this verdict** (full ledger in Section 8): GHCR cosign
> signature *existence* is `UNKNOWN` offline (signing is configured, presence unverified —
> C4 residual); live `make demo` wall-clock against the <15-min hero claim is `UNKNOWN`; and
> the true engineering cost to bring a second engine to `IRInterpreter` parity — the single
> number that governs whether the headline thesis is fixable — is `UNKNOWN` and routed to
> CP-4.

<!-- Zynax investment-grade DD report · Section 1 Executive Summary · issue #1406 -->
<!-- Synthesised from Waves A-D (issue #1405); HEAD audited main @ e3135a6. -->
<!-- No new claims: re-organisation of the Wave D D0 Executive Summary. Appendix A = wave findings docs. -->

# 2. Company / Product / Technology Overview

> **Scope of this section.** This is the reconstructed product-and-technology picture: what
> Zynax claims to be, what its architecture and roadmap assert, and — most importantly — a
> VERIFIED-vs-CLAIMED capability map drawn from the consolidated contradiction register and
> drift findings of the synthesis wave. It is a synthesis of the wave findings
> (`docs/due-diligence/2026-06-20-dd-wave-a-findings.md` and `-b`/`-c`/`-d`); the full 26
> per-agent packets live in Appendix A. Evidence is graded on the Part 2 §2.4 taxonomy
> (E1 executed proof → E7 external), and **VERIFIED (E1–E4) is held strictly separate from
> CLAIMED (E5–E6/roadmap)** throughout. Citations are repo-relative `path:line` or a wave-doc
> reference; nothing here is asserted ungrounded.
>
> **HEAD audited:** `main @ e3135a6` (the code state Waves A–C audited; Wave D synthesised those
> packets). No claim in this section post-dates that commit.

---

## 2.1 The thesis — "Kubernetes for AI workflows"

Zynax positions itself as a **declarative control plane for AI-agent workflows** — explicitly
"what Kubernetes is to containers, for AI workflows" (`README.md`, corroborated in
`docs/product/strategy.md`). The promise is **portability and decoupling**: an operator
authors a workflow once in YAML, it compiles to an **engine-neutral Workflow Intermediate
Representation (IR)**, and that IR executes on any pluggable engine — Temporal, Argo, others —
or routes to any capability provider behind a stable gRPC contract. The core value proposition
is escape from per-engine and per-framework lock-in: *write once, run on any engine, with no
mandatory SDK* (sources: `README.md`, `docs/product/strategy.md` §1–§2,
`docs/adr/ADR-012-workflow-ir.md`, `docs/adr/ADR-015-pluggable-workflow-engines.md` — all
tier E5/E6, i.e. intent and marketing, not proof of delivery).

The problem statement the project sets for itself: *"Workflows defined for Temporal cannot run
on LangGraph. Without a control plane, every engine requires a different workflow definition"*
(paraphrase, `docs/architecture/` principal-architect review). The mission is a versionable,
declarative, engine-neutral API so organisations write AI workflows once and run them anywhere,
GitOps-native and SDK-optional.

The diligence's single most important framing finding qualifies this thesis at the outset. The
headline portability claim — *"run on Temporal **or** Argo without a rewrite"* — is **real at
the IR / contract / submission boundary but contradicted at the execution boundary**: only the
Temporal engine actually interprets the IR; the Argo path serialises the IR to JSON and hands
it to a smoke-stub that asserts the payload is non-empty and exits 0, never invoking the
interpreter (`services/engine-adapter/.../argo_engine.go:62-98`;
`scripts/e2e/manifests/argo-ir-interpreter.yaml:10-14`; resolved 7 agents to 1 as contradiction
**C-ARGO** in the Wave D synthesis, tier E1/E2). The moat is therefore **half-built**: genuine
at the interface, a stub at execution. This single finding governs the verdict's portability
gating (§2.4, and Part 8 §8.3) and recurs throughout this report.

## 2.2 The three-layer model (Intent / Communication / Execution)

The architecture is organised around a **three-layer separation the project treats as
non-negotiable** (`AGENTS.md` §The Three-Layer Separation; `docs/adr/ADR-011`, `ADR-012`,
`ADR-015`):

| Layer | Name | What lives there | Hard rule |
|---|---|---|---|
| **Layer 1** | **Intent** | `kind: Workflow \| AgentDef \| Policy \| Capability` YAML in `spec/` | Never imported by services |
| **Layer 2** | **Communication** | gRPC contracts in `protos/zynax/v1/` + AsyncAPI events in `spec/asyncapi/` | No business logic |
| **Layer 3** | **Execution** | Pluggable engines + adapters | Always behind an interface |

The architecture agents verify the contract substrate as **genuine, not aspirational**: the IR
is engine-neutral with no engine-specific types leaking into it
(`protos/.../workflow_compiler.proto:205-241`), and the `WorkflowEngine` port is a clean
five-method interface — confirmed by Wave A architecture (5.1) and reflected in the D3
Architecture score of **6.0 (High confidence)** and green flag #3 ("genuinely engine-neutral IR
contract + clean 5-method WorkflowEngine port … the moat is real and hard-to-copy AT THE
INTERFACE"). The qualifier matters: the interface is real and defensible; the second
*implementation* behind it (Argo) is the stub described in §2.1. One residual weakness is that
the hexagonal layer boundary is **convention-only** — there is no import-boundary linter despite
`services/AGENTS.md` claiming it is "CI-enforced" (Wave D R23, tier E2) — though 11 separate Go
modules make cross-service `internal/` coupling mechanically impossible (green flag #6).

## 2.3 The five non-negotiable mandates

The constitution (`AGENTS.md`) commits to five mandates; the diligence verdict on each at HEAD:

| # | Mandate | ADR | Diligence status at HEAD |
|---|---|---|---|
| 1 | **No shared DB across services** | `ADR-008` | Holds — 11 separate Go modules; per-service repos incl. Postgres (Wave A/D, E2). |
| 2 | **No Layer 1→3 coupling** | — | Holds by convention; **not linter-enforced** (R23, E2). |
| 3 | **Contracts before implementations** (SPDD) | `ADR-019` | Embodied — canvas-before-code, but a **soft** PR gate (`pr-checks.yml:231-233`, E3). |
| 4 | **Declarative-first** | `ADR-011` | Holds — YAML intent compiles to IR (E4). |
| 5 | **Event-driven state machines over DAGs** | `ADR-014` | Holds — interpreter is state-machine-based (E2). |

The mandates are **real and largely lived**, but two (mandates 2 and 3) are *soft / convention*
rather than *hard / enforced* — a recurring "enforcement-shaped, not absence-shaped" pattern the
synthesis identifies as the project's signature gap class.

## 2.4 VERIFIED-vs-CLAIMED capability map

This is the core artifact of the overview. Each row pairs a claimed capability with its status
at HEAD, graded **VERIFIED** (proven by execution/code/CI/contract, E1–E4) /
**PARTIAL** (configured or half-built) / **CONTRADICTED** (claimed but disproven) /
**UNKNOWN** (not verifiable offline). It consolidates the contradiction register (C1–C8 +
C-ARGO) and the drift findings from the Wave D synthesis.

| Capability | Claimed | Status at HEAD | Evidence |
|---|---|:---:|---|
| **Engine-neutral Workflow IR** | Write once, engine-independent representation | **VERIFIED** | No engine types in IR (`workflow_compiler.proto:205-241`; Wave A 5.1, E4). |
| **Multi-engine execution portability** ("Temporal OR Argo, no rewrite") | Headline moat | **CONTRADICTED (at execution)** | Only Temporal interprets the IR; Argo never calls `IRInterpreter.Run` (`argo_engine.go:62-98`; `argo-ir-interpreter.yaml:10-14`). C-ARGO, 7:1, E1/E2. **Half-built.** |
| **Capability routing via stable `AgentService` gRPC, no mandatory SDK** | Integration moat | **VERIFIED** | 2-RPC contract, 5 cross-language adapters, zero SDK import (Wave C 5.26, E2/E4). |
| **Workflow-compiler is stateless** | CLAUDE.md | **VERIFIED (stronger than claimed)** | Unbounded in-memory map removed; only a stale proto comment remains (C7; `workflow_compiler.proto:50-53`, E2). |
| **cel-go guard evaluator, fail-closed** | M5.B → M6 | **VERIFIED** | `cel-go` v0.28.1 drives `evalGuard`, fail-closed; bespoke fail-open evaluator replaced (C8; `interpreter.go:203,220-259`, E2). |
| **agent-registry implemented (Postgres-backed)** | early README → M5.C/M6 | **VERIFIED** | `services/agent-registry` with in-memory **and** Postgres repos (C6; `postgres/repository.go`, E2). |
| **CloudEvents publishing over NATS** | early README → M6 | **VERIFIED** | Real NATS JetStream client + `CloudEvent` type + `cloudevents.proto` + `Publish` RPC (C5; `nats.go:27-43,53`, `handler.go:36-71`, E2/E4). Not a log-stub at HEAD. |
| **SBOM per release** | SECURITY.md → M6 | **VERIFIED** | syft SPDX per-service digest attached to Release (C3; `release.yml:527`, E3). |
| **cosign-signed images + SLSA provenance** | SECURITY.md → M6 | **PARTIAL (UNKNOWN at GHCR)** | cosign + SLSA wired in `release.yml` (E3), but signature *existence* on published GHCR images unverifiable offline (C4; U1). Configured, presence UNKNOWN. |
| **mTLS between all services** | SECURITY.md / ADR-020 | **PARTIAL (fails open)** | Every service falls back to `insecure.NewCredentials()` when certs unset; 2 prod overlays omit TLS (C2; `tlscreds.go:21`, E2/E3). Configurable, **not enforced**; docs overstate "enforced". |
| **"M3 & M4 Complete"** | early README | **CONTRADICTED → corrected** | Docs now re-label M3/M4 *Partial* with reasons; current HEAD **under-states** reality (C1; Wave D 5.13/5.25, E5+E2). |
| **Apache-2.0 licensed / CNCF-credible** | README badge + GOVERNANCE | **CONTRADICTED** | No top-level LICENSE file in tree or git history (`git log --all -- LICENSE` empty), despite 347 SPDX headers; OpenSSF badge "no data" (R1; Wave D, E1/E2). |
| **Kagent "complementary, not rivals"** | README / strategy | **PARTIAL (prose hedge)** | Mechanism real (`agent.proto`), but `grep kagent` (excl. docs) → 0 code hits — defensive framing, not shipped integration (C-COMPLEMENTARY, E2). |
| **Bench regression gate "live"** | architecture review | **CONTRADICTED** | No workflow invokes it; fail-open if run (C-BENCH / R17, E2). Decorative gate. |
| **≥90% domain test coverage; BDD at every gRPC boundary** | ADR-016 | **VERIFIED** | Blocking gate executed 92.1–100% on 7 domains; 306 BDD scenarios / all RPCs (Wave A 5.7; `coverage-gates.env`, E1/E3). |
| **Enterprise identity (RBAC/SSO/OIDC) + multi-tenant isolation** | implied by "K8s for AI" | **CONTRADICTED (absent)** | Single static bearer key; shared Temporal namespace; no audit log (R3/R4/R10; `auth.go:13-26`, `temporal.go:54-58`, E2). |

**Contradiction-register tally at HEAD (C1–C8):** 5 VERIFIED (C3, C5, C6, C7, C8) · 2 PARTIAL
(C2, C4) · 1 CONTRADICTED-then-corrected (C1) · 1 UNKNOWN residual (C4, GHCR signature
presence). The single material *cross-agent* conflict, **C-ARGO**, resolved to *half-built
portability* by the evidence hierarchy (execution path beats file presence).

**The pattern that emerges from the map.** Almost every "fixed M6" claim is now genuinely
**VERIFIED** (C3, C5, C6, C7, C8) — the §1.10-history of claims-preceding-delivery has been
materially reconciled, and the project now *under-states* rather than over-states its delivery
(the single most reassuring diligence signal). The remaining red rows cluster into two distinct
problem classes: (a) **enforcement-shaped gaps** — capabilities that exist but are opt-in or
soft (mTLS fails open; bench gate decorative; layer boundary convention-only); and (b) two
**genuine absences with outsized leverage** — the unfinished second engine (C-ARGO) and the
missing enterprise-identity/tenancy layer. The missing LICENSE file is a third, trivially-fixed
contradiction (a one-line PR) that nonetheless reads as Critical because it is an Apache §4
distribution defect and a CNCF hard blocker.

## 2.5 Differentiators & moat (as claimed)

`docs/product/strategy.md` §5 asserts five differentiators; the diligence status:

| Differentiator | Claimed defensibility | Status at HEAD |
|---|---|:---:|
| Engine-agnostic IR + multi-engine dispatch | HIGH | **VERIFIED at interface / CONTRADICTED at execution** (C-ARGO) |
| Capability routing via stable `AgentService` gRPC | HIGH | **VERIFIED** (5 adapters, no SDK) |
| Event-driven state machines (loops, HITL) | Medium-high | **VERIFIED** (state-machine interpreter, ADR-014) |
| Declarative + GitOps-native YAML, idempotent apply | Medium | **VERIFIED** |
| No mandatory SDK | Medium | **VERIFIED** (ADR-013) |
| Compile-time structural IR validation *(under-marketed)* | — | **VERIFIED** (`structural.go:11-58`) — a rare, hard-to-copy edge the project under-sells |

The named direct competitor is **Kagent** (CNCF Sandbox 2026), with a near-identical "control
plane for AI agents" tagline and a fuller buyer-visible surface (Web UI, MCP/OpenAPI discovery,
multi-LLM, HITL, GitOps). The diligence flags the IP moat as **thin** — every candidate
innovation reframes well-established prior art (Beam/Dapr/XState/Step-Functions/Envoy), and the
public IR/port design is replicable by a funded rival in roughly two quarters (Wave C 5.26;
Wave D R13). The durable edge is **execution discipline**, which is copyable; the only
non-copyable moat would be an ecosystem/adoption lead that does not yet exist (0 stars / forks /
named adopters).

## 2.6 Roadmap status (M1–M8)

The project versions its delivery against eight milestones (sources: `ROADMAP.md`,
`CLAUDE.md` §Per-Milestone Scope, `state/current-milestone.md`). Diligence-verified status:

| Milestone | Status | Version | Scope (delivered / claimed) |
|---|---|---|---|
| **M1** | Complete | v0.1.0 | 8 gRPC contracts, AsyncAPI, 140+ BDD scenarios, CI gates |
| **M2** | Complete | v0.1.0 | Workflow IR compiler (YAML→protobuf), JSON Schemas, validation |
| **M3** | **Partial** | v0.2.0 | Temporal engine (`IRInterpreterWorkflow`, dispatch); task-broker/agent-registry slipped to M5.C |
| **M4** | **Partial** | v0.3.0 | api-gateway REST, `zynax` CLI, compose; end-to-end dispatch not wired until M5.C |
| **M5** | Complete | v0.4.0 | 5 adapters, task-broker + agent-registry MVP, Python SDK base, end-to-end dispatch, e2e-demo green |
| **M6** | Complete | v0.5.0 | K8s readiness: mTLS, SBOM/cosign/SLSA, Postgres repos, Helm, NATS event-bus, memory-service, ArgoEngine, PyPI SDK, gRPC health, Prometheus `/metrics` |
| **M7** | **Active** | v0.6.0 (target) | Usable workflows + observability: data-flow bindings, log/event streaming, OTEL+Uptrace, context propagation, git MCP shim, expert-agent substrate, templates, test rigor, supply-chain, docs |
| **M8** | **Aspirational** | v1.0.0 | CNCF Sandbox submission: ≥2 maintainers from 2+ orgs, external security audit, production reference deployment |

**M3/M4 honesty (C1).** The early README claimed M3 and M4 "Complete"; both are now
re-labelled **Partial with documented reasons** — end-to-end dispatch was not wired until M5.C.
This is the clearest example of the project's Truth-Pass culture self-correcting an over-claim,
and it now under-states rather than over-states.

**M7 (active).** The current milestone targets *usable workflows + observability* — data-flow
output/input bindings, execution log/event streaming, OTEL + Uptrace, context propagation, a git
MCP shim, an expert-agent substrate, reusable templates, and first-run UX. As of mid-June 2026
M7 carried roughly 57 closed / 34 open issues with several EPICs complete (`state/`,
`ROADMAP.md`). The enterprise-identity and multi-tenant primitives (ADR-021/022) are referenced
as "planned M7" but are **not enforced at HEAD** (R3/R4) — a gap that bounds enterprise GTM.

**M8 (aspirational).** v1.0.0 / CNCF Sandbox is the long-run home, but its binding gate is
**social, not technical**: ≥2 cross-org maintainers, an external security audit, and a filed TOC
application. The diligence's strongest roadmap-realism finding is that **this gate is unbuyable
by velocity** — the (excellent) technical cadence can land every M7 item on time and v1.0/CNCF
still stalls (Wave D R12, `ROADMAP.md:257-261`). Compounding it: a missing LICENSE file (R1) and
zero named adopters are *hard* CNCF blockers today, and a CNCF-Sandbox rival (Kagent) is already
winning the distribution race on identical positioning.

## 2.7 Overview verdict

Zynax is a **CNCF-grade engineering substrate wrapped around a business that does not yet
exist.** The technology overview is one of high-conviction *what-the-asset-is* clarity: a
declarative, engine-neutral control plane built solo in ~2 months (~40k non-test LOC, 306 BDD
scenarios, 37 ADRs, cosign+SBOM+SLSA supply chain), with most M6 "fixed" claims now genuinely
VERIFIED and a documentation culture that under-states reality. The qualifiers are equally
clear and load-bearing: the **headline portability moat is half-built** (CONTRADICTED at
execution, C-ARGO), the **enterprise primitives are absent** (no identity, no tenancy), and a
**one-line LICENSE defect** sits unfixed against a CNCF rival with all the distribution. The
gaps are predominantly **enforcement-shaped and social, not absence-shaped** — which is why the
synthesis classes them as cheap/scoped conditions-precedent rather than viability kills, and
why the verdict lands at **Conditional / Watch** (overall **5.4/10**, Medium confidence) with
an **acqui-hire** best-fit structure. Full scoring, the risk register, and the investment
recommendation follow in Parts 7–9; the capability map above is their evidentiary spine.
# 3. Product & Market

> **Section scope.** This section synthesises the five product-, market-, and adoption-facing
> agents of Wave C (Product 5.3, Market 5.4, Competitive 5.19, Enterprise Adoption 5.20, Developer
> Experience 5.11). Full per-agent packets — including every sub-score, drift-test, and citation —
> live verbatim in `docs/due-diligence/2026-06-20-dd-wave-c-findings.md` (Appendix A). All claims
> carry an evidence citation (a repo `path:line`, an executed command→output (E1), or an external
> source (E7)) or are marked UNKNOWN; **VERIFIED** facts rest on code/CI/contract evidence (E1–E4),
> while roadmap and marketing statements are labelled **CLAIMED** (E5–E6). Repository state audited:
> `main` @ `e3135a6`.

## 3.0 Section verdict and scores

The product is **real and conversion-ready on a single beachhead** — a zero-secret, one-command
local-LLM hero path that runs to completion on Temporal, with both M7 usability keystones
(data-flow bindings, OpenTelemetry/Uptrace) actually implemented rather than merely ADR'd. Beyond
that beachhead the picture narrows sharply: the headline "run on Temporal **or** Argo without a
rewrite" wedge is half-built (only Temporal interprets the IR), a CNCF-Sandbox-backed direct rival
(Kagent) out-positions Zynax on every buyer-visible axis while Zynax carries **0 stars / 0 forks /
0 named adopters**, and the enterprise identity, tenancy, and audit layers a Fortune-500 buyer
checks first are absent or cosmetic. The recurring failure mode across all five lenses is the same
delivery-vs-narrative **drift** the diligence exists to catch — the strongest shipped edge
(compile-time IR validation) is under-marketed while the half-built one (portability) leads.

| § | Dimension | Agent | Score | Confidence |
|---|-----------|-------|:-----:|:----------:|
| 3.1 | Product | 5.3 | **6** | Medium |
| 3.2 | Market & TAM | 5.4 | **6** | Medium |
| 3.3 | Competitive (incl. Kagent) | 5.19 | **5** | Medium |
| 3.4 | Enterprise adoption | 5.20 | **4** | High |
| 3.5 | Developer experience | 5.11 | **7** | High |

Un-weighted mean of this cluster ≈ **5.6 / 10** (directional; final aggregation is the
orchestrator's). The spread is informative: the product *experience* on its beachhead scores well
(DX 7, Product 6), while the *enterprise and competitive market posture* scores poorly (Enterprise
4, Competitive 5) — Zynax is a strong developer artifact on a narrow segment, not yet a defensible
market position.

**Critical red flag promoted to the Executive Summary:** *Kagent out-positions Zynax on every
buyer-visible axis (CNCF backing, web UI, MCP discovery, HITL, multi-LLM, ArgoCD/GitOps) while
Zynax has zero adopters and a half-built portability moat* (5.19, Critical).

---

## 3.1 Product (Agent 5.3 — Score: 6, Medium)

**Mission.** Judge product vision, completeness, value proposition, and adoption barriers — can
the hero journey be completed from docs alone, and is the "<15 min, two engines, traces" promise
achievable at HEAD?

**Verdict.** Zynax has a genuinely real, honest, zero-secret hero path **on Temporal**: `make demo`
boots an Ollama overlay and a local model reviews a git diff to completion, printed straight from
the CLI, with traces available in a ready Uptrace overlay
(`Makefile:162-205`; `spec/workflows/examples/code-review-ollama.yaml:1-14`;
`docs/quickstart.md:146-163`). Both M7 usability keystones are implemented at HEAD — data-flow
output/input bindings parse and compile in the IR, and OpenTelemetry is wired into five services
with an env-gated Uptrace compose overlay (`protos/zynax/v1/workflow_compiler.proto:158-168`;
`services/workflow-compiler/internal/domain/manifest.go:262-269`; `libs/zynaxobs/providers.go`;
`infra/docker-compose/docker-compose.observability.yml`). That is a strong, conversion-ready core.
The problem is that the differentiating headline — "**run on Temporal or Argo without a rewrite**"
— does not survive contact with HEAD, and three first-impression docs over-claim the runnability of
examples that hang.

### Sub-dimension scores

| Sub-dimension | Score | Conf | Evidence |
|---|:--:|:--:|---|
| Hero-journey completability (docs alone, traced E2E) | 6 | Med | `docs/quickstart.md:3-20,146-163`; `Makefile:162-205`; `code-review-ollama.yaml:1-14` |
| Runnable example inventory (real vs illustrative) | 5 | High | `e2e-demo.yaml:1-16`; `code-review-ollama.yaml:5-8`; `code-review.yaml:2-9`; `docs/examples/index.md:19-32` |
| Value-prop clarity vs Kagent | 7 | Med | `README.md:7-9,45-59`; `docs/product/strategy.md:169-194` |
| Completeness for hero (data-flow ADR-029, obs ADR-030) | 7 | High | `workflow_compiler.proto:158-168`; `manifest.go:262-269`; `libs/zynaxobs/providers.go` |
| Adoption barriers (Temporal · YAML-only · Day-0 · no UI) | 5 | Med | `Makefile:160`; `strategy.md:279-285`; `state/current-milestone.md:71`; `strategy.md:124` |
| Roadmap realism (M7/M8) vs history | 7 | Med | `state/current-milestone.md:63-73`; `strategy.md:113-124,330-346` |

### Drift test

| Bold claim | Result | Basis |
|---|:--:|---|
| Hero promise: "<15 min, two engines, traces, from docs alone" | **PARTIAL** | Temporal run VERIFIED; traces-in-Uptrace VERIFIED; "or Argo" FAILED — Argo is submission-only with no dispatch (`argo_engine.go:62-98,280-289`) |
| "Two engines shipped & tested without a rewrite" (`README.md:7`, `strategy.md:118`) | **CONTRADICTED** | No Argo IR interpreter / capability dispatch; shipped template is an alpine smoke-stub (`scripts/e2e/manifests/argo-ir-interpreter.yaml`); Argo CI leg is advisory (`e2e-smoke.yml:56-61`) |
| Data-flow bindings implemented; compiler no longer rejects `output:` (ADR-029) | **VERIFIED** | `workflow_compiler.proto:158-168`; `manifest.go:262-335` — note `strategy.md:122` is now stale on this |

### Red flags

- **High —** "Two engines without a rewrite" is the headline wedge, but Argo cannot execute a real
  workflow. The Argo path only *submits* serialized IR JSON as a `workflow-ir` param to an external
  WorkflowTemplate the repo never ships in production form; the only committed template is an alpine
  smoke-stub that validates the payload is non-empty and exits 0 with zero capability dispatch
  (`services/engine-adapter/internal/infrastructure/argo_engine.go:62-98,280-289`;
  `scripts/e2e/manifests/argo-ir-interpreter.yaml`). A buyer testing the differentiating claim on
  Argo gets a workflow that reaches "Succeeded" without running anything. (Cross-refs Wave A 5.1
  Architecture, "Argo execution stubbed".)
- **Medium —** "Runnable" example drift: `docs/examples/index.md:19-32`, `docs/faq.md:24-28`, and
  `strategy.md:256-260` headline three event-driven workflows (code-review / feature-implementation
  / ci-pipeline) as "real, runnable … verified against the running stack", but the workflows' own
  YAML headers say they *hang* from the CLI — no event source, undeployed capabilities
  (`spec/workflows/examples/code-review.yaml:2-9`). "Compile green" is conflated with "runs E2E",
  and the actually-runnable hero (`code-review-ollama`) is absent from the Examples Index.
- **Medium —** Stale README Quickstart still narrates "capability dispatch pending M5.C" and points
  users at the hanging `code-review.yaml`; the README hero asciinema cast is a literal PLACEHOLDER
  (`README.md:35-41,333-355`; `state/current-milestone.md:73`). First-impression surfaces lag the
  honest `quickstart.md`.
- **Low —** Reference agents are real but deterministic dependency-free stubs that demonstrate the
  contract, not a working agentic review out of the box (`agents/examples/AGENTS.md:12-25`).

### Green flags

- A genuinely runnable, zero-secret, one-command hero path exists at HEAD — the single most
  important conversion artifact — and it is honest about which examples do and do not run
  (`Makefile:128-205`; `code-review-ollama.yaml:1-14`).
- Both M7 usability keystones are actually implemented (data-flow bindings compile; OTEL wired into
  five services + Uptrace overlay), making the "traces in Uptrace" leg of the hero promise real
  (`workflow_compiler.proto:158-168`; `libs/zynaxobs/providers.go`).
- Differentiation vs Kagent is crisp and buyer-legible across README + strategy, and the strategy
  doc is unusually honest (explicit shipped/partial/aspirational split + Truth-Pass culture)
  (`README.md:45-59`; `strategy.md:113-130,169-194`).
- Every quickstart command maps to a real Cobra CLI subcommand (apply/result/events/logs/validate/
  status/get/init) (`cmd/zynax/cmd/*.go`; `docs/quickstart.md:237-254`).

**Recommendations:** P0 reconcile the "runnable" claim across README / Examples-Index / FAQ to
match quickstart truth; P0 stop asserting Argo as a co-equal shipped engine until Argo-side IR
interpretation ships; P1 record the hero cast and prove the <15-min timing; P1 add
`code-review-ollama` to the Examples Index; P2 surface/advance a zero-Temporal eval mode.

---

## 3.2 Market & TAM (Agent 5.4 — Score: 6, Medium)

**Mission.** Frame the category, comparables, timing, SWOT, moat, and network effects — and
stress-test the engine-portability and Kagent-complementarity theses.

**Verdict.** Zynax addresses a real, articulable category — a control plane above agent
orchestration, spanning durable engines — at a favourable but slightly-early moment in the
2025–26 agent-orchestration inflection. It has two genuine, partially-proven technical moats: an
engine-neutral IR executing behind a clean 5-method `WorkflowEngine` port
(`services/engine-adapter/internal/domain/engine.go:17-41`), and a no-SDK `AgentService` gRPC
contract that turns any external system into a capability (`protos/zynax/v1/agent.proto:1-47`,
ADR-013). Both moats are narrow, copyable-with-effort, and — critically — unprotected by traction.
The fatal-class weakness is distribution: a CNCF-Sandbox-backed rival (Kagent) holds the identical
tagline while Zynax sits at a 0-star / 0-fork / 0-adopter baseline with a single maintainer
(`docs/product/strategy.md:62-64,337-338`). TAM/SAM/SOM is entirely assumption-based (E7) with no
bottom-up sizing committed in-repo.

### Sub-dimension scores

| Sub-dimension | Score | Conf | Evidence |
|---|:--:|:--:|---|
| Category definition & TAM/SAM/SOM framing | 6 | Low | `strategy.md:134-157`; E7 CNCF Cloud Native AI landscape (agent-orchestration = emergent, unsized) |
| Comparables / competitive matrix | 6 | Med | `docs/architecture/2026-05-28-competitive-positioning.md:24-36`; `strategy.md:196-208`; E7 (Kagent = CNCF Sandbox) |
| Market timing (early / right / late) | 7 | Med | `README.md:7-9,47-49`; `strategy.md:152-157`; E7 (LangGraph/CrewAI/AutoGen proliferation 2024-25) |
| SWOT | 6 | Med | `engine.go:17-41`; `strategy.md:62-64,337-338` (zero traction, single maintainer = fatal-class) |
| Moat & defensibility | 6 | Med | `agent.proto:1-47`; `argo_engine.go:58-203` + `temporal.go`; `strategy.md:218-229` (adapter library = low moat) |
| Network effects / expansion vectors | 5 | Low | `ROADMAP.md:213,233`; `spec/templates/`; `strategy.md:296-297` (0 community adapters) |

### Drift test

| Bold claim | Result | Basis |
|---|:--:|---|
| "Complementary to Kagent, not rivals" — real co-existence story | **PARTIAL (a hedge)** | `AgentService` mechanism is real and shipped (`agent.proto:1-47`), but framed in the strategy/positioning docs as a way to avoid "a fight Zynax cannot yet win on ecosystem size" — no combined Kagent+Zynax integration test or reference in-repo (`strategy.md:190-194`) |
| "Run on Temporal or Argo without a rewrite" (CI matrix) | **PARTIAL** | Matrix is real (`e2e-smoke.yml:60-61`; `scripts/e2e/e2e-argo.sh:178-266`) but asymmetric — Temporal runs happy+failure paths, Argo runs a single happy-path assertion (`e2e-smoke.yml:147-173`). Portability is proven, not equally hardened |
| "Defensible category — the control plane for AI agent workflows" | **PARTIAL** | Category articulable and IR/AgentService moats real, but contradicted-as-*defensible* by its own docs: a CNCF-backed rival holds the same tagline → contested, not owned (`strategy.md:57-64,152-157`) |

### Red flags

- **High —** A CNCF-Sandbox-backed direct competitor occupies the identical "control plane for AI
  agents" framing while Zynax has zero stars/forks/adopters and a single maintainer; the rival is
  winning the one dimension (distribution/ecosystem) the market rewards for an OSS control plane
  (`strategy.md:57-64,200,337-338`; `competitive-positioning.md:35,57`).
- **Medium —** The "complementary to Kagent" story is more hedge than proven co-existence: the
  `AgentService` mechanism is real, but there is no integration test, reference deployment, or
  evidence a Kagent-adopting buyer would add Zynax rather than treat it as redundant control-plane
  overhead (`competitive-positioning.md:69-83`; `strategy.md:190-194`).
- **Medium —** TAM/SAM/SOM is entirely assumption-based (E7) with no bottom-up sizing; the
  portability pain that justifies the category is not yet acute for the majority of buyers who run
  a single engine — risk of a "real but premature" market (no TAM artifact in `docs/product/`;
  `strategy.md:240-242`).
- **Low —** The network-effect flywheel is theoretical: 0 community-built adapters exist and the
  expansion vectors (templates marketplace, hosted Zynax Cloud) are all roadmap-CLAIMED
  (`strategy.md:296-297`; `ROADMAP.md:213,233`).

### Green flags

- A hard-to-copy, **partially proven** technical moat: one engine-neutral IR executes on two real,
  independent engines behind a clean port, validated end-to-end in CI
  (`engine.go:17-41`; `argo_engine.go:58-266` + `temporal.go`; `e2e-smoke.yml:56-61`).
- A strong integration moat: the single no-SDK, no-framework, any-language `AgentService` gRPC
  contract is the structural basis for both co-existence and an ecosystem flywheel
  (`agent.proto:1-47`).
- Unusual marketing/reality discipline (Truth-Pass): the strategy doc separates shipped / partial /
  aspirational and concedes its own commoditized surfaces and zero-traction baseline — a
  credibility asset rare in this category (`strategy.md:113-130,218-229,287-291`).
- Favourable timing: agent-framework proliferation creates real demand for a control-plane
  consolidation layer; Zynax is early-but-not-late (E7; `README.md:7-9,47-49`).

**Recommendations:** P0 publish one reference deployment running the *same* workflow YAML unchanged
on Temporal **and** Argo (ideally alongside a Kagent-registered capability) with traces — converting
the partially-proven moat into citable market proof; P1 author a bottom-up SAM/SOM + explicit ICP
(which segment feels engine-lock-in pain *today*); P1 land the first community adapter and ≥1 named
adopter before the M8 CNCF filing and instrument an adoption funnel; P2 tighten the Argo CI leg to
Temporal parity and reconcile lagging docs.

---

## 3.3 Competitive — including Kagent (Agent 5.19 — Score: 5, Medium)

**Mission.** Assess the competitive landscape — especially the named rival Kagent — and judge
whether differentiation is defensible, legible, and survives contact with a buyer.

**Verdict.** Zynax has **one genuinely rare, shipped, hard-to-copy edge** — compile-time structural
validation of the workflow IR (terminal/orphan/cycle/transition-target checks run *before* any
engine executes the IR — `services/workflow-compiler/internal/domain/validators/structural.go:11-58`)
— plus a clean no-SDK capability contract. But it leads its marketing with the differentiator that
is only half-built: multi-engine portability (Temporal interprets the IR; Argo is a non-interpreting
stub; LangGraph-as-engine does not exist). Against Kagent, Zynax loses on every buyer-visible axis,
and the "complementary to Kagent" story is architecturally possible but prose-only — zero Kagent
code anywhere in the repo. **This is the most severe finding in the section: a Critical red flag.**

### Sub-dimension scores

| Sub-dimension | Score | Conf | Evidence |
|---|:--:|:--:|---|
| Engine-agnostic IR + multi-engine portability (headline moat) | 5 | High | `workflow_compiler.proto:205-241`; `engine.go:17-41`; `argo-ir-interpreter.yaml:10-14`; `main.go:177-178`; `engine-adapter/AGENTS.md:4` |
| No-SDK `AgentService` capability contract | 7 | High | `agent.proto:3-7,34-47`; `task-broker/.../service.go:18,48`; E7 (Restate 5-lang SDKs) |
| Compile-time structural IR validation | 7 | High | `validators/structural.go:11-58`; `competitive-analysis.md:261` |
| GitOps-native + Compose-AND-K8s deploy | 5 | Med | `apply.go:60,104`; `gitops/watcher.go:37,107`; `positioning.md:26`; E7 (Kagent + ArgoCD) |
| Kagent head-to-head (where each wins) | 4 | High | `positioning.md:32,35,36`; E7 (thenewstack / kagent.dev / cncf#360); `strategy.md:290` |
| "Complementary to Kagent" buyer credibility | 3 | Med | grep `kagent` excl. docs → 0 code hits; `positioning.md:75-78`; E7 (Kagent HITL/ArgoCD) |
| Time-to-parity of the moat | 5 | Med | `ADR-012`/`ADR-015` (public design); E7 (Dapr Agents v1.0 / Restate AI-agent orchestration) |

### Positioning matrix (June 2026, code- and E7-grounded)

| Dimension | Zynax | Kagent | Temporal | Argo WF | LangGraph | Restate | Dapr (WF/Agents) |
|---|---|---|---|---|---|---|---|
| Core abstraction | Engine-neutral IR | K8s CRDs, pod-per-agent | Durable code | K8s DAG | Python StateGraph | Durable code + journal | Durable WF + agents |
| Engine portability | ✅ at IR / ❌ at execution (Temporal-only; Argo stub) | ❌ ADK/K8s lock-in | N/A | N/A | N/A | ❌ runtime | ❌ runtime |
| Compile-time topology validation | ✅✅ shipped | ❌ runtime | ❌ runtime | ⚠️ DAG-shape | ❌ runtime | ❌ runtime | ❌ runtime |
| No-SDK / language-agnostic agents | ✅✅ any gRPC | ✅ any container | ✅ multi-lang SDK | ✅ container | ❌ Python-first | ✅ 5-lang SDK | ✅ any lang |
| LLM-native built-in | ⚠️ via llm-adapter | ✅ 7+ providers | ❌ | ❌ | ✅ built-in | ⚠️ via OpenAI SDK | ✅ built-in |
| Human-in-the-loop | ✅ semantic IR state | ✅ shipped | ✅ signals | ⚠️ | ✅ interrupts | ✅ | ✅ |
| Web UI | ❌ none (M8 CLAIMED) | ✅ GUI + CLI | ✅ | ✅ | ⚠️ LangSmith | ⚠️ | ⚠️ |
| GitOps-native | ✅ apply + watcher | ✅ ArgoCD-integrated | ❌ | ✅ (Argo CD family) | ❌ | ⚠️ | ⚠️ |
| Deploy surface | ✅ Compose **and** K8s | ❌ K8s-only | Any | K8s-only | Library | Single binary / K8s | K8s-native |
| CNCF status | ❌ M8 target | ✅ Sandbox | ❌ | ✅ Graduated family | ❌ | ❌ | ✅ Incubating |
| Production-proven / adopters | ❌ **0 adopters** | Partial, growing | ✅ | ✅ | ✅ | ✅ (early) | ✅ v1.0 |

*Zynax rows VERIFIED from code (`workflow_compiler.proto:205-241`; `validators/structural.go:11-58`;
`agent.proto:3-7`; `apply.go:60`; `strategy.md:290`); rival rows are E7 — see source list in
`docs/due-diligence/2026-06-20-dd-wave-c-findings.md` (Kagent, Restate, Dapr Agents, LangGraph).*

**Reading.** Zynax leads only two rows defensibly — compile-time topology validation (uniquely
✅✅) and the no-SDK gRPC contract (tied) — plus the Compose-AND-K8s deploy surface against Kagent's
K8s-only. On every maturity/legibility row (UI, CNCF, adopters, LLM-native) a backed rival is
ahead. The portability row — its headline — is the one marked half-built.

### Where each rival wins

- **Kagent** wins on every buyer-visible axis today: CNCF Sandbox backing (Solo.io), a shipped web
  GUI + CLI, MCP/OpenAPI tool discovery, 7+ LLM providers, HITL, K8s-native, and ArgoCD/GitOps
  integration (E7). It has also *absorbed* much of the "control plane" Zynax claims it lacks.
- **Restate / Dapr Agents** win on maturity: both ship production durable multi-agent orchestration
  at v1.0 (E7), so a fast-follower starts from a production base, not zero.
- **Zynax** wins only for a narrow buyer who specifically needs non-Temporal/non-K8s engine
  portability *and* compile-time topology validation *and* tolerates zero adopters.

### Drift test

| Bold claim | Result | Basis |
|---|:--:|---|
| Engine-agnostic IR — run on Temporal OR Argo (OR LangGraph) without rewrite | **PARTIAL** | VERIFIED at contract/submission; CONTRADICTED at execution (Argo non-interpreting stub — `argo-ir-interpreter.yaml:10-14`); CONTRADICTED on LangGraph (positioning doc says "Temporal + LangGraph adapters" `positioning.md:28`, but engine switch is `temporal\|argo` `main.go:177-178`; LangGraph is a *capability* adapter, not an engine — `engine-adapter/AGENTS.md:4`) |
| "Complementary, not competitive with Kagent" | **PARTIAL** | `AgentService` makes it possible (`agent.proto:3-7`); no Kagent adapter/example/test exists (grep → 0); Kagent shipping HITL + ArgoCD erodes the "Kagent lacks a control plane" premise (`positioning.md:77`) |
| "No tool combines YAML + formal IR + capability registry + pluggable engines — gap unoccupied" | **PARTIAL** | Formal compile-step genuinely rare (`structural.go:11-58`; `competitive-analysis.md:261`), but Dapr Agents (v1.0) and Restate now crowd the adjacent space the 2026-04-30 doc called empty (E7) |

### Red flags

- **Critical —** Kagent out-positions Zynax on every buyer-visible axis (CNCF, UI, MCP/OpenAPI,
  HITL, multi-LLM, ArgoCD/GitOps) while Zynax has 0 adopters and a half-built portability moat
  (E7; `positioning.md:32,35,36`; `strategy.md:290`; `argo-ir-interpreter.yaml:10-14`).
- **High —** The boldest differentiator is overstated in Zynax's own positioning doc ("Temporal +
  LangGraph adapters") vs code (`temporal|argo`; Argo stubbed; LangGraph = capability)
  (`positioning.md:28`; `main.go:177-178`; `engine-adapter/AGENTS.md:4`).
- **High —** "Complementary to Kagent" is prose-only (zero Kagent code) and undercut by Kagent
  absorbing control-plane features (grep → 0; `positioning.md:75-78`; E7).
- **Medium —** Short time-to-parity on the copyable layer (public IR/port/validators via ADR-012/015
  + open proto) while Restate/Dapr already ship production durable agent orchestration (E7).
- **Low —** The strongest shipped edge (compile-time validation) is under-marketed vs the half-built
  portability lead (`structural.go:11-58`; `competitive-analysis.md:261`).

### Green flags

- Compile-time structural IR validation — rare, shipped, hard-to-copy without an IR
  (`validators/structural.go:11-58`).
- A minimal, legible no-SDK `AgentService` contract — the only credible technical basis for any
  future complementary integration (`agent.proto:3-7,34-47`).
- Engine-neutral IR + clean `WorkflowEngine` port (Wave A 5.1 scored these 9 and 8) — the structure
  for a real moat exists *if* execution parity ships (`workflow_compiler.proto:205-241`).

**Recommendations:** P0 re-lead the narrative on the shipped, rare edge (compile-time validation +
no-SDK contract) and correct the "LangGraph adapter = portability" claim — stop marketing
portability as proven until a second engine interprets the IR; P1 build a real Kagent adapter + e2e
demo or drop the complementary claim, and ship multi-engine *execution* parity + a cross-engine
test before the contract layer is copied; P2 add Restate/Dapr to the competitive matrix and refresh
it for June 2026.

---

## 3.4 Enterprise adoption (Agent 5.20 — Score: 4, High)

**Mission.** From a Fortune-500 buyer-side lens, assess whether a large enterprise could adopt and
operate Zynax — compliance/audit, enterprise authN/Z, operability, multi-platform support,
supportability — and drift-test the "production-ready" claim.

**Verdict.** Zynax has a strong *platform* substrate but is **not enterprise-adoptable today** — it
is pilot-able, not procurement-approvable. The three controls a Fortune-500 security review checks
first are all missing or cosmetic: **no RBAC/SSO/OIDC** (a single shared static bearer key, with the
scoped-token BDD test still a `pending` stub — `services/api-gateway/internal/api/auth.go:13-26`;
`services/api-gateway/tests/steps_test.go:242`), **no real multi-tenant isolation** (all workflows
run in one process-wide Temporal namespace; the manifest namespace is correlation metadata only —
`services/engine-adapter/internal/infrastructure/temporal.go:54-58,76`), and **no audit log** on
mutating operations (`services/api-gateway/internal/api/handler.go:41-50`). Real strengths sit
around the edges — shipped, tested policy/quota/rate-limit enforcement; genuine OTEL/Uptrace
observability; a clean additive upgrade story with a credible Postgres backup/restore runbook; and
vendor-neutral Helm + Compose deployment — but they rest on an identity and tenancy layer an
enterprise cannot accept.

### Sub-dimension scores

| Sub-dimension | Score | Conf | Evidence |
|---|:--:|:--:|---|
| Enterprise authN/Z (RBAC/SSO/OIDC) | 2 | High | `auth.go:13-26`; `steps_test.go:242` (pending); `SECURITY.md:90`; `strategy.md:321,360` |
| Multi-tenant isolation | 2 | High | `temporal.go:54-58,76`; `handler.go:66-72`; principal-architect-review:350; `reviews/04:148` (ADR-021 PROPOSED) |
| Compliance & audit (audit log / certs / residency) | 3 | High | `handler.go:41-50`; external-review:331,795; grep audit/SOC2/GDPR → absent |
| Policy enforcement | 6 | High | `workflow-compiler main.go:143-155`; `quota_check_test.go:35`; `ratelimit.go:17-69`; `policy.schema.json:5` |
| Operability (OTEL / runbooks / upgrade / LTS) | 6 | High | `api-gateway main.go:90`; `migration-v0.6.md:5-12`; `postgres README:43-68`; no runbooks dir; Wave A 5.9 |
| Multi-platform (cloud / air-gap / K8s+compose) | 4 | High | `find helm/charts`; Wave A 5.9 (amd64-only services); no air-gap guide |
| Supportability (2am incident / SLA / escalation) | 3 | High | `SECURITY.md:23-26` (vuln-only); no SUPPORT.md/MAINTAINERS.md; Wave A 5.24 (bus factor 1) |

### Drift test

| Bold claim | Result | Basis |
|---|:--:|---|
| "K8s Production-Ready" (M6 / v0.5.0 milestone) | **PARTIAL / OVERSTATED for enterprise** | True for K8s deploy mechanics; false for enterprise governance — no RBAC/SSO, cosmetic tenancy, no audit log, no support model, single-Postgres SPOF (`ROADMAP.md:160`, `README.md:407` vs `auth.go:13-26`, `temporal.go:54-58`) |
| `SECURITY.md:44` "multi-arch arm64 for all service images (✅)" | **CONTRADICTED** | Service images are amd64-only (Wave A 5.9; `ci.yml:861`); arm64 holds for tools/CLI, not services |
| `SECURITY.md:89` "mTLS between all platform services (✅ shipped)" | **CONTRADICTED / fails-open** | Code falls open to insecure creds when certs unset (Wave A 5.2; `tlscreds.go:20`); prod overlays omit `tlsSecretName` |

### Red flags

- **Critical —** No enterprise identity layer (RBAC/SSO/OIDC); a single shared static bearer key,
  binary authZ, no per-caller identity to audit or revoke — a hard procurement blocker that fails
  the first checkbox of nearly every enterprise security questionnaire (`auth.go:13-26`;
  `steps_test.go:242`).
- **Critical —** No multi-tenant isolation; the namespace is metadata only and all workflows execute
  in one shared Temporal namespace — noisy-neighbour and data-bleed risk, marketed as "Kubernetes
  for AI workflows" yet lacking K8s's foundational tenancy primitive
  (`temporal.go:54-58`; `reviews/04:148`).
- **High —** No audit log on mutating control-plane operations (apply / delete / publish-event);
  bearer + rate-limit only, so an action cannot be attributed to a principal — fails SOC2 CC6/CC7
  audit-trail expectations (`handler.go:41-50`).
- **High —** No production support / incident model: no SUPPORT.md, no MAINTAINERS.md, no
  on-call/escalation/SLA, bus factor 1 — no entity to contract for a 2am outage (the 48h SLA in
  `SECURITY.md:23-26` is vuln-disclosure only) (no SUPPORT.md/MAINTAINERS.md; Wave A 5.24).
- **Medium —** Single-instance Postgres SPOF + no operational runbooks; HA is an explicitly future
  CloudNativePG migration (ADR-026) (`helm/charts/postgres/README.md:67-68`; no `docs/runbooks`).
- **Medium —** Doc-vs-artifact drift undermines a security questionnaire: `SECURITY.md:44,89`
  overstate arm64 and mTLS as shipped (vs Wave A 5.2/5.9) — exactly what an enterprise review probes
  and finds wanting.

### Green flags

- Policy enforcement is real and tested: PolicyGate (routing + quota), QuotaChecker returning
  `RESOURCE_EXHAUSTED`, and per-IP rate-limiting at the gateway — more than most pre-1.0 control
  planes ship (`workflow-compiler main.go:143-155`; `quota_check_test.go:35`; `ratelimit.go:17-69`).
- A genuine observability stack: OTEL traces/metrics/logs wired in service code (off by default) +
  Uptrace deployable via Compose overlay or Helm subchart (`api-gateway main.go:90`;
  `opentelemetry.md:13`).
- A clean additive upgrade story and a real DB lifecycle runbook (`migration-v0.6.md:5-12`;
  `postgres README:43-68`).
- Cloud-neutral deployment: vendor-agnostic Helm umbrella + Compose, PDB/NetworkPolicy per service
  (`find helm/charts`; `pdb.yaml:3`, `networkpolicy.yaml:3`).

**Recommendations:** P0 replace the static bearer key with OIDC/JWT + role claims and per-principal
authZ (unblocks the single largest procurement gate), and stop labelling SECURITY.md controls as
shipped when they are not; P1 implement real multi-tenant isolation (finalize ADR-021) and add an
audit log + SUPPORT.md/MAINTAINERS.md; P2 author operator runbooks, ship arm64 service images (or
document amd64-only as a supported constraint), and publish SLO values.

---

## 3.5 Developer experience (Agent 5.11 — Score: 7, High)

**Mission.** Assess contributor and user experience — setup, build, local run, feedback loops,
tooling ergonomics, error messages, CLI UX, and time-to-first-success — and drift-test the "<15 min"
/ one-command-demo claims.

**Verdict.** Zynax's DX is well-engineered and **above market norm where it counts**: a
self-documenting 72-target Makefile (`make help` renders ★-annotated entry points —
`Makefile:40-42`), a thoughtful Cobra CLI with structured line-numbered errors and script-friendly
exit codes (`cmd/zynax/cmd/root.go:18-23`; `apply.go:121-137`; `status.go:26`), a complete
guard-railed compose lifecycle (`Makefile:85-99`), an honest pre-commit story
(`.pre-commit-config.yaml:1-11`), and a genuinely zero-secret local-LLM path needing only Docker
plus one model pull. The headline weakness is a precise instance of the same drift class at the
front door: the README markets `make demo` as "one command", but on a clean machine that single
command exits immediately with `❌ zynax CLI not found` because `make demo` only *checks* for the
CLI and never installs it, and it additionally needs a host `ollama` and a pulled model
(`README.md:24-33` vs `Makefile:162-175`). The true first-run is a four-prerequisite chain.

### Sub-dimension scores

| Sub-dimension | Score | Conf | Evidence |
|---|:--:|:--:|---|
| Time-to-first-success (steps & footguns) | 6 | High | `README.md:24-33`; `Makefile:162-175`; `quickstart.md:24-72`; `developer-guide.md:13-18` |
| Local dev loop (compose/logs/reset/speed) | 8 | High | `Makefile:85-99,115-126,207-209`; TOOLS_RUN `35-37,234-238` |
| Tooling ergonomics (help/errors/GOWORK/hooks) | 8 | High | `Makefile:40-42` (72 targets); `CLAUDE.md:68`; `.pre-commit-config.yaml:1-54` |
| Contributor friction (SPDD moat vs wall) | 5 | High | `wc -l CONTRIBUTING.md` → 515; `CLAUDE.md` «SPDD»; Wave A 5.12 (soft canvas gate) |
| CLI UX (discoverability/help/errors) | 8 | High | grep `Short:` → 21 cmds; `root.go:18-23`; `apply.go:121-137`; `gateway.go:100-103,330-332` |
| Documentation-as-DX (consume 5.10) | 7 | High | Wave A 5.10 (onboarding 8/High); `faq.md:13-33`; `local-dev.md:60-117` |

### Drift test

| Bold claim | Result | Basis |
|---|:--:|---|
| "`make demo` — one command, end-to-end" | **PARTIAL** | Wiring VERIFIED (`Makefile:162-205`), but it is a 4-prereq chain (CLI build + PATH + host ollama + model pull); demo exits if the CLI is absent (`Makefile:163-164`); setup documented elsewhere as two commands (`developer-guide.md:13-18`) |
| "Time-to-first-working-workflow <15 min, one command" | **PARTIAL (CLAIMED)** | A strategy-doc target (`strategy.md:294`), not instrumented; the documented manual path is 8 steps and wall-clock is dominated by uninstrumented GHCR/compose/model pulls — no committed measurement |
| "Docker-only — nothing else needs Go locally" | **PARTIAL** | True for lint/test/generate via the tools container, but `make install-cli` compiles with a local Go 1.26.3 toolchain (`Makefile:211-213`); Docker-only holds only via the release-binary route |

### Red flags

- **Medium —** "One command" is a multi-prereq chain; a clean-machine `make demo` exits with
  `❌ zynax CLI not found` — the boldest DX claim, at the highest-traffic surface (`README.md:24-33`
  vs `Makefile:162-175`).
- **Medium —** Stale README "M5 status note" *under*-claims shipped capability dispatch and
  contradicts the file's own service table (`README.md:333-337,355`; consumed from Wave A 5.10).
- **Medium —** Heavy contributor onboarding (515-line CONTRIBUTING + SPDD canvas-before-code for
  `feat:` PRs) over a bus-factor-1 maintainer with a 2-day review SLA (`CONTRIBUTING.md:13-24,41-42`;
  Wave A 5.24) — softened by `fix:`/`docs:`/`chore:` being SPDD-exempt and the canvas gate being soft.
- **Low —** No friendly remedy on connection-refused; CLI default 8080 ≠ stack's mapped 7080
  (`gateway.go:330-332`; `root.go:40` vs `quickstart.md:94-99`).
- **Low —** Per-service container spin-up per lint/test taxes the inner loop
  (`Makefile:234-238,267-273`).

### Green flags

- Self-documenting Makefile: 72 ★-annotated targets via `make help` (`Makefile:40-42`).
- Genuinely zero-secret, zero-cloud runnable path; demo model single-sourced from config so it
  cannot drift (`Makefile:152-160`; `quickstart.md:55-72`).
- Thoughtful, script-friendly CLI: SilenceUsage, env-default API URL, dry-run/validate previews,
  exit codes 0 (terminal) / 2 (running), structured line-numbered compiler errors (`root.go:18-23`;
  `apply.go:121-137`; `status.go:26`).
- Complete guard-railed compose lifecycle + honest pre-commit managed/system split
  (`Makefile:85-99`; `.pre-commit-config.yaml:1-11`).
- The GOWORK=off footgun is documented in four surfaces and auto-set by make
  (`CLAUDE.md:68`; `local-dev.md:90`; `CONTRIBUTING.md:136`; `developer-guide.md:124`).

**Recommendations:** P0 make the "one command" claim true (a `make quickstart` umbrella chaining
bootstrap → install-cli → ollama-pull → demo) or re-label it as "three steps"; P1 truth-pass the
README Quickstart (delete the stale M5 note, record the asciinema cast) and add a friendly
connection-refused CLI hint; P2 add a casual-contributor fast path stating the SPDD canvas is
`feat:`-only, and commit a measured cold-clone <15-min timing to upgrade the claim from CLAIMED to
VERIFIED.

---

## 3.6 Cross-cutting product & market themes

Synthesising across the five lenses, three themes recur and reinforce one another:

1. **The binding constraint is social, not technical.** Bus factor 1, zero named adopters, and a
   CNCF-backed rival on the identical positioning are flagged independently by Market (5.4),
   Competitive (5.19), and Enterprise (5.20). Engineering velocity cannot buy distribution,
   ecosystem, or a second maintainer. (See also Wave A/B governance findings.)

2. **The product is real on its beachhead and narrow beyond it.** The Temporal hero path runs and
   the M7 keystones shipped (Product 6, DX 7), but the "two engines" headline (Product, Competitive),
   enterprise identity and tenancy (Enterprise 4), and multi-tenancy are not there yet.

3. **Delivery-vs-narrative drift is the consistent failure mode — at the highest-traffic surfaces.**
   The same class of overstatement recurs in the README hero line, the Examples Index, the
   competitive positioning doc ("LangGraph adapter"), and SECURITY.md (arm64, mTLS). Notably, the
   strongest *shipped* edge — compile-time IR validation — is the one that is *under*-marketed.

A countervailing positive: the project's own Truth-Pass culture means its strategy docs more often
*under*-state than over-state reality (Market, DX green flags) — a genuine diligence asset and the
opposite of the historical concern, even though the drift gaps above show the discipline is not yet
applied uniformly across all surfaces.

> **Cross-references to other sections.** The Argo-stubbed execution finding is owned by §4
> Technology & Architecture (Wave A 5.1); the mTLS fail-open and shared-key findings are owned by §5
> Security (Wave A 5.2); the amd64-only images and no-DORA/SLO findings are owned by §6 Quality &
> Delivery (Wave A 5.9); the missing LICENSE/MAINTAINERS and bus-factor-1 findings are owned by §7
> Open Source, Governance & CNCF (Wave C 5.8/5.13/5.21). Full packets: Appendix A
> (`docs/due-diligence/2026-06-20-dd-wave-c-findings.md`).

# 4. Technology & Architecture

> **Scope.** This section synthesises the seven technical due-diligence packets that audited Zynax at
> `main` @ `e3135a6`: Architecture (Wave A §5.1), Engineering (Wave A §5.5), Scalability (Wave B §5.16),
> Performance (Wave B §5.6), Technical Debt (Wave B §5.14), Maintainability (Wave B §5.15), and
> Innovation & IP (Wave B §5.26). The full per-agent packets — handoff YAML, every sub-dimension score,
> and the complete evidence index — live in the wave findings documents
> (`docs/due-diligence/2026-06-20-dd-wave-a-findings.md` and `-b`), which constitute Appendix A.
>
> **Evidence discipline (framework Part 2).** Every factual claim below carries an evidence citation —
> a `repo-path:line`, an executed command and its output (E1), or an external reference (E7) — or is
> marked `UNKNOWN`. **VERIFIED** facts rest on E1–E4 (executed proof, source, CI/config, contract);
> **CLAIMED** items rest on E5–E6 (first-party docs, marketing/roadmap) and are never reported as
> verified. No new findings are introduced here: this is synthesis and prose-ification of what the
> source packets already established.

## 4.0 Dimension summary

| § | Dimension | Score | Confidence | Scale |
|---|-----------|:-----:|:----------:|-------|
| 4.1 | Architecture (D3) | **7 / 10** | High | Standard |
| 4.2 | Engineering (D4) | **8 / 10** | High | Standard |
| 4.3 | Scalability (D3/D6) | **5 / 10** | High | Standard |
| 4.4 | Performance (D6) | **6 / 10** | High | Standard |
| 4.5 | Technical debt (D4) | **8 / 10** | High | **Inverted** (high = low debt) |
| 4.6 | Maintainability (D4) | **7 / 10** | High | Standard |
| 4.7 | Innovation & IP (D16/D10) | **6 / 10** | Medium | Standard |

> **Note on aggregation.** The 7/10 for Architecture above is agent 5.1's **raw sub-score**; the
> headline **confidence-weighted D3 group score is 6.0** (§1 scorecard, Appendix C), which blends
> Architecture with Scalability (5.16, 5/10) per the Part 7 §7.4 dimension model. Both figures are
> correct at their level of aggregation; the report's overall score uses the weighted group scores.

**Headline.** The technology dimension is the strongest, most independently verifiable part of the
Zynax story — but it is verified *unevenly*. The engineering substrate (code quality, lint, coverage,
technical-debt hygiene, change-amplification) is consistently strong and reproduced by executed
commands. The architecture is genuinely well-built at the contract and topology level. The two
material weaknesses — both surfaced repeatedly across packets — are (1) the headline multi-engine
portability moat is real at the *interface* but **only Temporal interprets the IR** at execution, and
(2) the stateful runtime substrate (Postgres + NATS JetStream) is **single-instance, non-HA by
design**. A recurring meta-theme runs through every sub-section: *capabilities exist; their
enforcement or proof lags.* mTLS, the import boundary, the benchmark gate, the coverage re-gate, and
the multi-engine claim all exist but are opt-in, soft, partial, or unproven rather than hard-enforced.

---

## 4.1 Architecture — Score 7 / 10 (High confidence)

**Verdict.** Zynax's control-plane architecture is genuinely well-built at the contract and topology
level — and that is not a marketing assessment, it is reproduced in code, CI, and the proto contracts.
The design is not a monolith-in-disguise; the bones are sound, modular, and extensible. The single
material weakness is that the headline moat — multi-engine portability — is real at the
submission/interface boundary but only Temporal genuinely interprets the workflow IR. That gap, plus
the absence of any automated layer-boundary linter, keeps this a strong-7 rather than a 9.

### 4.1.1 The engine-neutral IR and the WorkflowEngine port (the genuine strength)

The clearest architectural asset is a genuinely engine-neutral Workflow IR. `WorkflowIR` is a pure
state-machine contract — `workflow_id` / `states` / `initial_state` / `ir_version` — carrying *zero*
Temporal or Argo types (`protos/zynax/v1/workflow_compiler.proto:205-241`). Human-in-the-loop is
modelled generically as a first-class state type, not bolted on
(`workflow_compiler.proto:122-135`, `STATE_TYPE_HUMAN_IN_THE_LOOP`), and engine identity is a
free-text string (`'temporal'` / `'argo'` / `'langgraph'`) rather than a typed enum that would bind
the contract to one engine (`protos/zynax/v1/engine_adapter.proto:107,157`). This scored **9 / 10** —
the highest sub-dimension in the section.

The engine abstraction is equally clean: a single 5-method `WorkflowEngine` port
(`Submit, Signal, Cancel, GetStatus, Watch`) at
`services/engine-adapter/internal/domain/engine.go:17-41`, which both engines satisfy with
compile-time conformance assertions (`argo_engine.go:314` —
`var _ domain.WorkflowEngine = (*ArgoEngine)(nil)`). Engine selection is a config switch per ADR-015
(`cmd/engine-adapter/main.go:185-197`). Adding a third engine is therefore a bounded task — implement
five methods plus a switch case (sub-score **8 / 10**).

### 4.1.2 Service topology and contract discipline (verified)

The 7-service topology holds the no-shared-DB / gRPC-only invariants in code, not just in docs
(sub-score **9 / 10**). All eight inter-service edges are gRPC
(`services/api-gateway/internal/infrastructure/clients.go:77-92`;
`engine-adapter/main.go:291,302`; `task-broker/.../registry_client.go:28`), there is no REST between
services, and the stateful services own *separate* Postgres databases
(`infra/docker-compose/postgres-zynax-init.sql:5-6` provisions `task_broker` and `agent_registry` as
distinct DBs). A grep for cross-service `internal/` imports returns none — ADR-001/008 hold mechanically.

Contract evolution is disciplined (sub-score **8 / 10**): a single `zynax.v1` namespace, a
`buf breaking` gate that runs on every proto-changed PR against the base branch
(`.github/workflows/pr-checks.yml:129-135`; `protos/buf.yaml:22-24`), explicit additive-only / reserved
rules (`protos/AGENTS.md:56-62`), and structured `CompilationError` responses rather than errors smuggled
through gRPC metadata (`workflow_compiler.proto:15-17`).

### 4.1.3 The portability gap (the central red flag)

The boldest claim in the entire thesis — "the same workflow runs on Temporal **or** Argo without a
rewrite" — drift-tests to **PARTIAL**, and this is the single most important architecture finding in
the report. It is **VERIFIED at the submission/contract boundary**: the same YAML compiles to the same
`WorkflowIR`, handed to the same `WorkflowEngine` port that both engines satisfy. It is
**CONTRADICTED at the execution boundary**: only Temporal's `IRInterpreterWorkflow` actually traverses
states and dispatches capabilities. The Argo path serialises the IR to JSON and hands it to a
cluster-side stub template that asserts the payload is non-empty and exits 0 —
"capability-dispatch parity ... is deliberately out of scope for the smoke gate"
(`scripts/e2e/manifests/argo-ir-interpreter.yaml:10-14,56-60`). The Argo engine's `Submit` never calls
`IRInterpreter.Run` (`services/engine-adapter/internal/infrastructure/argo_engine.go:62-98`), and the
e2e Argo leg asserts only that the Argo Workflow CR reached `phase==Succeeded`, with no
capability-dispatch or state-transition parity assertion (`scripts/e2e/e2e-argo.sh:232-266`). The
operational-portability sub-dimension scored **4 / 10**. A second engine is wired *structurally*, not
made *functionally equivalent*.

A related "two real engines validated per CI run" claim is also **PARTIAL**: the engine matrix exists
(`.github/workflows/e2e-smoke.yml:60-61`, `matrix: engine: [temporal, argo]`, `fail-fast: false`) but
the Argo leg is path-gated and is *not* in the branch-protection required-check set
(`e2e-smoke.yml:16-19`), and its assertions are markedly weaker than the Temporal leg's
happy + failure + helm-rollback coverage.

### 4.1.4 Secondary architecture findings

- **No automated layer-boundary enforcement (Medium).** The hexagonal / three-layer invariant
  (domain imports zero `api`/`infrastructure` packages) holds in practice — a grep over
  `services/*/internal/domain/` returns no `api`/`infrastructure` imports — but it is defended by code
  review and convention only. The linter config (`tools/golangci-lint.yml:17-33`) contains no
  `depguard`, `import-boundary`, or fitness-function check. One careless import re-couples the layers
  silently. (This finding recurs verbatim in §4.6 Maintainability, where the *onboarding doc itself*
  claims the boundary is CI-enforced.)
- **ADR-vs-code drift on idempotent apply (Medium).** Shipped api-gateway derives a deterministic
  SHA-256 manifest hash for the workflow id, yielding idempotent apply
  (`services/api-gateway/internal/domain/apply.go:29-35,111-131`), but ADR-034 is still
  status *Proposed* and specifies *random* 64-bit ids — the decision record contradicts the code at HEAD.
- **Dead contract field (Low).** `SubmitWorkflowRequest.engine_hint` is plumbed end-to-end from the
  `?engine=` query param but the engine-adapter handler ignores it
  (`services/engine-adapter/internal/api/handler.go:36-48`); engine selection is process-wide via env.
  Per-request engine routing is not implemented — multi-engine means multiple deployments.
- **Stale durability comment (Low).** `workflow_compiler.proto:50-53` still documents the IR store as
  an "unbounded in-memory map ... planned for M6," even though the C7 refactor that removed it has
  landed (see §4.4 and §4.5).

---

## 4.2 Engineering — Score 8 / 10 (High confidence)

**Verdict.** This is strong, production-credible engineering that would largely pass a Staff+ review at
a top org. Across six sampled domain packages the code is pure, cohesive, hexagonally clean, and
consistently documents *why* (with ADR/canvas citations) rather than *what*. Critically, the two
headline quality claims were **reproduced by executed commands (E1)**, not taken on narrative.

### 4.2.1 The two verified claims

The ≥90% domain-coverage claim is **VERIFIED** and reproduced across all seven Go services by executed
`go test ./internal/domain/... -cover` runs: task-broker 92.1%, engine-adapter 92.8%,
agent-registry 94.0%, api-gateway 96.7%, workflow-compiler 97.5/92.7/95.9% (domain/ir/validators),
memory-service 100%, event-bus 100%. It is a hard CI gate that fails the build below threshold
(`Makefile:286-302`; `tools/coverage-gates.env`), not a soft documentation target.

The strict-lint claim is also **VERIFIED**: the 14-linter v2 config (gosec, wrapcheck, errorlint,
cyclop ≤16, funlen, contextcheck, errcheck, …) with `default: none`
(`tools/golangci-lint.yml:17-33`) returns "0 issues." when run against task-broker, engine-adapter,
and workflow-compiler. There is **zero blanket `//nolint`** anywhere — every suppression names a
specific linter, and the higher-risk gosec cases carry inline justifications.

### 4.2.2 Code quality, boundaries, and consistency

Domain code quality scored **9 / 10**: sentinel errors with `%w` wrapping, scoped access control that
fails loud (`datacontext.go:44`, `ScopeError` on cross-run denial — no silent fallback), and tight
cohesion across the sampled packages. Layer boundaries scored **9 / 10**: even a 453-line
`engine-adapter/main.go` contains pure wiring and lifecycle code with zero business logic
(`cmd/engine-adapter/main.go:5,185`). Cross-service consistency comes from genuine shared platform
libraries — `libs/zynaxobs` (tracing/metrics/propagation) and `libs/zynaxconfig` are used by every
service rather than copy-pasted (sub-score **8 / 10**). Comment discipline (sub-score **9 / 10**)
explains rationale — determinism, detached-context lifetime, scope isolation — with ADR citations, a
Staff+ habit.

### 4.2.3 The one real blemish, and minor gaps

- **ADR-010 contradicted (Medium).** ADR-010 mandates "AgentRuntime is a Protocol — never a base
  class" (`docs/adr/ADR-010-pluggable-agent-runtime.md:54`), but the Python SDK ships
  `class Agent(AgentServiceServicer, ABC)` (`agents/sdk/src/zynax_sdk/agent.py:59`), and no
  `Protocol`/`runtime_checkable`/`AgentRuntime` exists in the SDK source. Functional impact is low (the
  gRPC proto is the true boundary) but it is exactly the doc-vs-code drift class this diligence exists
  to catch. The Python adapter/SDK quality sub-dimension scored **7 / 10** as a result.
- **Python lint-clean is PARTIAL, not verified.** The config (`strict = true`,
  `agents/sdk/pyproject.toml:41`) and the CI gate (`--cov-fail-under=90`, `mypy --strict`) are verified,
  but executing `mypy` in the audit sandbox hit a tool-version INTERNAL ERROR, so the green *result*
  is `UNKNOWN`-by-execution.
- **Two fail-open paths (Low).** `PolicyGate` allows compilation when its active-invocation counter
  errors (`policy_gate.go:184-190`) — a fail-*open* control — in contrast to the fail-*closed* CEL
  guard (`interpreter.go:220`). Documented as an availability tradeoff but deserving an explicit
  security sign-off (handed to §5 Security).
- **Three `funlen`-suppressed parsers** (`manifest.go:126,215`; `validators/structural.go:58`) are the
  domain layer's only complexity exceptions and the top refactor targets.

---

## 4.3 Scalability — Score 5 / 10 (High confidence)

**Verdict.** This is the weakest derived technical dimension, and the reason is precise: Zynax's
*stateless compute tier* genuinely scales out, but the *stateful substrate every workflow depends on is
non-HA by design.* The architecture is *designed* to scale yet is *single-instance where it counts* —
exactly matching the project's own self-rating of 4.5/10 in `docs/product/strategy.md:449`.

### 4.3.1 What scales (the credible half)

The compute tier is genuinely stateless with clean rollout hygiene (sub-score **6 / 10** on
statelessness): workflow-compiler stores no IR (the C7 limitation is closed — see §4.4), api-gateway
and event-bus hold no per-request state, and engine-adapter delegates fan-out and durability to
Temporal (`services/workflow-compiler/internal/api/server.go:21-24,46`). `RollingUpdate maxUnavailable:0`
plus readiness/liveness/startup probes and graceful `NOT_SERVING` drain give zero-downtime horizontal
scaling of the stateless tier (`api-gateway/templates/deployment.yaml:15-19`).

Helm scale plumbing is production-grade (sub-score **7 / 10**): every chart ships an HPA (CPU + memory),
a PDB, and resource requests/limits, with a `values-production` overlay that raises replicas to 2–3,
enables autoscaling, and wires the Postgres DSN and mTLS secrets
(`helm/zynax-task-broker/values-production.yaml:4-27`;
`helm/zynax-engine-adapter/values-production.yaml:3-10`). Failure-isolation primitives are engineered,
not assumed (sub-score **6 / 10**): Temporal activity `RetryPolicy` with non-retryable types
(`temporal_workflow.go:77-86`), task-broker in-flight recovery on restart
(`task-broker/main.go:150-160`), idempotent SHA-256 manifest apply, and a NATS DLQ with 5-step
exponential backoff (`nats.go:328-366`).

### 4.3.2 The two acknowledged SPOFs (High red flag)

The data/event tier carries two single-points-of-failure, both deliberate and documented:

1. **Non-HA Postgres.** Postgres runs as a single-instance StatefulSet (`spec.replicas: 1`) with HA
   explicitly out of scope — "HA / read-replicas are out of scope (EPIC #1073 'WON'T')"
   (`helm/charts/postgres/templates/statefulset.yaml:1-5,13`). ADR-026 confirms: "No built-in HA ...
   high availability, failover, and managed backups are not provided"
   (`docs/adr/ADR-026-postgres-distribution.md:160-163,175`). Connection pools use `pgxpool` with no
   tuning (default `MaxConns`) — `task-broker/.../postgres/repository.go:32-33`. The data-layer-scale
   sub-dimension scored **4 / 10**.
2. **Single-node JetStream.** Every NATS stream is created with `Replicas: 1`
   (`services/event-bus/internal/infrastructure/nats.go:139-145`) and the NATS subchart deploys a
   single un-clustered node (`helm/charts/nats/values.yaml:32-43` — no `cluster.enabled`). Durability
   semantics are otherwise solid (durable consumers, `AckExplicit`, `MaxDeliver=5`, backoff, DLQ,
   `FileStorage`), but JetStream is durable-to-disk, not fault-tolerant or throughput-replicated
   (sub-score **5 / 10**). The "K8s production-ready / Postgres-backed horizontal scale" claim
   therefore drift-tests to **PARTIAL** — true for stateless compute, contradicted for the stateful
   substrate.

### 4.3.3 The unsafe default (second High red flag)

The umbrella **default** deploy is not horizontally safe for the two stateful services. task-broker and
agent-registry fall back to in-memory mutex-guarded maps when no DB DSN is wired
(`services/task-broker/cmd/task-broker/main.go:122-124`), and the umbrella default leaves
`db.secretName` empty (`helm/zynax-umbrella/values.yaml:46-48,55-57`). A naive `replicas: 3` umbrella
deploy therefore gives each replica a disjoint state view — the exact bug ADR-021 was written to fix
(`docs/adr/ADR-021-horizontal-scale.md:18-24`). The safe path exists in `values-production.yaml`, but
it is opt-in rather than the default.

### 4.3.4 Multi-tenancy and other gaps

- **Half-real multi-tenancy (Medium).** Namespace isolation is genuinely enforced in memory-service
  (`pgvector.go:28` — `WHERE namespace = $N`; Redis key prefix `{ns}:{key}`) and in engine-adapter's
  data context (`datacontext.go:26-34`), but is *cosmetic* on the event bus (subject = `event.Type`, all
  namespaces share one stream — `nats.go:188-189`) and *absent* in task-broker/agent-registry. There
  are no per-tenant quotas, creating noisy-neighbour risk; ADR-022's isolation + rate-limit chokepoint
  is "planned for M7" (`docs/adr/ADR-022-event-bus-architecture.md:52`). The "namespace provides
  multi-tenant isolation" claim drift-tests to **CONTRADICTED (partial)**.
- **No backpressure-aware autoscaling (Medium).** HPA scales only on CPU/memory; task-broker fan-out
  and event-bus publish backlog do not surface as CPU, so a deep async backlog will not trigger
  scale-out (`hpa.yaml:16-28`).
- **Helm/doc drift (Medium).** The event-bus and memory-service charts still carry
  `appVersion: 'placeholder'` and "Do not deploy until EPIC merges," and the umbrella defaults them
  `enabled: false`, even though both services are fully implemented and e2e-tested (images ship since
  #1089/#1090) — `helm/zynax-umbrella/values.yaml:59-65` vs `scripts/e2e/values-e2e.yaml:56-83`.

---

## 4.4 Performance — Score 6 / 10 (High confidence)

**Verdict.** Zynax has done the micro-optimisation homework but not the systems-under-concurrency
homework. The CPU-bound core is genuinely fast and the headline figures are **reproduced by executed
benchmarks (E1)**, but the async dispatch tier has an unbounded, uncancellable fan-out, the benchmark
regression gate is built-but-not-wired, and there is zero load testing.

### 4.4.1 Verified performance wins (E1)

Both committed hot-path benchmarks run clean in the audit and beat their published targets by wide
margins:

| Benchmark | Measured (E1) | Target | Margin |
|-----------|---------------|--------|--------|
| `BenchmarkIRInterpreter` (5-state/10-action IR) | ~21.2 µs/op | <100 µs | ~5× |
| `BenchmarkParseManifest` (realistic `code-review.yaml`) | ~166 µs/op | <50 ms | ~300× |

(`services/engine-adapter/internal/domain/interpreter_bench_test.go:43-62`;
`workflow-compiler/internal/domain/manifest_bench_test.go:28-39`; baseline at
`tools/bench-baseline.txt:13-20`.) Both the "interpreter < 100µs" and "manifest compile < 50ms" claims
drift-test to **VERIFIED**. The hot paths are clean Go: the IR interpreter caches CEL programs in a
`sync.Map` to avoid per-eval recompilation on Temporal replays
(`interpreter.go:192-261`), and the Temporal path is performance-disciplined — bounded activity
timeouts, exponential-backoff retry, non-retryable classification (`temporal_workflow.go:77-110`),
scoring **8 / 10** on the engine-adapter concurrency sub-dimension.

### 4.4.2 The unbounded task-broker fan-out (High red flag)

The most severe performance finding: capability-dispatch fan-out spawns **one goroutine per task with
no worker pool, semaphore, or concurrency ceiling**, on a **detached context that can never be
cancelled and carries no deadline**. `DispatchTask` does `s.bg.Add(1); go func(){ executeAsync(detach(ctx), ...) }`
(`services/task-broker/internal/domain/service.go:80-87`), and the detached context's `Done()` returns
nil, `Err()` returns nil, and `Deadline()` is zero (`service.go:316-328`). The downstream agent gRPC
stream `Recv()` loop has no broker-side deadline — only an advisory `TimeoutSeconds` field the agent
must self-honour (`agent_executor.go:47,59-79`) — and a fresh gRPC connection is dialled per dispatch
(`agent_executor.go:35`). A grep for any concurrency limiter
(`maxConcurrent|worker.?pool|semaphore|errgroup|limiter`) in the service returns empty. A burst of
dispatches or a set of hung agents therefore produces unbounded goroutine and memory growth with no
caller-side cancellation; the 512Mi pod limit can OOM-kill before the memory-based HPA averages up.
This sub-dimension scored **4 / 10**, and it is the single most likely first failure at scale —
cross-referenced from §4.3 as the leading 10x break-point candidate.

### 4.4.3 Bench gate not wired; zero load testing

The benchmark regression claim ("benchstat gate live") drift-tests to **CONTRADICTED**. A `bench-gate`
verb exists and is tested (`cmd/zynax-ci/benchgate/benchgate.go`), and a baseline is committed, but no
workflow or CI script invokes `make bench` / `bench-gate` / `benchstat` (grep over `.github/`,
`tools/`, `automation/` returns only the README and baseline file), and the gate is fail-open by
default (`cmd/zynax-ci/cmd/bench_gate.go:32-33` — "a regression WARNS but exits 0"). The committed
baseline plus tested gate create the *appearance* of a guard while a perf regression on either hot path
would land on main undetected (sub-score **3 / 10**). Separately, there is **zero load/stress testing**
anywhere in the repo (no k6/vegeta/locust/wrk/ghz), so the entire 10x/100x scaling story is inferential
— honestly admitted in `docs/architecture/2026-06-18-architecture-review.md:242`. The C7 unbounded-memory
risk, by contrast, is genuinely eliminated (sub-score **9 / 10**; see §4.5). Lesser findings: NATS
`ensureStream` runs an `AddStream` round-trip on every publish, uncached (`nats.go:161-168`), and no
service ships pprof/expvar profiling endpoints.

---

## 4.5 Technical Debt — Score 8 / 10 (High confidence) · *inverted scale: high = low debt*

**Verdict.** Zynax carries strikingly little technical debt, and what exists is honestly documented and
tracked — the *opposite* of the "large but untracked debt" risk this diligence exists to catch (that
claim drift-tests to **CONTRADICTED**). The real debt is architectural, not hygienic, and it is the
same two lines already surfaced above: the single-engine reality and the minimal authz model.

### 4.5.1 Near-zero hygiene debt

Source-level debt-marker density is near zero: a grep for `TODO|FIXME|HACK|XXX` across all source
returns **five hits**, of which three are non-debt (regex detector strings, an `mktemp` template) and
only two are genuine — both EPIC- and issue-tagged Helm placeholders
(`helm/zynax-event-bus/values.yaml:10`; `helm/zynax-memory-service/values.yaml:10`). Lint suppression
is scoped — 215 `//nolint` directives, **zero bare**, every one naming a linter (sub-score **8 / 10**).
Python suppression is equally specific — 24 error-code-scoped `# type: ignore` plus 5 `noqa`, most
clustering on the untyped SDK base class that is itself the ADR-010 drift from §4.2. Dependency-debt
tracking is best-in-class (sub-score **9 / 10**): CVE pin floors cite the CVE id inline
(`agents/sdk/pyproject.toml:26,35`), the single pip-audit suppression is dated with a re-evaluation
date and rationale (`pyproject.toml:86-90`), and renovate is grouped, scheduled, and digest-pinned
(`renovate.json:45-160`). Deferrals carry GitHub issue numbers and dates, kept behind honest xfail
tripwires (`state/current-milestone.md:26,130`; `automation/tests/test_platform_readiness.py:52`) —
debt-tracking discipline scored **9 / 10**.

### 4.5.2 C7 and C8 remediations verified

The two mandated M6 remediation targets were drift-tested directly:

- **C7 (stateless compiler) — VERIFIED in code.** The `Server` struct holds no map
  (`server.go:27-31`) and `GetCompiledWorkflow` unconditionally returns `NOT_FOUND`
  (`server.go:168-171`) — the map was removed outright, a *stronger* fix than a bounded cache. The only
  residual is the **stale proto comment** at `workflow_compiler.proto:50-53` (doc drift, not a memory
  leak) — the same stale comment §4.1 and §4.4 also flag.
- **C8 (cel-go guard) — VERIFIED.** Real `github.com/google/cel-go` pinned at `v0.28.1`
  (`go.mod:7`; `interpreter.go:17,203,250`), and `evalGuard` is fail-**closed** on every path
  (`interpreter.go:216,220-259`), replacing the old bespoke fail-open evaluator.

### 4.5.3 The architectural debt lines and the remediation bill

The two material debt items are inherited from §4.1/§4.3 and costed here: the single-engine reality
(only Temporal interprets the IR; Argo is a non-interpreting stub —
`scripts/e2e/manifests/argo-ir-interpreter.yaml:13-14`), scored **5 / 10**, and the deferred
multi-tenancy / no-RBAC model (single shared static bearer token disabled when empty —
`auth.go:13-25`; multi-namespace policy deferred to M7+ — `config.go:60-61`), scored **6 / 10**. The
packet's planning-grade remediation estimate:

| Tranche | Work | Estimate |
|---------|------|----------|
| Now (P0) | Reconcile stale proto comment + ADR-034; re-label portability claim (docs/contract only) | ≈ 0.5 eng-week |
| Next (P1) | Argo IR-interpreter parity + cross-engine test; real authz/RBAC + multi-namespace policy | ≈ 7–13 eng-weeks |
| Later (P2) | Wire fuzz/benchstat into CI; global coverage floor; justify idiomatic nolints | ≈ 1–2 eng-weeks |
| **Total** | | **≈ 8.5–15.5 eng-weeks** |

This is modest for a 7-service platform, and the debt is *concentrated* in two architecture lines
rather than spread as scattered hygiene debt — a favourable acquirer profile.

---

## 4.6 Maintainability — Score 7 / 10 (High confidence)

**Verdict.** On the *technical* axes this is a genuinely low-change-cost, highly modular,
self-documenting codebase that a new owner could comprehend and hand off in well under 90 days. Two
things hold the score at 7 rather than 8–9: the **people** axis (bus factor = 1) and a **doc-vs-reality
drift inside the onboarding contract itself**.

### 4.6.1 Low change amplification, proven by trace

The headline maintainability strength is change amplification proven by *tracing real commits*, not
asserted (sub-score **8 / 10**). Recent features each stayed inside a single module — `#1339`
(data-context scoping) touched 3 files all in one service's `internal/domain/`; even a proto contract
change (`#1249`, IR output/input bindings) rippled to only **4 files** (the `.proto`, both
auto-regenerated stubs, and a `.feature`) with **zero service code touched**. The blast radius is
bounded by additive-only field discipline (`protos/AGENTS.md:56-57`), the `buf-breaking` gate, and stub
auto-regeneration (`proto-generate.yml`).

Modularity is enforced by the *build system itself* (sub-score **8 / 10**): 11 separate Go modules
(7 services + 2 libs + 2 cmd modules) mean Go's `internal/` visibility rule makes cross-service
coupling *mechanically impossible*. Shared libs are small (~700 LoC) and confined to wiring/infra,
never reaching `internal/domain/`. The onboarding substrate is unusually strong (sub-score **8 / 10**):
21 layered `AGENTS.md` files, a 35-row Knowledge Base Index, per-service Pre-Code Checklist + uniform
directory layout, 37 ADRs, 8 pattern guides, and 62 REASONS Canvases capturing per-feature *why* and
recorded Superseded/Rejected decision history. AI-authorship risk is mitigated (sub-score **7 / 10**):
zero AI-attribution scaffolding leaked into Go source, ~10K LoC of human-ownable service-internal code,
and 1,924 lines of externalised `ai-learnings`.

### 4.6.2 The import boundary is claimed CI-enforced but is not (Medium red flag)

The low-coupling invariant that underwrites the entire maintainability case is **documented as
CI-enforced but is convention-only**. `services/AGENTS.md:22` states "Import layering enforced (CI fails
on violations)" and `:46` references "CI-enforced import analysis," but a grep for
`depguard|import-boundary|importas|gomodguard|forbidigo` in the linter config, and for `import-layering`
in the Makefile / `cmd/zynax-ci/` / workflows, returns **empty**. The "layer/import boundaries are
CI-enforced" claim drift-tests to **CONTRADICTED**. The boundary holds in practice — but a new owner
trusting the onboarding contract would over-rely on a defence that does not exist, and one careless
import silently re-couples the layers. This is the §4.1 boundary-linter finding, *compounded* by the
fact that the drift sits inside the onboarding document. (ADR-010 drift from §4.2 is also recorded here
as a Low onboarding trap.)

### 4.6.3 Bus factor = 1 (High red flag, cross-referenced)

The dominant sustainability risk is bus factor = 1: one human committer owns ~100% of non-bot history
(772 commits, no second human author in `git shortlog`), and `MAINTAINERS.md` does not exist (open
issue #494). Cross-referenced from Wave A §5.24 and scored there, it is recorded in this section's
knowledge-concentration sub-dimension at **3 / 10**. The excellent docs/canvas corpus *lowers*
hand-off cost — comprehension and hand-off in under 90 days is "likely yes" — but the day-to-day
evolution velocity and the SPDD/prompt corpus still concentrate in one person, so sustained velocity
after a departure is "uncertain." This is the social, not technical, binding constraint flagged across
the whole audit.

---

## 4.7 Innovation & IP — Score 6 / 10 (Medium confidence)

**Verdict.** Zynax's value is overwhelmingly in **disciplined, integrated execution, not in protectable
invention.** All four candidate innovations are well-built *reframings of established prior art*, and
the source packet is explicit that the durable moat is execution discipline and first-mover
positioning, not defensible IP.

### 4.7.1 Each candidate reframes self-cited prior art

| Candidate | Sub-score | Embodiment | Prior art (E7 / self-cited) |
|-----------|:---------:|------------|------------------------------|
| Engine-agnostic Workflow IR | 7 | Strong (proto + I/O-free interpreter) | Apache Beam portable runner; Argo IR; Dapr Workflow |
| Multi-engine portability (operational) | 4 | **Weak** — only Temporal interprets | — |
| Event-driven state machine (ADR-014) | 6 | Embodied (loops + HITL run) | XState, AWS Step Functions, Temporal (`ADR-014:22-25`, self-cited) |
| Adapter-first no-SDK routing (ADR-013) | 7 | **Strongest** — proven by 5 adapters | Dapr pluggable components; Envoy ext-proc; Knative |
| SPDD AI-native methodology | 6 | Partly embodied in tooling | GitHub spec-kit, AWS Kiro (concurrent) |

None of the four is mechanism-novel — the ADRs themselves cite the prior art they build on. The
patentable / durable-IP surface is thin; the protection is execution quality and integration, which is
copyable rather than a moat (Medium red flag). The innovation-vs-execution split scored **7 / 10**:
genuine engineering capital, but execution capital.

### 4.7.2 The most-defensible and the weakest embodiments

The **most-defensible candidate is adapter-first no-SDK routing** — the cleanest embodied innovation, and
the only candidate whose drift test is fully **VERIFIED**. A minimal 2-RPC `AgentService` contract
(`ExecuteCapability` stream + `GetCapabilitySchema`) *is* the extension boundary
(`protos/zynax/v1/agent.proto:31-47`), proven by five heterogeneous adapters (Go: llm/ci/git/http;
Python: langgraph) that implement the servicer directly. Tellingly, even the LangGraph *framework*
adapter imports no SDK (`grep zynax_sdk agents/adapters/langgraph` → empty), so "no SDK required" holds
even for the framework case (`ADR-013:20-23`).

The **boldest "novel" claim — engine-agnostic portability — is the weakest at embodiment** (sub-score
**4 / 10**, drift-test **PARTIAL**). It is consumed directly from §4.1: real at the interface, but only
Temporal interprets the IR; the Argo path is a non-interpreting stub
(`docs/due-diligence/2026-06-20-dd-wave-a-findings.md:238-244`; `argo_engine.go:62-98`). Even ADR-037's
lightweight in-process engine is status *Rejected*/superseded. An innovation wired structurally but not
proven functionally is precisely the "easily replicated / not embodied" red flag.

SPDD is the strongest *process*-IP candidate and is partly embodied in executable tooling — a real Go
canvas validator (`cmd/zynax-ci/validate/canvas.go:36-170`), AI-context leak controls, and a closed
Draft→applied/rejected learnings loop. But its headline canvas-before-code gate is soft (any-canvas
passes; Draft-only warns — Wave A §5.12), its productivity multiplier is **CLAIMED (E5), never measured**
(no DORA baseline), and the idea is commoditising in real time via GitHub spec-kit and AWS Kiro (E7).

### 4.7.3 The honest moat answer

Asked what a competent team could *not* rebuild in six months, the packet's answer is candid: nothing in
the technical candidates (IR + port + gRPC contract are all reproducible patterns). The non-reproducible
asset is the accumulated learnings corpus plus the integrated SPDD discipline — but that is process
capital, single-maintainer-authored (capped by the bus-factor-1 risk of §4.6), and commoditising. No
patent, trade-secret, or network-effect protection was found in the repo, and there are no external
adopters. The durable edge is execution speed and first-mover positioning in the
"control-plane-for-AI-workflows" niche — which feeds directly into the moat/defensibility risk and a
valuation discount in §8 (Risk) and §9 (Financial & Investment).

---

## 4.8 Synthesis — the verified/enforcement split, and what gates the dimension

Three patterns recur across all seven packets and define the technology verdict:

1. **The engineering substrate is the strongest verified asset.** Coverage (≥90% reproduced on all 7
   services, E1), strict lint (0 issues, E1), benchmark wins (5–300× targets, E1), near-zero
   debt-marker density, low change amplification (proven by commit trace), and best-in-class dependency
   hygiene are all **VERIFIED** by execution or config — not narrative. This is real, defensible
   engineering capital.

2. **The gaps are enforcement-shaped, not absence-shaped.** The single most repeated finding across
   §4.1, §4.4, §4.6, and §4.7 is that *capabilities exist but their enforcement or proof lags*: the
   import boundary holds but is not linted (and the onboarding doc falsely claims it is); the benchmark
   gate is built but not wired and fail-open; the engine matrix exists but the Argo leg is non-required;
   the coverage gate is real but re-checks only changed services. None of these is a fabrication — each
   is a partial, soft, or opt-in version of the headline claim.

3. **Two material findings gate the "category" thesis.** The portability moat — "runs on Temporal *or*
   Argo without a rewrite" — is **PARTIAL**: real at the IR/contract/port boundary, contradicted at
   execution because only Temporal interprets the IR. And the runtime substrate (Postgres + JetStream)
   is **single-instance, non-HA by design**, with an umbrella default that silently reverts the
   stateful services to in-memory maps. The §4.7 conclusion follows: the durable moat is *execution
   discipline and first-mover positioning, not protectable IP.*

The consistent, evidence-grounded P0 recommendation across the packets is the same: **re-label the
portability claim precisely** ("engine-neutral IR with Temporal as the reference interpreter; Argo
submission validated, execution parity in progress") until a second engine genuinely interprets the IR
— and re-label the scalability claim to name the two acknowledged SPOFs. These are docs/contract-only
fixes (≈ 0.5 eng-week) that convert the highest-trust risk in the dimension — delivery-vs-narrative
drift — into an honest, defensible position.

# 5. Security & Supply Chain

> **Scope.** This section synthesises the two security-bearing ground-truth audits — the Wave A
> Security agent (5.2) and the Wave B OpenSSF Readiness agent (5.22) — into a single posture
> assessment across three lenses: 5.1 the shipped security posture (authentication, transport,
> supply chain, container hardening, CI gates); 5.2 the OpenSSF/Scorecard view (least-privilege
> tokens, SHA-pinned actions, the missing LICENSE file, the decorative Scorecard badge); and 5.3
> the threat-model and attack-surface summary. The full per-agent packets are reproduced in
> **Appendix A** (the wave findings documents
> `docs/due-diligence/2026-06-20-dd-wave-a-findings.md` and
> `docs/due-diligence/2026-06-20-dd-wave-b-findings.md`).
>
> **Evidence standard (framework Part 2 §2.4).** Every factual claim below carries an evidence
> citation — a `repo-path:line`, an executed command and its output (E1), or an external URL
> (E7) — or is explicitly marked `UNKNOWN`. `VERIFIED` facts rest on executed proof, code,
> configuration, or contract (E1–E4). Statements resting only on documentation, ADRs, or
> marketing (E5–E6) are labelled `CLAIMED` and are kept strictly separate from verified facts.
> No new claims are introduced here; this is a re-organisation and prose synthesis of what the
> two source audits already established.

---

## 5.0 Headline assessment

| Dimension | Score | Confidence | Source agent |
|-----------|:-----:|:----------:|--------------|
| Security posture (D5) | **7 / 10** | Medium | Wave A 5.2 |
| OpenSSF Readiness (D5/D7) | **6 / 10** | High | Wave B 5.22 |

Zynax presents a **two-speed security profile**: a genuinely strong, CNCF-credible
*supply-chain and container-hardening* core, wrapped in *enforcement and presentation gaps*
that an external audit would surface first. The single recurring pattern — already identified
as the cross-cutting theme of the entire diligence — is that **the controls exist but are
opt-in, soft, or decorative rather than hard-enforced**. mTLS is implemented but fails open;
the supply-chain trifecta is wired but its registry artifacts are unverified; the Scorecard
badge is displayed but no score is published; the project declares Apache-2.0 in every file
header but ships no `LICENSE` file.

The net verdict is **production-credible for an MVP control plane, but not yet
enterprise-governance-grade or audit-clean**. None of the gaps is a code-level zero-day; all
are configuration, enforcement, or hygiene gaps that are individually low-effort to remediate.
The highest-severity finding — mTLS fail-open with two production overlays omitting TLS — is a
**Critical-class red flag** that is promoted to the Executive Summary and the risk register.

---

## 5.1 Security posture

**Verdict (Wave A 5.2 — Score 7, Medium confidence).** Zynax has a genuinely strong,
CNCF-credible supply-chain and container-hardening posture: cosign keyless signing, syft SPDX
SBOM, SLSA L2 provenance attestation, digest pinning with a drift gate, distroless-nonroot
images, a full Helm `securityContext`, and layered blocking CVE/SAST/secret gates. Its gateway
authentication is timing-safe and fails closed. The **material weakness is transport
enforcement**: mTLS is correctly implemented but is opt-in by Helm overlay and fails open in
code, which makes the documented claim of "mTLS enforced on all inter-service gRPC" an
overstatement of the shipped default.

### 5.1.1 Sub-dimension scorecard

| Sub-dimension | Score | Confidence | Key evidence |
|---------------|:-----:|:----------:|--------------|
| Inter-service transport / mTLS | 6 | High | `tlscreds.go:20` (insecure fallback); `tlscreds.go:47` (`RequireAndVerifyClientCert`); `helm/zynax-api-gateway/values-production.yaml` (no `tlsSecretName`) |
| api-gateway authN/authZ | 7 | High | `auth.go:20` (`ConstantTimeCompare`); `main.go:45` (fail-closed startup) |
| Supply chain: SBOM + sign + provenance | 8 | Medium | `release.yml:201` cosign; `:527` syft SBOM; `:510` SLSA L2 |
| Image digest pinning + drift gate | 9 | High | `images/images.yaml`; `pr-checks.yml:336` |
| Container hardening | 9 | High | `Dockerfile.service:49`; `_helpers.tpl:76`/`:89` |
| CI security gates (CVE/SAST/secrets) | 8 | High | `ci.yml:689`/`:870`; `pr-checks.yml:60`/`:366` |
| Network attack surface (K8s) | 7 | Medium | `networkpolicy.yaml:12` (Ingress+Egress); `:15` (no `from` selector) |
| Secrets hygiene | 8 | Medium | `ci.yml:286`; `pr-checks.yml:385` |

### 5.1.2 Authentication and authorisation

API-gateway authentication is the strongest authentication-layer signal in the audit. The
bearer-token check uses `crypto/subtle.ConstantTimeCompare`, making it timing-safe
(`services/api-gateway/internal/api/auth.go:20`), and it **fails closed**: the gateway refuses
to start with an empty key unless an operator explicitly sets `ZYNAX_GW_DEV_INSECURE=1`
(`services/api-gateway/cmd/api-gateway/main.go:45`), at which point a warning is emitted
(`main.go:68`). This is a defensible, correctly-built control for the gateway boundary.

The limitation is the **authorisation model**, not the authentication mechanism. The gateway
uses a *single shared static bearer key* with no per-caller identity, scopes, RBAC, or SSO
(`services/api-gateway/internal/api/auth.go:14`). This is adequate for a control-plane MVP but
**insufficient for an enterprise governance buyer**, who will expect per-tenant identity and
role-scoped access. The Wave A audit rates this a Medium-severity red flag; it is the single
largest authorisation gap blocking the enterprise-governance persona.

### 5.1.3 Transport security — the mTLS fail-open finding (drift C2)

This is the highest-severity finding in the security posture. mTLS is **correctly implemented**:
`services/api-gateway/internal/infrastructure/tlscreds.go:47` configures
`tls.RequireAndVerifyClientCert` — genuine mutual TLS — and the certificate plumbing is wired
through cert-manager. The problem is **enforcement**, on three compounding fronts:

1. **Code fails open, not closed.** When any of the certificate / key / CA paths is empty,
   `tlsCreds()` returns `insecure.NewCredentials()`
   (`services/api-gateway/internal/infrastructure/tlscreds.go:20`), and the same pattern is
   present across the other six services' infrastructure layers. A missing credential silently
   degrades to plaintext gRPC rather than refusing to start.
2. **The chart default is insecure.** `helm/zynax-engine-adapter/values.yaml:51` ships
   `tlsSecretName: ""` by default — secure transport is something an operator must opt into per
   overlay, not the baseline.
3. **Two production overlays omit TLS.** Critically, the api-gateway and workflow-compiler
   *production* overlays do **not** set `tlsSecretName`
   (`helm/zynax-api-gateway/values-production.yaml` carries 22 lines, none of them
   `tlsSecretName`). An operator who applies the production overlay believing it hardens the
   deployment still runs the gateway's upstream gRPC in plaintext.

**Drift verdict — C2 "mTLS enforced between all services": PARTIAL.** Real mTLS is available
(`tlscreds.go:47`) but the code fails open (`tlscreds.go:20`), the chart default is insecure
(`engine-adapter/values.yaml:51`), and the production overlay omits TLS. This directly
contradicts the assertions in `docs/adr/ADR-020-zero-trust-auth.md:50` and `SECURITY.md:89`
that "mTLS [is] enforced on all inter-service gRPC". The ADR itself concedes that Compose stays
insecure (`ADR-020:51`), but the production-overlay gap goes further than the documented
exception. There is **no startup guard that refuses to run without TLS in a production
namespace**, unlike the bearer-key guard at `main.go:45` — a guard that would convert this from
fail-open to fail-closed.

### 5.1.4 Supply-chain trifecta — cosign + SBOM + SLSA (drift C3, C4)

The supply-chain controls are the project's strongest CNCF-credibility asset, all wired into
the release pipeline:

- **cosign keyless signing (OIDC)** — `release.yml:201` (per-digest on the merge/promote path)
  and `release.yml:504` (on the version-tag path).
- **syft SPDX SBOM per service digest** — `release.yml:527`, with the resulting `sbom-*`
  artifacts collected into the GitHub Release (`release.yml:577`).
- **SLSA L2 provenance** — `actions/attest-build-provenance` at `release.yml:510` (ADR-025).

**Drift verdict — C3 "SBOM per release": VERIFIED.** syft generates an SPDX SBOM from each
service version digest (`release.yml:527`) and attaches it to the GitHub Release
(`release.yml:577`); the early `SECURITY.md:42` claim is corroborated by this CI evidence.

**Drift verdict — C4 "cosign-signed images": PARTIAL.** The signing is config-VERIFIED
(`release.yml:201`, `:504`), and Wave B independently confirmed that the *release git tags* are
ED25519-signed and verify locally (`git tag -v v0.5.0` → "Good git signature … ED25519"). But
the **existence of the cosign signature / SLSA attestation on the published GHCR images is
UNKNOWN**: the audit could not run `cosign verify` or `gh attestation verify` (cosign not
installed in the audit environment; no registry network access). Signing is configured;
artifact existence in the registry is unverified. Both Wave A and Wave B flag this as the same
inherited unknown — it is the one supply-chain claim that requires registry access to close,
and is the most consequential confirmatory-diligence item in this section.

### 5.1.5 Image pinning and container hardening

These are the two highest-scoring sub-dimensions (9/10 each), both VERIFIED:

- **Digest pinning with a single source of truth.** `images/images.yaml` carries a `sha256`
  digest pin for every base and service entry; a pre-merge drift gate
  (`pr-checks.yml:336` — `images check`) blocks divergence; Dockerfiles consume the pinned
  digests via banner-marked regions (`Dockerfile.service:25`, `:49`).
- **Container hardening shipped end-to-end.** Images are `gcr.io/distroless/static:nonroot`
  (`Dockerfile.service:49`) built as static, stripped binaries (`CGO_ENABLED=0`, `-trimpath`,
  `-ldflags -s -w` at `Dockerfile.service:43`). The Helm library enforces `runAsNonRoot` /
  `runAsUser 1001`, `RuntimeDefault` seccomp, `readOnlyRootFilesystem`, `drop ALL` capabilities,
  and no privilege escalation (`_helpers.tpl:76`, `:89`), and these contexts are actually wired
  into the Deployments (`deployment.yaml:31`) — not merely defined in helpers.

### 5.1.6 CI security gates and secrets hygiene

The CI security gate stack is layered and largely blocking (8/10, High confidence):
`govulncheck` on changed Go services (`ci.yml:689`), `bandit` + `pip-audit` across the SDK and
agents (`ci.yml:716`), Trivy failing on CRITICAL/HIGH on the staging image (`ci.yml:870`),
`dependency-review` blocking new HIGH CVEs (`pr-checks.yml:60`), `gitleaks` secret scanning in
both `ci.yml:286` and over the PR commit range (`pr-checks.yml:366`/`:385`), and CodeQL SARIF
upload. Configuration is sourced from `envconfig` rather than embedded secrets
(`main.go:56`). The two standing exceptions are documented and time-boxed: one suppressed
Python CVE (`pip-audit --ignore-vuln PYSEC-2026-196` at `ci.yml:730`) and one Trivy suppression
(`.trivyignore` DS002, accepted-until 2026-11-01). Both are auditable, but they are standing
exceptions to otherwise-blocking gates (Low-severity flag).

### 5.1.7 Security posture — red and green flags

**Red flags (severity-ordered):**

1. **High — mTLS fails open.** Services default to `insecure.NewCredentials()`
   (`tlscreds.go:20`); the api-gateway and workflow-compiler production overlays omit
   `tlsSecretName`; `ADR-020:50` / `SECURITY.md:89` overstate enforcement.
2. **Medium — NetworkPolicy ingress is port-scoped, not source-scoped.** No `from` selector
   means any pod may dial the service ports (`networkpolicy.yaml:15`) — not zero-trust
   default-deny.
3. **Medium — single shared static bearer key**, no RBAC / scopes / SSO (`auth.go:14`).
4. **Low — two standing CVE suppressions** (`pip-audit` PYSEC-2026-196; Trivy DS002) — both
   documented and time-boxed (`ci.yml:730`, `.trivyignore`).

**Green flags:**

- Timing-safe, fail-closed gateway authentication (`auth.go:20`, `main.go:45`).
- Full supply-chain trifecta — cosign + SBOM + SLSA L2 (`release.yml:201`, `:527`, `:510`).
- Strong container hardening wired into actual Deployments (`Dockerfile.service:49`,
  `_helpers.tpl:76`/`:89`, `deployment.yaml:31`).
- Digest-pinning source of truth + pre-merge drift gate + auditable exceptions
  (`images/images.yaml`, `pr-checks.yml:336`, `.trivyignore`).
- Layered blocking CVE / SAST / secret gates (`ci.yml:689`/`:870`, `pr-checks.yml:60`/`:366`).

---

## 5.2 OpenSSF / Scorecard readiness

**Verdict (Wave B 5.22 — Score 6, High confidence).** Zynax has a genuinely strong
supply-chain-hygiene core that builds directly on the Wave A supply-chain ground truth, but the
headline OpenSSF result is dragged down by **two trivially-fixable FAILs and one governance
FAIL**. The estimated realistic Scorecard grade is **≈ 6.0–6.8 / 10** — derived per-check, not
from an official run, because no published Scorecard record exists.

### 5.2.1 Per-check Scorecard view

| Scorecard check | Score | Result | Key evidence |
|-----------------|:-----:|--------|--------------|
| Token-Permissions | 9 | VERIFIED | 21/21 workflows declare top-level `permissions`; `contents: read` default (`ci.yml:41`) |
| Pinned-Dependencies | 9 | VERIFIED | Zero tag-pinned actions (grep empty); images digest-pinned (`images/images.yaml`); renovate `pinDigests:true` |
| Dangerous-Workflow | 9 | VERIFIED | No `pull_request_target`; untrusted input via `env:` + quoted var (`pr-checks.yml:218`) |
| Dependency-Update-Tool | 9 | VERIFIED | `renovate.json:3`; 7 `renovate[bot]` commits confirm it runs live |
| Maintained | 9 | VERIFIED | 824 commits in 90 days; HEAD dated current |
| Security-Policy | 9 | VERIFIED | `SECURITY.md` present |
| Branch-Protection | 8 | VERIFIED (1 gap) | Ruleset 17547241: linear history, signatures, squash-only, 12 strict checks; gap: `required_approving_review_count=0` |
| Signed-Releases | 8 | VERIFIED (git) / PARTIAL (cosign artifact UNKNOWN) | `git tag -v v0.5.0` → Good ED25519 signature; cosign/SLSA artifact existence UNKNOWN |
| Vulnerabilities | 8 | VERIFIED | govulncheck + pip-audit + Trivy + dependency-review (`pr-checks.yml:70`, `ci.yml:689`/`:870`) |
| CI-Tests | 9 | VERIFIED | `test-unit` + `test-integration` are required checks; testcontainers integration |
| SAST | 5 | PARTIAL | No CodeQL analyze/init — `codeql-action` used **only** to upload Trivy SARIF (`ci.yml:891`) |
| Fuzzing | 4 | PARTIAL | Native go-fuzz harnesses exist (`FuzzEvalGuard`, `FuzzParseManifest`) but no CI/scheduled fuzz job, no OSS-Fuzz |
| Code-Review | 2 | FAIL | `required_approving_review_count=0`; last 15 merged PRs had 0 reviews |
| License | 1 | FAIL | No top-level LICENSE file; README badge/footer link a missing target |
| Scorecard badge drift | 2 | FAIL | Badge displayed but no `scorecard.yml`; live API returns HTTP 404 ("no data") |

### 5.2.2 The strong backbone — SHA-pinned actions and least-privilege tokens

The supply-chain-hygiene backbone is best-in-class and VERIFIED by executed command and config:

- **Pinned-Dependencies (9).** All external GitHub Actions are pinned by 40-character SHA —
  a repo-wide grep for `@main|@master|@vN|@latest` across all 21 workflows returns empty.
  Examples: `actions/dependency-review-action@a1d282b…` (`pr-checks.yml:70`) and
  `github/codeql-action/upload-sarif@03e4368…` (`ci.yml:891`). Container images are
  digest-pinned via `images.yaml` with the pre-merge drift gate, and renovate's
  `pinDigests:true` (`renovate.json:135`) keeps the SHAs current.
- **Token-Permissions (9).** Every one of the 21 workflows declares a top-level `permissions:`
  block defaulting to least-privilege `contents: read` (`ci.yml:41`, `pr-checks.yml:35`,
  `release.yml:55`). Write scopes are narrow and job-scoped — `security-events: write` only on
  the SARIF-upload job (`ci.yml:802`), `packages: write` only on the image-cleanup workflow
  (`pr-image-cleanup.yml:42`). This is the single highest-confidence Scorecard win.
- **Dangerous-Workflow (9).** No `pull_request_target` anywhere; the sole `workflow_run`
  (`release.yml`) is the trusted post-merge retag; untrusted `pull_request.title` flows through
  `env:` and is referenced as a quoted shell variable (`pr-checks.yml:218`, `:220`) — the
  injection-safe pattern, not direct interpolation.

### 5.2.3 The decorative badge and the missing LICENSE — two presentation FAILs (drift)

Two simple checks fail at the easiest bar, and both are Part 1 §1.10-class
displayed-vs-actual drift:

- **Drift verdict — "README Scorecard badge reflects a real grade": CONTRADICTED.** The README
  displays an OpenSSF Scorecard badge (`README.md:16`) but there is no `scorecard.yml` workflow
  to produce a result (`ls .github/workflows/scorecard.yml` → absent), and the live Scorecard
  API returns **HTTP 404 with an empty body** (`curl …/projects/github.com/zynax-io/zynax` →
  404). An evaluator clicking the badge gets "no data," not a grade — the badge is decorative,
  undermining the project's own CNCF-credibility narrative.
- **Drift verdict — "Apache-2.0 LICENSE file present": CONTRADICTED.** There is **no top-level
  LICENSE file** in the repo or in git history (`ls LICENSE* COPYING*` → absent;
  `git ls-files | grep -i license` → only `docs/adr/ADR-005-apache-license.md`). Apache-2.0 is
  declared only in ADR-005 and 347 SPDX headers. Yet the README badge (`README.md:13`) and
  footer (`README.md:527`) both **link to a non-existent `LICENSE` file**. The Scorecard License
  check fails, GitHub's repo-license auto-detection fails, and the README links are broken. For
  a CNCF-aspiring Apache-2.0 project this is both an embarrassing FAIL and a trivially-fixable
  one. (Cross-confirmed in Wave C.)

### 5.2.4 Code-Review, SAST depth, and fuzzing

- **Code-Review ≈ 0 (FAIL).** The branch-protection ruleset sets
  `required_approving_review_count=0` and `require_code_owner_review=false`; the last 15 merged
  PRs (#1455–#1471) all show 0 reviews and an empty review decision. Every change self-merges on
  green CI with no second human. Scorecard's Code-Review check scores near-zero. This is both a
  Scorecard depressor and a governance / bus-factor red flag that reinforces the Wave A
  Repo-Health bus-factor-of-1 finding.
- **SAST is shallow (PARTIAL).** There is no CodeQL `analyze`/`init` run — the `codeql-action`
  is used *only* to upload Trivy SARIF (`ci.yml:891`; grep for `codeql-action/analyze|init`
  empty). Scorecard SAST is capped without a real CodeQL/SonarCloud analysis step; current
  coverage is CVE/lint-grade (govulncheck, bandit, Trivy), not code-flow SAST.
- **Fuzzing is harness-only (PARTIAL).** Native go-fuzz functions exist (`FuzzEvalGuard`,
  `FuzzParseManifest`) with a manual `Makefile` target, but no CI/scheduled fuzz job and no
  OSS-Fuzz integration. Scorecard credits presence, not continuous coverage.

### 5.2.5 OpenSSF — red and green flags

**Red flags (severity-ordered):** (1) **High** — decorative Scorecard badge, no published score
(`README.md:16`; API 404); (2) **High** — License check FAILS, broken README links
(`README.md:13`/`:527`; `ls LICENSE*` absent); (3) **Medium** — Code-Review ≈ 0; (4) **Medium**
— SAST SARIF-only, no CodeQL analyze; (5) **Low** — fuzzing harness-only; (6) **Low** —
`SECURITY.md` multi-arch claim contradicted by amd64-only service images (cross-ref Wave A 5.9).

**Green flags:** Pinned-Dependencies best-in-class; Token-Permissions exemplary (21/21);
Dangerous-Workflow clean; ED25519-signed release tags plus the cosign/SLSA/SBOM config; and a
six-check VERIFIED backbone (Branch-Protection, CI-Tests, Vulnerabilities, Maintained,
Security-Policy, Dependency-Update-Tool).

---

## 5.3 Threat model & attack-surface summary

The two audits together map a coherent attack surface for a Kubernetes-resident,
gRPC-meshed control plane. The dominant theme is that **the platform's defences are real but
configuration-conditional**: the same control is secure under the right overlay and open under
the default.

### 5.3.1 External boundary — the api-gateway

The api-gateway is the single externally-reachable ingress and is the **best-defended**
boundary: timing-safe bearer authentication that fails closed (`auth.go:20`, `main.go:45`),
running on a hardened distroless-nonroot image with a full `securityContext`. The residual
external-boundary risk is authorisation granularity, not authentication — a single shared key
gives no per-caller attribution, so a leaked token grants full control-plane access with no
scoping (`auth.go:14`).

### 5.3.2 Internal mesh — lateral movement (the primary surface)

The most material attack surface is **east-west, inside the namespace**, and it arises from the
intersection of two findings:

- **Plaintext transport under the default / production overlay.** Because `tlsCreds()` returns
  `insecure.NewCredentials()` when cert paths are empty (`tlscreds.go:20`) and the default chart
  (`engine-adapter/values.yaml:51`) plus the api-gateway / workflow-compiler production overlays
  omit `tlsSecretName`, inter-service gRPC can run unencrypted in production.
- **Port-scoped, not source-scoped, NetworkPolicy.** Every service ships a NetworkPolicy with
  both Ingress and Egress policy types (`networkpolicy.yaml:12`) and egress is restricted to DNS
  plus named upstream gRPC ports (`:21`). But ingress restricts *ports only* — there is no
  `from` source selector (`networkpolicy.yaml:15`) — so any pod in the cluster may dial those
  ports. This is not a true zero-trust default-deny ingress.

Combined, a co-resident workload could connect to, observe, or inject inter-service gRPC
traffic with no certificate challenge and no source restriction. This is a **configuration-
dependent exposure, not a code-level zero-day**: it manifests only when TLS is left at the
fail-open default. The remediations are the Wave A P0/P1 items — a fail-closed transport flag
(mirroring the bearer-key guard at `main.go:45`), completing the two production overlays, and
adding `from` selectors for default-deny ingress.

### 5.3.3 Supply-chain surface

The supply-chain surface is **well-defended on every front the audit could verify**: digest
pinning with a drift gate closes the mutable-tag injection vector; SHA-pinned actions close the
action-supply-chain vector; least-privilege tokens limit blast radius; cosign + SBOM + SLSA
provide artifact provenance. The **one unverified link** is whether the cosign signatures and
SLSA attestations actually exist on the published GHCR images (C4 PARTIAL — `cosign verify` was
unrunnable). Until that is confirmed with registry access, the artifact-verification leg of the
supply chain is config-VERIFIED but artifact-UNKNOWN.

### 5.3.4 Governance / process surface

The Code-Review-≈-0 finding (§5.2.4) and the single-maintainer bus factor expose a **process
attack surface**: with zero required approvals and self-merge on green CI, a single compromised
maintainer credential — or a single mistaken commit — reaches `main` with no second-human gate.
The strong branch protection (signatures, linear history, 12 strict status checks) mitigates
*accidental* and *unsigned* changes but not the *single-actor* concentration. This is a
governance liability that an M8 CNCF external-audit gate will surface.

### 5.3.5 Private-annex note

The Wave A audit produced a private (do-not-publish) annex elaborating the fail-open transport
*exploitation path* and the unverified-signature-chain caveat. Per the diligence framework's
disclosure policy, any unfixed-vulnerability exploit detail belongs in a private annex —
however, **no public-unsafe exploit was identified**: the transport exposure is a known,
in-repo-documented configuration condition (ADR-020 concedes Compose stays insecure), and the
signature-chain item is an unverified-claim, not an exploit. The detail therefore mirrors repo
policy and adds no non-public risk; it is retained in the private annex only for the
orchestrator's confirmatory checklist (run `cosign verify` / `gh attestation verify` against a
published GHCR image; confirm the two production overlays).

---

## 5.4 Drift register (Security & Supply Chain)

| Claim | Register | Verdict | Evidence |
|-------|:--------:|:-------:|----------|
| mTLS enforced on all inter-service gRPC | C2 | **PARTIAL** | Real mTLS (`tlscreds.go:47`) but fails open (`tlscreds.go:20`); 2 prod overlays omit `tlsSecretName` |
| SBOM generated per release | C3 | **VERIFIED** | syft SPDX per-service digest (`release.yml:527`), attached to Release (`:577`) |
| cosign-signed images | C4 | **PARTIAL** | Signing + SLSA wired (`release.yml:201`,`:510`); GHCR signature existence UNKNOWN (no registry access) |
| OpenSSF Scorecard badge reflects a real score | README §badge | **CONTRADICTED** | No `scorecard.yml`; live API HTTP 404 → "no data" (`README.md:16`) |
| Apache-2.0 LICENSE file present | §1.9 | **CONTRADICTED** | No top-level LICENSE in repo or git history; broken README links (`README.md:13`/`:527`) |
| GitHub Actions SHA-pinned + tokens least-privilege | §5.22 | **VERIFIED** | Zero tag-pinned actions (grep empty); 21/21 workflows least-privilege (`ci.yml:41`) |

---

## 5.5 Open questions, unknowns & recommendations

### 5.5.1 Unknowns ledger

- **GHCR signature / attestation existence** — cosign was unavailable and there was no registry
  network access, so `cosign verify` / `gh attestation verify` could not be run. Signing is
  config-VERIFIED; artifact existence is UNKNOWN. If signatures are in fact absent, C4 downgrades
  from PARTIAL to CONTRADICTED. **(Top confirmatory-diligence item for this section.)**
- **NetworkPolicy runtime enforcement** depends on a CNI that honours it (Calico/Cilium) — not
  verifiable from the repo.
- **Exact live Scorecard numeric grade** — no published record (API 404) and no `scorecard.yml`
  to generate one; the ≈ 6.0–6.8 estimate is per-check, not an official run.
- **Production DB / overlay intent** — whether the two production overlays omit `tlsSecretName`
  intentionally (e.g., a TLS-terminating ingress in front of the gateway) or as a gap is not
  resolvable from the repo.
- Secrets hygiene rests on the gitleaks gate rather than an exhaustive per-file scan (Medium
  confidence).

### 5.5.2 Prioritised recommendations

**P0 (security-posture, Wave A 5.2)**

- Make insecure transport **fail closed in production**: have services refuse to start without
  TLS credentials unless an explicit `ZYNAX_DEV_INSECURE=1` flag is set (mirror the gateway's
  API-key guard at `main.go:45`), and add `tlsSecretName` to the api-gateway and
  workflow-compiler production overlays. *Rationale: today a misconfigured prod deploy silently
  runs plaintext gRPC while "mTLS enforced" is documented — fail-open plus missing overlays is
  the single highest-severity gap.*

**P0 (OpenSSF, Wave B 5.22)**

- Add a top-level **LICENSE file** with the full Apache-2.0 text — flips Scorecard License 0→10,
  repairs the broken README links, and enables GitHub license auto-detection.
- Add a **`scorecard.yml`** workflow (`ossf/scorecard-action`) so the README badge reflects a
  real published score — or remove the badge until a score exists.

**P1**

- Add source `from` selectors (default-deny ingress) to the NetworkPolicies so only named peer
  services may reach each port (Wave A 5.2).
- Run `cosign verify` / `gh attestation verify` against a published GHCR image in CI to convert
  C4 from PARTIAL to VERIFIED, and reconcile `SECURITY.md` / `ADR-020` mTLS wording to "mTLS
  supported via cert-manager; enforced when the production overlay is applied" (Wave A 5.2).
- Raise Code-Review off the floor: require ≥ 1 approving review (or a CODEOWNERS review) on
  `main`, and add a real CodeQL `analyze` step for code-flow SAST (Wave B 5.22).

**P2**

- Introduce per-caller identity / scoped tokens (or OIDC) at the api-gateway, and time-box the
  two standing CVE suppressions to the quarterly review (Wave A 5.2).
- Add a scheduled CI fuzz job over the existing `FuzzEvalGuard` / `FuzzParseManifest` harnesses,
  and register an OpenSSF Best-Practices badge via self-certification (Wave B 5.22).

---

## 5.6 Cross-references

- **§4 Technology & Architecture** — the mTLS fail-open default intersects the three-layer /
  zero-trust posture; the supply-chain credibility supports the architecture's CNCF-fit claim.
- **§6 Quality & Delivery** — Code-Review ≈ 0, no CI fuzz job, and SARIF-only SAST connect to the
  testing-rigor and DevOps findings; the standing CVE suppressions touch the CI-gate analysis.
- **§7 Open Source, Governance & CNCF** — the OpenSSF posture *feeds* CNCF readiness: a
  technically strong supply chain undercut by the License/badge FAILs and Code-Review ≈ 0 is
  exactly the hygiene a CNCF Sandbox review and the M8 external security audit will flag first;
  the single-maintainer bus factor reinforces the Repo-Health governance concern.
- **§8 Risk** — the mTLS fail-open finding (C2) is promoted to the risk register as the
  highest-severity security item; the unverified GHCR signature chain (C4) is the top
  confirmatory-diligence item.
# 6. Quality & Delivery

> **Scope.** This section assesses the four delivery-quality dimensions of the Zynax
> platform: test rigor (6.1), the CI/CD and release pipeline (6.2), documentation
> (6.3), and the AI-native development methodology that produces most of the code
> (6.4). All findings derive from the Wave A ground-truth audit of `main` @ `e3135a6`
> (audit date 2026-06-20); the full per-agent packets are preserved in
> Appendix A and in `docs/due-diligence/2026-06-20-dd-wave-a-findings.md`.
>
> **Evidence discipline (Part 2 §2.4).** Every claim below carries a `path:line`
> citation, an executed-command result (labelled **E1**), or is marked **UNKNOWN**.
> Facts grounded in code/CI/contract (E1–E4) are labelled **VERIFIED**; statements
> resting only on docs/marketing (E5–E6) or roadmap are labelled **CLAIMED** and are
> never promoted to VERIFIED. The single recurring theme of this section — established
> across all four dimensions — is that **Zynax's quality assets are real and largely
> verified by execution, while the residual gaps are enforcement-shaped (opt-in / soft /
> partial) rather than absence-shaped.**

## 6.0 Section summary

| Dimension | Agent | Score | Confidence | Headline |
|---|---|:---:|:---:|---|
| 6.1 Testing | 5.7 (D9) | **8 / 10** | High | 306 BDD scenarios cover all 33 RPCs; ≥90% domain coverage proven by execution against an exit-1 gate. Gap: the gate re-checks only *changed* services. |
| 6.2 DevOps / CI-CD | 5.9 (D8) | **8 / 10** | High | 21 workflows; build-once / promote-by-retag verified (scan == deploy). Gap: deployed service images are amd64-only. |
| 6.3 Documentation | 5.10 (D11) | **7 / 10** | High | Broad, accurate (7/8 sampled claims VERIFIED); the §1.10 doc-vs-tooling lag is resolved. Gap: the README self-contradicts its own milestone status. |
| 6.4 AI-native dev / SPDD | 5.12 (D10) | **7 / 10** | High | Coherent methodology run at scale (37 Implemented canvases) with a closed learnings loop. Gap: canvas-before-code is a *soft* gate. |

**Un-weighted mean of this section ≈ 7.5 / 10.** Treat as directional; final
confidence-weighting and contradiction resolution are the Wave D orchestrator's
(Part 4). No **Critical** red flag arises in this section; the most material delivery
finding promoted to the executive summary and risk register is the **amd64-only
service-image** constraint (6.2), with the **changed-services-only coverage re-gate**
(6.1) and the **soft canvas gate** (6.4) as Medium enforcement gaps.

---

## 6.1 Testing — Score: 8 / 10 (High)

**Mission.** Assess the test pyramid (BDD, unit, integration, E2E, fuzz, benchmark,
property, chaos) and — critically — whether the quality gates are *real and blocking*
rather than coverage theater.

**Verdict.** Zynax has a genuinely strong, multi-tier test discipline that survives
contact with executed proof. This is one of the project's two strongest verified assets
(alongside the supply-chain pipeline of 6.2). The score is held at a strong 8 rather
than a 9 by four honest, real gaps — all enforcement-shaped, none indicating absence of
the capability.

### 6.1.1 BDD contract coverage — VERIFIED

The headline "140+ BDD scenarios / every RPC covered" claim (`ROADMAP.md:78`,
`README.md:412`) is not just met but materially exceeded. The audit counted **306
scenarios across 18 `.feature` files**, and a per-RPC grep loop confirmed that **all 33
gRPC RPCs across the 7 services are referenced in at least one feature file**
(`CompileWorkflow` through `WatchWorkflow`). The full contract suite passes by
execution:

> **E1** — `cd protos/tests && GOWORK=off go test ./...` → all 10 suites `ok`.

This earns the highest sub-score (9, High). The scenarios run over in-process bufconn
gRPC (`protos/tests/testserver/server.go:19`), realising ADR-016's BDD-before-code
discipline at the gRPC boundary.

One reader-beware caveat keeps this from being unqualified: the proto-contract BDD tests
validate against **in-test stubs** (e.g. `brokerStub` at
`protos/tests/task_broker_service/steps_test.go:245`), not the production service-domain
code. The 306-scenario count therefore attests *contract semantics*, not that the
shipped service implementations satisfy them — that is the unit and integration tiers'
job. The headline number is real but should not be over-read.

### 6.1.2 Domain coverage gate — VERIFIED, but per-changed-service

The "≥90% coverage on `internal/domain`, gate blocks" claim (`AGENTS.md:108,123`;
ADR-016) is **VERIFIED twice over** — the gate exists *and* blocks, and the threshold is
currently met by every service:

- The gate is genuinely fail-closed: `_test-go.yml:155-181` sets `failed=true` then
  `$failed && exit 1`, with the threshold sourced from a single file
  (`tools/coverage-gates.env:4`, `COVERAGE_DOMAIN_GATE=90`).
- All seven service domains were measured **≥90% at HEAD by executed proof** (E1,
  `GOWORK=off go test ./internal/domain/...` per service):

| Service domain | Measured coverage |
|---|:---:|
| task-broker | 92.1% |
| workflow-compiler | 97.5 / 92.7 / 95.9% |
| engine-adapter | 92.8% |
| agent-registry | 94.0% |
| memory-service | 100% |
| event-bus | 100% |
| api-gateway | 96.7% |

The gating is layered and consistent across the codebase, all wired to the same
source-of-truth file: adapter ≥85% (`_test-go.yml:265-280`), `cmd/zynax` ≥79%
(`:307-322`), `cmd/zynax-ci` ≥80% (`:327-344`), and Python ≥90%
(`_test-python.yml:65,78`, `--cov-fail-under=90`) — all `exit 1` on breach.

**The one material caveat (the changed-services-only re-gate gap).** The domain gate
measures and enforces coverage **only for services whose `<SVC>_CHANGED==true`** on a
given PR (`_test-go.yml:132-134`: if not changed → "SKIPPED — not changed"; `:160`: the
gate runs only `if [ -f domain-coverage.out ]`, which is produced only for changed
services). An **unchanged** service could therefore drift below 90% on a PR and never be
re-gated until it is next touched. This is a genuine silent-drift window and scores the
sub-dimension at 6. It is, however, **mitigated today**: because all seven domains are
≥90% at HEAD (E1 above), there is no active drift to catch, which makes the recommended
fix (a global per-PR floor, or a scheduled full-matrix coverage job) low-risk — it
simply locks in the current state.

### 6.1.3 Integration, E2E, fuzz, and benchmark tiers

The remaining tiers are real but partially gated:

- **Integration (testcontainers).** A required CI job runs
  `go test -tags=integration -race` against a real Docker daemon and **fails (does not
  skip) when empty** (`ci.yml:653-666`), backed by an anti-erosion self-check that fails
  if an allowlisted suite loses its `//go:build integration` files
  (`ci.yml:636-651`, #553). This is well-engineered. The gap: of the four services that
  own integration suites (task-broker, agent-registry, memory-service, event-bus), only
  **two are gated** — `INTEGRATION_SUITES='task-broker agent-registry'`
  (`ci.yml:608-610,627`); event-bus and memory-service are excluded for "pre-existing
  deterministic failures" (event-bus godog step arity + DLQ timeout; memory-service
  DeleteNamespace cascade). Honestly documented, but a real coverage hole on the
  persistence and eventing paths.
- **E2E smoke.** `scripts/e2e/e2e-happy.sh:168-300` is a genuine end-to-end assertion:
  it POSTs a real workflow to `/api/v1/apply`, polls `GET /workflows/{id}` to a terminal
  succeeded state, and asserts CloudEvents on the NATS JetStream stream plus a
  memory-service KV roundtrip — not a no-op. It runs a `[temporal, argo]` matrix
  (`e2e-smoke.yml:55-61`). The qualifier: it is **gated, not a true required gate** — a
  no-op shim (`e2e-smoke-skip.yml:39-46`) satisfies the required check for any PR not
  touching e2e paths, so most PRs never run a real cluster E2E, and the argo leg is
  advisory only.
- **Fuzz / benchmark.** Two functional fuzz targets (`FuzzParseManifest`,
  `FuzzEvalGuard`) with a committed seed corpus, and benchmarks with a committed baseline
  (`tools/bench-baseline.txt`, `Makefile:357-377`), all exist and run clean (E1:
  `go test -fuzz=^FuzzParseManifest$ -fuzztime=5s` → 187 new-interesting inputs, 0
  crashers). But **no CI workflow invokes `make fuzz` / `make bench` / `benchstat`** —
  they are local-only and do not gate. The committed bench-baseline regression-guard
  therefore never fires.
- **Test smells.** The core `engine-adapter` service-level BDD suite is **entirely
  hard-skipped** pending a Temporal server (`services/engine-adapter/tests/steps_test.go:17`,
  "deferred to M6/M7"); the wired Temporal interpreter path is exercised only by the
  gated, non-required e2e-smoke. There is no property-based, mutation, or chaos testing;
  `pgregory.net/rapid` is a dangling go.sum entry imported by zero source files.

### 6.1.4 Drift test (Part 2 §2.6)

| Boldest claim | Result | Evidence |
|---|:---:|---|
| "140+ BDD scenarios across all services" (`ROADMAP.md:78`) | **VERIFIED** | 306 scenarios / 18 features; full suite green (E1) |
| "Every proto method has a BDD scenario" (ADR-016) | **VERIFIED** | 33/33 RPCs referenced in features (per-RPC grep loop) |
| "≥90% domain coverage, gate blocks" (`AGENTS.md:108,123`) | **VERIFIED** | exit-1 gate (`_test-go.yml:155-181`); 7/7 domains ≥90% (E1) — *caveat: per-changed-service, not a global floor* |

### 6.1.5 Sub-scores, flags, recommendations

| Sub-dimension | Score | Confidence |
|---|:---:|:---:|
| BDD contract coverage (every RPC ≥1 scenario) | 9 | High |
| Domain coverage gate ≥90% exists AND blocks | 9 | High |
| Coverage gate is per-changed-service (not global) | 6 | High |
| Integration tests (testcontainers, in CI) | 7 | High |
| E2E smoke applies workflow + asserts execution | 8 | High |
| Fuzz / benchmark presence + gating | 5 | High |
| Test smells | 6 | High |

**Red flags (severity-ordered).** (Medium) coverage gate enforces only on changed
services — silent-drift window (`_test-go.yml:132-134`), mitigated today;
(Medium) fuzz + benchmarks do not gate CI; (Medium) engine-adapter service-level BDD
hard-skipped; (Medium) integration gate covers 2 of 4 testcontainer services;
(Low) e2e-smoke "temporal" required only nominally via a shim; (Low) proto-BDD runs
against stubs, not production code; (Low) no property/mutation/chaos testing.

**Green flags.** Real, blocking, single-source-of-truth coverage gates across all tiers,
all exit-1; ≥90% domain coverage proven by execution on all seven services; BDD-before-code
at the gRPC boundary (306 green scenarios, every RPC covered); the integration gate is
anti-erosion engineered (fails when empty, self-checks its own allowlist); the E2E
happy-path is a true end-to-end assertion; functional fuzzing with a committed corpus.

**Recommendations.** **P1** — make the domain coverage gate a global per-PR floor (or a
scheduled full matrix) to close the silent-drift window; wire `make fuzz` +
benchstat-vs-baseline into a scheduled CI job so they gate; restore engine-adapter
service-level BDD via a Temporal dev-container and promote the event-bus / memory-service
integration suites. **P2** — adopt `rapid` for real property tests on the IR parser or
remove the dangling dependency; promote the e2e-smoke argo leg from advisory to required
(per #1092).

---

## 6.2 DevOps / CI-CD — Score: 8 / 10 (High)

**Mission.** Assess CI/CD, release engineering, automation, reproducibility, build
speed, quality gates, versioning, and operational observability of the delivery pipeline
itself.

**Verdict.** This is a genuinely strong, supply-chain-grade pipeline that is well above
the market norm for a project of this size, and — together with test rigor (6.1) — it is
one of the two most defensible verified assets in the diligence. The two material
weaknesses are the **amd64-only deployed service images** and the inability to confirm
supply-chain artifacts in the registry from a read-only sandbox.

### 6.2.1 Build-once / promote-by-retag — VERIFIED (the standout)

The defining property of the pipeline is ADR-027's **build-once / promote-by-retag**:
production service images are built **exactly once** in pre-merge CI, scan-gated by
Trivy, and then promoted to production purely by manifest retag — the deployed binary is
provably the scanned binary (**scan == deploy**, no post-merge rebuild nondeterminism).

- Single pre-merge build to a staging lane: `ci.yml:789-913` (build-images: build +
  Trivy + SBOM).
- Promotion is retag-only: `release.yml:160-204` (`imagetools create` retag
  staging → `main-<sha>` → `latest`, then `cosign sign`) — **`release.yml` contains zero
  image-build steps**.
- The decision is recorded as a one-way-door ADR
  (`ADR-027-shift-left-pipeline.md:32-47,88-90`, "build exactly once… promote… never
  rebuild"; the rebuild path is explicitly rejected).
- The loop is proven to run in production by execution:

> **E1** — `git log --grep='skip ci'` → 15 consecutive
> `chore(images): sync digests after main-<sha>` commits (incl. HEAD~1, `5a26f51`),
> evidencing the live merge → retag → digest-sync cycle.

### 6.2.2 Blocking gates and digest pinning — VERIFIED

Twelve real merge-blocking status checks are enforced through a modern repository ruleset
(not the legacy branch-protection API, which is why a naive `branches/main/protection`
call 404s):

> **E1** — `gh api repos/zynax-io/zynax/rulesets/17547241` → `required_status_checks`
> contexts: `dco`, `test-unit`, `security`, `lint-proto`, `lint-go`, `lint-python`,
> GitHub Actions workflow lint, Conventional Commit title, PR size label,
> Secret scan (gitleaks), `e2e smoke (temporal)`, `test-integration`;
> `strict_required_status_checks_policy: true`.

The SKIPPED-required-check bypass footgun is explicitly engineered against: `test-unit`
and `test-integration` use `if: always()` and self-checks so a required gate can never
silently pass as neutral/no-op (`ci.yml:573-591,602-651`; #986, #553).

Digest pinning is the supply-chain bedrock and is **live-green by execution**:

> **E1** — `cd cmd/zynax-ci && GOWORK=off go run . images check` →
> "✅ All consumer files are aligned with images/images.yaml."

`images/images.yaml` is the single source of truth for sha256 digests; the drift gate
runs three ways (`ci.yml:456-460` lint-go; `pr-checks.yml:321-337` dedicated job;
Makefile `check-images`).

### 6.2.3 Release engineering, CI-as-code, observability

- **Release engineering (8, High).** Semver tag-driven (`release.yml:46-47`); cosign
  keyless signing at merge and version (`:201,504`); SLSA L2 provenance attestation +
  SPDX SBOM on version images (`:510-533`); the CLI is cross-compiled across five
  platforms (`:308-315`, linux/darwin/windows × amd64/arm64); the SDK publishes to PyPI
  via an OIDC Trusted Publisher with **zero stored keys**
  (`sdk-publish.yml:31,107-111`).
- **CI-as-tested-code (ADR-036, 8, High).** Brittle inline shell/python has been
  consolidated into a Go CLI (`zynax-ci`) with **22 test files** and an enforced ≥80%
  coverage gate (`tools/coverage-gates.env:7`; `_test-go.yml:327,339`), used for digest
  sync, canvas/schema/milestone validation, and dependency/expert-mapping drift checks.
- **Automation hygiene (8, High).** The post-merge digest bot commits with a skip-ci
  marker; loop-safety is reasoned in ADR-027 (no CI run → no `workflow_run` → no retag →
  no loop) and confirmed by the 15 live bot commits above.
- **Observability (7, Medium).** Failures emit `::error::` + remediation hints; Trivy
  SARIF flows to the Security tab; the weekly audit fails loudly rather than spamming
  `[AUTO]` issues (`weekly-audit.yml:1-8,207-232`). Gaps: no flaky-test retry/quarantine
  harness, and no pipeline DORA telemetry surfaced in-repo (build-speed rests on
  caching + change-detection *design*, E3, not measured timings).

### 6.2.4 Drift test (Part 2 §2.6)

| Boldest claim | Result | Evidence |
|---|:---:|---|
| "21 workflows in `.github/workflows/`" (ADR-027 / §1.11) | **VERIFIED** | `ls .github/workflows | wc -l` → 21 (E1); matches §1.11 exactly |
| "build-once-promote-by-retag — production images never rebuilt after merge" | **VERIFIED** | `release.yml:160-204` retag-only (zero build steps); live `skip ci` digest-sync history (E1) |
| "The 12 listed required gates actually BLOCK merge" | **VERIFIED** | ruleset 17547241, strict policy (E1) — *governance nuance: `required_approving_review_count:0`, automation can self-merge on green* |
| "Multi-arch (arm64) builds for shipped artifacts" | **PARTIAL** | arm64 verified for tools/ci-runner + CLI binaries; **amd64-only for deployed service/adapter images** (`ci.yml:861`; `release.yml` retag-only) |

### 6.2.5 Sub-scores, flags, recommendations

| Sub-dimension | Score | Confidence |
|---|:---:|:---:|
| Blocking gates (required vs advisory) | 9 | High |
| Reproducibility (build-once / retag, ADR-027) | 9 | High |
| Digest pinning (ADR-024, check-images) | 9 | High |
| Release engineering (semver/sign/SBOM/PyPI OIDC) | 8 | High |
| Automation hygiene (digest bot, skip-ci, proto regen) | 8 | High |
| Build speed / caching, Docker-only, GOWORK=off | 8 | High |
| CI-as-tested-code (ADR-036) | 8 | High |
| Pipeline observability | 7 | Medium |

**Red flags (severity-ordered).** (Medium) **production service & adapter images are
amd64-only** — the single ADR-027 build (`ci.yml:861`) is `linux/amd64`, retag adds no
arch, so arm64 K8s and Apple-Silicon local Docker cannot run the official service images;
`service-release.yml:80` has multi-arch but is `workflow_dispatch`-only (not the live
release path). (Low) doc-vs-config drift: `e2e-smoke.yml:17-19` says it is "NOT a
required PR gate," but the ruleset requires `e2e smoke (temporal)` (correctly shimmed) —
a §1.10-class drift. (Low) `ai-context-budget.yml:44-45` "Advisory only — never fail CI"
vs `continue-on-error:false`. (Low) token/bus-factor concentration: retag + digest push
depend on a single admin-owned `BOT_GITHUB_TOKEN` as the sole ruleset-bypass actor.

**Green flags.** Build-once / promote-by-retag (scan == deploy), formalised as a
one-way-door ADR; digest SoT enforced three ways, live-green; SKIPPED-required-check
bypass explicitly defeated; CI-as-tested-code (Go CLI, 22 tests, ≥80% gate); shift-left
security in the merge gate (Hadolint → Trivy CRITICAL/HIGH exit-1 → SARIF → SBOM; cosign;
SLSA L2; SDK → PyPI OIDC, zero stored keys); enforced coverage gates as a single SoT.

**Recommendations.** **P1** — make production service/adapter images multi-arch (add
`linux/arm64` to the build-images platforms or a buildx matrix), or explicitly document
amd64-only as a supported-platform constraint in the Helm/quickstart docs. **P2** —
reconcile the e2e-smoke and ai-context-budget doc-vs-config drifts; add flaky-test
handling and surface basic pipeline DORA metrics; reconsider `required_approving_review_count:0`
given the M8 CNCF aspiration and the single-maintainer bus factor.

> **Cross-reference — supply-chain artifacts.** Cosign signing and SLSA attestation are
> **config-VERIFIED** (E3: `release.yml:201,510-533`) but their **existence in GHCR is
> UNKNOWN** — `cosign` is absent locally and the audit sandbox has no registry pull
> access (`cosign version` → command not found). This leaves contradiction-register row
> **C4** as *config-VERIFIED / artifact-UNKNOWN*; it is a Wave D confirmatory item, not a
> finding of absence. Full security-posture treatment is in Section 5.

---

## 6.3 Documentation — Score: 7 / 10 (High)

**Mission.** Assess documentation coverage, accuracy versus code, cross-doc consistency,
maintenance burden, and the onboarding path; verify whether the §1.10 doc-vs-tooling lag
persists at HEAD.

**Verdict.** Zynax's documentation is broad, layered, and — on the sampled claims —
accurate to HEAD: **7 of 8 sampled claims VERIFIED** against source/config, the
onboarding path is clean, and genuine single-source-of-truth mechanisms curb structural
drift. The Truth-Pass culture is demonstrably working. The one persistent weakness is the
**README itself**, which self-contradicts and under-claims shipped capability. Net:
strong, honest, slightly aging at the front door.

### 6.3.1 Coverage, accuracy, onboarding — strong and VERIFIED

Coverage is broad and layered (8, High): install/quickstart/developer-guide/authoring/
observability docs, **37 ADRs** (`ls docs/adr/ADR-*.md | wc -l`), per-service AGENTS.md,
and **10 example manifests** (`spec/workflows/examples/`, incl. `code-review-ollama.yaml`).

Accuracy is high (8, High): of eight concrete doc claims sampled against source/config,
**seven VERIFIED and one PARTIAL**. Verified items include the CLI default URL
(`cmd/zynax/cmd/root.go:40` ↔ `docker-compose.yml:282` port mapping), every quickstart
CLI subcommand existing in `cmd/zynax/cmd/`, the default demo model derived from a single
source (`llm-adapter.config.yaml:27` == `Makefile:154`), and engine portability
(temporal.go + argo_engine.go both implement `WorkflowEngine`). The one PARTIAL is
`make demo`: the target is fully **wired** (`Makefile:162-205`, `code-review-ollama.yaml`)
but its runtime end-to-end success could not be executed in the read-only audit
environment (E2/E3, not E1), and the hero asciinema cast is still a placeholder.

The onboarding path is clean and low-friction (8, High): clone → `make demo` →
quickstart → authoring → examples, with a zero-secret local-LLM default
(`README.md:24-33`) and a command reference that explicitly cites `cmd/zynax/cmd/` as the
source-of-record.

### 6.3.2 The §1.10 doc-vs-tooling lag — RESOLVED at HEAD (green flag)

A central diligence question was whether the framework's flagged "known lag" — CLAUDE.md
and the SPDD guide citing pre-PR#1400 command names — still persisted. It is **RESOLVED
at this HEAD**: both files now cite the live 5-verb surface (`/plan`, `/deliver`,
`/lib:spdd-*`), matching the `.claude/commands/` tree exactly
(`CLAUDE.md:86-110`; `docs/patterns/spdd-guide.md:7-45`). This is concrete evidence that
the Truth-Pass reconciliation habit is *operating*, not merely claimed. (A low-severity
residual lag survives only in `pr-checks.yml:240` and `APPLY_LOG.md`, treated in 6.4.)

### 6.3.3 The README self-contradiction — the principal red flag

The single Medium red flag is that **the README internally self-contradicts and lags its
own service table by ~2 milestones.** The Quickstart "M5 status note"
(`README.md:333-337` and `:355`) tells users that capability dispatch is "pending M5.C"
and that they must register an adapter first — while the *same file's* Service Status
table (`README.md:446-464`) and the ROADMAP report that dispatch shipped in v0.4.0 (M5.C)
with the end-to-end demo green. A new evaluator reading top-down hits the stale note
first and may conclude the product is **less** complete than it is. This is the §1.10
narrative-vs-delivery drift class — here notably in the *opposite (under-claiming)*
direction, which is the safe direction, but it still misleads. Three smaller stale lines
compound it: task-broker described as "In-memory" (`:227`) versus "Postgres-backed
pgx/v5" (`:451`); "Helm charts (planned, #241)" (`:503`) despite Helm shipping in M6; and
"ADR-001 – ADR-019" (`:504`) versus the 37 ADRs on disk.

Two Low red flags round it out: the hero "See it run" asciinema cast is a non-functional
placeholder (`README.md:37`, honestly flagged in `docs/casts/README.md:35`); and
CLAUDE.md's milestone-program map (`:129`, M7→M-dx→M8) has not absorbed the M-UX
insertion that ROADMAP.md already carries (`ROADMAP.md:57-61`, M7→M-UX→M-dx→M8).

The underlying driver is maintenance surface area (6, Medium): 130+ markdown files
across multiple dated review directories, with the README as the chronic laggard. This is
offset by genuine CI-enforced SoT mechanisms — `images.yaml` + `make check-images`
(`README.md:168-179`) and the config-derived `DEMO_MODEL` (`Makefile:154`).

### 6.3.4 Drift test (Part 2 §2.6)

| Boldest README claim | Result | Evidence |
|---|:---:|---|
| "Run on Temporal **or** Argo without a rewrite" | **VERIFIED** | temporal.go + argo_engine.go both implement `WorkflowEngine`; `e2e-smoke.yml:61` `[temporal, argo]` matrix |
| "No SDK required — any gRPC service is a capability via AgentService" | **VERIFIED** | `agent.proto:34-46` (`ExecuteCapability`/`GetCapabilitySchema`); ADR-013; 5 adapters shipped |
| "`make demo` runs the hero workflow end-to-end with a real model" | **PARTIAL** | fully wired (`Makefile:162-205`) but runtime not executable in audit env (E2/E3, not E1); cast is a placeholder |

> **Note on the engine-portability claim.** The Documentation agent VERIFIED this claim
> at the *contract/wiring* level (both engines implement the interface; the CI matrix
> exists). The Architecture agent (Section 4) independently graded the same claim
> **PARTIAL** at the *execution* level — only Temporal interprets the IR; the Argo leg
> runs a non-interpreting stub. Both are consistent: the documentation accurately
> describes what the contract does, while the deeper architectural finding is that
> operational parity is not yet reached. The contradiction is recorded for Wave D.

### 6.3.5 Sub-scores, flags, recommendations

| Sub-dimension | Score | Confidence |
|---|:---:|:---:|
| Coverage map | 8 | High |
| Accuracy vs code/CI (8 samples) | 8 | High |
| Cross-doc consistency | 6 | High |
| Maintenance / SoT / drift | 6 | Medium |
| Onboarding from docs alone | 8 | High |
| Truth-Pass honesty / drift-lag | 8 | High |

**Red flags.** (Medium) README self-contradicts and under-claims shipped capability
(`README.md:333-337,355` vs `446-464`; plus `:227,:503,:504`); (Low) hero asciinema cast
is a dead placeholder; (Low) CLAUDE.md milestone map omits the M-UX insertion.

**Green flags.** The §1.10 doc-vs-tooling lag is already RESOLVED at HEAD; high sampled
accuracy (7/8 VERIFIED) — docs describe HEAD, not aspiration; CI-enforced
single-source-of-truth (images.yaml + check-images; config-derived DEMO_MODEL);
low-friction, zero-secret onboarding with source-of-record cross-links.

**Recommendations.** **P1** — truth-pass the README: delete/replace the stale "M5 status
note" and fix the three stale lines (task-broker, Helm, ADR count). **P2** — record and
embed the `make demo` asciinema cast; sync CLAUDE.md to include M-UX.

---

## 6.4 AI-native development / SPDD — Score: 7 / 10 (High)

**Mission.** Assess whether Zynax's SPDD pipeline, command surface, context management,
and learnings loop are a genuine asset or a liability — material because the methodology
produces most of the platform's code.

**Verdict.** Zynax has built a coherent, unusually disciplined AI-native development
methodology — REASONS Canvas governance (ADR-019), three-tier KB security (ADR-018), a
consolidated 5-verb command surface, and a genuinely closed learnings flywheel — and it
has **run at scale (37 Implemented canvases** with real expert-guide LoC deltas). The
headline weakness is that the canvas-before-code **gate is softer than advertised**. The
system is a real productivity asset and partly-novel process IP, tempered by
single-maintainer bus factor.

### 6.4.1 Canvas-before-code is a soft gate — the principal finding

ADR-019 advertises "Canvas before code" as an *enforced* gate (`ADR-019:35-41,66-69`,
Aligned required before generate). The audit found a real CI gate that is, in practice,
**soft** (sub-score 6, High):

- A `canvas-freshness` job exists and runs on `feat:` PRs (`pr-checks.yml:204`).
- But the "Canvas present" check passes whenever **any** canvas exists anywhere in
  `docs/spdd/` (`pr-checks.yml:231-233`:
  `canvas_existing=find docs/spdd -name canvas.md | head -1`). Since ~30 canvases already
  exist, this branch **never fails** — a `feat:` PR with no canvas for *its own* issue
  still passes.
- A `Draft` canvas emits only a `ValidationWarning`, not a `ValidationError`
  (`cmd/zynax-ci/validate/canvas.go:124-129`) — so "Aligned before merge" is **not
  machine-enforced**; it rests on human review plus branch protection.

This is the drift-test result: *"Canvas-before-code is an enforced gate"* → **PARTIAL**.
It is the most likely leak point for AI-authored quality issues and the primary P1
recommendation.

The validator itself is correct where it runs (7, High) — it checks the 7 REASONS
sections, header fields, status enum, security marker, and absence of a committed private
file, and it flagged two real issues by execution:

> **E1** — `cd cmd/zynax-ci && GOWORK=off go run . validate canvas ../../docs/spdd/`
> → most canvases OK, exit 1; FAIL on `1359-...` (invalid Status `Superseded`) and on a
> `canvas.private.md` found on disk. The latter is a disk false-positive — `git ls-files`
> shows only `canvas.md` tracked (`.gitignore:53`), so the leak control holds at the git
> layer.

### 6.4.2 The learnings loop — the standout green flag

The closed, traceable learnings loop is the strongest asset in this dimension (8, High).
`docs/ai-learnings/APPLY_LOG.md:15-99` shows a **Draft → applied/rejected lifecycle**
with source-session traceability and real committed LoC deltas to the expert guides (e.g.
`ci-release.md` +20L), and it **deliberately rejects** structural-workaround and
env-constraint patterns rather than polluting the guides (3 structural rows rejected, 2
pending at 2026-06-18). The accumulated guides are substantial, not stubs
(`ci-release.md` 647L, `go-services.md` 633L). This is a working flywheel, not ceremony —
the drift-test claim *"Canvases map to shipped features"* is **VERIFIED** (37 Implemented;
the 214 canvas ↔ `temporal_workflow.go`/`interpreter.go` shipped code).

### 6.4.3 Command surface, KB tiering, self-hosting

- **Command surface (8, High).** The live `.claude/commands/` tree exactly matches the
  claimed shape (5 verbs + milestone + README + 19 `lib/` + 8 `experts/`); the verbs
  encode hard-won operational discipline directly into the prompt surface — DCO
  Signed-off-by, Assisted-by (not Co-Authored-By), squash-only (because
  `required_signatures` blocks rebase), runtime-evidence-not-config, re-run-stateful-paths-twice
  (`deliver.md:33-47`). Consolidation from a prior 20+ command sprawl to 5
  milestone-agnostic verbs is a genuine cognitive-load reduction (8, Medium).
- **Context management / KB tiering (8, High).** ADR-018/019 three-tier classification is
  backed by *real, enforced* controls: gitleaks AI-context config
  (`pr-checks.yml:385-388`), the ai-context-budget gate (`ai-context-budget.yml:7-13`),
  CODEOWNERS on every KB path (`CODEOWNERS:15-20`), and gitignored private canvases
  (`.gitignore:53`). Best-in-class for an AI-authored public repo.
- **Self-hosting automation (6, High).** Honestly delivered to a boundary: the
  orchestrator + expert mesh are authored as real Zynax manifests with passing schema
  tests, but the live `zynax apply` e2e is explicitly **deferred to M7 (#1103)** behind a
  cleanly-skipping platform gate (`automation/README.md:16-43`;
  `test_platform_readiness.py:48-57`) — not yet running on itself, but not over-claimed
  either.

### 6.4.4 Drift test (Part 2 §2.6)

| Boldest claim | Result | Evidence |
|---|:---:|---|
| "Canvas-before-code is an ENFORCED gate (ADR-019), not a convention" | **PARTIAL** | gate job real (`pr-checks.yml:204`) but passes if any canvas exists (`:231-233`); Draft only warns (`canvas.go:124-129`) |
| "Canvases map to shipped features (no canvas/feature drift)" | **VERIFIED** | 37 Implemented; 214 canvas ↔ `temporal_workflow.go`/`interpreter.go` shipped |
| "CLAUDE.md §SPDD + spdd-guide.md still lag the live commands (§1.10)" | **CONTRADICTED — lag RESOLVED** in those two files; residual lag persists in `pr-checks.yml:240`, `APPLY_LOG.md`, and internal `.claude/commands/` cross-refs to retired files |

### 6.4.5 Sub-scores, flags, recommendations

| Sub-dimension | Score | Confidence |
|---|:---:|:---:|
| SPDD enforcement (gate vs convention) | 6 | High |
| Canvas validator correctness (E1) | 7 | High |
| Command surface quality | 8 | High |
| Cognitive-load reduction | 8 | Medium |
| Context mgmt / KB tiering (ADR-018) | 8 | High |
| Hallucination/quality prevention | 8 | High |
| Self-hosting automation real? | 6 | High |
| Defensible IP vs overhead | 7 | Medium |

**Red flags (severity-ordered).** (Medium) soft canvas-freshness gate — any-canvas-passes
+ Draft-only-warns → "Aligned before merge" not machine-enforced; (Low) command-name lag
survives inside the consolidated files themselves (`learn.md:193,352`,
`experts/go-services.md:287-288` → retired files; `pr-checks.yml:240`; `APPLY_LOG.md:3,8`);
(Low) validator enum drift ('Superseded' rejected; gitignored private.md hard-flagged on
disk); (Low) single-maintainer-authored IP, ADR-033 still Proposed, full expert library
deferred to M-dx.

**Green flags.** A closed, disciplined learnings loop that rejects structural-workaround
patterns and shows real committed deltas (`APPLY_LOG.md:15-99`); multi-layer KB-leak
control (gitleaks-ai-context + budget gate + CODEOWNERS + gitignored private canvases +
security-review); verbs encode operational scars as reusable guardrails
(`deliver.md:33-47`); self-hosting honestly bounded behind a cleanly-skipping gate, not
over-claimed.

**Recommendations.** **P1** — harden the canvas-freshness gate: require a canvas whose
directory matches the PR's issue number, and fail (not warn) if its Status is not Aligned
at merge time. **P2** — run `/reconcile` to sweep the residual command-name lag and add
'Superseded' to the validator status enum; commit a velocity/quality baseline (pre- vs
post-SPDD DORA or defect-rate) to upgrade the productivity-multiplier claim from CLAIMED
(E5) to VERIFIED (E1).

> **Productivity multiplier is CLAIMED, not VERIFIED.** No velocity/DORA
> baseline-vs-SPDD comparison is committed; the multiplier rests on first-party docs
> (E5), not measured proof. The methodology is a coherent productivity asset, but its
> headline value-prop is unevidenced and its defensibility is copyable process discipline
> tied to a single maintainer (cross-reference Section 4.7 Innovation & IP).

---

## 6.5 Cross-cutting observations and unknowns ledger

**The unifying theme.** Across all four dimensions the verified assets are real and
substantial — 306 BDD scenarios over every RPC, ≥90% domain coverage proven by execution,
build-once / promote-by-retag, an end-to-end e2e assertion, a closed learnings loop, and
genuine SoT mechanisms. The residual gaps are **uniformly enforcement-shaped, not
absence-shaped**: the coverage re-gate is per-changed-service (6.1), the e2e/fuzz/bench
gates are advisory or local-only (6.1), the multi-arch story stops short of deployed
service images (6.2), and the canvas gate is soft (6.4). The capabilities exist; the
hardening of their gates is the work that remains.

**Open questions / unknowns (honest ledger).**

- **E1 runtime not executable in the read-only audit env.** The kind-cluster e2e, the
  Postgres testcontainer integration suites, and `make demo` end-to-end were verified to
  E2/E3 wiring only — no live green run was obtained (6.1, 6.3).
- **Supply-chain artifacts in GHCR.** Cosign signatures and SLSA attestations are
  config-VERIFIED but their existence in the registry is **UNKNOWN** (cosign absent, no
  registry pull access) — contradiction-register row C4 remains an open Wave D
  confirmatory item (6.2).
- **No measured pipeline telemetry.** Build-speed and the SPDD productivity multiplier
  rest on design (E3) and docs (E5) respectively; no DORA/velocity numbers are committed
  (6.2, 6.4).
- **Enforcement-window questions.** Will a wide refactor that leaves an untouched
  low-coverage package below the floor be caught (6.1)? Is Aligned-before-merge enforced
  anywhere but human review (6.4)?

**Cross-references.** 6.1 ↔ 6.2 (CI required-check set, e2e-smoke-skip shim, coverage
gates as a shared SoT); 6.2 → Section 5 (cosign/SBOM/SLSA artifact existence, C4); 6.3 ↔
Section 4 (engine-portability claim VERIFIED at contract level here, PARTIAL at execution
level there); 6.4 → Section 4.7 (SPDD/Canvas/learnings as the primary Innovation/IP
candidate; defensibility is copyable process discipline); 6.4 ↔ Section 7 (single-maintainer
bus factor, automation token concentration, `required_approving_review_count:0` feed the
governance and CNCF-readiness picture).

# 7. Open Source, Governance & CNCF

> **Section verdict (un-weighted, directional):** Open-source health **5.8 (High)** · Governance
> **7 (High)** · CNCF readiness **5 (High)** · Repository health **7 (High)** · Future-roadmap
> realism **7 (High)**. The defining finding of this section is that Zynax has the *governance,
> transparency, and engineering discipline of a project far more mature than its actual community* —
> and that the binding constraint on every social objective (v1.0, CNCF Sandbox, acquisition
> durability) is **people, not code**. Two hard but cheap-to-fix execution gaps (a missing LICENSE
> file, a missing MAINTAINERS.md) gate CNCF; one structural fact (bus factor = 1) gates everything.

Evidence convention follows Part 2 §2.4: `VERIFIED` facts rest on E1–E4 (executed command,
source, CI/config, or contract); `CLAIMED` facts rest on E5–E6 (first-party docs, roadmap,
marketing) and are labelled as such. Source packets for this section are the read-only outputs of
Wave C agents 5.8 / 5.13 / 5.21 / 5.25 and Wave A agent 5.24, all evaluated at `main @ e3135a6`;
the full §3.4 packets live in Appendix A (see `docs/due-diligence/2026-06-20-dd-wave-c-findings.md`
and `docs/due-diligence/2026-06-20-dd-wave-a-findings.md`).

---

## 7.0 Section scorecard

| Sub-section | Agent | Dim | Score | Confidence | Most severe red flag | Strongest green flag |
|---|---|:---:|:---:|:---:|---|---|
| 7.1 Open-source health | 5.8 | D7 | 6 | High | No top-level LICENSE file (never in git history); bus factor 1 | Exemplary Truth-Pass transparency + honest solo-maintainer governance |
| 7.2 Governance | 5.13 | D12 | 7 | High | `MAINTAINERS.md` referenced but absent (#494); "no single company controls" contradicted | Self-correcting milestone re-labelling (M3/M4 → Partial with reasons) |
| 7.3 CNCF readiness | 5.21 | D7 | 5 | High | **Critical** — missing LICENSE makes the project non-donatable; bus factor 1 + 0 adopters layered on top | Unusually honest self-assessment that says "don't file until prerequisites are real" |
| 7.4 Repository health | 5.24 | D16 | 7 | High | Bus factor = 1 (one human identity, 772 commits) | Strict signed, squash-only, linear history; 94% issue closure |
| 7.5 Future-roadmap realism | 5.25 | D12 | 7 | High | Binding v1.0/CNCF constraint is **social**, not buyable by velocity | Data-flow keystone (ADR-029) shipped and execution-verified; EPIC #1167 + W.2–W.5 closed |

**Section un-weighted mean ≈ 6.4 / 10.** The five agents converge from independent angles on a
single thesis (§7.6): the engineering substrate is well ahead of the community substrate.

---

## 7.1 Open-source health — Score: 6 (High)

**Mandate.** License hygiene, governance maturity, bus factor, contribution funnel, issue/PR
management, release cadence, and transparency / Truth-Pass culture, with a drift test on any
"community" claim against actual stars / forks / contributors.

Zynax presents the governance and transparency posture of a project far more mature than its
community, paired with one embarrassing legal-hygiene gap and the single-maintainer ceiling that
gates everything social. `GOVERNANCE.md`, `CONTRIBUTING.md`, the issue/PR template set, and the
`SECURITY.md` disclosure policy are all best-in-class for a solo CNCF-aspirant, and the transparency
is genuinely exemplary — the project's own documents state "zero stars, zero forks, zero external
adopters" rather than inflating a community. Two material defects pull the score to a 6.

### 7.1.1 The LICENSE gap (VERIFIED red flag, High)

The most concrete defect is the absence of a top-level `LICENSE` file. This is not a stale or
deleted artifact — it has never existed in git history: `ls LICENSE` returns "no such file" (E1),
`git ls-files | grep -i license` returns only `docs/adr/ADR-005-apache-license.md` (E1), and
`git log --all -- LICENSE` is empty (E1). The README badge and License section both link to the
non-existent file (`README.md:13,527`, E2), and `GOVERNANCE.md:442` overstates the situation as
"Apache 2.0 license (done)" (E5). The decision is recorded (ADR-005 Accepted) and SPDX headers are
pervasive — every one of the 21 workflows plus the Makefile and source files carry
legally operative artifact that Apache-2.0 §4 requires to accompany distribution. There is no
vendored third-party source (`find -iname vendor`/`third_party` empty, E1), so a missing `NOTICE` is
lower-risk, but the `LICENSE` itself is a one-line fix and a hard CNCF blocker.

### 7.1.2 ADR-005 enforcement drift (Medium)

ADR-005 makes two enforcement claims that are contradicted at HEAD. It asserts SPDX headers are
"enforced by license-eye in CI" (`ADR-005:16`, E5), but no such tool is wired anywhere
(`grep -rniE 'license-eye|reuse|.licenserc'` over workflows/Makefile/tools returns nothing, E1). It
asserts "Contributors must sign CLA (automated via GitHub CLA bot)" (`ADR-005:17`, E5), but the
project uses DCO, not a CLA — `ci.yml:56-74` runs a `dco` job verifying `Signed-off-by` on every
commit (E3), and no CLA bot exists. An Accepted ADR describing controls that were never implemented
is precisely the delivery-vs-narrative drift class this diligence targets.

### 7.1.3 Transparency, funnel, and cadence (green)

Against those gaps sit strong positives. Transparency is the standout: 39 public ADRs (E1), a
documented Truth-Pass that already purged phantom CHANGELOG entries (#473, E2), and a strategy
document that states "0 stars, 0 forks, 0 external adopters" to its own face
(`strategy.md:290,338`, E5) — a rare case where self-reported status is *more* pessimistic than
reality (live signals: 1 star, 0 forks, E1). The contribution funnel is thorough and approachable
(prerequisites table, one-command `make bootstrap`, 2-business-day response SLA,
`CONTRIBUTING.md:13-66`, E5), though the heavy AGENTS.md / DCO / SPDD reading load before a first PR
is real Day-0 friction. Issue/PR management is mature: 7 structured issue templates, an 8 KB PR
checklist, a `config.yml` routing questions to Discussions and security to private advisories, a
documented triage process, and a 94% issue-closure ratio with zero orphaned PRs (E1/E3, corroborated
by 5.24). Releases are real and SemVer'd — `git tag` shows v0.4.0 and v0.5.0 with matching GitHub
Releases carrying SBOM/cosign artifacts (E1), and `SECURITY.md:14-26` defines private disclosure via
GitHub Advisories with explicit SLAs (E5). One process-vs-practice nit: the elaborate RFC machinery
(`GOVERNANCE.md:180-207`) has produced zero RFCs — only the template exists (E1) — because decisions
de facto route through ADRs.

**Drift test.** "Apache-2.0 LICENSE file present" → **CONTRADICTED** (file never existed). "SPDX
headers enforced by license-eye; contributors sign a CLA" → **CONTRADICTED** (DCO, not CLA; no
license-eye). "Healthy / growing community" → **UNKNOWN — no false claim made**: the repo's own docs
state zero traction and match the live signals, the desired Truth-Pass outcome. "Documented,
followable governance" → **VERIFIED** (`GOVERNANCE.md:98-137`).

**Sub-dimension scores (5.8).**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| License hygiene (LICENSE/NOTICE/SPDX/third-party) | 4 | High | `ls LICENSE`→absent; `git log --all -- LICENSE`→empty; `README.md:13,527`; SPDX headers present |
| ADR-005 enforcement drift (license-eye/CLA) | 3 | High | `ADR-005:16-17` vs grep (no license-eye); `ci.yml:56-74` (DCO not CLA) |
| Governance maturity (roles / decision-making) | 8 | High | `GOVERNANCE.md:27-137,434-447`; broken MAINTAINERS.md links `:86` |
| Bus factor (consumed from 5.24) | 3 | High | `git shortlog -sne`→1 human (772); #494 open |
| Contribution funnel | 7 | High | `CONTRIBUTING.md:13-66`; AGENTS.md reading load = Day-0 friction |
| Issue/PR management | 8 | High | `ISSUE_TEMPLATE/config.yml`; `GOVERNANCE.md:211-238`; 94% closure |
| Release cadence + tags | 7 | High | `git tag`→v0.4.0/v0.5.0; `gh release list`; `SECURITY.md:14-26` |
| Transparency / Truth-Pass culture | 8 | High | 39 ADRs; `strategy.md:290,338`; 0 RFCs |

---

## 7.2 Governance — Score: 7 (High)

**Mandate.** Decision discipline (ADRs / one-way doors), planning quality (milestones, exit
criteria, realism vs cadence), execution strategy (claim / track / close, cross-machine safety,
truth-pass), risk management, and ownership / sustainability — plus a drift test on a past
milestone's "Complete" claim.

Zynax has unusually mature *process* governance for a two-month-old project, dragged down by one
structural fact and a stale charter. The decision substrate is strong and machine-checked: 37 ADRs,
all present as files and listed in `INDEX.md` with status/date/governs columns, including a recorded
*Rejected* ADR-037 (negative decisions kept, not hidden — `INDEX.md:14-50`, E2). Program-management
quality is investment-grade: `M7-planning.md` carries a 22-deliverable brief-coverage map, an 8-row
risk register with likelihood/impact/mitigation/owner, a 12-criterion acceptance matrix with
concrete done-checks, per-issue Definition of Done, and rollout/rollback (`M7-planning.md:485-712`,
E5/E3), validated against a strict JSON-Schema milestone state machine
(`milestone.schema.json:29-62`, E4). Merge governance is enforced, not merely documented:
ADR-023 squash-only + required signatures + linear history (0 merge commits, E1), DCO, a 7-type
conventional-commit gate, and a defined AI-contributor policy (human sponsor of record,
`Assisted-by` not `Co-Authored-By`, `GOVERNANCE.md:318-368`, E5).

### 7.2.1 The signature green flag: honest milestone re-labelling

The strongest governance signal is the project's self-correcting "Truth Pass." M3 and M4 were
down-graded from *Complete* to *Partial*, with the explicit reason carried in the canonical state
file and CLAUDE.md: "M3/M4 are partial because task-broker and agent-registry were not delivered in
those milestones. Both completed under M5.C (#460)" (`state/current-milestone.md:17-23`, E2/E5). A
recurring Truth-Pass habit (M5.A #458; the 2026-06-17 M7 reconcile) actively re-aligns narrative to
shipped reality (`current-milestone.md:80-82`, E2). This is exactly the anti-drift behaviour the
diligence framework's §1.10 history exists to reward — the inverse of the optimistic over-claiming
that the framework was built to detect.

### 7.2.2 Founder dependency and the absent MAINTAINERS.md (High)

The dominant unmitigated risk is founder dependency, partly misrepresented by the charter.
`MAINTAINERS.md` is referenced by `GOVERNANCE.md:86`, by CODEOWNERS, and by the add/remove-maintainer
process (§10) — yet `ls MAINTAINERS.md` returns "no such file" (E1) and the artifact is an open
backlog issue (#494). The `@zynax-io/maintainers` team that CODEOWNERS routes every sensitive path
to is effectively one human (bus factor 1, per 5.24), and every ADR that carries a Deciders field
names the same single person (`ADR-026:9`, `ADR-029:9`, E2). Against this, `GOVERNANCE.md:6` asserts
"Governance is neutral — no single company controls decisions" — a CLAIMED statement contradicted by
the repository state. The process is robust *on paper* (solo-maintainer phase, RFC, supermajority),
but its robustness-to-founder-leaving is untested because no second human has ever exercised it.

### 7.2.3 Charter currency drift (Medium)

The anti-drift project's own charter has itself drifted. `GOVERNANCE.md:247-258` §6
Milestone→Version Mapping is materially stale and self-contradicts the live `ROADMAP.md:267-275` and
`state/milestone.yaml` (it maps M2→v0.2.0 where the actual is v0.1.0; M6→"Production Hardening
v1.0.0-rc.1" where the actual is "K8s Production-Ready v0.5.0"; M7→"Developer Experience"; and it
references the retired `zynaxctl` CLI name) (E2/E3). The §12 CNCF checklist still lists "External
security audit (target: v0.5.0)" unchecked though v0.5.0 shipped 2026-06-12 (E5). The documented RFC
gate (required for any proto/architecture/governance change) has never been used — a load-bearing
governance process that is partly theatrical until a second contributor triggers it. Minor
ADR-vs-code reconciliation debt rounds out the list: `INDEX.md:36` mislabels ADR-023 as
"rebase-merge only" where its body is squash-only (E2), and ADR-034 ("Proposed", random ids)
contradicts the shipped deterministic idempotent apply (cross-ref §4 Architecture / agent 5.1).

**Drift test.** Past "Complete" claim — *"M5 Complete, 7/7 DoD met"* (`ROADMAP.md:138-146`) →
**VERIFIED**: the boldest DoD items check out in code (cel-go is a real dependency driving guard
evaluation, `engine-adapter/go.mod:7`, `interpreter.go:201-215`, E2; agent-registry shipped with
full `internal/{api,domain,infrastructure}`, E1). *M3/M4 honestly re-labelled* → **VERIFIED** (the
green flag above). *"GOVERNANCE.md current & no single company controls"* → **CONTRADICTED** (stale
§6 table; MAINTAINERS.md missing; bus factor 1). *ADR process followed & INDEX complete/current* →
**PARTIAL** (37/37 indexed, but the ADR-023 row contradicts its own body and three "Proposed" ADRs
cover already-decided features).

**Sub-dimension scores (5.13).**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Decision discipline (ADRs / one-way doors) | 8 | High | `INDEX.md:14-50` (37 indexed, incl. Rejected ADR-037); `grep Deciders → 13/37` |
| Planning quality (scoping / exit criteria / realism) | 8 | High | `M7-planning.md:607-629,686-712`; `milestone.schema.json:29-62` |
| Execution strategy (claim/track/close, truth-pass) | 8 | High | `ROADMAP.md:138-146`; `current-milestone.md:80-82`; 5.24 squash-only/0 merges |
| Risk management (register + blocker surfacing) | 7 | High | `M7-planning.md:485-496`; `:277` ("Depends on #N") |
| Ownership & sustainability (CODEOWNERS / succession) | 4 | High | `CODEOWNERS:5-31`; `ls MAINTAINERS.md`→missing; one human; single ADR Decider |
| Governance-document currency | 5 | High | `GOVERNANCE.md:247-258,444` stale vs `ROADMAP.md:267-275`; 0 RFCs filed; INDEX ADR-023 mislabel |

---

## 7.3 CNCF readiness — Score: 5 (High) · Verdict: NOT YET (do not file)

**Mandate.** Map Zynax against CNCF Sandbox criteria at HEAD as a TOC reviewer would, and judge
readiness honestly.

The headline verdict is **structurally close but not Sandbox-ready — do NOT file now** — and the
single hardest blocker is not the social gap everyone expected but the **missing top-level LICENSE
file** (§7.1.1). This is a **Critical** red flag and the only one in this section: a canonical
license file is a CNCF Sandbox hard requirement and a basic legal expectation, and its absence makes
the project non-donatable as it stands (`git log --all -- LICENSE` empty; `git check-ignore LICENSE`
exit 1 confirms it is genuinely missing, not hidden, E1). Layered on top are the well-known social
gates, each independently confirmed: bus factor = 1 with no `MAINTAINERS.md` or `OWNERS` file (#494
open, on the M8 milestone, E1); zero named external adopters and no `ADOPTERS.md`
(`find -iname ADOPTERS*` empty; `strategy.md:290,338` self-reports the 0/0/0 baseline, E1/E5); and a
CNCF-backed direct competitor (Kagent, CNCF Sandbox 2026) occupying the same "control plane for AI
agents" framing, which sharpens the "why not them" question a TOC will ask.

What keeps this a 5 rather than a 2 is that the *structural* substrate is genuinely strong and the
project's self-assessment is unusually honest. Governance artifacts are near-complete: the
`CODE_OF_CONDUCT.md` adopts the CNCF CoC verbatim (`:5`, E2), `GOVERNANCE.md` is a full 12-section
neutral-governance document with a CNCF alignment checklist (`:1-451`, E5), `SECURITY.md` provides
private-advisory disclosure with SLAs and a `cosign verify` recipe (`:18-26,65-71`, E5), the roadmap
is public with the M8 social prerequisites explicitly unchecked (`ROADMAP.md:251-261`, E5), and the
supply chain is CNCF-credible (cosign keyless + syft SPDX SBOM + SLSA L2 wired in `release.yml`,
consumed from agent 5.2). Dev cadence is high and the history is signed and linear (consumed from
5.24). Crucially, the project's own strategy §8 already maps the Sandbox criteria with the social
gaps marked ❌ and explicitly advises "do not file until prerequisites are real"
(`strategy.md:330-346`, E5) — exactly the integrity a TOC rewards.

Two caveats temper the "structurally strong" reading. First, the OpenSSF Scorecard "✅" rests on a
live API badge (`README.md:16`), not a committed scan — `grep -rln scorecard .github/workflows/`
returns exit 1 (no `scorecard.yml`, E1), and the numeric score is UNKNOWN (no network). Second, the
"gap is purely social" framing (Part 1 §1.9) is itself **PARTIAL**: two *structural* artifacts
(LICENSE and MAINTAINERS/OWNERS) are also missing, so the gap is not purely social — one hard legal
artifact is absent too. Trademark/neutrality scores low (4): no trademark policy file
(`find -iname '*trademark*'` empty; deferred to M8.A, E1), and the documented vendor-neutrality is
contradicted by de-facto single-individual control, with the open-core monetization path flagged
in-doc as a neutrality tension (`strategy.md:365-366`, E5).

**Drift test.** "Apache-2.0 LICENSE present (§1.9 / GOVERNANCE §12 done)" → **CONTRADICTED**.
"OpenSSF Scorecard ✅" → **PARTIAL** (badge present, no committed workflow, value UNKNOWN).
"Structural alignment strong; gap is purely social" → **PARTIAL** (LICENSE + MAINTAINERS/OWNERS are
structural and also missing).

**Sub-dimension scores (5.21).**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Structural governance artifacts | 8 | High | `GOVERNANCE.md:1-451`; `CODE_OF_CONDUCT.md:5`; `SECURITY.md:18-26` |
| License (LICENSE file) | 2 | High | `git log --all -- LICENSE`→none; `README.md:13` dangling |
| Public roadmap + versioning | 8 | High | `ROADMAP.md:251-275` |
| Maintainer prereq + MAINTAINERS/OWNERS | 1 | High | `git shortlog` one human; `ls MAINTAINERS.md OWNERS` absent; #494 OPEN |
| Named adopters | 0 | High | `strategy.md:290,338`; no ADOPTERS file |
| Security disclosure + supply chain | 8 | Medium | `SECURITY.md:18-26`; 5.2 `release.yml:201,527,510` |
| Healthy dev cadence | 8 | High | 5.24 (121→259→444 commits/mo) |
| Trademark / neutrality | 4 | Medium | no trademark file; `strategy.md:340` |
| Differentiation vs Kagent | 7 | Medium | Part 1 §1.6; `strategy.md:184,200` |

**Critical-path recommendations (5.21).** P0 — commit the Apache-2.0 LICENSE at root and fix the
dangling references (hard gate, trivial fix). P0 — recruit and document a second-org maintainer;
create `MAINTAINERS.md` (close #494). P1 — land ≥1 named adopter (`ADOPTERS.md`) and a public
community cadence before filing; add `scorecard.yml` + a TRADEMARK policy. P2 — do not file the
Sandbox application until P0+P1 are real, and recruit a TOC sponsor in parallel — honoring the
project's own counsel (`strategy.md:345`).

---

## 7.4 Repository health — Score: 7 (High)

**Mandate.** Objective repo-health signals — commit activity, contributor distribution, branch/PR/
merge hygiene, drift artifacts, automation noise, and repo cleanliness — with a drift test on
claimed velocity.

This is a living, exceptionally well-tended, and disciplined repository with one dominant structural
weakness. Cadence is high and accelerating — `git log` per-month counts of 121 (Apr) → 259 (May) →
444 (Jun, first 20 days) on a ~2-month-old, 824-commit repo with HEAD hours old at audit (E1).
History is strictly linear: `git log --merges -100` returns 0 (squash-only per ADR-023, E1), human
commits are signed (`%G?` → "E" for humans, "N" only on bot digest-sync commits — though the signing
key is absent from the audit environment's trust store, so cryptographic validity is UNKNOWN at
Medium confidence, E1), and `gh pr list --state open` is empty (no orphaned PRs, E1). Automation
noise is maturely handled: the `[AUTO]` skeleton-issue mechanism was deliberately retired in favor of
loud red `weekly-audit.yml` runs (`:4-8`, E3), the historical burst (#1035–#1054) is fully closed,
and ~0 auto-issues remain open. The repo is clean (no committed binaries or build artifacts; a
curated `.gitignore:1-57`, E1) with a 94% issue-closure ratio (39 open / 632 closed, E1) where the
open set is genuine roadmap/epic/DD work, not stale clutter.

The one thing dragging the score down hard is **bus factor = 1** (the recurring red flag of this
entire section): `git shortlog -sne` shows a single human identity (772 commits across two emails),
with no second human author in the full shortlog or the last-50-commit window (E1). It is a single
point of failure and the unmet CNCF "≥2 maintainers" gate; `MAINTAINERS.md` remains open issue #494.
Two cosmetic, local-only nits round out the findings: ~28 squash-merged but unpruned local branches
and one uncommitted `go.work.sum` drift (148 insertions, not in committed history, E1); and the
hand-curated CHANGELOG `[Unreleased]` section lags ~115 closed M7 issues with only one entry — the
public-facing surface most prone to the project's documented narrative-drift class (a phantom-entry
purge already happened, `CHANGELOG.md:44` / #473), though the canonical state file is current.

**Drift test.** "M7 ~92% done" → **PARTIAL** (direction VERIFIED via M7.K PRs #1431–#1441 on main;
exact milestone numerator not isolable read-only). "Squash-only, signed (ADR-023)" → **VERIFIED**
(0 merge commits; human commits signed). "[AUTO] noise must be triaged" → **VERIFIED — mechanism
retired**. "High, sustained velocity" → **VERIFIED** (121→259→444 commits/month).

**Sub-dimension scores (5.24).**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Commit cadence & recency | 9 | High | per-month 121/259/444; HEAD 2026-06-20; 824 commits |
| Contributor distribution (bus factor) | 3 | High | `git shortlog`→one human (772); bots 45+7 |
| Branch/PR/merge hygiene & discipline | 8 | High | `--merges -100`→0; `gh pr list open`→empty |
| Commit signing posture | 8 | Medium | `%G?`→human "E", bot "N"; key absent locally |
| `[AUTO]` pile & triage | 8 | High | `weekly-audit.yml:4-8`; 23 AUTO total, 0 open |
| Repo cleanliness | 8 | High | no artifact/binary tracked; `.gitignore:1-57` |
| Open-vs-closed issues | 8 | High | 39 open / 632 closed (94%) |
| CHANGELOG / state currency | 6 | High | CHANGELOG `[Unreleased]` lags ~115 M7 closes |

---

## 7.5 Future-roadmap realism — Score: 7 (High)

**Mandate.** Judge the realism, sequencing, and risk of the M7/M8 roadmap and the credibility of the
path to v1.0 and CNCF, calibrated against the project's documented history of optimistic labelling.

The forward roadmap is credible, well-sequenced, and — critically — calibrated by delivery, not
narrative. The M7 keystone (workflow data-flow, ADR-029) is genuinely done, and verifiable by
execution rather than assertion: ADR-029 is Accepted (`:6`, E5), the additive proto binding fields
exist (`workflow_compiler.proto:160,168`, E4), the engine-adapter interpreter threads a run-scoped
data context via `NewScopedWorkflowDataContext` / `ResolveInputs` / `WriteOutputs`
(`interpreter.go:66,131,152`, E2), real example workflows consume the bindings (E2), and EPIC #1167
plus all five stories (W.2–W.5, #1176–1179) are CLOSED on GitHub (E1). The milestone is ~95% complete
on live state (114 closed / 6 open, and 5 of those 6 are due-diligence meta-issues — only #1370 is an
open feature EPIC, E1), delivered at the exceptional, accelerating cadence documented in §7.4.

The reason this is a strong-7 and not a 9 is **the destination, not the engine**: the binding
constraint on v1.0/CNCF is social (≥2 cross-org maintainers, external security audit, TOC sponsor —
`ROADMAP.md:257-261`, E5), against a bus-factor-of-1 repository with zero named adopters. This is a
gate no amount of commit velocity can clear; the technical roadmap can land on time and v1.0/CNCF
still stalls. This finding converges with 5.21 (CNCF), 5.13 (Governance), and 5.24 (Repo Health) —
four agents, independently, on the same social ceiling.

The drift-test result is itself a headline. The prior-milestone optimism is real but
**since-corrected**: M3/M4 were labelled Complete then re-labelled Partial (§1.10 C1;
`current-milestone.md:17-23`), and M5 closed with five issues deferred to M6 (`M5-plan.md:9`). The
calibration verdict is that **the optimism pattern has inverted at M7** — the project moved from
*claim-ahead-of-delivery* (M3/M4) to *delivery-ahead-of-doc-update*. The current status-surface
drift runs in the *opposite* polarity to the old problem: canvas-W is still "Status: Aligned" with
its W.2–W.5 acceptance boxes unchecked (`canvas.md:9,92-122`), `M7-planning.md §4a` lists W.2–W.5 as
⬜ open, and `state/milestone.yaml:27` still lists already-closed EPICs — while all of that work is
in fact CLOSED and code-verified (E1/E2). Honest direction, but the same reconciliation-debt class
the Truth Pass exists to catch. A subsidiary drift: the v1.0 path is internally inconsistent —
`ROADMAP.md:56-59` now inserts M-UX + M-dx (four hops to v1.0) while `M7-planning.md:53` still
describes a three-milestone program, and the Version Plan table (`ROADMAP.md:265-275`) omits both —
so the true hop-count to v1.0 is ambiguous, a mild scope-expansion-without-clean-disclosure signal.

Scope discipline is otherwise strong: ADR-backed explicit non-goals (LLM framework, DAG workflows,
required SDK — `ROADMAP.md:296-300`, E5), a scope-creep risk rated High/High with a concrete
program-split mitigation (`M7-planning.md:496`, E5), and a correctly sequenced, dependency-aware
critical path (Q→W→C→X→T→D) that held in actual delivery (`M7-planning.md:426-459`, E5). Three
forward-commitment ADRs (031 context, 033 expert substrate, 034 manifest-id) remain Proposed though
their issues are closed — a tracked promote-on-alignment debt, not a blocking slip.

**Drift test.** "A prior milestone was labelled Complete but silently deferred scope" → **VERIFIED
historically, SINCE-CORRECTED** (optimism inverted at M7). "M7 keystone (ADR-029) is done" →
**VERIFIED**. "M7 is ~92% complete" → **VERIFIED** (live gh 114/6 ≈ 95%). "v1.0 path is the stated
single program M7→M-dx→M8" → **CONTRADICTED** (path grew; internally inconsistent).

**Sub-dimension scores (5.25).**

| Sub-dimension | Score | Confidence | Evidence |
|---|---|---|---|
| Keystone delivery — data-flow (ADR-029) | 9 | High | `ADR-029:6` Accepted; `workflow_compiler.proto:160,168`; `interpreter.go:66,131,152`; gh #1167/#1176-1179 CLOSED |
| M7 EPIC completion vs plan | 8 | High | gh 114 closed / 6 open (~95%); only #1370 an open feature EPIC |
| Sequencing realism | 8 | High | `M7-planning.md:459` critical path; waves `472-478` |
| Velocity vs ambition | 6 | Medium | 5.24 121→259→444 commits/mo vs social M8 gates |
| Roadmap honesty (non-goals, scope creep) | 7 | High | `ROADMAP.md:296-300`; `M7-planning.md:496`; version inconsistency `ROADMAP.md:56-59` vs `265-275` |
| Proposed-ADR backlog freshness | 7 | High | ADR-031/033/034 Proposed; `M7-planning.md:568-569` tracks promotion |
| Critical path to v1.0 / CNCF | 6 | High | `ROADMAP.md:257-261`; Part 1 §1.9 (social gate) |

---

## 7.6 Cross-cutting synthesis

Five agents, working independently, converge on three findings that should carry into the risk
register (§8) and investment thesis (§9).

1. **The binding constraint is social, confirmed from four angles.** Bus factor 1, zero named
   adopters, a missing `MAINTAINERS.md`, and a CNCF-backed rival (Kagent) on the identical
   positioning are flagged independently by Open Source (5.8), Governance (5.13), CNCF (5.21),
   Repository Health (5.24), and Future Roadmap (5.25). Every M8 / v1.0 checkbox is a community or
   governance item, not a feature. Velocity — which is exceptional — cannot buy it, and it is the
   slowest-moving variable in the plan. This is the single most material forward risk in the
   section.

2. **Two hard, cheap-to-fix governance gaps gate CNCF, and one is purely an execution oversight.**
   The missing top-level LICENSE file (a Critical CNCF blocker, a one-line fix) and the missing
   `MAINTAINERS.md` (#494, open) are independently surfaced by 5.8, 5.13, 5.21, and 5.24. Neither is
   a design problem; both are pure execution debt that any P0 sprint closes. The "gap is purely
   social" framing is therefore PARTIAL — one hard *legal* artifact is also absent.

3. **Honesty culture is a genuine, diligence-grade asset.** Across Open Source, Governance, and
   Roadmap, the project's own documents *under-state* rather than over-state reality: it reports
   "zero stars/forks/adopters" to its own face, re-labelled M3/M4 from Complete to Partial with
   reasons, advises "do not file [CNCF] until prerequisites are real," and — at M7 — has flipped from
   claim-ahead-of-delivery to delivery-ahead-of-doc-update. The data-flow keystone shipped and is
   execution-verifiable. This is the opposite of the §1.10 over-claiming history the framework was
   designed to catch, and it is a positive signal that materially de-risks the delivery-vs-narrative
   concern threaded through this diligence.

**Net read for this section.** The engineering and process substrate is well ahead of the community
substrate. The product can run on its beachhead, the discipline is real and verified, and the
honesty is a moat-grade asset — but the OSS/CNCF/sustainability posture is gated by a single
person and two missing files. The remediation cost for the two file gaps is trivial; the remediation
cost for bus factor 1 is recruitment, time, and a second organisation — the genuinely hard part, and
the one that no roadmap velocity will solve.

### Open questions / unknowns (section ledger)

- Is the missing LICENSE an oversight or deliberate, and do published GHCR images / the PyPI
  `zynax-sdk` wheel bundle a LICENSE internally? Not inspectable read-only; repo-root LICENSE is
  definitively absent.
- Who is the designated second maintainer / successor? `MAINTAINERS.md` is referenced repeatedly but
  absent (#494 open); no recruitment thread is visible in-repo.
- Is `required_signatures` / main-protection actually enforced server-side, or only configured?
  Branch-protection rule contents are not readable from the working tree; commit signatures are
  present but cryptographically UNKNOWN locally (key absent from trust store).
- Actual OpenSSF Scorecard numeric value (no network; badge-only, no committed `scorecard.yml`).
- Will the documented governance survive the founder leaving? The solo-maintainer phase, RFC process,
  and supermajority votes have never been exercised by a second human — robustness is asserted, not
  demonstrated.
- The true hop-count to v1.0 (3 vs 4 milestones) and any committed calendar dates for M-UX / M-dx /
  M8 — the timeline is sequence-only (CLAIMED), so "will v1.0 ship on time" is UNKNOWN by
  construction.
# 8. Risk

> **Scope of this section.** This section consolidates the diligence's risk view into a single
> investment-grade register, resolves the contradiction history against current HEAD, and names the
> concentration, key-person, and unknown-driven risks that bound the recommendation. It is a
> *synthesis*: every row, verdict, and count is traceable to the Wave A–D source packets
> (`docs/due-diligence/2026-06-20-dd-wave-a-findings.md`, `-b`, `-c`, `-d`) — no new claim is
> introduced here. Throughout, **VERIFIED** facts rest on executed proof or code/config/contract
> (evidence tiers E1–E4); **CLAIMED** facts rest only on first-party docs, README, or roadmap
> (E5–E6) and are labelled as such. The risk register is reproduced from the Wave D §5.23 packet in
> the Part 7 §7.3 format; the contradiction register reproduces the Part 4 orchestrator's
> current-HEAD verdicts. Per-agent full findings live in Appendix A.

The diligence's central, recurring finding is that **Zynax's gaps are enforcement-shaped and
social, not absence-shaped** — capabilities largely exist but are opt-in, soft, or single-authored.
That shape is what makes the risk profile *elevated but manageable*: the risks are unusually
**known, bounded, and honestly documented in-repo** (the project's own docs *under-state* reality —
the opposite of the §1.10 over-claim history), and **none is uninsurable, hidden, or an absolute
deal-breaker**. But four Critical risks each cap the maturity score until cleared, and they cluster
correlatedly around two roots — a **single human** and an **unfinished moat**.

---

## 8.1 Aggregate risk profile

**Counts (27 de-duplicated, clustered register rows): Critical = 4 · High = 9 · Medium = 11 · Low = 3.**

Per Part 7 §7.4, **any unmitigated Critical caps the overall profile at "High"**, and R1 (no
top-level `LICENSE` file) is unmitigated *as-shipped*. The aggregate profile label is therefore
**High** until R1 is resolved.

| Profile dimension | Value | Note |
|---|---|---|
| Critical | **4** | R1, R2, R3, R4 — all **condition-precedent**, each cheap or scoped to fix |
| High | **9** | R5–R13 — condition-precedent or price-adjustment class |
| Medium | **11** | R14–R24 — monitor + remediation plan |
| Low | **3** | R25–R27 — note only |
| **Total** | **27** | de-duplicated across all 26 diligence agents |
| Unmitigated Critical | **1** | R1 (missing LICENSE, as-shipped) → drives the "High" label |
| **Absolute deal-breakers** | **0** | every Critical is a cheap/scoped condition-precedent, not a viability or legality kill |

> *Source:* Wave D §5.23 aggregate (`docs/due-diligence/2026-06-20-dd-wave-d-findings.md`),
> de-duplicating red flags and open questions across the 23 Wave A/B/C packets.

The distinction between **0 deal-breakers** and **4 Criticals** is the most important framing in
this section. None of the Criticals threatens the asset's *existence* — each is a scoped,
remediable condition-precedent (the cheapest, R1, is a one-line commit). What they collectively do
is invoke the Part 8 §8.3 gating rules, which is why the actionable recommendation is held at
**Conditional / Watch** even though the raw confidence-weighted dimension mean (≈6.4) sits in the
Proceed-Conditional band. Per §8.3, a Critical is never averaged away. The Critical that most
threatens the *investment* — as distinct from the asset — is **R2, bus factor = 1**: it is the root
multiplier under the security, test-enforcement, governance, and doc-drift findings, and it is the
unbuyable-by-velocity gate for both CNCF graduation and any acquirer hand-off.

---

## 8.2 Full risk register (Part 7 §7.3 format)

> Severity-ordered, reproduced from the Wave D §5.23 register. Severity is derived from the Part 7
> §7.2 Likelihood × Impact matrix. **Class:** DB = deal-breaker, CP = condition-precedent, MON =
> monitor. Every Description is traceable to a cited Wave A/B/C agent + evidence; no row contains a
> new claim. The §5.23 packet scores risk on an *inverted* scale (high score = well-mitigated); the
> severities below are the externally-meaningful labels.

### Critical (4) — cap the maturity score until resolved (§8.3); all condition-precedent, none an absolute deal-breaker

| ID | Theme | Description | Sev | Likelihood | Impact | Detectability | Mitigation | Owner | Class | Source + evidence |
|----|-------|-------------|-----|-----------|--------|--------------|-----------|-------|-------|-------------------|
| **R1** | Legal / IP / License | No top-level `LICENSE` file in repo or git history despite Apache-2.0 intent (ADR-005, README badge, 347 SPDX headers). Apache §4 distribution defect; CNCF Sandbox hard requirement; broken README link; GOVERNANCE "done" overstated. | **Critical** | Almost certain (true now) | Severe (non-donatable; legal defect) | High (trivially observable) | **One-line fix** — commit `LICENSE` + `NOTICE`. UNMITIGATED as-shipped → drives the "High" aggregate (§7.4). | Maintainer | **CP** | 5.8 final.md:647-652; 5.22 final.md:1213-1218 |
| **R2** | People / Bus-factor | Bus factor = 1: one human = 772 commits, zero second human author; no `MAINTAINERS.md` (#494 open); `required_approving_review_count=0` so every PR self-merges with 0 human review; sole ADR Decider; entire prompt/canvas IP corpus single-authored. | **Critical** | Likely (single point of failure) | Severe (continuity + CNCF gate + IP custody) | High (`git shortlog`) | Recruit ≥2 cross-org maintainers; publish `MAINTAINERS.md`; require ≥1 review. Partial mitigant: exemplary docs/modularity ease a 90-day hand-off. | Founder / Board | **CP** | 5.24 final.md:1852-1857; 5.8 final.md:653-657; 5.22 final.md:1220-1224; 5.25 final.md:2083-2087 |
| **R3** | Operational / Enterprise | No enterprise identity layer: single shared static bearer key, binary authZ, no RBAC/SSO/OIDC (scoped-token step `pending`). No per-user identity to audit or revoke; cannot integrate a Fortune-500 IdP. | **Critical** | Almost certain (no IdP integrable today) | High (hard procurement blocker) | High (`auth.go:13-26`) | Funded RBAC/SSO/OIDC roadmap; deferred Post-M8 — must pull forward for enterprise GTM. | Eng lead | **CP** (enterprise GTM) | 5.20 final.md:1606-1611; 5.2 final.md:502-505 |
| **R4** | Operational / Enterprise | No multi-tenant isolation: namespace is metadata only; all workflows run in one shared Temporal namespace. Noisy-neighbour + data-bleed; "Kubernetes for AI workflows" lacks K8s's tenancy primitive. ADR-021 still PROPOSED. | **Critical** | Likely (any 2-tenant deploy) | High (regulated-buyer blocker) | Medium (`temporal.go:54-58`) | Namespace-scoped Temporal isolation + per-tenant quotas (ADR-021/022, "planned M7", not enforced). | Eng lead | **CP** (regulated buyers) | 5.20 final.md:1612-1616; 5.16 final.md:904-908 |

### High (9) — condition precedent or price adjustment

| ID | Theme | Description | Sev | Likelihood | Impact | Detectability | Mitigation | Owner | Class | Source + evidence |
|----|-------|-------------|-----|-----------|--------|--------------|-----------|-------|-------|-------------------|
| **R5** | Technical | Portability moat half-built: only Temporal interprets the IR; the Argo path serialises IR to JSON and hands it to a smoke-stub WorkflowTemplate that checks payload≠empty and exits 0 — capability-dispatch parity "deliberately out of scope". The headline "run on Temporal OR Argo without a rewrite" is CONTRADICTED at execution (contradicted by 5.4 alone, who conflates submission with interpretation; 7 agents to 1). | **High** | Almost certain (Argo can't dispatch) | High (kills the boldest moat if tested) | High (`argo-ir-interpreter.yaml:10-14`) | Finish Argo IRInterpreter + cross-engine parity test, OR re-label marketing precisely. | Eng lead | **CP** | 5.1 final.md:238-244; 5.3 final.md:182-188; 5.14 final.md:451; 5.26 final.md:1439-1443; 5.19 final.md:1355-1360 |
| **R6** | Market / Competitive | CNCF-Sandbox-backed Kagent (Solo.io) owns the identical "control plane for AI agents" framing on every buyer-visible axis (Web UI, CLI, MCP/OpenAPI discovery, multi-LLM, HITL, GitOps/ArgoCD) while Zynax has 0 stars/forks/named adopters; Restate + Dapr Agents crowd the adjacent durable-orchestration space. | **High** | Almost certain (rival winning distribution) | High (existential to the category thesis) | High (E7 + strategy.md:290) | Differentiate on the finished edge (compile-time IR validation), land adopters, co-exist via AgentService. | Founder | **CP/MON** | 5.19 final.md:1348-1354; 5.4 final.md:393-398 |
| **R7** | Technical / Operational | Two stateful SPOFs: Postgres single-instance StatefulSet (replicas:1, no in-chart failover/backups) + NATS JetStream single un-clustered node (Replicas:1). Compute scales (HPA on 5 svcs) but bottoms out on a non-HA shared substrate every workflow depends on. | **High** | Likely (any node/instance loss) | High (full data/event-tier outage) | Medium (`statefulset.yaml`; `nats.go:139-145`) | CloudNativePG migration (ADR-026, future) + clustered JetStream; add backup/DR runbooks. | SRE / Eng | **CP** (production) | 5.16 final.md:879-885; 5.20 final.md:1627-1631 |
| **R8** | Execution / Roadmap | Recurring delivery-vs-narrative drift (the §1.10 class the Truth-Pass exists to kill): Argo marketed as "Shipped"; "runnable" examples that hang from the CLI headlined "verified"; decorative OpenSSF Scorecard badge (live API 404); bench gate documented "live" but unwired/fail-open; README self-contradicts its own service table by ~2 milestones; competitive doc claims a non-existent "LangGraph engine". | **High** | Likely (recurs each milestone) | High (erodes the integrity asset; misleads buyers) | High (multiple agents) | Sweep with the project's own `/reconcile` Truth-Pass each milestone; tie marketing surfaces to HEAD. | Maintainer | **CP** | 5.3 final.md:189-201; 5.6 final.md:215-219; 5.22 final.md:1207-1212; 5.10 final.md:1416-1423; 5.19 final.md:1355-1360 |
| **R9** | Technical / Performance | Task-broker capability-dispatch fan-out is unbounded AND uncancellable: one goroutine per task, no pool/semaphore, on a detached context whose `Done()`/`Err()` return nil and `Deadline()`=zero; agent gRPC stream has no broker-side deadline + a new conn dialed per dispatch. A dispatch burst or hung agents → unbounded goroutine/memory growth; the 512Mi pod OOM-kills before memory-HPA averages up. | **High** | Possible (burst/hung-agent dependent) | High (control-plane OOM/leak, no cancellation) | Medium (`service.go:80-87,316-328`) | Worker pool / weighted semaphore + broker-side deadline from `task.TimeoutSeconds` + pooled conn. | Eng lead | **CP/MON** | 5.6 final.md:208-214; 5.16 cross-ref |
| **R10** | Operational / Enterprise | No audit log on mutating control-plane ops (apply/delete/publish-event); bearer + rate-limit only, no who-did-what record. Combined with the shared key (R3), actions cannot be attributed to a principal — fails SOC2 CC6/CC7 audit-trail expectations. | **High** | Almost certain (none exists) | High (enterprise/compliance blocker) | High (`handler.go:41-50`) | Add principal-attributed audit log (depends on R3 identity). | Eng lead | **CP** (enterprise) | 5.20 final.md:1617-1621 |
| **R11** | Operational / Enterprise | No production support/incident model: no `SUPPORT.md`, no `MAINTAINERS.md`, no on-call/escalation/SLA, bus factor 1. No entity a buyer can contract for a 2am outage (the SECURITY.md 48h SLA is vuln-disclosure only). | **High** | Almost certain (no model exists) | High (no operability assurance for buyers) | High (`ls` → absent) | `SUPPORT.md` + on-call/escalation/SLA; gated on R2 maintainer recruitment. | Founder | **CP** (enterprise) | 5.20 final.md:1622-1626 |
| **R12** | People / Execution | The binding v1.0/CNCF constraint is SOCIAL (≥2 cross-org maintainers + external security audit + filed TOC application) and is NOT solvable by the project's (excellent) velocity. The technical roadmap can land on time and v1.0/CNCF still stalls. | **High** | Likely (gate unmet today) | High (caps the CNCF/category thesis) | High (`ROADMAP.md:257-261`) | Maintainer recruitment + adopter acquisition + commission the security audit on a stated timeline. | Founder / Board | **CP/MON** | 5.25 final.md:2083-2087; 5.21 final.md:1850-1855 |
| **R13** | Market / IP | IP moat is thin and time-to-replicate is short: every candidate innovation reframes well-established prior art (Beam/Dapr/XState/Step-Functions/Envoy); the public IR/port/validator design is replicable by a funded rival in ~2 quarters; SPDD process-IP is being commoditized in real time (GitHub spec-kit, AWS Kiro). The durable edge is execution discipline (copyable), not protectable IP. | **High** | Likely (rivals already shipping) | High (no defensible moat for valuation) | Medium (E7 + ADRs) | Build ecosystem/adoption lead (the only non-copyable moat); lead with the finished validation edge. | Founder | **MON** | 5.26 final.md:1445-1449; 5.19 final.md:1367-1371; 5.4 final.md:404-408 |

### Medium (11) — monitor + remediation plan

| ID | Theme | Description | Sev | Likelihood | Impact | Detectability | Mitigation | Owner | Class | Source + evidence |
|----|-------|-------------|-----|-----------|--------|--------------|-----------|-------|-------|-------------------|
| **R14** | Security | mTLS fails OPEN: services fall back to `insecure.NewCredentials()` when cert paths unset, chart default is insecure, and the api-gateway + workflow-compiler PRODUCTION overlays omit `tlsSecretName` — plaintext gRPC possible even under "production". ADR-020:50 / SECURITY.md:89 overstate "mTLS enforced on all inter-service gRPC". | Medium | Possible (overlay/config dependent) | High (plaintext inter-service traffic) | High (`tlscreds.go:20`) | Fail-closed in prod namespaces; set `tlsSecretName` in all prod overlays; reconcile the ADR/SECURITY claim. | Eng / SRE | MON | 5.2 final.md:491-497; 5.20 final.md:1632-1634 |
| **R15** | Technical / Scalability | Umbrella default is NOT horizontally safe: task-broker & agent-registry fall back to in-memory mutex-maps when no DB DSN is wired, and the umbrella DEFAULT leaves `db.secretName` empty — scaling to >1 replica without the DSN gives each replica a disjoint state view (the failure ADR-021 exists to fix). Safe path is opt-in. | Medium | Possible (mis-config dependent) | High (silent split-brain state) | Medium (`values.yaml:46-48`) | Make DB DSN required / fail-closed in the umbrella default; document the footgun. | Eng | MON | 5.16 final.md:886-891 |
| **R16** | Security / Performance | No backpressure-aware autoscaling + no connection-pool tuning: HPA scales only on CPU/memory (blind to task/event backlog); `pgxpool` uses defaults (no `MaxConns`) so N replicas × default pool can exhaust the single Postgres's slots. No load test validates any of this. | Medium | Possible (under load) | Medium-High | Medium (`hpa.yaml`; `repository.go:33`) | Queue-depth custom-metric HPA + tuned `pgxpool` `MaxConns`/lifetime + first load test/SLOs. | Eng | MON | 5.16 final.md:898-903; 5.6 final.md:220-225 |
| **R17** | Execution / Performance | Bench regression gate implemented + baseline committed but NEVER wired into CI and fail-open even when run; architecture review markets it "live". A perf regression lands on main undetected; decorative guard = drift. | Medium | Likely (gate inactive) | Medium | High (grep → no workflow) | Wire `make bench`→`benchstat`→bench-gate into scheduled CI; flip `BENCH_GATE_ENFORCE`; stop claiming "live". | Eng | MON | 5.6 final.md:215-219; 5.7 final.md:978-980 |
| **R18** | Security / Testing | Test-enforcement gaps: domain-coverage gate re-checks only CHANGED services (unchanged can drift); engine-adapter (core execution) service-BDD hard-skipped; 2 of 4 integration suites excluded (event-bus, memory-service); e2e-smoke "required" satisfied by a no-op shim on most PRs; fuzz harness-only, no CodeQL `analyze` (SARIF-upload only). | Medium | Possible (silent regression) | Medium | High (workflow YAML) | Global coverage floor; real CodeQL `analyze`; wire engine-adapter BDD; close the 2 integration holes. | Eng | MON | 5.7 final.md:975-995; 5.22 final.md:1225-1231 |
| **R19** | Technical / Performance | No load/stress testing anywhere (no k6/vegeta/ghz/locust); the entire 10x/100x scaling story is inferential; Postgres pools at pgx defaults; EventBus throughput unmeasured; no pprof/profiling endpoints; benches cover only 2 of 7 services. | Medium | Likely (unknown break-points) | Medium | Medium (grep empty) | Add a minimal load harness + publish first SLOs + pprof endpoints. | Eng | MON | 5.6 final.md:220-225,230-234; 5.16 |
| **R20** | Market / Commercial | No network-effect flywheel + no bottom-up TAM/SAM/SOM: 0 community adapters, expansion vectors (template marketplace, Zynax Cloud) all roadmap-CLAIMED; portability pain not yet acute for single-engine buyers — "real but premature" market risk. | Medium | Possible | High (timing / category-sizing) | Medium (strategy.md qualitative only) | Author bottom-up SAM/SOM + ICP; land first community adapter + named adopter. | Founder | MON | 5.4 final.md:404-411; 5.26 final.md:1450-1451 |
| **R21** | Operational / Enterprise | Doc-vs-artifact drift on the enterprise security surface: SECURITY.md claims arm64 service images (false — amd64-only) and mTLS shipped on all services (fails-open). These overstated controls are exactly what an enterprise review probes and finds wanting. | Medium | Likely (questionnaire probe) | Medium | High (`SECURITY.md:44,89`) | Reconcile SECURITY.md to HEAD; ship multi-arch service images on the live release path. | Maintainer | MON | 5.20 final.md:1632-1634; 5.9 final.md:1213-1218 |
| **R22** | Technical / DevOps | Production service & adapter container images are amd64-only (pre-merge build = linux/amd64; release = retag-only; the multi-arch `service-release.yml` is `workflow_dispatch`-only). arm64 K8s nodes + Apple-Silicon local Docker cannot run official service images. | Medium | Likely (arm64 environments) | Medium | High (`ci.yml:861`) | Move multi-arch service build onto the live release path. | Eng / CI | MON | 5.9 final.md:1213-1218 |
| **R23** | Technical / Architecture | Layer-boundary / hexagonal invariant is convention-only — no depguard/import-linter/fitness function — yet `services/AGENTS.md:22,46` claim it is "CI-enforced". One careless import silently re-introduces coupling; the onboarding contract itself carries the drift. (Mitigant: 11 separate Go modules make cross-service `internal/` coupling mechanically impossible.) | Medium | Possible (silent regression) | Medium | Medium (`golangci-lint.yml`; grep empty) | Add an import-boundary linter; correct the AGENTS.md "CI-enforced" claim. | Eng | MON | 5.15 final.md:665-670; 5.1 final.md:245-249 |
| **R24** | Governance | Governance docs misrepresent reality: GOVERNANCE.md "no single company controls" / "Apache license done" contradicted; `MAINTAINERS.md` referenced in 4 places but absent; Milestone→Version table materially stale; RFC process (required for proto/arch changes) has 0 RFCs ever filed — load-bearing gates never exercised. ADR-vs-code reconciliation debt (ADR-034 Proposed vs shipped idempotent apply; INDEX mislabels ADR-023). | Medium | Likely (charter drift now) | Medium | High (`GOVERNANCE.md`) | Truth-Pass the charter; create `MAINTAINERS.md` (#494); reconcile ADRs/INDEX to HEAD. | Maintainer | MON | 5.13 final.md:1112-1131; 5.8 final.md:660-665; 5.1 final.md:250-254 |

### Low (3) — note only

| ID | Theme | Description | Sev | Likelihood | Impact | Detectability | Mitigation | Owner | Class | Source + evidence |
|----|-------|-------------|-----|-----------|--------|--------------|-----------|-------|-------|-------------------|
| **R25** | Security | Two named dependency CVEs suppressed (pip-audit PYSEC-2026-196; trivy DS002 root-tools-image) — both documented + time-boxed (DS002 accepted-until 2026-11-01), but standing exceptions to blocking gates. | Low | Possible | Low | High (`.trivyignore`; `ci.yml:730`) | Re-evaluate at the dated deadlines; keep dated. | Eng | MON | 5.2 final.md:506-509; 5.14 final.md:419 |
| **R26** | Security / Network | NetworkPolicy ingress is port-scoped, not source-scoped (no `from` selector) — any pod may dial the service ports; not true zero-trust default-deny ingress. (Runtime enforcement also depends on a CNI that honors it.) | Low | Possible | Low-Medium | Medium (`networkpolicy.yaml:15`) | Add source selectors / default-deny ingress. | SRE | MON | 5.2 final.md:498-501 |
| **R27** | Documentation / DX | First-impression surfaces lag honest docs: README hero asciinema cast is a PLACEHOLDER; "make demo one command" needs 4 prereqs (CLI on PATH, ollama, pulled model) and exits if the CLI isn't pre-installed; CLI default URL (8080) differs from the stack port (7080). Cosmetic-trust + Day-0 conversion harm, not functional. | Low | Likely (first-run) | Low | High (`README.md:24-41`) | Record the cast; make demo install/check prereqs; document the port. | Maintainer | MON | 5.11 final.md:877-883; 5.3 final.md:196-201; 5.10 final.md:1424-1428 |

**Register total: Critical = 4 · High = 9 · Medium = 11 · Low = 3 · Total = 27.** Per §7.4, the ≥1
unmitigated Critical (R1, as-shipped) sets the overall profile label to **"High"** until resolved.
All four Criticals are condition-precedent (cheap/scoped); the register contains **0 absolute
deal-breakers**.

---

## 8.3 Contradiction register & resolutions

> The Part 1 §1.10 drift history makes the contradiction register a load-bearing diligence artifact:
> every bold claim was re-tested against current HEAD (`main` @ `e3135a6`) under the evidence
> hierarchy **E1 > E2 > E3 > E4 > E5 > E6** (executed proof and code beat docs and README). The
> orchestrator closed two registers (C5, C6) itself via read-only `grep` because no single agent
> owned them. **C1–C8 tally: 5 VERIFIED (C3, C5, C6, C7, C8) · 2 PARTIAL (C2, C4) · 1
> CONTRADICTED-then-corrected (C1); C4 also carries an UNKNOWN residual on GHCR signature presence.**

| # | Claim | HEAD verdict | Evidence + source (tier) |
|---|-------|:---:|---|
| **C1** | "M3 & M4 Complete" | **CONTRADICTED → RESOLVED-honest** | Docs now re-label M3/M4 *Partial* with reasons in the canonical state file; the original over-claim is corrected and docs now *under-state* reality (5.13/5.25, E5+E2). Original claim false; current HEAD honest. |
| **C2** | "mTLS between all services" | **PARTIAL (fails open)** | mTLS code exists but every service falls back to `insecure.NewCredentials()` when certs unset; 2 prod overlays omit `tlsSecretName` (5.2 `tlscreds.go:21` — orchestrator-confirmed across 5 services; E2/E3). Configurable, **not enforced**; ADR-020/SECURITY.md overstate "enforced". |
| **C3** | "SBOM per release" | **VERIFIED** | syft SPDX per-service digest attached to each Release (5.2/5.9 `release.yml:527`; E3). |
| **C4** | "cosign-signed images" | **PARTIAL (UNKNOWN at GHCR)** | cosign + SLSA provenance wired in `release.yml` (E3), but signature *existence* on published GHCR images could not be verified offline (no registry/cosign access). Configured; presence UNKNOWN (5.2/5.9). |
| **C5** | "CloudEvents publishing (NATS)" | **VERIFIED** | event-bus ships a real NATS JetStream client (`nats.go:27-43`), a `CloudEvent` domain type + JSON wire envelope (`nats.go:53`), a `cloudevents.proto` contract, and a `Publish` RPC (`handler.go:36-71`) — orchestrator grep (E2/E4). Not a log-stub at HEAD. |
| **C6** | "agent-registry implemented" | **VERIFIED** | `services/agent-registry` exists with domain/service/handler + **both** in-memory and Postgres repositories (`postgres/repository.go`) — orchestrator grep (E2). Delivered M5.C, Postgres-backed M6 as claimed. |
| **C7** | "stateless workflow-compiler" | **VERIFIED (fixed, stronger than claimed)** | The unbounded in-memory map was removed entirely; only a stale proto comment remains (5.14 `workflow_compiler.proto:50-53`; E2). |
| **C8** | "cel-go guard evaluator (fail-closed)" | **VERIFIED** | `cel-go` v0.28.1 drives `evalGuard`, fail-closed (5.14 `interpreter.go:203,220-259`; E2). The bespoke fail-open evaluator was replaced. |

The pattern across C1–C8 is reassuring: of the eight historical drift claims, five are now fully
VERIFIED by code, three are honest corrections or precisely-bounded partials, and the two partials
(C2 mTLS, C4 cosign) are *configured but not proven enforced/present* rather than absent — they map
directly onto register rows R14 and the U1 unknown. C1 is the clearest signal of the Truth-Pass
culture: the docs now under-state reality, the inverse of the §1.10 over-claim that motivated the
whole register.

### C-ARGO — the headline portability moat (cross-agent conflict)

The single material *cross-agent* conflict the orchestrator had to adjudicate concerns the central
thesis claim: **is the engine-agnostic portability moat real?** One agent (5.4 Market) read
`argo_engine.go` as "real at HEAD" (~314 LoC) and judged the "stubbed" doc to lag reality. Seven
zone-credible agents (5.1 Architecture — the zone owner — plus 5.3, 5.6, 5.14, 5.19, 5.26) cited
the *execution path*: the Argo path serialises the IR to JSON and hands it to a smoke-stub
WorkflowTemplate that asserts `payload != empty` and exits 0; `argo_engine.go:62-98` **never calls
`IRInterpreter.Run`**, and capability dispatch is "deliberately out of scope"
(`scripts/e2e/manifests/argo-ir-interpreter.yaml:10-14`; `e2e-argo.sh:232-266`, CR-phase only).

**Resolution: Position B holds (7:1).** By the evidence hierarchy, B cites the execution path
(E1/E2 — the interpreter is never invoked) while A cites only file presence/LoC and conflates
**submission** (real) with **interpretation** (stub). The verdict: portability is **REAL at the
IR / contract / submission boundary, CONTRADICTED at the capability-dispatch execution boundary** —
i.e. **the moat is half-built**. A buyer who tests "run on Argo" gets a workflow that reports
Success without running anything. This is a Part 8 §8.3 *contradicted-core-thesis* trigger, which
independently pulls the recommendation floor toward Pass (Revisit) for any portability-dependent
deal structure until parity ships or the claim is re-labelled. The **residual** — the true
engineering cost to bring Argo (or any second engine) to `IRInterpreter` parity (a one-quarter
sidecar vs a structural re-implementation) — is **UNKNOWN** and is the single number that governs
whether the headline thesis is fixable before a funded fast-follower replicates the public IR/port
design. It is routed to risk **R5** and condition-precedent **CP-4** (unknown **U4**).

### Other cross-agent conflicts resolved

| Conflict | Positions | Resolution | Becomes |
|---|---|---|---|
| **C-LICENSE** | 5.8/5.21/5.22 (all High) "no LICENSE" vs README badge + GOVERNANCE "Apache done" | No LICENSE in tree or history (`git log --all -- LICENSE` empty — orchestrator-confirmed); badge/GOVERNANCE are E6 over-claims → **CONTRADICTED** | **R1 (Critical)** |
| **C-COMPLEMENTARY** | Kagent "complementary" claim vs `grep kagent` (excl. docs) → 0 code hits (5.4/5.19/5.18) | **PARTIAL** — a defensive prose hedge, not shipped integration; mechanism (`agent.proto`) is real | risk context for R6 |
| **C-BENCH** | architecture-review markets the bench gate "live" vs no workflow invokes it; fail-open if run (5.6) | **CONTRADICTED** — decorative gate | **R17 (Medium)** |

---

## 8.4 Concentration & key-person risk

The register's most important structural property is that the risks are **correlated, not
independent** — they cluster around a small number of roots, and the dominant root is a single
person.

**Primary concentration — bus factor = 1 (R2, Critical).** One human identity authored 772 of
~772 non-bot commits (5.24 `git shortlog` — orchestrator-confirmed), is the sole ADR Decider, owns
the entire prompt/canvas IP corpus, and — because `required_approving_review_count=0` — merges every
change with zero human review. That single identity is the single point of failure for code,
security judgement, CI custody (`BOT_GITHUB_TOKEN` is the sole bypass actor), governance, docs, and
the methodology IP. R2 is the explicit **root multiplier** under R3 (identity), R5 (single
interpreter), R12 (the social CNCF gate), R18 (test-enforcement gaps), and R24 (governance
misrepresentation): every one of those gaps exists in part because there is no second pair of eyes.
It is also the *unbuyable-by-velocity* gate — the project's (excellent) engineering throughput
cannot manufacture a second cross-org maintainer, an external audit sign-off, or IP custody
diversity.

R2 is a **condition-precedent, not a deal-breaker**, for one concrete reason: the exceptional
documentation and the build-system-enforced modularity (11 separate Go modules make cross-service
coupling mechanically impossible) make a **90-day hand-off feasible**. The mitigation is to recruit
≥2 cross-org maintainers, publish `MAINTAINERS.md` (#494), and require ≥1 human review on merge
(CP-2). Under Part 8 §8.3, bus-factor-1 with no succession plan **caps the acquisition-readiness
sub-verdict at Conditional** and makes key-person retention + IP-assignment the load-bearing
diligence item for any non-acqui-hire structure.

**Secondary concentration — single-engine reality (R5, High).** The entire portability thesis rests
on one interpreter (Temporal); the second engine (Argo) is structurally wired but functionally a
stub. Until cross-engine parity ships, the moat is a single-engine moat in fact even though it is
multi-engine in design.

**Tertiary concentration — the non-HA stateful tier (R7, High).** Stateless compute scales (HPA on
five services), but every workflow depends on a single-instance Postgres StatefulSet and a
single-node JetStream — the shared chokepoint beneath an otherwise-scalable plane, with the umbrella
default silently reverting to in-memory state (R15) compounding the footgun.

The mitigating fact across all three is the green-flag finding that **the risks are known, bounded,
and honestly documented in-repo** (5.8 honest self-assessment; 5.14 debt tracked, dated,
issue-tagged; 5.25 drift now under-states reality). No uninsurable or hidden risk surfaced, and most
Criticals are one-PR fixes — which is precisely why the profile is *elevated but manageable* rather
than fatal.

---

## 8.5 Unknowns & assumptions ledger

> What diligence could **not** verify at HEAD — the honest boundary of this assessment (framework
> §6.3: the unknowns ledger must be non-empty and honest). Each is a confirmatory-session item for
> management/technical follow-up; none is asserted as a finding.

| # | Unknown / assumption | Why unresolved | Confirmatory-session ask |
|---|----------------------|----------------|--------------------------|
| **U1** | GHCR image signature / SLSA-attestation **existence** (C4 residual) | cosign unrunnable offline; no registry access. Signing is CONFIGURED, presence UNKNOWN. | Run `cosign verify` against a published GHCR digest live. |
| **U2** | Live `make demo` runtime success + wall-clock (<15-min hero claim) | Static-only audit; no Docker/Ollama in the diligence env. | Run end-to-end on a clean machine; time it; run the stateful path twice. |
| **U3** | Argo CI-leg green/red + live bench-gate behaviour at HEAD | Not observable offline. | Show the latest CI runs; demonstrate the bench gate failing a planted regression. |
| **U4** | True eng-cost to bring Argo (or any 2nd engine) to `IRInterpreter` parity (C-ARGO residual) | The single number governing the headline thesis; not derivable from code. | Sidecar/operator running the existing `IRInterpreter` in-cluster vs full DAG re-impl — scope + timeline. |
| **U5** | Real multi-service per-step latency + 10x/100x break-points | Zero load tests anywhere; the scaling story is inferential. | Provide a load harness + SLOs, or commit to producing them as a CP. |
| **U6** | Live GitHub stars/forks/named adopters + post-May traction; bottom-up TAM/SAM/SOM | Out of repo scope; the 0/0/0 baseline is a May-doc CLAIM, not re-pulled. | Any private design-partner/pilot? A defensible SAM/SOM? Is portability a top-3 buying criterion for any segment today? |
| **U7** | Prod Helm DB topology (separate Postgres instances vs separate DBs in one instance) | Not confirmed in `infra/helm`. | Show the production Helm values + DB HA/backup posture. |
| **U8** | Maintainer's openness to a co-maintainer / key-person + IP-assignment arrangement | A social fact, not in the repo; the gate for both CNCF and any non-acqui-hire structure. | Direct conversation — willingness, timeline, IP custody. |
| **U9** | Monetization-vs-CNCF model decision (open-core Scenario A conflicts with CNCF neutrality) | The strategy doc defers the decision. | Which model, and is it decided before or after a CNCF bid? (A one-way-door ADR.) |
| **A1** | Dollar valuation is **assumption-based** (E7 comps + explicit model), not derivable from in-repo financials | No revenue, no priced round. | Treat the ~$2.0–4.5M IP floor / ~$4–8M post-seed-if-triggers-move as a frame, not a quote; ±40% eng-years sensitivity. |

The two unknowns that most move the recommendation are **U4** (the cost to finish the portability
moat — it governs the §8.3 contradicted-core-thesis cap) and **U8** (the maintainer's willingness to
add a co-maintainer — it governs the bus-factor cap and the CNCF path). Both are *social/scoping*
questions answerable in a single confirmatory session, not deep technical unknowns — consistent with
this section's through-line that Zynax's binding risks are enforcement-shaped and social rather than
absence-shaped.

---

## 8.6 Conditions precedent (risk-driven)

The four Criticals and the contradicted core thesis map onto a small, ordered set of
conditions-precedent. They are reproduced here in risk terms; the financial/deal framing is in
Section 9.

| CP | Resolves | Action | Timing |
|---|---|---|---|
| **CP-1** | R1 | Commit a top-level Apache-2.0 `LICENSE` (+ `NOTICE`); reconcile the OpenSSF badge. | Before close (one-line fix) |
| **CP-2** | R2 | Credible maintainer-succession + bus-factor plan: ≥2 cross-org maintainers, `MAINTAINERS.md` (#494), require ≥1 human review on merge. | Before close or price/earnout |
| **CP-3** | R3/R4/R10 | Funded plan + timeline for enterprise identity (RBAC/SSO/OIDC + principal-attributed audit) and multi-tenant isolation. | CP for any enterprise GTM |
| **CP-4** | R5/R8 | Ship Argo IR-interpretation parity (+ a cross-engine parity test) OR re-price/re-label on "engine-neutral IR, Temporal reference interpreter". | §8.3 contradicted-thesis cap |
| **CP-5** | R6/R20 | Traction trigger: ≥1 named pilot/adopter or first community adapter in motion. | Monitor → de-risking trigger |
| **CP-6** | R7/R9/R16/R19 | (Managed-service only) close the stateful SPOFs, bound the task-broker fan-out, publish a load-test/SLO baseline before any SLA-bearing commercialization. | Before SLA-bearing GTM |

**Net risk verdict.** Risk profile **High** (driven by one unmitigated Critical, R1), but **0
absolute deal-breakers** and a register that is known, bounded, and honestly self-documented. The
four Criticals are condition-precedent and largely cheap to clear; the dominant correlated root is
bus-factor-1 (R2). Under the Part 8 §8.3 gating rules, the unmitigated Criticals cap the ceiling at
Conditional/Watch, the contradicted portability thesis pulls the floor toward Pass (Revisit) for any
portability-dependent structure, and bus-factor-1 caps the acquisition-readiness sub-verdict — the
risk basis for the overall **Conditional / Watch** recommendation carried in Sections 1 and 9.
# 9. Financial & Investment

> **Section verdict.** Zynax is a high-quality engineering *asset* wrapped around a business
> that does not yet exist. The replacement-cost floor is real and execution-verified
> (~8–14 engineering-years embodied, ~$2.0M–4.5M build-cost floor); the going-concern value
> is gated almost entirely on facts the diligence could **not** verify in-repo — distribution,
> a second maintainer, a finished portability moat. The financial dimension (D13) scores
> **5.0/10 (Medium)**; acquisition-readiness (D15) scores **5.0/10 (Medium-High)**. The
> Part 8 recommendation is **Conditional / Watch (confidence: Medium)**, best-fit structure
> **acqui-hire (primary)**.
>
> *Scope & evidence note.* This section synthesises the Wave D §5.17 Investment and §5.18
> Business-Strategy packets plus the Part 4 orchestrator verdict. Every figure here is
> **assumption-based valuation framing** (E7 comparables + an explicit model), **not a quote** —
> there is no revenue, no committed pipeline, and no priced round to anchor a market price.
> Per the evidence taxonomy (Part 2 §2.4), VERIFIED (E1–E4) facts are kept strictly separate
> from CLAIMED (E5–E6/roadmap) ones throughout. Full source packets:
> `docs/due-diligence/2026-06-20-dd-wave-d-findings.md` (and Waves A/B/C), Appendix A.

---

## 9.1 Cost-to-Build — Replacement Cost (§5.17)

The single most defensible financial fact in this diligence is the **replacement cost** of the
embodied asset, because it rests on measured repository scale at the audited HEAD
(`main` @ `e3135a6`) and on an *execution-verified* quality bar — not on narrative. Sub-dimension
score: **7/10 (Medium)** — the highest sub-score in the entire financial picture.

The embodied IP, measured by commands run during the audit wave:

| Asset | Scale signal | Eng-effort basis |
|-------|--------------|------------------|
| 7 Go control-plane services | 18,631 non-test Go LOC + **30,887 test LOC** (1.66× test:code) | hexagonal design + ≥90% coverage bar ≈ 2–3× naive LOC effort |
| Python SDK + 5 adapters | 21,089 Python LOC (ci / git / http / langgraph / llm) | cross-language adapter-first contract, typed |
| gRPC contracts | 9 protos / 1,829 LOC, `buf` breaking gate, 18 features / **306 BDD scenarios** | contract-first BDD is high-effort-per-line |
| Decision + doc surface | **37 ADRs**, 208 markdown docs | architecture reasoning embodied, not just code |
| CI / supply chain | 21 workflows / 4,620 LOC; cosign + SBOM + SLSA; coverage/lint gates | the rigor that makes the rest trustworthy |
| Deploy | 9 Helm charts (HPA / PDB / NetworkPolicy / hardening) | production-shaped K8s packaging |
| Velocity proxy | 824 commits over ~2 months (2026-04-20 → 2026-06-20), 769 by one human | exceptional solo throughput |

**The model and its explicit assumptions.** The estimate is built on roughly 40k hand-written
non-test LOC (Go + Python) plus 31k test LOC, 306 BDD scenarios, 37 ADRs, and a full
supply-chain CI surface. The load-bearing assumption is that a replicating team must clear the
**same** verified quality bar — coverage proven at 92.1–100% across all seven domains
(Wave A §5.7), `golangci-lint` clean across 14 linters with no blanket suppression (§5.5), and
the cosign + SBOM + SLSA supply-chain trifecta (§5.2). Matching that bar roughly *doubles* a
naive LOC-only estimate. Applying an industry rule-of-thumb (E7, COCOMO-class productivity of
~3–6k quality LOC per engineer-year for contract-first, fully-tested, hardened infrastructure)
yields:

> **Estimate: 8–14 engineering-years** to faithfully replicate the embodied asset at this
> quality, central case ~10–11 eng-years, with **±40% sensitivity** depending on whether the
> replicator must match the same coverage/supply-chain bar.

A critical interpretive note carries straight from the source packet: the actual *solo, ~2-month*
build is an **outlier productivity signal, not a contradiction** of the 8–14 eng-year figure. The
exceptional velocity compresses calendar time; it does not reduce the embodied effort a normally-staffed
team would expend to reach the same place. Drift test on the boldest cost claim —
*"replacement cost is high enough to anchor a real valuation"* → **VERIFIED**, because the scale
is measured and the quality is execution-proven rather than asserted.

---

## 9.2 Cost-to-Maintain & Sustain (§5.17)

Sub-dimension score: **5/10 (Medium)**. The defining finding is a sharp asymmetry between
**code-maintenance cost (low)** and **going-concern cost (people-dominated, structurally high)**.

- **Code load is low.** Near-zero technical debt — 5 raw debt markers, only 2 genuine; 215
  *scoped* `//nolint` directives and zero bare ones; best-in-class Renovate (grouped, digest-pinned);
  dated CVE suppressions that carry re-evaluate dates (Wave B §5.14, scored 8 on debt). Build-system-enforced
  modularity across **11 separate Go modules** makes cross-service `internal/` coupling mechanically
  impossible, holding long-run maintenance drag down and making the asset cheap to transplant (§5.15).
- **Infra cost is modest.** The compute tier is stateless; the stateful tier is Postgres + NATS.
- **People cost is the real expense, and it is unbounded by one person.** Bus factor = 1 (769 of
  824 commits from a single human identity, via `git shortlog`); `MAINTAINERS.md` is absent
  (issue #494). A single maintainer cannot simultaneously sustain 7 services + 5 adapters + a CNCF
  community — a credible going-concern needs **2–3 FTE**. This is a *people* cost, not a code cost
  (Wave A §5.24; Wave C §5.8/§5.13).
- **Cost-to-make-sellable is elevated by the stateful tier.** Non-HA Postgres (`replicas: 1`) plus
  single-node JetStream are SPOFs (Wave B §5.16); making them production-grade is a prerequisite
  expense for any SLA-bearing managed plan.

---

## 9.3 Commercialization Paths, Prerequisites & Timing (§5.17)

Commercialization paths score **5/10 (Medium)**; monetization *timing* scores **4/10 (Medium)** —
the lowest sub-score, reflecting that the project is **pre-proof**. The vectors are clear and the
architecture supports them, but the highest-value vector is blocked on a missing identity layer and
*every* vector is gated on traction that is zero today.

| Vector | Status | Gating prerequisite |
|--------|--------|---------------------|
| Zynax Cloud (managed control plane) | roadmap-CLAIMED (`ROADMAP.md:233`) | close stateful SPOFs + bound task-broker fan-out + publish load-test/SLO baseline (§5.16/§5.6) |
| Enterprise add-ons (RBAC/SSO/audit) | **unbuildable today** | no enterprise identity layer — single static bearer key, no RBAC/SSO/OIDC, scoped-token authz a pending stub (§5.20, `auth.go:13-26`) |
| Usage metering / quotas | **partial substrate SHIPPED** | PolicyGate + QuotaChecker + per-IP rate-limit real and tested (§5.20, `quota_check_test.go:35`, `ratelimit.go:17-69`) |
| Template / adapter marketplace | roadmap-CLAIMED (`ROADMAP.md:213`) | flywheel is 0-node — zero community-built adapters (§5.4) |
| Managed-trial funnel | conversion artifact EXISTS | genuinely runnable zero-secret Temporal hero path (`make demo`) (§5.3) |

**Monetization timing — the binding finding.** Monetization is premature until two things are
true: (a) a traction signal exists (≥1 named adopter, non-zero community) and (b) the enterprise
identity layer ships. Today there is zero traction, zero adopters, and a CNCF-Sandbox-backed rival
(Kagent) on identical positioning. The proof milestone is **distribution** — and the binding v1.0/CNCF
constraint is *social* (≥2 cross-org maintainers, an external audit, a TOC sponsor), which the
project's excellent technical velocity **cannot buy** (Wave C §5.25, "unbuyable by velocity").

Drift test on the commercialization thesis — *"there is a defensible, monetizable moat TODAY"* →
**CONTRADICTED.** Only Temporal interprets the IR; the Argo path serialises the IR to JSON and hands
it to a smoke-stub `WorkflowTemplate` that asserts payload ≠ empty and exits 0 — `argo_engine.go:62-98`
never calls `IRInterpreter.Run` (Wave A §5.1, High; `argo-ir-interpreter.yaml:10-14`). A buyer who
tests the differentiating claim on Argo gets a workflow that reports *Success* without running anything.
This is a §8.3 contradicted-core-thesis trigger. A second drift claim —
*"the project is investable/donatable as-is"* → **CONTRADICTED**: no top-level LICENSE file
(`git log --all -- LICENSE` empty), `MAINTAINERS.md` absent, OpenSSF badge renders "no data" (§5.22/§5.8/§5.21).
These are cheap-to-fix oversights but hard conditions precedent for any license-clean or donatable structure.

---

## 9.4 Risk-Adjusted Thesis & Valuation Framing (§5.17)

The risk-adjusted thesis scores **4/10 (Medium)**. The investment risk is **concentrated and
largely social / distribution, not technical**:

- **Bull case** — a best-in-class engineering substrate plus a genuine (if partial) IR-portability
  moat plus a rare honesty culture, all cheaply convertible into a fundable wedge *if* distribution
  moves. Grounded in VERIFIED evidence: supply-chain trifecta (§5.2), ≥90% coverage gate + 306 BDD
  scenarios (§5.5/§5.7), a genuinely engine-neutral IR contract (§5.1).
- **Bear case** — a CNCF-backed rival wins distribution while the headline portability moat is
  functionally unbuilt on the second engine and the team is one person. Grounded in §5.1 (Argo
  execution contradicted), §5.24 (bus factor 1), §5.19/§5.4 (Kagent out-positioning at 0 adopters).

### Valuation framing (assumption-based, E7 comps — **NOT a quote**)

> Every figure below is a *frame*, not a market price. There is no revenue and no priced round.
> Treat the eng-years model as carrying **±40% sensitivity**, and treat the dollar figures as
> assumption-driven (A1 in the unknowns ledger).

| Lens | Frame | Basis & key assumption |
|------|-------|------------------------|
| **Cost-to-replace floor** | **~$2.0M–4.5M** embodied build cost | 8–14 eng-years × fully-loaded ~$250–350k/eng-year (E7 infra-eng comp). A floor on IP value for an asset/acqui-hire lens, not a market price. |
| **Pre-traction OSS-infra seed comp** | **~$4M–8M post** on a small seed | E7: early infra/devtools seed rounds with strong tech + zero revenue typically price ~$1M–3M raised at ~$6M–15M post; Zynax's eng quality supports the *upper-tech* end, but zero distribution + single maintainer + contested category pull it to the *lower* band — and only if the two de-risking triggers are credibly in motion. |
| **Acqui-hire lens (most-likely best fit)** | embodied-IP floor (~$2–4.5M) **+ key-person premium** | The structure where the contradicted portability thesis and zero traction hurt *least*: the acquirer buys substrate + builder into its own distribution. |

**The assumption that swings everything:** all figures assume the **0/0/0 adopter baseline holds**.
A single credible enterprise pilot or a co-maintainer materially re-rates the seed case upward;
their continued absence pushes the call toward asset/acqui-hire or Pass.

---

## 9.5 Business Strategy (§5.18) — Model, GTM, Distribution, Narrative

Overall §5.18 score: **5/10 (Medium)** — *a coherent, unusually honest strategy with the right
instincts, wrapped around a business that does not yet exist.* The category is real, the wedge is
legible, the sequencing is correct, and the platform thesis is genuine and partially proven; but
every load-bearing element of the *business* — revenue, a named adopter, a community, a second
maintainer, a finished moat — is absent or aspirational.

| Sub-dimension | Score | Confidence | Key evidence |
|---------------|:-----:|:----------:|--------------|
| Business-model fit & timing | 6 | Medium | `strategy.md:357-391`; policy+rate-limit shipped (§5.20) but enterprise checkboxes absent |
| GTM motion & beachhead→expansion | 6 | Medium | `strategy.md:233-262`; only 2 of ~13 examples run to completion (§5.3) |
| Distribution strategy | 4 | High | `Makefile:162-204` (Day-0 layer shipped) vs `README.md:37` asciinema PLACEHOLDER |
| Partnership posture | 5 | Medium | `strategy.md:200-204`; Kagent "complementary" prose-only, 0 code (§5.19) |
| Strategic-narrative coherence | 6 | Medium | `strategy.md:43-130`; lead wedge half-built, rare-edge under-led (§5.19) |

**Business-model recommendation.** Stay on the **CNCF-donated neutral core + services / managed
add-ons** path through M8, preserving Scenario-A (open-core + Zynax Cloud) optionality but *not*
exercising it. The enterprise add-on surface (RBAC/SSO/multi-tenant/audit) is unbuilt (§5.20,
scored 4/10), and feature-gating a 0-adopter product only suppresses the adoption the business
needs. The CNCF-neutrality vs single-vendor-revenue tension (`strategy.md:376-391`) is real and
should be resolved in a **one-way-door ADR before any scaling**, so open-core feature-gating cannot
foreclose the CNCF bid.

**GTM wedge.** Bottom-up, developer-led; beachhead = **agentic software-engineering automation**
(code-review / CI) — the correct motion and the only segment with shipped runnable proof. But the
wedge is **mis-aimed**: messaging leads with multi-engine portability (half-built — Argo cannot
interpret the IR) instead of the genuinely-shipped, rare, hard-to-copy edge — **compile-time
structural IR validation** (`structural.go:11-58`) plus the no-SDK AgentService capability contract
and GitOps. Leading with a contradicted claim against a CNCF-backed rival is the worst footing and
re-introduces the exact delivery-vs-narrative drift the project's own Truth-Pass culture exists to
catch.

**Distribution & partnership posture.** Distribution is correctly named as the #1 lever
("distribution, not design"), and the *cheap* Day-0 layer is genuinely shipped — one-command
`make demo` (SCENARIO/PR/STREAM modes), a zero-secret Ollama overlay, an `EVAL_TEMPORAL` lightweight
engine, a README-first runnable demo, 13 example workflows + 3 templates, and a good-first-issue lane
(`Makefile:162-204`, `README.md:24-33`). But the **high-leverage** distribution machinery the
strategy itself names is aspirational: the README headline asciinema cast — the named conversion
lever — is a literal **PLACEHOLDER** (`README.md:37`); reusable-templates (#1171) and hosted-playground
(#1389) are unchecked; and there are no MAINTAINERS/SUPPORT/ADOPTERS files, no community cadence,
0 stars/forks/adopters, 0 community adapters. **Net (drift test → PARTIAL): the project is executing
the easy 10% of its own distribution thesis; the thesis outruns the investment.** On partnerships,
"wrap-the-engines" is an ally posture in principle, but Temporal is a hard runtime dependency, a
Day-0 deterrent, and usable directly without Zynax; the engine-agnostic hedge that would neutralize
single-vendor dependency is half-built; cloud partnerships are absent; and the one competitively
decisive partnership — Kagent co-existence — is **prose-only, zero code** (§5.19).

**Narrative coherence.** The narrative is unusually coherent and self-aware — clear category, one
defensible wedge, an honest shipped/partial/aspirational split, correct sequencing, and a documented
Truth-Pass culture that *purged a premature CNCF badge*. This honesty directly de-risks the single
thing diligence most fears (delivery-vs-narrative drift) and is a genuine credibility asset. The
verdict: **a coherent thesis with unproven execution** — an IC underwrites the *team and the
architecture's optionality*, not the *business*, because the business is entirely forward-looking
and a funded rival is ahead on the only dimension (distribution) that decides an OSS control plane.

---

## 9.6 Financial Dimension Scorecard (D13 / D15)

| Dim | Group | Weighted score | Confidence | Anchor evidence |
|-----|-------|:--------------:|:----------:|-----------------|
| **D13** | Financial | **5.0** | Medium | replacement cost 8–14 eng-years / ~$2.0–4.5M floor (§5.17); monetization roadmap-CLAIMED only, 0 revenue (§5.18); enterprise SKU unbuildable until identity ships (§5.20) |
| **D15** | Acquisition Readiness *(synth)* | **5.0** | Med-High | acqui-hire fit, team/IP > product (§5.17); 4 Critical condition-precedent risks, 0 deal-breakers (§5.23); incubate path blocked by LICENSE/adopters (§5.21/§5.18) |

> **Reconciliation note.** §5.17's standalone D15 input is 5.0; the orchestrator's
> §8.3-discounted weighted overall across all D1–D16 is 5.4. The 0.4 spread sits well inside
> Medium confidence and is immaterial to the recommendation band.

---

## 9.7 Investment Recommendation (Part 8, §8.5 shape — restated)

```
RECOMMENDATION:     Conditional / Watch  (confidence: Medium)
BEST-FIT STRUCTURE: Acqui-hire (primary) · milestone-gated seed (secondary) · CNCF incubate not-yet
OVERALL SCORE:      5.4 / 10   |   RISK PROFILE: High (4 Critical · 9 High · 11 Medium · 3 Low; 0 deal-breakers)
THESIS IN ONE LINE: A CNCF-grade, execution-verified engineering substrate and a real-but-half-built
                    portability moat, built by an exceptional solo engineer — held back from
                    investability by zero distribution against a CNCF-backed rival, a contradicted
                    headline thesis, and bus-factor-1; back the builder + the IP, not the product,
                    and only against milestones.
```

The raw confidence-weighted mean of the dimension scores is ≈ **6.4/10** (Proceed-Conditional band
on §8.1). The reported actionable overall is discounted to **5.4** because, under Part 8 §8.3, a
Critical is **never averaged away**. Three §8.3 gating rules each *independently* bind the verdict
below any "Proceed":

1. **≥1 unmitigated Critical → cap at Conditional/Watch.** R1 (no LICENSE) is unmitigated
   as-shipped; R3/R4 (enterprise identity / multi-tenant isolation) and zero distribution are
   unmitigated Criticals. *Caps the ceiling.*
2. **Contradicted core-thesis claim → cap at Pass (Revisit later).** C-ARGO: "run on Temporal OR
   Argo without a rewrite" is contradicted at execution (7:1 evidence). *Pulls the floor toward
   Pass-Revisit on any structure whose value rests on the portability moat, until Argo parity ships
   or the claim is re-labelled (CP-4).*
3. **Bus factor = 1, no succession → cap acquisition-readiness at Conditional.** R2. *Makes
   key-person retention the load-bearing diligence item for any non-acqui-hire structure.*

**Swing factors (what would change the call):**

- One named external adopter / first community adapter (today 0/0/0) — retires the Critical
  distribution risk and flips Conditional → Proceed-track. [§5.4/§5.19/§5.25]
- Argo IRInterpreter parity shipped + a cross-engine parity test — removes the §8.3
  contradicted-core-thesis cap and makes the moat real. [§5.1/§5.3]
- A signed co-maintainer / key-person retention + IP-assignment — lifts the bus-factor cap on
  acquisition-readiness and unblocks the CNCF path. [§5.24/§5.13]
- LICENSE + `MAINTAINERS.md` committed — cheap, but a hard precedent for any donatable / clean
  structure. [§5.8/§5.21/§5.22]

**Conditions precedent (must-resolve for any "Proceed"):** CP-1 commit a top-level Apache-2.0
LICENSE (+ NOTICE) and reconcile the OpenSSF badge; CP-2 credible maintainer-succession plan
(≥2 cross-org maintainers, `MAINTAINERS.md`, require ≥1 human review on merge); CP-3 funded plan +
timeline for enterprise identity (RBAC/SSO/OIDC + principal-attributed audit) and multi-tenant
isolation; CP-4 resolve the portability thesis (ship Argo IR-interpretation parity + a cross-engine
test, OR re-price/re-label on "engine-neutral IR, Temporal reference interpreter"); CP-5 traction
trigger (≥1 named pilot/adopter or first community adapter in motion); CP-6 (managed-service only)
close stateful SPOFs, bound the task-broker fan-out, and publish a load-test/SLO baseline before
any SLA-bearing commercialization.

---

## 9.8 Deal-Structure Options & Best-Fit (Part 8 §8.4)

| Structure | Fit for Zynax | Rationale at HEAD |
|-----------|:-------------:|-------------------|
| **Acqui-hire** | **BEST FIT (primary)** | Team/IP materially exceeds product maturity and bus-factor risk dominates (§8.4). The embodied control-plane IP + supply-chain rigor transplant cleanly (11 modules, near-zero debt, §5.14/§5.15) and the proven solo builder is the multiplier. This is the structure where the contradicted portability thesis and zero traction hurt **least** — the acquirer supplies its own distribution. Diligence focus: key-person retention, knowledge transfer, IP assignment. |
| **Equity (milestone-gated seed)** | **Secondary** | Strong tech + early/pre-traction market. Viable IF (and only if) the two de-risking triggers — a named adopter and a co-maintainer — begin to move; release tranches against them. Frame ~$4M–8M post, lower-band. |
| **Asset / technology acquisition** | Possible, but undervalues the builder | Tech is clean and transplantable (license cleanliness pending CP-1), but a pure asset deal forfeits the key-person premium that is the multiplier. |
| **Strategic partnership / OEM** | Premature | No traction, no reference deployment; roadmap alignment unproven; the Kagent "complementary" story is prose-only. |
| **Incubate / sponsor (CNCF path)** | **Right long-run home — NOT YET available** | Blocked today by the missing LICENSE (R1) and 0 adopters (§5.21 verdict: "NOT YET"). The correct destination once the social gates clear. |
| **Pass / monitor** | The fallback if triggers stall | Re-evaluation triggers = the swing factors in §9.7. |

**Why acqui-hire is the recommended best-fit.** The recurring, cross-agent finding is that
**team/IP > product maturity** with bus-factor risk as the dominant fact — the textbook §8.4
acqui-hire condition. The asset's maintainability and modularity (§5.14/§5.15) make integration cost
low; the embodied IP floor (~$2–4.5M) plus a key-person premium for a proven solo builder of a
CNCF-grade substrate gives a defensible valuation anchor; and the two facts that most damage every
*other* structure — the contradicted portability thesis and zero distribution — are precisely the
facts an acquirer with its own go-to-market can absorb. A pure asset acquisition undervalues the
builder; a strategic acquisition is premature; the CNCF path, though the right long-run home, is
gated shut today.

---

## 9.9 Financial Unknowns & Assumptions Ledger (carried to confirmatory diligence)

| # | Unknown / assumption | Confirmatory-session ask |
|---|----------------------|--------------------------|
| A1 | **Dollar valuation is assumption-based** (E7 comps + explicit model), not derivable from in-repo financials — no revenue, no priced round | Treat ~$2.0–4.5M IP floor / ~$4–8M post-seed-if-triggers-move as a frame, not a quote; carry ±40% eng-years sensitivity |
| U4 | **True eng-cost to bring Argo (or any 2nd engine) to IRInterpreter parity** — the single number governing whether the headline thesis is a 1-quarter fix or a structural gap | Sidecar/operator running the existing IRInterpreter in-cluster vs full DAG re-impl — scope + timeline |
| U6 | **Live stars/forks/named adopters + any post-May traction; bottom-up TAM/SAM/SOM** — the 0/0/0 baseline is a May-doc CLAIM, not re-pulled | Any private design-partner/pilot? A defensible SAM/SOM? Is portability a top-3 buying criterion for any segment today? |
| U8 | **Maintainer's openness to a co-maintainer / key-person + IP-assignment** — the gate for both CNCF and any non-acqui-hire structure | Direct conversation — willingness, timeline, IP custody |
| U9 | **Monetization-vs-CNCF model decision** — open-core (Scenario A) conflicts with CNCF neutrality | Which model, and is it decided before or after a CNCF bid? (a one-way-door ADR) |

---

*Cross-references: Section 3 (Product & Market — distribution, TAM, Kagent competitive read);
Section 4 (Technology — the engine-neutral IR and the Argo execution contradiction); Section 8
(Risk — the consolidated 27-row register and the four Critical condition-precedent risks); Section 10
(Conclusion — conditions precedent and confirmatory-diligence timeline). Full source packets:
`docs/due-diligence/2026-06-20-dd-wave-d-findings.md` (§5.17, §5.18) and Waves A/B/C, Appendix A.*
# 10. Conclusion & Next Steps

> **Evidence standard (Part 2 §2.4).** This section restates and operationalises the verdict
> reached in the Wave D synthesis capstone (issue #1405); it introduces no new claim. Every
> factual statement traces to an evidence citation — a `repo-path:line` (E2/E3/E4), an executed
> command and its output (E1), or an external source (E7) — or is explicitly marked `UNKNOWN`.
> **VERIFIED** material (E1–E4) is kept separate from **CLAIMED** material (E5–E6 / roadmap).
> Per-agent detail is Appendix A (`docs/due-diligence/2026-06-20-dd-wave-a-findings.md`, `-b`,
> `-c`, `-d`); the audited code state is `main @ e3135a6`.

## 10.1 Final recommendation (restated)

The recommendation is **Conditional / Watch** at **Medium** confidence, with an overall score of
**5.4 / 10** and a **High** risk profile (4 Critical · 9 High · 11 Medium · 3 Low; **0 absolute
deal-breakers**). The raw confidence-weighted mean of the sixteen dimension scores is **6.4** —
inside the Proceed-Conditional band — but Part 8 §8.3 forbids averaging a Critical away, and three
gating rules each bind the actionable verdict below "Proceed" regardless of that mean: (1) at least
one *unmitigated* Critical (no top-level LICENSE as-shipped; no enterprise identity/tenancy; zero
distribution) caps the ceiling at Conditional/Watch; (2) a *contradicted core-thesis* claim — engine
portability is real at the IR/submission boundary but contradicted at execution, where only Temporal
interprets the IR — pulls the floor toward Pass (Revisit) for any portability-dependent structure;
and (3) bus-factor-1 with no succession caps acquisition-readiness at Conditional.

```
RECOMMENDATION:     Conditional / Watch   (confidence: Medium)
OVERALL SCORE:      5.4 / 10              (raw confidence-weighted mean 6.4, §8.3-discounted)
RISK PROFILE:       High                  (4 Critical · 9 High · 11 Medium · 3 Low; 0 deal-breakers)
BEST-FIT STRUCTURE: Acqui-hire (primary) · milestone-gated seed (secondary) · CNCF incubate not-yet
THESIS (one line):  A CNCF-grade, execution-verified engineering substrate and a real-but-half-built
                    portability moat, built by an exceptional solo engineer — back the builder + IP,
                    not the product, and only against milestones.
```

The substance of the verdict is unchanged from the synthesis: Zynax is **a best-in-class engineering
substrate wrapped around a business that does not yet exist.** The asset — a CNCF-grade control plane
built solo in roughly two months, ~40k non-test LOC, 306 BDD scenarios, 37 ADRs, and a
cosign + SBOM + SLSA supply chain — is real and execution-verified; the *commercial and social*
substrate is absent or aspirational. The dominant fact is that the gaps are **enforcement-shaped and
social, not absence-shaped**: capabilities exist but are opt-in, soft, or single-authored, and none
of the four Criticals is an absolute viability or legality kill — each is a cheap or scoped
condition-precedent. The single most reassuring diligence signal is the Truth-Pass culture in which
the project's own docs *under-state* reality (the opposite of the §1.10 history), which is precisely
why the dominant bus-factor risk is treated as a condition-precedent rather than a kill.

**Best-fit structure.** Acqui-hire is primary because team/IP materially exceeds product maturity and
bus-factor risk dominates: the 11 Go-module substrate transplants cleanly at near-zero technical debt
and the proven solo builder is the multiplier, and the contradicted portability thesis and zero
traction hurt least when an acquirer supplies its own distribution. A **milestone-gated seed** is the
secondary structure, available only once the two de-risking triggers (a named adopter and a
co-maintainer) begin to move; a **CNCF incubate/sponsor** path is the right long-run home but is **not
yet available**, blocked by the missing LICENSE plus zero adopters. This conclusion is concordant with
the independent §5.17 Investment draft (Conditional/Watch, Medium, acqui-hire primary); the only delta
is the overall figure — 5.4 here versus 5.17's 5.0 D15 input — a 0.4 spread inside Medium confidence,
immaterial to the band.

## 10.2 Conditions precedent (the must-resolve list)

The four Critical risks are condition-precedent, not deal-breakers; clearing the list below converts
the verdict from a gated Conditional/Watch toward a Proceed-track. Each is mapped to its risk ID and a
cheap-to-fix versus scoped-investment classification.

| CP | Condition precedent | Risk | Cost class | Gate it lifts |
|----|---------------------|:----:|:----------:|---------------|
| **CP-1** | Commit a top-level Apache-2.0 **LICENSE** (+ NOTICE); reconcile the OpenSSF badge | R1 | One-line fix | Apache §4 distribution defect; CNCF hard blocker |
| **CP-2** | Credible maintainer-succession / **bus-factor** plan: ≥2 cross-org maintainers, `MAINTAINERS.md` (#494), require ≥1 human review on merge | R2 | Social / before close or price-earnout | §8.3 acquisition-readiness cap; CNCF ≥2-maintainer gate |
| **CP-3** | Funded plan + timeline for **enterprise identity** (RBAC/SSO/OIDC + principal-attributed audit) and **multi-tenant isolation** | R3 / R4 / R10 | Scoped roadmap | Enterprise-procurement blocker (any enterprise GTM) |
| **CP-4** | Resolve the **portability thesis**: ship Argo IR-interpretation parity (+ a cross-engine parity test) OR re-price/re-label on "engine-neutral IR, Temporal reference interpreter" | R5 / R8 | 1-quarter vs structural (UNKNOWN) | §8.3 contradicted-core-thesis cap |
| **CP-5** | **Traction trigger**: ≥1 named pilot/adopter or first community adapter in motion | R6 / R20 | Go-to-market | Critical distribution risk (0/0/0 today) |
| **CP-6** | *(Managed-service only)* close the stateful **SPOFs**, bound the task-broker fan-out, publish a load-test/SLO baseline before any SLA-bearing commercialization | R7 / R9 / R16 / R19 | Bounded engineering | Production/SLA readiness |

CP-1 and CP-2 are the two **closing conditions** for almost any structure: CP-1 is a one-line commit
yet currently unmitigated as-shipped and is the single fact driving the "High" aggregate profile
([§5.23 register R1; 5.8 final.md:647-652 / 5.22 final.md:1213-1218; `git log --all -- LICENSE` empty
→ E1]), and CP-2 addresses the diligence's central finding, bus-factor-1 ([5.24 git shortlog
→ one human, 772 commits, `required_approving_review_count=0`; #494 `MAINTAINERS.md` absent]). CP-3
is a condition precedent specifically for any **enterprise** go-to-market, where the platform today
fails the first checkboxes of a procurement questionnaire — single static bearer key, no per-principal
audit, shared Temporal namespace ([5.20 final.md:1606-1616; auth.go:13-26; temporal.go:54-58]). CP-4
is the §8.3 trigger: a buyer who tests "run on Argo" today gets a workflow that reports Success without
running anything, because `argo_engine.go:62-98` never calls `IRInterpreter.Run` and the Argo path
hands the IR to a smoke-stub that asserts payload ≠ empty and exits 0
([scripts/e2e/manifests/argo-ir-interpreter.yaml:10-14 — E2; resolved 7 agents to 1]). The single
governing number — the true engineering cost to bring Argo (or any second engine) to interpreter
parity, a one-quarter sidecar versus a structural re-implementation — is **UNKNOWN** at HEAD and is
the most important question a confirmatory session must answer.

## 10.3 Confirmatory-diligence checklist

The static, offline audit could not verify the items below. A management/technical session must close
each before any "Proceed" decision; together they constitute the unknowns ledger carried forward from
the synthesis (U1–U9, A1). Several are runtime confirmations of things that are *configured* in the
repo but whose live behaviour was not observable in the diligence environment — keep CLAIMED/CONFIGURED
strictly separate from VERIFIED until each is executed live.

| # | What to verify (UNKNOWN at HEAD) | Why it is open | Confirmatory ask (the live test) |
|---|----------------------------------|----------------|----------------------------------|
| **U1** | GHCR image **cosign signature / SLSA-attestation existence** (C4 residual) | cosign unrunnable offline; no registry access. Signing is CONFIGURED in `release.yml`, presence UNKNOWN | Run `cosign verify` against a published GHCR digest **live** — E1 |
| **U2** | Live `make demo` **runtime success + wall-clock** (the <15-min hero claim) | Static-only audit; no Docker/Ollama in the diligence env | Run end-to-end on a clean machine; time it; run the stateful path **twice** |
| **U3** | **Argo CI-leg** green/red + live bench-gate behaviour at HEAD | Not observable offline; the bench gate is unwired/fail-open if run (R17) | Show the latest CI runs; demonstrate the bench gate failing a planted regression |
| **U4** | True eng-cost to bring Argo (or any 2nd engine) to **IRInterpreter parity** (C-ARGO residual) | The single number governing the headline thesis; not derivable from code | Sidecar/operator running the existing interpreter in-cluster vs full DAG re-impl — scope + timeline |
| **U5** | Real multi-service **latency + 10x/100x break-points** | Zero load tests anywhere; the scaling story is inferential | Provide a load harness + SLOs, or commit to producing them as a CP |
| **U6** | Live **stars/forks/named adopters** + any post-May traction; bottom-up TAM/SAM/SOM | Out of repo scope; the 0/0/0 baseline is a May-doc CLAIM, not re-pulled | Any private design-partner/pilot? A defensible SAM/SOM? Is portability a top-3 buying criterion today? |
| **U7** | Production **Helm DB topology** + HA/backup posture | Not confirmed in `infra/helm` | Show the production Helm values + DB HA/backup runbooks |
| **U8** | Maintainer **openness to a co-maintainer / key-person + IP-assignment** | A social fact, not in the repo; the gate for both CNCF and any non-acqui-hire structure | Direct conversation — willingness, timeline, IP custody |
| **U9** | **Monetization-vs-CNCF model** decision (open-core conflicts with CNCF neutrality) | The strategy doc defers the decision | Which model, and is it decided before or after a CNCF bid? (a one-way-door ADR) |
| **A1** | Dollar **valuation** is assumption-based (E7 comps), not derivable from in-repo financials | No revenue, no priced round | Treat ~$2.0–4.5M IP floor / ~$4–8M post-seed-if-triggers-move as a frame, not a quote (±40% eng-years sensitivity) |

The three highest-leverage live tests are **U1** (cosign verify against GHCR — the one outstanding gap
in an otherwise VERIFIED supply-chain trifecta), **U2** (a clean-machine `make demo`, run twice to
catch any second-run persistence bite — the static audit verified config, not runtime), and **U4** (the
Argo-parity cost that determines whether CP-4 is a one-quarter fix or a structural gap).

## 10.4 Indicative timeline & re-evaluation triggers

This is a **milestone-gated option, not a now-deal.** Sequence the conditions precedent so the cheapest,
highest-signal items front-load the diligence, and release any capital in tranches against the
de-risking triggers rather than at a single close.

| Horizon | Milestone | Conditions / triggers |
|---------|-----------|------------------------|
| **Confirmatory session (days)** | Close the offline UNKNOWNs | Run U1–U3 live; obtain U4/U6–U9 answers; commit **CP-1** LICENSE (one-line) as a good-faith precondition |
| **Pre-close (weeks)** | Clear the closing conditions | **CP-1** committed; **CP-2** maintainer-succession plan + `MAINTAINERS.md` (#494) + ≥1-review-on-merge agreed; **CP-4** either Argo-parity scoped with a credible timeline OR every portability surface re-labelled to HEAD reality |
| **Tranche 1 — de-risking window (1–2 quarters)** | Distribution + key-person proof | **CP-5** ≥1 named adopter / first community adapter in motion; **CP-2** ≥2 cross-org maintainers signed; **CP-4** Argo IRInterpreter parity + cross-engine parity test shipped |
| **Tranche 2 — enterprise/managed readiness** | Procurement + SLA readiness | **CP-3** enterprise identity (RBAC/SSO/OIDC + audit) and multi-tenant isolation funded and landing; **CP-6** SPOFs closed, fan-out bounded, load-test/SLO baseline published |
| **Long-run** | CNCF path | LICENSE + ≥2-maintainer + external audit + adopters all clear → CNCF incubate/sponsor becomes available |

**Re-evaluation triggers (what would move the verdict).** Any of the four swing factors materially
changes the recommendation and should re-open this diligence:

- **One named external adopter / first community adapter** (today 0/0/0) — retires the Critical
  distribution risk and flips Conditional → Proceed-track [§5.4, §5.19, §5.25].
- **Argo IRInterpreter parity shipped + a cross-engine parity test** — removes the §8.3
  contradicted-core-thesis cap and makes the portability moat real [§5.1, §5.3].
- **A signed co-maintainer / key-person retention + IP-assignment** — lifts the bus-factor cap on
  acquisition-readiness and unblocks the CNCF path [§5.24, §5.13].
- **LICENSE + `MAINTAINERS.md` committed** — cheap, but a hard precedent for any license-clean or
  donatable structure [§5.8, §5.21, §5.22].

Conversely, the verdict should be **re-evaluated downward** on negative triggers: a funded
fast-follower replicating the public IR/port design before parity ships (the IP moat is replicable in
~2 quarters; R13), continued delivery-vs-narrative drift surviving a milestone Truth-Pass (R8), or a
second milestone passing with the 0/0/0 distribution baseline unchanged while the CNCF-backed rival
extends its lead (R6/R12). Dated standing exceptions also carry their own clocks — the trivy DS002
suppression is accepted only until 2026-11-01 (R25) — and should be re-checked at deadline.

**Bottom line.** Back the builder and the substrate, not the product, and only against milestones. The
asset is real and the risks are known, bounded, and honestly documented; the path from Conditional to
Proceed runs through a short list of cheap-or-scoped conditions precedent and a confirmatory session
that closes the offline unknowns. The next and final deliverable is the assembly of this conclusion
and the preceding nine sections into the Part 10 document and the Part 9 executive presentation
(issue #1406) — assembly and presentation of what Waves A–D already established, with no new analysis
required.

# Appendices & Executive Presentation

> Supporting apparatus for the assembled Zynax investment-grade due-diligence report (issue
> #1406). Today: 2026-06-21. Repository HEAD audited: `main` @ `e3135a6`. This part carries the
> per-agent findings index, the evidence-citation convention, the verbatim machine-readable
> scorecard, the glossary, and the board-level slide outline. It introduces **no new claims** —
> every figure traces to a Wave A–D source packet (full §3.4 packets live in the wave findings
> docs, referenced below by repo-relative name).

---

## Appendix A — Per-agent full findings (index)

The diligence ran **26 agents** across four parallelization waves (framework §3.2). Each agent's
complete §3.4 handoff packet — `overall_score`, per-sub-dimension scores with `path:line`
evidence, the mandatory drift test, red/green flags, unknowns, cross-references, and
recommendations — lives in its **wave findings document**. This appendix is a pointer table only;
the packets are **not duplicated** here (they are the report's Appendix A corpus). Refer to the
four wave documents:

- `docs/due-diligence/2026-06-20-dd-wave-a-findings.md` — Wave A (ground truth)
- `docs/due-diligence/2026-06-20-dd-wave-b-findings.md` — Wave B (derived technical)
- `docs/due-diligence/2026-06-20-dd-wave-c-findings.md` — Wave C (product / market / governance)
- `docs/due-diligence/2026-06-20-dd-wave-d-findings.md` — Wave D (synthesis + orchestrator)

| # | Agent | Dimension group(s) | Wave | Findings doc (full §3.4 packet) |
|---|-------|--------------------|:----:|----------------------------------|
| 5.1 | Architecture | D3 | A | `2026-06-20-dd-wave-a-findings.md` |
| 5.2 | Security & Supply-chain | D5 | A | `2026-06-20-dd-wave-a-findings.md` |
| 5.5 | Engineering | D4 | A | `2026-06-20-dd-wave-a-findings.md` |
| 5.7 | Testing | D9 | A | `2026-06-20-dd-wave-a-findings.md` |
| 5.9 | DevOps / CI-CD | D8 | A | `2026-06-20-dd-wave-a-findings.md` |
| 5.10 | Documentation | D11 | A | `2026-06-20-dd-wave-a-findings.md` |
| 5.12 | AI-native Workflow / SPDD | D10 | A | `2026-06-20-dd-wave-a-findings.md` |
| 5.24 | Repository Health | D16 | A | `2026-06-20-dd-wave-a-findings.md` |
| 5.6 | Performance | D6 | B | `2026-06-20-dd-wave-b-findings.md` |
| 5.14 | Technical Debt | D4 | B | `2026-06-20-dd-wave-b-findings.md` |
| 5.15 | Maintainability | D4 | B | `2026-06-20-dd-wave-b-findings.md` |
| 5.16 | Scalability | D3/D6 | B | `2026-06-20-dd-wave-b-findings.md` |
| 5.22 | OpenSSF / Scorecard | D5/D7 | B | `2026-06-20-dd-wave-b-findings.md` |
| 5.26 | Innovation & IP | D10/D16 | B | `2026-06-20-dd-wave-b-findings.md` |
| 5.3 | Product | D1 | C | `2026-06-20-dd-wave-c-findings.md` |
| 5.4 | Market & TAM | D2 | C | `2026-06-20-dd-wave-c-findings.md` |
| 5.19 | Competitive (incl. Kagent) | D2 | C | `2026-06-20-dd-wave-c-findings.md` |
| 5.8 | Open Source Health | D7 | C | `2026-06-20-dd-wave-c-findings.md` |
| 5.13 | Governance | D12 | C | `2026-06-20-dd-wave-c-findings.md` |
| 5.20 | Enterprise Adoption | D14 | C | `2026-06-20-dd-wave-c-findings.md` |
| 5.21 | CNCF Readiness | D7 | C | `2026-06-20-dd-wave-c-findings.md` |
| 5.25 | Future Roadmap Realism | D12 | C | `2026-06-20-dd-wave-c-findings.md` |
| 5.11 | Developer Experience | D1 | C | `2026-06-20-dd-wave-c-findings.md` |
| 5.23 | Risk Assessment | D15 | D | `2026-06-20-dd-wave-d-findings.md` |
| 5.17 | Investment Analysis | D13/D15 | D | `2026-06-20-dd-wave-d-findings.md` |
| 5.18 | Business Strategy | D13 | D | `2026-06-20-dd-wave-d-findings.md` |

The orchestrator's **D0 Executive Summary** — the confidence-weighted D1–D16 scorecard, the
C1–C8 + C-ARGO contradiction resolutions, the aggregate risk profile, and the Part 8 verdict — is
the head of `2026-06-20-dd-wave-d-findings.md`. Of the 16 dimension groups, **D15 (Acquisition
Readiness)** and **D0** are *synthesized*, not direct agent inputs; all other dimensions map to the
contributing agents shown above.

---

## Appendix B — Evidence index & citation convention

### How evidence is cited (framework §2.4)

Every factual claim in this report carries one of three citation forms, or is explicitly marked
`UNKNOWN — not found`. Citations are graded strongest-to-weakest:

| Tier | Kind | Form | Counts as |
|:----:|------|------|:---------:|
| **E1** | Executed proof (command + output) | `cmd → output` | VERIFIED |
| **E2** | Source code | `repo-path:line` | VERIFIED |
| **E3** | Configuration / CI | workflow YAML, Makefile, Helm values `:line` | VERIFIED |
| **E4** | Contract / schema | proto, JSON Schema, AsyncAPI `:line` | VERIFIED |
| **E5** | First-party doc (ADR, AGENTS.md) | `repo-path:line` | CLAIMED |
| **E6** | Marketing / README claim | `repo-path:line` | CLAIMED |
| **E7** | External (CVE DB, competitor docs, comps) | URL | CLAIMED |

**Rule:** a fact is `VERIFIED` only on E1–E4; a claim resting on E5–E6 (or roadmap) is `CLAIMED`
and labelled as such; aspirational items are kept strictly separate from delivered ones. VERIFIED
and CLAIMED are never merged in any score or statement in this report.

### Highest-signal citations per dimension

The single most load-bearing citation behind each dimension score (full citation sets in the wave
docs):

| Dim | Group | Highest-signal evidence | Tier |
|-----|-------|-------------------------|:----:|
| D1 | Product | hero path runs zero-secret (5.3 `03-product.md:103-111`); "one command" demo needs 4 prereqs (5.11 `Makefile:162-175`) | E1/E3 |
| D2 | Market | Kagent owns identical framing at 0 adopters (5.19 `final.md:1348-1354`); compile-time IR validation shipped (5.19 `structural.go:11-58`) | E2/E7 |
| D3 | Architecture | engine-neutral IR, no engine types leak (5.1 `workflow_compiler.proto:205-241`); Argo never calls `IRInterpreter.Run` (5.1 `argo_engine.go:62-98`) | E2/E4 |
| D4 | Engineering | 14-linter 0-issues, no blanket `//nolint` (5.5 `golangci-lint.yml:17-33`); 11 Go modules block cross-service coupling (5.15) | E2/E3 |
| D5 | Security | cosign+SBOM+SLSA trifecta (5.2 `release.yml:201,510,527`); mTLS fails open to insecure creds (5.2 `tlscreds.go:21`) | E2/E3 |
| D6 | Performance | hot-path benches beat targets 5–300× (5.6 scorecard); unbounded task-broker fan-out (5.6 `service.go:80-87`); zero load tests | E1/E2 |
| D7 | Open Source | no top-level LICENSE, `git log --all -- LICENSE` empty (5.8); bus factor 1 (5.13); CNCF "NOT YET" (5.21) | E1/E2 |
| D8 | DevOps | build-once / promote-by-retag, scan==deploy (5.9 `release.yml:160-204`); 21 workflows | E3 |
| D9 | Testing | ≥90% blocking gate executed 92.1–100% on 7 domains (5.7 `coverage-gates.env`); 306 BDD scenarios / all RPCs | E1/E3 |
| D10 | AI Workflow | closed traceable learnings loop (5.12 `APPLY_LOG.md:15-99`); canvas-before-code is a soft gate (5.12 `pr-checks.yml:231-233`) | E2/E3 |
| D11 | Documentation | §1.10 doc-vs-tooling lag resolved at HEAD (5.10 `CLAUDE.md:86-110`); README self-contradicts milestone table (5.10) | E2/E5 |
| D12 | Governance | Truth-Pass down-graded M3/M4 with reasons (5.13); v1.0 gate is social/unbuyable-by-velocity (5.25 `final.md:2083-2087`) | E2/E5 |
| D13 | Financial | replacement cost ~8–14 eng-yrs / ~$2.0–4.5M floor (5.17 `17-investment.md:229-253`); 0 revenue, monetization CLAIMED only (5.18) | E2/E7 |
| D14 | Enterprise | single static bearer key, no RBAC/SSO (5.20 `auth.go:13-26`); shared Temporal namespace (5.20 `temporal.go:54-58`) | E2 |
| D15 | Acquisition | acqui-hire fit, team/IP > product (5.17); 4 Critical condition-precedent risks, 0 deal-breakers (5.23 `23-risk.md:23-32`) | E2/E5 |
| D16 | Repo Health & Innovation | linear signed history, 0 merge commits (5.24 `git log --merges → 0`); bus factor 1 (5.24 `git shortlog`) | E1 |

---

## Appendix C — Machine-readable scorecard (JSON)

> Copied verbatim from the Wave D orchestrator output
> (`docs/due-diligence/2026-06-20-dd-wave-d-findings.md`, §"Machine-readable scorecard (JSON)").

```json
{
  "overall_score": 5.4,
  "overall_score_raw_weighted_mean": 6.4,
  "overall_confidence": "Medium",
  "recommendation": "Conditional / Watch",
  "best_fit_structure": "Acqui-hire (primary); milestone-gated seed (secondary); CNCF incubate not-yet-available",
  "risk_profile": "High",
  "risk_counts": { "critical": 4, "high": 9, "medium": 11, "low": 3, "total": 27, "deal_breakers": 0, "unmitigated_critical": 1 },
  "dimension_scores": {
    "D1": { "name": "Product", "score": 5.5, "confidence": "Medium" },
    "D2": { "name": "Market", "score": 5.5, "confidence": "Medium" },
    "D3": { "name": "Architecture", "score": 6.0, "confidence": "High" },
    "D4": { "name": "Engineering", "score": 7.7, "confidence": "High" },
    "D5": { "name": "Security", "score": 6.4, "confidence": "Medium-High" },
    "D6": { "name": "Performance", "score": 5.6, "confidence": "High" },
    "D7": { "name": "Open Source", "score": 6.0, "confidence": "High" },
    "D8": { "name": "DevOps", "score": 8.0, "confidence": "High" },
    "D9": { "name": "Testing", "score": 8.0, "confidence": "High" },
    "D10": { "name": "AI Workflow", "score": 6.7, "confidence": "High" },
    "D11": { "name": "Documentation", "score": 7.0, "confidence": "High" },
    "D12": { "name": "Governance", "score": 7.0, "confidence": "High" },
    "D13": { "name": "Financial", "score": 5.0, "confidence": "Medium" },
    "D14": { "name": "Enterprise Readiness", "score": 4.0, "confidence": "High" },
    "D15": { "name": "Acquisition Readiness (synthesized)", "score": 5.0, "confidence": "Medium-High" },
    "D16": { "name": "Repo Health & Innovation", "score": 6.6, "confidence": "High" }
  },
  "contradiction_register": {
    "C1": "CONTRADICTED-then-corrected (docs now honest/under-state)",
    "C2": "PARTIAL (mTLS fails open; configurable not enforced)",
    "C3": "VERIFIED",
    "C4": "PARTIAL (configured; GHCR signature existence UNKNOWN)",
    "C5": "VERIFIED (NATS JetStream CloudEvents real)",
    "C6": "VERIFIED (agent-registry + Postgres repo)",
    "C7": "VERIFIED (stronger than claimed)",
    "C8": "VERIFIED",
    "C_ARGO": "RESOLVED — portability real at submission, CONTRADICTED at execution (half-built); residual eng-cost OPEN"
  },
  "critical_red_flags": [
    "R2 Bus factor = 1 + 0-review self-merge (concentration root multiplier)",
    "R1 No top-level LICENSE file (Apache §4 defect + CNCF hard blocker; unmitigated)",
    "R3/R4 No enterprise identity + no multi-tenant isolation (procurement blocker)",
    "Zero distribution vs CNCF-backed Kagent (ecosystem moat is 0-node)"
  ],
  "core_thesis_status": "CONTRADICTED at execution (C-ARGO; §8.3 contradicted-core-thesis cap applies)",
  "gating_rules_applied": [
    "§8.3 unmitigated Critical -> cap at Conditional/Watch",
    "§8.3 contradicted core thesis (Argo portability) -> floor pulled toward Pass(Revisit) until corrected",
    "§8.3 bus factor 1, no succession -> acquisition-readiness capped at Conditional"
  ],
  "swing_factors": [
    "One named external adopter / first community adapter (today 0/0/0)",
    "Argo IRInterpreter parity shipped + cross-engine parity test",
    "Signed co-maintainer / key-person retention + IP-assignment",
    "LICENSE + MAINTAINERS.md committed (cheap precedent for any donatable/clean structure)"
  ],
  "unknowns": [
    "GHCR cosign signature / SLSA-attestation existence (offline-unverifiable)",
    "Live make demo runtime success + wall-clock (<15-min claim)",
    "Argo CI-leg green/red + live bench-gate behaviour at HEAD",
    "True eng-cost to bring Argo to IRInterpreter parity (1-quarter vs structural)",
    "Real multi-service latency + 10x/100x break-points (zero load tests)",
    "Live stars/forks/adopters + post-May traction; bottom-up TAM/SAM/SOM",
    "Prod Helm DB topology (instances vs DBs); HA/backup posture",
    "Maintainer openness to co-maintainer + IP-assignment",
    "Monetization-vs-CNCF model decision (open-core conflicts with neutrality)",
    "Dollar valuation is assumption-based (E7 comps; no revenue/priced round)"
  ]
}
```

---

## Appendix D — Tooling: the prompts used (this framework)

This diligence was produced by **executing a portable, repository-specific agent framework**, not by ad-hoc review. The full framework — the Part 1 context packet, the Part 2 methodology (0–10 scale, confidence bands, the E1–E7 evidence taxonomy, the mandatory drift test), the Part 3 wave / anti-overlap strategy, the Part 4 master-orchestrator prompt, and all 26 Part 5 specialised agent prompts — is committed at [`docs/due-diligence/2026-06-18-zynax-due-diligence-framework.md`](2026-06-18-zynax-due-diligence-framework.md). The four wave findings docs listed in Appendix A are the captured run outputs (Waves A→D); **this report is their synthesis**. The prompts are provider-portable (copy-paste into any agent session) and were executed **read-only at HEAD `e3135a6`** over Waves A (ground truth) → B (derived technical) → C (product/market/governance) → D (synthesis + orchestrator), in that dependency order.

---

## Appendix E — Glossary

| Term | Meaning |
|------|---------|
| Workflow IR | Engine-neutral intermediate representation a YAML workflow compiles to (ADR-012). |
| Layer 1/2/3 | Intent (YAML) / Communication (gRPC+AsyncAPI) / Execution (engines+adapters). |
| SPDD | Structured Prompt-Driven Development — REASONS Canvas before code (ADR-019). |
| REASONS Canvas | The pre-implementation design artifact for `feat:` PRs (`docs/spdd/`). |
| Adapter / Capability | A gRPC service implementing `AgentService`; how agents plug in without an SDK (ADR-013). |
| Engine adapter | A pluggable `WorkflowEngine` impl (Temporal, Argo) executing the IR (ADR-015). |
| Truth Pass | The internal exercise that reconciled claims with delivery (M5.A, issue #458). |
| Drift | Gap between documented claims and verified implementation. |
| Bus factor | Number of people whose loss would stall the project. |
| CNCF Sandbox | The entry tier for CNCF projects; the M8 ambition. |
| C-ARGO | The cross-agent conflict over whether ArgoEngine truly interprets the IR; resolved as half-built portability. |
| Condition precedent (CP) | A risk classed as a fixable gate that must close before a "Proceed" verdict (vs a deal-breaker). |
| VERIFIED vs CLAIMED | VERIFIED rests on E1–E4 evidence; CLAIMED rests on E5–E6/roadmap and is labelled separately. |

---

## Part 9 — Executive Presentation Outline (board / IC / acquirer)

> 12–15 slides, **one content bullet per slide**. Lead with the verdict; one chart per slide;
> every quantitative claim footnoted to its evidence tier; **VERIFIED vs CLAIMED** marked; **no
> Critical risk hidden** (framework Part 9, lines 1608–1629). Footnote markers cite the source
> packet behind each figure.

1. **Title & verdict** — *Conditional / Watch* (confidence Medium); overall **5.4/10** (§8.3-gated;
   raw weighted mean 6.4); risk profile **High** (4 Critical · 9 High · 11 Medium · 3 Low, 0
   deal-breakers). [VERIFIED scorecard; Wave D D0]
2. **What Zynax is** — the "Kubernetes for AI workflows" thesis in one diagram: YAML intent →
   engine-neutral IR → pluggable engines/adapters (three layers). [Part 1 §1.1]
3. **Why now** — agent-orchestration land-grab + the portability/lock-in pain the engine-neutral IR
   targets; an honest, Truth-Pass-built asset against that tailwind. [5.4 / 5.19]
4. **The product today** — hero path runs zero-secret end-to-end (VERIFIED, E1), but enterprise
   identity/tenancy are absent and the "one command" demo needs 4 prereqs (drift). [5.3 / 5.11]
5. **Differentiation & moat** — genuinely engine-neutral IR + a clean 5-method `WorkflowEngine`
   port and no-SDK adapter contract (VERIFIED, real at the interface); IP replicable in ~2
   quarters (CLAIMED risk). [5.1 / 5.26 / 5.19]
6. **Architecture** — three enforced layers; **portability is HALF-BUILT**: only Temporal
   interprets the IR, Argo submits to a smoke-stub and exits 0 (C-ARGO, 7:1 evidence,
   CONTRADICTED at execution). [5.1 `argo_engine.go:62-98`, E2 — **shown, not hidden**]
7. **Engineering & quality** — 14-linter 0-issues, near-zero debt, 11 Go modules enforcing
   modularity; **D4 7.7 · VERIFIED**. [5.5 / 5.14 / 5.15]
8. **Testing & DevOps** — ≥90% blocking coverage gate proven **92.1–100%** on 7 domains, 306 BDD
   scenarios; build-once/promote-by-retag CI; **D9 8.0 · D8 8.0 · VERIFIED**. [5.7 / 5.9]
9. **Security & supply chain** — cosign + SBOM + SLSA trifecta VERIFIED (E3); **but mTLS fails
   OPEN** to insecure creds across 5 services and there is no top-level LICENSE; GHCR signature
   presence UNKNOWN offline. [5.2 / 5.22 — Critical & High surfaced]
10. **Open source, governance & CNCF** — honest self-correcting governance (D12 7.0) undercut by
    **bus factor = 1 + 0-review self-merge** and a CNCF path "NOT YET" (no LICENSE / MAINTAINERS /
    0 adopters). [5.8 / 5.13 / 5.21 / 5.24 — Critical surfaced]
11. **Scorecard** — D1–D16 heatmap with confidence: strengths D8/D9 (8.0), D4 (7.7); weaknesses
    D14 Enterprise (4.0), D13 Financial (5.0), D1/D2 (5.5). [VERIFIED; Wave D scorecard]
12. **Risk register** — 4 Criticals, each a **condition-precedent not a deal-breaker** (LICENSE ·
    bus-factor-1 · enterprise identity · multi-tenancy) + top Highs (half-built portability, mTLS
    fail-open, stateful SPOFs). [5.23 — no Critical hidden]
13. **Financials & deal** — replacement cost **~8–14 eng-years / ~$2.0–4.5M IP floor** (CLAIMED,
    E7 comps + model; 0 revenue); best-fit structure **acqui-hire (primary)**, milestone-gated
    seed secondary, CNCF incubate not-yet. [5.17 / 5.18]
14. **Swing factors & conditions precedent** — what flips the call: 1 named adopter (0/0/0 today),
    Argo IR parity + cross-engine test, a signed co-maintainer + IP assignment, a committed
    LICENSE; CP-1…CP-6. [Wave D verdict]
15. **Recommendation & next steps** — **back the builder + IP, not the product, and only against
    milestones**; confirmatory session must answer the unknowns ledger (GHCR signatures, live
    `make demo` wall-clock, Argo-parity cost, real traction/TAM). [Wave D D0 · Appendix C unknowns]
