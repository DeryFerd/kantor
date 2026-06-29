package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/kana-consultant/kantor/backend/internal/model"
	repository "github.com/kana-consultant/kantor/backend/internal/repository"
)

var ErrPersonalAccessTokenNotFound = errors.New("personal access token not found")

type CreatePersonalAccessTokenParams struct {
	UserID      string
	Name        string
	TokenHash   string
	TokenPrefix string
	ExpiresAt   *time.Time
}

func (r *Repository) CreatePersonalAccessToken(ctx context.Context, params CreatePersonalAccessTokenParams) (model.PersonalAccessToken, error) {
	ctx, cancel := repository.QueryContext(ctx)
	defer cancel()

	var token model.PersonalAccessToken
	var lastUsedAt, expiresAt sql.NullTime
	err := repository.DB(ctx, r.db).QueryRow(ctx, `
		INSERT INTO personal_access_tokens (user_id, name, token_hash, token_prefix, expires_at)
		VALUES ($1::uuid, $2, $3, $4, $5)
		RETURNING id::text, name, token_prefix, last_used_at, expires_at, created_at
	`, params.UserID, params.Name, params.TokenHash, params.TokenPrefix, params.ExpiresAt).Scan(
		&token.ID, &token.Name, &token.TokenPrefix, &lastUsedAt, &expiresAt, &token.CreatedAt,
	)
	if err != nil {
		return model.PersonalAccessToken{}, err
	}

	token.LastUsedAt = authNullTimePointer(lastUsedAt)
	token.ExpiresAt = authNullTimePointer(expiresAt)
	return token, nil
}

func (r *Repository) ListPersonalAccessTokens(ctx context.Context, userID string) ([]model.PersonalAccessToken, error) {
	ctx, cancel := repository.QueryContext(ctx)
	defer cancel()

	rows, err := repository.DB(ctx, r.db).Query(ctx, `
		SELECT id::text, name, token_prefix, last_used_at, expires_at, created_at
		FROM personal_access_tokens
		WHERE user_id = $1::uuid AND revoked_at IS NULL
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.PersonalAccessToken, 0)
	for rows.Next() {
		var token model.PersonalAccessToken
		var lastUsedAt, expiresAt sql.NullTime
		if err := rows.Scan(&token.ID, &token.Name, &token.TokenPrefix, &lastUsedAt, &expiresAt, &token.CreatedAt); err != nil {
			return nil, err
		}
		token.LastUsedAt = authNullTimePointer(lastUsedAt)
		token.ExpiresAt = authNullTimePointer(expiresAt)
		out = append(out, token)
	}
	return out, rows.Err()
}

func (r *Repository) GetActivePersonalAccessTokenByHash(ctx context.Context, tokenHash string) (userID string, tenantID string, err error) {
	ctx, cancel := repository.QueryContext(ctx)
	defer cancel()

	err = repository.DB(ctx, r.db).QueryRow(ctx, `
		SELECT user_id::text, tenant_id::text
		FROM personal_access_tokens
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > NOW())
	`, tokenHash).Scan(&userID, &tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrPersonalAccessTokenNotFound
		}
		return "", "", err
	}
	return userID, tenantID, nil
}

func (r *Repository) TouchPersonalAccessToken(ctx context.Context, tokenHash string) error {
	ctx, cancel := repository.QueryContext(ctx)
	defer cancel()

	_, err := repository.DB(ctx, r.db).Exec(ctx, `
		UPDATE personal_access_tokens SET last_used_at = NOW() WHERE token_hash = $1
	`, tokenHash)
	return err
}

func (r *Repository) RevokePersonalAccessToken(ctx context.Context, userID string, tokenID string) error {
	ctx, cancel := repository.QueryContext(ctx)
	defer cancel()

	tag, err := repository.DB(ctx, r.db).Exec(ctx, `
		UPDATE personal_access_tokens
		SET revoked_at = NOW()
		WHERE id = $1::uuid AND user_id = $2::uuid AND revoked_at IS NULL
	`, tokenID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPersonalAccessTokenNotFound
	}
	return nil
}
