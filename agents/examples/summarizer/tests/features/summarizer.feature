# SPDX-License-Identifier: Apache-2.0
Feature: Document Summarisation Agent
  As an orchestrator
  I want to submit summarise tasks to the summariser agent
  So that I receive concise summaries without coupling to the AI runtime

  Background:
    Given the summariser agent is running with a FakeAgentContext

  Scenario: Execute task returns RESULT as the final event
    Given a task with capability "summarize" and documents ["The quick brown fox."]
    When the agent executes the task
    Then the last event type is RESULT
    And the result payload contains a non-empty "summary"

  Scenario: Progress events are emitted before the result
    Given a task with multiple documents
    When the agent executes the task
    Then at least one PROGRESS event is emitted before RESULT

  Scenario: Agent stores result in memory after summarisation
    Given a task with valid documents
    When the agent executes the task
    Then context.memory.set was called with the summary

  Scenario: Empty documents returns ERROR event
    Given a task with capability "summarize" and documents []
    When the agent executes the task
    Then the last event type is ERROR
    And the error message mentions "empty"

  Scenario: Runtime is swappable without changing this feature file
    Given the agent is configured with DirectRuntime instead of LangGraphRuntime
    When the same summarise task is executed
    Then the last event type is RESULT
