package oauth

import "testing"

func TestValidateRedirectURI(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		// Accepted URIs.
		{name: "https external", uri: "https://example.com/callback", wantErr: false},
		{name: "https with path and query", uri: "https://app.example.com/oauth/callback?foo=bar", wantErr: false},
		{name: "http localhost by name", uri: "http://localhost/callback", wantErr: false},
		{name: "http localhost with port", uri: "http://localhost:8080/callback", wantErr: false},
		{name: "http 127.0.0.1", uri: "http://127.0.0.1/callback", wantErr: false},
		{name: "http 127.0.0.1 with port", uri: "http://127.0.0.1:3000/callback", wantErr: false},
		{name: "http ::1 IPv6", uri: "http://[::1]:9000/callback", wantErr: false},

		// Rejected URIs — dangerous or non-https schemes.
		{name: "javascript scheme", uri: "javascript:alert(1)", wantErr: true},
		{name: "data scheme", uri: "data:text/html,<script>alert(1)</script>", wantErr: true},
		{name: "file scheme", uri: "file:///etc/passwd", wantErr: true},
		{name: "http non-localhost", uri: "http://evil.com/callback", wantErr: true},
		{name: "http non-loopback IP", uri: "http://192.168.1.1/callback", wantErr: true},
		{name: "no scheme", uri: "example.com/callback", wantErr: true},
		{name: "empty string", uri: "", wantErr: true},
		{name: "scheme only", uri: "https://", wantErr: true},
		{name: "custom deep-link scheme", uri: "myapp://callback", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRedirectURI(tc.uri)
			if tc.wantErr && err == nil {
				t.Errorf("ValidateRedirectURI(%q) = nil, want error", tc.uri)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("ValidateRedirectURI(%q) = %v, want nil", tc.uri, err)
			}
		})
	}
}
