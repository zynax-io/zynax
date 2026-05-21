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
    go build -trimpath -o /agent-registry ./cmd/agent-registry/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates \
    && addgroup -S zynax && adduser -S zynax -G zynax
USER zynax
COPY --from=builder /agent-registry /agent-registry
ENTRYPOINT ["/agent-registry"]
