package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
)

func (s *Store) CreateIssue(ctx context.Context, issue *model.Issue) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		issue.ID = NewID()
		now := time.Now().UTC()
		issue.CreatedAt = now
		issue.UpdatedAt = now

		// Auto-increment number per project
		var maxNum sql.NullInt64
		if err := tx.QueryRowContext(ctx,
			`SELECT MAX(number) FROM issues WHERE project_id = ?`, issue.ProjectID,
		).Scan(&maxNum); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("scan max number: %w", err)
		}
		if maxNum.Valid {
			issue.Number = int(maxNum.Int64) + 1
		} else {
			issue.Number = 1
		}

		// Auto-set lifecycle timestamps
		if model.IsActiveStatus(issue.Status) {
			issue.StartedAt = &now
		}
		if model.IsTerminalStatus(issue.Status) {
			issue.CompletedAt = &now
		}

		// Default sort order to number * 1000 for spacing
		if issue.SortOrder == 0 {
			issue.SortOrder = float64(issue.Number) * 1000
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO issues (id, project_id, number, title, description, type, status, priority,
			 assignee_id, reporter_id, parent_id, sort_order, story_points, estimated_hours,
			 started_at, completed_at, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			issue.ID, issue.ProjectID, issue.Number, issue.Title, issue.Description,
			issue.Type, issue.Status, issue.Priority,
			nullString(issue.AssigneeID), issue.ReporterID, nullString(issue.ParentID),
			issue.SortOrder, nullInt(issue.StoryPoints), nullFloat(issue.EstimatedHours),
			nullTime(issue.StartedAt), nullTime(issue.CompletedAt),
			issue.CreatedAt, issue.UpdatedAt,
		)
		return err
	})
}

func (s *Store) GetIssue(ctx context.Context, id string) (*model.Issue, error) {
	issue, err := s.scanIssue(s.db.QueryRowContext(ctx,
		`SELECT i.id, i.project_id, i.number, i.title, i.description, i.type, i.status, i.priority,
		 i.assignee_id, i.reporter_id, i.parent_id, i.sort_order, i.story_points, i.estimated_hours,
		 i.started_at, i.completed_at, i.created_at, i.updated_at
		 FROM issues i WHERE i.id = ?`, id))
	if err != nil {
		return nil, err
	}
	issue.Labels, _ = s.GetIssueLabels(ctx, issue.ID) //nolint:errcheck // supplementary data
	return issue, nil
}

func (s *Store) GetIssueByNumber(ctx context.Context, projectID string, number int) (*model.Issue, error) {
	issue, err := s.scanIssue(s.db.QueryRowContext(ctx,
		`SELECT id, project_id, number, title, description, type, status, priority,
		 assignee_id, reporter_id, parent_id, sort_order, story_points, estimated_hours,
		 started_at, completed_at, created_at, updated_at
		 FROM issues WHERE project_id = ? AND number = ?`, projectID, number))
	if err != nil {
		return nil, err
	}
	issue.Labels, _ = s.GetIssueLabels(ctx, issue.ID) //nolint:errcheck // supplementary data
	return issue, nil
}

func (s *Store) ListIssues(ctx context.Context, filter model.IssueFilter) ([]model.Issue, int, error) {
	var conditions []string
	var args []interface{}

	if filter.ProjectID != "" {
		conditions = append(conditions, "i.project_id = ?")
		args = append(args, filter.ProjectID)
	}
	if len(filter.Status) > 0 {
		placeholders := make([]string, len(filter.Status))
		for idx, s := range filter.Status {
			placeholders[idx] = "?"
			args = append(args, s)
		}
		conditions = append(conditions, "i.status IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(filter.Type) > 0 {
		placeholders := make([]string, len(filter.Type))
		for idx, t := range filter.Type {
			placeholders[idx] = "?"
			args = append(args, t)
		}
		conditions = append(conditions, "i.type IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(filter.Priority) > 0 {
		placeholders := make([]string, len(filter.Priority))
		for idx, p := range filter.Priority {
			placeholders[idx] = "?"
			args = append(args, p)
		}
		conditions = append(conditions, "i.priority IN ("+strings.Join(placeholders, ",")+")")
	}
	if filter.AssigneeID != "" {
		conditions = append(conditions, "i.assignee_id = ?")
		args = append(args, filter.AssigneeID)
	}
	if filter.ParentID != nil {
		conditions = append(conditions, "i.parent_id = ?")
		args = append(args, *filter.ParentID)
	}
	if filter.Query != "" {
		conditions = append(conditions, "(i.title LIKE ? OR i.description LIKE ?)")
		q := "%" + filter.Query + "%"
		args = append(args, q, q)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count
	var total int
	countQuery := "SELECT COUNT(*) FROM issues i " + where
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("scan count: %w", err)
	}

	// Sort
	orderBy := "i.created_at DESC"
	if filter.SortBy != "" {
		dir := "ASC"
		if filter.SortDir == "desc" {
			dir = "DESC"
		}
		allowed := map[string]string{
			"number": "i.number", "title": "i.title", "status": "i.status",
			"priority": "i.priority", "created_at": "i.created_at", "updated_at": "i.updated_at",
			"sort_order": "i.sort_order", "story_points": "i.story_points",
		}
		if col, ok := allowed[filter.SortBy]; ok {
			orderBy = col + " " + dir
		}
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 25
	} else if limit > 10000 {
		limit = 10000
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		`SELECT i.id, i.project_id, i.number, i.title, i.description, i.type, i.status, i.priority,
		 i.assignee_id, i.reporter_id, i.parent_id, i.sort_order, i.story_points, i.estimated_hours,
		 i.started_at, i.completed_at, i.created_at, i.updated_at
		 FROM issues i %s ORDER BY %s LIMIT ? OFFSET ?`, where, orderBy)

	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close() //nolint:errcheck // best-effort cleanup

	var issues []model.Issue
	for rows.Next() {
		issue, err := s.scanIssueRow(rows)
		if err != nil {
			return nil, 0, err
		}
		issue.Labels, _ = s.GetIssueLabels(ctx, issue.ID) //nolint:errcheck // supplementary data
		issues = append(issues, *issue)
	}
	return issues, total, rows.Err()
}

func (s *Store) ListIssuesByStatus(ctx context.Context, projectID string) (map[string][]model.Issue, error) {
	issues, _, err := s.ListIssues(ctx, model.IssueFilter{
		ProjectID: projectID,
		SortBy:    "sort_order",
		SortDir:   "asc",
		Limit:     1000,
	})
	if err != nil {
		return nil, err
	}

	result := make(map[string][]model.Issue)
	for _, status := range model.IssueStatuses {
		result[status] = []model.Issue{}
	}
	for i := range issues {
		result[issues[i].Status] = append(result[issues[i].Status], issues[i])
	}
	return result, nil
}

func (s *Store) UpdateIssue(ctx context.Context, issue *model.Issue) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		issue.UpdatedAt = time.Now().UTC()
		_, err := tx.ExecContext(ctx,
			`UPDATE issues SET title=?, description=?, type=?, status=?, priority=?,
			 assignee_id=?, parent_id=?, sort_order=?, story_points=?, estimated_hours=?,
			 started_at=?, completed_at=?, updated_at=?
			 WHERE id=?`,
			issue.Title, issue.Description, issue.Type, issue.Status, issue.Priority,
			nullString(issue.AssigneeID), nullString(issue.ParentID),
			issue.SortOrder, nullInt(issue.StoryPoints), nullFloat(issue.EstimatedHours),
			nullTime(issue.StartedAt), nullTime(issue.CompletedAt),
			issue.UpdatedAt, issue.ID,
		)
		return err
	})
}

// UpdateIssueStatus handles lifecycle timestamp management on status transitions.
func (s *Store) UpdateIssueStatus(ctx context.Context, id, newStatus string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		now := time.Now().UTC()

		var currentStatus string
		var startedAt sql.NullTime
		err := tx.QueryRowContext(ctx,
			`SELECT status, started_at FROM issues WHERE id = ?`, id,
		).Scan(&currentStatus, &startedAt)
		if err != nil {
			return err
		}

		updates := "status = ?, updated_at = ?"
		args := []interface{}{newStatus, now}

		// Set started_at when first moving to an active status
		if model.IsActiveStatus(newStatus) && !startedAt.Valid {
			updates += ", started_at = ?"
			args = append(args, now)
		}

		// Set completed_at when moving to terminal status
		if model.IsTerminalStatus(newStatus) {
			updates += ", completed_at = ?"
			args = append(args, now)
		}

		// Clear completed_at if reopening
		if !model.IsTerminalStatus(newStatus) && model.IsTerminalStatus(currentStatus) {
			updates += ", completed_at = NULL"
		}

		args = append(args, id)
		_, err = tx.ExecContext(ctx,
			fmt.Sprintf("UPDATE issues SET %s WHERE id = ?", updates), args...)
		return err
	})
}

func (s *Store) MoveIssue(ctx context.Context, id, newStatus string, sortOrder float64) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		now := time.Now().UTC()

		var currentStatus string
		var startedAt sql.NullTime
		err := tx.QueryRowContext(ctx,
			`SELECT status, started_at FROM issues WHERE id = ?`, id,
		).Scan(&currentStatus, &startedAt)
		if err != nil {
			return err
		}

		updates := "status = ?, sort_order = ?, updated_at = ?"
		args := []interface{}{newStatus, sortOrder, now}

		if model.IsActiveStatus(newStatus) && !startedAt.Valid {
			updates += ", started_at = ?"
			args = append(args, now)
		}
		if model.IsTerminalStatus(newStatus) {
			updates += ", completed_at = ?"
			args = append(args, now)
		}
		if !model.IsTerminalStatus(newStatus) && model.IsTerminalStatus(currentStatus) {
			updates += ", completed_at = NULL"
		}

		args = append(args, id)
		_, err = tx.ExecContext(ctx,
			fmt.Sprintf("UPDATE issues SET %s WHERE id = ?", updates), args...)
		return err
	})
}

func (s *Store) DeleteIssue(ctx context.Context, id string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM issues WHERE id = ?`, id)
		return err
	})
}

// MapIssueIDToNumber returns every issue id in the project mapped to its display number.
func (s *Store) MapIssueIDToNumber(ctx context.Context, projectID string) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, number FROM issues WHERE project_id = ?`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck // best-effort cleanup

	out := make(map[string]int)
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

// MapParentIDToChildNumbers maps each parent issue id to its direct children's issue numbers (sorted by number).
func (s *Store) MapParentIDToChildNumbers(ctx context.Context, projectID string) (map[string][]int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT parent_id, number FROM issues WHERE project_id = ? AND parent_id IS NOT NULL ORDER BY number`,
		projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck // best-effort cleanup

	out := make(map[string][]int)
	for rows.Next() {
		var pid string
		var n int
		if err := rows.Scan(&pid, &n); err != nil {
			return nil, err
		}
		out[pid] = append(out[pid], n)
	}
	return out, rows.Err()
}

func (s *Store) GetChildIssues(ctx context.Context, parentID string) ([]model.Issue, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, number, title, description, type, status, priority,
		 assignee_id, reporter_id, parent_id, sort_order, story_points, estimated_hours,
		 started_at, completed_at, created_at, updated_at
		 FROM issues WHERE parent_id = ? ORDER BY number`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck // best-effort cleanup

	var issues []model.Issue
	for rows.Next() {
		issue, err := s.scanIssueRow(rows)
		if err != nil {
			return nil, err
		}
		issues = append(issues, *issue)
	}
	return issues, rows.Err()
}

func (s *Store) scanIssue(row *sql.Row) (*model.Issue, error) {
	var i model.Issue
	var assigneeID, parentID sql.NullString
	var storyPoints sql.NullInt64
	var estimatedHours sql.NullFloat64
	var startedAt, completedAt sql.NullTime

	err := row.Scan(&i.ID, &i.ProjectID, &i.Number, &i.Title, &i.Description,
		&i.Type, &i.Status, &i.Priority,
		&assigneeID, &i.ReporterID, &parentID,
		&i.SortOrder, &storyPoints, &estimatedHours,
		&startedAt, &completedAt, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("issue not found")
		}
		return nil, err
	}
	i.AssigneeID = scanNullString(assigneeID)
	i.ParentID = scanNullString(parentID)
	i.StoryPoints = scanNullInt(storyPoints)
	i.EstimatedHours = scanNullFloat(estimatedHours)
	i.StartedAt = scanNullTime(startedAt)
	i.CompletedAt = scanNullTime(completedAt)
	return &i, nil
}

func (s *Store) scanIssueRow(rows *sql.Rows) (*model.Issue, error) {
	var i model.Issue
	var assigneeID, parentID sql.NullString
	var storyPoints sql.NullInt64
	var estimatedHours sql.NullFloat64
	var startedAt, completedAt sql.NullTime

	err := rows.Scan(&i.ID, &i.ProjectID, &i.Number, &i.Title, &i.Description,
		&i.Type, &i.Status, &i.Priority,
		&assigneeID, &i.ReporterID, &parentID,
		&i.SortOrder, &storyPoints, &estimatedHours,
		&startedAt, &completedAt, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		return nil, err
	}
	i.AssigneeID = scanNullString(assigneeID)
	i.ParentID = scanNullString(parentID)
	i.StoryPoints = scanNullInt(storyPoints)
	i.EstimatedHours = scanNullFloat(estimatedHours)
	i.StartedAt = scanNullTime(startedAt)
	i.CompletedAt = scanNullTime(completedAt)
	return &i, nil
}
