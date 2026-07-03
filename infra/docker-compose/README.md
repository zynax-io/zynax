<!-- SPDX-License-Identifier: Apache-2.0 -->

# `infra/docker-compose/` — build & test harnesses only

> **The Compose *runtime* was removed in M8** (ADR-041 / #1501): kind is the one
> runtime model — bring the platform up with `zynax up` (or `make demo`); see
> [docs/quickstart.md](../../docs/quickstart.md). Agent discovery lives in the
> `Agent` custom resource (ADR-039 —
> [migration guide](../../docs/patterns/agent-crd-migration.md)).

What remains here is **not a runtime**. These files power the Docker-based
developer/CI toolchain and test backing services:

| File | Role | Used by |
|------|------|---------|
| `docker-compose.tools.yml` | The pinned tools image harness (buf, linters, pytest, …) | `make bootstrap` / `make lint` / `make test` / `make generate-protos` |
| `docker-compose.test.yml` | Test-time containers for the tools harness | `make test*` targets |
| `docker-compose.services.yml` | Backing services (Postgres, …) for `//go:build integration` tests | `make test-integration` (CI `test-integration` job) |

Nothing here starts Zynax services. If you are looking for the platform,
you want the kind path:

```bash
zynax up        # cluster + charts (add --profile lite, or --engine argo)
make demo       # same bring-up + the hero workflow
```
