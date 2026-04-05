-- +goose Up
-- Add heartbeat tracking for TTL-based actor expiration.

ALTER TABLE mcp_actors ADD COLUMN last_heartbeat_at DATETIME;
UPDATE mcp_actors SET last_heartbeat_at = created_at WHERE last_heartbeat_at IS NULL;

-- +goose Down
-- SQLite <3.35 cannot DROP COLUMN; accept the extra column on downgrade.
