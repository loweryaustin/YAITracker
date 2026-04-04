package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
)

func (s *Store) CreateSession(ctx context.Context, userID string, duration time.Duration) (*model.Session, error) {
	sess := &model.Session{
		ID:        NewToken(),
		UserID:    userID,
		ExpiresAt: time.Now().UTC().Add(duration),
		CreatedAt: time.Now().UTC(),
	}

	err := s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO sessions (id, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`,
			sess.ID, sess.UserID, sess.ExpiresAt, sess.CreatedAt,
		)
		return err
	})
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *Store) GetSession(ctx context.Context, id string) (*model.Session, error) {
	var sess model.Session
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = ? AND expires_at > ?`,
		id, time.Now().UTC(),
	).Scan(&sess.ID, &sess.UserID, &sess.ExpiresAt, &sess.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found or expired")
		}
		return nil, err
	}
	return &sess, nil
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
		return err
	})
}

func (s *Store) DeleteUserSessions(ctx context.Context, userID string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
		return err
	})
}

func (s *Store) CleanExpiredSessions(ctx context.Context) (int64, error) {
	var count int64
	err := s.writeTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, time.Now().UTC())
		if err != nil {
			return err
		}
		count, _ = result.RowsAffected()
		return nil
	})
	return count, err
}
