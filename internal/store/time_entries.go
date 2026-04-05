package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
)

func normalizeMCPActorID(s string) string {
	return strings.TrimSpace(s)
}

func (s *Store) StartTimer(ctx context.Context, issueID, userID, actorType, sessionID, description, mcpActorID string) (*model.TimeEntry, error) {
	if actorType == "human" && sessionID == "" {
		return nil, fmt.Errorf("human timer requires an active work session")
	}

	mcpActorID = normalizeMCPActorID(mcpActorID)
	var mcpArg interface{}
	if actorType == "agent" {
		if mcpActorID != "" {
			mcpArg = mcpActorID
		} else {
			mcpArg = nil
		}
	} else {
		mcpArg = nil
	}

	entry := &model.TimeEntry{
		ID:          NewID(),
		IssueID:     issueID,
		UserID:      userID,
		ActorType:   actorType,
		SessionID:   sessionID,
		McpActorID:  mcpActorID,
		Description: description,
		StartedAt:   time.Now().UTC(),
		Source:      "timer",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	err := s.writeTx(ctx, func(tx *sql.Tx) error {
		if actorType == "human" {
			// Auto-stop any existing human timer (context switch)
			var prevID string
			var prevStartedAt time.Time
			err := tx.QueryRowContext(ctx,
				`SELECT id, started_at FROM time_entries
				 WHERE user_id = ? AND actor_type = 'human' AND ended_at IS NULL`, userID,
			).Scan(&prevID, &prevStartedAt)
			if err == nil {
				now := time.Now().UTC()
				dur := int64(now.Sub(prevStartedAt).Seconds())
				_, err = tx.ExecContext(ctx,
					`UPDATE time_entries SET ended_at = ?, duration = ?, updated_at = ? WHERE id = ?`,
					now, dur, now, prevID,
				)
				if err != nil {
					return fmt.Errorf("auto-stop previous human timer: %w", err)
				}
			} else if err != sql.ErrNoRows {
				return fmt.Errorf("check existing human timer: %w", err)
			}
		} else {
			// Agent: reject if same issue already has an active agent timer for this MCP actor slot
			// (legacy: mcp_actor_id NULL/empty shares one slot via ifnull(..., '')).
			var existing string
			err := tx.QueryRowContext(ctx,
				`SELECT id FROM time_entries
				 WHERE issue_id = ? AND actor_type = 'agent' AND ended_at IS NULL
				   AND ifnull(mcp_actor_id, '') = ifnull(?, '')`, issueID, mcpArg,
			).Scan(&existing)
			if err == nil {
				return fmt.Errorf("active agent timer already running on this issue (id: %s)", existing)
			}
			if err != sql.ErrNoRows {
				return fmt.Errorf("check existing agent timer: %w", err)
			}
		}

		var sessID interface{}
		if entry.SessionID != "" {
			sessID = entry.SessionID
		}

		var desc interface{}
		if entry.Description != "" {
			desc = entry.Description
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO time_entries (id, issue_id, user_id, session_id, actor_type, mcp_actor_id, description, started_at, source, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			entry.ID, entry.IssueID, entry.UserID,
			sessID, entry.ActorType, mcpArg, desc,
			entry.StartedAt, entry.Source,
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
			`SELECT id, issue_id, user_id, actor_type, started_at FROM time_entries
			 WHERE user_id = ? AND actor_type = 'human' AND ended_at IS NULL`, userID,
		).Scan(&entry.ID, &entry.IssueID, &entry.UserID, &entry.ActorType, &startedAt)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("no active human timer")
			}
			return err
		}

		now := time.Now().UTC()
		duration := int64(now.Sub(startedAt).Seconds())
		entry.StartedAt = startedAt
		entry.EndedAt = &now
		entry.Duration = &duration

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

func (s *Store) StopTimerByID(ctx context.Context, timerID string) (*model.TimeEntry, error) {
	var entry model.TimeEntry
	err := s.writeTx(ctx, func(tx *sql.Tx) error {
		var startedAt time.Time
		var sessionID, mcp sql.NullString
		err := tx.QueryRowContext(ctx,
			`SELECT id, issue_id, user_id, session_id, actor_type, mcp_actor_id, started_at FROM time_entries
			 WHERE id = ? AND ended_at IS NULL`, timerID,
		).Scan(&entry.ID, &entry.IssueID, &entry.UserID, &sessionID, &entry.ActorType, &mcp, &startedAt)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("no active timer with id %s", timerID)
			}
			return fmt.Errorf("find timer: %w", err)
		}
		if sessionID.Valid {
			entry.SessionID = sessionID.String
		}
		if mcp.Valid {
			entry.McpActorID = mcp.String
		}

		now := time.Now().UTC()
		duration := int64(now.Sub(startedAt).Seconds())
		entry.StartedAt = startedAt
		entry.EndedAt = &now
		entry.Duration = &duration

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

func (s *Store) GetActiveTimers(ctx context.Context, userID string) ([]model.TimeEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, issue_id, user_id, session_id, actor_type, mcp_actor_id, description, started_at, source, created_at, updated_at
		 FROM time_entries WHERE user_id = ? AND ended_at IS NULL ORDER BY started_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("get active timers: %w", err)
	}
	defer rows.Close()

	var entries []model.TimeEntry
	for rows.Next() {
		var e model.TimeEntry
		var sessionID, desc, mcp sql.NullString
		if err := rows.Scan(&e.ID, &e.IssueID, &e.UserID, &sessionID, &e.ActorType, &mcp,
			&desc, &e.StartedAt, &e.Source, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		if sessionID.Valid {
			e.SessionID = sessionID.String
		}
		if mcp.Valid {
			e.McpActorID = mcp.String
		}
		if desc.Valid {
			e.Description = desc.String
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) StopOrphanedTimers(ctx context.Context, maxDuration time.Duration) (int, error) {
	cutoff := time.Now().UTC().Add(-maxDuration)
	var count int
	err := s.writeTx(ctx, func(tx *sql.Tx) error {
		now := time.Now().UTC()
		result, err := tx.ExecContext(ctx,
			`UPDATE time_entries
			 SET ended_at = ?, duration = CAST((julianday(?) - julianday(started_at)) * 86400 AS INTEGER), updated_at = ?
			 WHERE ended_at IS NULL AND started_at < ?`,
			now, now, now, cutoff,
		)
		if err != nil {
			return fmt.Errorf("stop orphaned timers: %w", err)
		}
		n, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("rows affected: %w", err)
		}
		count = int(n)
		return nil
	})
	return count, err
}

func (s *Store) GetActiveTimer(ctx context.Context, userID string) (*model.TimeEntry, error) {
	var entry model.TimeEntry
	var desc, sessionID sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, issue_id, user_id, session_id, actor_type, description, started_at, source, created_at, updated_at
		 FROM time_entries WHERE user_id = ? AND actor_type = 'human' AND ended_at IS NULL`, userID,
	).Scan(&entry.ID, &entry.IssueID, &entry.UserID, &sessionID, &entry.ActorType, &desc,
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
	if sessionID.Valid {
		entry.SessionID = sessionID.String
	}
	return &entry, nil
}

func (s *Store) CreateManualTimeEntry(ctx context.Context, entry *model.TimeEntry) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		entry.ID = NewID()
		entry.Source = "manual"
		if entry.ActorType == "" {
			entry.ActorType = "human"
		}
		now := time.Now().UTC()
		entry.CreatedAt = now
		entry.UpdatedAt = now

		var sessID interface{}
		if entry.SessionID != "" {
			sessID = entry.SessionID
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO time_entries (id, issue_id, user_id, session_id, actor_type, description, started_at, ended_at, duration, source, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			entry.ID, entry.IssueID, entry.UserID,
			sessID, entry.ActorType,
			entry.Description,
			entry.StartedAt, nullTime(entry.EndedAt), entry.Duration, entry.Source,
			entry.CreatedAt, entry.UpdatedAt,
		)
		return err
	})
}

func (s *Store) GetTimeEntry(ctx context.Context, id string) (*model.TimeEntry, error) {
	var entry model.TimeEntry
	var desc, sessionID sql.NullString
	var endedAt sql.NullTime
	var duration sql.NullInt64

	var mcp sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, issue_id, user_id, session_id, actor_type, mcp_actor_id, description, started_at, ended_at, duration, source, created_at, updated_at
		 FROM time_entries WHERE id = ?`, id,
	).Scan(&entry.ID, &entry.IssueID, &entry.UserID, &sessionID, &entry.ActorType, &mcp, &desc,
		&entry.StartedAt, &endedAt, &duration, &entry.Source, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("time entry not found")
		}
		return nil, err
	}
	if mcp.Valid {
		entry.McpActorID = mcp.String
	}
	if desc.Valid {
		entry.Description = desc.String
	}
	if sessionID.Valid {
		entry.SessionID = sessionID.String
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
		`SELECT te.id, te.issue_id, te.user_id, te.session_id, te.actor_type,
		        te.description, te.started_at, te.ended_at,
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
		var desc, sessionID, mcp sql.NullString
		var endedAt sql.NullTime
		var duration sql.NullInt64

		if err := rows.Scan(&e.ID, &e.IssueID, &e.UserID, &sessionID, &e.ActorType, &mcp,
			&desc, &e.StartedAt, &endedAt, &duration, &e.Source, &e.CreatedAt, &e.UpdatedAt,
			&u.ID, &u.Name, &u.Email); err != nil {
			return nil, err
		}
		if desc.Valid {
			e.Description = desc.String
		}
		if sessionID.Valid {
			e.SessionID = sessionID.String
		}
		if mcp.Valid {
			e.McpActorID = mcp.String
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
			`UPDATE time_entries SET description=?, started_at=?, ended_at=?, duration=?, actor_type=?, updated_at=?
			 WHERE id=?`,
			entry.Description, entry.StartedAt, nullTime(entry.EndedAt), entry.Duration,
			entry.ActorType, entry.UpdatedAt, entry.ID,
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

// DailySummary holds aggregated time data for a single day.
type DailySummary struct {
	TotalSecs  int64
	HumanSecs  int64
	AgentSecs  int64
	IssueCount int
}

func (s *Store) GetDailySummary(ctx context.Context, userID string, date time.Time) (*DailySummary, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.AddDate(0, 0, 1)

	var total, human, agent sql.NullInt64
	var issueCount int
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(duration), 0),
		        COALESCE(SUM(CASE WHEN actor_type = 'human' THEN duration ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN actor_type = 'agent' THEN duration ELSE 0 END), 0),
		        COUNT(DISTINCT issue_id)
		 FROM time_entries
		 WHERE user_id = ? AND started_at >= ? AND started_at < ? AND duration IS NOT NULL`,
		userID, dayStart, dayEnd,
	).Scan(&total, &human, &agent, &issueCount)
	if err != nil {
		return nil, fmt.Errorf("get daily summary: %w", err)
	}

	return &DailySummary{
		TotalSecs:  total.Int64,
		HumanSecs:  human.Int64,
		AgentSecs:  agent.Int64,
		IssueCount: issueCount,
	}, nil
}

func (s *Store) GetActiveTimersWithIssues(ctx context.Context, userID string) ([]model.TimeEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT te.id, te.issue_id, te.user_id, te.session_id, te.actor_type, te.mcp_actor_id,
		        te.description, te.started_at, te.source, te.created_at, te.updated_at,
		        i.number, i.title, i.project_id, p.key
		 FROM time_entries te
		 JOIN issues i ON te.issue_id = i.id
		 JOIN projects p ON i.project_id = p.id
		 WHERE te.user_id = ? AND te.ended_at IS NULL
		 ORDER BY te.started_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("get active timers with issues: %w", err)
	}
	defer rows.Close()

	var entries []model.TimeEntry
	for rows.Next() {
		var e model.TimeEntry
		var sessionID, desc, mcp sql.NullString
		var issueNumber int
		var issueTitle, projectID, projectKey string
		if err := rows.Scan(&e.ID, &e.IssueID, &e.UserID, &sessionID, &e.ActorType, &mcp,
			&desc, &e.StartedAt, &e.Source, &e.CreatedAt, &e.UpdatedAt,
			&issueNumber, &issueTitle, &projectID, &projectKey); err != nil {
			return nil, err
		}
		if sessionID.Valid {
			e.SessionID = sessionID.String
		}
		if mcp.Valid {
			e.McpActorID = mcp.String
		}
		if desc.Valid {
			e.Description = desc.String
		}
		e.Issue = &model.Issue{Number: issueNumber, Title: issueTitle, ProjectID: projectID}
		e.Issue.ProjectKey = projectKey
		entries = append(entries, e)
	}
	return entries, rows.Err()
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
