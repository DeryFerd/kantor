package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// The stdio MCP server is a thin JSON-RPC proxy: it reads newline-delimited
// messages from stdin, forwards each to the running Kantor /mcp endpoint with
// the configured personal access token, and writes the response to stdout. This
// keeps a single source of truth for the tool catalog (the server's route
// table) and reuses the full tenant + auth + RBAC chain for every call.
func main() {
	baseURL := strings.TrimRight(os.Getenv("KANTOR_BASE_URL"), "/")
	pat := strings.TrimSpace(os.Getenv("KANTOR_PAT"))
	host := strings.TrimSpace(os.Getenv("KANTOR_TENANT_HOST"))

	if baseURL == "" || pat == "" {
		fmt.Fprintln(os.Stderr, "kantor-mcp: KANTOR_BASE_URL and KANTOR_PAT are required")
		os.Exit(1)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	for {
		line, err := reader.ReadBytes('\n')
		if payload := bytes.TrimSpace(line); len(payload) > 0 {
			if resp, ok := forward(client, baseURL, pat, host, payload); ok {
				_, _ = writer.Write(resp)
				_ = writer.WriteByte('\n')
				_ = writer.Flush()
			}
		}
		if err != nil {
			if err != io.EOF {
				fmt.Fprintln(os.Stderr, "kantor-mcp: read error:", err)
			}
			return
		}
	}
}

func forward(client *http.Client, baseURL string, pat string, host string, payload []byte) ([]byte, bool) {
	req, err := http.NewRequest(http.MethodPost, baseURL+"/mcp", bytes.NewReader(payload))
	if err != nil {
		fmt.Fprintln(os.Stderr, "kantor-mcp: build request:", err)
		return nil, false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+pat)
	if host != "" {
		req.Host = host
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kantor-mcp: request failed:", err)
		return nil, false
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		fmt.Fprintln(os.Stderr, "kantor-mcp: read response:", err)
		return nil, false
	}

	if resp.StatusCode == http.StatusAccepted || len(bytes.TrimSpace(data)) == 0 {
		return nil, false
	}
	return data, true
}
