package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
)

func (s *Store) CreateComment(ctx context.Context, c *model.Comment) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		c.ID = NewID()
		now := time.Now().UTC()
		c.CreatedAt = now
		c.UpdatedAt = now

		_, err := tx.ExecContext(ctx,
			`INSERT INTO comments (id, issue_id, author_id, body, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			c.ID, c.IssueID, c.AuthorID, c.Body, c.CreatedAt, c.UpdatedAt,
		)
		return err
	})
}

func (s *Store) GetComment(ctx context.Context, id string) (*model.Comment, error) {
	var c model.Comment
	err := s.db.QueryRowContext(ctx,
		`SELECT c.id, c.issue_id, c.author_id, c.body, c.created_at, c.updated_at
		 FROM comments c WHERE c.id = ?`, id,
	).Scan(&c.ID, &c.IssueID, &c.AuthorID, &c.Body, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("comment not found")
		}
		return nil, err
	}
	return &c, nil
}

func (s *Store) ListComments(ctx context.Context, issueID string) ([]model.Comment, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT c.id, c.issue_id, c.author_id, c.body, c.created_at, c.updated_at,
		        u.id, u.name, u.email
		 FROM comments c JOIN users u ON c.author_id = u.id
		 WHERE c.issue_id = ? ORDER BY c.created_at`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck // best-effort cleanup

	var comments []model.Comment
	for rows.Next() {
		var c model.Comment
		var u model.User
		if err := rows.Scan(&c.ID, &c.IssueID, &c.AuthorID, &c.Body, &c.CreatedAt, &c.UpdatedAt,
			&u.ID, &u.Name, &u.Email); err != nil {
			return nil, err
		}
		c.Author = &u
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (s *Store) UpdateComment(ctx context.Context, c *model.Comment) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		c.UpdatedAt = time.Now().UTC()
		_, err := tx.ExecContext(ctx,
			`UPDATE comments SET body=?, updated_at=? WHERE id=?`,
			c.Body, c.UpdatedAt, c.ID,
		)
		return err
	})
}

func (s *Store) DeleteComment(ctx context.Context, id string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM comments WHERE id = ?`, id)
		return err
	})
}
