#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Validate state/milestone.yaml against state/milestone.schema.json.

Run via `make validate-milestone-state` (containerized — see Makefile).
state/milestone.yaml is updated only by /milestone-close and /milestone-new,
never by hand; this validator is the CI gate for those commands' output.
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

STATE_FILE = Path("state/milestone.yaml")
SCHEMA_FILE = Path("state/milestone.schema.json")


def main() -> int:
    data = yaml.safe_load(STATE_FILE.read_text(encoding="utf-8"))
    schema = json.loads(SCHEMA_FILE.read_text(encoding="utf-8"))

    Draft202012Validator.check_schema(schema)
    validator = Draft202012Validator(schema)
    errors = sorted(validator.iter_errors(data), key=lambda e: list(e.absolute_path))

    if errors:
        for err in errors:
            path = "/".join(str(p) for p in err.absolute_path) or "<root>"
            print(f"FAIL {STATE_FILE}: {path}: {err.message}")
        return 1

    print(f"OK {STATE_FILE} conforms to {SCHEMA_FILE}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
