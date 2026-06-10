# SPDX-License-Identifier: Apache-2.0
# Zynax — gRPC Health Checking Protocol BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes the contract every Zynax gRPC service must honour by exposing
# the standard grpc.health.v1.Health service.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: Kubernetes 1.24+ native gRPC probes (livenessProbe.grpc /
# readinessProbe.grpc) and grpc-health-probe rely on the server implementing
# grpc.health.v1.Health. Services must report SERVING for both the overall ""
# key and a per-service named key on startup, and must flip to NOT_SERVING
# before GracefulStop() so load balancers drain in-flight requests during
# rolling restarts. (issue #656, ADR-016)

Feature: gRPC Health Checking Protocol contract — Kubernetes-native probes
  As a platform engineer deploying Zynax to Kubernetes
  I want every gRPC service to implement grpc.health.v1.Health
  So that native gRPC probes and grpc-health-probe work without a sidecar

  Background:
    Given a gRPC service with the standard Health server is running

  Scenario: Service reports SERVING on the overall key at startup
    When a HealthCheckRequest is sent with service ""
    Then the health status is SERVING

  Scenario: Service reports SERVING on its per-service named key at startup
    When a HealthCheckRequest is sent with the service's named key
    Then the health status is SERVING

  Scenario: Unknown service name is reported NOT_FOUND
    When a HealthCheckRequest is sent with service "does.not.Exist"
    Then the gRPC health status is NOT_FOUND

  Scenario: Service reports NOT_SERVING after a graceful shutdown signal
    When the service receives a graceful shutdown signal
    And a HealthCheckRequest is sent with service ""
    Then the health status is NOT_SERVING
