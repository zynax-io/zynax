# SPDX-License-Identifier: Apache-2.0
# Zynax — Agent Registry BDD Feature File (CRD era)
#
# ADR-039 / #1584: the Agent custom resource (zynax.io/v1alpha1) is the single
# source of truth for agent identity and SchedulerService.SelectAgent is the
# dispatch surface. Every push-era AgentRegistryService RPC is deprecated and
# answers UNIMPLEMENTED with a migration pointer until its M9 hard removal.
# The push-era behavioural contract remains specified (for M9 reference) in
# protos/tests/features/agent_registry_service.feature.

Feature: Agent Registry retirement (ADR-039)
  As a platform operator migrating to CRD-native agent identity
  I want every push-era registry RPC to answer UNIMPLEMENTED with guidance
  So that legacy callers fail fast toward the Agent CR migration path

  Background:
    Given the agent registry is running and healthy

  Scenario Outline: Push-era RPC answers UNIMPLEMENTED with the migration pointer
    When the <rpc> RPC is called
    Then the call fails with code UNIMPLEMENTED
    And the error message mentions "ADR-039"
    And the error message mentions "agent-crd-migration"

    Examples:
      | rpc              |
      | RegisterAgent    |
      | DeregisterAgent  |
      | GetAgent         |
      | ListAgents       |
      | FindByCapability |
