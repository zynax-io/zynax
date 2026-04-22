<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax Governance

> Zynax follows CNCF governance best practices and is working toward CNCF Sandbox
> submission. Governance is neutral — no single company controls decisions.

---

## Table of Contents

1. [Roles & Responsibilities](#1-roles--responsibilities)
2. [Decision-Making](#2-decision-making)
3. [DCO and Commit Hygiene](#3-dco-and-commit-hygiene)
4. [RFC Process](#4-rfc-process)
5. [Issue Triage Process](#5-issue-triage-process)
6. [Release Process](#6-release-process)
7. [Roadmap Management](#7-roadmap-management)
8. [AI Agent Contributors](#8-ai-agent-contributors)
9. [Conflict Resolution](#9-conflict-resolution)
10. [Adding and Removing Maintainers](#10-adding-and-removing-maintainers)
11. [Communication Channels](#11-communication-channels)
12. [CNCF Alignment](#12-cncf-alignment)

---

## 1. Roles & Responsibilities

### Contributor

**How to become one:** Submit a PR that is merged, or a substantive issue/discussion.

**Rights:**
- Open issues, comment on issues
- Open PRs and participate in discussions
- Vote in Discussions (non-binding)

**Responsibilities:**
- Follow the [Code of Conduct](CODE_OF_CONDUCT.md)
- Follow engineering standards in `AGENTS.md` and `CONTRIBUTING.md`
- Sign the DCO on every commit

---

### Reviewer

**How to become one:** Nominated by a maintainer after demonstrating consistent
quality contributions (typically 5+ merged PRs and active PR review participation).

**Rights:**
- All Contributor rights
- Approve PRs (counts toward the 1-approval requirement for `fix`, `docs`, `test`)
- Be listed in `CODEOWNERS` for specific areas of the codebase

**Responsibilities:**
- Review assigned PRs within 2 business days
- Provide constructive, actionable feedback using the prefix convention
  (`BLOCKER:`, `Nit:`, `Question:`, etc. — see `CONTRIBUTING.md §9`)
- Actively monitor the `area:` they own

---

### Maintainer

**How to become one:** Nominated by an existing maintainer and approved by
supermajority (>2/3) of current maintainers. Requires:
- Track record as Reviewer
- Deep understanding of the architecture (can pass a code review of core services)
- Represented organisation is not over-represented (CNCF diversity requirement)
- Time commitment: ≥ 4 hours/week

**Rights:**
- All Reviewer rights
- Merge PRs (after required approvals)
- Manage labels, milestones, and the GitHub Project board
- Triage issues
- Cut releases and manage release branches
- Represent the project in CNCF contexts

**Responsibilities:**
- Triage new issues within 3 business days
- Drive the roadmap forward
- Maintain the health and quality bar of the codebase
- Participate in governance decisions

**Current maintainers:** Listed in [MAINTAINERS.md](MAINTAINERS.md).

---

### Emeritus Maintainer

A maintainer who steps down retains their history and recognition in
`MAINTAINERS.md` under "Emeritus". No active responsibilities or rights.
Re-activation follows the normal Maintainer nomination process.

---

## 2. Decision-Making

Zynax uses **lazy consensus** for most decisions: a proposal is accepted if no
maintainer objects within the defined period. Explicit votes are called when
lazy consensus fails or the decision type requires it.

### Solo Maintainer Phase (current)

Until there are ≥ 2 active maintainers from ≥ 2 organisations, the project
operates in **solo maintainer mode**:

- **CI must pass** for every merge, no exceptions.
- **Non-breaking changes** (bug fixes, docs, tests, minor features): the solo
  maintainer may self-merge after all CI checks are green.
- **Breaking changes and new proto contracts**: RFC required + 5-business-day
  public comment period on the RFC PR before merge. The solo maintainer merges
  after the comment period closes with no unresolved objections.
- **Governance changes**: 10-business-day comment period on the PR.

This policy exists to ensure the community can participate even when no second
maintainer exists to formally approve.

### Multi-Maintainer Phase (once ≥ 2 active maintainers)

| Decision type | Process | Minimum period |
|--------------|---------|---------------|
| Bug fix, docs, minor feature | 1 maintainer approval + CI green | N/A |
| Significant feature or new service | RFC + 2 maintainer approvals | 5 business days |
| New proto contract or breaking change | RFC + 2 maintainer approvals + `PROTO REVIEWED` | 5 business days |
| Architectural decision (new ADR) | RFC + 2 maintainer approvals | 5 business days |
| Roadmap change (milestone goals) | Maintainer discussion + lazy consensus | 3 business days |
| New Maintainer or Reviewer | Nomination + supermajority vote | 5 business days |
| Governance change | Supermajority vote | 10 business days |
| CNCF-related decisions | All active maintainers must vote | 10 business days |

### Supermajority

A supermajority requires votes from strictly more than 2/3 of active maintainers.
Abstentions do not count for or against. A maintainer absent for more than 30 days
is considered inactive for voting purposes.

---

## 3. DCO and Commit Hygiene

### DCO Sign-Off

Every commit must include a `Signed-off-by` trailer certifying the
[Developer Certificate of Origin](https://developercertificate.org/):

```
Signed-off-by: Your Full Name <your@email.com>
```

Add automatically with `git commit -s`. The DCO bot enforced in CI blocks merges
without it. `Signed-off-by` is reserved for humans — AI tools cannot certify DCO
and must use `Assisted-by:` instead (see §7).

Fixing a missing sign-off:

```bash
# Single commit
git commit --amend -s --no-edit

# Multiple commits (rebase and re-sign all since diverging from main)
git rebase --signoff main
git push --force-with-lease
```

### Commit Message Hygiene

- **Subject line:** conventional commit format, ≤ 72 characters
- **Valid types:** `feat` `fix` `refactor` `docs` `test` `ci` `chore`
- **Breaking changes:** include `BREAKING CHANGE:` footer
- **No `@mentions`** in commit messages — they generate GitHub notifications
- **No emoji** in commit messages
- **AI attribution:** `Assisted-by: Claude/<model-id>` — never `Co-Authored-By:`

PR titles follow the same format and are validated by CI.

---

## 4. RFC Process

An RFC (Request for Comments) is required for any decision classified as
"significant" or higher in §2.

### When an RFC Is Required

- New service or new layer
- Changes to proto contracts (any change)
- Changes to the three-layer architecture
- New adapter type or engine integration
- Changes to `AGENTS.md` (engineering contract)
- Changes to the governance document itself

### RFC Lifecycle

```
Draft → Under Review → Accepted / Rejected / Withdrawn
```

1. Copy `docs/rfcs/RFC-000-template.md` to `docs/rfcs/RFC-<NNN>-<short-title>.md`
   (RFC number assigned by a maintainer in the PR review).
2. Open a PR with just the RFC document. Title: `rfc: <title>`.
3. Minimum **5 business day** comment period. Maintainers and community comment on
   the PR.
4. The RFC author addresses comments and updates the document.
5. A maintainer merges the RFC with status `Accepted` or `Rejected`.
6. Implementation begins only after `Accepted` is merged.

---

## 5. Issue Triage Process

Maintainers triage new issues within **3 business days** of opening.

### Triage Steps (in order)

1. **Validate** — is this a real issue or a question (redirect to Discussions)?
2. **Categorise** — add `type:` label (bug, feature, enhancement, docs, etc.)
3. **Scope** — add `area:` label (which service or layer)
4. **Prioritise** — add `priority:` label (critical, high, medium, low)
5. **Status** — add `status: needs-design` (if RFC required) or `status: ready`
6. **Milestone** — assign to the appropriate milestone if known
7. **Mark** — remove `status: needs-triage`

### Priority Definitions

| Priority | Definition | Target |
|----------|-----------|--------|
| `priority: critical` | Security vulnerability or data loss | Fix before next patch release |
| `priority: high` | Blocking users from core workflows | Target next milestone |
| `priority: medium` | Important but has workaround | Scheduled in roadmap |
| `priority: low` | Nice to have | Backlog |

### Closing Stale Issues

Issues with no activity for **90 days** are marked `status: stale` by the
stale-bot. If no response within 14 days, they are closed with the note:
"Closed as stale. Reopen with updated context if still relevant."

---

## 6. Release Process

Releases follow [Semantic Versioning 2.0](https://semver.org). Release managers
rotate among maintainers.

### Milestone → Version Mapping

| Milestone | Version | Description |
|-----------|---------|-------------|
| M1 — Contracts Foundation | v0.1.0 | gRPC contracts, AsyncAPI spec, generated stubs |
| M2 — Workflow IR | v0.2.0 | WorkflowIR proto + compiler service skeleton |
| M3 — Temporal Execution | v0.3.0 | Temporal-backed engine adapter |
| M4 — YAML System + CLI | v0.4.0 | `zynaxctl` CLI + YAML manifest validation |
| M5 — Observability | v0.5.0 | OTel traces, metrics dashboards, audit log |
| M6 — Production Hardening | v1.0.0-rc.1 | HA, multi-tenancy, security audit |
| M7 — Developer Experience | v1.0.0-rc.2 | SDKs, docs site, quickstart |
| M8 — CNCF Sandbox | v1.0.0 | Sandbox submission, governance complete |

A version tag is cut when **all** milestone acceptance criteria are closed and CI
is green on `main`. Patch versions (v0.x.y) may be cut between milestones for
critical bug fixes without waiting for the next minor release.

### Release Cadence

- **Patch releases** (v0.x.y): as needed for critical bug fixes
- **Minor releases** (v0.x.0): when a roadmap milestone is complete
- **Major release** (v1.0.0): CNCF Sandbox submission milestone

### Release Checklist

A release is managed as a GitHub Issue with this task list:

```markdown
## Release v0.2.0 (Milestone 3: Temporal execution)

- [ ] All milestone issues closed or explicitly deferred
- [ ] `CHANGELOG.md` updated: `make changelog`
- [ ] All CI checks green on `main`
- [ ] Release branch cut: `git checkout -b release/v0.2.0`
- [ ] Release notes drafted (from CHANGELOG.md)
- [ ] Security scan clean: `make security`
- [ ] SBOM generated: `make sbom`
- [ ] Images built and signed: `make release-images`
- [ ] Helm chart version bumped in `infra/helm/Chart.yaml`
- [ ] Tag created and signed: `git tag -s v0.2.0`
- [ ] Tag pushed: `git push origin v0.2.0`
- [ ] GitHub Release created from tag
- [ ] Announcement posted in Discussions
```

---

## 7. Roadmap Management

The [ROADMAP.md](ROADMAP.md) is the narrative roadmap. The authoritative
execution roadmap lives in the [GitHub Project board](https://github.com/orgs/zynax-io/projects/1).

### GitHub Projects Structure

The board has three views:
- **Kanban** (current sprint): `Backlog | Ready | In Progress | In Review | Done`
- **Milestone Table**: all issues grouped by milestone, with status
- **Roadmap Timeline**: milestone dates as swimlanes

### Roadmap Change Process

Roadmap changes (adding, removing, or reprioritising milestone items) require:
1. A maintainer proposes the change in a GitHub Discussion tagged `roadmap`.
2. Lazy consensus: 3 business days for other maintainers to object.
3. The proposing maintainer updates `ROADMAP.md` and the Project board.

Community members may propose roadmap additions via a `type: feature` issue.
The issue will be evaluated at the next triage cycle.

---

## 8. AI Agent Contributors

Zynax explicitly welcomes AI-assisted contributions and provides a defined framework
for AI agents (Claude, Copilot, GPT-4, custom agents) to participate responsibly.

### Human Sponsorship Requirement

Every PR — regardless of how it was generated — requires a **human sponsor**:
- The human is the PR author of record on GitHub.
- The human is accountable for the correctness, security, and quality of the change.
- The human signs the DCO (`Signed-off-by:`).
- AI tools are attributed via `Assisted-by:` trailer — never `Co-Authored-By:` or
  `Signed-off-by:`, which are reserved for humans certifying the DCO.

There is no "autonomous AI PR" in Zynax. Every change has a human who reviewed
and is accountable for it.

### AI Contribution Labels

| Label | Meaning |
|-------|---------|
| `ai-assisted` | AI tools were used in generating code, docs, or tests |
| `ai-reviewed` | AI was used to review the PR (in addition to human review) |

Both labels are informational. They do not change the quality bar or approval
requirements.

### What AI Tools May Do

- Generate code, tests, documentation drafts
- Propose commit messages and PR descriptions
- Review PRs and flag potential issues
- Generate BDD `.feature` file drafts (human must validate)
- Generate proto definitions (human must validate against contract rules)

### What AI Tools May Not Do

- Approve or merge PRs (GitHub access not granted to bots)
- Self-assign issues or claim work autonomously
- Push to branches without human review of the diff
- Modify governance, `AGENTS.md`, or `CONTRIBUTING.md` without RFC

### Claude Code Specific

Contributors using [Claude Code](https://claude.ai/code) (the Anthropic CLI):
- Claude Code appends `Co-Authored-By: Claude Sonnet 4.x` automatically — **remove it**.
  Replace with `Assisted-by: Claude Code/claude-sonnet-4-6` in the commit footer.
- Add the `ai-assisted` label to the PR.
- The human contributor must review all changes before pushing.
- Do not use Claude Code to generate responses to maintainer comments in PR threads
  or GitHub Discussions. Engage with maintainers directly and personally.

---

## 9. Conflict Resolution

### Technical Disagreements

1. Discuss in the PR or issue comment thread. Keep it focused on technical merit.
2. If no resolution after 3 business days, escalate to a dedicated GitHub Discussion
   tagged `technical-decision`.
3. Maintainers attempt lazy consensus.
4. If still unresolved, a formal vote is called (majority of active maintainers).
5. The decision is documented in an ADR.

### Code of Conduct Violations

Report to conduct@zynax.io (or the email in `CODE_OF_CONDUCT.md`). Maintainers
who are the subject of a report must recuse themselves from that discussion.

Enforcement levels: warning → temporary ban → permanent ban.

---

## 10. Adding and Removing Maintainers

### Adding a Maintainer

1. Any maintainer nominates a contributor in a PR to `MAINTAINERS.md`.
2. The PR describes the nominee's contributions and qualifications.
3. Minimum 5 business day comment period.
4. Supermajority vote (>2/3 of current active maintainers approving).
5. PR merged → nominee invited to the `zynax-io/maintainers` GitHub team.

### Removing a Maintainer (Voluntary)

A maintainer stepping down:
1. Opens a PR moving themselves from "Active" to "Emeritus" in `MAINTAINERS.md`.
2. PR merged by any other maintainer.

### Removing a Maintainer (Involuntary)

Triggered by sustained inactivity (>6 months) or Code of Conduct violation:
1. Any maintainer opens a PR to move the inactive maintainer to "Emeritus".
2. The affected maintainer is notified via PR and email.
3. Supermajority vote required for Code of Conduct removal.
4. Inactivity removal requires lazy consensus (2 business days).

---

## 11. Communication Channels

| Channel | Purpose | Moderated by |
|---------|---------|-------------|
| [GitHub Issues](https://github.com/zynax-io/zynax/issues) | Bug reports, features, ADRs | Maintainers |
| [GitHub Discussions](https://github.com/zynax-io/zynax/discussions) | Questions, ideas, proposals | Maintainers |
| [GitHub PRs](https://github.com/zynax-io/zynax/pulls) | Code review | Reviewers + Maintainers |
| conduct@zynax.io | Code of conduct reports | All maintainers |
| security@zynax.io | Security vulnerability reports | All maintainers |

All project-relevant technical communication must happen in GitHub to maintain
a public, searchable record. Private channels (email, DM) are for conduct and
security only.

---

## 12. CNCF Alignment

Zynax targets CNCF Sandbox submission at v1.0.0 (Milestone 8).

CNCF requirements we are tracking:
- [ ] Adopt CNCF Code of Conduct (done — see `CODE_OF_CONDUCT.md`)
- [ ] Neutral governance (no single company controls) — documented here
- [ ] ≥ 2 maintainers from ≥ 2 different organisations
- [ ] Apache 2.0 license (done)
- [ ] DCO sign-off on all commits (enforced by CI)
- [ ] External security audit (target: v0.5.0)
- [ ] SBOM per release
- [ ] cosign-signed container images
- [ ] Documented roadmap with versioning

---

*This document is governed by itself: changes require RFC + supermajority vote.*
