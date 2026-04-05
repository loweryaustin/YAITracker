package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"yaitracker.com/loweryaustin/internal/model"
)

func (s *Store) CreateLabel(ctx context.Context, label *model.Label) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		label.ID = NewID()
		_, err := tx.ExecContext(ctx,
			`INSERT INTO labels (id, project_id, name, color) VALUES (?, ?, ?, ?)`,
			label.ID, label.ProjectID, label.Name, label.Color,
		)
		return err
	})
}

func (s *Store) GetLabel(ctx context.Context, id string) (*model.Label, error) {
	var l model.Label
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, color FROM labels WHERE id = ?`, id,
	).Scan(&l.ID, &l.ProjectID, &l.Name, &l.Color)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("label not found")
		}
		return nil, err
	}
	return &l, nil
}

// GetLabelByName returns the label in the project with the given name, compared case-insensitively.
// If no row matches, it returns nil, nil.
func (s *Store) GetLabelByName(ctx context.Context, projectID, name string) (*model.Label, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	var l model.Label
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, color FROM labels WHERE project_id = ? AND lower(name) = lower(?)`,
		projectID, name,
	).Scan(&l.ID, &l.ProjectID, &l.Name, &l.Color)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &l, nil
}

func (s *Store) ListLabels(ctx context.Context, projectID string) ([]model.Label, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, color FROM labels WHERE project_id = ? ORDER BY name`,
		projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []model.Label
	for rows.Next() {
		var l model.Label
		if err := rows.Scan(&l.ID, &l.ProjectID, &l.Name, &l.Color); err != nil {
			return nil, err
		}
		labels = append(labels, l)
	}
	return labels, rows.Err()
}

func (s *Store) UpdateLabel(ctx context.Context, label *model.Label) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`UPDATE labels SET name=?, color=? WHERE id=?`,
			label.Name, label.Color, label.ID,
		)
		return err
	})
}

func (s *Store) DeleteLabel(ctx context.Context, id string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM labels WHERE id = ?`, id)
		return err
	})
}

func (s *Store) AddIssueLabel(ctx context.Context, issueID, labelID string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO issue_labels (issue_id, label_id) VALUES (?, ?)`,
			issueID, labelID,
		)
		return err
	})
}

func (s *Store) RemoveIssueLabel(ctx context.Context, issueID, labelID string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`DELETE FROM issue_labels WHERE issue_id = ? AND label_id = ?`,
			issueID, labelID,
		)
		return err
	})
}

func (s *Store) GetIssueLabels(ctx context.Context, issueID string) ([]model.Label, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT l.id, l.project_id, l.name, l.color
		 FROM labels l JOIN issue_labels il ON l.id = il.label_id
		 WHERE il.issue_id = ? ORDER BY l.name`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []model.Label
	for rows.Next() {
		var l model.Label
		if err := rows.Scan(&l.ID, &l.ProjectID, &l.Name, &l.Color); err != nil {
			return nil, err
		}
		labels = append(labels, l)
	}
	return labels, rows.Err()
}
