# REASONS Canvas — named engine-conformance suite (M9.C)

> **All content in this Canvas is Tier 1 (public-safe).**
> Run `/lib:spdd-security-review <path>` before committing.

**Issue:** #1692
**Author:** Oscar Gómez Manresa
**Date:** 2026-07-08
**Status:** Aligned

> Story issues: step 1 → #1620 · steps 2–4 → filed via `/lib:spdd-story` on alignment
> (the suite's name/versioning scheme is the open design question a human aligns first).

---

## R — Requirements

- **Problem.** The portability claim — "write once, run on Temporal or Argo" — is proved
  internally (the e2e matrix runs the same `spec/workflows/examples/` manifests through the
  identical compile→IR→dispatch path on both engines on every infra/service PR, including the
  Workflow CRD GitOps path) but is invisible externally: no name, no version, no published
  per-engine result an adopter can check. ROADMAP M9 exit criterion 3 requires formalising it.
  One known asymmetry exists: the Workflow CRD reconcile e2e assertion runs only on the
  temporal leg (#1620).
- **Done — portability is proved, published, and reproducible:**
  - A named, versioned conformance-suite definition lives in-repo (scenario set, engine legs,
    pass criteria); the existing e2e matrix IS the runner — no second harness.
  - Every release publishes a machine-readable per-engine pass/fail matrix; release notes
    link it (#1692 acceptance criteria).
  - #1620 closed: the CRD reconcile assertion passes on BOTH legs — the suite starts
    symmetric.
  - An engine-adapter author runs the suite against one engine with a single documented
    command and gets the same matrix output locally.
  - The suite is the regression net for the M9.A/M9.B hard-removals (lands early in M9).

## E — Entities

- **Conformance suite** — the named, versioned definition: scenario list (from
  `spec/workflows/examples/`), engine legs (temporal, argo), pass criteria (identical
  observable workflow outcomes per scenario per engine).
- **e2e matrix** (`e2e-smoke.yml`, kind + Helm harness) — the existing dual-engine runner;
  gains result-artifact emission, not new scenarios machinery.
- **Matrix artifact** — machine-readable per-engine, per-scenario pass/fail report generated
  by an e2e run; uploaded per release and linkable from release notes.
- **Engine leg** — one engine adapter exercised by the suite; Temporal and Argo today; the
  suite contract is the onboarding target for engine N+1.
- **Workflow CRD GitOps path** — the `kind: Workflow` CR reconcile route (ADR-043); its e2e
  assertion is temporal-only today (#1620) and must hold on both legs.

```
spec/workflows/examples/ --compile→IR→dispatch--> temporal leg --\
                                                                  >-- matrix artifact --> release notes
                          --compile→IR→dispatch--> argo leg   --/        (named suite vX)
```

## A — Approach

- **WILL:** formalise, not rebuild — name and version the scenario set the e2e matrix already
  runs; emit a machine-readable matrix from the existing workflow; publish it per release
  alongside the release assets; document the one-command local run for adapter authors; close
  the #1620 leg asymmetry first so the suite is honest from day one.
- **WON'T:** no second test harness (the e2e matrix is the runner — ADR-040 delegation
  discipline applied to our own tooling); no new engine adapters (the suite lowers their
  cost; it does not add them); no immediate corpus growth to the Fork A 20-scenario target
  (follow-on stories once the named suite exists — see docs/spdd/471-fork-a/canvas.md); no
  per-PR conformance gate beyond today's e2e (per-release publication; PR cadence unchanged);
  no new ADR — naming/publishing an existing suite is reversible (ADR-015/016/040 govern).
- **Positioning fit:** this epic IS the engine-portability wedge made checkable
  (docs/product/positioning.md). All new user-facing copy (suite README, release-notes line,
  how-to) leads with "one manifest, N engines — proved per release", never generic
  control-plane framing.
- Governing ADRs: ADR-015 (pluggable engines — the invariant under test), ADR-016 (layered
  testing — the suite formalises the e2e tier), ADR-040 (delegation — reuse the harness),
  ADR-043 (CRD path included in conformance).

## S — Structure

- `.github/workflows/e2e-smoke.yml` — matrix-artifact emission + release-asset upload hook.
- `docs/conformance/` (new) — suite definition (scenarios, legs, pass criteria, versioning
  scheme) + adapter-author how-to ("run the suite locally against one engine").
- `spec/workflows/examples/` — the scenario corpus (unchanged contents; gains suite
  membership annotations/manifest list).
- e2e harness assertions — CRD reconcile assertion extended to the argo leg (#1620).
- Release notes template / release workflow — matrix link line.
- No gRPC contracts touched; no service code changes expected.

## O — Operations

> Step 1 is independent and lands first (symmetry before naming). Steps 2–4 are drafted here
> for alignment; their story issues are filed via `/lib:spdd-story` once a human aligns this
> canvas (suite name + versioning scheme are the open design decisions).

> **Naming decision — settled 2026-08-04 (maintainer).** The suite is the
> **Zynax Engine Conformance Suite**, short form **ZECS**. It versions with the platform
> release (first published version: **ZECS v0.8.0**), per this section's "version scheme tied
> to releases" — the suite is a claim about a specific released build, so an independent
> version line would only add a mapping to maintain.
>
> Rationale: the name states what is being tested — that a workflow *engine* conforms — which
> matches this canvas's own vocabulary (engine legs, engine-adapter authors) and reads
> conventionally beside other ecosystem conformance suites. Alternatives considered and
> rejected: "Zynax Workflow Portability Suite" (leads with the promise, not the mechanism —
> better marketing framing, weaker as a conformance artifact) and a bare "Zynax Conformance
> Suite" (leaves room for future adapter/capability profiles, but is vague about today's
> scope, and a profile qualifier can be added later without renaming).
>
> Referenced as `ZECS v<platform version>` in release notes; the definition lives under
> `docs/conformance/`.

1. **Leg symmetry** (#1620, `test:`, ✅ done) — extend the Workflow CRD reconcile e2e assertion
   to the argo engine leg; both legs assert identical reconcile behaviour; e2e green. Root cause
   was not an argo dispatch bug — the CR reconciles to `Dispatched=True` on both legs — but the
   assertion's `kubectl get workflow` short name resolved to the Argo CRD (`workflows.argoproj.io`,
   installed only on the argo leg) instead of `workflows.zynax.io`. The script now pins the
   fully-qualified name and the `matrix.engine == 'temporal'` guard is dropped.
2. **Suite definition** (#1773, `docs:`, ✅ done) — **ZECS v0.8.0** published under
   `docs/conformance/`: definition (scenarios, legs, pass criteria, versioning) plus
   `scenarios.yaml`, a membership manifest annotating `spec/workflows/examples/` (no shadow
   copy). The pass criteria are transcribed from what `scripts/e2e/*.sh` actually assert, so
   the definition ships a gap list: 4 scenarios, only `e2e-demo.yaml` on both legs; the legs
   assert different depths (enforced intersection = terminal status); output equality and the
   failure path are temporal-only; the CRD scenario asserts dispatch, not completion; the argo
   leg is advisory on `main` (G1–G6). Membership↔corpus↔engine-leg drift is caught by
   `zynax-ci check conformance` (`make check-conformance` + an unconditional `conformance-check`
   CI job) — legs are read from the engine-adapter's selectable engines and the e2e matrix, so
   no engine name is hardcoded. No second harness; PR e2e cadence unchanged.
3. **Matrix artifact** (#1774, `ci:`, ✅ done) — every e2e run emits `zecs-matrix.json`, one JSON
   document covering **every** engine the engine-adapter can be configured with: each leg records
   its per-scenario step outcomes, a fan-in `ZECS matrix` job renders them, and the artifact is
   retained per run (90 d; per-leg inputs 14 d). Advisory by construction — it reports, it never
   gates, and it cannot turn an e2e leg red. Correctness property, asserted not intended: only an
   observed success renders `PASS`; a leg that never ran is present as `NOT_RUN`, never omitted;
   and the two skips stay distinct (`not_in_leg` = the manifest says this leg does not run it, vs
   `not_executed` = it was meant to and produced nothing, which makes the leg `INCOMPLETE` and the
   run `complete: false`). Enforcement is carried per leg and the required legs aggregate
   separately (`enforced_result`), so an advisory red argo leg never reads as merge-blocking (G6).
   `make conformance-matrix ENGINE=<engine>` reproduces the same document locally by running that
   leg's manifest runners.
4. **Release publication** (story on alignment, `ci:`/`docs:`) — release flow uploads/links
   the matrix; release-notes template gains the conformance line; adapter-author how-to
   published under docs/conformance/.

## N — Norms

- Commit hygiene: DCO `Signed-off-by` + `Assisted-by: Claude/<model>`; SSH-signed; one PR per
  story; subjects ≤ 72 chars.
- Workflow edits: banner-marked image refs via `make sync-images` only (ADR-024); workflow
  lint (actionlint) green; respect the fork-PR read-only-token patterns in `.github/`.
- Test placement per ADR-016: suite assertions live at the e2e tier; any new gRPC boundary
  assertion would need a `.feature` first (none expected).
- Docs: repo-relative links; Diátaxis placement (how-to for adapter authors, reference for
  the suite definition).
- Requirements change → `/lib:spdd-prompt-update` first; never code ahead of the canvas.

## S — Safeguards (second S)

### Context Security (mandatory before committing this Canvas)
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no email addresses
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /lib:spdd-security-review passed on this file

### Feature Safeguards
- Never build a second harness — the suite is a NAME + REPORT over the existing e2e matrix;
  a parallel runner would fork the truth (ADR-040 discipline).
- Never publish a matrix from a run that skipped a leg — a partial matrix presented as
  conformance is worse than none; skipped legs must render as SKIPPED, never PASS.
- Never hardcode engine names in suite logic — legs enumerate from the engine-adapter
  registry/interface (ADR-015); the suite must absorb engine N+1 without edits to its core.
- Never gate PRs on the full conformance run — per-release cadence only; the PR-leg e2e gate
  stays exactly as it is.
- Never let the suite's scenario corpus drift from `spec/workflows/examples/` — one corpus,
  annotated; no shadow copies.
