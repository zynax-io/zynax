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
