# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f agents/adapters/langgraph/Dockerfile .)
#
# Two-stage build:
#   builder  — python:3.12-alpine + uv; installs locked dependencies into /app/.venv
#   runtime  — python:3.12-alpine; copies venv + source + proto stubs; runs as zynax (UID 1001)
#
# Required env vars at runtime: LANGGRAPH_MOUNTS (JSON), REGISTRY_ADDR, AGENT_ID, ADAPTER_ENDPOINT
# Optional: ZYNAX_LANGGRAPH_ADAPTER_GRPC_PORT (default 50058)
#
# LANGGRAPH_MOUNTS format (JSON array):
#   [{"capability_name": "my_cap", "module": "my_package.my_graph", "graph": "graph"}]
# The module must be importable from the container's PYTHONPATH. Mount graph
# modules via a volume or build a custom image extending this one.
#
# ── Base image: alpine (musl), not Debian slim ────────────────────────────────
# Debian trixie's python:3.12-slim carries perl-base and ncurses with unfixed
# CRITICAL/HIGH CVEs that red-wall the pre-merge Trivy gate (ADR-027) — packages
# this adapter never uses. Alpine drops them entirely. The historical objection
# (manylinux-only native wheels forcing musl source builds) no longer holds:
# every native dep in uv.lock — including langgraph's pydantic-core/orjson —
# ships musllinux wheels, and protobuf falls back to its pure-python wheel.
# `--no-build` enforces wheels-only, so a future lock bump that loses a musl
# wheel fails loudly at build time instead of attempting a source compile.
#
# ── Final image size ──────────────────────────────────────────────────────────
# Compressed (amd64): ~39 MB (was ~78 MB on slim; 80 MB target). The runtime stage carries
# only the python:3.12-alpine base + the frozen uv venv (no-dev) + source + proto
# stubs + bundled example graphs; the uv/build layer stays in the builder stage.

# BEGIN zynax-ci:images:python-alpine
ARG PYTHON_ALPINE_DIGEST=sha256:f7fd610959cae736251523b54eb26cecb74f60ffa60bf39d9faccf128b526ab8
# END zynax-ci:images:python-alpine
# renovate: datasource=docker depName=python versioning=docker
FROM python:3.12-alpine@${PYTHON_ALPINE_DIGEST} AS builder

WORKDIR /app

# Install uv for fast, deterministic dependency resolution
# renovate: datasource=docker depName=ghcr.io/astral-sh/uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# Install locked dependencies (without the project itself — source is copied separately)
#
# Runtime/gencode invariant: the protobuf runtime pinned in uv.lock must be >= the
# gencode version of the stubs copied below from protos/generated/python. protobuf
# rejects runtime < gencode at import, which would crash the adapter on startup
# (regressed in #1550: gencode bumped to 7.35.1 while the lock lagged at 7.35.0).
# CI's docker-change detection now rebuilds this image on uv.lock/pyproject changes.
COPY agents/adapters/langgraph/pyproject.toml agents/adapters/langgraph/uv.lock ./
RUN uv sync --no-dev --no-install-project --frozen --no-build

# renovate: datasource=docker depName=python versioning=docker
FROM python:3.12-alpine@${PYTHON_ALPINE_DIGEST}

WORKDIR /app

# Non-root user (ADR-002)
RUN adduser -S -H -u 1001 -G nogroup zynax

# Strip the base image's build-time-only packages (pip/setuptools/wheel). They are
# never used at runtime — the app runs the uv venv via `python -m` — and are a
# recurring source of Trivy CVE noise that red-walls the release gate (#1022).
RUN rm -rf /usr/local/lib/python3.12/site-packages/pip* \
           /usr/local/lib/python3.12/site-packages/setuptools* \
           /usr/local/lib/python3.12/site-packages/pkg_resources \
           /usr/local/lib/python3.12/site-packages/wheel* \
           /usr/local/bin/pip /usr/local/bin/pip3 /usr/local/bin/pip3.12

# Venv with all locked dependencies
COPY --from=builder /app/.venv /app/.venv

# Generated proto stubs (zynax.v1.*)
COPY protos/generated/python/ /protos/

# Adapter source package
COPY agents/adapters/langgraph/src/ /app/src/

# Bundled example graphs (importable as `examples.*`)
COPY agents/adapters/langgraph/examples/ /app/examples/

ENV PATH="/app/.venv/bin:$PATH"
ENV PYTHONPATH="/app/src:/app:/protos"

LABEL org.opencontainers.image.title="Zynax LangGraph Adapter" \
      org.opencontainers.image.description="Zynax capability adapter for LangGraph workflows. Runs LangGraph state-machine graphs as Zynax capabilities." \
      org.opencontainers.image.url="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.source="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.documentation="https://github.com/zynax-io/zynax/blob/main/agents/adapters/langgraph/Dockerfile" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.vendor="Zynax"

USER zynax

CMD ["python", "-m", "langgraph_adapter"]
