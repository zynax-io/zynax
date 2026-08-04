# Migrating from push registration to the `Agent` custom resource

> **Audience:** operators and adapter authors still calling
> `AgentRegistryService.RegisterAgent` (or applying `kind: AgentDef` through
> the api-gateway). Those paths are **gone as of M9 (ADR-039)**: the RPCs were
> deprecated in M8, answered `UNIMPLEMENTED` throughout v0.7.0, and were
> deleted from the contract in #1598. Agent identity lives in the
> `zynax.io/v1alpha1` **`Agent`** custom resource, and dispatch selection
> uses `SchedulerService.SelectAgent`. A client built against the removed
> RPCs now fails at compile time (Go/Python stubs) or with `UNIMPLEMENTED`
> from gRPC itself; `kind: AgentDef` applies answer HTTP 400
> `UNSUPPORTED_KIND`.

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
apply the CR.** Adapters shipped in this repo no longer carry a registration
client at all — it was deleted in #1598, so they simply serve their gRPC
capability surface and never dial the registry. Image order does not matter
during rollout.

### Adapter configuration

The address of the agent-registry is no longer adapter configuration. These keys
are **no longer read as of M9.A** and are ignored if still present:

| Adapter | Retired key |
|---|---|
| `adk`, `ci`, `git`, `http`, `llm` (Go) | `registry_endpoint` (YAML) |
| `langgraph` (Python) | `REGISTRY_ADDR` (env) |

**Order matters — roll the image before you touch the config.** The two directions
are *not* symmetric:

| Situation | Result |
|---|---|
| **New image**, config still sets the key | ✅ boots — the Go adapters decode with non-strict `yaml.Unmarshal` and the langgraph settings model is `extra="ignore"`, so an unknown key/env is ignored |
| **Old image**, key deleted from config | ❌ **fails at startup** — pre-M9.A adapters validate the key as *required* (`config: registry_endpoint is required`, or a pydantic `ValidationError`) and the pod never becomes Ready |

So the safe sequence is:

1. Roll out the M9.A adapter image (it stops requiring the key).
2. *Then* delete the key from your ConfigMaps / env / Helm values.

The reverse order hard-fails the pod. That is why the manifests in this repo
(`scripts/e2e/manifests/echo-worker.yaml`,
`infra/packages/code-review-rank/llm-adapter-config.yaml`,
`infra/ollama/llm-adapter.config.yaml`) still set the key behind an explanatory
comment: they pin `:main`-tagged images, which are rebuilt only *after* the code
change merges. They are cleaned up in a follow-up, once those images carry M9.A.

`ZYNAX_BROKER_REGISTRY_ADDR` on the **task-broker** is unrelated and stays: it
points at the agent-registry Deployment, which serves `SchedulerService.SelectAgent`
— the CRD-era selection path.

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

### CLI surface

| Push era | CRD era |
|---|---|
| `zynax apply <agentdef.yaml>` | `kubectl apply -f agent.yaml` |
| `zynax agent publish <file>` | retired — fails with this migration pointer |
| `zynax init expert` / `zynax agent init` | **unchanged** — an AgentDef manifest is still the expert-definition authoring format (ADR-028/ADR-033), validated locally by `zynax validate` and `make validate-spec`. Only *pushing* it to the api-gateway is gone. |

## Requirements

- The `Agent` CRD + scheduler RBAC ship with the `zynax-agent-registry`
  chart (≥ 0.3.0); enable the scheduler with `crdInformer.enabled: true`.
- The scheduler is namespace-scoped: apply Agent CRs in the release
  namespace.
- Optional live metrics: point `crdInformer.promUrl` at a Prometheus HTTP
  API; without it, selection runs readiness-filtered rotation (degraded
  mode — correct, never failing).

## Rollback

**The rollback window is closed.** It ran from M8 (deprecation) to M9 (hard
removal): until #1598 the push-era code paths still existed behind the
deprecation and reverting the M8 retirement PR restored them. The contract,
the adapter registration clients, and the gateway route are now deleted, so
the CRD path is the only path. Operators pinned to the push registry must
stay on a v0.7.x release and migrate before upgrading.

## Schedule

| Milestone | State |
|---|---|
| M7 | ADR-039 accepted; KIND-verified spike |
| M8 | `Agent` CR is the source of truth; push RPCs answer `UNIMPLEMENTED`; gateway `AgentDef` answers 410 |
| **M9 (now)** | Deprecated RPCs and push-era code removed; `kind: AgentDef` answers 400 `UNSUPPORTED_KIND`; retired adapter config keys and the `agent publish` alias swept (epic #1674 closed by #1699) |
