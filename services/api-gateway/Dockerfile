# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f services/api-gateway/Dockerfile .)
FROM golang:1.26.4-alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS builder
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
LABEL org.opencontainers.image.title="Zynax API Gateway" \
      org.opencontainers.image.description="HTTP REST entry point for the Zynax control plane. Accepts YAML workflow manifests on POST /api/v1/apply and routes them to the workflow compiler via gRPC." \
      org.opencontainers.image.url="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.source="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.documentation="https://github.com/zynax-io/zynax/blob/main/services/api-gateway/AGENTS.md" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.vendor="Zynax" \
      com.zynax.service.role="control-plane" \
      com.zynax.service.language="Go" \
      com.zynax.service.port="8080/tcp (REST), 9090/tcp (gRPC)"
ENTRYPOINT ["/api-gateway"]
