package mcpserver

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"yaitracker.com/loweryaustin/internal/store"
)

func resourceProjectList(st *store.Store) server.ResourceHandlerFunc {
	return func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		summaries, err := st.ListProjectSummaries(ctx)
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "yaitracker://projects",
				MIMEType: "application/json",
				Text:     toJSON(summaries),
			},
		}, nil
	}
}

func resourceProjectDetail(st *store.Store) server.ResourceTemplateHandlerFunc {
	return func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		key := extractURIParam(req.Params.URI, "projects/", "/")
		if key == "" {
			key = extractLastSegment(req.Params.URI, "projects/")
		}

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("project %s not found", key)
		}

		health, err := st.GetProjectHealth(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("get project health: %w", err)
		}
		result := map[string]interface{}{
			"project": p,
			"health":  health,
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     toJSON(result),
			},
		}, nil
	}
}

func resourceIssueDetail(st *store.Store) server.ResourceTemplateHandlerFunc {
	return func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		parts := parseIssueURI(req.Params.URI)
		if parts.key == "" || parts.number == 0 {
			return nil, fmt.Errorf("invalid issue URI")
		}

		p, err := st.GetProjectByKey(ctx, parts.key)
		if err != nil {
			return nil, fmt.Errorf("project %s not found", parts.key)
		}

		issue, err := st.GetIssueByNumber(ctx, p.ID, parts.number)
		if err != nil {
			return nil, fmt.Errorf("issue %s-%d not found", parts.key, parts.number)
		}

		comments, err := st.ListComments(ctx, issue.ID)
		if err != nil {
			return nil, fmt.Errorf("list comments: %w", err)
		}
		timeEntries, err := st.ListTimeEntries(ctx, issue.ID)
		if err != nil {
			return nil, fmt.Errorf("list time entries: %w", err)
		}
		totalTime, err := st.GetIssueTotalTime(ctx, issue.ID)
		if err != nil {
			return nil, fmt.Errorf("get issue total time: %w", err)
		}

		result := map[string]interface{}{
			"issue":              issue,
			"comments":           comments,
			"time_entries":       timeEntries,
			"total_time_seconds": totalTime,
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     toJSON(result),
			},
		}, nil
	}
}

func resourceBoard(st *store.Store) server.ResourceTemplateHandlerFunc {
	return func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		key := extractURIParam(req.Params.URI, "projects/", "/board")

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("project %s not found", key)
		}

		columns, err := st.ListIssuesByStatus(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("list issues by status: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     toJSON(columns),
			},
		}, nil
	}
}

func resourceVelocity(st *store.Store) server.ResourceTemplateHandlerFunc {
	return func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		key := extractURIParam(req.Params.URI, "projects/", "/velocity")

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("project %s not found", key)
		}

		velocity, err := st.GetVelocity(ctx, p.ID, 8)
		if err != nil {
			return nil, fmt.Errorf("get velocity: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     toJSON(velocity),
			},
		}, nil
	}
}

type issueURIParts struct {
	key    string
	number int
}

func parseIssueURI(uri string) issueURIParts {
	u, err := url.Parse(uri)
	if err != nil {
		return issueURIParts{}
	}
	// URI scheme: yaitracker, path: /projects/{key}/issues/{number}
	path := strings.TrimPrefix(u.Path, "/")
	if u.Host == "projects" {
		path = u.Host + "/" + path
	}
	parts := strings.Split(path, "/")
	// parts: ["projects", "{key}", "issues", "{number}"]
	if len(parts) >= 4 && parts[0] == "projects" && parts[2] == "issues" {
		num, err := strconv.Atoi(parts[3])
		if err != nil {
			return issueURIParts{}
		}
		return issueURIParts{key: parts[1], number: num}
	}
	return issueURIParts{}
}

func extractURIParam(uri, prefix, suffix string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	path := u.Host + "/" + strings.TrimPrefix(u.Path, "/")
	idx := strings.Index(path, prefix)
	if idx < 0 {
		return ""
	}
	rest := path[idx+len(prefix):]
	if suffix != "" {
		end := strings.Index(rest, suffix)
		if end >= 0 {
			return rest[:end]
		}
	}
	return rest
}

func extractLastSegment(uri, prefix string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	path := u.Host + "/" + strings.TrimPrefix(u.Path, "/")
	idx := strings.Index(path, prefix)
	if idx < 0 {
		return ""
	}
	rest := path[idx+len(prefix):]
	parts := strings.Split(rest, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return rest
}
