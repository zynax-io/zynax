# SPDX-License-Identifier: Apache-2.0
"""Unit tests for langgraph_adapter.config — AdapterConfig and GraphMount validation."""

import json

import pytest
from pydantic import ValidationError
from pydantic_settings.exceptions import SettingsError

from langgraph_adapter.config import AdapterConfig, GraphMount

# pydantic-settings raises SettingsError (not ValidationError) when a complex
# env-var field cannot be JSON-decoded.  Both are startup failures we want to
# catch; tests that exercise the JSON-parse path use this alias.
_StartupError = (ValidationError, SettingsError)

VALID_MOUNT = {"capability_name": "research_topic", "module": "my_pkg.graph", "graph": "graph"}
VALID_MOUNTS_JSON = json.dumps([VALID_MOUNT])


class TestGraphMount:
    """GraphMount requires capability_name, module, and graph."""

    def test_valid_mount(self) -> None:
        m = GraphMount(**VALID_MOUNT)
        assert m.capability_name == "research_topic"
        assert m.module == "my_pkg.graph"
        assert m.graph == "graph"

    def test_missing_capability_name_raises(self) -> None:
        with pytest.raises(ValidationError):
            GraphMount(module="m", graph="g")  # type: ignore[call-arg]

    def test_missing_module_raises(self) -> None:
        with pytest.raises(ValidationError):
            GraphMount(capability_name="cap", graph="g")  # type: ignore[call-arg]

    def test_missing_graph_raises(self) -> None:
        with pytest.raises(ValidationError):
            GraphMount(capability_name="cap", module="m")  # type: ignore[call-arg]


class TestAdapterConfigValid:
    """AdapterConfig loads from LANGGRAPH_MOUNTS and REGISTRY_ADDR."""

    def test_valid_config(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LANGGRAPH_MOUNTS", VALID_MOUNTS_JSON)
        monkeypatch.setenv("REGISTRY_ADDR", "localhost:50052")
        cfg = AdapterConfig()
        assert len(cfg.graph_mounts) == 1
        assert cfg.graph_mounts[0].capability_name == "research_topic"
        assert cfg.registry_addr == "localhost:50052"

    def test_multiple_mounts(self, monkeypatch: pytest.MonkeyPatch) -> None:
        mounts = [
            VALID_MOUNT,
            {"capability_name": "summarise", "module": "pkg.summarise", "graph": "wf"},
        ]
        monkeypatch.setenv("LANGGRAPH_MOUNTS", json.dumps(mounts))
        monkeypatch.setenv("REGISTRY_ADDR", "registry:50052")
        cfg = AdapterConfig()
        assert len(cfg.graph_mounts) == 2
        assert cfg.graph_mounts[1].capability_name == "summarise"


class TestAdapterConfigMissingEnvVars:
    """Missing required env vars raise ValidationError."""

    def test_missing_mounts_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.delenv("LANGGRAPH_MOUNTS", raising=False)
        monkeypatch.setenv("REGISTRY_ADDR", "localhost:50052")
        with pytest.raises(ValidationError):
            AdapterConfig()

    def test_missing_registry_addr_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LANGGRAPH_MOUNTS", VALID_MOUNTS_JSON)
        monkeypatch.delenv("REGISTRY_ADDR", raising=False)
        with pytest.raises(ValidationError):
            AdapterConfig()


class TestAdapterConfigMalformedMounts:
    """Malformed LANGGRAPH_MOUNTS raises ValidationError."""

    def test_invalid_json_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LANGGRAPH_MOUNTS", "not-json")
        monkeypatch.setenv("REGISTRY_ADDR", "localhost:50052")
        with pytest.raises(_StartupError):
            AdapterConfig()

    def test_json_object_not_array_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LANGGRAPH_MOUNTS", json.dumps({"key": "val"}))
        monkeypatch.setenv("REGISTRY_ADDR", "localhost:50052")
        with pytest.raises(_StartupError):
            AdapterConfig()

    def test_empty_array_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LANGGRAPH_MOUNTS", "[]")
        monkeypatch.setenv("REGISTRY_ADDR", "localhost:50052")
        with pytest.raises(ValidationError):
            AdapterConfig()

    def test_mount_missing_field_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        bad_mount = [{"capability_name": "cap", "module": "m"}]  # missing graph
        monkeypatch.setenv("LANGGRAPH_MOUNTS", json.dumps(bad_mount))
        monkeypatch.setenv("REGISTRY_ADDR", "localhost:50052")
        with pytest.raises(ValidationError):
            AdapterConfig()
