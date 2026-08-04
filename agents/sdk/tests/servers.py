# SPDX-License-Identifier: Apache-2.0
"""In-process gRPC test server implementation for AgentService."""

import json
import time

import grpc
from google.protobuf import timestamp_pb2

import sys, os

sys.path.insert(
    0, os.path.join(os.path.dirname(__file__), "../../../protos/generated/python")
)

from zynax.v1 import agent_pb2, agent_pb2_grpc


def _now() -> timestamp_pb2.Timestamp:
    ts = timestamp_pb2.Timestamp()
    ts.seconds = int(time.time())
    return ts


# ── AgentService ──────────────────────────────────────────────────────────────


class AgentServiceImpl(agent_pb2_grpc.AgentServiceServicer):
    """Contract-compliant in-process AgentService for BDD tests."""

    def ExecuteCapability(self, request, context):
        # Input validation
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

        cap = request.capability_name

        if cap == "summarize":
            if 0 < request.timeout_seconds <= 1:
                yield agent_pb2.TaskEvent(
                    task_id=request.task_id,
                    event_type=agent_pb2.TASK_EVENT_TYPE_FAILED,
                    timestamp=_now(),
                    error=agent_pb2.CapabilityError(
                        code="TIMEOUT", message="capability timed out"
                    ),
                )
                context.abort(grpc.StatusCode.DEADLINE_EXCEEDED, "timeout exceeded")
                return
            yield agent_pb2.TaskEvent(
                task_id=request.task_id,
                event_type=agent_pb2.TASK_EVENT_TYPE_PROGRESS,
                payload=b'{"progress": 50}',
                timestamp=_now(),
            )
            yield agent_pb2.TaskEvent(
                task_id=request.task_id,
                event_type=agent_pb2.TASK_EVENT_TYPE_COMPLETED,
                payload=b'{"summary": "done"}',
                timestamp=_now(),
            )

        elif cap == "always_fails":
            yield agent_pb2.TaskEvent(
                task_id=request.task_id,
                event_type=agent_pb2.TASK_EVENT_TYPE_FAILED,
                timestamp=_now(),
                error=agent_pb2.CapabilityError(
                    code="INTERNAL", message="capability always fails"
                ),
            )

        else:
            context.abort(grpc.StatusCode.NOT_FOUND, f"capability {cap!r} not found")
            return

    def GetCapabilitySchema(self, request, context):
        if not request.capability_name:
            context.abort(
                grpc.StatusCode.INVALID_ARGUMENT, "capability_name must not be empty"
            )
            return None
        if request.capability_name == "summarize":
            return agent_pb2.GetCapabilitySchemaResponse(
                capability_name="summarize",
                input_schema_json='{"type":"object","properties":{"documents":{"type":"array"}}}',
                output_schema_json='{"type":"object","properties":{"summary":{"type":"string"}}}',
                description="Summarises a list of documents into a short paragraph.",
            )
        context.abort(
            grpc.StatusCode.NOT_FOUND,
            f"capability {request.capability_name!r} not found",
        )
        return None


def _is_valid_json(data: bytes) -> bool:
    try:
        json.loads(data)
        return True
    except (ValueError, TypeError):
        return False
