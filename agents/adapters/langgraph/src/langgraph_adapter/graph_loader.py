# SPDX-License-Identifier: Apache-2.0
"""GraphLoader — imports and compiles LangGraph StateGraph instances at adapter startup."""

from __future__ import annotations

import importlib
from typing import Any

import structlog

from langgraph_adapter.config import GraphMount

log = structlog.get_logger()

# langgraph.graph.state.CompiledStateGraph; declared Any due to ignore_missing_imports.
CompiledGraph = Any


class GraphLoader:
    """Compiles all declared graph mounts once at adapter startup.

    Call ``load_all()`` before serving requests — a partially loaded graph set is
    never acceptable; the method raises immediately on the first failure (ADR-015).
    """

    @staticmethod
    def load_all(mounts: list[GraphMount]) -> dict[str, CompiledGraph]:
        """Import, validate, and compile every graph mount.

        Args:
            mounts: Ordered list of graph-to-capability mappings.

        Returns:
            Mapping from ``capability_name`` to the compiled graph, in mount order.

        Raises:
            RuntimeError: If any module fails to import, attribute is missing, or
                ``.compile()`` raises. The adapter must not start in this case.
        """
        compiled: dict[str, CompiledGraph] = {}
        for mount in mounts:
            compiled[mount.capability_name] = GraphLoader._load_one(mount)
        return compiled

    @staticmethod
    def _load_one(mount: GraphMount) -> CompiledGraph:
        """Import one module and compile its declared StateGraph attribute."""
        try:
            module = importlib.import_module(mount.module)
        except ImportError as exc:
            raise RuntimeError(f"Cannot import graph module '{mount.module}': {exc}") from exc
        graph = getattr(module, mount.graph, None)
        if graph is None:
            raise RuntimeError(f"Module '{mount.module}' has no attribute '{mount.graph}'")
        if not hasattr(graph, "compile"):
            raise RuntimeError(
                f"'{mount.module}.{mount.graph}' is not a StateGraph (no .compile())"
            )
        try:
            compiled = graph.compile()
        except Exception as exc:
            raise RuntimeError(f"Failed to compile '{mount.module}.{mount.graph}': {exc}") from exc
        log.info("graph_loaded", capability=mount.capability_name, module=mount.module)
        return compiled
