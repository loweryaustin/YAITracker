package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
)

func (s *Store) CreateOAuthToken(ctx context.Context, userID, clientName string) (*model.OAuthToken, error) {
	tok := &model.OAuthToken{
		ID:               NewID(),
		UserID:           userID,
		AccessToken:      NewToken(),
		RefreshToken:     NewToken(),
		AccessExpiresAt:  time.Now().UTC().Add(time.Hour),
		RefreshExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour),
		ClientName:       clientName,
		CreatedAt:        time.Now().UTC(),
	}

	err := s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO oauth_tokens (id, user_id, access_token, refresh_token, access_expires_at, refresh_expires_at, client_name, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			tok.ID, tok.UserID, tok.AccessToken, tok.RefreshToken,
			tok.AccessExpiresAt, tok.RefreshExpiresAt, tok.ClientName, tok.CreatedAt,
		)
		return err
	})
	if err != nil {
		return nil, err
	}
	return tok, nil
}

func (s *Store) GetOAuthTokenByAccess(ctx context.Context, accessToken string) (*model.OAuthToken, error) {
	var tok model.OAuthToken
	var clientName sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, access_token, refresh_token, access_expires_at, refresh_expires_at, client_name, created_at
		 FROM oauth_tokens WHERE access_token = ? AND access_expires_at > ?`,
		accessToken, time.Now().UTC(),
	).Scan(&tok.ID, &tok.UserID, &tok.AccessToken, &tok.RefreshToken,
		&tok.AccessExpiresAt, &tok.RefreshExpiresAt, &clientName, &tok.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("token not found or expired")
		}
		return nil, err
	}
	if clientName.Valid {
		tok.ClientName = clientName.String
	}
	return &tok, nil
}

func (s *Store) GetOAuthTokenByRefresh(ctx context.Context, refreshToken string) (*model.OAuthToken, error) {
	var tok model.OAuthToken
	var clientName sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, access_token, refresh_token, access_expires_at, refresh_expires_at, client_name, created_at
		 FROM oauth_tokens WHERE refresh_token = ?`,
		refreshToken,
	).Scan(&tok.ID, &tok.UserID, &tok.AccessToken, &tok.RefreshToken,
		&tok.AccessExpiresAt, &tok.RefreshExpiresAt, &clientName, &tok.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("refresh token not found")
		}
		return nil, err
	}
	if clientName.Valid {
		tok.ClientName = clientName.String
	}
	return &tok, nil
}

// RefreshOAuthToken implements refresh token rotation: deletes the old token and creates a new one.
func (s *Store) RefreshOAuthToken(ctx context.Context, refreshToken string) (*model.OAuthToken, error) {
	old, err := s.GetOAuthTokenByRefresh(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	if old.RefreshExpiresAt.Before(time.Now().UTC()) {
		return nil, fmt.Errorf("refresh token expired")
	}

	var newTok *model.OAuthToken
	err = s.writeTx(ctx, func(tx *sql.Tx) error {
		// Delete old token
		if _, err := tx.ExecContext(ctx, `DELETE FROM oauth_tokens WHERE id = ?`, old.ID); err != nil {
			return err
		}

		// Create new token
		newTok = &model.OAuthToken{
			ID:               NewID(),
			UserID:           old.UserID,
			AccessToken:      NewToken(),
			RefreshToken:     NewToken(),
			AccessExpiresAt:  time.Now().UTC().Add(time.Hour),
			RefreshExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour),
			ClientName:       old.ClientName,
			CreatedAt:        time.Now().UTC(),
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO oauth_tokens (id, user_id, access_token, refresh_token, access_expires_at, refresh_expires_at, client_name, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			newTok.ID, newTok.UserID, newTok.AccessToken, newTok.RefreshToken,
			newTok.AccessExpiresAt, newTok.RefreshExpiresAt, newTok.ClientName, newTok.CreatedAt,
		)
		return err
	})
	if err != nil {
		return nil, err
	}
	return newTok, nil
}

func (s *Store) DeleteOAuthToken(ctx context.Context, id string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM oauth_tokens WHERE id = ?`, id)
		return err
	})
}

func (s *Store) RevokeAllUserTokens(ctx context.Context, userID string) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM oauth_tokens WHERE user_id = ?`, userID)
		return err
	})
}

func (s *Store) CleanExpiredTokens(ctx context.Context) (int64, error) {
	var count int64
	err := s.writeTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx,
			`DELETE FROM oauth_tokens WHERE refresh_expires_at < ?`, time.Now().UTC())
		if err != nil {
			return err
		}
		var raErr error
		count, raErr = result.RowsAffected()
		return raErr
	})
	return count, err
}
