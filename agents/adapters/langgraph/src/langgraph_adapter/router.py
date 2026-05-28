# SPDX-License-Identifier: Apache-2.0
"""CapabilityRouter — dispatches ExecuteCapability requests to compiled LangGraph graphs."""

from __future__ import annotations

import json
from typing import Any

_GENERIC_INPUT_SCHEMA = json.dumps(
    {
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "type": "object",
        "description": "Graph input state (any JSON object).",
    }
).encode()

_GENERIC_OUTPUT_SCHEMA = json.dumps(
    {
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "type": "object",
        "description": "Final graph state after all nodes complete.",
    }
).encode()

# langgraph.graph.state.CompiledStateGraph; declared Any due to ignore_missing_imports.
CompiledGraph = Any


class CapabilityRouter:
    """Immutable map of ``capability_name → compiled_graph`` built at adapter startup."""

    def __init__(self, graphs: dict[str, CompiledGraph]) -> None:
        """Initialise the router from a pre-compiled graph map.

        Args:
            graphs: Mapping from capability name to compiled graph, as returned by
                ``GraphLoader.load_all()``. Treated as immutable after construction.
        """
        self._graphs: dict[str, CompiledGraph] = dict(graphs)

    def dispatch(self, capability_name: str) -> CompiledGraph:
        """Return the compiled graph for ``capability_name``.

        Args:
            capability_name: Snake-case capability identifier.

        Returns:
            The compiled graph registered under ``capability_name``.

        Raises:
            KeyError: When ``capability_name`` is not registered.
        """
        return self._graphs[capability_name]

    def get_schema(self, capability_name: str) -> tuple[bytes, bytes]:
        """Return ``(input_schema_bytes, output_schema_bytes)`` for a capability.

        Args:
            capability_name: Snake-case capability identifier.

        Returns:
            Tuple of JSON Schema bytes ``(input, output)``.

        Raises:
            KeyError: When ``capability_name`` is not registered.
        """
        if capability_name not in self._graphs:
            raise KeyError(capability_name)
        return _GENERIC_INPUT_SCHEMA, _GENERIC_OUTPUT_SCHEMA

    def capability_names(self) -> list[str]:
        """Return the list of all registered capability names."""
        return list(self._graphs.keys())
