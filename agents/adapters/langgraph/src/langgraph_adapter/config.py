# SPDX-License-Identifier: Apache-2.0
"""LangGraph adapter configuration — graph mounts and serving port via environment variables.

``LANGGRAPH_MOUNTS`` is a JSON-encoded list of ``GraphMount`` objects that map capability
names to Python module import paths and graph attribute names. It is required — the adapter
process fails fast if it is absent or malformed.

``REGISTRY_ADDR`` is no longer read (ADR-039, M9.A): agent identity is declared by the
``zynax.io/v1alpha1`` ``Agent`` custom resource, not pushed to the agent-registry on boot.
Deployments that still set it are unaffected — unknown environment variables are ignored.
"""

from __future__ import annotations

import json
from typing import Any

from pydantic import Field, field_validator, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class GraphMount(BaseSettings):
    """Mapping from a Zynax capability name to a LangGraph StateGraph definition.

    Each entry declares which Python module contains the graph and which attribute
    on that module holds the ``StateGraph`` instance. The graph is compiled once at
    adapter startup by ``GraphLoader`` — not per-request.

    Attributes:
        capability_name: Snake-case Zynax capability name (e.g. ``"research_topic"``).
        module: Dotted Python import path of the module containing the graph
            (e.g. ``"my_package.my_graph"``).
        graph: Attribute name of the ``StateGraph`` object within ``module``
            (e.g. ``"graph"``).
    """

    model_config = SettingsConfigDict(extra="ignore")

    capability_name: str = Field(..., min_length=1, max_length=64)
    module: str = Field(..., min_length=1)
    graph: str = Field(..., min_length=1)


class AdapterConfig(BaseSettings):
    """Top-level LangGraph adapter configuration loaded from environment variables.

    ``LANGGRAPH_MOUNTS`` is a JSON array of graph-mount objects. Each object must
    supply ``capability_name``, ``module``, and ``graph``.

    Raises ``ValidationError`` on startup if ``LANGGRAPH_MOUNTS`` is missing, empty,
    or contains malformed JSON. A missing required field inside a mount object also
    raises ``ValidationError``.

    Attributes:
        graph_mounts: Ordered list of graph-to-capability mappings loaded from
            ``LANGGRAPH_MOUNTS``.
        grpc_port: Port the adapter's own gRPC server binds to
            (``ZYNAX_LANGGRAPH_ADAPTER_GRPC_PORT``, default ``50058``).
    """

    model_config = SettingsConfigDict(env_prefix="", extra="ignore")

    graph_mounts: list[GraphMount] = Field(
        default_factory=list,
        alias="LANGGRAPH_MOUNTS",
    )
    grpc_port: int = Field(default=50058, alias="ZYNAX_LANGGRAPH_ADAPTER_GRPC_PORT", ge=1)

    @field_validator("graph_mounts", mode="before")
    @classmethod
    def _parse_mounts_json(cls, value: Any) -> list[dict[str, Any]]:
        """Deserialise the JSON string from the LANGGRAPH_MOUNTS env var.

        Args:
            value: Raw env-var value — expected to be a JSON-encoded list.

        Returns:
            A list of raw dicts for pydantic to coerce into ``GraphMount`` instances.

        Raises:
            ValueError: When ``value`` is not valid JSON or not a list.
        """
        if isinstance(value, list):
            return value
        if not isinstance(value, str):
            raise ValueError("LANGGRAPH_MOUNTS must be a JSON-encoded list string")
        try:
            parsed = json.loads(value)
        except json.JSONDecodeError as exc:
            raise ValueError(f"LANGGRAPH_MOUNTS is not valid JSON: {exc}") from exc
        if not isinstance(parsed, list):
            raise ValueError("LANGGRAPH_MOUNTS must be a JSON array")
        return parsed

    @model_validator(mode="after")
    def _require_at_least_one_mount(self) -> AdapterConfig:
        """Ensure at least one graph mount is declared.

        Returns:
            Self after validation.

        Raises:
            ValueError: When ``graph_mounts`` is an empty list.
        """
        if not self.graph_mounts:
            raise ValueError("LANGGRAPH_MOUNTS must contain at least one graph mount")
        return self
