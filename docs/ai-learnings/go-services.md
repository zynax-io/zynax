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
