# SPDX-License-Identifier: Apache-2.0
# Zynax — SchedulerService BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes the contract the CRD-native scheduler must honour.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: the task broker asks the scheduler for exactly ONE live,
# capable agent per dispatch. The scheduler answers from its informer-backed
# view of Agent custom resources, scoring candidates through the ordered
# ADR-039 §4 pipeline, and explains every decision with a structured
# rationale. Metrics unavailability degrades the scoring mode — it never
# fails a selection. Expert targeting is strict (ADR-028): no fallback.
# (ADR-039, ADR-040, ADR-028)

Feature: SchedulerService contract — scored, readiness-aware agent selection
  As a dispatching service routing work to capability providers
  I want exactly one scored agent per selection, with a structured rationale
  So that dispatch is deterministic, explainable, and never blind to liveness

  Background:
    Given a SchedulerService is running on a test gRPC server
    And the scheduler view is empty

  # ─── Happy path ───────────────────────────────────────────────────────────

  Scenario: One agent selected with a structured rationale
    Given a ready agent "agent-reviewer-a" declaring capability "code-review"
    And a ready agent "agent-reviewer-b" declaring capability "code-review"
    When SelectAgent is called with capability_name "code-review"
    Then the response contains exactly one agent
    And the rationale reports 2 candidates matched
    And the rationale reports 2 candidates ready
    And the rationale mode is SELECTION_MODE_METRICS_WEIGHTED
    And the rationale winning_factors are not empty

  Scenario: The selected agent carries a dialable endpoint
    Given a ready agent "agent-reviewer-a" declaring capability "code-review"
    When SelectAgent is called with capability_name "code-review"
    Then the selected agent is "agent-reviewer-a"
    And the selected agent endpoint is not empty

  # ─── Error contract ───────────────────────────────────────────────────────

  Scenario: Empty capability name is rejected
    When SelectAgent is called with capability_name ""
    Then the call fails with INVALID_ARGUMENT

  Scenario: No agent declares the capability
    Given a ready agent "agent-reviewer-a" declaring capability "code-review"
    When SelectAgent is called with capability_name "quantum-compile"
    Then the call fails with NOT_FOUND
    And the error message mentions "quantum-compile"

  Scenario: Hard constraints eliminate every candidate
    Given a ready agent "agent-reviewer-a" declaring capability "code-review"
    And agent "agent-reviewer-a" declares no gpu
    When SelectAgent is called with capability_name "code-review" requiring gpu
    Then the call fails with FAILED_PRECONDITION
    And the error message mentions "gpu"

  # ─── Readiness (the stale-liveness fix) ───────────────────────────────────

  Scenario: A not-ready agent is never selected
    Given a ready agent "agent-reviewer-a" declaring capability "code-review"
    And a not-ready agent "agent-reviewer-dead" declaring capability "code-review"
    When SelectAgent is called with capability_name "code-review"
    Then the selected agent is "agent-reviewer-a"
    And the rationale reports 2 candidates matched
    And the rationale reports 1 candidates ready

  Scenario: All candidates not ready fails rather than dispatching blind
    Given a not-ready agent "agent-reviewer-dead" declaring capability "code-review"
    When SelectAgent is called with capability_name "code-review"
    Then the call fails with FAILED_PRECONDITION
    And the error message mentions "ready"

  # ─── Expert targeting is strict (ADR-028) ─────────────────────────────────

  Scenario: Expert target restricts eligibility to declared experts
    Given a ready agent "agent-sec" declaring capability "code-review" with expert scope "security-reviewer"
    And a ready agent "agent-generalist" declaring capability "code-review"
    When SelectAgent is called with capability_name "code-review" and expert_target "security-reviewer"
    Then the selected agent is "agent-sec"
    And the rationale reports 1 candidates after expert filter

  Scenario: Expert target never falls back to non-expert agents
    Given a ready agent "agent-generalist" declaring capability "code-review"
    When SelectAgent is called with capability_name "code-review" and expert_target "security-reviewer"
    Then the call fails with FAILED_PRECONDITION
    And the error message mentions "security-reviewer"

  # ─── Metrics degradation (ADR-039 §3: never fail on metrics) ─────────────

  Scenario: Metrics backend unavailable degrades the mode, not the call
    Given a ready agent "agent-reviewer-a" declaring capability "code-review"
    And a ready agent "agent-reviewer-b" declaring capability "code-review"
    And the metrics backend is unavailable
    When SelectAgent is called with capability_name "code-review"
    Then the response contains exactly one agent
    And the rationale mode is SELECTION_MODE_DEGRADED_ROUND_ROBIN

  Scenario: Explicit round-robin policy reports its own mode
    Given a ready agent "agent-reviewer-a" declaring capability "code-review"
    When SelectAgent is called with capability_name "code-review" and policy SELECTION_POLICY_ROUND_ROBIN
    Then the response contains exactly one agent
    And the rationale mode is SELECTION_MODE_ROUND_ROBIN
