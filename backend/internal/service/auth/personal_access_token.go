package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	backendauth "github.com/kana-consultant/kantor/backend/internal/auth"
	"github.com/kana-consultant/kantor/backend/internal/model"
	authrepo "github.com/kana-consultant/kantor/backend/internal/repository/auth"
)

var ErrInvalidPersonalAccessToken = errors.New("invalid personal access token")

type patRepository interface {
	CreatePersonalAccessToken(ctx context.Context, params authrepo.CreatePersonalAccessTokenParams) (model.PersonalAccessToken, error)
	ListPersonalAccessTokens(ctx context.Context, userID string) ([]model.PersonalAccessToken, error)
	GetActivePersonalAccessTokenByHash(ctx context.Context, tokenHash string) (string, string, error)
	TouchPersonalAccessToken(ctx context.Context, tokenHash string) error
	RevokePersonalAccessToken(ctx context.Context, userID string, tokenID string) error
}

type PATService struct {
	repo patRepository
}

func NewPATService(repo patRepository) *PATService {
	return &PATService{repo: repo}
}

func (s *PATService) Issue(ctx context.Context, userID string, name string, expiresInDays *int, now time.Time) (string, model.PersonalAccessToken, error) {
	token, hash, prefix, err := backendauth.GeneratePersonalAccessToken()
	if err != nil {
		return "", model.PersonalAccessToken{}, err
	}

	var expiresAt *time.Time
	if expiresInDays != nil {
		expiry := now.UTC().AddDate(0, 0, *expiresInDays)
		expiresAt = &expiry
	}

	item, err := s.repo.CreatePersonalAccessToken(ctx, authrepo.CreatePersonalAccessTokenParams{
		UserID:      userID,
		Name:        name,
		TokenHash:   hash,
		TokenPrefix: prefix,
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		return "", model.PersonalAccessToken{}, err
	}
	return token, item, nil
}

func (s *PATService) List(ctx context.Context, userID string) ([]model.PersonalAccessToken, error) {
	return s.repo.ListPersonalAccessTokens(ctx, userID)
}

func (s *PATService) Revoke(ctx context.Context, userID string, tokenID string) error {
	return s.repo.RevokePersonalAccessToken(ctx, userID, tokenID)
}

// Authenticate resolves a personal access token to its owning user and tenant.
// It runs under the request's tenant-scoped connection, so RLS guarantees a
// token only resolves within the tenant it was issued for.
func (s *PATService) Authenticate(ctx context.Context, token string) (userID string, tenantID string, err error) {
	if !backendauth.IsPersonalAccessToken(token) {
		return "", "", ErrInvalidPersonalAccessToken
	}

	hash := backendauth.HashRefreshToken(token)
	userID, tenantID, err = s.repo.GetActivePersonalAccessTokenByHash(ctx, hash)
	if err != nil {
		return "", "", err
	}

	if touchErr := s.repo.TouchPersonalAccessToken(ctx, hash); touchErr != nil {
		slog.WarnContext(ctx, "failed to touch personal access token last_used_at", "error", touchErr)
	}
	return userID, tenantID, nil
}
