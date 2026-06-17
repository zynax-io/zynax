"""
conftest.py — fixtures for automation/tests/.

The ``zynax_client`` fixture is the seam between the platform-readiness e2e and a
running Zynax platform (api-gateway + Postgres-backed repos + EventBus). It is
*opt-in*: unless ``ZYNAX_PLATFORM_E2E=1`` is set AND a gateway endpoint is
configured, the fixture calls ``pytest.skip`` so the live-platform test SKIPS
CLEANLY (never ERROR, never a false xfail) on every machine that has no platform
— including the default PR check, where ``automation/tests/`` is not even wired
into CI.

The real assertion (``zynax apply`` of the orchestrator Workflow → aggregated
verdict + decision-log entry) runs only in the gated e2e job once a live platform
exposes the orchestration-side capability providers and an outputs/decision-log
read path. Until then the honest state is a clean skip, not a green expected
failure.

Opt-in contract:
  ZYNAX_PLATFORM_E2E=1            enable the live-platform leg (else skip)
  ZYNAX_GATEWAY_URL=<base url>    api-gateway base, e.g. http://localhost:8080
  ZYNAX_API_TOKEN=<bearer>        bearer token for POST /api/v1/apply (optional)
"""

import os

import pytest


def _platform_enabled() -> bool:
    """True only when the live-platform e2e is explicitly opted into."""
    return os.environ.get("ZYNAX_PLATFORM_E2E") == "1"


@pytest.fixture
def zynax_client():
    """Client for a live Zynax platform, or a clean skip when none is configured.

    Skips (does not fail/error) unless ``ZYNAX_PLATFORM_E2E=1`` and a gateway URL
    are set, so the live-platform test is a no-op on machines without a platform.
    """
    if not _platform_enabled():
        pytest.skip(
            "live-platform e2e disabled — set ZYNAX_PLATFORM_E2E=1 "
            "(+ ZYNAX_GATEWAY_URL) to run against a running Zynax platform"
        )

    gateway_url = os.environ.get("ZYNAX_GATEWAY_URL")
    if not gateway_url:
        pytest.skip("ZYNAX_PLATFORM_E2E=1 but ZYNAX_GATEWAY_URL is not set")

    # Imported lazily (so the skip path needs no network/HTTP dependency) and by
    # file location, so the import works regardless of how pytest is invoked.
    # Registered in sys.modules before exec so dataclass string annotations
    # (from __future__ import annotations) resolve against the module globals.
    import importlib.util
    from pathlib import Path
    import sys

    spec = importlib.util.spec_from_file_location(
        "platform_client", Path(__file__).with_name("platform_client.py")
    )
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)

    return module.ZynaxPlatformClient(
        gateway_url=gateway_url.rstrip("/"),
        api_token=os.environ.get("ZYNAX_API_TOKEN"),
    )
