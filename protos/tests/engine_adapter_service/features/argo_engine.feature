# SPDX-License-Identifier: Apache-2.0
# Zynax — ArgoEngine BDD Contract Specification
#
# This file is the SPECIFICATION. It is committed BEFORE the ArgoEngine
# implementation (ADR-016: contracts before code).
#
# Business context: the Argo Workflows engine is the second pluggable backend
# behind the EngineAdapterService contract (ADR-015). Operators choose between
# Temporal (stateful, durable) and Argo (K8s-native, DAG-first) per deployment.
# These scenarios verify that the adapter honours the same gRPC contract when
# dispatching to Argo Workflows.
#
# Tags:
#   @argo-submit  — workflow submission via Argo
#   @argo-query   — status query against an Argo-dispatched run
#   @argo-cancel  — cancellation of an Argo-dispatched run

Feature: ArgoEngine dispatch — Argo Workflows execution via EngineAdapterService
  As a workflow author selecting the Argo Workflows execution engine
  I want the engine adapter to translate WorkflowIR into an Argo Workflow resource
  So that my workflows execute on K8s-native Argo infrastructure transparently

  Background:
    Given an EngineAdapterService is running on a test gRPC server

  # ─── Submission ───────────────────────────────────────────────────────────

  @argo-submit
  Scenario: Submitting a WorkflowIR with engine_hint "argo" returns a run_id
    Given a compiled WorkflowIR for workflow "argo-dag-pipeline"
    And the SubmitWorkflowRequest has engine_hint "argo"
    When SubmitWorkflow is called with the IR
    Then the gRPC status is OK
    And the response contains a non-empty run_id
    And the workflow is executed by the "argo" engine

  @argo-submit
  Scenario: Submitting without a workflow_ir returns INVALID_ARGUMENT
    Given a SubmitWorkflowRequest with no workflow_ir
    And the SubmitWorkflowRequest has engine_hint "argo"
    When SubmitWorkflow is called with the IR
    Then the gRPC status is INVALID_ARGUMENT

  @argo-submit
  Scenario: Submitting a duplicate active run returns ALREADY_EXISTS
    Given a workflow run "argo-run-001" is already RUNNING
    When SubmitWorkflow is called with the same run_id "argo-run-001"
    Then the gRPC status is ALREADY_EXISTS
    And the error message contains "argo-run-001"

  # ─── Status query ─────────────────────────────────────────────────────────

  @argo-query
  Scenario: GetWorkflowStatus returns RUNNING for an active Argo run
    Given workflow run "argo-run-002" is in RUNNING state
    When GetWorkflowStatus is called with run_id "argo-run-002"
    Then the gRPC status is OK
    And the response status is RUNNING
    And the response includes a non-empty current_state
    And the response includes the workflow_id from the original IR

  @argo-query
  Scenario: GetWorkflowStatus returns NOT_FOUND for an unknown Argo run
    When GetWorkflowStatus is called with run_id "argo-run-unknown"
    Then the gRPC status is NOT_FOUND

  @argo-query
  Scenario: GetWorkflowStatus with empty run_id returns INVALID_ARGUMENT
    When GetWorkflowStatus is called
    Then the gRPC status is INVALID_ARGUMENT

  # ─── Cancellation ─────────────────────────────────────────────────────────

  @argo-cancel
  Scenario: CancelWorkflow transitions an Argo run to CANCELLED
    Given workflow run "argo-run-003" is in RUNNING state
    When CancelWorkflow is called with run_id "argo-run-003" and reason "operator-requested"
    Then the gRPC status is OK
    And GetWorkflowStatus for "argo-run-003" returns status CANCELLED
    And the cancellation reason is stored on the run record

  @argo-cancel
  Scenario: CancelWorkflow on an unknown Argo run returns NOT_FOUND
    When CancelWorkflow is called with run_id "argo-run-ghost"
    Then the gRPC status is NOT_FOUND

  @argo-cancel
  Scenario: CancelWorkflow on a completed Argo run returns FAILED_PRECONDITION
    Given workflow run "argo-run-004" is in COMPLETED state
    When CancelWorkflow is called with run_id "argo-run-004" and reason "late-cancel"
    Then the gRPC status is FAILED_PRECONDITION
