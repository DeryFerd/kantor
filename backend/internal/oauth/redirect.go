package oauth

import (
	"errors"
	"net/url"
)

// ErrInvalidRedirectURI is returned when a redirect URI does not satisfy the
// allowlist. Only https:// URIs and http://localhost (or 127.0.0.1 / ::1) are
// accepted. The localhost exception covers development MCP clients that run on
// the local machine.
var ErrInvalidRedirectURI = errors.New("redirect URI must use https or http://localhost")

// ValidateRedirectURI checks that uri is safe to use as an OAuth 2.x redirect
// target. It rejects javascript:, data:, file:, and any other non-http(s)
// scheme that could be exploited to redirect authorization codes to
// attacker-controlled locations.
func ValidateRedirectURI(uri string) error {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ErrInvalidRedirectURI
	}

	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		host := parsed.Hostname()
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			return nil
		}
		return ErrInvalidRedirectURI
	default:
		return ErrInvalidRedirectURI
	}
}
