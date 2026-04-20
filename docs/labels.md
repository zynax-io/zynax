<!-- SPDX-License-Identifier: Apache-2.0 -->

# GitHub Label Taxonomy

> Labels are the primary way issues and PRs are organised, filtered, and
> prioritised in Zynax. This document defines every label, its colour, and
> when to use it.
>
> Labels are managed via the GitHub UI or `gh label` CLI.
> **Do not create ad-hoc labels** — add them to this file and get it approved first.

---

## Label Groups

Labels use a `group: value` naming convention so they sort and filter predictably.

---

## `type:` — What Kind of Work

| Label | Colour | Description |
|-------|--------|-------------|
| `type: bug` | `#d73a4a` (red) | Something is broken — deviates from `.feature` file |
| `type: feature` | `#0075ca` (blue) | New capability not yet in any `.feature` file |
| `type: enhancement` | `#a2eeef` (teal) | Improvement to existing capability |
| `type: refactor` | `#e4e669` (yellow) | Code change with no behaviour change |
| `type: docs` | `#0075ca` (blue) | Documentation only |
| `type: test` | `#e4e669` (yellow) | Test coverage — no production code change |
| `type: ci` | `#e4e669` (yellow) | CI/CD pipeline changes |
| `type: chore` | `#e4e669` (yellow) | Maintenance: deps, tooling, cleanup |
| `type: security` | `#b60205` (dark red) | Security fix or security hardening |
| `type: performance` | `#0052cc` (dark blue) | Performance improvement |
| `type: epic` | `#3e4b9e` (indigo) | Parent issue tracking a multi-PR feature |
| `type: adr-proposal` | `#8b5cf6` (purple) | Proposed Architectural Decision Record |

---

## `area:` — Which Part of the Codebase

| Label | Colour | Description |
|-------|--------|-------------|
| `area: agent-registry` | `#bfd4f2` (light blue) | Agent identity + capability registry service |
| `area: task-broker` | `#bfd4f2` (light blue) | Capability routing + task dispatch service |
| `area: memory-service` | `#bfd4f2` (light blue) | Shared KV + vector memory service |
| `area: event-bus` | `#bfd4f2` (light blue) | NATS JetStream event backbone |
| `area: api-gateway` | `#bfd4f2` (light blue) | REST + gRPC gateway |
| `area: workflow-compiler` | `#bfd4f2` (light blue) | YAML → IR compiler |
| `area: engine-adapter` | `#bfd4f2` (light blue) | Temporal / LangGraph / Argo adapters |
| `area: protos` | `#d1ecf1` (cyan) | gRPC contract definitions |
| `area: agents/adapters` | `#bfd4f2` (light blue) | Python execution adapters |
| `area: agents/sdk` | `#bfd4f2` (light blue) | Python SDK |
| `area: spec` | `#d1ecf1` (cyan) | YAML schemas + example manifests |
| `area: infra` | `#f9d0c4` (salmon) | Docker, Helm, Kubernetes infra |
| `area: ci` | `#f9d0c4` (salmon) | CI/CD workflows |
| `area: docs` | `#f9d0c4` (salmon) | Documentation |
| `area: cli` | `#bfd4f2` (light blue) | `zynax` CLI tool |

---

## `priority:` — How Urgent

| Label | Colour | Description |
|-------|--------|-------------|
| `priority: critical` | `#b60205` (dark red) | Security vulnerability or data loss. Fix before next patch. |
| `priority: high` | `#d93f0b` (orange-red) | Blocks core workflow. Target next milestone. |
| `priority: medium` | `#e4e669` (yellow) | Important, workaround exists. Scheduled. |
| `priority: low` | `#cfd3d7` (grey) | Nice to have. Backlog. |

---

## `status:` — Where in the Lifecycle

| Label | Colour | Description |
|-------|--------|-------------|
| `status: needs-triage` | `#e4e669` (yellow) | New issue, not yet reviewed by maintainer |
| `status: needs-design` | `#8b5cf6` (purple) | Requires RFC or ADR before implementation |
| `status: ready` | `#0e8a16` (green) | Triaged, acceptance criteria clear, ready to pick up |
| `status: in-progress` | `#0075ca` (blue) | Assigned and being actively worked on |
| `status: blocked` | `#d73a4a` (red) | Cannot proceed — waiting on dependency |
| `status: in-review` | `#bfd4f2` (light blue) | PR open, under review |
| `status: stale` | `#cfd3d7` (grey) | No activity for 90 days — will be closed |

---

## `milestone:` — Which Roadmap Milestone

| Label | Colour | Description |
|-------|--------|-------------|
| `milestone: M1` | `#f9d0c4` (salmon) | Contracts Foundation |
| `milestone: M2` | `#f9d0c4` (salmon) | Workflow IR |
| `milestone: M3` | `#f9d0c4` (salmon) | Temporal Execution |
| `milestone: M4` | `#f9d0c4` (salmon) | YAML System + CLI |
| `milestone: M5` | `#f9d0c4` (salmon) | Adapter Library |
| `milestone: M6` | `#f9d0c4` (salmon) | K8s Production-Ready |
| `milestone: M7` | `#f9d0c4` (salmon) | Full Observability |
| `milestone: M8` | `#f9d0c4` (salmon) | CNCF Sandbox Submission |
| `milestone: unscheduled` | `#cfd3d7` (grey) | Accepted but not yet assigned to a milestone |

---

## Process Labels

| Label | Colour | Description |
|-------|--------|-------------|
| `good first issue` | `#7057ff` (violet) | Beginner-friendly. Clear scope, well-defined acceptance criteria. |
| `help wanted` | `#008672` (teal) | Maintainers want community contribution |
| `breaking change` | `#b60205` (dark red) | Requires major version bump. RFC required. |
| `needs-rfc` | `#8b5cf6` (purple) | RFC must be accepted before implementation begins |
| `PROTO REVIEWED` | `#0e8a16` (green) | Proto change has been reviewed by proto-owners |
| `ai-assisted` | `#d4c5f9` (lavender) | AI tools (Claude, Copilot, etc.) used in generating the change |
| `ai-reviewed` | `#d4c5f9` (lavender) | AI tools used to assist in reviewing the PR |
| `split-not-possible` | `#d93f0b` (orange) | PR > 400 lines; maintainer has approved exception |
| `do not merge` | `#b60205` (dark red) | Blocked from merge — see comments for reason |
| `duplicate` | `#cfd3d7` (grey) | Duplicate of another issue |
| `wontfix` | `#ffffff` (white) | Explicitly out of scope — closed with explanation |
| `invalid` | `#e4e669` (yellow) | Not a valid issue for this project |

---

## Applying Labels

### Maintainers apply:
- All `status:` labels (triage step)
- `priority:` labels
- `milestone:` labels
- `good first issue` and `help wanted`
- `PROTO REVIEWED`
- `breaking change`, `needs-rfc`

### Contributors apply:
- `ai-assisted` or `ai-reviewed` (self-declare on their own PRs)

### Automation (stale-bot) applies:
- `status: stale`

---

## Creating New Labels

To add a label:
1. Open a `type: docs` issue proposing the new label (name, colour, description).
2. Get 1 maintainer approval.
3. Add it to this document in the correct group.
4. Create it on GitHub: `gh label create "type: foo" --color "#hexcode" --description "..."`

Do not create labels directly on GitHub without updating this file — it becomes
the source of truth.

---

## Setting Up Labels via CLI

To bootstrap all labels in a new repository from this file:

```bash
# Example: create a single label
gh label create "type: bug" \
  --color "d73a4a" \
  --description "Something is broken — deviates from .feature file"

# For bulk creation, use the labels.json export + gh label import
gh label list --json name,color,description > .github/labels.json
gh label import .github/labels.json
```

A `make labels` target is planned for Milestone 4 to automate this.
