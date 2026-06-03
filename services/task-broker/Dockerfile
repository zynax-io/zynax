# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f services/task-broker/Dockerfile .)
FROM golang:1.26.3-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder
WORKDIR /workspace
COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY services/task-broker/go.mod services/task-broker/go.sum ./services/task-broker/
RUN cd services/task-broker && GOWORK=off go mod download
COPY protos/generated/go/ ./protos/generated/go/
COPY services/task-broker/ ./services/task-broker/
COPY tools/healthcheck/ ./tools/healthcheck/
RUN cd services/task-broker && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /task-broker ./cmd/task-broker/
RUN cd tools/healthcheck && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /healthcheck .

# renovate: datasource=docker depName=gcr.io/distroless/static
FROM gcr.io/distroless/static:nonroot@sha256:dfadf31470f770fcabd48903762dce126958e98d1ce320acf1216bbfaa42d79c
COPY --from=builder /task-broker /task-broker
COPY --from=builder /healthcheck /healthcheck
LABEL org.opencontainers.image.title="Zynax Task Broker" \
      org.opencontainers.image.description="gRPC task queue. Manages capability dispatch tasks between engine-adapter Temporal activities and registered adapters. Tracks task lifecycle: pending → claimed → completed/failed." \
      org.opencontainers.image.url="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.source="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.documentation="https://github.com/zynax-io/zynax/blob/main/services/task-broker/AGENTS.md" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.vendor="Zynax" \
      com.zynax.service.role="task-queue" \
      com.zynax.service.language="Go" \
      com.zynax.service.port="9093/tcp (gRPC)"
ENTRYPOINT ["/task-broker"]
