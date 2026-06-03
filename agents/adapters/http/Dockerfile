# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f agents/adapters/http/Dockerfile .)
FROM golang:1.26.3-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder

WORKDIR /workspace

COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY agents/adapters/http/go.mod agents/adapters/http/go.sum ./agents/adapters/http/
RUN cd agents/adapters/http && GOWORK=off go mod download

COPY protos/generated/go/ ./protos/generated/go/
COPY agents/adapters/http/ ./agents/adapters/http/
COPY tools/healthcheck/ ./tools/healthcheck/
RUN cd agents/adapters/http && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /http-adapter ./cmd/http-adapter/
RUN cd tools/healthcheck && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /healthcheck .

# renovate: datasource=docker depName=gcr.io/distroless/static
FROM gcr.io/distroless/static:nonroot@sha256:dfadf31470f770fcabd48903762dce126958e98d1ce320acf1216bbfaa42d79c
COPY --from=builder /http-adapter /http-adapter
COPY --from=builder /healthcheck /healthcheck
LABEL org.opencontainers.image.title="Zynax HTTP Adapter" \
      org.opencontainers.image.description="HTTP capability adapter. Registers agent capabilities and executes HTTP-based agent calls on behalf of the Temporal workflow engine. Configure via ADAPTER_CONFIG env pointing to an agent-def YAML." \
      org.opencontainers.image.url="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.source="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.documentation="https://github.com/zynax-io/zynax/blob/main/agents/adapters/http/AGENTS.md" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.vendor="Zynax" \
      com.zynax.service.role="adapter" \
      com.zynax.service.language="Go" \
      com.zynax.service.port="9095/tcp (gRPC)"
ENTRYPOINT ["/http-adapter"]
