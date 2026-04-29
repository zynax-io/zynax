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

Zynax uses a four-tier testing pyramid (ADR-016). Apply the right tier for the
scope of the code you are writing — not every change needs a BDD scenario.

### Which tier to use

| Tier | When to use | Tools |
|------|-------------|-------|
| **BDD** | Agent contracts, inter-service gRPC, E2E workflows | pytest-bdd, godog |
| **Unit / property-based** | Domain logic, routing, state transitions, message handling | pytest + hypothesis, testing + rapid |
| **Contract** | Any proto or schema change | `buf breaking`, `make validate-spec` |
| **Simulation** | Fault injection, retry storms, topology changes | testcontainers harness (coming) |

### Tier 1: BDD — system boundaries only

Use BDD when the behaviour you are defining is observable at a service boundary:
a gRPC contract, a capability execution path, or a full workflow end-to-end.

**Do NOT use BDD for** internal domain logic, scheduling algorithms, networking
internals, or performance characteristics. Use unit or property-based tests instead.

**BDD-first flow (required when BDD applies):**

```
1. Write .feature file  →  2. Commit it  →  3. Write step definitions  →  4. Write domain code  →  5. Pass
```

Commit the `.feature` file alone, before any implementation:

```gherkin
# services/agent-registry/tests/features/capability_discovery.feature
Feature: Capability-based agent discovery
  As a task-broker
  I want to find agents by capability
  So that I can route tasks without knowing agent identities

  Scenario: Find agents that support a capability
    Given two agents are registered with capability "summarize"
    And one agent is registered with capability "search"
    When I query for agents with capability "summarize"
    Then I receive exactly two agents
    And none of the agents have only "search" capability
```

```bash
git commit -s -m "test(agent-registry): add BDD scenarios for capability discovery

Defines expected behaviour for capability-based agent lookup.
No implementation yet — scenarios will fail until domain code is added.

Closes #123"
```

Only after the feature file is committed: write step definitions, then domain code.
All scenarios must pass before the PR is opened.

### Tier 2: Unit and property-based tests — domain logic

Domain logic (routing algorithms, state transitions, message handlers) is tested
with standard unit tests and property-based tests. No `.feature` file is required.

Property tests express invariants that hold across the entire input space:

```python
# Python — hypothesis
from hypothesis import given, strategies as st

@given(st.lists(st.builds(Agent), min_size=1))
def test_task_always_assigned(agents):
    result = route_task(agents, task_id="t1")
    assert result is not None
```

```go
// Go — rapid
func TestRoutingAlwaysAssigns(t *testing.T) {
    rapid.Check(t, func(tc *rapid.T) {
        agents := rapid.SliceOfN(genAgent(), 1, 10).Draw(tc, "agents").([]*Agent)
        result, err := RouteTask(agents, "task-1")
        require.NoError(t, err)
        require.NotNil(t, result)
    })
}
```

### Tier 3: Contract tests — enforced by CI

Every PR that touches `.proto` files must pass `buf breaking --against main`.
Every PR that modifies YAML schemas must pass `make validate-spec`.
These run automatically — no action needed beyond keeping them green.

### Tier 4: Simulation tests — distributed faults

For scenarios that require multiple agents, injected failures, or message
drops, use the simulation harness in `tests/simulation/` (introduced in a
follow-up PR). Write simulation tests for: retry storms, agent timeouts,
topology changes, eventual consistency violations.

### BDD as an AI prompting tool

When using Claude or another AI assistant, write the `.feature` file first and
ask the model to generate code that satisfies the scenarios. This forces precise
behaviour definition before generation and dramatically improves output quality:

```
Feature: Task distribution

  Scenario: Assign to least-loaded agent
    Given 3 available agents with loads 2, 5, 1
    When a task is submitted
    Then it is assigned to the agent with load 1

  Scenario: Reassign on agent failure
    Given an agent is assigned a task
    When the agent becomes unresponsive for 5 seconds
    Then the task is reassigned to another agent
```

Then: "Generate production Go code that satisfies these scenarios, including tests."

---

## 8. Pull Request Process

### Opening a PR

1. **Open an issue first** for any non-trivial change. Discuss before coding. Do not
   open a PR for a feature or fix that has no associated issue.
2. **Fork** the repository and push your branch.
3. Open a PR against `main` using the PR template.
4. The PR title must follow Conventional Commits (enforced by CI):
   ```
   feat(agent-registry): Add capability-based agent discovery
   ```
5. Fill in every section of the PR template. Empty sections block merge.
6. Mark as **Draft** if the work is in progress or you want early feedback.
7. Convert to **Ready for Review** only when all CI checks pass locally.

### During Review and After Approval

While the PR is open and under review:
- Push **fixup commits** for each round of feedback — do not amend or rebase while
  reviewers are actively reading the diff.
- Do not force-push during review. Reviewers lose their place in the diff.

After all approvals are given and blocking comments are resolved:
1. Squash and rebase onto the latest `main`:
   ```bash
   git fetch origin
   git rebase -i origin/main --autosquash
   ```
2. Push with `--force-with-lease` (never bare `--force`):
   ```bash
   git push --force-with-lease
   ```
3. The maintainer will squash-merge.

### Review Requirements

**Solo maintainer phase (current):** CI must pass. The maintainer may self-merge
for non-breaking changes. Breaking changes and proto changes require a 5-day RFC
comment period before merge. See `GOVERNANCE.md §2`.

**Multi-maintainer phase (once ≥ 2 maintainers exist):**

| Change type | Required approvals |
|-------------|-------------------|
| `docs` only | 1 maintainer or reviewer |
| `test` only | 1 maintainer or reviewer |
| `fix`, `refactor`, `ci`, `chore` | 1 maintainer |
| `feat` | 1 maintainer (scales to 2 as team grows) |
| Proto change | 1 maintainer + `PROTO REVIEWED` label |
| Breaking change | 1 maintainer + RFC accepted |
| Governance / AGENTS.md | 5-day comment period + maintainer approval |

### Merge Requirements

All of the following must be true before merge:
- [ ] All CI checks green (lint, test-unit, test-integration, security)
- [ ] DCO bot: all commits signed off
- [ ] Required approvals obtained (see table above)
- [ ] No unresolved review comments
- [ ] PR description complete (no empty required sections)
- [ ] CHANGELOG.md entry added (for user-visible changes)

### Merge Strategy

Zynax uses **squash-and-merge** for feature branches. The squash commit message is
the PR title (which must be a valid Conventional Commit). Individual commits within
the PR are for the reviewer's benefit; the main branch history shows one commit per
PR.

**Exception:** PR chains (stacked PRs) use **rebase-merge** to preserve the
individual meaningful commits. State this in the PR description.

---

## 9. Code Review Etiquette

Good code review is a skill. These rules apply to both authors and reviewers.

### For Authors

- **Respond to all comments** — even if only `Done` or `Disagree, because...`
- **Do not resolve other people's comments** — the commenter resolves their own.
- **Push fixup commits** during review — do not amend or force-push while reviewers
  are actively reading. Amending mid-review loses the diff context for the reviewer.
  ```bash
  git commit -s --fixup HEAD~1
  ```
- **Squash after approval**, not before. See §8 "During Review and After Approval".
- **Keep PRs current** — rebase onto `main` if the branch becomes stale
  (`git rebase origin/main`). Never merge `main` into your branch.
- **Be receptive** — review comments improve the code; they are not criticism of you.
- **Do not use AI tools to respond to maintainer comments** — engage directly and personally.

### For Reviewers

**Use explicit severity prefixes on every comment:**

| Prefix | Meaning | Blocks merge? |
|--------|---------|--------------|
| `BLOCKER:` | Must be fixed before merge | Yes |
| `CONCERN:` | Architecturally significant, needs discussion | Usually yes |
| `Nit:` | Style or preference, take it or leave it | No |
| `Question:` | I want to understand, not requesting a change | No |
| `Suggestion:` | Could be better, author decides | No |

**Good reviewer behaviour:**
- Approve explicitly when satisfied. Do not leave PRs in limbo.
- Distinguish blocking from non-blocking before submitting your review.
- Explain the WHY of blockers, not just "change this".
- Review within **2 business days** of being assigned.
- If you cannot review in time, say so and suggest another reviewer.

**What to check:**
1. Does the PR description explain WHY, not just WHAT?
2. Is the `.feature` file written before the implementation?
3. Do the commit messages follow the conventions?
4. Are the layer boundaries respected (domain → no imports from api/infra)?
5. Is the PR size justified if over 400 lines?
6. Are new behaviours observable (structured logs, metrics, traces)?

---

## 10. Issue Workflow

### Issue Lifecycle

```
[opened] → needs-triage → ready → in-progress → [PR merged] → closed
                        ↓
                   needs-design → RFC → ready
```

### Triage (maintainers)

Maintainers triage new issues within **3 business days**. Triage means:
- Add `type:` label
- Add `area:` label
- Add `priority:` label
- Add `milestone:` if applicable
- Remove `status: needs-triage`
- Add `status: ready` or `status: needs-design`

### Claiming an Issue

1. Comment `I'd like to work on this` on the issue.
2. A maintainer will assign it to you and add `status: in-progress`.
3. Do not open a PR without being assigned — parallel work creates waste.
4. If you cannot continue, comment and unassign yourself so others can pick it up.

### Issue Quality Bar

Issues are the permanent record of why changes were made. Write them for
a reader who has no context — including yourself in 6 months.

A good issue has:
- **Problem statement** — what is broken or missing, with evidence
- **Expected behaviour** — what should happen instead
- **Acceptance criteria** — how will we know it is done? (Gherkin welcome)
- **Context** — relevant ADRs, related issues, affected services

### Epics

Large features that span multiple PRs are tracked as **Epic issues**. An epic:
- Has the `type: epic` label
- Contains a task list of child issues: `- [ ] #201 feat: capability indexing`
- Is the parent reference for all PRs in the chain
- Is closed only when all child issues are closed

---

## 11. AI-Assisted Contributions

Zynax welcomes contributions where AI tools (Claude, Copilot, GPT-4, etc.) assist
with drafting code, documentation, or tests. AI assistance is a productivity tool,
not a shortcut past quality standards.

### Rules for AI-Assisted Work

1. **The human author is fully responsible** for every line in the PR, regardless
   of how it was generated. Review AI output with the same rigor as your own.

2. **Declare AI assistance** — add the `ai-assisted` label to the PR and include
   the tool and model in the PR description:
   ```
   AI assistance: Claude Code / claude-sonnet-4-6 (code generation, documentation drafts)
   ```

3. **Attribution via `Assisted-by:` trailer** — use the git trailer for AI attribution.
   `Co-Authored-By:` and `Signed-off-by:` are reserved for humans only; adding an AI
   tool there misrepresents the DCO, which only a human can certify.
   ```
   git commit -s -m "feat: my change" --trailer "Assisted-by: Claude Code/claude-sonnet-4-6"
   ```
   Or add it manually to the commit message footer:
   ```
   Assisted-by: Claude Code/claude-sonnet-4-6
   ```

4. **AI-generated code is held to the same standards** as human-written code.
   `mypy --strict`, `golangci-lint`, BDD scenarios, layer boundaries — no exceptions.

5. **Trim AI verbosity** — AI tools often over-comment, over-document, and pad
   descriptions. Remove all of it before committing. Commit messages, comments,
   and PR descriptions must meet the same conciseness standard as hand-written text.

6. **No AI-generated secrets, credentials, or personally identifiable data**
   — AI models can hallucinate plausible-looking but incorrect or sensitive values.

7. **AI agents acting as contributors** must have a human sponsor who is accountable
   for the work. The human sponsor is the PR author of record. See `GOVERNANCE.md §8`.

8. **Human engagement in discussions** — do not use AI tools to generate responses
   to maintainer feedback or in issue/PR threads. Communicate directly and personally.

### For Claude Code Users

If you are using Claude Code (the Anthropic CLI) to contribute:
- Remove any `Co-Authored-By: Claude ...` lines Claude Code appends automatically —
  they violate the DCO convention. Use `Assisted-by:` instead (see rule 3 above).
- Add the `ai-assisted` label to your PR.
- Review every file changed before pushing. Pay particular attention to comments and
  docstrings — Claude tends to be more verbose than this project's standards require.

### AI Knowledge Base Authorization Policy

The files that AI assistants auto-load (`CLAUDE.md`, all `AGENTS.md` files,
`docs/ai-assistant-setup.md`, and the future `.ai/` and `.claude/` directories)
are **restricted paths** governed by ADR-018.

**Why this matters:** these files are published to a public repository. Merged
content cannot be reliably unpublished. Content that looks like documentation
can act as a prompt injection payload that silently shifts AI assistant behavior
for every contributor who clones the repo.

**Rules for KB path changes:**

1. **Maintainer approval required** — `@zynax-io/maintainers` must review and
   approve all changes to KB paths. Branch protection enforces this via
   CODEOWNERS. You cannot self-approve a KB change.
2. **Secret and PII scan must pass** — the `gitleaks-ai-context` CI gate scans
   KB file content on every PR. Red gate = no merge.
3. **No prompt-injection payloads** — KB content must be neutral engineering
   documentation. Avoid instruction-like phrasing ("always respond with…",
   "ignore previous instructions"). Reviewers check for this explicitly.
4. **Content must match reviewed source material** — KB entries should be
   derived from merged code, ADRs, or documented decisions. Do not add
   speculative or aspirational content.

See [docs/adr/ADR-018-ai-kb-authorization-model.md](docs/adr/ADR-018-ai-kb-authorization-model.md)
for the full rationale and threat model. See
[docs/knowledge-base-policy.md](docs/knowledge-base-policy.md) for the
content sanitization rules, scanner reference, and step-by-step reviewer
verification process.

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
