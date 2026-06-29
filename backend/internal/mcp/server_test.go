package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

func noopHandler() http.HandlerFunc {
	return func(http.ResponseWriter, *http.Request) {}
}

func TestBuildCatalogFiltersAndNames(t *testing.T) {
	router := chi.NewRouter()
	handler := noopHandler()
	router.Route("/api/v1", func(api chi.Router) {
		api.Get("/hris/employees", handler)
		api.Get("/hris/employees/{employeeID}", handler)
		api.Post("/hris/employees", handler)
		api.Post("/auth/login", handler)
		api.Post("/admin/users/{userID}/toggle-super-admin", handler)
		api.Get("/admin/settings/registration", handler)
		api.Get("/auth/pat", handler)
		api.Post("/auth/pat", handler)
		api.Delete("/auth/pat/{tokenID}", handler)
		api.Post("/auth/change-password", handler)
	})
	router.Get("/healthz", handler)

	tools, err := BuildCatalog(router)
	if err != nil {
		t.Fatalf("BuildCatalog returned error: %v", err)
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}

	wantPresent := []string{"get_hris_employees", "get_hris_employees_employeeid", "post_hris_employees"}
	for _, name := range wantPresent {
		if !names[name] {
			t.Errorf("expected tool %q to be present, got %v", name, names)
		}
	}

	wantAbsent := []string{
		"post_auth_login",
		"post_admin_users_userid_toggle_super_admin",
		"get_admin_settings_registration",
		"get_auth_pat",
		"post_auth_pat",
		"delete_auth_pat_tokenid",
		"post_auth_change_password",
		"get_healthz",
	}
	for _, name := range wantAbsent {
		if names[name] {
			t.Errorf("expected tool %q to be excluded", name)
		}
	}
}

func TestPathParamsAndSchema(t *testing.T) {
	tool := ToolSpec{
		Name:         "put_hris_bonuses_bonusid",
		Method:       http.MethodPut,
		PathTemplate: "/api/v1/hris/bonuses/{bonusID}",
		PathParams:   pathParams("/api/v1/hris/bonuses/{bonusID}"),
	}
	if len(tool.PathParams) != 1 || tool.PathParams[0] != "bonusID" {
		t.Fatalf("unexpected path params: %v", tool.PathParams)
	}

	schema := tool.inputSchema()
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("schema properties missing")
	}
	if _, ok := properties["bonusID"]; !ok {
		t.Errorf("expected bonusID property")
	}
	if _, ok := properties["body"]; !ok {
		t.Errorf("expected body property for write method")
	}
}

type stubExecutor struct {
	lastRequest *http.Request
	status      int
	body        []byte
}

func (e *stubExecutor) Execute(req *http.Request) (int, []byte, error) {
	e.lastRequest = req
	return e.status, e.body, nil
}

func TestServerToolsCallDispatches(t *testing.T) {
	tools := []ToolSpec{{
		Name:         "get_hris_employees_employeeid",
		Method:       http.MethodGet,
		PathTemplate: "/api/v1/hris/employees/{employeeID}",
		PathParams:   []string{"employeeID"},
	}}
	executor := &stubExecutor{status: http.StatusOK, body: []byte(`{"success":true}`)}
	server := NewServer(tools, executor, "", "test")

	message := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_hris_employees_employeeid","arguments":{"employeeID":"abc-123","query":{"include":"salary"}}}}`)
	encoded, isNotification := server.Handle(context.Background(), message, "Bearer kantor_pat_secret", "tenant.example.com")
	if isNotification {
		t.Fatalf("tools/call should not be a notification")
	}

	if executor.lastRequest == nil {
		t.Fatalf("executor was not called")
	}
	if got := executor.lastRequest.URL.Path; got != "/api/v1/hris/employees/abc-123" {
		t.Errorf("unexpected path: %q", got)
	}
	if got := executor.lastRequest.URL.Query().Get("include"); got != "salary" {
		t.Errorf("unexpected query: %q", got)
	}
	if got := executor.lastRequest.Header.Get("Authorization"); got != "Bearer kantor_pat_secret" {
		t.Errorf("auth header not forwarded: %q", got)
	}
	if got := executor.lastRequest.Host; got != "tenant.example.com" {
		t.Errorf("host not forwarded: %q", got)
	}

	var parsed response
	if err := json.Unmarshal(encoded, &parsed); err != nil {
		t.Fatalf("response not valid json: %v", err)
	}
	if parsed.Error != nil {
		t.Fatalf("unexpected error in response: %+v", parsed.Error)
	}
}

func TestServerInitialize(t *testing.T) {
	server := NewServer(nil, &stubExecutor{}, "", "test")
	encoded, _ := server.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`), "", "")

	var parsed struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal(encoded, &parsed); err != nil {
		t.Fatalf("invalid initialize response: %v", err)
	}
	if parsed.Result.ProtocolVersion != protocolVersion {
		t.Errorf("unexpected protocol version: %q", parsed.Result.ProtocolVersion)
	}
}
