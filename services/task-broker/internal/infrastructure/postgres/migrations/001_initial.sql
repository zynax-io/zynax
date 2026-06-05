-- SPDX-License-Identifier: Apache-2.0
-- task-broker schema — exclusively owned by task-broker (ADR-008)
CREATE TABLE IF NOT EXISTS tasks (
    task_id           TEXT        PRIMARY KEY,
    workflow_id       TEXT        NOT NULL,
    capability_name   TEXT        NOT NULL,
    input_payload     BYTEA,
    timeout_seconds   INTEGER     NOT NULL DEFAULT 0,
    max_retries       INTEGER     NOT NULL DEFAULT 0,
    retry_count       INTEGER     NOT NULL DEFAULT 0,
    status            INTEGER     NOT NULL DEFAULT 0,
    dispatched_to     TEXT        NOT NULL DEFAULT '',
    result_payload    BYTEA,
    error_code        TEXT        NOT NULL DEFAULT '',
    error_message     TEXT        NOT NULL DEFAULT '',
    error_details     JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    dispatched_at     TIMESTAMPTZ,
    completed_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS tasks_workflow_id_idx   ON tasks (workflow_id);
CREATE INDEX IF NOT EXISTS tasks_status_idx        ON tasks (status);
CREATE INDEX IF NOT EXISTS tasks_dispatched_to_idx ON tasks (dispatched_to);
-- keyset cursor: (created_at, task_id) is the stable sort key used in List pagination
CREATE INDEX IF NOT EXISTS tasks_cursor_idx        ON tasks (created_at, task_id);
