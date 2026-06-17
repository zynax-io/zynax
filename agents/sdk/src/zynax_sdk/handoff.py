# SPDX-License-Identifier: Apache-2.0
"""Agent handoff context contract + helpers (canvas EPIC C, step C.4).

When the broker dispatches a capability to an agent it hands off a deterministic
**context** so the run stays traceable and data-scoped end to end. This module
defines that contract — :class:`HandoffContext` — and the two helpers an agent
author uses to honour it:

* :func:`inbound_context` — read the context an agent **receives** from the
  ``ExecuteCapabilityRequest`` proto fields and the inbound gRPC metadata.
* :func:`outbound_metadata` — emit the context an agent **returns / forwards**
  when it itself calls another Zynax hop, so correlation and trace survive the
  next handoff.

The carrier keys mirror the Go gateway exactly (``services/api-gateway`` and
``libs/zynaxobs``) so a request-id set at the gateway is the same value an agent
observes — no bespoke header formats (canvas Norms / ADR-031):

================  ============================  =================================
field             gRPC metadata key             source of truth
================  ============================  =================================
request_id        ``request-id``                api-gateway correlation interceptor
namespace         ``x-namespace``               api-gateway correlation interceptor
traceparent       ``traceparent`` (W3C)         tracing interceptor (C.2)
tracestate        ``tracestate`` (W3C)          tracing interceptor (C.2)
================  ============================  =================================

The data-context (EPIC W / C.3) is scoped server-side by ``namespace`` +
``workflow_id`` (the run id); the agent receives those identifiers — never the
data store handle itself — so it cannot reach across runs (canvas Safeguards).

**Safeguard:** only correlation ids and W3C trace headers cross the handoff —
never auth tokens, session data, secrets, or credentials (canvas C Safeguards).
"""

from __future__ import annotations

from collections.abc import Sequence
from dataclasses import dataclass
from typing import Any

# gRPC metadata keys carrying the correlation context. These mirror
# ``requestIDMetaKey`` / ``namespaceMetaKey`` in
# ``services/api-gateway/internal/infrastructure/clients.go`` and the W3C trace
# headers propagated by the tracing interceptors — keep them byte-for-byte equal.
_REQUEST_ID_KEY = "request-id"
_NAMESPACE_KEY = "x-namespace"
_TRACEPARENT_KEY = "traceparent"
_TRACESTATE_KEY = "tracestate"

# Keys an agent must never read out of inbound metadata into a handoff context —
# documented here so the safeguard is explicit and testable.
_FORBIDDEN_KEYS = frozenset(
    {"authorization", "cookie", "x-api-key", "set-cookie", "proxy-authorization"}
)


@dataclass(frozen=True)
class HandoffContext:
    """The deterministic context an agent receives on, and returns from, a handoff.

    Immutable by design: a handler reads it but never mutates it, so the same
    correlation and trace identifiers flow unchanged to the next hop. Construct
    it from a request with :func:`inbound_context`; turn it back into gRPC
    metadata for a downstream call with :func:`outbound_metadata`.

    Attributes:
        request_id: Stable correlation id set at the gateway; appears in every
            downstream span and log line for the run (canvas C.2). Empty when the
            call did not originate behind the gateway.
        namespace: Tenant / routing namespace; also half of the data-context
            scope key (canvas C.3). Empty when unset.
        workflow_id: The workflow run id; with ``namespace`` it scopes the
            data-context so a state cannot read another run's data.
        task_id: The opaque per-task identifier from the originating request.
        traceparent: W3C ``traceparent`` carrying the remote span, or empty.
        tracestate: W3C ``tracestate`` vendor data, or empty.
    """

    request_id: str = ""
    namespace: str = ""
    workflow_id: str = ""
    task_id: str = ""
    traceparent: str = ""
    tracestate: str = ""

    def is_traced(self) -> bool:
        """Return ``True`` when a W3C ``traceparent`` was carried on the handoff."""
        return bool(self.traceparent)


def _carrier_from_metadata(
    metadata: Sequence[tuple[str, Any]] | None,
) -> dict[str, str]:
    """Build a lower-cased carrier dict from gRPC invocation metadata.

    Only the correlation and W3C trace keys are read; forbidden keys (auth,
    cookies, api keys) are dropped so they can never leak into a handoff context.
    """
    carrier: dict[str, str] = {}
    if not metadata:
        return carrier
    allowed = {
        _REQUEST_ID_KEY,
        _NAMESPACE_KEY,
        _TRACEPARENT_KEY,
        _TRACESTATE_KEY,
    }
    for key, value in metadata:
        lower = key.lower()
        if lower in _FORBIDDEN_KEYS:
            continue
        if lower in allowed:
            carrier[lower] = value.decode() if isinstance(value, bytes) else str(value)
    return carrier


def _invocation_metadata(context: Any) -> Sequence[tuple[str, Any]] | None:
    """Return ``context.invocation_metadata()`` if the context exposes it."""
    getter = getattr(context, "invocation_metadata", None)
    if getter is None:
        return None
    try:
        metadata: Sequence[tuple[str, Any]] = getter()
    except Exception:  # pragma: no cover - defensive: never fail a handler here
        return None
    return metadata


def inbound_context(request: Any, context: Any = None) -> HandoffContext:
    """Read the deterministic handoff context an agent receives.

    Correlation (``request_id``, ``namespace``) and W3C trace headers are read
    from the inbound gRPC metadata when ``context`` is supplied; ``request_id``,
    ``workflow_id`` and ``task_id`` fall back to the proto request fields so the
    context is populated even outside a transport (e.g. in unit tests). The proto
    ``request_id`` wins only when the metadata did not carry one, keeping the
    gateway-set value authoritative.

    Args:
        request: The ``ExecuteCapabilityRequest`` proto message (or any object
            exposing ``request_id`` / ``workflow_id`` / ``task_id``).
        context: The gRPC ``ServicerContext`` (or any object exposing
            ``invocation_metadata()``); optional.

    Returns:
        A frozen :class:`HandoffContext`. Never raises on missing fields — absent
        identifiers are returned as empty strings.
    """
    carrier = _carrier_from_metadata(_invocation_metadata(context))

    request_id = carrier.get(_REQUEST_ID_KEY, "") or _attr(request, "request_id")
    namespace = carrier.get(_NAMESPACE_KEY, "") or _attr(request, "namespace")

    return HandoffContext(
        request_id=request_id,
        namespace=namespace,
        workflow_id=_attr(request, "workflow_id"),
        task_id=_attr(request, "task_id"),
        traceparent=carrier.get(_TRACEPARENT_KEY, ""),
        tracestate=carrier.get(_TRACESTATE_KEY, ""),
    )


def outbound_metadata(ctx: HandoffContext) -> list[tuple[str, str]]:
    """Emit gRPC metadata that forwards the handoff context to the next hop.

    Returns the correlation and W3C trace headers as an ordered metadata list
    suitable for ``stub.Method(req, metadata=outbound_metadata(ctx))``. Unset
    fields are omitted so an absent identifier is never sent as an empty header,
    and the ordering is deterministic (request-id, namespace, traceparent,
    tracestate) so two equal contexts always produce identical metadata.

    Only correlation ids and trace headers are emitted — never auth tokens or
    secrets (canvas C Safeguards).

    Args:
        ctx: The :class:`HandoffContext` to forward.

    Returns:
        An ordered list of ``(key, value)`` metadata pairs; possibly empty.
    """
    md: list[tuple[str, str]] = []
    if ctx.request_id:
        md.append((_REQUEST_ID_KEY, ctx.request_id))
    if ctx.namespace:
        md.append((_NAMESPACE_KEY, ctx.namespace))
    if ctx.traceparent:
        md.append((_TRACEPARENT_KEY, ctx.traceparent))
    if ctx.tracestate:
        md.append((_TRACESTATE_KEY, ctx.tracestate))
    return md


def _attr(obj: Any, name: str) -> str:
    """Return ``str(obj.<name>)`` or ``""`` when the attribute is missing/falsy."""
    value = getattr(obj, name, "")
    return str(value) if value else ""
