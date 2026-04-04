package model

import "time"

type Comment struct {
	ID        string    `json:"id"`
	IssueID   string    `json:"issue_id"`
	AuthorID  string    `json:"author_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Author *User `json:"author,omitempty"`
}

type Label struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
}

type ActivityLog struct {
	ID         string    `json:"id"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id,omitempty"`
	UserID     string    `json:"user_id"`
	Action     string    `json:"action"`
	Field      string    `json:"field,omitempty"`
	OldValue   string    `json:"old_value,omitempty"`
	NewValue   string    `json:"new_value,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	CreatedAt  time.Time `json:"created_at"`

	User *User `json:"user,omitempty"`
}
