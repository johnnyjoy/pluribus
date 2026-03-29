package synthesis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const anthropicVersion = "2023-06-01"

type anthropicRequest struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	Messages  []anthropicMsg `json:"messages"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type anthropicBackend struct {
	url    string
	model  string
	apiKey string
	client *http.Client
}

func newAnthropicBackend(cfg *Config, client *http.Client) (Backend, error) {
	key, err := cfg.ResolvedAPIKey()
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = "https://api.anthropic.com/v1"
	}
	full := base + "/messages"
	return &anthropicBackend{
		url:    full,
		model:  strings.TrimSpace(cfg.Model),
		apiKey: key,
		client: client,
	}, nil
}

func (b *anthropicBackend) Generate(ctx context.Context, prompt string) (string, error) {
	if b == nil || b.client == nil {
		return "", fmt.Errorf("synthesis: anthropic backend not initialized")
	}
	body := anthropicRequest{
		Model:     b.model,
		MaxTokens: 4096,
		Messages:  []anthropicMsg{{Role: "user", Content: prompt}},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("synthesis: marshal anthropic request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.url, bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("synthesis: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", b.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	resp, err := b.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("synthesis: request: %w", err)
	}
	defer resp.Body.Close()
	slurp, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("synthesis: %s: %s", resp.Status, string(slurp))
	}
	var out anthropicResponse
	if err := json.Unmarshal(slurp, &out); err != nil {
		return "", fmt.Errorf("synthesis: decode anthropic response: %w", err)
	}
	for _, block := range out.Content {
		if block.Type == "text" && block.Text != "" {
			return block.Text, nil
		}
	}
	return "", nil
}
