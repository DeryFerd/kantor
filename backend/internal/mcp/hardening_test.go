package mcp

import (
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestBuildCatalogExcludesExportAndBinary(t *testing.T) {
	router := chi.NewRouter()
	h := noopHandler()
	router.Route("/api/v1", func(api chi.Router) {
		api.Get("/hris/employees", h)
		api.Get("/hris/employees/export", h)
		api.Get("/marketing/leads/export", h)
		api.Get("/tracker/extension/download", h)
		api.Get("/files/{type}/{id}/{filename}", h)
	})

	tools, err := BuildCatalog(router)
	if err != nil {
		t.Fatalf("BuildCatalog: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	if !names["get_hris_employees"] {
		t.Error("expected get_hris_employees to be present")
	}
	for _, excluded := range []string{
		"get_hris_employees_export",
		"get_marketing_leads_export",
		"get_tracker_extension_download",
		"get_files_type_id_filename",
	} {
		if names[excluded] {
			t.Errorf("expected %q to be excluded from the tool surface", excluded)
		}
	}
}

func TestClampPerPage(t *testing.T) {
	tests := []struct {
		name        string
		perPage     string
		endpointMax int
		want        string
	}{
		{"over global cap", "10000", 0, "100"},
		{"over endpoint max", "500", 100, "100"},
		{"under cap unchanged", "20", 100, "20"},
		{"tighter endpoint max", "80", 50, "50"},
		{"absent unchanged", "", 0, ""},
		{"non-numeric unchanged", "abc", 0, "abc"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := url.Values{}
			if tc.perPage != "" {
				q.Set("per_page", tc.perPage)
			}
			clampPerPage(q, tc.endpointMax)
			if got := q.Get("per_page"); got != tc.want {
				t.Errorf("per_page = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBoundToolText(t *testing.T) {
	small := []byte(`{"ok":true}`)
	if got := boundToolText(small); got != string(small) {
		t.Errorf("small body should pass through, got %q", got)
	}

	big := make([]byte, maxToolResponseBytes+5000)
	for i := range big {
		big[i] = 'x'
	}
	out := boundToolText(big)
	if len(out) <= maxToolResponseBytes {
		// truncated body + appended note; note makes it a bit longer than the cap
		t.Errorf("expected truncated output with a note, len=%d", len(out))
	}
	if !strings.Contains(out, "truncated") {
		t.Error("truncated output must include a note guiding the client to paginate/filter")
	}
	if strings.Count(out, "x") > maxToolResponseBytes {
		t.Error("payload portion must be capped at maxToolResponseBytes")
	}
}
