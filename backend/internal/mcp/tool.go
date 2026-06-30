package mcp

import (
	"fmt"
	"net/http"
)

type ToolSpec struct {
	Name         string
	Description  string
	Method       string
	PathTemplate string
	PathParams   []string
	// Meta carries curated query/pagination metadata so AI clients can discover
	// filters and page through all data. Nil for endpoints without an annotation
	// (they fall back to the generic opaque query schema).
	Meta *EndpointMeta
}

// QueryParam describes one real query-string parameter of an endpoint.
type QueryParam struct {
	Name        string
	Type        string // "string", "integer", "boolean"
	Description string
	Enum        []string
}

// EndpointMeta is the curated annotation for an endpoint.
type EndpointMeta struct {
	Description    string
	Query          []QueryParam
	Paginated      bool
	PerPageDefault int
	PerPageMax     int // 0 = no enforced cap
}

func (t ToolSpec) hasBody() bool {
	return t.Method == http.MethodPost || t.Method == http.MethodPut || t.Method == http.MethodPatch
}

func (t ToolSpec) inputSchema() map[string]interface{} {
	properties := map[string]interface{}{}
	required := make([]string, 0, len(t.PathParams))

	for _, param := range t.PathParams {
		properties[param] = map[string]interface{}{
			"type":        "string",
			"description": "Path parameter " + param,
		}
		required = append(required, param)
	}

	properties["query"] = t.querySchema()

	if t.hasBody() {
		properties["body"] = map[string]interface{}{
			"type":        "object",
			"description": "JSON request body for this endpoint.",
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// querySchema renders the query object. With a curated EndpointMeta it exposes
// each real filter as a typed, described property and surfaces page/per_page
// (with defaults and caps) so the client knows to paginate through ALL results.
// Without metadata it falls back to a free-form key/value object.
func (t ToolSpec) querySchema() map[string]interface{} {
	if t.Meta == nil || (len(t.Meta.Query) == 0 && !t.Meta.Paginated) {
		return map[string]interface{}{
			"type":        "object",
			"description": "Optional query-string parameters as key/value pairs.",
		}
	}

	props := map[string]interface{}{}
	for _, param := range t.Meta.Query {
		prop := map[string]interface{}{"type": param.Type}
		if param.Description != "" {
			prop["description"] = param.Description
		}
		if len(param.Enum) > 0 {
			prop["enum"] = param.Enum
		}
		props[param.Name] = prop
	}

	if t.Meta.Paginated {
		props["page"] = map[string]interface{}{
			"type":        "integer",
			"minimum":     1,
			"description": "1-based page number (default 1).",
		}
		perPage := map[string]interface{}{"type": "integer", "minimum": 1}
		desc := fmt.Sprintf("Items per page (default %d", t.Meta.PerPageDefault)
		if t.Meta.PerPageMax > 0 {
			desc += fmt.Sprintf(", max %d", t.Meta.PerPageMax)
			perPage["maximum"] = t.Meta.PerPageMax
		}
		desc += "). The response is paginated — page through every page to retrieve all data."
		perPage["description"] = desc
		props["per_page"] = perPage
	}

	return map[string]interface{}{
		"type":        "object",
		"description": "Query-string filters and pagination.",
		"properties":  props,
	}
}

func (t ToolSpec) descriptor() map[string]interface{} {
	description := t.Description
	if t.Meta != nil && t.Meta.Description != "" {
		description = t.Meta.Description + " (" + t.Method + " " + t.PathTemplate + ")"
	}
	return map[string]interface{}{
		"name":        t.Name,
		"description": description,
		"inputSchema": t.inputSchema(),
		"annotations": map[string]interface{}{
			"readOnlyHint":    t.Method == http.MethodGet,
			"destructiveHint": t.Method == http.MethodDelete,
			"idempotentHint":  t.Method == http.MethodGet || t.Method == http.MethodPut || t.Method == http.MethodDelete,
		},
	}
}
