package model

import "time"

type TimeEntry struct {
	ID          string     `json:"id"`
	IssueID     string     `json:"issue_id"`
	UserID      string     `json:"user_id"`
	SessionID   string     `json:"session_id,omitempty"`
	ActorType   string     `json:"actor_type"`
	Description string     `json:"description,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	Duration    *int64     `json:"duration,omitempty"`
	Source      string     `json:"source"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	User  *User  `json:"user,omitempty"`
	Issue *Issue `json:"issue,omitempty"`
}

type WorkSession struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Description string     `json:"description,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	Duration    *int64     `json:"duration,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TimesheetRow struct {
	Issue   Issue   `json:"issue"`
	Entries [7]int64 `json:"entries"` // seconds per day, Mon-Sun
	Total   int64   `json:"total"`
}

type TimesheetData struct {
	WeekStart  time.Time      `json:"week_start"`
	Rows       []TimesheetRow `json:"rows"`
	DailyTotal [7]int64       `json:"daily_total"`
	WeekTotal  int64          `json:"week_total"`
}
