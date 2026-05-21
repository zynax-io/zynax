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
    go build -trimpath -ldflags "-s -w" -o /http-adapter ./cmd/http-adapter/

# renovate: datasource=docker depName=gcr.io/distroless/static
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /http-adapter /http-adapter
ENTRYPOINT ["/http-adapter"]
