# SPDX-License-Identifier: Apache-2.0
"""Unit tests for llm_adapter.config — ProviderConfig validation."""

import pytest
from pydantic import ValidationError

from llm_adapter.config import ProviderConfig


class TestProviderRequired:
    """LLM_PROVIDER is required with no default."""

    def test_missing_provider_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.delenv("LLM_PROVIDER", raising=False)
        with pytest.raises(ValidationError):
            ProviderConfig()


class TestOpenAIProvider:
    """OpenAI provider requires OPENAI_API_KEY; model defaults to gpt-4o."""

    def test_valid_config(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LLM_PROVIDER", "openai")
        monkeypatch.setenv("OPENAI_API_KEY", "sk-test-key")
        cfg = ProviderConfig()
        assert cfg.provider == "openai"
        assert cfg.openai is not None
        assert cfg.openai.model == "gpt-4o"

    def test_model_override(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LLM_PROVIDER", "openai")
        monkeypatch.setenv("OPENAI_API_KEY", "sk-test-key")
        monkeypatch.setenv("LLM_MODEL", "gpt-4o-mini")
        cfg = ProviderConfig()
        assert cfg.openai is not None
        assert cfg.openai.model == "gpt-4o-mini"

    def test_missing_api_key_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LLM_PROVIDER", "openai")
        monkeypatch.delenv("OPENAI_API_KEY", raising=False)
        with pytest.raises(ValidationError):
            ProviderConfig()

    def test_api_key_not_in_repr(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LLM_PROVIDER", "openai")
        monkeypatch.setenv("OPENAI_API_KEY", "sk-super-secret")
        cfg = ProviderConfig()
        assert cfg.openai is not None
        assert "sk-super-secret" not in repr(cfg.openai)
        assert "sk-super-secret" not in repr(cfg)


class TestBedrockProvider:
    """Bedrock provider requires AWS_REGION; model defaults to claude-3-5-sonnet."""

    def test_valid_config(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LLM_PROVIDER", "bedrock")
        monkeypatch.setenv("AWS_REGION", "us-east-1")
        cfg = ProviderConfig()
        assert cfg.provider == "bedrock"
        assert cfg.bedrock is not None
        assert cfg.bedrock.region == "us-east-1"
        assert cfg.bedrock.model == "anthropic.claude-3-5-sonnet-20241022-v2:0"

    def test_missing_region_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LLM_PROVIDER", "bedrock")
        monkeypatch.delenv("AWS_REGION", raising=False)
        with pytest.raises(ValidationError):
            ProviderConfig()


class TestOllamaProvider:
    """Ollama provider requires OLLAMA_BASE_URL; model defaults to llama3.2."""

    def test_valid_config(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LLM_PROVIDER", "ollama")
        monkeypatch.setenv("OLLAMA_BASE_URL", "http://localhost:11434")
        cfg = ProviderConfig()
        assert cfg.provider == "ollama"
        assert cfg.ollama is not None
        assert cfg.ollama.base_url == "http://localhost:11434"
        assert cfg.ollama.model == "llama3.2"

    def test_missing_base_url_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LLM_PROVIDER", "ollama")
        monkeypatch.delenv("OLLAMA_BASE_URL", raising=False)
        with pytest.raises(ValidationError):
            ProviderConfig()


class TestMaxTokens:
    """LLM_MAX_TOKENS defaults to 4096 and is configurable."""

    def test_default_max_tokens(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LLM_PROVIDER", "ollama")
        monkeypatch.setenv("OLLAMA_BASE_URL", "http://localhost:11434")
        cfg = ProviderConfig()
        assert cfg.max_tokens == 4096

    def test_custom_max_tokens(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LLM_PROVIDER", "ollama")
        monkeypatch.setenv("OLLAMA_BASE_URL", "http://localhost:11434")
        monkeypatch.setenv("LLM_MAX_TOKENS", "2048")
        cfg = ProviderConfig()
        assert cfg.max_tokens == 2048


class TestUnknownProvider:
    """Unknown provider name raises ValidationError."""

    def test_invalid_provider_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LLM_PROVIDER", "gemini")
        monkeypatch.delenv("OPENAI_API_KEY", raising=False)
        with pytest.raises(ValidationError):
            ProviderConfig()
