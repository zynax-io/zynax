# SPDX-License-Identifier: Apache-2.0
"""OpenTelemetry traces + logs for Zynax capability handlers (canvas O.6).

This module mirrors the Go ``libs/zynaxobs`` provider bootstrap: telemetry is
**off by default** and only activates when ``ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT``
is set, so an agent runs with zero exporter overhead and no collector configured.

What it provides:

* :func:`init_telemetry` — installs OTLP/gRPC tracer + logger providers and the
  W3C trace-context propagator as the OpenTelemetry globals (no-op when unset).
* :func:`extract_context` — reads the inbound W3C ``traceparent`` from a gRPC
  request's invocation metadata so a handler span is stitched to the broker's
  trace across the dispatch hop.
* :func:`capability_span` — starts a ``capability.<name>`` span as a child of the
  extracted context; logs emitted inside are correlated by ``trace_id``.

Only trace context is read from inbound metadata — never auth tokens or session
data (canvas O.5 / Safeguards).
"""

from __future__ import annotations

import logging
import os
from collections.abc import Iterator, Sequence
from contextlib import contextmanager
from typing import Any

from opentelemetry import trace
from opentelemetry.context import Context
from opentelemetry.trace import Span, Status, StatusCode

# Zynax-prefixed OTLP collector endpoint (canvas Norms: env prefix `ZYNAX_OTEL_`).
# Telemetry is opt-in: unset ⇒ no providers installed ⇒ no-op tracer/logger.
_ENDPOINT_ENV = "ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT"

# Instrumentation scope name for the tracer obtained from the global provider.
_TRACER_NAME = "zynax_sdk"

# W3C trace-context header carried in inbound gRPC metadata by the upstream
# otelgrpc client stats handler (see services/*/tracing.go).
_TRACEPARENT_KEY = "traceparent"
_TRACESTATE_KEY = "tracestate"

_initialized = False


def _endpoint() -> str:
    """Return the configured OTLP endpoint, or an empty string when disabled."""
    return os.getenv(_ENDPOINT_ENV, "")


def is_enabled() -> bool:
    """Return ``True`` when an OTLP endpoint is configured (telemetry is on)."""
    return bool(_endpoint())


def init_telemetry(
    service_name: str,
    service_version: str = "0.1.0",
) -> bool:
    """Install OTLP/gRPC tracer + logger providers and the W3C propagator.

    Idempotent. When ``ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT`` is unset this is a
    no-op and the global no-op tracer/logger remain in place, so an agent runs
    unchanged without a collector (canvas Norms: off by default, zero overhead
    when disabled). When set, span and log records carry the semconv resource
    attributes ``service.name`` / ``service.version`` and are exported via
    OTLP/gRPC, identically to the Go services.

    Args:
        service_name: ``service.name`` resource attribute (e.g. ``"llm-adapter"``).
        service_version: ``service.version`` resource attribute.

    Returns:
        ``True`` when real providers were installed, ``False`` when telemetry is
        disabled (endpoint unset) or already initialised.
    """
    global _initialized

    # The W3C propagator is cheap and harmless to install even when disabled, so
    # extract_context() works the moment an endpoint is configured upstream.
    from opentelemetry.propagate import set_global_textmap
    from opentelemetry.trace.propagation.tracecontext import (
        TraceContextTextMapPropagator,
    )

    set_global_textmap(TraceContextTextMapPropagator())

    if _initialized:
        return False

    endpoint = _endpoint()
    if not endpoint:
        return False

    # Imported lazily: the SDK is only needed when telemetry is on, keeping the
    # import cost off the disabled path.
    from opentelemetry._logs import set_logger_provider
    from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
    from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
    from opentelemetry.sdk.resources import Resource
    from opentelemetry.sdk.trace import TracerProvider
    from opentelemetry.sdk.trace.export import BatchSpanProcessor

    resource = Resource.create(
        {
            "service.name": service_name,
            "service.version": service_version,
        }
    )

    span_exporter, log_exporter = _otlp_exporters(endpoint)

    tracer_provider = TracerProvider(resource=resource)
    if span_exporter is not None:
        tracer_provider.add_span_processor(BatchSpanProcessor(span_exporter))
    trace.set_tracer_provider(tracer_provider)

    logger_provider = LoggerProvider(resource=resource)
    if log_exporter is not None:
        logger_provider.add_log_record_processor(BatchLogRecordProcessor(log_exporter))
    set_logger_provider(logger_provider)

    # Bridge stdlib logging into the OTLP logs pipeline so handler log lines are
    # exported and correlated with the active span's trace_id in the UI.
    handler = LoggingHandler(level=logging.NOTSET, logger_provider=logger_provider)
    root = logging.getLogger()
    if not any(isinstance(h, LoggingHandler) for h in root.handlers):
        root.addHandler(handler)

    _initialized = True
    return True


def _otlp_exporters(endpoint: str) -> tuple[Any | None, Any | None]:
    """Build the OTLP/gRPC span + log exporters, or ``(None, None)`` if absent.

    The OTLP exporter pins ``protobuf<7`` while the SDK core pins ``protobuf>=7``,
    so it is an optional ``[otlp]`` extra. When it is not installed this returns
    no exporters and logs a warning rather than raising — providers are still
    installed so spans/logs flow to any in-process processor and the agent never
    crashes on a telemetry import (canvas Safeguards: never block the request
    path on the exporter).
    """
    try:
        from opentelemetry.exporter.otlp.proto.grpc._log_exporter import (
            OTLPLogExporter,
        )
        from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import (
            OTLPSpanExporter,
        )
    except ImportError:
        logging.getLogger(__name__).warning(
            "OTLP exporter not installed; spans/logs will not be exported. "
            "Install the 'otlp' extra (zynax-sdk[otlp]) to enable live export."
        )
        return None, None
    return (
        OTLPSpanExporter(endpoint=endpoint),
        OTLPLogExporter(endpoint=endpoint),
    )


def _metadata_to_carrier(
    metadata: Sequence[tuple[str, Any]] | None,
) -> dict[str, str]:
    """Build a W3C carrier dict from gRPC invocation metadata.

    Only the ``traceparent`` / ``tracestate`` keys are read — never auth tokens
    or session data (canvas O.5 safeguard). gRPC metadata keys are lower-cased.
    """
    carrier: dict[str, str] = {}
    if not metadata:
        return carrier
    for key, value in metadata:
        lower = key.lower()
        if lower in (_TRACEPARENT_KEY, _TRACESTATE_KEY):
            carrier[lower] = value.decode() if isinstance(value, bytes) else str(value)
    return carrier


def extract_context(
    metadata: Sequence[tuple[str, Any]] | None,
) -> Context | None:
    """Extract the W3C trace context from inbound gRPC invocation metadata.

    The upstream broker injects ``traceparent`` into the metadata of the
    outbound ``ExecuteCapability`` call via the otelgrpc client stats handler;
    extracting it here stitches the handler span to the originating trace.

    Args:
        metadata: The request's invocation metadata
            (``context.invocation_metadata()``), or ``None``.

    Returns:
        A propagation :class:`Context` carrying the remote span, or ``None`` when
        no ``traceparent`` is present (so the handler span starts a new trace).
    """
    carrier = _metadata_to_carrier(metadata)
    if _TRACEPARENT_KEY not in carrier:
        return None
    from opentelemetry.propagate import extract

    return extract(carrier)


def _invocation_metadata(context: Any) -> Sequence[tuple[str, Any]] | None:
    """Return ``context.invocation_metadata()`` if the context exposes it."""
    getter = getattr(context, "invocation_metadata", None)
    if getter is None:
        return None
    try:
        metadata: Sequence[tuple[str, Any]] = getter()
    except Exception:  # pragma: no cover — defensive: never fail a handler on telemetry
        return None
    return metadata


@contextmanager
def capability_span(
    capability_name: str,
    context: Any = None,
) -> Iterator[Span]:
    """Start a ``capability.<name>`` span around a capability handler.

    The span is a child of the trace context extracted from ``context``'s inbound
    gRPC metadata when present, otherwise a new root span. When telemetry is
    disabled the global no-op tracer yields a no-op span — zero overhead. The
    span is marked ``ERROR`` and the exception recorded if the handler raises.

    Args:
        capability_name: The capability identifier; the span is named
            ``capability.<capability_name>``.
        context: The gRPC ``ServicerContext`` (or any object exposing
            ``invocation_metadata()``); used to extract the parent trace context.

    Yields:
        The active :class:`~opentelemetry.trace.Span`.
    """
    tracer = trace.get_tracer(_TRACER_NAME)
    parent = extract_context(_invocation_metadata(context))
    with tracer.start_as_current_span(
        f"capability.{capability_name}",
        context=parent,
        attributes={"zynax.capability.name": capability_name},
    ) as span:
        try:
            yield span
        except (
            BaseException
        ) as exc:  # re-raised: telemetry never swallows handler errors
            span.set_status(Status(StatusCode.ERROR, str(exc)))
            span.record_exception(exc)
            raise
