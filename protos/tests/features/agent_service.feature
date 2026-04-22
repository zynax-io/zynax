# SPDX-License-Identifier: Apache-2.0
# Zynax — AgentService BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes the contract every adapter and agent in Zynax must honour.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: Any system that can serve ExecuteCapability becomes a
# first-class Zynax capability without adopting the SDK (ADR-013). This
# contract is the only gRPC interface required to join the platform.

Feature: AgentService contract — ExecuteCapability streaming RPC
  As a task broker dispatching work to capability providers
  I want every agent and adapter to implement a single ExecuteCapability contract
  So that any system can become a platform capability without knowing Zynax internals

  Background:
    Given an agent implementing AgentService is running on a test gRPC server

  # ─── Happy path ────────────────────────────────────────────────────────────

  Scenario: Successful capability execution streams progress then completes
    Given a valid ExecuteCapabilityRequest for capability "summarize"
    And the input payload is valid JSON: {"documents": ["hello world"]}
    When ExecuteCapability is called
    Then the stream emits at least one TaskEvent with event_type PROGRESS
    And the final TaskEvent has event_type COMPLETED
    And the COMPLETED event payload is valid JSON
    And the stream closes cleanly after the COMPLETED event

  Scenario: Every event in the stream carries the originating task_id
    Given a valid ExecuteCapabilityRequest with task_id "task-contract-99"
    When ExecuteCapability is called and the stream is fully consumed
    Then every TaskEvent in the stream has task_id "task-contract-99"

  Scenario: Every PROGRESS and COMPLETED event has a populated timestamp
    Given a valid ExecuteCapabilityRequest for capability "summarize"
    When ExecuteCapability is called
    Then every TaskEvent has a non-zero timestamp

  # ─── Timeout handling ─────────────────────────────────────────────────────

  Scenario: Agent honours timeout_seconds and emits FAILED with TIMEOUT code
    Given an ExecuteCapabilityRequest with timeout_seconds set to 1
    And the agent simulates a capability that runs for 5 seconds
    When ExecuteCapability is called
    Then the stream receives a TaskEvent of type FAILED within 2 seconds
    And the CapabilityError code is "TIMEOUT"
    And the gRPC status is DEADLINE_EXCEEDED

  # ─── Failure paths ────────────────────────────────────────────────────────

  Scenario: Capability failure produces a structured CapabilityError
    Given an ExecuteCapabilityRequest for capability "always_fails"
    When ExecuteCapability is called
    Then the stream emits exactly one TaskEvent with event_type FAILED
    And the TaskEvent.error.code is a non-empty string
    And the TaskEvent.error.message is a non-empty string
    And no further events are emitted after the FAILED event

  Scenario: Unknown capability returns NOT_FOUND without entering the stream
    Given an ExecuteCapabilityRequest for capability "nonexistent_cap"
    When ExecuteCapability is called
    Then the gRPC status is NOT_FOUND
    And the error message contains "nonexistent_cap"
    And no TaskEvent is emitted

  # ─── Input validation ─────────────────────────────────────────────────────

  Scenario: Empty capability_name is rejected before execution begins
    Given an ExecuteCapabilityRequest with capability_name set to ""
    When ExecuteCapability is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "capability_name"

  Scenario: Empty task_id is rejected before execution begins
    Given an ExecuteCapabilityRequest with task_id set to ""
    When ExecuteCapability is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "task_id"

  Scenario: Non-JSON input_payload is rejected before execution begins
    Given an ExecuteCapabilityRequest with input_payload set to "not valid json"
    When ExecuteCapability is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "input_payload"

  # ─── Stream ordering invariants ───────────────────────────────────────────

  Scenario: Stream always terminates with COMPLETED or FAILED — never with PROGRESS
    Given a valid ExecuteCapabilityRequest
    When ExecuteCapability is called and the stream is fully consumed
    Then the final TaskEvent has event_type COMPLETED or FAILED

  Scenario: No events are emitted after a terminal event
    Given a valid ExecuteCapabilityRequest for capability "always_fails"
    When ExecuteCapability is called and the stream is fully consumed
    Then no TaskEvent is received after the first FAILED event

  # ─── GetCapabilitySchema ──────────────────────────────────────────────────

  Scenario: GetCapabilitySchema returns schema for a known capability
    When GetCapabilitySchema is called with capability_name "summarize"
    Then the gRPC status is OK
    And the response capability_name is "summarize"
    And the response input_schema_json is valid JSON
    And the response output_schema_json is valid JSON
    And the response description is non-empty

  Scenario: GetCapabilitySchema returns NOT_FOUND for an unknown capability
    When GetCapabilitySchema is called with capability_name "nonexistent_cap"
    Then the gRPC status is NOT_FOUND
    And the error message contains "nonexistent_cap"

  Scenario: GetCapabilitySchema with empty capability_name is rejected
    Given a GetCapabilitySchemaRequest with capability_name set to ""
    When GetCapabilitySchema is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "capability_name"
