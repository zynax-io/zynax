# CLAUDE.md — Zynax

Claude Code reads this file automatically. The full engineering contracts live
in `AGENTS.md` files throughout the repository — read those before working in
any layer.

## Milestone Status

| Milestone | Status | Version | Review |
|-----------|--------|---------|--------|
| M1 — Contracts Foundation | **Complete** | v0.1.0 | [Engineering Review](docs/milestones/M1-engineering-review.md) · [Release Notes](docs/milestones/M1-release-notes.md) |
| M2 — Workflow IR | **Complete** | v0.1.0 | [Epic #101](https://github.com/zynax-io/zynax/issues/101) |
| M3 — Temporal Execution | **Next** | v0.2.0 | [Epic #214](https://github.com/zynax-io/zynax/issues/214) |
| M4 — YAML System + CLI | Not started | v0.3.0 | — |

M1 delivered: 8 gRPC contracts, AsyncAPI spec, JSON schemas, Go + Python generated stubs,
140+ BDD scenarios across all services, 5 CI gates. All work tracked in Epic #1 (closed).

M2 delivered: YAML parser + WorkflowGraph builder (#83), structural validators (#84),
semantic validators (#85), WorkflowGraph → WorkflowIR serialization (#86), gRPC API
layer with CompileWorkflow / ValidateManifest / GetCompiledWorkflow (#87), BDD contract
steps (#154), coverage gate ≥90% + make test pipeline (#155, #142).

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
| `docs/patterns/` | Code templates: Go service, Python agent, proto interop, BDD, Helm, SPDD guide |
| `docs/patterns/spdd-guide.md` | Full SPDD workflow — REASONS Canvas, 6 steps, worked examples |
| `docs/spdd/` | REASONS Canvas artifacts — one `canvas.md` per `feat:` issue |
| `state/current-milestone.md` | Active milestone, open PRs, known blockers |

## AI attribution

Every commit **must** carry both trailers or the DCO check fails:

```
Signed-off-by: Oscar Gómez Manresa <ogomezmanresa@gmail.com>
Assisted-by: Claude/claude-sonnet-4-6
```

- `Signed-off-by` — the **human author's** DCO certification. AI cannot certify DCO.
- `Assisted-by` — records AI involvement; use the exact model ID from the session.
- **Never** use `Co-Authored-By:` for AI — reserved for humans certifying DCO.
- **Never** add `🤖 Generated with [Claude Code]` lines to commit messages.
- See `docs/ai-assistant-setup.md` and `CONTRIBUTING.md §AI Contribution`.

## Conventional commit scope rules

PR titles and commit subjects must use a valid type. Scope is optional but recommended.

| Type | When to use |
|------|-------------|
| `feat` | New capability visible to users or callers |
| `fix` | Bug fix |
| `docs` | `AGENTS.md`, `README`, ADR, or any documentation-only change |
| `ci` | GitHub Actions workflows, Makefile CI targets |
| `chore` | Dependency updates, tooling, housekeeping |
| `refactor` | Code restructuring with no behaviour change |
| `test` | Test-only changes |

| Scope | Maps to |
|-------|---------|
| `(workflow-compiler)` | `services/workflow-compiler/` |
| `(agent-registry)` | `services/agent-registry/` and other named services |
| `(protos)` | `protos/` — proto or generated stub changes |
| `(spec)` | `spec/` — YAML schemas or example manifests |
| `(infra)` | `infra/`, Docker, Helm |
| `(agents)` | `agents/` — Python adapters or SDK |
| `(ci)` | Omit scope when type is already `ci` |
| `(docs)` | Omit scope when type is already `docs` |

Rejected prefixes (CI will fail): `spec:`, `proto:`, `adr:`, `service:`, `security:`, `make:`.

## PR size quick-reference

| Lines changed | Status |
|--------------|--------|
| ≤ 200 | Ideal |
| 201–400 | Acceptable — no explanation needed |
| 401–900 | Justify in PR description why it cannot be split |
| > 900 | **Blocked** — decompose before opening PR |

Exclusions from the count: generated stubs (`*.pb.go`, `*_pb2.py`), lock files,
CI workflow files (`.github/workflows/`), schema fixtures.

Split strategy: one commit per logical change; one PR per issue or tightly related
issue group. Never squash unrelated work.

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

## SPDD — feat: PR Workflow

Every `feat:` PR **requires a REASONS Canvas committed before any implementation code.**
This is enforced by ADR-019 and `/spdd-generate` will refuse to run from an unaligned Canvas.

**Prompt-first rule:** requirements change → update Canvas → then patch code. Never the reverse.

```
/spdd-analysis <issue>          → research: codebase scan, ADRs, risk table, Tier 2 flags
/spdd-story <issue>             → decompose into INVEST stories (maps to Canvas O section)
/spdd-reasons-canvas <issue>    → generate docs/spdd/<issue>-<slug>/canvas.md (status: Draft)
/spdd-security-review <canvas>  → Tier 2 scan, injection check — must PASS before commit
[human reviews and sets status: Aligned]
/spdd-generate <canvas>         → implement one Operations step; stop; wait for review
/spdd-prompt-update <canvas>    → requirements changed: update Canvas first, resets to Draft
/spdd-sync <canvas>             → after a refactor: sync Canvas to implementation reality
/spdd-api-test <canvas>         → generate BDD .feature file for a new gRPC boundary
```

Canvas is **Tier 1 only** (public-safe). Move sensitive context to `canvas.private.md` (gitignored).
**Scope:** `feat:` PRs only — `fix:`, `refactor:`, `docs:`, `ci:`, `chore:` are exempt.
Full guide: `docs/patterns/spdd-guide.md` · Template: `docs/spdd/CANVAS_TEMPLATE.md`

## Per-Milestone Scope

| Milestone | In scope | Out of scope / defer |
|-----------|----------|----------------------|
| **M1** (Complete) | Proto contracts, AsyncAPI spec, generated stubs, BDD scenarios, CI gates | Service implementations, DB schemas, runtime |
| **M2** (Complete) | WorkflowIR structured fields in `workflow_compiler.proto`, `WorkflowCompilerService` skeleton (in-memory), JSON Schema for WorkflowIR | Temporal integration, persistence, CLI |
| **M3** (Next) | Temporal-backed `EngineAdapterService` implementation — gated by SPDD Canvas #214 | Other engine adapters, K8s deployment |
| **M4+** | CLI, YAML validation, observability, production hardening | — |

For M3: touch `services/engine-adapter/`. Start with the SPDD Canvas at
`docs/spdd/214-temporal-execution/canvas.md` — no code before Canvas is Aligned.

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
| Any SPDD violation (`feat:` PR without Canvas, Tier 2 in Canvas, code before Canvas update) | See SPDD section above — ADR-019, `docs/patterns/spdd-guide.md` |

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
