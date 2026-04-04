package server

import (
	mcpgo "github.com/mark3labs/mcp-go/server"
	"yaitracker.com/loweryaustin/internal/store"
)

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
