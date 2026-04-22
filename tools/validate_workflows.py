#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
# Validates all Workflow YAML manifests in a directory against
# spec/schemas/workflow.schema.json.
# Usage: python tools/validate_workflows.py <schema-path> <yaml-dir>
import json
import sys
from pathlib import Path

import jsonschema
import yaml


def _fix_bool_keys(obj):
    # PyYAML uses YAML 1.1 which parses 'on'/'off' as booleans True/False.
    # Workflow manifests use 'on:' as a transition list key — convert back.
    if isinstance(obj, dict):
        return {
            ("on" if k is True else "off" if k is False else k): _fix_bool_keys(v)
            for k, v in obj.items()
        }
    if isinstance(obj, list):
        return [_fix_bool_keys(i) for i in obj]
    return obj


def main() -> None:
    if len(sys.argv) != 3:
        print("Usage: validate_workflows.py <schema.json> <yaml-dir>")
        sys.exit(1)

    schema_path = Path(sys.argv[1])
    yaml_dir = Path(sys.argv[2])

    schema = json.loads(schema_path.read_text())
    validator = jsonschema.Draft202012Validator(schema)

    errors_found = False
    validated = 0
    for yaml_file in sorted(yaml_dir.glob("*.yaml")):
        doc = yaml.safe_load(yaml_file.read_text())
        if not isinstance(doc, dict) or doc.get("kind") != "Workflow":
            continue
        doc = _fix_bool_keys(doc)
        errs = sorted(validator.iter_errors(doc), key=lambda e: str(e.path))
        if errs:
            errors_found = True
            print(f"FAIL {yaml_file.name}:")
            for e in errs:
                print(f"  {e.json_path}: {e.message}")
        else:
            print(f"  OK  {yaml_file.name}")
            validated += 1

    if not errors_found and validated == 0:
        print(f"  (no Workflow manifests found in {yaml_dir})")

    if errors_found:
        sys.exit(1)


if __name__ == "__main__":
    main()
