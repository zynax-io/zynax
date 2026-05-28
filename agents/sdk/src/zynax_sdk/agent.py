# SPDX-License-Identifier: Apache-2.0
"""Zynax Agent base class — abstract runtime for capability providers."""

from __future__ import annotations

import asyncio
import inspect
import json
import time
from abc import ABC
from typing import Any, Callable, ClassVar, Generator

import grpc
from google.protobuf import timestamp_pb2

from zynax.v1 import agent_pb2, agent_pb2_grpc


def capability(name: str) -> Callable[[Callable[..., Any]], Callable[..., Any]]:
    """Register a method as a named Zynax capability handler.

    Decorated methods must be async generators that yield TaskEvent objects::

        @capability("summarize")
        async def summarize(self, request: Any, context: Any) -> AsyncGenerator[Any, None]:
            yield self.report_progress(request.task_id, {"step": 1})
            yield self.report_completed(request.task_id, {"summary": "done"})

    Args:
        name: The capability identifier used in ``ExecuteCapabilityRequest.capability_name``.

    Returns:
        A decorator that attaches ``_capability_name`` to the wrapped method and
        registers it during ``Agent.__init_subclass__``.
    """

    def decorator(func: Callable[..., Any]) -> Callable[..., Any]:
        func._capability_name = name  # type: ignore[attr-defined]
        return func

    return decorator


def _now() -> timestamp_pb2.Timestamp:
    ts = timestamp_pb2.Timestamp()
    ts.seconds = int(time.time())
    return ts


def _is_valid_json(data: bytes) -> bool:
    try:
        json.loads(data)
        return True
    except (ValueError, TypeError):
        return False


class Agent(agent_pb2_grpc.AgentServiceServicer, ABC):  # type: ignore[misc]
    """Abstract base class for Zynax capability providers.

    Subclass and decorate async generator methods with ``@capability("name")``.
    The gRPC server lifecycle is the caller's responsibility.

    Example::

        class Summarizer(Agent):
            @capability("summarize")
            async def summarize(self, request: Any, context: Any) -> AsyncGenerator[Any, None]:
                yield self.report_progress(request.task_id, {"step": 1})
                yield self.report_completed(request.task_id, {"summary": "done"})
    """

    _capabilities: ClassVar[dict[str, Callable[..., Any]]] = {}

    def __init_subclass__(cls, **kwargs: Any) -> None:
        super().__init_subclass__(**kwargs)
        caps: dict[str, Callable[..., Any]] = {}
        for _attr_name, method in inspect.getmembers(cls, predicate=callable):
            cap_name: str | None = getattr(method, "_capability_name", None)
            if cap_name is not None:
                caps[cap_name] = method
        cls._capabilities = caps

    def ExecuteCapability(
        self,
        request: Any,
        context: Any,
    ) -> Generator[Any, None, None]:
        """Route the request to the registered capability handler and stream events.

        Validates ``capability_name``, ``task_id``, and ``input_payload`` before
        dispatching.  Unknown capabilities yield a FAILED event rather than raising.

        Args:
            request: ``ExecuteCapabilityRequest`` proto message.
            context: gRPC ``ServicerContext`` used to abort on validation failure.

        Yields:
            ``TaskEvent`` proto messages produced by the capability handler,
            followed by at most one FAILED event if the capability is not found.
        """
        if not request.capability_name:
            context.abort(
                grpc.StatusCode.INVALID_ARGUMENT, "capability_name must not be empty"
            )
            return
        if not request.task_id:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "task_id must not be empty")
            return
        if request.input_payload and not _is_valid_json(request.input_payload):
            context.abort(
                grpc.StatusCode.INVALID_ARGUMENT, "input_payload must be valid JSON"
            )
            return

        handler = self._capabilities.get(request.capability_name)
        if handler is None:
            yield self.report_failed(
                request.task_id,
                "CAPABILITY_NOT_FOUND",
                f"capability {request.capability_name!r} not found",
            )
            return

        loop = asyncio.new_event_loop()
        try:
            agen = handler(self, request, context)
            while True:
                try:
                    event = loop.run_until_complete(agen.__anext__())
                    yield event
                except StopAsyncIteration:
                    break
        finally:
            loop.close()

    def GetCapabilitySchema(
        self,
        request: Any,
        context: Any,
    ) -> Any:
        """Return schema metadata for a registered capability.

        Args:
            request: ``GetCapabilitySchemaRequest`` proto message containing
                ``capability_name``.
            context: gRPC ``ServicerContext`` used to abort on validation failure.

        Returns:
            ``GetCapabilitySchemaResponse`` with placeholder JSON schemas, or
            ``None`` after aborting the context when the capability is unknown.
        """
        if not request.capability_name:
            context.abort(
                grpc.StatusCode.INVALID_ARGUMENT, "capability_name must not be empty"
            )
            return None
        if request.capability_name not in self._capabilities:
            context.abort(
                grpc.StatusCode.NOT_FOUND,
                f"capability {request.capability_name!r} not found",
            )
            return None
        return agent_pb2.GetCapabilitySchemaResponse(
            capability_name=request.capability_name,
            input_schema_json="{}",
            output_schema_json="{}",
            description=f"Capability '{request.capability_name}'",
        )

    @staticmethod
    def report_progress(task_id: str, payload: dict[str, Any]) -> Any:
        """Create a PROGRESS TaskEvent.

        Args:
            task_id: Opaque identifier from the originating ``ExecuteCapabilityRequest``.
            payload: Arbitrary JSON-serialisable progress data yielded to the caller.

        Returns:
            A ``TaskEvent`` with ``event_type`` set to ``TASK_EVENT_TYPE_PROGRESS``.
        """
        return agent_pb2.TaskEvent(
            task_id=task_id,
            event_type=agent_pb2.TASK_EVENT_TYPE_PROGRESS,
            payload=json.dumps(payload).encode(),
            timestamp=_now(),
        )

    @staticmethod
    def report_completed(task_id: str, payload: dict[str, Any]) -> Any:
        """Create a COMPLETED TaskEvent.

        Args:
            task_id: Opaque identifier from the originating ``ExecuteCapabilityRequest``.
            payload: Final JSON-serialisable result data for the task.

        Returns:
            A ``TaskEvent`` with ``event_type`` set to ``TASK_EVENT_TYPE_COMPLETED``.
        """
        return agent_pb2.TaskEvent(
            task_id=task_id,
            event_type=agent_pb2.TASK_EVENT_TYPE_COMPLETED,
            payload=json.dumps(payload).encode(),
            timestamp=_now(),
        )

    @staticmethod
    def report_failed(task_id: str, code: str, message: str) -> Any:
        """Create a FAILED TaskEvent with structured CapabilityError.

        Args:
            task_id: Opaque identifier from the originating ``ExecuteCapabilityRequest``.
            code: Machine-readable error code (e.g. ``"CAPABILITY_NOT_FOUND"``).
            message: Human-readable description of the failure.

        Returns:
            A ``TaskEvent`` with ``event_type`` set to ``TASK_EVENT_TYPE_FAILED``
            and a populated ``CapabilityError`` field.
        """
        return agent_pb2.TaskEvent(
            task_id=task_id,
            event_type=agent_pb2.TASK_EVENT_TYPE_FAILED,
            timestamp=_now(),
            error=agent_pb2.CapabilityError(code=code, message=message),
        )
