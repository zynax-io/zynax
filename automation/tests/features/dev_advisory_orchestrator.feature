# SPDX-License-Identifier: Apache-2.0
# Zynax — Dev-Advisory Orchestrator Workflow BDD Contract (EPIC #881 O3, #1098)
#
# This file is the SPECIFICATION (ADR-016). It describes the behaviour the
# dev-advisory-orchestrator Workflow manifest must encode: parallel fan-out of
# the 9 expert review capabilities with strict context isolation, weighted-
# consensus aggregation, an act state limited to auto_allowed actions, and
# human escalation. Step implementations arrive with the runtime binding
# (O5, #1100) and the platform-readiness e2e (O8, #1103).
#
# Business context: the orchestrator is the on-platform /m6-orchestrate — a
# thin coordinator (8K token budget) that sees expert OUTPUTS only, never code
# files (ADR-028). Aggregation weights are registry labels on the expert
# AgentDefs; strategy and thresholds are translated 1:1 from the archived
# orchestrator config (docs/archive/dev-advisory/orchestrator/config.yaml).

Feature: Dev-advisory orchestrator — fan-out, aggregate, act
  As the Zynax platform maintainer
  I want a declarative Workflow that fans out to the 9 domain experts,
  aggregates their reviews into a weighted-consensus verdict, and acts
  So that issue and PR advisory runs are self-hosted on Zynax itself

  Background:
    Given the Workflow manifest "automation/workflows/dev-advisory-orchestrator.yaml" is loaded
    And the 9 expert AgentDef manifests under "automation/workflows/experts/" are registered

  # ─── fan_out ──────────────────────────────────────────────────────────────

  Scenario: Fan-out dispatches all 9 expert review capabilities in parallel
    Given a pull request against "main" triggers a new workflow instance
    When the workflow enters the "fan_out" state
    Then 9 "review" capability dispatches start before any transition is awaited
    And each dispatch names exactly one expert from the registered AgentDef set
    And no expert is dispatched more than once

  Scenario: Each expert receives only its own bounded context slice
    When the "review" capability is dispatched for expert "arch-adr"
    Then the input payload contains only the context_slice declared by the
      "arch-adr" AgentDef, bound at dispatch time
    And no other expert's context slice or output is present in the payload

  Scenario: The orchestrator never reads code files
    When the workflow aggregates in the "aggregate" state
    Then the aggregation input contains expert outputs only
    And no raw repository file content is present in the orchestrator context

  Scenario: Expert timeout produces a partial aggregation, not a stall
    Given 8 experts complete and one expert exceeds its 5m timeout
    When the "review.timeout" event arrives
    Then the workflow transitions to "aggregate" with partial results

  # ─── aggregate ────────────────────────────────────────────────────────────

  Scenario: Aggregation computes a weighted-consensus verdict
    Given all 9 expert reviews are collected in the workflow context
    When the "aggregate_reviews" capability completes
    Then the workflow context holds an "aggregated_verdict" with a confidence
      of "low", "medium" or "high"
    And vote weights are derived from each expert's "aggregation-weight"
      registry label multiplied by its confidence score
    And only actions with aggregate weight >= 3.0 are included in the verdict

  Scenario: A clean verdict proceeds to act
    Given the aggregated verdict requires no escalation
    When the "aggregate_reviews.completed" event arrives
    Then the workflow transitions to the "act" state

  # ─── escalation ───────────────────────────────────────────────────────────

  Scenario: Two high-confidence contradictions escalate to a human
    Given 2 high-confidence experts recommend contradictory actions
    When the "aggregate_reviews.completed" event arrives
    Then the workflow transitions to the "escalate" state
    And the "escalate" state is human_in_the_loop

  Scenario: Any tier-2 flag escalates to a human
    Given any expert raises a tier-2 flag in its flags array
    When the "aggregate_reviews.completed" event arrives
    Then the workflow transitions to the "escalate" state

  Scenario: A low-confidence top action escalates to a human
    Given the top recommended action has aggregate confidence "low"
    When the "aggregate_reviews.completed" event arrives
    Then the workflow transitions to the "escalate" state

  Scenario: A human approval resumes the act state
    Given the workflow is paused in the "escalate" state
    When the "human.approved" event arrives
    Then the workflow transitions to the "act" state

  # ─── act — auto_allowed only ──────────────────────────────────────────────

  Scenario: Act executes auto_allowed actions only
    Given the aggregated verdict recommends "auto-label" and "post-pr-comment"
    When the workflow enters the "act" state
    Then both actions are executed automatically
    And the executed actions are recorded in the workflow context

  Scenario Outline: A prohibited_auto_action is never executed automatically
    Given the aggregated verdict recommends "<action>"
    When the workflow enters the "act" state
    Then "<action>" is not executed
    And executing "<action>" requires explicit human approval

    Examples:
      | action          |
      | merge           |
      | push            |
      | bump-dependency |
      | close-issue     |
      | delete-branch   |
      | force-push      |

  # ─── decision log ─────────────────────────────────────────────────────────

  Scenario: Every run records a decision-log entry
    Given a workflow instance reaches the terminal "done" state
    Then a decision-log entry is recorded with the verdict, expert votes,
      and any auto actions taken
    And the entry conforms to the decision-log schema
