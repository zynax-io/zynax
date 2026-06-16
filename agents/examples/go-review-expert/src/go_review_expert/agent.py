# SPDX-License-Identifier: Apache-2.0
"""Go review expert — a deterministic, rule-based Go code reviewer.

This is the reference *expert* agent (per ADR-033: the runtime substrate). It runs
on the Zynax SDK like any capability provider; the runtime ``kind: AgentDef``
registration that turns it into a dispatchable in-workflow expert is delivered
separately (step X.3, #1203). Here we only build the SDK agent + capability schema
+ BDD.

The review is rule-based and dependency-free so the example is deterministic and
offline. Each rule is a (regex, severity, message) triple; swap :data:`_RULES` or
the whole :func:`review` body for an LLM call to make it a real expert — the agent
contract and capability schema stay identical.
"""

from __future__ import annotations

import json
import re
from collections.abc import AsyncGenerator
from typing import Any

from zynax_sdk import Agent, capability

# (compiled pattern, severity, human-readable finding) — checked line by line.
_RULES: list[tuple[re.Pattern[str], str, str]] = [
    (re.compile(r"\bpanic\("), "error", "avoid panic in library code; return an error"),
    (
        re.compile(r"\bfmt\.Print(ln|f)?\("),
        "warning",
        "remove debug fmt.Print* before merge; use a structured logger",
    ),
    (
        re.compile(r"_\s*[,)]?\s*=\s*[^=].*\(\)|=\s*.*\bif\s+err\b"),
        "info",
        "ensure returned errors are checked, not discarded with _",
    ),
    (
        re.compile(r"//\s*(TODO|FIXME|XXX)\b", re.IGNORECASE),
        "warning",
        "unresolved TODO/FIXME left in the diff",
    ),
]


def review(diff: str) -> list[dict[str, Any]]:
    """Run the rule set over a Go source/diff string.

    Args:
        diff: Go source or unified-diff text to inspect.

    Returns:
        A list of finding dicts ``{"line", "severity", "message"}`` in file order.
    """
    findings: list[dict[str, Any]] = []
    for lineno, line in enumerate(diff.splitlines(), start=1):
        for pattern, severity, message in _RULES:
            if pattern.search(line):
                findings.append({"line": lineno, "severity": severity, "message": message})
    return findings


class GoReviewExpert(Agent):  # type: ignore[misc]  # SDK base is untyped (no py.typed)
    """Reviews a Go diff and returns structured findings (rule-based)."""

    @capability("go_review")  # type: ignore[untyped-decorator]  # untyped SDK decorator
    async def go_review(self, request: Any, context: Any) -> AsyncGenerator[Any, None]:
        """Review ``input_payload.diff`` and stream structured findings.

        Args:
            request: ``ExecuteCapabilityRequest`` whose ``input_payload`` is a JSON
                object of the form ``{"diff": "<go source or diff>"}``.
            context: gRPC ``ServicerContext`` (unused — kept for the SDK contract).

        Yields:
            A PROGRESS ``TaskEvent`` then a terminal COMPLETED ``TaskEvent`` with
            ``{"findings": [...], "finding_count": N, "approved": bool}``. If ``diff``
            is missing, a single FAILED ``TaskEvent`` with code ``EMPTY_INPUT``.
        """
        del context  # unused; the SDK validated the request before dispatch
        payload: dict[str, Any] = json.loads(request.input_payload) if request.input_payload else {}
        diff = payload.get("diff")
        if not isinstance(diff, str) or not diff.strip():
            yield self.report_failed(
                request.task_id,
                "EMPTY_INPUT",
                "input_payload.diff must be a non-empty string",
            )
            return

        yield self.report_progress(request.task_id, {"step": "reviewing"})
        findings = review(diff)
        has_error = any(f["severity"] == "error" for f in findings)
        yield self.report_completed(
            request.task_id,
            {
                "findings": findings,
                "finding_count": len(findings),
                "approved": not has_error,
            },
        )
