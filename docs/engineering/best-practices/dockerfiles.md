<!-- SPDX-License-Identifier: Apache-2.0 -->

# Dockerfile Best Practices — Zynax

> Scope: `infra/docker/Dockerfile.*`, service Dockerfiles  
> Enforcement: `trivy` scan in release pipeline (#565); reviewed in PR

---

## Multi-stage Build Template (Go service)

```dockerfile
# ─── Stage 1: Build ─────────────────────────────────────────────────────────
FROM golang:1.26.3-alpine AS builder

# Use build cache mounts — dramatically speeds up rebuilds
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    true

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/service ./cmd/service/

# ─── Stage 2: Runtime ────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

# OCI image labels (SLSA provenance when cosign is added)
LABEL org.opencontainers.image.title="zynax-<service>"
LABEL org.opencontainers.image.source="https://github.com/zynax-io/zynax"
LABEL org.opencontainers.image.licenses="Apache-2.0"

COPY --from=builder /out/service /service

# Non-root user (nonroot = uid 65532 on distroless)
USER nonroot:nonroot

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/service", "--healthcheck"]

EXPOSE 50051

ENTRYPOINT ["/service"]
```

**Key rules:**
1. **Multi-stage** — build stage discarded; only binary in final image
2. **Pin base image by tag** (1.26.3-alpine) — Renovate keeps this updated
3. **`CGO_ENABLED=0 -trimpath`** — reproducible static binary
4. **Distroless or Alpine final** — no shell, no package manager in production image
5. **Non-root user** — `nonroot` on distroless, `zynax` (uid 1001) on Alpine variants
6. **HEALTHCHECK** — required for container orchestrators (#463)
7. **OCI labels** — machine-readable metadata for SBOM and provenance

---

## Current Pattern (Alpine final)

The current service Dockerfiles use Alpine as the final stage (not distroless) with a
dedicated `zynax` user:

```dockerfile
FROM alpine:3.21 AS runtime
RUN addgroup -S zynax && adduser -S zynax -G zynax
USER zynax
```

This is acceptable for now. The distroless migration is recommended for M6 when SBOM
and cosign are added (smaller attack surface, no package manager).

---

## .dockerignore

```
.git/
docs/
*.md
*.yaml        # include only what the Dockerfile COPYs
infra/
spec/
tools/
```

A tight `.dockerignore` prevents the build context from including unnecessary files,
reducing build time and preventing accidental secret leakage from `.env` files.

---

## No Secrets in Layers

```dockerfile
# ❌ NEVER — secret baked into layer history
ENV AWS_SECRET_ACCESS_KEY=abc123

# ✅ Inject secrets at runtime via environment variables or secrets mounts
RUN --mount=type=secret,id=api_key cat /run/secrets/api_key
```

Trivy and gitleaks scan for secrets in image layers. Any hardcoded secret in a layer
fails the release pipeline.

---

## Go Builder Toolchain Alignment

The Go builder stage must use the same Go version as `go.work`:

```
go.work: go 1.26.3          (source of truth)
Dockerfile: golang:1.26.3-alpine   (must match)
```

Mismatches cause `GOTOOLCHAIN=local` errors and silent build failures (#601 fixed this).

---

## HEALTHCHECK

Every service Dockerfile must include a HEALTHCHECK. Current services are missing this
(tracked by #463 / canvas `docs/spdd/463-health-probes/canvas.md`).

```dockerfile
# For gRPC services, use grpc_health_probe (available in ci-runner and tools images)
HEALTHCHECK --interval=15s --timeout=5s --start-period=15s \
    CMD ["/usr/local/bin/grpc_health_probe", "-addr=:50051"] || exit 1

# For HTTP services, curl or wget
HEALTHCHECK --interval=15s --timeout=5s \
    CMD wget -qO- http://localhost:7080/healthz || exit 1
```

---

## Pinning Tool Versions in Dockerfile.tools

`infra/docker/Dockerfile.tools` must pin every tool version:

```dockerfile
ARG GOLANGCI_LINT_VERSION=1.62.2
ARG BUF_VERSION=1.47.2
ARG GOVULNCHECK_VERSION=v1.1.3

RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}
```

- Never use `@latest` — Renovate manages version bumps via PR
- Renovate's `regexManagers` watches these ARG lines for update PRs

---

## Anti-patterns

| Anti-pattern | Correct approach |
|---|---|
| `RUN apt-get install …` in final stage | Use multi-stage; install in builder only |
| `ADD` instead of `COPY` for local files | Use `COPY` unless you need URL fetch or auto-extract |
| Single-stage build | Always multi-stage for Go/Python services |
| No `.dockerignore` | Always have `.dockerignore` that excludes `.git`, `docs/`, local dev files |
| `golang:1.22-alpine` with `go 1.26.3` in go.mod | Pin base image to match `go.work` toolchain version |
| GOPATH `/root/go/bin/` on Alpine | Use `/go/bin/` — Alpine GOPATH is `/go`, not `/root/go` |
| `USER root` in final stage | Use dedicated non-root user |
| No HEALTHCHECK | All services must have HEALTHCHECK for Compose + K8s probes |
