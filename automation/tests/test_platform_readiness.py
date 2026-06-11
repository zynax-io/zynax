# automation/tests/test_platform_readiness.py
import json
import yaml
import pytest
from pathlib import Path

try:
    import jsonschema

    HAS_JSONSCHEMA = True
except ImportError:
    HAS_JSONSCHEMA = False

SCHEMA_PATH = "spec/schemas/agent-def.schema.json"
ORCHESTRATOR_YAML = "automation/workflows/dev-advisory-orchestrator.yaml"
EXPERT_YAMLS = list(Path("automation/workflows/experts/").glob("*.yaml"))


@pytest.mark.xfail(
    strict=True,
    reason=(
        "Wave 4 aspirational: requires M6.H (Postgres-backed repos, #626) + "
        "M6.I (event-bus implementation, #772) + automation/workflows/ AgentDef "
        "YAMLs to exist. Fails until all three prerequisites land."
    ),
)
class TestPlatformReadiness:
    def test_orchestrator_agentdef_schema_valid(self):
        """AgentDef YAML validates against Zynax JSON schema."""
        assert HAS_JSONSCHEMA, "jsonschema package required"
        with open(SCHEMA_PATH) as f:
            schema = json.load(f)
        with open(ORCHESTRATOR_YAML) as f:
            doc = yaml.safe_load(f)
        jsonschema.validate(doc, schema)

    def test_expert_agentdefs_schema_valid(self):
        """All 9 expert AgentDef YAMLs validate against schema."""
        assert HAS_JSONSCHEMA, "jsonschema package required"
        assert len(EXPERT_YAMLS) == 9, f"Expected 9 experts, got {len(EXPERT_YAMLS)}"
        with open(SCHEMA_PATH) as f:
            schema = json.load(f)
        for p in EXPERT_YAMLS:
            with open(p) as f:
                doc = yaml.safe_load(f)
            jsonschema.validate(doc, schema)

    def test_orchestrator_executes_on_platform(self, zynax_client):
        """Orchestrator AgentDef loads and executes on a running Zynax platform."""
        result = zynax_client.apply(ORCHESTRATOR_YAML)
        assert result.workflow_id is not None
        status = zynax_client.wait_for_completion(result.workflow_id, timeout=60)
        assert status == "COMPLETED"
        outputs = zynax_client.get_outputs(result.workflow_id)
        assert "aggregated_verdict" in outputs
        assert outputs["aggregated_verdict"]["confidence"] in ("low", "medium", "high")
