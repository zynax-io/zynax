# SPDX-License-Identifier: Apache-2.0
Feature: Workflow Compilation

  Background:
    Given the workflow compiler is running

  Scenario: Valid workflow YAML compiles successfully
    Given a valid Workflow YAML with states [review, fix, merge, done]
    When ApplyWorkflow is called
    Then the gRPC status is OK
    And the compiled IR has 4 states
    And the initial_state is "review"
    And state "done" has type TERMINAL

  Scenario: YAML with unknown capability is rejected
    Given a Workflow YAML with action capability "nonexistent_cap"
    And "nonexistent_cap" is not registered in agent-registry
    When ApplyWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error mentions "unknown capability"

  Scenario: YAML with no terminal state is rejected
    Given a Workflow YAML where no state has type: terminal
    When ApplyWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error mentions "terminal state"

  Scenario: Orphan unreachable state is rejected
    Given a Workflow YAML with state "orphan" that no transition points to
    When ApplyWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error mentions "unreachable state"

  Scenario: ApplyWorkflow is idempotent
    Given workflow "test-workflow" has been applied
    When the same YAML is applied again with identical content
    Then the gRPC status is OK
    And no duplicate workflow record exists

  Scenario: DryRun compiles without executing or querying registry
    Given a valid Workflow YAML
    When DryRun is called
    Then the response contains a compiled IR
    And no workflow execution is started
    And agent-registry receives no requests
