# SPDX-License-Identifier: Apache-2.0
Feature: Agent Memory

  Background:
    Given the memory service is running

  Scenario: Set and get a key-value entry
    Given workflow "agent:test-01" is active
    When entry key="context" value="hello world" is set for workflow "agent:test-01"
    Then GetEntry for key="context" in workflow "agent:test-01" returns "hello world"

  Scenario: Entry expires after TTL
    Given an entry with key="tmp" value="x" ttl=1s is set in workflow "agent:ttl-test"
    When 2 seconds pass
    Then GetEntry for key="tmp" in workflow "agent:ttl-test" returns NOT_FOUND

  Scenario: Get a key from a workflow with no entries returns NOT_FOUND
    When GetEntry is called for key="missing" in workflow "agent:ghost"
    Then the gRPC status is NOT_FOUND

  Scenario: Vector namespace isolation — vectors do not cross workflow boundaries
    Given a vector is stored in workflow "agent:vec-a"
    When SearchSimilar is called in workflow "agent:vec-b" with top_k=5
    Then 0 results are returned

  Scenario: Semantic search returns results by similarity score
    Given 3 vectors are stored in workflow "agent:search-01" with cosine similarities [0.9, 0.7, 0.4] to the query
    When SearchSimilar is called in workflow "agent:search-01" with top_k=2
    Then 2 results are returned ordered by score descending

  Scenario: DeleteNamespace cascades to all entries and vectors
    Given workflow "agent:del-test" has 3 KV entries and 2 vectors
    When the namespace "agent:del-test" is deleted
    Then GetEntry for key="ns-key-0" in workflow "agent:del-test" returns NOT_FOUND
    And SearchSimilar in workflow "agent:del-test" returns 0 results
