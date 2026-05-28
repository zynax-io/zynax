# SPDX-License-Identifier: Apache-2.0
"""Ollama provider — streams token chunks via httpx.AsyncClient REST API."""

from __future__ import annotations

import json
from collections.abc import AsyncGenerator
from typing import TYPE_CHECKING

import httpx
import structlog

if TYPE_CHECKING:
    from llm_adapter.config import OllamaProviderConfig

log = structlog.get_logger()

_CHAT_PATH = "/api/chat"


async def stream_tokens(
    config: OllamaProviderConfig,
    prompt: str,
    system: str | None,
    temperature: float,
    max_tokens: int,
) -> AsyncGenerator[bytes, None]:
    """Stream token chunks from the Ollama REST API.

    Args:
        config: Ollama provider configuration (base_url + model).
        prompt: User prompt.
        system: Optional system message.
        temperature: Sampling temperature.
        max_tokens: Maximum token ceiling (mapped to ``num_predict``).

    Yields:
        UTF-8 encoded token chunk bytes.

    Raises:
        RuntimeError: On HTTP or connection error; mapped to UPSTREAM_ERROR by the handler.
    """
    messages = []
    if system:
        messages.append({"role": "system", "content": system})
    messages.append({"role": "user", "content": prompt})

    body = {
        "model": config.model,
        "messages": messages,
        "stream": True,
        "options": {"temperature": temperature, "num_predict": max_tokens},
    }
    url = config.base_url.rstrip("/") + _CHAT_PATH

    timeout = httpx.Timeout(connect=5.0, read=120.0, write=10.0, pool=5.0)
    try:
        async with httpx.AsyncClient(timeout=timeout) as client:
            async with client.stream("POST", url, json=body) as response:
                response.raise_for_status()
                async for line in response.aiter_lines():
                    if not line:
                        continue
                    data = json.loads(line)
                    content = data.get("message", {}).get("content", "")
                    if content:
                        yield content.encode()
                    if data.get("done"):
                        break
    except httpx.HTTPError as exc:
        log.warning("ollama_upstream_error", error=str(exc)[:256])
        raise RuntimeError(str(exc)[:512]) from exc
