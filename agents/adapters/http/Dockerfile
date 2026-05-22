# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f agents/adapters/http/Dockerfile .)
FROM golang:1.26.3-alpine@sha256:f44b851aa23dfa219d18db6eab743203245429d355cb619cf96a2ffe2a84ba7a AS builder

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
FROM gcr.io/distroless/static:nonroot@sha256:dfadf31470f770fcabd48903762dce126958e98d1ce320acf1216bbfaa42d79c
COPY --from=builder /http-adapter /http-adapter
COPY --from=builder /healthcheck /healthcheck
ENTRYPOINT ["/http-adapter"]
