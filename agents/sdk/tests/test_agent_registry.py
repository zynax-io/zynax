# SPDX-License-Identifier: Apache-2.0
"""BDD step definitions for agent_registry_service.feature."""
import json

import grpc
import pytest
from pytest_bdd import scenarios, given, when, then, parsers

import sys, os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../protos/generated/python"))
from zynax.v1 import agent_registry_pb2, agent_registry_pb2_grpc

scenarios("features/agent_registry_service.feature")


# ── helpers ───────────────────────────────────────────────────────────────────

def _stub(channel):
    return agent_registry_pb2_grpc.AgentRegistryServiceStub(channel)


def _cap_def(name, input_schema=b"", output_schema=b""):
    return agent_registry_pb2.CapabilityDef(
        name=name, input_schema=input_schema, output_schema=output_schema
    )


def _register(stub, agent_id, caps, endpoint="localhost:50051", labels=None):
    cap_defs = [_cap_def(c) for c in caps]
    agent = agent_registry_pb2.AgentDef(
        agent_id=agent_id,
        endpoint=endpoint,
        capabilities=cap_defs,
        labels=labels or {},
    )
    return stub.RegisterAgent(agent_registry_pb2.RegisterAgentRequest(agent=agent))


# ── Background ────────────────────────────────────────────────────────────────

@given("an AgentRegistryService is running on a test gRPC server")
def _registry_running(agent_registry_channel, ctx):
    ctx.stub = _stub(agent_registry_channel)


@given("the registry is empty")
def _registry_empty(ctx):
    pass  # clear_registry autouse fixture handles this


# ── Registration Given steps ──────────────────────────────────────────────────

@given(parsers.parse('a valid AgentDef with agent_id "{aid}"'))
def _valid_agent_def(aid, ctx):
    ctx.pending_agent_id = aid
    ctx.pending_caps = []
    ctx.pending_endpoint = "localhost:50051"
    ctx.pending_labels = {}


@given(parsers.parse('the AgentDef declares capabilities {caps_json}'))
def _agent_def_caps(caps_json, ctx):
    ctx.pending_caps = json.loads(caps_json)


@given(parsers.parse('the AgentDef endpoint is "{endpoint}"'))
def _agent_def_endpoint(endpoint, ctx):
    ctx.pending_endpoint = endpoint


@given(parsers.parse("the AgentDef has labels {labels_json}"))
def _agent_def_labels(labels_json, ctx):
    labels = json.loads(labels_json)
    ctx.pending_labels = labels
    # If an agent was already registered (e.g. "agent X is registered with capability Y"),
    # deregister and re-register with the labels so GetAgent returns them.
    if hasattr(ctx, "last_registered_id") and hasattr(ctx, "stub"):
        aid = ctx.last_registered_id
        caps = getattr(ctx, "last_registered_caps", ["placeholder"])
        ctx.stub.DeregisterAgent(
            agent_registry_pb2.DeregisterAgentRequest(agent_id=aid)
        )
        _register(ctx.stub, aid, caps, labels=labels)


@given(parsers.parse('agent "{aid}" is registered with capability "{cap}"'))
def _agent_registered_single_cap(aid, cap, ctx):
    _register(ctx.stub, aid, [cap])
    ctx.last_registered_id = aid
    ctx.last_registered_caps = [cap]


@given(parsers.parse("agent \"{aid}\" is registered with capabilities {caps_json}"))
def _agent_registered_multi_cap(aid, caps_json, ctx):
    _register(ctx.stub, aid, json.loads(caps_json))


@given(parsers.parse('agent "{aid}" is registered with labels {labels_json}'))
def _agent_registered_with_labels(aid, labels_json, ctx):
    labels = json.loads(labels_json)
    _register(ctx.stub, aid, ["placeholder"], labels=labels)


@given(parsers.parse('DeregisterAgent has been called for "{aid}"'))
def _deregister_agent(aid, ctx):
    ctx.stub.DeregisterAgent(agent_registry_pb2.DeregisterAgentRequest(agent_id=aid))


@given(parsers.parse("{n:d} agents are registered with capability \"{cap}\""))
def _n_agents_registered(n, cap, ctx):
    ctx.registered_n = n
    for i in range(n):
        _register(ctx.stub, f"agent-bulk-{i}", [cap])


@given(parsers.parse('ListAgents has been called with page_size {ps:d} returning next_page_token "tok-1"'))
def _list_agents_page1(ps, ctx):
    resp = ctx.stub.ListAgents(
        agent_registry_pb2.ListAgentsRequest(page_size=ps)
    )
    ctx.page_token = resp.next_page_token


@given(parsers.re(r'a RegisterAgentRequest with agent_id set to "(?P<val>[^"]*)"'))
def _register_req_empty_agent_id(val, ctx):
    ctx.custom_register_req = agent_registry_pb2.RegisterAgentRequest(
        agent=agent_registry_pb2.AgentDef(
            agent_id=val,
            endpoint="localhost:50051",
        )
    )


@given(parsers.re(r'a RegisterAgentRequest with endpoint set to "(?P<val>[^"]*)"'))
def _register_req_empty_endpoint(val, ctx):
    ctx.custom_register_req = agent_registry_pb2.RegisterAgentRequest(
        agent=agent_registry_pb2.AgentDef(
            agent_id="agent-valid-id",
            endpoint=val,
        )
    )


@given("a RegisterAgentRequest where one CapabilityDef has name set to \"\"")
def _register_req_empty_cap_name(ctx):
    ctx.custom_register_req = agent_registry_pb2.RegisterAgentRequest(
        agent=agent_registry_pb2.AgentDef(
            agent_id="agent-valid-id",
            endpoint="localhost:50051",
            capabilities=[_cap_def("")],
        )
    )


@given(parsers.parse('a RegisterAgentRequest where one CapabilityDef has input_schema "{val}"'))
def _register_req_bad_input_schema(val, ctx):
    ctx.custom_register_req = agent_registry_pb2.RegisterAgentRequest(
        agent=agent_registry_pb2.AgentDef(
            agent_id="agent-valid-id",
            endpoint="localhost:50051",
            capabilities=[_cap_def("summarize", input_schema=val.encode())],
        )
    )


@given(parsers.parse('a RegisterAgentRequest where one CapabilityDef has output_schema "{val}"'))
def _register_req_bad_output_schema(val, ctx):
    ctx.custom_register_req = agent_registry_pb2.RegisterAgentRequest(
        agent=agent_registry_pb2.AgentDef(
            agent_id="agent-valid-id",
            endpoint="localhost:50051",
            capabilities=[_cap_def("summarize", output_schema=val.encode())],
        )
    )


@given(parsers.re(r'a FindByCapabilityRequest with capability_name set to "(?P<val>[^"]*)"'))
def _find_req_empty_cap(val, ctx):
    ctx.find_cap_name = val


@given(parsers.re(r'a GetAgentRequest with agent_id set to "(?P<val>[^"]*)"'))
def _get_agent_req_empty(val, ctx):
    ctx.get_agent_id = val


# ── When steps ────────────────────────────────────────────────────────────────

@when("RegisterAgent is called with the AgentDef")
def _register_agent_with_def(ctx):
    cap_defs = [_cap_def(c) for c in ctx.pending_caps]
    agent = agent_registry_pb2.AgentDef(
        agent_id=ctx.pending_agent_id,
        endpoint=ctx.pending_endpoint,
        capabilities=cap_defs,
        labels=ctx.pending_labels,
    )
    try:
        ctx.register_resp = ctx.stub.RegisterAgent(
            agent_registry_pb2.RegisterAgentRequest(agent=agent)
        )
        ctx.grpc_error = None
    except grpc.RpcError as e:
        ctx.register_resp = None
        ctx.grpc_error = e


@when("RegisterAgent is called")
def _register_agent_custom(ctx):
    try:
        ctx.register_resp = ctx.stub.RegisterAgent(ctx.custom_register_req)
        ctx.grpc_error = None
    except grpc.RpcError as e:
        ctx.register_resp = None
        ctx.grpc_error = e


@when(parsers.parse('RegisterAgent is called again with agent_id "{aid}"'))
def _register_agent_again(aid, ctx):
    try:
        ctx.register_resp = ctx.stub.RegisterAgent(
            agent_registry_pb2.RegisterAgentRequest(
                agent=agent_registry_pb2.AgentDef(
                    agent_id=aid,
                    endpoint="localhost:50051",
                    capabilities=[_cap_def("summarize")],
                )
            )
        )
        ctx.grpc_error = None
    except grpc.RpcError as e:
        ctx.register_resp = None
        ctx.grpc_error = e


@when(parsers.parse('FindByCapability is called with capability_name "{cap}"'))
def _find_by_cap(cap, ctx):
    ctx.find_cap_name = cap
    try:
        ctx.find_resp = ctx.stub.FindByCapability(
            agent_registry_pb2.FindByCapabilityRequest(capability_name=cap)
        )
        ctx.grpc_error = None
    except grpc.RpcError as e:
        ctx.find_resp = None
        ctx.grpc_error = e


@when("FindByCapability is called")
def _find_by_cap_stored(ctx):
    try:
        ctx.find_resp = ctx.stub.FindByCapability(
            agent_registry_pb2.FindByCapabilityRequest(capability_name=ctx.find_cap_name)
        )
        ctx.grpc_error = None
    except grpc.RpcError as e:
        ctx.find_resp = None
        ctx.grpc_error = e


@when("ListAgents is called with no label selector")
def _list_agents_no_selector(ctx):
    try:
        ctx.list_resp = ctx.stub.ListAgents(agent_registry_pb2.ListAgentsRequest())
        ctx.grpc_error = None
    except grpc.RpcError as e:
        ctx.list_resp = None
        ctx.grpc_error = e


@when(parsers.parse('ListAgents is called with label selector "{sel}"'))
def _list_agents_selector(sel, ctx):
    try:
        ctx.list_resp = ctx.stub.ListAgents(
            agent_registry_pb2.ListAgentsRequest(label_selector=sel)
        )
        ctx.grpc_error = None
    except grpc.RpcError as e:
        ctx.list_resp = None
        ctx.grpc_error = e


@when(parsers.parse("ListAgents is called with page_size {ps:d} and no page_token"))
def _list_agents_page_size(ps, ctx):
    try:
        ctx.list_resp = ctx.stub.ListAgents(
            agent_registry_pb2.ListAgentsRequest(page_size=ps)
        )
        ctx.grpc_error = None
    except grpc.RpcError as e:
        ctx.list_resp = None
        ctx.grpc_error = e


@when(parsers.parse('ListAgents is called with page_size {ps:d} and page_token "tok-1"'))
def _list_agents_page_token(ps, ctx):
    try:
        ctx.list_resp = ctx.stub.ListAgents(
            agent_registry_pb2.ListAgentsRequest(
                page_size=ps, page_token=ctx.page_token
            )
        )
        ctx.grpc_error = None
    except grpc.RpcError as e:
        ctx.list_resp = None
        ctx.grpc_error = e


@when(parsers.parse('GetAgent is called with agent_id "{aid}"'))
def _get_agent(aid, ctx):
    ctx.get_agent_id = aid
    try:
        ctx.get_resp = ctx.stub.GetAgent(
            agent_registry_pb2.GetAgentRequest(agent_id=aid)
        )
        ctx.grpc_error = None
    except grpc.RpcError as e:
        ctx.get_resp = None
        ctx.grpc_error = e


@when("GetAgent is called")
def _get_agent_stored(ctx):
    try:
        ctx.get_resp = ctx.stub.GetAgent(
            agent_registry_pb2.GetAgentRequest(agent_id=ctx.get_agent_id)
        )
        ctx.grpc_error = None
    except grpc.RpcError as e:
        ctx.get_resp = None
        ctx.grpc_error = e


@when(parsers.parse('DeregisterAgent is called with agent_id "{aid}"'))
def _deregister(aid, ctx):
    try:
        ctx.deregister_resp = ctx.stub.DeregisterAgent(
            agent_registry_pb2.DeregisterAgentRequest(agent_id=aid)
        )
        ctx.grpc_error = None
    except grpc.RpcError as e:
        ctx.deregister_resp = None
        ctx.grpc_error = e


# ── Then steps ────────────────────────────────────────────────────────────────

@then(parsers.parse('the response contains agent_id "{aid}"'))
def _resp_contains_agent_id(aid, ctx):
    assert ctx.register_resp.agent_id == aid


@then(parsers.parse('GetAgent for "{aid}" returns status {status}'))
def _get_agent_status(aid, status, ctx):
    resp = ctx.stub.GetAgent(agent_registry_pb2.GetAgentRequest(agent_id=aid))
    code = getattr(agent_registry_pb2, f"AGENT_STATUS_{status}")
    assert resp.status == code, f"Expected {status}, got {resp.status}"


@then(parsers.parse('GetAgent for "{aid}" returns both declared capabilities'))
def _get_agent_caps(aid, ctx):
    resp = ctx.stub.GetAgent(agent_registry_pb2.GetAgentRequest(agent_id=aid))
    names = {c.name for c in resp.capabilities}
    for cap in ctx.pending_caps:
        assert cap in names, f"{cap!r} not in {names}"


@then(parsers.parse('GetAgent for "{aid}" returns a non-zero registered_at timestamp'))
def _get_agent_timestamp(aid, ctx):
    resp = ctx.stub.GetAgent(agent_registry_pb2.GetAgentRequest(agent_id=aid))
    assert resp.registered_at.seconds > 0


@then(parsers.parse('the response contains agent "{aid}"'))
def _resp_contains_agent(aid, ctx):
    agents = (
        ctx.find_resp.agents if hasattr(ctx, "find_resp") and ctx.find_resp is not None
        else ctx.list_resp.agents if hasattr(ctx, "list_resp") and ctx.list_resp is not None
        else []
    )
    ids = [a.agent_id for a in agents]
    assert aid in ids, f"{aid!r} not in {ids}"


@then(parsers.parse('the response does not contain agent "{aid}"'))
def _resp_not_contains_agent(aid, ctx):
    agents = (
        ctx.find_resp.agents if hasattr(ctx, "find_resp") and ctx.find_resp is not None
        else ctx.list_resp.agents if hasattr(ctx, "list_resp") and ctx.list_resp is not None
        else []
    )
    ids = [a.agent_id for a in agents]
    assert aid not in ids, f"{aid!r} unexpectedly in {ids}"


@then("the response contains no agents")
def _resp_no_agents(ctx):
    agents = (
        ctx.find_resp.agents if hasattr(ctx, "find_resp") and ctx.find_resp is not None
        else ctx.list_resp.agents if hasattr(ctx, "list_resp") and ctx.list_resp is not None
        else []
    )
    assert len(agents) == 0


@then(parsers.parse('the response agent_id is "{aid}"'))
def _resp_agent_id(aid, ctx):
    assert ctx.get_resp.agent_id == aid


@then(parsers.parse('the response includes capability "{cap}"'))
def _resp_includes_cap(cap, ctx):
    names = {c.name for c in ctx.get_resp.capabilities}
    assert cap in names


@then(parsers.parse('the response includes label "{key}" with value "{val}"'))
def _resp_includes_label(key, val, ctx):
    assert ctx.get_resp.labels.get(key) == val


@then("the response status is REGISTERED")
def _resp_status_registered(ctx):
    assert ctx.get_resp.status == agent_registry_pb2.AGENT_STATUS_REGISTERED


@then(parsers.parse("the gRPC status is {status}"))
def _grpc_status(status, ctx):
    if status == "OK":
        assert ctx.grpc_error is None
    else:
        code = getattr(grpc.StatusCode, status)
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


@then(parsers.parse("the response contains exactly {n:d} agents"))
def _resp_exactly_n(n, ctx):
    agents = ctx.list_resp.agents
    assert len(agents) == n, f"Expected {n}, got {len(agents)}"


@then("the response next_page_token is non-empty")
def _next_token_nonempty(ctx):
    assert ctx.list_resp.next_page_token, "next_page_token is empty"


@then("the response next_page_token is empty")
def _next_token_empty(ctx):
    assert not ctx.list_resp.next_page_token, \
        f"next_page_token={ctx.list_resp.next_page_token!r} is not empty"


@then("the response contains at least 1 agent")
def _resp_at_least_one(ctx):
    assert len(ctx.list_resp.agents) >= 1
