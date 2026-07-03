<!-- SPDX-License-Identifier: Apache-2.0 -->

# Running Zynax with raw `docker compose` — RETIRED

> ## The Compose runtime was removed in M8 (ADR-041 / #1501)
>
> [**ADR-041**](adr/ADR-041-kind-native-unified-runtime.md) made local Kubernetes (kind) the one
> runtime model, and [**ADR-039**](adr/ADR-039-crd-native-scheduler.md) moved agent discovery to
> the `Agent` custom resource — a path Compose cannot serve. The runtime compose files and their
> `make run-local` / `make demo-compose` targets no longer exist.
>
> This stub keeps old links alive. What to use instead:
>
> - **First run:** [`docs/quickstart.md`](quickstart.md) — `zynax up` (or `make demo`).
> - **Local development:** [`docs/local-dev-kind.md`](local-dev-kind.md).
> - **Agent registration migration:** [`docs/patterns/agent-crd-migration.md`](patterns/agent-crd-migration.md).
>
> The Docker **build-tools harness** (`infra/docker-compose/docker-compose.tools.yml`,
> `docker-compose.test.yml`) is unrelated to the retired runtime and still powers
> `make bootstrap / lint / test`.
