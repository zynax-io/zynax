# Migrating from push registration to the `Agent` custom resource

> **Audience:** operators and adapter authors still calling
> `AgentRegistryService.RegisterAgent` (or applying `kind: AgentDef` through
> the api-gateway). **Since M8 (ADR-039)** those paths answer
> `UNIMPLEMENTED` / HTTP 410 — agent identity lives in the
> `zynax.io/v1alpha1` **`Agent`** custom resource, and dispatch selection
> uses `SchedulerService.SelectAgent`. Hard removal of the deprecated RPCs
> is scheduled for **M9**.

## Why this changed

The push registry had three structural defects (ADR-039): a second source of
truth that could drift from the applied manifest, self-asserted liveness (a
crashed agent kept receiving work), and a bespoke re-implementation of
lifecycle/watch/cache machinery Kubernetes already provides. In the CRD era:

- **Identity** is the `Agent` CR — GitOps-diffable, `kubectl`-able.
- **Liveness** is reconciled from the backing Service's EndpointSlice — a
  crashed agent is authoritatively not-ready within seconds.
- **Selection** is `SelectAgent`: readiness-filtered, metrics-aware, with a
  structured rationale on every decision.

## Before → after

**Before (retired):** the adapter self-registered on boot…

```text
adapter boot → RegisterAgent(agent_id, endpoint, capabilities) → registry row
task-broker  → FindByCapability → round-robin over rows (dead or alive)
```

…or an operator applied `kind: AgentDef` through the gateway.

**After:** declare the agent once, next to its Deployment/Service:

```yaml
apiVersion: zynax.io/v1alpha1
kind: Agent
metadata:
  name: echo-worker            # becomes agent_id "namespace/name"
spec:
  endpointRef:                 # the Service fronting the agent's gRPC port
    serviceName: echo-worker
    port: 50058
  capabilities:
    - id: echo                 # ^[a-z0-9_]{1,64}$ — maps to CapabilityDef.name
      description: "Echoes the input payload"
      inputSchema: '{"type":"object"}'
```

```bash
kubectl apply -f agent.yaml
kubectl get agents            # READY flips true once endpoints serve
```

That is the whole migration for most agents: **delete the registration call,
apply the CR.** Adapters shipped in this repo already tolerate the retired
registry (they log `push registration retired (ADR-039)` and keep serving),
so image order does not matter during rollout.

### Scoring hints (optional)

`spec.capabilities[]` carries scheduler hints the push API never had:
`selectors.{language,tags}`, `cost.{tokenPrice,latencyClass}`,
`resources.gpu`, `models`, `protocols`. See the CRD schema
(`helm/zynax-agent-registry/crds/agents.zynax.io.yaml`) for the full shape,
and label the CR with `zynax.io/expert-scope` for strict expert targeting
(ADR-028).

### Read paths

| Push era | CRD era |
|---|---|
| `GetAgent` / `ListAgents` | `kubectl get agent <name>` / `kubectl get agents` |
| `FindByCapability` | `SchedulerService.SelectAgent` (one scored agent + rationale) |
| `DeregisterAgent` | `kubectl delete agent <name>` |

## Requirements

- The `Agent` CRD + scheduler RBAC ship with the `zynax-agent-registry`
  chart (≥ 0.3.0); enable the scheduler with `crdInformer.enabled: true`.
- The scheduler is namespace-scoped: apply Agent CRs in the release
  namespace.
- Optional live metrics: point `crdInformer.promUrl` at a Prometheus HTTP
  API; without it, selection runs readiness-filtered rotation (degraded
  mode — correct, never failing).

## Rollback (pre-M9 window)

The push-era code paths still exist behind the deprecation until M9:
reverting the M8 retirement PR restores them. After M9's hard removal the
CRD path is the only path — plan migrations before then.

## Schedule

| Milestone | State |
|---|---|
| M7 | ADR-039 accepted; KIND-verified spike |
| **M8 (now)** | `Agent` CR is the source of truth; push RPCs answer `UNIMPLEMENTED`; gateway `AgentDef` answers 410 |
| M9 | Deprecated RPCs and push-era code removed |
