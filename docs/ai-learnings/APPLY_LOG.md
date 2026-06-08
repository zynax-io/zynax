# Expert Learning Apply Log

> Append-only. Each `/m6-learn` run adds one entry. Human edits the **Status** column.
> **Delta** is filled in automatically by `/m6-learn --apply` after commit.
>
> Status lifecycle:
> - `pending` → leave it, re-evaluated next run
> - `pending` → edit to `applied` + set Delta to `committed` → run `/m6-learn --apply`
> - `pending` → edit to `rejected` → stays rejected, synthesizer skips it forever

---

<!-- runs appended below by /m6-learn -->

## Run 2026-06-08 12:00 — domains: go-services, ci-release

| # | Domain | Pattern | Source sessions | Status | Delta |
|---|--------|---------|-----------------|--------|-------|
| 1 | go-services | Shared workspace: verify+set branch before every git op | #818, #819, #826, #827, #828 | applied | committed |
| 2 | go-services | Stage with explicit file paths, never git add . | #817, #818, #819, #826, #828 | applied | committed |
| 3 | go-services | Never use gh api update-branch — creates unsigned merge commit | #825, #826 | applied | committed |
| 4 | go-services | git stash unsafe across branches — use restore or targeted checkout | #818, #819 | applied | committed |
| 5 | ci-release | Stage specific files by path in shared workspace | #860, #875 | applied | committed |
| 6 | ci-release | Cherry-pick rescue for commits that land on wrong branch | #867, #819, #828 | applied | committed |

**Summary:** 6 proposed | 6 applied | 0 rejected | 0 pending
