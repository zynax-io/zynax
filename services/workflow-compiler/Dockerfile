# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f services/workflow-compiler/Dockerfile .)
FROM golang:1.26.3-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder
WORKDIR /workspace
COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY services/workflow-compiler/go.mod services/workflow-compiler/go.sum ./services/workflow-compiler/
RUN cd services/workflow-compiler && GOWORK=off go mod download
COPY protos/generated/go/ ./protos/generated/go/
COPY services/workflow-compiler/ ./services/workflow-compiler/
COPY tools/healthcheck/ ./tools/healthcheck/
RUN cd services/workflow-compiler && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /workflow-compiler ./cmd/workflow-compiler/
RUN cd tools/healthcheck && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /healthcheck .

# renovate: datasource=docker depName=gcr.io/distroless/static
FROM gcr.io/distroless/static:nonroot@sha256:dfadf31470f770fcabd48903762dce126958e98d1ce320acf1216bbfaa42d79c
COPY --from=builder /workflow-compiler /workflow-compiler
COPY --from=builder /healthcheck /healthcheck
LABEL org.opencontainers.image.title="Zynax Workflow Compiler" \
      org.opencontainers.image.description="Stateless YAML-to-IR compiler. Translates declarative workflow manifests into typed WorkflowIR for dispatch by the engine adapter. Scales horizontally — holds no persistent state." \
      org.opencontainers.image.url="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.source="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.documentation="https://github.com/zynax-io/zynax/blob/main/services/workflow-compiler/AGENTS.md" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.vendor="Zynax" \
      com.zynax.service.role="compiler" \
      com.zynax.service.language="Go" \
      com.zynax.service.port="9092/tcp (gRPC)"
ENTRYPOINT ["/workflow-compiler"]
