# SPDX-License-Identifier: Apache-2.0
# Zynax — MemoryService BDD Contract Specification
#
# This file is the SPECIFICATION. It is written BEFORE the implementation.
# It describes the contract the memory service must honour.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: Long-running AI workflows need shared context across agent
# invocations — storing intermediate results, passing state between steps, and
# enabling semantic retrieval of prior outputs. Agents must not hold state
# internally (twelve-factor); all shared context flows through this contract.
# Every operation is scoped to a workflow_id for strict isolation. (ADR-001)

Feature: MemoryService contract — shared KV and vector storage
  As an agent executing a capability within a workflow
  I want to read and write shared context scoped to my workflow run
  So that downstream agents in the same workflow can access prior results

  Background:
    Given a MemoryService is running on a test gRPC server

  # ─── Key-value store ──────────────────────────────────────────────────────

  Scenario: Store and retrieve a key-value entry
    Given workflow "wf-42" is running
    When Set is called with key "summary" value "the PR looks good" scoped to "wf-42"
    And Get is called with key "summary" scoped to "wf-42"
    Then the response value is "the PR looks good"

  Scenario: KV entries are isolated by workflow_id
    Given Set is called with key "result" value "alpha" scoped to workflow "wf-01"
    When Get is called with key "result" scoped to workflow "wf-02"
    Then the gRPC status is NOT_FOUND

  Scenario: Get for an unknown key returns NOT_FOUND
    When Get is called with key "nonexistent" scoped to workflow "wf-42"
    Then the gRPC status is NOT_FOUND
    And the error message contains "nonexistent"

  Scenario: Set overwrites the value for an existing key
    Given Set has been called with key "count" value "1" scoped to "wf-42"
    When Set is called again with key "count" value "2" scoped to "wf-42"
    And Get is called with key "count" scoped to "wf-42"
    Then the response value is "2"

  Scenario: Delete removes a key-value entry
    Given Set has been called with key "result" value "done" scoped to "wf-42"
    When Delete is called with key "result" scoped to "wf-42"
    Then Get for key "result" scoped to "wf-42" returns NOT_FOUND

  Scenario: Delete on an unknown key returns NOT_FOUND
    When Delete is called with key "ghost" scoped to "wf-42"
    Then the gRPC status is NOT_FOUND

  Scenario: ListKeys returns all keys for a workflow
    Given Set has been called with key "a" scoped to "wf-42"
    And Set has been called with key "b" scoped to "wf-42"
    And Set has been called with key "c" scoped to "wf-99"
    When ListKeys is called scoped to workflow "wf-42"
    Then the response contains key "a"
    And the response contains key "b"
    And the response does not contain key "c"

  Scenario: ListKeys with a prefix filter returns only matching keys
    Given Set has been called with key "doc:1" scoped to "wf-42"
    And Set has been called with key "doc:2" scoped to "wf-42"
    And Set has been called with key "meta:1" scoped to "wf-42"
    When ListKeys is called scoped to "wf-42" with prefix "doc:"
    Then the response contains key "doc:1"
    And the response contains key "doc:2"
    And the response does not contain key "meta:1"

  Scenario: ListKeys for a workflow with no keys returns an empty list
    When ListKeys is called scoped to workflow "wf-empty"
    Then the response contains no keys
    And the gRPC status is OK

  # ─── TTL expiry ───────────────────────────────────────────────────────────

  Scenario: KV entry with ttl_seconds expires after the TTL elapses
    Given Set is called with key "temp" value "x" ttl_seconds 1 scoped to "wf-42"
    When 2 seconds elapse
    And Get is called with key "temp" scoped to "wf-42"
    Then the gRPC status is NOT_FOUND

  Scenario: KV entry without ttl_seconds does not expire
    Given Set is called with key "permanent" value "y" and no TTL scoped to "wf-42"
    When 2 seconds elapse
    And Get is called with key "permanent" scoped to "wf-42"
    Then the response value is "y"

  # ─── Vector store ─────────────────────────────────────────────────────────

  Scenario: Store a vector embedding and retrieve by similarity
    Given a vector with embedding [0.1, 0.2, 0.3] and text "fix the auth bug"
    When StoreVector is called with the embedding scoped to "wf-42"
    And QueryVector is called with embedding [0.11, 0.19, 0.31] top_k 1 scoped to "wf-42"
    Then the response contains 1 VectorResult
    And the top VectorResult text is "fix the auth bug"
    And the top VectorResult similarity_score is greater than 0.95

  Scenario: QueryVector returns results ranked by similarity descending
    Given vectors "A" "B" "C" are stored with varying similarity to the query
    When QueryVector is called with top_k 3
    Then the results are ordered by similarity_score descending

  Scenario: QueryVector top_k limits the number of results returned
    Given 10 vectors are stored scoped to "wf-42"
    When QueryVector is called with top_k 3 scoped to "wf-42"
    Then the response contains exactly 3 VectorResults

  Scenario: Vector entries are isolated by workflow_id
    Given a vector is stored scoped to workflow "wf-01"
    When QueryVector is called scoped to workflow "wf-02"
    Then the response contains no VectorResults
    And the gRPC status is OK

  Scenario: QueryVector with no stored vectors returns an empty list
    When QueryVector is called scoped to workflow "wf-empty" with top_k 5
    Then the response contains no VectorResults
    And the gRPC status is OK

  Scenario: DeleteVector removes a stored embedding
    Given a vector with id "vec-001" is stored scoped to "wf-42"
    When DeleteVector is called with id "vec-001" scoped to "wf-42"
    And QueryVector is called with a similar embedding scoped to "wf-42"
    Then the response does not contain vector id "vec-001"

  Scenario: DeleteVector on an unknown id returns NOT_FOUND
    When DeleteVector is called with id "ghost-vec" scoped to "wf-42"
    Then the gRPC status is NOT_FOUND

  # ─── Input validation ─────────────────────────────────────────────────────

  Scenario: Set with empty workflow_id is rejected
    Given a SetRequest with workflow_id set to ""
    When Set is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "workflow_id"

  Scenario: Set with empty key is rejected
    Given a SetRequest with key set to ""
    When Set is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "key"

  Scenario: StoreVector with empty embedding is rejected
    Given a StoreVectorRequest with no embedding values
    When StoreVector is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "embedding"

  Scenario: StoreVector with empty workflow_id is rejected
    Given a StoreVectorRequest with workflow_id set to ""
    When StoreVector is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "workflow_id"

  Scenario: QueryVector with top_k of zero is rejected
    Given a QueryVectorRequest with top_k set to 0
    When QueryVector is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "top_k"

  # ─── MGet ─────────────────────────────────────────────────────────────────

  Scenario: MGet returns values for all existing keys
    Given keys "ctx:a" and "ctx:b" are set for workflow "wf-mget" with values "va" and "vb"
    When MGet is called for workflow "wf-mget" with keys ["ctx:a", "ctx:b"]
    Then the gRPC status is OK
    And the response contains entry with key "ctx:a" and value "va"
    And the response contains entry with key "ctx:b" and value "vb"

  Scenario: MGet silently omits missing keys
    Given key "ctx:exists" is set for workflow "wf-mget-miss" with value "v1"
    When MGet is called for workflow "wf-mget-miss" with keys ["ctx:exists", "ctx:missing"]
    Then the gRPC status is OK
    And the response contains entry with key "ctx:exists"
    And the response does not contain entry with key "ctx:missing"

  Scenario: MGet with empty workflow_id is rejected
    Given an MGetRequest with workflow_id set to ""
    When MGet is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "workflow_id"

  Scenario: MGet with empty keys list is rejected
    Given an MGetRequest for workflow "wf-mget-empty" with no keys
    When MGet is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "keys"

  # ─── MSet ─────────────────────────────────────────────────────────────────

  Scenario: MSet stores multiple entries and returns correct count
    When MSet is called for workflow "wf-mset" with entries key "k1" value "v1" and key "k2" value "v2"
    Then the gRPC status is OK
    And the response count is 2
    And Get for workflow "wf-mset" key "k1" returns "v1"
    And Get for workflow "wf-mset" key "k2" returns "v2"

  Scenario: MSet entries with TTL are retrievable before expiry
    When MSet is called for workflow "wf-mset-ttl" with entry key "k-ttl" value "v-ttl" ttl 3600
    Then the gRPC status is OK
    And Get for workflow "wf-mset-ttl" key "k-ttl" returns "v-ttl"

  Scenario: MSet with empty workflow_id is rejected
    Given an MSetRequest with workflow_id set to ""
    When MSet is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "workflow_id"

  Scenario: MSet with empty entries list is rejected
    Given an MSetRequest for workflow "wf-mset-empty" with no entries
    When MSet is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "entries"

  # ─── DeleteNamespace ──────────────────────────────────────────────────────

  Scenario: DeleteNamespace removes all KV entries for a workflow
    Given keys "ns:a" and "ns:b" are set for workflow "wf-del-ns"
    When DeleteNamespace is called for workflow "wf-del-ns"
    Then the gRPC status is OK
    And the response deleted_count is 2
    And Get for workflow "wf-del-ns" key "ns:a" returns NOT_FOUND
    And Get for workflow "wf-del-ns" key "ns:b" returns NOT_FOUND

  Scenario: DeleteNamespace on an empty namespace returns OK with count zero
    When DeleteNamespace is called for workflow "wf-empty-ns"
    Then the gRPC status is OK
    And the response deleted_count is 0

  Scenario: DeleteNamespace with empty workflow_id is rejected
    Given a DeleteNamespaceRequest with workflow_id set to ""
    When DeleteNamespace is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error message mentions "workflow_id"
