# Learnings: CI / Release Engineer

> Format: each entry has `Seen in:` (issue/session) and `Date:` (YYYY-MM-DD).
> Updated by `/m6-learn` after each batch. Human-reviewed before merging.

---

## Effective patterns

- **`images/images.yaml` is the SoT for all image references — never hardcode tags in workflows.**
  The drift-check gate (`cmd/zynax-ci images check`) runs on every PR and fails if any
  workflow file references an image tag that diverges from `images.yaml`.
  Seen in: M6.Images #856–#858. Date: 2026-06-06.

- **`cache-from: type=gha` on all `docker/build-push-action` steps cuts build time by 60–80%.**
  Without the GHA cache, each CI run rebuilds all layers from scratch.
  Always set `cache-to: type=gha,mode=max` as well.
  Seen in: M5.F CI sprint #542. Date: 2026-06-06.

- **All CI jobs run in the `ci-runner` container — never install tools in `run:` steps.**
  The container has Go, buf, cosign, yq, docker, helm, and Python pre-installed.
  Installing tools in `run:` adds 2–5 min per job and diverges from the container spec.
  Seen in: #552 (switch to ci-runner container mode) PR #850. Date: 2026-06-06.

- **`provenance: true` + `sbom: true` on `docker/build-push-action` generates SLSA attestations.**
  These appear in GHCR as `unknown/unknown` manifests — this is correct and expected (ADR-025).
  Do NOT add `provenance: false` to suppress them.
  Seen in: M6.C #489 PR #833. Date: 2026-06-06.

- **buf breaking against `main` catches proto backward-incompatibility before merge.**
  Field removals and number changes fail immediately. Adding fields (new field numbers) is safe.
  Always set `against: https://github.com/zynax-io/zynax.git#branch=main`.
  Seen in: M1 proto contracts gate. Date: 2026-06-06.

---

## Edge cases discovered

- **GHCR package names with slashes need URL encoding (`%2F`) in `gh api` calls.**
  `gh api /orgs/zynax-io/packages/container/zynax%2Fapi-gateway/versions` works.
  `gh api /orgs/zynax-io/packages/container/zynax/api-gateway/versions` returns 404.
  Seen in: M6.Images verification step design. Date: 2026-06-06.

- **`buf breaking` produces exit code 1 for any breaking change, including deprecations.**
  Deprecating a field (marking it `deprecated = true`) without removing it is NOT breaking,
  but buf may still warn. Use `buf breaking --error-format=json` to distinguish errors from warnings.
  Seen in: M1 proto gate. Date: 2026-06-06.

- **`cosign sign` requires `COSIGN_EXPERIMENTAL=1` and the workflow must run with `id-token: write`.**
  Missing the permission causes a cryptic OIDC error: "failed to get OIDC token".
  The permission must be on the job-level `permissions:` block, not just the workflow-level one.
  Seen in: M6.C #489 PR #833. Date: 2026-06-06.

- **Parallel matrix jobs that all push to the same GHCR tag race and corrupt manifests.**
  When building multi-arch images, always use `docker/build-push-action` with
  `platforms: linux/amd64,linux/arm64` in a SINGLE step — not two separate matrix jobs.
  Seen in: M5.F release pipeline. Date: 2026-06-06.

---

## Failed approaches

- **Using `merge-queue` as a required CI gate.**
  Removed in #544 because it caused false positives and blocked valid PRs.
  The current gate model is: branch protection + required checks + auto-merge.
  Seen in: M5 BATCH 0. Date: 2026-06-06.

---

## Proposed expert prompt updates

*(none yet — populate after first batch of CI/release expert sessions)*

## Session — 2026-06-08 (issue #875 — expert mesh YAML configs)

### Effective patterns
- Python `yaml.safe_load` loop is the fastest local YAML validity gate before push; no yamllint config exists in repo and `make lint` doesn't cover YAML — use a quick Python snippet
- `gh api repos/zynax-io/zynax/pulls/N --method PATCH --field title='...'` works for PR title updates when `gh pr edit` returns a GraphQL deprecation error (Projects Classic association)
- `gh run rerun <run_id> --failed` triggers re-execution of only failed jobs without pushing a new commit — useful after fixing a PR title

### Edge cases discovered
- PR title subject length ≤72 chars is enforced by `amannn/action-semantic-pull-request`; em-dash counts as 1 char (safe to use); check with `echo -n '<subject>' | wc -m`
- `git checkout -b <branch>` with staged files from another branch: staged files travel with the checkout — run `git restore --staged <paths>` to unstage unrelated changes before committing
- `gh pr edit --title` can fail with GraphQL Projects Classic deprecation error; use PATCH API method instead

### Proposed expert prompt update
Add to CI/release expert guide: "After `gh pr create`, immediately check that the PR title subject is ≤72 characters: `echo -n '<subject>' | wc -m`. Fix via `gh api repos/.../pulls/<N> --method PATCH --field title='...'` (not `gh pr edit` which can fail on Projects Classic). After fixing, run `gh run rerun <run_id> --failed` to recheck without a new commit."

## Session — 2026-06-08 (issue #860)

### Avoid `make lint` before committing (Docker overwrites files)
**Seen in:** #860. **Date:** 2026-06-08

Docker-based `make lint` mounts the workspace and runs formatters (`gofmt`, `ruff-format`, etc.) which can overwrite uncommitted file changes. For pre-commit validation use targeted syntax checks (`bash -n <script>`, `python3 -c "import yaml; yaml.safe_load(...)"`, `yamllint`) instead of the full Docker lint. Run `make lint` only after the commit is in git history.

### Staging only named files prevents unintended inclusions
**Seen in:** #860. **Date:** 2026-06-08

On a shared workspace where sibling agents have dirty tracked files, `git add .` picks up unrelated changes from other agents' work. Always stage specific files by path: `git add Makefile .github/workflows/tools-image.yml scripts/bump-ci-runner.sh`.

### `gh run view --json jobs` for CI status
**Seen in:** #860. **Date:** 2026-06-08

`gh run view <run_id> --json status,conclusion,jobs --jq '.jobs[]'` gives a complete at-a-glance view of all job outcomes without repeated polling via `gh pr checks`.

---

## Session — 2026-06-08 (issue #867)

### GHCR retention via REST API pattern
**Seen in:** #867. **Date:** 2026-06-08

Use `gh api "/orgs/zynax-io/packages/container/${PKG}/versions" --paginate` to list all versions. Filter `main-sha` eligible-for-deletion versions with jq: `select((.metadata.container.tags | any(test("^main-[a-f0-9]"))) and (.metadata.container.tags | all(. != "latest" and . != "main" and (test("^v[0-9]") | not))))`. Sort by `.updated_at` descending, slice from `KEEP` index onward, delete via `-X DELETE`. Define `GHCR_KEEP_VERSIONS: "5"` at workflow `env:` level. URL-encode nested package names with `%2F` (e.g. `zynax%2Ftools`).

### Cherry-pick to rescue a commit on wrong branch
**Seen in:** #867. **Date:** 2026-06-08

Background agent branch switching causes commits to land on wrong branches. Rescue: `SHA=$(git rev-parse HEAD) && git checkout <correct-branch> && git reset --hard origin/main && git cherry-pick $SHA && git push --force-with-lease`.

## Session — 2026-06-08 (orchestrator batch #824,#816,#860)

### Claim-check: verify closed issues before any work
**Seen in:** #860. **Date:** 2026-06-08

`gh issue list --state open` can return issues already closed by a prior session in the same day
(GitHub API eventual consistency lag). Before opening any branch:

```bash
gh issue view <N> --json state --jq .state           # must be OPEN
gh pr list --state merged --search "<N>" --json number,mergedAt | jq .
```

If a merged PR references the issue: stop immediately, report the merge SHA and PR number.

---

## Session — 2026-06-08 (#838 native arm64 + #877 dev-advisory + #862 ADR-024)

### Effective patterns

- **Split-arch matrix + `merge-and-sign` is the correct native multi-arch pattern (no QEMU).**
  Use `resolve-matrix` to output a `platform_matrix` (services × platforms), then
  `build-platform` with `runs-on: ${{ matrix.platform == 'amd64' && 'ubuntu-24.04' || 'ubuntu-24.04-arm' }}`,
  then a `merge-and-sign` fan-in using `docker buildx imagetools create` to assemble the
  multi-arch index. `provenance: false` on intermediate per-platform images avoids duplicate
  attestations — apply only to the merged manifest index in `merge-and-sign`.
  Seen in: #838 PR #968–#971. Date: 2026-06-08.

- **`docker buildx imagetools inspect --format '{{json .Manifest.Digest}}'` captures the merged index digest for cosign.**
  This is the reliable way to get the merged manifest digest after `imagetools create`.
  Seen in: #838. Date: 2026-06-08.

- **Temp-file approach for complex CI multi-line output (not shell heredocs).**
  Write intermediate results to /tmp files using `printf '%s\n' "$var"` chains.
  Heredocs inside `run:` blocks break YAML parsing (YAML treats `<<` as anchor reference).
  Seen in: #877 PR #969. Date: 2026-06-08.

- **Context slice via `grep -E` on `git diff --name-only`.**
  To filter PR files by expert glob patterns (`context_slice.files`), convert `**` → `[^/]*`
  and `*` → `.*` for extended regex. Clean and fast for advisory workflows.
  Seen in: #877 PR #969. Date: 2026-06-08.

- **Job-level `outputs:` declaration is required for cross-job result passing.**
  Each expert job must declare `outputs: expert_output: ${{ steps.expert.outputs.EXPERT_OUTPUT }}`
  and write `EXPERT_OUTPUT<<EXPERT_EOF` heredoc to `$GITHUB_OUTPUT`. Without the job-level
  `outputs:` block, the collate job cannot consume the result.
  Seen in: #877 PR #969. Date: 2026-06-08.

### Edge cases discovered

- **NEVER use the `update-branch` API (`gh api -X PUT .../pulls/N/update-branch`) in multi-agent environments.**
  It creates an unsigned merge commit in GitHub's internal state that contaminates concurrent PRs
  via the shared ref namespace — causes DCO failures and cross-PR squash-merge pollution.
  Fix: `git fetch origin main && git rebase origin/main && git push --force-with-lease origin <branch>`.
  If GitHub still shows `BEHIND`: use `gh pr merge --squash --auto` and wait for self-resolution.
  Seen in: #838 (release.yml changes landed in ADR-024 PR #970). Date: 2026-06-08.

- **`gh` CLI is NOT installed in the `ci-runner` container.**
  Jobs needing `gh` (PR comments, API calls) must run on `ubuntu-24.04` (hosted runner),
  not the `ci-runner` container. Use `continue-on-error: true` for advisory-only steps.
  Seen in: #877 PR #969. Date: 2026-06-08.

- **`printf 'string with -- dashes\n'` fails in dash shell inside containers.**
  Bash interprets `--` as end-of-options and `printf` treats it as an invalid option flag.
  Fix: store the string in a variable first — `DIVIDER='---'; printf '%s\n' "$DIVIDER"`.
  Seen in: #877 PR #969. Date: 2026-06-08.

- **`git rebase` silently skips commits already present in the rebase target.**
  When two agents race and one's changes land in main before the other rebases, `git rebase`
  outputs `warning: skipped already applied commit`. Always verify `git diff origin/main -- <file>`
  after rebase to confirm the expected diff is present.
  Seen in: #838. Date: 2026-06-08.

- **`gh pr merge --squash` fails with "not mergeable: head branch not up to date"** when main
  advances while CI runs. Fix: `git pull --rebase origin main && git push --force-with-lease`,
  then retry merge (or use `--auto` flag).
  Seen in: #877 PR #969. Date: 2026-06-08.

- **`docker buildx imagetools create` does NOT copy OCI annotations from source manifests.**
  When merging per-platform digests into a multi-arch index, add explicit `--annotation` flags:
  `--annotation "index:org.opencontainers.image.description=..."`. Without this, report-image-meta
  gates that check for OCI annotations will fail even though the image was pushed.
  The `index:` prefix targets the manifest list (not individual platform manifests).
  Fix path: add four annotation flags (description, title, source, revision) to imagetools create.
  Seen in: #866 gate / #977 regression. Date: 2026-06-08.

- **Schema `additionalProperties: false` blocks forward-looking example files.**
  If a JSON schema uses `additionalProperties: false` and an example YAML uses a field not
  yet declared, tests fail on main rather than only on the PR that adds the field.
  Always audit example files when tightening schema strictness. The `output` field in
  action definitions was used in `spec/workflows/examples/code-review.yaml` but absent
  from the schema, causing three test failures on main.
  Seen in: #976 PR test-go. Date: 2026-06-08.

- **`tools-image.yml` path filter excludes `.github/workflows/` — the new workflow is inert
  until a Dockerfile or `cmd/zynax-ci/**` change triggers it.**
  PR #975 changed only `tools-image.yml` itself, which is not in the workflow's `paths:` filter.
  The native arm64 3-job workflow is on main but has never been exercised. Verify path filters
  match the actual files being changed when updating CI infrastructure.
  Seen in: #839 post-merge. Date: 2026-06-08.

- **Report-step failure ≠ image not pushed.** When a composite action used in a "Report" step
  fails, GitHub marks the job as failed but preceding build+push steps already succeeded and
  images are in GHCR. Post-merge verifier must check GHCR directly via API rather than relying
  on workflow conclusion.
  Seen in: #839 post-merge. Date: 2026-06-08.

- **Hidden image consumers not in `images/images.yaml` consumers list cause silent drift.**
  `dev-advisory.yml` had 8 occurrences of the ci-runner digest but was not listed in the
  `consumers:` array, so `make check-images` passed while the file drifted. Expand the
  consumers list whenever a new workflow is added that references a tracked image digest.
  Seen in: #839 post-merge. Date: 2026-06-08.

---

## Session — 2026-06-08 (post-merge sweep PRs #974, #976, #977)

### test-unit SKIPPED ≠ tests passed — required-check bypass

**Seen in:** PR #974. **Date:** 2026-06-08

`test-unit` has `if: always() && !contains(needs.*.result, 'failure') && !contains(needs.*.result, 'cancelled')`.
When `test-go` fails, this condition evaluates to `false` — so `test-unit` is **SKIPPED**, not run.
GitHub branch protection treats a SKIPPED required check as neutral (not blocking), allowing the merge.
PR #974 merged despite `test-go: FAILURE` (broken test + `cmd/zynax-ci` coverage 74.6% < 80% gate) because
`test-unit` showed SKIPPED, not FAILED, in the status checks.

**Fix:** Change `test-unit` to `if: always()` and fail explicitly inside the step when upstream tests failed:
```yaml
test-unit:
  if: always()
  steps:
    - name: All tests passed
      run: |
        if [[ "${{ needs.test-go.result }}" == "failure" || "${{ needs.test-python.result }}" == "failure" ]]; then
          echo "::error::test-go=${{ needs.test-go.result }}  test-python=${{ needs.test-python.result }}"
          exit 1
        fi
        echo "✅ test-go=${{ needs.test-go.result }}  test-python=${{ needs.test-python.result }}"
```

### Dockerfile.service missing libs/ copy breaks services with replace directives

**Seen in:** PR #976 (triggered task-broker rebuild). **Date:** 2026-06-08

`Dockerfile.service` copies only `protos/generated/go/` and `services/${SVC}/` into the build context.
`task-broker/go.mod` has `replace github.com/zynax-io/zynax/libs/zynaxconfig => ../../libs/zynaxconfig`.
`go mod download` inside Docker resolves this to `/workspace/libs/zynaxconfig` — which was never copied.
The failure was latent since PR #907/909 (Jun 6) but masked because no subsequent PR triggered a
task-broker rebuild until PR #976 changed `protos/generated/go/` (which is in the `task_broker=true`
path-filter in `release.yml`).

**Fix:** Add `COPY libs/ ./libs/` (and the corresponding `go.mod`/`go.sum` prefetch) in `Dockerfile.service`
before the `go mod download` step. If `libs/` grows, scope it to `COPY libs/zynaxconfig/ ./libs/zynaxconfig/`.

### Coverage gate is a no-op when coverage.out is absent

**Seen in:** PR #974 analysis. **Date:** 2026-06-08

The `cmd/zynax coverage gate` step in `_test-go.yml` opens with `if [ -f "cmd/zynax/coverage.out" ]; then … fi`
and has **no `else` clause**. If the prior test step fails before writing `coverage.out`, the gate step
exits 0 silently — the gate never fires. Same pattern on `cmd/zynax-ci`.

**Fix:** Add `else echo "::error::coverage.out not found for cmd/zynax — aborting"; exit 1` to each gate.
Also change the empty-total line from `exit 0` to `exit 1`.

### Post-merge: check GHCR directly — Release failure ≠ image not pushed

**Seen in:** PR #977 Release failure. **Date:** 2026-06-08

In `merge-and-sign`, step order is: (1) `docker buildx imagetools create` → manifest pushed to GHCR,
(2) annotation check → failed, (3) cosign sign → skipped.
PR #977 left `workflow-compiler` and `api-gateway` in GHCR tagged `main` and `main-f11dfe34` but
**without cosign signatures**. The Release workflow showed red but the images were already public.
Post-merge verification must query GHCR directly (`gh api /orgs/…/packages/…/versions`) rather than
inferring image state from the workflow conclusion.

**Reminder:** The annotation-check failure in PR #977 was fixed by PR #979 (added `--annotation` flags
to `imagetools create`). A re-sign pass was not performed; the Jun-8 `main` images remain unsigned.

## Session — 2026-06-09 (issues #840, #878, post-mrg #796)

### #840 — Python adapters in multi-arch release pipeline
- **`svc_dockerfile` associative array is the extension point**: When adding adapters to `release.yml`, check the `svc_dockerfile` pattern (not just matrix entries). The `http-adapter` entry was the correct reference. Seen in: #840.
- **ci-adapter and git-adapter are Go-based, not Python**: Despite the "Python adapter images" issue title, ci-adapter and git-adapter use Go/distroless base images. Python adapters watch `protos/generated/python/` for change detection; Go adapters watch `protos/generated/go/`. Seen in: #840.
- **All 4 adapter Dockerfiles already had OCI LABEL annotations**: No Dockerfile changes were needed — annotations had been added in an earlier session. Seen in: #840.

### #878 — Wave 1 orchestrator aggregation (dual-runtime design)
- **`aggregation-protocol.md` as dual-runtime document**: Rewriting this file as a `claude -p` system prompt (not just documentation) enables the same protocol to be used in both the GHA CI orchestrator job and the CLI `/m6-orchestrate` command. One config, two runtimes. Seen in: #878.
- **Orchestrator job must use `ubuntu-24.04`, not `ci-runner`**: The `gh` CLI is not installed in the ci-runner container. Any job that posts PR comments must run on the hosted runner. Seen in: #878 (same pattern as #877).

### post-mrg #796 — engine-adapter
- **engine-adapter is NOT pinned in docker-compose.services.yml**: Only `http-adapter` has a digest pin there. engine-adapter uses a mutable `:main` tag in the dev docker-compose. No digest update needed post-merge. Seen in: post-mrg #796.
- **Release workflow annotation check failure ≠ image not published**: The `report-image-meta` action checks annotations after `imagetools create` pushes the image. If the annotation check fails, the image is already in GHCR — verify separately with `gh api /orgs/zynax-io/packages/container/zynax%2F<svc>/versions`. This is a recurring pattern. Seen in: post-mrg #796, #839, #977.

## Session — 2026-06-09 (issues #806, #879)

### #806 — SDK PyPI publish workflow
- **Trusted Publisher environment name**: The `environment: pypi` in sdk-publish.yml must exactly match the environment configured on pypi.org in step O1 (#805). Verify by reading the canvas O1 description or the prior PR before writing the workflow.
- **Tag-scoped workflows don't need branch filters**: `on: push: tags: ["v*.*.*"]` is sufficient — no `branches:` filter needed. Adding an accidental `branches:` filter would silently prevent tag-triggered publishing.

### #879 — Wave 2 gated actions
- **Background agent cleanup causes shared-worktree branch switch**: When a domain agent completes (DONE phase with `git checkout main`), it switches the shared working tree to main. Any uncommitted changes on the feature branch are wiped. Pattern: commit immediately after editing, never rely on uncommitted state surviving across background agent completions.
- **`make lint` runs in Docker and is non-destructive to files**: The lint container reads files but doesn't write them. File reversions seen mid-session were caused by shared-worktree branch switches (another agent's cleanup), not by the linter itself.
- **Context-free agent spawning causes duplicate PRs**: Sending a context-free "proceed" message to a new agent after the original completed caused it to create a duplicate PR (#996) against already-merged work (#995). Fix: never spawn a new agent to handle a completion notification — read the current branch state directly and perform remaining steps (PR create, merge) from the orchestrator.
- **`gh pr close <N>` is idempotent**: If a duplicate PR is already closed, `gh pr close` returns `already closed` but exits 0. Safe to call without checking.
- **Wave 2 action-execution step placement**: The Wave 2 step belongs in the `orchestrate-and-comment` job (same job as orchestrator), not a separate job. This avoids re-running the full expert fan-out for action decisions and has access to `steps.decision-log.outputs.*`.

## Session — 2026-06-09 (issue #1003)

### Rebase interference during conflicted rebase (git-ops)
- **Editor/linter touching conflicted files mid-rebase corrupts the rebase**: During a conflicted rebase, an automation that re-read and rewrote the conflicted files (clearing the `<<<<<<<`/`=======`/`>>>>>>>` markers) left the rebase in a broken half-applied state — `git rebase --continue` then reported "no rebase in progress" while HEAD sat detached on `origin/main` with the replayed commit's changes scattered as *uncommitted* edits and one file silently dropped. Seen in: #1003.
- **The pushed pre-rebase commit is the safe restore point**: Recover with `git reset --hard origin/<branch>` (the empty/feature branch was pushed before the rebase), then redo the rebase with `git -c core.editor=true rebase origin/main` so no editor opens, and resolve conflicts *programmatically* (python/sed rewriting the conflict block) instead of with tools that re-read and re-touch files. Seen in: #1003.
- **Verify the PR's real head branch after a tangled rebase**: The corruption spawned a stray slugged claim branch (`<type>/<N>-<slug>`) with its own remote tracking ref, diverging from the PR's actual head (`<type>/<N>`, still at the pre-rebase SHA). Always `gh pr view <N> --json headRefName,headRefOid`, force-push the rebased commit to the PR's *real* head branch, and delete the stray. Seen in: #1003.
- **Sibling EPIC steps merging mid-flight conflict on shared state docs**: While delivering one O-step, sibling steps (#1004 PR #1008, #1005 PR #1009) merged and edited the same `M6-planning.md` table and `canvas.md` O-step list. Expect a mid-flight rebase; resolve by taking main's rows wholesale and re-applying only *your* row — the row-level conflict is mechanical, not semantic. Seen in: #1003.

## Session — 2026-06-09 (issue #841)

### Effective patterns
- **Audit-and-document is a valid "minimize" deliverable when images are already near-optimal** (distroless Go ~8 MB, slim+uv Python ~75 MB): verification + documented justification in Dockerfile headers + a SECURITY.md size table satisfies the acceptance criteria with zero build-breakage risk.
- **`.dockerignore` test-file exclusions must come AFTER module-allowlist negations**: the repo re-includes whole module trees (`!agents/adapters/llm/`), so `*_test.go`/`tests/`/`*.feature` patterns placed last actually strip test files from the context.
- Header-comment-only Dockerfile edits never touch image-ref banner regions, so `make sync-images` is unnecessary and `make check-images` stays green.

### Edge cases discovered
- **Do NOT swap a working `python:3.12-slim` + uv build to alpine**: musl forces fragile source compilation of manylinux ML wheels (openai/boto3/grpcio/pydantic-core). Document the rejection inline; keep slim.
- Canvas path `docs/spdd/837-native-multiarch/canvas.md` named in #841 does not exist (837 is a `ci:` epic, SPDD-exempt); proceed from the issue body scope table.

### Post-merge observation (orchestrator session)
- **Release workflow failing on main independent of this batch**: Trivy container scan exits 1 on `langgraph-adapter`/`llm-adapter` amd64 — observed even on the protos-only merge `a0d8d5b` (#1019), so it is a CVE-DB/scan-config issue recurring on every main merge, not a Dockerfile regression. Separately, "Merge & sign — engine-adapter" failed on `56a8ac2` (#1018) at the manifest-merge/cosign step (both per-arch builds succeeded) — infra, not code. No new clean image digests published while Release is red → no digest pins to update. Needs separate investigation (Trivy allowlist + cosign/OIDC sign step).

## Session — 2026-06-09 (issues #867, #883)

### Issue already merged but still OPEN (#867)

- **Check git log before implementing to detect pre-existing work**: Running `git log --oneline --follow -- .github/workflows/<file>.yml | head -5` immediately revealed the GHCR retention cap was already merged (commit ba13142 in PR #960), avoiding duplicate effort. Always confirm both the GitHub issue state AND the git history before writing any code. Seen in: #867.
- **Squash-merge can leave issues open when PR body lacks `Closes #N`**: PR #960 shipped the GHCR retention cap but left issue #867 open because the PR body didn't include an explicit `Closes #867`. Resolution: close the issue manually with a reference to the merge commit after confirming implementation is in main. Seen in: #867.
- **Atomic claim branch must be cleaned up even when no work is done**: The deterministic claim push creates a remote branch `<type>/<N>` even when the issue is found to be pre-delivered. Always delete the claim branch as part of the "already done" exit path. Seen in: #867.

### BEHIND mergeStateStatus blocks squash-merge even after CI passes (#883)

- **`gh pr merge --squash` fails immediately when `mergeStateStatus` is `BEHIND`**: GitHub re-requires all checks to pass on the *rebased* HEAD — a green CI run on the pre-rebase HEAD is not accepted. The correct pattern is: rebase → push → `gh pr merge --squash --auto` → wait for GitHub to fire the merge once checks pass. Never retry `gh pr merge` directly in a loop — set `--auto` and let GitHub trigger the merge. Seen in: #883, #808.
- **Parallel M6 activity continuously pushes to main**: On an active milestone, main accumulates commits from concurrent agents every few minutes. Any PR that takes >3 min in CI will be BEHIND by the time checks complete. Treat `--auto` as the default pattern for all PRs, not an exception. Seen in: #883.

### Proposed expert prompt update

- Rule: Before implementing any story, run `git log --oneline --follow -- .github/workflows/<file>.yml | head -5` (or equivalent for the relevant file) to detect if the issue was already shipped in a prior commit. If the commit message references the issue number, check `gh issue view <N>` and close it with a reference if still OPEN.
  Category: domain
  Reason: Squash-merge workflows can leave issues open even after the implementing PR is merged — recurring pattern in active milestones.

- Rule: After `gh pr merge --squash` fails with "required status checks are expected", always check `gh pr view <N> --json mergeStateStatus` — if `BEHIND`, rebase + push + set `--auto` rather than retrying the direct merge. The `--auto` flag fires the merge once checks pass on the rebased HEAD without requiring another manual attempt.
  Category: structural-workaround
  Reason: Active main branches (parallel M6 delivery) cause BEHIND status on nearly every PR where CI takes >3 minutes — affects all story types.

## Session — 2026-06-09 (issues #807, #880)

### Effective patterns
- **UNSTABLE mergeStateStatus with zero required checks = mergeable directly**: When `gh pr view --json mergeStateStatus` returns `UNSTABLE` and `[.statusCheckRollup[] | select(.isRequired==true)] | length` is 0, `gh pr merge --squash` succeeds without waiting. Advisory-only checks that are pending/failed do not block merge.
- **Rebase before merge when BEHIND**: After concurrent merges to main, a PR's branch falls BEHIND. Run `git -C <coord-wt> fetch origin && git -C <coord-wt> rebase --signoff origin/main && git push --force-with-lease` then use `--auto` to arm GitHub's auto-merge.
- **Claim commit must include `-s`**: `git commit --allow-empty -s -m "...[claim]"` — DCO is enforced on all commits including empty claim commits. Missing `-s` causes an immediate DCO failure that blocks all CI checks.

### Edge cases discovered
- **Full CI run after rebase --signoff = ~30 checks, 15–20 minutes**: Includes CodeQL, Expert LLM reviews, orchestrator aggregation, test-go, lint-go, security. LLM Expert checks appear mid-run and extend the total time window.
- **Background sleep loops are killed after one iteration on this platform**: Polling loops that rely on `sleep` in a background subagent context exit prematurely. Restart the poll cycle on each notification rather than expecting a persistent background loop.
- **`docker buildx imagetools create` does not propagate OCI annotations**: When assembling multi-arch index, add explicit `--annotation "index:..."` flags; without them the OCI annotation gate fails even though the image is pushed. (Confirmed in #807 ci-runner context.)

### Proposed expert prompt update
- Rule: Always use `git commit --allow-empty -s -m "...[claim]"` for the atomic branch claim commit — `-s` is required for DCO on every commit.
  Category: domain
  Reason: DCO check fails immediately on any commit missing Signed-off-by, including empty claim commits.
- Rule: After a force-push rebase, wait for CLEAN (not just absence of FAILURE) before merging. Use `--auto` after the rebase so GitHub handles the final merge gate.
  Category: domain
  Reason: BEHIND/BLOCKED persist briefly after rebase while GitHub re-evaluates head; only CLEAN is the safe merge signal.

## Session — 2026-06-09 (issue #880 addendum)

### Effective patterns
- **Read Wave N-1 workflow before writing Wave N**: `dev-advisory.yml` (Wave 2) established naming conventions (`# Plane: near-term`, `continue-on-error: true`, `ubuntu-24.04` for `gh` CLI, `printf`-based body construction) that all Wave 3 jobs followed consistently.
- **Python3 `yaml.safe_load` is a reliable YAML structural validator**: fast, available on host without Docker, catches missing `jobs:` key and position errors.
- **Programmatic invariant verification before commit**: check job keys, runners, `continue-on-error`, `if: always()` presence before staging for a strong pre-commit confidence check.

### Edge cases discovered
- **Omitting the top-level `jobs:` key**: jobs placed at workflow root level — YAML parses without structural error but GitHub Actions rejects it. Always verify `jobs:` key is present.
- **`workflow_run` + `push` + `schedule` triggers require per-job `if:` guards**: all three triggers fire all four jobs unless each job guards its own trigger with `if: github.event_name == 'xxx'`.
- **`contains(toJSON(github.event.head_commit.modified), 'services/')` for path-filtering in job `if:` conditions**: `paths:` filters work at workflow level only; use this pattern for job-level path filtering on push events.
- **`mergeState: UNKNOWN` during CI on advisory workflows**: Wave 0/1 advisory jobs running as `continue-on-error` show as pending, keeping `mergeStateStatus` UNKNOWN. Check the `state` field directly rather than relying on `mergeStateStatus` alone.

### Proposed expert prompt update
- Rule: After writing any GitHub Actions YAML, verify (1) `jobs:` top-level key exists, (2) per-job `if:` guards match the workflow `on:` triggers, and (3) `workflow_run`-triggered jobs include `if: github.event.workflow_run.conclusion == 'success'` unless intentionally running on failure. Validate with `python3 -c 'import yaml; yaml.safe_load(open("file.yml"))'`.
  Category: domain
  Reason: Missing `jobs:` key and wrong trigger guards are silent errors at the YAML level but fail GitHub Actions schema validation immediately.

## Session — 2026-06-09 (post-merge PR #1032, issue #810)

### Effective patterns
- Running `make check-images` before attempting a digest-bump PR correctly reveals false positives from the Wave 3 post-merge completeness mesh auto-issue generator.
- Checking the Release workflow's job-level conclusions (skipped vs success) is the definitive signal for whether images were published — faster than querying GHCR directly.

### Edge cases discovered
- The Wave 3 completeness mesh auto-created two duplicate digest-drift issues (#1035, #1038) even though no actual drift existed — false positives because the mesh fires on each push to main and this PR triggered two runs in rapid succession.
- `make check-images` returned clean despite two open drift issues — always run `check-images` locally as the authoritative signal before any digest-bump work.

### Proposed expert prompt update
- Rule: "Before running `make sync-images`, always run `make check-images` first. If it returns clean, any open digest-drift auto-issues are false positives — close them with explanation and skip the digest-bump PR. Do not open a digest-bump PR when the working tree is clean."
  Category: domain

## Session — 2026-06-09 (post-merge PR #1030, issue #882)

### Effective patterns
- Test/automation-only PRs (no Go or Dockerfile changes) reliably produce zero image artifacts; Phase 3-5 can be confidently skipped after confirming changed files are all under `automation/tests/`.
- Querying `gh api repos/.../actions/runs?branch=main` filtered by `head_sha` gives a fast, reliable signal for all CI conclusions.

### Edge cases discovered
- PR #1030 touched `automation/tests/__init__.py` in addition to the 2 files in the task description. Always use `gh pr view --json files` as authoritative source, not the task description.

### Proposed expert prompt update
- Rule: "Fetch actual changed files from `gh pr view --json files` before Phase 1 assessment; do not rely on the task description's file list, which may be incomplete."
  Category: domain

## Session — 2026-06-09 (post-merge PR #1034, issue #803)

### Effective patterns
- `docker-compose.yml` uses floating `:main` tags for internal services; `docker-compose.services.yml` pins adapter images by digest. Check both separately.
- `images/images.yaml` tracks only base/toolchain images — service images (workflow-compiler, api-gateway, etc.) are never added there.

### Edge cases discovered
- `docker-compose.services.yml` only contains backing stores (NATS, Redis) and `http-adapter`. Core platform services (workflow-compiler, api-gateway, etc.) live in `docker-compose.yml` with floating `:main` tags — no digest pin update needed for these.

### Proposed expert prompt update
- Rule: "workflow-compiler and other core platform services in `docker-compose.yml` use floating `:main` tags by design. Only adapter services (e.g., `http-adapter`) in `docker-compose.services.yml` use digest pins. Skip Phase 4a digest update when the service is not referenced with a digest pin in any compose file."
  Category: domain

## Session — 2026-06-10 (post-merge PR #1062 / issue #804)

Post-merge verification of an engine-adapter change. Outcome: SKIP — image published, CI green, no digest pin to update.

### Effective patterns
- `grep -rn "<svc>@sha256"` across the whole repo BEFORE assuming a matrix service has a digest pin. engine-adapter is in the release matrix and publishes an image, but is referenced only by mutable tag (`ghcr.io/zynax-io/zynax/engine-adapter:main`) — it has no `@sha256:` pin anywhere, so the correct terminal state is SKIP (no branch, no PR).
- The authoritative "image published for this exact commit" signal is a `success` Release run PLUS the GHCR tag `main-<short-sha>` on the package's newest version — not the run conclusion alone.

### Edge cases discovered
- A single merge SHA can show multiple "Post-Merge Completeness (Wave 3)" runs (re-runs). Treat any one `success` as sufficient; don't block waiting for a specific run instance. (Workflow demoted to schedule-only "Weekly Audit" / `weekly-audit.yml` in #1113 — per-merge runs no longer exist.)

## Session — 2026-06-10 (post-merge: PRs #1062, #1065; issues #804, #491)

### Effective patterns
- Filtering `actions/runs?branch=main` by `head_sha` returns all runs for a merge immediately; check run state BEFORE entering a wait loop to avoid burning the 20-min budget on already-complete runs (#1062).
- Verify the GHCR image tag against the short merge SHA (`main-<sha7>`) as positive proof release.yml rebuilt the right commit, not a stale cache (#1062).

### Edge cases discovered
- Most services are consumed via the floating `main` tag and are NOT digest-pinned (only http-adapter + base nats/redis are). Before flagging a compose pin stale, `grep -rn '<svc>@sha256' <wt> --include=*.yml --include=*.yaml`; a matrix service that was built but has zero digest references is a valid DONE no-op, not SKIP and not an empty PR (#1062).
- release.yml change-detection + build matrix cover only 5 Go services (api-gateway, engine-adapter, workflow-compiler, task-broker, agent-registry). `event-bus` and `memory-service` have NO matrix entry — touching them builds no image. Do not assume "touched 7 services" == "7 images built" (#1065).
- **A shared `libs/*` module added with `replace =>` in service go.mods breaks release.yml image builds** unless `infra/docker/Dockerfile.service` COPYs the new lib into the build context. This passes all PR checks (release builds run only post-merge) and lands red on main. Fix: add the COPY line in the same feature PR; if missed, ship a fast `fix(ci): copy libs/<name> into service build context` PR (#1065 → #1067).

### Failed approaches
- Passing a nonexistent path (`deploy/`) to `grep -rn` aborts the whole call (exit 2). Use `grep -r --include=*.yaml <worktree-root>` instead of guessing subdirs (#1062).

### Orchestrator process note
- The pre-spawn reconcile (layer 2) `gh pr list --state merged --search "<N> in:body"` FALSE-POSITIVES on canvas/review PRs that merely reference an issue number in a list (#804→#820 canvas PR, #491→#631 review doc). The issue being still OPEN is the reliable signal; treat `in:body` matches as advisory only and confirm against issue state before dropping a claim.

### gh / DCO gotchas (both post-merge + domain agents hit these)
- `gh pr create --milestone "M6"` fails — pass either the full title `K8s Production-Ready (M6)` OR (simpler, repo-wide) the LABEL `--label "milestone: M6"`. Resolve titles with `gh api repos/zynax-io/zynax/milestones --jq '.[].title'`.
- `gh pr checks --json` is unsupported; use `gh pr view <n> --json statusCheckRollup`.
- DCO: do NOT pass `-s` to `git commit` (nor `--signoff` to `git rebase`) when the message file already has a `Signed-off-by` line — it duplicates the trailer. Put one signoff in the file + plain `git commit -F`, or use `-s` with no trailer in the file.

## Session — 2026-06-10 (issues #1070, #656 post-merge)

### Effective patterns
- Wiring new live-cluster assertion steps into the e2e-smoke gate: insert between the existing `cluster-up` and `helm-upgrade` steps and keep teardown in `if: always()` — mirrors the established structure so runner behaviour is unchanged (#1070).
- Verify workflow wiring via the per-step `conclusion` array (`gh api .../jobs/<id> --jq '.steps[]'`) rather than the aggregate job result — proves new steps ran in the right order and that failure propagated, even when the overall gate is "failed" (#1070).
- Post-merge: `grep -rn '<svc>@sha256' --include=*.yml --include=*.yaml` before flagging pins stale. A PR touching matrix services where NONE are digest-pinned (floating `main` tag) is a DONE no-op, not SKIP and not an empty PR. SKIP is reserved for "no matrix services affected at all" (#1083).
- Cross-check GHCR image `created_at` against the merge time to confirm images were freshly rebuilt, not stale carry-overs (#1083).

### Edge cases discovered
- The `e2e smoke` gate's happy-path assertion fails on `ubuntu-latest`: api-gateway on the kind host port returns `curl (56) connection reset` (placeholder images, no live engine). This is a runtime/environment gap of the gate itself — non-required, recurs on every services/helm PR, and does NOT indicate a defect in the PR under test. "Bring up cluster + deploy stack" succeeding while only the assertion step fails confirms the chart/probe changes are healthy (#1070, #656).

### Failed approaches
- `gh pr merge --squash --delete-branch` from a linked worktree exits non-zero ("main already checked out") but the server-side squash-merge still succeeds. Confirm via `gh pr view --json state,mergeCommit`; delete the branch with `gh api -X DELETE repos/<org>/<repo>/git/refs/heads/<branch>` (#1070).

### Proposed expert prompt update
- Rule: To merge a PR whose ONLY red check is a non-required gate (e.g. `e2e smoke`) while all required checks pass, use `gh pr merge --squash --admin` (—`--auto` waits forever on the failed non-required check). Reserve this for explicitly non-required gates with a known environment cause.
  Category: structural-workaround
  Reason: The non-required e2e gate fails on the hosted runner env gap on every services/helm PR; without --admin those PRs cannot land despite green required checks.

## Session — 2026-06-15 (M7 Wave 0 — Q EPIC #1172: issues #1212 #1213 #1214 + post-merge #1224)

### Effective patterns
- **Zero-statement coverage detection (honest gate):** the canonical test for a package with no coverable statements is `go tool cover -func=<profile> | grep -vc '^total:'` == 0 (no function rows). Prefer this over inspecting coverprofile body length or the `[no statements]` string. The gate logic is duplicated across the Makefile `test-coverage` target AND `.github/workflows/_test-go.yml` (both gate and measurement steps) — fix ALL sites; `SERVICE_LIST` in the workflow currently omits event-bus, so the Makefile path was the live failure (#1213).
- **Trusted Publisher provenance is sourced from the workflow, not the doc:** extract the GitHub Environment from the publish workflow's job-level `environment:` key (here `sdk-publish.yml` → `environment: pypi`), not from any pre-existing doc table — tables drift; PyPI Trusted Publisher registration must match the workflow value exactly or the OIDC exchange fails. The §14 table had drifted to "release" (#1214).
- **`make sync-images`/`check-images` ran cleanly in-sandbox** and gave fast local confidence before pushing a Dockerfile/image change (#1212).

### Edge cases discovered
- **The `tools` image is NOT digest-tracked in `images/images.yaml`** (no consumer pins a tools digest) — `make sync-images` is a no-op for a tools-image change; its published digest updates post-merge via `tools-image.yml`. Do not fabricate a digest (#1212).
- **pip in the tools image is unpinned, bundled with `python:3.14-alpine`.** PYSEC-2026-196 is closed by an explicit `RUN python -m pip install --no-cache-dir --upgrade "pip==26.1.2"` in the final stage of `infra/docker/Dockerfile.tools` (#1212).
- **`make security-agents` cannot be verified pre-merge for a tools-image bump** — it runs pip-audit inside the tools image, which still carries the OLD pip until the post-merge `tools-image.yml` rebuild. Verification belongs to the post-merge gate (#1212).
- **For `[Unreleased]`-target provenance:** CHANGELOG `[Unreleased]` is the correct home for a forward release-notes pointer when the target version (v0.6.0) is unreleased; record "_pending_" placeholder rows for first-publish fields rather than inventing a version/URL (#1214).
- **Multi-arch tools-image build leaves a stranded partial tag on a single-arch leg failure:** the build splits into per-arch matrix jobs; when the arm64 leg failed, GHCR was left with a stranded `amd64-<sha>` SHA tag and an un-updated `latest`. Always confirm the `latest` tag specifically — a fresh version row alone is insufficient (it may be a partial single-arch push). `apk add` exit 255 under arm64 QEMU emulation is the common transient mode here; amd64 succeeding on the identical Dockerfile is strong signal it is environmental — a `gh run rerun <id> --failed` is the right remediation, not a code change (post-merge #1224).

### Failed approaches
- **`gh api .../pulls/<N>/update-branch` to satisfy strict up-to-date branch protection breaks DCO:** its server-side merge commit has no `Signed-off-by` and fails the `dco` check. With required_signatures + DCO + strict-up-to-date all enabled, the only clean path is `git rebase origin/main` + `git push --force-with-lease` (re-signs, stays signed-off), then `gh pr merge --squash --auto` to win concurrent-merge races on busy main (#1212, #1213).

### Proposed expert prompt update
- Rule: To satisfy a "branch is behind base" / strict-up-to-date merge requirement on this repo, do NOT use `gh api .../pulls/<N>/update-branch` — its merge commit lacks `Signed-off-by` and fails `dco`. Instead `git rebase origin/main` + `git push --force-with-lease`, then `gh pr merge --squash --auto`.
  Category: structural-workaround
  Reason: This repo enforces DCO + required_signatures + strict-up-to-date simultaneously; the API update-branch path silently violates DCO and manual squash-merge loses races on high-traffic main.
- Rule: After a post-merge multi-arch tools/ci-runner image build, verify the `latest` tag moved (not just that a new version row exists) — a single-arch matrix leg failure strands an `<arch>-<sha>` tag while `latest` stays on the old image. For an `apk add` exit-255 arm64-QEMU failure, remediate with `gh run rerun <id> --failed` (transient), not a Dockerfile change.
  Category: domain
  Reason: Prevents declaring a CVE-bump "live" when the consumed `:latest` image still carries the vulnerable toolchain.

## Session — 2026-06-16 (post-merge PR #1237, issue #1180)
domain: ci-release · post-merge verification of refactor(engine-adapter)

### Effective patterns
- Reading release.yml header comments first revealed the ADR-027 retag-only model: images build once pre-merge in ci.yml `build-images` staging lane, release.yml promotes via `workflow_run` after CI completes. Prevents a false ERROR when no "Release" run is tied directly to the merge-SHA push.
- Query GHCR versions filtered by `tags | test("^main")` to skip the `.sig` cosign artifact that occupies `.[0]` and would be mistaken for the image digest.

### Edge cases discovered
- The release workflow's github-actions bot commits the digest sync to `images/images.yaml` (`chore(images): sync digests … [skip ci]`, DCO-signed) as a recorded ADR-027 exception — by the time the verifier inspects main, pins are already current; no manual digest PR is warranted. Contrary to the generic guide note, service-image digests (engine-adapter etc.) DO live in images.yaml in this repo via that bot.
- engine-adapter is not digest-pinned in docker-compose.services.yml (only http-adapter is); the primary docker-compose.yml uses a floating `:main` tag. The Phase-4a sed loop is a no-op for it.

### Failed approaches
- Looked for a workflow literally named "Release" tied to the merge-SHA push; it only materializes after CI completes via `workflow_run`. Wait for CI green first, then re-query `event=workflow_run`.

### Proposed expert prompt update
- Rule: This repo follows ADR-027 retag-only: release.yml fires via `workflow_run` AFTER the CI run on main completes (not on the push). Wait for CI green, then query `actions/runs?event=workflow_run` for the Release run. Its github-actions bot auto-commits digest sync to images/images.yaml as `chore(images): sync digests … [skip ci]` — verify pins are current there before assuming a manual digest PR is needed; service-image digests legitimately live in images.yaml via that bot.
  Category: domain

## Session — 2026-06-16 (post-merge PRs #1248 #1247 #1249, issues #1185 #1181 #1176)
domain: ci-release / post-mrg · M7 batch post-merge verification (3 verifiers; digest PR #1251)

### Effective patterns
- `gh run view <release_run_id> --json jobs` is the ONLY reliable promotion proof: a "Release" run existing — even with conclusion `success` — does NOT mean images were rebuilt. #1249's Release run was success but its retag promoted 0 images (proto-only); #1250's run is what actually re-promoted event-bus.
- Cross-reference each PR's changed files against the retag-on-merge model (promotes only `staging/pr-<that-PR-head>`) to reason about which services were truly rebuilt — this explained how event-bus recovered: #1250 touched `event-bus/go.sum` → CI rebuilt its staging image → #1250's retag re-promoted it, so the skipped #1247 retag left no permanent gap.
- `gh pr view --json files` first immediately classifies a libs-only / proto-only change and short-circuits matrix mapping.

### Edge cases discovered
- The release pipeline auto-syncs `images/images.yaml` ONLY, never `infra/docker-compose/docker-compose.services.yml` — compose pins drift silently and reconciling them is the verifier's real job when images.yaml is already current (here http-adapter `b12750bf`→`d2d6e87a`).
- A "Release" run triggers even on a libs-only / non-service merge but runs only the retag/promote job with all service-build matrix jobs `skipped`. A FAILED CI on main skips the ENTIRE Release run — that is why #1247's event-bus retag was skipped while main was transiently red from the #1248 go.sum drift.
- `ci.yml build-images` matrix runs only on `pull_request`/`workflow_dispatch`, never on push — the on-merge path is retag-only, never a fresh build.
- A required check that is path-gated to `skipped` is treated as satisfied by the ruleset; a required check that simply never reports keeps the PR BLOCKED forever — distinguish the two before assuming a stuck PR.

### Failed approaches
- Literal `[skip ci]` in a digest PR's commit message AND body silently suppressed the PR's CI (only CodeQL ran), leaving auto-merge BLOCKED with "no required checks reported". Fix: amend the commit + edit the body to use "skip-ci marker" phrasing, force-push to re-trigger CI.
- `gh api -f body=@file` does NOT read the file (`@` only works with `-F`) — it set the body literally to `@/path`; use `gh api -F body=@file`.
- `gh run view <jobID> --log-failed` returns HTTP 404 — pass the RUN id, not a job id, and let `--log-failed` find the failing job.

### Proposed expert prompt update
- Rule: Verify post-merge promotion via `gh run view <release_run_id> --json jobs` and confirm the "Retag staging → main" job conclusion is `success` (not `skipped`); a Release run merely existing/succeeding is not proof. A failed CI on main skips the entire Release. The guide's matrix-service list is stale (event-bus IS in the matrix) — use the per-PR changed-file → staging-image mapping instead.
  Category: domain
- Rule: NEVER put the literal `[skip ci]`/`[ci skip]` token in a digest PR's commit message OR body, even when quoting the bot's skip-ci digest-sync commit — write "skip-ci marker". The token in a PR body silently skips the PR's CI and leaves auto-merge BLOCKED with "no required checks reported".
  Category: structural-workaround
  Reason: Recurring, hard-to-diagnose, and unique to this verifier role which routinely references the bot's skip-ci digest-sync commits.

## Session — 2026-06-16 (orchestrator finalization — issues #1177 #1186 #1198)

### Effective patterns
- Orchestrator finalizes PRs from the coordinator worktree: when domain subagents
  produce the commit + branch + PR but then stall, the coordinator drives push →
  `gh pr checks --watch` → squash-merge directly. Foreground coordinator work is not
  subject to the background-agent watchdog, so the long CI wait completes reliably.
- Transient SBOM flake recovery: the pre-merge build-images "Generate SBOM
  (CycloneDX JSON)" step intermittently fails with `No files were found ... sbom-<svc>.json`
  while Build/Trivy pass. It is infra-transient and unrelated to the PR's code —
  `gh run rerun <run-id> --failed` clears it. Confirmed: it did not recur on rerun.

### Edge cases discovered
- Post-merge digest maintenance is fully automated here (ADR-027): release.yml commits
  `chore(images): sync digests after main-<sha>` with a skip-ci marker after each merge.
  A post-merge verifier is a SKIP when no `bump digest` issues are open — the digest pins
  are kept current by the pipeline, not by hand.

### Failed approaches
- Relying on background subagents to complete the merge: all three stalled at ~600s
  ("stream watchdog") during the terminal push/CI-wait phase. They are fine for
  implement+commit+push+open-PR, but the orchestrator must own the CI-wait/merge tail.

### Proposed expert prompt update
- Rule: When a dispatched expert subagent reports a stall/timeout but its `## Result`
  shows a pushed branch and/or open PR, do NOT discard the work — reconcile live state
  (`git ls-remote` + `gh pr list --head <branch>`) and finalize the PR from the
  coordinator worktree (push if unpushed, `gh pr checks --watch`, squash-merge).
  Category: structural-workaround
  Reason: Background subagents reliably trip the ~600s stream watchdog during the long
  CI-wait phase; the implementation work is already done and must not be thrown away.

## Session — 2026-06-16 (issue #1215)
Story: Q.4 — verify + document Go module consumption path (pkg.go.dev). PR #1258, `docs/contributing/go-module-consumption.md`.

### Effective patterns
- proxy.golang.org `@latest` as import-availability proof: `curl -s https://proxy.golang.org/<module>/@latest` returns JSON with the correct `Subdir` + pseudo-version, definitively proving a monorepo submodule is `go get`-able without a tagged release. Empty `@v/list` (200, no body) distinguishes "no semver tags" from "not resolvable."
- Empty-branch atomic claim then push commit: pushing `docs/<N>` empty first won the mutex; ls-remote check before push confirmed no contention.
- DCO `-s` flag + an explicit Signed-off-by in the `-F` message file does NOT duplicate the trailer — git dedupes identical trailers.

### Edge cases discovered
- pkg.go.dev returns 404 for a module page until it has been requested via the proxy at least once (lazy indexing). A 404 on the HTML page is NOT evidence of non-consumability — the proxy `@latest` JSON is the authoritative signal.
- This gh version rejects `merged`/`closingIssuesReferences` JSON fields; use `state`/`mergeCommit`/`mergedAt` and verify closure via `gh issue view`.

### Failed approaches
- None.

### Proposed expert prompt update
- Rule: For docs/release issues asking to verify Go module consumption / pkg.go.dev import path in a monorepo, the authoritative verification is `curl -s https://proxy.golang.org/<module-path>/@latest` (non-empty JSON with Version+Subdir = consumable). Empty `@v/list` means no module-path-prefixed semver tags exist (proxy serves a `v0.0.0-<ts>-<commit>` pseudo-version); a 404 on the pkg.go.dev HTML page is lazy-indexing, not non-consumability. Document both the proxy evidence and the monorepo caveats (repo-level vs `<subdir>/vX.Y.Z` tags; in-repo `replace`/`go.work` per ADR-017 are internal-only, ignored by downstream consumers).
  Category: domain
  Reason: Module-consumption verification recurs for any monorepo release/docs task; the proxy-vs-pkg.go.dev distinction is non-obvious and easy to misreport as a failure.

## Session — 2026-06-17 (issue #1302, Makefile↔CI pip-audit drift)

### Effective patterns
- Read both sides of a drift before editing (Makefile security-agents + ci.yml:666): copy CI's exact flag string so the Makefile line is a verbatim match, not an approximation — eliminates re-drift.
- Local-gate-as-acceptance: run the gate (`make security`) and capture the exit code for the PR test-plan row — clean rc=0 proof without parsing noisy output.

### Structural-workaround (shared-tree / Docker)
- After a container-backed `make` target (e.g. `make security` → uv/pip-audit) runs inside a /tmp worktree, it leaves root-owned `.venv`/cache dirs. `git worktree remove --force` may DE-REGISTER the worktree yet fail to delete the dir ("Permiso denegado"). Do NOT re-run `git worktree remove` (it then reports "not a worktree") — finish with `docker run --rm -v /tmp:/tmp alpine rm -rf <worktree-path>` then `git worktree prune`.

## Session — 2026-06-17 cycle 3 (issue #1204)

### Effective patterns
- "CI runs/tests X" ACs: grep the workflow for the loop's SOURCE variable, not just the loop's presence. A `for x in ${{ env.LIST }}` over a hardcoded-empty `LIST` is a silently-inert gate (green but covers nothing — same class as a skipped required check). Fix with in-job discovery: an `id:` step writing `agents=<glob>` to `$GITHUB_OUTPUT`, consumed as `${{ steps.<id>.outputs.agents }}`, mirroring the Makefile's `$(shell find …)` glob so make and CI never diverge.
- actionlint runs shellcheck and exits 1 on warnings; distinguish NEW from baseline by running against `git show origin/main:<file>`. Prefer a shell-glob `for dir in agents/examples/*/` over `find|xargs basename` (avoids SC2038).
- A `.github/workflows/`-only PR does NOT trigger the python/go lint/test lanes (changes-filter path is `^agents/`), so they show "skipping" and the effect lands on the next matching PR — note this in the PR body.
