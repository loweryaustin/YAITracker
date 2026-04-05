package server

import (
	"context"
	"net/http"
	"strings"

	mcpgo "github.com/mark3labs/mcp-go/server"

	"yaitracker.com/loweryaustin/internal/auth"
	"yaitracker.com/loweryaustin/internal/store"
)

// MCPActorIDHeader is the HTTP header clients send with a server-registered MCP actor id
// (from POST /api/v1/mcp/actors). The Sidecar proxy sets this automatically.
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
		actor := auth.NormalizeMCPActorID(r.Header.Get(MCPActorIDHeader))
		if actor == "" {
			return ctx
		}
		_, err = s.store.GetMCPActorForUser(ctx, user.ID, actor)
		if err != nil {
			return auth.ContextWithMCPActorInvalidHeader(ctx)
		}
		return auth.ContextWithMCPActorID(ctx, actor)
	}
}
