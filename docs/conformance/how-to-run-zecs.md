<!-- SPDX-License-Identifier: Apache-2.0 -->
# How to run ZECS against your engine adapter

> **How-to.** The tasks an engine-adapter author performs, in order, to get their engine
> reported as a ZECS leg. What the suite *is* — scenarios, what each leg actually asserts,
> known gaps, the result-document shape — is the reference, [`README.md`](README.md); this page
> does not repeat it and links to it wherever the answer belongs there.

You are adding engine N+1 to the engine-adapter (ADR-015) and want the same, checkable
conformance evidence the `temporal` and `argo` legs have. Nothing below asks you to write a test
harness: ZECS is a name and a report over the e2e matrix that already exists (ADR-040), so your
work is to make your engine *selectable*, *declared*, and *run*.

**You need:** Docker, `kind`, `kubectl`, `helm`, and a Go toolchain matching
`cmd/zynax-ci/go.mod` (the two `make` targets below run `go run` directly, not the tools image).
Your engine's control plane must be installable into a kind cluster.

---

## 1 — Make your engine selectable

Add the engine-name constant and its `buildEngine` case in
`services/engine-adapter/cmd/engine-adapter/main.go`. That switch is the single source of truth
for the leg set — no engine name is hardcoded anywhere in the suite.

Then run the drift guard:

```bash
make check-conformance
```

It now **fails**, naming your engine:

```text
engines selectable in services/engine-adapter/cmd/engine-adapter/main.go but not declared as
ZECS legs: [<your-engine>] — add the leg or record why it is unconformant
```

That failure is the suite noticing your engine by itself. Red is correct here: until the leg is
really run, the suite must not look complete.

## 2 — Declare the leg

Edit [`scenarios.yaml`](scenarios.yaml):

1. add your engine to `legs:`;
2. add an `enforcement:` entry — start at `advisory` (it is what a leg with no branch-protection
   check honestly is; see [README §3](README.md#3--engine-legs));
3. add an entry for your leg to **every** scenario: either `status: run` with a `runner:` and a
   `ci_step:`, or `status: not_run` with a `reason:` and a `gap:`. Omission is a check failure —
   a leg that never ran must never read as a pass.

Re-run `make check-conformance` until it is green. Reuse the existing `runner:` scripts wherever
your engine can satisfy them; a new script per engine would fork the truth.

## 3 — Bring up a cluster running your engine

```bash
E2E_ENGINE=<engine> scripts/e2e/cluster-up.sh
```

See [`scripts/e2e/README.md`](../../scripts/e2e/README.md) for the harness's own options. If your
engine needs a control plane installed first, add it there, behind the same `E2E_ENGINE` switch
the `argo` leg uses — not in a parallel script.

## 4 — Run the suite and read the matrix

One command, one leg:

```bash
make conformance-matrix ENGINE=<engine>
```

It executes exactly the `runner:` scripts your manifest declares `run` for that leg — the same
scripts CI runs — and writes two files that are two views of one document:

- `bin/zecs-matrix.json` — the result document (identical tool, schema and semantics as CI's
  artifact);
- `bin/zecs-matrix.md` — the table, **rendered from that JSON**, the same rendering a release
  publishes.

Re-render at any time without re-running anything:

```bash
make conformance-render MATRIX=bin/zecs-matrix.json
```

Shape of the rendering (abridged — a one-leg run on a two-leg suite):

```markdown
### Engine conformance — ZECS v0.8.0

ZECS v0.8.0 — argo NOT_RUN (advisory), temporal PASS (required) · required legs: PASS
· all legs: INCOMPLETE · INCOMPLETE run — not a conformance result

| Scenario | argo (advisory) | temporal (required) |
| --- | --- | --- |
| `echo-happy-path` | SKIPPED — not executed | PASS |
| `hello-world-outputs` | SKIPPED — not in leg (G3) | PASS |
```

## 5 — Read the result honestly

Three things this document is built not to let you misread — the full rules are
[README §10.2](README.md#102--reading-it-honestly):

- **`complete` before `result`.** Your single-leg local run is *never* a suite result: the legs
  you did not run are present as `NOT_RUN` and the run is `complete: false`.
- **`SKIPPED — not in leg` is not `SKIPPED — not executed`.** The first is your manifest saying
  the leg does not run that scenario (expected, gap-tracked). The second means it was meant to
  run here and produced nothing — that one makes your leg `INCOMPLETE`.
- **A green leg is only as strong as what that leg asserts.** Before quoting a PASS, read
  [README §5](README.md#5--pass-criteria-what-the-runner-actually-asserts) (verbatim assertions)
  and [§7](README.md#7--known-gaps) (what ZECS does not yet prove).

## 6 — Get your leg into CI

Local evidence is not published evidence. To make your leg appear in the matrix every e2e run
emits:

1. add your engine to `jobs.e2e.strategy.matrix.engine` in
   [`.github/workflows/e2e-smoke.yml`](../../.github/workflows/e2e-smoke.yml) (the drift guard
   already requires this list to equal `legs:`);
2. give each scenario step an `id:` **equal to** that scenario's `ci_step` for your leg;
3. add `<ci_step>=${{ steps.<id>.outcome }}` to the *Record ZECS scenario outcomes* step.

`make check-conformance` resolves every `ci_step` against the job's real step ids, so a rename
fails at PR time rather than surfacing later as an `INCOMPLETE` leg in a published matrix.

Your leg starts advisory. Making it a required check on `main` is a branch-protection decision,
recorded per leg in `enforcement:` and carried into every published result — see
[README §9](README.md#9--cadence-and-enforcement).

## 7 — What a release then says about your engine

Every release attaches `zecs-matrix.json` and publishes the rendered table in its notes, with
your leg's enforcement shown next to its result — see
[README §11](README.md#11--publication-per-release). Nothing more is required of you: once the
leg is declared and running, publication picks it up with no edit to the release flow.

---

## Troubleshooting

| Symptom | What it means | Fix |
|---|---|---|
| `make check-conformance`: *engines selectable … not declared as ZECS legs* | step 1 done, step 2 not | declare the leg in `scenarios.yaml` |
| `make check-conformance`: *ci_step … is not the id of any step* | the manifest and the workflow disagree on a step id | rename in both, or fix the typo |
| Your scenario renders `SKIPPED — not executed`, leg `INCOMPLETE` | the leg produced no outcome under that `ci_step` key | check the step id, and that the step ran at all |
| Other engines render `NOT_RUN` | expected for a single-leg local run | run the other legs, or read `complete: false` as intended |
| `conformance render`: *matrix schema_version …* | the document came from a different revision's tool | render it with the `zynax-ci` built from that revision |
| `make conformance-matrix` exits 0 with `FAIL` cells | the emitter reports; it never gates | read the leg's log output above the document |

## Related

- [`README.md`](README.md) — the ZECS reference (scenarios, legs, pass criteria, gaps, matrix)
- [`scenarios.yaml`](scenarios.yaml) — the membership manifest you edit in step 2
- [ADR-015 — pluggable workflow engines](../adr/ADR-015-pluggable-workflow-engines.md)
- [ADR-040 — Kubernetes-native delegation boundary](../adr/ADR-040-kubernetes-native-delegation-boundary.md)
  — why there is no second harness to port
