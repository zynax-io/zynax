<!-- SPDX-License-Identifier: Apache-2.0 -->

# code-review-rank — atomic, GitOps-ready package

Deploys the **code-review-with-rank** workflow end-to-end on top of an existing
Zynax platform, in one `kubectl apply -k`. It adds the capability-provider infra
(a local Ollama server + the `llm-adapter`) and submits the logical manifests
(an `AgentDef` declaring both capabilities + the rank `Workflow`) so the workflow
runs to a ranked code review with zero further steps.

## What it contains

| File | Kind | Purpose |
|------|------|---------|
| `ollama.yaml` | Deployment + PVC + Service | In-cluster Ollama, serves `qwen2.5-coder:0.5b`; pulls it on first start, caches on the PVC; readiness gated on the model being present. |
| `llm-adapter-config.yaml` | ConfigMap | The adapter's YAML config: capabilities `codereview` + `summarize`, ollama provider at `http://ollama:11434`. |
| `llm-adapter.yaml` | Deployment + Service | The `llm-adapter` gRPC capability provider; self-registers both capabilities with the agent-registry (advertises `llm-adapter:50070`). |
| `agentdef.yaml` | ConfigMap | The logical `AgentDef` (both capabilities) the apply-Job POSTs to the gateway. |
| `workflow.yaml` | ConfigMap | The self-contained rank `Workflow` (diff baked inline) the apply-Job POSTs. |
| `apply-job.yaml` | Job | Waits for ollama's model to be pulled **and** the adapter to be serving, then POSTs the AgentDef and Workflow to the api-gateway with a Bearer token; prints the `run_id`. Gating on the model first makes a cold `apply -k` succeed on the first run. |
| `kustomization.yaml` | Kustomization | Bundles all of the above into namespace `zynax`. |

## Prerequisites

- A running Zynax platform in namespace `zynax` (api-gateway, workflow-compiler,
  engine-adapter, task-broker, agent-registry, temporal, event-bus).
- The platform's gateway auth secret `zynax-gw-api-key` (key `api-key`) in the
  `zynax` namespace — the apply-Job reads it for the `Authorization: Bearer` header.

## Deploy

```bash
kubectl apply -k infra/packages/code-review-rank/
```

This brings up Ollama (first run pulls the ~398 MB model — under a minute), then
the `llm-adapter` (which self-registers `codereview` + `summarize`), then the
apply-Job submits the AgentDef and Workflow and prints the `run_id`.

### Watch it come up

```bash
kubectl -n zynax rollout status deploy/ollama --timeout=600s
kubectl -n zynax exec deploy/ollama -- ollama list          # qwen2.5-coder:0.5b present
kubectl -n zynax rollout status deploy/llm-adapter --timeout=300s
kubectl -n zynax logs deploy/llm-adapter | grep -i registr   # capabilities registered
kubectl -n zynax wait --for=condition=complete job/code-review-rank-apply --timeout=300s
kubectl -n zynax logs job/code-review-rank-apply             # prints RUN_ID=...
```

### Check the result

Poll the run to `WORKFLOW_STATUS_COMPLETED`, then read the ranked review (the
`summarize` step's completion). Using the CLI against a local port-forward:

```bash
kubectl -n zynax port-forward svc/zynax-api-gateway 18080:8080 &
KEY=$(kubectl -n zynax get secret zynax-gw-api-key -o jsonpath='{.data.api-key}' | base64 -d)
zynax --api-key "$KEY" --api-url http://localhost:18080 status workflow <RUN_ID>
zynax --api-key "$KEY" --api-url http://localhost:18080 result <RUN_ID>
```

LLM inference on a 3B CPU model is slow — the `codereview` and `summarize` steps
each take ~30 s to a few minutes. The capability timeout is 300 s.

## Run the workflow

Deploying the package **runs the rank workflow once** automatically (the apply-Job submits it). To run it **again**, or to review **your own** diff:

```bash
# Re-run the bundled review (fresh run_id, same baked-in sample diff):
kubectl -n zynax delete job code-review-rank-apply
kubectl apply -k infra/packages/code-review-rank/   # apply-Job waits for model+adapter, then resubmits

# Review YOUR diff via the CLI (port-forward + the bearer key):
kubectl -n zynax port-forward svc/zynax-api-gateway 18080:8080 &
KEY=$(kubectl -n zynax get secret zynax-gw-api-key -o jsonpath='{.data.api-key}' | base64 -d)
# Either edit the diff baked into workflow.yaml (the `code-review-rank-workflow` ConfigMap),
# or apply your own Workflow that dispatches `codereview` then `summarize`:
zynax --api-key "$KEY" --api-url http://localhost:18080 apply <your-workflow>.yaml
zynax --api-key "$KEY" --api-url http://localhost:18080 logs <RUN_ID> --follow   # tail BEFORE it finishes to see the ranked text
```

`codereview` reviews the diff; its output is fed via data-flow into `summarize`, which ranks the findings by severity. The ranked review is the `summarize` step's completion.

## Configurable Ollama endpoint

To use an **external** Ollama (e.g. on the host) instead of the in-cluster one,
change **one line** in `llm-adapter-config.yaml` — `<docker-host-ip>` is the host
address reachable from inside the cluster (on kind, the bridge-network gateway from
`docker network inspect kind`; note the host's Ollama must bind `0.0.0.0`, not just
localhost):

```yaml
provider:
  ollama_base_url: "http://<docker-host-ip>:11434"   # was http://ollama:11434
```

and scale the in-cluster server down (`kubectl -n zynax scale deploy/ollama --replicas=0`).
The model name is the adjacent `model:` line.

## Flux / GitOps

The package is a plain kustomize overlay, so a Flux `Kustomization` can reconcile
it directly. Commit this directory to your GitOps repo and point a Kustomization
at it:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: code-review-rank
  namespace: flux-system
spec:
  interval: 10m
  path: ./infra/packages/code-review-rank
  prune: true
  targetNamespace: zynax
  sourceRef:
    kind: GitRepository
    name: zynax            # your GitRepository source
  wait: true
```

`prune: true` makes Flux remove the package's resources when the directory is
deleted from Git — mirroring the manual teardown below.

## Teardown

```bash
kubectl delete -k infra/packages/code-review-rank/
```

This removes Ollama, the llm-adapter, the ConfigMaps, and the apply-Job — the
platform itself is left untouched. (The registry entry for `llm-adapter` remains
as an audit record; it is harmless and re-used on the next deploy.)
