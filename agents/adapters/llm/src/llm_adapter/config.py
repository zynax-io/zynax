# SPDX-License-Identifier: Apache-2.0
"""LLM adapter configuration — provider selection and credentials via environment variables.

Provider is selected by ``LLM_PROVIDER``. Each provider loads its own required
env vars at startup and fails fast if any are missing. API key values are stored
as ``pydantic.SecretStr`` and never appear in repr, logs, or error messages.
"""

from __future__ import annotations

from typing import Annotated, Literal

from pydantic import Field, SecretStr, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class OpenAIProviderConfig(BaseSettings):
    """Settings for the OpenAI provider.

    Required env vars:
        OPENAI_API_KEY: API key for the OpenAI platform.
    """

    model_config = SettingsConfigDict(env_prefix="", extra="ignore")

    api_key: SecretStr = Field(alias="OPENAI_API_KEY")
    model: str = Field(default="gpt-4o", alias="LLM_MODEL")


class BedrockProviderConfig(BaseSettings):
    """Settings for the AWS Bedrock provider.

    Required env vars:
        AWS_REGION: AWS region where the Bedrock endpoint is deployed.
    """

    model_config = SettingsConfigDict(env_prefix="", extra="ignore")

    region: str = Field(alias="AWS_REGION")
    model: str = Field(
        default="anthropic.claude-3-5-sonnet-20241022-v2:0",
        alias="LLM_MODEL",
    )


class OllamaProviderConfig(BaseSettings):
    """Settings for the Ollama provider.

    Required env vars:
        OLLAMA_BASE_URL: Base URL for the locally running Ollama server.
    """

    model_config = SettingsConfigDict(env_prefix="", extra="ignore")

    base_url: str = Field(alias="OLLAMA_BASE_URL")
    model: str = Field(default="llama3.2", alias="LLM_MODEL")


_PROVIDER = Literal["openai", "bedrock", "ollama"]


class ProviderConfig(BaseSettings):
    """Top-level LLM provider configuration loaded from environment variables.

    ``LLM_PROVIDER`` selects the active provider. Provider-specific settings are
    validated at instantiation time — a missing required env var raises
    ``ValidationError`` before the adapter process starts.

    Attributes:
        provider: Active LLM provider name. Must be one of "openai", "bedrock", "ollama".
        max_tokens: Maximum token ceiling enforced by the adapter before calling
            the provider. Configurable via ``LLM_MAX_TOKENS``.
        openai: Populated when ``provider == "openai"``.
        bedrock: Populated when ``provider == "bedrock"``.
        ollama: Populated when ``provider == "ollama"``.
    """

    model_config = SettingsConfigDict(env_prefix="", extra="ignore")

    provider: Annotated[_PROVIDER, Field(alias="LLM_PROVIDER")]
    max_tokens: int = Field(default=4096, alias="LLM_MAX_TOKENS", ge=1)

    openai: OpenAIProviderConfig | None = None
    bedrock: BedrockProviderConfig | None = None
    ollama: OllamaProviderConfig | None = None

    @model_validator(mode="after")
    def _load_provider_config(self) -> ProviderConfig:
        """Load and validate the active provider's settings.

        Raises:
            ValueError: When the declared provider cannot be initialised due to
                missing or invalid environment variables.
        """
        if self.provider == "openai":
            self.openai = OpenAIProviderConfig()  # type: ignore[call-arg]
        elif self.provider == "bedrock":
            self.bedrock = BedrockProviderConfig()  # type: ignore[call-arg]
        elif self.provider == "ollama":
            self.ollama = OllamaProviderConfig()  # type: ignore[call-arg]
        return self
