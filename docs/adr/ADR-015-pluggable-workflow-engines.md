# ADR-015: Pluggable Workflow Engines

**Status:** Accepted  **Date:** 2025-04-01

## Decision
Zynax does NOT build a workflow engine.
Instead, it builds a `WorkflowEngine` interface with pluggable backends.

## Rationale
Building a workflow engine is a massive, multi-year effort:
- Temporal took years to build and battle-test
- Argo, Prefect, Airflow — each is a significant project

The `WorkflowEngine` interface abstracts this complexity:
- Temporal: durable execution, activities, signals, queries
- LangGraph: Python-native, graph-based, good for agent workflows
- Argo: Kubernetes-native, YAML workflows, GitOps-friendly

## Engine Selection
- Temporal is the primary target (M3) — best durability/reliability story
- LangGraph is secondary (M5) — best for pure Python agent workflows
- Argo is tertiary (M6) — best for K8s-native deployments

Engine selected via config (`ZYNAX_ENGINE_ACTIVE_ENGINE`).
No code change required to swap engines.

## What Zynax Owns vs Engines
Zynax owns: IR, routing, capability dispatch, observability, YAML API
Engines own: durable execution, activity retry, workflow persistence, scheduling
