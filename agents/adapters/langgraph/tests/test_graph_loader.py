# SPDX-License-Identifier: Apache-2.0
"""Unit tests for GraphLoader — import, validation, and compile-time failures."""

from __future__ import annotations

import types
from unittest.mock import MagicMock, patch

import pytest

from langgraph_adapter.config import GraphMount
from langgraph_adapter.graph_loader import GraphLoader


def _mount(
    capability: str = "research", module: str = "my_pkg.graph", graph: str = "graph"
) -> GraphMount:
    return GraphMount(capability_name=capability, module=module, graph=graph)


def _fake_module(
    graph_attr: str = "graph", has_compile: bool = True, compile_raises: bool = False
) -> types.ModuleType:
    """Build a fake Python module with a mock StateGraph attribute."""
    mod = types.ModuleType("my_pkg.graph")
    mock_graph = MagicMock()
    if not has_compile:
        del mock_graph.compile
    elif compile_raises:
        mock_graph.compile.side_effect = RuntimeError("compile failed")
    else:
        mock_graph.compile.return_value = MagicMock(name="CompiledGraph")
    setattr(mod, graph_attr, mock_graph)
    return mod


class TestGraphLoaderSuccess:
    """load_all() returns a compiled graph keyed by capability_name."""

    def test_load_single_mount_returns_compiled_graph(self) -> None:
        mod = _fake_module()
        with patch("importlib.import_module", return_value=mod):
            result = GraphLoader.load_all([_mount()])
        assert "research" in result
        assert result["research"] is mod.graph.compile.return_value

    def test_load_multiple_mounts(self) -> None:
        mod1 = _fake_module()
        mod2 = _fake_module()
        mounts = [
            _mount("cap_a", "pkg.a", "graph"),
            _mount("cap_b", "pkg.b", "graph"),
        ]
        modules = {"pkg.a": mod1, "pkg.b": mod2}
        with patch("importlib.import_module", side_effect=lambda m: modules[m]):
            result = GraphLoader.load_all(mounts)
        assert set(result.keys()) == {"cap_a", "cap_b"}

    def test_compile_is_called_once(self) -> None:
        mod = _fake_module()
        with patch("importlib.import_module", return_value=mod):
            GraphLoader.load_all([_mount()])
        mod.graph.compile.assert_called_once_with()


class TestGraphLoaderImportFailure:
    """Raises RuntimeError immediately when a module cannot be imported."""

    def test_import_error_raises_runtime_error(self) -> None:
        with patch("importlib.import_module", side_effect=ImportError("no module")):
            with pytest.raises(RuntimeError, match="Cannot import graph module"):
                GraphLoader.load_all([_mount()])

    def test_error_message_includes_module_name(self) -> None:
        with patch("importlib.import_module", side_effect=ImportError("not found")):
            with pytest.raises(RuntimeError, match="my_pkg.graph"):
                GraphLoader.load_all([_mount(module="my_pkg.graph")])

    def test_fails_on_first_bad_mount(self) -> None:
        good_mod = _fake_module()
        call_count = 0

        def _side_effect(module: str) -> types.ModuleType:
            nonlocal call_count
            call_count += 1
            if module == "bad.module":
                raise ImportError("missing")
            return good_mod

        mounts = [_mount("a", "good.module", "graph"), _mount("b", "bad.module", "graph")]
        with patch("importlib.import_module", side_effect=_side_effect):
            with pytest.raises(RuntimeError):
                GraphLoader.load_all(mounts)
        assert call_count == 2


class TestGraphLoaderAttributeFailure:
    """Raises RuntimeError when the graph attribute is missing or not a StateGraph."""

    def test_missing_attribute_raises_runtime_error(self) -> None:
        mod = types.ModuleType("pkg")
        with patch("importlib.import_module", return_value=mod):
            with pytest.raises(RuntimeError, match="has no attribute"):
                GraphLoader.load_all([_mount(graph="nonexistent")])

    def test_attribute_without_compile_raises_runtime_error(self) -> None:
        mod = _fake_module(has_compile=False)
        with patch("importlib.import_module", return_value=mod):
            with pytest.raises(RuntimeError, match="not a StateGraph"):
                GraphLoader.load_all([_mount()])


class TestGraphLoaderCompileFailure:
    """Raises RuntimeError when .compile() raises."""

    def test_compile_error_raises_runtime_error(self) -> None:
        mod = _fake_module(compile_raises=True)
        with patch("importlib.import_module", return_value=mod):
            with pytest.raises(RuntimeError, match="Failed to compile"):
                GraphLoader.load_all([_mount()])
