# infra/ — Engineering Contract

> Infrastructure as code. Every resource is reproducible, reviewable, auditable.
> No manual kubectl. No clicking in consoles. Everything in Git.
> Helm chart templates: `docs/patterns/helm-charts.md`.

---

## Helm Chart Layout

All Helm charts live in `helm/`:

```
helm/
├── zynax-lib/                 ← shared library chart (macros only, no installable templates)
├── zynax-<service>/           ← one chart per Go service (7 total)
├── charts/
│   ├── nats/                  ← NATS JetStream subchart (wraps nats/nats)
│   ├── postgres/              ← Postgres 16 subchart (wraps bitnami/postgresql)
│   ├── temporal/              ← Temporal subchart (wraps temporalio/temporal v1.2.0)
│   └── cert-manager/          ← cert-manager resources: ClusterIssuer + per-service Certificates (ADR-020)
└── zynax-umbrella/            ← umbrella chart — deploys the full platform
```

### cert-manager prerequisite (ADR-020 mTLS)

`helm/charts/cert-manager/` creates ClusterIssuer and Certificate resources — it does **not** install cert-manager itself. cert-manager must be pre-installed before enabling this chart:

```bash
helm repo add jetstack https://charts.jetstack.io
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager --create-namespace \
  --set crds.enabled=true
```

Enable in the umbrella chart with `--set zynax-cert-manager.enabled=true`. See [docs/infra/environment-parity.md](../docs/infra/environment-parity.md) for per-environment TLS settings.

`helm/zynax-umbrella/` aggregates all 7 service charts and the 3 cluster dependency
subcharts. Use it for e2e harness (EPIC G #770) and staging/production deployments.
Run `helm dependency update helm/zynax-umbrella/` after any sub-chart version bump.

The `ct lint` gate (A.12 #791) enforces required resources on every `helm/` change.

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

## Inter-Service mTLS Environment Variables (ADR-020)

All five Go platform services read the following shared env vars for mutual TLS.
When all three are set the service uses `credentials.NewTLS` with cert hot-reload.
When any is empty the service falls back to `insecure.NewCredentials` (dev only).

| Variable | Default | Description |
|----------|---------|-------------|
| `ZYNAX_TLS_CERT` | _(empty)_ | Path to the service's TLS certificate PEM file |
| `ZYNAX_TLS_KEY` | _(empty)_ | Path to the service's TLS private key PEM file |
| `ZYNAX_TLS_CA` | _(empty)_ | Path to the CA certificate bundle PEM for verifying peer certificates |

**Local dev:** run the `cert-gen` Docker Compose service once to populate the
`certs-data` volume, then mount it and set `ZYNAX_TLS_*` on each service.
**K8s:** cert-manager issues per-service certificates; the Helm chart injects
these paths via `ZYNAX_TLS_*` env vars from a projected `Certificate` volume.

Never commit certificate or key files. Never set `InsecureSkipVerify: true`.

---

## Hard Rules

- Never store secrets in `values.yaml` or `values-production.yaml`.
- Always use `minAvailable: 1` in PodDisruptionBudget.
- Always use `maxUnavailable: 0` in RollingUpdate strategy (zero-downtime).
- NetworkPolicy defaults to deny-all; explicit allow for gRPC (50051) and metrics (9090).
