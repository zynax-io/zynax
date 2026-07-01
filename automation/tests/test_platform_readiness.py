# automation/tests/test_platform_readiness.py
import json
from pathlib import Path

import pytest
import yaml

try:
    import jsonschema

    HAS_JSONSCHEMA = True
except ImportError:
    HAS_JSONSCHEMA = False

REPO_ROOT = Path(__file__).resolve().parents[2]
SCHEMA_PATH = REPO_ROOT / "spec/schemas/agent-def.schema.json"
WORKFLOW_SCHEMA_PATH = REPO_ROOT / "spec/schemas/workflow.schema.json"
ORCHESTRATOR_YAML = REPO_ROOT / "automation/workflows/dev-advisory-orchestrator.yaml"
HELLO_WORLD_YAML = REPO_ROOT / "spec/workflows/examples/hello-world.yaml"
EXPERT_YAMLS = list((REPO_ROOT / "automation/workflows/experts").glob("*.yaml"))


class TestPlatformReadiness:
    @pytest.mark.skipif(not HAS_JSONSCHEMA, reason="jsonschema package required")
    def test_orchestrator_workflow_schema_valid(self):
        """Orchestrator manifest validates against workflow.schema.json.

        Delivered by O3 (#1098): the orchestrator is kind: Workflow per
        ADR-028, so it validates against the Workflow schema (the experts
        keep validating against agent-def.schema.json below).
        """
        with open(WORKFLOW_SCHEMA_PATH) as f:
            schema = json.load(f)
        with open(ORCHESTRATOR_YAML) as f:
            doc = yaml.safe_load(f)
        jsonschema.validate(doc, schema)

    @pytest.mark.skipif(not HAS_JSONSCHEMA, reason="jsonschema package required")
    def test_expert_agentdefs_schema_valid(self):
        """All 9 expert AgentDef YAMLs validate against schema (O2, #1097)."""
        assert len(EXPERT_YAMLS) == 9, f"Expected 9 experts, got {len(EXPERT_YAMLS)}"
        with open(SCHEMA_PATH) as f:
            schema = json.load(f)
        for p in EXPERT_YAMLS:
            with open(p) as f:
                doc = yaml.safe_load(f)
            jsonschema.validate(doc, schema)

    def test_declared_output_read_path(self, zynax_client):
        """apply → COMPLETED → GET /outputs returns a run's declared output
        (ADR-042, M7.U). This is the proof that closes #1103 gap #4 — the
        api-gateway workflow-outputs read path.

        Uses hello-world.yaml — a zero-dependency echo workflow that declares a
        terminal ``outputs: {message: $.states.greet.output.message}`` — so this
        leg is INDEPENDENT of gaps #2 (CEL-vs-Go-template guards) and #3 (missing
        capability providers) that keep the orchestrator leg below gated. The
        same path is also asserted live in CI by scripts/e2e/hello-world-smoke.sh
        (e2e-smoke.yml). Gated on ``ZYNAX_PLATFORM_E2E=1`` (else a clean skip).
        """
        result = zynax_client.apply(str(HELLO_WORLD_YAML))
        assert result.workflow_id is not None, "apply must return a run id"

        status = zynax_client.wait_for_completion(result.workflow_id, timeout=60)
        assert status in ("COMPLETED", "SUCCEEDED"), (
            f"hello-world run did not reach a terminal success state: {status}"
        )

        outputs = zynax_client.get_outputs(result.workflow_id)
        assert outputs.get("message") == "Hello from Zynax", (
            f"expected the declared 'message' output, got: {outputs!r}"
        )

    def test_orchestrator_executes_on_platform(self, zynax_client):
        """`zynax apply` of the orchestrator Workflow on a live platform yields
        an aggregated verdict + a decision-log entry (O8, #1103).

        This is a real e2e — the ``xfail`` marker is gone. The ``zynax_client``
        fixture SKIPS CLEANLY (not error, not a false xfail) unless a running
        platform is opted into via ``ZYNAX_PLATFORM_E2E=1`` (+ ``ZYNAX_GATEWAY_URL``),
        so the PR-level check is a clean skip and the assertion runs only in the
        gated e2e job against a real platform (Postgres + EventBus).
        """
        result = zynax_client.apply(str(ORCHESTRATOR_YAML))
        assert result.workflow_id is not None, "apply must return a run id"

        status = zynax_client.wait_for_completion(result.workflow_id, timeout=60)
        assert status in ("COMPLETED", "SUCCEEDED"), (
            f"orchestrator run did not reach a terminal success state: {status}"
        )

        outputs = zynax_client.get_outputs(result.workflow_id)

        # Outputs are a flat map<string,string> (ADR-042): complex values are JSON
        # strings the consumer parses — so the aggregated verdict is decoded here,
        # not indexed as a nested dict. The read path itself is gap #4, now closed
        # and covered by test_declared_output_read_path above. gaps #2 (CEL-vs-Go
        # -template guards) and #3 (missing capability providers) still keep this
        # orchestrator leg from running on the platform, so it stays gated (a clean
        # skip via the zynax_client fixture — never an XPASS that reddens the build).
        assert "aggregated_verdict" in outputs, (
            "run produced no aggregated_verdict output"
        )
        verdict = json.loads(outputs["aggregated_verdict"])
        assert verdict.get("confidence") in (
            "low",
            "medium",
            "high",
        ), f"unexpected verdict confidence: {verdict!r}"

        # Decision-log entry — recorded on every path by the terminal `done`
        # state (record_decision capability). The verdict alone is not enough;
        # readiness requires a durable decision record.
        decision_log = outputs.get("decision_log_artifact") or outputs.get(
            "decision_log"
        )
        assert decision_log, "run produced no decision-log entry"
