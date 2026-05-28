# SPDX-License-Identifier: Apache-2.0
"""OpenAI provider — streams token chunks via openai.AsyncOpenAI."""

from __future__ import annotations

from collections.abc import AsyncGenerator
from typing import TYPE_CHECKING

import openai
import structlog

if TYPE_CHECKING:
    from llm_adapter.config import OpenAIProviderConfig

log = structlog.get_logger()


def _build_messages(prompt: str, system: str | None) -> list[dict[str, str]]:
    """Assemble the messages list for the chat completions API.

    Args:
        prompt: User prompt text.
        system: Optional system instruction.

    Returns:
        Ordered list of role/content dicts.
    """
    msgs: list[dict[str, str]] = []
    if system:
        msgs.append({"role": "system", "content": system})
    msgs.append({"role": "user", "content": prompt})
    return msgs


async def stream_tokens(
    config: OpenAIProviderConfig,
    prompt: str,
    system: str | None,
    temperature: float,
    max_tokens: int,
) -> AsyncGenerator[bytes, None]:
    """Stream token chunks from the OpenAI chat completions API.

    Args:
        config: OpenAI provider configuration (API key + model).
        prompt: User prompt.
        system: Optional system message.
        temperature: Sampling temperature (0–2).
        max_tokens: Maximum token ceiling.

    Yields:
        UTF-8 encoded token chunk bytes.

    Raises:
        RuntimeError: When the API returns an error; mapped to UPSTREAM_ERROR by the handler.
    """
    client = openai.AsyncOpenAI(api_key=config.api_key.get_secret_value())
    try:
        stream = await client.chat.completions.create(
            model=config.model,
            messages=_build_messages(prompt, system),  # type: ignore[arg-type]
            temperature=temperature,
            max_tokens=max_tokens,
            stream=True,
        )
        async for chunk in stream:
            if chunk.choices and chunk.choices[0].delta.content:
                yield chunk.choices[0].delta.content.encode()
    except openai.OpenAIError as exc:
        log.warning("openai_upstream_error", error=str(exc)[:256])
        raise RuntimeError(str(exc)[:512]) from exc
    finally:
        await client.close()
