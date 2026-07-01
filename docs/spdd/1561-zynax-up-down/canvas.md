# REASONS Canvas — `zynax up` / `zynax down`: make-free kind lifecycle CLI verbs

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/lib:spdd-security-review <path>` before committing.

**Issue:** #1561 (epic M7.V) — stories #1562 (O1), #1563 (O2), #1564 (O3)
**Author:** Oscar Gómez Manresa
**Date:** 2026-07-01
**Status:** Aligned
**Aligned:** 2026-07-01 (maintainer-authorized; grounded in the ADR-041 kind runtime + the existing `scripts/e2e/cluster-up.sh` harness and the `cmd/zynax/cmd/doctor.go` shell-out precedent).

---

## R — Requirements

> Problem statement: what breaks or is missing without this feature?

- The installed `zynax` binary can talk to a running platform but **cannot manage its lifecycle**. Bringing the platform up locally requires `make demo` / `make kind-up` (ADR-041), which need `make` **and** a repo-root working directory.
- ADR-041 Decision #3 fronts the lifecycle with `make kind-up` / `make kind-down` wrapping [scripts/e2e/cluster-up.sh](../../../scripts/e2e/cluster-up.sh) — but there is no CLI-native, `make`-free, run-from-anywhere entry point.
- The **engine-portability wedge** ("write once, run on Temporal or Argo") is only reachable today via a Makefile variable (`ENGINE=argo make demo`), not a first-class, discoverable CLI flag.

> Definition of done: the observable outcomes that confirm delivery.

- `zynax up` brings the full platform up on a local kind cluster to the same healthy state as `make kind-up` (verified: `zynax doctor` all ✓ and `http://localhost:8080/healthz` → 200), from **any** directory inside a checkout.
- `zynax up --engine temporal|argo` and `--profile full|lite` select engine/profile; a workflow applied afterward reaches a terminal state on the chosen engine.
- `zynax down` deletes the cluster and is idempotent.
- Run outside a checkout, both verbs exit non-zero with a clone-pointing message and never exec the script.
- Unit tests assert every flag→env mapping and repo-root resolution path with **no real cluster**.

---

## E — Entities

> Tier 1 abstractions only.

```
ClusterLifecycleVerb (up | down) ──resolves──> RepoRoot
RepoRoot ──locates──> BringUpScript (scripts/e2e/cluster-up.sh)
RepoRoot ──locates──> TeardownScript (scripts/e2e/cluster-down.sh)
ClusterLifecycleVerb ──maps flags to──> ScriptEnv (PROFILE, E2E_ENGINE, KIND_LOAD_IMAGES, CLUSTER_NAME, NAMESPACE)
ClusterRunner ──streams stdout/stderr of──> Script   (injectable; recording fake in tests)
RepoRootResolver ──flag → env → walk-up sentinel──> RepoRoot | errNoRepoRoot
```

- **RepoRoot:** the Zynax checkout directory that holds the bring-up scripts. Discovered, never assumed.
- **ClusterRunner:** an injected function that execs a script with a streamed stdio + an env overlay — the seam that makes the verbs unit-testable without kind/helm/kubectl.
- **ScriptEnv:** the env-var contract the scripts already own (`${VAR:-default}` semantics); the CLI is a 1:1 flag→env translator, not a re-implementation.

---

## A — Approach

> Solution strategy. What we WILL and WON'T do; ADRs that govern.

**We will:**
- Add `zynax up` and `zynax down` as Cobra verbs in `cmd/zynax/cmd/`, grouped in the existing `beginnerGroupID` alongside `doctor`.
- **Wrap the existing, idempotent, CI-proven scripts** (`scripts/e2e/cluster-up.sh` / `cluster-down.sh`) via an **injectable, live-streaming runner** — following the shell-out precedent in [cmd/zynax/cmd/doctor.go](../../../cmd/zynax/cmd/doctor.go) (injectable `commander`) and the streaming exec in [cmd/zynax/cmd/mcp.go](../../../cmd/zynax/cmd/mcp.go). Stream (don't buffer) because bring-up runs for minutes.
- Resolve the repo root three ways — `--repo-root` flag → `ZYNAX_REPO_ROOT` env → walk up from cwd for the sentinel `scripts/e2e/cluster-up.sh` — and fail with a friendly clone-pointing error outside a checkout.
- Map flags 1:1 to the scripts' env contract; build child env as `os.Environ()` + overrides appended last so power-user vars still pass through. Let the scripts own `profile`/`engine` validation (no drift).
- Record the CLI-native entry point as an **amendment to ADR-041** (extends Decision #3; runtime decision unchanged).

**We will NOT:**
- Auto-resolve a workflow's `capability:` refs to specific adapters and provision only those (no capability→adapter registry or adapter Helm charts exist) — deferred; overlaps **M-dx #1359**. Workflow-specific adapters keep coming up as `kind: AgentDef` inside a `kind: Scenario` that today's `zynax apply` already provisions.
- Change `zynax apply` at all.
- Add `zynax up --run <workflow>` (bring up **and** run) — follow-on with its own canvas.
- Embed kind configs / Helm charts in the binary or fetch a release bundle — repo checkout is assumed (ADR-041 "clone → one command").
- Add gRPC/proto types to the CLI or route through the api-gateway — `up`/`down` shell out to kind/helm/kubectl only.

**Positioning fit (user-facing):** `zynax up --engine temporal|argo` brings up the **same** platform on either engine with a single flag — the engine-portability wedge as one CLI command. All new copy (`--help`, the pre-run banner, docs) leads with "write once, run on Temporal or Argo", never the generic "control plane" framing. See [docs/product/positioning.md](../../product/positioning.md).

**Governing ADRs:** ADR-041 (kind-native unified runtime — amended here), ADR-015 (pluggable workflow engines — the `--engine` flag), ADR-019 (REASONS Canvas before feat code), ADR-017 (`GOWORK=off`), ADR-001 (CLI has no cross-layer/gRPC coupling).

---

## S — Structure (first S)

> System placement. Which packages/files does this feature touch?

```
cmd/zynax/cmd/
├── up.go        ← upCmd + flags; upDeps, clusterRunner, streamRunner, runUp
├── down.go      ← downCmd + --cluster-name/--repo-root; runDown (reuses up.go types)
├── reporoot.go  ← findRepoRoot, isRepoRoot, sentinel/env/flag consts, errNoRepoRoot (shared)
└── up_test.go   ← recording fake runner; flag→env + repo-root resolution tests
```

- Standalone module `github.com/zynax-io/zynax/cmd/zynax` (not in go.work; `GOWORK=off`). Deps unchanged (cobra + stdlib `os/exec`).
- Reuses `beginnerGroupID` and the `"zynax"` namespace default (`doctorNamespace`) from existing files so they never drift.
- No gRPC contracts. No api-gateway endpoints. No proto. No `.feature` file (CLI is not a gRPC boundary owner — cmd/zynax/AGENTS.md).
- Wrapped scripts (not modified): `scripts/e2e/cluster-up.sh`, `scripts/e2e/cluster-down.sh`.

---

## O — Operations

> Ordered, testable steps. Each = one reviewable PR, mapped to an issue.

1. **#1562 (O1)** — `reporoot.go` (repo-root discovery: flag → env → walk-up sentinel) + the injectable `streamRunner` + `zynax down` wrapping `cluster-down.sh` (forwards `CLUSTER_NAME`). Unit tests with a recording runner assert env forwarding + all repo-root paths (incl. the not-found error, runner never called). *Smallest vertical slice; lands the shared plumbing.*
2. **#1563 (O2)** — `zynax up` wrapping `cluster-up.sh`: flag→env mapping (`--profile`→`PROFILE`, `--engine`→`E2E_ENGINE`, `--no-load-images` omits `KIND_LOAD_IMAGES` (default on), `--cluster-name`→`CLUSTER_NAME`, `--namespace`→`NAMESPACE`); reuse the O1 runner + repo-root finder; wedge-first banner. Tests assert every mapping (defaults incl. `KIND_LOAD_IMAGES=1`; lite+argo; `--no-load-images`; overrides; script path).
3. **#1564 (O3)** — docs + ADR alignment: `cmd/zynax/AGENTS.md`, `docs/local-dev-kind.md` (row beside `make kind-up`/`kind-down`), `docs/local-dev.md`, `README.md` golden path; **ADR-041 amendment** recording the CLI-native entry points.

---

## N — Norms

> Cross-cutting standards (root + layer AGENTS.md, docs/patterns).

- **Commit hygiene:** `Signed-off-by:` (DCO) required; `Assisted-by: Claude/<model>` for AI attribution — never `Co-Authored-By` for AI.
- **Conventional commits / PR titles:** feat/fix/refactor/docs/test/ci/chore; scope maps to directory (`cli`); one logical change per commit, one PR per issue.
- **CLI module:** `GOWORK=off` for every `go build`/`go test` (ADR-017); HTTP-REST only, no gRPC/proto types; no imports from `services/*/internal/`; no `os.Exit` outside `Execute()` (a non-nil `RunE` gives Cobra exit 1 — no custom code needed); no `panic`.
- **Shell-out safety:** the exec'd path is a fixed repo-relative script under a verified checkout root, never prompt/tool input — carry a justified `#nosec G204`. Run the **local** golangci-lint (v2.x, stricter than `make lint` on gosec/goconst) before committing.
- **PR size:** ≤ 200 ideal / 201–400 acceptable (tests count; docs/AGENTS.md/README excluded).

---

## S — Safeguards (second S)

> Things that MUST NEVER happen in this feature.

### Context Security (complete before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics (only `localhost:8080`, public script paths, `kind`/`helm`/`kubectl` tool names).
- [x] No PII: only the existing public author attribution; no email addresses.
- [x] No prompt injection: no instruction-like phrasing that overrides AGENTS.md rules.
- [x] All entities in E are public-safe abstractions.
- [x] `/lib:spdd-security-review` self-review PASS (2026-07-01 — Tier 1; no Tier 2 / injection / abstraction-leak / authority findings).

### Feature Safeguards

- **Never** exec a user-supplied binary or pass user/tool input as the command name — only the fixed `scripts/e2e/cluster-{up,down}.sh` path, resolved under a verified checkout root.
- **Never** assume a working directory: always resolve the repo root (flag → env → sentinel walk) and fail loudly (clone-pointing error) rather than silently exec the wrong thing.
- **Never** hardcode engine names in logic — `--engine` is passed through to the script's `E2E_ENGINE` (ADR-015: engines stay pluggable).
- **Never** add gRPC/proto types to the CLI or call the api-gateway from `up`/`down` (ADR-001 / cmd/zynax AGENTS.md).
- **Never** let the kind path diverge from the production Helm charts — `up` wraps the *same* `scripts/e2e/cluster-up.sh` used by CI, so local mirrors prod (ADR-041).
- **Never** target Docker Compose with new work (ADR-041: Compose deprecated, no new investment).
- **Never** commit implementation code for this epic before this Canvas is Aligned (ADR-019).
