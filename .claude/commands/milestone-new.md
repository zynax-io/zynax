---
description: Scaffold the next milestone — GitHub milestone, planning doc skeleton, state/milestone.yaml active block. The only sanctioned writer of milestone.yaml besides /milestone-close.
argument-hint: "<name> <version> \"<title>\"  (next milestone per ROADMAP.md)"
---

# /milestone-new — Milestone Lifecycle: Open

Scaffold the next milestone. This command (with `/milestone-close`) is the ONLY sanctioned
writer of `state/milestone.yaml` — every other command reads it.

Inputs (from arguments; ask the operator if missing): short name (e.g. the next sequential
code), semver version target, human title. Cross-check against `ROADMAP.md` — the roadmap
defines the milestone sequence; this command instantiates it, never invents it.

---

## STEP 0 — Preconditions

```bash
git fetch origin --prune && git checkout main && git pull --rebase origin main
[ -z "$(git status --porcelain)" ] || { echo "dirty tree — STOP"; exit 1; }

# The previous milestone must be rotated to history (active block empty/placeholder).
# If /milestone-close has not run, STOP and run it first.
grep -A2 '^active:' state/milestone.yaml
```

## STEP 1 — GitHub milestone

```bash
NEW_NAME="$1"; NEW_VERSION="$2"; NEW_TITLE="$3"
GH_TITLE="${NEW_TITLE} (${NEW_NAME})"

# Reuse if it already exists (idempotent), else create
NUMBER=$(gh api "repos/{owner}/{repo}/milestones?state=all" \
  --jq ".[] | select(.title == \"$GH_TITLE\") | .number")
if [ -z "$NUMBER" ]; then
  NUMBER=$(gh api -X POST "repos/{owner}/{repo}/milestones" \
    -f title="$GH_TITLE" -f state=open --jq .number)
fi
echo "GitHub milestone #$NUMBER: $GH_TITLE"

# Milestone label (idempotent)
gh label create "milestone: ${NEW_NAME}" --color "BFD4F2" \
  --description "${NEW_TITLE} milestone" 2>/dev/null || true
```

## STEP 2 — Planning doc skeleton

Create `docs/milestones/<name>-planning.md` from the structure of the previous planning doc
(read the most recent `planning_doc` in `state/milestone.yaml` history): goal statement,
EPIC table (empty), dependency notes, risk table, exit criteria tied to the version target.

## STEP 3 — Write the active block in state/milestone.yaml

```yaml
active:
  name: <name>
  title: "<title>"
  github_milestone_number: <number>
  version: <version>
  status: active
  planning_doc: docs/milestones/<name>-planning.md
  labels:
    milestone: "milestone: <name>"
  open_epics: []   # filled as EPICs are created
```

```bash
make validate-milestone-state   # must pass before committing
```

## STEP 4 — Update state/current-milestone.md

New header section for the milestone: status Active, link to the planning doc, empty
progress table.

## STEP 5 — Commit + PR (never direct to main — ADR-023)

```bash
git checkout -b "chore/open-${NEW_NAME}"
git add state/milestone.yaml state/current-milestone.md "docs/milestones/${NEW_NAME}-planning.md"
git commit -s -m "chore(release): open ${NEW_NAME} — ${NEW_TITLE}

Scaffolds the GitHub milestone, planning doc, and milestone.yaml active block.

Assisted-by: Claude/<model-id-of-this-session>"
git push -u origin HEAD
gh pr create --title "chore(release): open ${NEW_NAME} — ${NEW_TITLE}" \
  --body "Opens milestone ${GH_TITLE} (target ${NEW_VERSION})." \
  --label "type: chore" --label "milestone: ${NEW_NAME}"
gh pr checks --watch --interval 30   # FOREGROUND — never end the turn while CI runs
gh pr merge --squash
```

## STEP 6 — Done report

```
=== Milestone opened: <name> — <title> ===
GitHub:   milestone #<number> + label "milestone: <name>"
Planning: docs/milestones/<name>-planning.md (skeleton — fill EPIC table next)
Config:   state/milestone.yaml active block written (PR #N merged)
Next:     create EPIC issues, then /milestone-plan.
```
