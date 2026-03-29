package synthesis

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type openAIBackend struct {
	url    string
	model  string
	apiKey string
	client *http.Client
}

func newOpenAIBackend(cfg *Config, client *http.Client) (Backend, error) {
	key, err := cfg.ResolvedAPIKey()
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	full := base + "/chat/completions"
	return &openAIBackend{
		url:    full,
		model:  strings.TrimSpace(cfg.Model),
		apiKey: key,
		client: client,
	}, nil
}

func (b *openAIBackend) Generate(ctx context.Context, prompt string) (string, error) {
	if b == nil || b.client == nil {
		return "", fmt.Errorf("synthesis: openai backend not initialized")
	}
	return postOpenAICompatibleChat(ctx, b.client, b.url, b.model, b.apiKey, prompt)
}
