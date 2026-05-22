<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax — Deep Architectural & Platform-Engineering Review

**Repository:** `github.com/zynax-io/zynax` · Apache-2.0 · CNCF Sandbox candidate
**Review date:** 2026-05-22
**Reviewer mandate:** Platform Engineering · Configuration · Dependency management · K8s readiness
**Branch reviewed:** `main` (cloned for analysis, milestone **M4 complete**)
**Method:** Full clone, file-level inspection of every `go.mod`, every service `main.go`, all
Dockerfiles, compose files, the Makefile, CI workflows, Renovate config, and the spec/schema
layer. Every claim is grounded in a specific file; suspected bugs were verified before reporting
and two initial suspicions were dropped after verification (see *Non-findings*).

---

## Executive Summary

Zynax is a **well-above-average early-stage platform**. The contracts-first discipline (proto +
JSON-Schema + 140+ BDD scenarios), the hexagonal `internal/{api,domain,infrastructure}` layout,
distroless-nonroot images, SHA-pinned GitHub Actions, DCO enforcement, Renovate grouping, syft
SBOMs, and a single-source-of-truth coverage-gate file are all things most "production" repos
never get to. **A shallow review would invent problems here; this one credits what works and
targets the real gaps.**

The honest gap is this: **the project is positioned as "Kubernetes-native / cloud-native control
plane," but at M4 there is no Kubernetes layer at all** — no Helm chart, no manifest, no
`Chart.yaml` anywhere in the tree (`find . -name Chart.yaml` → empty), even though `README.md:423`
claims *"infra/ — Docker-first dev environment + Helm charts."* The operational substrate today
is **docker-compose only**. That is fine for M4, but it means the "Kubernetes-native operation"
objective is **greenfield to be built**, not legacy to be fixed — and the README currently
overstates reality.

The second theme is **convergence debt**: the codebase grew service-by-service and three
independent conventions calcified for the same concern (configuration loading, env-var naming,
health/observability). Nothing is broken in the supported `make run-local` path, but each new
service currently *copies a slightly different sibling*, so the divergence compounds. The window
to standardize is now — before services 6, 7, 8 land.

| Dimension | Score (0–10) | One-line verdict |
|---|---|---|
| Config maintainability | **4** | 2 mechanisms, 5 env-prefix conventions, no shared lib |
| Dependency maintainability | **8** | Versions aligned, Renovate mature; not *enforced* |
| Operational complexity | **5** | Clean local DX; observability inconsistent; no K8s |
| Developer experience | **8** | Docker-only, auto-discovery, strong docs |
| Kubernetes readiness | **2** | Charts don't exist yet; probes inconsistent |
| Scalability | **5** | Stateless services + Temporal good; unproven |
| Security posture | **8** | Distroless, SBOM, gitleaks, pinned actions; cosign pending |
| CI/CD maturity | **8** | Path-filtered, SHA-pinned, stub-drift gates |

---

## 1. Verified Findings Inventory

These are the concrete, file-level observations everything else builds on.

### 1.1 — Two configuration mechanisms for one concern

Four services use `kelseyhightower/envconfig` with struct tags; **`engine-adapter` hand-rolls**
its own `getEnv` / `getEnvInt` / `parseLogLevel`
(`services/engine-adapter/cmd/engine-adapter/main.go:212–250`). `parseLogLevel` is duplicated
across all five services rather than shared.

**Tracked:** [#667](https://github.com/zynax-io/zynax/issues/667) (M6)

### 1.2 — Five env-var naming conventions

The prefix passed to `envconfig.Process` differs per service, and the field names are inconsistent:

| Service | Prefix | Port var | Log var |
|---|---|---|---|
| agent-registry | `ZYNAX_REGISTRY` | `ZYNAX_REGISTRY_GRPC_PORT` | `ZYNAX_REGISTRY_LOG_LEVEL` |
| task-broker | `ZYNAX_BROKER` | `ZYNAX_BROKER_GRPC_PORT` | `ZYNAX_BROKER_LOG_LEVEL` |
| api-gateway | `ZYNAX_GW` | `ZYNAX_GW_HTTP_PORT` | `ZYNAX_GW_LOG_LEVEL` |
| workflow-compiler | *(empty, full tags)* | `ZYNAX_WC_PORT` ← note: not `_GRPC_PORT` | `ZYNAX_WC_LOG_LEVEL` |
| engine-adapter | *(hand-rolled)* | `ZYNAX_ENGINE_ADAPTER_GRPC_PORT` | `ZYNAX_ENGINE_ADAPTER_LOG_LEVEL` |

Abbreviation policy is itself inconsistent: `REGISTRY`, `BROKER`, `GW`, `WC`, `ENGINE_ADAPTER`
(full). The engine-adapter even **breaks its own prefix**: it reads `ZYNAX_ENGINE_ACTIVE_ENGINE`,
not `ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE` (`main.go:44`).

**Tracked:** [#666](https://github.com/zynax-io/zynax/issues/666) (env name fix, M5),
[#667](https://github.com/zynax-io/zynax/issues/667) (shared config lib, M6)

### 1.3 — A stale default that only "works" because compose overrides it

`api-gateway` defaults `COMPILER_ADDR` to `localhost:50051`
(`services/api-gateway/cmd/api-gateway/main.go:27`), but `workflow-compiler` binds **`50054`**
(`ZYNAX_WC_PORT` default `50054`). The two never connect on defaults; `docker-compose.yml` masks
it by setting `ZYNAX_GW_COMPILER_ADDR=workflow-compiler:50054`. Anyone running the gateway
outside compose silently mis-dials.

**Tracked:** [#661](https://github.com/zynax-io/zynax/issues/661) (M5)

### 1.4 — Only one service has a `config` package

`workflow-compiler/internal/config/config.go` exists; the other four inline config into
`main.go`. New contributors have no single pattern to copy.

**Tracked:** [#667](https://github.com/zynax-io/zynax/issues/667) (M6)

### 1.5 — "MetricsPort" exposes no metrics

`engine-adapter` and `api-gateway` open an HTTP server on a port named `MetricsPort`
(`9095` / their metrics port) that serves **only `/healthz`, `/readyz`, `/startupz`** — there
is no `/metrics` handler and no Prometheus/OTel import. The name promises observability the
code doesn't deliver.

**Tracked:** [#491](https://github.com/zynax-io/zynax/issues/491) (M7.A — wire Prometheus + OTel)

### 1.6 — Three health-check models across five services

| Service | HTTP probes | Prometheus `/metrics` | OTel tracing |
|---|---|---|---|
| workflow-compiler | `/healthz` only | ✅ `promhttp` | ✅ `otelgrpc` |
| api-gateway | `/healthz` `/readyz` `/startupz` | ❌ | ❌ |
| engine-adapter | `/healthz` `/readyz` `/startupz` | ❌ | ❌ |
| agent-registry | none (compose does TCP dial) | ❌ | ❌ |
| task-broker | none (compose does TCP dial) | ❌ | ❌ |

Real metrics + tracing exist in exactly **one of five** services. This is the single biggest
operability inconsistency.

**Tracked:** [#491](https://github.com/zynax-io/zynax/issues/491) (metrics/tracing, M7.A),
[#487](https://github.com/zynax-io/zynax/issues/487) / [#463](https://github.com/zynax-io/zynax/issues/463) / [#656](https://github.com/zynax-io/zynax/issues/656)
(health probe semantics, M6.A)

### 1.7 — Five near-identical Dockerfiles

Each service Dockerfile differs from the others by ~16 lines (service-name substitution only).
Quality is high (multi-stage, `gcr.io/distroless/static:nonroot`, `CGO_ENABLED=0`, `-trimpath
-ldflags "-s -w"`, `GOWORK=off`); duplication is high.

**Tracked:** [#668](https://github.com/zynax-io/zynax/issues/668) (M6)

### 1.8 — Two broken Makefile targets

`make sbom` and `make scan-image` pass `services/$(SVC)/` as the Docker build context, but
every service Dockerfile requires **repo-root context** (it COPYs `protos/generated/go/...`
and `tools/healthcheck/...`). Both targets always fail with "file not found in build context".

**Tracked:** [#662](https://github.com/zynax-io/zynax/issues/662) (M5)

### 1.9 — Hardcoded service list includes non-existent services

`Makefile:8` hardcodes `GO_SERVICES` including `memory-service` and `event-bus`, which are
stubs — only an `AGENTS.md` and a `.feature` file, no `go.mod`, not in `go.work`. Any
target that iterates `GO_SERVICES` fails for those stubs.

**Tracked:** [#663](https://github.com/zynax-io/zynax/issues/663) (M5),
[#574](https://github.com/zynax-io/zynax/issues/574) (remove stubs from SERVICE_LIST docs, M5)

### 1.10 — Adapter config contradicts real ports

`agents/adapters/http/agent-def.yaml.example` sets `registry_endpoint: "agent-registry:9091"`,
but `agent-registry` actually serves gRPC on **`50052`**. The `dev`-profile compose
bind-mounts this example as the live config.

**Tracked:** [#665](https://github.com/zynax-io/zynax/issues/665) (M5)

### 1.11 — Doc/reality drifts

- README claims `requires Go 1.25+` while every `go.mod` and `go.work` pins **`go 1.26.3`**.
- README:423 claims `infra/ — Docker-first dev environment + Helm charts` (no charts exist).
- `tools/healthcheck/go.mod` pins `go 1.26` (no patch) while all others pin `1.26.3`.

**Tracked:** [#664](https://github.com/zynax-io/zynax/issues/664) (M5)

### 1.12 — Supply-chain signing is reserved, not done

`service-release.yml:29` keeps `id-token: write` for cosign OIDC signing + SLSA provenance
"(#239)" — planned, not implemented. SBOMs (CycloneDX via syft) *are* generated and attached.

**Tracked:** [#489](https://github.com/zynax-io/zynax/issues/489) (M6.C — supply chain hardening)

---

## 2. Major Architectural Issues

1. **Configuration has no center.** Each service is its own island of env-var parsing. The thing
   that validates user manifests is rigorous (JSON Schemas in `spec/schemas/`); the thing that
   configures the validators is ad-hoc.

2. **The control plane has no plane to control it on.** "Kubernetes-native" is the headline, but
   the deployment target is compose. Every operability primitive that matters in K8s — resource
   requests/limits, HPA, PDB, NetworkPolicy, ServiceAccount, ConfigMap/Secret projection,
   rolling-update strategy — is undefined because the chart doesn't exist.

3. **Observability is a per-service afterthought, not a platform contract.** One service has
   metrics+tracing; the rest have neither. You cannot build an SLO on this surface yet.

4. **Convergence debt is accelerating.** With auto-discovery for adapters/agents but copy-paste
   for services, the *services* are the ones drifting fastest.

---

## 3. Configuration Anti-Patterns (detailed)

- **Mechanism sprawl** (§1.1): mixing `envconfig` and hand-rolled parsing. `engine-adapter`'s
  `getEnv` treats `""` as unset while `envconfig` distinguishes unset from empty — subtly
  different semantics for the same platform.

- **Naming entropy** (§1.2): five prefix styles. Operators can't predict the variable name for
  "log level" without reading source; Helm templating must special-case each service; an
  org-wide policy like "set `*_LOG_LEVEL=debug` everywhere" is impossible to express.

- **Defaults that lie** (§1.3): `localhost:50051`. A wrong default is a latent production
  incident the moment someone runs outside compose.

- **Misnamed ports** (§1.5): `MetricsPort` with no metrics. A Prometheus `ServiceMonitor`
  author will scrape `:9095/metrics` and get 404s.

- **No layering, no validation, no reload.** No notion of global→env→service→runtime precedence,
  no fail-fast validation beyond type coercion.

- **Secrets handled well** in the one place they appear — the http-adapter resolves `$ENV_NAME`
  references at startup and refuses inline credentials and runtime-supplied URLs (SSRF guard).
  That pattern should become the platform standard.

---

## 4. Dependency-Management Assessment

**Strength, with one structural caveat.** Across all 11 `go.mod` files the shared versions are
identical — `grpc v1.80.0`, `protobuf v1.36.11`, `yaml.v3 v3.0.1`, `envconfig v1.4.0`. Go
toolchain is `1.26.3` everywhere except `tools/healthcheck` (`1.26`). Renovate groups deps,
pins action digests, and automerges patches.

**Caveat:** the alignment is *currently true* but **not enforced**. Eleven independent modules
can drift the instant Renovate is paused or two grouped PRs merge out of order. There is no CI
gate asserting "shared deps match across modules."

**Tracked:** [#669](https://github.com/zynax-io/zynax/issues/669) (M6)

---

## 5. Recommendations

### A. Quick Wins (days — M5)

| # | Action | Issue |
|---|--------|-------|
| A1 | Fix `api-gateway` `COMPILER_ADDR` default: `50051` → `50054` | [#661](https://github.com/zynax-io/zynax/issues/661) |
| A2 | Fix `sbom` and `scan-image` Makefile targets — use repo-root context `-f services/$(SVC)/Dockerfile .` | [#662](https://github.com/zynax-io/zynax/issues/662) |
| A3 | Auto-discover `GO_SERVICES` from `go.work` (same as adapters) | [#663](https://github.com/zynax-io/zynax/issues/663) |
| A4 | Rename `MetricsPort` → `HealthPort` (or wire real `/metrics` — see #491) | [#491](https://github.com/zynax-io/zynax/issues/491) |
| A5 | Correct README Go version claim and Helm chart claim | [#664](https://github.com/zynax-io/zynax/issues/664) |
| A6 | Fix http-adapter example `registry_endpoint` port (`9091` → `50052`) | [#665](https://github.com/zynax-io/zynax/issues/665) |
| A7 | Align `ZYNAX_ENGINE_ACTIVE_ENGINE` → `ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE` | [#666](https://github.com/zynax-io/zynax/issues/666) |

### B. Medium Refactors (weeks — M6)

**B1 — Shared `libs/zynaxconfig` package.** Create a new Go module with `Base` struct
(`LogLevel`, `GRPCPort`, `HealthPort`) and a generic `Load[T](service string, dst *T) error`
that wraps `envconfig.Process`, validates, and configures `slog`. Each service embeds `Base`
and declares only its own extra fields. Standardize grammar to `ZYNAX_<SERVICE>_<FIELD>` with
full service tokens (`BROKER`, `REGISTRY`, `GATEWAY`, `COMPILER`, `ENGINE`).

**Tracked:** [#667](https://github.com/zynax-io/zynax/issues/667)

**B2 — Shared `libs/zynaxobs` package.** Standard gRPC `StatsHandler` (otelgrpc), `promhttp`
`/metrics` handler, and `health.New()` registering `/healthz` + `/readyz` + `/startupz` + gRPC
`grpc_health_v1` with consistent semantics.

**Tracked:** [#491](https://github.com/zynax-io/zynax/issues/491) (wire metrics/tracing)

**B3 — Templated Dockerfile.** One `infra/docker/Dockerfile.service` parameterized on
`--build-arg SVC=…` eliminates five near-identical files.

**Tracked:** [#668](https://github.com/zynax-io/zynax/issues/668)

**B4 — Dependency-alignment CI gate.** `zynax-ci check deps` fails if shared `go.mod` versions
diverge across modules.

**Tracked:** [#669](https://github.com/zynax-io/zynax/issues/669)

### C. Large Architectural Improvements (quarters — M6/M7)

**C1 — Kubernetes layer.** Umbrella Helm chart with per-service subcharts (Deployment, Service,
ServiceAccount, HPA, PDB, NetworkPolicy, ConfigMap, ServiceMonitor). The `ZYNAX_<SERVICE>_*`
grammar from B1 maps directly to ConfigMap → env projection.

**Tracked:** [#241](https://github.com/zynax-io/zynax/issues/241) [#242](https://github.com/zynax-io/zynax/issues/242) [#244](https://github.com/zynax-io/zynax/issues/244) [#245](https://github.com/zynax-io/zynax/issues/245) (M6)

**C2 — Observability platform contract.** `ServiceMonitor` per chart, standard RED metrics
on every gRPC server, OTel tracing across the full call chain, first SLOs.

**Tracked:** [#467](https://github.com/zynax-io/zynax/issues/467) (M7.A)

**C3 — GitOps + supply-chain finish.** Argo CD `Application` per env, cosign keyless signing +
SLSA provenance (#239), Kyverno/Gatekeeper admission baseline.

**Tracked:** [#465](https://github.com/zynax-io/zynax/issues/465) (M6.C), [#489](https://github.com/zynax-io/zynax/issues/489), [#244](https://github.com/zynax-io/zynax/issues/244)

---

## 6. Proposed Target Architecture

### 6.1 Folder structure (additive)

```
zynax/
  libs/
    zynaxconfig/     ← Base config + Load() + validation      (B1, #667)
    zynaxobs/        ← metrics + tracing + health              (B2, #491)
  services/<svc>/internal/config/config.go   ← embeds zynaxconfig.Base
  infra/
    docker/Dockerfile.service                ← single parameterized build  (B3, #668)
    helm/
      zynax/                                 ← umbrella chart
        Chart.yaml  values.yaml  values-staging.yaml  values-production.yaml
        charts/<svc>/                        ← per-service subchart
      kustomize/overlays/{dev,staging,prod}/
    gitops/
      apps/<env>/application.yaml            ← Argo CD
```

### 6.2 Configuration hierarchy (precedence, lowest → highest)

```
1. Service compiled defaults        (struct `default:` tags — service-owned)
2. Helm chart values.yaml           (per-service defaults, GitOps-tracked)
3. Helm global: block               (org-wide shared config)
4. values-<env>.yaml / Kustomize    (environment overrides)
5. ConfigMap → env  ZYNAX_<SVC>_*   (rendered, non-secret)
6. Secret/ExternalSecret → env      (credentials, never in values)
7. Pod-spec env / `helm --set`      (runtime override)
8. CLI flags                        (operator/CLI tools only)
```

No new config DSL. Validation stays fail-fast in `zynaxconfig.Load`. Dynamic reload is
*deliberately out of scope* until a concrete need exists.

### 6.3 Implementation roadmap

| Phase | Scope | Milestone |
|---|---|---|
| **P0 (week 1)** | All Quick Wins §A — pure fixes + doc truth | M5 |
| **P1 (weeks 2–4)** | `libs/zynaxconfig` (B1) + `libs/zynaxobs` (B2); migrate one service, prove, then rest | M6 |
| **P2 (weeks 4–6)** | Dockerfile consolidation (B3) + dep-alignment gate (B4) | M6 |
| **P3 (weeks 6–12)** | Helm umbrella + subcharts (C1) targeting dev cluster; `ServiceMonitor`s; first SLOs (C2) | M6 |
| **P4 (quarter 2)** | GitOps (Argo CD), cosign+SLSA (#489), admission baseline (C3) | M6→M7 |

---

## 7. Configuration Extraction Reference

| Current location | Hardcoded / divergent value | Suggested config name | Suggested default | Owner | Issue |
|---|---|---|---|---|---|
| 5× service `main.go` | `parseLogLevel` + `"info"` | `ZYNAX_<SVC>_LOG_LEVEL` | `info` | `zynaxconfig.Base` | [#667](https://github.com/zynax-io/zynax/issues/667) |
| 5× service `main.go` | gRPC port | `ZYNAX_<SVC>_GRPC_PORT` | per-svc | `zynaxconfig.Base` | [#667](https://github.com/zynax-io/zynax/issues/667) |
| engine-adapter, api-gw | `MetricsPort` (no metrics served) | `ZYNAX_<SVC>_HEALTH_PORT` | `9090` | `zynaxconfig.Base` | [#491](https://github.com/zynax-io/zynax/issues/491) |
| api-gateway `main.go:27` | `COMPILER_ADDR=localhost:50051` (stale) | `ZYNAX_GATEWAY_COMPILER_ADDR` | `workflow-compiler:50054` | service config | [#661](https://github.com/zynax-io/zynax/issues/661) |
| api-gateway/engine/broker | `*_ADDR` peer endpoints | `ZYNAX_<SVC>_<PEER>_ADDR` | K8s svc DNS | ConfigMap | [#667](https://github.com/zynax-io/zynax/issues/667) |
| engine-adapter `main.go:44` | `ZYNAX_ENGINE_ACTIVE_ENGINE` (off-grammar) | `ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE` | `temporal` | service config | [#666](https://github.com/zynax-io/zynax/issues/666) |
| engine-adapter | Temporal host/ns/queue (`localhost:7233`) | `ZYNAX_ENGINE_ADAPTER_TEMPORAL_{HOST,NAMESPACE,TASK_QUEUE}` | env-specific | ConfigMap | [#667](https://github.com/zynax-io/zynax/issues/667) |
| api-gateway | `API_KEY`, `DEV_INSECURE` | `ZYNAX_GATEWAY_API_KEY` / `_DEV_INSECURE` | unset / `false` | **Secret** / values | — |
| http-adapter example | `registry_endpoint: agent-registry:9091` (wrong) | adapter `registry_endpoint` | `agent-registry:50052` | ConfigMap/values | [#665](https://github.com/zynax-io/zynax/issues/665) |
| compose `temporal` | `POSTGRES_PWD: temporal` (dev creds inline) | `ZYNAX_TEMPORAL_DB_PASSWORD` | — | **Secret** | — |
| Dockerfiles ×5 | `golang:1.26.3-alpine`, distroless tag | build-arg `GO_VERSION` / `BASE_IMAGE` | pinned | `infra/docker` + Renovate | [#668](https://github.com/zynax-io/zynax/issues/668) |
| `Makefile:8` | hardcoded `GO_SERVICES` incl. stubs | derive from `go.work` | — | Makefile | [#663](https://github.com/zynax-io/zynax/issues/663) |

---

## 8. Maintainability Scorecard

| Dimension | Score | Rationale | Target after roadmap | Issues |
|---|---:|---|---:|---|
| **Config maintainability** | **4 / 10** | 2 mechanisms, 5 naming conventions, stale default, misnamed metrics port | 9 | [#661](https://github.com/zynax-io/zynax/issues/661) [#666](https://github.com/zynax-io/zynax/issues/666) [#667](https://github.com/zynax-io/zynax/issues/667) |
| **Dependency maintainability** | **8 / 10** | Versions aligned, Renovate mature, hermetic builds. Not enforced. | 9 | [#669](https://github.com/zynax-io/zynax/issues/669) |
| **Operational complexity** | **5 / 10** | Excellent local DX; inconsistent probes, metrics on 1/5, no K8s | 8 | [#241](https://github.com/zynax-io/zynax/issues/241) [#491](https://github.com/zynax-io/zynax/issues/491) |
| **Developer experience** | **8 / 10** | Broken sbom/scan targets, single-service debug footgun | 9 | [#661](https://github.com/zynax-io/zynax/issues/661) [#662](https://github.com/zynax-io/zynax/issues/662) |
| **Kubernetes readiness** | **2 / 10** | No charts/manifests exist; "K8s-native" is aspirational | 8 | [#241](https://github.com/zynax-io/zynax/issues/241)–[#245](https://github.com/zynax-io/zynax/issues/245) |
| **Scalability** | **5 / 10** | Stateless + Temporal/NATS design sound; unproven, no resource/HPA | 8 | [#241](https://github.com/zynax-io/zynax/issues/241) [#491](https://github.com/zynax-io/zynax/issues/491) |
| **Security posture** | **8 / 10** | Distroless, SBOM, gitleaks, pinned actions. Open: cosign/SLSA, broken local scan | 9 | [#489](https://github.com/zynax-io/zynax/issues/489) [#662](https://github.com/zynax-io/zynax/issues/662) |
| **CI/CD maturity** | **8 / 10** | Path-filtered, SHA-pinned, proto/stub-drift gates. Add helm/kubeconform + signing | 9 | [#489](https://github.com/zynax-io/zynax/issues/489) [#669](https://github.com/zynax-io/zynax/issues/669) |
| **Overall** | **≈ 6 / 10** | Strong M4 foundation with clear additive path to production | **≈ 8.5** | |

---

## 9. Non-findings (claims verified and rejected)

Reported honestly so the team can trust the rest:

- **"Compose env vars don't match the code."** *False alarm.* The `ZYNAX_BROKER_*` / `ZYNAX_GW_*`
  compose names look unmatched against struct tags like `GRPC_PORT`, but `envconfig.Process("ZYNAX_BROKER", …)`
  prepends the prefix, so they reconcile correctly. The naming is *inconsistent across services* (a real
  finding, §1.2) but *not broken* in compose.

- **"Tools image name drifts between README and Makefile."** *False alarm.* The cloned `README.md`
  and `Makefile` agree on `ghcr.io/zynax-io/zynax/tools`.

- **"Dependency versions have drifted across modules."** *False.* They are identical across all 11
  modules; the only real concern is lack of *enforcement* (§4, B4).

---

*Guiding bias throughout: simplicity over abstraction, explicitness over magic, operational clarity
over flexibility. Every recommendation is additive and reversible, reuses tooling already trusted in
this repo (envconfig, Helm, Renovate, syft, Argo CD), and avoids introducing a config DSL.*
