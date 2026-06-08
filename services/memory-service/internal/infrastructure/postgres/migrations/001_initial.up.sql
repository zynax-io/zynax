-- SPDX-License-Identifier: Apache-2.0
-- memory-service schema — exclusively owned by memory-service (ADR-008)

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS memory_vectors (
    id          TEXT        NOT NULL,
    namespace   TEXT        NOT NULL,
    embedding   vector,
    metadata    JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (namespace, id)
);

-- HNSW index for approximate nearest-neighbour search (cosine distance).
CREATE INDEX IF NOT EXISTS memory_vectors_hnsw_idx
    ON memory_vectors
    USING hnsw (embedding vector_cosine_ops);

-- Namespace lookup index for fast DELETE/SELECT by namespace.
CREATE INDEX IF NOT EXISTS memory_vectors_namespace_idx
    ON memory_vectors (namespace);
