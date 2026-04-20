# SPDX-License-Identifier: Apache-2.0
# Zynax — Proto Stub Generation Pipeline BDD Contract
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes what the buf generate pipeline must produce and how it
# integrates with CI to prevent stale stubs from reaching main.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: All platform services (Go) and agents/adapters (Python)
# depend on generated proto stubs. Without a reliable pipeline every
# contributor regenerates stubs differently, producing drift between Go and
# Python clients and breaking inter-service compatibility. (ADR-003)

Feature: Proto stub generation pipeline
  As a contributor
  I want to run a single make target to regenerate all stubs
  So that Go and Python stubs are always in sync with the proto sources

  Background:
    Given the file "protos/buf.gen.yaml" exists
    And the file "protos/buf.yaml" exists

  # ─── buf.gen.yaml configuration ───────────────────────────────────────────

  Scenario: buf.gen.yaml specifies Go stub output path
    When "protos/buf.gen.yaml" is parsed
    Then it contains a plugin entry for "protoc-gen-go"
    And the Go output path is "generated/go"

  Scenario: buf.gen.yaml specifies Go gRPC stub output path
    When "protos/buf.gen.yaml" is parsed
    Then it contains a plugin entry for "protoc-gen-go-grpc"
    And the gRPC output path is "generated/go"

  Scenario: buf.gen.yaml specifies Python stub output path
    When "protos/buf.gen.yaml" is parsed
    Then it contains a plugin entry for Python protobuf generation
    And the Python output path is "generated/python"

  Scenario: buf.gen.yaml specifies Python gRPC stub output path
    When "protos/buf.gen.yaml" is parsed
    Then it contains a plugin entry for Python gRPC generation
    And the Python gRPC output path is "generated/python"

  # ─── Generated Go stubs ───────────────────────────────────────────────────

  Scenario: make generate-protos produces Go stubs for all proto files
    Given all .proto files in protos/zynax/v1/ are present
    When make generate-protos is run inside the dev Docker image
    Then Go stub files are written under "protos/generated/go/"
    And every proto file has a corresponding "_pb.go" stub file
    And every service proto has a corresponding "_grpc.pb.go" stub file

  Scenario: Generated Go stubs declare the correct Go package
    Given make generate-protos has been run
    When the Go stubs in "protos/generated/go/" are inspected
    Then they declare package "zynaxv1"
    And they import "google.golang.org/grpc"

  Scenario: Generated Go stubs include all service client interfaces
    Given make generate-protos has been run
    When "protos/generated/go/zynax/v1/agent_grpc.pb.go" is inspected
    Then it contains the "AgentServiceClient" interface
    And it contains the "AgentServiceServer" interface

  # ─── Generated Python stubs ───────────────────────────────────────────────

  Scenario: make generate-protos produces Python stubs for all proto files
    Given all .proto files in protos/zynax/v1/ are present
    When make generate-protos is run inside the dev Docker image
    Then Python stub files are written under "protos/generated/python/"
    And every proto file has a corresponding "_pb2.py" stub file
    And every service proto has a corresponding "_pb2_grpc.py" stub file

  Scenario: Generated Python stubs are importable
    Given make generate-protos has been run
    When the Python stubs in "protos/generated/python/" are imported
    Then no ImportError is raised

  Scenario: Generated Python stubs include service stub classes
    Given make generate-protos has been run
    When "protos/generated/python/zynax/v1/agent_pb2_grpc.py" is inspected
    Then it contains the "AgentServiceStub" class
    And it contains the "AgentServiceServicer" class

  # ─── Makefile targets ─────────────────────────────────────────────────────

  Scenario: make generate-protos runs buf generate inside Docker
    Given a Makefile exists at the repository root
    When the "generate-protos" target is inspected
    Then it invokes "buf generate" with "protos/buf.gen.yaml"
    And it runs inside the keel-tools Docker image

  Scenario: make lint-protos runs buf lint inside Docker
    Given a Makefile exists at the repository root
    When the "lint-protos" target is inspected
    Then it invokes "buf lint"
    And it is included as a dependency of the "lint" target

  Scenario: Stubs are regenerated cleanly after a proto change
    Given an existing set of generated stubs
    When a field is added to a proto message and make generate-protos is run
    Then the new field appears in both Go and Python stubs
    And no stale generated files remain from the previous run

  # ─── CI freshness gate ────────────────────────────────────────────────────

  Scenario: CI fails if proto files changed but stubs were not regenerated
    Given a pull request that modifies a .proto file
    And make generate-protos was NOT run before committing
    When the "proto-stubs-fresh" CI check runs
    Then the check fails with a message to run "make generate-protos"

  Scenario: CI passes if proto files changed and stubs were regenerated
    Given a pull request that modifies a .proto file
    And make generate-protos was run and the updated stubs were committed
    When the "proto-stubs-fresh" CI check runs
    Then the check passes

  Scenario: CI passes if no proto files changed
    Given a pull request that does not modify any .proto files
    When the "proto-stubs-fresh" CI check runs
    Then the check passes without inspecting stubs

  # ─── buf lint integration ─────────────────────────────────────────────────

  Scenario: buf lint passes on all existing proto files
    Given all .proto files in protos/zynax/v1/ are present
    When buf lint is run from the protos/ directory
    Then it reports zero errors

  Scenario: buf format check passes on all existing proto files
    Given all .proto files in protos/zynax/v1/ are present
    When buf format --diff --exit-code is run from the protos/ directory
    Then it reports no formatting differences

  # ─── Committed generated stubs (#30) ──────────────────────────────────────
  # These scenarios verify the initial generated stubs are committed to the
  # repo, activating the proto-stubs-fresh freshness gate in CI.

  Scenario: Go stubs are committed for every proto service
    Given the repository contains protos/generated/go/zynax/v1/
    Then it contains "agent.pb.go" and "agent_grpc.pb.go"
    And it contains "agent_registry.pb.go" and "agent_registry_grpc.pb.go"
    And it contains "task_broker.pb.go" and "task_broker_grpc.pb.go"
    And it contains "workflow_compiler.pb.go" and "workflow_compiler_grpc.pb.go"
    And it contains "engine_adapter.pb.go" and "engine_adapter_grpc.pb.go"
    And it contains "memory.pb.go" and "memory_grpc.pb.go"
    And it contains "event_bus.pb.go" and "event_bus_grpc.pb.go"
    And it contains "cloudevents.pb.go"

  Scenario: Python stubs are committed for every proto service
    Given the repository contains protos/generated/python/zynax/v1/
    Then it contains "agent_pb2.py" and "agent_pb2_grpc.py"
    And it contains "agent_registry_pb2.py" and "agent_registry_pb2_grpc.py"
    And it contains "task_broker_pb2.py" and "task_broker_pb2_grpc.py"
    And it contains "workflow_compiler_pb2.py" and "workflow_compiler_pb2_grpc.py"
    And it contains "engine_adapter_pb2.py" and "engine_adapter_pb2_grpc.py"
    And it contains "memory_pb2.py" and "memory_pb2_grpc.py"
    And it contains "event_bus_pb2.py" and "event_bus_pb2_grpc.py"
    And it contains "cloudevents_pb2.py" and "cloudevents_pb2_grpc.py"

  Scenario: Generated stubs are not excluded by .gitignore
    Given the file ".gitignore" at the repository root
    When the gitignore rules are inspected
    Then "protos/generated/" is not excluded
    And a comment confirms the stubs are intentionally committed
