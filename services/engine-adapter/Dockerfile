# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f services/engine-adapter/Dockerfile .)
FROM golang:1.26.4-alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS builder
WORKDIR /workspace
COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY services/engine-adapter/go.mod services/engine-adapter/go.sum ./services/engine-adapter/
RUN cd services/engine-adapter && GOWORK=off go mod download
COPY protos/generated/go/ ./protos/generated/go/
COPY services/engine-adapter/ ./services/engine-adapter/
COPY tools/healthcheck/ ./tools/healthcheck/
RUN cd services/engine-adapter && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /engine-adapter ./cmd/engine-adapter/
RUN cd tools/healthcheck && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /healthcheck .

# renovate: datasource=docker depName=gcr.io/distroless/static
FROM gcr.io/distroless/static:nonroot@sha256:dfadf31470f770fcabd48903762dce126958e98d1ce320acf1216bbfaa42d79c
COPY --from=builder /engine-adapter /engine-adapter
COPY --from=builder /healthcheck /healthcheck
LABEL org.opencontainers.image.title="Zynax Engine Adapter" \
      org.opencontainers.image.description="Execution engine bridge. Dispatches compiled WorkflowIR to Temporal (and future engines) via the pluggable WorkflowEngine interface. Wires Temporal activities to registered adapter capabilities." \
      org.opencontainers.image.url="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.source="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.documentation="https://github.com/zynax-io/zynax/blob/main/services/engine-adapter/AGENTS.md" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.vendor="Zynax" \
      com.zynax.service.role="execution-engine" \
      com.zynax.service.language="Go" \
      com.zynax.service.port="9091/tcp (gRPC)"
ENTRYPOINT ["/engine-adapter"]
