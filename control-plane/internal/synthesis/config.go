package synthesis

import (
	"fmt"
	"os"
	"strings"
)

// Provider identifiers (YAML synthesis.provider).
const (
	ProviderOllama    = "ollama"
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
)

// Config is optional server-side text generation for run-multi only (single-process; no separate service).
// Default: enabled=false — prefer client-side LLM synthesis.
type Config struct {
	Enabled bool `yaml:"enabled"`
	// Provider: ollama | openai | anthropic
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	// TimeoutSeconds HTTP timeout for provider calls (default 120).
	TimeoutSeconds int `yaml:"timeout_seconds"`
	// BaseURL optional. Ollama default http://127.0.0.1:11434; OpenAI default https://api.openai.com/v1; Anthropic default https://api.anthropic.com/v1
	BaseURL string `yaml:"base_url"`
	// APIKey optional inline secret (dev). Prefer APIKeyEnv for production.
	APIKey string `yaml:"api_key"`
	// APIKeyEnv names an environment variable to read (e.g. OPENAI_API_KEY). If empty, OPENAI_API_KEY / ANTHROPIC_API_KEY are used by provider.
	APIKeyEnv string `yaml:"api_key_env"`
}

// ApplyDefaults sets TimeoutSeconds when unset.
func ApplyDefaults(c *Config) {
	if c == nil {
		return
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = 120
	}
}

// Validate checks synthesis config when enabled. When !Enabled, returns nil.
func (c *Config) Validate() error {
	if c == nil || !c.Enabled {
		return nil
	}
	p := strings.ToLower(strings.TrimSpace(c.Provider))
	if p == "" {
		return fmt.Errorf("synthesis: enabled requires synthesis.provider (ollama, openai, or anthropic)")
	}
	if p != ProviderOllama && p != ProviderOpenAI && p != ProviderAnthropic {
		return fmt.Errorf("synthesis: unknown provider %q (use ollama, openai, or anthropic)", c.Provider)
	}
	if strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("synthesis: model is required when synthesis.enabled is true")
	}
	switch p {
	case ProviderOllama:
		return nil
	case ProviderOpenAI, ProviderAnthropic:
		_, err := c.resolvedAPIKey()
		return err
	default:
		return nil
	}
}

func (c *Config) resolvedAPIKey() (string, error) {
	if strings.TrimSpace(c.APIKey) != "" {
		return strings.TrimSpace(c.APIKey), nil
	}
	env := strings.TrimSpace(c.APIKeyEnv)
	if env == "" {
		p := strings.ToLower(strings.TrimSpace(c.Provider))
		if p == ProviderOpenAI {
			env = "OPENAI_API_KEY"
		} else {
			env = "ANTHROPIC_API_KEY"
		}
	}
	v := os.Getenv(env)
	if v == "" {
		return "", fmt.Errorf("synthesis: missing API key for provider %s (set synthesis.api_key, synthesis.api_key_env, or export %s)", c.Provider, env)
	}
	return v, nil
}

// ResolvedAPIKey returns the API key for OpenAI/Anthropic providers. Not used for Ollama.
func (c *Config) ResolvedAPIKey() (string, error) {
	if c == nil {
		return "", fmt.Errorf("synthesis: nil config")
	}
	return c.resolvedAPIKey()
}
