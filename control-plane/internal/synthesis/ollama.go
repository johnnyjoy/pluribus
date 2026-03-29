package synthesis

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type ollamaBackend struct {
	url    string
	model  string
	client *http.Client
}

func newOllamaBackend(cfg *Config, client *http.Client) (Backend, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = "http://127.0.0.1:11434"
	}
	full := base + "/v1/chat/completions"
	return &ollamaBackend{
		url:    full,
		model:  strings.TrimSpace(cfg.Model),
		client: client,
	}, nil
}

func (b *ollamaBackend) Generate(ctx context.Context, prompt string) (string, error) {
	if b == nil || b.client == nil {
		return "", fmt.Errorf("synthesis: ollama backend not initialized")
	}
	return postOpenAICompatibleChat(ctx, b.client, b.url, b.model, "", prompt)
}
