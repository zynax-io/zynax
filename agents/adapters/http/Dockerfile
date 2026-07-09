# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f agents/adapters/http/Dockerfile .)
# BEGIN zynax-ci:images:golang-alpine
ARG GO_ALPINE_DIGEST=sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2
# END zynax-ci:images:golang-alpine
# BEGIN zynax-ci:images:distroless-static
ARG DISTROLESS_DIGEST=sha256:dfadf31470f770fcabd48903762dce126958e98d1ce320acf1216bbfaa42d79c
# END zynax-ci:images:distroless-static
FROM golang:1.26.5-alpine@${GO_ALPINE_DIGEST} AS builder

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
FROM gcr.io/distroless/static:nonroot@${DISTROLESS_DIGEST}
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
