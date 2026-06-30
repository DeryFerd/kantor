package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAnnotatedToolExposesFiltersAndPagination(t *testing.T) {
	tool := ToolSpec{
		Name:         "get_hris_employees",
		Method:       http.MethodGet,
		PathTemplate: "/api/v1/hris/employees",
		Meta:         annotationFor(http.MethodGet, "/api/v1/hris/employees"),
	}
	if tool.Meta == nil {
		t.Fatal("expected annotation for GET /api/v1/hris/employees")
	}

	query, ok := tool.inputSchema()["properties"].(map[string]interface{})["query"].(map[string]interface{})
	if !ok {
		t.Fatal("query schema missing")
	}
	props, ok := query["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected named query properties, got opaque object")
	}
	for _, want := range []string{"search", "department", "status", "page", "per_page"} {
		if _, ok := props[want]; !ok {
			t.Errorf("query schema missing %q", want)
		}
	}

	// read-only hint should be set for a GET tool
	ann := tool.descriptor()["annotations"].(map[string]interface{})
	if ann["readOnlyHint"] != true {
		t.Errorf("GET tool should be readOnlyHint=true, got %v", ann["readOnlyHint"])
	}
}

func TestUnannotatedToolFallsBackToOpaqueQuery(t *testing.T) {
	tool := ToolSpec{Name: "get_unknown", Method: http.MethodGet, PathTemplate: "/api/v1/unknown/thing"}
	query := tool.inputSchema()["properties"].(map[string]interface{})["query"].(map[string]interface{})
	if _, hasProps := query["properties"]; hasProps {
		t.Error("unannotated tool should keep the opaque query object (no properties)")
	}
}

func TestBuildRequestEncodesNumbersAndArrays(t *testing.T) {
	tool := ToolSpec{Name: "get_x", Method: http.MethodGet, PathTemplate: "/api/v1/x"}
	args := map[string]json.RawMessage{
		"query": json.RawMessage(`{"per_page": 1000000, "tag": ["a","b"], "read": true}`),
	}
	req, err := tool.buildRequest(context.Background(), "", args, "", "")
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	q := req.URL.Query()
	if got := q.Get("per_page"); got != "1000000" {
		t.Errorf("integer query mangled: per_page=%q (want 1000000)", got)
	}
	if got := q["tag"]; len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("array query not expanded to repeated keys: %v", got)
	}
	if got := q.Get("read"); got != "true" {
		t.Errorf("bool query mangled: read=%q", got)
	}
}

func TestAnnotationKeysAreWellFormed(t *testing.T) {
	methods := map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true}
	for key := range endpointAnnotations {
		parts := strings.SplitN(key, " ", 2)
		if len(parts) != 2 || !methods[parts[0]] || !strings.HasPrefix(parts[1], "/api/v1/") {
			t.Errorf("malformed annotation key %q (want 'METHOD /api/v1/...')", key)
		}
	}
}
