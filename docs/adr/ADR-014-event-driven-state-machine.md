# ADR-014: Event-Driven State Machines, Not DAGs

**Status:** Accepted  **Date:** 2025-04-01

## Decision
Keel workflows are event-driven state machines.
They are NOT DAGs (directed acyclic graphs).

## Rationale
DAGs fail for real AI workflows:
- Loops (fix → review → fix) are impossible in acyclic graphs
- Human-in-the-loop requires pausing the graph — awkward
- Long-running (days) requires external state — DAG has none
- Async events as triggers — not natively supported

State machines support all of these natively:
- Loops: states can transition back to previous states
- Human-in-the-loop: WAITING state type with signal-based resume
- Long-running: state persists indefinitely until an event arrives
- Async events: every state transition is triggered by an event

## Inspiration
- XState (JavaScript state machine library)
- AWS Step Functions
- Temporal workflows (durable execution)

## What This Means for Adapters
Temporal is the preferred engine because it natively models durable, event-driven
execution. LangGraph graphs are acyclic by default but support cycles — the
LangGraph adapter handles translation.
