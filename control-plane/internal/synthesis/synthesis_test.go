package synthesis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestNewBackend_disabled(t *testing.T) {
	b, err := NewBackend(&Config{Enabled: false})
	if err != nil {
		t.Fatalf("NewBackend: %v", err)
	}
	if b != nil {
		t.Fatal("expected nil backend when disabled")
	}
}

func TestSynthesisConfig_Validate_enabledMissingProvider(t *testing.T) {
	c := &Config{Enabled: true, Model: "x"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for missing provider")
	}
}

func TestSynthesisConfig_Validate_unknownProvider(t *testing.T) {
	c := &Config{Enabled: true, Provider: "azure", Model: "x"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestSynthesisConfig_Validate_openaiMissingKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	c := &Config{Enabled: true, Provider: ProviderOpenAI, Model: "gpt-4"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestSynthesisConfig_Validate_openaiWithKey(t *testing.T) {
	c := &Config{Enabled: true, Provider: ProviderOpenAI, Model: "gpt-4", APIKey: "sk-test"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestSynthesisConfig_Validate_anthropicEnv(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "k")
	defer os.Unsetenv("ANTHROPIC_API_KEY")
	c := &Config{Enabled: true, Provider: ProviderAnthropic, Model: "claude-3-5-sonnet-20241022"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestOllamaBackend_Generate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]string{"content": "hello"}}},
		})
	}))
	defer srv.Close()
	cfg := &Config{
		Enabled:        true,
		Provider:       ProviderOllama,
		Model:          "m",
		TimeoutSeconds: 30,
		BaseURL:        strings.TrimSuffix(srv.URL, "/"),
	}
	b, err := NewBackend(cfg)
	if err != nil {
		t.Fatal(err)
	}
	out, err := b.Generate(context.Background(), "prompt")
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello" {
		t.Errorf("got %q", out)
	}
}

func TestOpenAIBackend_Generate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("missing bearer")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]string{"content": "ok"}}},
		})
	}))
	defer srv.Close()
	cfg := &Config{
		Enabled:        true,
		Provider:       ProviderOpenAI,
		Model:          "gpt-4",
		APIKey:         "sk-test",
		TimeoutSeconds: 30,
		BaseURL:        strings.TrimSuffix(srv.URL, "/") + "/v1",
	}
	b, err := NewBackend(cfg)
	if err != nil {
		t.Fatal(err)
	}
	out, err := b.Generate(context.Background(), "p")
	if err != nil {
		t.Fatal(err)
	}
	if out != "ok" {
		t.Errorf("got %q", out)
	}
}

func TestAnthropicBackend_Generate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "anthropic-key" {
			t.Errorf("missing x-api-key")
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Error("missing anthropic-version")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{{"type": "text", "text": "anthropic-out"}},
		})
	}))
	defer srv.Close()
	cfg := &Config{
		Enabled:        true,
		Provider:       ProviderAnthropic,
		Model:          "claude-3-5-sonnet-20241022",
		APIKey:         "anthropic-key",
		TimeoutSeconds: 30,
		BaseURL:        strings.TrimSuffix(srv.URL, "/") + "/v1",
	}
	b, err := NewBackend(cfg)
	if err != nil {
		t.Fatal(err)
	}
	out, err := b.Generate(context.Background(), "p")
	if err != nil {
		t.Fatal(err)
	}
	if out != "anthropic-out" {
		t.Errorf("got %q", out)
	}
}
