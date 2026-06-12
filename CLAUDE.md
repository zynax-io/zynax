# CLAUDE.md — Zynax

Claude Code reads this file automatically. The full engineering contracts live
in `AGENTS.md` files throughout the repository — read those before working in
any layer.

## Milestone Status

> Live status, per-EPIC progress, and active blockers: [state/current-milestone.md](state/current-milestone.md).
> Milestone goals and sequence: [ROADMAP.md](ROADMAP.md).

## Key pointers

Each directory has an `AGENTS.md`; see [AGENTS.md §Knowledge Base Index](AGENTS.md#knowledge-base-index) for all reference docs and patterns. Notable entry points:

| Path | What it covers |
|------|---------------|
| `AGENTS.md` | Constitution: three-layer arch, mandates, hard constraints, anti-patterns |
| `services/AGENTS.md` | Go service layout, testing, service-layer anti-patterns |
| `agents/AGENTS.md` | Python adapter/SDK pattern, path selection |
| `protos/AGENTS.md` | Proto naming, backward-compat rules, BDD contract tests |
| `cmd/zynax/AGENTS.md` | Standalone CLI — not in go.work; HTTP REST to api-gateway only |
| `docs/adr/INDEX.md` | ADR register — check before proposing any design change |
| `state/current-milestone.md` | Active milestone, open PRs, known blockers |

## AI attribution

> AI attribution rules (trailers, `Assisted-by`, no `Co-Authored-By` for AI):
> see [AGENTS.md §Hard Constraints](AGENTS.md#hard-constraints) and [docs/ai-assistant-setup.md](docs/ai-assistant-setup.md).

## Conventional commit rules

> Commit type rules, valid types, rejected prefixes, and scope-directory mapping:
> see [AGENTS.md §Hard Constraints](AGENTS.md#hard-constraints).

## PR size

≤ 200 lines ideal · 201–400 acceptable · 401–900 justify in description · > 900 **blocked**.
Exclusions (mirror of the `skipPattern` in `.github/workflows/pr-size.yml` — keep in sync):
generated stubs (`*.pb.go`, `*.pb.py`, `/generated/`), lock files (`*.sum`, `*.lock`),
binary fixtures (`*.png`, `*.jpg`, `*.gif`, `*.svg`), `CHANGELOG.md`, `.github/workflows/`,
`AGENTS.md`, `docs/`, `state/`, `.claude/`, `images/images.yaml`, `infra/helm/`, `spec/`,
`automation/`, `Makefile`, `CLAUDE.md`, `ROADMAP.md`, `README.md`.
One commit per logical change · one PR per issue · never squash unrelated work.

## Development workflow

```bash
make bootstrap       # one-time setup (pulls ghcr.io/zynax-io/zynax/tools:latest from GHCR)
make lint            # proto + Go + Python lint
make test            # all unit tests
make generate-protos # regenerate Go + Python stubs (commit the output)
                     # Note: stubs auto-regenerate on main via proto-generate.yml
                     # when .proto or buf config files change (post-merge gate).
make validate-spec   # AsyncAPI + capability schema validation
make security        # govulncheck + bandit + pip-audit
make sync-images     # update banner-marked image refs from images/images.yaml (SoT)
make check-images    # verify banner-marked regions match images/images.yaml (CI gate)
```

> **Image versions** are managed in `images/images.yaml`. Do not hand-edit banner-marked
> regions in workflow files or Dockerfiles — use `make sync-images` to update them.

All commands run inside Docker — only prerequisite is Docker Desktop.

## Testing

**GOWORK=off is required for every `go` command inside `services/*/`, `cmd/zynax/`, and `protos/tests/`.** The workspace root `go.work` lists modules that break the toolchain without this flag (ADR-017).

```bash
cd protos/tests/<service>    # or any service dir
GOWORK=off go test ./... -race -timeout 60s
```

Tiers (ADR-016): BDD at gRPC boundaries (`protos/tests/`), unit ≥ 90% on `internal/domain/`, `buf breaking` as CI gate. BDD `.feature` file committed before any implementation.

## Architecture Invariants

> No shared DB · No Layer 1→3 coupling · Contracts before implementations.
> See [AGENTS.md §The Three-Layer Separation](AGENTS.md#the-three-layer-separation-non-negotiable) and [§Five Non-Negotiable Mandates](AGENTS.md#five-non-negotiable-mandates).
> Engineering culture (15 enforced principles, DORA targets): [docs/contributing/engineering-manifesto.md](docs/contributing/engineering-manifesto.md).

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

> Live progress: [state/current-milestone.md](state/current-milestone.md)

| Milestone | In scope | Out of scope / defer |
|-----------|----------|----------------------|
| **M1** (Complete) | Proto contracts, AsyncAPI spec, generated stubs, BDD scenarios, CI gates | Service implementations, DB schemas, runtime |
| **M2** (Complete) | WorkflowIR structured fields in `workflow_compiler.proto`, `WorkflowCompilerService` skeleton (in-memory), JSON Schema for WorkflowIR | Temporal integration, persistence, CLI |
| **M3** (Partial) | Temporal-backed `EngineAdapterService` — `WorkflowEngine` interface, `IRInterpreterWorkflow`, `DispatchCapabilityActivity`, `TemporalEngine`, gRPC wiring | Other engine adapters, K8s deployment · task-broker delivered later in M5.C |
| **M4** (Partial) | api-gateway REST layer, `zynax` CLI, `kind: AgentDef` routing, Docker Compose runner, GitOps watch | Observability, production hardening · agent-registry delivered later in M5.C |
| **M5** (Complete, v0.4.0) | M5.A docs alignment, M5.B engine fixes, M5.C capability dispatch (task-broker, agent-registry), M5.D security baseline, M5.E DX polish, all 5 adapters, e2e-demo | Persistence, K8s deployment, event-bus (all delivered in M6) |
| **M6** (Complete, v0.5.0) | K8s production-readiness: mTLS, supply-chain hardening, Postgres-backed repos, Helm charts, EventBus over NATS, images.yaml SoT, memory-service, ArgoEngine, multi-namespace, policy/rate-limit, SDK on PyPI, e2e harness, multi-arch builds, gRPC health, Prometheus /metrics | M7 observability (OTel), M8 CNCF submission |

## AI Anti-Patterns

> Full anti-patterns table (Go/Python/proto/commit/SPDD rules): [AGENTS.md §AI Anti-patterns](AGENTS.md#ai-anti-patterns).

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
