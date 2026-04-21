# CLAUDE.md — Zynax

Claude Code reads this file automatically. The full engineering contracts live
in `AGENTS.md` files throughout the repository — read those before working in
any layer.

## Milestone Status

| Milestone | Status | Version | Review |
|-----------|--------|---------|--------|
| M1 — Contracts Foundation | **Complete** | v0.1.0 | [Engineering Review](docs/milestones/M1-engineering-review.md) · [Release Notes](docs/milestones/M1-release-notes.md) |
| M2 — Workflow IR | Not started | v0.1.0 | — |
| M3 — Temporal Execution | Not started | v0.2.0 | — |
| M4 — YAML System + CLI | Not started | v0.3.0 | — |

M1 delivered: 8 gRPC contracts, AsyncAPI spec, JSON schemas, Go + Python generated stubs,
140+ BDD scenarios across all services, 5 CI gates. All work tracked in Epic #1 (closed).

## Key pointers

| Directory | AGENTS.md covers |
|-----------|-----------------|
| `/` | Three-layer architecture, workflow model, hard constraints |
| `services/` | Go service structure, domain/api/infra separation |
| `agents/` | Python adapter pattern, gRPC stub usage |
| `protos/` | Proto naming, backward-compatibility rules |
| `spec/` | YAML manifest schemas |
| `infra/` | Docker, env var conventions |

## AI attribution

- Use `Assisted-by: Claude/claude-sonnet-4-6` in commit footers.
- **Never** use `Co-Authored-By:` for AI — reserved for humans certifying DCO.
- **Never** add `🤖 Generated with [Claude Code]` lines to commit messages.
- See `docs/ai-assistant-setup.md` and `CONTRIBUTING.md §AI Contribution`.

## Development workflow

```bash
make bootstrap       # one-time setup (builds zynax-tools Docker image)
make lint            # proto + Go + Python lint
make test            # all unit tests
make generate-protos # regenerate Go + Python stubs (commit the output)
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

Testing tiers per ADR-016:
- BDD (10–15%): system boundaries, gRPC contracts — `protos/tests/`
- Unit/property (≥40%): domain logic — per-service `_test.go`
- Contract (CI gate): `buf breaking` on every proto PR
- Simulation (M3+): fault injection, retry storms
