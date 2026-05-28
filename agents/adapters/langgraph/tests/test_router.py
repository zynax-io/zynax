# SPDX-License-Identifier: Apache-2.0
"""Unit tests for CapabilityRouter — dispatch, schema retrieval, and KeyError paths."""

from __future__ import annotations

import json
from unittest.mock import MagicMock

import pytest

from langgraph_adapter.router import CapabilityRouter


def _make_graphs(*names: str) -> dict:
    return {name: MagicMock(name=f"CompiledGraph<{name}>") for name in names}


class TestCapabilityRouterDispatch:
    """dispatch() returns the registered graph or raises KeyError."""

    def test_dispatch_known_capability_returns_graph(self) -> None:
        graphs = _make_graphs("research_topic")
        router = CapabilityRouter(graphs)
        result = router.dispatch("research_topic")
        assert result is graphs["research_topic"]

    def test_dispatch_unknown_raises_key_error(self) -> None:
        router = CapabilityRouter(_make_graphs("cap"))
        with pytest.raises(KeyError):
            router.dispatch("unknown")

    def test_dispatch_empty_name_raises_key_error(self) -> None:
        router = CapabilityRouter(_make_graphs("cap"))
        with pytest.raises(KeyError):
            router.dispatch("")


class TestCapabilityRouterSchema:
    """get_schema() returns bytes tuple or raises KeyError for unknown capability."""

    def test_get_schema_returns_bytes_tuple(self) -> None:
        router = CapabilityRouter(_make_graphs("cap"))
        inp, out = router.get_schema("cap")
        assert isinstance(inp, bytes)
        assert isinstance(out, bytes)

    def test_input_schema_is_valid_json(self) -> None:
        router = CapabilityRouter(_make_graphs("cap"))
        inp, _ = router.get_schema("cap")
        schema = json.loads(inp)
        assert schema.get("type") == "object"

    def test_output_schema_is_valid_json(self) -> None:
        router = CapabilityRouter(_make_graphs("cap"))
        _, out = router.get_schema("cap")
        schema = json.loads(out)
        assert schema.get("type") == "object"

    def test_get_schema_unknown_raises_key_error(self) -> None:
        router = CapabilityRouter(_make_graphs("cap"))
        with pytest.raises(KeyError):
            router.get_schema("no_such_capability")


class TestCapabilityRouterNames:
    """capability_names() returns all registered names."""

    def test_all_names_listed(self) -> None:
        router = CapabilityRouter(_make_graphs("a", "b", "c"))
        assert set(router.capability_names()) == {"a", "b", "c"}

    def test_empty_graphs_returns_empty_list(self) -> None:
        router = CapabilityRouter({})
        assert router.capability_names() == []
