package mcpserver

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"yaitracker.com/loweryaustin/internal/store"
)

func NewMCPServer(st *store.Store) *server.MCPServer {
	s := server.NewMCPServer(
		"YAITracker",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
	)

	registerTools(s, st)
	registerResources(s, st)

	return s
}

func registerResources(s *server.MCPServer, st *store.Store) {
	s.AddResource(mcp.NewResource(
		"yaitracker://projects",
		"Project list",
		mcp.WithResourceDescription("List all YAITracker projects with summary stats"),
		mcp.WithMIMEType("application/json"),
	), resourceProjectList(st))

	s.AddResourceTemplate(mcp.NewResourceTemplate(
		"yaitracker://projects/{key}",
		"Project detail",
		mcp.WithTemplateDescription("Project details with health and tags"),
		mcp.WithTemplateMIMEType("application/json"),
	), resourceProjectDetail(st))

	s.AddResourceTemplate(mcp.NewResourceTemplate(
		"yaitracker://projects/{key}/issues/{number}",
		"Issue detail",
		mcp.WithTemplateDescription("Issue with comments, time entries, and labels"),
		mcp.WithTemplateMIMEType("application/json"),
	), resourceIssueDetail(st))

	s.AddResourceTemplate(mcp.NewResourceTemplate(
		"yaitracker://projects/{key}/board",
		"Board state",
		mcp.WithTemplateDescription("Kanban board state for a project"),
		mcp.WithTemplateMIMEType("application/json"),
	), resourceBoard(st))

	s.AddResourceTemplate(mcp.NewResourceTemplate(
		"yaitracker://projects/{key}/velocity",
		"Velocity data",
		mcp.WithTemplateDescription("Velocity chart data for a project"),
		mcp.WithTemplateMIMEType("application/json"),
	), resourceVelocity(st))
}
