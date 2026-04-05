-- +goose Up
-- Server-issued MCP actor registration; agent timers always reference mcp_actors.id.

CREATE TABLE mcp_actors (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label      TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at DATETIME
);

CREATE INDEX idx_mcp_actors_user ON mcp_actors(user_id);

-- Register distinct non-empty client-supplied ids that were already stored on time_entries.
INSERT OR IGNORE INTO mcp_actors (id, user_id, label, created_at)
SELECT DISTINCT trim(te.mcp_actor_id), te.user_id, 'imported', datetime('now')
FROM time_entries te
WHERE te.actor_type = 'agent'
  AND te.mcp_actor_id IS NOT NULL
  AND trim(te.mcp_actor_id) != '';

-- One migrated actor per user who had NULL/empty mcp_actor_id on agent rows.
INSERT INTO mcp_actors (id, user_id, label, created_at)
SELECT lower(hex(randomblob(16))), u.user_id, 'migrated-null-slot', datetime('now')
FROM (
    SELECT DISTINCT user_id
    FROM time_entries
    WHERE actor_type = 'agent'
      AND (mcp_actor_id IS NULL OR trim(mcp_actor_id) = '')
) u;

-- Point previously NULL/empty agent rows at that user's migrated actor.
UPDATE time_entries
SET mcp_actor_id = (
    SELECT ma.id
    FROM mcp_actors ma
    WHERE ma.user_id = time_entries.user_id
      AND ma.label = 'migrated-null-slot'
    LIMIT 1
)
WHERE actor_type = 'agent'
  AND (mcp_actor_id IS NULL OR trim(mcp_actor_id) = '');

DROP INDEX IF EXISTS idx_active_agent_timer;

CREATE UNIQUE INDEX idx_active_agent_timer ON time_entries (
    issue_id,
    mcp_actor_id
) WHERE ended_at IS NULL
  AND actor_type = 'agent'
  AND mcp_actor_id IS NOT NULL
  AND trim(mcp_actor_id) != '';

-- +goose Down

DROP INDEX IF EXISTS idx_active_agent_timer;

CREATE UNIQUE INDEX idx_active_agent_timer ON time_entries (
    issue_id,
    ifnull(mcp_actor_id, '')
) WHERE ended_at IS NULL AND actor_type = 'agent';

DROP TABLE IF EXISTS mcp_actors;

