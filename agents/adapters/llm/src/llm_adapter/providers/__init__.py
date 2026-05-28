# SPDX-License-Identifier: Apache-2.0
"""Provider factory — returns a streaming callable for the active LLM provider."""

from __future__ import annotations

from collections.abc import AsyncGenerator, Callable

from llm_adapter.config import (
    BedrockProviderConfig,
    OllamaProviderConfig,
    OpenAIProviderConfig,
    ProviderConfig,
)

ProviderFunc = Callable[
    [str, str | None, float, int],
    AsyncGenerator[bytes, None],
]


def _openai_func(cfg: OpenAIProviderConfig) -> ProviderFunc:
    """Bind OpenAI config into a provider callable."""
    from llm_adapter.providers.openai import stream_tokens  # noqa: PLC0415

    def _call(
        prompt: str,
        system: str | None,
        temperature: float,
        max_tokens: int,
    ) -> AsyncGenerator[bytes, None]:
        return stream_tokens(cfg, prompt, system, temperature, max_tokens)

    return _call


def _bedrock_func(cfg: BedrockProviderConfig) -> ProviderFunc:
    """Bind Bedrock config into a provider callable."""
    from llm_adapter.providers.bedrock import stream_tokens  # noqa: PLC0415

    def _call(
        prompt: str,
        system: str | None,
        temperature: float,
        max_tokens: int,
    ) -> AsyncGenerator[bytes, None]:
        return stream_tokens(cfg, prompt, system, temperature, max_tokens)

    return _call


def _ollama_func(cfg: OllamaProviderConfig) -> ProviderFunc:
    """Bind Ollama config into a provider callable."""
    from llm_adapter.providers.ollama import stream_tokens  # noqa: PLC0415

    def _call(
        prompt: str,
        system: str | None,
        temperature: float,
        max_tokens: int,
    ) -> AsyncGenerator[bytes, None]:
        return stream_tokens(cfg, prompt, system, temperature, max_tokens)

    return _call


def get_provider(config: ProviderConfig) -> ProviderFunc:
    """Return a streaming callable for the configured LLM provider.

    Resolves the active provider from ``config.provider`` and binds the
    provider-specific configuration at construction time. The returned callable
    accepts ``(prompt, system, temperature, max_tokens)`` and yields UTF-8
    encoded token chunks as bytes.

    Args:
        config: Validated provider configuration.

    Returns:
        A callable that, when invoked, returns an async generator of bytes.
    """
    if config.provider == "openai":
        if config.openai is None:
            raise ValueError("openai config missing for provider=openai")
        return _openai_func(config.openai)
    if config.provider == "bedrock":
        if config.bedrock is None:
            raise ValueError("bedrock config missing for provider=bedrock")
        return _bedrock_func(config.bedrock)
    if config.ollama is None:
        raise ValueError("ollama config missing for provider=ollama")
    return _ollama_func(config.ollama)
