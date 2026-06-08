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

-- NOTE: An HNSW index on `embedding vector` is not created here because pgvector
-- requires a fixed-dimension vector column (e.g. vector(1536)) for HNSW/IVFFlat
-- indexes. Without a fixed dimension, exact cosine search via the <=> operator
-- works correctly; an HNSW index can be added once the service is wired to a
-- specific embedding model that fixes the dimension.

-- Namespace lookup index for fast DELETE/SELECT by namespace.
CREATE INDEX IF NOT EXISTS memory_vectors_namespace_idx
    ON memory_vectors (namespace);
