<!-- SPDX-License-Identifier: Apache-2.0 -->

# Quick Start — Zynax on kind

The single how-to for running Zynax locally on a [kind](https://kind.sigs.k8s.io/) cluster that
mirrors production. The **five-minute golden path** (boot → first workflow → switch engines) lives
in the root [README "See it run"](../README.md#see-it-run--the-five-minute-golden-path); this page
goes one level deeper — the real-model code-review demo, the engine-portability switch, scaling
beyond kind, and observability.

> **Docker Compose is deprecated** as the local runtime ([ADR-041](adr/ADR-041-kind-native-unified-runtime.md)):
> it cannot run the Argo engine and structurally diverges from production. The legacy Compose runbook
> is kept for reference only at [running-with-docker-compose.md](running-with-docker-compose.md).

---

## Prerequisites

- **Docker** Engine / Docker Desktop.
- **[kind](https://kind.sigs.k8s.io/)**, **`kubectl`**, and **[Helm](https://helm.sh/)** on your PATH.
- A host with **~4 CPU / 8 GB RAM** (the resource floor — see [ADR-041](adr/ADR-041-kind-native-unified-runtime.md)).
- The **`zynax` CLI** — `make install-cli` (builds → `~/bin/zynax`; ensure `~/bin` is on PATH), or a
  release binary (see the root [README § zynax CLI](../README.md#zynax-cli--user-facing-binary)).

No Go, Python, or `buf` is needed locally — `make demo` builds and loads images inside containers.

---

## 1. Boot the cluster

```bash
git clone https://github.com/zynax-io/zynax.git
cd zynax

make demo            # create kind cluster → load images → install Helm umbrella → wait rollout
```

`make demo` runs the full lifecycle and prints a **Platform ready** banner with the gateway URL
(`http://localhost:8080`) once every Deployment is Available. The banner is never a premature "go"
signal — it is gated on `kubectl rollout status` for all services.

Lean laptop? `PROFILE=lite make demo` swaps the durable Temporal trio for a single in-cluster dev
Temporal and disables the event-bus / memory-service.

---

## 2. Reach the gateway

The api-gateway is auth-enabled. Fetch the bearer key the cluster provisioned:

```bash
export ZYNAX_API_KEY=$(kubectl -n zynax get secret zynax-gw-api-key -o jsonpath='{.data.api-key}' | base64 -d)
```

The CLI defaults to `--api-url http://localhost:8080` (the kind NodePort `30080` mapped to host
`8080`). The NodePort can be reset by kube-proxy on repeat runs; if `localhost:8080` is flaky,
port-forward to a **different** local port (the NodePort already holds `8080`) and target it with
`--api-url`:

```bash
kubectl -n zynax port-forward svc/zynax-api-gateway 18080:8080 &
# then add --api-url http://localhost:18080 to the zynax commands below
```

---

## 3. First success — the zero-dependency workflow

[`hello-world.yaml`](../spec/workflows/examples/hello-world.yaml) dispatches the in-cluster `echo`
capability and completes — **no model, no secret**. Validate it locally, then submit:

```bash
zynax validate spec/workflows/examples/hello-world.yaml   # static + data-flow checks (no gateway)
zynax apply spec/workflows/examples/hello-world.yaml       # submit
# run_id: wf-<hex>

zynax status workflow wf-<hex>
# WORKFLOW_STATUS_COMPLETED

zynax logs wf-<hex>                                        # the lifecycle events for the run
```

`WORKFLOW_STATUS_COMPLETED` is your first success: the engine dispatched the in-cluster `echo`
capability and ran to a terminal state with zero secrets.

> **No need to shuttle the id around.** `apply` records your most recent run locally, so a bare
> `zynax status` / `zynax logs` (no id) targets it. An explicit id always overrides.

---

## 4. The real-model code-review demo (optional)

To run a workflow where a real model reviews a git diff, you need a model available to the
llm-adapter. Pull it once on the host so the in-cluster adapter can reach it:

```bash
ollama pull qwen2.5-coder:3b           # default model; see infra/docker-compose/ollama/llm-adapter.config.yaml
```

Then apply [`code-review-ollama.yaml`](../spec/workflows/examples/code-review-ollama.yaml) — it runs
to completion from the CLI alone (its initial state dispatches the `codereview` capability over a
real git diff, then transitions to a terminal state):

```bash
zynax apply spec/workflows/examples/code-review-ollama.yaml
zynax logs <run-id> --follow            # stream every state + step output as it completes
zynax result <run-id>                   # print just the model's review text
```

> Other examples under [spec/workflows/examples/](../spec/workflows/examples/) (e.g.
> `code-review.yaml`) are **reference specs** that wait on external GitHub/review events — use them to
> learn the data-flow patterns, but they do not run to completion from the CLI alone. Drive one
> forward with `zynax events publish <run-id> review.approved --data reviewer=alice`.

---

## 5. Switch engines — the portability wedge

The same manifest runs unchanged on Temporal **or** Argo — selection flows through the cluster, never
the workflow file:

```bash
ENGINE=argo make demo     # (or E2E_ENGINE=argo make demo)
```

This is the wedge: write once, run on whichever engine your org already operates. Argo is only
runnable locally because the runtime is Kubernetes ([ADR-041](adr/ADR-041-kind-native-unified-runtime.md),
[#1370](https://github.com/zynax-io/zynax/issues/1370)).

---

## 6. Scaling beyond kind

kind, k3s / k3d, and managed Kubernetes are the **same runtime model** at larger scale — the Helm
umbrella `make demo` installs is the production chart. Point `kubectl` at any cluster and
`helm upgrade --install` the same umbrella. See [docs/local-dev.md](local-dev.md) and the Helm chart
README under `infra/helm/` for cluster targeting, values overrides, and multi-namespace deploys.

---

## 7. Observability (optional)

Telemetry is **off by default** and env-gated. To see traces, metrics, logs, and APM correlated by
`trace_id`/`span_id` in the Uptrace UI, follow [docs/observability/](observability/) (`opentelemetry.md`
and `uptrace.md`) for the in-cluster OTel collector + Uptrace wiring. The golden path above works
without it.

---

## 8. Tear down

```bash
make kind-down       # delete the kind cluster (wraps scripts/e2e/cluster-down.sh)
zynax down           # CLI-native equivalent — same script, make-free, from any directory
```

---

## Command reference

Every command this guide shows is a real `zynax` subcommand. Verify the surface yourself with
`zynax --help` (and `zynax <cmd> --help`); the source of record is [cmd/zynax/cmd/](../cmd/zynax/cmd/).

| Command | Purpose | Key flags |
|---------|---------|-----------|
| `zynax up` | Create/reuse a kind cluster + deploy the platform | `--profile full\|lite`, `--engine temporal\|argo`, `--no-load-images`, `--cluster-name`, `--namespace`, `--repo-root` |
| `zynax down` | Delete the local kind cluster | `--cluster-name`, `--repo-root` |
| `zynax validate <file>` | Local schema + data-flow checks (no gateway) | `--schema-dir`, `--format text\|json` |
| `zynax init workflow\|expert [name]` | Scaffold a manifest from a template | `-o/--output`, `--template-dir` |
| `zynax apply <file>` | Submit a manifest to the gateway | `--dry-run`, `--engine` |
| `zynax status workflow <run-id>` | Status (exit 0 terminal, 2 running) | — |
| `zynax logs <run-id>` | Stream lifecycle events | `--follow/-f`, `--format text\|json` |
| `zynax result <run-id>` | Print the capability output | — |
| `zynax get workflow <run-id>` | Full run snapshot | — |
| `zynax delete workflow <run-id>` | Cancel/remove a run | — |
| `zynax events publish <run-id> <event-type>` | Inject an event into a running workflow | `--data key=value` (repeatable) |

Global flags (any subcommand): `--api-url` (`$ZYNAX_API_URL`, default `http://localhost:8080`),
`--api-key` (`$ZYNAX_API_KEY`, the gateway bearer token), `--insecure`.

---

## What next

- **[Developer Guide](developer-guide.md)** — the Make targets and daily workflow.
- **[Local Development Guide](local-dev.md)** — CLI install options and cluster targeting.
- **[Human-validation guide](contributing/human-validation-guide.md)** — how to validate a change
  actually runs (the standard the demo path is checked against).
- **[docs/casts/](casts/)** — recorded terminal walkthroughs of these flows.
- **[spec/workflows/examples/](../spec/workflows/examples/)** — the reference workflows and their
  data-flow patterns.
</content>
</invoke>
