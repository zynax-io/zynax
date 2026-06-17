# SPDX-License-Identifier: Apache-2.0
"""Unit + example tests for the agent handoff context contract (canvas C.4)."""

from __future__ import annotations

import os
import sys
from typing import Any

sys.path.insert(
    0, os.path.join(os.path.dirname(__file__), "../../../protos/generated/python")
)
from zynax.v1 import agent_pb2  # noqa: E402

from zynax_sdk import (  # noqa: E402
    HandoffContext,
    inbound_context,
    outbound_metadata,
)

# A valid W3C traceparent (version-traceid-spanid-flags).
_TRACEPARENT = "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"
_TRACESTATE = "vendor=opaque"


class _MockContext:
    """gRPC ServicerContext mock exposing invocation_metadata()."""

    def __init__(self, metadata: list[tuple[str, Any]] | None = None) -> None:
        self._metadata = metadata or []

    def invocation_metadata(self) -> list[tuple[str, Any]]:
        return self._metadata


def _request(**kwargs: Any) -> agent_pb2.ExecuteCapabilityRequest:
    return agent_pb2.ExecuteCapabilityRequest(
        capability_name=kwargs.pop("capability_name", "summarize"),
        task_id=kwargs.pop("task_id", "task-1"),
        request_id=kwargs.pop("request_id", ""),
        workflow_id=kwargs.pop("workflow_id", ""),
    )


def test_inbound_reads_correlation_and_trace_from_metadata() -> None:
    ctx_obj = _MockContext(
        [
            ("request-id", "req-abc"),
            ("x-namespace", "team-a"),
            ("traceparent", _TRACEPARENT),
            ("tracestate", _TRACESTATE),
        ]
    )
    hc = inbound_context(_request(workflow_id="wf-9", task_id="t-2"), ctx_obj)

    assert hc == HandoffContext(
        request_id="req-abc",
        namespace="team-a",
        workflow_id="wf-9",
        task_id="t-2",
        traceparent=_TRACEPARENT,
        tracestate=_TRACESTATE,
    )
    assert hc.is_traced() is True


def test_inbound_falls_back_to_proto_request_id_when_metadata_absent() -> None:
    # No transport context: request_id/workflow_id come from the proto fields.
    hc = inbound_context(_request(request_id="req-proto", workflow_id="wf-1"))

    assert hc.request_id == "req-proto"
    assert hc.workflow_id == "wf-1"
    assert hc.namespace == ""
    assert hc.is_traced() is False


def test_metadata_request_id_wins_over_proto_field() -> None:
    ctx_obj = _MockContext([("request-id", "req-meta")])
    hc = inbound_context(_request(request_id="req-proto"), ctx_obj)

    assert hc.request_id == "req-meta"


def test_inbound_drops_forbidden_auth_keys() -> None:
    # Safeguard: auth tokens/cookies must never enter a handoff context.
    ctx_obj = _MockContext(
        [
            ("request-id", "req-x"),
            ("authorization", "Bearer super-secret"),
            ("cookie", "session=abc"),
            ("x-api-key", "key-123"),
        ]
    )
    hc = inbound_context(_request(), ctx_obj)

    assert hc.request_id == "req-x"
    for value in vars(hc).values():
        assert "secret" not in value and "session" not in value
        assert "key-123" not in value


def test_outbound_metadata_round_trips_and_is_ordered() -> None:
    hc = HandoffContext(
        request_id="req-abc",
        namespace="team-a",
        traceparent=_TRACEPARENT,
        tracestate=_TRACESTATE,
    )
    md = outbound_metadata(hc)

    assert md == [
        ("request-id", "req-abc"),
        ("x-namespace", "team-a"),
        ("traceparent", _TRACEPARENT),
        ("tracestate", _TRACESTATE),
    ]
    # Re-extracting the emitted metadata reproduces the correlation + trace.
    again = inbound_context(_request(), _MockContext(md))
    assert again.request_id == hc.request_id
    assert again.namespace == hc.namespace
    assert again.traceparent == hc.traceparent


def test_outbound_metadata_omits_unset_fields() -> None:
    assert outbound_metadata(HandoffContext()) == []
    assert outbound_metadata(HandoffContext(request_id="r")) == [("request-id", "r")]


def test_outbound_metadata_is_deterministic() -> None:
    hc = HandoffContext(request_id="r", namespace="n", traceparent=_TRACEPARENT)
    assert outbound_metadata(hc) == outbound_metadata(hc)


def test_bytes_metadata_values_are_decoded() -> None:
    ctx_obj = _MockContext([("request-id", b"req-bytes")])
    hc = inbound_context(_request(), ctx_obj)
    assert hc.request_id == "req-bytes"


def test_inbound_tolerates_context_without_metadata() -> None:
    # A context object that does not expose invocation_metadata() is handled.
    hc = inbound_context(_request(request_id="r"), object())
    assert hc.request_id == "r"


def test_frozen_context_is_immutable() -> None:
    hc = HandoffContext(request_id="r")
    try:
        hc.request_id = "mutated"  # type: ignore[misc]
    except Exception as exc:  # FrozenInstanceError
        assert "request_id" in str(exc) or "cannot assign" in str(exc).lower()
    else:  # pragma: no cover - frozen dataclass must reject assignment
        raise AssertionError("HandoffContext should be immutable")


def test_example_agent_reads_and_forwards_context() -> None:
    """End-to-end: an agent reads its inbound context and forwards it downstream.

    This is the canvas C.4 example — an agent author receives a deterministic
    context and passes it to the next Zynax hop unchanged.
    """
    inbound = _MockContext(
        [
            ("request-id", "req-e2e"),
            ("x-namespace", "prod"),
            ("traceparent", _TRACEPARENT),
        ]
    )
    request = _request(workflow_id="run-42", task_id="task-7")

    # 1. Agent reads the deterministic context it was handed.
    hc = inbound_context(request, inbound)
    assert hc.request_id == "req-e2e"
    assert hc.workflow_id == "run-42"

    # 2. Agent forwards it on its own downstream call — correlation survives.
    forwarded = outbound_metadata(hc)
    assert ("request-id", "req-e2e") in forwarded
    assert ("x-namespace", "prod") in forwarded
    assert ("traceparent", _TRACEPARENT) in forwarded
