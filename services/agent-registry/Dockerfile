# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f services/agent-registry/Dockerfile .)
FROM golang:1.26.4-alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS builder
WORKDIR /workspace
COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY services/agent-registry/go.mod services/agent-registry/go.sum ./services/agent-registry/
RUN cd services/agent-registry && GOWORK=off go mod download
COPY protos/generated/go/ ./protos/generated/go/
COPY services/agent-registry/ ./services/agent-registry/
COPY tools/healthcheck/ ./tools/healthcheck/
RUN cd services/agent-registry && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /agent-registry ./cmd/agent-registry/
RUN cd tools/healthcheck && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /healthcheck .

# renovate: datasource=docker depName=gcr.io/distroless/static
FROM gcr.io/distroless/static:nonroot@sha256:dfadf31470f770fcabd48903762dce126958e98d1ce320acf1216bbfaa42d79c
COPY --from=builder /agent-registry /agent-registry
COPY --from=builder /healthcheck /healthcheck
LABEL org.opencontainers.image.title="Zynax Agent Registry" \
      org.opencontainers.image.description="Adapter registration service. Maintains the live catalog of adapter capabilities for routing decisions. Adapters register on startup and heartbeat to remain discoverable." \
      org.opencontainers.image.url="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.source="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.documentation="https://github.com/zynax-io/zynax/blob/main/services/agent-registry/AGENTS.md" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.vendor="Zynax" \
      com.zynax.service.role="registry" \
      com.zynax.service.language="Go" \
      com.zynax.service.port="9094/tcp (gRPC)"
ENTRYPOINT ["/agent-registry"]
