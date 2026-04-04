package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
)

func (s *Store) StartTimer(ctx context.Context, issueID, userID string) (*model.TimeEntry, error) {
	entry := &model.TimeEntry{
		ID:        NewID(),
		IssueID:   issueID,
		UserID:    userID,
		StartedAt: time.Now().UTC(),
		Source:    "timer",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	err := s.writeTx(ctx, func(tx *sql.Tx) error {
		// Check for existing active timer
		var existing string
		err := tx.QueryRowContext(ctx,
			`SELECT id FROM time_entries WHERE user_id = ? AND ended_at IS NULL`, userID,
		).Scan(&existing)
		if err == nil {
			return fmt.Errorf("active timer already running (id: %s)", existing)
		}
		if err != sql.ErrNoRows {
			return err
		}

		_, err = tx.ExecContext(ctx,
			`INSERT INTO time_entries (id, issue_id, user_id, started_at, source, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			entry.ID, entry.IssueID, entry.UserID, entry.StartedAt, entry.Source,
			entry.CreatedAt, entry.UpdatedAt,
		)
		return err
	})
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (s *Store) StopTimer(ctx context.Context, userID string) (*model.TimeEntry, error) {
	var entry model.TimeEntry
	err := s.writeTx(ctx, func(tx *sql.Tx) error {
		var startedAt time.Time
		err := tx.QueryRowContext(ctx,
			`SELECT id, issue_id, user_id, started_at FROM time_entries
			 WHERE user_id = ? AND ended_at IS NULL`, userID,
		).Scan(&entry.ID, &entry.IssueID, &entry.UserID, &startedAt)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("no active timer")
			}
			return err
		}

		now := time.Now().UTC()
		duration := int64(now.Sub(startedAt).Seconds())
		entry.StartedAt = startedAt
		entry.EndedAt = &now
		dur := duration
		entry.Duration = &dur

		_, err = tx.ExecContext(ctx,
			`UPDATE time_entries SET ended_at = ?, duration = ?, updated_at = ? WHERE id = ?`,
			now, duration, now, entry.ID,
		)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (s *Store) GetActiveTimer(ctx context.Context, userID string) (*model.TimeEntry, error) {
	var entry model.TimeEntry
	var desc sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, issue_id, user_id, description, started_at, source, created_at, updated_at
		 FROM time_entries WHERE user_id = ? AND ended_at IS NULL`, userID,
	).Scan(&entry.ID, &entry.IssueID, &entry.UserID, &desc,
		&entry.StartedAt, &entry.Source, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if desc.Valid {
		entry.Description = desc.String
	}
	return &entry, nil
}

func (s *Store) CreateManualTimeEntry(ctx context.Context, entry *model.TimeEntry) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		entry.ID = NewID()
		entry.Source = "manual"
		now := time.Now().UTC()
		entry.CreatedAt = now
		entry.UpdatedAt = now

		_, err := tx.ExecContext(ctx,
			`INSERT INTO time_entries (id, issue_id, user_id, description, started_at, ended_at, duration, source, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			entry.ID, entry.IssueID, entry.UserID, entry.Description,
			entry.StartedAt, nullTime(entry.EndedAt), entry.Duration, entry.Source,
			entry.CreatedAt, entry.UpdatedAt,
		)
		return err
	})
}

func (s *Store) GetTimeEntry(ctx context.Context, id string) (*model.TimeEntry, error) {
	var entry model.TimeEntry
	var desc sql.NullString
	var endedAt sql.NullTime
	var duration sql.NullInt64

	err := s.db.QueryRowContext(ctx,
		`SELECT id, issue_id, user_id, description, started_at, ended_at, duration, source, created_at, updated_at
		 FROM time_entries WHERE id = ?`, id,
	).Scan(&entry.ID, &entry.IssueID, &entry.UserID, &desc,
		&entry.StartedAt, &endedAt, &duration, &entry.Source, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("time entry not found")
		}
		return nil, err
	}
	if desc.Valid {
		entry.Description = desc.String
	}
	entry.EndedAt = scanNullTime(endedAt)
	if duration.Valid {
		d := duration.Int64
		entry.Duration = &d
	}
	return &entry, nil
}

func (s *Store) ListTimeEntries(ctx context.Context, issueID string) ([]model.TimeEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT te.id, te.issue_id, te.user_id, te.description, te.started_at, te.ended_at,
		        te.duration, te.source, te.created_at, te.updated_at,
		        u.id, u.name, u.email
		 FROM time_entries te JOIN users u ON te.user_id = u.id
		 WHERE te.issue_id = ? ORDER BY te.started_at DESC`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.TimeEntry
	for rows.Next() {
		var e model.TimeEntry
		var u model.User
		var desc sql.NullString
		var endedAt sql.NullTime
		var duration sql.NullInt64

		if err := rows.Scan(&e.ID, &e.IssueID, &e.UserID, &desc,
			&e.StartedAt, &endedAt, &duration, &e.Source, &e.CreatedAt, &e.UpdatedAt,
			&u.ID, &u.Name, &u.Email); err != nil {
			return nil, err
		}
		if desc.Valid {
			e.Description = desc.String
		}
		e.EndedAt = scanNullTime(endedAt)
		if duration.Valid {
			d := duration.Int64
			e.Duration = &d
		}
		e.User = &u
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) UpdateTimeEntry(ctx context.Context, entry *model.TimeEntry) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		entry.UpdatedAt = time.Now().UTC()
		_, err := tx.ExecContext(ctx,
			`UPDATE time_entries SET description=?, started_at=?, ended_at=?, duration=?, updated_at=?
			 WHERE id=?`,
			entry.Description, entry.StartedAt, nullTime(entry.EndedAt), entry.Duration,
			entry.UpdatedAt, entry.ID,
		)
		return err
	})
}

func (s *Store) DeleteTimeEntry(ctx context.Context, id string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM time_entries WHERE id = ?`, id)
		return err
	})
}

func (s *Store) GetIssueTotalTime(ctx context.Context, issueID string) (int64, error) {
	var total sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(duration), 0) FROM time_entries WHERE issue_id = ? AND duration IS NOT NULL`,
		issueID,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	if total.Valid {
		return total.Int64, nil
	}
	return 0, nil
}
