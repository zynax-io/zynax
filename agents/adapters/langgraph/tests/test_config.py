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
    """AdapterConfig loads from LANGGRAPH_MOUNTS."""

    def test_valid_config(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LANGGRAPH_MOUNTS", VALID_MOUNTS_JSON)
        cfg = AdapterConfig()
        assert len(cfg.graph_mounts) == 1
        assert cfg.graph_mounts[0].capability_name == "research_topic"

    def test_multiple_mounts(self, monkeypatch: pytest.MonkeyPatch) -> None:
        mounts = [
            VALID_MOUNT,
            {"capability_name": "summarise", "module": "pkg.summarise", "graph": "wf"},
        ]
        monkeypatch.setenv("LANGGRAPH_MOUNTS", json.dumps(mounts))
        cfg = AdapterConfig()
        assert len(cfg.graph_mounts) == 2
        assert cfg.graph_mounts[1].capability_name == "summarise"

    def test_stale_registry_addr_env_is_ignored(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """A deployment still exporting the retired REGISTRY_ADDR still boots (ADR-039)."""
        monkeypatch.setenv("LANGGRAPH_MOUNTS", VALID_MOUNTS_JSON)
        monkeypatch.setenv("REGISTRY_ADDR", "zynax-agent-registry:50052")
        cfg = AdapterConfig()
        assert len(cfg.graph_mounts) == 1
        assert not hasattr(cfg, "registry_addr")


class TestAdapterConfigMissingEnvVars:
    """Missing required env vars raise ValidationError."""

    def test_missing_mounts_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.delenv("LANGGRAPH_MOUNTS", raising=False)
        with pytest.raises(ValidationError):
            AdapterConfig()


class TestAdapterConfigMalformedMounts:
    """Malformed LANGGRAPH_MOUNTS raises ValidationError."""

    def test_invalid_json_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LANGGRAPH_MOUNTS", "not-json")
        with pytest.raises(_StartupError):
            AdapterConfig()

    def test_json_object_not_array_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LANGGRAPH_MOUNTS", json.dumps({"key": "val"}))
        with pytest.raises(_StartupError):
            AdapterConfig()

    def test_empty_array_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("LANGGRAPH_MOUNTS", "[]")
        with pytest.raises(ValidationError):
            AdapterConfig()

    def test_mount_missing_field_raises(self, monkeypatch: pytest.MonkeyPatch) -> None:
        bad_mount = [{"capability_name": "cap", "module": "m"}]  # missing graph
        monkeypatch.setenv("LANGGRAPH_MOUNTS", json.dumps(bad_mount))
        with pytest.raises(ValidationError):
            AdapterConfig()
