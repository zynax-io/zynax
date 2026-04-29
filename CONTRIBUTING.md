<!-- SPDX-License-Identifier: Apache-2.0 -->

# Contributing to Zynax

Zynax is built by its community and held to the standards of best-in-class
CNCF projects. Read this document before opening your first PR.

---

## 1. Before You Start

1. **Read [`AGENTS.md`](AGENTS.md)** — the engineering constitution. Required for all
   contributors, human and AI alike.
2. **Read [`docs/git-workflow.md`](docs/git-workflow.md)** — the definitive reference
   for branching, commits, and PR decomposition.
3. **Check existing ADRs** in `docs/adr/` — your idea may already be decided.
4. **Open an issue first** for any non-trivial change (new capabilities, design
   decisions, contract changes, anything that touches more than one service).
5. **Sign the DCO** — every commit must have `Signed-off-by: Your Name <email>`.

---

## 2. Community & Communication

| Channel | Purpose |
|---------|---------|
| [GitHub Issues](https://github.com/zynax-io/zynax/issues) | Bug reports, feature requests, ADR proposals — actionable work items only |
| [GitHub Discussions](https://github.com/zynax-io/zynax/discussions) | Questions, ideas, design exploration |
| [GitHub PRs](https://github.com/zynax-io/zynax/pulls) | Code review, implementation discussion |

Do not use issues for questions. English is the working language. Response SLA:
2 business days for initial response on issues and PRs.

---

## 3. Development Environment

**Prerequisites:** Docker Desktop. Everything else runs inside Docker.

```bash
make bootstrap        # one-time setup
make lint             # proto + Go + Python lint
make test             # all tests (spec + unit + BDD)
make generate-protos  # regenerate Go + Python stubs
make validate-spec    # validate all YAML manifests
make security         # govulncheck + bandit + pip-audit
make dev-up           # start local stack (PostgreSQL, Redis, NATS, all services)
```

---

## 4. Engineering Standards

See [`AGENTS.md §Hard Constraints`](AGENTS.md#hard-constraints) and
[`AGENTS.md §Five Non-Negotiable Mandates`](AGENTS.md#five-non-negotiable-mandates)
for the complete and authoritative list. Per-layer standards live in
`services/AGENTS.md`, `agents/AGENTS.md`, and `protos/AGENTS.md`.

CI enforces all standards. PRs fail if any check is red.

---

## 5. Git Workflow & Commit Hygiene

> Full reference: [`docs/git-workflow.md`](docs/git-workflow.md)

### Branch naming

```
<type>/issue-<number>-<short-kebab-description>
```

`type` must be: `feat`, `fix`, `docs`, `test`, `refactor`, `ci`, `chore`

### Commit format (Conventional Commits — CI-enforced)

```
<type>(<scope>): <short description, imperative, ≤ 72 chars total>

<body: WHY this change is needed>

Closes #123
Signed-off-by: Your Name <your@email.com>
Assisted-by: Claude/claude-sonnet-4-6   ← AI-assisted only
```

**Rules:** imperative mood · no period · no `@mentions` in subject · no emojis ·
always rebase (`git rebase origin/main`), never merge main into feature branches ·
push with `--force-with-lease`, never bare `--force`.

### DCO and GPG signing

Every commit requires:
- **`Signed-off-by:`** — add with `git commit -s`. DCO bot blocks merge without it.
- **GPG signature** — set `git config commit.gpgsign true`. Branch protection rejects
  unsigned commits.

Both serve different purposes: GPG proves identity; DCO certifies contribution rights.

---

## 6. PR Size & Decomposition

| Lines changed | Status |
|--------------|--------|
| ≤ 200 | Ideal |
| 201–400 | Acceptable |
| 401–900 | Justify in PR description why it cannot be split |
| > 900 | **Blocked** — decompose before requesting review |

**Exclusions from count:** generated code (`*.pb.go`, `*_pb2.py`), lock files,
schema fixtures, CI workflow files.

Split large features as a **PR chain** — a sequence of small, mergeable PRs where
each one compiles, passes tests, and delivers observable value. Reference the parent
issue from every PR in the chain.

---

## 7. Layered Testing Strategy

ADR-016 defines four tiers. Apply the right tier — not every change needs a BDD
scenario.

| Tier | When to use | Tools |
|------|-------------|-------|
| **BDD** | Agent contracts, inter-service gRPC, E2E workflows | pytest-bdd, godog |
| **Unit / property** | Domain logic, routing, state transitions | pytest + hypothesis, testing + rapid |
| **Contract** | Any proto or schema change | `buf breaking`, `make validate-spec` |
| **Simulation** | Fault injection, retry storms (M3+) | testcontainers harness |

**BDD-first flow (required when BDD applies):**
Write `.feature` file → commit it alone → write step definitions → write domain code → pass.
Commit the `.feature` before any implementation. All scenarios must pass before the PR is opened.

See [`docs/patterns/bdd-contract-testing.md`](docs/patterns/bdd-contract-testing.md)
for templates, bufconn setup, and godog patterns.

---

## 8. Pull Request Process

1. **Open an issue first** for any non-trivial change.
2. Open a PR against `main` using the PR template.
3. PR title must follow Conventional Commits (CI-enforced), total ≤ 72 chars.
4. Fill every section of the PR template. Empty required sections block merge.
5. Mark as **Draft** while work is in progress.

**During review:** push fixup commits — do not amend or force-push while reviewers
are actively reading the diff.

**After approval:** squash and rebase onto `main`, then push with `--force-with-lease`.
The maintainer will squash-merge.

**Merge requirements:**
- [ ] All CI checks green: `dco · lint · test-unit · test-integration · security`
- [ ] Required approvals obtained (see `GOVERNANCE.md §2`)
- [ ] No unresolved review comments
- [ ] PR description complete

---

## 9. Code Review Etiquette

**Use severity prefixes on every comment:**

| Prefix | Blocks merge? |
|--------|--------------|
| `BLOCKER:` | Yes |
| `CONCERN:` | Usually yes |
| `Nit:` | No |
| `Question:` / `Suggestion:` | No |

**Authors:** respond to all comments (`Done` or `Disagree, because…`). Do not
resolve other people's comments. Do not use AI tools to respond to maintainer
feedback — engage directly.

**Reviewers:** approve explicitly when satisfied. Explain the WHY of blockers.
Review within 2 business days of assignment.

---

## 10. Issue Workflow

**Lifecycle:** `opened → needs-triage → ready → in-progress → [PR merged] → closed`

Maintainers triage within 3 business days, adding `type:`, `area:`, `priority:`,
and `milestone:` labels.

To claim an issue: comment `I'd like to work on this`. Wait for assignment before
opening a PR — parallel work creates waste.

**Epics** (`type: epic`): large features spanning multiple PRs. Contains a task list
of child issues. Closed only when all child issues are closed.

---

## 11. AI-Assisted Contributions

See [`CLAUDE.md §AI attribution`](CLAUDE.md) for the full policy on commit trailers,
model attribution, and SPDD Canvas requirements. Key rules:

1. **Human author is fully responsible** for every line, regardless of how it was
   generated. Review AI output with the same rigour as your own code.
2. **`Assisted-by:` trailer** — use this for AI attribution. `Co-Authored-By:` is
   reserved for humans certifying DCO. AI cannot certify DCO.
3. **Same quality standards** — `mypy --strict`, `golangci-lint`, BDD scenarios,
   layer boundaries. No exceptions for AI-generated code.
4. **No AI-generated secrets, credentials, or PII.**
5. **AI Knowledge Base Authorization** — all changes to `AGENTS.md`, `CLAUDE.md`,
   and `docs/ai-*` require maintainer approval (ADR-018, CODEOWNERS). These paths
   are restricted to prevent prompt injection.
6. **Dependency updates** — for Renovate PR CI failures, follow
   [`docs/engineering/renovate-fix-sop.md`](docs/engineering/renovate-fix-sop.md).

---

## 12. Adding a New Service

1. Create the directory structure (see `services/AGENTS.md` for the template).
2. Copy `services/agent-registry/` as a starting template and adapt.
3. Add the service to `go.work` and to `SERVICES` in `Makefile`.
4. Add the Helm chart from the base template in `infra/helm/`.
5. Add to `infra/docker/docker-compose.yml` and `.github/workflows/ci.yml`.
6. Write an ADR documenting the new service's purpose and boundaries.
7. Write `.feature` files for the first capability before any implementation.
8. Add to `.github/CODEOWNERS`.

---

## 13. Changing Proto Contracts

Proto contracts are the **public API** of every service — reviewed with extra
scrutiny.

**Backward-compatible (allowed in minor versions):** adding fields, RPCs, enum
values, or message types.

**Breaking (require major version bump + migration guide):** removing or renaming
fields/RPCs, changing field numbers or types, changing package name.

**Process for any proto change:**
1. Open an RFC in `docs/rfcs/`.
2. Get RFC approved (maintainers + affected service owners).
3. `make lint-proto` — must pass with zero errors.
4. `buf breaking --against main` — justify any breaking changes.
5. Add `PROTO REVIEWED` label to the PR.
6. Update all contract tests and `CHANGELOG.md`.

---

## 14. First Contribution

1. Pick a [`good first issue`](https://github.com/zynax-io/zynax/issues?q=is%3Aopen+label%3A%22good+first+issue%22).
2. Comment `I'd like to work on this`. Wait for assignment.
3. Follow the workflow: fork → branch → BDD file first → implement → clean commits.
4. Keep the PR ≤ 200 lines, include a `.feature` file, and ensure CI passes.
5. Ask questions in [Discussions](https://github.com/zynax-io/zynax/discussions), not issues.

---

## Questions?

Open a [GitHub Discussion](https://github.com/zynax-io/zynax/discussions) tagged
`question`. Issues are reserved for actionable work items.
