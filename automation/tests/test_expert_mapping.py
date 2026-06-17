# SPDX-License-Identifier: Apache-2.0
# automation/tests/test_expert_mapping.py
#
# EPIC X (#1170) step X.5 (#1205): the authoring <-> runtime expert mapping
# drift guard (ADR-033). Asserts the live repo reconciles and that each of the
# three drift-guard rules trips on a planted violation.

import importlib.util
from pathlib import Path

import pytest
import yaml

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "automation/scripts/check_expert_mapping.py"
MAPPING_PATH = REPO_ROOT / "automation/experts/runtime_mapping.yaml"

_spec = importlib.util.spec_from_file_location("check_expert_mapping", SCRIPT_PATH)
assert _spec is not None and _spec.loader is not None
check_mod = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(check_mod)


def test_live_repo_reconciles():
    """The committed mapping reconciles against every surface (no drift)."""
    assert check_mod.check() == []


def test_main_returns_zero_on_clean_repo():
    """The CLI entrypoint exits 0 when the mapping is consistent."""
    assert check_mod.main() == 0


def test_every_authoring_expert_is_declared():
    """Rule 1: each .claude authoring expert appears in the mapping file."""
    declared = {e["authoring"] for e in check_mod.load_mapping()}
    assert check_mod.discover_authoring_experts() <= declared


def test_runtime_references_resolve():
    """Rule 2: a named runtime_mapping points at agents/examples/<name>."""
    runtime_agents = check_mod.discover_runtime_agents()
    for entry in check_mod.load_mapping():
        rm = entry["runtime_mapping"]
        if rm and rm != check_mod.AUTHORING_ONLY:
            assert rm in runtime_agents, f"{entry['authoring']}: {rm} unresolved"


def test_mapping_matches_adr_table():
    """Rule 3: the mapping file equals ADR-033's human-readable table."""
    adr_table = check_mod.parse_adr_table()
    declared = {e["authoring"]: e["runtime_mapping"] for e in check_mod.load_mapping()}
    assert declared == adr_table


def test_missing_declaration_fails(monkeypatch):
    """Rule 1 trips when an authoring expert is dropped from the mapping."""
    monkeypatch.setattr(
        check_mod, "discover_authoring_experts", lambda: {"phantom-expert"}
    )
    errors = check_mod.check()
    assert any("phantom-expert" in e for e in errors)


def test_dangling_runtime_reference_fails(tmp_path, monkeypatch):
    """Rule 2 trips when runtime_mapping names a non-existent agent."""
    doc = yaml.safe_load(MAPPING_PATH.read_text())
    doc["experts"][0]["runtime_mapping"] = "does-not-exist"
    bad = tmp_path / "runtime_mapping.yaml"
    bad.write_text(yaml.safe_dump(doc))
    monkeypatch.setattr(check_mod, "MAPPING_PATH", bad)
    errors = check_mod.check()
    assert any("does-not-exist" in e for e in errors)


def test_adr_table_disagreement_fails(tmp_path, monkeypatch):
    """Rule 3 trips when the mapping diverges from the ADR-033 table."""
    doc = yaml.safe_load(MAPPING_PATH.read_text())
    doc["experts"][1]["runtime_mapping"] = "drifted-expert"
    bad = tmp_path / "runtime_mapping.yaml"
    bad.write_text(yaml.safe_dump(doc))
    monkeypatch.setattr(check_mod, "MAPPING_PATH", bad)
    monkeypatch.setattr(
        check_mod, "discover_runtime_agents", lambda: {"drifted-expert"}
    )
    errors = check_mod.check()
    assert any("ADR-033 table" in e for e in errors)


def test_empty_runtime_mapping_fails(tmp_path, monkeypatch):
    """Rule 1 trips when runtime_mapping is empty."""
    doc = yaml.safe_load(MAPPING_PATH.read_text())
    doc["experts"][0]["runtime_mapping"] = ""
    bad = tmp_path / "runtime_mapping.yaml"
    bad.write_text(yaml.safe_dump(doc))
    monkeypatch.setattr(check_mod, "MAPPING_PATH", bad)
    errors = check_mod.check()
    assert any("runtime_mapping" in e for e in errors)


@pytest.mark.parametrize("slug", sorted(check_mod.discover_authoring_experts()))
def test_each_authoring_expert_has_nonempty_mapping(slug):
    """Every authoring expert declares a non-empty runtime_mapping value."""
    declared = {e["authoring"]: e["runtime_mapping"] for e in check_mod.load_mapping()}
    assert declared.get(slug)
