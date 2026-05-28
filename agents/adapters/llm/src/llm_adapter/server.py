# SPDX-License-Identifier: Apache-2.0
"""AgentServicer — grpc.aio servicer implementing AgentService for the llm-adapter."""

from __future__ import annotations

import grpc
import grpc.aio  # type: ignore[import-untyped]
from zynax.v1 import agent_pb2  # type: ignore[import-untyped]
from zynax.v1.agent_pb2_grpc import AgentServiceServicer  # type: ignore[import-untyped]

from llm_adapter.router import CapabilityRouter


class AgentServicer(AgentServiceServicer):
    """Async gRPC servicer implementing the AgentService contract for llm-adapter."""

    def __init__(self, router: CapabilityRouter) -> None:
        """Initialise with a pre-built capability router.

        Args:
            router: Immutable router built from ``ProviderConfig`` at startup.
        """
        self._router = router

    async def ExecuteCapability(  # type: ignore[override]
        self,
        request: agent_pb2.ExecuteCapabilityRequest,
        context: grpc.aio.ServicerContext,  # type: ignore[type-arg]
    ) -> object:
        """Stream TaskEvents for the requested capability.

        Validates the capability name, delegates execution to the registered
        handler, and streams PROGRESS followed by a terminal event.

        Args:
            request: gRPC request containing capability_name and input_payload.
            context: Async gRPC context for status/abort signalling.

        Yields:
            TaskEvent messages (PROGRESS × N, then COMPLETED or FAILED).
        """
        if not request.capability_name:
            await context.abort(grpc.StatusCode.INVALID_ARGUMENT, "capability_name is required")
            return

        try:
            handler = self._router.dispatch(request.capability_name)
        except KeyError:
            await context.abort(
                grpc.StatusCode.NOT_FOUND,
                f"unknown capability: {request.capability_name}",
            )
            return

        async for event in handler.stream(
            request.task_id, request.input_payload, request.timeout_seconds
        ):
            yield event

    async def GetCapabilitySchema(  # type: ignore[override]
        self,
        request: agent_pb2.GetCapabilitySchemaRequest,
        context: grpc.aio.ServicerContext,  # type: ignore[type-arg]
    ) -> agent_pb2.GetCapabilitySchemaResponse:
        """Return the JSON Schema for a named capability.

        Args:
            request: Contains ``capability_name`` to look up.
            context: Async gRPC context for status/abort signalling.

        Returns:
            Response with ``input_schema_json`` and ``output_schema_json``.
        """
        if not request.capability_name:
            await context.abort(grpc.StatusCode.INVALID_ARGUMENT, "capability_name is required")
            return agent_pb2.GetCapabilitySchemaResponse()

        try:
            inp, out = self._router.get_schema(request.capability_name)
        except KeyError:
            await context.abort(
                grpc.StatusCode.NOT_FOUND,
                f"unknown capability: {request.capability_name}",
            )
            return agent_pb2.GetCapabilitySchemaResponse()

        return agent_pb2.GetCapabilitySchemaResponse(
            capability_name=request.capability_name,
            input_schema_json=inp.decode(),
            output_schema_json=out.decode(),
        )
