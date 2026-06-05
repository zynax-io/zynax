-- SPDX-License-Identifier: Apache-2.0
-- agent-registry schema — exclusively owned by agent-registry (ADR-008)
CREATE TABLE IF NOT EXISTS agents (
    id              TEXT        PRIMARY KEY,
    name            TEXT        NOT NULL DEFAULT '',
    description     TEXT        NOT NULL DEFAULT '',
    endpoint        TEXT        NOT NULL DEFAULT '',
    labels          JSONB,
    capabilities    JSONB,
    status          INTEGER     NOT NULL DEFAULT 0,
    registered_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS agents_status_idx ON agents (status);

-- heartbeats is reserved for future heartbeat tracking (M6+ scope)
CREATE TABLE IF NOT EXISTS heartbeats (
    agent_id    TEXT        NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (agent_id, received_at)
);
