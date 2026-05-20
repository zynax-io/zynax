# SPDX-License-Identifier: Apache-2.0
"""Unit tests for the Agent base class — routing, events, validation, cancellation."""

from __future__ import annotations

import asyncio
import json
import os
import sys
from typing import Any
from unittest.mock import MagicMock

import pytest

sys.path.insert(
    0, os.path.join(os.path.dirname(__file__), "../../../protos/generated/python")
)
from zynax.v1 import agent_pb2  # noqa: E402

from zynax_sdk import Agent, capability  # noqa: E402


# ── test doubles ──────────────────────────────────────────────────────────────


class _MockContext:
    """Minimal gRPC context mock that records abort() calls."""

    def __init__(self) -> None:
        self.aborted = False
        self.abort_code: Any = None
        self.abort_details: str = ""

    def abort(self, code: Any, details: str) -> None:
        self.aborted = True
        self.abort_code = code
        self.abort_details = details


class _GreetAgent(Agent):
    """Agent with a single 'greet' capability that emits PROGRESS then COMPLETED."""

    @capability("greet")
    async def greet(self, request: Any, context: Any) -> Any:
        yield self.report_progress(request.task_id, {"step": 1})
        yield self.report_completed(request.task_id, {"greeting": "hello"})


class _MultiCapAgent(Agent):
    """Agent with two capabilities for multi-registration tests."""

    @capability("ping")
    async def ping(self, request: Any, context: Any) -> Any:
        yield self.report_completed(request.task_id, {"pong": True})

    @capability("echo")
    async def echo(self, request: Any, context: Any) -> Any:
        yield self.report_completed(request.task_id, {"payload": "echoed"})


class _CancelAgent(Agent):
    """Agent whose handler raises asyncio.CancelledError immediately."""

    @capability("cancel_me")
    async def cancel_me(self, request: Any, context: Any) -> Any:
        raise asyncio.CancelledError("simulated cancel")
        yield  # pragma: no cover — required to make this an async generator


# ── helpers ───────────────────────────────────────────────────────────────────


def _make_request(**kwargs: Any) -> Any:
    defaults: dict[str, Any] = dict(
        capability_name="greet",
        task_id="task-1",
        workflow_id="wf-1",
        input_payload=b'{"name": "world"}',
        timeout_seconds=0,
    )
    defaults.update(kwargs)
    return agent_pb2.ExecuteCapabilityRequest(**defaults)


def _collect(gen: Any) -> list[Any]:
    return list(gen)


# ── routing ───────────────────────────────────────────────────────────────────


class TestRouting:
    def setup_method(self) -> None:
        self.agent = _GreetAgent()

    def test_routing_hit_calls_handler(self) -> None:
        """@capability("greet") handler is called when capability_name == "greet"."""
        events = _collect(self.agent.ExecuteCapability(_make_request(), _MockContext()))
        assert len(events) == 2
        assert events[0].event_type == agent_pb2.TASK_EVENT_TYPE_PROGRESS
        assert events[1].event_type == agent_pb2.TASK_EVENT_TYPE_COMPLETED

    def test_routing_miss_yields_exactly_one_failed_event(self) -> None:
        """Unknown capability_name → exactly one FAILED event, CAPABILITY_NOT_FOUND."""
        events = _collect(
            self.agent.ExecuteCapability(
                _make_request(capability_name="unknown"), _MockContext()
            )
        )
        assert len(events) == 1
        assert events[0].event_type == agent_pb2.TASK_EVENT_TYPE_FAILED
        assert events[0].error.code == "CAPABILITY_NOT_FOUND"

    def test_routing_miss_does_not_raise(self) -> None:
        """Unknown capability must not raise an exception."""
        try:
            _collect(
                self.agent.ExecuteCapability(
                    _make_request(capability_name="does_not_exist"), _MockContext()
                )
            )
        except Exception as exc:  # pragma: no cover
            pytest.fail(f"ExecuteCapability raised unexpectedly: {exc}")

    def test_multiple_capabilities_routed_independently(self) -> None:
        agent = _MultiCapAgent()
        ping_events = _collect(
            agent.ExecuteCapability(
                _make_request(capability_name="ping"), _MockContext()
            )
        )
        echo_events = _collect(
            agent.ExecuteCapability(
                _make_request(capability_name="echo"), _MockContext()
            )
        )
        assert ping_events[0].event_type == agent_pb2.TASK_EVENT_TYPE_COMPLETED
        assert echo_events[0].event_type == agent_pb2.TASK_EVENT_TYPE_COMPLETED


# ── event helpers ─────────────────────────────────────────────────────────────


class TestEventHelpers:
    def test_report_progress_fields(self) -> None:
        ev = Agent.report_progress("task-1", {"n": 1})
        assert ev.event_type == agent_pb2.TASK_EVENT_TYPE_PROGRESS
        assert ev.task_id == "task-1"
        assert json.loads(ev.payload) == {"n": 1}
        assert ev.timestamp.seconds > 0

    def test_report_completed_fields(self) -> None:
        ev = Agent.report_completed("task-2", {})
        assert ev.event_type == agent_pb2.TASK_EVENT_TYPE_COMPLETED
        assert ev.task_id == "task-2"
        assert ev.timestamp.seconds > 0

    def test_report_failed_fields(self) -> None:
        ev = Agent.report_failed("task-3", "TIMEOUT", "timed out")
        assert ev.event_type == agent_pb2.TASK_EVENT_TYPE_FAILED
        assert ev.task_id == "task-3"
        assert ev.error.code == "TIMEOUT"
        assert ev.error.message == "timed out"
        assert ev.timestamp.seconds > 0

    def test_report_failed_error_code_and_message_nonempty(self) -> None:
        ev = Agent.report_failed("task-4", "INTERNAL", "something broke")
        assert ev.error.code
        assert ev.error.message


# ── task_id propagation ───────────────────────────────────────────────────────


class TestTaskIdPropagation:
    def test_every_event_carries_task_id(self) -> None:
        agent = _GreetAgent()
        events = _collect(
            agent.ExecuteCapability(_make_request(task_id="my-task-99"), _MockContext())
        )
        assert events, "expected at least one event"
        for ev in events:
            assert ev.task_id == "my-task-99", f"wrong task_id on {ev}"


# ── input validation ──────────────────────────────────────────────────────────


class TestValidation:
    def setup_method(self) -> None:
        self.agent = _GreetAgent()

    def test_empty_capability_name_aborts(self) -> None:
        ctx = _MockContext()
        _collect(self.agent.ExecuteCapability(_make_request(capability_name=""), ctx))
        assert ctx.aborted

    def test_empty_task_id_aborts(self) -> None:
        ctx = _MockContext()
        _collect(self.agent.ExecuteCapability(_make_request(task_id=""), ctx))
        assert ctx.aborted

    def test_invalid_json_payload_aborts(self) -> None:
        ctx = _MockContext()
        _collect(
            self.agent.ExecuteCapability(_make_request(input_payload=b"not-json"), ctx)
        )
        assert ctx.aborted

    def test_valid_payload_does_not_abort(self) -> None:
        ctx = _MockContext()
        events = _collect(
            self.agent.ExecuteCapability(
                _make_request(input_payload=b'{"key": "value"}'), ctx
            )
        )
        assert not ctx.aborted
        assert events


# ── GetCapabilitySchema ───────────────────────────────────────────────────────


class TestGetCapabilitySchema:
    def setup_method(self) -> None:
        self.agent = _GreetAgent()

    def test_schema_for_known_capability(self) -> None:
        import grpc

        ctx = MagicMock(spec=grpc.ServicerContext)
        resp = self.agent.GetCapabilitySchema(
            agent_pb2.GetCapabilitySchemaRequest(capability_name="greet"), ctx
        )
        assert resp is not None
        assert resp.capability_name == "greet"
        assert json.loads(resp.input_schema_json) == {}
        assert json.loads(resp.output_schema_json) == {}
        assert resp.description

    def test_schema_unknown_capability_aborts(self) -> None:
        ctx = _MockContext()
        result = self.agent.GetCapabilitySchema(
            agent_pb2.GetCapabilitySchemaRequest(capability_name="unknown"), ctx
        )
        assert ctx.aborted
        assert result is None

    def test_schema_empty_capability_name_aborts(self) -> None:
        ctx = _MockContext()
        result = self.agent.GetCapabilitySchema(
            agent_pb2.GetCapabilitySchemaRequest(capability_name=""), ctx
        )
        assert ctx.aborted
        assert result is None


# ── cancellation ──────────────────────────────────────────────────────────────


class TestCancellation:
    def test_cancelled_error_propagates_and_is_not_swallowed(self) -> None:
        """Agent must not swallow asyncio.CancelledError from a handler."""
        agent = _CancelAgent()
        with pytest.raises(asyncio.CancelledError):
            _collect(
                agent.ExecuteCapability(
                    _make_request(capability_name="cancel_me"), _MockContext()
                )
            )
