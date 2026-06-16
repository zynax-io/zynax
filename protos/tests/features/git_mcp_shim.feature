# SPDX-License-Identifier: Apache-2.0
# Zynax — git-adapter MCP shim BDD Contract Specification
#
# This file is the SPECIFICATION. It describes the contract the MCP shim must
# honour. It is a thin Model Context Protocol surface over the existing
# git-adapter capability handlers — NO Git logic is reimplemented here.
# See protos/AGENTS.md §7 for contract test rules.
#
# Business context: an authoring agent (e.g. Claude Code) drives Git operations
# over MCP tools. Each tool maps 1:1 onto an existing git-adapter capability
# (open_pr / request_review / get_diff), so there is a single audited Git and
# credential surface. (ADR-032 — one Git implementation, two surfaces.)
#
# Canvas: docs/spdd/1169-git-mcp-shim/canvas.md  · Step G.2
# Parent epic: #1169 (Git MCP shim over git-adapter) · Story: #1198

Feature: git-adapter MCP shim — Git capabilities as MCP tools
  As an authoring agent
  I want the git-adapter capabilities exposed as MCP tools
  So that I can open PRs, request reviews, and fetch diffs over MCP
  without a second Git implementation or a second credential surface

  Background:
    Given a git-adapter configured with capabilities "open_pr", "request_review", "get_diff"
    And the MCP shim is started over stdio with that configuration

  # ─── tool discovery (explicit allow-list) ────────────────────────────────────

  Scenario: tools/list advertises exactly the configured capabilities
    When the client sends a "tools/list" request
    Then the response lists exactly the tools "open_pr", "request_review", "get_diff"
    And no tool outside the configured allow-list is advertised
    And each advertised tool carries the adapter's input schema and description

  # ─── 1:1 dispatch into existing handlers ─────────────────────────────────────

  Scenario: tools/call open_pr dispatches 1:1 into the adapter handler
    Given a "tools/call" request for tool "open_pr" with arguments title, head, base
    When the request is dispatched
    Then exactly the "open_pr" adapter capability is executed once
    And the tool result text is the adapter's COMPLETED payload verbatim
    And isError is false

  Scenario: PROGRESS events from the adapter are not surfaced to the MCP caller
    Given the adapter emits PROGRESS before COMPLETED for "get_diff"
    When a "tools/call" request for tool "get_diff" is dispatched
    Then the MCP response contains only the terminal result
    And no PROGRESS event leaks into the tool result

  Scenario: a FAILED adapter capability becomes an MCP tool error
    Given the adapter returns a FAILED TaskEvent with code "INVALID_INPUT"
    When a "tools/call" request for tool "open_pr" is dispatched
    Then isError is true
    And the tool result text contains the CapabilityError code and message

  # ─── tool-surface authz (allow-list guard) ───────────────────────────────────

  Scenario: a tool outside the allow-list is rejected before dispatch
    Given a "tools/call" request for tool "delete_repo"
    When the request is dispatched
    Then the response is a JSON-RPC error with code -32601
    And the adapter is never invoked for that tool

  # ─── arg / SSRF safety (target pinned in adapter config) ─────────────────────

  Scenario: the caller cannot redirect the target repository via tool arguments
    Given a "tools/call" request for tool "open_pr" with an "owner" or "repo" argument
    When the request is dispatched
    Then the adapter still targets the owner and repo pinned in its configuration
    And no caller-supplied owner, repo, or remote reaches the privileged Git call

  # ─── JSON-RPC framing ────────────────────────────────────────────────────────

  Scenario: initialize returns the MCP protocol version and server info
    When the client sends an "initialize" request
    Then the response carries a non-empty protocolVersion
    And the response advertises the "tools" capability

  Scenario: a notification produces no response
    When the client sends a "notifications/initialized" notification
    Then the shim produces no JSON-RPC response

  Scenario: an unknown method returns method-not-found
    When the client sends a request for method "resources/list"
    Then the response is a JSON-RPC error with code -32601
