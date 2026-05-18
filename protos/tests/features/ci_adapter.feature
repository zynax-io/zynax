# SPDX-License-Identifier: Apache-2.0
# Zynax — ci-adapter BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes the contract the ci-adapter must honour.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: The ci-adapter wraps CI pipeline operations (trigger workflow,
# poll run status) as Zynax capabilities. It is the boundary between the Zynax
# workflow engine and CI providers (GitHub Actions, Jenkins). (ADR-013)
#
# Canvas: docs/spdd/382-ci-adapter/canvas.md
# Parent epic: #382 (ci-adapter — CI pipeline trigger)

Feature: ci-adapter — CI pipeline trigger capability adapter
  As a platform operator
  I want a ci-adapter that wraps CI pipeline operations as Zynax capabilities
  So that workflow steps can trigger builds and poll run status
  without the control plane knowing which CI provider is in use

  Background:
    Given a ci-adapter configured for provider "github-actions-stub"
    And the adapter is registered with AgentRegistryService

  # ─── trigger_workflow ────────────────────────────────────────────────────────

  Scenario: trigger_workflow dispatches workflow_dispatch and returns run ID and URL
    Given a valid ExecuteCapabilityRequest for capability "trigger_workflow"
    And the input payload contains repository, workflow_id, and ref
    And the provider creates a run with ID "run-12345" and URL "https://github.example/actions/runs/12345"
    When ExecuteCapability is called
    Then the final TaskEvent has event_type COMPLETED
    And the COMPLETED payload contains "run_id" equal to "run-12345"
    And the COMPLETED payload contains a non-empty "run_url" field

  Scenario: trigger_workflow emits FAILED TIMEOUT when run ID does not appear within poll timeout
    Given a valid ExecuteCapabilityRequest for capability "trigger_workflow"
    And the provider does not create a run within trigger_poll_timeout_seconds
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "TIMEOUT"

  Scenario: trigger_workflow emits PROGRESS while waiting for run ID to appear
    Given a valid ExecuteCapabilityRequest for capability "trigger_workflow" with timeout_seconds 10
    And the provider creates a run after 2 poll cycles
    When ExecuteCapability is called
    Then the stream emits at least one TaskEvent with event_type PROGRESS before COMPLETED
    And every PROGRESS event has task_id echoed and timestamp populated

  # ─── get_run_status ─────────────────────────────────────────────────────────

  Scenario: get_run_status emits PROGRESS per poll cycle with run URL and current status
    Given a valid ExecuteCapabilityRequest for capability "get_run_status"
    And the input payload contains run_id and repository
    And the provider run is in state "in_progress" for 2 poll cycles then "completed"
    When ExecuteCapability is called
    Then the stream emits at least two TaskEvents with event_type PROGRESS
    And every PROGRESS payload contains "run_url" and "status"

  Scenario: get_run_status emits COMPLETED with conclusion on terminal run state
    Given a valid ExecuteCapabilityRequest for capability "get_run_status"
    And the provider run has concluded with conclusion "success"
    When ExecuteCapability is called
    Then the final TaskEvent has event_type COMPLETED
    And the COMPLETED payload contains "conclusion" equal to "success"

  Scenario: get_run_status emits FAILED when run concludes with failure
    Given a valid ExecuteCapabilityRequest for capability "get_run_status"
    And the provider run has concluded with conclusion "failure"
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "UPSTREAM_ERROR"

  # ─── Timeout ────────────────────────────────────────────────────────────────

  Scenario: timeout_seconds breach produces FAILED with TIMEOUT code
    Given the provider delays all responses beyond the timeout
    And an ExecuteCapabilityRequest for capability "get_run_status" with timeout_seconds 1
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "TIMEOUT"

  # ─── Rate-limit and auth error mapping ──────────────────────────────────────

  Scenario: GitHub API 429 produces FAILED with RESOURCE_EXHAUSTED
    Given the provider returns HTTP 429 for any request
    And a valid ExecuteCapabilityRequest for capability "trigger_workflow"
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "RESOURCE_EXHAUSTED"

  Scenario: GitHub API 403 produces FAILED with RESOURCE_EXHAUSTED
    Given the provider returns HTTP 403 for any request
    And a valid ExecuteCapabilityRequest for capability "trigger_workflow"
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "RESOURCE_EXHAUSTED"

  # ─── Capability routing ──────────────────────────────────────────────────────

  Scenario: Unknown capability returns NOT_FOUND without entering the stream
    Given an ExecuteCapabilityRequest for capability "nonexistent_op"
    When ExecuteCapability is called
    Then the gRPC status is NOT_FOUND
    And no TaskEvent is emitted

  # ─── Provider support gate ───────────────────────────────────────────────────

  Scenario: provider jenkins-stub produces FAILED with INTERNAL and "not implemented"
    Given a ci-adapter configured for provider "jenkins-stub"
    And a valid ExecuteCapabilityRequest for capability "trigger_workflow"
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "INTERNAL"
    And the CapabilityError message contains "not implemented"

  # ─── Security: no credential leak ───────────────────────────────────────────

  Scenario: Run URL is never constructed from input_payload fields
    Given the adapter is configured with a static workflow endpoint
    And an ExecuteCapabilityRequest with input_payload containing "run_url" pointing elsewhere
    When ExecuteCapability is called with capability "trigger_workflow"
    Then the adapter uses only the statically configured endpoint

  Scenario: Credential values never appear in CapabilityError message
    Given the provider returns an authentication error
    And the adapter is configured with a token containing "secret-ci-token"
    When ExecuteCapability is called with capability "trigger_workflow"
    Then the CapabilityError message does not contain "secret-ci-token"
