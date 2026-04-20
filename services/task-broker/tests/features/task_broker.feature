# SPDX-License-Identifier: Apache-2.0
# Task Broker BDD Feature — written before implementation
# Step definitions: tests/unit/task_broker_test.go (godog)

Feature: Task Scheduling and Assignment

  Background:
    Given the task broker is running

  Scenario: Submit task returns task_id immediately
    When a task with capability "summarize" and priority NORMAL is submitted
    Then the response contains a valid task_id
    And the task state is PENDING

  Scenario: Task assigned to eligible agent
    Given agent "a1" is ACTIVE with capability "summarize"
    And a PENDING task with capability "summarize" exists
    When the broker runs an assignment cycle
    Then the task state becomes ASSIGNED
    And the assigned_agent_id is "a1"

  Scenario: Task stays PENDING when no eligible agent
    Given no ACTIVE agents with capability "rare-skill" exist
    When a task with capability "rare-skill" is submitted
    Then after the assignment cycle the task state is still PENDING
    And no error is returned

  Scenario: Failed task is retried
    Given a task with max_retries=3 is FAILED with attempt_count=1
    When the retry cycle runs
    Then the task state becomes PENDING again

  Scenario: Task permanently FAILED after exhausting retries
    Given a task with max_retries=3 is FAILED with attempt_count=3
    When the task is marked FAILED again
    Then the task state is FAILED permanently
    And no further retry is scheduled

  Scenario: WatchTask delivers state transitions in order
    Given a client watches task "t-123"
    When the task transitions PENDING→ASSIGNED→RUNNING→SUCCEEDED
    Then the stream delivers those 4 events in order

  Scenario: High priority task assigned before low priority
    Given LOW priority task T1 and HIGH priority task T2 are both PENDING with capability "write"
    When one assignment cycle runs
    Then T2 is assigned and T1 remains PENDING
