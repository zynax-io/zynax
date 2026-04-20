# SPDX-License-Identifier: Apache-2.0
Feature: Agent Memory

  Background:
    Given the memory service is running

  Scenario: Set and get a key-value entry
    Given namespace "agent:test-01" exists
    When entry key="context" value="hello world" is set
    Then GetEntry for key="context" returns "hello world"

  Scenario: Entry expires after TTL
    Given an entry with ttl=1s is set in namespace "agent:ttl-test"
    When 2 seconds pass
    Then GetEntry returns NOT_FOUND

  Scenario: Cannot write to non-existent namespace
    When SetEntry is called for namespace "agent:ghost"
    Then the gRPC status is NOT_FOUND

  Scenario: Vector dimension mismatch is rejected
    Given namespace "agent:vec-01" has vector_dimensions=1536
    When UpsertVector is called with a 768-dimension embedding
    Then the gRPC status is INVALID_ARGUMENT

  Scenario: Semantic search returns results by similarity score
    Given 3 vectors are upserted with cosine similarities [0.9, 0.7, 0.4] to the query
    When SearchSimilar is called with top_k=2 min_score=0.5
    Then 2 results are returned ordered by score descending

  Scenario: DeleteNamespace cascades to all entries and vectors
    Given namespace "agent:del-test" has 3 KV entries and 2 vectors
    When the namespace is deleted
    Then GetEntry returns NOT_FOUND for all 3 keys
    And SearchSimilar returns 0 results
