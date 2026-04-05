package model

import "time"

// MCPActor is a server-issued identity for MCP clients (one human user may have many).
type MCPActor struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	Label           string     `json:"label"`
	CreatedAt       time.Time  `json:"created_at"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at,omitempty"`
}
