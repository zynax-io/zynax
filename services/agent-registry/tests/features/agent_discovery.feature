# SPDX-License-Identifier: Apache-2.0
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
