package runmulti

import (
	"context"
	"fmt"

	"control-plane/internal/synthesis"
)

// LLMCaller generates text from a prompt (injectable for tests).
type LLMCaller interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// NewSynthesisLLM wires optional backend synthesis for run-multi. Pass nil to leave run-multi without LLM (orchestration only via handler errors).
func NewSynthesisLLM(backend synthesis.Backend) LLMCaller {
	if backend == nil {
		return nil
	}
	return &synthesisLLM{backend: backend}
}

type synthesisLLM struct {
	backend synthesis.Backend
}

func (s *synthesisLLM) Generate(ctx context.Context, prompt string) (string, error) {
	if s.backend == nil {
		return "", fmt.Errorf("runmulti: synthesis backend is nil")
	}
	text, err := s.backend.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("runmulti: synthesis: %w", err)
	}
	return text, nil
}
