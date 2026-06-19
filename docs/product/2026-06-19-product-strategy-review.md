<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax — Product Strategy Review (2026-06-19)

**Document type:** Point-in-time Product Strategy Review
**Review date:** 2026-06-19
**Baselines diffed against:**
- [docs/product/2026-06-18-product-strategy-review.md](2026-06-18-product-strategy-review.md) (prior dated review)
- [docs/product/strategy.md](strategy.md) (living strategy, §7.1 Day-0 audit)
- [docs/product/2026-06-18-ux-roadmap-realignment.md](2026-06-18-ux-roadmap-realignment.md) (UX program realignment)

**Scope:** Tier 1 (public-safe) — grounded claims only
**Live signals updated:** via `gh api` (2026-06-19)

---

## 1. Executive Summary

**Single most important product finding (the delta since 2026-06-18):** Zynax has **collapsed the Day-0 adoption barrier** that the prior review and `strategy.md §7.1` named as the binding constraint. EPIC **#1370 (First-run User Experience)** landed a tight cluster of ~11 PRs on 2026-06-18 that turned the hero use case from "real but gated" into "runnable from a fresh clone with **zero paid secrets**." Verified on `main`:

- **Zero-secret local path:** an Ollama overlay (`COMPOSE_DEMO`, [Makefile:22](../../Makefile)) lets the whole stack run on a local Qwen2.5-Coder 3B model — **no paid API key** (PRs #1433, #1437; story #1374, #1386).
- **One-command demo:** `make demo` ([Makefile:132](../../Makefile)) boots the minimal stack + Ollama overlay and runs the hero workflow end-to-end (story #1360, closed).
- **Visible output:** a new `zynax result` subcommand ([cmd/zynax/cmd/result.go](../../cmd/zynax/cmd/result.go), PR #1438) surfaces capability result payloads — previously the workflow ran but the user saw nothing meaningful (story #1378).
- **Doesn't crash without secrets:** git/ci/llm adapters tolerate missing secrets instead of crash-looping (PR #1432; story #1375).
- **Runnable hero example:** [spec/workflows/examples/code-review-ollama.yaml](../../spec/workflows/examples/code-review-ollama.yaml) completes from the CLI (PR #1435; story #1376); `code-review.yaml` is now marked a reference.
- **Quickstart reconciled to the real CLI surface** and leads with the Ollama example ([docs/quickstart.md](../../quickstart.md) on main, story #1379).

This is the single most important product move since v0.5.0: the prior review's three Critical Day-0 recommendations (#2 keystone demo, #3 zero-Temporal, #6 friction) were the **highest-leverage unit of work remaining before M8** — and the first-run UX half of that is now substantially **shipped**, not aspirational.

**What did not move:** the adoption *signals* (1 star, 0 forks, 3 contributors — unchanged from 2026-06-18) and the **social blockers** (single maintainer, zero named external adopters, undifferentiated-vs-Kagent messaging). The product has removed the *friction* that would have wasted any inbound traffic; it has not yet generated the *traffic*. The constraint has shifted one notch downstream: from "the first run fails / needs a paid key" to "nobody has been invited to take the first run yet."

| Question | 2026-06-18 answer | 2026-06-19 answer |
|---|---|---|
| Can a stranger run the hero workflow with no paid key? | ⚠ No — needed LLM key + Temporal; output invisible | ✅ **Yes** — `make demo` + Ollama, `zynax result` shows the review |
| Time-to-first-workflow | 🟡 8–12 min, key + image friction | 🟡 **One command** (`make demo`), minus an `ollama pull` prereq |
| Is the architecture differentiated? | ✅ Engine-agnostic IR + adapters | ✅ Unchanged |
| What blocks adoption today? | Day-0 friction **and** social | **Social only** — friction substantially closed |
| External adopters | 0 | 0 (unchanged) |

---

## 2. Positioning Check — Are We Still Differentiated?

**From [README.md](../../README.md):** unchanged tagline — *"Zynax is to AI workflows what Kubernetes is to containers — a control plane that abstracts the execution layer behind a declarative, versionable API."*

**Drift check (the canonical test — does every user-facing surface lead with the engine-portability wedge?):**

- **Quickstart now leads with the runnable zero-secret path**, not the generic control-plane framing — a positive shift. The first line of [docs/quickstart.md](../../quickstart.md) (on main) is the `make demo` one-command path; the body promises "a fresh clone to a **traced workflow run** in minutes." This is the strategy.md §7.1 prescription ("make the hero demo the literal first thing") finally honored at the docs layer.
- **Still drifting:** the README tagline itself still competes on the contested "control plane for AI agents" category rather than leading with *engine-agnostic IR + multi-engine dispatch*. This is the **same drift flagged on 2026-06-18 (rec #4) and unchanged** — the Kagent-differentiated messaging recommendation remains **Not started**.
- **The hero asciinema cast** (the README-first visual proof) is still **not recorded** — [docs/casts/](../casts/) on main contains only a `README.md` placeholder. This is the named human follow-up; the engineering substrate for it (a workflow that actually completes from the CLI) is now in place, so the cast is finally *recordable*.

**Verdict:** Positioning at the **docs/onboarding layer improved** (leads with the runnable wedge); positioning at the **tagline/marketing layer is unchanged** (still on the contested category). Differentiation vs Kagent remains real (engine-agnostic + Compose-and-K8s + no-mandatory-SDK + now zero-secret local eval) but **under-marketed**.

---

## 3. Real vs Aspirational — Capability Status (the EPIC #1370 shift)

Reconciled to live milestone state and PR evidence. The key shift since 2026-06-18 is in the **Day-0 / first-run rows**, which moved from 🟡 partial / 📅 aspirational to ✅ shipped.

| Capability | 2026-06-18 status | Today (2026-06-19) | Proof |
|---|---|---|---|
| **Zero-secret local-LLM path (no paid key)** | 📅 Aspirational (key required) | ✅ **Shipped** | Ollama overlay `COMPOSE_DEMO` [Makefile:22](../../Makefile); PRs #1433 #1437; stories #1374 #1386 closed |
| **One-command demo (`make demo`)** | 📅 Aspirational ("M7 initiative") | ✅ **Shipped** | [Makefile:132](../../Makefile) `demo:` target; story #1360 closed |
| **Runnable hero example completes from CLI** | 🟡 Partial (examples existed, dispatch gated) | ✅ **Shipped** | [code-review-ollama.yaml](../../spec/workflows/examples/code-review-ollama.yaml); PR #1435; story #1376 closed |
| **Capability output visible to user** | ❌ Not surfaced | ✅ **Shipped** | `zynax result` [cmd/zynax/cmd/result.go](../../cmd/zynax/cmd/result.go); PR #1438; story #1378 closed |
| **Adapters tolerate missing secrets** | ❌ Crash-loop | ✅ **Shipped** | PR #1432; story #1375 closed |
| **Quickstart reconciled to real CLI** | 🟡 Drifted from CLI | ✅ **Shipped** | docs/quickstart.md (main); story #1379 closed |
| **Engine-agnostic dispatch (Temporal + Argo)** | ✅ Shipped (M6) | ✅ Shipped | Unchanged (e2e-smoke matrix) |
| **End-to-end capability dispatch** | ✅ Shipped (M6) | ✅ Shipped | Unchanged |
| **mTLS + SBOM/cosign/SLSA** | ✅ Shipped (M6) | ✅ Shipped | Unchanged |
| **OTEL + Uptrace observability** | 🟡 Partial (EPIC O done) | ✅ Shipped (EPIC O complete) | Unchanged |
| **Workflow data-flow (output→input)** | 🟡 Partial (keystone W) | 🟡 Partial | Compiler `output:` work; not the #1370 cluster |
| **Zero-Temporal lightweight engine** | 📅 Aspirational (#1359) | 📅 **Aspirational — deferred to M-dx** | #1359 OPEN, still in M7 milestone |
| **Declarative scenario manifest** | — | 📅 **Aspirational — needs own canvas** | #1385 OPEN (feat, crosses spec boundary) |
| **Declarative context-injection** | — | 📅 **Aspirational — needs own canvas** | #1387 OPEN (feat, crosses boundary) |
| **Hero asciinema cast in README** | 📅 Aspirational | 📅 **Aspirational — human follow-up** | [docs/casts/](../casts/) placeholder only |
| **Web UI** | ❌ None | ❌ None | M8 roadmap |

**Real-vs-aspirational capability split shift:** The Day-0 onboarding surface moved decisively from **aspirational → shipped**. What remains aspirational is now a *thinner, well-scoped tail*: the zero-Temporal engine (deferred, not abandoned), two declarative-manifest features that correctly require their own REASONS Canvas before implementation (ADR-019), and one human content task (the cast). This is a healthy split — the gated items are deliberate deferrals with traceable issues, not narrative gaps.

---

## 4. Adoption Funnel — Live Signals (2026-06-19)

**Current (via `gh api` 2026-06-19):**

```
{ "stars": 1, "forks": 0, "watchers": 0, "open_issues": 43 }
contributors: 3
```

**Open milestones (via `gh api`):**
```
Usable Workflows + Observability (M7)    open: 10   closed: 115
CNCF Sandbox (M8)                         open: 7    closed: 0
Developer Experience (M-dx)               open: 13   closed: 0
User Experience (M-UX)                    open: 2    closed: 0
```

| Metric | 2026-06-18 | 2026-06-19 | Trend | Interpretation |
|---|---|---|---|---|
| **GitHub stars** | 1 | 1 | → | No new awareness signal (no public launch yet) |
| **Forks** | 0 | 0 | → | No external pickup |
| **Watchers** | 0 | 0 | → | Not on external radar |
| **Contributors** | 3 | 3 | → | Bus-factor unchanged; single named maintainer |
| **Open issues** | 52 | **43** | ↓↓ | EPIC #1370 cluster drained 9+ stories in one day |
| **M7 closed** | 93 | **115** | ↑↑ | +22 issues closed; M7 now 10 open / 115 closed (~92% done) |
| **External adopters** | 0 | 0 | → | **Still zero — the largest CNCF blocker** |
| **Latest release** | v0.5.0 | v0.5.0 | → | v0.6.0 not yet tagged (M7 ~92% closed) |

**Time-to-first-workflow audit (the conversion gate, strategy.md §7.2 target: < 15 min, one command):**

| Dimension | 2026-06-18 | 2026-06-19 | Source |
|---|---|---|---|
| Commands to first run | ~6 manual steps (clone → bootstrap → run-local → apply → logs → stop) | **`make demo`** (one command) | [Makefile:132](../../Makefile); story #1360 |
| Paid secret required | Yes (LLM key) | **No** (local Qwen via Ollama) | Ollama overlay; story #1374 #1386 |
| Meaningful visible result | No (`logs` only) | **Yes** (`zynax result`) | [cmd/zynax/cmd/result.go](../../cmd/zynax/cmd/result.go); #1378 |
| Residual friction | Temporal + large image pull | **`ollama pull qwen2.5-coder:3b`** prereq (one-time host model pull); Temporal still in stack | [Makefile:126](../../Makefile); #1359 deferred |

**Funnel verdict:** The conversion gate moved from *"multi-step, needs a paid key, output invisible"* to *"one command, zero paid secrets, review printed."* This is the single biggest funnel improvement in the project's history. **The remaining friction is the `ollama pull` model download and the still-present Temporal dependency** (zero-Temporal mode #1359 deferred to M-dx) — both are now *the* residual Day-0 items, down from a much longer list. **The funnel is fixed at the supply side; the demand side (traffic) is untouched** — signals are flat because no public invitation has been issued.

---

## 5. Beachhead Validation — Is the Hero Use Case Holding?

**Hero use case (strategy.md §6.2):** *"Ship a multi-agent code-review workflow as YAML, run it on Temporal or Argo without a rewrite, watch it execute end-to-end with traces — in under 15 minutes."*

**Validation evidence (strengthened since 2026-06-18):**

1. **The hero workflow now completes from the CLI on a free local model** — [code-review-ollama.yaml](../../spec/workflows/examples/code-review-ollama.yaml) + `make demo` + `zynax result`. Previously the beachhead was "real and dogfooded internally" but a stranger could not reproduce it without a paid key. **Now a stranger can.** This is the decisive beachhead upgrade: external reproducibility.
2. **End-to-end dispatch** unchanged (M6 task-broker + agent-registry; 5 adapters; Argo + Temporal).
3. **Observability** unchanged (EPIC O complete; Uptrace stack; connected traces).
4. **Multi-engine portability** unchanged (TemporalEngine + ArgoEngine in CI matrix).

**Verdict:** ✅ The beachhead is **real, runnable, and now externally reproducible without a paid secret** — a strict improvement. The only remaining gap is the same as before: **external validation** (zero named adopters, zero outside forks). The product has now removed every technical reason a stranger could not validate it themselves; the missing ingredient is the *invitation* (the asciinema cast + a public launch).

---

## 6. Recommendations — Classified by User Type + Adoption Lever

Each tagged **(user type, adoption lever)** for `/plan` handoff. The Day-0 friction recommendations from 2026-06-18 are now substantially **closed** (see §7), which sharpens the priority order toward **distribution and community**.

### 6.1 Critical (gates M8 Sandbox filing)

1. **Record and embed the hero asciinema cast, then issue a public launch** *(developer + product-owner, distribution)*
   - **Why:** The engineering substrate is now done — the workflow completes from the CLI with visible output. The cast is the last mile between "reproducible" and "discoverable," and it is now *recordable* for the first time.
   - **Evidence:** [docs/casts/](../casts/) holds only a placeholder; [docs/quickstart.md](../../quickstart.md) already references casts/. Prior review rec #2 success metric ("`make demo` runs hero in < 15 min") is met; the "≥1 external star/fork" half is not.
   - **Action:** Record `make demo` → `zynax result` as an asciinema cast; embed in README above the fold; then post to one external channel (HN/Reddit/CNCF Slack).
   - **Success metric:** First non-seeded star or fork; ≥1 inbound Discussion from outside.

2. **Recruit a second maintainer from a second org** *(maintainer/governance, community)* — **carried unchanged from 2026-06-18 rec #1.**
   - **Why:** Single maintainer is the top CNCF Sandbox blocker; 6–12 week lead time; contributors flat at 3.
   - **Evidence:** [GOVERNANCE.md](../../GOVERNANCE.md); `gh` contributors=3.
   - **Success metric:** 2nd maintainer reviewing PRs + co-signing releases by M8 gate.

3. **Make the README tagline Kagent-differentiated** *(product/marketing, positioning)* — **carried unchanged from 2026-06-18 rec #4 (Not started).**
   - **Why:** Docs now lead with the runnable wedge, but the README tagline still competes on the contested "control plane for AI agents" category. Now that the zero-secret local path is a genuine differentiator vs K8s-only Kagent, lead with it.
   - **Evidence:** [README.md](../../README.md) tagline; [docs/architecture/2026-05-28-competitive-positioning.md](../../docs/architecture/2026-05-28-competitive-positioning.md).
   - **Success metric:** README leads with *engine-agnostic + zero-secret local eval + Compose-and-K8s*; comparison table referenced in CFP material.

### 6.2 High (gates M7 close-out and v0.6.0)

4. **Close out the M7 first-run tail: cut the `ollama pull` prereq into `make demo`** *(operator/evaluator, Day-0 friction)*
   - **Why:** The one residual onboarding step is a manual `ollama pull qwen2.5-coder:3b` ([Makefile:126](../../Makefile)). Folding the model pull into the demo target (or documenting expected first-run download time) removes the last "why did it fail?" surprise.
   - **Success metric:** `make demo` from a fresh host needs no manual pre-step; first-run time documented.

5. **Schedule the zero-Temporal lightweight engine (#1359)** *(operator/evaluator, Day-0 friction)*
   - **Why:** Deferred to M-dx, but Temporal remains the heaviest residual dependency in the demo stack. Still the right next friction cut after #1370.
   - **Evidence:** #1359 OPEN (M7 milestone); strategy.md §7.1.
   - **Success metric:** `--eval-mode` in-process engine; demo runs with no Temporal container.

6. **Canvas + schedule the declarative scenario/context manifests (#1385, #1387)** *(developer, ecosystem)*
   - **Why:** These are the "configure your **own** scenario declaratively" half of EPIC #1370 — the differentiating self-serve story. Both correctly require their own REASONS Canvas (ADR-019) before code.
   - **Evidence:** #1385, #1387 OPEN; [2026-06-18-ux-roadmap-realignment.md §6](2026-06-18-ux-roadmap-realignment.md).
   - **Success metric:** Aligned canvases for #1385/#1387; a user runs a non-hero scenario without editing code.

### 6.3 Medium (gates M8 adoption posture)

7. **Establish public cadence (monthly digest)** *(maintainer/community, governance)* — **carried from 2026-06-18 rec #5 (Minimal).** Success: ≥3 digests; first external Discussion.

8. **Grow the example library around the beachhead** *(developer, ecosystem)* — **carried from 2026-06-18 rec #7.** Now easier: the Ollama harness makes each new example zero-secret-runnable. Success: ≥2 new runnable workflows.

---

## 7. Longitudinal Delta — Prior Recommendations vs Current Status

Diffing against [docs/product/2026-06-18-product-strategy-review.md §7](2026-06-18-product-strategy-review.md):

| Prior recommendation (2026-06-18) | 2026-06-18 status | 2026-06-19 status | What moved it |
|---|---|---|---|
| **#2 Land first-run keystone + publish flagship demo** | 🟡 In-flight | 🟢 **Substantially closed** | EPIC #1370 shipped `make demo`, runnable Ollama example, `zynax result` (PRs #1432–#1440). Cast still pending → §6 rec #1. |
| **#3 Ship zero-Temporal lightweight eval mode** | 📅 Aspirational | 📅 **Deferred (M-dx)** | #1359 OPEN; sidestepped for now by the zero-secret Ollama path, which removed the *paid-key* barrier without removing Temporal. |
| **#6 Cut Day-0 friction (contributor + no-crash)** | 🟡 Partial | 🟢 **Adapter-crash half closed** | PR #1432 (adapters tolerate missing secrets). Contributor fast-lane still open. |
| **#1 Recruit 2nd maintainer** | ⚠ Open | ⚠ **Open (unchanged)** | No movement; contributors=3. |
| **#4 Kagent-differentiated messaging** | ❌ Not started | 🟡 **Partial** | Docs/quickstart now lead with the runnable wedge (#1379); README tagline unchanged. |
| **#5 Public cadence** | ⚠ Minimal | ⚠ **Open (unchanged)** | No digest posted. |
| **#7 Grow examples** | ✅ Partial | ✅ **Improved** | `code-review-ollama.yaml` added; Ollama harness makes new examples zero-secret. |
| **#8 Dogfood via DevAuto** | ✅ Partial | ✅ **Held** | EPIC #1370 itself was delivered via the SPDD/orchestrate substrate. |
| **#9 Preserve monetization optionality** | ✅ Held | ✅ **Held** | No feature-gating. |

**Net delta:** The **two Day-0 Critical recommendations (#2, #6) substantially closed in 24 hours** via EPIC #1370 — the fastest closure of a Critical adoption recommendation in the review series. The **social Criticals (#1 maintainer, #4 messaging, #5 cadence) are unmoved**, confirming the constraint has shifted cleanly from *engineering friction* to *distribution and community*.

---

## 8. Market & Competitive Reality Check (2026-06-19)

- **Kagent** status unchanged from 2026-06-18 (CNCF Sandbox 2026; K8s-native; no cross-engine vision). No fresh signal this period.
- **Zynax's defensible position — strengthened:** the zero-secret local-LLM evaluation path is now a *concrete* differentiator vs K8s-only Kagent. An evaluator can `make demo` Zynax on a laptop with a free local model; Kagent assumes a cluster. This is exactly the "lowest-friction integration" wedge the prior review named — now demonstrable, not just claimed.
- **Zynax's vulnerability — unchanged:** zero adoption signals, single maintainer, Kagent's CNCF first-mover backing. The corrective action remains **social** (cast, launch, recruitment, messaging), not technical.

**Honest verdict:** The category is still contested and won on community, not architecture. But Zynax has now **removed every technical excuse** an evaluator could have for not trying it. The product is "demo-ready"; the project is not yet "launch-executed."

---

## 9. Risks & Honest Weaknesses (Delta Update)

| Risk | 2026-06-18 | 2026-06-19 | Mitigation |
|---|---|---|---|
| **Day-0 friction (paid key, invisible output, crash-loop)** | High/Open | 🟢 **Closed** | EPIC #1370 (zero-secret Ollama, `zynax result`, no-crash adapters) |
| **Temporal dependency deters evaluators** | High/Open | 🟡 **Reduced** | Paid-key barrier gone; Temporal still in stack (#1359 deferred) |
| **Single maintainer / bus-factor** | Top blocker | **Unchanged — top blocker** | Recruit now |
| **Undifferentiated vs Kagent (README)** | Open | 🟡 **Partial** (docs improved, tagline not) | §6 rec #3 |
| **Zero external adopters** | Open | **Unchanged** | §6 rec #1 (launch) |
| **No hero cast / public launch** | Open | **Unchanged but now unblocked** | Substrate ready; record + launch |

**Net:** Technical Day-0 risk **closed**. Remaining risks are entirely **market/community** — and one is now *unblocked* (the cast can finally be recorded because the demo actually completes).

---

## 10. Conclusion

**In 24 hours, Zynax closed its single most-cited adoption barrier.** EPIC #1370 turned the hero use case from "real but requires a paid key and shows no output" into "one command, zero paid secrets, prints the review" — verified on `main` (`make demo`, `zynax result`, the Ollama overlay, the runnable example, the reconciled quickstart). The real-vs-aspirational split shifted decisively: the Day-0 surface is now **shipped**, and the residual aspirational tail is thin and well-scoped (zero-Temporal #1359 deferred, declarative manifests #1385/#1387 awaiting their own canvases, the asciinema cast a human follow-up).

**The binding constraint has moved one notch downstream — from supply to demand.** The product no longer wastes inbound traffic; it simply has none yet. Adoption signals are flat (1 star, 0 forks, 3 contributors) because no public invitation has been issued. The next highest-leverage unit of work is therefore **distribution and community, not features**: record the now-recordable hero cast, issue a public launch, recruit a second maintainer, and sharpen the README to lead with the zero-secret engine-agnostic wedge that genuinely separates Zynax from Kagent.

**Recommended focus (next 30 days):** (1) Record the hero cast and launch publicly — the engineering is done. (2) Recruit a second maintainer (6–12 wk lead). (3) Make the README tagline Kagent-differentiated. (4) Close the M7 tail (fold `ollama pull` into `make demo`; schedule #1359). These are the moves that convert a now-friction-free product into actual adoption before the M8 Sandbox filing.

---

## 11. Appendix — Sources & Methodology

### A. Live signals (2026-06-19, via `gh api`)
```
stars: 1   forks: 0   watchers: 0   open_issues: 43   contributors: 3
Milestones: M7 open:10 closed:115 | M8 open:7 closed:0 | M-dx open:13 closed:0 | M-UX open:2 closed:0
Latest release: v0.5.0 (2026-06-12) — no v0.6.0 tag yet
```

### B. EPIC #1370 PR evidence (merged 2026-06-18, verified on main)
`#1432` adapters tolerate missing secrets · `#1433` Ollama overlay · `#1434` NotFound non-retryable · `#1435` runnable code-review-ollama example · `#1436`/`#1377` event-injection CLI · `#1437` Qwen default model · `#1438` `zynax result` payload surfacing · `#1440` human-validation guide · plus docs/canvas (`#1379`, `#1382`, `#1383`, `#1396`, `#1397`).

### C. On-disk / on-main artifacts cited
[Makefile:22,132](../../Makefile) (`COMPOSE_DEMO`, `demo:`) · [cmd/zynax/cmd/result.go](../../cmd/zynax/cmd/result.go) · [spec/workflows/examples/code-review-ollama.yaml](../../spec/workflows/examples/code-review-ollama.yaml) · [infra/docker-compose/docker-compose.ollama.yml](../../infra/docker-compose/docker-compose.ollama.yml) · [docs/quickstart.md](../../quickstart.md) · [docs/casts/](../casts/) (placeholder).

### D. Still-gated (verified OPEN in M7)
`#1359` zero-Temporal engine (deferred → M-dx) · `#1385` declarative scenario manifest (needs canvas) · `#1387` declarative context-injection (needs canvas) · hero asciinema cast (human follow-up).

### E. Methodology
Every claim grounded in `file:line`, doc section, issue#, or `gh` result. Shipped/partial/aspirational split applied uniformly. Live metrics pulled from `gh api`, not memory. Longitudinal comparison against the three named baselines; deltas recorded with the PR/issue that moved them. Read-only — no files written, no branch, no PR.
