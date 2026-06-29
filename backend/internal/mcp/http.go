package mcp

import (
	"io"
	"net/http"
	"strings"
)

const maxRequestBytes = 1 << 20

// HTTPHandler exposes the MCP server over a single-message Streamable HTTP
// endpoint: the client POSTs one JSON-RPC message and receives one JSON-RPC
// response (or 202 for notifications).
func (s *Server) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if s.authorize != nil {
			token := bearerToken(r.Header.Get("Authorization"))
			if token == "" || !s.authorize(token) {
				resourceMetadata := requestBaseURL(r) + "/.well-known/oauth-protected-resource"
				w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+resourceMetadata+`"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		raw, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBytes))
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}

		encoded, isNotification := s.Handle(r.Context(), raw, r.Header.Get("Authorization"), r.Host)
		if isNotification {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(encoded)
	})
}

func bearerToken(header string) string {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
