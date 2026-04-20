<!-- SPDX-License-Identifier: Apache-2.0 -->

# GitHub Repository Setup Guide

> This is the step-by-step playbook for creating and configuring the
> `zynax-io/zynax` repository from scratch. Run through this once, in order.
> Each section is idempotent — safe to re-run if a step was missed.
>
> **Prerequisites:** `gh` CLI installed and authenticated as the org owner.
> Install: `brew install gh` then `gh auth login`.

---

## Table of Contents

1. [Create the GitHub Organisation](#1-create-the-github-organisation)
2. [Create the Repository](#2-create-the-repository)
3. [Push the Initial Codebase](#3-push-the-initial-codebase)
4. [Configure Branch Protection](#4-configure-branch-protection)
5. [Set Up GitHub Teams](#5-set-up-github-teams)
6. [Install Required GitHub Apps](#6-install-required-github-apps)
7. [Create Labels](#7-create-labels)
8. [Create GitHub Milestones](#8-create-github-milestones)
9. [Set Up GitHub Discussions](#9-set-up-github-discussions)
10. [Create the GitHub Project Board](#10-create-the-github-project-board)
11. [Configure Repository Settings](#11-configure-repository-settings)
12. [Set Up Secrets](#12-set-up-secrets)
13. [Verify Everything](#13-verify-everything)

---

## 1. Create the GitHub Organisation

1. Go to [github.com/organizations/new](https://github.com/organizations/new)
2. **Organisation account name:** `zynax-io`
3. **Contact email:** your email
4. **Plan:** Free (upgrade to Team later if private repos are needed)
5. Skip the "Invite members" step — add members later via `MAINTAINERS.md` process
6. Set the org profile:
   - **Display name:** Zynax
   - **Description:** Declarative control plane for AI agent workflows
   - **URL:** (leave blank until zynax.io DNS is set up)

---

## 2. Create the Repository

```bash
# Create the repo under the org
gh repo create zynax-io/zynax \
  --public \
  --description "Declarative, cloud-native, engine-agnostic control plane for AI agent workflows" \
  --license apache-2.0 \
  --gitignore Go \
  --clone

# Do NOT initialise with a README — you already have one
```

Or via the UI:
1. Go to [github.com/organizations/zynax-io/repositories/new](https://github.com/organizations/zynax-io/repositories/new)
2. Name: `zynax`
3. Visibility: **Public**
4. **Do not** initialise with README, .gitignore, or license (you have these already)
5. Create repository

### Repository Settings to Set Now

```bash
cd zynax   # The cloned repo directory

# Set homepage (update when zynax.io is live)
gh repo edit zynax-io/zynax \
  --homepage "https://github.com/zynax-io/zynax" \
  --topics "ai,workflow,control-plane,kubernetes,cncf,temporal,langgraph,go,cloud-native"
```

---

## 3. Push the Initial Codebase

```bash
# From your local zynax/ directory
git remote add origin https://github.com/zynax-io/zynax.git

# Verify your GPG key is configured
git config --global user.signingkey   # Must show a key ID
git config --global commit.gpgsign    # Must show "true"

# Create the initial commit (signed + DCO)
git add .
git commit -s -m "chore: initial repository bootstrap

Establishes the full project structure including:
- Three-layer architecture (Intent / Communication / Execution)
- All ADRs (001-015) documenting architectural decisions
- Proto contracts for all platform services
- CONTRIBUTING.md, GOVERNANCE.md, SECURITY.md, ROADMAP.md
- AGENTS.md engineering contracts (root + per-service + per-layer)
- CI pipeline, Docker Compose dev environment, Helm chart templates
- BDD feature files for core agent-registry scenarios
- GitHub community health files (issue templates, PR template, CODEOWNERS)

Signed-off-by: Your Name <your@email.com>"

git push -u origin main
```

---

## 4. Configure Branch Protection

Branch protection rules enforce:
- GPG-signed commits (every commit must be verified)
- DCO bot approval (every commit must have Signed-off-by)
- CI must pass (no merges on red CI)
- No force-push to `main`
- No deletion of `main`

### Via `gh` CLI

```bash
gh api repos/zynax-io/zynax/branches/main/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"contexts":["lint","test-unit","test-integration","security","dco"]}' \
  --field enforce_admins=false \
  --field required_pull_request_reviews='{"dismiss_stale_reviews":true,"require_code_owner_reviews":true,"required_approving_review_count":0}' \
  --field restrictions=null \
  --field required_linear_history=true \
  --field allow_force_pushes=false \
  --field allow_deletions=false \
  --field required_signatures=true
```

> **`required_approving_review_count: 0`** is correct for the solo maintainer
> phase — CI and DCO are the gates; no second human approval required yet.
> Change to `1` when a second maintainer joins.

> **`required_signatures: true`** enforces GPG signing at the GitHub level.

### Via UI (alternative)

1. **Settings → Branches → Add rule**
2. Branch name pattern: `main`
3. Check:
   - [x] Require a pull request before merging
     - [x] Dismiss stale pull request approvals when new commits are pushed
     - [x] Require review from Code Owners
     - Required approving reviews: **0** (solo phase; change to 1 later)
   - [x] Require status checks to pass before merging
     - [x] Require branches to be up to date before merging
     - Status checks: `lint`, `test-unit`, `test-integration`, `security`, `dco`
   - [x] Require conversation resolution before merging
   - [x] Require signed commits
   - [x] Require linear history
   - [x] Do not allow bypassing the above settings

### Protect `release/*` branches

```bash
gh api repos/zynax-io/zynax/branches \
  --method POST \
  -f pattern="release/*" \
  -f required_signatures=true \
  -f allow_force_pushes=false \
  -f allow_deletions=false
```

---

## 5. Set Up GitHub Teams

Teams control who can do what in the repository.

```bash
# Create teams in the org
gh api orgs/zynax-io/teams --method POST \
  -f name="maintainers" \
  -f description="Zynax core maintainers — merge access, release management" \
  -f privacy="closed"

gh api orgs/zynax-io/teams --method POST \
  -f name="reviewers" \
  -f description="Zynax reviewers — PR review access" \
  -f privacy="closed"

gh api orgs/zynax-io/teams --method POST \
  -f name="proto-owners" \
  -f description="Proto contract owners — required reviewers for proto changes" \
  -f privacy="closed"

# Add yourself to maintainers (replace YOUR_GITHUB_USERNAME)
gh api orgs/zynax-io/teams/maintainers/memberships/YOUR_GITHUB_USERNAME \
  --method PUT \
  -f role="maintainer"

# Give the maintainers team write access to the repo
gh api orgs/zynax-io/teams/maintainers/repos/zynax-io/zynax \
  --method PUT \
  -f permission="maintain"

gh api orgs/zynax-io/teams/reviewers/repos/zynax-io/zynax \
  --method PUT \
  -f permission="push"

gh api orgs/zynax-io/teams/proto-owners/repos/zynax-io/zynax \
  --method PUT \
  -f permission="push"
```

---

## 6. Install Required GitHub Apps

Install these from the GitHub Marketplace. Each links to its install page.

### Mandatory

| App | Purpose | Install |
|-----|---------|---------|
| **DCO** | Enforce `Signed-off-by:` on every commit | [probot/dco](https://github.com/apps/dco) |
| **commitlint** / conventional-commits-linter | Enforce commit message format | [conventional-commits-linter](https://github.com/apps/conventional-commits-linter) |

> If using GitHub Actions instead of apps, see `.github/workflows/ci.yml` —
> the DCO check and commitlint can run as Actions steps without installing apps.

### Recommended

| App | Purpose |
|-----|---------|
| **Stale** (GitHub Actions stale.yml) | Auto-label + close stale issues |
| **Dependabot** | Weekly dependency updates (configure in `.github/dependabot.yml`) |

### Dependabot Setup

Create `.github/dependabot.yml`:

```yaml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    labels:
      - "type: chore"
      - "area: ci"

  - package-ecosystem: "pip"
    directory: "/agents"
    schedule:
      interval: "weekly"
    labels:
      - "type: chore"
      - "area: agents/sdk"

  - package-ecosystem: "docker"
    directory: "/infra/docker"
    schedule:
      interval: "weekly"
    labels:
      - "type: chore"
      - "area: infra"

  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
    labels:
      - "type: chore"
      - "area: ci"
```

### Stale Issues Setup

Create `.github/workflows/stale.yml`:

```yaml
name: Mark stale issues and PRs

on:
  schedule:
    - cron: '0 8 * * 1'   # Every Monday at 08:00 UTC

permissions:
  issues: write
  pull-requests: write

jobs:
  stale:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/stale@v9
        with:
          stale-issue-message: >
            This issue has had no activity for 90 days. It will be closed in
            14 days unless there is new activity or a maintainer removes the
            `status: stale` label.
          close-issue-message: >
            Closed due to inactivity. Reopen with updated context if still
            relevant.
          stale-pr-message: >
            This PR has had no activity for 30 days. It will be closed in
            14 days unless there is new activity.
          close-pr-message: >
            Closed due to inactivity. Reopen if you resume work on it.
          days-before-issue-stale: 90
          days-before-issue-close: 14
          days-before-pr-stale: 30
          days-before-pr-close: 14
          stale-issue-label: 'status: stale'
          stale-pr-label: 'status: stale'
          exempt-issue-labels: 'priority: critical,priority: high,good first issue,help wanted'
          exempt-pr-labels: 'do not merge'
```

---

## 7. Create Labels

Delete GitHub's default labels first, then create the full Zynax taxonomy from
`docs/labels.md`.

```bash
# Delete all default labels
gh label list --json name -q '.[].name' | \
  xargs -I{} gh label delete "{}" --yes

# Create all labels — paste each block or run as a script
# See docs/labels.md for the complete list.

# --- type: ---
gh label create "type: bug"         --color "d73a4a" --description "Something is broken — deviates from .feature file"
gh label create "type: feature"     --color "0075ca" --description "New capability not yet in any .feature file"
gh label create "type: enhancement" --color "a2eeef" --description "Improvement to existing capability"
gh label create "type: refactor"    --color "e4e669" --description "Code change with no behaviour change"
gh label create "type: docs"        --color "0075ca" --description "Documentation only"
gh label create "type: test"        --color "e4e669" --description "Test coverage — no production code change"
gh label create "type: ci"          --color "e4e669" --description "CI/CD pipeline changes"
gh label create "type: chore"       --color "e4e669" --description "Maintenance: deps, tooling, cleanup"
gh label create "type: security"    --color "b60205" --description "Security fix or hardening"
gh label create "type: performance" --color "0052cc" --description "Performance improvement"
gh label create "type: epic"        --color "3e4b9e" --description "Parent issue tracking a multi-PR feature"
gh label create "type: adr-proposal" --color "8b5cf6" --description "Proposed Architectural Decision Record"

# --- area: ---
gh label create "area: agent-registry"    --color "bfd4f2" --description "Agent identity + capability registry"
gh label create "area: task-broker"       --color "bfd4f2" --description "Capability routing + task dispatch"
gh label create "area: memory-service"    --color "bfd4f2" --description "Shared KV + vector memory"
gh label create "area: event-bus"         --color "bfd4f2" --description "NATS JetStream event backbone"
gh label create "area: api-gateway"       --color "bfd4f2" --description "REST + gRPC gateway"
gh label create "area: workflow-compiler" --color "bfd4f2" --description "YAML → IR compiler"
gh label create "area: engine-adapter"    --color "bfd4f2" --description "Temporal / LangGraph / Argo adapters"
gh label create "area: protos"            --color "d1ecf1" --description "gRPC contract definitions"
gh label create "area: agents/adapters"   --color "bfd4f2" --description "Python execution adapters"
gh label create "area: agents/sdk"        --color "bfd4f2" --description "Python SDK"
gh label create "area: spec"              --color "d1ecf1" --description "YAML schemas + example manifests"
gh label create "area: infra"             --color "f9d0c4" --description "Docker, Helm, Kubernetes"
gh label create "area: ci"                --color "f9d0c4" --description "CI/CD workflows"
gh label create "area: docs"              --color "f9d0c4" --description "Documentation"
gh label create "area: cli"               --color "bfd4f2" --description "zynax CLI tool"

# --- priority: ---
gh label create "priority: critical" --color "b60205" --description "Security/data loss. Fix before next patch."
gh label create "priority: high"     --color "d93f0b" --description "Blocks core workflow. Target next milestone."
gh label create "priority: medium"   --color "e4e669" --description "Important, workaround exists."
gh label create "priority: low"      --color "cfd3d7" --description "Nice to have. Backlog."

# --- status: ---
gh label create "status: needs-triage"  --color "e4e669" --description "New issue, not yet reviewed by maintainer"
gh label create "status: needs-design"  --color "8b5cf6" --description "Requires RFC or ADR before implementation"
gh label create "status: ready"         --color "0e8a16" --description "Triaged, ready to pick up"
gh label create "status: in-progress"   --color "0075ca" --description "Assigned and being actively worked on"
gh label create "status: blocked"       --color "d73a4a" --description "Cannot proceed — waiting on dependency"
gh label create "status: in-review"     --color "bfd4f2" --description "PR open, under review"
gh label create "status: stale"         --color "cfd3d7" --description "No activity for 90 days"

# --- milestone: ---
gh label create "milestone: M1" --color "f9d0c4" --description "Contracts Foundation"
gh label create "milestone: M2" --color "f9d0c4" --description "Workflow IR"
gh label create "milestone: M3" --color "f9d0c4" --description "Temporal Execution"
gh label create "milestone: M4" --color "f9d0c4" --description "YAML System + CLI"
gh label create "milestone: M5" --color "f9d0c4" --description "Adapter Library"
gh label create "milestone: M6" --color "f9d0c4" --description "K8s Production-Ready"
gh label create "milestone: M7" --color "f9d0c4" --description "Full Observability"
gh label create "milestone: M8" --color "f9d0c4" --description "CNCF Sandbox Submission"
gh label create "milestone: unscheduled" --color "cfd3d7" --description "Accepted but not yet assigned to milestone"

# --- process ---
gh label create "good first issue"    --color "7057ff" --description "Beginner-friendly. Clear scope, defined acceptance criteria."
gh label create "help wanted"         --color "008672" --description "Maintainers want community contribution"
gh label create "breaking change"     --color "b60205" --description "Requires major version bump. RFC required."
gh label create "needs-rfc"           --color "8b5cf6" --description "RFC must be accepted before implementation"
gh label create "PROTO REVIEWED"      --color "0e8a16" --description "Proto change reviewed by proto-owners"
gh label create "ai-assisted"         --color "d4c5f9" --description "AI tools used in generating this change"
gh label create "ai-reviewed"         --color "d4c5f9" --description "AI tools used to assist in reviewing"
gh label create "split-not-possible"  --color "d93f0b" --description "PR >400 lines; maintainer approved exception"
gh label create "do not merge"        --color "b60205" --description "Blocked from merge — see comments"
gh label create "duplicate"           --color "cfd3d7" --description "Duplicate of another issue"
gh label create "wontfix"             --color "ffffff" --description "Explicitly out of scope"
gh label create "invalid"             --color "e4e669" --description "Not a valid issue for this project"
```

---

## 8. Create GitHub Milestones

Milestones map to roadmap milestones. Create them all upfront so issues can be
assigned from day one.

```bash
gh api repos/zynax-io/zynax/milestones --method POST \
  -f title="Contracts Foundation (M1)" \
  -f description="All communication contracts defined. gRPC protos + AsyncAPI + capability schema." \
  -f state="open"

gh api repos/zynax-io/zynax/milestones --method POST \
  -f title="Workflow IR (M2)" \
  -f description="YAML manifests compile to canonical engine-agnostic IR." \
  -f state="open"

gh api repos/zynax-io/zynax/milestones --method POST \
  -f title="Temporal Execution (M3)" \
  -f description="Workflow IR executes on Temporal. Engine abstraction proven." \
  -f state="open"

gh api repos/zynax-io/zynax/milestones --method POST \
  -f title="YAML System + CLI (M4)" \
  -f description="Users can zynax apply workflow.yaml and see it run." \
  -f state="open"

gh api repos/zynax-io/zynax/milestones --method POST \
  -f title="Adapter Library (M5)" \
  -f description="Existing systems become capabilities without SDK adoption." \
  -f state="open"

gh api repos/zynax-io/zynax/milestones --method POST \
  -f title="K8s Production-Ready (M6)" \
  -f description="Production deployment on Kubernetes. Argo engine support." \
  -f state="open"

gh api repos/zynax-io/zynax/milestones --method POST \
  -f title="Full Observability (M7)" \
  -f description="End-to-end observability across all workflow execution layers." \
  -f state="open"

gh api repos/zynax-io/zynax/milestones --method POST \
  -f title="CNCF Sandbox (M8)" \
  -f description="Community, governance, and technical maturity for CNCF Sandbox." \
  -f state="open"
```

---

## 9. Set Up GitHub Discussions

Enable Discussions and create the categories.

### Enable Discussions

**Settings → General → Features → Discussions: [✓] Enable**

Or:
```bash
gh api repos/zynax-io/zynax --method PATCH -f has_discussions=true
```

### Create Discussion Categories

Do this in the GitHub UI (Discussions → pencil icon → Manage categories):

| Category | Format | Description |
|----------|--------|-------------|
| **Announcements** | Announcement | Release notes, project updates. Maintainers only can create. |
| **Q&A** | Question/Answer | Questions about using Zynax. Community help. |
| **Ideas** | Open-ended | Early-stage ideas before they become issues. |
| **Technical Design** | Open-ended | Architecture discussions, design alternatives, RFC pre-drafts. |
| **Roadmap** | Open-ended | Proposals for roadmap changes, milestone prioritisation. |

Delete the default categories (General, Show and tell, Polls) that you don't need.

---

## 10. Create the GitHub Project Board

The board tracks execution of the roadmap. Create one project for the organisation.

### Create the Project

```bash
# Create the org-level project
gh project create --owner zynax-io --title "Zynax Platform Roadmap" --format table
```

Or via UI:
1. Go to `github.com/orgs/zynax-io/projects`
2. **New project** → **Board** (start with board, add views later)
3. Name: **Zynax Platform Roadmap**
4. Visibility: **Public**

### Link the Repository

```bash
# Get the project number from the output of the create command, or from the URL
gh project link --owner zynax-io <PROJECT_NUMBER> --repo zynax-io/zynax
```

### Add Custom Fields

In the project settings (project → **...** → Settings → Custom fields):

| Field | Type | Values |
|-------|------|--------|
| **Status** | Single select | `Backlog`, `Ready`, `In Progress`, `In Review`, `Done` |
| **Milestone** | Single select | `M1`, `M2`, `M3`, `M4`, `M5`, `M6`, `M7`, `M8`, `Unscheduled` |
| **Priority** | Single select | `Critical`, `High`, `Medium`, `Low` |
| **Area** | Single select | (list of services/layers from `docs/labels.md`) |
| **PR Size** | Text | Free text — filled from PR size self-check |

### Configure the Three Views

**View 1: Kanban (default)**
- Group by: `Status`
- Columns: `Backlog | Ready | In Progress | In Review | Done`
- Filter: `is:open`

**View 2: Milestone Table**
- Layout: Table
- Group by: `Milestone`
- Sort by: `Priority` (descending)
- Columns: Title, Status, Priority, Area, Assignee

**View 3: Roadmap Timeline**
- Layout: Roadmap
- Group by: `Milestone`
- Date field: Target iteration (set milestone target dates when known)

### Automation

In project settings → **Workflows**, enable:
- **Item added to project** → set Status to `Backlog`
- **Item closed** → set Status to `Done`
- **Pull request merged** → set Status to `Done`

---

## 11. Configure Repository Settings

```bash
# Merge strategy: squash only (enforces one-commit-per-PR on main)
gh api repos/zynax-io/zynax --method PATCH \
  -f allow_squash_merge=true \
  -f allow_merge_commit=false \
  -f allow_rebase_merge=true \
  -f squash_merge_commit_title="PR_TITLE" \
  -f squash_merge_commit_message="PR_BODY" \
  -f delete_branch_on_merge=true \
  -f has_wiki=false \
  -f has_projects=false
```

> `allow_rebase_merge=true` is kept for stacked PR chains (explicitly declared
> in the PR description). `allow_merge_commit=false` prevents sloppy merges.
> `delete_branch_on_merge=true` keeps the branch list clean.

---

## 12. Set Up Secrets

Add secrets for CI via the GitHub UI or CLI:

```bash
# Container registry (GitHub Container Registry — no extra secret needed for GHCR)
# For DockerHub:
gh secret set DOCKERHUB_USERNAME --body "your-username"
gh secret set DOCKERHUB_TOKEN --body "your-token"

# cosign key pair for image signing (generate once)
cosign generate-key-pair
gh secret set COSIGN_PRIVATE_KEY < cosign.key
gh secret set COSIGN_PASSWORD --body "your-key-passphrase"
# Add cosign.pub to the repository (not a secret — public verification key)

# For future use: CNCF / cloud provider credentials
# Add when CI targets GKE / EKS / AKS in M6+
```

---

## 13. Verify Everything

Run this checklist after completing all steps:

```bash
# 1. Verify the repo exists and is public
gh repo view zynax-io/zynax

# 2. Verify branch protection
gh api repos/zynax-io/zynax/branches/main/protection | jq '{
  signed_commits: .required_signatures.enabled,
  linear_history: .required_linear_history.enabled,
  status_checks: .required_status_checks.contexts,
  no_force_push: (.allow_force_pushes.enabled | not)
}'

# 3. Verify labels (count should be 40+)
gh label list | wc -l

# 4. Verify milestones (should be 8)
gh api repos/zynax-io/zynax/milestones | jq 'length'

# 5. Verify merge settings (squash + rebase only, no merge commits)
gh api repos/zynax-io/zynax | jq '{
  squash: .allow_squash_merge,
  rebase: .allow_rebase_merge,
  merge_commit: .allow_merge_commit,
  delete_on_merge: .delete_branch_on_merge
}'

# 6. Test a signed commit reaches the repo
git commit --allow-empty -s -m "chore: verify GPG signing and DCO on push"
git push
# Check the commit on GitHub — it should show "Verified" badge
git push origin --delete HEAD  # Clean up the test commit's remote ref
git reset --hard HEAD~1        # Remove local test commit
```

### Manual Checks (UI)

- [ ] Discussions are enabled with 5 categories
- [ ] Project board has 3 views: Kanban, Milestone Table, Roadmap Timeline
- [ ] Issue templates: Bug Report, Feature Request, ADR Proposal, Documentation — no blank issues
- [ ] `CODEOWNERS` file is active (Settings → Code and automation → Code owners → green)
- [ ] DCO bot is installed and active on the repo
- [ ] `conduct@zynax.io` and `security@zynax.io` are set up and forwarding to your inbox

---

## What to Do After Setup

1. **Create the Milestone 1 issues** — one issue per checklist item in `ROADMAP.md §M1`.
   Use the Feature Request template, set `milestone: M1` and the appropriate `area:` label.
2. **Add issues to the Project board** — they will auto-enter `Backlog`.
3. **Pick up the first `good first issue`** — move it to `Ready`, assign yourself,
   start with the `.feature` file.
4. **Update `MAINTAINERS.md`** — add yourself as the initial maintainer.

---

## Ongoing Maintenance

| Task | Frequency | How |
|------|-----------|-----|
| Triage new issues | Every 3 business days | Apply labels, set milestone, set status |
| Merge Dependabot PRs | Weekly | Review + merge if CI is green |
| Check stale issues | Weekly | Review `status: stale` items |
| Update Project board | As issues progress | Move cards between Kanban columns |
| Cut a release | When milestone is complete | Follow `GOVERNANCE.md §5` release checklist |
