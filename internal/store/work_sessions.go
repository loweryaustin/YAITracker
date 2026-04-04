package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
)

func (s *Store) CreateWorkSession(ctx context.Context, userID, description string) (*model.WorkSession, error) {
	ws := &model.WorkSession{
		ID:          NewID(),
		UserID:      userID,
		Description: description,
		StartedAt:   time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	err := s.writeTx(ctx, func(tx *sql.Tx) error {
		var existing string
		err := tx.QueryRowContext(ctx,
			`SELECT id FROM work_sessions WHERE user_id = ? AND ended_at IS NULL`, userID,
		).Scan(&existing)
		if err == nil {
			return fmt.Errorf("active work session already exists (id: %s)", existing)
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("check active session: %w", err)
		}

		_, err = tx.ExecContext(ctx,
			`INSERT INTO work_sessions (id, user_id, description, started_at, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			ws.ID, ws.UserID, ws.Description, ws.StartedAt, ws.CreatedAt, ws.UpdatedAt,
		)
		return err
	})
	if err != nil {
		return nil, err
	}
	return ws, nil
}

func (s *Store) GetActiveWorkSession(ctx context.Context, userID string) (*model.WorkSession, error) {
	var ws model.WorkSession
	var desc sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, description, started_at, created_at, updated_at
		 FROM work_sessions WHERE user_id = ? AND ended_at IS NULL`, userID,
	).Scan(&ws.ID, &ws.UserID, &desc, &ws.StartedAt, &ws.CreatedAt, &ws.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get active work session: %w", err)
	}
	if desc.Valid {
		ws.Description = desc.String
	}
	return &ws, nil
}

func (s *Store) EndWorkSession(ctx context.Context, userID string) (*model.WorkSession, error) {
	var ws model.WorkSession
	err := s.writeTx(ctx, func(tx *sql.Tx) error {
		var desc sql.NullString
		var startedAt time.Time
		err := tx.QueryRowContext(ctx,
			`SELECT id, user_id, description, started_at FROM work_sessions
			 WHERE user_id = ? AND ended_at IS NULL`, userID,
		).Scan(&ws.ID, &ws.UserID, &desc, &startedAt)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("no active work session")
			}
			return fmt.Errorf("find active session: %w", err)
		}
		if desc.Valid {
			ws.Description = desc.String
		}
		ws.StartedAt = startedAt

		now := time.Now().UTC()
		duration := int64(now.Sub(startedAt).Seconds())
		ws.EndedAt = &now
		ws.Duration = &duration

		_, err = tx.ExecContext(ctx,
			`UPDATE work_sessions SET ended_at = ?, duration = ?, updated_at = ? WHERE id = ?`,
			now, duration, now, ws.ID,
		)
		if err != nil {
			return fmt.Errorf("end work session: %w", err)
		}

		// Auto-stop human timer if running
		var timerID string
		var timerStartedAt time.Time
		err = tx.QueryRowContext(ctx,
			`SELECT id, started_at FROM time_entries
			 WHERE user_id = ? AND actor_type = 'human' AND ended_at IS NULL`, userID,
		).Scan(&timerID, &timerStartedAt)
		if err == nil {
			timerDuration := int64(now.Sub(timerStartedAt).Seconds())
			_, err = tx.ExecContext(ctx,
				`UPDATE time_entries SET ended_at = ?, duration = ?, updated_at = ? WHERE id = ?`,
				now, timerDuration, now, timerID,
			)
			if err != nil {
				return fmt.Errorf("auto-stop human timer: %w", err)
			}
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("check human timer: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return &ws, nil
}
