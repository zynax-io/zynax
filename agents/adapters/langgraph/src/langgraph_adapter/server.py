# SPDX-License-Identifier: Apache-2.0
"""AgentServicer — gRPC servicer wiring ExecuteCapability and GetCapabilitySchema."""

from __future__ import annotations

from collections.abc import AsyncGenerator

import grpc
import structlog
from zynax.v1 import agent_pb2
from zynax.v1.agent_pb2_grpc import AgentServiceServicer

from langgraph_adapter.handler import LangGraphHandler
from langgraph_adapter.router import CapabilityRouter

log = structlog.get_logger()


class AgentServicer(AgentServiceServicer):
    """gRPC ``AgentService`` implementation for the langgraph-adapter."""

    def __init__(self, router: CapabilityRouter, handler: LangGraphHandler) -> None:
        """Initialise with a pre-built router and a shared handler instance."""
        self._router = router
        self._handler = handler

    async def ExecuteCapability(  # type: ignore[override]
        self,
        request: agent_pb2.ExecuteCapabilityRequest,
        context: grpc.aio.ServicerContext,
    ) -> AsyncGenerator[agent_pb2.TaskEvent, None]:
        """Stream TaskEvents for the requested capability."""
        if not request.capability_name:
            await context.abort(grpc.StatusCode.INVALID_ARGUMENT, "capability_name is required")
            return
        try:
            graph = self._router.dispatch(request.capability_name)
        except KeyError:
            await context.abort(
                grpc.StatusCode.NOT_FOUND,
                f"unknown capability: {request.capability_name}",
            )
            return
        async for event in self._handler.stream(
            graph,
            request.task_id,
            request.input_payload,
            float(request.timeout_seconds or 60),
        ):
            yield event

    async def GetCapabilitySchema(  # type: ignore[override]
        self,
        request: agent_pb2.GetCapabilitySchemaRequest,
        context: grpc.aio.ServicerContext,
    ) -> agent_pb2.GetCapabilitySchemaResponse:
        """Return JSON Schema bytes for a registered capability."""
        try:
            inp, out = self._router.get_schema(request.capability_name)
        except KeyError:
            await context.abort(
                grpc.StatusCode.NOT_FOUND,
                f"unknown capability: {request.capability_name}",
            )
            return agent_pb2.GetCapabilitySchemaResponse()
        return agent_pb2.GetCapabilitySchemaResponse(input_schema=inp, output_schema=out)
