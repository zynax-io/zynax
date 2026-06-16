# SPDX-License-Identifier: Apache-2.0
"""BDD step definitions for go_review.feature."""

import json
import types

from pytest_bdd import given, parsers, scenarios, then, when
from zynax.v1 import agent_pb2

from go_review_expert import GoReviewExpert

scenarios("features/go_review.feature")


def _fake_context():
    return types.SimpleNamespace(abort=lambda code, msg: None)


def _run(agent, diff):
    payload = {"diff": diff} if diff is not None else {}
    req = agent_pb2.ExecuteCapabilityRequest(
        capability_name="go_review",
        task_id="task-review-1",
        input_payload=json.dumps(payload).encode(),
    )
    return list(agent.ExecuteCapability(req, _fake_context()))


@given("a GoReviewExpert", target_fixture="state")
def _agent():
    return types.SimpleNamespace(agent=GoReviewExpert(), events=[])


@when("go_review is called with diff:")
def _call_with_diff(state, docstring):
    state.events = _run(state.agent, docstring)


@when("go_review is called with an empty diff")
def _call_empty(state):
    state.events = _run(state.agent, None)


def _completed_body(state):
    return json.loads(state.events[-1].payload)


@then("the final event is COMPLETED")
def _final_completed(state):
    assert state.events[-1].event_type == agent_pb2.TASK_EVENT_TYPE_COMPLETED


@then("the final event is FAILED")
def _final_failed(state):
    assert state.events[-1].event_type == agent_pb2.TASK_EVENT_TYPE_FAILED


@then(parsers.parse("the finding_count is {n:d}"))
def _finding_count(n, state):
    assert _completed_body(state)["finding_count"] == n


@then("the review is approved")
def _approved(state):
    assert _completed_body(state)["approved"] is True


@then("the review is not approved")
def _not_approved(state):
    assert _completed_body(state)["approved"] is False


@then(parsers.parse('there is an "{severity}" finding with message containing "{text}"'))
def _has_finding(severity, text, state):
    findings = _completed_body(state)["findings"]
    assert any(f["severity"] == severity and text in f["message"] for f in findings), (
        f"no {severity} finding mentioning {text!r} in {findings}"
    )


@then(parsers.parse('the error code is "{code}"'))
def _error_code(code, state):
    assert state.events[-1].error.code == code
