package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
)

func (s *Store) CreateProject(ctx context.Context, p *model.Project) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		p.ID = NewID()
		now := time.Now().UTC()
		p.CreatedAt = now
		p.UpdatedAt = now

		_, err := tx.ExecContext(ctx,
			`INSERT INTO projects (id, key, name, description, status, target_date, budget_hours, created_by, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.ID, p.Key, p.Name, p.Description, p.Status,
			nullTime(p.TargetDate), nullFloat(p.BudgetHours),
			p.CreatedBy, p.CreatedAt, p.UpdatedAt,
		)
		if err != nil {
			return err
		}

		// Add creator as admin member
		_, err = tx.ExecContext(ctx,
			`INSERT INTO project_members (project_id, user_id, role) VALUES (?, ?, 'admin')`,
			p.ID, p.CreatedBy,
		)
		return err
	})
}

func (s *Store) GetProjectByID(ctx context.Context, id string) (*model.Project, error) {
	p, err := s.scanProject(s.db.QueryRowContext(ctx,
		`SELECT id, key, name, description, status, target_date, budget_hours, created_by, created_at, updated_at
		 FROM projects WHERE id = ?`, id))
	if err != nil {
		return nil, err
	}
	p.Tags, _ = s.GetProjectTags(ctx, p.ID)
	return p, nil
}

func (s *Store) GetProjectByKey(ctx context.Context, key string) (*model.Project, error) {
	p, err := s.scanProject(s.db.QueryRowContext(ctx,
		`SELECT id, key, name, description, status, target_date, budget_hours, created_by, created_at, updated_at
		 FROM projects WHERE key = ?`, key))
	if err != nil {
		return nil, err
	}
	p.Tags, _ = s.GetProjectTags(ctx, p.ID)
	return p, nil
}

func (s *Store) ListProjects(ctx context.Context) ([]model.Project, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, key, name, description, status, target_date, budget_hours, created_by, created_at, updated_at
		 FROM projects ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		p, err := s.scanProjectRow(rows)
		if err != nil {
			return nil, err
		}
		p.Tags, _ = s.GetProjectTags(ctx, p.ID)
		projects = append(projects, *p)
	}
	return projects, rows.Err()
}

func (s *Store) ListProjectSummaries(ctx context.Context) ([]model.ProjectSummary, error) {
	projects, err := s.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]model.ProjectSummary, 0, len(projects))
	for _, p := range projects {
		var ps model.ProjectSummary
		ps.Project = p

		s.db.QueryRowContext(ctx,
			`SELECT COUNT(*),
			        SUM(CASE WHEN status NOT IN ('done','cancelled') THEN 1 ELSE 0 END),
			        SUM(CASE WHEN status IN ('in_progress','in_review') THEN 1 ELSE 0 END),
			        SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END),
			        COALESCE(SUM(story_points), 0),
			        COALESCE(SUM(CASE WHEN status = 'done' THEN story_points ELSE 0 END), 0)
			 FROM issues WHERE project_id = ?`, p.ID,
		).Scan(&ps.TotalIssues, &ps.OpenIssues, &ps.InProgressIssues, &ps.DoneIssues,
			&ps.TotalPoints, &ps.DonePoints)

		if ps.TotalIssues > 0 {
			ps.ProgressPercent = float64(ps.DoneIssues) / float64(ps.TotalIssues) * 100
		}

		var totalSecs sql.NullInt64
		s.db.QueryRowContext(ctx,
			`SELECT COALESCE(SUM(duration), 0) FROM time_entries WHERE issue_id IN (SELECT id FROM issues WHERE project_id = ?)`, p.ID,
		).Scan(&totalSecs)
		if totalSecs.Valid {
			ps.TotalTimeSeconds = totalSecs.Int64
		}

		summaries = append(summaries, ps)
	}
	return summaries, nil
}

func (s *Store) UpdateProject(ctx context.Context, p *model.Project) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		p.UpdatedAt = time.Now().UTC()
		_, err := tx.ExecContext(ctx,
			`UPDATE projects SET name=?, description=?, status=?, target_date=?, budget_hours=?, updated_at=?
			 WHERE id=?`,
			p.Name, p.Description, p.Status, nullTime(p.TargetDate), nullFloat(p.BudgetHours),
			p.UpdatedAt, p.ID,
		)
		return err
	})
}

func (s *Store) DeleteProject(ctx context.Context, id string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
		return err
	})
}

// Members

func (s *Store) AddProjectMember(ctx context.Context, projectID, userID, role string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT OR REPLACE INTO project_members (project_id, user_id, role) VALUES (?, ?, ?)`,
			projectID, userID, role,
		)
		return err
	})
}

func (s *Store) RemoveProjectMember(ctx context.Context, projectID, userID string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`DELETE FROM project_members WHERE project_id = ? AND user_id = ?`,
			projectID, userID,
		)
		return err
	})
}

func (s *Store) GetProjectMembers(ctx context.Context, projectID string) ([]model.ProjectMember, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT pm.project_id, pm.user_id, pm.role, u.id, u.email, u.name, u.role
		 FROM project_members pm JOIN users u ON pm.user_id = u.id
		 WHERE pm.project_id = ? ORDER BY u.name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []model.ProjectMember
	for rows.Next() {
		var m model.ProjectMember
		var u model.User
		if err := rows.Scan(&m.ProjectID, &m.UserID, &m.Role, &u.ID, &u.Email, &u.Name, &u.Role); err != nil {
			return nil, err
		}
		m.User = &u
		members = append(members, m)
	}
	return members, rows.Err()
}

func (s *Store) scanProject(row *sql.Row) (*model.Project, error) {
	var p model.Project
	var targetDate sql.NullTime
	var budgetHours sql.NullFloat64
	err := row.Scan(&p.ID, &p.Key, &p.Name, &p.Description, &p.Status,
		&targetDate, &budgetHours, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, err
	}
	p.TargetDate = scanNullTime(targetDate)
	p.BudgetHours = scanNullFloat(budgetHours)
	return &p, nil
}

func (s *Store) scanProjectRow(rows *sql.Rows) (*model.Project, error) {
	var p model.Project
	var targetDate sql.NullTime
	var budgetHours sql.NullFloat64
	err := rows.Scan(&p.ID, &p.Key, &p.Name, &p.Description, &p.Status,
		&targetDate, &budgetHours, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.TargetDate = scanNullTime(targetDate)
	p.BudgetHours = scanNullFloat(budgetHours)
	return &p, nil
}
