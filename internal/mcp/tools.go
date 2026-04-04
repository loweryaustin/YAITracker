package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

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
		mcp.WithDescription("List issues for a project with optional filters"),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithString("status", mcp.Description("Filter by status (comma-separated)")),
		mcp.WithString("type", mcp.Description("Filter by type")),
		mcp.WithString("assignee", mcp.Description("Filter by assignee name")),
		mcp.WithString("query", mcp.Description("Search in title and description")),
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

	// Time tracking tools
	s.AddTool(mcp.NewTool("log_time",
		mcp.WithDescription("Log time manually for an issue"),
		mcp.WithString("project_key", mcp.Required(), mcp.Description("Project key")),
		mcp.WithNumber("number", mcp.Required(), mcp.Description("Issue number")),
		mcp.WithNumber("hours", mcp.Required(), mcp.Description("Hours to log")),
		mcp.WithString("description", mcp.Description("What was done")),
	), toolLogTime(st))

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

		// Need a user ID for created_by. Use first admin.
		users, _ := st.ListUsers(ctx)
		if len(users) == 0 {
			return errResult(fmt.Errorf("no users exist")), nil
		}
		creatorID := users[0].ID

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
		return textResult(fmt.Sprintf("%d issues (showing %d):\n%s", total, len(issues), toJSON(issues))), nil
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

		users, _ := st.ListUsers(ctx)
		if len(users) == 0 {
			return errResult(fmt.Errorf("no users exist")), nil
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
			ReporterID: users[0].ID,
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

		users, _ := st.ListUsers(ctx)
		if len(users) == 0 {
			return errResult(fmt.Errorf("no users exist")), nil
		}

		c := &model.Comment{
			IssueID:  issue.ID,
			AuthorID: users[0].ID,
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

func toolLogTime(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := mcp.ParseString(req, "project_key", "")
		number := mcp.ParseInt(req, "number", 0)
		hours := mcp.ParseFloat64(req, "hours", 0)
		desc := mcp.ParseString(req, "description", "")

		p, err := st.GetProjectByKey(ctx, key)
		if err != nil {
			return errResult(fmt.Errorf("project %s not found", key)), nil
		}
		issue, err := st.GetIssueByNumber(ctx, p.ID, number)
		if err != nil {
			return errResult(fmt.Errorf("issue %s-%d not found", key, number)), nil
		}

		users, _ := st.ListUsers(ctx)
		if len(users) == 0 {
			return errResult(fmt.Errorf("no users exist")), nil
		}

		durationSecs := int64(hours * 3600)
		entry := &model.TimeEntry{
			IssueID:     issue.ID,
			UserID:      users[0].ID,
			Description: desc,
			Duration:    &durationSecs,
		}
		// Set started_at for manual entries
		entry.StartedAt = time.Now().UTC()
		st.CreateManualTimeEntry(ctx, entry)

		return textResult(fmt.Sprintf("Logged %.1fh on %s-%d", hours, key, number)), nil
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
