# SPDX-License-Identifier: Apache-2.0
"""ChatCompletionHandler — validates input, streams provider tokens, emits TaskEvents."""

from __future__ import annotations

import asyncio
import json
from collections.abc import AsyncGenerator
from datetime import UTC, datetime

from google.protobuf.timestamp_pb2 import Timestamp  # type: ignore[import-untyped]
from pydantic import BaseModel, Field, ValidationError
from zynax.v1 import agent_pb2  # type: ignore[import-untyped]

from llm_adapter.config import ProviderConfig
from llm_adapter.providers import ProviderFunc, get_provider

_MAX_ERR = 512


class _ChatInput(BaseModel):
    """Validated chat completion input payload."""

    prompt: str = Field(..., min_length=1)
    system: str | None = None
    temperature: float = Field(default=0.7, ge=0.0, le=2.0)
    max_tokens: int | None = Field(default=None, ge=1)


def _ts() -> Timestamp:
    """Return the current UTC time as a protobuf Timestamp."""
    ts = Timestamp()
    ts.FromDatetime(datetime.now(UTC))
    return ts


def _progress(task_id: str, chunk: bytes) -> agent_pb2.TaskEvent:
    """Build a PROGRESS TaskEvent."""
    return agent_pb2.TaskEvent(
        task_id=task_id,
        event_type=agent_pb2.TASK_EVENT_TYPE_PROGRESS,
        payload=chunk,
        timestamp=_ts(),
    )


def _completed(task_id: str, full_text: str) -> agent_pb2.TaskEvent:
    """Build a COMPLETED TaskEvent with the full response payload."""
    return agent_pb2.TaskEvent(
        task_id=task_id,
        event_type=agent_pb2.TASK_EVENT_TYPE_COMPLETED,
        payload=json.dumps({"response": full_text}).encode(),
        timestamp=_ts(),
    )


def _failed(task_id: str, code: str, message: str) -> agent_pb2.TaskEvent:
    """Build a FAILED TaskEvent with a sanitised error message."""
    return agent_pb2.TaskEvent(
        task_id=task_id,
        event_type=agent_pb2.TASK_EVENT_TYPE_FAILED,
        error=agent_pb2.CapabilityError(code=code, message=message[:_MAX_ERR]),
        timestamp=_ts(),
    )


def _parse_input(payload: bytes) -> _ChatInput:
    """Decode and validate the input payload JSON.

    Args:
        payload: Raw bytes from ``ExecuteCapabilityRequest.input_payload``.

    Returns:
        Validated ``_ChatInput`` instance.

    Raises:
        ValueError: On JSON decode failure or schema validation error.
    """
    try:
        data = json.loads(payload)
    except (json.JSONDecodeError, UnicodeDecodeError) as exc:
        raise ValueError(f"invalid JSON in input_payload: {exc}") from exc
    try:
        return _ChatInput.model_validate(data)
    except ValidationError as exc:
        raise ValueError(str(exc)) from exc


class ChatCompletionHandler:
    """Handles chat_completion capability execution.

    Validates the input payload, invokes the configured provider, and
    streams PROGRESS events followed by exactly one terminal event.
    """

    def __init__(self, config: ProviderConfig) -> None:
        """Initialise with provider configuration.

        Args:
            config: Validated provider configuration; provider callable is
                built once at construction time.
        """
        self._max_tokens: int = config.max_tokens
        self._provider: ProviderFunc = get_provider(config)

    async def stream(
        self,
        task_id: str,
        input_payload: bytes,
        timeout_seconds: int,
    ) -> AsyncGenerator[agent_pb2.TaskEvent, None]:
        """Execute the capability and stream TaskEvents to the caller.

        Args:
            task_id: Unique task identifier; echoed on every event.
            input_payload: JSON-encoded ``_ChatInput`` bytes.
            timeout_seconds: Total wall-clock budget for the provider call.

        Yields:
            PROGRESS events (one per token chunk), then exactly one terminal
            COMPLETED or FAILED event.
        """
        try:
            inp = _parse_input(input_payload)
        except ValueError as exc:
            yield _failed(task_id, "INVALID_INPUT", str(exc))
            return

        max_tokens = inp.max_tokens or self._max_tokens
        chunks: list[bytes] = []
        emitted = False

        try:
            async with asyncio.timeout(float(timeout_seconds)):
                async for chunk in self._provider(
                    inp.prompt, inp.system, inp.temperature, max_tokens
                ):
                    ev = _progress(task_id, chunk)
                    yield ev
                    chunks.append(chunk)
                    emitted = True
        except TimeoutError:
            if not emitted:
                yield _progress(task_id, b"")
            yield _failed(task_id, "TIMEOUT", "request exceeded timeout")
            return
        except RuntimeError as exc:
            if not emitted:
                yield _progress(task_id, b"")
            yield _failed(task_id, "UPSTREAM_ERROR", str(exc))
            return

        if not emitted:
            yield _progress(task_id, b"")
        yield _completed(task_id, b"".join(chunks).decode(errors="replace"))
