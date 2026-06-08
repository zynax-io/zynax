# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f agents/adapters/llm/Dockerfile .)
#
# Two-stage build:
#   builder  — python:3.12-slim + uv; installs locked dependencies into /app/.venv
#   runtime  — python:3.12-slim; copies venv + source + proto stubs; runs as zynax (UID 1001)
#
# Required env vars at runtime: LLM_PROVIDER, REGISTRY_ADDR, AGENT_ID, ADAPTER_ENDPOINT
# Provider-specific: OPENAI_API_KEY | AWS_REGION | OLLAMA_BASE_URL
# Optional: LLM_MODEL, LLM_MAX_TOKENS, ZYNAX_LLM_ADAPTER_GRPC_PORT (default 50057)

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
COPY agents/adapters/llm/pyproject.toml agents/adapters/llm/uv.lock ./
RUN uv sync --no-dev --no-install-project --frozen

# renovate: datasource=docker depName=python versioning=docker
FROM python:3.12-slim@${PYTHON_SLIM_DIGEST}

WORKDIR /app

# Non-root user (ADR-002)
RUN useradd -r -u 1001 -g nogroup zynax

# Venv with all locked dependencies
COPY --from=builder /app/.venv /app/.venv

# Generated proto stubs (zynax.v1.*)
COPY protos/generated/python/ /protos/

# Adapter source package
COPY agents/adapters/llm/src/ /app/src/

ENV PATH="/app/.venv/bin:$PATH"
ENV PYTHONPATH="/app/src:/protos"

USER zynax

CMD ["python", "-m", "llm_adapter"]
