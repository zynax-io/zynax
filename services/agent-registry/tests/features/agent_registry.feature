# SPDX-License-Identifier: Apache-2.0
# Zynax — Agent Registry BDD Feature File
#
# RPCs covered: RegisterAgent, DeregisterAgent, GetAgent, ListAgents, FindByCapability.
# Phantom RPCs removed in #526: Heartbeat, WatchAgentEvents (not in agent_registry.proto).

Feature: Agent Registration
  As an orchestrator or autonomous agent
  I want to register agents with their capabilities and endpoint
  So that other agents and the task broker can discover and collaborate with them

  Background:
    Given the agent registry is running and healthy

  # ─── Happy path ───────────────────────────────────────────────────────────

  Scenario: Successfully register a new agent
    Given an agent spec with id "analyst-01" and capabilities ["summarize", "search"]
    When the agent is registered
    Then the response contains a non-empty agent_id
    And the agent_id matches the requested id "analyst-01"
    And the response contains a valid registered_at timestamp
    And the agent is discoverable by capability "summarize"
    And the agent is discoverable by capability "search"

  Scenario: Agent with rich metadata is registered correctly
    Given an agent spec with id "writer-01" and capabilities ["write"]
    And the spec includes metadata: {"model": "gpt-4", "region": "eu-west-1"}
    When the agent is registered
    Then the metadata is persisted and retrievable via GetAgent

  # ─── Idempotent re-registration (issue #1463) ────────────────────────────

  Scenario: Re-registering an existing agent_id is idempotent
    Given an agent with id "existing-agent-01" is already registered
    When a new agent registration is attempted with id "existing-agent-01"
    Then the response status is OK (not NOT_FOUND)
    And the agent_id matches the requested id "existing-agent-01"

  # ─── Validation failures ─────────────────────────────────────────────────

  Scenario: Reject agent with no capabilities
    Given an agent spec with id "empty-agent" and capabilities []
    When the agent is registered
    Then the response status is INVALID_ARGUMENT
    And the error message mentions "at least one capability"

  Scenario: Reject agent with too many capabilities
    Given an agent spec with id "overloaded-agent" and 51 capabilities
    When the agent is registered
    Then the response status is INVALID_ARGUMENT
    And the error message mentions the capability limit of 50

  Scenario: Reject agent with invalid capability format
    Given an agent spec with id "bad-caps" and capabilities ["InvalidCap", "UPPERCASE"]
    When the agent is registered
    Then the response status is INVALID_ARGUMENT
    And the error message mentions valid capability format
