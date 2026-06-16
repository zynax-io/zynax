# SPDX-License-Identifier: Apache-2.0
"""BDD step definitions for echo.feature."""

import json
import types

from pytest_bdd import given, parsers, scenarios, then, when
from zynax.v1 import agent_pb2

from echo_agent import EchoAgent

scenarios("features/echo.feature")


def _fake_context():
    """A no-op gRPC ServicerContext stand-in for in-process dispatch."""
    return types.SimpleNamespace(abort=lambda code, msg: None)


def _run(agent, payload_bytes):
    req = agent_pb2.ExecuteCapabilityRequest(
        capability_name="echo",
        task_id="task-echo-1",
        input_payload=payload_bytes,
    )
    return list(agent.ExecuteCapability(req, _fake_context()))


@given("an EchoAgent", target_fixture="state")
def _agent():
    return types.SimpleNamespace(agent=EchoAgent(), events=[])


@when(parsers.parse("echo is called with payload {payload}"))
def _call_with_payload(payload, state):
    state.events = _run(state.agent, payload.encode())


@when("echo is called with an empty payload")
def _call_empty(state):
    state.events = _run(state.agent, b"")


@then("the stream emits a PROGRESS event")
def _has_progress(state):
    assert any(e.event_type == agent_pb2.TASK_EVENT_TYPE_PROGRESS for e in state.events)


@then("the final event is COMPLETED")
def _final_completed(state):
    assert state.events
    assert state.events[-1].event_type == agent_pb2.TASK_EVENT_TYPE_COMPLETED


@then(parsers.parse("the completed payload echoes {expected}"))
def _payload_echoes(expected, state):
    completed = state.events[-1]
    body = json.loads(completed.payload)
    assert body["echo"] == json.loads(expected)
