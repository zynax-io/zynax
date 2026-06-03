# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f services/workflow-compiler/Dockerfile .)
FROM golang:1.26.4-alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS builder
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
