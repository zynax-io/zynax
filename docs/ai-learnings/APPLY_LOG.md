# Expert Learning Apply Log

> Append-only. Each `/m6-learn` run adds one entry. Human edits the **Status** column.
> **Delta** is filled in automatically by `/m6-learn --apply` after commit.
>
> Status lifecycle:
> - `pending` → leave it, re-evaluated next run
> - `pending` → edit to `applied` + set Delta to `pending-commit` → run `/m6-learn --apply`
> - `pending` → edit to `rejected` → stays rejected, synthesizer skips it forever

---

<!-- runs appended below by /m6-learn -->
