# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f agents/adapters/http/Dockerfile .)
FROM golang:1.26.3-alpine AS builder

WORKDIR /workspace

COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY agents/adapters/http/go.mod agents/adapters/http/go.sum ./agents/adapters/http/
RUN cd agents/adapters/http && GOWORK=off go mod download

COPY protos/generated/go/ ./protos/generated/go/
COPY agents/adapters/http/ ./agents/adapters/http/
RUN cd agents/adapters/http && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -o /http-adapter ./cmd/http-adapter/

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
    && addgroup -S zynax && adduser -S zynax -G zynax

COPY --from=builder /http-adapter /http-adapter

USER zynax

ENTRYPOINT ["/http-adapter"]
