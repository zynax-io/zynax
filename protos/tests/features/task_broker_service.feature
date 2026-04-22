# SPDX-License-Identifier: Apache-2.0
# Zynax — TaskBrokerService BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes the contract the task broker must honour.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: The task broker is the central dispatcher of the platform.
# It receives capability requests from workflow engine adapters, finds a
# matching agent via the registry, and tracks each task through its full
# lifecycle including retries and cancellation. Without this contract,
# the platform has no ability to coordinate work across agents. (ADR-001)

Feature: TaskBrokerService contract — capability routing and task lifecycle
  As a workflow engine adapter submitting work to the platform
  I want to dispatch capability requests and track their outcomes
  So that workflows can coordinate multi-agent work reliably

  Background:
    Given a TaskBrokerService is running on a test gRPC server
    And an AgentRegistryService is available to the broker

  # ─── Dispatch ─────────────────────────────────────────────────────────────

  Scenario: Dispatch a capability request returns a task_id
    Given agent "agent-summarizer" is registered with capability "summarize"
    And a valid WorkflowTask for capability "summarize" with valid input payload
    When DispatchTask is called with the WorkflowTask
    Then the response contains a non-empty task_id
    And GetTask for that task_id returns status PENDING

  Scenario: Broker routes task to the agent declaring the capability
    Given agent "agent-a" is registered with capability "summarize"
    And agent "agent-b" is registered with capability "review_code"
    When DispatchTask is called for capability "review_code"
    Then the task is routed to agent "agent-b"
    And agent "agent-a" receives no dispatch

  Scenario: Dispatched task records the originating workflow_id
    Given agent "agent-a" is registered with capability "summarize"
    And a WorkflowTask with workflow_id "wf-contract-01"
    When DispatchTask is called
    Then GetTask returns workflow_id "wf-contract-01"

  Scenario: No agent available for capability returns NOT_FOUND
    Given no agent is registered for capability "unknown_cap"
    When DispatchTask is called for capability "unknown_cap"
    Then the gRPC status is NOT_FOUND
    And the error message contains "unknown_cap"
    And no task record is created

  Scenario: Dispatch with timeout_seconds propagates to agent invocation
    Given agent "agent-a" is registered with capability "summarize"
    And a WorkflowTask with timeout_seconds set to 30
    When DispatchTask is called
    Then the agent receives an ExecuteCapabilityRequest with timeout_seconds 30

  # ─── Acknowledgement ──────────────────────────────────────────────────────

  Scenario: Acknowledge task completion stores result payload
    Given a dispatched task with task_id "task-99" in DISPATCHED state
    When AcknowledgeTask is called with task_id "task-99" status COMPLETED
    And the result payload is valid JSON: {"summary": "done"}
    Then GetTask for "task-99" returns status COMPLETED
    And GetTask for "task-99" returns the result payload

  Scenario: Acknowledge task failure without retry eligibility marks FAILED
    Given a dispatched task with task_id "task-88" and max_retries 0
    When AcknowledgeTask is called with task_id "task-88" status FAILED
    Then GetTask for "task-88" returns status FAILED
    And the error detail is stored on the task record

  Scenario: Acknowledge failure with retries remaining transitions to RETRYING
    Given a dispatched task with task_id "task-77" max_retries 2 retry_count 1
    When AcknowledgeTask is called with task_id "task-77" status FAILED
    Then GetTask for "task-77" returns status RETRYING
    And GetTask for "task-77" returns retry_count 2

  Scenario: Task exhausting all retries transitions to FAILED
    Given a dispatched task with task_id "task-66" max_retries 2 retry_count 2
    When AcknowledgeTask is called with task_id "task-66" status FAILED
    Then GetTask for "task-66" returns status FAILED

  Scenario: Acknowledge an unknown task_id returns NOT_FOUND
    When AcknowledgeTask is called with task_id "ghost-task" status COMPLETED
    Then the gRPC status is NOT_FOUND
    And the error message contains "ghost-task"

  # ─── Cancellation ─────────────────────────────────────────────────────────

  Scenario: Cancel a PENDING task transitions it to CANCELLED
    Given a dispatched task with task_id "task-55" in PENDING state
    When CancelTask is called with task_id "task-55"
    Then GetTask for "task-55" returns status CANCELLED

  Scenario: Cancel a DISPATCHED task transitions it to CANCELLED
    Given a dispatched task with task_id "task-44" in DISPATCHED state
    When CancelTask is called with task_id "task-44"
    Then GetTask for "task-44" returns status CANCELLED

  Scenario: Cancel a COMPLETED task returns FAILED_PRECONDITION
    Given a task with task_id "task-33" in COMPLETED state
    When CancelTask is called with task_id "task-33"
    Then the gRPC status is FAILED_PRECONDITION
    And the error message mentions "COMPLETED"

  Scenario: Cancel an unknown task_id returns NOT_FOUND
    When CancelTask is called with task_id "nonexistent-task"
    Then the gRPC status is NOT_FOUND

  # ─── Query ────────────────────────────────────────────────────────────────

  Scenario: GetTask returns full task record including timestamps
    Given a dispatched task with task_id "task-22"
    When GetTask is called with task_id "task-22"
    Then the response includes a non-zero dispatched_at timestamp
    And the response includes the original capability_name
    And the response includes the original workflow_id

  Scenario: ListTasks filters by workflow_id
    Given task "task-A" belongs to workflow "wf-001"
    And task "task-B" belongs to workflow "wf-002"
    When ListTasks is called with workflow_id filter "wf-001"
    Then the response contains task "task-A"
    And the response does not contain task "task-B"

  Scenario: ListTasks filters by status
    Given task "task-A" has status COMPLETED
    And task "task-B" has status FAILED
    When ListTasks is called with status filter COMPLETED
    Then the response contains task "task-A"
    And the response does not contain task "task-B"

  Scenario: GetTask for an unknown task_id returns NOT_FOUND
    When GetTask is called with task_id "nonexistent-task"
    Then the gRPC status is NOT_FOUND
    And the error message contains "nonexistent-task"

  # ─── Input validation ─────────────────────────────────────────────────────

  Scenario: DispatchTask with empty capability_name is rejected
    Given a WorkflowTask with capability_name set to ""
    When DispatchTask is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "capability_name"

  Scenario: DispatchTask with empty workflow_id is rejected
    Given a WorkflowTask with workflow_id set to ""
    When DispatchTask is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "workflow_id"

  Scenario: DispatchTask with non-JSON input_payload is rejected
    Given a WorkflowTask with input_payload set to "not valid json"
    When DispatchTask is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "input_payload"

  Scenario: AcknowledgeTask with empty task_id is rejected
    Given an AcknowledgeTaskRequest with task_id set to ""
    When AcknowledgeTask is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "task_id"

  Scenario: AcknowledgeTask with UNSPECIFIED status is rejected
    Given an AcknowledgeTaskRequest with status TASK_STATUS_UNSPECIFIED
    When AcknowledgeTask is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "status"

  Scenario: AcknowledgeTask COMPLETED without result payload is rejected
    Given an AcknowledgeTaskRequest with status COMPLETED and empty payload
    When AcknowledgeTask is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "result_payload"

  # ─── Pagination ───────────────────────────────────────────────────────────

  Scenario: ListTasks first page returns page_size results and a next_page_token
    Given 5 tasks exist in workflow "wf-paged"
    When ListTasks is called with page_size 3 and no page_token
    Then the response contains exactly 3 tasks
    And the response next_page_token is non-empty

  Scenario: ListTasks subsequent page returns remaining results
    Given 5 tasks exist in workflow "wf-paged"
    And ListTasks has been called with page_size 3 returning next_page_token "tok-1"
    When ListTasks is called with page_size 3 and page_token "tok-1"
    Then the response contains exactly 2 tasks
    And the response next_page_token is empty

  Scenario: ListTasks last page has empty next_page_token
    Given 2 tasks exist in workflow "wf-small"
    When ListTasks is called with page_size 10 and no page_token
    Then the response contains exactly 2 tasks
    And the response next_page_token is empty

  Scenario: ListTasks with page_size 0 uses server default
    Given 3 tasks exist in workflow "wf-default"
    When ListTasks is called with page_size 0 and no page_token
    Then the gRPC status is OK
    And the response contains at least 1 task
