# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f services/engine-adapter/Dockerfile .)
FROM golang:1.25-alpine AS builder
WORKDIR /workspace
COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY services/engine-adapter/go.mod services/engine-adapter/go.sum ./services/engine-adapter/
RUN cd services/engine-adapter && GOWORK=off go mod download
COPY protos/generated/go/ ./protos/generated/go/
COPY services/engine-adapter/ ./services/engine-adapter/
RUN cd services/engine-adapter && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -o /engine-adapter ./cmd/engine-adapter/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates \
    && addgroup -S zynax && adduser -S zynax -G zynax
USER zynax
COPY --from=builder /engine-adapter /engine-adapter
ENTRYPOINT ["/engine-adapter"]
