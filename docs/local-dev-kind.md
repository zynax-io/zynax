<!-- SPDX-License-Identifier: Apache-2.0 -->
# Local development on kind (ADR-041)

ADR-041 makes a local **kind** cluster the unified developer/demo runtime —
local runs on the same prod-mirroring Helm charts as CI and production, and
Docker Compose is retired as the primary path. One command still does
everything; the runtime underneath is now Kubernetes.

> Prerequisites: Docker, `kind`, `kubectl`, `helm`. Resource floor ~4 CPU / 8 GB
> RAM (the lean profile is lighter — see below).

## One command

```bash
make demo                 # full prod-mirroring stack on kind, runs the hero workflow
make demo PROFILE=lite    # the lean stack — same workflow, far lighter (see below)
```

`make demo` creates a kind cluster, side-loads the local service images,
installs the `zynax-umbrella` chart, waits for every Deployment, runs the echo
hero workflow against the gateway on `http://localhost:8080`, and prints
**Platform ready**. Tear down with `make kind-down`.

## Cluster lifecycle (without the hero run)

```bash
make kind-up                 # full profile (3-node, prod-mirroring)
make kind-up PROFILE=lite    # lean profile (single-node, trimmed)
make kind-down               # delete the cluster
```

## CLI-native lifecycle (`zynax up` / `zynax down`)

The `zynax` CLI fronts the **same** lifecycle without `make`, and works from any
directory inside a checkout:

```bash
zynax up                     # = make kind-up (full profile)
zynax up --profile lite      # lean single-node profile
zynax up --engine argo       # same platform on the Argo engine — the wedge, one flag
zynax down                   # = make kind-down
```

Both wrap the **same** `scripts/e2e/cluster-up.sh` / `cluster-down.sh` as the
`make` targets ([ADR-041](adr/ADR-041-kind-native-unified-runtime.md)), so the
runtime is identical — `zynax up` is just a discoverable, `make`-free entry
point that resolves the repo root itself. Flags map 1:1 to the script env
(`--profile`→`PROFILE`, `--engine`→`E2E_ENGINE`, `--no-load-images` omits
`KIND_LOAD_IMAGES`, `--cluster-name`, `--namespace`). Outside a checkout, point
at one with `--repo-root` or `$ZYNAX_REPO_ROOT`. After bring-up, run a workflow
with `zynax apply` (see the README golden path).

## Two profiles

| | `PROFILE=full` (default) | `PROFILE=lite` |
|---|--------------------------|----------------|
| Topology | 3-node (`kind-config.yaml`) | **single-node** (`kind-config-lite.yaml`) |
| Temporal | 5-pod chart + admintools + schema Job | **1** in-memory `start-dev` pod (`scripts/e2e/manifests/temporal-dev.yaml`) |
| event-bus + NATS | on | **off** |
| memory-service + Redis | on | **off** |
| Postgres PVC | 10 GiB | **2 GiB** |
| Pods | 18 | **8** |
| Whole-machine RAM | ~2.5 GiB | **~1.4 GiB** |
| Image/layer disk | ~5.9 GiB | **~2.9 GiB** |

`full` mirrors production and is what CI runs (it also unlocks
`E2E_ENGINE=argo`). `lite` is the lean laptop profile: same charts and images,
the components the hero `echo` workflow does not exercise removed, and one node
instead of three. The hero workflow runs identically (~3s) on both. Measured
before/after numbers and the methodology: **`docs/benchmarks/kind-lean-resources.md`**.

## How it works

`make demo` → `scripts/demo/kind-demo.sh` → `scripts/e2e/cluster-up.sh` (the
single bring-up source of truth, shared with the CI e2e harness). The `PROFILE`
env flows through all three. `lite` layers `scripts/e2e/values-lite.yaml` over
`values-e2e.yaml`, applies the dev-Temporal manifest in place of the chart, and
selects the single-node kind config.

## Engine portability

The same workflow runs on Temporal or Argo on the same cluster:

```bash
E2E_ENGINE=argo make demo     # installs the Argo Workflows control plane, same workflow
```

## Legacy Docker Compose (deprecated)

`make demo-compose` still runs the old Compose Ollama demo, but Compose is
deprecated under ADR-041 and receives no new feature work. Prefer `make demo`.
