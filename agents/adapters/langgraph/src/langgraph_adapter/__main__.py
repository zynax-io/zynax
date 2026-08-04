# SPDX-License-Identifier: Apache-2.0
"""langgraph-adapter bootstrap — load graphs, serve, drain on SIGTERM.

Agent identity and capabilities are declared by the Agent custom resource
(zynax.io/v1alpha1, ADR-039); the adapter announces nothing at boot.
"""

from __future__ import annotations

import asyncio
import logging
import signal
import sys

import grpc.aio  # type: ignore[import-untyped]
from google.protobuf import (
    timestamp_pb2 as _timestamp_pb2,  # noqa: F401 — must precede zynax pb2 imports to seed the descriptor pool
)
from zynax.v1.agent_pb2_grpc import (
    add_AgentServiceServicer_to_server,  # type: ignore[import-untyped]
)

from langgraph_adapter.config import AdapterConfig
from langgraph_adapter.graph_loader import GraphLoader
from langgraph_adapter.handler import LangGraphHandler
from langgraph_adapter.router import CapabilityRouter
from langgraph_adapter.server import AgentServicer

log = logging.getLogger(__name__)


async def main() -> None:
    """Bootstrap: load graphs, serve the AgentService, drain on SIGTERM."""
    logging.basicConfig(level=logging.INFO, format="%(message)s")
    config = AdapterConfig()  # type: ignore[call-arg]
    graphs = _load_graphs(config)
    router = CapabilityRouter(graphs)
    server = await _start_server(router, config.grpc_port)
    await _wait_for_signal()
    await _shutdown(server)


def _load_graphs(config: AdapterConfig) -> dict[str, object]:
    """Load and compile all graphs declared in config, exit on failure.

    Args:
        config: Validated adapter config containing the list of graph mounts.

    Returns:
        Map of capability name → compiled graph (exits non-zero on any failure).
    """
    try:
        return GraphLoader.load_all(config.graph_mounts)
    except RuntimeError as exc:
        log.error("graph load failed — aborting", extra={"err": str(exc)})
        sys.exit(1)


async def _start_server(router: CapabilityRouter, port: int) -> grpc.aio.Server:
    """Create and start the gRPC server on the given port.

    Args:
        router: The capability router wired into the servicer.
        port: TCP port to bind (``[::]:<port>``).

    Returns:
        A started ``grpc.aio.Server`` instance.
    """
    server = grpc.aio.server()
    handler = LangGraphHandler()
    add_AgentServiceServicer_to_server(AgentServicer(router, handler), server)
    server.add_insecure_port(f"[::]:{port}")
    await server.start()
    log.info("langgraph-adapter serving", extra={"port": port})
    return server


async def _wait_for_signal() -> None:
    """Block until SIGTERM or SIGINT is received."""
    stop_event = asyncio.Event()
    loop = asyncio.get_running_loop()
    loop.add_signal_handler(signal.SIGTERM, stop_event.set)
    loop.add_signal_handler(signal.SIGINT, stop_event.set)
    await stop_event.wait()


async def _shutdown(server: grpc.aio.Server) -> None:
    """Stop the gRPC server, draining in-flight calls.

    Args:
        server: The running gRPC server to stop.
    """
    await server.stop(grace=5)
    log.info("langgraph-adapter stopped")


if __name__ == "__main__":
    asyncio.run(main())
