# SPDX-License-Identifier: Apache-2.0
# Task Broker service-level BDD — exercises real TaskService + memoryRepo via bufconn.
# Proto-contract shape is tested separately in protos/tests/task_broker_service/.

Feature: Task Broker service-level behaviour

  Background:
    Given the task broker service is running

  Scenario: Dispatch task creates task in PENDING state
    Given agent "agent-a" handles capability "summarize"
    And the repo holds updates until released
    When I dispatch a task with capability "summarize" for workflow "wf-bdd-01"
    Then the response contains a non-empty task_id
    And GetTask returns status PENDING

  Scenario: Acknowledge COMPLETED transitions task to COMPLETED
    Given a task "task-ack-01" in DISPATCHED state for workflow "wf-bdd-02"
    When AcknowledgeTask is called with status COMPLETED and a valid result for task "task-ack-01"
    Then GetTask for "task-ack-01" returns status COMPLETED

  Scenario: Acknowledge FAILED with retries remaining transitions task to RETRYING
    Given a task "task-retry-01" in DISPATCHED state for workflow "wf-bdd-03" with max_retries 2
    When AcknowledgeTask is called with status FAILED for task "task-retry-01"
    Then GetTask for "task-retry-01" returns status RETRYING

  Scenario: Cancel a PENDING task transitions it to CANCELLED
    Given a task "task-cancel-01" in PENDING state for workflow "wf-bdd-04"
    When CancelTask is called for task "task-cancel-01"
    Then GetTask for "task-cancel-01" returns status CANCELLED

  Scenario: GetTask for an unknown task_id returns NOT_FOUND
    When GetTask is called for task_id "no-such-task-bdd"
    Then the error code is NOT_FOUND
