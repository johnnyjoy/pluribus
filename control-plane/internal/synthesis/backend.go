// Package synthesis provides an optional, narrow backend for text generation (run-multi only).
// It is not a general AI framework; providers are isolated behind a small interface.
package synthesis

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Backend generates text from a single user prompt (server-side synthesis for run-multi).
type Backend interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// NewBackend returns a provider implementation when cfg.Enabled is true.
// When cfg.Enabled is false, returns (nil, nil) — callers should omit run-multi LLM wiring.
func NewBackend(cfg *Config) (Backend, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, nil
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	p := strings.ToLower(strings.TrimSpace(cfg.Provider))
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	client := &http.Client{Timeout: timeout}

	switch p {
	case ProviderOllama:
		return newOllamaBackend(cfg, client)
	case ProviderOpenAI:
		return newOpenAIBackend(cfg, client)
	case ProviderAnthropic:
		return newAnthropicBackend(cfg, client)
	default:
		return nil, fmt.Errorf("synthesis: unknown provider %q (use ollama, openai, or anthropic)", cfg.Provider)
	}
}
