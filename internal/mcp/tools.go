package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"yaitracker.com/loweryaustin/internal/auth"
	"yaitracker.com/loweryaustin/internal/model"
	"yaitracker.com/loweryaustin/internal/store"
)

func registerTools(s *server.MCPServer, st *store.Store) {
	// Project tools
	s.AddTool(mcp.NewTool("list_projects",
		mcp.WithDescription("List all projects with summary stats. Optionally filter by tags."),
		mcp.WithString("tag", mcp.Description("Filter by tag")),
	), toolListProjects(st))

	s.AddTool(mcp.NewTool("create_project",
		mcp.WithDescription("Create a new project"),
		mcp.WithString("key", mcp.Required(), mcp.Description("Unique project key (uppercase)")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Project name")),
		mcp.WithString("description", mcp.Description("Project description")),
		mcp.WithString("tags", mcp.Description("Comma-separated tags")),
	), toolCreateProject(st))

	s.AddTool(mcp.NewTool("delete_project",
		mcp.WithDescription("Permanently delete a project and all its issues, comments, time entries, labels, and tags. This cannot be undone."),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key to delete")),
		mcp.WithBoolean("confirm", mcp.Required(), mcp.Description("Must be true to confirm deletion")),
	), toolDeleteProject(st))

	s.AddTool(mcp.NewTool("tag_project",
		mcp.WithDescription("Add or remove a tag on a project"),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithString("tag", mcp.Required(), mcp.Description("Tag to add/remove")),
		mcp.WithString("group", mcp.Description("Tag group name")),
		mcp.WithString("action", mcp.Description("'add' or 'remove' (default: add)")),
	), toolTagProject(st))

	s.AddTool(mcp.NewTool("list_tags",
		mcp.WithDescription("List all tags with usage counts and groups"),
	), toolListTags(st))

	// Issue tools
	s.AddTool(mcp.NewTool("list_issues",
		mcp.WithDescription("List issues for a project. Returns concise one-line-per-issue format by default; set format=json for full detail (adds parent_number and child_numbers for hierarchy)."),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithString("status", mcp.Description("Filter by status (comma-separated)")),
		mcp.WithString("type", mcp.Description("Filter by type")),
		mcp.WithString("assignee", mcp.Description("Filter by assignee name")),
		mcp.WithString("query", mcp.Description("Search in title and description")),
		mcp.WithString("format", mcp.Description("'json' for full detail, default is concise text")),
	), toolListIssues(st))

	s.AddTool(mcp.NewTool("get_issue",
		mcp.WithDescription("Get full issue detail including comments, time entries, and labels"),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Issue number")),
	), toolGetIssue(st))

	s.AddTool(mcp.NewTool("create_issue",
		mcp.WithDescription("Create a new issue"),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Issue title")),
		mcp.WithString("description", mcp.Description("Issue description (markdown)")),
		mcp.WithString("type", mcp.Description("Issue type: bug, task, feature, improvement")),
		mcp.WithString("priority", mcp.Description("Priority: none, low, medium, high, urgent")),
		mcp.WithString("status", mcp.Description("Status: backlog, todo, in_progress, in_review, done")),
		mcp.WithNumber("story_points", mcp.Description("Story point estimate")),
		mcp.WithNumber("estimated_hours", mcp.Description("Hours estimate")),
		mcp.WithNumber("parent_number", mcp.Description("Optional parent issue number in the same project")),
	), toolCreateIssue(st))

	s.AddTool(mcp.NewTool("update_issue",
		mcp.WithDescription("Update an issue's fields"),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Issue number")),
		mcp.WithString("title", mcp.Description("New title")),
		mcp.WithString("description", mcp.Description("New description")),
		mcp.WithString("type", mcp.Description("New type")),
		mcp.WithString("priority", mcp.Description("New priority")),
		mcp.WithNumber("story_points", mcp.Description("Story points")),
		mcp.WithNumber("estimated_hours", mcp.Description("Estimated hours")),
		mcp.WithBoolean("clear_parent", mcp.Description("When true, remove the parent link (ignores parent_number)")),
		mcp.WithNumber("parent_number", mcp.Description("When > 0, set parent to this issue number in the same project")),
	), toolUpdateIssue(st))

	s.AddTool(mcp.NewTool("move_issue",
		mcp.WithDescription("Change issue status (move on the board)"),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Issue number")),
		mcp.WithString("status", mcp.Required(), mcp.Description("New status")),
	), toolMoveIssue(st))

	s.AddTool(mcp.NewTool("delete_issue",
		mcp.WithDescription("Permanently delete an issue and all its comments and time entries. Cannot be undone."),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Issue number")),
		mcp.WithBoolean("confirm", mcp.Required(), mcp.Description("Must be true to confirm deletion")),
	), toolDeleteIssue(st))

	s.AddTool(mcp.NewTool("add_comment",
		mcp.WithDescription("Add a comment to an issue"),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Issue number")),
		mcp.WithString("body", mcp.Required(), mcp.Description("Comment body (markdown)")),
	), toolAddComment(st))

	s.AddTool(mcp.NewTool("add_issue_label",
		mcp.WithDescription("Attach a label to an issue; creates the label in the project if it does not exist"),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Issue number")),
		mcp.WithString("label_name", mcp.Required(), mcp.Description("Label name (case-insensitive match to existing labels)")),
		mcp.WithString("color", mcp.Description("Hex color when creating a new label (default #64748b)")),
	), toolAddIssueLabel(st))

	s.AddTool(mcp.NewTool("search_issues",
		mcp.WithDescription("Search issues across all projects"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
	), toolSearchIssues(st))

	// Timer tools (human sessions/timers are UI-only; MCP is agent-only)
	s.AddTool(mcp.NewTool("start_timer",
		mcp.WithDescription("Start an agent timer on an issue. Returns timer_id for later stop_timer call. Prefer begin_work for typical workflows."),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Issue number")),
		mcp.WithString("description", mcp.Description("What will be worked on")),
	), toolStartTimer(st))

	s.AddTool(mcp.NewTool("stop_timer",
		mcp.WithDescription("Stop an active timer. Provide timer_id (returned by start_timer) OR project_key+number to identify which timer to stop."),
		mcp.WithString("timer_id", mcp.Description("Timer ID returned by start_timer")),
		mcp.WithString("project_key", mcp.Description("Project key (alternative to timer_id)")),
		mcp.WithNumber("number", mcp.Description("Issue number (alternative to timer_id)")),
		mcp.WithString("actor_type", mcp.Description("Filter by 'human' or 'agent' when using project_key+number")),
	), toolStopTimer(st))

	s.AddTool(mcp.NewTool("get_session_status",
		mcp.WithDescription("Get current work session, all active timers, and utilization metrics."),
	), toolGetSessionStatus(st))

	s.AddTool(mcp.NewTool("get_time_entries",
		mcp.WithDescription("Get time entries for an issue"),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Issue number")),
	), toolGetTimeEntries(st))

	// Analytics tools
	s.AddTool(mcp.NewTool("get_velocity",
		mcp.WithDescription("Get velocity data for a project"),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithNumber("weeks", mcp.Description("Number of weeks (default: 8)")),
	), toolGetVelocity(st))

	s.AddTool(mcp.NewTool("get_estimation_accuracy",
		mcp.WithDescription("Get estimation accuracy report for a project"),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
	), toolGetEstimationAccuracy(st))

	s.AddTool(mcp.NewTool("get_project_health",
		mcp.WithDescription("Get project health summary"),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
	), toolGetProjectHealth(st))

	s.AddTool(mcp.NewTool("compare_by_tag",
		mcp.WithDescription("Compare project metrics grouped by tag (e.g. compare Go vs PHP)"),
		mcp.WithString("group_by", mcp.Required(), mcp.Description("Tag group name (language, framework, etc.)")),
	), toolCompareByTag(st))

	s.AddTool(mcp.NewTool("predict_new_project",
		mcp.WithDescription("Predict timeline and bug rate for a new project based on historical data"),
		mcp.WithString("tags", mcp.Required(), mcp.Description("Comma-separated tags for the new project")),
		mcp.WithNumber("points", mcp.Required(), mcp.Description("Estimated story points for the project")),
	), toolPredictNewProject(st))

	// Compound workflow tools
	s.AddTool(mcp.NewTool("begin_work",
		mcp.WithDescription("Start working on an issue. Ensures a work session exists, moves the issue to in_progress, and starts an agent timer. Returns issue details and timer_id."),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Issue number")),
	), toolBeginWork(st))

	s.AddTool(mcp.NewTool("complete_work",
		mcp.WithDescription("Finish working on an issue. Stops the active timer, adds a summary comment, and moves the issue to done. Returns duration logged."),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Issue number")),
		mcp.WithString("summary", mcp.Required(), mcp.Description("Completion summary (added as comment)")),
	), toolCompleteWork(st))
}

func toJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ") //nolint:errcheck // marshalling known types
	return string(b)
}

// issueDetailResponse is the issue payload for get_issue with a stable parent_number for agents.
type issueDetailResponse struct {
	model.Issue
	ParentNumber *int `json:"parent_number,omitempty"`
}

func issuesToMCPJSON(ctx context.Context, st *store.Store, projectID string, issues []model.Issue) ([]map[string]interface{}, error) {
	idToNum, err := st.MapIssueIDToNumber(ctx, projectID)
	if err != nil {
		return nil, err
	}
	parentToKids, err := st.MapParentIDToChildNumbers(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(issues))
	for i := range issues {
		raw, err := json.Marshal(issues[i])
		if err != nil {
			return nil, err
		}
		var m map[string]interface{}
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		if issues[i].ParentID != nil {
			if n, ok := idToNum[*issues[i].ParentID]; ok {
				m["parent_number"] = n
			}
		}
		if kids := parentToKids[issues[i].ID]; len(kids) > 0 {
			m["child_numbers"] = kids
		}
		out = append(out, m)
	}
	return out, nil
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: text}},
	}
}

func errResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: " + err.Error()}},
		IsError: true,
	}
}

// --- Tool implementations ---

func toolListProjects(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		summaries, err := st.ListProjectSummaries(ctx)
		if err != nil {
			return errResult(err), nil
		}
		return textResult(toJSON(summaries)), nil
	}
}

func toolCreateProject(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := strings.ToUpper(mcp.ParseString(req, "key", ""))
		name := mcp.ParseString(req, "name", "")

		creatorID, err := userFromCtx(ctx)
		if err != nil {
			return errResult(err), nil
		}

		p := &model.Project{
			Key:         key,
			Name:        name,
			Description: mcp.ParseString(req, "description", ""),
			Status:      "active",
			CreatedBy:   creatorID,
		}

		if err := st.CreateProject(ctx, p); err != nil {
			return errResult(err), nil
		}

		if tags := mcp.ParseString(req, "tags", ""); tags != "" {
			for _, t := range strings.Split(tags, ",") {
				t = strings.TrimSpace(strings.ToLower(t))
				if t != "" {
					if err := st.AddProjectTag(ctx, p.ID, t, ""); err != nil {
						return errResult(fmt.Errorf("add project tag: %w", err)), nil
					}
				}
			}
		}

		return textResult(fmt.Sprintf("Created project %s (key: %s, id: %s)", p.Name, p.Key, p.ID)), nil
	}
}

func toolDeleteProject(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := strings.ToUpper(mcp.ParseString(req, "project_key", ""))
		confirm := mcp.ParseBoolean(req, "confirm", false)

		if !confirm {
			return errResult(fmt.Errorf("set confirm=true to delete project %s and all its data", key)), nil
		}

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
		}

		if err := st.DeleteProject(ctx, p.ID); err != nil {
			return errResult(fmt.Errorf("delete project: %w", err)), nil
		}

		return textResult(fmt.Sprintf("Deleted project %s (%s) and all associated data", key, p.Name)), nil
	}
}

func toolDeleteIssue(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := strings.ToUpper(mcp.ParseString(req, "project_key", ""))
		number := mcp.ParseInt(req, "number", 0)
		confirm := mcp.ParseBoolean(req, "confirm", false)

		if !confirm {
			return errResult(fmt.Errorf("set confirm=true to delete %s-%d", key, number)), nil
		}

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil //nolint:nilerr // error conveyed via errResult
		}

		if err := st.DeleteIssue(ctx, issue.ID); err != nil {
			return errResult(fmt.Errorf("delete issue: %w", err)), nil
		}

		return textResult(fmt.Sprintf("Deleted %s-%d: %s", key, number, issue.Title)), nil
	}
}

func toolTagProject(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		tag := mcp.ParseString(req, "tag", "")
		group := mcp.ParseString(req, "group", "")
		action := mcp.ParseString(req, "action", "add")

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
		}

		if action == "remove" {
			if err := st.RemoveProjectTag(ctx, p.ID, tag); err != nil {
				return errResult(fmt.Errorf("remove project tag: %w", err)), nil
			}
			return textResult(fmt.Sprintf("Removed tag '%s' from %s", tag, key)), nil
		}

		if err := st.AddProjectTag(ctx, p.ID, tag, group); err != nil {
			return errResult(fmt.Errorf("add project tag: %w", err)), nil
		}
		return textResult(fmt.Sprintf("Added tag '%s' to %s", tag, key)), nil
	}
}

func toolListTags(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tags, err := st.ListAllTags(ctx)
		if err != nil {
			return errResult(fmt.Errorf("list tags: %w", err)), nil
		}
		return textResult(toJSON(tags)), nil
	}
}

func toolListIssues(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
		}

		filter := model.IssueFilter{
			ProjectID: p.ID,
			Query:     mcp.ParseString(req, "query", ""),
			Limit:     100,
		}
		if s := mcp.ParseString(req, "status", ""); s != "" {
			filter.Status = strings.Split(s, ",")
		}
		if t := mcp.ParseString(req, "type", ""); t != "" {
			filter.Type = strings.Split(t, ",")
		}

		issues, total, err := st.ListIssues(ctx, filter)
		if err != nil {
			return errResult(fmt.Errorf("list issues: %w", err)), nil
		}

		if mcp.ParseString(req, "format", "") == "json" {
			enriched, err := issuesToMCPJSON(ctx, st, p.ID, issues)
			if err != nil {
				return errResult(fmt.Errorf("list issues: %w", err)), nil
			}
			return textResult(fmt.Sprintf("%d issues (showing %d):\n%s", total, len(enriched), toJSON(enriched))), nil
		}

		var lines []string
		for i := range issues {
			pts := ""
			if issues[i].StoryPoints != nil {
				pts = fmt.Sprintf(", %dpts", *issues[i].StoryPoints)
			}
			lines = append(lines, fmt.Sprintf("%s-%d [%s, %s%s] %s",
				key, issues[i].Number, issues[i].Status, issues[i].Priority, pts, issues[i].Title))
		}
		return textResult(fmt.Sprintf("%d issues (showing %d):\n%s", total, len(lines), strings.Join(lines, "\n"))), nil
	}
}

func toolGetIssue(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		number := mcp.ParseInt(req, "number", 0)

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
		}

		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil //nolint:nilerr // error conveyed via errResult
		}

		comments, err := st.ListComments(ctx, issue.ID)
		if err != nil {
			return errResult(fmt.Errorf("list comments: %w", err)), nil
		}
		timeEntries, err := st.ListTimeEntries(ctx, issue.ID)
		if err != nil {
			return errResult(fmt.Errorf("list time entries: %w", err)), nil
		}
		totalTime, err := st.GetIssueTotalTime(ctx, issue.ID)
		if err != nil {
			return errResult(fmt.Errorf("get issue total time: %w", err)), nil
		}
		children, err := st.GetChildIssues(ctx, issue.ID)
		if err != nil {
			return errResult(fmt.Errorf("get child issues: %w", err)), nil
		}

		childNums := make([]int, len(children))
		for i := range children {
			childNums[i] = children[i].Number
		}

		var parentNum *int
		if issue.ParentID != nil {
			parent, err := st.GetIssue(ctx, *issue.ParentID)
			if err == nil && parent.ProjectID == issue.ProjectID {
				n := parent.Number
				parentNum = &n
			}
		}

		detail := issueDetailResponse{Issue: *issue, ParentNumber: parentNum}

		result := map[string]interface{}{
			"issue":              detail,
			"comments":           comments,
			"time_entries":       timeEntries,
			"total_time_seconds": totalTime,
			"children":           children,
			"child_numbers":      childNums,
		}
		return textResult(toJSON(result)), nil
	}
}

func toolCreateIssue(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
		}

		reporterID, err := userFromCtx(ctx)
		if err != nil {
			return errResult(err), nil
		}

		issueType := mcp.ParseString(req, "type", "task")
		status := mcp.ParseString(req, "status", "backlog")
		priority := mcp.ParseString(req, "priority", "none")

		issue := &model.Issue{
			ProjectID:   p.ID,
			Title:       mcp.ParseString(req, "title", ""),
			Description: mcp.ParseString(req, "description", ""),
			Type:        issueType,
			Status:      status,
			Priority:    priority,
			ReporterID:  reporterID,
		}

		sp := mcp.ParseInt(req, "story_points", 0)
		if sp > 0 {
			issue.StoryPoints = &sp
		}
		eh := mcp.ParseFloat64(req, "estimated_hours", 0)
		if eh > 0 {
			issue.EstimatedHours = &eh
		}

		parentNumber := mcp.ParseInt(req, "parent_number", 0)
		pid, err := parentIDFromOptionalNumber(ctx, st, p.ID, parentNumber)
		if err != nil {
			return errResult(err), nil
		}
		issue.ParentID = pid

		if err := st.CreateIssue(ctx, issue); err != nil {
			return errResult(err), nil
		}

		return textResult(fmt.Sprintf("Created %s-%d: %s", key, issue.Number, issue.Title)), nil
	}
}

func toolUpdateIssue(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		number := mcp.ParseInt(req, "number", 0)

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil //nolint:nilerr // error conveyed via errResult
		}

		if v := mcp.ParseString(req, "title", ""); v != "" {
			issue.Title = v
		}
		if v := mcp.ParseString(req, "description", ""); v != "" {
			issue.Description = v
		}
		if v := mcp.ParseString(req, "type", ""); v != "" {
			issue.Type = v
		}
		if v := mcp.ParseString(req, "priority", ""); v != "" {
			issue.Priority = v
		}
		sp := mcp.ParseInt(req, "story_points", 0)
		if sp > 0 {
			issue.StoryPoints = &sp
		}
		eh := mcp.ParseFloat64(req, "estimated_hours", 0)
		if eh > 0 {
			issue.EstimatedHours = &eh
		}

		clearParent := mcp.ParseBoolean(req, "clear_parent", false)
		parentNumber := mcp.ParseInt(req, "parent_number", 0)
		if clearParent {
			issue.ParentID = nil
		} else if parentNumber > 0 {
			parent, err := st.GetIssueByNumber(ctx, p.ID, parentNumber)
			if err != nil {
				return errResult(fmt.Errorf("parent issue #%d not found in project", parentNumber)), nil //nolint:nilerr // error conveyed via errResult
			}
			if parent.ID == issue.ID {
				return errResult(fmt.Errorf("issue cannot be its own parent")), nil
			}
			cycle, err := parentAssignCreatesCycle(ctx, st, issue.ID, parent.ID)
			if err != nil {
				return errResult(fmt.Errorf("validate parent: %w", err)), nil
			}
			if cycle {
				return errResult(fmt.Errorf("setting this parent would create a cycle")), nil
			}
			pid := parent.ID
			issue.ParentID = &pid
		}

		if err := st.UpdateIssue(ctx, issue); err != nil {
			return errResult(fmt.Errorf("update issue: %w", err)), nil
		}
		return textResult(fmt.Sprintf("Updated %s-%d", key, number)), nil
	}
}

func toolMoveIssue(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		number := mcp.ParseInt(req, "number", 0)
		status := mcp.ParseString(req, "status", "")

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil //nolint:nilerr // error conveyed via errResult
		}

		oldStatus := issue.Status
		if err := st.UpdateIssueStatus(ctx, issue.ID, status); err != nil {
			return errResult(fmt.Errorf("update issue status: %w", err)), nil
		}
		return textResult(fmt.Sprintf("Moved %s-%d from %s to %s", key, number, oldStatus, status)), nil
	}
}

func toolAddComment(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		number := mcp.ParseInt(req, "number", 0)
		body := mcp.ParseString(req, "body", "")

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil //nolint:nilerr // error conveyed via errResult
		}

		userID, err := userFromCtx(ctx)
		if err != nil {
			return errResult(err), nil
		}

		c := &model.Comment{
			IssueID:  issue.ID,
			AuthorID: userID,
			Body:     body,
		}
		if err := st.CreateComment(ctx, c); err != nil {
			return errResult(fmt.Errorf("create comment: %w", err)), nil
		}
		return textResult(fmt.Sprintf("Added comment to %s-%d", key, number)), nil
	}
}

// resolveOrCreateLabel returns an existing project label by name or creates it.
// If two callers create the same name concurrently, CreateLabel may fail on the
// unique (project_id, name) constraint; we then load the row another writer inserted.
func resolveOrCreateLabel(ctx context.Context, st *store.Store, projectID, labelName, color string) (*model.Label, error) {
	lbl, err := st.GetLabelByName(ctx, projectID, labelName)
	if err != nil {
		return nil, fmt.Errorf("lookup label: %w", err)
	}
	if lbl != nil {
		return lbl, nil
	}
	lbl = &model.Label{ProjectID: projectID, Name: labelName, Color: color}
	if err := st.CreateLabel(ctx, lbl); err != nil {
		lbl, lookupErr := st.GetLabelByName(ctx, projectID, labelName)
		if lookupErr != nil {
			return nil, fmt.Errorf("create label: %w", err)
		}
		if lbl == nil {
			return nil, fmt.Errorf("create label: %w", err)
		}
	}
	return lbl, nil
}

func toolAddIssueLabel(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		number := mcp.ParseInt(req, "number", 0)
		labelName := strings.TrimSpace(mcp.ParseString(req, "label_name", ""))
		if labelName == "" {
			return errResult(fmt.Errorf("label_name is required")), nil
		}

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil //nolint:nilerr // error conveyed via errResult
		}

		color := strings.TrimSpace(mcp.ParseString(req, "color", ""))
		if color == "" {
			color = "#64748b"
		}

		lbl, err := resolveOrCreateLabel(ctx, st, p.ID, labelName, color)
		if err != nil {
			return errResult(err), nil
		}

		if err := st.AddIssueLabel(ctx, issue.ID, lbl.ID); err != nil {
			return errResult(fmt.Errorf("add issue label: %w", err)), nil
		}
		return textResult(fmt.Sprintf("Added label %q to %s-%d", lbl.Name, key, number)), nil
	}
}

func toolSearchIssues(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := mcp.ParseString(req, "query", "")
		issues, total, err := st.ListIssues(ctx, model.IssueFilter{
			Query: query,
			Limit: 50,
		})
		if err != nil {
			return errResult(fmt.Errorf("search issues: %w", err)), nil
		}
		return textResult(fmt.Sprintf("Found %d issues:\n%s", total, toJSON(issues))), nil
	}
}

// userFromCtx returns the authenticated user's ID from context (set by bearer
// token auth for HTTP MCP, or by the Sidecar proxy for stdio clients).
func userFromCtx(ctx context.Context) (string, error) {
	if u := auth.UserFromContext(ctx); u != nil {
		return u.ID, nil
	}
	return "", fmt.Errorf("authenticate with Authorization: Bearer <access_token> from POST /api/v1/auth/token")
}

func toolStartTimer(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		number := mcp.ParseInt(req, "number", 0)
		desc := mcp.ParseString(req, "description", "")

		userID, err := userFromCtx(ctx)
		if err != nil {
			return errResult(err), nil
		}

		if !StrictAgentWorkflow() {
			if err := auth.AgentMCPAuthError(ctx); err != nil {
				return errResult(err), nil
			}
			p, err := st.GetProjectByKey(ctx, key)
			if err != nil {
				return errResult(fmt.Errorf("get project: %w", err)), nil
			}
			issue, err := st.GetIssueByNumber(ctx, p.ID, number)
			if err != nil {
				return errResult(fmt.Errorf("get issue: %w", err)), nil
			}
			entry, err := st.StartTimer(ctx, issue.ID, userID, "agent", "", desc, auth.MCPActorIDFromContext(ctx))
			if err != nil {
				return errResult(fmt.Errorf("start timer: %w", err)), nil
			}
			return textResult(fmt.Sprintf("Timer started on %s-%d (timer_id: %s, actor: agent, started: %s)",
				key, number, entry.ID, entry.StartedAt.Format(time.RFC3339))), nil
		}

		_, entry, _, err := beginAgentWork(ctx, st, userID, key, number, desc)
		if err != nil {
			return errResult(err), nil
		}

		return textResult(fmt.Sprintf("Timer started on %s-%d (timer_id: %s, actor: agent, started: %s) — same workflow as begin_work",
			key, number, entry.ID, entry.StartedAt.Format(time.RFC3339))), nil
	}
}

func toolStopTimer(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		timerID := mcp.ParseString(req, "timer_id", "")
		key := mcp.ParseString(req, "project_key", "")
		number := mcp.ParseInt(req, "number", 0)
		actorFilter := mcp.ParseString(req, "actor_type", "")

		if timerID != "" {
			entry, err := st.StopTimerByID(ctx, timerID)
			if err != nil {
				return errResult(fmt.Errorf("stop timer: %w", err)), nil
			}
			durationMin := float64(*entry.Duration) / 60
			return textResult(fmt.Sprintf("Timer stopped (id: %s). Duration: %.1f minutes", entry.ID, durationMin)), nil
		}

		if key != "" && number > 0 {
			p, err := st.GetProjectByKey(ctx, key)
			if err != nil {
				return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
			}
			issue, err := st.GetIssueByNumber(ctx, p.ID, number)
			if err != nil {
				return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil //nolint:nilerr // error conveyed via errResult
			}

			userID, err := userFromCtx(ctx)
			if err != nil {
				return errResult(err), nil
			}

			timers, err := st.GetActiveTimers(ctx, userID)
			if err != nil {
				return errResult(fmt.Errorf("get active timers: %w", err)), nil
			}
			actorCtx := auth.MCPActorIDFromContext(ctx)
			for i := range timers {
				if timers[i].IssueID != issue.ID {
					continue
				}
				if actorFilter != "" && timers[i].ActorType != actorFilter {
					continue
				}
				if timers[i].ActorType == "agent" {
					if err := auth.AgentMCPAuthError(ctx); err != nil {
						return errResult(err), nil
					}
					if !auth.MCPActorIDsEqual(timers[i].McpActorID, actorCtx) {
						continue
					}
				}
				entry, err := st.StopTimerByID(ctx, timers[i].ID)
				if err != nil {
					return errResult(fmt.Errorf("stop timer: %w", err)), nil
				}
				durationMin := float64(*entry.Duration) / 60
				return textResult(fmt.Sprintf("Timer stopped on %s-%d (%s). Duration: %.1f minutes", key, number, entry.ActorType, durationMin)), nil
			}
			return errResult(fmt.Errorf("no active timer found on %s-%d", key, number)), nil
		}

		return errResult(fmt.Errorf("provide either timer_id or project_key+number")), nil
	}
}

func toolGetSessionStatus(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, err := userFromCtx(ctx)
		if err != nil {
			return errResult(err), nil
		}

		var result strings.Builder

		ws, err := st.GetActiveWorkSession(ctx, userID)
		if err != nil {
			return errResult(fmt.Errorf("get active work session: %w", err)), nil
		}
		if ws != nil {
			elapsed := time.Since(ws.StartedAt)
			fmt.Fprintf(&result, "Work session active: %s (elapsed: %.1f min)\n",
				ws.Description, elapsed.Minutes())
		} else {
			result.WriteString("No active work session.\n")
		}

		timers, err := st.GetActiveTimers(ctx, userID)
		if err != nil {
			return errResult(fmt.Errorf("get active timers: %w", err)), nil
		}
		if len(timers) == 0 {
			result.WriteString("No active timers.\n")
		} else {
			fmt.Fprintf(&result, "\nActive timers (%d):\n", len(timers))
			var totalTimerSecs float64
			for i := range timers {
				elapsed := time.Since(timers[i].StartedAt)
				totalTimerSecs += elapsed.Seconds()
				issueRef := timers[i].IssueID
				if issue, err := st.GetIssue(ctx, timers[i].IssueID); err == nil {
					if p, err := st.GetProjectByID(ctx, issue.ProjectID); err == nil {
						issueRef = fmt.Sprintf("%s-%d", p.Key, issue.Number)
					}
				}
				fmt.Fprintf(&result, "  - %s [%s] timer_id:%s elapsed:%.1fmin\n",
					issueRef, timers[i].ActorType, timers[i].ID, elapsed.Minutes())
			}

			if ws != nil {
				sessionSecs := time.Since(ws.StartedAt).Seconds()
				if sessionSecs > 0 {
					utilization := (totalTimerSecs / sessionSecs) * 100
					fmt.Fprintf(&result, "\nUtilization: %.0f%% (%.1fmin focused / %.1fmin session)\n",
						utilization, totalTimerSecs/60, sessionSecs/60)
				}
			}
		}

		return textResult(result.String()), nil
	}
}

func toolGetTimeEntries(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		number := mcp.ParseInt(req, "number", 0)

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil //nolint:nilerr // error conveyed via errResult
		}

		entries, err := st.ListTimeEntries(ctx, issue.ID)
		if err != nil {
			return errResult(fmt.Errorf("list time entries: %w", err)), nil
		}
		total, err := st.GetIssueTotalTime(ctx, issue.ID)
		if err != nil {
			return errResult(fmt.Errorf("get issue total time: %w", err)), nil
		}
		return textResult(fmt.Sprintf("Time entries (total: %dh %dm):\n%s",
			total/3600, (total%3600)/60, toJSON(entries))), nil
	}
}

func toolGetVelocity(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		weeks := mcp.ParseInt(req, "weeks", 8)

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
		}
		velocity, err := st.GetVelocity(ctx, p.ID, weeks)
		if err != nil {
			return errResult(fmt.Errorf("get velocity: %w", err)), nil
		}
		return textResult(toJSON(velocity)), nil
	}
}

func toolGetEstimationAccuracy(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
		}
		report, err := st.GetEstimationReport(ctx, p.ID)
		if err != nil {
			return errResult(fmt.Errorf("get estimation report: %w", err)), nil
		}
		return textResult(toJSON(report)), nil
	}
}

func toolGetProjectHealth(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
		}
		health, err := st.GetProjectHealth(ctx, p.ID)
		if err != nil {
			return errResult(fmt.Errorf("get project health: %w", err)), nil
		}
		return textResult(toJSON(health)), nil
	}
}

func toolCompareByTag(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		groupBy := mcp.ParseString(req, "group_by", "")
		comparisons, err := st.CompareByTag(ctx, groupBy)
		if err != nil {
			return errResult(fmt.Errorf("compare by tag: %w", err)), nil
		}
		return textResult(toJSON(comparisons)), nil
	}
}

func toolPredictNewProject(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tagsStr := mcp.ParseString(req, "tags", "")
		points := mcp.ParseInt(req, "points", 0)

		var tags []string
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(strings.ToLower(t))
			if t != "" {
				tags = append(tags, t)
			}
		}

		pred, err := st.PredictNewProject(ctx, tags, points)
		if err != nil {
			return errResult(err), nil
		}
		return textResult(toJSON(pred)), nil
	}
}

// --- Compound workflow tools ---

func toolBeginWork(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		number := mcp.ParseInt(req, "number", 0)

		userID, err := userFromCtx(ctx)
		if err != nil {
			return errResult(err), nil
		}

		issue, entry, wsID, err := beginAgentWork(ctx, st, userID, key, number, "")
		if err != nil {
			return errResult(err), nil
		}

		return textResult(fmt.Sprintf("Working on %s-%d: %s\ntimer_id: %s\nsession: %s",
			key, number, issue.Title, entry.ID, wsID)), nil
	}
}

func toolCompleteWork(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		number := mcp.ParseInt(req, "number", 0)
		summary := mcp.ParseString(req, "summary", "")

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil //nolint:nilerr // error conveyed via errResult
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil //nolint:nilerr // error conveyed via errResult
		}

		userID, err := userFromCtx(ctx)
		if err != nil {
			return errResult(err), nil
		}

		if err := auth.AgentMCPAuthError(ctx); err != nil {
			return errResult(err), nil
		}

		if StrictAgentWorkflow() {
			ok, err := activeAgentTimerOnIssue(ctx, st, userID, issue.ID)
			if err != nil {
				return errResult(err), nil
			}
			if !ok {
				return errResult(fmt.Errorf(
					"complete_work requires an active agent timer on this issue; call begin_work (or start_timer with strict workflow) first",
				)), nil
			}
		}

		// Stop this MCP actor's agent timer(s) on this issue (same matching rules as activeAgentTimerOnIssue).
		var durationSec int64
		actorCtx := auth.MCPActorIDFromContext(ctx)
		timers, err := st.GetActiveTimers(ctx, userID)
		if err != nil {
			return errResult(fmt.Errorf("get active timers: %w", err)), nil
		}
		for i := range timers {
			if timers[i].IssueID != issue.ID || timers[i].ActorType != "agent" {
				continue
			}
			if !auth.MCPActorIDsEqual(timers[i].McpActorID, actorCtx) {
				continue
			}
			stopped, err := st.StopTimerByID(ctx, timers[i].ID)
			if err == nil && stopped.Duration != nil {
				durationSec += *stopped.Duration
			}
		}

		// Add summary as a comment.
		if summary != "" {
			c := &model.Comment{
				IssueID:  issue.ID,
				AuthorID: userID,
				Body:     summary,
			}
			if err := st.CreateComment(ctx, c); err != nil {
				return errResult(fmt.Errorf("create comment: %w", err)), nil
			}
		}

		// Move issue to done.
		if issue.Status != "done" {
			if err := st.MoveIssue(ctx, issue.ID, "done", 0); err != nil {
				return errResult(fmt.Errorf("move issue: %w", err)), nil
			}
		}

		durationMin := float64(durationSec) / 60

		//nolint:errcheck // audit log is best-effort after issue completion
		st.LogActivity(ctx, &model.ActivityLog{
			EntityType: "issue",
			EntityID:   issue.ID,
			UserID:     userID,
			Action:     "mcp_complete_work",
			Field:      "mcp_actor_id",
			NewValue:   actorCtx,
		})

		return textResult(fmt.Sprintf("Completed %s-%d: %s\nTime logged: %.1f minutes",
			key, number, issue.Title, durationMin)), nil
	}
}
