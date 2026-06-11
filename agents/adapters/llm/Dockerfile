# SPDX-License-Identifier: Apache-2.0
# Build context: repo root  (docker build -f agents/adapters/llm/Dockerfile .)
#
# Two-stage build:
#   builder  — python:3.12-alpine + uv; installs locked dependencies into /app/.venv
#   runtime  — python:3.12-alpine; copies venv + source + proto stubs; runs as zynax (UID 1001)
#
# Required env vars at runtime: LLM_PROVIDER, REGISTRY_ADDR, AGENT_ID, ADAPTER_ENDPOINT
# Provider-specific: OPENAI_API_KEY | AWS_REGION | OLLAMA_BASE_URL
# Optional: LLM_MODEL, LLM_MAX_TOKENS, ZYNAX_LLM_ADAPTER_GRPC_PORT (default 50057)
#
# ── Base image: alpine (musl), not Debian slim ────────────────────────────────
# Debian trixie's python:3.12-slim carries perl-base and ncurses with unfixed
# CRITICAL/HIGH CVEs that red-wall the pre-merge Trivy gate (ADR-027) — packages
# this adapter never uses. Alpine drops them entirely. The historical objection
# (manylinux-only native wheels forcing musl source builds) no longer holds:
# every native dep in uv.lock ships musllinux wheels (grpcio, pydantic-core,
# aiohttp, …) and protobuf falls back to its pure-python wheel. `--no-build`
# enforces wheels-only, so a future lock bump that loses a musl wheel fails
# loudly at build time instead of attempting a source compile.
#
# ── Final image size ──────────────────────────────────────────────────────────
# Compressed (amd64): ~48 MB (was ~75 MB on slim; 80 MB target). The runtime stage carries
# only the python:3.12-alpine base + the frozen uv venv (no-dev) + source + proto
# stubs; the uv/build layer is left behind in the builder stage.

# BEGIN zynax-ci:images:python-alpine
ARG PYTHON_ALPINE_DIGEST=sha256:4d5e6e9fdab73ef5833d6c9c4e12503e838b7e19a56362d51b7206b2739ca103
# END zynax-ci:images:python-alpine
# renovate: datasource=docker depName=python versioning=docker
FROM python:3.12-alpine@${PYTHON_ALPINE_DIGEST} AS builder

WORKDIR /app

# Install uv for fast, deterministic dependency resolution
# renovate: datasource=docker depName=ghcr.io/astral-sh/uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# Install locked dependencies (without the project itself — source is copied separately)
COPY agents/adapters/llm/pyproject.toml agents/adapters/llm/uv.lock ./
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
COPY agents/adapters/llm/src/ /app/src/

ENV PATH="/app/.venv/bin:$PATH"
ENV PYTHONPATH="/app/src:/protos"

LABEL org.opencontainers.image.title="Zynax LLM Adapter" \
      org.opencontainers.image.description="Zynax capability adapter for LLM inference. Supports multiple providers: OpenAI, Ollama, Amazon Bedrock." \
      org.opencontainers.image.url="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.source="https://github.com/zynax-io/zynax" \
      org.opencontainers.image.documentation="https://github.com/zynax-io/zynax/blob/main/agents/adapters/llm/Dockerfile" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.vendor="Zynax"

USER zynax

CMD ["python", "-m", "llm_adapter"]
