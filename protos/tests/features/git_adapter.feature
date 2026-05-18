# SPDX-License-Identifier: Apache-2.0
# Zynax — git-adapter BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes the contract the git-adapter must honour.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: The git-adapter wraps GitHub/GitLab operations (open PR,
# request review, get diff) as Zynax capabilities. It is the boundary between
# the Zynax workflow engine and source-code hosting providers. (ADR-013)
#
# Canvas: docs/spdd/381-git-adapter/canvas.md
# Parent epic: #381 (git-adapter — GitHub/GitLab operations)

Feature: git-adapter — source-code hosting capability adapter
  As a platform operator
  I want a git-adapter that wraps GitHub/GitLab operations as Zynax capabilities
  So that workflow steps can open PRs, request reviews, and fetch diffs
  without the control plane knowing which provider is in use

  Background:
    Given a git-adapter configured for provider "github-stub"
    And the adapter is registered with AgentRegistryService

  # ─── open_pr ────────────────────────────────────────────────────────────────

  Scenario: open_pr creates a pull request and returns the URL
    Given a valid ExecuteCapabilityRequest for capability "open_pr"
    And the input payload contains title, head branch, and base branch
    When ExecuteCapability is called
    Then the final TaskEvent has event_type COMPLETED
    And the COMPLETED payload contains a non-empty "pr_url" field
    And the COMPLETED payload contains a non-empty "pr_number" field

  Scenario: open_pr emits PROGRESS events while the PR is being created
    Given the provider API introduces a delay before returning
    And a valid ExecuteCapabilityRequest for capability "open_pr" with timeout_seconds 10
    When ExecuteCapability is called
    Then the stream emits at least one TaskEvent with event_type PROGRESS before COMPLETED
    And every PROGRESS event has task_id echoed and timestamp populated

  # ─── request_review ─────────────────────────────────────────────────────────

  Scenario: request_review polls for confirmation and returns COMPLETED
    Given a valid ExecuteCapabilityRequest for capability "request_review"
    And the input payload contains pr_number and reviewer list
    And the provider confirms the review request within 2 poll cycles
    When ExecuteCapability is called
    Then the stream emits at least one TaskEvent with event_type PROGRESS
    And the final TaskEvent has event_type COMPLETED

  Scenario: request_review emits FAILED TIMEOUT when confirmation is not received
    Given a valid ExecuteCapabilityRequest for capability "request_review" with timeout_seconds 1
    And the provider never confirms the review request
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "TIMEOUT"

  # ─── get_diff ───────────────────────────────────────────────────────────────

  Scenario: get_diff returns diff bytes for a valid PR
    Given a valid ExecuteCapabilityRequest for capability "get_diff"
    And the input payload contains pr_number
    And the diff size is under 4 MB
    When ExecuteCapability is called
    Then the final TaskEvent has event_type COMPLETED
    And the COMPLETED payload contains a non-empty "diff" field
    And the COMPLETED payload contains truncated set to false

  Scenario: get_diff sets truncated true when diff exceeds 4 MB
    Given a valid ExecuteCapabilityRequest for capability "get_diff"
    And the provider returns a diff larger than 4 MB
    When ExecuteCapability is called
    Then the final TaskEvent has event_type COMPLETED
    And the COMPLETED payload contains truncated set to true
    And the "diff" field contains only the first 4 MB

  # ─── Rate-limit and auth error mapping ──────────────────────────────────────

  Scenario: GitHub API 429 produces FAILED with RESOURCE_EXHAUSTED
    Given the provider returns HTTP 429 for any request
    And a valid ExecuteCapabilityRequest for capability "open_pr"
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "RESOURCE_EXHAUSTED"

  Scenario: GitHub API 403 produces FAILED with RESOURCE_EXHAUSTED
    Given the provider returns HTTP 403 for any request
    And a valid ExecuteCapabilityRequest for capability "open_pr"
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "RESOURCE_EXHAUSTED"

  # ─── Timeout ────────────────────────────────────────────────────────────────

  Scenario: timeout_seconds breach produces FAILED with TIMEOUT code
    Given the provider delays all responses beyond the timeout
    And an ExecuteCapabilityRequest for capability "get_diff" with timeout_seconds 1
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "TIMEOUT"

  # ─── Capability routing ──────────────────────────────────────────────────────

  Scenario: Unknown capability returns NOT_FOUND without entering the stream
    Given an ExecuteCapabilityRequest for capability "nonexistent_op"
    When ExecuteCapability is called
    Then the gRPC status is NOT_FOUND
    And no TaskEvent is emitted

  Scenario: Empty capability_name is rejected as INVALID_ARGUMENT
    Given an ExecuteCapabilityRequest with capability_name set to ""
    When ExecuteCapability is called
    Then the gRPC status is INVALID_ARGUMENT

  # ─── Provider support gate ───────────────────────────────────────────────────

  Scenario: provider gitlab returns FAILED with INTERNAL and "not implemented"
    Given a git-adapter configured for provider "gitlab"
    And a valid ExecuteCapabilityRequest for capability "open_pr"
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "INTERNAL"
    And the CapabilityError message contains "not implemented"

  # ─── Credential safety ──────────────────────────────────────────────────────

  Scenario: Credential values never appear in CapabilityError message
    Given the provider returns an authentication error
    And the adapter is configured with a token containing "secret-token-value"
    When ExecuteCapability is called with capability "open_pr"
    Then the CapabilityError message does not contain "secret-token-value"
