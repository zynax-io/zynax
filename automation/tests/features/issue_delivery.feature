# SPDX-License-Identifier: Apache-2.0
# automation/tests/features/issue_delivery.feature
#
# BDD contract for the issue-delivery Workflow (EPIC #881 — ADR-016:
# contract before implementation): the intake → plan → route leg (O4, #1099)
# and the inject → implement → verify → decide delivery leg (O6, #1101).
# Steps are bound by automation/tests/test_issue_delivery.py, which evaluates
# the declarative manifest (automation/workflows/issue-delivery.yaml) against
# fixture issues — no running platform required for these legs (the live e2e
# is O8, #1103).

Feature: Issue-delivery workflow — intake, planning, routing, and delivery
  As the self-hosted automation plane
  I want the issue-delivery workflow to read an issue, identify the next
  issue via the planner, route to exactly one expert AgentDef, drive that
  expert to an implemented change, verify it, and record the decision
  So that one GitHub issue can be driven to an implemented change on-platform

  Background:
    Given the workflow manifest "automation/workflows/issue-delivery.yaml"
    And the manifest validates against "spec/schemas/workflow.schema.json"

  Scenario: Manifest declares the full delivery state machine
    Then the initial state is "intake"
    And the states are exactly "intake, plan, route, inject, implement, verify, decide, blocked, failed"
    And the states "decide, blocked, failed" are terminal

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

  Scenario: Inject resolves the routed expert's context-slice binding
    When the "inject" state dispatches its capability
    Then the capability is "resolve_context_slice"
    And the input keys are exactly "expert, capability"
    And the requested capability is "review" for the routed expert
    And the action outputs record the agent id, slice files and max_tokens

  Scenario: Implement drives exactly the routed expert's review capability
    When the "implement" state dispatches its capability
    Then the capability is "review"
    And the dispatch is keyed by the routed expert
    And the input covers the review contract's required fields except context_slice
    And no literal context_slice is ever inlined in the manifest
    And the decision record carries only the resolved slice as context references
    And the trigger value is a member of the review contract's trigger enum

  Scenario: A verified change reaches decide and records a decision-log row
    Given a fixture issue titled "feat(automation): delivery leg of issue-delivery"
    And the planner replies next_issue 1101 with no blockers
    And the routed expert produces a change
    And the verification gates pass
    When the workflow runs the full delivery leg
    Then the workflow ends in the terminal state "decide"
    And delivery_outcome is "success"
    And the decide state records a decision via "record_decision"
    And the decide state emits the next-issue CloudEvent via "emit_next_issue"

  Scenario: Failing gates still record a durable decision
    Given a fixture issue titled "feat(automation): delivery leg of issue-delivery"
    And the planner replies next_issue 1101 with no blockers
    And the routed expert produces a change
    And the verification gates fail
    When the workflow runs the full delivery leg
    Then the workflow ends in the terminal state "decide"
    And delivery_outcome is "gates_failed"

  Scenario: A gate-runner malfunction ends in the failed terminal state
    Given the verification gate runner reports a failure event
    Then the workflow ends in state "failed"
    And failure_reason explains the verify failure

  Scenario: Destructive actions are never auto-executed
    Then every capability the manifest dispatches is non-destructive
    And the recorded decision carries the verbatim prohibited_auto_actions
      | merge | push | bump-dependency | close-issue | delete-branch | force-push |
    And the recorded decision declares human_action_required
