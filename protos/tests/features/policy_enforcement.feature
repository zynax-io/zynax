# SPDX-License-Identifier: Apache-2.0
# Zynax — Policy Enforcement BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation
# (ADR-016: contracts before code). It describes the policy-enforcement
# contract the control plane must honour once RoutingPolicy, RateLimit, and
# CapabilityQuota (protos/zynax/v1/policy.proto) are enforced.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: the control plane needs an enforcement layer so that a
# single badly-behaved workflow cannot saturate the api-gateway, exhaust
# task-broker capacity, or route work to an engine a namespace is not allowed
# to use. The three policy primitives — RateLimit (per-source-IP token
# bucket), CapabilityQuota (concurrent invocations per namespace), and
# RoutingPolicy (allowed engines per namespace) — are static config in M6
# (no admin API). (REASONS Canvas #768)

Feature: Policy enforcement — rate limits, capability quotas, and routing policies
  As a platform operator protecting shared control-plane capacity
  I want routing policies, rate limits, and capability quotas enforced
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

  # ─── CapabilityQuota: concurrent invocations → RESOURCE_EXHAUSTED ──────────

  Scenario: Submission exceeding the namespace capability quota is rejected
    Given a CapabilityQuota of 2 max_concurrent for namespace "team-a"
    And namespace "team-a" already has 2 capability invocations in flight
    When a workflow for namespace "team-a" is submitted for compilation
    Then the gRPC status is RESOURCE_EXHAUSTED
    And the error message contains "team-a"
    And no WorkflowIR is emitted

  Scenario: Submission within the namespace capability quota is admitted
    Given a CapabilityQuota of 2 max_concurrent for namespace "team-b"
    And namespace "team-b" has 0 capability invocations in flight
    When a workflow for namespace "team-b" is submitted for compilation
    Then the workflow is compiled successfully
    And a WorkflowIR is emitted

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
