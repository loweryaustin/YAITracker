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
		mcp.WithDescription("List issues for a project. Returns concise one-line-per-issue format by default; set format=json for full detail."),
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

	s.AddTool(mcp.NewTool("search_issues",
		mcp.WithDescription("Search issues across all projects"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
	), toolSearchIssues(st))

	// Work session tools
	s.AddTool(mcp.NewTool("start_session",
		mcp.WithDescription("Start a human work session (clock in). Only one active session per user."),
		mcp.WithString("description", mcp.Description("Session description (e.g. 'morning dev block')")),
	), toolStartSession(st))

	s.AddTool(mcp.NewTool("end_session",
		mcp.WithDescription("End the active work session (clock out). Auto-stops any running human timer. Returns session summary with duration."),
	), toolEndSession(st))

	// Timer tools
	s.AddTool(mcp.NewTool("start_timer",
		mcp.WithDescription("Start a real-time timer on an issue. Returns timer_id for later stop_timer call. Human timers require an active session and auto-stop previous human timer. Agent timers can run concurrently on different issues."),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Issue number")),
		mcp.WithString("description", mcp.Description("What will be worked on")),
		mcp.WithString("actor_type", mcp.Description("'human' or 'agent' (default: agent)")),
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
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
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

		creatorID, err := userFromCtx(ctx, st)
		if err != nil {
			return errResult(err), nil
		}

		p := &model.Project{
			Key:       key,
			Name:      name,
			Description: mcp.ParseString(req, "description", ""),
			Status:    "active",
			CreatedBy: creatorID,
		}

		if err := st.CreateProject(ctx, p); err != nil {
			return errResult(err), nil
		}

		if tags := mcp.ParseString(req, "tags", ""); tags != "" {
			for _, t := range strings.Split(tags, ",") {
				t = strings.TrimSpace(strings.ToLower(t))
				if t != "" {
					st.AddProjectTag(ctx, p.ID, t, "")
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
			return errResult(fmt.Errorf("project %s not found", key)), nil
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
			return errResult(fmt.Errorf("project %s not found", key)), nil
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil
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
			return errResult(fmt.Errorf("project %s not found", key)), nil
		}

		if action == "remove" {
			st.RemoveProjectTag(ctx, p.ID, tag)
			return textResult(fmt.Sprintf("Removed tag '%s' from %s", tag, key)), nil
		}

		st.AddProjectTag(ctx, p.ID, tag, group)
		return textResult(fmt.Sprintf("Added tag '%s' to %s", tag, key)), nil
	}
}

func toolListTags(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tags, _ := st.ListAllTags(ctx)
		return textResult(toJSON(tags)), nil
	}
}

func toolListIssues(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil
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

		issues, total, _ := st.ListIssues(ctx, filter)

		if mcp.ParseString(req, "format", "") == "json" {
			return textResult(fmt.Sprintf("%d issues (showing %d):\n%s", total, len(issues), toJSON(issues))), nil
		}

		var lines []string
		for _, i := range issues {
			pts := ""
			if i.StoryPoints != nil {
				pts = fmt.Sprintf(", %dpts", *i.StoryPoints)
			}
			lines = append(lines, fmt.Sprintf("%s-%d [%s, %s%s] %s",
				key, i.Number, i.Status, i.Priority, pts, i.Title))
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
			return errResult(fmt.Errorf("project %s not found", key)), nil
		}

		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil
		}

		comments, _ := st.ListComments(ctx, issue.ID)
		timeEntries, _ := st.ListTimeEntries(ctx, issue.ID)
		totalTime, _ := st.GetIssueTotalTime(ctx, issue.ID)
		children, _ := st.GetChildIssues(ctx, issue.ID)

		result := map[string]interface{}{
			"issue":        issue,
			"comments":     comments,
			"time_entries":  timeEntries,
			"total_time_seconds": totalTime,
			"children":     children,
		}
		return textResult(toJSON(result)), nil
	}
}

func toolCreateIssue(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil
		}

		reporterID, err := userFromCtx(ctx, st)
		if err != nil {
			return errResult(err), nil
		}

		issueType := mcp.ParseString(req, "type", "task")
		status := mcp.ParseString(req, "status", "backlog")
		priority := mcp.ParseString(req, "priority", "none")

		issue := &model.Issue{
			ProjectID:  p.ID,
			Title:      mcp.ParseString(req, "title", ""),
			Description: mcp.ParseString(req, "description", ""),
			Type:       issueType,
			Status:     status,
			Priority:   priority,
			ReporterID: reporterID,
		}

		sp := mcp.ParseInt(req, "story_points", 0)
		if sp > 0 {
			issue.StoryPoints = &sp
		}
		eh := mcp.ParseFloat64(req, "estimated_hours", 0)
		if eh > 0 {
			issue.EstimatedHours = &eh
		}

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
			return errResult(fmt.Errorf("project %s not found", key)), nil
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil
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

		st.UpdateIssue(ctx, issue)
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
			return errResult(fmt.Errorf("project %s not found", key)), nil
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil
		}

		oldStatus := issue.Status
		st.UpdateIssueStatus(ctx, issue.ID, status)
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
			return errResult(fmt.Errorf("project %s not found", key)), nil
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil
		}

		userID, err := userFromCtx(ctx, st)
		if err != nil {
			return errResult(err), nil
		}

		c := &model.Comment{
			IssueID:  issue.ID,
			AuthorID: userID,
			Body:     body,
		}
		st.CreateComment(ctx, c)
		return textResult(fmt.Sprintf("Added comment to %s-%d", key, number)), nil
	}
}

func toolSearchIssues(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := mcp.ParseString(req, "query", "")
		issues, total, _ := st.ListIssues(ctx, model.IssueFilter{
			Query: query,
			Limit: 50,
		})
		return textResult(fmt.Sprintf("Found %d issues:\n%s", total, toJSON(issues))), nil
	}
}

// userFromCtx returns the authenticated user's ID from context (set by bearer
// token auth). Falls back to the first user in the database for backward
// compatibility with unauthenticated clients.
func userFromCtx(ctx context.Context, st *store.Store) (string, error) {
	if u := auth.UserFromContext(ctx); u != nil {
		return u.ID, nil
	}
	users, err := st.ListUsers(ctx)
	if err != nil || len(users) == 0 {
		return "", fmt.Errorf("no users exist -- configure a bearer token in .cursor/mcp.json")
	}
	return users[0].ID, nil
}

func toolStartSession(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		desc := mcp.ParseString(req, "description", "")

		userID, err := userFromCtx(ctx, st)
		if err != nil {
			return errResult(err), nil
		}

		ws, err := st.CreateWorkSession(ctx, userID, desc)
		if err != nil {
			return errResult(fmt.Errorf("start session: %w", err)), nil
		}

		return textResult(fmt.Sprintf("Work session started (id: %s) at %s", ws.ID, ws.StartedAt.Format(time.RFC3339))), nil
	}
}

func toolEndSession(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, err := userFromCtx(ctx, st)
		if err != nil {
			return errResult(err), nil
		}

		ws, err := st.EndWorkSession(ctx, userID)
		if err != nil {
			return errResult(fmt.Errorf("end session: %w", err)), nil
		}

		durationMin := float64(*ws.Duration) / 60
		return textResult(fmt.Sprintf("Work session ended. Duration: %.1f minutes", durationMin)), nil
	}
}

func toolStartTimer(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		number := mcp.ParseInt(req, "number", 0)
		actorType := mcp.ParseString(req, "actor_type", "agent")

		if actorType != "human" && actorType != "agent" {
			return errResult(fmt.Errorf("actor_type must be 'human' or 'agent', got '%s'", actorType)), nil
		}

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil
		}

		userID, err := userFromCtx(ctx, st)
		if err != nil {
			return errResult(err), nil
		}

		sessionID := ""
		if actorType == "human" {
			ws, _ := st.GetActiveWorkSession(ctx, userID)
			if ws == nil {
				return errResult(fmt.Errorf("no active work session -- call start_session first")), nil
			}
			sessionID = ws.ID
		}

		desc := mcp.ParseString(req, "description", "")
		entry, err := st.StartTimer(ctx, issue.ID, userID, actorType, sessionID, desc)
		if err != nil {
			return errResult(fmt.Errorf("start timer: %w", err)), nil
		}

		return textResult(fmt.Sprintf("Timer started on %s-%d (timer_id: %s, actor: %s, started: %s)",
			key, number, entry.ID, actorType, entry.StartedAt.Format(time.RFC3339))), nil
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
				return errResult(fmt.Errorf("project %s not found", key)), nil
			}
			issue, err := st.GetIssueByNumber(ctx, p.ID, number)
			if err != nil {
				return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil
			}

			userID, err := userFromCtx(ctx, st)
			if err != nil {
				return errResult(err), nil
			}

			timers, err := st.GetActiveTimers(ctx, userID)
			if err != nil {
				return errResult(fmt.Errorf("get active timers: %w", err)), nil
			}
			for _, t := range timers {
				if t.IssueID == issue.ID && (actorFilter == "" || t.ActorType == actorFilter) {
					entry, err := st.StopTimerByID(ctx, t.ID)
					if err != nil {
						return errResult(fmt.Errorf("stop timer: %w", err)), nil
					}
					durationMin := float64(*entry.Duration) / 60
					return textResult(fmt.Sprintf("Timer stopped on %s-%d (%s). Duration: %.1f minutes", key, number, entry.ActorType, durationMin)), nil
				}
			}
			return errResult(fmt.Errorf("no active timer found on %s-%d", key, number)), nil
		}

		return errResult(fmt.Errorf("provide either timer_id or project_key+number")), nil
	}
}

func toolGetSessionStatus(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, err := userFromCtx(ctx, st)
		if err != nil {
			return errResult(err), nil
		}

		var result strings.Builder

		ws, _ := st.GetActiveWorkSession(ctx, userID)
		if ws != nil {
			elapsed := time.Since(ws.StartedAt)
			result.WriteString(fmt.Sprintf("Work session active: %s (elapsed: %.1f min)\n",
				ws.Description, elapsed.Minutes()))
		} else {
			result.WriteString("No active work session.\n")
		}

		timers, _ := st.GetActiveTimers(ctx, userID)
		if len(timers) == 0 {
			result.WriteString("No active timers.\n")
		} else {
			result.WriteString(fmt.Sprintf("\nActive timers (%d):\n", len(timers)))
			var totalTimerSecs float64
			for _, t := range timers {
				elapsed := time.Since(t.StartedAt)
				totalTimerSecs += elapsed.Seconds()
				issueRef := t.IssueID
				if issue, err := st.GetIssue(ctx, t.IssueID); err == nil {
					if p, err := st.GetProjectByID(ctx, issue.ProjectID); err == nil {
						issueRef = fmt.Sprintf("%s-%d", p.Key, issue.Number)
					}
				}
				result.WriteString(fmt.Sprintf("  - %s [%s] timer_id:%s elapsed:%.1fmin\n",
					issueRef, t.ActorType, t.ID, elapsed.Minutes()))
			}

			if ws != nil {
				sessionSecs := time.Since(ws.StartedAt).Seconds()
				if sessionSecs > 0 {
					utilization := (totalTimerSecs / sessionSecs) * 100
					result.WriteString(fmt.Sprintf("\nUtilization: %.0f%% (%.1fmin focused / %.1fmin session)\n",
						utilization, totalTimerSecs/60, sessionSecs/60))
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
			return errResult(fmt.Errorf("project %s not found", key)), nil
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil
		}

		entries, _ := st.ListTimeEntries(ctx, issue.ID)
		total, _ := st.GetIssueTotalTime(ctx, issue.ID)
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
			return errResult(fmt.Errorf("project %s not found", key)), nil
		}
		velocity, _ := st.GetVelocity(ctx, p.ID, weeks)
		return textResult(toJSON(velocity)), nil
	}
}

func toolGetEstimationAccuracy(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil
		}
		report, _ := st.GetEstimationReport(ctx, p.ID)
		return textResult(toJSON(report)), nil
	}
}

func toolGetProjectHealth(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil
		}
		health, _ := st.GetProjectHealth(ctx, p.ID)
		return textResult(toJSON(health)), nil
	}
}

func toolCompareByTag(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		groupBy := mcp.ParseString(req, "group_by", "")
		comparisons, _ := st.CompareByTag(ctx, groupBy)
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

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil
		}

		userID, err := userFromCtx(ctx, st)
		if err != nil {
			return errResult(err), nil
		}

		// Ensure a work session exists; create one if missing, update description if reusing.
		sessionDesc := fmt.Sprintf("Working on %s-%d: %s", key, number, issue.Title)
		ws, _ := st.GetActiveWorkSession(ctx, userID)
		if ws == nil {
			ws, err = st.CreateWorkSession(ctx, userID, sessionDesc)
			if err != nil {
				return errResult(fmt.Errorf("create session: %w", err)), nil
			}
		} else {
			st.UpdateWorkSessionDescription(ctx, ws.ID, sessionDesc)
		}

		// Stop any active agent timers so we don't leak time on old issues.
		activeTimers, _ := st.GetActiveTimers(ctx, userID)
		for _, t := range activeTimers {
			if t.ActorType == "agent" {
				st.StopTimerByID(ctx, t.ID)
			}
		}

		// Move issue to in_progress if not already there.
		if issue.Status != "in_progress" {
			if err := st.MoveIssue(ctx, issue.ID, "in_progress", 0); err != nil {
				return errResult(fmt.Errorf("move issue: %w", err)), nil
			}
		}

		// Start an agent timer with the issue title as description.
		entry, err := st.StartTimer(ctx, issue.ID, userID, "agent", "", issue.Title)
		if err != nil {
			return errResult(fmt.Errorf("start timer: %w", err)), nil
		}

		return textResult(fmt.Sprintf("Working on %s-%d: %s\ntimer_id: %s\nsession: %s",
			key, number, issue.Title, entry.ID, ws.ID)), nil
	}
}

func toolCompleteWork(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		number := mcp.ParseInt(req, "number", 0)
		summary := mcp.ParseString(req, "summary", "")

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil
		}

		userID, err := userFromCtx(ctx, st)
		if err != nil {
			return errResult(err), nil
		}

		// Stop active timer(s) on this issue.
		var durationSec int64
		timers, _ := st.GetActiveTimers(ctx, userID)
		for _, t := range timers {
			if t.IssueID == issue.ID {
				stopped, err := st.StopTimerByID(ctx, t.ID)
				if err == nil && stopped.Duration != nil {
					durationSec += *stopped.Duration
				}
			}
		}

		// Add summary as a comment.
		if summary != "" {
			c := &model.Comment{
				IssueID:  issue.ID,
				AuthorID: userID,
				Body:     summary,
			}
			st.CreateComment(ctx, c)
		}

		// Move issue to done.
		if issue.Status != "done" {
			if err := st.MoveIssue(ctx, issue.ID, "done", 0); err != nil {
				return errResult(fmt.Errorf("move issue: %w", err)), nil
			}
		}

		durationMin := float64(durationSec) / 60
		return textResult(fmt.Sprintf("Completed %s-%d: %s\nTime logged: %.1f minutes",
			key, number, issue.Title, durationMin)), nil
	}
}
