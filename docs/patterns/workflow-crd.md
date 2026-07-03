<!-- SPDX-License-Identifier: Apache-2.0 -->

# Authoring & GitOps: the `Workflow` CRD front-end

The `Workflow` custom resource (`zynax.io/v1alpha1`) lets a platform team manage
Zynax workflows the way they manage everything else in Kubernetes — `kubectl
apply` or GitOps-sync a declarative object — **without Zynax becoming a workflow
database**. A controller embedded in the api-gateway reconciles each `Workflow`
CR by calling the *existing* compile→submit path; execution and run state stay in
the engine (Temporal/Argo). The same workflow still graduates across engines —
the CRD is only an authoring surface, so the engine-portability wedge is intact
(ADR-040 §3, ADR-043).

> **What the CR is not:** it is **not** the source of truth for run state. Its
> `status` is a thin mirror (compile/dispatch conditions + the dispatched
> identifiers). For live progress and results, use `zynax status` / `zynax
> result` (or your engine's UI) — never `kubectl get` the CR.

## Enabling the controller

The `Workflow` CRD ships in the api-gateway chart's `crds/` and installs with the
release. The controller itself is **off by default**; enable it in your values:

```yaml
zynax-api-gateway:
  crdController:
    enabled: true      # starts a namespaced, Lease-elected controller
    # watchNamespace: ""   # empty => the pod's own namespace (downward API)
```

It watches only its own namespace (least-privilege `Role`; a cluster-scope watch
is never used).

## Authoring a `Workflow` CR

A `Workflow` CR carries the same state machine as a `kind: Workflow` manifest,
under `spec`, plus an optional `engine` hint. A minimal example
([docs/examples/workflow-crd-sample.yaml](../examples/workflow-crd-sample.yaml)):

```yaml
apiVersion: zynax.io/v1alpha1
kind: Workflow
metadata:
  name: hello-review
  namespace: default
spec:
  engine: temporal          # optional; omit to use the platform default
  initial_state: review
  states:
    review:
      type: normal
      actions:
        - capability: code_review
          input:
            repo: "{{ .trigger.repo }}"
      "on":                  # NOTE the quotes — see the YAML gotcha below
        - event: capability.completed
          goto: done
    done:
      type: terminal
      outputs:
        result: "$.states.review.output.summary"
```

> **YAML gotcha — quote `"on":`.** kubectl's YAML reader treats a bare `on` key
> as a boolean, so an unquoted transition key becomes `true` and the CR is
> rejected. Always write `"on":`. (The `zynax apply --crd` bridge below sidesteps
> this by emitting JSON, so you only need the quotes when authoring CR YAML by
> hand.)

Apply it two ways:

```bash
# Straight kubectl — the GitOps-native path.
kubectl apply -f workflow.yaml

# Or bridge a Workflow *manifest* (apiVersion: zynax.io/v1) to the CR path:
zynax apply --crd --engine temporal workflow-manifest.yaml
```

`zynax apply --crd` converts the manifest to a CR (mapping `apiVersion
zynax.io/v1` → `v1alpha1`, lifting `--engine` into `spec.engine`) and
`kubectl apply`s it on your current context.

## Inspecting

```console
$ kubectl get workflows.zynax.io
NAME           ENGINE     DISPATCHED   RUN-ID               AGE
hello-review   temporal   True         wf-af47154cc5206e31  12s
```

The `status` conditions tell you the reconcile outcome:

- `Compiled` — the manifest compiled to the engine-agnostic IR.
- `Dispatched` — the run was submitted to the engine; `status.runID` is the
  engine run id.

A `Compiled=False` condition carries the compile error — fix the spec and
re-apply; the controller does not crash-loop on a bad manifest.

## GitOps

The CR is a plain Kubernetes object, so it drops straight into a GitOps flow:

1. Commit the `Workflow` CR YAML to your config repo.
2. Argo CD / Flux syncs it to the cluster.
3. The controller reconciles it → compile→submit → a run is dispatched.
4. `kubectl get workflows.zynax.io` (or your GitOps UI) shows the definition and
   its last compile/dispatch status; drift is detected for free.

**Re-sync is safe.** Reconcile is gated on `metadata.generation` vs
`status.observedGeneration` (ADR-043 §4): a GitOps resync, a controller restart,
or a leader change of an unchanged CR starts **no** new run. A run is (re)dispatched
only when you actually change the `spec`.

## See also

- [docs/adr/ADR-043-workflow-crd-front-end.md](../adr/ADR-043-workflow-crd-front-end.md) — the decision.
- [docs/adr/ADR-040-kubernetes-native-delegation-boundary.md](../adr/ADR-040-kubernetes-native-delegation-boundary.md) §3 — why the CRD is a thin front-end.
- [docs/patterns/agent-crd-migration.md](agent-crd-migration.md) — the sibling `Agent` CRD (M8.C).
