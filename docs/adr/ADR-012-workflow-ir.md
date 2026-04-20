# ADR-012: Canonical Workflow Intermediate Representation (IR)

**Status:** Accepted  **Date:** 2025-04-01

## Decision
Introduce a Canonical Workflow IR — an engine-agnostic intermediate representation
that sits between YAML (Layer 1) and workflow engines (Layer 3).

## Rationale
Without IR:
- Temporal adapter must parse YAML directly — coupled to schema changes
- Adding a new engine requires new YAML format — breaks existing workflows
- Semantic mismatches between engines are resolved ad-hoc in each adapter

With IR:
- Compiler owns schema parsing — adapters receive typed IR structs
- Adding a new engine = one new IR→engine translation — YAML unchanged
- Semantic normalisation happens once (in compiler) — adapters are thin

## IR Design Principles
- Engine-agnostic: no Temporal/Argo/LangGraph concepts in IR types
- Stable: IR types change only with ADR and major version bump
- Complete: captures all workflow semantics expressible in YAML v1

## What IR Contains
States, transitions, actions (capability calls), triggers, timeout specs, guards
