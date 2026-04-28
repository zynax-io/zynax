# Zynax — Engineering Constitution

> Authoritative contract for contributors and AI assistants.
> Read entirely before writing any code. Every rule here is backed by an ADR.
>
> **Architecture reference:** Keep this file as a constitution — immutable principles
> only. Detailed patterns live in `docs/patterns/`. Current state in `state/`.

---

## What Is Zynax?

> **Zynax is a declarative, cloud-native, engine-agnostic control plane
> for AI agent workflows.**

It IS:
- The **Kubernetes of AI workflows** — a control plane that abstracts execution
- A **declarative intent layer** — workflows defined in YAML, not code
- An **engine-agnostic adapter** — Temporal, LangGraph, Argo are plugins
- A **capability router** — agents are capabilities, not identities

It is NOT an LLM framework, a workflow engine, or a DevOps tool.

---

## The Three-Layer Separation (Non-Negotiable)

```
┌─────────────────────────────────────────────────────────┐
│  LAYER 1 — INTENT                                       │
│  YAML manifests · Declarative · Versionable             │
│  spec/workflows/ · spec/schemas/                        │
├─────────────────────────────────────────────────────────┤
│  LAYER 2 — COMMUNICATION                                │
│  gRPC (sync) + AsyncAPI/NATS (async) · Typed contracts  │
│  protos/zynax/v1/ · spec/asyncapi/                      │
├─────────────────────────────────────────────────────────┤
│  LAYER 3 — EXECUTION                                    │
│  Workflow Engine Plugins · Pluggable · Swappable        │
│  services/engine-adapter/ · agents/adapters/            │
└─────────────────────────────────────────────────────────┘
```

**Layer violations are hard blockers at code review:**
- Layer 1 (YAML) never imports from Layer 3.
- Layer 2 (contracts) never contains business logic.
- Layer 3 (execution) is always behind an interface — never a hard dependency.

---

## Architecture

```
     YAML Manifests (Intent)
              ↓ zynax apply
       API Gateway (Go)          ← auth · rate limit · REST-to-gRPC
              ↓
     Workflow Compiler (Go)      ← YAML → Canonical IR
              ↓
      Engine Adapter (Go)        ← IR → Temporal / LangGraph / Argo
              ↓
        Task Broker (Go)         ← Capability routing · retry
              ↓
    Execution Adapters Layer     ← LLM · HTTP · Git · CI · LangGraph
              ↓
     Event Bus — NATS (Go)       ← All async events (AsyncAPI spec)
              ↓
     Memory Service (Go)         ← KV + Vector context
```

---

## Development Model

Everything runs inside Docker. **Prerequisites: Docker Desktop only.**

```bash
make bootstrap        # one-time setup
make lint             # proto + Go + Python lint
make test             # all tests (spec + unit + BDD)
make generate-protos  # regenerate Go + Python stubs
make validate-spec    # validate all YAML manifests
make security         # govulncheck + bandit + pip-audit
```

---

## Four Non-Negotiable Mandates

**1 — API Mandate:** All services expose capabilities only via versioned gRPC.
No shared databases. No cross-service imports. Proto files are reviewed like production code.

**2 — Twelve-Factor:** Config via env vars. Stateless processes. Logs to stdout. Port binding.
Backing services as attached resources.

**3 — Clean Code:** Go functions ≤ 30 lines. Python functions ≤ 20 lines.
No magic numbers. No dead code. No `panic` in production. All errors wrapped.

**4 — Layered Testing:** BDD at system boundaries (`.feature` file before any implementation).
Unit/property tests for domain logic (≥ 90% coverage). `buf breaking` as CI gate.
See ADR-016.

---

## Definition of Done

A feature is DONE when **all** are true:

- [ ] System-boundary changes: `.feature` file committed before implementation
- [ ] Domain logic: unit or property tests (≥ 90% coverage on `internal/domain/`)
- [ ] `make test` green · `make lint` clean · `make security` clean
- [ ] Health probes implemented
- [ ] Structured logs + metrics + traces for new behaviour
- [ ] YAML schema updated if new manifest kind added
- [ ] Proto change: backward-compatible OR new version + migration guide
- [ ] ADR created for any architectural decision
- [ ] Required approvals obtained (see `GOVERNANCE.md §2`)

---

## Hard Constraints

**Commit hygiene:**
- Subject ≤ 72 characters, imperative mood, no period
- No `@mentions` in commit messages — issue refs in footer only (`Closes #123`)
- No emojis in commit messages
- Always rebase (`git rebase origin/main`), never merge main into feature branches
- `Assisted-by: Claude/claude-sonnet-4-6` for AI — never `Co-Authored-By:` for AI
- Every commit needs `Signed-off-by: Oscar Gómez Manresa <ogomezmanresa@gmail.com>`

**PR title (CI-enforced `conventional-commit` check):**
- Format: `<type>: <subject>` · total ≤ 72 characters
- Valid types: `feat` `fix` `refactor` `docs` `test` `ci` `chore`
- Rejected: `spec:` `proto:` `adr:` `service:` `make:` `security:`
- Use `docs:` for spec/ADR changes · `chore:` for Makefile/tooling

**Go services:**
- Never `panic` in production · never discard errors (`_ = f()`)
- Never import from another service's `internal/`
- Never hardcode engine names — always behind an interface
- Never disable TLS verification
- Close HTTP response bodies, file handles, and archive readers via `defer`
- `GOWORK=off` for all `go test` and `go` commands inside service directories

**Python agents/adapters:**
- Never call platform services via HTTP — only gRPC stubs
- Never instantiate platform clients in Runtime — use `context.*`
- Never require SDK adoption — adapters work without it
- Never hardcode LLM model names — env var always
- Close all I/O resources in `finally` blocks or context managers

---

## AI Anti-patterns

Observed mistakes in AI-assisted contributions — check before writing code.

| Anti-pattern | Correct approach |
|--------------|-----------------|
| `spec:` / `proto:` / `adr:` / `service:` / `make:` as PR type | Use `docs:` for specs/ADRs, `feat:` or `chore:` for proto, `chore:` for Makefile |
| `go test ./...` without `GOWORK=off` in any service or `protos/tests/` | `GOWORK=off go test ./...` — every invocation, no exceptions (ADR-017) |
| Importing a domain type from one service into another | Cross-service data flows through gRPC only — define the message in proto |
| Editing `protos/generated/` by hand | Edit `.proto` sources, then `make generate-protos` |
| Python code inside `services/` | Python lives only in `agents/` (ADR-009) |
| `Co-Authored-By: Claude …` in commits | Use `Assisted-by: Claude/claude-sonnet-4-6` |
| Omitting `Signed-off-by:` | DCO gate blocks merge |
| New gRPC method without a `.feature` file first | Write and commit `.feature` first (ADR-016) |
| `panic` in production code paths | Return a gRPC status error |
| Hardcoding engine names in business logic | Route through an engine interface (ADR-015) |
| `InsecureSkipVerify: true` in production | TLS on by default; use bufconn in tests only |
| Calling a platform service via HTTP instead of gRPC | Generate stubs with `make generate-protos` |
| Multi-line commit message with zero-indented lines in `run: \|` YAML | Use `printf` with `\n` escape sequences instead |

---

## Knowledge Base Index

| What you need | Where to look |
|--------------|---------------|
| Go service templates (bootstrap, domain, repo, API, Dockerfile) | `docs/patterns/go-service-patterns.md` |
| Python agent options A–D, config, testing, BDD template | `docs/patterns/python-agent-guide.md` |
| Multi-language proto consuming guide | `docs/patterns/proto-interop.md` |
| BDD contract testing (bufconn, godog, two-file split) | `docs/patterns/bdd-contract-testing.md` |
| Helm chart templates (Deployment, HPA, NetworkPolicy, PDB) | `docs/patterns/helm-charts.md` |
| Architecture Decision Records | `docs/adr/INDEX.md` |
| Current milestone and active PRs | `state/current-milestone.md` |
| Per-layer rules | `services/AGENTS.md` · `agents/AGENTS.md` · `protos/AGENTS.md` · `spec/AGENTS.md` · `infra/AGENTS.md` |

---

*Zynax — The control plane for AI-driven systems · Apache 2.0 · CNCF Sandbox Candidate*
