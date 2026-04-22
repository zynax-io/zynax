#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
# Validates that all JSON Schema files in spec/schemas/ are well-formed JSON
# and carry the required $schema field.
# Usage: python tools/validate_json_schemas.py <glob-or-dir>
#        python tools/validate_json_schemas.py spec/schemas/
import json
import sys
from pathlib import Path


def main() -> None:
    if len(sys.argv) < 2:
        print("Usage: validate_json_schemas.py <schema-dir>")
        sys.exit(1)

    schema_dir = Path(sys.argv[1])
    schema_files = sorted(schema_dir.glob("*.json")) if schema_dir.is_dir() else [schema_dir]

    errors_found = False
    for schema_file in schema_files:
        print(f"── json parse: {schema_file}")
        try:
            data = json.loads(schema_file.read_text())
            if "$schema" not in data:
                raise ValueError("missing $schema field")
            print("  ok")
        except Exception as e:
            print(f"  FAIL: {e}")
            errors_found = True

    if errors_found:
        sys.exit(1)
    print("✅ JSON Schemas are well-formed")


if __name__ == "__main__":
    main()
