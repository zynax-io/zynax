# SPDX-License-Identifier: Apache-2.0
"""Shared pytest fixtures for zynax-sdk BDD contract tests."""

import sys
import os
import types
from concurrent.futures import ThreadPoolExecutor

import grpc
import pytest

_tests_dir = os.path.dirname(__file__)
_proto_dir = os.path.join(_tests_dir, "../../../protos/generated/python")
sys.path.insert(0, _tests_dir)
sys.path.insert(0, _proto_dir)

from servers import AgentServiceImpl  # noqa: E402
from zynax.v1 import agent_pb2_grpc  # noqa: E402


@pytest.fixture(scope="module")
def grpc_channel():
    """In-process gRPC channel backed by AgentServiceImpl."""
    server = grpc.server(ThreadPoolExecutor(max_workers=4))
    agent_pb2_grpc.add_AgentServiceServicer_to_server(AgentServiceImpl(), server)
    port = server.add_insecure_port("127.0.0.1:0")
    server.start()
    channel = grpc.insecure_channel(f"127.0.0.1:{port}")
    yield channel
    channel.close()
    server.stop(grace=0)


@pytest.fixture
def ctx():
    """Per-test mutable state bag for BDD steps."""
    return types.SimpleNamespace()
