package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
)

func (s *Store) CreateUser(ctx context.Context, user *model.User) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		user.ID = NewID()
		now := time.Now().UTC()
		user.CreatedAt = now
		user.UpdatedAt = now

		_, err := tx.ExecContext(ctx,
			`INSERT INTO users (id, email, name, password, role, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			user.ID, user.Email, user.Name, user.Password, user.Role, user.CreatedAt, user.UpdatedAt,
		)
		return err
	})
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, email, name, password, role, failed_attempts, locked_until, created_at, updated_at
		 FROM users WHERE id = ?`, id))
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, email, name, password, role, failed_attempts, locked_until, created_at, updated_at
		 FROM users WHERE email = ?`, email))
}

func (s *Store) ListUsers(ctx context.Context) ([]model.User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, email, name, password, role, failed_attempts, locked_until, created_at, updated_at
		 FROM users ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck // best-effort cleanup

	var users []model.User
	for rows.Next() {
		u, err := s.scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func (s *Store) UpdateUser(ctx context.Context, user *model.User) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		user.UpdatedAt = time.Now().UTC()
		_, err := tx.ExecContext(ctx,
			`UPDATE users SET email=?, name=?, role=?, updated_at=? WHERE id=?`,
			user.Email, user.Name, user.Role, user.UpdatedAt, user.ID,
		)
		return err
	})
}

func (s *Store) UpdateUserPassword(ctx context.Context, id, password string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`UPDATE users SET password=?, updated_at=? WHERE id=?`,
			password, time.Now().UTC(), id,
		)
		return err
	})
}

func (s *Store) IncrementFailedAttempts(ctx context.Context, id string) (int, error) {
	var count int
	err := s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`UPDATE users SET failed_attempts = failed_attempts + 1 WHERE id = ?`, id)
		if err != nil {
			return err
		}
		return tx.QueryRowContext(ctx,
			`SELECT failed_attempts FROM users WHERE id = ?`, id).Scan(&count)
	})
	return count, err
}

func (s *Store) LockUser(ctx context.Context, id string, until time.Time) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`UPDATE users SET locked_until = ? WHERE id = ?`, until, id)
		return err
	})
}

func (s *Store) ResetFailedAttempts(ctx context.Context, id string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`UPDATE users SET failed_attempts = 0, locked_until = NULL WHERE id = ?`, id)
		return err
	})
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func (s *Store) scanUser(row *sql.Row) (*model.User, error) {
	var u model.User
	var lockedUntil sql.NullTime
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Password, &u.Role,
		&u.FailedAttempts, &lockedUntil, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	u.LockedUntil = scanNullTime(lockedUntil)
	return &u, nil
}

func (s *Store) scanUserRow(rows *sql.Rows) (*model.User, error) {
	var u model.User
	var lockedUntil sql.NullTime
	err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Password, &u.Role,
		&u.FailedAttempts, &lockedUntil, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.LockedUntil = scanNullTime(lockedUntil)
	return &u, nil
}
