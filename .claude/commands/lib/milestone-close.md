---
description: Close the active milestone — verify EPICs done, truth-pass docs, signed version tag, GitHub Release, rotate state/milestone.yaml active→history. The only sanctioned writer of milestone.yaml besides /milestone open.
argument-hint: "[--dry-run]  verify + report only, change nothing"
---

# /milestone close — Milestone Lifecycle: Close

Close the active milestone end-to-end. This command (with `/milestone open`) is the ONLY
sanctioned writer of `state/milestone.yaml` — every other command reads it.

> **Destructive surface:** pushes a signed version tag and creates a public GitHub Release.
> Run `--dry-run` first in any session where the milestone state is uncertain.

---

## STEP 0 — Load config + sync

```bash
git fetch origin --prune && git checkout main && git pull --rebase origin main
[ -z "$(git status --porcelain)" ] || { echo "dirty tree — STOP"; exit 1; }

# ── Active-milestone config (SSoT: state/milestone.yaml) ────────────────────
CFG=state/milestone.yaml
MILESTONE_NAME=$(awk '/^active:/{f=1} f && /^  name:/{print $2; exit}' "$CFG")
MILESTONE_TITLE=$(awk -F'"' '/^active:/{f=1} f && /^  title:/{print $2; exit}' "$CFG")
MILESTONE_NUMBER=$(awk '/^active:/{f=1} f && /^  github_milestone_number:/{print $2; exit}' "$CFG")
MILESTONE_VERSION=$(awk '/^active:/{f=1} f && /^  version:/{print $2; exit}' "$CFG")
PLANNING_DOC=$(awk '/^active:/{f=1} f && /^  planning_doc:/{print $2; exit}' "$CFG")
MILESTONE_LABEL=$(awk -F'"' '/^    milestone:/{print $2; exit}' "$CFG")
GH_MILESTONE="${MILESTONE_TITLE} (${MILESTONE_NAME})"
# ─────────────────────────────────────────────────────────────────────────────
echo "Closing: $GH_MILESTONE → $MILESTONE_VERSION"
```

## STEP 1 — Verify every EPIC is actually closed (abort if not)

```bash
# Live query is authoritative; the static open_epics list is only a hint.
OPEN_EPICS=$(gh issue list --milestone "$GH_MILESTONE" --label "type: epic" \
  --state open --limit 50 --json number,title --jq '.[] | "#\(.number) \(.title)"')
if [ -n "$OPEN_EPICS" ]; then
  echo "ABORT — open EPICs remain in $GH_MILESTONE:"
  echo "$OPEN_EPICS"
  exit 1
fi

# Also report (not gate) any open non-EPIC stragglers, so the human can decide
# to move them to the next milestone before re-running.
gh issue list --milestone "$GH_MILESTONE" --state open --limit 100 --json number,title \
  --jq '.[] | "straggler: #\(.number) \(.title)"'
```

If `--dry-run`: print the verification result and STOP here.

## STEP 2 — Truth-pass all doc surfaces

Run `/reconcile` (plans first, executes on approval) to reconcile every status surface
(README / ROADMAP / ARCHITECTURE / CLAUDE / state / planning doc / canvases) to live GitHub
state. Do not proceed until its PR is merged.

## STEP 3 — Signed version tag

```bash
git checkout main && git pull --rebase origin main
git tag -s "$MILESTONE_VERSION" -m "Release $MILESTONE_VERSION"
git push origin "$MILESTONE_VERSION"
# The tag push triggers the release workflow (retag model — ADR-027):
# images retagged to the version, cosign-signed, SBOMs attached.
```

## STEP 4 — GitHub Release

```bash
# Wait for the tag-triggered workflow to finish before creating the release notes
gh run list --limit 5 --json name,status,headBranch
gh release create "$MILESTONE_VERSION" --generate-notes \
  --title "$MILESTONE_VERSION: $MILESTONE_TITLE"
```

## STEP 5 — Close the GitHub milestone

```bash
gh api -X PATCH "repos/{owner}/{repo}/milestones/${MILESTONE_NUMBER}" -f state=closed
```

## STEP 6 — Rotate active → history in state/milestone.yaml

Edit `state/milestone.yaml` (this command is a sanctioned writer):
1. Prepend the active block to `history:` with `released: <today YYYY-MM-DD>`.
2. Leave `active:` EMPTY of milestone content except a placeholder comment —
   `/milestone open` fills it. (If the next milestone is already decided, run
   `/milestone open` immediately after and let it write the block.)
3. Clear `open_epics`.
4. Run `make validate-milestone-state` — must pass before committing.

## STEP 7 — Update state/current-milestone.md header

Mark the closed milestone Complete with its version; note that the next milestone is not
yet open (or hand off to `/milestone open`).

## STEP 8 — Commit + PR (never direct to main — ADR-023)

```bash
git checkout -b "chore/release-${MILESTONE_VERSION}"
git add state/milestone.yaml state/current-milestone.md
git commit -s -m "chore(release): close ${MILESTONE_NAME}, rotate milestone state

${GH_MILESTONE} released as ${MILESTONE_VERSION}.

Assisted-by: Claude/<model-id-of-this-session>"
git push -u origin HEAD
gh pr create --title "chore(release): close ${MILESTONE_NAME} — ${MILESTONE_VERSION}" \
  --body "Rotates active→history in state/milestone.yaml after the ${MILESTONE_VERSION} release." \
  --label "type: chore" --label "$MILESTONE_LABEL"
gh pr checks --watch --interval 30   # FOREGROUND — never end the turn while CI runs
gh pr merge --squash
```

## STEP 9 — Done report

```
=== Milestone closed: <name> ===
Release:   <version> — <release URL>
Milestone: <GitHub milestone URL> (closed)
Config:    state/milestone.yaml rotated (PR #N merged)
Next:      run /milestone open to scaffold the next milestone.
```
