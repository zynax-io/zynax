# SPDX-License-Identifier: Apache-2.0
# Zynax — llm-adapter BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes the contract the llm-adapter must honour.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: The llm-adapter wraps LLM provider APIs (OpenAI, AWS Bedrock,
# Ollama) as Zynax capabilities via a config-only adapter layer. Workflow steps
# call chat_completion without knowing which provider is active. (ADR-013)
#
# Canvas: docs/spdd/383-llm-adapter/canvas.md
# Parent epic: #383 (llm-adapter — LLM provider capability adapter)

Feature: llm-adapter — LLM provider capability adapter
  As a platform operator
  I want an llm-adapter that wraps LLM provider APIs as Zynax capabilities
  So that workflow steps can call chat_completion regardless of which provider is active
  without credentials or provider details leaking into the workflow definition

  Background:
    Given an llm-adapter configured for provider "openai-stub"
    And the adapter is registered with AgentRegistryService

  # ─── chat_completion — provider parity ──────────────────────────────────────

  Scenario: chat_completion with OpenAI provider streams PROGRESS then COMPLETED
    Given an llm-adapter configured for provider "openai-stub"
    And a valid ExecuteCapabilityRequest for capability "chat_completion"
    And the input payload contains model and messages fields
    When ExecuteCapability is called
    Then the stream emits at least one TaskEvent with event_type PROGRESS
    And the final TaskEvent has event_type COMPLETED
    And the COMPLETED payload contains a non-empty "content" field

  Scenario: chat_completion with Bedrock provider streams PROGRESS then COMPLETED
    Given an llm-adapter configured for provider "bedrock-stub"
    And a valid ExecuteCapabilityRequest for capability "chat_completion"
    And the input payload contains model and messages fields
    When ExecuteCapability is called
    Then the stream emits at least one TaskEvent with event_type PROGRESS
    And the final TaskEvent has event_type COMPLETED
    And the COMPLETED payload contains a non-empty "content" field

  Scenario: chat_completion with Ollama provider streams PROGRESS then COMPLETED
    Given an llm-adapter configured for provider "ollama-stub"
    And a valid ExecuteCapabilityRequest for capability "chat_completion"
    And the input payload contains model and messages fields
    When ExecuteCapability is called
    Then the stream emits at least one TaskEvent with event_type PROGRESS
    And the final TaskEvent has event_type COMPLETED
    And the COMPLETED payload contains a non-empty "content" field

  # ─── Timeout ────────────────────────────────────────────────────────────────

  Scenario: timeout_seconds exceeded emits FAILED with TIMEOUT code
    Given the provider delays the response beyond the timeout
    And an ExecuteCapabilityRequest for capability "chat_completion" with timeout_seconds 1
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "TIMEOUT"

  # ─── Input validation ────────────────────────────────────────────────────────

  Scenario: Missing required field in input_payload emits FAILED with INVALID_INPUT
    Given an ExecuteCapabilityRequest for capability "chat_completion"
    And the input payload is missing the required "messages" field
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "INVALID_INPUT"

  Scenario: Wrong type in input_payload emits FAILED with INVALID_INPUT
    Given an ExecuteCapabilityRequest for capability "chat_completion"
    And the input payload has "messages" set to a non-array value
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "INVALID_INPUT"

  # ─── Provider error mapping ──────────────────────────────────────────────────

  Scenario: Provider API error emits FAILED with UPSTREAM_ERROR and sanitised message
    Given the provider returns an API error with body containing "quota_exceeded"
    And a valid ExecuteCapabilityRequest for capability "chat_completion"
    When ExecuteCapability is called
    Then the final TaskEvent has event_type FAILED
    And the CapabilityError code is "UPSTREAM_ERROR"
    And the CapabilityError message is non-empty
    And the CapabilityError message does not contain raw API response body content

  # ─── Schema introspection ────────────────────────────────────────────────────

  Scenario: GetCapabilitySchema returns the declared schema for chat_completion
    When GetCapabilitySchema is called with capability_name "chat_completion"
    Then the gRPC status is OK
    And the response input_schema_json matches the schema declared in the adapter config
    And the response output_schema_json matches the schema declared in the adapter config
    And the response description is non-empty

  # ─── Capability routing ──────────────────────────────────────────────────────

  Scenario: Unknown capability name returns NOT_FOUND
    Given an ExecuteCapabilityRequest for capability "nonexistent_model_call"
    When ExecuteCapability is called
    Then the gRPC status is NOT_FOUND
    And no TaskEvent is emitted

  # ─── Credential safety ──────────────────────────────────────────────────────

  Scenario: API key values never appear in any TaskEvent payload
    Given the adapter is configured with an API key containing "sk-secret-openai-key"
    And a valid ExecuteCapabilityRequest for capability "chat_completion"
    When ExecuteCapability is called
    Then no emitted TaskEvent payload contains "sk-secret-openai-key"

  Scenario: API key values never appear in CapabilityError message on provider error
    Given the adapter is configured with an API key containing "sk-secret-openai-key"
    And the provider returns an authentication error
    When ExecuteCapability is called with capability "chat_completion"
    Then the CapabilityError message does not contain "sk-secret-openai-key"
    And the CapabilityError message does not contain the raw API error response
