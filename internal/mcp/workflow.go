package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strings"

	"yaitracker.com/loweryaustin/internal/model"
	"yaitracker.com/loweryaustin/internal/store"
)

// StrictAgentWorkflow controls whether complete_work requires an active agent timer on the issue
// and aligns start_timer with begin_work. Default is true (strict). Set YAITRACKER_STRICT_AGENT_WORKFLOW
// to false, 0, no, or off to disable for operator/testing.
func StrictAgentWorkflow() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("YAITRACKER_STRICT_AGENT_WORKFLOW")))
	if v == "" {
		return true
	}
	return v != "false" && v != "0" && v != "no" && v != "off"
}

// activeAgentTimerOnIssue reports whether the user has an active agent timer on the given issue.
func activeAgentTimerOnIssue(ctx context.Context, st *store.Store, userID, issueID string) (bool, error) {
	timers, err := st.GetActiveTimers(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("get active timers: %w", err)
	}
	for i := range timers {
		t := &timers[i]
		if t.IssueID == issueID && t.ActorType == "agent" {
			return true, nil
		}
	}
	return false, nil
}

// beginAgentWork ensures a work session, moves the issue to in_progress, and starts an agent timer.
// Used by begin_work and start_timer. It does not stop agent timers on other issues so one user can run
// parallel agent timers on distinct issues.
func beginAgentWork(ctx context.Context, st *store.Store, userID, key string, number int, timerDesc string) (*model.Issue, *model.TimeEntry, string, error) {
	p, err := st.GetProjectByKey(ctx, key)
	if err != nil {
		return nil, nil, "", fmt.Errorf("project %s not found", key)
	}
	issue, err := st.GetIssueByNumber(ctx, p.ID, number)
	if err != nil {
		return nil, nil, "", fmt.Errorf("issue %s-%d not found", key, number)
	}

	desc := strings.TrimSpace(timerDesc)
	if desc == "" {
		desc = issue.Title
	}

	sessionDesc := fmt.Sprintf("Working on %s-%d: %s", key, number, issue.Title)
	ws, err := st.GetActiveWorkSession(ctx, userID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("get active work session: %w", err)
	}
	if ws == nil {
		ws, err = st.CreateWorkSession(ctx, userID, sessionDesc)
		if err != nil {
			return nil, nil, "", fmt.Errorf("create session: %w", err)
		}
	}
	// Work session policy 1a: when a session already exists, do not overwrite its description on
	// every begin_work — parallel agents would fight over the single work_sessions row.

	if issue.Status != "in_progress" {
		if err := st.MoveIssue(ctx, issue.ID, "in_progress", 0); err != nil {
			return nil, nil, "", fmt.Errorf("move issue: %w", err)
		}
	}

	entry, err := st.StartTimer(ctx, issue.ID, userID, "agent", "", desc)
	if err != nil {
		return nil, nil, "", fmt.Errorf("start timer: %w", err)
	}

	return issue, entry, ws.ID, nil
}
