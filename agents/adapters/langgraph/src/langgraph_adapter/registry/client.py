# SPDX-License-Identifier: Apache-2.0
"""Registry client — RegisterAgent / DeregisterAgent with exponential-backoff retry."""

from __future__ import annotations

import asyncio
import logging
from typing import Any

import grpc
from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict
from zynax.v1 import agent_registry_pb2  # type: ignore[import-untyped]

log = logging.getLogger(__name__)

_MAX_ATTEMPTS = 5
_BASE_DELAY = 2.0


class RegistrySettings(BaseSettings):
    """Registry and gRPC endpoint settings loaded from environment variables.

    Attributes:
        agent_id: Unique agent identifier for registration (``AGENT_ID``).
        adapter_endpoint: gRPC address the task-broker dials for this adapter
            (``ADAPTER_ENDPOINT``), e.g. ``"langgraph-adapter:50058"``.
        grpc_port: Port the adapter's own gRPC server binds to
            (``ZYNAX_LANGGRAPH_ADAPTER_GRPC_PORT``, default ``50058``).
    """

    model_config = SettingsConfigDict(env_prefix="", extra="ignore")

    agent_id: str = Field(..., alias="AGENT_ID", min_length=1)
    adapter_endpoint: str = Field(..., alias="ADAPTER_ENDPOINT", min_length=1)
    grpc_port: int = Field(default=50058, alias="ZYNAX_LANGGRAPH_ADAPTER_GRPC_PORT", ge=1)


def _is_unimplemented(exc: Exception) -> bool:
    """Return True when exc is the CRD-era UNIMPLEMENTED answer (ADR-039)."""
    code_fn = getattr(exc, "code", None)
    return callable(code_fn) and code_fn() == grpc.StatusCode.UNIMPLEMENTED


def _is_transient(exc: Exception) -> bool:
    """Return True when exc carries a retriable gRPC status code."""
    code_fn = getattr(exc, "code", None)
    if not callable(code_fn):
        return False
    return code_fn() in (
        grpc.StatusCode.UNAVAILABLE,
        grpc.StatusCode.INTERNAL,
        grpc.StatusCode.DEADLINE_EXCEEDED,
    )


def build_agent_def(
    settings: RegistrySettings,
    capability_names: list[str],
    schemas: dict[str, tuple[bytes, bytes]],
) -> Any:
    """Build an ``AgentDef`` proto from registry settings and capability list.

    Args:
        settings: Registry settings containing ``agent_id`` and ``adapter_endpoint``.
        capability_names: Ordered list of capability names to register (one per mount).
        schemas: Map of capability name → ``(input_schema_bytes, output_schema_bytes)``.

    Returns:
        A populated ``agent_registry_pb2.AgentDef`` instance.
    """
    caps = [
        agent_registry_pb2.CapabilityDef(
            name=name,
            input_schema=schemas.get(name, (b"", b""))[0],
            output_schema=schemas.get(name, (b"", b""))[1],
        )
        for name in capability_names
    ]
    return agent_registry_pb2.AgentDef(
        agent_id=settings.agent_id,
        name="langgraph-adapter",
        endpoint=settings.adapter_endpoint,
        capabilities=caps,
    )


async def register_agent(agent_def: Any, stub: Any) -> None:
    """Register the adapter with the agent-registry using exponential-backoff retry.

    Args:
        agent_def: Populated ``AgentDef`` proto built by ``build_agent_def``.
        stub: Async ``AgentRegistryServiceStub`` to call ``RegisterAgent`` on.

    Raises:
        Exception: On non-transient gRPC error, or after all retry attempts exhausted.
    """
    req = agent_registry_pb2.RegisterAgentRequest(agent=agent_def)
    delay = _BASE_DELAY
    last_exc: Exception | None = None
    for attempt in range(_MAX_ATTEMPTS):
        if attempt > 0:
            log.info("retrying registration", extra={"attempt": attempt + 1, "delay_s": delay})
            await asyncio.sleep(delay)
            delay *= 2
        try:
            await stub.RegisterAgent(req)
            log.info("agent registered", extra={"agent_id": agent_def.agent_id})
            return
        except Exception as exc:
            # CRD-era registry (ADR-039): push registration is retired and the
            # RPC answers UNIMPLEMENTED — discovery flows through the Agent
            # custom resource instead. Keep serving; nothing to retry.
            if _is_unimplemented(exc):
                log.info(
                    "push registration retired (ADR-039) — relying on Agent CR discovery",
                    extra={"agent_id": agent_def.agent_id},
                )
                return
            if not _is_transient(exc):
                raise
            last_exc = exc
            log.warning(
                "registration attempt failed",
                extra={"attempt": attempt + 1, "err": str(exc)},
            )
    raise RuntimeError(f"register failed after {_MAX_ATTEMPTS} attempts") from last_exc


async def deregister_agent(agent_id: str, stub: Any) -> None:
    """Deregister the adapter from the agent-registry (no retry).

    Args:
        agent_id: The agent identifier supplied at registration time.
        stub: Async ``AgentRegistryServiceStub`` to call ``DeregisterAgent`` on.
    """
    req = agent_registry_pb2.DeregisterAgentRequest(agent_id=agent_id)
    try:
        await stub.DeregisterAgent(req)
    except Exception as exc:
        if _is_unimplemented(exc):
            log.info(
                "push deregistration retired (ADR-039) — Agent CR lifecycle owns removal",
                extra={"agent_id": agent_id},
            )
            return
        raise
    log.info("agent deregistered", extra={"agent_id": agent_id})
