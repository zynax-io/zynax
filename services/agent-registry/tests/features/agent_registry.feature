# SPDX-License-Identifier: Apache-2.0
# Keel — Agent Registry BDD Feature File
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It is the source of truth for what the agent-registry service does.
# See AGENTS.md §6.2 for feature file writing rules.

Feature: Agent Registration
  As an orchestrator or autonomous agent
  I want to register agents with their capabilities and endpoint
  So that other agents and the task broker can discover and collaborate with them

  Background:
    Given the agent registry is running and healthy

  # ─── Happy path ───────────────────────────────────────────────────────────

  Scenario: Successfully register a new agent
    Given an agent spec with id "analyst-01" and capabilities ["summarize", "search"]
    When the agent is registered with a valid request_id
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

  # ─── Idempotency ─────────────────────────────────────────────────────────

  Scenario: Registration is idempotent for the same request_id
    Given an agent spec with id "analyst-02"
    And a registration was already completed with request_id "req-abc-123"
    When the same registration is attempted again with request_id "req-abc-123"
    Then the response is identical to the first registration
    And no duplicate agent record is created

  # ─── Validation failures ─────────────────────────────────────────────────

  Scenario: Reject duplicate agent_id from different request
    Given an agent with id "existing-agent-01" is already registered
    When a new agent registration is attempted with id "existing-agent-01" and a different request_id
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

  Scenario: Reject agent with missing request_id
    Given an agent spec with id "no-req-id"
    When the agent is registered without a request_id
    Then the response status is INVALID_ARGUMENT
    And the error message mentions "request_id"

Feature: Agent Discovery
  As an orchestrator or task broker
  I want to discover agents by capability
  So that I can route tasks to capable agents

  Background:
    Given the agent registry is running and healthy
    And the following agents are registered:
      | id           | capabilities           | status  |
      | agent-sum-01 | summarize, search      | ACTIVE  |
      | agent-sum-02 | summarize              | ACTIVE  |
      | agent-wri-01 | write, summarize       | ACTIVE  |
      | agent-off-01 | summarize              | OFFLINE |

  Scenario: Discover active agents by capability
    When agents are listed by capability "summarize"
    Then the response contains exactly 3 agents
    And the response includes "agent-sum-01", "agent-sum-02", and "agent-wri-01"
    And the response does NOT include "agent-off-01"

  Scenario: Discovery includes OFFLINE agents when explicitly requested
    When agents are listed by capability "summarize" with status filter [ACTIVE, OFFLINE]
    Then the response contains exactly 4 agents
    And the response includes "agent-off-01"

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

Feature: Agent Heartbeat
  As a registered agent
  I want to send periodic heartbeats
  So that the registry knows I am alive and can route tasks to me

  Background:
    Given the agent registry is running and healthy
    And an agent with id "heartbeat-agent" is registered and ACTIVE

  Scenario: Agent remains ACTIVE after regular heartbeats
    When the agent sends 5 heartbeats at regular intervals
    Then the agent status remains ACTIVE
    And the last_heartbeat_at timestamp is updated after each heartbeat

  Scenario: Agent transitions to OFFLINE after missing heartbeats
    Given the heartbeat miss threshold is configured to 3
    When the agent misses 3 consecutive heartbeat windows
    Then the agent status transitions to OFFLINE
    And a WatchAgentEvents subscriber receives an AGENT_STATUS_CHANGED event

  Scenario: Agent recovers to ACTIVE after resuming heartbeats
    Given the agent is currently OFFLINE due to missed heartbeats
    When the agent sends a heartbeat
    Then the agent status transitions back to ACTIVE

Feature: Agent Deregistration
  As an agent or orchestrator
  I want to deregister an agent gracefully
  So that it is no longer discoverable after shutdown

  Background:
    Given the agent registry is running and healthy

  Scenario: Successfully deregister an existing agent
    Given an agent with id "departing-agent" is registered
    When the agent is deregistered with a valid request_id
    Then the response contains a deregistered_at timestamp
    And the agent is no longer discoverable by capability
    And GetAgent returns NOT_FOUND for the deregistered id

  Scenario: Deregister non-existent agent returns NOT_FOUND
    When an agent with id "ghost-agent-99" is deregistered
    Then the response status is NOT_FOUND
