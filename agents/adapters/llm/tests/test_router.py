# SPDX-License-Identifier: Apache-2.0
"""Unit tests for CapabilityRouter — dispatch, schema retrieval, and KeyError paths."""

from __future__ import annotations

import json
from unittest.mock import MagicMock

import pytest

from llm_adapter.handler import ChatCompletionHandler
from llm_adapter.router import CapabilityRouter


def _make_config(provider: str = "ollama") -> MagicMock:
    """Build a minimal ProviderConfig mock."""
    cfg = MagicMock()
    cfg.provider = provider
    cfg.max_tokens = 512

    if provider == "openai":
        cfg.openai = MagicMock()
        cfg.openai.api_key.get_secret_value.return_value = "sk-test"
        cfg.openai.model = "gpt-4o"
        cfg.bedrock = None
        cfg.ollama = None
    elif provider == "bedrock":
        cfg.bedrock = MagicMock()
        cfg.bedrock.region = "us-east-1"
        cfg.bedrock.model = "anthropic.claude-3-5-sonnet"
        cfg.openai = None
        cfg.ollama = None
    else:
        cfg.ollama = MagicMock()
        cfg.ollama.base_url = "http://localhost:11434"
        cfg.ollama.model = "llama3.2"
        cfg.openai = None
        cfg.bedrock = None

    return cfg


class TestCapabilityRouterDispatch:
    """dispatch() returns the registered handler or raises KeyError."""

    def test_dispatch_chat_completion_returns_handler(self) -> None:
        router = CapabilityRouter(_make_config())
        handler = router.dispatch("chat_completion")
        assert isinstance(handler, ChatCompletionHandler)

    def test_dispatch_unknown_raises_key_error(self) -> None:
        router = CapabilityRouter(_make_config())
        with pytest.raises(KeyError):
            router.dispatch("unknown_capability")

    def test_dispatch_empty_name_raises_key_error(self) -> None:
        router = CapabilityRouter(_make_config())
        with pytest.raises(KeyError):
            router.dispatch("")


class TestCapabilityRouterSchema:
    """get_schema() returns valid JSON Schema bytes or raises KeyError."""

    def test_get_schema_returns_bytes_tuple(self) -> None:
        router = CapabilityRouter(_make_config())
        inp, out = router.get_schema("chat_completion")
        assert isinstance(inp, bytes)
        assert isinstance(out, bytes)

    def test_input_schema_is_valid_json_with_prompt(self) -> None:
        router = CapabilityRouter(_make_config())
        inp, _ = router.get_schema("chat_completion")
        schema = json.loads(inp)
        assert "prompt" in schema["properties"]
        assert "prompt" in schema["required"]

    def test_output_schema_is_valid_json_with_response(self) -> None:
        router = CapabilityRouter(_make_config())
        _, out = router.get_schema("chat_completion")
        schema = json.loads(out)
        assert "response" in schema["properties"]

    def test_get_schema_unknown_raises_key_error(self) -> None:
        router = CapabilityRouter(_make_config())
        with pytest.raises(KeyError):
            router.get_schema("no_such_capability")


class TestCapabilityRouterNames:
    """capability_names() lists all registered capabilities."""

    def test_chat_completion_in_names(self) -> None:
        router = CapabilityRouter(_make_config())
        assert "chat_completion" in router.capability_names()
