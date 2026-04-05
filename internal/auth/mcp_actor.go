package auth

import (
	"context"
	"strings"
)

type mcpActorIDKey struct{}

const maxMCPActorIDLen = 128

// ContextWithMCPActorID attaches an MCP client identity for HTTP MCP requests.
// Used to attribute agent timers and to match complete_work / stop_timer to the caller.
func ContextWithMCPActorID(ctx context.Context, actorID string) context.Context {
	actorID = NormalizeMCPActorID(actorID)
	if len(actorID) > maxMCPActorIDLen {
		actorID = actorID[:maxMCPActorIDLen]
	}
	return context.WithValue(ctx, mcpActorIDKey{}, actorID)
}

// MCPActorIDFromContext returns the MCP actor id from ContextWithMCPActorID, or "".
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
