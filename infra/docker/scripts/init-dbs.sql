-- SPDX-License-Identifier: Apache-2.0
-- AgentMesh — Local dev database initialisation
-- Creates one database per service (see ADR-008: no shared databases)
-- Runs automatically on first `make dev-up`

CREATE DATABASE agent_registry;
CREATE DATABASE task_broker;
CREATE DATABASE memory_service;

-- Enable pgvector extension in memory_service (needed for vector embeddings)
\c memory_service
CREATE EXTENSION IF NOT EXISTS vector;

\c agent_registry
\c task_broker
