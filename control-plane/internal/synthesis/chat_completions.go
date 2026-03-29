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

type chatCompletionRequest struct {
	Model     string    `json:"model"`
	Messages  []msgPart `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

type msgPart struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// postOpenAICompatibleChat POSTs to an OpenAI-compatible chat completions URL (Ollama, OpenAI, Azure OpenAI-compatible, etc.).
func postOpenAICompatibleChat(ctx context.Context, client *http.Client, fullURL, model, bearerToken, prompt string) (string, error) {
	body := chatCompletionRequest{
		Model:     model,
		Messages:  []msgPart{{Role: "user", Content: prompt}},
		MaxTokens: 4096,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("synthesis: marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("synthesis: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(bearerToken) != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("synthesis: request: %w", err)
	}
	defer resp.Body.Close()
	slurp, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("synthesis: %s: %s", resp.Status, string(slurp))
	}
	var out chatCompletionResponse
	if err := json.Unmarshal(slurp, &out); err != nil {
		return "", fmt.Errorf("synthesis: decode response: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", nil
	}
	return out.Choices[0].Message.Content, nil
}
