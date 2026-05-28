# SPDX-License-Identifier: Apache-2.0
"""Bedrock provider — streams token chunks via aiobotocore converse-stream API."""

from __future__ import annotations

from collections.abc import AsyncGenerator
from typing import TYPE_CHECKING, Any

import aiobotocore.session  # type: ignore[import-untyped]
import structlog

if TYPE_CHECKING:
    from llm_adapter.config import BedrockProviderConfig

log = structlog.get_logger()


def _build_converse_request(
    model_id: str,
    prompt: str,
    system: str | None,
    temperature: float,
    max_tokens: int,
) -> dict[str, Any]:
    """Assemble a Bedrock converse_stream request dict.

    Args:
        model_id: Bedrock model identifier.
        prompt: User prompt text.
        system: Optional system instruction.
        temperature: Sampling temperature.
        max_tokens: Maximum token ceiling.

    Returns:
        Dict suitable for ``client.converse_stream(**req)``.
    """
    req: dict[str, Any] = {
        "modelId": model_id,
        "messages": [{"role": "user", "content": [{"text": prompt}]}],
        "inferenceConfig": {"maxTokens": max_tokens, "temperature": temperature},
    }
    if system:
        req["system"] = [{"text": system}]
    return req


async def stream_tokens(
    config: BedrockProviderConfig,
    prompt: str,
    system: str | None,
    temperature: float,
    max_tokens: int,
) -> AsyncGenerator[bytes, None]:
    """Stream token chunks from AWS Bedrock converse-stream API.

    Args:
        config: Bedrock provider configuration (region + model).
        prompt: User prompt.
        system: Optional system message.
        temperature: Sampling temperature.
        max_tokens: Maximum token ceiling.

    Yields:
        UTF-8 encoded token chunk bytes.

    Raises:
        RuntimeError: On API or client error; mapped to UPSTREAM_ERROR by the handler.
    """
    session = aiobotocore.session.AioSession()
    try:
        async with session.create_client("bedrock-runtime", region_name=config.region) as client:
            req = _build_converse_request(config.model, prompt, system, temperature, max_tokens)
            response = await client.converse_stream(**req)
            async for event in response["stream"]:
                delta = event.get("contentBlockDelta", {}).get("delta", {})
                text = delta.get("text", "")
                if text:
                    yield text.encode()
    except Exception as exc:
        log.warning("bedrock_upstream_error", error=str(exc)[:256])
        raise RuntimeError(str(exc)[:512]) from exc
