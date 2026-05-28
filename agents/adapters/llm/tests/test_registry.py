# SPDX-License-Identifier: Apache-2.0
"""Unit tests for llm_adapter.registry.client."""

from __future__ import annotations

from unittest.mock import AsyncMock, MagicMock

import grpc
import pytest
from zynax.v1 import agent_registry_pb2  # type: ignore[import-untyped]

from llm_adapter.registry.client import (
    _MAX_ATTEMPTS,
    AdapterSettings,
    _is_transient,
    build_agent_def,
    deregister_agent,
    register_agent,
)

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


class _TransientError(Exception):
    """Fake gRPC error with a retriable status code."""

    def code(self) -> grpc.StatusCode:
        return grpc.StatusCode.UNAVAILABLE


class _PermanentError(Exception):
    """Fake gRPC error with a non-retriable status code."""

    def code(self) -> grpc.StatusCode:
        return grpc.StatusCode.NOT_FOUND


def _settings() -> AdapterSettings:
    return AdapterSettings(
        AGENT_ID="llm-test",
        ADAPTER_ENDPOINT="llm-adapter:50057",
        REGISTRY_ADDR="registry:50052",
    )


def _fake_def() -> agent_registry_pb2.AgentDef:
    return agent_registry_pb2.AgentDef(agent_id="llm-test", endpoint="llm-adapter:50057")


# ---------------------------------------------------------------------------
# _is_transient
# ---------------------------------------------------------------------------


class TestIsTransient:
    """_is_transient classifies gRPC status codes correctly."""

    def test_unavailable_is_transient(self) -> None:
        assert _is_transient(_TransientError()) is True

    def test_not_found_is_not_transient(self) -> None:
        assert _is_transient(_PermanentError()) is False

    def test_plain_exception_is_not_transient(self) -> None:
        assert _is_transient(ValueError("oops")) is False


# ---------------------------------------------------------------------------
# build_agent_def
# ---------------------------------------------------------------------------


class TestBuildAgentDef:
    """build_agent_def populates AgentDef from settings and schemas."""

    def test_capability_names_included(self) -> None:
        settings = _settings()
        schemas = {"chat_completion": (b'{"type":"object"}', b'{"type":"object"}')}
        agent_def = build_agent_def(settings, ["chat_completion"], schemas)
        assert len(agent_def.capabilities) == 1
        assert agent_def.capabilities[0].name == "chat_completion"

    def test_agent_id_and_endpoint_set(self) -> None:
        settings = _settings()
        agent_def = build_agent_def(settings, [], {})
        assert agent_def.agent_id == "llm-test"
        assert agent_def.endpoint == "llm-adapter:50057"

    def test_schema_bytes_copied(self) -> None:
        settings = _settings()
        inp, out = b'{"in":1}', b'{"out":1}'
        agent_def = build_agent_def(settings, ["chat_completion"], {"chat_completion": (inp, out)})
        cap = agent_def.capabilities[0]
        assert cap.input_schema == inp
        assert cap.output_schema == out


# ---------------------------------------------------------------------------
# register_agent
# ---------------------------------------------------------------------------


class TestRegisterAgent:
    """register_agent retries on transient errors and raises on permanent ones."""

    @pytest.mark.asyncio
    async def test_succeeds_on_first_attempt(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setattr("llm_adapter.registry.client.asyncio.sleep", AsyncMock())
        stub = MagicMock()
        stub.RegisterAgent = AsyncMock(return_value=MagicMock())
        await register_agent(_fake_def(), stub)
        stub.RegisterAgent.assert_awaited_once()

    @pytest.mark.asyncio
    async def test_retries_on_transient_then_succeeds(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.setattr("llm_adapter.registry.client.asyncio.sleep", AsyncMock())
        stub = MagicMock()
        stub.RegisterAgent = AsyncMock(
            side_effect=[_TransientError(), _TransientError(), MagicMock()]
        )
        await register_agent(_fake_def(), stub)
        assert stub.RegisterAgent.await_count == 3

    @pytest.mark.asyncio
    async def test_raises_on_permanent_error(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setattr("llm_adapter.registry.client.asyncio.sleep", AsyncMock())
        stub = MagicMock()
        stub.RegisterAgent = AsyncMock(side_effect=_PermanentError())
        with pytest.raises(_PermanentError):
            await register_agent(_fake_def(), stub)
        stub.RegisterAgent.assert_awaited_once()

    @pytest.mark.asyncio
    async def test_raises_after_max_attempts_exhausted(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.setattr("llm_adapter.registry.client.asyncio.sleep", AsyncMock())
        stub = MagicMock()
        stub.RegisterAgent = AsyncMock(side_effect=_TransientError())
        with pytest.raises(RuntimeError):
            await register_agent(_fake_def(), stub)
        assert stub.RegisterAgent.await_count == _MAX_ATTEMPTS


# ---------------------------------------------------------------------------
# deregister_agent
# ---------------------------------------------------------------------------


class TestDeregisterAgent:
    """deregister_agent calls stub once with the correct agent_id."""

    @pytest.mark.asyncio
    async def test_calls_stub_once(self) -> None:
        stub = MagicMock()
        stub.DeregisterAgent = AsyncMock(return_value=MagicMock())
        await deregister_agent("llm-test", stub)
        stub.DeregisterAgent.assert_awaited_once()

    @pytest.mark.asyncio
    async def test_propagates_errors(self) -> None:
        stub = MagicMock()
        stub.DeregisterAgent = AsyncMock(side_effect=RuntimeError("registry down"))
        with pytest.raises(RuntimeError):
            await deregister_agent("llm-test", stub)


# ---------------------------------------------------------------------------
# AdapterSettings
# ---------------------------------------------------------------------------


class TestAdapterSettings:
    """AdapterSettings loads from env vars and validates required fields."""

    def test_valid_settings(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("AGENT_ID", "llm-a1")
        monkeypatch.setenv("ADAPTER_ENDPOINT", "llm-adapter:50057")
        monkeypatch.setenv("REGISTRY_ADDR", "registry:50052")
        s = AdapterSettings()  # type: ignore[call-arg]
        assert s.agent_id == "llm-a1"
        assert s.grpc_port == 50057

    def test_custom_grpc_port(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("AGENT_ID", "llm-a1")
        monkeypatch.setenv("ADAPTER_ENDPOINT", "llm-adapter:50060")
        monkeypatch.setenv("REGISTRY_ADDR", "registry:50052")
        monkeypatch.setenv("ZYNAX_LLM_ADAPTER_GRPC_PORT", "50060")
        s = AdapterSettings()  # type: ignore[call-arg]
        assert s.grpc_port == 50060

    def test_missing_agent_id_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        from pydantic import ValidationError

        monkeypatch.delenv("AGENT_ID", raising=False)
        monkeypatch.setenv("ADAPTER_ENDPOINT", "llm-adapter:50057")
        monkeypatch.setenv("REGISTRY_ADDR", "registry:50052")
        with pytest.raises(ValidationError):
            AdapterSettings()  # type: ignore[call-arg]
