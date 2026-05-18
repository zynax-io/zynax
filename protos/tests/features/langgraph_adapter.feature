# SPDX-License-Identifier: Apache-2.0
# Zynax — langgraph-adapter BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes the contract the langgraph-adapter must honour.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: The langgraph-adapter wraps compiled LangGraph state-machine
# graphs as Zynax capabilities. Each mapped capability corresponds to one graph.
# The adapter streams per-node PROGRESS events then a terminal COMPLETED or FAILED
# event. No code change to the graph is required — mapping is config-only. (ADR-013)
#
# Canvas: docs/spdd/384-langgraph-adapter/canvas.md
# Parent epic: #384 (LangGraph Adapter)

Feature: langgraph-adapter — LangGraph state-machine capability adapter
  As a platform operator
  I want a langgraph-adapter that maps compiled LangGraph graphs to Zynax capabilities
  So that workflow steps can invoke complex multi-node AI pipelines
  without the control plane knowing the graph implementation details

  Background:
    Given a langgraph-adapter configured with a capability "run_graph"
    And the capability maps to a registered LangGraph graph "stub_graph"
    And the adapter is registered with AgentRegistryService

  # ─── Happy path — per-node streaming ────────────────────────────────────────

  Scenario: Mapped capability streams PROGRESS per node then COMPLETED with final state
    Given a valid ExecuteCapabilityRequest for capability "run_graph"
    And the graph "stub_graph" has nodes: "node_a", "node_b", "node_c"
    When ExecuteCapability is called
    Then the stream emits a TaskEvent with event_type PROGRESS for each node that fires
    And the PROGRESS events appear in graph execution order
    And the final TaskEvent has event_type COMPLETED
    And the COMPLETED payload contains the serialised final graph state

  Scenario: PROGRESS events include node name and timestamp
    Given a valid ExecuteCapabilityRequest for capability "run_graph"
    When ExecuteCapability is called
    Then every PROGRESS event has task_id echoed and timestamp populated
    And each PROGRESS payload contains a non-empty "node" field

  Scenario: Ticker PROGRESS is emitted when no node fires within 2 seconds
    Given the graph "stub_graph" has a node that takes longer than 2 seconds
    And a valid ExecuteCapabilityRequest for capability "run_graph" with timeout_seconds 30
    When ExecuteCapability is called
    Then the stream emits a PROGRESS event with a heartbeat indicator within 3 seconds

  # ─── Graph state serialisation ───────────────────────────────────────────────

  Scenario: Final graph state is serialised with json.dumps default=str fallback
    Given the graph "stub_graph" produces state containing a non-JSON-serialisable datetime
    And a valid ExecuteCapabilityRequest for capability "run_graph"
    When ExecuteCapability is called
    Then the final TaskEvent has event_type COMPLETED
    And the COMPLETED payload is valid JSON

  # ─── Timeout ────────────────────────────────────────────────────────────────

  Scenario: timeout_seconds exceeded emits FAILED with TIMEOUT code
    Given the graph "stub_graph" runs longer than the timeout
    And an ExecuteCapabilityRequest for capability "run_graph" with timeout_seconds 1
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "TIMEOUT"

  # ─── Input validation ────────────────────────────────────────────────────────

  Scenario: Input missing a required field emits FAILED with INVALID_INPUT
    Given the capability "run_graph" declares a required input field "query"
    And an ExecuteCapabilityRequest with input_payload missing the "query" field
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "INVALID_INPUT"

  Scenario: Input with wrong field type emits FAILED with INVALID_INPUT
    Given the capability "run_graph" declares "query" as type string
    And an ExecuteCapabilityRequest with input_payload setting "query" to a number
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "INVALID_INPUT"

  # ─── Graph exception handling ────────────────────────────────────────────────

  Scenario: Graph exception during execution produces FAILED with sanitised message
    Given the graph "stub_graph" raises an exception during node execution
    And a valid ExecuteCapabilityRequest for capability "run_graph"
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "INTERNAL"
    And the CapabilityError message does not contain a stack trace
    And the CapabilityError message does not contain internal file paths

  # ─── Schema introspection ────────────────────────────────────────────────────

  Scenario: GetCapabilitySchema returns the declared schema for the registered capability
    When GetCapabilitySchema is called with capability_name "run_graph"
    Then the gRPC status is OK
    And the response input_schema_json matches the schema declared in the adapter config
    And the response output_schema_json is non-empty valid JSON
    And the response description is non-empty

  # ─── Capability routing ──────────────────────────────────────────────────────

  Scenario: Unknown capability name returns NOT_FOUND
    Given an ExecuteCapabilityRequest for capability "nonexistent_graph"
    When ExecuteCapability is called
    Then the gRPC status is NOT_FOUND
    And no TaskEvent is emitted
    And the error message contains "nonexistent_graph"

  # ─── Startup validation ──────────────────────────────────────────────────────

  Scenario: Adapter fails to start if any graph fails to import
    Given the adapter config references graph module "missing.module"
    When the adapter initialises
    Then the adapter exits with a non-zero status
    And the error message identifies "missing.module" as the cause

  Scenario: Adapter fails to start if any graph raises a compile-time error
    Given the adapter config references a graph module that raises an ImportError
    When the adapter initialises
    Then the adapter exits with a non-zero status
