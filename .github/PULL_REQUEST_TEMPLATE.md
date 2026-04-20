<!--
Before opening this PR, confirm:
  1. An issue exists and you are assigned to it.
  2. You have read CONTRIBUTING.md and docs/git-workflow.md.
  3. The .feature file was committed BEFORE any implementation.
  4. All CI checks pass locally: make lint test-unit test-integration
-->

## Why

> Explain the problem this change solves. What breaks or is missing without it?
> Link to the issue. Do NOT describe what the code does — the diff shows that.

Closes #

---

## What Changed

> One sentence per logical change. Focus on intent, not implementation.
> If this is a stacked PR, reference the parent: `Stacked on #NNN`

-
-

---

## Type of Change

- [ ] `feat` — new capability
- [ ] `fix` — bug fix
- [ ] `refactor` — no behaviour change
- [ ] `docs` — documentation only
- [ ] `test` — tests only
- [ ] `ci` — CI/CD changes
- [ ] `chore` — maintenance (deps, tooling)

---

## PR Size Self-Check

Lines changed (excluding generated code, lock files, fixtures): ___

- [ ] ≤ 400 lines — proceed
- [ ] 401–800 lines — explain below why it cannot be split
- [ ] > 800 lines — **blocked**: decompose before requesting review

_If over 400 lines, explain here:_

---

## Engineering Checklist

**Git hygiene**
- [ ] Every commit follows Conventional Commits format
- [ ] Every commit has `Signed-off-by:` (DCO)
- [ ] No WIP / fixup commits remain (history cleaned with `git rebase -i main`)
- [ ] Branch rebased onto current `main` (no merge commits from main)

**BDD**
- [ ] `.feature` file committed before any implementation code (link below)
- [ ] All scenarios pass locally: `make test-unit`

**Code quality**
- [ ] `make lint` passes with zero new suppressions
- [ ] Go: `domain/` has zero imports from `api/` or `infrastructure/`
- [ ] Python: `mypy --strict` passes; no untyped `Any`
- [ ] No `print()` / no `panic()` in production paths
- [ ] Functions ≤ 30 lines (Go) / ≤ 20 lines (Python)

**Testing**
- [ ] Unit tests pass: `make test-unit`
- [ ] Integration tests pass: `make test-integration`
- [ ] `domain/` coverage ≥ 90% (Python services)

**Security**
- [ ] No secrets, tokens, or PII in code or fixtures
- [ ] `make security` passes with no new Medium+ findings
- [ ] Input validation on all new API-facing inputs

**Observability**
- [ ] New behaviour emits structured log events (`slog` / `structlog`)
- [ ] New behaviour updates at least one Prometheus metric
- [ ] New gRPC handler has an OpenTelemetry span

**Architecture**
- [ ] No shared database access across service boundaries
- [ ] If proto changed: backward-compatible OR new version + migration guide
- [ ] Layer boundaries respected (see `AGENTS.md §1`)

**Documentation**
- [ ] ADR created if an architectural decision was made (`docs/adr/ADR-NNN-*.md`)
- [ ] `CHANGELOG.md` entry added for user-visible changes
- [ ] Service `AGENTS.md` updated if domain model changed

---

## Feature Files

Link to `.feature` files written for this change (must exist before implementation):

-

---

## AI Assistance

- [ ] No AI assistance used
- [ ] AI-assisted — tool/model: ___________________
      (Add label `ai-assisted`. Add `Assisted-by: <tool>/<model>` to squash commit footer.
      Do NOT use `Co-Authored-By:` for AI tools — that tag is reserved for humans.)

---

## Testing Notes

How did you verify this works? What edge cases did you test? Any manual testing
steps a reviewer should try?

---

## Stacked PR Chain (if applicable)

If this is one PR in a chain, list the sequence:

- [ ] #___ — description (merged / open)
- [x] #___ — **this PR**
- [ ] #___ — description (pending)
