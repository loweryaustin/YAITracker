package server

import (
	"context"
	"net/http"
	"strings"

	mcpgo "github.com/mark3labs/mcp-go/server"

	"yaitracker.com/loweryaustin/internal/auth"
	"yaitracker.com/loweryaustin/internal/store"
)

// MCPActorIDHeader is the HTTP header clients send to identify concurrent MCP agents
// (same user, same issue, distinct timers). Optional; empty/absent uses legacy single-slot behavior.
const MCPActorIDHeader = "X-Yaitracker-Mcp-Actor-Id"

type Server struct {
	store     *store.Store
	secret    string
	cors      string
	mcpServer *mcpgo.MCPServer
}

func New(st *store.Store, secret, cors string, mcpServer *mcpgo.MCPServer) *Server {
	return &Server{
		store:     st,
		secret:    secret,
		cors:      cors,
		mcpServer: mcpServer,
	}
}

func (s *Server) Store() *store.Store {
	return s.store
}

// mcpContextFunc returns a context function that authenticates MCP requests
// via Bearer token and injects the resolved user into the context.
func (s *Server) mcpContextFunc() func(ctx context.Context, r *http.Request) context.Context {
	return func(ctx context.Context, r *http.Request) context.Context {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return ctx
		}
		token := strings.TrimPrefix(header, "Bearer ")
		tok, err := s.store.GetOAuthTokenByAccess(ctx, token)
		if err != nil {
			return ctx
		}
		user, err := s.store.GetUserByID(ctx, tok.UserID)
		if err != nil {
			return ctx
		}
		ctx = auth.ContextWithUser(ctx, user)
		actor := r.Header.Get(MCPActorIDHeader)
		if actor != "" {
			ctx = auth.ContextWithMCPActorID(ctx, actor)
		}
		return ctx
	}
}
