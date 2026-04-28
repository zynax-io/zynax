# CLAUDE.md — Zynax

Claude Code reads this file automatically. The full engineering contracts live
in `AGENTS.md` files throughout the repository — read those before working in
any layer.

## Milestone Status

| Milestone | Status | Version | Review |
|-----------|--------|---------|--------|
| M1 — Contracts Foundation | **Complete** | v0.1.0 | [Engineering Review](docs/milestones/M1-engineering-review.md) · [Release Notes](docs/milestones/M1-release-notes.md) |
| M2 — Workflow IR | **In progress** | v0.1.0 | Epic #101 |
| M3 — Temporal Execution | Not started | v0.2.0 | — |
| M4 — YAML System + CLI | Not started | v0.3.0 | — |

M1 delivered: 8 gRPC contracts, AsyncAPI spec, JSON schemas, Go + Python generated stubs,
140+ BDD scenarios across all services, 5 CI gates. All work tracked in Epic #1 (closed).

M2 progress: workflow-compiler bootstrap (#82), domain types + YAML parser (#83 part 1),
WorkflowGraph builder (#83 part 2). Remaining: structural validators (#84), semantic
validators (#85), IR serialization (#86), gRPC API layer (#87).

## Key pointers

| Directory | AGENTS.md covers |
|-----------|-----------------|
| `/` | Three-layer architecture, workflow model, hard constraints |
| `services/` | Go service structure, domain/api/infra separation |
| `agents/` | Python adapter pattern, gRPC stub usage |
| `protos/` | Proto naming, backward-compatibility rules |
| `spec/` | YAML manifest schemas |
| `infra/` | Docker, env var conventions |
| `docs/adr/INDEX.md` | Searchable ADR register — check here before proposing a design change |
| `docs/architecture/` | Architecture reviews, competitive analysis |

## AI attribution

Every commit **must** carry both trailers or the DCO check fails:

```
Signed-off-by: Your Name <your@email.com>
Assisted-by: Claude/claude-sonnet-4-6
```

- `Signed-off-by` — required by the DCO gate on every commit, human or AI-assisted.
- `Assisted-by` — records AI involvement; use the exact model ID from the session.
- **Never** use `Co-Authored-By:` for AI — reserved for humans certifying DCO.
- **Never** add `🤖 Generated with [Claude Code]` lines to commit messages.
- See `docs/ai-assistant-setup.md` and `CONTRIBUTING.md §AI Contribution`.

## Development workflow

```bash
make bootstrap       # one-time setup (builds zynax-tools Docker image)
make lint            # proto + Go + Python lint
make test            # all unit tests
make generate-protos # regenerate Go + Python stubs (commit the output)
                     # Note: stubs auto-regenerate on main via proto-generate.yml
                     # when .proto or buf config files change (post-merge gate).
make validate-spec   # AsyncAPI + capability schema validation
```

All commands run inside Docker — only prerequisite is Docker Desktop.

## Testing approach (M1 contracts layer)

BDD `.feature` files are committed before any boundary implementation (ADR-016).
Go BDD tests live in `protos/tests/<service>/` and use [godog](https://github.com/cucumber/godog).

**Critical:** run contract tests with `GOWORK=off` — the `go.work` workspace lists
service modules not yet created (M2+), which break `go test` without this flag:

```bash
cd protos/tests/<service>
GOWORK=off go test ./... -v -timeout 60s
```

CI enforces this in `.github/workflows/ci.yml` `test-unit` job (Godog BDD step).

**Also applies to service modules.** Running `go test ./...` inside `services/<svc>/`
also picks up the workspace root's `go.work`. Use `GOWORK=off` for all `go` commands
run from within any service directory — not just `protos/tests/`.

Testing tiers per ADR-016:
- BDD (10–15%): system boundaries, gRPC contracts — `protos/tests/`
- Unit/property (≥40%): domain logic — per-service `_test.go`
- Contract (CI gate): `buf breaking` on every proto PR
- Simulation (M3+): fault injection, retry storms

## Architecture Invariants

These three rules must never be broken regardless of milestone:

1. **No shared database.** Each service owns its own schema/namespace. Cross-service
   reads go through gRPC, never through a shared table or ORM model.
2. **No Layer 1→3 coupling.** YAML manifests (`spec/`) are never imported by Go
   services. The Workflow Compiler transforms Layer 1 → Layer 2 (WorkflowIR).
3. **Contracts before implementations.** `.proto` files and `.feature` files are
   committed and CI-green before any service implementation begins (ADR-016).

## Per-Milestone Scope

| Milestone | In scope | Out of scope / defer |
|-----------|----------|----------------------|
| **M1** (Complete) | Proto contracts, AsyncAPI spec, generated stubs, BDD scenarios, CI gates | Service implementations, DB schemas, runtime |
| **M2** (next) | WorkflowIR structured fields in `workflow_compiler.proto`, `WorkflowCompilerService` skeleton (in-memory), JSON Schema for WorkflowIR | Temporal integration, persistence, CLI |
| **M3** | Temporal-backed `EngineAdapterService` implementation | Other engine adapters, K8s deployment |
| **M4+** | CLI, YAML validation, observability, production hardening | — |

For M2: touch `protos/zynax/v1/workflow_compiler.proto` and `services/workflow-compiler/`.
Do not touch `services/engine-adapter/` or any Temporal code — that is M3.

## Common AI Anti-Patterns

Things that have gone wrong in this repo — avoid these:

| Anti-pattern | Correct approach |
|--------------|-----------------|
| Writing Python in `services/` | All platform services are Go (ADR-009). Python lives only in `agents/` |
| Editing `protos/generated/` directly | Run `make generate-protos` — generated files are never hand-edited |
| Mocking the database in integration tests | Use `testcontainers-go` for real DB (ADR-016) |
| Adding complexity beyond the issue scope | Implement exactly what the issue asks; open a follow-up for anything extra |
| Using `Co-Authored-By:` for AI | Use `Assisted-by: Claude/<model>` — DCO is human-only |
| Omitting `Signed-off-by:` from a commit | Every commit needs `Signed-off-by: Name <email>` or the DCO gate fails — include it even on AI-assisted commits |
| PR title prefix `spec:` / `proto:` / `adr:` | Use `docs:` for spec/ADR changes, `feat:`/`chore:` for proto changes |
| Running `go test` in `protos/tests/` without `GOWORK=off` | Always prefix: `GOWORK=off go test ./...` (ADR-017) |
| Running `go test` in `services/<svc>/` without `GOWORK=off` | Same rule — applies to ALL go commands in any service directory |
| Embedding multi-line scripts in `run: \|` YAML blocks | Extract to `tools/<name>.py` — un-indented Python terminates the YAML scalar |
| Using `govulncheck@latest` with Go 1.22 | Pin to `GOVULNCHECK_VERSION` env var — @latest requires Go ≥ 1.25 |
| `golang:1.22-alpine` COPY paths using `/root/go/bin/` | Use `/go/bin/` — GOPATH on Alpine is `/go`, not `/root/go` |
| Importing domain types across services | Cross-service communication is gRPC only, never shared types |

## Decision-Making Guide

**Create an issue vs just fix it:** If the change touches an interface visible to
other layers (proto field, event schema, API contract), open an issue first. For
internal refactors within a single service, fix directly.

**Create an ADR vs just do it:** Any decision that another engineer would reverse
without knowing the rationale needs an ADR. One-way doors always get ADRs.
Reversible implementation choices do not.

**Ask the user vs proceed:** Proceed if the task is within the current issue scope
and the approach is consistent with existing ADRs. Ask if the work would require
touching files outside the stated scope, or if two valid approaches exist with
materially different tradeoffs.
