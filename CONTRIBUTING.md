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
| Go | ≥ 1.22 | [go.dev](https://go.dev) |
| Python | ≥ 3.12 | `uv python install 3.12` |
| uv | latest | `curl -LsSf https://astral.sh/uv/install.sh \| sh` |
| Docker | ≥ 24 | [docker.com](https://docker.com) |
| buf | 1.28+ | `brew install bufbuild/buf/buf` |
| make | any | pre-installed on macOS/Linux |

### Setup

```bash
git clone https://github.com/zynax-io/zynax.git
cd zynax
make bootstrap        # Install all tools + pre-commit hooks (includes commitlint + DCO)
make dev-up           # Start local stack: PostgreSQL, Redis, NATS, all services
make dev-ps           # Verify everything is healthy
```

### Running Tests

```bash
make test-unit             # Fast unit tests — run constantly
make test-integration      # Requires Docker — run before pushing
make test-unit-svc SVC=agent-registry   # Single service focus
```

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
6. After all approvals: rebase onto `main`, squash, `git push --force-with-lease`.

**Merge requirements:**
- [ ] All CI checks green (lint, test-unit, test-integration, security, dco)
- [ ] No unresolved review comments · PR description complete · CHANGELOG updated

**Merge strategy:** squash-and-merge for feature branches; rebase-merge for PR chains.
Solo maintainer phase: CI must pass; maintainer may self-merge non-breaking changes.
Breaking and proto changes require a 5-day RFC comment period (see `GOVERNANCE.md §2`).

---

## 9. Code Review Etiquette

**Authors:** respond to all comments; push fixup commits during review (no amend);
squash after approval; rebase `main` to keep branches current; no AI-generated replies.

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
6. Add the service to `infra/docker/docker-compose.yml`.
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
