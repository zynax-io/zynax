# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f agents/adapters/ci/Dockerfile .)
# BEGIN zynax-ci:images:golang-alpine
ARG GO_ALPINE_DIGEST=sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2
# END zynax-ci:images:golang-alpine
# BEGIN zynax-ci:images:distroless-static
ARG DISTROLESS_DIGEST=sha256:dfadf31470f770fcabd48903762dce126958e98d1ce320acf1216bbfaa42d79c
# END zynax-ci:images:distroless-static
FROM golang:1.26.5-alpine@${GO_ALPINE_DIGEST} AS builder

WORKDIR /workspace

COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY agents/adapters/ci/go.mod agents/adapters/ci/go.sum ./agents/adapters/ci/
RUN cd agents/adapters/ci && GOWORK=off go mod download

COPY protos/generated/go/ ./protos/generated/go/
COPY agents/adapters/ci/ ./agents/adapters/ci/
COPY tools/healthcheck/ ./tools/healthcheck/
RUN cd agents/adapters/ci && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /ci-adapter ./cmd/ci-adapter/
RUN cd tools/healthcheck && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /healthcheck .

# renovate: datasource=docker depName=gcr.io/distroless/static
FROM gcr.io/distroless/static:nonroot@${DISTROLESS_DIGEST}
COPY --from=builder /ci-adapter /ci-adapter
COPY --from=builder /healthcheck /healthcheck
COPY --from=builder /workspace/agents/adapters/ci/agent-def.yaml.example /etc/ci-adapter/config.yaml
LABEL org.opencontainers.image.title="Zynax CI Adapter" \
      org.opencontainers.image.description="Zynax capability adapter for CI pipeline operations. Provides CI execution gRPC capabilities." \
      org.opencontainers.image.url="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.source="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.documentation="https://github.com/zynax-io/zynax/blob/main/agents/adapters/ci/Dockerfile" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.vendor="Zynax"
ENTRYPOINT ["/ci-adapter"]
