# SPDX-License-Identifier: Apache-2.0
# Zynax — WorkflowCompilerService BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes the contract the workflow compiler must honour.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: YAML manifests are the user-facing intent layer (Layer 1).
# The workflow compiler transforms them into an engine-agnostic Intermediate
# Representation (IR) for transmission to any engine adapter. This contract
# decouples the YAML authoring experience from execution engine details.
# (ADR-001, ADR-011, ADR-012)

Feature: WorkflowCompilerService contract — YAML manifest compilation
  As an API gateway submitting user-authored workflow manifests
  I want to compile YAML into a validated engine-agnostic IR
  So that the compiled workflow can be dispatched to any engine adapter

  Background:
    Given a WorkflowCompilerService is running on a test gRPC server

  # ─── Successful compilation ───────────────────────────────────────────────

  Scenario: Compile a valid workflow manifest returns a WorkflowIR
    Given a valid Workflow YAML with 3 states and one terminal state
    When CompileWorkflow is called with the manifest
    Then the gRPC status is OK
    And the response contains a WorkflowIR
    And the WorkflowIR workflow_id is populated
    And the compilation_duration_ms is greater than zero

  Scenario: Compiled IR preserves the workflow name and namespace
    Given a Workflow YAML with name "code-review" and namespace "team-alpha"
    When CompileWorkflow is called
    Then the WorkflowIR name is "code-review"
    And the WorkflowIR namespace is "team-alpha"

  Scenario: Compilation with warnings still returns a WorkflowIR
    Given a Workflow YAML that is valid but uses a deprecated field
    When CompileWorkflow is called
    Then the gRPC status is OK
    And the response contains a WorkflowIR
    And the response contains at least one warning message

  Scenario: Dry-run returns IR without side effects
    Given a valid Workflow YAML
    And the CompileWorkflowRequest has dry_run set to true
    When CompileWorkflow is called
    Then the gRPC status is OK
    And the response contains a WorkflowIR
    And no workflow record is persisted

  # ─── Validation-only path ─────────────────────────────────────────────────

  Scenario: ValidateManifest on a valid YAML reports valid with zero errors
    Given a valid Workflow YAML
    When ValidateManifest is called
    Then the response valid field is true
    And the response contains zero CompilationErrors
    And no WorkflowIR is returned

  Scenario: ValidateManifest on an invalid YAML reports errors without compiling
    Given a Workflow YAML with no terminal state
    When ValidateManifest is called
    Then the response valid field is false
    And the response contains at least one CompilationError
    And no WorkflowIR is returned

  Scenario: ValidateManifest returns all errors found — not just the first
    Given a Workflow YAML with an orphan state and a duplicate state name
    When ValidateManifest is called
    Then the response contains a CompilationError with code ORPHAN_STATE
    And the response contains a CompilationError with code DUPLICATE_STATE_NAME

  # ─── Structural validation errors ─────────────────────────────────────────

  Scenario: Reject a manifest with no terminal state
    Given a Workflow YAML where no state has type "terminal"
    When CompileWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the response contains a CompilationError with code NO_TERMINAL_STATE

  Scenario: Reject a manifest with an orphan state
    Given a Workflow YAML where state "fix" is never referenced in transitions
    When CompileWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the response contains a CompilationError with code ORPHAN_STATE
    And the CompilationError names the state "fix"

  Scenario: Reject a manifest with duplicate state names
    Given a Workflow YAML where state name "review" appears twice
    When CompileWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the response contains a CompilationError with code DUPLICATE_STATE_NAME
    And the CompilationError names the state "review"

  Scenario: Reject a manifest with a transition to an unknown state
    Given a Workflow YAML where a transition targets state "nonexistent"
    When CompileWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the response contains a CompilationError with code UNKNOWN_STATE_REFERENCE
    And the CompilationError names the state "nonexistent"

  Scenario: Reject a manifest with no initial state
    Given a Workflow YAML where no state is marked as initial
    When CompileWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the response contains a CompilationError with code NO_INITIAL_STATE

  Scenario: Reject a manifest with multiple initial states
    Given a Workflow YAML where two states are marked as initial
    When CompileWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the response contains a CompilationError with code MULTIPLE_INITIAL_STATES

  Scenario: CompilationError includes line_number when YAML is malformed
    Given a Workflow YAML with a syntax error on line 7
    When CompileWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the response contains a CompilationError with code YAML_PARSE_ERROR
    And the CompilationError line_number is 7

  # ─── Input validation ─────────────────────────────────────────────────────

  Scenario: CompileWorkflow with empty manifest_yaml is rejected
    Given a CompileWorkflowRequest with manifest_yaml set to empty bytes
    When CompileWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "manifest_yaml"

  Scenario: CompileWorkflow with non-YAML bytes is rejected
    Given a CompileWorkflowRequest with manifest_yaml set to "not yaml {"
    When CompileWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the response contains a CompilationError with code YAML_PARSE_ERROR

  Scenario: ValidateManifest with empty manifest_yaml is rejected
    Given a ValidateManifestRequest with manifest_yaml set to empty bytes
    When ValidateManifest is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "manifest_yaml"
