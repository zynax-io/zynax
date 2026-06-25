<!-- SPDX-License-Identifier: Apache-2.0 -->
# Lean kind resource comparison (ADR-041) — measured

How much the local runtime shrinks once the stack goes kind-native and the lean
profile drops the components Kubernetes makes redundant. ADR-041 retires Docker
Compose as the primary path; this quantifies what the move costs and what the
lean profile (`PROFILE=lite`) buys back.

**Measured:** 2026-06-25, single host (16 CPU / 15 GiB RAM), `kind v0.23.0`
(node `v1.29.2`), `helm v3.21.0`, Docker Compose. Workload = the deterministic
`echo` hero workflow (`spec/workflows/examples/e2e-demo.yaml`) — no model, so the
demo time reflects **stack/orchestration** latency, not LLM inference. Reproduce
with `scripts/bench/stack-resources.sh` (see `scripts/bench/README.md`).

## Results

| Stack | Pods / containers | Workload CPU (idle) | Workload RAM (idle) | Whole-machine RAM¹ | Whole-machine CPU¹ | PVC reserved | Image/layer disk² | Demo echo (cold / warm) |
|-------|-------------------|---------------------|---------------------|--------------------|--------------------|--------------|-------------------|-------------------------|
| **Docker Compose** (base, `make run-local`) | 13 | ~0 (negligible) | ~520 MiB | **~0.52 GiB** (no control plane) | negligible | named vols (~0.14 GiB) | ~1.3 GiB (1.15 img + 0.14 vol) | 3s / 3s |
| **full-kind** (3-node, `PROFILE=full`) | 18 | 35m | 369 MiB | **~2.50 GiB** (2497 MiB) | 195m | **11.0 GiB** | ~5.9 GiB | 3s / 4s |
| **lean-kind** (3-node, `PROFILE=lite`) | 8 | 16m | 253 MiB | **~2.03 GiB** (2075 MiB) | 138m | 2.0 GiB | ~5.3 GiB | 3s / 3s |
| **lean-kind** (1-node, `PROFILE=lite` today³) | 8 | 11m | 209 MiB | **~1.40 GiB** (1430 MiB) | 126m | 2.0 GiB | **~2.9 GiB** | 9s / 3s |

¹ `kubectl top nodes` summed across the kind node containers — the true
machine cost (control-plane + kube-system + workloads + per-node
kubelet/containerd). Compose has no control plane, so its whole-machine figure is
just `docker stats`.
² kind: `du /var/lib/containerd` across every node (the same images load into
each node). Compose: sum of unique image sizes (upper bound; shared layers
dedupe on disk) + named-volume sizes.
³ `PROFILE=lite` uses the single-node `kind-config-lite.yaml`; `PROFILE=full`
keeps the 3-node `kind-config.yaml` it needs for headroom.

## The reduction: full-kind → lean-kind (single-node)

| Dimension | full-kind (before) | lean-kind 1-node (after) | Reduction |
|-----------|--------------------|--------------------------|-----------|
| Pods | 18 | 8 | **−56%** |
| Whole-machine RAM | ~2.50 GiB | ~1.40 GiB | **−44%** |
| Workload RAM | 369 MiB | 209 MiB | −43% |
| Whole-machine CPU | 195m | 126m | −35% |
| PVC reserved | 11.0 GiB | 2.0 GiB | **−82%** |
| Image/layer disk | ~5.9 GiB | ~2.9 GiB | **−51%** |
| Demo (echo, warm) | 4s | 3s | same (~3s) |

## What changed, and the two levers

The lean profile pulls on **two independent levers** — together they roughly
**halve** the kind footprint:

1. **Component removal** (`values-lite.yaml` + `temporal-dev.yaml`): the 5-pod
   Temporal chart + admintools + Postgres-backed schema Job → **one** in-memory
   `temporal server start-dev` pod; drop **memory-service + Redis** and
   **event-bus + NATS**; trim the **Postgres PVC 10 GiB → 2 GiB**. This is most of
   the pod cut (18→8) and the PVC cut (11→2 GiB, −82%).
2. **Single-node topology** (`kind-config-lite.yaml`): the full profile needs
   3 nodes for headroom; the lean stack fits on **one**. That removes two nodes'
   worth of kubelet/containerd/kindnet/kube-proxy overhead **and** stops the
   service images loading into three nodes' containerd. This lever alone takes
   whole-machine RAM 2.03 → 1.40 GiB and disk 5.3 → 2.9 GiB (−45%).

## Key findings (honest reading)

- **The kind tax is real and irreducible.** Even lean single-node kind (~1.4 GiB)
  sits **above** Docker Compose (~0.5 GiB). The gap (~0.9 GiB) is the Kubernetes
  control-plane (kube-system ~0.45 GiB) plus per-node kubelet/containerd. No lean
  profile escapes it — it is the deliberate price of "local mirrors production"
  (ADR-041). Lean single-node kind roughly **halves the gap** vs full 3-node kind.
- **The demo runs just as fast.** ~3s warm on every stack — trimming the stack
  does not slow orchestration. (1-node cold-start is one-off higher at 9s:
  port-forward + first dispatch on a cold single node.)
- **At the app-workload level, Compose is actually the heaviest** (~520 MiB) — it
  bundles durable Temporal (auto-setup, ~170 MiB), a *second* Postgres, the
  Temporal UI, and extra adapters. Lean kind's app workload is **209 MiB**.
  Compose only "wins" on the whole-machine number because it carries no
  orchestration layer.
- **The full e2e profile is already trimmed** (`values-e2e.yaml` caps Temporal
  roles and `numHistoryShards`), so the workload-RAM deltas are smaller than a
  naive "5 pods → 1 pod" estimate; the dominant wins are **pods, PVC, and disk**.

## Caveats

- `kubectl top` reports working-set (includes some page cache); `docker stats`
  reports RSS-ish. Cross-runtime RAM is therefore approximate — hence both a
  per-engine workload figure and a whole-machine figure.
- Single measurement, at-rest + one warm demo run — directional, not a
  statistical benchmark. Absolute numbers scale with the host (this one is well
  above the 4 CPU / 8 GiB floor); the **ratios** are the takeaway.
- agent-registry still runs Postgres here; ADR-039 (M8) makes it stateless and
  drops that DB — a further cut not yet reflected.

## Raw harness rows (auto-appended by `scripts/bench/stack-resources.sh`)

| Profile | Pods/containers | CPU | Memory | PVC reserved | Image+layer disk | Demo (apply→done) |
|---------|-----------------|-----|--------|--------------|------------------|-------------------|
| full-kind | 18 | 35m (used) | 369Mi (used) | 11.0Gi | 5904MB | 3s cold / 4s warm |
| lean-kind | 8 | 16m (used) | 253Mi (used) | 2.0Gi | 5328MB | 3s cold / 3s warm |
| compose | 13 | ~0% (stats) | 516Mi (stats) | n/a (named vols) | ~1.3GB | 3s cold / 3s warm |
| lean-kind-1node | 8 | 11m (used) | 209Mi (used) | 2.0Gi | 2929MB | 9s cold / 3s warm |
