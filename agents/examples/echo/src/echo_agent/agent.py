# SPDX-License-Identifier: Apache-2.0
"""Echo agent — copies its input payload to its output payload.

This is the canonical "write-your-own-agent" starting point. It shows the three
moving parts every Zynax SDK agent needs:

1. Subclass :class:`zynax_sdk.Agent`.
2. Decorate an async generator method with ``@capability("<name>")``.
3. Yield :meth:`report_progress` events then a terminal :meth:`report_completed`
   (or :meth:`report_failed`) event.

The SDK handles all gRPC routing and ``TaskEvent`` streaming.
"""

from __future__ import annotations

import json
from collections.abc import AsyncGenerator
from typing import Any

from zynax_sdk import Agent, capability


class EchoAgent(Agent):  # type: ignore[misc]  # SDK base is untyped (no py.typed)
    """A minimal agent that echoes its JSON input back as its result."""

    @capability("echo")  # type: ignore[untyped-decorator]  # untyped SDK decorator
    async def echo(self, request: Any, context: Any) -> AsyncGenerator[Any, None]:
        """Echo the request ``input_payload`` back in the completed event.

        Args:
            request: ``ExecuteCapabilityRequest`` proto message. ``input_payload``
                must be a JSON object; the SDK already validated it as JSON.
            context: gRPC ``ServicerContext`` (unused — kept for the SDK contract).

        Yields:
            One PROGRESS ``TaskEvent`` then one COMPLETED ``TaskEvent`` whose
            payload is ``{"echo": <decoded input>}``.
        """
        del context  # unused; the SDK validated the request before dispatch
        payload: dict[str, Any] = json.loads(request.input_payload) if request.input_payload else {}
        yield self.report_progress(request.task_id, {"step": "received"})
        yield self.report_completed(request.task_id, {"echo": payload})
