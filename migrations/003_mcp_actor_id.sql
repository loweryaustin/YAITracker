-- +goose Up

-- Distinguish concurrent MCP agents on the same issue (Phase 2). Empty / NULL means
-- legacy single anonymous agent slot per issue (ifnull groups NULL with '' in the index).
ALTER TABLE time_entries ADD COLUMN mcp_actor_id TEXT;

DROP INDEX IF EXISTS idx_active_agent_timer;
CREATE UNIQUE INDEX idx_active_agent_timer ON time_entries (
    issue_id,
    ifnull(mcp_actor_id, '')
) WHERE ended_at IS NULL AND actor_type = 'agent';

-- +goose Down

DROP INDEX IF EXISTS idx_active_agent_timer;
CREATE UNIQUE INDEX idx_active_agent_timer ON time_entries(issue_id)
    WHERE ended_at IS NULL AND actor_type = 'agent';

ALTER TABLE time_entries DROP COLUMN mcp_actor_id;
