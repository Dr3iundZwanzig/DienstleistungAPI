package database

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	CreateRefreshTokenParams
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	RevokedAt *time.Time `json:"revoked_at"`
}

type CreateRefreshTokenParams struct {
	Token     string    `json:"token"`
	UserID    uuid.UUID `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (c Client) CreateRefreshToken(params CreateRefreshTokenParams) (RefreshToken, error) {
	query := `
		INSERT INTO refresh_tokens (
			token,
			created_at,
			updated_at,
			user_id,
			expires_at
		) VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?)
	`
	_, err := c.db.Exec(query, params.Token, params.UserID.String(), params.ExpiresAt)
	if err != nil {
		return RefreshToken{}, err
	}

	now := time.Now().UTC()
	return RefreshToken{
		CreateRefreshTokenParams: params,
		CreatedAt:                now,
		UpdatedAt:                now,
		RevokedAt:                nil,
	}, nil
}

func (c Client) RevokeRefreshToken(tokenHash string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE token = ?
	`
	_, err := c.db.Exec(query, tokenHash)
	return err
}

func (c Client) GetRefreshToken(token string) (RefreshToken, error) {
	query := `
		SELECT token, created_at, updated_at, user_id, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token = ?
	`
	var rt RefreshToken
	var userID string
	err := c.db.QueryRow(query, token).
		Scan(&rt.Token, &rt.CreatedAt, &rt.UpdatedAt, &userID, &rt.ExpiresAt, &rt.RevokedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return RefreshToken{}, nil
		}
		return RefreshToken{}, err
	}

	rt.UserID, err = uuid.Parse(userID)
	if err != nil {
		return RefreshToken{}, err
	}

	return rt, nil
}

func (c Client) DeleteRefreshToken(tokenHash string) error {
	query := `
		DELETE FROM refresh_tokens
		WHERE token = ?
	`
	_, err := c.db.Exec(query, tokenHash)
	return err
}

// revokes token --> neuer refresh token
func (c Client) RotateRefreshToken(oldTokenHash string, params CreateRefreshTokenParams) (RefreshToken, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return RefreshToken{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	revokeQuery := `
		UPDATE refresh_tokens
		SET revoked_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE token = ?
		  AND revoked_at IS NULL
		  AND julianday(expires_at) > julianday('now')
	`
	revokeResult, err := tx.Exec(revokeQuery, oldTokenHash)
	if err != nil {
		return RefreshToken{}, err
	}

	rowsAffected, err := revokeResult.RowsAffected()
	if err != nil {
		return RefreshToken{}, err
	}
	if rowsAffected != 1 {
		return RefreshToken{}, errors.New("refresh token is invalid, expired, or already revoked")
	}

	insertQuery := `
		INSERT INTO refresh_tokens (
			token,
			created_at,
			updated_at,
			user_id,
			expires_at
		) VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?)
	`
	_, err = tx.Exec(insertQuery, params.Token, params.UserID.String(), params.ExpiresAt)
	if err != nil {
		return RefreshToken{}, err
	}

	err = tx.Commit()
	if err != nil {
		return RefreshToken{}, err
	}

	now := time.Now().UTC()
	return RefreshToken{
		CreateRefreshTokenParams: params,
		CreatedAt:                now,
		UpdatedAt:                now,
		RevokedAt:                nil,
	}, nil
}
