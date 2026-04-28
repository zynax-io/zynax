# SPDX-License-Identifier: Apache-2.0
# Zynax — AgentRegistryService BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes the contract the agent registry must honour.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: Agents and adapters announce themselves and their
# capabilities on startup. The task broker queries this registry at dispatch
# time to find which agents can handle a given capability. Without a registry
# contract, capability routing is impossible and agents are invisible to the
# platform. (ADR-001, ADR-013)

Feature: AgentRegistryService contract — agent registration and capability discovery
  As a task broker routing work to capability providers
  I want agents to register their capabilities and be discoverable by name
  So that work can be dispatched without hardcoded agent addresses or identities

  Background:
    Given an AgentRegistryService is running on a test gRPC server
    And the registry is empty

  # ─── Registration ─────────────────────────────────────────────────────────

  Scenario: Register an agent with capabilities
    Given a valid AgentDef with agent_id "agent-summarizer"
    And the AgentDef declares capabilities ["summarize", "extract_keywords"]
    And the AgentDef endpoint is "localhost:50051"
    When RegisterAgent is called with the AgentDef
    Then the response contains agent_id "agent-summarizer"
    And GetAgent for "agent-summarizer" returns status REGISTERED
    And GetAgent for "agent-summarizer" returns both declared capabilities
    And GetAgent for "agent-summarizer" returns a non-zero registered_at timestamp

  Scenario: Registered agent is immediately discoverable by capability
    Given agent "agent-summarizer" is registered with capability "summarize"
    When FindByCapability is called with capability_name "summarize"
    Then the response contains agent "agent-summarizer"

  Scenario: Registration preserves labels for selector queries
    Given a valid AgentDef with agent_id "agent-prod"
    And the AgentDef has labels {"env": "production", "tier": "standard"}
    When RegisterAgent is called with the AgentDef
    And ListAgents is called with label selector "env=production"
    Then the response contains agent "agent-prod"

  # ─── Discovery ────────────────────────────────────────────────────────────

  Scenario: FindByCapability returns only agents that declare the capability
    Given agent "agent-a" is registered with capabilities ["summarize", "translate"]
    And agent "agent-b" is registered with capabilities ["review_code"]
    When FindByCapability is called with capability_name "summarize"
    Then the response contains agent "agent-a"
    And the response does not contain agent "agent-b"

  Scenario: FindByCapability with no matching agents returns an empty list
    Given agent "agent-a" is registered with capability "review_code"
    When FindByCapability is called with capability_name "summarize"
    Then the response contains no agents
    And the gRPC status is OK

  Scenario: ListAgents returns all registered agents when no filter is given
    Given agent "agent-a" is registered with capability "summarize"
    And agent "agent-b" is registered with capability "review_code"
    When ListAgents is called with no label selector
    Then the response contains agent "agent-a"
    And the response contains agent "agent-b"

  Scenario: ListAgents with label selector returns only matching agents
    Given agent "agent-prod" is registered with labels {"env": "production"}
    And agent "agent-staging" is registered with labels {"env": "staging"}
    When ListAgents is called with label selector "env=production"
    Then the response contains agent "agent-prod"
    And the response does not contain agent "agent-staging"

  Scenario: GetAgent returns the full AgentDef including capabilities and labels
    Given agent "agent-full" is registered with capability "summarize"
    And the AgentDef has labels {"owner": "team-ai"}
    When GetAgent is called with agent_id "agent-full"
    Then the response agent_id is "agent-full"
    And the response includes capability "summarize"
    And the response includes label "owner" with value "team-ai"
    And the response status is REGISTERED

  # ─── Deregistration ───────────────────────────────────────────────────────

  Scenario: Deregister a registered agent
    Given agent "agent-123" is registered with capability "summarize"
    When DeregisterAgent is called with agent_id "agent-123"
    Then the gRPC status is OK
    And GetAgent for "agent-123" returns status DEREGISTERED

  Scenario: Deregistered agent is no longer returned by FindByCapability
    Given agent "agent-123" is registered with capability "summarize"
    And DeregisterAgent has been called for "agent-123"
    When FindByCapability is called with capability_name "summarize"
    Then the response does not contain agent "agent-123"

  Scenario: Deregistered agent is no longer returned by ListAgents
    Given agent "agent-123" is registered with capability "summarize"
    And DeregisterAgent has been called for "agent-123"
    When ListAgents is called with no label selector
    Then the response does not contain agent "agent-123"

  # ─── Duplicate and conflict handling ──────────────────────────────────────

  Scenario: Registering an agent_id that is already registered is rejected
    Given agent "agent-abc" is registered with capability "summarize"
    When RegisterAgent is called again with agent_id "agent-abc"
    Then the gRPC status is ALREADY_EXISTS
    And the error message contains "agent-abc"

  # ─── Not found ────────────────────────────────────────────────────────────

  Scenario: GetAgent for an unknown agent_id returns NOT_FOUND
    When GetAgent is called with agent_id "nonexistent-agent"
    Then the gRPC status is NOT_FOUND
    And the error message contains "nonexistent-agent"

  Scenario: DeregisterAgent for an unknown agent_id returns NOT_FOUND
    When DeregisterAgent is called with agent_id "ghost-agent"
    Then the gRPC status is NOT_FOUND
    And the error message contains "ghost-agent"

  # ─── Input validation ─────────────────────────────────────────────────────

  Scenario: RegisterAgent with empty agent_id is rejected
    Given a RegisterAgentRequest with agent_id set to ""
    When RegisterAgent is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "agent_id"

  Scenario: RegisterAgent with empty endpoint is rejected
    Given a RegisterAgentRequest with endpoint set to ""
    When RegisterAgent is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "endpoint"

  Scenario: RegisterAgent with an empty capability name is rejected
    Given a RegisterAgentRequest where one CapabilityDef has name set to ""
    When RegisterAgent is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "capability_name"

  Scenario: FindByCapability with empty capability_name is rejected
    Given a FindByCapabilityRequest with capability_name set to ""
    When FindByCapability is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "capability_name"

  Scenario: GetAgent with empty agent_id is rejected
    Given a GetAgentRequest with agent_id set to ""
    When GetAgent is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "agent_id"

  # ─── CapabilityDef schema contract ────────────────────────────────────────

  Scenario: CapabilityDef with non-JSON input_schema is rejected
    Given a RegisterAgentRequest where one CapabilityDef has input_schema "not valid json"
    When RegisterAgent is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "input_schema"

  Scenario: CapabilityDef with non-JSON output_schema is rejected
    Given a RegisterAgentRequest where one CapabilityDef has output_schema "not valid json"
    When RegisterAgent is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "output_schema"

  # ─── Pagination ───────────────────────────────────────────────────────────

  Scenario: ListAgents first page returns page_size results and a next_page_token
    Given 5 agents are registered with capability "summarize"
    When ListAgents is called with page_size 3 and no page_token
    Then the response contains exactly 3 agents
    And the response next_page_token is non-empty

  Scenario: ListAgents subsequent page returns remaining results
    Given 5 agents are registered with capability "summarize"
    And ListAgents has been called with page_size 3 returning next_page_token "tok-1"
    When ListAgents is called with page_size 3 and page_token "tok-1"
    Then the response contains exactly 2 agents
    And the response next_page_token is empty

  Scenario: ListAgents last page has empty next_page_token
    Given 2 agents are registered with capability "summarize"
    When ListAgents is called with page_size 10 and no page_token
    Then the response contains exactly 2 agents
    And the response next_page_token is empty

  Scenario: ListAgents with page_size 0 uses server default
    Given 3 agents are registered with capability "summarize"
    When ListAgents is called with page_size 0 and no page_token
    Then the gRPC status is OK
    And the response contains at least 1 agent
