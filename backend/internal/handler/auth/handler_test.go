package auth

import (
	"net/http/httptest"
	"testing"
)

func TestRequestPublicBaseURL(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		forwarded    string
		secureCookie bool
		want         string
	}{
		{
			name:         "production ignores request host",
			host:         "app.example.com",
			secureCookie: true,
			want:         "",
		},
		{
			name:         "development localhost defaults to http",
			host:         "localhost:3000",
			secureCookie: false,
			want:         "http://localhost:3000",
		},
		{
			name:         "development localhost honors forwarded https",
			host:         "localhost:3000",
			forwarded:    "https",
			secureCookie: false,
			want:         "https://localhost:3000",
		},
		{
			name:         "development loopback ip allowed",
			host:         "127.0.0.1:5173",
			secureCookie: false,
			want:         "http://127.0.0.1:5173",
		},
		{
			name:         "development public host rejected",
			host:         "evil.example",
			secureCookie: false,
			want:         "",
		},
		{
			name:         "invalid forwarded proto ignored",
			host:         "localhost:3000",
			forwarded:    "javascript",
			secureCookie: false,
			want:         "http://localhost:3000",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "http://example.test/api/v1/auth/forgot-password", nil)
			req.Host = tc.host
			if tc.forwarded != "" {
				req.Header.Set("X-Forwarded-Proto", tc.forwarded)
			}

			got := requestPublicBaseURL(req, tc.secureCookie)
			if got != tc.want {
				t.Fatalf("requestPublicBaseURL() = %q, want %q", got, tc.want)
			}
		})
	}
}
