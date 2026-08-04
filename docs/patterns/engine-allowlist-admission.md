<!-- SPDX-License-Identifier: Apache-2.0 -->

# Engine allow-list: Kubernetes admission policy

Which engines a namespace may use ("this namespace may use Temporal but not
Argo") is governed by a **CEL `ValidatingAdmissionPolicy`** bound to the
`Workflow` custom resource (ADR-045, M8.G) — standard, auditable,
GitOps-diffable Kubernetes policy, not Zynax-specific code. The compiler keeps
only engine-*fit* intelligence (capability↔engine matching); the coarse
allow-list is the platform's job.

## How it fits together

```text
kubectl apply Workflow CR ──→ API server admission
                              └─ ValidatingAdmissionPolicy (CEL on spec.engine)
                                 ├─ spec.engine unset/empty → ADMIT (platform default engine)
                                 ├─ spec.engine ∈ allow-list → ADMIT → reconcile → dispatch
                                 └─ otherwise → DENY with the policy message

zynax apply (REST) ──→ api-gateway ──→ workflow-compiler
                                       └─ checkRoutingPolicy (engine-hint annotation)
```

Two paths, two guards (**the ADR-045 §3 interim dual-guard**):

- The **CR path** (kubectl / GitOps) is guarded at **admission** — the
  controller never sees a denied object.
- The **REST path** (`zynax apply` → gateway → compiler) never touches the API
  server, so admission cannot see it; the compiler's `checkRoutingPolicy`
  (reading the `zynax.io/engine-hint` manifest annotation) stays live for it.
  The two converge on admission only if/when REST is retired — not scheduled.

## Enabling the policy

Off by default. Requires **Kubernetes ≥ 1.30** (`ValidatingAdmissionPolicy` is
GA and default-on in `admissionregistration.k8s.io/v1`; the e2e kind harness
runs `kindest/node:v1.30.0`).

```yaml
# helm values (api-gateway chart or via the umbrella)
zynax-api-gateway:
  admissionPolicy:
    enabled: true
    allowedEngines: [temporal]   # engines this release namespace may use
```

This renders three objects:

| Object | Scope | Role |
|--------|-------|------|
| `ValidatingAdmissionPolicy …-engine-allowlist` | cluster | the CEL rule on `spec.engine` |
| `ValidatingAdmissionPolicyBinding …-engine-allowlist-<ns>` | cluster | scopes enforcement to the release namespace; points at the params object |
| ConfigMap `…-engine-allowlist-params` | namespace | carries `allowedEngines` (comma-separated) |

**Per-namespace policy:** each namespace gets its own binding + params
ConfigMap (the binding's `namespaceSelector` pins one namespace). Different
namespaces can carry different allow-lists against the same policy object.
For copy-pasteable YAML and a live transcript, see
[Worked example: two namespaces, two allow-lists](#worked-example-two-namespaces-two-allow-lists).

## Semantics

- **Unset or empty `spec.engine` always admits** — it means "use the platform
  default engine". The policy never forces you to pin an engine.
- **Empty `allowedEngines` = no restriction** — mirrors the compiler gate's
  `RoutingPolicyConfig` semantics.
- A **missing params ConfigMap admits** (`parameterNotFoundAction: Allow`) —
  the params object is policy *config*, not a gate.
- **Fail-closed:** unlike the compiler's fail-open gate, admission with
  `failurePolicy: Fail` rejects when the policy itself cannot evaluate. This
  is a deliberate behaviour change (ADR-045) and the reason the flag defaults
  off — enabling it is a per-deployment decision.

## What a denial looks like

```console
$ kubectl -n zynax apply -f wf-eval.yaml
Error from server (Forbidden): error when creating "wf-eval.yaml": workflows.zynax.io "pinned-eval" is forbidden: ValidatingAdmissionPolicy 'zynax-api-gateway-engine-allowlist' with binding 'zynax-api-gateway-engine-allowlist-zynax' denied request: engine 'eval' is not in this namespace's allow-list [temporal,argo] (ADR-045); omit spec.engine to use the platform default
```

The object is never persisted; the controller never reconciles it; nothing is
dispatched. The denial names the **binding** that rejected it — which is how
you tell two namespaces' allow-lists apart when debugging.

## Worked example: two namespaces, two allow-lists

**Goal:** the release namespace `zynax` may use `temporal` **or** `argo`;
namespace `team-b` may use `temporal` only. One shared
`ValidatingAdmissionPolicy` object, two bindings, two params ConfigMaps.

### 1. The release namespace — from the chart

```yaml
# values.yaml (api-gateway chart, or nested under the umbrella)
zynax-api-gateway:
  admissionPolicy:
    enabled: true
    allowedEngines: [temporal, argo]
```

Render exactly what that produces before applying it:

```bash
helm template zynax-api-gateway helm/zynax-api-gateway \
  --namespace zynax \
  --set admissionPolicy.enabled=true \
  --set 'admissionPolicy.allowedEngines={temporal,argo}' \
  --show-only templates/validatingadmissionpolicy.yaml
```

### 2. The second namespace — copy-paste

The chart renders objects for **its own release namespace only**. A second
namespace needs its own binding + params ConfigMap, applied with `kubectl` or
committed to your GitOps repo. Nothing below re-declares the CEL rule:
`policyName` points at the policy object the chart already installed, so the
two namespaces cannot drift apart on the *rule* — only on the *allow-list*.

```yaml
# team-b.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: team-b
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: engine-allowlist-params
  namespace: team-b
data:
  # Comma-separated. Empty string = no restriction.
  allowedEngines: "temporal"
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicyBinding
metadata:
  # Bindings are CLUSTER-scoped: the name must be unique across the cluster,
  # so suffix it with the namespace it governs.
  name: zynax-api-gateway-engine-allowlist-team-b
spec:
  # Same policy object as the release namespace — only the params differ.
  policyName: zynax-api-gateway-engine-allowlist
  validationActions: [Deny]
  paramRef:
    name: engine-allowlist-params
    namespace: team-b
    parameterNotFoundAction: Allow
  matchResources:
    namespaceSelector:
      matchLabels:
        kubernetes.io/metadata.name: team-b
```

```bash
kubectl apply -f team-b.yaml
```

> **`namespaceSelector` matches namespace LABELS, not namespace names.** The
> form above is safe to copy-paste with no extra step *because*
> `kubernetes.io/metadata.name` is a label the API server sets and keeps in
> sync on every namespace (`NamespaceDefaultLabelName`, on by default since
> 1.21) — it is not a magic "name" field:
>
> ```console
> $ kubectl get namespace team-b -o jsonpath={.metadata.labels}
> {"kubernetes.io/metadata.name":"team-b"}
> ```
>
> Select on **any other** label and you must apply that label yourself —
> see [A binding that matches nothing allows everything](#a-binding-that-matches-nothing-allows-everything).

### 3. Verification transcript

Live output, kind `kindest/node:v1.30.0`, with the `workflows.zynax.io` CRD and
the rendered chart objects above installed. `wf-argo.yaml` is the CR below;
`wf-eval.yaml` and `wf-temporal.yaml` differ only in `metadata.name` and
`spec.engine`.

```yaml
# wf-argo.yaml
apiVersion: zynax.io/v1alpha1
kind: Workflow
metadata:
  name: pinned-argo
spec:
  engine: argo
  initial_state: greet
  states:
    greet:
      actions:
        - capability: echo
          input:
            message: "hello"
      "on":
        - event: echo.completed
          goto: done
    done:
      type: terminal
```

```console
$ kubectl -n zynax apply -f wf-argo.yaml
workflow.zynax.io/pinned-argo created

$ kubectl -n team-b apply -f wf-argo.yaml
Error from server (Forbidden): error when creating "wf-argo.yaml": workflows.zynax.io "pinned-argo" is forbidden: ValidatingAdmissionPolicy 'zynax-api-gateway-engine-allowlist' with binding 'zynax-api-gateway-engine-allowlist-team-b' denied request: engine 'argo' is not in this namespace's allow-list [temporal] (ADR-045); omit spec.engine to use the platform default

$ kubectl -n zynax apply -f wf-eval.yaml
Error from server (Forbidden): error when creating "wf-eval.yaml": workflows.zynax.io "pinned-eval" is forbidden: ValidatingAdmissionPolicy 'zynax-api-gateway-engine-allowlist' with binding 'zynax-api-gateway-engine-allowlist-zynax' denied request: engine 'eval' is not in this namespace's allow-list [temporal,argo] (ADR-045); omit spec.engine to use the platform default

$ kubectl -n team-b apply -f wf-temporal.yaml
workflow.zynax.io/pinned-temporal created
```

The same `argo` CR is admitted in `zynax` and denied in `team-b`, and the two
denials quote different allow-lists (`[temporal,argo]` vs `[temporal]`) from
different bindings — that is the whole feature.

## Failure modes worth knowing before you rely on this

### A namespace with no binding is not governed at all

The policy object alone enforces nothing; a binding is what activates it. A
namespace that no binding selects is **completely unconstrained** — not "denied
by default", and not covered by the release namespace's list:

```console
$ kubectl create namespace team-c
namespace/team-c created

$ kubectl -n team-c apply -f wf-eval.yaml
workflow.zynax.io/pinned-eval created
```

`eval` was denied in `zynax` a moment earlier. Adding a namespace to the
cluster therefore *opts it out* of the allow-list until someone adds its
binding — treat the binding as part of namespace provisioning.

### A binding that matches nothing allows everything

The worst failure mode for an allow-list: a binding whose `namespaceSelector`
matches no namespace is silently inert. Here the binding and params exist and
say `temporal` only, but nobody applied the selector label to `team-d`:

```yaml
  matchResources:
    namespaceSelector:
      matchLabels:
        zynax.io/engine-policy: restricted   # a label YOU must apply
```

```console
$ kubectl apply -f team-d.yaml
namespace/team-d created
configmap/engine-allowlist-params created
validatingadmissionpolicybinding.admissionregistration.k8s.io/zynax-api-gateway-engine-allowlist-restricted created

$ kubectl -n team-d apply -f wf-argo.yaml
workflow.zynax.io/pinned-argo created
```

Admitted — the allow-list was never consulted. (The CR went in 15 s after the
binding, so this is the selector missing its target, not the propagation delay
described below.) Label the namespace and the same CR is rejected:

```console
$ kubectl label namespace team-d zynax.io/engine-policy=restricted
namespace/team-d labeled

$ kubectl -n team-d delete workflow pinned-argo
workflow.zynax.io "pinned-argo" deleted

$ kubectl -n team-d apply -f wf-argo.yaml
Error from server (Forbidden): error when creating "wf-argo.yaml": workflows.zynax.io "pinned-argo" is forbidden: ValidatingAdmissionPolicy 'zynax-api-gateway-engine-allowlist' with binding 'zynax-api-gateway-engine-allowlist-restricted' denied request: engine 'argo' is not in this namespace's allow-list [temporal] (ADR-045); omit spec.engine to use the platform default
```

Always prove a new binding with a **negative** test (apply a CR that must be
denied). A positive test passing tells you nothing — an inert binding passes it.

### Bindings do not take effect instantly

A new binding, a params edit, and a namespace-label edit all reach the API
server's admission machinery a beat after `kubectl apply` returns. Measured on
kind v1.30 with a retry loop: a `Workflow` applied 0.04 s after its binding was
created was still **admitted**; the next attempt, 1.1 s later, was denied.

Consequence for scripts and CI: never assert enforcement in the command
immediately following the `apply` — retry the negative test for a few seconds
before calling it a failure.

## Scaling past two namespaces

### One binding, many namespaces, per-namespace allow-lists

Omit `paramRef.namespace` and the params ConfigMap is resolved **in the
namespace of the request object**. One binding then governs every labelled
namespace while each keeps its own allow-list — no per-namespace binding to
add on provisioning:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: zynax-api-gateway-engine-allowlist-per-namespace
spec:
  policyName: zynax-api-gateway-engine-allowlist
  validationActions: [Deny]
  paramRef:
    name: engine-allowlist-params
    # namespace omitted on purpose: resolved in the request's namespace
    parameterNotFoundAction: Allow
  matchResources:
    namespaceSelector:
      matchLabels:
        zynax.io/engine-policy: per-namespace
```

With `team-e` carrying `allowedEngines: "temporal"` and `team-f`
`allowedEngines: "argo"`, a single binding produces mirror-image denials:

```console
$ kubectl -n team-e apply -f wf-argo.yaml
Error from server (Forbidden): error when creating "wf-argo.yaml": workflows.zynax.io "pinned-argo" is forbidden: ValidatingAdmissionPolicy 'zynax-api-gateway-engine-allowlist' with binding 'zynax-api-gateway-engine-allowlist-per-namespace' denied request: engine 'argo' is not in this namespace's allow-list [temporal] (ADR-045); omit spec.engine to use the platform default

$ kubectl -n team-f apply -f wf-temporal.yaml
Error from server (Forbidden): error when creating "wf-temporal.yaml": workflows.zynax.io "pinned-temporal" is forbidden: ValidatingAdmissionPolicy 'zynax-api-gateway-engine-allowlist' with binding 'zynax-api-gateway-engine-allowlist-per-namespace' denied request: engine 'temporal' is not in this namespace's allow-list [argo] (ADR-045); omit spec.engine to use the platform default

$ kubectl -n team-f apply -f wf-argo.yaml
workflow.zynax.io/pinned-argo created
```

Trade-off: this pattern hands allow-list authorship to whoever can write a
ConfigMap in the namespace. Keep the explicit per-namespace binding of the
worked example when the allow-list must stay under platform-team RBAC.

### Overlapping bindings intersect — any Deny wins

Every binding that matches is evaluated independently; a CR must satisfy all
of them. Adding a stricter binding (`allowedEngines: "eval"`) over `team-f`,
whose own binding allows `argo`, denies the CR:

```console
$ kubectl -n team-f apply -f wf-argo.yaml
Error from server (Forbidden): error when creating "wf-argo.yaml": workflows.zynax.io "pinned-argo" is forbidden: ValidatingAdmissionPolicy 'zynax-api-gateway-engine-allowlist' with binding 'zynax-api-gateway-engine-allowlist-overlap' denied request: engine 'argo' is not in this namespace's allow-list [eval] (ADR-045); omit spec.engine to use the platform default
```

So a cluster-wide baseline binding plus per-team bindings composes as an
intersection, never a union — you cannot widen a namespace's allow-list by
adding a second, more permissive binding.

### Auditing what is actually enforced

The selector column is the one that tells you whether a namespace is really
governed:

```console
$ kubectl get validatingadmissionpolicybinding -o custom-columns=BINDING:.metadata.name,PARAMS-NS:.spec.paramRef.namespace,SELECTOR:.spec.matchResources.namespaceSelector.matchLabels
BINDING                                            PARAMS-NS   SELECTOR
zynax-api-gateway-engine-allowlist-overlap         team-f      map[kubernetes.io/metadata.name:team-f]
zynax-api-gateway-engine-allowlist-per-namespace   <none>      map[zynax.io/engine-policy:per-namespace]
zynax-api-gateway-engine-allowlist-restricted      team-d      map[zynax.io/engine-policy:restricted]
zynax-api-gateway-engine-allowlist-team-b          team-b      map[kubernetes.io/metadata.name:team-b]
zynax-api-gateway-engine-allowlist-zynax           zynax       map[kubernetes.io/metadata.name:zynax]
```

`PARAMS-NS: <none>` marks the request-namespace lookup above. Cross-check the
list against `kubectl get namespaces`: any namespace that no row selects is
unconstrained.

### These allow-lists cover the CR path only

Per-namespace bindings govern `kubectl`/GitOps `Workflow` CRs. A `zynax apply`
(REST) submission never reaches the API server, so none of the above applies
to it — it stays behind the compiler's `checkRoutingPolicy`, which is
configured per deployment via `ZYNAX_POLICY_*` env vars, not by these
ConfigMaps. Restricting a namespace here does **not** restrict the REST path
for it; that dual-guard is the interim design of ADR-045 §3 and is not
scheduled for removal.

## What this policy is NOT

- It is **not quota**. The concurrent-invocation quota is a runtime concern
  admission cannot see; it is currently **unenforced on both gates** (the
  compiler's dead quota check was removed; the engine-adapter `QuotaChecker`
  exists as a contract but is not wired). See ADR-045 §2.
- It is **not engine-fit decisioning**. Capability↔engine matching and hint
  semantics stay in the compiler (protected core, ADR-040 §6). The CEL rule is
  pure set-membership on one spec field.

## Verifying it live

```text
kubectl api-resources | grep validatingadmissionpolic   # served at v1 on ≥1.30
kubectl get validatingadmissionpolicy                    # the policy object
kubectl get validatingadmissionpolicybinding             # the per-ns binding
scripts/e2e/e2e-workflow-crd.sh                          # deny + allow assertions
```

The e2e harness enables the policy with `allowedEngines: [temporal, argo]` and
asserts a CR pinning an engine outside the list is denied at admission with
the policy message (`scripts/e2e/e2e-workflow-crd.sh`, M8.G #1637).

For a multi-namespace setup, the check that matters is a **negative** one per
namespace — see
[Failure modes worth knowing](#failure-modes-worth-knowing-before-you-rely-on-this).

## Related

- [ADR-045](../adr/ADR-045-admission-policy-delegation.md) — admission-policy
  delegation (decision + dual-guard + quota fate)
- [ADR-043](../adr/ADR-043-workflow-crd-front-end.md) — the thin `Workflow`
  CRD front-end (the attach point)
- [ADR-040](../adr/ADR-040-kubernetes-native-delegation-boundary.md) — the
  Kubernetes-native delegation boundary (thin-Zynax)
- Canvas: [`docs/spdd/1575-admission-policy/canvas.md`](../spdd/1575-admission-policy/canvas.md)
