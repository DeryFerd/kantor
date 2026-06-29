package mcp

import (
	"io"
	"net/http"
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
