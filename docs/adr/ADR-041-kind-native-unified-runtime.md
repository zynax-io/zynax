<!-- SPDX-License-Identifier: Apache-2.0 -->
# ADR-041: Local-Kubernetes (kind) as the unified runtime model — Docker Compose retired as the primary path

**Status:** Accepted  **Date:** 2026-06-25
**Related:** ADR-039 (CRD-native scheduler — the Phase-2 unifier), ADR-040 (Kubernetes-native delegation boundary), ADR-037 (rejected zero-Temporal eval engine; superseded by #1456), ADR-015 (pluggable workflow engines), ADR-020 (mTLS via cert-manager), ADR-026 (Postgres chart). Resolves the open decision in EPIC #1370 / O26 / #1495.

---

## Context

Zynax runs in **two divergent runtimes** today:

- **Docker Compose** — the laptop/dev/demo on-ramp (`make run-local`, `make demo`). Discovery is push-based gRPC `RegisterAgent`; there is no Kubernetes API server, so there are no CRDs, no NetworkPolicy, no cert-manager mTLS, no HPA, and the **Argo engine cannot run at all** (it is Kubernetes-native).
- **Helm / Kubernetes** — the production control plane (prod-tested in M6, v0.5.0). Full manifests, mTLS, RBAC, NetworkPolicy, both Temporal and Argo engines.

Three accepted decisions now pull hard toward Kubernetes:

1. **ADR-039** makes the `Agent` Custom Resource the single source of truth and turns `agent-registry` into a stateless, informer-backed scheduler. Its own Consequences state this *"removes the Docker-Compose discovery path … and directly erodes the EPIC #1370 first-run wedge,"* and require the M8 cutover to either retarget first-run to local-K8s **or** keep a flagged Compose "lite" mode. This is the load-bearing open question (tracked as O26 / #1495).
2. **ADR-040** establishes that Zynax delegates generic primitives to Kubernetes and treats *building a Zynax service to replace a Kubernetes primitive as an anti-pattern.* Keeping push-registration alive purely for Compose is exactly such a parallel mechanism.
3. The **engine-portability wedge** ("write once, run on Temporal or Argo") cannot be demonstrated on Compose, because Argo needs Kubernetes.

The repository already has everything required for a Kubernetes-everywhere model: prod-tested Helm charts under `helm/`, a kind-based e2e harness (`scripts/e2e/cluster-up.sh`, `.github/workflows/e2e-smoke.yml`) that deploys the full stack on **kind** with a dual-engine (Temporal + Argo) matrix, and a CRD-native scheduler **already spiked** (`spike/adr-039-crd-scheduler-proof`, 7/7 checks green on kind). What is missing is only a *developer-facing* local-K8s path — kind is CI-only and undocumented for laptops.

Maintaining Compose as a first-class parallel runtime therefore means two discovery code paths, ongoing behavior drift (Compose can never mirror prod), and carrying deprecated push RPCs indefinitely — directly against ADR-039/040.

---

## Decision

**We will standardize every environment — local, CI, and production — on Kubernetes, using a local single-node cluster (`kind`) as the developer/demo runtime, and retire Docker Compose as the sanctioned primary path.**

1. **One runtime model.** Local development, the demo, and CI all run on Kubernetes. Locally this is **kind** (the tool already proven in our e2e harness); k3s / k3d / managed clusters are the *same model* at larger scale. "It runs the same way everywhere" becomes literally true.
2. **Retire Docker Compose as the primary local runtime.** It is **deprecated** during the transition and **removed** once the kind path reaches parity for the developer and demo journeys. No new feature work targets Compose.
3. **One-command UX is preserved by wrapping the cluster lifecycle.** A single `make demo` creates the kind cluster, `kind load`s images, installs the Helm umbrella, runs the hero workflow, shows the result, and offers teardown — the user still types **one command**, even though the runtime is Kubernetes. A `make kind-up` / `make kind-down` pair fronts `scripts/e2e/cluster-up.sh`.
4. **Two-phase execution:**
   - **Phase 1 (now, M7→M8 bridge):** make kind + the existing Helm charts the developer/demo runtime; rewrite README/quickstart/local-dev to lead with the kind path; re-point the EPIC #1370 closeout stories (O18–O26) to kind. Discovery stays push-based, but now runs *inside the cluster*, so the **topology already matches production**.
   - **Phase 2 (M8, ADR-039):** swap discovery push → `Agent` CRD. At that point kind ≡ k3s ≡ prod down to scheduling semantics, and the push `RegisterAgent` RPCs are hard-removed (their only remaining consumer — Compose — is gone).
5. **Engines.** kind-local unlocks **Argo locally** alongside Temporal, so the engine-portability wedge ("write once, run on Temporal *or* Argo") is demonstrable on a laptop — which Compose never could. The lightweight Temporal eval profile (#1456) continues as an in-cluster Temporal *dev* profile to keep first-run light.

---

## Rationale

| Option | Assessment |
|--------|------------|
| **kind everywhere; retire Compose** | ✅ **Chosen** — one runtime model; local mirrors prod; unlocks Argo-local; aligns with ADR-039/040; deletes the dual-discovery + deprecated-RPC debt; CNCF-idiomatic local-dev story. |
| Dual-path: Compose (eval) + K8s (prod) behind a `REGISTRY_BACKEND` flag | ✗ Rejected — two discovery code paths, permanent behavior drift, carries deprecated push RPCs indefinitely, and stands up a Zynax-built primitive that ADR-040 explicitly rejects. |
| Compose-only / status quo | ✗ Rejected — cannot run Argo, structurally diverges from prod, and ADR-039 removes its discovery path in M8 regardless. |
| Embed k3s *inside* Compose | ✗ Rejected — that is "run a local cluster" with extra wrapping; kind already does this cleanly and is proven in CI. |

---

## Consequences

- **Positive:**
  - A single runtime model; local **mirrors production**, so K8s-only behavior (NetworkPolicy, cert-manager mTLS, RBAC, CRDs, scheduling) is exercised on the laptop instead of first appearing in prod.
  - **Argo is runnable locally**, making the engine-portability wedge demonstrable end-to-end on a laptop.
  - Deletes the dual-discovery maintenance and lets ADR-039 hard-remove the push RPCs on schedule.
  - The EPIC #1370 onboarding investment **converges with** the production path instead of diverging from it; ADR-039's load-bearing open question (O26/#1495) is resolved here.
  - CNCF-idiomatic: kind is the standard cloud-native local-dev story.
- **Negative / trade-off (load-bearing):**
  - The laptop prerequisite rises from **"Docker only"** to **"Docker + kind + kubectl + Helm + ~4 CPU / 8 GB RAM."** The "Docker-only, five-minute" framing weakens, and cold first-run is slower (cluster create + image load + Helm install) than `docker compose up`. **Mitigation:** a single `make demo` wraps the whole lifecycle (one command preserved); publish the resource floor up front; `kind load` prebuilt images to cut cold-start; keep the lightweight in-cluster Temporal dev profile.
  - A genuinely **zero-infrastructure** path (no Docker, no cluster — the binary-only idea behind #1359/#1456) is explicitly **out of scope** for first-run; deferred to M-dx, revisited only if a binary-only run becomes a hard requirement.
- **Neutral / follow-up required:**
  - Re-point EPIC #1370 closeout stories O18–O26 (#1488–#1495, #1463) to kind; add `make kind-up` / `make demo`-on-kind; rewrite README / `docs/quickstart.md` / `docs/local-dev.md` to the kind path; surface `scripts/e2e` as supported developer tooling.
  - **Deprecate then remove** `infra/docker-compose/**` and its Make targets on a published timeline.
  - **Audit other-milestone issues** for Compose assumptions and re-align them with this model (this ADR triggers that sweep).
  - Phase-2 CRD cutover and push-RPC removal remain owned by **ADR-039** (M8).

---

## Amendment (2026-07-01): CLI-native lifecycle entry points (`zynax up` / `zynax down`)

Decision #3 above wraps the kind cluster lifecycle behind `make kind-up` / `make kind-down` (fronting `scripts/e2e/cluster-up.sh` / `cluster-down.sh`). This amendment adds **CLI-native** entry points that wrap the **same** scripts:

- **`zynax up`** brings the full umbrella up on a local kind cluster; **`zynax down`** tears it down. They shell out (streaming) to `scripts/e2e/cluster-up.sh` / `cluster-down.sh` — the identical, CI-proven, idempotent path — mapping flags 1:1 to the scripts' existing env-var contract (`--profile`→`PROFILE`, `--engine`→`E2E_ENGINE`, `--no-load-images`→omit `KIND_LOAD_IMAGES`, `--cluster-name`→`CLUSTER_NAME`, `--namespace`→`NAMESPACE`).
- They are **`make`-free and location-independent**: the `zynax` binary resolves the repo root (`--repo-root` flag → `ZYNAX_REPO_ROOT` → walk-up sentinel `scripts/e2e/cluster-up.sh`), so bring-up works from any subdirectory of a checkout, and fails with a clone-pointing error outside one.

**This does not change the runtime decision.** kind + the production Helm charts remain the single runtime model; `zynax up`/`down` are a second, more discoverable *entry point* to the lifecycle Decision #3 already established — not a new runtime, not a divergent topology, and (per ADR-041) never a Docker-Compose path. A repo checkout is still assumed ("clone → one command"); a clone-free binary-only bring-up remains out of scope (deferred to M-dx, consistent with the zero-infrastructure trade-off above).

Tracked by **EPIC #1561 (M7.V)** — stories #1562 (`reporoot` + `zynax down`), #1563 (`zynax up`), #1564 (docs + this amendment). REASONS Canvas: `docs/spdd/1561-zynax-up-down/canvas.md`.
