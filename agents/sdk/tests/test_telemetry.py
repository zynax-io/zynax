# SPDX-License-Identifier: Apache-2.0
"""Unit tests for SDK OpenTelemetry traces + logs (canvas O.6)."""

from __future__ import annotations

import logging
import os
import sys
from typing import Any

import pytest
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.sdk.trace.export.in_memory_span_exporter import (
    InMemorySpanExporter,
)
from opentelemetry.trace import format_trace_id

sys.path.insert(
    0, os.path.join(os.path.dirname(__file__), "../../../protos/generated/python")
)
from zynax.v1 import agent_pb2  # noqa: E402

from zynax_sdk import Agent, capability  # noqa: E402
from zynax_sdk import telemetry  # noqa: E402

# A valid W3C traceparent (version-traceid-spanid-flags). 32-hex trace id.
_TRACE_ID_HEX = "0af7651916cd43dd8448eb211c80319c"
_TRACEPARENT = f"00-{_TRACE_ID_HEX}-b7ad6b7169203331-01"


@pytest.fixture()
def in_memory_tracer(monkeypatch: Any) -> Any:
    """Route spans into an in-memory exporter via a controlled tracer.

    OpenTelemetry's global TracerProvider can only be set once per process, so we
    cannot rely on ``set_tracer_provider`` here. Instead we patch the module-level
    ``trace.get_tracer`` so ``capability_span`` obtains a tracer from a provider
    we own and can inspect.
    """
    exporter = InMemorySpanExporter()
    provider = TracerProvider()
    provider.add_span_processor(SimpleSpanProcessor(exporter))
    tracer = provider.get_tracer("zynax_sdk_test")
    monkeypatch.setattr(telemetry.trace, "get_tracer", lambda *_a, **_k: tracer)
    telemetry._initialized = False
    yield exporter
    exporter.clear()


class _MockContext:
    """gRPC context mock exposing invocation_metadata()."""

    def __init__(self, metadata: list[tuple[str, Any]] | None = None) -> None:
        self._metadata = metadata or []

    def invocation_metadata(self) -> list[tuple[str, Any]]:
        return self._metadata


def _make_request(**kwargs: Any) -> Any:
    defaults: dict[str, Any] = dict(
        capability_name="greet",
        task_id="task-1",
        input_payload=b"{}",
    )
    defaults.update(kwargs)
    return agent_pb2.ExecuteCapabilityRequest(**defaults)


# ── is_enabled / endpoint gating ───────────────────────────────────────────────


class TestEnabledGating:
    def test_disabled_when_endpoint_unset(self, monkeypatch: Any) -> None:
        monkeypatch.delenv(telemetry._ENDPOINT_ENV, raising=False)
        assert telemetry.is_enabled() is False

    def test_enabled_when_endpoint_set(self, monkeypatch: Any) -> None:
        monkeypatch.setenv(telemetry._ENDPOINT_ENV, "http://collector:4317")
        assert telemetry.is_enabled() is True


# ── init_telemetry ──────────────────────────────────────────────────────────────


class TestInitTelemetry:
    def test_init_noop_when_endpoint_unset(self, monkeypatch: Any) -> None:
        monkeypatch.delenv(telemetry._ENDPOINT_ENV, raising=False)
        telemetry._initialized = False
        assert telemetry.init_telemetry("test-svc") is False

    def test_init_idempotent_when_already_initialised(self, monkeypatch: Any) -> None:
        monkeypatch.setenv(telemetry._ENDPOINT_ENV, "http://collector:4317")
        telemetry._initialized = True
        assert telemetry.init_telemetry("test-svc") is False

    def test_otlp_exporters_absent_returns_none(self, monkeypatch: Any) -> None:
        # The OTLP extra is not installed in CI; the import fails and degrades.
        span_exp, log_exp = telemetry._otlp_exporters("http://collector:4317")
        assert span_exp is None
        assert log_exp is None

    def test_otlp_exporters_present_when_importable(self, monkeypatch: Any) -> None:
        import types as _types

        grpc_pkg = "opentelemetry.exporter.otlp.proto.grpc"
        log_mod = _types.ModuleType(f"{grpc_pkg}._log_exporter")
        trace_mod = _types.ModuleType(f"{grpc_pkg}.trace_exporter")

        class _FakeLogExporter:
            def __init__(self, endpoint: str) -> None:
                self.endpoint = endpoint

        class _FakeSpanExporter:
            def __init__(self, endpoint: str) -> None:
                self.endpoint = endpoint

        log_mod.OTLPLogExporter = _FakeLogExporter  # type: ignore[attr-defined]
        trace_mod.OTLPSpanExporter = _FakeSpanExporter  # type: ignore[attr-defined]
        monkeypatch.setitem(sys.modules, f"{grpc_pkg}._log_exporter", log_mod)
        monkeypatch.setitem(sys.modules, f"{grpc_pkg}.trace_exporter", trace_mod)

        span_exp, log_exp = telemetry._otlp_exporters("http://collector:4317")
        assert isinstance(span_exp, _FakeSpanExporter)
        assert isinstance(log_exp, _FakeLogExporter)
        assert span_exp.endpoint == "http://collector:4317"

    def test_init_installs_providers_when_endpoint_set(self, monkeypatch: Any) -> None:
        monkeypatch.setenv(telemetry._ENDPOINT_ENV, "http://collector:4317")
        telemetry._initialized = False
        root = logging.getLogger()
        try:
            assert telemetry.init_telemetry("test-svc", "9.9.9") is True
            assert telemetry._initialized is True
            from opentelemetry.sdk._logs import LoggingHandler

            assert any(isinstance(h, LoggingHandler) for h in root.handlers)
            # second call no-ops (already initialised) and does not duplicate handler
            assert telemetry.init_telemetry("test-svc") is False
            n = sum(isinstance(h, LoggingHandler) for h in root.handlers)
            assert n == 1
        finally:
            for h in list(root.handlers):
                from opentelemetry.sdk._logs import LoggingHandler as _LH

                if isinstance(h, _LH):
                    root.removeHandler(h)
            telemetry._initialized = False


# ── metadata carrier / extract_context ─────────────────────────────────────────


class TestExtractContext:
    def test_extract_none_metadata_returns_none(self) -> None:
        assert telemetry.extract_context(None) is None

    def test_extract_without_traceparent_returns_none(self) -> None:
        assert telemetry.extract_context([("authorization", "secret")]) is None

    def test_carrier_only_reads_trace_keys(self) -> None:
        carrier = telemetry._metadata_to_carrier(
            [
                ("traceparent", _TRACEPARENT),
                ("tracestate", "foo=bar"),
                ("authorization", "Bearer secret"),
                ("x-session", "sess-1"),
            ]
        )
        assert carrier == {"traceparent": _TRACEPARENT, "tracestate": "foo=bar"}

    def test_carrier_decodes_bytes_values(self) -> None:
        carrier = telemetry._metadata_to_carrier(
            [("traceparent", _TRACEPARENT.encode())]
        )
        assert carrier["traceparent"] == _TRACEPARENT

    def test_extract_with_traceparent_returns_context(self) -> None:
        from opentelemetry.propagate import set_global_textmap
        from opentelemetry.trace.propagation.tracecontext import (
            TraceContextTextMapPropagator,
        )

        set_global_textmap(TraceContextTextMapPropagator())
        ctx = telemetry.extract_context([("traceparent", _TRACEPARENT)])
        assert ctx is not None
        span_ctx = trace.get_current_span(ctx).get_span_context()
        assert format_trace_id(span_ctx.trace_id) == _TRACE_ID_HEX


# ── capability_span ─────────────────────────────────────────────────────────────


class TestCapabilitySpan:
    def test_span_named_capability_dot_name(self, in_memory_tracer: Any) -> None:
        with telemetry.capability_span("summarize", _MockContext()):
            pass
        spans = in_memory_tracer.get_finished_spans()
        assert len(spans) == 1
        assert spans[0].name == "capability.summarize"
        assert spans[0].attributes["zynax.capability.name"] == "summarize"

    def test_span_child_of_inbound_traceparent(self, in_memory_tracer: Any) -> None:
        from opentelemetry.propagate import set_global_textmap
        from opentelemetry.trace.propagation.tracecontext import (
            TraceContextTextMapPropagator,
        )

        set_global_textmap(TraceContextTextMapPropagator())
        ctx = _MockContext([("traceparent", _TRACEPARENT)])
        with telemetry.capability_span("greet", ctx):
            pass
        spans = in_memory_tracer.get_finished_spans()
        assert format_trace_id(spans[0].context.trace_id) == _TRACE_ID_HEX

    def test_span_records_exception_and_reraises(self, in_memory_tracer: Any) -> None:
        with pytest.raises(ValueError, match="boom"):
            with telemetry.capability_span("fail", _MockContext()):
                raise ValueError("boom")
        spans = in_memory_tracer.get_finished_spans()
        assert spans[0].status.status_code.name == "ERROR"
        assert any(e.name == "exception" for e in spans[0].events)

    def test_span_with_none_context(self, in_memory_tracer: Any) -> None:
        with telemetry.capability_span("nocontext", None):
            pass
        spans = in_memory_tracer.get_finished_spans()
        assert spans[0].name == "capability.nocontext"

    def test_invocation_metadata_failure_is_swallowed(
        self, in_memory_tracer: Any
    ) -> None:
        class _Bad:
            def invocation_metadata(self) -> Any:
                raise RuntimeError("metadata unavailable")

        with telemetry.capability_span("robust", _Bad()):
            pass
        spans = in_memory_tracer.get_finished_spans()
        assert spans[0].name == "capability.robust"


# ── integration: Agent.ExecuteCapability wraps handlers in a span ───────────────


class _GreetAgent(Agent):
    @capability("greet")
    async def greet(self, request: Any, context: Any) -> Any:
        yield self.report_completed(request.task_id, {"ok": True})


class TestAgentDispatchTracing:
    def test_dispatch_emits_capability_span(self, in_memory_tracer: Any) -> None:
        agent = _GreetAgent()
        events = list(agent.ExecuteCapability(_make_request(), _MockContext()))
        assert events[0].event_type == agent_pb2.TASK_EVENT_TYPE_COMPLETED
        spans = in_memory_tracer.get_finished_spans()
        assert [s.name for s in spans] == ["capability.greet"]

    def test_unknown_capability_emits_no_span(self, in_memory_tracer: Any) -> None:
        agent = _GreetAgent()
        list(
            agent.ExecuteCapability(
                _make_request(capability_name="unknown"), _MockContext()
            )
        )
        assert in_memory_tracer.get_finished_spans() == ()
