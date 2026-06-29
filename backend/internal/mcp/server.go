package mcp

import (
	"context"
	"encoding/json"
)

const serverName = "kantor-mcp"

type Server struct {
	tools    []ToolSpec
	byName   map[string]ToolSpec
	executor Executor
	baseURL  string
	version  string
}

func NewServer(tools []ToolSpec, executor Executor, baseURL string, version string) *Server {
	byName := make(map[string]ToolSpec, len(tools))
	for _, tool := range tools {
		byName[tool.Name] = tool
	}
	return &Server{
		tools:    tools,
		byName:   byName,
		executor: executor,
		baseURL:  baseURL,
		version:  version,
	}
}

// Handle processes a single JSON-RPC message. It returns the encoded response,
// or isNotification=true when the message is a notification that expects no
// reply.
func (s *Server) Handle(ctx context.Context, raw []byte, authHeader string, host string) (encoded []byte, isNotification bool) {
	var req request
	if err := json.Unmarshal(raw, &req); err != nil {
		return marshal(newError(nil, codeParseError, "parse error")), false
	}

	switch req.Method {
	case "initialize":
		return marshal(newResult(req.ID, s.initializeResult())), false
	case "ping":
		return marshal(newResult(req.ID, map[string]interface{}{})), false
	case "tools/list":
		return marshal(newResult(req.ID, map[string]interface{}{"tools": s.toolDescriptors()})), false
	case "tools/call":
		return marshal(s.callTool(ctx, req, authHeader, host)), false
	default:
		if len(req.ID) == 0 {
			return nil, true
		}
		return marshal(newError(req.ID, codeMethodMissing, "method not found: "+req.Method)), false
	}
}

func (s *Server) initializeResult() map[string]interface{} {
	return map[string]interface{}{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{"listChanged": false},
		},
		"serverInfo": map[string]interface{}{
			"name":    serverName,
			"version": s.version,
		},
	}
}

func (s *Server) toolDescriptors() []map[string]interface{} {
	descriptors := make([]map[string]interface{}, 0, len(s.tools))
	for _, tool := range s.tools {
		descriptors = append(descriptors, tool.descriptor())
	}
	return descriptors
}

func (s *Server) callTool(ctx context.Context, req request, authHeader string, host string) response {
	var params struct {
		Name      string                     `json:"name"`
		Arguments map[string]json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return newError(req.ID, codeInvalidParams, "invalid params")
	}

	tool, ok := s.byName[params.Name]
	if !ok {
		return newError(req.ID, codeInvalidParams, "unknown tool: "+params.Name)
	}

	httpReq, err := tool.buildRequest(ctx, s.baseURL, params.Arguments, authHeader, host)
	if err != nil {
		return newError(req.ID, codeInvalidParams, err.Error())
	}

	status, body, err := s.executor.Execute(httpReq)
	if err != nil {
		return newError(req.ID, codeInternalError, "tool execution failed: "+err.Error())
	}

	return newResult(req.ID, map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": string(body)},
		},
		"isError": status >= 400,
	})
}
