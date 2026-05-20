# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f services/task-broker/Dockerfile .)
FROM golang:1.26.3-alpine AS builder
WORKDIR /workspace
COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY services/task-broker/go.mod services/task-broker/go.sum ./services/task-broker/
RUN cd services/task-broker && GOWORK=off go mod download
COPY protos/generated/go/ ./protos/generated/go/
COPY services/task-broker/ ./services/task-broker/
RUN cd services/task-broker && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -o /task-broker ./cmd/task-broker/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates \
    && addgroup -S zynax && adduser -S zynax -G zynax
USER zynax
COPY --from=builder /task-broker /task-broker
ENTRYPOINT ["/task-broker"]
