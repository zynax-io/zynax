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

## Hard Rules

- Never store secrets in `values.yaml` or `values-production.yaml`.
- Always use `minAvailable: 1` in PodDisruptionBudget.
- Always use `maxUnavailable: 0` in RollingUpdate strategy (zero-downtime).
- NetworkPolicy defaults to deny-all; explicit allow for gRPC (50051) and metrics (9090).
