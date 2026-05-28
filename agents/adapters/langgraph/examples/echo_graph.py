# SPDX-License-Identifier: Apache-2.0
"""Echo graph — minimal operator example.

Copies ``input["message"]`` to ``output["reply"]``.

Mount in LANGGRAPH_MOUNTS:
    [{"capability_name": "echo", "module": "examples.echo_graph", "graph": "graph"}]

The ``graph`` attribute is a ``StateGraph`` builder. ``GraphLoader`` calls
``.compile()`` at adapter startup — do not pre-compile it here.
"""

from __future__ import annotations

from langgraph.graph import END, StateGraph  # type: ignore[import-untyped]


def _echo_node(state: dict) -> dict:  # type: ignore[type-arg]
    """Return the message unchanged as the reply."""
    return {"reply": state.get("message", "")}


graph = StateGraph(dict)
graph.add_node("echo", _echo_node)
graph.set_entry_point("echo")
graph.add_edge("echo", END)
