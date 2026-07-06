<!-- SPDX-License-Identifier: Apache-2.0 -->

# Engine allow-list: Kubernetes admission policy

Which engines a namespace may use ("this namespace may use Temporal but not
Argo") is governed by a **CEL `ValidatingAdmissionPolicy`** bound to the
`Workflow` custom resource (ADR-045, M8.G) — standard, auditable,
GitOps-diffable Kubernetes policy, not Zynax-specific code. The compiler keeps
only engine-*fit* intelligence (capability↔engine matching); the coarse
allow-list is the platform's job.

## How it fits together

```
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

```
$ kubectl apply -f my-workflow.yaml
The workflows "my-workflow" is invalid: ValidatingAdmissionPolicy
'zynax-api-gateway-engine-allowlist' with binding
'zynax-api-gateway-engine-allowlist-zynax' denied request: engine 'argo' is
not in this namespace's allow-list [temporal] (ADR-045); omit spec.engine to
use the platform default
```

The object is never persisted; the controller never reconciles it; nothing is
dispatched.

## What this policy is NOT

- It is **not quota**. The concurrent-invocation quota is a runtime concern
  admission cannot see; it is currently **unenforced on both gates** (the
  compiler's dead quota check was removed; the engine-adapter `QuotaChecker`
  exists as a contract but is not wired). See ADR-045 §2.
- It is **not engine-fit decisioning**. Capability↔engine matching and hint
  semantics stay in the compiler (protected core, ADR-040 §6). The CEL rule is
  pure set-membership on one spec field.

## Verifying it live

```
kubectl api-resources | grep validatingadmissionpolic   # served at v1 on ≥1.30
kubectl get validatingadmissionpolicy                    # the policy object
kubectl get validatingadmissionpolicybinding             # the per-ns binding
scripts/e2e/e2e-workflow-crd.sh                          # deny + allow assertions
```

The e2e harness enables the policy with `allowedEngines: [temporal, argo]` and
asserts a CR pinning an engine outside the list is denied at admission with
the policy message (`scripts/e2e/e2e-workflow-crd.sh`, M8.G #1637).

## Related

- ADR-045 — admission-policy delegation (decision + dual-guard + quota fate)
- ADR-043 — the thin `Workflow` CRD front-end (the attach point)
- ADR-040 — the Kubernetes-native delegation boundary (thin-Zynax)
- Canvas: `docs/spdd/1575-admission-policy/canvas.md`
