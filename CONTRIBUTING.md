<!-- SPDX-License-Identifier: Apache-2.0 -->

# Contributing to Zynax

Thank you for your interest in contributing. Zynax is built by its community and held
to the standards of the best-in-class CNCF projects. This guide is the single source
of truth for how work flows from idea to merged code.

Read this document before opening your first PR.

---

## 1. Before You Start

1. **Read [`AGENTS.md`](AGENTS.md)** — the full engineering contract. Required for all
   contributors, human and AI alike.
2. **Read [`docs/git-workflow.md`](docs/git-workflow.md)** — the definitive reference
   for branching, commits, and PR decomposition.
3. **Check existing ADRs** in `docs/adr/` — your idea may already be decided.
4. **Open an issue first** for any non-trivial change. Discuss before coding.
   "Non-trivial" means: new capabilities, design decisions, changes to contracts,
   anything that touches more than one service.
5. **Sign the DCO** — every commit must have `Signed-off-by: Your Name <email>` in
   the commit message (see §5). Enforced by the DCO bot.

---

## 2. Community & Communication

| Channel | Purpose | When to use |
|---------|---------|------------|
| [GitHub Issues](https://github.com/zynax-io/zynax/issues) | Bug reports, feature requests, ADR proposals | Actionable work items |
| [GitHub Discussions](https://github.com/zynax-io/zynax/discussions) | Questions, ideas, design exploration | Before an issue exists |
| [GitHub PRs](https://github.com/zynax-io/zynax/pulls) | Code review, implementation discussion | After an issue is triaged |

**Rules:**
- Do not use issues for questions — use Discussions. Issues are reserved for
  actionable work items.
- Do not DM maintainers for technical support — use Discussions so others benefit.
- English is the working language for all GitHub communication.
- Be patient. Maintainers are volunteers. Response SLA: **2 business days** for
  initial response on issues and PRs.

---

## 3. Development Environment

### Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | ≥ 1.26 | [go.dev](https://go.dev) |
| Python | ≥ 3.12 | `uv python install 3.12` |
| uv | latest | `curl -LsSf https://astral.sh/uv/install.sh \| sh` |
| Docker | ≥ 24 | [docker.com](https://docker.com) |
| buf | 1.28+ | `brew install bufbuild/buf/buf` |
| pre-commit | latest | `pip install pre-commit` or `brew install pre-commit` |
| make | any | pre-installed on macOS/Linux |

### Setup

```bash
git clone https://github.com/zynax-io/zynax.git
cd zynax
make bootstrap        # Pulls ghcr.io/zynax-io/zynax/tools:latest from GHCR + runs `pre-commit install`
```

`make bootstrap` wires the pre-commit hooks into `.git/hooks/` so they fire
automatically on every `git commit`. **You must run this once per clone.**
If `pre-commit` is not yet installed, `make bootstrap` prints a warning —
install it first, then re-run `make bootstrap`.

```bash
make dev-up           # Start local stack: PostgreSQL, Redis, NATS, all services
make dev-ps           # Verify everything is healthy
```

### Running Tests

```bash
make test-unit             # Fast unit tests — run constantly
make test-integration      # Requires Docker — run before pushing
make test-unit-svc SVC=agent-registry   # Single service focus
```

#### Unit vs integration test separation

Tests that require external services (NATS, Redis, Temporal, a real database) must
carry a Go build tag on the **very first line** of the file, before the package declaration:

```go
//go:build integration

package mypackage_test
```

- `make test-unit` runs `go test -tags="" ./...` — build-tagged files are **excluded**
- `make test-integration` runs `go test -tags=integration ./...` — they are **included**
- CI enforces this: the `test-unit` job never passes `-tags=integration`

Use `testcontainers-go` inside integration tests to spin up real backing services.
The `//go:build integration` tag prevents these tests from silently failing on
machines where Docker or the service is not running.

### Testing the zynax CLI end-to-end

The `zynax` CLI is a standalone module under `cmd/zynax/`. To test it against the
running platform:

```bash
# 1. Install the CLI (requires Go 1.26 locally, or download from GitHub Releases)
make install-cli            # builds cmd/zynax and installs to ~/bin/zynax

# 2. Start the local stack
make run-local              # api-gateway :7080, Temporal UI :7088

# 3. Apply a workflow
export ZYNAX_API_URL=http://localhost:7080
zynax apply spec/workflows/examples/code-review.yaml
# → run_id: wf-<hex>

zynax status workflow wf-<hex>
zynax logs wf-<hex>

# 4. Stop when done
make stop-local
```

CLI unit tests (no running stack needed):
```bash
cd cmd/zynax && GOWORK=off go test ./... -race -timeout 60s
```

### Pre-commit hooks

The hooks are declared in `.pre-commit-config.yaml`. They do **nothing** until you
activate them once per clone — `make bootstrap` does this automatically.

**Activate manually** (if you didn't run `make bootstrap`):

```bash
pip install pre-commit   # or: brew install pre-commit
pre-commit install       # wires hooks into .git/hooks/pre-commit
```

After that, on every `git commit` the following checks run automatically:

| Hook | What it checks | How it's installed |
|------|---------------|-------------------|
| `gitleaks` | Secret / credential scan | **Auto** — pre-commit manages it |
| `ruff` | Python lint + formatting | **Auto** — pre-commit manages it |
| `gofmt` | Go formatting (formats in-place) | Needs Go toolchain locally |
| `golangci-lint` | Go static analysis per module (`tools/golangci-lint-precommit.sh`) | Needs `golangci-lint` (`make install-ci-tools`) |
| `mypy` | Python type-checking (`agents/sdk/src/`) | Needs `mypy` (`uv tool install mypy`) |

`gitleaks` and `ruff` are **managed hooks** — pre-commit downloads and caches them
automatically on first run; no manual install needed.

`gofmt`, `golangci-lint`, and `mypy` use `language: system` — if the binary is not
in your `PATH` the hook will fail and block the commit. Install them as part of the
normal dev setup (`make install-ci-tools` handles golangci-lint and zynax-ci).

> The same checks run in CI via `make lint` (Docker-based). If you're missing a
> local tool, use `git commit --no-verify` and note it in the PR — CI will catch it.

Run all hooks manually at any time: `pre-commit run --all-files`

**Bypassing hooks:** Use `git commit --no-verify` only for legitimate reasons (e.g. a
work-in-progress checkpoint on a branch). Add a PR comment explaining why — reviewers
will ask if you don't.

### Linting

```bash
make lint              # Check all: golangci-lint + ruff + mypy + buf
make lint-fix          # Auto-fix what can be auto-fixed
make lint-proto        # Proto-only: buf lint + buf breaking
```

---

## 4. Engineering Standards

See [`AGENTS.md §Hard Constraints`](AGENTS.md#hard-constraints) and the per-layer
files `services/AGENTS.md`, `agents/AGENTS.md`, and `protos/AGENTS.md`.
CI enforces all standards — PRs fail if any check is red.

### Forcing a full CI run

By default, CI skips unaffected lanes (e.g. a docs-only PR does not run Go tests).
Three opt-in mechanisms force every lint, test, and security lane to run regardless
of what changed:

| Mechanism | How | When to use |
|-----------|-----|-------------|
| `workflow_dispatch` with `force: true` | `gh workflow run ci.yml -f force=true` | Release prep, ad-hoc full validation |
| PR label `ci: force-full` | Add the label to your PR | Suspect cross-service regression |
| Commit keyword `[full-ci]` | Include `[full-ci]` in the commit message or PR title | Single commit, targeted need |

When a force signal is active, the run summary shows a warning annotation identifying
the triggering mechanism.

---

## 5. Git Workflow & Commit Hygiene

> Full reference: [`docs/git-workflow.md`](docs/git-workflow.md)

**Branch naming:** `<type>/issue-<number>-<short-kebab-description>`
Valid types: `feat`, `fix`, `docs`, `test`, `refactor`, `ci`, `chore`.
Feature branches must not live longer than 7 days without merging or going Draft.

**Commit format** ([Conventional Commits](https://conventionalcommits.org) — CI-enforced):

```
<type>(<scope>): <short description, imperative, total header ≤ 72 chars>

<body: WHY this change is needed>

Closes #123
Signed-off-by: Your Name <your@email.com>
Assisted-by: Claude/claude-sonnet-4-6    ← AI-assisted only
```

Rules: imperative mood · no period · no `@mentions` in subject · no emojis ·
clean history before opening PR (`git rebase -i main`).

**Keeping branches current:** always rebase, never merge main into your branch.
Push after rebase with `--force-with-lease`, never bare `--force`.

**GPG signing (required):** set `git config commit.gpgsign true`. Branch protection
rejects unsigned commits.

**DCO sign-off (required):** add with `git commit -s`. DCO bot blocks merge without it.
GPG proves identity; DCO certifies you have the right to contribute (Apache 2.0).

---

## 6. PR Size & Decomposition

| Lines changed | Status |
|--------------|--------|
| ≤ 200 | Ideal |
| 201–400 | Acceptable |
| 401–900 | Justify in PR description why it cannot be split |
| > 900 | **Blocked** — decompose before requesting review |

**Exclusions from count:** generated code (`*.pb.go`, `*_pb2.py`), lock files,
schema fixtures, CI workflow files (`.github/workflows/`).

Large features ship as a **PR chain** — a sequence of small, mergeable PRs where
each one compiles, passes tests, and delivers observable value. Reference the
parent issue from every PR in the chain. For stacked PRs (each based on the
previous), use `Stacked on #NNN` in the PR description.

---

## 7. Layered Testing Strategy

Zynax uses a four-tier testing pyramid (ADR-016).

| Tier | When to use | Tools |
|------|-------------|-------|
| **BDD** | Agent contracts, inter-service gRPC, E2E workflows | pytest-bdd, godog |
| **Unit / property-based** | Domain logic, routing, state transitions | pytest + hypothesis, testing + rapid |
| **Contract** | Any proto or schema change | `buf breaking`, `make validate-spec` |
| **Simulation** | Fault injection, retry storms, topology changes | testcontainers harness (coming) |

**BDD-first rule (required when BDD applies):** commit the `.feature` file before any
implementation. BDD applies at service boundaries only — not internal domain logic.
See [`docs/patterns/bdd-contract-testing.md`](docs/patterns/bdd-contract-testing.md).

---

## 8. Pull Request Process

1. **Open an issue first** for any non-trivial change. No PR without an issue.
2. Fork, branch, implement — open PR against `main` using the PR template.
3. Title must follow Conventional Commits (CI-enforced). Fill all template sections.
4. Mark **Draft** if in progress; convert to **Ready** only when CI passes locally.
5. Push fixup commits during review — do not force-push while reviewers are active.
6. After all approvals: rebase onto `main`, `git push --force-with-lease`, then merge.

**Merge requirements:**
- [ ] All CI checks green (lint, test-unit, test-integration, security, dco)
- [ ] No unresolved review comments · PR description complete · CHANGELOG updated

**Merge strategy: squash-and-merge** (`gh pr merge <PR> --squash`). No merge commits.
`required_linear_history` is enforced by branch protection — merge commits are rejected
for all actors including admins (ADR-023). `--rebase` is blocked by `required_signatures`
(GitHub cannot auto-sign replayed commits); squash-merge is GitHub-signed and linear.

Sequence before every merge:
```bash
git fetch origin main
git rebase origin/main        # resolve conflicts if any
git push --force-with-lease
gh pr checks <PR> --watch
gh pr merge <PR> --squash
git push origin --delete <branch>   # delete remote branch immediately after merge
```

**Branch lifecycle:**
- Create branches off fresh `origin/main` immediately before work.
- Delete the remote branch immediately after merge (`git push origin --delete <branch>`).
  GitHub's "Automatically delete head branches" setting enforces this for merge-button ops.
- **Never reopen a closed PR or stale branch.** If commits from a closed branch are still
  wanted, cherry-pick or rebase them onto a fresh branch off current `main`, open a new PR,
  run CI, then squash-merge.

Solo maintainer phase: CI must pass; maintainer may self-merge non-breaking changes.
Breaking and proto changes require a 5-day RFC comment period (see `GOVERNANCE.md §2`).

---

## 9. Code Review Etiquette

**Authors:** respond to all comments; push fixup commits during review (no amend);
rebase onto current `main` immediately before merge; no AI-generated replies.

**Reviewers:** use explicit severity prefixes:

| Prefix | Meaning | Blocks merge? |
|--------|---------|--------------|
| `BLOCKER:` | Must be fixed before merge | Yes |
| `CONCERN:` | Architecturally significant | Usually yes |
| `Nit:` | Style or preference | No |
| `Question:` | Understanding, not a change request | No |
| `Suggestion:` | Could be better, author decides | No |

Approve explicitly when satisfied. Review within 2 business days of assignment.

---

## 10. Issue Workflow

```
[opened] → needs-triage → ready → in-progress → [PR merged] → closed
                        ↓
                   needs-design → RFC → ready
```

Maintainers triage within 3 business days: assign `type:`, `area:`, `priority:`,
and `status:` labels. Comment `I'd like to work on this` to claim; wait for assignment.

**Epics** (`type: epic`): contain a task list of child issues; are the parent reference
for all PRs in the chain; close only when all child issues close.

---

## 11. AI-Assisted Contributions

Zynax welcomes AI-assisted contributions. Quality standards are identical to
hand-written work — AI assistance is a productivity tool, not an exception.

1. **Human author fully responsible** — review AI output with the same rigour as your own.
2. **Declare AI assistance** — add the `ai-assisted` PR label.
3. **Use `Assisted-by:` trailer** — never `Co-Authored-By:` for AI (DCO is human-only):
   `Assisted-by: Claude/claude-sonnet-4-6`
4. **Same standards apply** — lint, BDD, layer boundaries, no AI exceptions.
5. **Trim verbosity** — remove AI over-commenting and padding before committing.
6. **No AI-generated secrets, credentials, or PII.**
7. **AI agents need a human sponsor** — the human is the PR author of record.
8. **No AI responses in discussions** — communicate directly with maintainers.

KB files (`CLAUDE.md`, all `AGENTS.md`, `docs/ai-assistant-setup.md`) are restricted
paths governed by ADR-018: maintainer approval required, `gitleaks-ai-context` CI gate
enforced. See [ADR-018](docs/adr/ADR-018-ai-kb-authorization-model.md) for the full policy.

---

## 12. Adding a New Service

1. Create the directory structure (see `services/AGENTS.md` for the template).
2. Copy `services/agent-registry/` as a template and adapt.
3. Add the service to `go.work` (Go workspace).
4. Add the service to the `SERVICES` variable in `Makefile`.
5. Create the Helm chart from the base template in `infra/helm/`.
6. Add the service to `infra/docker-compose/docker-compose.yml`.
7. Add an ADR in `docs/adr/` documenting the new service's purpose and boundaries.
8. Write `.feature` files for the first capability before any implementation.
9. Add to `.github/workflows/ci.yml` path filters.
10. Add to `.github/CODEOWNERS`.

---

## 13. Changing Proto Contracts

Proto contracts are the **public API** of every service. They are reviewed with
extra scrutiny and versioning discipline.

### Backward-Compatible Changes (allowed in minor versions)

- Adding new fields to messages (with `optional`)
- Adding new RPC methods
- Adding new enum values (never remove or renumber)
- Adding new message types

### Breaking Changes (require major version bump + migration guide)

- Removing or renaming fields
- Changing field numbers
- Changing field types
- Removing RPC methods
- Changing package name

### Process for Any Proto Change

1. Open an RFC in `docs/rfcs/` using `RFC-000-template.md`.
2. Get RFC approved (2 maintainers + affected service owners).
3. Implement the change on a branch.
4. `make lint-proto` — must pass with zero errors.
5. `buf breaking --against main` — breaking changes are caught here; justify each.
6. Add the label `PROTO REVIEWED` to your PR.
7. Update all contract tests (BDD scenarios for every RPC method).
8. Update `CHANGELOG.md` with migration notes.

---

## 14. First Contribution

If this is your first contribution to Zynax, welcome. Here is the fastest path to
your first merged PR:

1. **Pick a `good first issue`** — these are pre-triaged, scoped, and have clear
   acceptance criteria. Browse them [here](https://github.com/zynax-io/zynax/issues?q=is%3Aopen+label%3A%22good+first+issue%22).

2. **Comment on the issue** — say you'd like to work on it. Wait for assignment.

3. **Follow the workflow** — fork, branch, BDD file first, implement, clean commits.

4. **Ask questions in Discussions** — not issues, not DMs.

5. **Your first PR** — the maintainer who reviews it will be your guide for any
   gaps. First-time contributors get patient, detailed review.

### What Makes a Good First PR

- It is small (≤ 200 lines).
- It has a `.feature` file.
- The commit message body explains WHY.
- The PR description references the issue.
- All CI checks pass.

That is all. You do not need to know the whole codebase. You need to know your
change thoroughly.

---

## Questions?

Open a [GitHub Discussion](https://github.com/zynax-io/zynax/discussions) tagged
`question`. Do not use issues for questions — they are reserved for actionable work.
