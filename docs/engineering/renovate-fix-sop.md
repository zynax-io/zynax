<!-- SPDX-License-Identifier: Apache-2.0 -->

# Renovate Dependency Update Fix SOP

**Scope:** Go module PRs opened by the Renovate bot that fail CI.
**SPDD contract:** every fix is an issue → branch → PR. No drive-by fixes.

---

## 1. When This Applies

Renovate opens a dependency PR. One or more of these required checks fails:

```
dco · lint · test-unit · test-integration · security
```

> `Conventional Commit title` is **informational only** — it never blocks merge.
> Fix it by renaming the PR title via `gh api -X PATCH repos/zynax-io/zynax/pulls/<N> -f title="..."`.

---

## 2. Failure Catalogue

### 2a. DCO failure on bot commits

**Symptom:** `dco` check fails; all commits authored by `renovate[bot]`.

**Root cause:** Renovate commits have no `Signed-off-by:`. The DCO check
now exempts `[bot]` email suffixes (PR #261). If this recurs the bot
exemption in `.github/workflows/ci.yml` may have been reverted.

**Fix:** verify the exemption block is present in the DCO loop:

```bash
author_email=$(git log -1 --format="%ae" "$sha")
if echo "$author_email" | grep -qE "\[bot\]"; then
  continue
fi
```

---

### 2b. Missing `go.sum` entries / `renovate/artifacts` failure

**Symptom:** `lint` or `test-unit` fails with:

```
missing go.sum entry for module providing package ...
```

**Root cause:** `protos/generated/go/go.sum` was never committed, or the
branch was opened before the go.sum baseline was established.

**Permanent fix:** `protos/generated/go/go.sum` must exist on `main`
(committed via PR #262). If it regresses, run the baseline procedure in
[§4 — Baseline Go Sum](#4--baseline-go-sum).

---

### 2c. Merge conflict in `go.sum` or `go.mod`

**Symptom:** GitHub shows "This branch has conflicts"; CI won't run.

**Root cause:** Another dependency PR was merged to `main` after Renovate
opened this branch, advancing `go.sum`. Renovate's rebase creates a
content conflict.

**Fix:** see [§3 — Standard Go Module Fix](#3--standard-go-module-fix).

---

### 2d. `go` directive too old

**Symptom:** `go mod tidy` exits with:

```
requires go >= X.Y (running go A.B; GOTOOLCHAIN=local)
```

**Root cause:** the new dependency version requires a newer Go toolchain
than the module's `go` directive declares.

**Fix:** `go mod tidy` will raise the directive automatically; commit the
change alongside the go.sum update. Use a Docker image matching the target
Go version (see [§3](#3--standard-go-module-fix)).

---

## 3. Standard Go Module Fix

Use this procedure for **2b**, **2c**, and **2d**.

### Step 1 — Identify the Renovate branch and version bump

```bash
gh pr view <PR_NUMBER> --repo zynax-io/zynax --json headRefName,title
# note: headRefName is the branch; title shows old→new versions
```

### Step 2 — Reset to latest main

```bash
git fetch origin main <branch>
git checkout <branch>
git reset --hard origin/main      # drop all previous fix commits
```

> `--hard` is safe here: we are about to re-apply the version bump from
> scratch. The Renovate commit itself is never needed — only the version
> numbers it introduced.

### Step 3 — Apply the version bump to `go.mod`

Edit `go.mod` in each affected module manually:

```bash
# protos/generated/go/go.mod
sed -i 's/google.golang.org\/grpc vOLD/google.golang.org\/grpc vNEW/' \
    protos/generated/go/go.mod

# services/workflow-compiler/go.mod
# (repeat for each bumped package)
```

Or open the file and change the version numbers directly.

### Step 4 — Run `go mod tidy` for each module

Use the Docker image matching the module's `go` directive:

```bash
# protos/generated/go  (go directive = 1.24.0 after grpc v1.80+)
docker run --rm \
  -v "$(pwd)/protos/generated/go:/work" \
  -w /work \
  golang:1.25-alpine \
  sh -c "GOWORK=off go mod tidy"

# services/workflow-compiler  (go directive = 1.25)
docker run --rm \
  -v "$(pwd)/services/workflow-compiler:/work/services/workflow-compiler" \
  -v "$(pwd)/protos/generated/go:/work/protos/generated/go" \
  -w /work/services/workflow-compiler \
  golang:1.25-alpine \
  sh -c "GOWORK=off go mod tidy"
```

> Always mount **both** modules when tidying `workflow-compiler` — it uses
> a `replace` directive pointing to `../../protos/generated/go`.

> If `go mod tidy` fails with "requires go >= X", use the image for that
> version: `golang:X-alpine`.

### Step 5 — Commit and force-push

```bash
git add protos/generated/go/go.mod protos/generated/go/go.sum \
        services/workflow-compiler/go.mod services/workflow-compiler/go.sum

git commit -m "$(cat <<'EOF'
fix(deps): go mod tidy after <package> vOLD → vNEW bump

<One line explaining why tidy was needed — e.g. "grpc v1.80 requires go
>= 1.24; bump proto module go directive from 1.22 to 1.24.0".>

Signed-off-by: Oscar Gómez Manresa <ogomezmanresa@gmail.com>
Assisted-by: Claude/claude-sonnet-4-6
EOF
)"

git push --force-with-lease origin <branch>
```

### Step 6 — Verify CI triggers

```bash
# wait ~15s then check:
gh api "repos/zynax-io/zynax/actions/runs?branch=<branch>&per_page=3" \
  --jq '.workflow_runs[] | {name, status, conclusion}'
```

If no runs appear after 30s, the push may not have triggered CI (transient
GitHub issue). Push another empty commit:

```bash
git commit --allow-empty -m "ci: force CI run

Signed-off-by: Oscar Gómez Manresa <ogomezmanresa@gmail.com>
Assisted-by: Claude/claude-sonnet-4-6"
git push origin <branch>
```

---

## 4. Baseline Go Sum

If `protos/generated/go/go.sum` is ever missing from `main`:

```bash
git checkout main
docker run --rm \
  -v "$(pwd)/protos/generated/go:/work" \
  -w /work \
  golang:1.25-alpine \
  sh -c "GOWORK=off go mod tidy"

git add protos/generated/go/go.sum protos/generated/go/go.mod
git commit -m "fix: add missing protos/generated/go go.sum baseline

Signed-off-by: Oscar Gómez Manresa <ogomezmanresa@gmail.com>
Assisted-by: Claude/claude-sonnet-4-6"
# open a PR — do not push directly to main
```

---

## 5. Merge Order for Dependency PRs

When multiple Renovate PRs are open simultaneously, merge in this order
to minimise conflict cascades:

1. Infrastructure (Docker base images, GitHub Actions)
2. Proto-only modules (`protos/generated/go`)
3. Services that depend on protos (`services/workflow-compiler`)
4. Packages with deep transitive dependency trees last
   (e.g. otelgrpc after grpc-go)

After each merge, check all remaining open Renovate PRs for conflicts
before proceeding to the next.

---

## 6. SPDD Compliance Checklist

Before opening any fix PR:

- [ ] A GitHub issue exists documenting the failure (root cause + scope)
- [ ] The fix branch contains exactly one logical change per commit
- [ ] Commit message follows `fix(deps): ...` with both trailers
- [ ] `GOWORK=off go mod tidy` was run for every affected module
- [ ] CI is green on all 5 required checks before requesting merge
- [ ] PR description references the issue number

---

*See also:* [dependency-strategy.md](dependency-strategy.md) for version
pinning policy and upgrade cadence.
