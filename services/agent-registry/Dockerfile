# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f services/agent-registry/Dockerfile .)
FROM golang:1.26.3-alpine AS builder
WORKDIR /workspace
COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY services/agent-registry/go.mod services/agent-registry/go.sum ./services/agent-registry/
RUN cd services/agent-registry && GOWORK=off go mod download
COPY protos/generated/go/ ./protos/generated/go/
COPY services/agent-registry/ ./services/agent-registry/
RUN cd services/agent-registry && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /agent-registry ./cmd/agent-registry/

# renovate: datasource=docker depName=gcr.io/distroless/static
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /agent-registry /agent-registry
ENTRYPOINT ["/agent-registry"]
