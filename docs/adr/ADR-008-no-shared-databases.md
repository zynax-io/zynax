# ADR-008: No Shared Databases Between Services (Amazon API Mandate)

**Status:** Accepted  **Date:** 2025-04-01

## Decision
Each service owns its own database schema. No two services share a database
instance in production (they may share an instance in local dev for convenience,
but must use separate schemas/databases).

## Rationale
This is the Amazon API Mandate applied to data storage:
- Independent deployability: a service can be updated without coordinating DB migrations.
- Independent scalability: each service DB can be sized and scaled independently.
- Failure isolation: a DB outage in one service does not cascade.
- Clear ownership: "who owns this table?" always has a single answer.

## Consequences
- Cross-service data queries require gRPC API calls.
- Eventual consistency must be accepted for cross-service data.
- No foreign keys across service boundaries.
- Each service manages its own migrations (Alembic).
- In local dev (docker-compose): single PostgreSQL instance with per-service databases.
