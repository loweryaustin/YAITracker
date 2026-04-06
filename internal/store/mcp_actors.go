package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
)

// CreateMCPActor registers a new MCP actor for the user and returns it with a server-issued id.
func (s *Store) CreateMCPActor(ctx context.Context, userID, label string) (*model.MCPActor, error) {
	now := time.Now().UTC()
	a := &model.MCPActor{
		ID:              NewID(),
		UserID:          userID,
		Label:           label,
		CreatedAt:       now,
		LastHeartbeatAt: &now,
	}
	err := s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO mcp_actors (id, user_id, label, created_at, last_heartbeat_at) VALUES (?, ?, ?, ?, ?)`,
			a.ID, a.UserID, a.Label, a.CreatedAt, a.LastHeartbeatAt,
		)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create mcp actor: %w", err)
	}
	return a, nil
}

// GetMCPActorForUser returns the actor if it belongs to the user and is not revoked.
func (s *Store) GetMCPActorForUser(ctx context.Context, userID, actorID string) (*model.MCPActor, error) {
	var a model.MCPActor
	var hb sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, label, created_at, last_heartbeat_at FROM mcp_actors
		 WHERE id = ? AND user_id = ? AND revoked_at IS NULL`,
		actorID, userID,
	).Scan(&a.ID, &a.UserID, &a.Label, &a.CreatedAt, &hb)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("mcp actor not found or revoked")
		}
		return nil, fmt.Errorf("get mcp actor: %w", err)
	}
	if hb.Valid {
		a.LastHeartbeatAt = &hb.Time
	}
	return &a, nil
}

// ListMCPActors lists non-revoked actors for a user.
func (s *Store) ListMCPActors(ctx context.Context, userID string) (out []model.MCPActor, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, label, created_at, revoked_at, last_heartbeat_at FROM mcp_actors
		 WHERE user_id = ? AND revoked_at IS NULL ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list mcp actors: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("list mcp actors: %w", cerr)
		}
	}()

	for rows.Next() {
		var a model.MCPActor
		var revoked, hb sql.NullTime
		if scanErr := rows.Scan(&a.ID, &a.UserID, &a.Label, &a.CreatedAt, &revoked, &hb); scanErr != nil {
			return nil, scanErr
		}
		if revoked.Valid {
			a.RevokedAt = &revoked.Time
		}
		if hb.Valid {
			a.LastHeartbeatAt = &hb.Time
		}
		out = append(out, a)
	}
	if rerr := rows.Err(); rerr != nil {
		return nil, rerr
	}
	return out, nil
}

// TouchActorHeartbeat updates last_heartbeat_at for a non-revoked actor owned by userID.
func (s *Store) TouchActorHeartbeat(ctx context.Context, userID, actorID string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE mcp_actors SET last_heartbeat_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL`,
			time.Now().UTC(), actorID, userID,
		)
		if err != nil {
			return fmt.Errorf("touch actor heartbeat: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("mcp actor not found or revoked")
		}
		return nil
	})
}

// RevokeExpiredActors revokes actors whose last heartbeat is older than maxAge and stops
// any open agent timers that reference those actors. Returns the number of actors revoked.
func (s *Store) RevokeExpiredActors(ctx context.Context, maxAge time.Duration) (int64, error) {
	var count int64
	now := time.Now().UTC()
	cutoff := now.Add(-maxAge)
	err := s.writeTx(ctx, func(tx *sql.Tx) error {
		// Stop open agent timers for actors about to be revoked.
		if _, err := tx.ExecContext(ctx,
			`UPDATE time_entries
			 SET ended_at = ?,
			     duration = CAST((julianday(?) - julianday(started_at)) * 86400 AS INTEGER),
			     updated_at = ?
			 WHERE ended_at IS NULL
			   AND actor_type = 'agent'
			   AND mcp_actor_id IN (
			       SELECT id FROM mcp_actors
			       WHERE revoked_at IS NULL
			         AND last_heartbeat_at IS NOT NULL
			         AND last_heartbeat_at < ?
			   )`,
			now, now, now, cutoff,
		); err != nil {
			return fmt.Errorf("stop expired actor timers: %w", err)
		}

		res, err := tx.ExecContext(ctx,
			`UPDATE mcp_actors SET revoked_at = ?
			 WHERE revoked_at IS NULL
			   AND last_heartbeat_at IS NOT NULL
			   AND last_heartbeat_at < ?`,
			now, cutoff,
		)
		if err != nil {
			return fmt.Errorf("revoke expired actors: %w", err)
		}
		count, err = res.RowsAffected()
		return err
	})
	return count, err
}

// RevokeMCPActor marks an actor as revoked so it can no longer be used on MCP requests.
func (s *Store) RevokeMCPActor(ctx context.Context, userID, actorID string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE mcp_actors SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL`,
			time.Now().UTC(), actorID, userID,
		)
		if err != nil {
			return fmt.Errorf("revoke mcp actor: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("mcp actor not found or already revoked")
		}
		return nil
	})
}
