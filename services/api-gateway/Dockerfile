# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f services/api-gateway/Dockerfile .)
FROM golang:1.26.3-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder
WORKDIR /workspace
COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY services/api-gateway/go.mod services/api-gateway/go.sum ./services/api-gateway/
RUN cd services/api-gateway && GOWORK=off go mod download
COPY protos/generated/go/ ./protos/generated/go/
COPY services/api-gateway/ ./services/api-gateway/
COPY tools/healthcheck/ ./tools/healthcheck/
RUN cd services/api-gateway && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /api-gateway ./cmd/api-gateway/
RUN cd tools/healthcheck && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /healthcheck .

# renovate: datasource=docker depName=gcr.io/distroless/static
FROM gcr.io/distroless/static:nonroot@sha256:dfadf31470f770fcabd48903762dce126958e98d1ce320acf1216bbfaa42d79c
COPY --from=builder /api-gateway /api-gateway
COPY --from=builder /healthcheck /healthcheck
ENTRYPOINT ["/api-gateway"]
