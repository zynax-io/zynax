# SPDX-License-Identifier: Apache-2.0
"""In-process gRPC test server implementations for AgentService and AgentRegistryService."""
import base64
import json
import threading
import time

import grpc
from google.protobuf import timestamp_pb2

import sys, os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../protos/generated/python"))

from zynax.v1 import agent_pb2, agent_pb2_grpc
from zynax.v1 import agent_registry_pb2, agent_registry_pb2_grpc


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
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "capability_name must not be empty")
            return
        if not request.task_id:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "task_id must not be empty")
            return
        if request.input_payload and not _is_valid_json(request.input_payload):
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "input_payload must be valid JSON")
            return

        cap = request.capability_name

        if cap == "summarize":
            if 0 < request.timeout_seconds <= 1:
                yield agent_pb2.TaskEvent(
                    task_id=request.task_id,
                    event_type=agent_pb2.TASK_EVENT_TYPE_FAILED,
                    timestamp=_now(),
                    error=agent_pb2.CapabilityError(code="TIMEOUT", message="capability timed out"),
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
                error=agent_pb2.CapabilityError(code="INTERNAL", message="capability always fails"),
            )

        else:
            context.abort(grpc.StatusCode.NOT_FOUND, f"capability {cap!r} not found")
            return

    def GetCapabilitySchema(self, request, context):
        if not request.capability_name:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "capability_name must not be empty")
            return None
        if request.capability_name == "summarize":
            return agent_pb2.GetCapabilitySchemaResponse(
                capability_name="summarize",
                input_schema_json='{"type":"object","properties":{"documents":{"type":"array"}}}',
                output_schema_json='{"type":"object","properties":{"summary":{"type":"string"}}}',
                description="Summarises a list of documents into a short paragraph.",
            )
        context.abort(grpc.StatusCode.NOT_FOUND, f"capability {request.capability_name!r} not found")
        return None


def _is_valid_json(data: bytes) -> bool:
    try:
        json.loads(data)
        return True
    except (ValueError, TypeError):
        return False


# ── AgentRegistryService ──────────────────────────────────────────────────────

class AgentRegistryImpl(agent_registry_pb2_grpc.AgentRegistryServiceServicer):
    """In-memory AgentRegistry for BDD contract tests. Thread-safe."""

    def __init__(self):
        self._lock = threading.Lock()
        self._agents: dict[str, agent_registry_pb2.AgentDef] = {}

    def clear(self):
        with self._lock:
            self._agents.clear()

    # ── helpers ──────────────────────────────────────────────────────────────

    def _validate_capability_defs(self, caps, context):
        for cap in caps:
            if not cap.name:
                context.abort(grpc.StatusCode.INVALID_ARGUMENT, "capability_name must not be empty")
                return False
            if cap.input_schema and not _is_valid_json(cap.input_schema):
                context.abort(grpc.StatusCode.INVALID_ARGUMENT, "input_schema must be valid JSON")
                return False
            if cap.output_schema and not _is_valid_json(cap.output_schema):
                context.abort(grpc.StatusCode.INVALID_ARGUMENT, "output_schema must be valid JSON")
                return False
        return True

    def _matches_selector(self, agent: agent_registry_pb2.AgentDef, selector: str) -> bool:
        if not selector:
            return True
        for part in selector.split(","):
            part = part.strip()
            if "=" in part:
                k, v = part.split("=", 1)
                if agent.labels.get(k) != v:
                    return False
        return True

    # ── RPCs ──────────────────────────────────────────────────────────────────

    def RegisterAgent(self, request, context):
        agent = request.agent
        if not agent.agent_id:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "agent_id must not be empty")
            return None
        if not agent.endpoint:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "endpoint must not be empty")
            return None
        if not self._validate_capability_defs(agent.capabilities, context):
            return None

        with self._lock:
            existing = self._agents.get(agent.agent_id)
            if existing is not None and existing.status == agent_registry_pb2.AGENT_STATUS_REGISTERED:
                context.abort(grpc.StatusCode.ALREADY_EXISTS, f"agent {agent.agent_id!r} already registered")
                return None
            stored = agent_registry_pb2.AgentDef()
            stored.CopyFrom(agent)
            stored.status = agent_registry_pb2.AGENT_STATUS_REGISTERED
            stored.registered_at.CopyFrom(_now())
            self._agents[agent.agent_id] = stored

        return agent_registry_pb2.RegisterAgentResponse(
            agent_id=agent.agent_id,
            registered_at=_now(),
        )

    def DeregisterAgent(self, request, context):
        if not request.agent_id:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "agent_id must not be empty")
            return None
        with self._lock:
            if request.agent_id not in self._agents:
                context.abort(grpc.StatusCode.NOT_FOUND, f"agent {request.agent_id!r} not found")
                return None
            self._agents[request.agent_id].status = agent_registry_pb2.AGENT_STATUS_DEREGISTERED
        return agent_registry_pb2.DeregisterAgentResponse(deregistered_at=_now())

    def GetAgent(self, request, context):
        if not request.agent_id:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "agent_id must not be empty")
            return None
        with self._lock:
            agent = self._agents.get(request.agent_id)
        if agent is None:
            context.abort(grpc.StatusCode.NOT_FOUND, f"agent {request.agent_id!r} not found")
            return None
        return agent

    def ListAgents(self, request, context):
        with self._lock:
            all_agents = list(self._agents.values())

        # Filter by label selector and registration status
        filtered = [
            a for a in all_agents
            if a.status == agent_registry_pb2.AGENT_STATUS_REGISTERED
            and self._matches_selector(a, request.label_selector)
        ]

        page_size = request.page_size if request.page_size > 0 else len(filtered)
        offset = _decode_token(request.page_token) if request.page_token else 0

        page = filtered[offset: offset + page_size]
        next_offset = offset + page_size
        next_token = _encode_token(next_offset) if next_offset < len(filtered) else ""

        return agent_registry_pb2.ListAgentsResponse(agents=page, next_page_token=next_token)

    def FindByCapability(self, request, context):
        if not request.capability_name:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "capability_name must not be empty")
            return None
        with self._lock:
            matches = [
                a for a in self._agents.values()
                if a.status == agent_registry_pb2.AGENT_STATUS_REGISTERED
                and any(c.name == request.capability_name for c in a.capabilities)
            ]
        return agent_registry_pb2.FindByCapabilityResponse(agents=matches)


def _encode_token(offset: int) -> str:
    return base64.b64encode(str(offset).encode()).decode()


def _decode_token(token: str) -> int:
    try:
        return int(base64.b64decode(token).decode())
    except Exception:
        return 0
