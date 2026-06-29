package mcp

import "net/http"

type ToolSpec struct {
	Name         string
	Description  string
	Method       string
	PathTemplate string
	PathParams   []string
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

	properties["query"] = map[string]interface{}{
		"type":        "object",
		"description": "Optional query-string parameters as key/value pairs.",
	}

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

func (t ToolSpec) descriptor() map[string]interface{} {
	return map[string]interface{}{
		"name":        t.Name,
		"description": t.Description,
		"inputSchema": t.inputSchema(),
	}
}
