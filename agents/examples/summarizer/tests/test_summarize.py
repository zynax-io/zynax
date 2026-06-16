# SPDX-License-Identifier: Apache-2.0
"""BDD step definitions for summarize.feature."""

import json
import types

from pytest_bdd import given, parsers, scenarios, then, when
from zynax.v1 import agent_pb2

from summarizer_agent import SummarizerAgent

scenarios("features/summarize.feature")


def _fake_context():
    return types.SimpleNamespace(abort=lambda code, msg: None)


def _run(agent, payload):
    req = agent_pb2.ExecuteCapabilityRequest(
        capability_name="summarize",
        task_id="task-sum-1",
        input_payload=json.dumps(payload).encode() if payload is not None else b"",
    )
    return list(agent.ExecuteCapability(req, _fake_context()))


@given("a SummarizerAgent", target_fixture="state")
def _agent():
    return types.SimpleNamespace(agent=SummarizerAgent(), events=[])


@when("summarize is called with documents:")
def _call_with_docs(datatable, state):
    # datatable is a list of rows; row[0] is the header, each later row is one cell.
    docs = [row[0] for row in datatable[1:]]
    state.events = _run(state.agent, {"documents": docs})


@when("summarize is called with no documents")
def _call_no_docs(state):
    state.events = _run(state.agent, {})


def _completed_body(state):
    return json.loads(state.events[-1].payload)


@then("the final event is COMPLETED")
def _final_completed(state):
    assert state.events[-1].event_type == agent_pb2.TASK_EVENT_TYPE_COMPLETED


@then("the final event is FAILED")
def _final_failed(state):
    assert state.events[-1].event_type == agent_pb2.TASK_EVENT_TYPE_FAILED


@then(parsers.parse('the summary contains "{text}"'))
def _summary_contains(text, state):
    assert text in _completed_body(state)["summary"]


@then(parsers.parse("the document_count is {n:d}"))
def _doc_count(n, state):
    assert _completed_body(state)["document_count"] == n


@then(parsers.parse('the error code is "{code}"'))
def _error_code(code, state):
    assert state.events[-1].error.code == code
