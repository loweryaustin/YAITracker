package auth

import (
	"context"
	"fmt"
	"strings"
)

type mcpActorIDKey struct{}

type mcpActorInvalidHeaderKey struct{}

const maxMCPActorIDLen = 128

// ContextWithMCPActorID attaches the validated MCP actor id (from server registration).
func ContextWithMCPActorID(ctx context.Context, actorID string) context.Context {
	actorID = NormalizeMCPActorID(actorID)
	if len(actorID) > maxMCPActorIDLen {
		actorID = actorID[:maxMCPActorIDLen]
	}
	return context.WithValue(ctx, mcpActorIDKey{}, actorID)
}

// ContextWithMCPActorInvalidHeader marks ctx when the client sent an X-Yaitracker-Mcp-Actor-Id
// value that is not registered for the authenticated user.
func ContextWithMCPActorInvalidHeader(ctx context.Context) context.Context {
	return context.WithValue(ctx, mcpActorInvalidHeaderKey{}, true)
}

// MCPActorInvalidHeader reports whether the MCP actor header was present but invalid.
func MCPActorInvalidHeader(ctx context.Context) bool {
	v, ok := ctx.Value(mcpActorInvalidHeaderKey{}).(bool)
	return ok && v
}

// MCPActorIDFromContext returns the validated MCP actor id, or "".
func MCPActorIDFromContext(ctx context.Context) string {
	v, ok := ctx.Value(mcpActorIDKey{}).(string)
	if !ok {
		return ""
	}
	return v
}

// NormalizeMCPActorID trims whitespace for comparisons and storage.
func NormalizeMCPActorID(s string) string {
	return strings.TrimSpace(s)
}

// MCPActorIDsEqual reports whether two actor ids match after normalization.
func MCPActorIDsEqual(a, b string) bool {
	return NormalizeMCPActorID(a) == NormalizeMCPActorID(b)
}

// AgentMCPAuthError is non-nil when agent-scoped MCP operations cannot run
// (missing header/env actor id, or unknown/revoked id).
func AgentMCPAuthError(ctx context.Context) error {
	if MCPActorInvalidHeader(ctx) {
		return fmt.Errorf("mcp actor id is unknown or revoked for this user")
	}
	if NormalizeMCPActorID(MCPActorIDFromContext(ctx)) == "" {
		return fmt.Errorf("set X-Yaitracker-Mcp-Actor-Id to a registered MCP actor id (POST /api/v1/mcp/actors)")
	}
	return nil
}
