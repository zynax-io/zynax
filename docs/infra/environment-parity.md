<!-- SPDX-License-Identifier: Apache-2.0 -->

# Environment Parity — Helm Values Differences

> Closes [#243](https://github.com/zynax-io/zynax/issues/243) · Canvas A.13 of [EPIC #765](https://github.com/zynax-io/zynax/issues/765)

This document lists the Helm values that differ between local dev, staging,
and production. All other values are identical across environments.

---

## Replica Count

| Chart | Dev | Staging | Production |
|-------|-----|---------|------------|
| `zynax-api-gateway` | `1` | `2` | `3` |
| `zynax-workflow-compiler` | `1` | `2` | `3` |
| `zynax-engine-adapter` | `1` | `2` | `3` |
| `zynax-task-broker` | `1` | `2` | `3` |
| `zynax-agent-registry` | `1` | `2` | `3` |
| `zynax-event-bus` | `1` | `2` | `3` |
| `zynax-memory-service` | `1` | `1` | `2` |

---

## Resource Requests and Limits

Dev uses minimal allocations for local kind clusters. Staging/prod are sized
for real workloads.

| Setting | Dev | Staging | Production |
|---------|-----|---------|------------|
| `resources.requests.cpu` | `50m` | `100m` | `200m` |
| `resources.requests.memory` | `32Mi` | `64Mi` | `128Mi` |
| `resources.limits.cpu` | `200m` | `500m` | `1000m` |
| `resources.limits.memory` | `128Mi` | `256Mi` | `512Mi` |

---

## Image Tag Strategy

| Environment | `image.tag` value | Notes |
|-------------|-------------------|-------|
| Dev | `latest` (or a local build tag) | Allows rapid iteration; never use in staging/prod |
| Staging | `main-<short-sha>` | Pinned to a specific commit on `main`; set by CD |
| Production | `v<semver>` (e.g. `v0.5.0`) | Only tagged releases; set manually or by release workflow |

The default `image.tag: ""` in `values.yaml` falls back to `Chart.AppVersion`
(the release version). Override per environment.

---

## TLS / mTLS

| Setting | Dev | Staging | Production |
|---------|-----|---------|------------|
| `zynax-cert-manager.enabled` | `false` | `true` | `true` |
| `tlsSecretName` (per service) | `""` (insecure) | `zynax-<svc>-tls` | `zynax-<svc>-tls` |
| cert-manager pre-installed | No (not needed) | Yes (required) | Yes (required) |

In dev, all inter-service gRPC runs over insecure credentials (Docker Compose or
kind without cert-manager). In staging/prod, `zynax-cert-manager` chart must be
installed before enabling `tlsSecretName` in each service chart (ADR-020).

---

## Autoscaling (HPA)

| Setting | Dev | Staging | Production |
|---------|-----|---------|------------|
| `autoscaling.enabled` | `false` | `true` | `true` |
| `autoscaling.minReplicas` | — | `2` | `3` |
| `autoscaling.maxReplicas` | — | `5` | `10` |
| `autoscaling.targetCPUUtilizationPercentage` | — | `70` | `70` |
| `autoscaling.targetMemoryUtilizationPercentage` | — | `80` | `80` |

---

## Log Level

| Chart | Dev | Staging | Production |
|-------|-----|---------|------------|
| All services (`env.logLevel`) | `debug` | `info` | `info` |

---

## Database (Postgres-backed services — EPIC M6.H #626)

| Setting | Dev | Staging | Production |
|---------|-----|---------|------------|
| `db.secretName` | `""` (in-memory fallback) | K8s Secret name | K8s Secret name |
| Postgres `replicaCount` | `1` | `1` | `2` (HA via Bitnami replication) |

Until EPIC M6.H (#626) ships, task-broker and agent-registry use the in-memory
adapter when `db.secretName` is empty. Set `ZYNAX_DB_ENABLED=true` and provide
a secret to activate the Postgres adapter.

---

## Applying Per-Environment Values

Use `-f values-<env>.yaml` to override. Example:

```bash
# Staging
helm upgrade --install zynax-api-gateway helm/zynax-api-gateway/ \
  -f helm/zynax-api-gateway/values-production.yaml \
  --set image.tag=main-abc1234 \
  --set tlsSecretName=zynax-api-gateway-tls

# Production
helm upgrade --install zynax-api-gateway helm/zynax-api-gateway/ \
  -f helm/zynax-api-gateway/values-production.yaml \
  --set image.tag=v0.5.0 \
  --set tlsSecretName=zynax-api-gateway-tls
```

See [infra/AGENTS.md](../../infra/AGENTS.md) for the cert-manager prerequisite
and [docs/patterns/helm-charts.md](../patterns/helm-charts.md) for chart templates.
