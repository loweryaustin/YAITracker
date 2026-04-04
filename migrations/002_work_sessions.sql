-- +goose Up

CREATE TABLE work_sessions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    description TEXT,
    started_at  DATETIME NOT NULL,
    ended_at    DATETIME,
    duration    INTEGER,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_work_sessions_user ON work_sessions(user_id);
CREATE UNIQUE INDEX idx_active_work_session ON work_sessions(user_id) WHERE ended_at IS NULL;

ALTER TABLE time_entries ADD COLUMN session_id TEXT REFERENCES work_sessions(id);
ALTER TABLE time_entries ADD COLUMN actor_type TEXT NOT NULL DEFAULT 'human';

DROP INDEX idx_active_timer;
CREATE UNIQUE INDEX idx_active_human_timer ON time_entries(user_id)
    WHERE ended_at IS NULL AND actor_type = 'human';
CREATE UNIQUE INDEX idx_active_agent_timer ON time_entries(issue_id)
    WHERE ended_at IS NULL AND actor_type = 'agent';

-- +goose Down

DROP INDEX IF EXISTS idx_active_agent_timer;
DROP INDEX IF EXISTS idx_active_human_timer;
CREATE UNIQUE INDEX idx_active_timer ON time_entries(user_id) WHERE ended_at IS NULL;

ALTER TABLE time_entries DROP COLUMN actor_type;
ALTER TABLE time_entries DROP COLUMN session_id;

DROP TABLE IF EXISTS work_sessions;
