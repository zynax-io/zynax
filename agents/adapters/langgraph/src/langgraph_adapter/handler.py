# SPDX-License-Identifier: Apache-2.0
"""LangGraphHandler — streams per-node PROGRESS events then COMPLETED from a compiled graph."""

from __future__ import annotations

import asyncio
import json
from collections.abc import AsyncGenerator
from typing import Any

import structlog
from google.protobuf import timestamp_pb2
from zynax.v1 import agent_pb2

log = structlog.get_logger()

_TICKER_SECS = 2.0
_DONE = object()


def _ts() -> timestamp_pb2.Timestamp:
    ts = timestamp_pb2.Timestamp()
    ts.GetCurrentTime()
    return ts


def _progress(task_id: str, payload: dict[str, Any]) -> agent_pb2.TaskEvent:
    return agent_pb2.TaskEvent(
        task_id=task_id,
        event_type=agent_pb2.TASK_EVENT_TYPE_PROGRESS,
        payload=json.dumps(payload, default=str).encode(),
        timestamp=_ts(),
    )


def _ticker(task_id: str) -> agent_pb2.TaskEvent:
    return _progress(task_id, {"ticker": True})


def _completed(task_id: str, payload_str: str) -> agent_pb2.TaskEvent:
    return agent_pb2.TaskEvent(
        task_id=task_id,
        event_type=agent_pb2.TASK_EVENT_TYPE_COMPLETED,
        payload=payload_str.encode(),
        timestamp=_ts(),
    )


def _failed(task_id: str, code: str, msg: str) -> agent_pb2.TaskEvent:
    return agent_pb2.TaskEvent(
        task_id=task_id,
        event_type=agent_pb2.TASK_EVENT_TYPE_FAILED,
        error=agent_pb2.CapabilityError(code=code, message=msg[:512]),
        timestamp=_ts(),
    )


async def _anext_or_done(ait: Any) -> Any:
    """Wrap __anext__ so StopAsyncIteration never propagates through asyncio.Task."""
    try:
        return await ait.__anext__()
    except StopAsyncIteration:
        return _DONE


async def _iter_nodes(ait: Any, deadline: float) -> AsyncGenerator[tuple[str, Any] | None, None]:
    """Yield ``(node, update)`` tuples or ``None`` as a 2-second ticker signal."""
    while True:
        remaining = deadline - asyncio.get_running_loop().time()
        if remaining <= 0:
            raise TimeoutError()
        try:
            chunk = await asyncio.wait_for(_anext_or_done(ait), min(_TICKER_SECS, remaining))
        except TimeoutError:
            if asyncio.get_running_loop().time() >= deadline:
                raise
            yield None
            continue
        if chunk is _DONE:
            return
        for node, upd in chunk.items():
            yield (node, upd)


async def _graph_events(
    graph: Any, input_state: dict[str, Any], task_id: str, deadline: float
) -> AsyncGenerator[agent_pb2.TaskEvent, None]:
    """Stream PROGRESS events from graph.astream(), then emit COMPLETED."""
    final_state: dict[str, Any] = {}
    ait = graph.astream(input_state, stream_mode="updates")
    try:
        async for item in _iter_nodes(ait, deadline):
            if item is None:
                yield _ticker(task_id)
            else:
                node, upd = item
                yield _progress(task_id, {"node": node, "update": upd})
                if isinstance(upd, dict):
                    final_state.update(upd)
    finally:
        await ait.aclose()
    yield _completed(task_id, json.dumps(final_state, default=str))


class LangGraphHandler:
    """Streams TaskEvents for a single compiled LangGraph capability invocation."""

    async def stream(
        self,
        graph: Any,
        task_id: str,
        input_payload: bytes,
        timeout_seconds: float,
    ) -> AsyncGenerator[agent_pb2.TaskEvent, None]:
        """Execute the compiled graph and stream TaskEvents.

        Args:
            graph: Compiled LangGraph graph (``CompiledStateGraph``).
            task_id: Task identifier echoed on every event.
            input_payload: JSON-encoded graph input state.
            timeout_seconds: Hard wall-clock limit for the entire invocation.

        Yields:
            ``PROGRESS`` events per node (or 2-second ticker), then
            exactly one terminal ``COMPLETED`` or ``FAILED`` event.
        """
        try:
            input_state: dict[str, Any] = json.loads(input_payload)
        except (json.JSONDecodeError, ValueError) as exc:
            yield _failed(task_id, "INVALID_INPUT", str(exc)[:512])
            return
        deadline = asyncio.get_running_loop().time() + timeout_seconds
        try:
            async for event in _graph_events(graph, input_state, task_id, deadline):
                yield event
        except TimeoutError:
            yield _failed(task_id, "TIMEOUT", "request exceeded timeout")
        except ValueError as exc:
            yield _failed(task_id, "INVALID_INPUT", str(exc)[:512])
        except Exception as exc:
            log.warning("graph_execution_error", error=str(exc)[:256])
            yield _failed(task_id, "INTERNAL", str(exc)[:512])
