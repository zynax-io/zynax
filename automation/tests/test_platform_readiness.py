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
ORCHESTRATOR_YAML = REPO_ROOT / "automation/workflows/dev-advisory-orchestrator.yaml"
EXPERT_YAMLS = list((REPO_ROOT / "automation/workflows/experts").glob("*.yaml"))


class TestPlatformReadiness:
    @pytest.mark.xfail(
        strict=True,
        reason=(
            "Wave 4: requires automation/workflows/dev-advisory-orchestrator.yaml "
            "(EPIC #881 O3, #1098 — kind: Workflow per ADR-028). Reworked against "
            "workflow.schema.json when the manifest lands; flips in O8 (#1103)."
        ),
    )
    def test_orchestrator_agentdef_schema_valid(self):
        """Orchestrator manifest validates against Zynax JSON schema."""
        assert HAS_JSONSCHEMA, "jsonschema package required"
        with open(SCHEMA_PATH) as f:
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

    @pytest.mark.xfail(
        strict=True,
        reason=(
            "Wave 4: requires a running Zynax platform (Postgres #626 + EventBus "
            "#772) and the orchestrator workflow (O3, #1098). Flips to a real e2e "
            "in O8 (#1103)."
        ),
    )
    def test_orchestrator_executes_on_platform(self, zynax_client):
        """Orchestrator workflow loads and executes on a running Zynax platform."""
        result = zynax_client.apply(str(ORCHESTRATOR_YAML))
        assert result.workflow_id is not None
        status = zynax_client.wait_for_completion(result.workflow_id, timeout=60)
        assert status == "COMPLETED"
        outputs = zynax_client.get_outputs(result.workflow_id)
        assert "aggregated_verdict" in outputs
        assert outputs["aggregated_verdict"]["confidence"] in ("low", "medium", "high")
