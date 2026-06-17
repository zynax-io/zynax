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

        # Aggregated verdict — the weighted-consensus output of the `aggregate`
        # state (dev-advisory-orchestrator.yaml).
        assert "aggregated_verdict" in outputs, (
            "run produced no aggregated_verdict output"
        )
        assert outputs["aggregated_verdict"].get("confidence") in (
            "low",
            "medium",
            "high",
        ), f"unexpected verdict confidence: {outputs['aggregated_verdict']!r}"

        # Decision-log entry — recorded on every path by the terminal `done`
        # state (record_decision capability). The verdict alone is not enough;
        # readiness requires a durable decision record.
        decision_log = outputs.get("decision_log_artifact") or outputs.get(
            "decision_log"
        )
        assert decision_log, "run produced no decision-log entry"
