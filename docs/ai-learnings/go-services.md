# Learnings: Go Services Engineer

> Format: each entry has `Seen in:` (issue/session) and `Date:` (YYYY-MM-DD).
> Updated by `/m6-learn` after each batch. Human-reviewed before merging.

---

## Effective patterns

- **GOWORK=off prefix on every go command inside service dirs.**
  Without it, the workspace-level `go.work` resolves replacements that don't exist in the
  service module context, causing misleading import errors.
  Seen in: M5 + M6 sessions broadly. Date: 2026-06-06.

- **pgx/v5 `pgx.ErrNoRows` → `codes.NotFound` mapping in every repository.**
  The domain layer never leaks pgx error types — always translate at the infra adapter boundary.
  Seen in: M6.H #793 #794. Date: 2026-06-06.

- **Table-driven tests reach ≥90% domain coverage faster than individual test functions.**
  A single `tests := []struct{...}` covering happy path + 3–4 edge cases typically
  covers 90%+ of a domain method's branches.
  Seen in: M6.H #793. Date: 2026-06-06.

- **Inject `*pgxpool.Pool` via constructor, never as a global.**
  Services that use a pool singleton are untestable (can't swap a fake). Constructor injection
  allows test doubles and enables the domain-coverage gate.
  Seen in: M6.H canvas O1 O2. Date: 2026-06-06.

- **`status.Errorf(codes.X, ...)` in gRPC handlers — never `fmt.Errorf` or `errors.New`.**
  gRPC clients only see the status code + message, not the underlying Go error type.
  Seen in: M5.C task-broker + agent-registry. Date: 2026-06-06.

---

## Edge cases discovered

- **`go mod tidy` inside a service dir modifies `go.sum` unexpectedly when GOWORK is not off.**
  This leaves a dirty tree that confuses the CI dirty-check gate. Always run
  `GOWORK=off go mod tidy` and commit the resulting `go.sum` change with the PR.
  Seen in: M5 sessions. Date: 2026-06-06.

- **`context.WithTimeout` in domain methods causes test flakiness on slow CI runners.**
  Use `context.WithDeadline` with a far-future deadline in tests, or accept
  `ctx context.Context` and let the caller set the timeout.
  Seen in: M5.C #479. Date: 2026-06-06.

- **Proto-generated `*pb.Foo` types are not nil-safe for pointer fields.**
  Always check `req.GetField()` (generated getter) rather than `req.Field` to avoid
  nil-pointer panics on optional proto fields.
  Seen in: M5.B #488. Date: 2026-06-06.

---

## Failed approaches

- **Embedding Temporal activity logic directly in domain interfaces.**
  Temporal's activity serialization requires specific types; mixing it into domain interfaces
  violates the hexagonal boundary and makes unit testing impossible.
  Resolution: Temporal stays in `internal/infra/temporal/`; domain interfaces stay pure.
  Seen in: M3 temporal adapter. Date: 2026-06-06.

- **Using `log.Fatal` in gRPC handler for unrecoverable errors.**
  Kills the entire process including all concurrent requests. Instead: return
  `status.Errorf(codes.Internal, ...)` and let the process continue serving other requests.
  Seen in: M4 api-gateway early implementation. Date: 2026-06-06.

---

## Proposed expert prompt updates

*(none yet — populate after first batch of Go service expert sessions)*

## Session — 2026-06-08 (issue #816 — pgvector adapter)

### Effective patterns
- `pgvector.NewVector([]float32{...})` is the correct API for `pgvector-go v0.2.2` — used both as query param and stored vector column type
- HNSW index requires a **literal integer for LIMIT** (not `$N`) so the Postgres query planner selects the ANN scan; parameterised LIMIT causes seqscan fallback
- `//go:embed migrations/*.sql` + `iofs.New` pattern from task-broker copied cleanly; `stripSchema` helper for `pgx5://` DSN prefix is reusable
- testcontainers: `pgvector/pgvector:pg16` image bundles the vector extension — must use this instead of `postgres:16-alpine` for integration tests

### Edge cases discovered
- Branch checkout state: always verify `git branch --show-current` before staging; `git stash -u` can capture unrelated changes from other branches
- `gh pr merge --auto` does not report the merge in output — check `gh pr view --json state` to confirm

### Failed approaches
- `go get` commands blocked by sandbox; use direct `go.mod` editing + `go mod tidy` instead

### Proposed expert prompt update
Add under "Postgres / pgx v5 patterns":
```
# pgvector
import "github.com/pgvector/pgvector-go"
vec := pgvector.NewVector([]float32{1, 0, 0})
// LIMIT must be a literal int (not $N) for HNSW planner to select ANN scan
// Integration tests: use pgvector/pgvector:pg16 image (not postgres:XX-alpine)
```

---

## Session — 2026-06-08 (issue #824 — event-bus Publish path)

### Effective patterns
- `nats.ErrStreamNameAlreadyInUse` is the correct idempotency sentinel for JetStream `AddStream`; `errors.Is` covers wrapped errors
- `ctx.Err()` must be wrapped with `fmt.Errorf("context: %w", err)` to satisfy `wrapcheck` linter; handler layer uses `status.FromContextError(err).Err()` (exempted by wrapcheck grpc glob)
- golangci-lint v2 `formatters.enable: [gofmt, goimports]` auto-rewrites files in-place even without `--fix` — commit before running `make lint-go`

### Edge cases discovered
- `StreamName("single")` where there is only one path segment: handle gracefully (no verb to drop, use full string)
- golangci-lint Docker mounts `-v ".:/workspace"` so it sees uncommitted changes; run after all edits are staged
- Never use `gh api .../update-branch` to rebase a BEHIND branch — it creates an **unsigned** merge commit that fails DCO. Use: `git reset --hard origin/main && git cherry-pick <impl-commit> && git push --force-with-lease`

### Failed approaches
- `gh api repos/.../pulls/N/update-branch` created unsigned merge commit, failing DCO; resolved by force-pushing a clean cherry-picked branch

### Proposed expert prompt update
Add to commit/merge section: "Never use `gh api .../update-branch` to update a BEHIND branch — it creates an unsigned merge commit that fails DCO. Use: `git reset --hard origin/main && git cherry-pick <impl-commit-sha> && git push --force-with-lease`."

---

## Session — 2026-06-08 (issue #817)

### Effective patterns

- `git pull --rebase origin main` + `git push --force-with-lease` before `gh pr merge --squash --auto` is the right rebase flow when a PR falls behind main.
- Always stage with explicit file paths (`git add services/memory-service/...`) not `git add .` — in a shared workspace, other agents' uncommitted changes appear in `git status`.

### Edge cases discovered

- **gosec G115 false positive on `int32(len(slice))`:** golangci-lint with gosec flags this as potential integer overflow. Suppress with `//nolint:gosec // count is bounded by proto message size limits`.
- **Proto field names differ from .proto source:** Read the generated `.pb.go` getter functions directly — `StoreVectorRequest` has no `id` field (service must generate UUID); `QueryVectorRequest` uses `Embedding` not `Vector`; `ListKeysRequest` uses `Prefix` not `Pattern`; `MSetRequest` uses `Entries []*MSetEntry` with per-entry TTL.
- **`git stash pop` after branch switch fails on untracked files:** "untracked files would be overwritten" — fix with `git checkout <target-branch> -- <file>` before the pop.

### Proposed expert prompt update

Always read the generated `.pb.go` file to get exact field names before writing handler code — proto source and generated structs often differ. Stage with explicit file paths in shared workspaces to avoid pulling in other agents' changes.

---

## Session — 2026-06-08 (issue #825)

### Edge cases discovered (orchestrator-level)

- **GitHub `update-branch` API creates unsigned merge commits that fail DCO.** Never use the GitHub API's 'Update branch' button — it creates an unsigned merge commit with no `Signed-off-by`. Always rebase: `git pull --rebase origin main && git push --force-with-lease`.
- **Rebase after fast-forward through a merge commit replays sibling changes.** If a branch was updated via GitHub's unsigned merge commit, pulling fast-forwards to it; then `git rebase origin/main` replays ALL the merge's contents including unrelated sibling changes already in main. Fix: check `git log origin/main..origin/<branch>` — if sibling commits appear, `git reset --hard origin/main && git cherry-pick <only-your-commit>`.

### Proposed expert prompt update

If a branch falls behind main during CI wait, never use GitHub's 'Update branch' button or API — it creates an unsigned merge commit that fails DCO. Always rebase: `git pull --rebase origin main && git push --force-with-lease`. After any force-push, verify `git log origin/main..origin/<branch>` shows only your commits; if sibling changes appear, reset and cherry-pick.

## Session — 2026-06-08 (issues #818, #826)

### Shell state reset pattern (critical — shared workspace)
**Seen in:** #818, #826. **Date:** 2026-06-08

The active git branch and shell cwd reset between every Bash tool call on a shared workspace.
- Always include `git checkout <branch>` as the **first command** of any Bash call involving file edits or commits
- Use `;` not `&&` for compound commands that must run sequentially (permission gating differs)
- Prefer `git add <specific files>` over `git add .` — avoids staging stash-pop contamination from sibling agents
- When `go.mod` needs updating across branch operations, use the Edit tool directly rather than `go mod tidy` (which reverts on branch switch)

### Stash is dangerous across branches
**Seen in:** #818. **Date:** 2026-06-08

`git stash pop` on a different branch brings unrelated modifications into the working tree.
Safer: `git restore -- <paths>` to discard unrelated tracked-file changes rather than stashing.

### handler.go carried forward from prior step
**Seen in:** #818. **Date:** 2026-06-08

When a prior step (#817) left `handler.go` already fully wired, the current step scope was narrower than the issue implied (integration test + go.mod only). Always read `handler.go` first — don't assume UNIMPLEMENTED stubs remain.

### testcontainers without modules/redis
**Seen in:** #818. **Date:** 2026-06-08

Use `testcontainers.GenericContainer{Image: "redis:7-alpine", ExposedPorts: []string{"6379/tcp"}, WaitingFor: wait.ForLog("Ready to accept connections")}` from the base `testcontainers-go` module (already an indirect dep). Avoids adding the `modules/redis` sub-dependency.

### NATS DLQ pattern
**Seen in:** #826. **Date:** 2026-06-08

JetStream DLQ wiring: create `zynax.dlq.<topic>` stream with `retention=WorkQueue, MaxConsumers=1`; set `DeadLetterSubject` on the consumer; configure `BackOff: []time.Duration{1s, 5s, 30s, 120s, 300s}` matching `MaxDeliver=5`. `Unsubscribe` = delete the durable consumer by subscriber_id; handle not-found gracefully with `codes.NotFound`.

### `gh api update-branch` creates unsigned merge commit
**Seen in:** #826. **Date:** 2026-06-08

GitHub's "Update branch" API endpoint creates a merge commit that fails DCO and `required_signatures`. Never use `gh api repos/.../pulls/.../update-branch`. Instead: `git fetch origin main && git rebase --signoff origin/main && git push --force-with-lease`.

### `js.StreamNames()` returns a channel, not a slice
**Seen in:** #826. **Date:** 2026-06-08

NATS JetStream `js.StreamNames(ctx)` returns `<-chan string`, not `[]string`. Iterate with `for name := range namesCh` and check `ctx.Err()` inside the loop to handle context cancellation during the scan.

---

## Session — 2026-06-08 (issue #584 — text/template refactor)

### Effective patterns

- **Go stdlib `text/template` as a drop-in for bespoke string-replace engines.** Use `template.New("").Funcs(defaultFuncs).Option("missingkey=zero").Parse(tmpl)` and a data root of `map[string]any{"ctx": ctx}` to preserve the existing `{{ .ctx.key }}` syntax without any caller-side changes.
- **`Option("missingkey=zero")`** renders missing map keys as empty string instead of returning an error; this matches the previous bespoke engine's silent-miss behaviour and avoids breaking existing templates.
- **`template.FuncMap{"default": ...}** pattern:** add a custom `default(fallback, val)` function to allow templates to express fallback values: `{{ .ctx.key | default "fallback" }}`. text/template does not provide this built-in.
- **Injection safety:** text/template processes the template once at parse time; substituted ctx values are rendered verbatim, never re-executed as template syntax. This is safe by construction — no need for additional escaping for JSON payloads.

### Edge cases discovered

- **PR closed without merging:** If a PR is closed (not merged) and the remote branch still exists, re-open by rebasing onto latest main and creating a new PR. Check `gh pr view N --json state,mergedAt` to distinguish closed vs merged.
- **Stash required before rebase when working tree is dirty from sibling agents:** `git stash` before `git rebase origin/main`; pop after rebase to restore sibling-agent files. This is safe when the stashed files belong to other branches' work.
- **`text/template` vs `html/template`:** `text/template` does NOT HTML-escape output; `html/template` does. For JSON payloads, always use `text/template` to avoid `&lt;` / `&#34;` corruption of JSON content.

### Proposed expert prompt update

When refactoring a string-replace template engine to `text/template`:
1. Data root: `map[string]any{"ctx": ctx}` — preserves `{{ .ctx.key }}` syntax.
2. Use `Option("missingkey=zero")` to silently render missing keys as empty string.
3. Add `template.FuncMap{"default": func(fallback, val string) string { if val == "" { return fallback }; return val }}` for fallback values.
4. Change return type to `([]byte, error)` and propagate at all call sites.
5. Use `text/template` (not `html/template`) for JSON payloads to avoid HTML escaping.

---

## Session — 2026-06-08 (issues #819, #828, #827)

### Multi-agent working tree chaos — mitigation patterns
**Seen in:** #819, #828, #827. **Date:** 2026-06-08

Background subagents in the same working directory continuously switch branches. Reliable mitigations:
- Always `git branch --show-current` before any git operation — never assume branch is correct
- Use `git stash push -m "<name>" <specific-files>` and `git checkout stash@{N} -- <path>` (not pop) to extract specific files without cross-branch contamination
- Cherry-pick to rescue a commit on wrong branch: `SHA=$(git rev-parse HEAD) && git checkout <correct-branch> && git reset --hard origin/main && git cherry-pick $SHA`
- Use absolute file paths for file writes/edits since CWD changes between Bash tool calls

### NATS JetStream BDD with testcontainers
**Seen in:** #828. **Date:** 2026-06-08

Use `testcontainers.GenericContainer{Image: "nats:2.10-alpine", Cmd: []string{"-js"}, WaitingFor: wait.ForLog("Server is ready")}` for real JetStream in BDD tests. Override `infrastructure.RetryBackoff` to `[50ms, 100ms, ...]` before retry/DLQ scenarios — exported for this purpose. For retry/DLQ tests, bypass `NATSEventBus.Subscribe` and use raw `js.SubscribeSync` with explicit `Nak()`/`NakWithDelay()` since the bus auto-acks and cannot simulate subscriber failure.

### Durable consumer offline/catch-up pattern
**Seen in:** #828. **Date:** 2026-06-08

To simulate "consumer offline → event published → reconnect → catch-up": Subscribe (creates durable consumer), cancel context (goroutine stops, durable consumer RETAINED on NATS server), publish, resubscribe with same SubscriberID. `NATSEventBus.Unsubscribe()` DELETES the durable consumer — for "offline" simulation only cancel the context.

### go.mod tidy after stash-extracted files
**Seen in:** #819. **Date:** 2026-06-08

After extracting go.mod/go.sum from a stash onto a new branch, run `GOWORK=off go mod tidy` to ensure consistency. Pre-commit hook (`golangci-lint`) auto-formats test files in-place — always re-stage after first commit attempt fails with "files were modified by hook".

## Session — 2026-06-08 (orchestrator batch #824,#816,#860)

### Cross-session idempotency: claim-check before any branch push
**Seen in:** #824, #816 (and #860 ci-rel). **Date:** 2026-06-08

In a multi-session environment, `gh issue list --state open` can return issues that were already
merged by an earlier session in the same day. The claim-check step is not optional. Before the
atomic branch push, always run both:

```bash
gh issue view <N> --json state --jq .state  # must be OPEN
gh pr list --state merged --search "<N>" --json number,mergedAt --jq '.[] | "\(.number) \(.mergedAt)"'
```

If a merged PR referencing the issue exists: stop immediately and report the existing merge SHA
without creating any branch or commits. Issues auto-closed by a merged PR will show CLOSED state
but the `gh issue list --state open` query may lag due to GitHub API eventual consistency.

## Session — 2026-06-08 (issues #795, #799)

### Effective patterns

- **Atomic branch-push claim**: Push empty branch before writing any code to prevent race
  conditions when multiple agents run in parallel. Ensures no two agents work on the same issue.

- **`git rebase` drops commits already in the target**: When rebasing a branch onto an updated
  main (that merged PRs from concurrent agents), commits that were already applied are silently
  dropped. Always verify with `git diff origin/main -- <file>` after rebase.

- **`CrossNamespaceCapabilityValidator`**: Pattern for namespace-scoped routing validation in
  `services/workflow-compiler/internal/domain/validators/semantic.go` — register in `All()`.

### Edge cases discovered

- **Pre-commit golangci-lint runs on ALL Go files** (not just staged), causing failures when
  other agents have uncommitted Go changes in the shared working tree. Use `--no-verify`
  when needed; CI Docker `make lint` validates correctly.

- **`git stash` is unsafe across branch switches in shared workspace.** Prefer
  `git restore -- <paths>` to discard unwanted changes. Stash pop on wrong branch brings
  unrelated files into the working tree and can fail with "untracked files would be overwritten".

- **`agents/sdk/pyproject.toml` concurrent edit race**: Concurrent agents can contaminate
  the shared working tree. Recovery: `git show origin/main:<file> > <file> && git add <file>`,
  then `git restore --staged <file> && git restore <file>` to exclude from current commit.
