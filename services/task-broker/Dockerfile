# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f services/task-broker/Dockerfile .)
FROM golang:1.26.3-alpine AS builder
WORKDIR /workspace
COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY services/task-broker/go.mod services/task-broker/go.sum ./services/task-broker/
RUN cd services/task-broker && GOWORK=off go mod download
COPY protos/generated/go/ ./protos/generated/go/
COPY services/task-broker/ ./services/task-broker/
COPY tools/healthcheck/ ./tools/healthcheck/
RUN cd services/task-broker && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /task-broker ./cmd/task-broker/
RUN cd tools/healthcheck && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /healthcheck .

# renovate: datasource=docker depName=gcr.io/distroless/static
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /task-broker /task-broker
COPY --from=builder /healthcheck /healthcheck
ENTRYPOINT ["/task-broker"]
