# SPDX-License-Identifier: Apache-2.0
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
