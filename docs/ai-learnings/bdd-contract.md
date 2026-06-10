# Learnings: BDD / Contract Engineer

> Format: each entry has `Seen in:` (issue/session) and `Date:` (YYYY-MM-DD).
> Updated by `/m6-learn` after each batch. Human-reviewed before merging.

---

## Effective patterns

- **Commit the `.feature` file first, in its own commit — always before any implementation.**
  ADR-016 is enforced in PR review. A PR that combines the `.feature` file and implementation
  in a single commit will be asked to split. Two-commit pattern:
  `Commit 1: feat(protos): add <service>.feature — 6 scenarios`
  `Commit 2: feat(<service>): implement gRPC method + step definitions`
  Seen in: M5.C task-broker + agent-registry BDD. Date: 2026-06-06.

- **Background steps are for shared preconditions across ALL scenarios in a feature — not setup for one.**
  Using `Background:` for a single scenario's precondition forces unnecessary setup for every
  other scenario. Use `Background:` only for invariants (e.g., "service is running").
  Seen in: M5.C feature file reviews. Date: 2026-06-06.

- **`Scenario Outline` + `Examples:` table is the correct pattern for multiple similar scenarios.**
  Never write N identical scenarios differing only in data — use a table.
  Seen in: M5.C task-broker feature files. Date: 2026-06-06.

- **Step definitions save the response and error in struct fields — Then steps assert, When steps act.**
  Never assert in a When step. Never perform actions in a Then step. The struct pattern
  (`s.lastResp`, `s.lastErr`) is the standard in this repo.
  Seen in: M5.C step definition implementation. Date: 2026-06-06.

- **Every gRPC method needs ≥4 scenarios: happy path + NotFound + InvalidArgument + (method-specific).**
  Fewer than 4 scenarios per method is a review blocker. The fourth scenario is usually
  AlreadyExists, PermissionDenied, or a data-shape edge case.
  Seen in: M5.C BDD reviews. Date: 2026-06-06.

---

## Edge cases discovered

- **`godog.ScenarioContext.Step` regex must not use named capture groups — only positional.**
  Named groups (`(?P<name>...)`) cause godog to fail step matching. Use `([^"]*)` etc.
  Seen in: M5.C step definition compilation. Date: 2026-06-06.

- **`buf breaking` treats `optional` field additions as breaking in proto3 syntax.**
  In proto3 all fields are implicitly optional. Adding `optional` keyword to an existing
  field IS a breaking change (changes the wire format). Add new fields with new field numbers.
  Seen in: M6.I event-bus proto design. Date: 2026-06-06.

- **Integration test containers (testcontainers-go) must be explicitly terminated in `TestMain`.**
  If the container survives the test run, it occupies the port and causes flaky failures on
  subsequent runs in the same CI job. Always `defer container.Terminate(ctx)`.
  Seen in: M5.C agent-registry BDD integration tests. Date: 2026-06-06.

---

## Failed approaches

- **Writing step definitions that call production gRPC servers directly (not test containers).**
  BDD tests are integration tests that must be self-contained. Calling a live service from a
  step definition makes the test dependent on environment state and non-deterministic.
  Always spin up a test server (testcontainers-go or in-process gRPC server) in the step suite.
  Seen in: Early BDD design. Date: 2026-06-06.

---

## Proposed expert prompt updates

*(none yet — populate after first batch of BDD contract expert sessions)*

## Session — 2026-06-09 (issue #801)

### Effective patterns
- Adding a brand-new `.proto` file is always non-breaking for `buf breaking`; verify with `buf breaking --against 'https://...#branch=main,subdir=protos'` to match the pr-checks.yml gate exactly.

### Edge cases discovered
- **Python proto stubs are post-processed by `ruff`/`ruff-format` pre-commit hooks, not by `buf generate` alone** (buf.gen.yaml uses unpinned remote plugins, no buf.lock). A raw `buf generate` on clean main shows ~1100 lines of phantom Python-stub format drift. Canonical path: run `make generate-protos`, then `git add -A` and commit — let the hook reformat/re-stage; verify the net staged diff is only the new `<name>.*` stubs.
- **`gh run rerun <id> --failed` can spawn a duplicate full CI run via concurrency, leaving cancelled jobs surfacing as "fail"** in `gh pr checks`. Re-run the cancelled run id directly (`gh run rerun <id>`, no `--failed`) so its contexts resolve to success — otherwise branch protection stays blocked on a phantom failure.

## Session — 2026-06-09 (issue #882)

### Effective patterns
- `conftest.py` stub with `pytest.fail(...)` inside the fixture: because tests use `pytest.mark.xfail(strict=True)`, a fixture-level fail propagates as XFAIL (not ERROR).
- `try/except ImportError` guard with `HAS_JSONSCHEMA` flag ensures graceful degradation when a package is absent, avoiding collection errors that bypass xfail.
- Pre-commit hooks (ruff) auto-fix imports; re-stage after hook failure and commit again — do not amend.

### Edge cases discovered
- Issue body specifies `SCHEMA_PATH = "spec/schemas/agent-def.json"` but actual file is `spec/schemas/agent-def.schema.json`. Use issue's path verbatim — the mismatch is intentional (one of the xfail reasons).
- Issue body specifies `len(EXPERT_YAMLS) == 9` but `automation/experts/` has 10 files. The `automation/workflows/experts/` path doesn't exist yet, yielding 0 files, so assertion fails as expected.
- `gh pr merge --squash --auto` is required (not bare `--squash`) when branch protection requires branch to be up to date — queues merge automatically once CI passes on a current SHA.

### Proposed expert prompt updates
- Rule: "When a repo has strict branch protection ('require branch to be up to date'), always use `gh pr merge --squash --auto` instead of bare `--squash`. This queues the merge to happen automatically once CI passes, avoiding rebase-wait-retry loops."
  Category: structural-workaround
- Rule: "Before writing a test file, check `spec/schemas/` for the actual filename of schemas referenced in the issue. Use the exact path from the issue body — mismatches are intentional and are part of the xfail contract."
  Category: domain

## Session — 2026-06-10 (issue #811)

### Effective patterns
- Mirror-then-diverge: cloning `e2e-happy.sh` wholesale and replacing only the engine-specific steps (`?engine=argo`, assert Argo `Workflow` CR phase) kept config/helpers/polling identical — the diff reads as "same harness, engine swapped".
- Source-anchored assertion: confirmed via `argo_engine.go` that the Argo `Workflow` resource name == runID == api-gateway run_id, so `kubectl get workflow.argoproj.io <run_id> -o jsonpath='{.status.phase}'` locates the CR with no label guessing; cited the file path in a comment.
- Verified the `?engine=` query param at its source (api-gateway `handler.go: r.URL.Query().Get("engine")`) rather than trusting issue text.
- Matched PR typing to the predecessor: #810's merged PR was `test(infra):` / label `type: test`, not the canvas's `feat:` — look up the sibling PR's actual type.

### Edge cases discovered
- shellcheck absent on host; run it via `docker run --rm -v <wt>:/mnt koalaman/shellcheck:stable <relpath>` for real lint coverage.
