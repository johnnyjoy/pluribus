package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"control-plane/internal/app"

	"github.com/go-chi/chi/v5"
)

func testInnerRouter(t *testing.T) http.Handler {
	t.Helper()
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":true}`)
	})
	return r
}

func TestHTTPHandler_initialize(t *testing.T) {
	h := NewHTTPHandler(testInnerRouter(t))
	srv := httptest.NewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	res, err := http.Post(srv.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var out struct {
		Result map[string]any `json:"result"`
		Error  any            `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Error != nil {
		t.Fatalf("error: %v", out.Error)
	}
	si, _ := out.Result["serverInfo"].(map[string]any)
	if si["name"] != mcpServerNameHTTP {
		t.Fatalf("serverInfo.name = %v", si["name"])
	}
	if si["version"] != Version {
		t.Fatalf("serverInfo.version = %v", si["version"])
	}
}

func TestHTTPHandler_toolsCall_health_loopback(t *testing.T) {
	h := NewHTTPHandler(testInnerRouter(t))
	srv := httptest.NewServer(h)
	defer srv.Close()

	reqBody := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"health","arguments":{}}}`
	res, err := http.Post(srv.URL, "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Error != nil {
		t.Fatalf("error: %v", out.Error)
	}
	if out.Result.IsError {
		t.Fatalf("isError: %+v", out.Result)
	}
	if len(out.Result.Content) == 0 || !strings.Contains(out.Result.Content[0].Text, `"ok":true`) {
		t.Fatalf("text: %+v", out.Result.Content)
	}
}

func TestHTTPHandler_promptsGet(t *testing.T) {
	h := NewHTTPHandler(testInnerRouter(t))
	srv := httptest.NewServer(h)
	defer srv.Close()
	body := `{"jsonrpc":"2.0","id":3,"method":"prompts/get","params":{"name":"` + PromptMemoryGrounding + `"}}`
	res, err := http.Post(srv.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out struct {
		Result map[string]any `json:"result"`
		Error  any            `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Error != nil {
		t.Fatal(out.Error)
	}
	msgs, _ := out.Result["messages"].([]any)
	if len(msgs) == 0 {
		t.Fatal("no messages")
	}
}

func TestHTTPHandler_resourcesRead(t *testing.T) {
	h := NewHTTPHandler(testInnerRouter(t))
	srv := httptest.NewServer(h)
	defer srv.Close()
	body := `{"jsonrpc":"2.0","id":4,"method":"resources/read","params":{"uri":"` + URIDisciplineDoctrine + `"}}`
	res, err := http.Post(srv.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out struct {
		Result map[string]any `json:"result"`
		Error  any            `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Error != nil {
		t.Fatal(out.Error)
	}
	contents, _ := out.Result["contents"].([]any)
	if len(contents) == 0 {
		t.Fatal("no contents")
	}
}

func TestWrapHandler_disabled(t *testing.T) {
	inner := testInnerRouter(t)
	cfg := &app.Config{MCP: &app.MCPConfig{Disabled: true}}
	wrapped := WrapHandler(inner, cfg)

	t.Run("mcp bypassed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/mcp", strings.NewReader(`{}`))
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for unhandled path, got %d", rec.Code)
		}
	})
}

func TestWrapHandler_integrationPath(t *testing.T) {
	inner := testInnerRouter(t)
	wrapped := WrapHandler(inner, &app.Config{})
	req := httptest.NewRequest(http.MethodPost, "/v1/mcp", bytes.NewBufferString(
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}
