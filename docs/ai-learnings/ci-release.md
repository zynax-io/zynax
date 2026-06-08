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
