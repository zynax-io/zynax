---
name: Bump ci-runner digest
about: Update ci-runner digest references after a tools-image.yml rebuild
labels: ["type: chore", "area: ci"]
assignees: ''
---

<!--
This issue is opened automatically by tools-image.yml after every successful
ci-runner rebuild. It tracks the manual step of updating the pinned digest
across all workflow files.

Do NOT open this issue manually unless you have just rebuilt the ci-runner
image outside of the normal workflow.
-->

## ci-runner digest bump

**New digest:** `sha256:<replace-me>`
**Built by:** (workflow run URL)

> **The one question this answers:** is the toolchain everyone builds against current and pinned?
> **Why it matters:** a fresh, pinned digest keeps CI **reproducible** and the **supply chain**
> current (ADR-024/025) — the boring-but-load-bearing trust signal for maintainers, enterprises,
> and CNCF supply-chain review. Drift here silently breaks builds and erodes that trust.

---

## Steps

```bash
# 1. Create a fresh branch off main
git fetch origin --prune
git checkout main && git pull --rebase origin main
git checkout -b chore/bump-ci-runner-<short-digest>

# 2. Pin the new digest (updates images.yaml + re-stamps consumers)
make bump-ci-runner NEW_DIGEST=sha256:<replace-me>

# 3. Verify the consumers would re-stamp cleanly (dry-run, parity with --check)
(cd cmd/zynax-ci && GOWORK=off go run . bump-runner sha256:<replace-me> --root "$(git rev-parse --show-toplevel)" --dry-run)

# 4. Commit
git add -u
git commit -s -m "chore(ci): bump ci-runner digest to <short-digest>

Assisted-by: Claude/claude-sonnet-4-6"

# 5. Push and open PR
git push -u origin HEAD
gh pr create --title "chore(ci): bump ci-runner digest to <short-digest>" \
  --label "type: chore,area: ci" \
  --body "Closes #<this-issue>"

# 6. Wait for CI green, then rebase-merge and delete branch
gh pr checks <PR> --watch
git rebase origin/main && git push --force-with-lease
gh pr merge <PR> --rebase
git push origin --delete chore/bump-ci-runner-<short-digest>
```

---

## Checklist

- [ ] `make bump-ci-runner NEW_DIGEST=sha256:<replace-me>` ran successfully
- [ ] `zynax-ci bump-runner sha256:<replace-me> --dry-run` exits 0
- [ ] `config/ci-runner-digest.txt` updated
- [ ] All 18 workflow digest refs updated (ci.yml ×8, pr-checks.yml ×7, _test-go.yml ×1, _test-python.yml ×1, ai-context-budget.yml ×1)
- [ ] Branch created from fresh `origin/main`
- [ ] PR opened with title `chore(ci): bump ci-runner digest to <short-digest>`
- [ ] CI green
- [ ] Rebase on main → `gh pr merge --rebase` → branch deleted
