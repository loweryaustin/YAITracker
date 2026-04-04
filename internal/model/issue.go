package model

import "time"

type Issue struct {
	ID             string     `json:"id"`
	ProjectID      string     `json:"project_id"`
	Number         int        `json:"number"`
	Title          string     `json:"title"`
	Description    string     `json:"description,omitempty"`
	Type           string     `json:"type"`
	Status         string     `json:"status"`
	Priority       string     `json:"priority"`
	AssigneeID     *string    `json:"assignee_id,omitempty"`
	ReporterID     string     `json:"reporter_id"`
	ParentID       *string    `json:"parent_id,omitempty"`
	SortOrder      float64    `json:"sort_order"`
	StoryPoints    *int       `json:"story_points,omitempty"`
	EstimatedHours *float64   `json:"estimated_hours,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	Assignee *User   `json:"assignee,omitempty"`
	Reporter *User   `json:"reporter,omitempty"`
	Labels   []Label `json:"labels,omitempty"`
	Project  *Project `json:"project,omitempty"`
}

type IssueFilter struct {
	ProjectID  string
	Status     []string
	Type       []string
	Priority   []string
	AssigneeID string
	ParentID   *string
	Query      string
	SortBy     string
	SortDir    string
	Limit      int
	Offset     int
}

var (
	IssueStatuses  = []string{"backlog", "todo", "in_progress", "in_review", "done", "cancelled"}
	IssueTypes     = []string{"bug", "task", "feature", "improvement"}
	IssuePriorities = []string{"none", "low", "medium", "high", "urgent"}
)

func IsTerminalStatus(status string) bool {
	return status == "done" || status == "cancelled"
}

func IsActiveStatus(status string) bool {
	return status == "in_progress" || status == "in_review"
}
