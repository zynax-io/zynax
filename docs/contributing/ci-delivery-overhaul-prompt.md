<!-- SPDX-License-Identifier: Apache-2.0 -->

# CI/CD + Delivery Tooling Overhaul — Master Implementation Prompt

> **Tracking:** umbrella EPIC [#1107](https://github.com/zynax-io/zynax/issues/1107) ·
> tooling EPIC [#1108](https://github.com/zynax-io/zynax/issues/1108) (Part 0) ·
> pipeline EPIC [#1109](https://github.com/zynax-io/zynax/issues/1109) (Parts 1–3) ·
> stories #1110–#1122. Locked operator decisions are recorded in the #1107 body.

> **What this document is:** A self-contained specification and execution prompt for
> implementing the full CI/CD hardening and delivery-tooling generalization for Zynax.
> Feed it verbatim to an implementation agent, or use it as the planning spec for the
> sequence of PRs and GitHub issues described below.
>
> **Pre-reading:** Before acting, the agent must read:
> - `CLAUDE.md` and `AGENTS.md` (architecture invariants, commit rules, DCO, attribution)
> - `state/current-milestone.md` (active milestone state)
> - `docs/adr/INDEX.md` (existing decisions that constrain the design)
> - `.github/workflows/ci.yml`, `release.yml`, `pr-size.yml`, `pr-checks.yml`,
>   `dev-advisory.yml`, `post-merge-completeness.yml`
> - `.claude/commands/m6-plan.md`, `m6-orchestrate.md`, `m6-issue-generate.md`,
>   `resume-m6.md` (the commands being generalized)
> - `state/milestone.yaml` if it already exists (this spec creates it)

---

## Pre-flight questions — ask the operator before starting any code

1. **GHCR package visibility for staging lane.** Pre-merge images will be pushed to
   `ghcr.io/zynax-io/zynax/staging/<svc>:pr-<sha>`. Should those packages be public
   (consistent with the existing public registry) or private? Private requires an
   explicit `--visibility private` on the package after first push, and means
   contributors cannot pull staging images locally without authentication.
   *Recommended: public, consistent with existing packages.*

2. **Trivy severity gate.** The pre-merge image scan blocks the PR on CVEs at which
   threshold? Options: `CRITICAL` only, or `CRITICAL,HIGH`.
   *Recommended: `CRITICAL,HIGH` to match the existing `make security` standard.*

3. **Unfixable upstream CVEs in base images.** When a HIGH CVE exists in
   `gcr.io/distroless/static` or `golang:alpine` but has no upstream fix yet, the
   pre-merge gate will block all PRs indefinitely. Is a `.trivyignore` exception file
   acceptable for confirmed unfixable/accepted-risk CVEs? If yes, who approves
   additions to that file?
   *Recommended: yes, with a `# accepted-until: YYYY-MM-DD reason:` annotation on
   each entry so they expire automatically.*

4. **Digest bot-commit model.** After a merge to main, the retag job must commit the
   updated digests to `images/images.yaml`. The cleanest approach is a direct
   `[skip ci]` bot commit to main from the Actions job. This requires the repository
   setting "Allow GitHub Actions to create and approve pull requests" + write
   permission on the `GITHUB_TOKEN`. Is that acceptable, or should the digest update
   open an automated PR that auto-merges on CI green?
   *Recommended: direct bot commit with `[skip ci]` — this is the Flux/ArgoCD model
   and avoids the infinite-loop problem of a digest-update PR triggering CI.*

5. **dev-advisory.yml artifacts.** Deleting `dev-advisory.yml` also orphans the
   `automation/experts/*.yaml` and `automation/orchestrator/` directory (the LLM
   config). Should those be: (a) deleted, (b) moved to `docs/archive/dev-advisory/`,
   or (c) kept in place for potential Wave 4 re-use?
   *Recommended: move to `docs/archive/dev-advisory/` — preserves the work without
   it running on every PR.*

6. **Manifesto home.** Where should the engineering manifesto live?
   - `docs/contributing/engineering-manifesto.md` (recommended — visible to external
     contributors, linked from CONTRIBUTING.md)
   - `docs/adr/ADR-027-engineering-manifesto.md` (ADR format, immutable record)
   *Recommended: `docs/contributing/engineering-manifesto.md` — a manifesto is living
   guidance, not an immutable decision record.*

7. **ADR for shift-left pipeline.** The "build once in PR, retag on merge" model is a
   one-way architectural door (committing to it changes the branching invariants for
   GHCR and the release pipeline). Should this get ADR-027?
   *Recommended: yes. Record the decision before implementing.*

8. **PR SHA lookup during retag.** On a squash-merge to main, `github.sha` is the
   merge commit SHA, not the PR head SHA that was used to build the staging image.
   Recovery options: (a) store `pr-head-sha` as an OCI label on the staging image
   during pre-merge build, then `crane inspect` to recover it in the retag job; or
   (b) trigger retag from `workflow_run` on "CI" completion where `pull_request.*`
   context is available. Which approach?
   *Recommended: (a) OCI label — self-contained, no dependency on workflow ordering.*

---

## Dependency order (execute in this sequence)

```
Part 0: Foundation                     ← MUST BE FIRST (everything else reads from here)
  0A. Create state/milestone.yaml
  0B. Generalize m6-* → milestone-* commands
  0C. Deduplicate state from docs surfaces

Part 1: Retire advisory CI             ← Reduces noise immediately, low risk
  1A. Delete dev-advisory.yml
  1B. Demote post-merge-completeness → weekly-audit.yml
  1C. Update PR size exclusion list

Part 2: ADR for shift-left pipeline    ← Decision record before implementation
  2A. Write ADR-027

Part 3: CI hardening                   ← The structural pipeline change
  3A. Pre-merge image build + Trivy + Hadolint + SBOM
  3B. PR image cleanup on PR close
  3C. Convert release.yml to retag model (atomic digest sync)
  3D. Supply chain hardening (SHA pins, permissions, SLSA, dependency-review)

Part 4: Update existing M6 issues      ← Reconcile open issues with new approach
  4A. Re-scope #873 and #881 (DevAuto EPICs)
  4B. Update #1089 (release.yml matrix → now a ci.yml build-images task)
  4C. Update #771 (CI-E2E — block on Part 3, update scope)
  4D. Update #1073 (Postgres migration — link to new digest atomicity model)

Part 5: New M6 EPICs + SPDD canvases   ← GitHub issues + canvas for Parts 0-3
  5A. EPIC: Delivery Tooling Generalization (Part 0)
  5B. EPIC: CI/CD Pipeline Hardening (Parts 1-3)

Part 6: Engineering Manifesto          ← Capstone document
```

---

## Part 0 — Foundation: milestone-agnostic delivery tooling

**Why first:** Every subsequent command, canvas, and CI workflow references the active
milestone. Hardcoding M6 into commands means repeating this entire migration for M7.
`state/milestone.yaml` eliminates that cost permanently.

### 0A. Create `state/milestone.yaml`

New file. This is the single machine-readable source of truth read at runtime by all
delivery commands. No command may hardcode a milestone name, number, or label.

```yaml
# state/milestone.yaml
# Machine-readable milestone config. Updated by /milestone-close and /milestone-new.
# Never edit by hand — use those commands or the validate-milestone-state make target.
active:
  name: M6
  title: "K8s Production-Ready"
  github_milestone_number: 6
  version: v0.5.0
  status: active              # active | closing | complete
  planning_doc: docs/milestones/M6-planning.md
  labels:
    milestone: "milestone: M6"
  open_epics: [766, 767, 768, 467, 769, 770, 1073, 1086]
  # ^ update when EPICs close; commands may also query the API at runtime

history:
  - name: M5
    title: "Production Hardening"
    version: v0.4.0
    released: 2026-05-29
    github_milestone_number: 5
    planning_doc: docs/milestones/M5-plan.md
  - name: M1
    title: "Contracts Foundation"
    version: v0.1.0
    released: 2026-04-21
    github_milestone_number: 1
    planning_doc: docs/milestones/M5-plan.md
```

Also create `state/milestone.schema.json` (JSON Schema for the above).
Add `validate-milestone-state` to `Makefile` (runs `python3 -c "import yaml,jsonschema; ..."`
inside Docker). Add the schema check to `pr-checks.yml` as a fast non-blocking advisory
step initially, promoted to required once the schema is stable.

**Question during 0A:** Should `open_epics` be a static list (maintained by hand as EPICs
close) or derived dynamically at runtime from `gh issue list --milestone $number --label
"type: epic" --state open`? Dynamic is more accurate for long-running commands; static is
safer for offline/local use. *Recommended: keep both — static list as a hint, commands
query the API when GH_TOKEN is available and fall back to the static list.*

### 0B. Generalize `m6-*` → `milestone-*` commands (hard rename, no aliases)

**Delete** these files:
- `.claude/commands/m6-plan.md`
- `.claude/commands/m6-orchestrate.md`
- `.claude/commands/m6-issue-generate.md`
- `.claude/commands/resume-m6.md`

**Create** these replacements. Each must open `state/milestone.yaml` at the start and
inject the active milestone's `name`, `github_milestone_number`, `labels`, and
`planning_doc` into every GitHub API call and label reference. No M6/M7/M8 string
may appear in the file body.

| Old command | New command | Core change |
|-------------|-------------|-------------|
| `m6-plan.md` | `milestone-plan.md` | Read milestone number from config; output next `/issue-deliver` commands |
| `m6-orchestrate.md` | `milestone-orchestrate.md` | Read milestone and labels from config; route to expert subagents |
| `m6-issue-generate.md` | `issue-deliver.md` | Generic story delivery; milestone/labels injected from config |
| `resume-m6.md` | `resume-milestone.md` | Entry-point for a delivery session; reads config |
| *(new)* | `milestone-close.md` | Close GitHub milestone, tag version, generate release notes, update config |
| *(new)* | `milestone-new.md` | Scaffold next milestone: GitHub milestone + planning doc + config update |

For `milestone-close.md`, the steps are:
1. Read `state/milestone.yaml` active block.
2. Confirm all EPICs in `open_epics` are actually closed (query GitHub; abort if not).
3. Run `/repo-clean` truth-pass to reconcile all docs surfaces.
4. Push version tag: `git tag -s v<version> -m "Release v<version>"`.
5. Trigger `release-tag.yml` (or wait for it to auto-trigger on tag push).
6. `gh release create v<version> --generate-notes --title "v<version>: <title>"`.
7. Move active block to history in `state/milestone.yaml`; clear `open_epics`.
8. Update `state/current-milestone.md` header.
9. Commit: `chore(release): close M<N>, prepare for M<N+1>`.

**Update** `.claude/settings.json` skill registrations to reflect new names.
**Update** `CLAUDE.md §SPDD command table` with new names.
**Update** `AGENTS.md` wherever `m6-*` is referenced.

**Question during 0B:** The `spdd-story.md` command hardcodes `milestone: M6` in the
label it applies to new story issues (line that reads `--label "milestone: M6"`). Should
this also be updated to read from `state/milestone.yaml`, or is `spdd-story.md` already
SPDD-exempt and milestone-label injection is done by the calling command (`issue-deliver`)?
*Recommended: `spdd-story.md` stays generic (no milestone label); `issue-deliver.md`
injects the label after story creation.*

### 0C. Deduplicate milestone state from documentation surfaces

Make each surface serve exactly one purpose. Do not remove historical content — only
remove live-status content that now lives in `state/current-milestone.md`.

| Surface | Change |
|---------|--------|
| `CLAUDE.md §Per-Milestone Scope` | Keep the narrative rows (they're historical context for the architecture). Remove the ✅/🚧/⚠ status badges and open-issue counts. Add: `> Live progress: [state/current-milestone.md](state/current-milestone.md)` |
| `CLAUDE.md §Milestone Status` | Shrink to two lines + two links. Remove the paragraph of prose about M6 delivery state. |
| `README.md` | Remove inline milestone progress table. Keep the one-liner: "Active milestone: M6 🚧 — see [state/current-milestone.md](state/current-milestone.md)" |
| `ROADMAP.md` | Strip per-EPIC status columns from the milestone checklist. Keep milestone names, version targets, GitHub Milestone links. Remove all issue numbers. |
| `state/current-milestone.md` | Add header: `<!-- Canonical status file. Updated by /milestone-close and /repo-clean. Do not edit by hand. -->` |

---

## Part 1 — Retire advisory CI

### 1A. Delete `dev-advisory.yml`

Action: `git rm .github/workflows/dev-advisory.yml`.

Move `automation/experts/`, `automation/orchestrator/`, and `scripts/invoke-llm.sh`
to `docs/archive/dev-advisory/` (preserves the work, stops it from running).

Rationale: The 8-expert + orchestrator Wave 0+1+2 model runs on every PR, posts advisory
comments that are not actionable gates, consumes API quota per-PR, and duplicates
signal already provided by the deterministic checks in `ci.yml` and `pr-checks.yml`.
The Wave 4 vision (#881) is valuable but will be re-expressed as a Zynax native
workflow — not a GitHub Actions LLM call.

Collateral labels and references to remove: any `area: automation` label on issues
that referred only to dev-advisory (not to the Wave 4 runtime); update `CLAUDE.md`
to remove the Wave 0–3 mention under M6 DevAuto.

### 1B. Demote `post-merge-completeness.yml` → `weekly-audit.yml`

**File rename:** `.github/workflows/post-merge-completeness.yml` →
`.github/workflows/weekly-audit.yml`

**Remove triggers:** `push: branches: [main]` and `workflow_run: workflows: ["Release"]`.
**Keep trigger:** `schedule: - cron: '0 2 * * 0'` and `workflow_dispatch`.

**Remove all `continue-on-error: true`** from individual jobs. The weekly audit should
fail loudly and visibly as a failed workflow run; the on-call person sees it Monday
morning. `continue-on-error` was appropriate when this was an advisory mesh; it is not
appropriate for a real audit.

**Remove all `gh issue create` / `[AUTO]` issue creation steps.** When the audit finds
something, the failed workflow run is the signal. An engineer investigates and opens a
proper issue with full context, not an auto-generated skeleton.

**Remove `post-merge-verdict` fan-in job.** Replace with a `audit-summary` job that
writes a clear pass/fail table to `GITHUB_STEP_SUMMARY` and exits non-zero on any
failure. The job graph: all three audit jobs → `audit-summary`.

**Questions during 1B:**
- The `post-merge-image-test` job currently only runs on `workflow_run` (Release
  success). With the retag model (Part 3), images exist on main after every merge.
  Should the weekly audit pull `:latest` for smoke testing, or `:main-<sha>` for a
  specific commit? *Recommended: `:latest` — simpler, matches what a user would pull.*
- Should the drift check (images.yaml vs live GHCR) be kept in the weekly audit, or
  retired entirely since the retag model makes drift structurally impossible?
  *Recommended: keep it as a sanity check but demote to warning-only (no exit code 1).*

### 1C. Update PR size exclusions in `pr-size.yml`

Current `skipPattern` (line 51 of `pr-size.yml`):
```
/\.(pb\.go|pb\.py|sum|lock|png|jpg|gif|svg)$|\/generated\/|CHANGELOG\.md$|
^\.github\/workflows\/|AGENTS\.md$|^docs\/|^state\/|^\.claude\//
```

**Add to skipPattern:**
- `^images\/images\.yaml$` — SoT registry file; changes atomically with releases, not with code
- `^infra\/helm\/` — Helm chart templates and values; config, not logic
- `^spec\/` — AsyncAPI + JSON Schema fixtures; contract files, not implementation
- `^automation\/` — LLM config yaml files; being archived anyway
- `^Makefile$` — build orchestration; changes are typically one-liners
- `^CLAUDE\.md$` — project doc; not code
- `^ROADMAP\.md$`, `^README\.md$` — narrative docs
- `^docs\/spdd\/` — canvas files; prose, not code

**Updated full skipPattern:**
```javascript
const skipPattern = /\.(pb\.go|pb\.py|sum|lock|png|jpg|gif|svg)$
  |\/generated\/
  |CHANGELOG\.md$
  |^\.github\/workflows\/
  |AGENTS\.md$
  |^docs\/
  |^state\/
  |^\.claude\/
  |^images\/images\.yaml$
  |^infra\/helm\/
  |^spec\/
  |^automation\/
  |^Makefile$
  |^CLAUDE\.md$
  |^ROADMAP\.md$
  |^README\.md$/x;
```

Also: update `CLAUDE.md §PR size` exclusions list to match.

---

## Part 2 — ADR-027: Shift-Left Pipeline Model

Create `docs/adr/ADR-027-shift-left-pipeline.md` before writing any workflow code.

The ADR must record:
- **Decision:** Container images are built exactly once in the PR's pre-merge CI, pushed
  to a staging lane in GHCR (`staging/<svc>:pr-<sha>`), scanned with Trivy, and
  signed with a pre-merge attestation. On merge to main, the staging image is **retagged**
  (not rebuilt) to `<svc>:main-<sha>` and `:latest`. On version tag, it is retagged to
  `<svc>:v*.*.*`. The image in production is the exact binary that passed the security gate.
- **Consequences:**
  - Positive: Supply-chain integrity (scan == deploy), no rebuild nondeterminism, faster
    post-merge pipeline (retag is ~10s, build is ~5 min).
  - Negative: Staging images accumulate; requires a PR-close cleanup job. The PR head
    SHA must be recoverable from the merge event (via OCI label on the staging image).
- **Relationship to ADR-024** (image reference management): The retag model replaces the
  post-merge build. The `images/images.yaml` digest update moves from a manual/advisory
  step to an atomic bot commit on every merge. Digest drift becomes structurally impossible.
- **Relationship to ADR-025** (SLSA provenance): SLSA Build L1 attestations continue to
  be generated by `docker/build-push-action` defaults. The retag step must preserve
  the attestation manifest from the staging image.

---

## Part 3 — CI hardening: shift-left security pipeline

### 3A. Add `build-images` job to `ci.yml` (pre-merge)

Add after `lint` and before `e2e-smoke`. The job runs only when
`needs.changes.outputs.docker == 'true'` (extend the existing change detection output
to include a `docker` flag: any change under `services/*/` or `infra/docker/Dockerfile.*`
or `agents/adapters/*/Dockerfile`).

**Job structure:**

```yaml
build-images:
  name: "Build + scan: ${{ matrix.service }}"
  runs-on: ubuntu-24.04
  needs: [lint, changes]
  if: needs.changes.outputs.docker == 'true'
  permissions:
    contents: read
    packages: write
    security-events: write
    id-token: write          # cosign OIDC pre-merge attestation
  strategy:
    fail-fast: false
    matrix:
      service: [api-gateway, engine-adapter, workflow-compiler,
                task-broker, agent-registry, event-bus, memory-service]
  steps:
    - uses: actions/checkout@<sha>  # pin to SHA
    - uses: docker/setup-buildx-action@<sha>
    - uses: docker/login-action@<sha>
      with:
        registry: ghcr.io
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}

    - name: Hadolint — Dockerfile lint
      uses: hadolint/hadolint-action@<sha>
      with:
        dockerfile: services/${{ matrix.service }}/Dockerfile
        # failure-threshold: warning  (initially; promote to error once clean)

    - name: Build and push to staging lane
      id: build
      uses: docker/build-push-action@<sha>
      with:
        context: services/${{ matrix.service }}
        push: true
        tags: ghcr.io/zynax-io/zynax/staging/${{ matrix.service }}:pr-${{ github.sha }}
        labels: |
          org.opencontainers.image.revision=${{ github.sha }}
          io.zynax.pr-head-sha=${{ github.sha }}
        cache-from: type=gha,scope=${{ matrix.service }}
        cache-to: type=gha,mode=max,scope=${{ matrix.service }}
        # provenance: true is the default — preserves SLSA attestation

    - name: Trivy scan (CRITICAL+HIGH = fail)
      uses: aquasecurity/trivy-action@<sha>
      with:
        image-ref: ghcr.io/zynax-io/zynax/staging/${{ matrix.service }}:pr-${{ github.sha }}
        severity: CRITICAL,HIGH
        exit-code: '1'
        ignore-unfixed: false
        trivyignores: .trivyignore     # for accepted-risk upstream CVEs only
        format: sarif
        output: trivy-${{ matrix.service }}.sarif

    - name: Upload Trivy SARIF to GitHub Security tab
      uses: github/codeql-action/upload-sarif@<sha>
      if: always()   # upload even on failure so findings are visible
      with:
        sarif_file: trivy-${{ matrix.service }}.sarif

    - name: Generate SBOM (CycloneDX JSON)
      uses: anchore/sbom-action@<sha>
      with:
        image: ghcr.io/zynax-io/zynax/staging/${{ matrix.service }}:pr-${{ github.sha }}
        format: cyclonedx-json
        output-file: sbom-${{ matrix.service }}.json
      if: always()

    - name: Upload SBOM artifact
      uses: actions/upload-artifact@<sha>
      if: always()
      with:
        name: sbom-${{ matrix.service }}-${{ github.sha }}
        path: sbom-${{ matrix.service }}.json
        retention-days: 30
```

**Add `build-images` to required status checks.** Document in the PR description that
the branch protection rule must be updated after this workflow lands:
```
gh api -X PATCH repos/zynax-io/zynax/branches/main/protection \
  --jq '.required_status_checks.contexts += ["Build + scan: api-gateway", ...]'
```
(Do this for each matrix entry.)

**Update `e2e-smoke.yml`** to pull from the staging lane instead of building locally:
replace the `docker build` step with `docker pull ghcr.io/zynax-io/zynax/staging/<svc>:pr-${{ github.sha }}`.

**Question during 3A:** The change-detection matrix currently produces per-service flags
(`go_api_gateway`, `go_engine_adapter`, etc.) but `build-images` uses a fixed matrix.
Should `build-images` run the full matrix on any `docker == 'true'` change, or should
each service only rebuild when its own files change? Running all services on any docker
change is simpler but slower. Per-service skipping requires a more complex `if:` condition.
*Recommended: run all services when any Dockerfile changes (a base image change affects
all), but skip the matrix entirely when no Dockerfiles changed.*

### 3B. Add `pr-image-cleanup.yml` (new file)

Trigger: `pull_request: types: [closed]`

Deletes staging lane images for the closed PR's head SHA to avoid GHCR accumulation.

```yaml
name: PR Image Cleanup
on:
  pull_request:
    types: [closed]
permissions:
  packages: write
jobs:
  cleanup:
    name: Delete staging images for pr-${{ github.event.pull_request.head.sha }}
    runs-on: ubuntu-24.04
    strategy:
      fail-fast: false
      matrix:
        service: [api-gateway, engine-adapter, workflow-compiler,
                  task-broker, agent-registry, event-bus, memory-service]
    steps:
      - name: Delete staging package version
        uses: actions/delete-package-versions@<sha>
        with:
          package-name: zynax/staging/${{ matrix.service }}
          package-type: container
          token: ${{ secrets.GITHUB_TOKEN }}
          # match the specific pr-<sha> tag
```

### 3C. Convert `release.yml` to retag model

**Current:** `release.yml` triggers on `push: branches: [main]` and builds multi-arch
images from scratch.

**Target:** On push to main, retag the staging image to `main-<sha>` and `latest`.
On version tag, retag to `v*.*.*`.

**Key implementation detail — recovering PR head SHA:**
During the pre-merge build, the staging image was labeled with:
`io.zynax.pr-head-sha=${{ github.sha }}`
where `${{ github.sha }}` at PR time is the PR branch head SHA.

On the squash-merge commit, `github.sha` is the new merge commit SHA. To find the staging
image, inspect the merge commit's parents or use the label:
```bash
# Option A: read label from the staged image of the first parent
# On merge, github.event.before is the SHA of main before the merge,
# github.sha is the merge commit. The PR head SHA was the branch tip.
# github.event.pull_request is NOT available on push events.
#
# Solution: store the PR head SHA as an OCI label on the staging image during build,
# then crane inspect it during the retag job:
STAGING_IMG="ghcr.io/zynax-io/zynax/staging/<svc>:pr-${MERGE_SHA}"
# ^ This won't work because we used the PR head SHA as the tag.
#
# Better: use git to find the branch head SHA from the merge commit.
# A squash-merge on GitHub creates: merge_commit with one parent (main).
# The PR head SHA is in the merge commit's message as "Squashed commit ... from ..."
# OR: use the GitHub API: GET /repos/{owner}/{repo}/commits/{merge_sha} → parents[0] is main,
# there is no second parent for a squash merge.
#
# REAL SOLUTION: trigger the retag from workflow_run on "CI" completion,
# which has access to github.event.workflow_run.head_sha (the PR branch SHA).
```

**Revised trigger for retag:**

```yaml
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
    branches: [main]
```

This provides `github.event.workflow_run.head_sha` which is the PR branch head SHA
used to tag the staging image. This is the cleanest solution.

**Retag job structure:**

```yaml
retag-on-merge:
  name: Retag staging → main-<sha> + latest
  runs-on: ubuntu-24.04
  if: github.event.workflow_run.conclusion == 'success'
  permissions:
    contents: write    # for the images.yaml bot commit
    packages: write
    id-token: write    # cosign OIDC
  steps:
    - uses: actions/checkout@<sha>
      with:
        token: ${{ secrets.GITHUB_TOKEN }}

    - uses: docker/login-action@<sha>
      with:
        registry: ghcr.io
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}

    - name: Install crane + cosign
      run: |
        # install crane (google/go-containerregistry) and cosign

    - name: Retag staging → prod
      env:
        PR_SHA: ${{ github.event.workflow_run.head_sha }}
        MERGE_SHA: ${{ github.event.workflow_run.head_commit.id || github.sha }}
      run: |
        for svc in api-gateway engine-adapter workflow-compiler \
                   task-broker agent-registry event-bus memory-service; do
          STAGING="ghcr.io/zynax-io/zynax/staging/${svc}:pr-${PR_SHA}"
          PROD_SHA="ghcr.io/zynax-io/zynax/${svc}:main-${MERGE_SHA}"
          PROD_LATEST="ghcr.io/zynax-io/zynax/${svc}:latest"

          docker buildx imagetools create -t "${PROD_SHA}" -t "${PROD_LATEST}" "${STAGING}"

          cosign sign --yes "${PROD_SHA}"
          cosign sign --yes "${PROD_LATEST}"
        done

    - name: Update images.yaml digests
      run: |
        # For each service image, compute the new digest and update images/images.yaml
        for svc in api-gateway engine-adapter ...; do
          DIGEST=$(crane digest "ghcr.io/zynax-io/zynax/${svc}:latest")
          # Use yq or python to update the digest field for this service in images.yaml
          python3 tools/update-image-digest.py --name "${svc}" --digest "${DIGEST}"
        done

    - name: Commit digest update to main [skip ci]
      run: |
        git config user.name "github-actions[bot]"
        git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
        git add images/images.yaml
        git diff --cached --quiet || \
          git commit -m "chore(images): sync digests after main-${MERGE_SHA} [skip ci]

        Automated digest update after merge of PR branch ${PR_SHA}.
        Updated by release.yml retag job.

        Signed-off-by: github-actions[bot] <41898282+github-actions[bot]@users.noreply.github.com>"
        git push
```

**Retag on version tag (separate job or separate trigger):**

```yaml
on:
  push:
    tags: ['v*.*.*']
```

Steps:
1. Find `main-<sha>` for this tag: `git log -1 --format=%H ${{ github.ref_name }}`
2. Retag `main-<sha>` → `v*.*.*` for each service.
3. Cosign sign the version tag.
4. `gh release create ${{ github.ref_name }} --generate-notes`.
5. Trigger SDK + proto-stubs publish as `workflow_call`.
6. Generate SBOM for the version-tagged images and attach to the release.

**Remove** the existing multi-arch build jobs from `release.yml`. The new file is
structurally simpler: no BuildKit setup, no Dockerfile context, no push matrix. Just
crane retag + cosign + git commit.

Also update `tools-image.yml` to the same retag model for the CI runner image.

**Question during 3C:** The existing `release.yml` builds `agents/adapters/*/` Python
adapter images alongside Go service images. Those adapters have their own Dockerfiles.
Should they follow the same staging-lane pre-merge build model, or should they continue
to be built post-merge? *Recommended: yes, same model — they go to production and should
be scanned before merge. Add adapters to the `build-images` matrix.*

### 3D. Supply chain hardening

Apply to all workflow files:

1. **Pin every action to a commit SHA.** Replace `uses: actions/checkout@v4` with
   `uses: actions/checkout@<full-sha> # v4.x.x`. Use `pin-github-action` CLI or manual
   pinning. For every third-party action (docker/*, sigstore/cosign-installer, etc.).

2. **Add `permissions:` blocks.** At the workflow level, set `permissions: contents: read`
   as the default. Escalate per-job only:
   - `packages: write` — only in build/retag/publish jobs
   - `id-token: write` — only in cosign signing jobs
   - `pull-requests: write` — only in pr-size labeling
   - `issues: write` — only in jobs that create issues (remove entirely from deleted workflows)
   - `security-events: write` — only in Trivy SARIF upload jobs
   - `contents: write` — only in the digest bot-commit job

3. **Add `dependency-review-action` to `pr-checks.yml`.** Catches new HIGH CVEs in
   `go.mod` / `pyproject.toml` changes before merge.

4. **Add SLSA L2 provenance attestation** to the release-tag job using
   `actions/attest-build-provenance`. This upgrades from the auto-generated Build L1
   provenance to a signed, verifiable L2 attestation attached to the release.

5. **Add a `.trivyignore` file** at repo root with a comment template:
   ```
   # Format: CVE-YYYY-NNNNN
   # accepted-until: YYYY-MM-DD
   # reason: <why this is accepted / link to upstream fix>
   # affected: <which image / which layer>
   ```
   Initially empty. Reviewed quarterly.

---

## Part 4 — Update existing M6 issues

### 4A. Re-scope EPIC #873 (M6.DevAuto) and EPIC #881 (Wave 4)

**#873 (DevAuto EPIC):** Waves 0–3 are being retired by Part 1 of this spec.
Update the EPIC body to:
- Mark Waves 0–3 as superseded (not deleted — the code did run and the learnings in
  `docs/ai-learnings/` are real).
- Retain Wave 4 as the living scope: expressing orchestrator + experts as Zynax
  `agent-def` workflows on the platform itself.
- Change EPIC status to "blocked on platform readiness" — same as before, but now the
  blocker is the new CI pipeline being stable (Part 3) rather than Waves 0–3.
- Add dependency link to the new CI-hardening EPIC (Part 5B).

**#881 (Wave 4 EPIC):** No implementation change. Update body to:
- Remove the "unblocked" claim (Waves 0–3 being retired doesn't mean Wave 4 is unblocked;
  it means the near-term proxy is gone and Wave 4 is the correct remaining scope).
- Update the vision: the SPDD commands (`/milestone-orchestrate`, `/issue-deliver`) are
  the near-term automation layer; Wave 4 expresses that same logic as Zynax manifests.
- Add dependency on the new delivery-tooling EPIC (Part 5A): Wave 4 uses the generalized
  commands as its specification.

### 4B. Update story #1089 (release.yml build matrix)

Current title: `ci(infra): add event-bus + memory-service to the release.yml build matrix`
Current state: `status: ready` — it is about to be made irrelevant by Part 3.

**Update:** The retag model removes the build matrix from `release.yml` entirely. The
equivalent task is now: "add event-bus and memory-service to the `build-images` matrix
in `ci.yml`." This is structurally different (pre-merge, not post-merge).

Update the issue body to reflect the new scope, retitle to:
`ci: add event-bus + memory-service to pre-merge build-images matrix`

Add a comment explaining the dependency on the new CI-hardening EPIC (Part 5B).
Mark as `status: blocked` on the CI-hardening EPIC landing.

### 4C. Update EPIC #771 (M6.CI-E2E)

EPIC #771 covers the e2e smoke gate and engine matrix test. With the shift-left model:
- The e2e smoke gate now runs against pre-built staging images rather than building in the gate.
- Story `#1071` (engine matrix) is now about testing against two pre-built images, not
  building them in the e2e job.
- Story `#1092` (promote e2e-smoke to required) can only be done after the `build-images`
  job is required — the full pre-merge pipeline must be stable first.

Update #771 body to add the CI-hardening EPIC as a blocker for #1092.
Update #1071 body to note the e2e job will pull staging images, not build them.

### 4D. Update EPIC #1073 (Postgres migration)

The Postgres migration registers a new image in `images.yaml`. With the retag model,
that image's digest will be updated atomically on every merge (not via a separate [AUTO]
issue). Update the issue body to reference the new digest management model and remove
any language about "run make sync-images as a follow-up step."

---

## Part 5 — New M6 EPICs and SPDD canvases

### 5A. EPIC: Delivery Tooling Generalization

**Create a new GitHub issue:**

```
Title: epic(tooling): generalize delivery commands — milestone-agnostic SSoT + lifecycle commands
Labels: type: epic, area: ci, milestone: M6, priority: high
```

Body:
> Introduces `state/milestone.yaml` as the single machine-readable source of truth for
> the active milestone, replacing all hardcoded M6 references in delivery commands.
> Renames `m6-plan`, `m6-orchestrate`, `m6-issue-generate`, `resume-m6` to generic
> `milestone-plan`, `milestone-orchestrate`, `issue-deliver`, `resume-milestone`.
> Adds `milestone-close` and `milestone-new` for full milestone lifecycle management.
> Deduplicates state from README/ROADMAP/CLAUDE.md.
>
> Stories: 0A (state/milestone.yaml), 0B (command rename), 0C (doc dedup).
> SPDD canvas required before any feat code (ADR-019).
> Run `/spdd-analysis <this-issue>` → `/spdd-reasons-canvas <this-issue>` first.

### 5B. EPIC: CI/CD Pipeline Hardening

**Create a new GitHub issue:**

```
Title: epic(ci): shift-left security pipeline — pre-merge build+scan, retag model, retire advisory CI
Labels: type: epic, area: ci, milestone: M6, priority: high
```

Body:
> Restructures the CI/CD pipeline so that container images are built and security-scanned
> exactly once, pre-merge. On merge to main, images are retagged (not rebuilt), making the
> binary in production the exact binary that passed the security gate.
>
> Retires the dev-advisory.yml (Wave 0–3) and demotes post-merge-completeness to a weekly
> audit. Updates PR size exclusions. Adds ADR-027 (shift-left pipeline model).
>
> Stories: 1A (retire dev-advisory), 1B (weekly-audit), 1C (PR size), 2A (ADR-027),
>          3A (build-images pre-merge), 3B (PR image cleanup), 3C (release retag model),
>          3D (supply-chain hardening).
>
> SPDD canvas required for the feat: stories. `ci:` and `chore:` stories are SPDD-exempt.
> Run `/spdd-analysis <this-issue>` → `/spdd-reasons-canvas <this-issue>` first.

**For each EPIC, run the SPDD pipeline:**
```
/spdd-analysis <epic-issue>
/spdd-reasons-canvas <epic-issue>
/spdd-security-review docs/spdd/<slug>/canvas.md
[human reviews and sets Status: Aligned]
/spdd-generate docs/spdd/<slug>/canvas.md    ← one O-step at a time
```

---

## Part 6 — Engineering Manifesto

Create: `docs/contributing/engineering-manifesto.md`

The manifesto is a living reference document — not an ADR (mutable), not a guide
(it is normative, not instructional). It is the engineering culture of the project
expressed as laws with structural enforcement. Every principle must cite the workflow,
check, or policy that enforces it — principles without enforcement are aspirations, not laws.

### Preamble

> This document is the engineering constitution of Zynax. It applies to all contributors —
> human and automated. It is not aspirational: every principle listed here is structurally
> enforced by the CI pipeline, branch protection, or pre-merge checks. When a principle is
> not yet enforced, it is marked ⏳ with the issue that will add the enforcement.
>
> These principles are derived from DORA research, CNCF project patterns (Kubernetes, Flux,
> ArgoCD, Helm, Prometheus), and the Google DevOps Handbook. They are calibrated for a
> project that ships to production continuously, values correctness over speed, and is
> developed by a combination of human engineers and AI agents.

### Principles

**P1 — Main is production, always.**

The `main` branch is the deployment artifact. Every commit on `main` is deployable,
right now, without any stabilization period, merge window, or manual step. If you would
not be comfortable deploying it at 2am on a Sunday, it must not be on `main`.

*Enforced by:* Branch protection with all required checks (no bypass for admins —
`ADR-023`). Squash-merge only. No direct pushes.

---

**P2 — The PR is the unit of correctness.**

A PR contains everything needed to verify the change: production code, tests, updated
documentation, updated configuration, updated digests. There are no "fix CI" PRs, no
"update digest" issues generated after the fact, no "I'll add tests in the next PR."
If the change is not verifiable from the PR alone, it is not ready.

*Enforced by:* Pre-merge required checks (`ci.yml`). Digest sync is part of the merge
pipeline, not a follow-up. `canvas-freshness` check enforces SPDD alignment.

---

**P3 — Build once. Promote by tag.**

A container image is built exactly once during the PR lifecycle. It is scanned for CVEs
before the merge is allowed. On merge, it is retagged — not rebuilt. On version release,
it is retagged again. The image in production is the binary that passed the security gate.

Rebuilding after merge introduces nondeterminism: base layer updates, network fetches,
toolchain differences. "It built the same way" is not a supply-chain guarantee.

*Enforced by:* `release.yml` (retag-only job). `build-images` job in `ci.yml` (pre-merge
build + scan). `ADR-027`.

---

**P4 — Shift security left. Everything that can fail pre-merge, must.**

CVE scanning, Dockerfile linting, secret scanning, dependency review, supply-chain
attestation — all happen before merge. Post-merge security is a weekly drift audit for
new CVEs disclosed after merge, not a gate for things we could have caught earlier.

The weekly audit finding a HIGH CVE is a signal to open a properly-scoped PR. It never
triggers automated issue factories.

*Enforced by:* Trivy scan in `build-images` (required check). `gitleaks-scan` in
`pr-checks.yml` (required). `govulncheck`/`bandit`/`pip-audit` in `ci.yml` (required).
`dependency-review-action` in `pr-checks.yml`. `weekly-audit.yml` (schedule only).

---

**P5 — Small batches ship faster and safer.**

Optimal PR size is ≤200 net lines of production code. Hard limit is 900 lines. Not
because large changes are wrong in principle, but because review quality degrades with
size, bisection complexity grows quadratically, and rollback risk compounds. DORA data
consistently shows: batch size is the primary driver of deployment frequency and change
failure rate. High performers ship 46× more frequently with 5× lower failure rates.

*Enforced by:* `pr-size.yml` (hard limit: 900 lines, required check). `CLAUDE.md §PR size`.

---

**P6 — One issue. One PR. One logical change.**

Each GitHub issue maps to exactly one PR. Each PR contains exactly one logical change —
a change that can be described with a single conventional-commit subject line. Each commit
within the PR is independently revertible.

This makes `git bisect` O(log n). It makes rollback a one-command operation. It makes
review focused. It makes the changelog meaningful.

*Enforced by:* Conventional commit check in `pr-checks.yml` (required). DCO sign-off
required on every commit. Squash-merge model.

---

**P7 — No post-merge corrective actions.**

Post-merge has exactly two responsibilities:
1. Retag the verified image to its production name.
2. Commit the updated digest to `images/images.yaml` with `[skip ci]`.

That is all. No re-running tests. No re-running security scans. No issue factories.
No "completeness meshes." If a post-merge check fails, it means a pre-merge check was
missing — fix the pre-merge check, don't add more post-merge band-aids.

*Enforced by:* `release.yml` (retag only). `weekly-audit.yml` (schedule only, not a gate).
`post-merge-completeness.yml` deleted in favour of `weekly-audit.yml`.

---

**P8 — No patches. Fix the root cause.**

A "patch" is any change that works around a broken system rather than fixing it:
a `continue-on-error: true` on a flaky test, an `|| true` in a script to silence an
error, an `xfail` with no time-bounded tracking issue, a `.trivyignore` entry with no
expiry date, a `[skip ci]` on a non-automated commit.

When a patch is genuinely necessary (upstream unfixable CVE, known-flaky external
dependency), it must have: (1) a dated expiry annotation, (2) a tracking issue, and
(3) a comment explaining the root cause and the upstream fix timeline.

*Enforced by:* `actionlint` catches undefined env vars and common shell anti-patterns.
Code review. `.trivyignore` entries require `accepted-until:` annotation (checked by
a custom linter in `pr-checks.yml`).

---

**P9 — Operations are idempotent.**

Every CI job, every script, every Make target can be run twice and produce the same
result. Build jobs that push images use content-addressed digests — re-running them
produces the same digest. The retag job is a no-op if the tag already exists and
points to the same digest. The bot commit is skipped if `images/images.yaml` has no
changes.

Idempotency means re-running a failed job is always safe. It is the difference between
a pipeline that can be recovered in 5 minutes and one that requires manual cleanup.

*Enforced by:* `docker buildx imagetools create` is idempotent (overwrites the tag).
`git commit` is skipped when there are no staged changes. Make targets are designed to
be re-entrant.

---

**P10 — Operations are atomic.**

A change either completes fully or leaves the system in its previous state. There are
no partial states: no "digest partially updated", no "image pushed but not signed",
no "release created but SBOM missing."

If an operation must be atomic across multiple steps, those steps run in a single job
with `set -euo pipefail` and a cleanup trap. If a step fails, the job fails and no
subsequent step runs.

*Enforced by:* All workflow shell scripts use `set -euo pipefail`. Retag + sign + digest
commit run in a single job with no `continue-on-error`. SBOM upload failure fails the job.

---

**P11 — Actions are pinned to SHAs. Tags are not immutable.**

A GitHub Actions tag (`@v4`) is mutable. An attacker who compromises the upstream action
repository can push a new commit to that tag and execute arbitrary code in your pipeline.
Pinning to a commit SHA guarantees you run the exact code you reviewed.

Every third-party `uses:` must be pinned to a full commit SHA with the version as a
comment: `uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2`

*Enforced by:* `actionlint` in `pr-checks.yml` (required check, fails on unpinned actions).

---

**P12 — Least privilege everywhere.**

Every workflow job declares only the permissions it needs. The default at the workflow
level is `permissions: contents: read`. Write permissions are escalated per-job only
when the job genuinely needs them, with a comment explaining why.

No global `permissions: write-all`. No ambient secrets accessed by jobs that don't need
them. OIDC tokens (`id-token: write`) only in signing jobs.

*Enforced by:* `actionlint` enforces explicit `permissions:` blocks. PR review enforces
the escalation comments. `ADR-020` (zero-trust auth).

---

**P13 — Signals are binary. Advisory output is not a gate.**

A CI check either passes or fails. There is no third state. `continue-on-error: true`
on a gate is a contradiction — it is not a gate. An "advisory" check that never blocks
anything is noise that erodes trust in the entire pipeline.

The pipeline has two classes of jobs:
- **Gates:** required status checks in branch protection. Failure blocks merge. Zero
  tolerance for flakiness — fix the flakiness.
- **Audits:** scheduled weekly runs. Failure is visible as a failed workflow run.
  Not required checks. Never on PR triggers.

There is no third class. "Advisory on PR" was an experiment. It is retired.

*Enforced by:* Branch protection `required_status_checks`. `weekly-audit.yml` is
schedule-only. `dev-advisory.yml` deleted.

---

**P14 — Automated releases from tags. No release checklists.**

A release is created by pushing a version tag: `git tag -s v0.5.0`. The tag triggers
the retag job, GitHub Release creation with auto-generated notes from conventional
commits, SDK publish to PyPI, and SBOM attachment. The process completes in ~5 minutes
with no human steps after pushing the tag.

Release notes are generated from conventional commits since the last tag. The quality
of release notes is a function of commit message quality — which is enforced by P6.

*Enforced by:* `release-tag.yml` (tag trigger). `gh release create --generate-notes`.
`conventional-commit` check (required, ensures commit messages are parseable).

---

**P15 — The pipeline is documentation. Keep it simple.**

The `.github/workflows/` directory is the authoritative specification of how the project
is built, tested, and released. A new contributor should be able to understand the full
pipeline in 30 minutes. Each workflow file has one responsibility, a clear name, and a
comment header explaining what it does and why it exists.

Workflows are DRY: reusable workflows (`_test-go.yml`, `_test-python.yml`) eliminate
copy-paste. Shared scripts live in `scripts/` and are called from workflows, not
inlined. The build matrix is the same across `build-images` (pre-merge), the retag job
(post-merge), and `weekly-audit.yml` — defined once in a config that all three read.

Complexity is a cost. Every job added is a job that can fail, must be maintained, and
must be understood by contributors. Prefer removing jobs to adding them.

*Not formally enforced — enforced by code review and by the principle that a workflow
file should fit on one screen.*

---

### DORA targets

| Metric | Target | Structural enabler |
|--------|--------|--------------------|
| Deployment frequency | Multiple per day | Tag-triggered release, no merge windows |
| Lead time for changes | <1 day p50 | PR size limit, fast CI (<12 min), auto-merge on green |
| Change failure rate | <5% | Pre-merge image scan, BDD contract tests, e2e-smoke required |
| MTTR | <1 hour | Linear squash-merge history, `git bisect`, revert-as-PR |

### CNCF reference patterns

| Project | Pattern we adopt |
|---------|-----------------|
| Kubernetes | Path-aware change detection matrix; reusable workflows; OWNERS model |
| Flux | Image automation with `[skip ci]` bot digest commits; GitOps reconciliation |
| Helm | Conventional commits → auto changelog; chart-testing in CI |
| Prometheus | Multi-arch native builds (no QEMU); minimal distroless base images |
| ArgoCD | Cosign keyless OIDC signing; SBOM on every release; SLSA provenance |
| containerd | buf-based proto contract testing; strict layer boundary enforcement |

---

## Questions to ask DURING implementation

At each part boundary, ask the operator:

**During Part 0:**
- "I am about to delete `m6-plan.md`, `m6-orchestrate.md`, `m6-issue-generate.md`,
  and `resume-m6.md`. The `.claude/settings.json` skill registrations reference these
  by name. Should I update the skill names to `milestone-plan`, `milestone-orchestrate`,
  `issue-deliver`, `resume-milestone` in the same PR, or a follow-up PR?"

- "The `spdd-story.md` command applies `milestone: M6` as a label on story issues it
  creates. Should I update this to read from `state/milestone.yaml`, or leave it to the
  calling command (`issue-deliver`) to add the label after story creation?"

**During Part 1:**
- "I am about to `git rm .github/workflows/dev-advisory.yml`. The `automation/`
  directory has expert YAML configs and `scripts/invoke-llm.sh`. Should these go to
  `docs/archive/dev-advisory/` or be deleted entirely? If archived, they will still be
  in git history either way."

- "The `post-merge-image-test` job currently only runs on `workflow_run: Release`.
  After Part 3, images are retagged on every merge (not just Release tags). Should the
  weekly audit pull `:latest` from all services, or only services whose Dockerfiles
  changed in the last week?"

**During Part 3A:**
- "The pre-merge build matrix includes 7 Go services. The Python adapter images in
  `agents/adapters/*/` have separate Dockerfiles. Should they be in the same
  `build-images` matrix or a separate job with different logic?"

- "Hadolint has a DL3008 rule that flags `apt-get install` without `--no-install-recommends`.
  Your Dockerfiles may currently have such patterns. Should I fix them in this PR or create
  a follow-up issue and configure Hadolint to `ignore: [DL3008]` initially?"

**During Part 3C:**
- "The retag job triggers on `workflow_run: workflows: ["CI"] types: [completed]`. This
  means every CI run on main (including push of the `[skip ci]` digest commit) triggers
  a retag. The `[skip ci]` digest commit should not trigger CI, and does not because of
  the `[skip ci]` token. But confirm: is the retag job expected to fire on every push
  to main (including non-PR pushes like the bot commit), or only after a PR merge?"

**During Part 4:**
- "Issue #1089 is currently `status: ready` and assigned to nobody. Before I update the
  scope and mark it `status: blocked`, confirm: is there any ongoing work on this issue
  in any branch? `git branch -r | grep 1089`"

---

## Acceptance criteria

**Part 0 complete when:**
- `state/milestone.yaml` exists and `make validate-milestone-state` passes.
- `/milestone-plan` produces next commands without any M6 string in the command source.
- `m6-plan.md`, `m6-orchestrate.md`, `m6-issue-generate.md`, `resume-m6.md` are deleted.
- CLAUDE.md no longer has inline EPIC status counts; links to `state/current-milestone.md`.

**Part 1 complete when:**
- `dev-advisory.yml` is deleted; no PR receives an AI advisory comment.
- `post-merge-completeness.yml` is renamed to `weekly-audit.yml`; push/workflow_run
  triggers removed; `[AUTO]` issue creation removed.
- A test PR (trivial docs change) shows the updated size count with new exclusions.

**Part 2 complete when:**
- `docs/adr/ADR-027-shift-left-pipeline.md` exists with Status: Accepted.

**Part 3 complete when:**
- A PR that changes `services/api-gateway/Dockerfile` triggers `build-images` pre-merge.
- A PR with a test `RUN curl evil.example.com` (injected CVE) fails Trivy and blocks merge.
- Merging any PR produces: staging image retagged to `main-<sha>`, `images.yaml` updated
  with new digest in a `[skip ci]` bot commit, cosign signature verifiable via
  `cosign verify ghcr.io/zynax-io/zynax/api-gateway:latest`.
- `release.yml` contains no `docker/build-push-action` step.
- A version tag push creates a GitHub Release with auto-generated notes.

**Part 4 complete when:**
- #873 body reflects Wave 0–3 retirement; Wave 4 scope and blockers updated.
- #1089 title and scope reflect pre-merge build-images task; `status: blocked` on Part 3.
- #771 body notes e2e gate dependency on `build-images` being required.

**Part 5 complete when:**
- Two new GitHub EPICs exist with `type: epic, milestone: M6` labels.
- SPDD canvas exists for each EPIC at `docs/spdd/<issue>-<slug>/canvas.md`.
- `/spdd-security-review` returns PASS on both canvases.

**Part 6 complete when:**
- `docs/contributing/engineering-manifesto.md` exists with all 15 principles.
- `CONTRIBUTING.md` links to it in the "How we work" section.
- `CLAUDE.md §Architecture Invariants` links to it.

---

*Document generated: 2026-06-10*
*Assisted-by: Claude/claude-sonnet-4-6*
