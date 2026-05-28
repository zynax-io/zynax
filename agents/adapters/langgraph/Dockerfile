# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f agents/adapters/langgraph/Dockerfile .)
#
# Two-stage build:
#   builder  — python:3.12-slim + uv; installs locked dependencies into /app/.venv
#   runtime  — python:3.12-slim; copies venv + source + proto stubs; runs as zynax (UID 1001)
#
# Required env vars at runtime: LANGGRAPH_MOUNTS (JSON), REGISTRY_ADDR, AGENT_ID, ADAPTER_ENDPOINT
# Optional: ZYNAX_LANGGRAPH_ADAPTER_GRPC_PORT (default 50058)
#
# LANGGRAPH_MOUNTS format (JSON array):
#   [{"capability_name": "my_cap", "module": "my_package.my_graph", "graph": "graph"}]
# The module must be importable from the container's PYTHONPATH. Mount graph
# modules via a volume or build a custom image extending this one.

# renovate: datasource=docker depName=python versioning=docker
FROM python:3.12-slim@sha256:68f78b84b35ab1d74a1a7a04a5447b2d1f5fcf0b7f38dee97a3a1de3c2eac65e AS builder

WORKDIR /app

# Install uv for fast, deterministic dependency resolution
# renovate: datasource=docker depName=ghcr.io/astral-sh/uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# Install locked dependencies (without the project itself — source is copied separately)
COPY agents/adapters/langgraph/pyproject.toml agents/adapters/langgraph/uv.lock ./
RUN uv sync --no-dev --no-install-project --frozen

# renovate: datasource=docker depName=python versioning=docker
FROM python:3.12-slim@sha256:68f78b84b35ab1d74a1a7a04a5447b2d1f5fcf0b7f38dee97a3a1de3c2eac65e

WORKDIR /app

# Non-root user (ADR-002)
RUN useradd -r -u 1001 -g nogroup zynax

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

USER zynax

CMD ["python", "-m", "langgraph_adapter"]
