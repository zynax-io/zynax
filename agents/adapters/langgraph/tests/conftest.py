# SPDX-License-Identifier: Apache-2.0
"""Pytest configuration — adds generated proto stubs to sys.path."""

import os
import sys

_PROTO_PYTHON = os.path.abspath(
    os.path.join(os.path.dirname(__file__), "../../../../protos/generated/python")
)
if _PROTO_PYTHON not in sys.path:
    sys.path.insert(0, _PROTO_PYTHON)

# Pre-register the well-known timestamp descriptor required by agent_registry_pb2.
import google.protobuf.timestamp_pb2  # noqa: E402, F401
