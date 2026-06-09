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
#
# ── Final image size ──────────────────────────────────────────────────────────
# Compressed (amd64): ~78 MB (within the 80 MB target). The runtime stage carries
# only the python:3.12-slim base + the frozen uv venv (no-dev) + source + proto
# stubs + bundled example graphs; the uv/build layer stays in the builder stage.
# We deliberately stay on python:3.12-slim rather than -alpine: langgraph/langchain
# pull in pydantic-core and other native wheels that ship as manylinux (glibc)
# builds and would force slow, fragile musl source compilation on Alpine.
# `uv sync --no-dev --frozen` already yields a minimal deterministic dep set.

# BEGIN zynax-ci:images:python-slim
ARG PYTHON_SLIM_DIGEST=sha256:090ba77e2958f6af52a5341f788b50b032dd4ca28377d2893dcf1ecbdfdfe203
# END zynax-ci:images:python-slim
# renovate: datasource=docker depName=python versioning=docker
FROM python:3.12-slim@${PYTHON_SLIM_DIGEST} AS builder

WORKDIR /app

# Install uv for fast, deterministic dependency resolution
# renovate: datasource=docker depName=ghcr.io/astral-sh/uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# Install locked dependencies (without the project itself — source is copied separately)
COPY agents/adapters/langgraph/pyproject.toml agents/adapters/langgraph/uv.lock ./
RUN uv sync --no-dev --no-install-project --frozen

# renovate: datasource=docker depName=python versioning=docker
FROM python:3.12-slim@${PYTHON_SLIM_DIGEST}

WORKDIR /app

# Non-root user (ADR-002)
RUN useradd -r -u 1001 -g nogroup zynax

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
