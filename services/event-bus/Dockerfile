# SPDX-License-Identifier: Apache-2.0
# Build context must be repo root: docker build -f services/event-bus/Dockerfile .
ARG GO_VERSION=1.26.4
# BEGIN zynax-ci:images:golang-alpine
ARG GO_ALPINE_DIGEST=sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f
# END zynax-ci:images:golang-alpine
# BEGIN zynax-ci:images:distroless-static
ARG DISTROLESS_DIGEST=sha256:dfadf31470f770fcabd48903762dce126958e98d1ce320acf1216bbfaa42d79c
# END zynax-ci:images:distroless-static

# renovate: datasource=docker depName=golang
FROM golang:${GO_VERSION}-alpine@${GO_ALPINE_DIGEST} AS builder
WORKDIR /workspace
# Copy go.mod/go.sum for replace-directive dependencies first (layer caching).
# libs/zynaxconfig is a local replace target — it must be present before go mod download.
COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY libs/zynaxconfig/go.mod libs/zynaxconfig/go.sum ./libs/zynaxconfig/
COPY services/event-bus/go.mod services/event-bus/go.sum ./services/event-bus/
RUN cd services/event-bus && GOWORK=off go mod download
COPY protos/generated/go/ ./protos/generated/go/
COPY libs/zynaxconfig/ ./libs/zynaxconfig/
COPY services/event-bus/ ./services/event-bus/
COPY tools/healthcheck/ ./tools/healthcheck/
RUN cd services/event-bus && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /svc ./cmd/event-bus/
RUN cd tools/healthcheck && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /healthcheck .

# renovate: datasource=docker depName=gcr.io/distroless/static
FROM gcr.io/distroless/static:nonroot@${DISTROLESS_DIGEST}
COPY --from=builder /svc /svc
COPY --from=builder /healthcheck /healthcheck
LABEL org.opencontainers.image.url="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.source="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.documentation="https://github.com/zynax-io/zynax/blob/main/services/event-bus/AGENTS.md" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.vendor="Zynax" \
      com.zynax.service.language="Go"
ENTRYPOINT ["/svc"]
