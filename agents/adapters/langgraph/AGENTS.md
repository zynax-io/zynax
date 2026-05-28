# agents/adapters/langgraph — LangGraph Capability Adapter

Python adapter service that mounts LangGraph `StateGraph` instances as named Zynax capabilities. Each node in the graph emits a `PROGRESS` task event; the final graph state is delivered as `COMPLETED`.

## Module

`langgraph-adapter` · `src/langgraph_adapter/`

## Capabilities

Capabilities are declared at runtime via the `LANGGRAPH_MOUNTS` environment variable. Each mount maps a capability name to a Python module that exports a `StateGraph` builder.

| Concept | Detail |
|---------|--------|
| Capability name | Set by `capability_name` in `LANGGRAPH_MOUNTS` |
| Input | JSON object passed as the graph's initial state |
| Output | `PROGRESS` events per node update; `COMPLETED` with final state JSON |
| Timeout | Hard deadline from `ZYNAX_LANGGRAPH_ADAPTER_GRPC_PORT` environment (default: no timeout beyond gRPC deadline) |

## LANGGRAPH_MOUNTS Format

JSON array. Each entry declares one capability:

```json
[
  {
    "capability_name": "echo",
    "module": "examples.echo_graph",
    "graph": "graph"
  }
]
```

| Field | Required | Description |
|-------|----------|-------------|
| `capability_name` | ✓ | Snake-case Zynax capability name (1–64 chars). |
| `module` | ✓ | Dotted Python import path of the module containing the graph (must be on `PYTHONPATH`). |
| `graph` | ✓ | Attribute name of the `StateGraph` object in that module. Must have a `.compile()` method — do not pre-compile. |

`GraphLoader` calls `.compile()` once at startup. The adapter exits non-zero if any mount fails to import or compile.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LANGGRAPH_MOUNTS` | ✓ | — | JSON array of graph mount objects (see above). |
| `AGENT_ID` | ✓ | — | Unique agent identifier registered with agent-registry. |
| `ADAPTER_ENDPOINT` | ✓ | — | gRPC address the task-broker dials, e.g. `langgraph-adapter:50058`. |
| `REGISTRY_ADDR` | ✓ | — | agent-registry gRPC address, e.g. `agent-registry:50052`. |
| `ZYNAX_LANGGRAPH_ADAPTER_GRPC_PORT` | — | `50058` | TCP port the adapter's gRPC server binds to. |

## gRPC Port

Default: **50058** (override via `ZYNAX_LANGGRAPH_ADAPTER_GRPC_PORT`).

## Operator Pattern — Custom Graphs

To mount your own graph:

1. Write a module that exports a `StateGraph` attribute (not compiled):
   ```python
   # my_graphs/research.py
   from langgraph.graph import END, StateGraph
   graph = StateGraph(dict)
   graph.add_node("search", search_node)
   graph.add_node("summarise", summarise_node)
   graph.set_entry_point("search")
   graph.add_edge("search", "summarise")
   graph.add_edge("summarise", END)
   ```

2. Add the module's parent directory to `PYTHONPATH` (via Docker volume or a custom image).

3. Set `LANGGRAPH_MOUNTS`:
   ```json
   [{"capability_name": "research_topic", "module": "my_graphs.research", "graph": "graph"}]
   ```

See `examples/echo_graph.py` for the minimal template.

## Testing

```bash
cd agents/adapters/langgraph
uv run pytest tests/ -v
uv run pytest tests/ --cov=src --cov-fail-under=80 -v
```

## Reference

Canvas: `docs/spdd/384-langgraph-adapter/canvas.md`
