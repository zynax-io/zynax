# SPDX-License-Identifier: Apache-2.0
Feature: Workflow Engine Adapter

  Background:
    Given the Temporal engine is available

  Scenario: Submit IR creates a running execution
    Given a compiled WorkflowIR with initial_state "review"
    When Submit is called
    Then a valid ExecutionID is returned
    And Query returns status RUNNING

  Scenario: Signal transitions workflow to next state
    Given a running execution in state "review"
    When Signal is sent with event "review.approved"
    Then Query returns the state after "review.approved" transition

  Scenario: Capability action dispatches to task-broker
    Given a running execution that reaches an action: capability "summarize"
    When the engine executes that action
    Then a SubmitTask call is made to task-broker with capability "summarize"

  Scenario: Cancel terminates a running execution
    Given a running execution
    When Cancel is called with reason "user requested"
    Then Query returns status CANCELLED

  Scenario: Engine is swappable without API changes
    Given KEEL_ENGINE_ACTIVE_ENGINE=langgraph
    When Submit is called with a WorkflowIR
    Then the LangGraph engine handles execution
    And the gRPC response format is identical to the Temporal response
