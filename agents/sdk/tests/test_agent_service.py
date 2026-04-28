# SPDX-License-Identifier: Apache-2.0
"""BDD step definitions for agent_service.feature."""
import json
import time

import grpc
import pytest
from pytest_bdd import scenarios, given, when, then, parsers

import sys, os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../protos/generated/python"))
from zynax.v1 import agent_pb2, agent_pb2_grpc

scenarios("features/agent_service.feature")


# ── helpers ───────────────────────────────────────────────────────────────────

def _stub(channel):
    return agent_pb2_grpc.AgentServiceStub(channel)


def _make_req(**kwargs):
    defaults = dict(
        capability_name="summarize",
        task_id="task-test-1",
        workflow_id="wf-1",
        input_payload=b'{"documents": ["hello"]}',
        timeout_seconds=0,
    )
    defaults.update(kwargs)
    return agent_pb2.ExecuteCapabilityRequest(**defaults)


def _collect(stream):
    """Drain a streaming RPC into (events_list, grpc_error_or_None)."""
    events = []
    error = None
    try:
        for ev in stream:
            events.append(ev)
    except grpc.RpcError as e:
        error = e
    return events, error


# ── Background ────────────────────────────────────────────────────────────────

@given("an agent implementing AgentService is running on a test gRPC server")
def _agent_service_running(grpc_channel, ctx):
    ctx.stub = _stub(grpc_channel)


# ── Given steps ───────────────────────────────────────────────────────────────

@given(parsers.parse('a valid ExecuteCapabilityRequest for capability "{cap}"'))
def _valid_req_for_cap(cap, ctx):
    ctx.req = _make_req(capability_name=cap)


@given("a valid ExecuteCapabilityRequest")
def _valid_req(ctx):
    ctx.req = _make_req()


@given(parsers.parse('the input payload is valid JSON: {payload}'))
def _set_payload(payload, ctx):
    ctx.req = _make_req(
        capability_name=ctx.req.capability_name,
        task_id=ctx.req.task_id,
        input_payload=payload.encode(),
    )


@given(parsers.parse('a valid ExecuteCapabilityRequest with task_id "{tid}"'))
def _valid_req_with_task_id(tid, ctx):
    ctx.req = _make_req(task_id=tid)


@given(parsers.parse("an ExecuteCapabilityRequest with timeout_seconds set to {n:d}"))
def _req_with_timeout(n, ctx):
    ctx.req = _make_req(timeout_seconds=n)


@given("the agent simulates a capability that runs for 5 seconds")
def _long_running_cap(ctx):
    ctx.req = _make_req(
        capability_name=ctx.req.capability_name,
        task_id=ctx.req.task_id,
        timeout_seconds=ctx.req.timeout_seconds,
    )


@given(parsers.parse('an ExecuteCapabilityRequest for capability "{cap}"'))
def _req_for_cap(cap, ctx):
    ctx.req = _make_req(capability_name=cap)


@given(parsers.parse('an ExecuteCapabilityRequest with capability_name set to "{val}"'))
def _req_empty_cap(val, ctx):
    ctx.req = _make_req(capability_name=val)


@given(parsers.parse('an ExecuteCapabilityRequest with task_id set to "{val}"'))
def _req_empty_task_id(val, ctx):
    ctx.req = _make_req(task_id=val)


@given(parsers.parse('an ExecuteCapabilityRequest with input_payload set to "{val}"'))
def _req_bad_payload(val, ctx):
    ctx.req = _make_req(input_payload=val.encode())


@given(parsers.parse('a GetCapabilitySchemaRequest with capability_name set to "{val}"'))
def _schema_req_empty(val, ctx):
    ctx.schema_cap = val


# ── When steps ────────────────────────────────────────────────────────────────

@when("ExecuteCapability is called")
def _execute_cap(ctx):
    stream = ctx.stub.ExecuteCapability(ctx.req)
    ctx.events, ctx.grpc_error = _collect(stream)


@when("ExecuteCapability is called and the stream is fully consumed")
def _execute_cap_full(ctx):
    stream = ctx.stub.ExecuteCapability(ctx.req)
    ctx.events, ctx.grpc_error = _collect(stream)


@when(parsers.parse('GetCapabilitySchema is called with capability_name "{cap}"'))
def _get_schema(cap, ctx):
    ctx.schema_cap = cap
    try:
        ctx.schema_resp = ctx.stub.GetCapabilitySchema(
            agent_pb2.GetCapabilitySchemaRequest(capability_name=cap)
        )
        ctx.grpc_error = None
    except grpc.RpcError as e:
        ctx.schema_resp = None
        ctx.grpc_error = e


@when("GetCapabilitySchema is called")
def _get_schema_with_stored_cap(ctx):
    try:
        ctx.schema_resp = ctx.stub.GetCapabilitySchema(
            agent_pb2.GetCapabilitySchemaRequest(capability_name=ctx.schema_cap)
        )
        ctx.grpc_error = None
    except grpc.RpcError as e:
        ctx.schema_resp = None
        ctx.grpc_error = e


# ── Then steps — stream content ───────────────────────────────────────────────

@then(parsers.parse("the stream emits at least one TaskEvent with event_type {etype}"))
def _at_least_one_progress(etype, ctx):
    code = getattr(agent_pb2, f"TASK_EVENT_TYPE_{etype}")
    assert any(e.event_type == code for e in ctx.events), \
        f"No {etype} event in {[e.event_type for e in ctx.events]}"


@then(parsers.parse("the final TaskEvent has event_type {etype}"))
def _final_event_type(etype, ctx):
    code = getattr(agent_pb2, f"TASK_EVENT_TYPE_{etype}")
    assert ctx.events, "No events received"
    assert ctx.events[-1].event_type == code, \
        f"Expected {etype}, got {ctx.events[-1].event_type}"


@then("the COMPLETED event payload is valid JSON")
def _completed_payload_json(ctx):
    completed = [e for e in ctx.events if e.event_type == agent_pb2.TASK_EVENT_TYPE_COMPLETED]
    assert completed
    json.loads(completed[-1].payload)


@then("the stream closes cleanly after the COMPLETED event")
def _stream_closes_cleanly(ctx):
    assert ctx.grpc_error is None


@then(parsers.parse('every TaskEvent in the stream has task_id "{tid}"'))
def _every_event_has_task_id(tid, ctx):
    assert ctx.events
    for ev in ctx.events:
        assert ev.task_id == tid, f"Event task_id={ev.task_id!r}, expected {tid!r}"


@then("every TaskEvent has a non-zero timestamp")
def _every_event_has_timestamp(ctx):
    assert ctx.events
    for ev in ctx.events:
        assert ev.timestamp.seconds > 0, f"Timestamp zero on event {ev}"


@then(parsers.parse("the stream receives a TaskEvent of type FAILED within {n:d} seconds"))
def _failed_within_seconds(n, ctx):
    failed = [e for e in ctx.events if e.event_type == agent_pb2.TASK_EVENT_TYPE_FAILED]
    assert failed, "No FAILED event received"


@then(parsers.parse('the CapabilityError code is "{code}"'))
def _error_code(code, ctx):
    failed = [e for e in ctx.events if e.event_type == agent_pb2.TASK_EVENT_TYPE_FAILED]
    assert failed
    assert failed[0].error.code == code, f"Got code={failed[0].error.code!r}"


@then(parsers.parse("the stream emits exactly one TaskEvent with event_type {etype}"))
def _exactly_one_event(etype, ctx):
    code = getattr(agent_pb2, f"TASK_EVENT_TYPE_{etype}")
    matching = [e for e in ctx.events if e.event_type == code]
    assert len(matching) == 1, f"Expected exactly 1 {etype}, got {len(matching)}"


@then("the TaskEvent.error.code is a non-empty string")
def _error_code_nonempty(ctx):
    failed = [e for e in ctx.events if e.event_type == agent_pb2.TASK_EVENT_TYPE_FAILED]
    assert failed and failed[0].error.code


@then("the TaskEvent.error.message is a non-empty string")
def _error_message_nonempty(ctx):
    failed = [e for e in ctx.events if e.event_type == agent_pb2.TASK_EVENT_TYPE_FAILED]
    assert failed and failed[0].error.message


@then("no further events are emitted after the FAILED event")
def _no_events_after_failed(ctx):
    for i, ev in enumerate(ctx.events):
        if ev.event_type == agent_pb2.TASK_EVENT_TYPE_FAILED:
            assert i == len(ctx.events) - 1, "Events found after FAILED"
            return


@then("no TaskEvent is received after the first FAILED event")
def _no_event_after_first_failed(ctx):
    for i, ev in enumerate(ctx.events):
        if ev.event_type == agent_pb2.TASK_EVENT_TYPE_FAILED:
            assert i == len(ctx.events) - 1
            return


@then("no TaskEvent is emitted")
def _no_events(ctx):
    assert ctx.events == [], f"Expected no events, got {ctx.events}"


@then("the final TaskEvent has event_type COMPLETED or FAILED")
def _final_is_terminal(ctx):
    assert ctx.events
    terminal = {agent_pb2.TASK_EVENT_TYPE_COMPLETED, agent_pb2.TASK_EVENT_TYPE_FAILED}
    assert ctx.events[-1].event_type in terminal


# ── Then steps — gRPC status ──────────────────────────────────────────────────

@then(parsers.parse("the gRPC status is {status}"))
def _grpc_status(status, ctx):
    code = getattr(grpc.StatusCode, status)
    if status == "OK":
        assert ctx.grpc_error is None
    else:
        assert ctx.grpc_error is not None, f"Expected {status} but got OK"
        assert ctx.grpc_error.code() == code, \
            f"Expected {code}, got {ctx.grpc_error.code()}"


@then(parsers.parse('the error message contains "{text}"'))
def _error_contains(text, ctx):
    assert ctx.grpc_error is not None
    assert text in ctx.grpc_error.details(), \
        f"{text!r} not in {ctx.grpc_error.details()!r}"


@then(parsers.parse('the error message mentions "{field}"'))
def _error_mentions(field, ctx):
    assert ctx.grpc_error is not None
    assert field in ctx.grpc_error.details(), \
        f"{field!r} not in {ctx.grpc_error.details()!r}"


# ── Then steps — GetCapabilitySchema ──────────────────────────────────────────

@then("the gRPC status is OK")
def _status_ok(ctx):
    assert ctx.grpc_error is None


@then(parsers.parse('the response capability_name is "{name}"'))
def _resp_cap_name(name, ctx):
    assert ctx.schema_resp.capability_name == name


@then("the response input_schema_json is valid JSON")
def _resp_input_schema_json(ctx):
    json.loads(ctx.schema_resp.input_schema_json)


@then("the response output_schema_json is valid JSON")
def _resp_output_schema_json(ctx):
    json.loads(ctx.schema_resp.output_schema_json)


@then("the response description is non-empty")
def _resp_description(ctx):
    assert ctx.schema_resp.description
