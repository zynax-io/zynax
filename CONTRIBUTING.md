<!-- SPDX-License-Identifier: Apache-2.0 -->

# Contributing to Zynax

Thank you for your interest in contributing. Zynax is built by its community and held
to the standards of the best-in-class CNCF projects. This guide is the single source
of truth for how work flows from idea to merged code.

**Read this document completely before opening your first PR.** Every rule here has
a reason, and knowing the reason makes it easier to apply in edge cases.

---

## Table of Contents

1. [Before You Start](#1-before-you-start)
2. [Community & Communication](#2-community--communication)
3. [Development Environment](#3-development-environment)
4. [Engineering Standards](#4-engineering-standards)
5. [Git Workflow & Commit Hygiene](#5-git-workflow--commit-hygiene)
6. [PR Size & Decomposition](#6-pr-size--decomposition)
7. [Layered Testing Strategy](#7-layered-testing-strategy)
8. [Pull Request Process](#8-pull-request-process)
9. [Code Review Etiquette](#9-code-review-etiquette)
10. [Issue Workflow](#10-issue-workflow)
11. [AI-Assisted Contributions](#11-ai-assisted-contributions)
12. [Adding a New Service](#12-adding-a-new-service)
13. [Changing Proto Contracts](#13-changing-proto-contracts)
14. [First Contribution](#14-first-contribution)

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

These are enforced in CI. PRs fail if any check is red.

### Both Go and Python

- Functions ≤ 30 lines (Go) / ≤ 20 lines (Python). One responsibility.
- No magic numbers — use named constants.
- No dead code. If it is not tested, it does not exist.
- Comments explain **WHY**, never WHAT. The code explains what.
- No secrets in code, configs, or committed files.

### Go services

- `golangci-lint` must pass with zero suppressions (or justified suppressions).
- `domain/` layer: zero imports from `api/` or `infrastructure/`.
- All errors wrapped: `fmt.Errorf("context: %w", err)`.
- No `panic` in production code paths.
- Structured logging via `slog` with context propagation.
- Never expose credentials, tokens, or auth URLs in logs or output.
- Sanitize file paths to prevent traversal attacks before any I/O.
- TLS verification must remain enabled by default — never disable it for convenience.
- Avoid shell execution (`exec.Command`) for git, kubectl, helm, or similar operations; use Go libraries instead.
- Close HTTP response bodies, file handles, and archive readers via `defer` on every code path.
- Delete temporary files on all code paths — success and error alike.
- Machine-readable output (tables, YAML, JSON) → `stdout`; human-readable status messages → `stderr`.

### Python agents/adapters

- `mypy --strict` must pass. No untyped `Any` without justification.
- `ruff` must pass with zero suppressions.
- No `print()` — use `structlog` for all logging.
- No bare `except:` — catch specific exception types.
- `domain/` coverage ≥ 90%.

See `AGENTS.md §4` and `services/AGENTS.md` for complete standards.

---

## 5. Git Workflow & Commit Hygiene

> The git log is public documentation. A clean history is a gift to every future
> contributor. A messy history is technical debt that compounds over years.

See [`docs/git-workflow.md`](docs/git-workflow.md) for the full reference. The rules
that matter most are here.

### Branch Strategy

Zynax uses **trunk-based development**: short-lived feature branches off `main`.

```
main
 ├── feat/ISSUE-123-capability-based-discovery    ← your branch
 ├── fix/ISSUE-456-retry-storm-on-unavailability
 ├── docs/ISSUE-789-update-adapter-guide
 └── release/v0.2.0                               ← release cut by maintainer
```

**Branch naming:**
```
<type>/ISSUE-<number>-<short-kebab-description>
```
- `type` must be one of: `feat`, `fix`, `docs`, `test`, `refactor`, `ci`, `chore`
- `ISSUE-<number>` links to the GitHub issue (required for `feat` and `fix`)
- `short-kebab-description` ≤ 5 words, lowercase

**Branch lifetime:** Feature branches must not live longer than **7 days** without
merging or converting to a Draft PR. Stale branches are deleted after 30 days.

### Commit Atomicity — The Core Rule

> One commit = one logical change that leaves the codebase in a working state.

**Good:** Each commit compiles, passes tests, and makes one coherent change.
**Bad:** "WIP", "fix", "more changes", "address review comments" (as final commits).

Before opening a PR, clean your history with interactive rebase:
```bash
git rebase -i main    # Squash WIP commits, write proper messages
```

The PR reviewer sees your cleaned commit history, not your working state.

### Commit Message Format

[Conventional Commits](https://conventionalcommits.org) — enforced by `commitlint`:

```
<type>(<scope>): <short description in imperative mood>
<blank line>
<body: WHY this change is needed. What problem it solves. What was considered.>
<blank line>
<footer: Closes #123, BREAKING CHANGE, Signed-off-by, Assisted-by>
```

**Subject line rules:**
- Full header (`type(scope): description`) ≤ 72 characters
- Imperative mood — "Add support for X", not "Added" or "Adding"
- Capitalized first word after the colon
- No period at the end
- No `@mentions` anywhere in the commit message — GitHub references belong in the footer only (`Closes #123`)
- No emojis

**Types:** `feat`, `fix`, `docs`, `test`, `chore`, `refactor`, `perf`, `ci`, `build`

**Scopes:** `agent-registry`, `task-broker`, `memory-service`, `api-gateway`,
`event-bus`, `workflow-compiler`, `engine-adapter`, `protos`, `agents`, `infra`,
`ci`, `docs`, `spec`

**Good commit messages:**
```
feat(agent-registry): Add capability-based agent discovery

Agents currently register by name. The task-broker cannot route to the
best available agent for a given capability. This adds capability indexing
to the registry so task-broker can query "who can handle summarize?" rather
than needing to know agent names.

Closes #123
Signed-off-by: Jane Doe <jane@example.com>
```

```
fix(task-broker): Prevent retry storm when all agents unavailable

When no agents can handle a capability, task-broker was retrying
immediately on each assignment attempt, causing CPU spikes. Now uses
exponential backoff (1s, 2s, 4s, max 30s) with jitter.

Closes #456
Signed-off-by: Jane Doe <jane@example.com>
```

**Bad commit messages:**
```
fix bug                      ← no scope, no description, no context
WIP task broker changes      ← WIP commits must not reach the PR
address review comments.     ← period at end; does not describe what changed
fixed the @jane issue        ← @mentions not allowed in commit message
✨ add new feature           ← no emojis
```

**Breaking changes** — append `!` and add footer:
```
feat(protos)!: Rename AgentConfig to AgentSpec

BREAKING CHANGE: All consumers of zynax.v1.AgentConfig must update field
references to AgentSpec. No field renaming — only the message name changes.
Migration: docs/migrations/v0.2-to-v0.3.md

Closes #789
Signed-off-by: Jane Doe <jane@example.com>
```

### Keeping Branches Current

**Never merge `main` into your feature branch.** Always rebase:

```bash
git fetch origin
git rebase origin/main
```

Merging `main` into a feature branch pollutes the commit history with merge commits
and makes the eventual squash harder to reason about. Rebase keeps your branch
linearly on top of the latest main.

When you need to push after a rebase, use `--force-with-lease`, never bare `--force`:

```bash
git push --force-with-lease
```

`--force-with-lease` refuses the push if someone else has pushed to the same branch
since your last fetch, protecting against accidentally overwriting another person's work.
`--force` has no such protection.

### During Review

While a PR is under review:
- **Push fixup commits** for each round of feedback — do not amend or force-push.
  ```bash
  git commit -s --fixup HEAD~1
  ```
- **Squash after approval**, not before. Interactive rebase once all reviewers
  have approved and there are no open blocking comments:
  ```bash
  git rebase -i origin/main --autosquash
  git push --force-with-lease
  ```
- **Do not resolve other people's review comments** — the commenter resolves their own.

### GPG Commit Signing (Required)

Every commit must be GPG-signed. This is enforced by GitHub branch protection.

```bash
# Generate a key (if you don't have one)
gpg --full-generate-key

# Get your key ID
gpg --list-secret-keys --keyid-format=long

# Tell git to use it
git config --global user.signingkey <YOUR_KEY_ID>
git config --global commit.gpgsign true

# Sign commits automatically from now on (no -S flag needed)
git commit -m "feat: ..."
```

Add your public key to your GitHub account:
**GitHub → Settings → SSH and GPG keys → New GPG key**

```bash
# Export your public key
gpg --armor --export <YOUR_KEY_ID>
```

Unsigned commits will be **rejected at push time** by branch protection.
The `Verified` badge on GitHub confirms the commit is signed.

### DCO Sign-Off (Required)

Every commit must also include:
```
Signed-off-by: Your Full Name <your@email.com>
```

Add automatically with `git commit -s`. The DCO bot blocks merges without it.
Note: `commit.gpgsign true` handles signing; `-s` handles the DCO footer.
Both are required. They serve different purposes:
- **GPG signature** — cryptographically proves the commit came from you
- **DCO sign-off** — legally certifies you have the right to contribute

This certifies you have the right to submit the contribution under the project's
Apache 2.0 license (see [developercertificate.org](https://developercertificate.org)).

---

## 6. PR Size & Decomposition

> A PR is a unit of review. Reviewers must hold it in their head entirely.
> The larger the PR, the lower the review quality — every study confirms this.

### The Principle Behind the Size Rule

> Every merged PR must leave the codebase in a working state **and** deliver
> observable functional value — something that can be run, tested, or demonstrated.

A 50-line PR that adds a struct with no wiring, no test, and no behaviour is
worse than a 600-line PR that delivers a complete, testable capability end-to-end.
Size is a proxy for reviewability, not a goal in itself.

Split your work to maximise *functional value per PR*, not to minimise line count.

### Size Targets

| Lines changed | Status | Condition |
|--------------|--------|-----------|
| ≤ 200 | Ideal | Always preferred when the PR delivers complete value at this size |
| 201–400 | Acceptable | Explain in PR description why the extra lines are necessary |
| 401–800 | Justified extension | The only justification: splitting would produce a PR with no functional value. State this explicitly. |
| > 800 | Blocked | Decompose before requesting review. If genuinely impossible, get maintainer approval before starting. |

**Exclusions from line count:** generated code (`*.pb.go`, `*_pb2.py`), lock files,
schema fixtures, migration files.

### The Functional Value Test

Before opening a PR, ask: "If this PR were the only one that merged today,
would a user or the test suite be able to observe something new?"

- A new gRPC method with a BDD scenario that passes ✅ — observable
- A new domain struct with no service or test wiring ❌ — not yet observable
- A refactor that keeps all existing tests passing ✅ — observable (no regression)
- Proto definition with no generated code ❌ — nothing works yet

### How to Decompose

Large features ship as a **PR chain** — a sequence of small, mergeable PRs:

```
Issue #123: Add capability-based agent discovery
  │
  ├── PR #201  feat(protos): add capability fields to AgentSpec         [~80 lines]
  ├── PR #202  feat(agent-registry): index capabilities on registration  [~150 lines]
  ├── PR #203  feat(agent-registry): add capability query RPC            [~120 lines]
  ├── PR #204  feat(task-broker): route tasks by capability              [~200 lines]
  └── PR #205  test: BDD scenarios for end-to-end capability routing     [~180 lines]
```

Each PR:
- Merges cleanly to `main` on its own
- Leaves the codebase in a working state
- Has a test (even if minimal) that proves it works
- References the parent issue

### Stacked PRs

For a PR chain where each PR depends on the previous one, open them as stacked
(base each on the previous branch, not on `main`). When the foundation PR merges,
rebase the next one onto `main`.

```bash
git checkout -b feat/ISSUE-123-protos           # PR #201
# ... work ...
git checkout -b feat/ISSUE-123-registry         # PR #202, based on previous
git checkout feat/ISSUE-123-registry
git rebase feat/ISSUE-123-protos
```

Use the `Stacked on #201` line in the PR description to make the chain visible.

### When You Cannot Split

Sometimes a change is genuinely indivisible (e.g., an atomic schema migration +
the code that uses it). In that case:
1. Explain in the PR description why it cannot be split.
2. Add the label `split-not-possible` with a justification comment.
3. A maintainer must approve the exception before review begins.

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
   for the work. The human sponsor is the PR author of record. See `GOVERNANCE.md §7`.

8. **Human engagement in discussions** — do not use AI tools to generate responses
   to maintainer feedback or in issue/PR threads. Communicate directly and personally.

### For Claude Code Users

If you are using Claude Code (the Anthropic CLI) to contribute:
- Remove any `Co-Authored-By: Claude ...` lines Claude Code appends automatically —
  they violate the DCO convention. Use `Assisted-by:` instead (see rule 3 above).
- Add the `ai-assisted` label to your PR.
- Review every file changed before pushing. Pay particular attention to comments and
  docstrings — Claude tends to be more verbose than this project's standards require.

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
