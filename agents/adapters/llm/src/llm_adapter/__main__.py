# SPDX-License-Identifier: Apache-2.0
"""llm-adapter bootstrap — load config, register, serve, deregister on SIGTERM."""

from __future__ import annotations

import asyncio
import logging
import signal

import grpc.aio  # type: ignore[import-untyped]
from zynax.v1 import agent_registry_pb2_grpc  # type: ignore[import-untyped]
from zynax.v1.agent_pb2_grpc import (
    add_AgentServiceServicer_to_server,  # type: ignore[import-untyped]
)

from llm_adapter.config import ProviderConfig
from llm_adapter.registry.client import (
    AdapterSettings,
    build_agent_def,
    deregister_agent,
    register_agent,
)
from llm_adapter.router import CapabilityRouter
from llm_adapter.server import AgentServicer

log = logging.getLogger(__name__)


async def main() -> None:
    """Bootstrap: load config, register with agent-registry, serve, deregister on exit."""
    logging.basicConfig(level=logging.INFO, format="%(message)s")
    provider_cfg = ProviderConfig()  # type: ignore[call-arg]
    settings = AdapterSettings()  # type: ignore[call-arg]
    router = CapabilityRouter(provider_cfg)
    channel = grpc.aio.insecure_channel(settings.registry_addr)
    stub = agent_registry_pb2_grpc.AgentRegistryServiceStub(channel)
    agent_def = _build_def(settings, router)
    await register_agent(agent_def, stub)
    server = await _start_server(router, settings.grpc_port)
    await _wait_for_signal()
    await _shutdown(settings.agent_id, stub, server, channel)


def _build_def(settings: AdapterSettings, router: CapabilityRouter) -> object:
    """Construct the AgentDef proto from settings and router schemas.

    Args:
        settings: Adapter settings containing agent_id and endpoint.
        router: Built capability router; provides names and schema bytes.

    Returns:
        Populated AgentDef proto ready for registration.
    """
    cap_names = router.capability_names()
    schemas = {n: router.get_schema(n) for n in cap_names}
    return build_agent_def(settings, cap_names, schemas)


async def _start_server(router: CapabilityRouter, port: int) -> grpc.aio.Server:
    """Create and start the gRPC server on the given port.

    Args:
        router: The capability router wired into the servicer.
        port: TCP port to bind (``[::]:<port>``).

    Returns:
        A started ``grpc.aio.Server`` instance.
    """
    server = grpc.aio.server()
    add_AgentServiceServicer_to_server(AgentServicer(router), server)
    server.add_insecure_port(f"[::]:{port}")
    await server.start()
    log.info("llm-adapter serving", extra={"port": port})
    return server


async def _wait_for_signal() -> None:
    """Block until SIGTERM or SIGINT is received."""
    stop_event = asyncio.Event()
    loop = asyncio.get_running_loop()
    loop.add_signal_handler(signal.SIGTERM, stop_event.set)
    loop.add_signal_handler(signal.SIGINT, stop_event.set)
    await stop_event.wait()


async def _shutdown(
    agent_id: str,
    stub: object,
    server: grpc.aio.Server,
    channel: grpc.aio.Channel,
) -> None:
    """Deregister, stop the gRPC server, and close the registry channel.

    Args:
        agent_id: The registered agent identifier to deregister.
        stub: Registry stub used for DeregisterAgent.
        server: The running gRPC server to stop.
        channel: The registry channel to close.
    """
    try:
        await deregister_agent(agent_id, stub)
    except Exception as exc:
        log.warning("deregister failed", extra={"err": str(exc)})
    await server.stop(grace=5)
    await channel.close()
    log.info("llm-adapter stopped")


if __name__ == "__main__":
    asyncio.run(main())
