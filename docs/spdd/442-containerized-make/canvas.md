# REASONS Canvas — Fully Containerized Makefile-Based Dev Workflow

**Issue:** #442
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-13
**Status:** Aligned

---

## R — Requirements

### Problem statement

Without this feature, four categories of developer friction exist:

1. **Go adapter modules escape CI gates.** `GO_SERVICES` only lists platform services under `services/`. Adapter modules under `agents/adapters/` (already in `go.work`) are silently excluded from `make lint`, `make test`, `make security`, and `make test-coverage`. New adapters accumulate lint debt undetected until CI catches them post-push.

2. **`gitleaks` requires a local binary.** It is the only CI tool not containerized through `zynax-tools:local`. Developers on machines without a local `gitleaks` install cannot run `make gitleaks`, creating an inconsistency with the "Docker Desktop only" prerequisite stated in `AGENTS.md`.

3. **No single local CI entrypoint.** `make test` covers spec, unit, BDD, and coverage but omits lint, security, and secret scanning. Developers must know and run multiple targets in the correct sequence to reproduce CI locally.

4. **Python agents hardcoded in Makefile.** `AGENTS := summarizer researcher calculator` must be updated manually when new agents are added under `agents/examples/`. New agents escape `lint-agents`, `test-unit-agents`, and `security-agents` coverage silently until someone notices the variable is stale.

### Definition of done

- `make lint` covers `services/`, `agents/adapters/`, and `agents/sdk` + all discovered example agents.
- `make test` covers `services/`, `agents/adapters/`, `agents/sdk`, and all discovered example agents.
- `make security` covers `services/`, `agents/adapters/`, `agents/sdk`, and all discovered example agents.
- `make gitleaks` runs containerized via `TOOLS_IMAGE` with no local binary required.
- `make ci` runs all gates in sequence and exits 0 iff the full pipeline is green.
- Adding a new `agents/adapters/<name>` Go module to `go.work` causes it to appear in lint/test/security without any Makefile edit.
- Adding a new `agents/examples/<name>/pyproject.toml` causes it to appear in lint/test/security without any Makefile edit.

---

## E — Entities

```
Makefile
├── GO_SERVICES         (existing) — platform service modules under services/
├── GO_ADAPTERS         (new) — Go adapter modules under agents/adapters/
│                         source of truth: go.work workspace entries
├── AGENTS              (replaced by auto-discovery) — Python example agents
│                         discovered via: agents/examples/*/pyproject.toml
├── TOOLS_RUN           (existing) — docker run wrapper for zynax-tools:local
└── Targets (existing)
    ├── lint            → lint-protos + lint-go + lint-agents
    ├── test            → validate-spec + test-unit + test-bdd + test-coverage
    ├── security        → security-go + security-agents
    └── gitleaks        → (currently local binary; containerized by step 2)

Dockerfile.tools (infra/docker/Dockerfile.tools)
└── Stage 1 (go-tools): adds gitleaks binary with pinned version + Renovate comment

New Makefile Targets
├── lint-go-adapters       — golangci-lint per adapter module (TOOLS_RUN)
├── test-unit-adapters     — GOWORK=off go test per adapter module (TOOLS_RUN)
├── test-coverage-adapters — coverage gate ≥80% per adapter module (TOOLS_RUN)
├── security-go-adapters   — govulncheck per adapter module (TOOLS_RUN)
└── ci                     — validate-spec → lint → test → security → gitleaks
```

---

## A — Approach

### What we WILL do

- Add `GO_ADAPTERS` Makefile variable listing Go adapter modules from `go.work` (initially `agents/adapters/http`; expands as `git`, `ci` adapters land).
- Add `lint-go-adapters`, `test-unit-adapters`, `test-coverage-adapters`, `security-go-adapters` targets that iterate `GO_ADAPTERS` with the same `TOOLS_RUN` + `GOWORK=off` pattern as the existing service targets.
- Extend `lint`, `test`, `security` aggregate targets to call the new adapter targets.
- Replace the hardcoded `AGENTS` variable with shell auto-discovery: `find agents/examples -maxdepth 2 -name pyproject.toml | xargs -I{} dirname {} | xargs -I{} basename {}`.
- Add `gitleaks` binary to `Dockerfile.tools` Stage 1 with a pinned version matching `.pre-commit-config.yaml` and a `# renovate: datasource=github-releases depName=gitleaks/gitleaks` comment.
- Update the `gitleaks` Makefile target to invoke `TOOLS_RUN` instead of the local binary.
- Add `make ci` target sequencing: `validate-spec → lint → test → security → gitleaks`.

### What we WON'T do

- We will NOT modify `ci.yml` — change-aware per-module detection was already added in #439–441.
- We will NOT containerize the pre-commit `golangci-lint` hook (pre-commit `language: system` hooks cannot invoke Docker; this is a known limitation documented in `.pre-commit-config.yaml`).
- We will NOT implement dynamic go.work parsing in Makefile (shell `grep` on go.work is sufficient and has no external dependencies).
- We will NOT change any Go service or adapter source code.
- We will NOT change any Python agent source code.
- We will NOT implement parallel gate execution in `make ci` (sequential fail-fast is correct for local dev).

### ADR references

- ADR-006 (Monorepo): `go.work` is the authoritative registry for Go workspace modules — `GO_ADAPTERS` must reflect it.
- ADR-009 (Language strategy): Go adapters belong in the `lint-go` target family, not `lint-agents`.
- ADR-016 (Layered testing): coverage gate ≥90% for `internal/domain/` applies across all Go modules. Adapters are integration-heavy; a ≥80% gate on their total coverage is the chosen floor.
- ADR-017 (Contract test isolation): every `go test` and `go` command inside module directories must use `GOWORK=off`.

---

## S — Structure

**Files changed:**

| File | Change |
|------|--------|
| `Makefile` | Add `GO_ADAPTERS`; replace `AGENTS` with auto-discovery; add `lint-go-adapters`, `test-unit-adapters`, `test-coverage-adapters`, `security-go-adapters`, `ci` targets; extend aggregates |
| `infra/docker/Dockerfile.tools` | Add `gitleaks` binary to Stage 1 with pinned version + Renovate comment |
| `docs/local-dev.md` | Verify or add `make ci` reference as recommended pre-push command |

**No gRPC contracts, proto files, or service implementations are touched.**

---

## O — Operations

1. **Step 1 (#443) — Extend Makefile with Go adapter lint, test, and security targets.**
   Add `GO_ADAPTERS` variable; add `lint-go-adapters`, `test-unit-adapters`, `test-coverage-adapters`, `security-go-adapters` targets; extend `lint`, `test`, `security` aggregates. All via `TOOLS_RUN` + `GOWORK=off`.

2. **Step 2 (#444) — Containerize gitleaks via TOOLS_IMAGE.**
   Add `gitleaks` binary to `Dockerfile.tools` Stage 1 with pinned version (v8.21.2) and Renovate comment. Update `make gitleaks` to use `TOOLS_RUN`. Verify `make build-tools` succeeds.

3. **Step 3 (#445) — Add `make ci` top-level target.**
   Add `ci` target sequencing all gates: `validate-spec → lint → test → security → gitleaks`. Add `★` marker to `make help` output. Verify/update `docs/local-dev.md` to reference `make ci`. Depends on step 2 (gitleaks must be containerized).

4. **Step 4 (#446) — Auto-discover Python agents in Makefile.**
   Replace hardcoded `AGENTS` variable with shell auto-discovery from `agents/examples/*/pyproject.toml`. Verify `lint-agents`, `test-unit-agents`, `security-agents` still pass. Verify single-agent targets (`lint-agent`, `test-unit-agent`) still work. Independent of steps 1–3.

---

## N — Norms

From root `AGENTS.md` Hard Constraints and `CLAUDE.md`:

- **Commit hygiene:** Every commit carries `Signed-off-by` and `Assisted-by` trailers per AGENTS.md §Hard Constraints. Never `Co-Authored-By:` for AI.
- **PR title convention:** `ci:` for Makefile CI gate changes, `chore:` for tooling/dependency housekeeping. Never `make:` or `security:` as type prefix (CI-rejected).
- **PR size:** ≤ 200 lines ideal. All four steps fit within that bound individually.
- **GOWORK=off:** Every `go test` and `go build` inside any module directory uses `GOWORK=off` (ADR-017). This norm propagates to every new Makefile target that invokes `go`.
- **TOOLS_RUN:** All tool invocations in Makefile targets use `$(TOOLS_RUN)` — no bare tool calls that would require a local install.
- **Renovate compatibility:** Any pinned version added to `Dockerfile.tools` must include a `# renovate: datasource=github-releases depName=<org>/<repo>` comment so Renovate can auto-propose upgrades.
- **No shell scripts for logic that belongs in Makefile:** New targets are Makefile recipes, not separate shell scripts, unless complexity demands it.
- **`make help` coverage:** Every user-facing target has a `## <description>` comment so it appears in `make help` output.

---

## S — Safeguards

### Context Security (mandatory before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no email addresses (email removed from N section per BLOCK finding — replaced with AGENTS.md §Hard Constraints reference)
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file (BLOCK finding resolved — email literal removed from N section)

### Feature Safeguards

- **Never** hardcode a module path in `GO_ADAPTERS` that is not present in `go.work` — the variable must stay consistent with the workspace definition (ADR-006).
- **Never** remove `GOWORK=off` from any `go` command in a new Makefile target — even if it appears to work without it locally (ADR-017).
- **Never** invoke a tool binary directly in a Makefile target that uses `ensure-tools` as a prerequisite — always go through `TOOLS_RUN` to guarantee the containerized version is used.
- **Never** commit a version to `Dockerfile.tools` without a matching `# renovate:` comment — un-annotated pins go stale silently.
- **Never** change `ci.yml` as part of this initiative — CI workflow changes were completed in #439–441 and are out of scope.
- **Never** modify any Go service, Go adapter, or Python agent source code as part of this initiative — scope is Makefile and Dockerfile.tools only.
