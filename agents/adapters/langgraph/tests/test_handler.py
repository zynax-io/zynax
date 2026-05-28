# SPDX-License-Identifier: Apache-2.0
"""Unit tests for LangGraphHandler — streaming, ticker, timeout, and error mapping."""

from __future__ import annotations

import asyncio
import json
from collections.abc import AsyncIterator
from unittest.mock import MagicMock

import pytest

from langgraph_adapter.handler import LangGraphHandler

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


async def _collect(gen: AsyncIterator) -> list:
    return [ev async for ev in gen]


def _input(payload: dict) -> bytes:
    return json.dumps(payload).encode()


def _make_graph(chunks: list[dict]) -> MagicMock:
    """Build a mock compiled graph whose astream yields the given chunks."""

    async def _astream(*_args, **_kwargs):
        for c in chunks:
            yield c

    mock = MagicMock()
    mock.astream = MagicMock(return_value=_astream())
    return mock


def _make_graph_factory(chunks_list: list[list[dict]]) -> MagicMock:
    """Build a graph mock that yields a fresh astream on each call."""
    call_count = 0

    async def _astream(*_args, **_kwargs):
        nonlocal call_count
        for c in chunks_list[call_count - 1]:
            yield c

    def _make_mock_astream(*_args, **_kwargs):
        nonlocal call_count
        call_count += 1
        return _astream(*_args, **_kwargs)

    mock = MagicMock()
    mock.astream = MagicMock(side_effect=_make_mock_astream)
    return mock


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


class TestLangGraphHandlerStream:
    """stream() yields PROGRESS per node then COMPLETED with final state."""

    @pytest.mark.asyncio
    async def test_yields_progress_per_node_then_completed(self) -> None:
        graph = _make_graph([{"node_a": {"x": 1}}, {"node_b": {"y": 2}}])
        handler = LangGraphHandler()
        events = await _collect(handler.stream(graph, "t1", _input({"query": "hi"}), 30.0))

        types = [ev.event_type for ev in events]
        from zynax.v1 import agent_pb2

        assert types[0] == agent_pb2.TASK_EVENT_TYPE_PROGRESS
        assert types[1] == agent_pb2.TASK_EVENT_TYPE_PROGRESS
        assert types[-1] == agent_pb2.TASK_EVENT_TYPE_COMPLETED

    @pytest.mark.asyncio
    async def test_progress_payload_contains_node_and_update(self) -> None:
        graph = _make_graph([{"my_node": {"result": "ok"}}])
        handler = LangGraphHandler()
        events = await _collect(handler.stream(graph, "t1", _input({}), 30.0))

        progress = events[0]
        payload = json.loads(progress.payload)
        assert payload["node"] == "my_node"
        assert payload["update"] == {"result": "ok"}

    @pytest.mark.asyncio
    async def test_completed_payload_contains_final_state(self) -> None:
        graph = _make_graph([{"n": {"k": "v"}}])
        handler = LangGraphHandler()
        events = await _collect(handler.stream(graph, "t1", _input({}), 30.0))

        completed = events[-1]
        state = json.loads(completed.payload.decode())
        assert state.get("k") == "v"

    @pytest.mark.asyncio
    async def test_task_id_echoed_on_every_event(self) -> None:
        graph = _make_graph([{"n": {"x": 1}}])
        handler = LangGraphHandler()
        events = await _collect(handler.stream(graph, "my-task-id", _input({}), 30.0))
        assert all(ev.task_id == "my-task-id" for ev in events)

    @pytest.mark.asyncio
    async def test_empty_graph_emits_completed_only(self) -> None:
        graph = _make_graph([])
        handler = LangGraphHandler()
        events = await _collect(handler.stream(graph, "t1", _input({}), 30.0))

        from zynax.v1 import agent_pb2

        assert len(events) == 1
        assert events[0].event_type == agent_pb2.TASK_EVENT_TYPE_COMPLETED


class TestLangGraphHandlerErrors:
    """Error paths yield exactly one FAILED event."""

    @pytest.mark.asyncio
    async def test_invalid_json_payload_yields_failed(self) -> None:
        handler = LangGraphHandler()
        graph = MagicMock()
        events = await _collect(handler.stream(graph, "t1", b"not json", 30.0))

        from zynax.v1 import agent_pb2

        assert len(events) == 1
        assert events[0].event_type == agent_pb2.TASK_EVENT_TYPE_FAILED
        assert events[0].error.code == "INVALID_INPUT"

    @pytest.mark.asyncio
    async def test_graph_value_error_yields_invalid_input(self) -> None:
        async def _astream(*_a, **_kw):
            raise ValueError("bad state")
            yield  # make async generator

        graph = MagicMock()
        graph.astream = MagicMock(return_value=_astream())
        handler = LangGraphHandler()
        events = await _collect(handler.stream(graph, "t1", _input({}), 30.0))

        from zynax.v1 import agent_pb2

        assert events[-1].event_type == agent_pb2.TASK_EVENT_TYPE_FAILED
        assert events[-1].error.code == "INVALID_INPUT"

    @pytest.mark.asyncio
    async def test_graph_runtime_error_yields_internal(self) -> None:
        async def _astream(*_a, **_kw):
            raise RuntimeError("boom")
            yield  # pragma: no cover — makes this an async generator

        graph = MagicMock()
        graph.astream = MagicMock(return_value=_astream())
        handler = LangGraphHandler()
        events = await _collect(handler.stream(graph, "t1", _input({}), 30.0))

        from zynax.v1 import agent_pb2

        assert events[-1].event_type == agent_pb2.TASK_EVENT_TYPE_FAILED
        assert events[-1].error.code == "INTERNAL"

    @pytest.mark.asyncio
    async def test_timeout_yields_failed_timeout(self) -> None:
        async def _slow(*_a, **_kw):
            await asyncio.sleep(10)
            yield {"n": {}}

        graph = MagicMock()
        graph.astream = MagicMock(return_value=_slow())
        handler = LangGraphHandler()
        events = await _collect(handler.stream(graph, "t1", _input({}), 0.05))

        from zynax.v1 import agent_pb2

        assert events[-1].event_type == agent_pb2.TASK_EVENT_TYPE_FAILED
        assert events[-1].error.code == "TIMEOUT"


class TestLangGraphHandlerTicker:
    """2-second ticker PROGRESS fires when a node takes > 2 s (mocked clock)."""

    @pytest.mark.asyncio
    async def test_ticker_fires_when_node_delays(self) -> None:
        call_count = 0

        async def _slow_astream(*_a, **_kw):
            nonlocal call_count
            call_count += 1
            await asyncio.sleep(0.15)  # > ticker (patched to 0.05)
            yield {"node_a": {"x": 1}}

        graph = MagicMock()
        graph.astream = MagicMock(return_value=_slow_astream())

        import langgraph_adapter.handler as _mod

        original = _mod._TICKER_SECS
        _mod._TICKER_SECS = 0.05
        try:
            handler = LangGraphHandler()
            events = await _collect(handler.stream(graph, "t1", _input({}), 5.0))
        finally:
            _mod._TICKER_SECS = original

        from zynax.v1 import agent_pb2

        ticker_events = [
            ev
            for ev in events
            if ev.event_type == agent_pb2.TASK_EVENT_TYPE_PROGRESS
            and json.loads(ev.payload).get("ticker")
        ]
        assert len(ticker_events) >= 1
