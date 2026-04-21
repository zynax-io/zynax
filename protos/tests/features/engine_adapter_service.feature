# SPDX-License-Identifier: Apache-2.0
# Zynax — EngineAdapterService BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes the contract every engine adapter must honour.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: The engine adapter is the boundary between the Zynax
# control plane and execution engines (Temporal, LangGraph, Argo Workflows).
# The control plane submits compiled IR and signals events without knowing
# which engine is running underneath. This single contract is what makes
# the engine layer swappable. (ADR-001, ADR-015)
#
# Tags:
#   @lifecycle — workflow submission, status query, cancellation (TestLifecycle)
#   @signals   — signal delivery and watch stream (TestSignals)

Feature: EngineAdapterService contract — workflow execution lifecycle
  As a Zynax control plane submitting compiled workflows for execution
  I want every execution engine to implement a single adapter contract
  So that workflows run on any engine without the control plane knowing its internals

  Background:
    Given an EngineAdapterService is running on a test gRPC server

  # ─── Submission ───────────────────────────────────────────────────────────

  @lifecycle
  Scenario: Submit a workflow IR returns a run_id
    Given a compiled WorkflowIR for workflow "code-review-wf"
    When SubmitWorkflow is called with the IR
    Then the gRPC status is OK
    And the response contains a non-empty run_id
    And GetWorkflowStatus for that run_id returns status RUNNING

  @lifecycle
  Scenario: Submitted workflow records the originating namespace
    Given a compiled WorkflowIR with namespace "team-alpha"
    When SubmitWorkflow is called
    Then GetWorkflowStatus returns namespace "team-alpha"

  @lifecycle
  Scenario: Submit with labels preserves them on the run record
    Given a WorkflowIR and SubmitWorkflowRequest labels {"env": "staging"}
    When SubmitWorkflow is called
    Then GetWorkflowStatus returns label "env" with value "staging"

  @lifecycle
  Scenario: Submit with engine_hint routes to that engine
    Given a compiled WorkflowIR
    And the SubmitWorkflowRequest has engine_hint "temporal"
    When SubmitWorkflow is called
    Then the workflow is executed by the "temporal" engine

  @lifecycle
  Scenario: Submitting a duplicate run_id returns ALREADY_EXISTS
    Given a workflow run "run-fixed-id" is already RUNNING
    When SubmitWorkflow is called with the same run_id "run-fixed-id"
    Then the gRPC status is ALREADY_EXISTS
    And the error message contains "run-fixed-id"

  # ─── Signals ──────────────────────────────────────────────────────────────

  @signals
  Scenario: Signal a running workflow triggers a state transition
    Given workflow run "run-abc" is in RUNNING state
    And the workflow is waiting on signal "review.approved"
    When SignalWorkflow is called with event_type "review.approved"
    Then the gRPC status is OK
    And WatchWorkflow emits a WorkflowEvent with event_type "review.approved"

  @signals
  Scenario: Signal a completed workflow returns FAILED_PRECONDITION
    Given workflow run "run-done" is in COMPLETED state
    When SignalWorkflow is called with event_type "any.signal"
    Then the gRPC status is FAILED_PRECONDITION
    And the error message mentions "COMPLETED"

  @signals
  Scenario: Signal an unknown run_id returns NOT_FOUND
    When SignalWorkflow is called with run_id "nonexistent-run"
    Then the gRPC status is NOT_FOUND
    And the error message contains "nonexistent-run"

  # ─── Cancellation ─────────────────────────────────────────────────────────

  @lifecycle
  Scenario: Cancel a running workflow transitions it to CANCELLED
    Given workflow run "run-abc" is in RUNNING state
    When CancelWorkflow is called with run_id "run-abc" and reason "user_cancelled"
    Then GetWorkflowStatus for "run-abc" returns status CANCELLED
    And the cancellation reason is stored on the run record

  @lifecycle
  Scenario: Cancel a PENDING workflow transitions it to CANCELLED
    Given workflow run "run-pending" is in PENDING state
    When CancelWorkflow is called with run_id "run-pending"
    Then GetWorkflowStatus for "run-pending" returns status CANCELLED

  @lifecycle
  Scenario: Cancel a completed workflow returns FAILED_PRECONDITION
    Given workflow run "run-done" is in COMPLETED state
    When CancelWorkflow is called with run_id "run-done"
    Then the gRPC status is FAILED_PRECONDITION
    And the error message mentions "COMPLETED"

  @lifecycle
  Scenario: Cancel an unknown run_id returns NOT_FOUND
    When CancelWorkflow is called with run_id "ghost-run"
    Then the gRPC status is NOT_FOUND

  # ─── Status query ─────────────────────────────────────────────────────────

  @lifecycle
  Scenario: GetWorkflowStatus returns the full run record
    Given workflow run "run-abc" is in RUNNING state
    When GetWorkflowStatus is called with run_id "run-abc"
    Then the response includes a non-empty current_state
    And the response includes a non-zero started_at timestamp
    And the response includes the workflow_id from the original IR
    And the response status is RUNNING

  @lifecycle
  Scenario: GetWorkflowStatus for a completed run includes finished_at
    Given workflow run "run-done" has reached COMPLETED state
    When GetWorkflowStatus is called with run_id "run-done"
    Then the response includes a non-zero finished_at timestamp

  @lifecycle
  Scenario: GetWorkflowStatus for an unknown run_id returns NOT_FOUND
    When GetWorkflowStatus is called with run_id "nonexistent-run"
    Then the gRPC status is NOT_FOUND
    And the error message contains "nonexistent-run"

  # ─── Watch stream ─────────────────────────────────────────────────────────

  @signals
  Scenario: WatchWorkflow streams events as the workflow progresses
    Given workflow run "run-abc" is in RUNNING state
    When WatchWorkflow is called with run_id "run-abc"
    Then the stream emits at least one WorkflowEvent
    And every WorkflowEvent carries run_id "run-abc"
    And every WorkflowEvent has a non-zero timestamp

  @signals
  Scenario: WatchWorkflow stream closes when workflow reaches terminal state
    Given workflow run "run-abc" will complete during the watch
    When WatchWorkflow is called with run_id "run-abc"
    Then the stream emits a WorkflowEvent with a terminal status
    And the stream closes cleanly after the terminal event

  @signals
  Scenario: WorkflowEvent includes state transition details
    Given workflow run "run-abc" transitions from state "review" to "approve"
    When WatchWorkflow emits the transition event
    Then the WorkflowEvent from_state is "review"
    And the WorkflowEvent to_state is "approve"

  @signals
  Scenario: WatchWorkflow for an unknown run_id returns NOT_FOUND
    When WatchWorkflow is called with run_id "nonexistent-run"
    Then the gRPC status is NOT_FOUND
    And no WorkflowEvent is emitted

  # ─── Input validation ─────────────────────────────────────────────────────

  @lifecycle
  Scenario: SubmitWorkflow with missing workflow_ir is rejected
    Given a SubmitWorkflowRequest with no workflow_ir
    When SubmitWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "workflow_ir"

  @signals
  Scenario: SignalWorkflow with empty run_id is rejected
    Given a SignalWorkflowRequest with run_id set to ""
    When SignalWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "run_id"

  @signals
  Scenario: SignalWorkflow with empty event_type is rejected
    Given a SignalWorkflowRequest with event_type set to ""
    When SignalWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "event_type"

  @lifecycle
  Scenario: CancelWorkflow with empty run_id is rejected
    Given a CancelWorkflowRequest with run_id set to ""
    When CancelWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "run_id"

  @lifecycle
  Scenario: GetWorkflowStatus with empty run_id is rejected
    Given a GetWorkflowStatusRequest with run_id set to ""
    When GetWorkflowStatus is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "run_id"
