# SPDX-License-Identifier: Apache-2.0
"""CapabilityRouter — maps capability names to handlers and JSON Schema bytes."""

from __future__ import annotations

import json

from llm_adapter.config import ProviderConfig
from llm_adapter.handler import ChatCompletionHandler

_CHAT_INPUT_SCHEMA: dict[str, object] = {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "required": ["prompt"],
    "properties": {
        "prompt": {"type": "string", "minLength": 1},
        "system": {"type": "string"},
        "temperature": {"type": "number", "minimum": 0, "maximum": 2},
        "max_tokens": {"type": "integer", "minimum": 1},
    },
    "additionalProperties": False,
}

_CHAT_OUTPUT_SCHEMA: dict[str, object] = {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "required": ["response"],
    "properties": {"response": {"type": "string"}},
}

_INPUT_BYTES: bytes = json.dumps(_CHAT_INPUT_SCHEMA).encode()
_OUTPUT_BYTES: bytes = json.dumps(_CHAT_OUTPUT_SCHEMA).encode()


class CapabilityRouter:
    """Routes capability names to handlers and exposes their JSON Schemas.

    Built once at adapter startup from ``ProviderConfig``. Immutable after
    construction — all state is read-only after ``__init__``.
    """

    def __init__(self, config: ProviderConfig) -> None:
        """Initialise the router with a single chat_completion capability.

        Args:
            config: Validated provider configuration used to build the handler.
        """
        self._handlers: dict[str, ChatCompletionHandler] = {
            "chat_completion": ChatCompletionHandler(config),
        }
        self._schemas: dict[str, tuple[bytes, bytes]] = {
            "chat_completion": (_INPUT_BYTES, _OUTPUT_BYTES),
        }

    def dispatch(self, capability_name: str) -> ChatCompletionHandler:
        """Return the handler for a named capability.

        Args:
            capability_name: Capability name from ``ExecuteCapabilityRequest``.

        Returns:
            The ``ChatCompletionHandler`` registered for this capability.

        Raises:
            KeyError: When ``capability_name`` is not registered.
        """
        return self._handlers[capability_name]

    def get_schema(self, capability_name: str) -> tuple[bytes, bytes]:
        """Return the (input_schema, output_schema) JSON bytes for a capability.

        Args:
            capability_name: Registered capability name.

        Returns:
            Tuple of (input_schema_json_bytes, output_schema_json_bytes).

        Raises:
            KeyError: When ``capability_name`` is not registered.
        """
        return self._schemas[capability_name]

    def capability_names(self) -> list[str]:
        """Return all registered capability names."""
        return list(self._handlers.keys())
