# automation/tests/platform_client.py
"""
ZynaxPlatformClient — thin REST client over api-gateway for the live-platform
readiness e2e (#1103, EPIC #881 O8).

This is exercised only when ``ZYNAX_PLATFORM_E2E=1`` and ``ZYNAX_GATEWAY_URL``
are set (see conftest.py). It maps each readiness assertion onto the gateway's
HTTP surface:

  apply()              → POST   /api/v1/apply              (kind: Workflow body)
  wait_for_completion()→ GET    /api/v1/workflows/{id}     (poll status)
  get_outputs()        → GET    /api/v1/workflows/{id}/outputs  (data-flow O,
                                 ADR-029 / M7 EPIC W) — aggregated verdict +
                                 decision-log entry

The client makes no assumptions beyond the documented endpoints; when a platform
does not yet expose the outputs/decision-log read path the e2e fails honestly
(rather than skipping), which is the correct signal that the platform is
incomplete for this assertion.
"""

from __future__ import annotations

from dataclasses import dataclass
import json
import time
import urllib.error
import urllib.request

# Terminal workflow states reported by api-gateway GET /api/v1/workflows/{id}.
_TERMINAL_OK = {"COMPLETED", "SUCCEEDED"}
_TERMINAL_FAIL = {"FAILED", "CANCELLED", "TERMINATED"}


@dataclass
class ApplyResult:
    """Result of POST /api/v1/apply for a kind: Workflow manifest."""

    workflow_id: str
    status: str


class ZynaxPlatformClient:
    """Minimal REST client for the api-gateway, used by the readiness e2e."""

    def __init__(self, gateway_url: str, api_token: str | None = None):
        self.gateway_url = gateway_url
        self.api_token = api_token

    # ── HTTP plumbing ─────────────────────────────────────────────────────
    def _request(self, method: str, path: str, body: bytes | None = None) -> dict:
        url = f"{self.gateway_url}{path}"
        headers = {"Content-Type": "application/json"}
        if self.api_token:
            headers["Authorization"] = f"Bearer {self.api_token}"
        req = urllib.request.Request(url, data=body, method=method, headers=headers)
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read().decode("utf-8")
        return json.loads(raw) if raw else {}

    # ── readiness operations ──────────────────────────────────────────────
    def apply(self, manifest_path: str) -> ApplyResult:
        """Apply a Workflow manifest; returns the created run/workflow id."""
        with open(manifest_path, "rb") as f:
            body = f.read()
        data = self._request("POST", "/api/v1/apply", body=body)
        workflow_id = data.get("run_id") or data.get("workflow_id")
        if not workflow_id:
            msg = f"apply returned no run id: {data!r}"
            raise AssertionError(msg)
        return ApplyResult(
            workflow_id=workflow_id, status=data.get("status", "PENDING")
        )

    def wait_for_completion(self, workflow_id: str, timeout: int = 60) -> str:
        """Poll GET /api/v1/workflows/{id} until terminal or timeout."""
        deadline = time.monotonic() + timeout
        last_status = "UNKNOWN"
        while time.monotonic() < deadline:
            data = self._request("GET", f"/api/v1/workflows/{workflow_id}")
            last_status = data.get("status", last_status)
            if last_status in _TERMINAL_OK or last_status in _TERMINAL_FAIL:
                return last_status
            time.sleep(2)
        return last_status

    def get_outputs(self, workflow_id: str) -> dict:
        """Read the run's published outputs (aggregated verdict + decision log)."""
        try:
            return self._request("GET", f"/api/v1/workflows/{workflow_id}/outputs")
        except urllib.error.HTTPError as exc:  # pragma: no cover - platform-dependent
            msg = (
                "outputs read path not available on this platform "
                f"(GET /api/v1/workflows/{{id}}/outputs → HTTP {exc.code}); "
                "the platform is incomplete for the readiness assertion"
            )
            raise AssertionError(msg) from exc
