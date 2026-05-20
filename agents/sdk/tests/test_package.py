# SPDX-License-Identifier: Apache-2.0
# Smoke test — verifies the package is importable and version is set.
# Unit tests for Agent base class and capability routing live in test_agent.py.
import zynax_sdk


def test_package_version_is_set() -> None:
    assert zynax_sdk.__version__ == "0.1.0"
