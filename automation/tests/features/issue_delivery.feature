# SPDX-License-Identifier: Apache-2.0
# automation/tests/features/issue_delivery.feature
#
# BDD contract for the intake → plan → route leg of the issue-delivery
# Workflow (EPIC #881 O4, #1099 — ADR-016: contract before implementation).
# Steps are bound by automation/tests/test_issue_delivery.py, which evaluates
# the declarative manifest (automation/workflows/issue-delivery.yaml) against
# fixture issues — no running platform required for this leg.

Feature: Issue-delivery workflow — intake, planning, and expert routing
  As the self-hosted automation plane
  I want the issue-delivery workflow to read an issue, identify the next
  issue via the planner, and route to exactly one expert AgentDef
  So that one GitHub issue can be driven to an implemented change on-platform

  Background:
    Given the workflow manifest "automation/workflows/issue-delivery.yaml"
    And the manifest validates against "spec/schemas/workflow.schema.json"

  Scenario: Manifest declares the O4 state machine
    Then the initial state is "intake"
    And the states are exactly "intake, plan, route, routed, blocked, failed"
    And the states "routed, blocked, failed" are terminal

  Scenario: Plan state calls the planner with its exact capability contract
    When the "plan" state dispatches its capability
    Then the capability is "identify_next_issue"
    And the action input keys are exactly the planner's required inputs
      | milestone | open_issues | in_progress | dependency_table |
    And the action outputs map every required planner output into the context
      | next_issue | blocked_by | ready_batch | rationale |

  Scenario: A fixture issue is classified and routed to the right expert
    Given a fixture issue titled "feat(automation): issue-intake + planning Workflow"
    And the planner replies next_issue 1099 with no blockers
    When the workflow runs the intake, plan and route states
    Then the decision is next_issue 1099, expert "planning-task-split", blocked_by empty

  Scenario Outline: The routing table selects one expert per issue class
    Given a fixture issue titled "<title>"
    And the planner replies next_issue 1099 with no blockers
    When the workflow runs the intake, plan and route states
    Then the selected expert is "<expert>"

    Examples:
      | title                                                | expert                |
      | feat(protos): add memory service RPC                 | api-contract          |
      | fix(task-broker): lease renewal race                 | persistence-state     |
      | ci(actions): pin runner image digests                | ci-release            |
      | fix(security): rotate cosign signing key             | security-supply-chain |
      | test(bdd): cover dispatch timeout path               | qa-bdd                |
      | docs(adr): record engine decision                    | arch-adr              |
      | docs(readme): refresh quickstart                     | docs-agents           |
      | chore(automation): tidy milestone state              | planning-task-split   |

  Scenario: A dependency-blocked plan ends in the blocked terminal state
    Given a fixture issue titled "feat(protos): add memory service RPC"
    And the planner replies next_issue 0 blocked by issues "1097, 1098"
    When the workflow runs the intake, plan and route states
    Then the workflow ends in state "blocked"
    And blocked_by is "1097, 1098"
