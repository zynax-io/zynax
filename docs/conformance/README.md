<!-- SPDX-License-Identifier: Apache-2.0 -->
# Zynax Engine Conformance Suite (ZECS)

> **Reference.** What the suite *is* — scenarios, engine legs, pass criteria, versioning.
> Running it against your own engine adapter is a how-to and lands with step 4
> ([#1775](https://github.com/zynax-io/zynax/issues/1775)).

**ZECS** is the named, versioned set of workflow scenarios Zynax executes on **every** engine
adapter to substantiate one claim: *one manifest, N engines*. It is a **name and a report over
the existing e2e matrix** — not a second test harness. The runner is
[`.github/workflows/e2e-smoke.yml`](../../.github/workflows/e2e-smoke.yml) and the scripts under
[`scripts/e2e/`](../../scripts/e2e/); ZECS adds no execution machinery of its own (ADR-040
delegation discipline, applied to our own tooling).

| | |
|---|---|
| Current version | **ZECS v0.8.0** (first published) |
| Engine legs | `temporal`, `argo` |
| Scenarios | 4 (1 runs on both legs; 3 run on one leg — see [Known gaps](#7--known-gaps)) |
| Corpus | [`spec/workflows/examples/`](../../spec/workflows/examples/) — annotated, never copied |
| Membership manifest | [`scenarios.yaml`](scenarios.yaml) |
| Consistency check | `make check-conformance` |
| Result matrix | `zecs-matrix.json` — [§10](#10--the-result-matrix); emitted per e2e run, reproducible locally with `make conformance-matrix ENGINE=<engine>` |
| Published results | not yet linked from a release — per-release publication is step 4 ([#1775](https://github.com/zynax-io/zynax/issues/1775)) |

---

## 1 — What ZECS is, and what it is not

**Is:** a fixed set of workflow scenarios, each submitted through a documented entry point,
executed on every engine leg, with the assertions each leg makes written down verbatim below.

**Is not:**

- **Not a runner.** No ZECS-owned harness exists. Anything ZECS claims is asserted by a script
  in `scripts/e2e/` that already runs in CI. If a criterion is not enforced there, it is listed
  in [Known gaps](#7--known-gaps) instead of stated as a pass criterion.
- **Not a PR gate.** ZECS does not change when or how e2e runs on pull requests. See
  [Cadence](#9--cadence-and-enforcement).
- **Not a functional test suite for the platform.** Compile-tier and unit-tier coverage of the
  example corpus (for instance `services/workflow-compiler/internal/api/examples_test.go`) is
  engine-independent and is therefore **not** conformance evidence.
- **Not proof that a workflow produces identical *outputs* on every engine.** Read
  [Pass criteria](#5--pass-criteria-what-the-runner-actually-asserts) before citing ZECS: today
  the cross-leg guarantee is terminal-outcome parity, not output parity.

## 2 — Versioning

ZECS versions with the **platform release**: the suite published alongside Zynax `v0.8.0` is
`ZECS v0.8.0`. Cite it as `ZECS v<platform version>`. There is no independent version line — the
suite is a claim about a specific released build, so a separate number would only add a mapping
to maintain. A change to the scenario set, the leg set, or the assertions takes effect in the
next platform release and is described in that release's notes.

The naming decision and the rejected alternatives are recorded in
[`docs/spdd/1692-engine-conformance-suite/canvas.md`](../spdd/1692-engine-conformance-suite/canvas.md).

## 3 — Engine legs

A **leg** is one engine adapter exercised by the suite. The leg set is **not** authored in this
document. It derives from the engines the engine-adapter can be configured with (ADR-015):

- **Source of truth:** the engines selectable via `ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE` in
  `services/engine-adapter/cmd/engine-adapter/main.go` (the `buildEngine` switch and its
  engine-name constants — the only place engine names appear outside that switch).
- Everything else *mirrors* that set: the `matrix.engine` list in `e2e-smoke.yml`, the `legs:`
  key in [`scenarios.yaml`](scenarios.yaml), and the table below.
- `make check-conformance` fails when those three disagree, so adding engine N+1 to the
  engine-adapter makes the suite red until the leg is really run — the suite absorbs a new
  engine without any edit to its logic (no engine name is hardcoded in the check).

| Leg | Engine implementation | Status in CI |
|-----|----------------------|--------------|
| `temporal` | `TemporalEngine` (`services/engine-adapter/internal/infrastructure/temporal.go`) | `e2e smoke (temporal)` — a **required** status check on `main` (path-conditional, with a shim for non-e2e PRs) |
| `argo` | `ArgoEngine` (`services/engine-adapter/internal/infrastructure/argo_engine.go`) | `e2e smoke (argo)` — **advisory**: it runs on every e2e-relevant PR but is not in the required-check set (decision recorded on [#1092](https://github.com/zynax-io/zynax/issues/1092)) |

> The asymmetry in the last column is real and matters to anyone reading a ZECS result: a red
> `argo` leg does not block a merge today. Every leg's enforcement is carried in the result
> matrix ([§10](#10--the-result-matrix)); it is mirrored machine-readably in
> [`scenarios.yaml`](scenarios.yaml) under `enforcement:`.

## 4 — Scenarios (ZECS v0.8.0)

| Scenario | Source | Submitted via | Legs that run it |
|----------|--------|---------------|------------------|
| `echo-happy-path` | `spec/workflows/examples/e2e-demo.yaml` | REST `POST /api/v1/apply` | `temporal`, `argo` |
| `workflow-crd-reconcile` | `scripts/e2e/manifests/workflow-cr.yaml` (fixture) | `kubectl apply` of a `workflows.zynax.io` CR (ADR-043) | `temporal`, `argo` |
| `hello-world-outputs` | `spec/workflows/examples/hello-world.yaml` | REST `POST /api/v1/apply` | `temporal` only |
| `capability-timeout-failure` | generated at runtime by `scripts/e2e/e2e-failure.sh` | REST `POST /api/v1/apply` | `temporal` only |

The machine-readable form — including the per-leg runner script and the `not_run` reasons — is
[`scenarios.yaml`](scenarios.yaml).

`capability-timeout-failure` is deliberately generated rather than committed: it references a
capability no agent serves, so publishing it under `spec/workflows/examples/` would advertise a
broken workflow as a reference.

## 5 — Pass criteria (what the runner actually asserts)

**Suite-level pass criterion.** A ZECS run passes for a leg when every scenario whose
`status: run` for that leg exits 0. A scenario that does not run on a leg is reported
**SKIPPED**, never PASS. A run in which a leg was skipped entirely is not a conformance result
for that leg.

**Scenario-level criteria are stated below verbatim from the scripts** — including what is *not*
asserted. This section is the contract; do not paraphrase it upward into stronger claims.

### 5.1 `echo-happy-path` — `spec/workflows/examples/e2e-demo.yaml`

**temporal leg** (`scripts/e2e/e2e-happy.sh`) asserts:

1. `POST /api/v1/apply` returns a `run_id`.
2. `GET /api/v1/workflows/{run_id}` reaches a terminal success status within 120 s, matching the
   alias set `succeeded | completed | *COMPLETED | *SUCCEEDED`; any terminal failure alias fails
   the run.
3. The JetStream stream `ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW` holds at least one message, and the
   last message on `zynax.v1.engine-adapter.workflow.state.entered` carries the
   `workflow.state.entered` CloudEvent type.
4. A `zynax.v1.engine-adapter.workflow.completed` CloudEvent is on that stream.
5. `MemoryService.Set` followed by `MemoryService.Get` returns the same sentinel value.

**argo leg** (`scripts/e2e/e2e-argo.sh`) asserts:

1. `POST /api/v1/apply?engine=argo` returns a `run_id`.
2. The same terminal-success alias check on `GET /api/v1/workflows/{run_id}` (the gateway
   projection).
3. The Argo `Workflow` CR named after the `run_id` reaches `.status.phase == Succeeded` — the
   engine's own source of truth, not just the gateway's view.

**Not asserted on either leg:** the workflow's outputs; per-state ordering or state-entry counts;
the run's duration. **Not asserted on the argo leg:** any CloudEvent, and the memory plane.
**Soft on the temporal leg:** whether the observed `state.entered` CloudEvent belongs to *this*
run — a payload that does not reference the `run_id` produces a warning, and the assertion still
passes on stream-level evidence.

The memory-service roundtrip (temporal step 5) is a *platform-plane* check that shares this
script; it exercises no part of the workflow run and is not engine-portability evidence.

### 5.2 `workflow-crd-reconcile` — `scripts/e2e/manifests/workflow-cr.yaml`

Runs on **both** legs (`scripts/e2e/e2e-workflow-crd.sh`); the CR pins no engine, so each leg
reconciles through whichever engine it deployed. Asserts:

1. Within 90 s the CR's status carries `Dispatched=True` **and** a non-empty `status.runID` — the
   controller compiled and submitted through the same path REST `apply` uses.
2. The status is a thin mirror: no key outside
   `observedGeneration | workflowID | runID | engine | conditions` (ADR-040 §3).
3. Idempotency: a metadata-only reconcile (annotation poke, `spec` unchanged) leaves `runID` and
   `observedGeneration` unchanged after 8 s — no duplicate dispatch.
4. Admission (ADR-045): a CR whose `spec.engine` is outside the namespace allow-list is denied at
   admission with the allow-list message; an allowed engine admits. **Skipped with a log line**
   when the ValidatingAdmissionPolicy is not installed.

**Not asserted:** that the dispatched run ever completes. Conformance for this scenario stops at
*dispatch*, on purpose — the CRD front-end is thin by design (ADR-043) and holds no run state.

### 5.3 `hello-world-outputs` — `spec/workflows/examples/hello-world.yaml`

**temporal leg only** (`scripts/e2e/hello-world-smoke.sh`; the step is gated
`if: matrix.engine == 'temporal'` in `e2e-smoke.yml`). Asserts:

1. Terminal success, same alias set as `echo-happy-path`.
2. `GET /api/v1/workflows/{run_id}/logs` contains `WorkflowExecutionCompleted`.
3. `GET /api/v1/workflows/{run_id}/outputs` returns `message` **exactly equal** to the string
   declared in the manifest (`Hello from Zynax`).

This is the **only** output-equality assertion in ZECS, and it runs on one leg. See gap **G3**.

### 5.4 `capability-timeout-failure` — generated fixture

**temporal leg only** (`scripts/e2e/e2e-failure.sh`). Asserts:

1. A workflow whose first state invokes an unserved capability reaches a **terminal failure**
   status within 120 s (reaching success fails the scenario).
2. A `zynax.v1.engine-adapter.workflow.failed` CloudEvent is on
   `ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW`.

**Soft:** the failure *reason* check. The script passes on a terminal failure even when the
reason string mentions no timeout, because the reason field is best-effort across
serializations. The observed elapsed time is logged against `ZYNAX_CAPABILITY_TIMEOUT` for
information only and is never enforced.

## 6 — What cross-leg parity actually means today

The portability claim ZECS supports at v0.8.0, stated at the strength the runner enforces:

> The same workflow manifest, compiled once through the same `compile → IR → dispatch` path,
> reaches the same **terminal outcome** on both the Temporal and the Argo engine — confirmed on
> each engine's own source of truth — and a `Workflow` CR reconciles to a dispatched run with an
> identical thin status and identical idempotency behaviour on both engines.

Everything stronger than that sentence is currently **unproved**. In particular the intersection
of the two legs' assertions for the shared scenario `echo-happy-path` is *terminal status
only*: the temporal leg additionally asserts eventing and the memory plane, the argo leg
additionally asserts the engine-native phase, and neither asserts workflow outputs.

## 7 — Known gaps

Recorded rather than hidden: a conformance suite that overstates what it verifies is worse than
none. These are the honest deltas between "identical observable workflow outcomes per scenario
per engine" and what ZECS v0.8.0 enforces.

| ID | Gap | Consequence |
|----|-----|-------------|
| **G1** | Corpus coverage: 2 of the 9 `kind: Workflow` manifests in `spec/workflows/examples/` are executed by ZECS, and only 1 (`e2e-demo.yaml`) on both legs | The portability claim rests on one shared workflow shape (single capability, single transition) |
| **G2** | Assertion depth differs per leg for the shared scenario — the legs are not asserting the same things | "Both legs green" is weaker than "both legs behaved identically"; the enforced intersection is terminal status |
| **G3** | Declared-output equality is asserted on the temporal leg only (`hello-world-outputs`) | Output portability across engines is unproven |
| **G4** | `workflow-crd-reconcile` asserts dispatch, not completion | The CRD path is proved to *start* portable runs, not to finish them |
| **G5** | The failure path (`capability-timeout-failure`) runs on the temporal leg only | Failure-semantics portability is unproven |
| **G6** | The `argo` leg is advisory, not required, on `main` | A red argo leg does not block a merge. The result matrix carries `enforcement` per leg and aggregates the required legs separately as `enforced_result` ([§10](#10--the-result-matrix)), so a result never claims more authority than branch protection gives it — but the gap itself is open |

Closing G1–G5 is corpus and assertion work that belongs to follow-on stories (the Fork A
20-scenario target referenced by the epic canvas); G6 is a branch-protection decision. None of
them is in scope for the suite *definition*.

## 8 — Corpus membership and how mismatches surface

One corpus, annotated — never copied. [`scenarios.yaml`](scenarios.yaml) references paths under
`spec/workflows/examples/` and adds no shadow copy of any manifest. Corpus manifests that are not
ZECS scenarios are listed explicitly under `non_members:` with a reason, so a manifest can never
fall out of the suite silently.

`make check-conformance` (`zynax-ci check conformance`) is the drift guard. It fails on:

1. a scenario whose `source.path` does not exist;
2. a `kind: Workflow` manifest in the corpus that is neither a scenario source nor a declared
   non-member (and, in reverse, a `non_members` entry that names no real corpus manifest, or one
   that is also a scenario source);
3. a `legs:` list that disagrees with the engines selectable in the engine-adapter;
4. a `legs:` list that disagrees with the `matrix.engine` list in `e2e-smoke.yml`;
5. a scenario that omits an entry for a declared leg, a `not_run` leg with no reason, or a `run`
   leg whose runner script does not exist;
6. duplicate or empty scenario ids;
7. a leg that does not declare its `enforcement`, or a `run` leg that declares no `ci_step` —
   the two fields the result matrix ([§10](#10--the-result-matrix)) depends on.

It deliberately stops short of resolving each `ci_step` against the step ids in `e2e-smoke.yml`.
A mismatch there is caught one layer later and just as loudly — the cell renders `not_executed`
and the leg `INCOMPLETE` in the next run's matrix — and `actionlint` already fails a workflow
whose recording step references a step id that no longer exists.

**Why a static check and not a test run:** rules 1–6 are answerable by reading files, in about a
second, with no cluster. Running the suite to discover that a manifest was renamed would cost a
45-minute two-cluster matrix and would only be reached on e2e-relevant PRs. The check runs
unconditionally in CI (`conformance-check` job in `ci.yml`) and locally as part of
`make validate-spec` → `make test`. It is emphatically **not** a conformance run and reports no
pass/fail per engine — it only proves the *definition* is internally consistent with the repo.

## 9 — Cadence and enforcement

| | |
|---|---|
| ZECS scenarios execute | on the existing gated e2e matrix: PRs touching `helm/**`, `services/**`, `engine-adapter/**`, `scripts/e2e/**`, or `e2e-smoke.yml`, plus `workflow_dispatch` |
| PR gating | unchanged by ZECS. `e2e smoke (temporal)` stays required (with the path shim); `e2e smoke (argo)` stays advisory. **No PR is gated on a full ZECS run** |
| Consistency check | every PR (`conformance-check`), seconds, no cluster |
| Per-engine matrix | emitted by the `ZECS matrix` job on every e2e run — **advisory, never a gate** ([§10](#10--the-result-matrix)). Linking it from a release is step 4 |

## 10 — The result matrix

Every e2e run emits one machine-readable result document, `zecs-matrix.json`, as a run artifact
(90-day retention; the per-leg inputs are kept 14 days). It is **JSON**, not YAML: it is written
and read by machines — the release flow, `jq`, an adapter author's script — and JSON is the format
those already speak. The job summary embeds that same document verbatim, so what a human reads in
the run and what a consumer parses cannot disagree. (A rendered table belongs with publication,
step 4 — [#1775](https://github.com/zynax-io/zynax/issues/1775).)

The matrix is a **report over the existing e2e matrix**, produced by the same run — no second
harness (ADR-040). Each leg records its per-scenario step outcomes; the `ZECS matrix` job renders
them. It reports and exits 0 even on a red matrix: nothing here gates a PR ([§9](#9--cadence-and-enforcement)).

### 10.1 Shape

```jsonc
{
  "schema_version": "1",
  "version": "v0.8.0",            // ZECS version = platform release (§2)
  "revision": "<sha>",            // what was under test
  "run_url": "…/actions/runs/…",
  "result": "PASS|FAIL|INCOMPLETE",          // all legs
  "enforced_result": "PASS|FAIL|INCOMPLETE", // legs whose check is REQUIRED on main
  "complete": true,
  "legs": [{
    "engine": "temporal",
    "enforcement": "required|advisory|unknown",  // §3, gap G6
    "executed": true,
    "complete": true,
    "result": "PASS|FAIL|INCOMPLETE|NOT_RUN",
    "scenarios": [{
      "id": "echo-happy-path",
      "result": "PASS|FAIL|SKIPPED",
      "planned": true,
      "skip_kind": "not_in_leg|not_executed",  // only when SKIPPED
      "asserts": ["terminal-success-status", "…"],  // what THIS leg checked (§5)
      "observed": "success"
    }]
  }],
  "notes": ["…"]
}
```

### 10.2 Reading it honestly

**Read `complete` before `result`.** A matrix is a conformance result only when `complete` is
true. These are the three things it is built not to let you misread:

1. **A leg that did not run is present and `NOT_RUN` — never absent.** Every engine the
   engine-adapter can be configured with gets a row, whether or not it produced any evidence. A
   missing leg artifact (cancelled job, lost runner) yields `NOT_RUN`, `complete: false`. You
   cannot mistake a one-leg run for a full one, because the other leg is right there saying it
   did not run.
2. **The two skips are different, and the dangerous one is loud.** `not_in_leg` means the
   membership manifest says this leg does not run that scenario — expected, carries the manifest's
   reason and its gap id (G3, G5), and does **not** make the leg incomplete. `not_executed` means
   the scenario was meant to run here and produced no success or failure — the leg becomes
   `INCOMPLETE` and the run `complete: false`. There is no path from "did not run" to `PASS`:
   only an observed success renders `PASS`, an observed failure renders `FAIL`, and *everything*
   else (skipped, cancelled, absent, unrecognised) renders `SKIPPED`.
3. **`PASS` on two legs is not proof the legs verified the same behaviour.** The legs assert
   different things (gap **G2**) — compare each cell's `asserts` and read
   [§6](#6--what-cross-leg-parity-actually-means-today). And `result` covers all legs while
   `enforced_result` covers only the legs whose e2e check is *required* on `main`: a red advisory
   leg makes `result` `FAIL` and leaves `enforced_result` `PASS`, because it blocks no merge
   today (gap **G6**).

### 10.3 Producing it locally

One command, one engine, against a cluster you brought up — same tool, same document, same schema
as CI:

```bash
E2E_ENGINE=<engine> scripts/e2e/cluster-up.sh      # your engine's stack (see scripts/e2e/README.md)
make conformance-matrix ENGINE=<engine>            # runs that leg's scenarios; writes zecs-matrix.json
```

It executes exactly the `runner:` scripts the manifest declares `run` for that leg — the same
scripts CI runs — and records each one's exit status. The legs you did not run appear as
`NOT_RUN`, so the document says plainly that a single-engine run is not a suite result. Adding
engine N+1 to the engine-adapter makes it a row here with no change to the tool.

`zynax-ci conformance matrix --help` documents the flags, including the CI form
(`--outcomes <dir>`) that renders a document from a run's recorded step outcomes.

## 11 — Related

- Epic [#1692](https://github.com/zynax-io/zynax/issues/1692) ·
  canvas [`docs/spdd/1692-engine-conformance-suite/canvas.md`](../spdd/1692-engine-conformance-suite/canvas.md)
- [ADR-015 — pluggable workflow engines](../adr/ADR-015-pluggable-workflow-engines.md) (the
  invariant under test; the leg set derives from it)
- [ADR-016 — layered testing strategy](../adr/ADR-016-layered-testing-strategy.md) (ZECS
  formalises the e2e tier; it adds no tier)
- [ADR-040 — Kubernetes-native delegation boundary](../adr/ADR-040-kubernetes-native-delegation-boundary.md)
  (why there is no second harness)
- [ADR-043 — Workflow CRD front-end](../adr/ADR-043-workflow-crd-front-end.md) (the CRD scenario)
- [ADR-045 — admission-policy delegation](../adr/ADR-045-admission-policy-delegation.md) (the
  admission assertions inside the CRD scenario)
- Harness reference: [`scripts/e2e/README.md`](../../scripts/e2e/README.md)
