# SPDX-License-Identifier: Apache-2.0
# Zynax — Policy Enforcement BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation
# (ADR-016: contracts before code). It describes the policy-enforcement
# contract the control plane must honour for RoutingPolicy and RateLimit
# (protos/zynax/v1/policy.proto). See protos/AGENTS.md §7 for contract rules.
#
# CapabilityQuota scenarios REMOVED (ADR-045 §2, M8.G #1636): the compiler's
# quota check was never enforced in production (the gate was always built with
# a nil invocation counter) and was deleted rather than delegated. The quota
# contract deliberately does NOT migrate to the engine-adapter QuotaChecker
# while that component has no production caller — a green contract against
# dead code would advertise protection that does not exist (ADR-020). Restore
# a quota contract only when a live enforcement path exists.
#
# The RateLimit scenarios are enforced at the Envoy Gateway edge since M8.F
# (ADR-044); the RoutingPolicy scenarios cover the compiler's REST-path
# dual-guard (ADR-045 §3 — the CR path is guarded by a
# ValidatingAdmissionPolicy on the Workflow CR).
#
# Business context: the control plane needs an enforcement layer so that a
# single badly-behaved workflow cannot saturate the api-gateway or route work
# to an engine a namespace is not allowed to use. (REASONS Canvas #768,
# partially superseded by docs/spdd/1575-admission-policy/canvas.md)

Feature: Policy enforcement — rate limits and routing policies
  As a platform operator protecting shared control-plane capacity
  I want routing policies and rate limits enforced
  So that no single workflow or namespace can starve or misuse the platform

  Background:
    Given the api-gateway is running with policy enforcement enabled
    And the workflow-compiler enforces namespace policy configuration

  # ─── RateLimit: per-source-IP token bucket → HTTP 429 ──────────────────────

  Scenario: Requests exceeding the per-IP rate limit are rejected with 429
    Given a RateLimit of 5 requests_per_second with burst 5 for the api-gateway
    And source IP "203.0.113.10" has exhausted its token bucket
    When source IP "203.0.113.10" sends POST "/api/v1/apply"
    Then the response HTTP status is 429
    And the response body contains code "RATE_LIMITED"

  Scenario: Requests within the per-IP rate limit are accepted
    Given a RateLimit of 5 requests_per_second with burst 5 for the api-gateway
    And source IP "203.0.113.11" has made no requests
    When source IP "203.0.113.11" sends POST "/api/v1/apply"
    Then the response HTTP status is 202

  # ─── RoutingPolicy: allowed engines → PERMISSION_DENIED ────────────────────

  Scenario: Workflow targeting an engine outside the routing policy is denied
    Given a RoutingPolicy for namespace "team-c" allowing engines "temporal"
    When a workflow for namespace "team-c" requests engine "argo"
    Then the gRPC status is PERMISSION_DENIED
    And the error message contains "argo"
    And no WorkflowIR is emitted

  Scenario: Workflow targeting an allowed engine passes the routing policy
    Given a RoutingPolicy for namespace "team-c" allowing engines "temporal"
    When a workflow for namespace "team-c" requests engine "temporal"
    Then the routing policy permits the workflow
    And a WorkflowIR is emitted
