# services/ — AGENTS.md

> Inherit all rules from the root `AGENTS.md`. This file adds service-specific
> implementation patterns that apply to **every** service in this directory.

---

## Service Checklist (Before Writing Any Code)

When creating or modifying a service, verify:

1. The `.feature` file is written and committed first.
2. The service has a `pyproject.toml` with all required dependencies.
3. The `config.py` uses `pydantic-settings`. No config files read at runtime.
4. The `domain/` layer has zero imports from `api/` or `infrastructure/`.
5. The `main.py` is wiring-only — no business logic.
6. The Helm chart exists with all required resources (HPA, PDB, NetworkPolicy).
7. Health probes are implemented and registered.
8. OTel instrumentation is initialized.
9. Prometheus metrics are initialized.
10. `.importlinter` is configured to enforce layer boundaries.

---

## gRPC Server Bootstrap Pattern

Every service `main.py` follows this exact pattern. Do not deviate.

```python
# src/<service>/main.py
import asyncio
import logging
import signal
from typing import Any

import grpc
import structlog
from opentelemetry.instrumentation.grpc import GrpcInstrumentorServer
from prometheus_client import start_http_server

from .<service>.api.handlers import <ServiceName>Handler
from .<service>.config import settings
from .<service>.observability import configure_logging, configure_tracing

logger = structlog.get_logger(__name__)


async def serve() -> None:
    configure_logging(settings.log_level)
    configure_tracing(settings.otel_endpoint, settings.service_name)

    # Start metrics server (separate port from gRPC)
    start_http_server(port=settings.metrics_port)

    GrpcInstrumentorServer().instrument()

    server = grpc.aio.server(
        interceptors=[
            AuthInterceptor(settings),
            LoggingInterceptor(),
            TracingInterceptor(),
        ]
    )

    # Register handlers
    add_<ServiceName>Servicer_to_server(<ServiceName>Handler(settings), server)

    # Health probe server (HTTP, separate from gRPC)
    health_app = create_health_app(server)
    await health_app.start(port=settings.health_port)

    listen_addr = f"[::]:{settings.grpc_port}"
    server.add_insecure_port(listen_addr)  # TLS handled by service mesh (Istio/Linkerd)

    await server.start()
    logger.info("server_started", port=settings.grpc_port, service=settings.service_name)

    # Graceful shutdown on SIGTERM (Kubernetes sends this)
    stop_event = asyncio.Event()
    loop = asyncio.get_event_loop()
    loop.add_signal_handler(signal.SIGTERM, stop_event.set)
    loop.add_signal_handler(signal.SIGINT, stop_event.set)

    await stop_event.wait()

    logger.info("server_stopping", grace_period_seconds=settings.shutdown_grace_seconds)
    await server.stop(grace=settings.shutdown_grace_seconds)
    logger.info("server_stopped")


if __name__ == "__main__":
    asyncio.run(serve())
```

---

## Domain Service Pattern

```python
# src/<service>/domain/services.py

from dataclasses import dataclass
from typing import Protocol

from .<service>.domain.models import AgentSpec, AgentId
from .<service>.domain.exceptions import AgentNotFound, AgentAlreadyExists


class AgentRepository(Protocol):
    """Port — implemented by infrastructure layer."""
    async def save(self, spec: AgentSpec) -> AgentId: ...
    async def find_by_id(self, agent_id: AgentId) -> AgentSpec | None: ...
    async def find_by_capability(self, capability: str) -> list[AgentSpec]: ...
    async def delete(self, agent_id: AgentId) -> None: ...


@dataclass(frozen=True)
class AgentRegistrar:
    """
    Domain service for agent registration and discovery.

    Pure Python. Zero I/O. Injected with repository via constructor.
    Testable in complete isolation from infrastructure.
    """

    _repository: AgentRepository

    async def register(self, spec: AgentSpec) -> AgentId:
        existing = await self._repository.find_by_id(spec.id)
        if existing is not None:
            raise AgentAlreadyExists(spec.id)
        self._validate_spec(spec)
        return await self._repository.save(spec)

    def _validate_spec(self, spec: AgentSpec) -> None:
        if len(spec.capabilities) > MAX_CAPABILITIES:
            raise CapabilityLimitExceeded(len(spec.capabilities), MAX_CAPABILITIES)
        if not spec.capabilities:
            raise InvalidAgentSpec("Agent must declare at least one capability")
```

---

## Repository Pattern

```python
# src/<service>/infrastructure/repositories.py

from sqlalchemy.ext.asyncio import AsyncSession

from .<service>.domain.models import AgentSpec, AgentId
from .<service>.domain.services import AgentRepository  # Implements this protocol
from .<service>.infrastructure.orm import AgentORM


class PostgresAgentRepository:
    """Implements AgentRepository protocol using PostgreSQL."""

    def __init__(self, session: AsyncSession) -> None:
        self._session = session

    async def save(self, spec: AgentSpec) -> AgentId:
        orm = AgentORM.from_domain(spec)
        self._session.add(orm)
        await self._session.flush()
        return orm.to_domain_id()

    async def find_by_id(self, agent_id: AgentId) -> AgentSpec | None:
        result = await self._session.get(AgentORM, str(agent_id))
        return result.to_domain() if result else None
```

---

## pyproject.toml Template

Every service uses this base. Adjust dependencies per service.

```toml
[project]
name = "zynax-<service-name>"
version = "0.1.0"
requires-python = ">=3.12"
description = "<Service description>"
license = { text = "Apache-2.0" }
authors = [{ name = "Zynax Contributors" }]

dependencies = [
    "grpcio>=1.60.0",
    "grpcio-tools>=1.60.0",
    "pydantic>=2.5.0",
    "pydantic-settings>=2.1.0",
    "structlog>=24.0.0",
    "prometheus-client>=0.19.0",
    "opentelemetry-api>=1.22.0",
    "opentelemetry-sdk>=1.22.0",
    "opentelemetry-instrumentation-grpc>=0.43b0",
    "sqlalchemy[asyncio]>=2.0.0",
    "asyncpg>=0.29.0",
]

[dependency-groups]
dev = [
    "pytest>=8.0.0",
    "pytest-asyncio>=0.23.0",
    "pytest-bdd>=7.0.0",
    "pytest-cov>=4.1.0",
    "testcontainers[postgres,redis]>=4.0.0",
    "mypy>=1.8.0",
    "ruff>=0.2.0",
    "import-linter>=2.0.0",
    "mutmut>=2.4.0",
    "grpcio-testing>=1.60.0",
    "factory-boy>=3.3.0",
    "faker>=22.0.0",
]

[tool.pytest.ini_options]
asyncio_mode = "auto"
testpaths = ["tests"]
addopts = [
    "--cov=src",
    "--cov-report=term-missing",
    "--cov-fail-under=90",
    "--strict-markers",
    "-v",
]

[tool.coverage.report]
exclude_lines = [
    "pragma: no cover",
    "if TYPE_CHECKING:",
    "raise NotImplementedError",
    "@abstractmethod",
]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
```

---

## Dockerfile Template

```dockerfile
# syntax=docker/dockerfile:1.6
# ─────────────────────────────────────────────────
# Stage 1: builder
# ─────────────────────────────────────────────────
FROM python:3.12-slim AS builder

# Install uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

WORKDIR /build
COPY pyproject.toml uv.lock ./

# Install deps into isolated layer (no editable install)
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --frozen --no-dev --no-editable

COPY src ./src

# ─────────────────────────────────────────────────
# Stage 2: runtime
# ─────────────────────────────────────────────────
FROM python:3.12-slim AS runtime

# Non-root user
RUN useradd --system --no-create-home --uid 1001 --gid 0 zynax

# Copy only the installed packages
COPY --from=builder /build/.venv /app/.venv
COPY --from=builder /build/src /app/src

ENV PATH="/app/.venv/bin:$PATH"
ENV PYTHONPATH="/app/src"
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

WORKDIR /app
USER zynax

# gRPC, metrics, health
EXPOSE 50051 9090 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD python -c "import grpc; c=grpc.insecure_channel('localhost:50051'); grpc.channel_ready_future(c).result(timeout=3)"

ENTRYPOINT ["python", "-m", "<service_module>.main"]
```

---

## Integration Test Pattern (testcontainers)

```python
# tests/conftest.py
import pytest
from testcontainers.postgres import PostgresContainer
from testcontainers.redis import RedisContainer

@pytest.fixture(scope="session")
def postgres_container():
    with PostgresContainer("postgres:16-alpine") as container:
        yield container

@pytest.fixture(scope="session")
def redis_container():
    with RedisContainer("redis:7-alpine") as container:
        yield container

@pytest.fixture
async def db_session(postgres_container):
    """Real DB session — never mock the database."""
    engine = create_async_engine(postgres_container.get_connection_url())
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    async with AsyncSession(engine) as session:
        yield session
        await session.rollback()
```
