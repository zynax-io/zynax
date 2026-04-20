# ADR-011: Declarative YAML as Primary Control Interface

**Status:** Accepted  **Date:** 2025-04-01

## Decision
YAML manifests (Kubernetes-style) are the primary user interface for Keel.
Users define workflows, agents, and policies in YAML — not code.

## Rationale
- Reproducibility: workflows are versionable, diffable, auditable
- Familiarity: the Kubernetes ecosystem has established YAML as the standard for declarative infrastructure
- Abstraction: decouples intent from execution engine
- GitOps: YAML in Git = audit trail + rollback

## Manifest Kinds (v1)
- `Workflow` — event-driven state machine definition
- `AgentDef` — capability provider declaration
- `Policy` — routing and scheduling policy
- `RoutingRule` — capability dispatch rule

## Rules
- YAML is NEVER imported by Go/Python code — it is compiled by workflow-compiler
- Breaking schema changes require new apiVersion (e.g. `keel.io/v2`)
- All manifest kinds have JSON Schema in `spec/schemas/`
