# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f services/api-gateway/Dockerfile .)
FROM golang:1.26.3-alpine AS builder
WORKDIR /workspace
COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY services/api-gateway/go.mod services/api-gateway/go.sum ./services/api-gateway/
RUN cd services/api-gateway && GOWORK=off go mod download
COPY protos/generated/go/ ./protos/generated/go/
COPY services/api-gateway/ ./services/api-gateway/
RUN cd services/api-gateway && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /api-gateway ./cmd/api-gateway/

# renovate: datasource=docker depName=gcr.io/distroless/static
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /api-gateway /api-gateway
ENTRYPOINT ["/api-gateway"]
