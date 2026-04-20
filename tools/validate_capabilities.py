#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
# Validates all capability declarations in AgentDef YAML files against
# spec/schemas/capability.schema.json.
# Usage: python tools/validate_capabilities.py <schema-path> <yaml-dir>
import json
import sys
from pathlib import Path

import jsonschema
import yaml


def main() -> None:
    if len(sys.argv) != 3:
        print("Usage: validate_capabilities.py <schema.json> <yaml-dir>")
        sys.exit(1)

    schema_path = Path(sys.argv[1])
    yaml_dir = Path(sys.argv[2])

    schema = json.loads(schema_path.read_text())
    validator = jsonschema.Draft202012Validator(schema)

    errors_found = False
    for yaml_file in sorted(yaml_dir.glob("*.yaml")):
        doc = yaml.safe_load(yaml_file.read_text())
        capabilities = doc.get("capabilities", [])
        if not capabilities:
            continue
        for cap in capabilities:
            errs = sorted(validator.iter_errors(cap), key=lambda e: str(e.path))
            if errs:
                errors_found = True
                print(f"FAIL {yaml_file.name} capability '{cap.get('name', '?')}':")
                for e in errs:
                    print(f"  {e.json_path}: {e.message}")
            else:
                print(f"  OK  {yaml_file.name} :: {cap.get('name', '?')}")

    if errors_found:
        sys.exit(1)


if __name__ == "__main__":
    main()
