package model

import "time"

type Project struct {
	ID          string     `json:"id"`
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
	BudgetHours *float64   `json:"budget_hours,omitempty"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	Tags    []ProjectTag    `json:"tags,omitempty"`
	Members []ProjectMember `json:"members,omitempty"`
}

type ProjectTag struct {
	ProjectID string `json:"project_id"`
	Tag       string `json:"tag"`
	GroupName string `json:"group_name,omitempty"`
}

type ProjectMember struct {
	ProjectID string `json:"project_id"`
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	User      *User  `json:"user,omitempty"`
}

type ProjectSummary struct {
	Project
	TotalIssues      int     `json:"total_issues"`
	OpenIssues       int     `json:"open_issues"`
	InProgressIssues int     `json:"in_progress_issues"`
	DoneIssues       int     `json:"done_issues"`
	TotalTimeSeconds int64   `json:"total_time_seconds"`
	TotalPoints      int     `json:"total_points"`
	DonePoints       int     `json:"done_points"`
	ProgressPercent  float64 `json:"progress_percent"`
}
