# SPDX-License-Identifier: Apache-2.0
"""Pytest configuration — adds generated proto stubs to sys.path."""

import importlib
import os
import sys

_PROTO_PYTHON = os.path.abspath(
    os.path.join(os.path.dirname(__file__), "../../../../protos/generated/python")
)
if _PROTO_PYTHON not in sys.path:
    sys.path.insert(0, _PROTO_PYTHON)

# Register google/protobuf/timestamp.proto in the default descriptor pool before the
# generated zynax stubs (which depend on it) are imported by the test modules. A
# dynamic import performs this side-effect-only registration without leaving an
# unused import or variable for static analysis to flag.
importlib.import_module("google.protobuf.timestamp_pb2")
