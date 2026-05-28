# SPDX-License-Identifier: Apache-2.0
"""Unit tests for llm-adapter providers — OpenAI, Bedrock, Ollama."""

from __future__ import annotations

import json
from collections.abc import AsyncIterator
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# ---------------------------------------------------------------------------
# Async test helpers
# ---------------------------------------------------------------------------


async def _collect(gen: AsyncIterator[bytes]) -> list[bytes]:
    """Drain an async generator into a list."""
    return [chunk async for chunk in gen]


def _make_openai_chunk(content: str | None) -> MagicMock:
    """Build a minimal openai stream chunk mock."""
    choice = MagicMock()
    choice.delta.content = content
    chunk = MagicMock()
    chunk.choices = [choice]
    return chunk


# ---------------------------------------------------------------------------
# OpenAI provider
# ---------------------------------------------------------------------------


class TestOpenAIProvider:
    """OpenAI streaming provider yields token chunks and handles errors."""

    def _cfg(self) -> MagicMock:
        cfg = MagicMock()
        cfg.api_key.get_secret_value.return_value = "sk-test"
        cfg.model = "gpt-4o"
        return cfg

    @pytest.mark.asyncio
    async def test_yields_utf8_bytes_for_each_delta(self) -> None:
        cfg = self._cfg()
        chunks = [
            _make_openai_chunk("hello"),
            _make_openai_chunk(" world"),
            _make_openai_chunk(None),
        ]

        async def _fake_stream(*_: object, **__: object) -> MagicMock:
            async def _gen() -> object:
                for c in chunks:
                    yield c

            return _gen()

        mock_client = MagicMock()
        mock_client.chat.completions.create = _fake_stream
        mock_client.close = AsyncMock()

        with patch("llm_adapter.providers.openai.openai") as mock_openai:
            mock_openai.AsyncOpenAI.return_value = mock_client
            mock_openai.OpenAIError = Exception

            from llm_adapter.providers.openai import stream_tokens

            result = await _collect(stream_tokens(cfg, "hi", None, 0.7, 100))

        assert result == [b"hello", b" world"]

    @pytest.mark.asyncio
    async def test_upstream_error_raises_runtime_error(self) -> None:
        cfg = self._cfg()
        FakeError = type("OpenAIError", (Exception,), {})

        async def _fail(*_: object, **__: object) -> None:
            raise FakeError("quota exceeded")

        mock_client = MagicMock()
        mock_client.chat.completions.create = _fail
        mock_client.close = AsyncMock()

        with patch("llm_adapter.providers.openai.openai") as mock_openai:
            mock_openai.AsyncOpenAI.return_value = mock_client
            mock_openai.OpenAIError = FakeError

            from llm_adapter.providers.openai import stream_tokens

            with pytest.raises(RuntimeError, match="quota exceeded"):
                await _collect(stream_tokens(cfg, "hi", None, 0.7, 100))

    @pytest.mark.asyncio
    async def test_system_message_prepended_before_user(self) -> None:
        cfg = self._cfg()
        captured: list[object] = []

        async def _capture(**kwargs: object) -> object:
            captured.append(kwargs)

            async def _empty() -> object:
                return
                yield  # make it an async generator

            return _empty()

        mock_client = MagicMock()
        mock_client.chat.completions.create = _capture
        mock_client.close = AsyncMock()

        with patch("llm_adapter.providers.openai.openai") as mock_openai:
            mock_openai.AsyncOpenAI.return_value = mock_client
            mock_openai.OpenAIError = Exception

            from llm_adapter.providers.openai import stream_tokens

            await _collect(stream_tokens(cfg, "prompt text", "be concise", 0.5, 50))

        assert captured
        msgs = captured[0]["messages"]  # type: ignore[index]
        assert msgs[0]["role"] == "system"
        assert msgs[1]["role"] == "user"
        assert msgs[1]["content"] == "prompt text"


# ---------------------------------------------------------------------------
# Bedrock provider
# ---------------------------------------------------------------------------


class TestBedrockProvider:
    """Bedrock converse-stream provider yields token chunks and handles errors."""

    def _cfg(self) -> MagicMock:
        cfg = MagicMock()
        cfg.region = "us-east-1"
        cfg.model = "anthropic.claude-3-5-sonnet-20241022-v2:0"
        return cfg

    @pytest.mark.asyncio
    async def test_yields_utf8_bytes_for_each_text_delta(self) -> None:
        cfg = self._cfg()
        events = [
            {"contentBlockDelta": {"delta": {"text": "hello"}}},
            {"contentBlockDelta": {"delta": {"text": " there"}}},
            {},  # event without text — skipped
        ]

        async def _event_stream() -> object:
            for ev in events:
                yield ev

        mock_client = AsyncMock()
        mock_client.converse_stream = AsyncMock(return_value={"stream": _event_stream()})
        mock_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_client.__aexit__ = AsyncMock(return_value=False)

        mock_session = MagicMock()
        mock_session.create_client.return_value = mock_client

        with patch("llm_adapter.providers.bedrock.aiobotocore") as mock_aio:
            mock_aio.session.AioSession.return_value = mock_session

            from llm_adapter.providers.bedrock import stream_tokens

            result = await _collect(stream_tokens(cfg, "hi", None, 0.7, 100))

        assert result == [b"hello", b" there"]

    @pytest.mark.asyncio
    async def test_upstream_error_raises_runtime_error(self) -> None:
        cfg = self._cfg()
        mock_client = AsyncMock()
        mock_client.converse_stream = AsyncMock(side_effect=RuntimeError("throttled"))
        mock_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_client.__aexit__ = AsyncMock(return_value=False)

        mock_session = MagicMock()
        mock_session.create_client.return_value = mock_client

        with patch("llm_adapter.providers.bedrock.aiobotocore") as mock_aio:
            mock_aio.session.AioSession.return_value = mock_session

            from llm_adapter.providers.bedrock import stream_tokens

            with pytest.raises(RuntimeError, match="throttled"):
                await _collect(stream_tokens(cfg, "hi", None, 0.7, 100))

    @pytest.mark.asyncio
    async def test_system_field_included_when_provided(self) -> None:
        cfg = self._cfg()
        captured: list[object] = []

        async def _capture(**kwargs: object) -> dict[str, object]:
            captured.append(kwargs)
            return {"stream": (x async for x in [])}  # type: ignore[misc]

        mock_client = AsyncMock()
        mock_client.converse_stream = _capture
        mock_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_client.__aexit__ = AsyncMock(return_value=False)

        mock_session = MagicMock()
        mock_session.create_client.return_value = mock_client

        with patch("llm_adapter.providers.bedrock.aiobotocore") as mock_aio:
            mock_aio.session.AioSession.return_value = mock_session

            from llm_adapter.providers.bedrock import stream_tokens

            await _collect(stream_tokens(cfg, "prompt", "be helpful", 0.7, 100))

        assert captured
        assert "system" in captured[0]  # type: ignore[operator]


# ---------------------------------------------------------------------------
# Ollama provider
# ---------------------------------------------------------------------------


class TestOllamaProvider:
    """Ollama REST streaming provider yields token chunks and handles HTTP errors."""

    def _cfg(self) -> MagicMock:
        cfg = MagicMock()
        cfg.base_url = "http://localhost:11434"
        cfg.model = "llama3.2"
        return cfg

    @pytest.mark.asyncio
    async def test_yields_utf8_bytes_and_stops_at_done(self) -> None:
        cfg = self._cfg()
        lines = [
            json.dumps({"message": {"content": "tok1"}, "done": False}),
            json.dumps({"message": {"content": "tok2"}, "done": True}),
            json.dumps({"message": {"content": "tok3"}, "done": False}),  # not reached
        ]

        async def _lines() -> object:
            for line in lines:
                yield line

        mock_response = AsyncMock()
        mock_response.raise_for_status = MagicMock()
        mock_response.aiter_lines = MagicMock(return_value=_lines())
        mock_response.__aenter__ = AsyncMock(return_value=mock_response)
        mock_response.__aexit__ = AsyncMock(return_value=False)

        mock_client = AsyncMock()
        mock_client.stream.return_value = mock_response
        mock_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_client.__aexit__ = AsyncMock(return_value=False)

        with patch("llm_adapter.providers.ollama.httpx") as mock_httpx:
            mock_httpx.AsyncClient.return_value = mock_client
            mock_httpx.Timeout = MagicMock(return_value=None)
            mock_httpx.HTTPError = Exception

            from llm_adapter.providers.ollama import stream_tokens

            result = await _collect(stream_tokens(cfg, "hi", None, 0.7, 100))

        assert result == [b"tok1", b"tok2"]

    @pytest.mark.asyncio
    async def test_upstream_http_error_raises_runtime_error(self) -> None:
        cfg = self._cfg()
        FakeHTTPError = type("HTTPError", (Exception,), {})

        mock_client = AsyncMock()
        mock_client.stream.side_effect = FakeHTTPError("connection refused")
        mock_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_client.__aexit__ = AsyncMock(return_value=False)

        with patch("llm_adapter.providers.ollama.httpx") as mock_httpx:
            mock_httpx.AsyncClient.return_value = mock_client
            mock_httpx.Timeout = MagicMock(return_value=None)
            mock_httpx.HTTPError = FakeHTTPError

            from llm_adapter.providers.ollama import stream_tokens

            with pytest.raises(RuntimeError, match="connection refused"):
                await _collect(stream_tokens(cfg, "hi", None, 0.7, 100))

    @pytest.mark.asyncio
    async def test_empty_content_lines_are_skipped(self) -> None:
        cfg = self._cfg()
        lines = [
            "",  # empty — skipped
            json.dumps({"message": {"content": ""}, "done": False}),  # empty content
            json.dumps({"message": {"content": "tok"}, "done": True}),
        ]

        async def _lines() -> object:
            for line in lines:
                yield line

        mock_response = AsyncMock()
        mock_response.raise_for_status = MagicMock()
        mock_response.aiter_lines = MagicMock(return_value=_lines())
        mock_response.__aenter__ = AsyncMock(return_value=mock_response)
        mock_response.__aexit__ = AsyncMock(return_value=False)

        mock_client = AsyncMock()
        mock_client.stream.return_value = mock_response
        mock_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_client.__aexit__ = AsyncMock(return_value=False)

        with patch("llm_adapter.providers.ollama.httpx") as mock_httpx:
            mock_httpx.AsyncClient.return_value = mock_client
            mock_httpx.Timeout = MagicMock(return_value=None)
            mock_httpx.HTTPError = Exception

            from llm_adapter.providers.ollama import stream_tokens

            result = await _collect(stream_tokens(cfg, "hi", None, 0.7, 100))

        assert result == [b"tok"]
