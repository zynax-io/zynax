# SPDX-License-Identifier: Apache-2.0
"""Summarizer agent — a deterministic extractive summary, no LLM required.

This reference agent shows a slightly richer pattern than ``echo``:

* reading and validating a structured input payload,
* emitting incremental PROGRESS events,
* returning a structured COMPLETED result,
* failing gracefully with :meth:`report_failed` and a machine-readable code.

The summary strategy is intentionally trivial (first sentence of each document,
truncated) so the example stays dependency-free and deterministic. Swap
:func:`_summarize` for an LLM call to make it real — the agent contract is the same.
"""

from __future__ import annotations

import json
from collections.abc import AsyncGenerator
from typing import Any

from zynax_sdk import Agent, capability

_MAX_CHARS = 280


def _first_sentence(text: str) -> str:
    """Return the first sentence of ``text`` (split on the first period)."""
    stripped = text.strip()
    head, _, _ = stripped.partition(".")
    return head.strip() if head else stripped


def _summarize(documents: list[str]) -> str:
    """Build an extractive summary from the first sentence of each document.

    Args:
        documents: Non-empty list of document bodies.

    Returns:
        The joined first sentences, truncated to ``_MAX_CHARS`` characters.
    """
    sentences = [_first_sentence(doc) for doc in documents if doc.strip()]
    summary = ". ".join(s for s in sentences if s)
    if len(summary) > _MAX_CHARS:
        summary = summary[: _MAX_CHARS - 1].rstrip() + "…"
    return summary


class SummarizerAgent(Agent):  # type: ignore[misc]  # SDK base is untyped (no py.typed)
    """Summarizes a list of input documents into a single short string."""

    @capability("summarize")  # type: ignore[untyped-decorator]  # untyped SDK decorator
    async def summarize(self, request: Any, context: Any) -> AsyncGenerator[Any, None]:
        """Summarize ``input_payload.documents`` into one extractive summary.

        Args:
            request: ``ExecuteCapabilityRequest`` whose ``input_payload`` is a JSON
                object of the form ``{"documents": ["...", "..."]}``.
            context: gRPC ``ServicerContext`` (unused — kept for the SDK contract).

        Yields:
            A PROGRESS ``TaskEvent`` then a terminal COMPLETED ``TaskEvent`` with
            ``{"summary": "...", "document_count": N}``. If ``documents`` is missing
            or empty, a single FAILED ``TaskEvent`` with code ``EMPTY_INPUT``.
        """
        del context  # unused; the SDK validated the request before dispatch
        payload: dict[str, Any] = json.loads(request.input_payload) if request.input_payload else {}
        documents = payload.get("documents")
        if not isinstance(documents, list) or not documents:
            yield self.report_failed(
                request.task_id,
                "EMPTY_INPUT",
                "input_payload.documents must be a non-empty list",
            )
            return

        yield self.report_progress(
            request.task_id, {"step": "summarizing", "documents": len(documents)}
        )
        summary = _summarize([str(d) for d in documents])
        yield self.report_completed(
            request.task_id,
            {"summary": summary, "document_count": len(documents)},
        )
