# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f services/workflow-compiler/Dockerfile .)
FROM golang:1.25-alpine AS builder
WORKDIR /workspace
COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY services/workflow-compiler/go.mod services/workflow-compiler/go.sum ./services/workflow-compiler/
RUN cd services/workflow-compiler && GOWORK=off go mod download
COPY protos/generated/go/ ./protos/generated/go/
COPY services/workflow-compiler/ ./services/workflow-compiler/
RUN cd services/workflow-compiler && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -o /workflow-compiler ./cmd/workflow-compiler/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates \
    && addgroup -S zynax && adduser -S zynax -G zynax
USER zynax
COPY --from=builder /workflow-compiler /workflow-compiler
ENTRYPOINT ["/workflow-compiler"]
