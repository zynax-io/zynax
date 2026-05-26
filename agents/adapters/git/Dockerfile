# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f agents/adapters/git/Dockerfile .)
FROM golang:1.26.3-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder

WORKDIR /workspace

COPY protos/generated/go/go.mod protos/generated/go/go.sum ./protos/generated/go/
COPY agents/adapters/git/go.mod agents/adapters/git/go.sum ./agents/adapters/git/
RUN cd agents/adapters/git && GOWORK=off go mod download

COPY protos/generated/go/ ./protos/generated/go/
COPY agents/adapters/git/ ./agents/adapters/git/
COPY tools/healthcheck/ ./tools/healthcheck/
RUN cd agents/adapters/git && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /git-adapter ./cmd/git-adapter/
RUN cd tools/healthcheck && CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "-s -w" -o /healthcheck .

# renovate: datasource=docker depName=gcr.io/distroless/static
FROM gcr.io/distroless/static:nonroot@sha256:dfadf31470f770fcabd48903762dce126958e98d1ce320acf1216bbfaa42d79c
COPY --from=builder /git-adapter /git-adapter
COPY --from=builder /healthcheck /healthcheck
COPY --from=builder /workspace/agents/adapters/git/agent-def.yaml.example /etc/git-adapter/config.yaml
ENTRYPOINT ["/git-adapter"]
