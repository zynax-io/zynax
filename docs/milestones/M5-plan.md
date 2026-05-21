<!-- SPDX-License-Identifier: Apache-2.0 -->

# M5 — Adapter Library Execution Plan

**Milestone:** Adapter Library (M5) · v0.4.0
**GitHub Milestone:** [Adapter Library (M5)](https://github.com/zynax-io/zynax/milestone/5)
**Parent epic:** [#377](https://github.com/zynax-io/zynax/issues/377)
**Status:** In Progress
**Last updated:** 2026-05-22 (rev 41 — distroless regression documented; #655 #656 filed; compose override added)

---

## Objective

**Ship v0.4.0: a fully wired, end-to-end capable Adapter Library.**

M5 is done when ALL of the following are true:

1. `make run-local && zynax apply spec/workflows/examples/code-review.yaml` produces real,
   observable state transitions through at least one capability dispatch (mock agent acceptable).
2. v0.4.0 tag exists on GitHub with downloadable CLI binaries and GHCR service images.
3. All five adapters (http ✅ + git + ci + llm + langgraph) have implementations merged.
4. The Python SDK `Agent` base class is implemented (Option A, #474).
5. cel-go replaces the bespoke guard evaluator (#476/#538).
6. SECURITY.md matches what is actually shipped.
7. CI pipeline runs in < 10 minutes per PR (M5.F target).

Epic #377 (Adapter Library) and all its child epics (#381–#384) must be closed.

---

## Critical Path

```
Fix CI first (#542 M5.F) ──► Complete M5.C dispatch (#460) ──► Complete adapters (#377)
         │                            │                                    │
   CI < 10 min/PR              E2E green demo               git+ci+llm+langgraph merged
   v0.4.0 tag (#558)           agent-registry live           Python SDK (#474) done
   GHCR images public          compose wired (#481)          cel-go (#476) done
```

**Why CI first:** Every subsequent PR in M5 takes 25+ minutes to merge today. Fixing
concurrency, change-detection, and the release pipeline (#545, #546, #547) cuts the feedback
loop by 3–4× before any adapter code is written. This accelerates everything else.

**Merge strategy:** No merge queue. Auto-merge (`allow_auto_merge: true`) with `strict: true`
branch protection — the branch must be up-to-date with main before merging. Rebasing is
the developer's responsibility; auto-merge fires automatically once all checks pass.

---

## Track Overview

| # | Track | Epic | Status | Priority |
|---|-------|------|--------|----------|
| **M5.F** | CI/CD Performance Sprint | [#542](https://github.com/zynax-io/zynax/issues/542) | 🔴 **Do first** | **P0** |
| **M5.F.R** | Release Pipeline | [#556](https://github.com/zynax-io/zynax/issues/556) | 🔴 **Do first** | **P0** |
| **M5.C** | Capability Dispatch E2E | [#460](https://github.com/zynax-io/zynax/issues/460) | 🟡 In Progress | **P0** |
| **M5.B** | Engine Correctness | [#459](https://github.com/zynax-io/zynax/issues/459) | 🟡 In Progress (3/4 done) | **P1** |
| **M5.A** | Truth Pass | [#458](https://github.com/zynax-io/zynax/issues/458) | 🟡 In Progress (2/3 done) | **P1** |
| **Adapters** | Adapter Library | [#377](https://github.com/zynax-io/zynax/issues/377) | 🟡 In Progress (1/5 done) | **P2** |
| **M5.D** | Security Baseline | [#461](https://github.com/zynax-io/zynax/issues/461) | ✅ Complete | — |
| **M5.E** | DX Polish | [#462](https://github.com/zynax-io/zynax/issues/462) | ✅ Complete | — |
| **Tooling** | Containerized Make | [#442](https://github.com/zynax-io/zynax/issues/442) | ✅ Complete | — |

---

## Resumption Guide — "What do I pick next?"

When resuming without context, work through this ordered list. Each group is a
dependency-ordered batch; within a group, any issue may be started.

### BATCH 0 — Unblock CI (P0 · Do immediately, in parallel where possible)

These issues have zero code dependencies. They are admin + YAML changes that cut
PR cycle time from 25 min → 7 min before any code work begins.

| Issue | Title | Size | Why first |
|-------|-------|------|-----------|
| [#547](https://github.com/zynax-io/zynax/issues/547) | Remove `test-integration` from required status checks | XS | ✅ Done |
| [#544](https://github.com/zynax-io/zynax/issues/544) | Enable GitHub Merge Queue + remove `strict: true` | XS | ✅ Done (superseded — merge queue removed; strict: true + allow_auto_merge enabled via API — see #589) |
| [#548](https://github.com/zynax-io/zynax/issues/548) | Enable `allow_auto_merge` on repository | XS | ✅ Done via API |
| [#545](https://github.com/zynax-io/zynax/issues/545) | Fix CI concurrency — cancel stale runs per branch | XS | ✅ Done |
| [#589](https://github.com/zynax-io/zynax/issues/589) | Remove `merge_group` trigger from all workflow files | XS | ✅ Done |
| [#546](https://github.com/zynax-io/zynax/issues/546) | Remove push-to-main forced-true in change detection | S | ✅ Done |
| [#557](https://github.com/zynax-io/zynax/issues/557) | Fix release race condition — unified release workflow | M | ✅ Done |
| [#558](https://github.com/zynax-io/zynax/issues/558) | Cut v0.4.0 — first versioned release tag | XS | ✅ Done (CHANGELOG promoted; tag push pending user action) |

**Engineer profile:** DevOps / GitHub Actions specialist. No Go/Python knowledge required.
Edit `.github/workflows/` YAML and GitHub repository settings only.

### BATCH 1 — Complete Release Pipeline (P0 · After #557+#558)

| Issue | Title | Size | Dependency |
|-------|-------|------|------------|
| [#559](https://github.com/zynax-io/zynax/issues/559) | Add task-broker to service-release matrix | XS | ✅ Done (delivered in #557) |
| [#560](https://github.com/zynax-io/zynax/issues/560) | Add http-adapter image to release pipeline | S | ✅ Done |
| [#561](https://github.com/zynax-io/zynax/issues/561) | Push service/adapter images to GHCR on every main merge | S | ✅ Done |
| [#601](https://github.com/zynax-io/zynax/issues/601) | Fix Go builder base image to 1.26.3-alpine in service Dockerfiles | XS | ✅ Done |
| [#562](https://github.com/zynax-io/zynax/issues/562) | Make GHCR service/adapter images publicly readable | XS | ✅ Done — 5 service/adapter images public |
| (admin) | Confirm zynax/tools published + set public | — | ✅ Done — tools-image.yml succeeded 2026-05-20; package set public |
| [#563](https://github.com/zynax-io/zynax/issues/563) | Deduplicate tools image — remove tools-publish.yml + delete zynax-tools | XS | ✅ Done |
| [#566](https://github.com/zynax-io/zynax/issues/566) | README packages section with GHCR image pull commands | S | ✅ Done |

**Engineer profile:** DevOps / GitHub Actions specialist.

### BATCH 2 — Engine Correctness (P1 · M5.B, independent of CI sprint)

These can run in parallel with BATCH 0/1.

| Issue | Title | Size | Dependency | Why |
|-------|-------|------|------------|-----|
| [#538](https://github.com/zynax-io/zynax/issues/538) | Integrate cel-go as guard evaluator | M | None | ✅ Done |
| [#539](https://github.com/zynax-io/zynax/issues/539) | Guard evaluator test suite + fuzz seed | S | After #538 | ✅ Done |
| [#540](https://github.com/zynax-io/zynax/issues/540) | Remove CEL misrepresentation from AGENTS.md | XS | After #538+#539 | ✅ Done |

**Engineer profile:** Go engineer familiar with cel-go library. Read ADR-014 and
`services/engine-adapter/internal/domain/interpreter.go` first. The bespoke
`evalGuard()` function (~80 lines) must be replaced with `cel-go` and made fail-closed
(return false, not true, for unrecognized expressions).

### BATCH 3 — Truth Pass completion (P1 · M5.A)

| Issue | Title | Size | Dependency |
|-------|-------|------|------------|
| [#535](https://github.com/zynax-io/zynax/issues/535) | Implement Agent base class (Python SDK) | M | None | ✅ Done |
| [#536](https://github.com/zynax-io/zynax/issues/536) | Python SDK unit tests (≥ 85% coverage) | S | After #535 | ✅ Done |
| [#537](https://github.com/zynax-io/zynax/issues/537) | Docs update — README, ARCHITECTURE.md, AGENTS.md | XS | After #535+#536 | ✅ Done |

**Engineer profile:** Python engineer with gRPC experience. Read `docs/spdd/474-python-sdk/canvas.md`
for the full design. Option A (minimal `Agent` base class, no framework lock-in) is the
chosen approach. All three child issues are complete.

### BATCH 4 — Capability Dispatch E2E (P0 · M5.C, hardest dependency chain)

These must be done strictly in order. Each step is blocked by the previous.

| Issue | Title | Size | Dependency | Engineer profile |
|-------|-------|------|------------|-----------------|
| [#530](https://github.com/zynax-io/zynax/issues/530) | Update task-broker AGENTS.md | XS | None (ready) | ✅ Done |
| [#531](https://github.com/zynax-io/zynax/issues/531) | Align task-broker BDD + godog steps | S | None (ready) | ✅ Done |
| [#532](https://github.com/zynax-io/zynax/issues/532) | Handler unit tests for all 5 gRPC methods | S | None (ready) | ✅ Done — api 84.9% |
| [#526](https://github.com/zynax-io/zynax/issues/526) | Trim agent-registry BDD to proto scope | XS | None (ready) | ✅ Done |
| [#527](https://github.com/zynax-io/zynax/issues/527) | agent-registry domain layer | M | After #526 ✅ | ✅ Done |
| [#528](https://github.com/zynax-io/zynax/issues/528) | agent-registry gRPC wiring + cmd + go.work | M | After #527 | ✅ Done |
| [#481](https://github.com/zynax-io/zynax/issues/481) | Add task-broker + agent-registry to docker-compose | S | After #528 | ✅ Done |

**E2E exit criterion:** `make run-local && zynax apply spec/workflows/examples/code-review.yaml`
must produce observable state transitions and at least one capability dispatch with a real
(mock-data) agent response. No green E2E = M5.C not done.

**Engineer profile for #527:** Go engineer with DDD experience. The agent-registry domain must
implement `AgentRepository` port (in-memory backing for M5; Postgres in M6), `AgentRegistryService`
application service with round-robin health tracking, and heartbeat timeout logic (mark unhealthy
after 2 min without ping). Reference: `services/task-broker/internal/domain/` for the pattern.

### BATCH 5 — CI DX improvements (P0 first pair · then P1 · M5.F Group B/C/E, after BATCH 0)

**#551 ✅ done. Do #552 next.** Once #552 merges, all jobs run inside the pre-baked Alpine
ci-runner container and no CI step downloads packages from the internet (other than
code-level dependency installs such as `go mod download` or `uv sync`).

**Self-contained requirement for #551:** `Dockerfile.ci-runner` must bake in every tool the CI
pipeline calls — Go 1.26.3, golangci-lint, govulncheck, godog, mockery, buf,
protoc-gen-go, protoc-gen-go-grpc, Python + uv, ruff, mypy, bandit, pip-audit, pytest,
gitleaks, wget, and the `zynax-ci` binary. No `apt-get install`, `go install`, or `pip install`
of tooling at run time. The image is rebuilt and published to
`ghcr.io/zynax-io/zynax/ci-runner:latest` on every change to its Dockerfile (same pattern as
`tools-image.yml`). Reference: `infra/docker/Dockerfile.tools` for the current tool list.

| Issue | Title | Size | Dependency | Priority |
|-------|-------|------|------------|----------|
| [#551](https://github.com/zynax-io/zynax/issues/551) | Create Dockerfile.ci-runner — self-contained Alpine image | S | ✅ Done | — |
| [#552](https://github.com/zynax-io/zynax/issues/552) | Switch all GH Actions jobs to ci-runner container mode | M | ✅ Done | **P0** |
| [#554](https://github.com/zynax-io/zynax/issues/554) | Force-full-pipeline trigger (dispatch, label, `[full-ci]`) | S | After #552 ✅ | ✅ Done |
| [#549](https://github.com/zynax-io/zynax/issues/549) | Extend changes job per-service module granularity | M | After #552 | P1 |
| [#550](https://github.com/zynax-io/zynax/issues/550) | Scope govulncheck to changed services only | M | After #549 | P1 |
| [#555](https://github.com/zynax-io/zynax/issues/555) | DRY/KISS refactor — reusable workflows, composite actions | L | After #552 | P2 |
| [#563](https://github.com/zynax-io/zynax/issues/563) | Deduplicate tools image — remove tools-publish.yml | XS | ✅ Done | — |
| [#564](https://github.com/zynax-io/zynax/issues/564) | Pin action digests + add linux/arm64 to zynax-ci | XS | After #552 | P2 |
| [#565](https://github.com/zynax-io/zynax/issues/565) | Add trivy container scan gate before GHCR push | S | After #552 | P2 |
| [#641](https://github.com/zynax-io/zynax/issues/641) | Per-service change detection for image builds in release.yml | M | After #552 | ✅ Done |
| [#642](https://github.com/zynax-io/zynax/issues/642) | Switch service Dockerfiles to distroless/static:nonroot + `-ldflags "-s -w"` | S | None | ✅ Done |

**Notes on #641 and #642:**
- **#641** adds a `changes` job to `release.yml` with per-service path filters (including `protos/generated/go/` as a shared dep), a `resolve-matrix` job that emits only the services that changed, a weekly scheduled rebuild, and a `rebuild_all` workflow_dispatch input. Version tag pushes always build all images unconditionally. See issue body for the `fromJson` matrix pattern. Savings: ~80% fewer image builds on non-release main pushes.
- **#642** replaces `alpine:3.20/3.21` runtime stages with `gcr.io/distroless/static:nonroot` and adds `-ldflags "-s -w"` to the builder `go build` commands in all 5 service Dockerfiles. No code changes needed — fully static `CGO_ENABLED=0` binaries are drop-in compatible. Removes `adduser`/`addgroup`/`USER zynax` (distroless ships UID 65532). Estimated savings: ≥40% compressed image size per service. `tools` and `ci-runner` images remain Alpine.
- Both can be done in the same session; #642 (#S) before #641 (#M) since it has no dependency.

**Engineer profile:** DevOps / GitHub Actions specialist with Docker/Alpine experience.

### BATCH 5 — Addendum: Distroless healthcheck regression (P0 · fix before anything else)

> **⚠️ Regression introduced by #642 / PR #653 (merged 2026-05-21)**
>
> `docker-compose.yml` healthchecks use `wget` and `nc` which are not present in
> `gcr.io/distroless/static:nonroot`. Both `make run-local` and `docker compose up --no-build`
> are broken — api-gateway never starts because the cascade of `service_healthy` dependencies
> fails. A temporary override (`infra/docker-compose/docker-compose.override.yml`) exists as a
> workaround until #655 is merged.

| Issue | Title | Size | Dependency |
|-------|-------|------|------------|
| [#655](https://github.com/zynax-io/zynax/issues/655) | Add static healthcheck binary to distroless Dockerfiles + fix compose | M | **P0 — do first** |
| [#656](https://github.com/zynax-io/zynax/issues/656) | Implement gRPC Health Checking Protocol in all platform services | L | After #655 · M6 prep |

**Temporary workaround** (until #655 merges):
```bash
docker compose \
  -f infra/docker-compose/docker-compose.yml \
  -f infra/docker-compose/docker-compose.override.yml \
  up -d --no-build
```

---

#### Architecture decision: how to add health probes to distroless images

Docker `HEALTHCHECK` instructions execute **inside the container**. `distroless/static` has no
shell (`sh`/`bash`), no POSIX utilities (`wget`, `nc`, `curl`). This section documents the
options evaluated, their trade-offs, and the chosen approach.

##### Option A — Custom static Go health-probe binary ✅ CHOSEN FOR #655

Build a minimal Go binary (`tools/healthcheck/`) in the existing builder stage. No new
Go dependencies — uses `net/http` and `net` from the stdlib. Binary is ~3–4 MB compressed.
Handles two URL schemes:

```
/healthcheck http://localhost:8080/healthz   → HTTP GET; exits 0 if 2xx
/healthcheck tcp://localhost:50052           → TCP dial; exits 0 if port accepts
```

**Pros:**
- Zero external dependencies — same Go toolchain, same builder stage, no new pinned SHAs
- ~3 MB binary (vs ~13 MB for grpc-health-probe)
- Works for all 5 service types (HTTP `/healthz` + bare TCP gRPC) without application code changes
- 100% auditable — < 80 lines of stdlib Go
- Handles both `docker-compose` and future `kubectl exec` debugging

**Cons:**
- Not the CNCF-standard approach (grpc-health-probe is more widely recognised in the ecosystem)
- We own the ~80-line binary (very low maintenance burden)
- TCP probe verifies port is open, not that gRPC service is _serving_ — acceptable for M5,
  superseded by Option D in M6 when services implement gRPC Health Checking Protocol

**Trade-off verdict:** Correct for M5. Fastest path to unblocking `make run-local` without
requiring application code changes or external binaries in the build.

##### Option B — grpc-health-probe (CNCF standard, v0.4.50 as of May 2026)

Download the pre-built static binary (~13 MB) from `ghcr.io/grpc-ecosystem/grpc-health-probe`.
Requires services to implement `grpc.health.v1.Health/Check`.

**Pros:** Battle-tested, multi-arch, maintained by grpc-ecosystem (CNCF), supports TLS/mTLS.
Standard tooling for CNCF-style projects.

**Cons:**
- +13 MB per image (vs +3 MB for custom binary)
- Requires `grpc_health_v1.RegisterHealthServer` in all 5 services — **not implemented today**
- External binary to download in `docker build` — fragile (network dependency, SHA rotation)
- Still needs a separate HTTP probe for api-gateway, workflow-compiler, engine-adapter `/healthz`
- Blocks on #656 (application code changes) before it can be used

**Trade-off verdict:** Right for M6+ once #656 lands. Not suitable for the immediate regression
fix because it requires application-level changes across all services.

##### Option C — Revert services to Alpine for local dev

Use `ARG BASE=gcr.io/distroless/static:nonroot` with `--build-arg BASE=alpine:3.21` locally.

**Pros:** No Dockerfile complexity; wget and nc are available in Alpine.

**Cons:**
- Production (GHCR) and local dev run **different runtimes** — bugs reproducible in only one
  environment; defeats the purpose of distroless. Local-prod parity is non-negotiable.

**Trade-off verdict:** Rejected.

##### Option D — Kubernetes-native gRPC probes (K8s 1.24+) · Deferred to #656

Implement `grpc.health.v1` in each service; Kubernetes uses `livenessProbe.grpc` natively.
No binary in the image, no exec probe. Correct long-term production approach.

**Cons:** Does nothing for `docker-compose` (K8s probes don't run inside containers).
Tracked by #656, deferred to M6+.

##### Option E — Disable healthchecks + restart: on-failure (current workaround)

No code changes; compose retries crashed containers.

**Cons:** No health signal; startup ordering is non-deterministic; poor developer experience.
Workaround only — `docker-compose.override.yml` is committed for this purpose.

---

#### Implementation plan for #655

```
tools/healthcheck/
├── go.mod    (module github.com/zynax-io/zynax/tools/healthcheck; go 1.26; no deps)
└── main.go   (~70 lines; CGO_ENABLED=0; HTTP GET + TCP dial; 5s timeout)
```

**Each Dockerfile change (6 files):**

In builder stage, after the service binary is built:
```dockerfile
COPY tools/healthcheck/ ./tools/healthcheck/
RUN cd tools/healthcheck && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /healthcheck .
```

In runtime stage, before ENTRYPOINT:
```dockerfile
COPY --from=builder /healthcheck /healthcheck
```

**docker-compose.yml healthcheck replacement (per service):**

| Service | Before | After |
|---------|--------|-------|
| agent-registry | `nc -z localhost 50052` | `CMD ["/healthcheck", "tcp://localhost:50052"]` |
| task-broker | `nc -z localhost 50053` | `CMD ["/healthcheck", "tcp://localhost:50053"]` |
| workflow-compiler | `wget .../healthz (9094)` | `CMD ["/healthcheck", "http://localhost:9094/healthz"]` |
| engine-adapter | `wget .../healthz (9095)` | `CMD ["/healthcheck", "http://localhost:9095/healthz"]` |
| api-gateway | `wget .../healthz (8080)` | `CMD ["/healthcheck", "http://localhost:8080/healthz"]` |

All use exec form (`CMD`, not `CMD-SHELL`) — no shell required.

---

#### Future path: #656 (M6)

Once #656 lands (gRPC Health Checking Protocol in all services), the gRPC service
healthchecks (agent-registry, task-broker) can optionally switch to `grpc-health-probe`
for richer semantic checking. The custom HTTP binary is retained for HTTP services.
In M6, Kubernetes Helm chart probes will use `livenessProbe.grpc` natively — no binary needed.

**Engineer profile for #655:** Go engineer. Read `agents/adapters/http/` as a reference
Go module and `services/api-gateway/Dockerfile` for the builder pattern.

**Engineer profile for #656:** Go engineer familiar with gRPC interceptors. Read
`google.golang.org/grpc/health/grpc_health_v1` — the implementation is ~5 lines per service.

---

### BATCH 6 — Security Hardening (P1 · independent, can be done in any order)

These issues address the security gaps identified in the 2026-05-20 principal architect review
(`docs/reviews/04-architecture-gaps.md`). All are independent — no dependency between them.

| Issue | Title | Size | Why |
|-------|-------|------|-----|
| [#567](https://github.com/zynax-io/zynax/issues/567) | Bearer token constant-time compare | XS | ✅ Done — timing-attack exposure in `auth.go` (G1) |
| [#568](https://github.com/zynax-io/zynax/issues/568) | ReadHeaderTimeout + MaxBytesReader on HTTP server | XS | ✅ Done — slow-read DoS vector (G2) |
| [#622](https://github.com/zynax-io/zynax/issues/622) | Add `context.WithTimeout` to all outgoing gRPC calls | S | Cascading hang risk across all services (NEW-1 from review) |
| [#623](https://github.com/zynax-io/zynax/issues/623) | Refuse to start without `ZYNAX_GW_API_KEY` in production | XS | Silent auth bypass on misconfiguration (NEW-4 from review) |

**Engineer profile:** Go engineer. All four changes are self-contained. Start with #567 and #568
(smallest), then #623 (startup guard), then #622 (gRPC deadlines — touches 4 services).
Reference: `docs/engineering/best-practices/go.md` for `crypto/subtle.ConstantTimeCompare`
and `ReadHeaderTimeout` patterns.

---

### BATCH 7 — Adapter implementations (P2 · #377, after M5.C complete)

These can start once compose wiring (#481) is green — adapters need a working registry to
register against.

#### git-adapter (#381)
| Issue | Step | Title | Status |
|-------|------|-------|--------|
| [#400](https://github.com/zynax-io/zynax/issues/400) | O2 | Go module scaffold + config layer | ⬜ Open |
| [#401](https://github.com/zynax-io/zynax/issues/401) | O3 | Capability handler (open_pr, request_review, get_diff) | ⬜ Open (blocked on #400) |
| [#402](https://github.com/zynax-io/zynax/issues/402) | O4 | Registry client + bootstrap | ⬜ Open (blocked on #401) |
| [#403](https://github.com/zynax-io/zynax/issues/403) | O5 | Dockerfile, docker-compose, AGENTS.md | ⬜ Open (blocked on #402) |

**Engineer profile:** Go engineer with GitHub REST API experience. Read
`docs/spdd/381-git-adapter/canvas.md` and `agents/adapters/http/` as reference implementation.
Capabilities: `open_pr` (POST /repos/{owner}/{repo}/pulls), `request_review`
(POST /repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers), `get_diff`
(GET /repos/{owner}/{repo}/pulls/{pull_number}/files). All three must be SSRF-safe.

#### ci-adapter (#382)
| Issue | Step | Title | Status |
|-------|------|-------|--------|
| [#405](https://github.com/zynax-io/zynax/issues/405) | O2 | Go module scaffold + config layer | ⬜ Open |
| [#406](https://github.com/zynax-io/zynax/issues/406) | O3 | CIHandler + PollLoop (trigger_workflow, get_run_status) | ⬜ Open (blocked on #405) |
| [#407](https://github.com/zynax-io/zynax/issues/407) | O4 | Registry client + bootstrap | ⬜ Open (blocked on #406) |
| [#408](https://github.com/zynax-io/zynax/issues/408) | O5 | Dockerfile, docker-compose, AGENTS.md | ⬜ Open (blocked on #407) |

**Engineer profile:** Go engineer with GitHub Actions API experience. Capabilities:
`trigger_workflow` (POST /repos/{owner}/{repo}/actions/workflows/{id}/dispatches) +
`get_run_status` (GET /repos/{owner}/{repo}/actions/runs/{run_id}). PollLoop must have
exponential backoff and a configurable max-poll-duration.

#### llm-adapter (#383) — Python
| Issue | Step | Title | Status |
|-------|------|-------|--------|
| [#410](https://github.com/zynax-io/zynax/issues/410) | O2 | Module scaffold + ProviderConfig | ⬜ Open |
| [#411](https://github.com/zynax-io/zynax/issues/411) | O3 | Provider handlers (OpenAI, Bedrock, Ollama) | ⬜ Open (blocked on #410) |
| [#412](https://github.com/zynax-io/zynax/issues/412) | O4 | Registry client + bootstrap | ⬜ Open (blocked on #411) |
| [#413](https://github.com/zynax-io/zynax/issues/413) | O5 | Dockerfile, docker-compose, AGENTS.md | ⬜ Open (blocked on #412) |

**Engineer profile:** Python engineer with LLM API experience. Read `docs/spdd/383-llm-adapter/canvas.md`.
Capability: `chat_completion`. Provider support: OpenAI (via `openai` SDK), AWS Bedrock
(via `boto3`), Ollama (via REST). Config must follow 12-Factor: provider type + API key
from env vars, model name from capability payload `config.model` (never hardcoded).
Streaming response must emit `PROGRESS` task events during generation.

#### langgraph-adapter (#384) — Python
| Issue | Step | Title | Status |
|-------|------|-------|--------|
| [#415](https://github.com/zynax-io/zynax/issues/415) | O2 | Module scaffold + GraphMount config | ⬜ Open |
| [#416](https://github.com/zynax-io/zynax/issues/416) | O3 | GraphLoader + LangGraphHandler | ⬜ Open (blocked on #415) |
| [#417](https://github.com/zynax-io/zynax/issues/417) | O4 | Registry client + bootstrap | ⬜ Open (blocked on #416) |
| [#418](https://github.com/zynax-io/zynax/issues/418) | O5 | Dockerfile, docker-compose, AGENTS.md | ⬜ Open (blocked on #417) |

**Engineer profile:** Python engineer with LangGraph experience. Read `docs/spdd/384-langgraph-adapter/canvas.md`.
The adapter mounts a `StateGraph` as a Zynax capability. Each node becomes a `capability`
the workflow can invoke. The `GraphMount` config maps graph IDs to capability names.
This is the proof-of-concept for engine-agnosticism (LangGraph apps become Zynax capabilities
without rewriting the graph).

---

## M5.F — CI/CD Performance Sprint (#542)

**Canvas:** SPDD exempt (ci/chore type per ADR-019)
**Goal:** Cut PR cycle time from ~25 min to ~7 min. Auto-merge + strict branch protection. Fix all broken release pipelines.
**Merge strategy:** No merge queue — auto-merge (`allow_auto_merge: true`, set via API) with `strict: true` branch protection. Developer rebases; auto-merge fires when all checks pass.

### Group A — Branch protection, concurrency, force-run (P0/P1)
| Issue | Title | Size | Priority |
|-------|-------|------|----------|
| [#544](https://github.com/zynax-io/zynax/issues/544) | Enable GitHub Merge Queue + remove `strict: true` | XS | ✅ Done (merge queue removed; strict: true + allow_auto_merge re-enabled via API — see #589) |
| [#547](https://github.com/zynax-io/zynax/issues/547) | Remove `test-integration` from required status checks | XS | ✅ Done |
| [#548](https://github.com/zynax-io/zynax/issues/548) | Enable `allow_auto_merge` | XS | ✅ Done via API |
| [#545](https://github.com/zynax-io/zynax/issues/545) | Fix CI concurrency — cancel stale runs per branch | XS | ✅ Done |
| [#589](https://github.com/zynax-io/zynax/issues/589) | Remove `merge_group` trigger from all workflow files | XS | ✅ Done |
| [#546](https://github.com/zynax-io/zynax/issues/546) | Remove push-to-main forced-true override | S | ✅ Done |
| [#554](https://github.com/zynax-io/zynax/issues/554) | Force-full-pipeline trigger (dispatch, label, `[full-ci]`) | S | P1 |

### Group B — Per-service change detection (P2)
| Issue | Title | Size |
|-------|-------|------|
| [#549](https://github.com/zynax-io/zynax/issues/549) | Extend changes job with per-service granularity | M |
| [#550](https://github.com/zynax-io/zynax/issues/550) | Scope govulncheck to changed services only | M |
| [#220](https://github.com/zynax-io/zynax/issues/220) | Parallel CI job matrix with shared Go module cache | M |

### Group C — Alpine CI runner sub-epic (#543)
| Issue | Title | Size |
|-------|-------|------|
| [#551](https://github.com/zynax-io/zynax/issues/551) | Create Dockerfile.ci-runner | S | ✅ Done |
| [#552](https://github.com/zynax-io/zynax/issues/552) | Switch all GH Actions jobs to ci-runner | M | ✅ Done |
| [#358](https://github.com/zynax-io/zynax/issues/358) | Publish tools image to public GHCR securely | S |

### Group D — M7 test gate integration
| Issue | Title | Milestone |
|-------|-------|-----------|
| [#553](https://github.com/zynax-io/zynax/issues/553) | Activate integration test suite as required CI gate | M7.C step 4 |

### Group E — DRY/KISS refactor
| Issue | Title | Size |
|-------|-------|------|
| [#555](https://github.com/zynax-io/zynax/issues/555) | DRY/KISS refactor — reusable workflows, composite actions, extracted scripts | L |

---

## M5.F.R — Release Pipeline & Artifact Visibility (#556)

**Parent epic:** [#542 M5.F](https://github.com/zynax-io/zynax/issues/542)
**Goal:** Every Zynax deliverable has a versioned, downloadable artifact published on GitHub Releases and GHCR.

### Current state: what is broken
- **No versioned release has ever been cut.** All install URLs return HTTP 404.
- **Release race condition.** All three release workflows conflict on the same tag.
- **task-broker excluded** from service-release matrix.
- **http-adapter** has a Dockerfile but zero release pipeline.
- **Two workflows build the tools image** to different names.
- ~~**Go toolchain mismatch** — service Dockerfiles used `golang:1.25-alpine` but `go.mod` requires `go 1.26.3`; service image builds failed with `GOTOOLCHAIN=local` error.~~ ✅ Fixed by [#601](https://github.com/zynax-io/zynax/issues/601)

### Child issues (ordered execution plan)
| Issue | Title | Size | Priority |
|-------|-------|------|----------|
| [#557](https://github.com/zynax-io/zynax/issues/557) | Fix release race condition — unified workflow | M | ✅ Done |
| [#558](https://github.com/zynax-io/zynax/issues/558) | Cut v0.4.0 — first versioned release tag | XS | ✅ Done (CHANGELOG promoted; tag push pending user action) |
| [#559](https://github.com/zynax-io/zynax/issues/559) | Add task-broker to service-release matrix | XS | ✅ Done (delivered in #557 — task-broker already in release.yml matrix) |
| [#560](https://github.com/zynax-io/zynax/issues/560) | Add http-adapter image to release pipeline | S | ✅ Done |
| [#561](https://github.com/zynax-io/zynax/issues/561) | Push service/adapter images to GHCR on every main merge | S | ✅ Done |
| [#601](https://github.com/zynax-io/zynax/issues/601) | Fix Go builder base image 1.25→1.26.3-alpine in service Dockerfiles | XS | ✅ Done · unblocked #562 |
| [#562](https://github.com/zynax-io/zynax/issues/562) | Make GHCR images publicly readable | XS | ✅ Done · unblocked tools+#563+#566 |
| (admin) | Confirm zynax/tools published + set public | — | ✅ Done — tools-image.yml succeeded 2026-05-20; package set public |
| [#563](https://github.com/zynax-io/zynax/issues/563) | Deduplicate tools image (remove tools-publish.yml) | XS | ✅ Done |
| [#564](https://github.com/zynax-io/zynax/issues/564) | Pin action digests + add linux/arm64 | XS | P2 |
| [#565](https://github.com/zynax-io/zynax/issues/565) | Add trivy container scan gate before GHCR push | S | P2 |
| [#566](https://github.com/zynax-io/zynax/issues/566) | README Docker Images section with pull commands | S | ✅ Done |
| [#641](https://github.com/zynax-io/zynax/issues/641) | Per-service change detection for image builds — skip unchanged images on main push | M | P2 |
| [#642](https://github.com/zynax-io/zynax/issues/642) | Switch service Dockerfiles to distroless/static:nonroot + add `-ldflags "-s -w"` | S | P2 |

**Cross-links:**
- #601 → #562 → tools-public → #563 → #566: Full chain — service Dockerfiles fixed → images made public → zynax/tools rebuilt and made public → old zynax-tools removed → README documented.
- #358 ↔ #563: #358 (publish tools image securely) is superseded by #563 (deduplication); close #358 when #563 merges.
- #235 → #489/#465: #235 (standalone SBOM) superseded by M6.C child #489; close #235 when M6 goes active.
- #239 → #489/#465: Same supersession pattern.
- #565 ↔ #236: Complementary; #565 = trivy in release, #236 = trivy in security CI.

---

## M5.A — Truth Pass (#458)

**Canvas:** [docs/spdd/458-truth-pass/canvas.md](../spdd/458-truth-pass/canvas.md)

| Issue | Title | Status |
|-------|-------|--------|
| [#472](https://github.com/zynax-io/zynax/issues/472) | Remove CNCF badge + update milestone status | ✅ Done |
| [#473](https://github.com/zynax-io/zynax/issues/473) | Audit CHANGELOG for phantom entries | ✅ Done |
| [#474](https://github.com/zynax-io/zynax/issues/474) | Python SDK decision → implement minimal Agent base class | ✅ Epic complete (BATCH 3) |
| **NEW** | Fix SECURITY.md — remove mTLS/SBOM/cosign false claims | ✅ Done (2026-05-20) |
| **NEW** | Add per-service status table to README | ⬜ Open (file as child of #458) |

### Python SDK epic (#474) — promoted

**Canvas:** [docs/spdd/474-python-sdk/canvas.md](../spdd/474-python-sdk/canvas.md)
**Decision:** Option A — implement minimal `Agent` base class.

| Issue | Step | Title | Status |
|-------|------|-------|--------|
| [#535](https://github.com/zynax-io/zynax/issues/535) | O1 | Implement Agent base class | ✅ Done |
| [#536](https://github.com/zynax-io/zynax/issues/536) | O2 | Unit tests (≥ 85% coverage) | ✅ Done |
| [#537](https://github.com/zynax-io/zynax/issues/537) | O3 | Docs update | ✅ Done |

---

## M5.B — Engine Correctness Hardening (#459)

**Canvas:** [docs/spdd/459-engine-correctness/canvas.md](../spdd/459-engine-correctness/canvas.md)

| Issue | Title | Status |
|-------|-------|--------|
| [#475](https://github.com/zynax-io/zynax/issues/475) | resolveTemplate map-iteration determinism | ✅ Done |
| [#476](https://github.com/zynax-io/zynax/issues/476) | Guard evaluator — cel-go epic | ⬜ Epic (see BATCH 2) |
| [#477](https://github.com/zynax-io/zynax/issues/477) | CompileWorkflow structured error list | ✅ Done |
| [#478](https://github.com/zynax-io/zynax/issues/478) | SSE WriteTimeout fix | ✅ Done |

### Guard evaluator epic (#476) — promoted

**Canvas:** [docs/spdd/476-guard-parser/canvas.md](../spdd/476-guard-parser/canvas.md)
**Decision:** Option A — integrate `github.com/google/cel-go` (fail-closed on eval error).

| Issue | Step | Title | Status |
|-------|------|-------|--------|
| [#538](https://github.com/zynax-io/zynax/issues/538) | O1 | Integrate cel-go into evalGuard | ✅ Done |
| [#539](https://github.com/zynax-io/zynax/issues/539) | O2 | Test suite + fuzz seed | ✅ Done |
| [#540](https://github.com/zynax-io/zynax/issues/540) | O3 | Remove CEL misrepresentation from docs | ✅ Done |

---

## M5.C — Capability Dispatch End-to-End (#460)

**Canvas:** [docs/spdd/460-capability-dispatch/canvas.md](../spdd/460-capability-dispatch/canvas.md)

### task-broker MVP (#479) — code complete, quality in progress

**Canvas:** [docs/spdd/479-task-broker/canvas.md](../spdd/479-task-broker/canvas.md)
Implementation merged: PRs #520, #522, #523. Domain coverage: 92.7%.

| Issue | Canvas step | Title | Status |
|-------|-------------|-------|--------|
| [#530](https://github.com/zynax-io/zynax/issues/530) | O6 | Update AGENTS.md | ✅ Done |
| [#531](https://github.com/zynax-io/zynax/issues/531) | O7 | Align service BDD + godog steps | ✅ Done |
| [#532](https://github.com/zynax-io/zynax/issues/532) | O8 | Handler unit tests (grpcErr coverage) | ⬜ Open |

### agent-registry MVP (#480) — pending

**Canvas:** [docs/spdd/480-agent-registry/canvas.md](../spdd/480-agent-registry/canvas.md)

| Issue | Canvas step | Title | Status |
|-------|-------------|-------|--------|
| [#526](https://github.com/zynax-io/zynax/issues/526) | O1 | Trim BDD to proto scope | ⬜ Open |
| [#527](https://github.com/zynax-io/zynax/issues/527) | O2 | Domain layer | ⬜ Open (blocked on #526) |
| [#528](https://github.com/zynax-io/zynax/issues/528) | O3 | gRPC wiring + cmd + go.work | ⬜ Open (blocked on #527) |

### compose wiring (#481)

| Issue | Title | Status |
|-------|-------|--------|
| [#481](https://github.com/zynax-io/zynax/issues/481) | Add task-broker + agent-registry to docker-compose | ⬜ Open (blocked on #528) |

---

## M5.D — Control Plane Security Baseline (#461) ✅ Complete

**Canvas:** [docs/spdd/461-security-baseline/canvas.md](../spdd/461-security-baseline/canvas.md)
All 5 child issues merged: #482 #483 #484 #485 #486.

---

## M5.E — Developer Experience Polish (#462) ✅ Complete

**Canvas:** [docs/spdd/462-dx-polish/canvas.md](../spdd/462-dx-polish/canvas.md)
Both child issues merged: #485 #486.

---

## Adapter Library (#377)

**Canvas:** [docs/spdd/377-adapter-library/canvas.md](../spdd/377-adapter-library/canvas.md)

### http-adapter (#380) ✅ Complete
All step issues merged: #391 #392 #393 #394 #395 #396 #397.

### git-adapter (#381) · go-adapter · Capabilities: `open_pr`, `request_review`, `get_diff`
**Canvas:** [docs/spdd/381-git-adapter/canvas.md](../spdd/381-git-adapter/canvas.md)
BDD done (#399). Implementation pending: #400 → #401 → #402 → #403.

### ci-adapter (#382) · go-adapter · Capabilities: `trigger_workflow`, `get_run_status`
**Canvas:** [docs/spdd/382-ci-adapter/canvas.md](../spdd/382-ci-adapter/canvas.md)
BDD done (#404). Implementation pending: #405 → #406 → #407 → #408.

### llm-adapter (#383) · Python · Capability: `chat_completion`
**Canvas:** [docs/spdd/383-llm-adapter/canvas.md](../spdd/383-llm-adapter/canvas.md)
BDD done (#409). Implementation pending: #410 → #411 → #412 → #413.

### langgraph-adapter (#384) · Python · Maps LangGraph StateGraph to capabilities
**Canvas:** [docs/spdd/384-langgraph-adapter/canvas.md](../spdd/384-langgraph-adapter/canvas.md)
BDD done (#414). Implementation pending: #415 → #416 → #417 → #418.

---

## Tooling (#442) ✅ Complete
All 4 child issues merged: #443 #444 #445 #446.

---

## Architecture Gaps to Address (M5 or promote to M6)

These gaps were identified in the 2026-05-20 principal architect review
(`docs/architecture/2026-05-20-principal-architect-review.md`) and do not yet have
open issues. File them before or during M5 execution:

| Gap | Severity | Issue | Milestone |
|-----|----------|-------|-----------|
| G1: Bearer-token constant-time compare | **High** | [#567](https://github.com/zynax-io/zynax/issues/567) | M5 |
| G2: ReadHeaderTimeout + MaxBytesReader | Medium | [#568](https://github.com/zynax-io/zynax/issues/568) | M5 |
| G3: Rate limiting on POST /apply | Medium | [#580](https://github.com/zynax-io/zynax/issues/580) | M6 |
| G4: No RetryPolicy on Temporal Activities | Medium | [#569](https://github.com/zynax-io/zynax/issues/569) | M5 |
| G5: Polling Watch Temporal load | Medium | [#492](https://github.com/zynax-io/zynax/issues/492) | M7 |
| G6: resolveTemplate bespoke engine | Low | [#584](https://github.com/zynax-io/zynax/issues/584) | M6 |
| G7: mergePayload drops non-strings | Low | [#571](https://github.com/zynax-io/zynax/issues/571) | M5 |
| G8: Action.Output parsed, never mapped | Low | [#581](https://github.com/zynax-io/zynax/issues/581) | M6 |
| G9: No CODEOWNERS | Low | ~~#573~~ **Closed — file exists** | — |
| G10: workflow-compiler retention contract violated | Medium | [#572](https://github.com/zynax-io/zynax/issues/572) | M5 |
| G11: No benchmarks | Medium | [#493](https://github.com/zynax-io/zynax/issues/493) | M7 |
| G12: No fuzz tests | Medium | [#539](https://github.com/zynax-io/zynax/issues/539) (partial) | M5/M7 |
| G13: No load tests | Medium | M7 backlog | M7 |
| G14+G15: Manifest hash ADR + YAML canonicalization | Low | [#583](https://github.com/zynax-io/zynax/issues/583) | M6 |
| G16: Background-context goroutines | Medium | [#570](https://github.com/zynax-io/zynax/issues/570) | M5 |
| G17: Stub services in SERVICE_LIST | Low | [#574](https://github.com/zynax-io/zynax/issues/574) | M5 |
| G18: No community channel | Medium | [#470](https://github.com/zynax-io/zynax/issues/470) | M8 |
| G19: Kagent positioning | **High** | [#575](https://github.com/zynax-io/zynax/issues/575) | M5 |
| G20: pkg.go.dev module reference | Medium | [#582](https://github.com/zynax-io/zynax/issues/582) | M6 |
| G21: Python SDK claims v0.1.0, is 3-line placeholder | High | [#474](https://github.com/zynax-io/zynax/issues/474) | M5 |
| G22: Summarizer phantom | Low | [#576](https://github.com/zynax-io/zynax/issues/576) | M5 |
| G23: Phantom AGENT_LIST entries | Low | [#577](https://github.com/zynax-io/zynax/issues/577) | M5 |
| G24: Compose omits task-broker/agent-registry | High | [#481](https://github.com/zynax-io/zynax/issues/481) | M5 |
| H8: ADR-021 scale plan | High | [#578](https://github.com/zynax-io/zynax/issues/578) | M6 |
| H9: Unimplemented gRPC skeletons | Medium | [#574](https://github.com/zynax-io/zynax/issues/574) | M5 |
| README status table | High | [#579](https://github.com/zynax-io/zynax/issues/579) | M5 |
| NEW-1: gRPC call deadlines | High | [#622](https://github.com/zynax-io/zynax/issues/622) | M5 |
| NEW-4: `ZYNAX_GW_API_KEY=""` bypass | High | [#623](https://github.com/zynax-io/zynax/issues/623) | M5 |
| H1: Stateless workflow-compiler (OOM risk R4) | **High** | [#466](https://github.com/zynax-io/zynax/issues/466) | M5 (**promoted from M6** 2026-05-21) |
| Architecture overhaul docs (tracking) | — | [#624](https://github.com/zynax-io/zynax/issues/624) | M5 |

---

## Blocked / Parking

- **Adapter implementations** (#400+, #405+, #410+, #415+) — wait for compose wiring (#481) to land first so adapters can be tested against the live registry.
- **#466 promoted M6→M5** (2026-05-21) — stateless compiler / drop in-memory IR store. OOM risk R4 from 2026-05-20 review; 3–5 day effort; unblocks horizontal scale before v0.4.0 ships.
