# SPDX-License-Identifier: Apache-2.0
# Zynax — Agent Registry BDD Feature File
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It is the source of truth for what the agent-registry service does.
# See AGENTS.md §6.2 for feature file writing rules.
#
# RPCs covered: RegisterAgent, DeregisterAgent, GetAgent, ListAgents, FindByCapability.
# Phantom RPCs removed in #526: Heartbeat, WatchAgentEvents (not in agent_registry.proto).
# Phantom fields/values removed: request_id, AGENT_STATUS_ACTIVE, AGENT_STATUS_OFFLINE.

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

  # ─── Validation failures ─────────────────────────────────────────────────

  Scenario: Reject duplicate agent_id
    Given an agent with id "existing-agent-01" is already registered
    When a new agent registration is attempted with id "existing-agent-01"
    Then the response status is ALREADY_EXISTS
    And the error message contains "existing-agent-01"

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

Feature: Agent Discovery
  As an orchestrator or task broker
  I want to discover agents by capability
  So that I can route tasks to capable agents

  Background:
    Given the agent registry is running and healthy
    And the following agents are registered:
      | id           | capabilities      |
      | agent-sum-01 | summarize, search |
      | agent-sum-02 | summarize         |
      | agent-wri-01 | write, summarize  |

  Scenario: Find agents by capability
    When agents are listed by capability "summarize"
    Then the response contains exactly 3 agents
    And the response includes "agent-sum-01", "agent-sum-02", and "agent-wri-01"

  Scenario: Discovery returns empty list when no matching agents
    When agents are listed by capability "nonexistent-capability"
    Then the response contains 0 agents
    And the response status is OK (not NOT_FOUND)

  Scenario: Discovery results are paginated
    Given 25 agents with capability "batch-test" are registered
    When agents are listed by capability "batch-test" with page_size 10
    Then the response contains exactly 10 agents
    And the response contains a non-empty next_page_token
    When the next page is requested using the page_token
    Then the response contains exactly 10 agents
    When the final page is requested
    Then the response contains exactly 5 agents
    And the response next_page_token is empty

Feature: Agent Deregistration
  As an agent or orchestrator
  I want to deregister an agent gracefully
  So that it is no longer discoverable after shutdown

  Background:
    Given the agent registry is running and healthy

  Scenario: Successfully deregister an existing agent
    Given an agent with id "departing-agent" is registered
    When the agent is deregistered
    Then the response contains a deregistered_at timestamp
    And the agent is no longer discoverable by capability
    And GetAgent returns NOT_FOUND for the deregistered id

  Scenario: Deregister non-existent agent returns NOT_FOUND
    When an agent with id "ghost-agent-99" is deregistered
    Then the response status is NOT_FOUND
