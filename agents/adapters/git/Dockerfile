# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f agents/adapters/git/Dockerfile .)
FROM golang:1.26.4-alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS builder

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
