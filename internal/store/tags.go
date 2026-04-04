package store

import (
	"context"
	"database/sql"

	"yaitracker.com/loweryaustin/internal/model"
)

func (s *Store) GetProjectTags(ctx context.Context, projectID string) ([]model.ProjectTag, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT project_id, tag, COALESCE(group_name, '') FROM project_tags WHERE project_id = ? ORDER BY group_name, tag`,
		projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []model.ProjectTag
	for rows.Next() {
		var t model.ProjectTag
		if err := rows.Scan(&t.ProjectID, &t.Tag, &t.GroupName); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (s *Store) AddProjectTag(ctx context.Context, projectID, tag, groupName string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO project_tags (project_id, tag, group_name) VALUES (?, ?, NULLIF(?, ''))`,
			projectID, tag, groupName,
		)
		return err
	})
}

func (s *Store) RemoveProjectTag(ctx context.Context, projectID, tag string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`DELETE FROM project_tags WHERE project_id = ? AND tag = ?`,
			projectID, tag,
		)
		return err
	})
}

type TagInfo struct {
	Tag       string `json:"tag"`
	GroupName string `json:"group_name,omitempty"`
	Count     int    `json:"count"`
}

func (s *Store) ListAllTags(ctx context.Context) ([]TagInfo, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT tag, COALESCE(group_name, ''), COUNT(*) as cnt
		 FROM project_tags GROUP BY tag, group_name ORDER BY cnt DESC, tag`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []TagInfo
	for rows.Next() {
		var t TagInfo
		if err := rows.Scan(&t.Tag, &t.GroupName, &t.Count); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (s *Store) ListTagGroups(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT group_name FROM project_tags WHERE group_name IS NOT NULL ORDER BY group_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (s *Store) SuggestTags(ctx context.Context, query string) ([]TagInfo, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT tag, COALESCE(group_name, ''), COUNT(*) as cnt
		 FROM project_tags WHERE tag LIKE ? GROUP BY tag, group_name ORDER BY cnt DESC LIMIT 20`,
		"%"+query+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []TagInfo
	for rows.Next() {
		var t TagInfo
		if err := rows.Scan(&t.Tag, &t.GroupName, &t.Count); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
