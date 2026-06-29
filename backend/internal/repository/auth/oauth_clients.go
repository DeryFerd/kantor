package auth

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	repository "github.com/kana-consultant/kantor/backend/internal/repository"
)

var ErrOAuthClientNotFound = errors.New("oauth client not found")

type OAuthClient struct {
	ClientID     string
	ClientName   string
	RedirectURIs []string
}

func (r *Repository) CreateOAuthClient(ctx context.Context, clientID string, clientName string, redirectURIs []string) error {
	ctx, cancel := repository.QueryContext(ctx)
	defer cancel()

	raw, err := json.Marshal(redirectURIs)
	if err != nil {
		return err
	}

	_, err = repository.DB(ctx, r.db).Exec(ctx, `
		INSERT INTO oauth_clients (client_id, client_name, redirect_uris)
		VALUES ($1, $2, $3)
	`, clientID, clientName, string(raw))
	return err
}

func (r *Repository) GetOAuthClient(ctx context.Context, clientID string) (OAuthClient, error) {
	ctx, cancel := repository.QueryContext(ctx)
	defer cancel()

	var client OAuthClient
	var raw string
	err := repository.DB(ctx, r.db).QueryRow(ctx, `
		SELECT client_id, client_name, redirect_uris
		FROM oauth_clients
		WHERE client_id = $1
	`, clientID).Scan(&client.ClientID, &client.ClientName, &raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OAuthClient{}, ErrOAuthClientNotFound
		}
		return OAuthClient{}, err
	}

	if err := json.Unmarshal([]byte(raw), &client.RedirectURIs); err != nil {
		return OAuthClient{}, err
	}
	return client, nil
}
