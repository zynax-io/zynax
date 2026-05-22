# infra/ — Engineering Contract

> Infrastructure as code. Every resource is reproducible, reviewable, auditable.
> No manual kubectl. No clicking in consoles. Everything in Git.
> Helm chart templates: `docs/patterns/helm-charts.md`.

---

## Required Helm Chart Resources

Every service chart MUST include (chart-testing enforces this):
`deployment.yaml` · `service.yaml` · `serviceaccount.yaml` · `hpa.yaml` · `pdb.yaml` · `networkpolicy.yaml`

See `docs/patterns/helm-charts.md` for the canonical templates for each.

---

## Docker Compose Rules (Local Dev)

- No hardcoded secrets — use `.env.local` (gitignored).
- Named volumes for data persistence.
- Health checks on all containers.
- Services use `depends_on` with `service_healthy` condition.
- Network named `zynax-net`.
- Expose only necessary ports to host.

`make dev-up` starts the full local stack. `make dev-reset` destroys data and restarts.

---

## Security Context (Required for Every Container)

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1001
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop: ["ALL"]
  seccompProfile:
    type: RuntimeDefault
```

No container may run as root or with elevated privileges.

---

## api-gateway Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ZYNAX_GW_HTTP_PORT` | `8080` | HTTP listen port |
| `ZYNAX_GW_COMPILER_ADDR` | `localhost:50054` | WorkflowCompilerService address |
| `ZYNAX_GW_ENGINE_ADDR` | `localhost:50055` | EngineAdapterService address |
| `ZYNAX_GW_REGISTRY_ADDR` | `localhost:50052` | AgentRegistryService address |
| `ZYNAX_GW_LOG_LEVEL` | `info` | Log level (`debug`/`info`/`warn`/`error`) |
| `ZYNAX_GW_API_KEY` | _(empty)_ | Bearer token for mutating endpoints (`POST /api/v1/apply`, `DELETE /api/v1/workflows/{id}`). When empty, auth is disabled and a `WARN api_key not set — auth disabled` line is logged at startup. Read-only endpoints (`GET`) are always open. |

Set `ZYNAX_GW_API_KEY` in your `.env.local` (gitignored). Never commit a real key.

---

## Hard Rules

- Never store secrets in `values.yaml` or `values-production.yaml`.
- Always use `minAvailable: 1` in PodDisruptionBudget.
- Always use `maxUnavailable: 0` in RollingUpdate strategy (zero-downtime).
- NetworkPolicy defaults to deny-all; explicit allow for gRPC (50051) and metrics (9090).
