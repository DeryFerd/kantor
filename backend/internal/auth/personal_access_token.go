package auth

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

const (
	PersonalAccessTokenPrefix           = "kantor_pat_"
	personalAccessTokenDisplayPrefixLen = 16
)

// GeneratePersonalAccessToken mints a high-entropy bearer token for an AI/MCP
// client. The plaintext is returned once to the caller; only the SHA256 hash is
// ever stored. The prefix is a non-secret slice kept for display in the UI.
func GeneratePersonalAccessToken() (token string, hash string, prefix string, err error) {
	secretBytes := make([]byte, 32)
	if _, err = rand.Read(secretBytes); err != nil {
		return "", "", "", err
	}

	token = PersonalAccessTokenPrefix + base64.RawURLEncoding.EncodeToString(secretBytes)
	hash = HashRefreshToken(token)
	prefix = token[:personalAccessTokenDisplayPrefixLen]
	return token, hash, prefix, nil
}

func IsPersonalAccessToken(token string) bool {
	return strings.HasPrefix(token, PersonalAccessTokenPrefix)
}
